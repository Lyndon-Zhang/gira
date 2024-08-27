package registry

/// ## server key
/// server key 会设置过期时间。
///	如果server异常关闭，key就没法续期，其他server就会被通知到
/// 但如果由于网络波动等原因，导致key没法续期，导致server key会被非正常delete掉, 其他server会被通知，等网络恢复正常，当前的server如果watch到后，应该要偿试重新注册自己。
///	所以这里有两种原因，会watch到自己被反注册，一个是意外情况，一个是正常关闭。重新注册应该区分开来分别处理。

/// ## go routinue
/// watch routine 监听peer变化
/// lease routine 定时续期
/// ctrl routine 监听程序结束

/// ## context
// app
//  |
//  ctx --- cancelCtx
//  |    |
//  |    --errCtx
//
//
// background
//  |
//  -----watchCtx---unregister self
//
// 程序结束后，手动关闭watchCtx, 等成功从注册表删除了自己再结束

// ## 特殊情况
// 1.程序退出时，如果server key被*自己*占用着，则要等删除了，再退出
// 2.程序退出时，如果server key没被*自己*占用着，则可以直接退出了
// 3.如果由于网络等异常原因，导致server key过期，则要重新抢占，还要设置重试(TODO)

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"

	"github.com/Lyndon-Zhang/gira"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type peer_registry struct {
	selfPrefix string // /peer/<<FullName>>
	prefix     string // /peer/
	// TODO 要加锁
	peers                  sync.Map //map[string]*gira.Peer
	isNormalUnregisterSelf bool
	SelfPeer               *gira.Peer
	ctx                    context.Context
	cancelFunc             context.CancelFunc
	watchStartRevision     int64
	selfRevision           int64
}

func newConfigPeerRegistry(r *Registry) (*peer_registry, error) {
	ctx, cancelFunc := context.WithCancel(r.ctx)
	self := &peer_registry{
		prefix:     "/peer/attribute/",
		selfPrefix: fmt.Sprintf("/peer/attribute/%s/", r.appFullName),
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
	return self, nil
}

func (self *peer_registry) RangePeers(f func(k any, v any) bool) {
	self.peers.Range(f)
}

// 根据app名查找节点
// 协程安全
func (self *peer_registry) getPeer(r *Registry, fullName string) *gira.Peer {
	if lastValue, ok := self.peers.Load(fullName); ok {
		lastPeer := lastValue.(*gira.Peer)
		return lastPeer
	}
	return nil
}

func (self *peer_registry) stop(r *Registry) error {
	log.Debug("peer registry on stop")
	if err := self.unregisterSelf(r); err != nil {
		return err
	}
	return nil
}

// func (self *peer_registry) Start(r *Registry) error {
// 	// 注册自己
// 	if err := self.registerSelf(r); err != nil {
// 		return err
// 	}
// 	if err := self.initPeers(r); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (self *peer_registry) Serve(r *Registry) error {
	// 侦听伙伴信息
	if err := self.watchPeers(r); err != nil {
		return err
	}
	return nil
}

func (self *peer_registry) notify(r *Registry) error {
	self.peers.Range(func(k any, v any) bool {
		peer := v.(*gira.Peer)
		self.onPeerAdd(r, peer)
		return true
	})
	return nil
}

func (self *peer_registry) onPeerAdd(r *Registry, peer *gira.Peer) {
	if r.isNotify == 0 {
		return
	}
	log.Debugw("peer registry on peer add", "full_name", peer.FullName)
	for _, handler := range r.peerWatchHandlers {
		handler.OnPeerAdd(peer)
	}
}

func (self *peer_registry) onPeerDelete(r *Registry, peer *gira.Peer) error {
	log.Debugw("peer registry on peer delete", "full_name", peer.FullName)
	for _, handler := range r.peerWatchHandlers {
		handler.OnPeerDelete(peer)
	}
	if peer.FullName == r.appFullName {
		if self.isNormalUnregisterSelf {
			self.cancelFunc()
		} else {
			log.Errorw("peer registry delete myself???", "is_normal", self.isNormalUnregisterSelf)
			if err := self.registerSelf(r); err != nil {
				log.Errorw("register self fail", "error", err)
				return err
			}
		}
	}
	return nil
}

func (self *peer_registry) onPeerUpdate(r *Registry, peer *gira.Peer) {
	log.Debugw("peer registry on peer update", "full_name", peer.FullName)
	for _, handler := range r.peerWatchHandlers {
		handler.OnPeerUpdate(peer)
	}
	r.peerResolver.onPeerUpdate(r, peer)
}

func (self *peer_registry) onKvPut(r *Registry, kv *mvccpb.KeyValue) error {
	pats := strings.Split(string(kv.Key), "/")
	if len(pats) != 5 {
		log.Errorw("peer registry got a invalid key", "key", string(kv.Key))
		return errors.New("invalid peer registry key", "key", string(kv.Key))
	}
	pats = pats[3:]
	fullName := pats[0]
	attrName := pats[1]
	name, serverId, err := gira.ParseAppFullName(fullName)
	if err != nil {
		log.Errorw("peer registry got a invalid key", "full_name", fullName)
		return err
	}
	attrValue := string(kv.Value)
	if lastValue, ok := self.peers.Load(fullName); ok {
		lastPeer := lastValue.(*gira.Peer)
		if attrName == GRPC_KEY {
			if lastPeer.Address == "" {
				// 新增节点
				lastPeer.Address = attrValue
				lastPeer.Url = formatPeerUrl(lastPeer.FullName)
				log.Debugw("peer registry add peer", "full_name", fullName, GRPC_KEY, attrValue)
				if lastPeer.FullName == r.appFullName {
					self.SelfPeer = lastPeer
				}
				self.onPeerAdd(r, lastPeer)
			} else if attrValue != lastPeer.Address {
				// 节点地址改变
				lastPeer.Address = attrValue
				self.onPeerUpdate(r, lastPeer)
				log.Debugw("peer registry update peer", "full_name", fullName, GRPC_KEY, attrValue, "last_attr_value", lastPeer.Address)
			} else {
				log.Debugw("peer registry update peer", "full_name", fullName, GRPC_KEY, attrValue, "last_attr_value", lastPeer.Address)
			}
		} else {
			if lastAttrValue, ok := lastPeer.Metadata[attrName]; ok {
				if lastAttrValue != attrValue {
					log.Debugw("peer registry update peer attr", "full_name", fullName, attrName, attrValue, "last_attr_value", lastAttrValue)
				} else {
					log.Debugw("peer registry update peer attr", "full_name", fullName, attrName, attrValue, "last_attr_value", lastAttrValue)
				}
			} else {
				log.Debugw("peer registry add peer attr", "full_name", fullName, attrName, attrValue)
			}
			lastPeer.Metadata[attrName] = attrValue
		}
	} else {
		peer := &gira.Peer{
			Id:       serverId,
			Name:     name,
			FullName: fullName,
			Metadata: make(map[string]string),
		}
		self.peers.Store(fullName, peer)
		if attrName == GRPC_KEY {
			// 新增节点
			log.Debugw("peer registry add peer", "full_name", fullName, GRPC_KEY, attrValue)
			peer.Address = attrValue
			peer.Url = formatPeerUrl(peer.FullName)
			if peer.FullName == r.appFullName {
				self.SelfPeer = peer
			}
			self.onPeerAdd(r, peer)
		} else {
			peer.Metadata[attrName] = attrValue
			log.Debugw("peer registry add peer attr", "full_name", fullName, attrName, attrValue)
		}
	}
	return nil
}

func (self *peer_registry) onKvDelete(r *Registry, kv *mvccpb.KeyValue) error {
	pats := strings.Split(string(kv.Key), "/")
	if len(pats) != 5 {
		log.Warnw("peer registry got a invalid peer", "key", string(kv.Key))
		return errors.New("invalid peer registry key", "key", string(kv.Key))
	}
	pats = pats[3:]
	fullName := pats[0]
	attrName := pats[1]
	if lastValue, ok := self.peers.Load(fullName); ok {
		lastPeer := lastValue.(*gira.Peer)
		if attrName == GRPC_KEY {
			//删除节点
			log.Warnw("peer registry remove peer", "full_name", fullName, GRPC_KEY, lastPeer.Address)
			lastPeer.Address = ""
			lastPeer.Url = ""
			self.onPeerDelete(r, lastPeer)
		} else {
			if lastAttrValue, ok := lastPeer.Metadata[attrName]; ok {
				delete(lastPeer.Metadata, attrName)
				log.Warnw("peer registry remove peer attr", "full_name", fullName, attrName, lastAttrValue)
			} else {
				log.Warnw("peer registry remove peer attr, but attr not found!!!!!", "full_name", fullName, attrName, "")
			}
		}
	} else {
		log.Warnw("peer registry remove peer, but peer not found", "full_name", fullName, "key", string(kv.Key))
	}
	return nil
}

// 只增加节点，但不通知handler, 等notify再通知
/*
func (self *peer_registry) onKvAdd(r *Registry, kv *mvccpb.KeyValue) error {
	pats := strings.Split(string(kv.Key), "/")
	if len(pats) != 4 {
		log.Warnw("peer registry got a invalid key", "key", string(kv.Key))
		return errors.ErrInvalidPeer
	}
	fullName := pats[2]
	attrName := pats[3]
	name, serverId, err := gira.ParseAppFullName(fullName)
	if err != nil {
		log.Warnw("peer registry got a invalid key", "full_name", fullName)
		return err
	}
	attrValue := string(kv.Value)
	if lastValue, ok := self.peers.Load(fullName); ok {
		lastPeer := lastValue.(*gira.Peer)
		if attrName == GRPC_KEY {
			if lastPeer.Address == "" {
				// 新增节点
				lastPeer.Address = attrValue
				lastPeer.Url = formatPeerUrl(lastPeer.FullName)
				if lastPeer.FullName == r.appFullName {
					self.SelfPeer = lastPeer
				}
				log.Infow("peer registry add peer", "full_name", fullName, GRPC_KEY, attrValue)
			} else if attrValue != lastPeer.Address {
				// 节点地址改变
				lastPeer.Address = attrValue
				log.Infow("peer registry update peer", "full_name", fullName, GRPC_KEY, attrValue, "last_attr_value", lastPeer.Address)
			} else {
				log.Infow("peer registry update peer", "full_name", fullName, GRPC_KEY, attrValue, "last_attr_value", lastPeer.Address)
			}
		} else {
			if lastAttrValue, ok := lastPeer.Kvs[attrName]; ok {
				// 更新属性
				if lastAttrValue != attrValue {
					log.Infow("peer registry update peer attr", "full_name", fullName, attrName, attrValue, "last_attr_value", lastAttrValue)
				} else {
					log.Infow("peer registry update peer attr", "full_name", fullName, attrName, attrValue, "last_attr_value", lastAttrValue)
				}
			} else {
				// 新增属性
				log.Infow("peer registry add peer attr", "full_name", fullName, attrName, attrValue)
			}
			lastPeer.Kvs[attrName] = attrValue
		}
	} else {
		peer := &gira.Peer{
			Id:       serverId,
			Name:     name,
			FullName: fullName,
			Kvs:      make(map[string]string),
		}
		self.peers.Store(fullName, peer)
		if attrName == GRPC_KEY {
			peer.Address = attrValue
			peer.Url = formatPeerUrl(peer.FullName)
			if peer.FullName == r.appFullName {
				self.SelfPeer = peer
			}
			// 新增节点
			log.Infow("peer registry add peer", "full_name", fullName, GRPC_KEY, attrValue)
		} else {
			peer.Kvs[attrName] = attrValue
			log.Infow("peer registry add peer attr", "full_name", fullName, attrName, attrValue)
		}
	}
	return nil
}
*/

func (self *peer_registry) initPeers(r *Registry) error {
	client := r.client
	kv := clientv3.NewKV(client)
	var getResp *clientv3.GetResponse
	var err error
	if getResp, err = kv.Get(self.ctx, self.prefix, clientv3.WithPrefix()); err != nil {
		return err
	}
	for _, kv := range getResp.Kvs {
		if err := self.onKvPut(r, kv); err != nil {
			return err
		}
	}
	self.watchStartRevision = getResp.Header.Revision + 1
	return nil
}

func (self *peer_registry) watchPeers(r *Registry) error {
	client := r.client
	watchStartRevision := self.watchStartRevision
	watcher := clientv3.NewWatcher(client)
	// r.application.Go(func() error {
	log.Debugw("peer registry watch peer started", "prefix", self.prefix, "watch_start_revision", watchStartRevision)
	watchRespChan := watcher.Watch(self.ctx, self.prefix, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix(), clientv3.WithPrevKV())
	for watchResp := range watchRespChan {
		// log.Info("etcd watch got events")
		for _, event := range watchResp.Events {
			switch event.Type {
			case mvccpb.PUT:
				// log.Info("etcd got put event")
				if err := self.onKvPut(r, event.Kv); err != nil {
					log.Warnw("peer registry put event fail", "error", err)
				}
			case mvccpb.DELETE:
				// log.Info("etcd got delete event")
				if err := self.onKvDelete(r, event.Kv); err != nil {
					log.Warnw("peer registry put event fail", "error", err)
				}
			}
		}
	}
	log.Debugw("peer registry watch exit")
	return nil
	// })
}

func (self *peer_registry) unregisterSelf(r *Registry) error {
	client := r.client
	kv := clientv3.NewKV(client)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	log.Debugw("peer registry unregister", "self_prefix", self.selfPrefix)
	self.isNormalUnregisterSelf = true
	// txn.If(clientv3.Compare(clientv3.Value(key), "!=", value), clientv3.Compare(clientv3.CreateRevision(key), "!=", 0))
	var txnResp *clientv3.TxnResponse
	var err error
	txn := kv.Txn(ctx)
	key := fmt.Sprintf("%s%s", self.selfPrefix, GRPC_KEY)
	value := r.config.Address

	txn.If(clientv3.Compare(clientv3.Value(key), "=", value), clientv3.Compare(clientv3.CreateRevision(key), "=", self.selfRevision)).
		Then(clientv3.OpDelete(self.selfPrefix, clientv3.WithPrefix())).
		Else(clientv3.OpGet(key))
	if txnResp, err = txn.Commit(); err != nil {
		log.Errorw("peer registry commit fail", "error", err)
		return err
	}
	if txnResp.Succeeded {
		// if len(txnResp.Responses[0].GetResponseRange().Kvs) == 0 {
		// key 没被任何程序占用，不用管，直接退出
		// log.Info("key 没被任何程序占用，不用管，直接退出")
		// log.Warnw("peer registry unregister fail")
		// self.cancelFunc()
		// } else {
		// 等监听到清理完成后再退出
		// log.Info("peer registry wait cleanup")
		// }
	} else {
		// key 被其他程序占用着，不用管，直接退出
		// log.Info(" key 被其他程序占用着，不用管，直接退出")
		// log.Warn("peer registry unregister fail", "locked_by", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		log.Warn("peer registry unregister fail", "locked_by", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		// self.cancelFunc()

	}
	return nil
}

func (self *peer_registry) registerSelf(r *Registry) error {
	client := r.client
	var err error
	var lease clientv3.Lease
	var leaseID clientv3.LeaseID
	if r.config.LeaseTimeout > 0 {
		// 申请一个租约 lease
		lease = clientv3.Lease(client)
		var leaseGrantResp *clientv3.LeaseGrantResponse
		// 申请一个5s的租约
		if leaseGrantResp, err = lease.Grant(self.ctx, 5); err != nil {
			return err
		}
		// 租约ID
		leaseID = leaseGrantResp.ID
	}
	// 需要同步的键值对
	advertises := make(map[string]string, 0)
	advertises[GRPC_KEY] = r.config.Address
	for _, v := range r.config.Advertise {
		advertises[v.Name] = v.Value
	}
	kv := clientv3.NewKV(client)
	for name, value := range advertises {
		var txnResp *clientv3.TxnResponse
		txn := kv.Txn(self.ctx)
		key := fmt.Sprintf("%s%s", self.selfPrefix, name)
		tx := txn.If(clientv3.Compare(clientv3.Value(key), "!=", value), clientv3.Compare(clientv3.CreateRevision(key), "!=", 0)).
			// tx := txn.If(clientv3.Compare(clientv3.CreateRevision(key), "!=", 0)).
			Then(clientv3.OpGet(key))
		if leaseID != 0 {
			tx.Else(clientv3.OpGet(key), clientv3.OpPut(key, value, clientv3.WithLease(leaseID)))
		} else {
			tx.Else(clientv3.OpGet(key), clientv3.OpPut(key, value))
		}
		if txnResp, err = txn.Commit(); err != nil {
			log.Errorw("peer registry commit fail", "error", err)
			return err
		}
		if txnResp.Succeeded {
			log.Errorw("etcd register fail", "key", key, "value", value, "locked_by", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
			return errors.New("peer already regist", "key", key)
		} else {
			if len(txnResp.Responses[0].GetResponseRange().Kvs) == 0 {
				if name == GRPC_KEY {
					self.selfRevision = txnResp.Responses[1].GetResponsePut().Header.Revision
				}
				log.Debugw("peer registry register peer", "key", key, "value", value)
			} else {
				if name == GRPC_KEY {
					self.selfRevision = txnResp.Responses[1].GetResponsePut().Header.Revision
				}
				log.Debugw("peer already register", "key", key, "value", value)
				// return errors.New("peer already regist", "key", key)
			}
		}
	}
	if leaseID != 0 {
		var keepRespChan <-chan *clientv3.LeaseKeepAliveResponse
		// 自动续租
		if keepRespChan, err = lease.KeepAlive(self.ctx, leaseID); err != nil {
			log.Errorw("peer registry lease keep alive fail", "error", err)
			return err
		}
		//判断续约应答的协程
		go func() error {
			for {
				select {
				case keepResp := <-keepRespChan:
					if keepRespChan == nil {
						log.Warn("peer registry lease expire")
						goto END
					} else if keepResp == nil {
						log.Warn("peer registry lease canceled")
						goto END
					} else {
						// KeepAlive每秒会续租一次,所以就会收到一次应答
						log.Warn("peer registry lease touch", keepResp.ID)
					}
				case <-r.ctx.Done():
					break
				}
			}
		END:
			return nil
		}()
	}
	return nil
}
