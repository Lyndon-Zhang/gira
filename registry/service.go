package registry

///
///
///
/// 注册表结构:
///   /unique_service/<<ServiceName>> => <<AppFullName>>
///   /service/<<ServiceGroupName>>/<<ServiceName>> => <<AppFullName>>
///   /peer_service/<<AppFullName>>/<<ServiceUniqueName>> => <<AppFullName>>
///
import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Lyndon-Zhang/gira"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type word_trie struct {
	mu      sync.Mutex
	headers map[string]*word_trie_header_node
}

func newWordTrie() *word_trie {
	return &word_trie{
		headers: make(map[string]*word_trie_header_node),
	}
}

type word_trie_header_node struct {
	mu    sync.Mutex
	value string
	path  string
	set   bool
	nodes map[string]*word_trie_node
}

type word_trie_node struct {
	value string
	path  string
	set   bool
	nodes map[string]*word_trie_node
}

func (trie *word_trie) debugTrace() {
	trie.mu.Lock()
	defer trie.mu.Unlock()
	for _, header := range trie.headers {
		header.debugTrace()
	}
}

func (trie *word_trie) add(path string) error {
	trie.mu.Lock()
	words := strings.Split(path, "/")
	// /前面的空格去掉
	if len(words) > 0 && len(words[0]) <= 0 {
		words = words[1:]
	}
	if len(words) <= 0 {
		trie.mu.Unlock()
		return nil
	}
	// 不能有空格
	for i := 1; i < len(words); i++ {
		if len(words[i]) <= 0 {
			trie.mu.Unlock()
			return errors.ErrInvalidArgs
		}
	}
	if header, ok := trie.headers[words[0]]; !ok {
		header := &word_trie_header_node{
			value: words[0],
			nodes: make(map[string]*word_trie_node),
		}
		trie.headers[words[0]] = header
		trie.mu.Unlock()
		return header.add(path, words[1:])
	} else {
		trie.mu.Unlock()
		return header.add(path, words[1:])
	}
}

func (trie *word_trie) delete(path string) error {
	trie.mu.Lock()
	words := strings.Split(path, "/")
	// /前面的空格去掉
	if len(words) > 0 && len(words[0]) <= 0 {
		words = words[1:]
	}
	if len(words) <= 0 {
		trie.mu.Unlock()
		return errors.ErrTODO.Trace()
	}
	if header, ok := trie.headers[words[0]]; !ok {
		trie.mu.Unlock()
		return errors.ErrTODO.Trace()
	} else {
		trie.mu.Unlock()
		if err := header.delete(words[1:]); err != nil {
			return err
		} else {
			if len(header.nodes) <= 0 && !header.set {
				delete(trie.headers, words[0])
			}
			return nil
		}
	}
}

func (wt *word_trie) search(path string) (matches []string) {
	wt.mu.Lock()
	words := strings.Split(path, "/")
	// /前面的空格去掉
	if len(words) > 0 && len(words[0]) <= 0 {
		words = words[1:]
	}
	if len(words) <= 0 {
		wt.mu.Unlock()
		return
	}
	if len(words[0]) <= 0 {
		for _, header := range wt.headers {
			matches = header.collect(matches)
		}
		wt.mu.Unlock()
		return
	} else if header, ok := wt.headers[words[0]]; !ok {
		wt.mu.Unlock()
		return
	} else {
		wt.mu.Unlock()
		matches = header.search(words[1:], matches)
		return
	}
}

func (header *word_trie_header_node) collect(result []string) []string {
	if header.set {
		result = append(result, header.path)
	}
	for _, c := range header.nodes {
		result = c.collect(result)
	}
	return result
}

func (header *word_trie_header_node) debugTrace() {
	header.mu.Lock()
	defer header.mu.Unlock()
	log.Debugw("header", "value", header.value, "path", header.path, "set", header.set)
	for _, node := range header.nodes {
		node.debugTrace()
	}
}

func (header *word_trie_header_node) add(path string, words []string) error {
	header.mu.Lock()
	defer header.mu.Unlock()
	nodes := header.nodes
	if len(words) <= 0 {
		header.path = path
		header.set = true
		return nil
	} else {
		var lastNode *word_trie_node
		var ok bool
		for _, v := range words {
			if lastNode, ok = nodes[v]; !ok {
				lastNode = &word_trie_node{
					value: v,
					nodes: make(map[string]*word_trie_node),
				}
				nodes[v] = lastNode
				nodes = lastNode.nodes
			} else {
				nodes = lastNode.nodes
			}
		}
		lastNode.path = path
		lastNode.set = true
		return nil
	}
}

func (header *word_trie_header_node) delete(words []string) error {
	header.mu.Lock()
	defer header.mu.Unlock()
	if len(words) <= 0 {
		header.set = false
		return nil
	} else {
		if c, ok := header.nodes[words[0]]; !ok {
			return nil
		} else {
			c.delete(words[1:])
			if len(c.nodes) <= 0 && !c.set {
				delete(header.nodes, words[0])
			}
			return nil
		}
	}
}
func (header *word_trie_header_node) search(words []string, matches []string) []string {
	header.mu.Lock()
	defer header.mu.Unlock()
	var lastNode *word_trie_node
	var ok bool
	nodes := header.nodes
	for i := 0; i < len(words); i++ {
		v := words[i]
		// 忽略/
		if i == len(words)-1 && len(v) <= 0 {
			break
		}
		if lastNode, ok = nodes[v]; !ok {
			return nil
		} else {
			nodes = lastNode.nodes
		}
	}
	for _, c := range nodes {
		matches = c.collect(matches)
	}
	return matches
}

func (node *word_trie_node) delete(words []string) error {
	if len(words) <= 0 {
		node.set = false
		return nil
	} else {
		if c, ok := node.nodes[words[0]]; !ok {
			return errors.ErrTODO.Trace()
		} else {
			if err := c.delete(words[1:]); err != nil {
				return err
			}
			if len(c.nodes) <= 0 && !c.set {
				delete(node.nodes, words[0])
			}
			return nil
		}
	}
}

func (node *word_trie_node) debugTrace() {
	log.Debugw("node", "value", node.value, "path", node.path, "set", node.set)
	for _, node := range node.nodes {
		node.debugTrace()
	}
}

func (node *word_trie_node) collect(result []string) []string {
	if node.set {
		result = append(result, node.path)
	}
	for _, c := range node.nodes {
		result = c.collect(result)
	}
	return result
}

type service_registry struct {
	peerServicePrefix  string   // /peer/service/<<AppFullName>><<ServiceName>>/			根据服务全名查找全部服务
	servicePrefix      string   // /service/<<ServiceName>>/      							可以根据服务名查找当前所在的服
	services           sync.Map // 全部service
	selfServices       sync.Map // 本节点上的service
	ctx                context.Context
	cancelFunc         context.CancelFunc
	watchStartRevision int64
	prefixIndex        *word_trie
}

func newConfigServiceRegistry(r *Registry) (*service_registry, error) {
	ctx, cancelFunc := context.WithCancel(r.ctx)
	self := &service_registry{
		prefixIndex:       newWordTrie(),
		servicePrefix:     "/service/",
		peerServicePrefix: fmt.Sprintf("/peer/service/%s/", r.appFullName),
		ctx:               ctx,
		cancelFunc:        cancelFunc,
	}
	return self, nil
}

func (self *service_registry) stop(r *Registry) error {
	log.Debug("service registry on stop")
	if err := self.unregisterServices(r); err != nil {
		log.Info(err)
	}
	return nil
}

func (self *service_registry) notify(r *Registry) error {
	self.services.Range(func(k any, v any) bool {
		service := v.(*gira.ServiceName)
		self.onServiceAdd(r, service)
		return true
	})
	return nil
}

// on service add callback
func (self *service_registry) onServiceAdd(r *Registry, service *gira.ServiceName) {
	if r.isNotify == 0 {
		return
	}
	log.Debugw("service registry on service add", "service_full_name", service.ServiceFullName, "peer", service.Peer.FullName)
	for _, handler := range r.serviceWatchHandlers {
		handler.OnServiceAdd(service)
	}
}

// on service delete callback
func (self *service_registry) onServiceDelete(r *Registry, service *gira.ServiceName) {
	log.Debugw("service registry on service delete", "service_full_name", service.ServiceFullName, "peer", service.Peer.FullName)
	for _, handler := range r.serviceWatchHandlers {
		handler.OnServiceDelete(service)
	}
}

func (self *service_registry) onKvAdd(r *Registry, kv *mvccpb.KeyValue) error {
	words := strings.Split(string(kv.Key), "/")
	var serviceTypeName string
	var serviceFullName string
	if len(words) <= 2 {
		log.Warnw("service registry got a invalid key", "key", string(kv.Key))
		return errors.ErrInvalidService
	}
	words = words[2:]
	if len(words) == 2 {
		serviceFullName = fmt.Sprintf("%s/%s", words[0], words[1])
		serviceTypeName = words[0]
	} else if len(words) == 1 {
		serviceFullName = words[0]
	} else {
		log.Warnw("service registry got a invalid key", "key", string(kv.Key))
		return errors.ErrInvalidService
	}
	value := string(kv.Value)
	if _, ok := self.services.Load(serviceFullName); ok {
		// lastService := lastValue.(*gira.ServiceName)
		// log.Warnw("service registry on kv add, but already exist", "service_name", serviceName, "peer", value, "last_peer", lastService.Peer.FullName)
	} else {
		// 新增service
		// log.Infow("service registry on kv add", "service_name", serviceName, "peer", value)
		peer := r.GetPeer(value)
		if peer == nil {
			log.Warnw("service registry on kv add, but peer not found", "service_full_name", serviceFullName, "peer", value)
			return errors.ErrPeerNotFound
		}
		service := &gira.ServiceName{
			ServiceFullName: serviceFullName,
			ServiceTypeName: serviceTypeName,
			Peer:            peer,
		}
		if peer == r.SelfPeer() {
			service.IsSelf = true
		}
		self.services.LoadOrStore(serviceFullName, service)
		self.prefixIndex.add(serviceFullName)
		if service.IsSelf {
			self.selfServices.Store(serviceFullName, service)
		}
		self.onServiceAdd(r, service)
	}
	return nil
}

func (self *service_registry) onKvDelete(r *Registry, kv *mvccpb.KeyValue) error {
	words := strings.Split(string(kv.Key), "/")
	var serviceFullName string
	if len(words) <= 2 {
		log.Warnw("service registry got a invalid key", "key", string(kv.Key))
		return errors.ErrInvalidService
	}
	words = words[2:]
	if len(words) == 2 {
		serviceFullName = fmt.Sprintf("%s/%s", words[0], words[1])
	} else if len(words) == 1 {
		serviceFullName = words[0]
	}
	// value := string(kv.Value) value没有值
	if lastValue, ok := self.services.Load(serviceFullName); ok {
		lastService := lastValue.(*gira.ServiceName)
		log.Debugw("service registry remove service", "service_full_name", serviceFullName, "last_peer", lastService.Peer.FullName)
		self.services.Delete(serviceFullName)
		self.prefixIndex.delete(strings.Join(words, "/"))
		self.onServiceDelete(r, lastService)
		if lastService.IsSelf {
			self.selfServices.Delete(serviceFullName)
		}
	} else {
		log.Warnw("service registry remove service, but service not found", "service_full_name", serviceFullName)
	}
	return nil
}

func (self *service_registry) initServices(r *Registry) error {
	client := r.client
	kv := clientv3.NewKV(client)
	var getResp *clientv3.GetResponse
	var err error
	// 删除自身之前注册，没清理干净的服务
	if getResp, err = kv.Get(self.ctx, self.peerServicePrefix, clientv3.WithPrefix()); err != nil {
		return err
	}
	for _, v := range getResp.Kvs {
		words := strings.Split(string(v.Key), "/")
		var serviceName string
		if len(words) == 6 {
			serviceName = fmt.Sprintf("%s/%s", words[4], words[5])
		} else if len(words) == 5 {
			serviceName = words[4]
		}
		txn := kv.Txn(self.ctx)
		serviceKey := fmt.Sprintf("%s%s", self.servicePrefix, serviceName)
		peerKey := fmt.Sprintf("%s%s", self.peerServicePrefix, serviceName)
		txn.If(clientv3.Compare(clientv3.CreateRevision(serviceKey), "!=", 0)).
			Then(clientv3.OpDelete(peerKey), clientv3.OpDelete(serviceKey))
		var txnResp *clientv3.TxnResponse
		if txnResp, err = txn.Commit(); err != nil {
			log.Errorw("service registry commit fail", "error", err)
			return err
		}
		if txnResp.Succeeded {
			log.Debugw("service registry unregister", "service_key", serviceKey, "peer_key", peerKey, "service_name", serviceName)
		} else {
			log.Warnw("service registry unregister", "service_key", serviceKey, "peer_key", peerKey, "service_name", serviceName)
		}
	}
	// 初始化服务
	if getResp, err = kv.Get(self.ctx, self.servicePrefix, clientv3.WithPrefix()); err != nil {
		return err
	}
	for _, kv := range getResp.Kvs {
		if err := self.onKvAdd(r, kv); err != nil {
			return err
		}
	}
	self.watchStartRevision = getResp.Header.Revision + 1
	return nil
}

func (self *service_registry) watchServices(r *Registry) error {
	client := r.client
	watchStartRevision := self.watchStartRevision
	watcher := clientv3.NewWatcher(client)
	// r.application.Go(func() error {
	watchRespChan := watcher.Watch(self.ctx, self.servicePrefix, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix(), clientv3.WithPrevKV())
	log.Debugw("service registry started", "service_prefix", self.servicePrefix, "watch_start_revision", watchStartRevision)
	for watchResp := range watchRespChan {
		// log.Info("etcd watch got events")
		for _, event := range watchResp.Events {
			switch event.Type {
			case mvccpb.PUT:
				// log.Info("etcd got put event")
				if err := self.onKvAdd(r, event.Kv); err != nil {
					log.Warnw("service registry put event fail", "error", err)
				}
			case mvccpb.DELETE:
				// log.Info("etcd got delete event")
				if err := self.onKvDelete(r, event.Kv); err != nil {
					log.Warnw("service registry put event fail", "error", err)
				}
			}
		}
	}
	log.Debug("service registry watch exit")
	return nil
	// })
}

func (self *service_registry) unregisterServices(r *Registry) error {
	client := r.client
	kv := clientv3.NewKV(client)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	log.Debugw("service registry unregister", "peer_prefix", self.peerServicePrefix)

	var txnResp *clientv3.TxnResponse
	var err error
	self.selfServices.Range(func(serviceName any, v any) bool {
		txn := kv.Txn(ctx)
		serviceKey := fmt.Sprintf("%s%s", self.servicePrefix, serviceName)
		peerKey := fmt.Sprintf("%s%s", self.peerServicePrefix, serviceName)
		service := v.(*gira.ServiceName)
		txn.If(clientv3.Compare(clientv3.CreateRevision(serviceKey), "=", service.CreateRevision)).
			Then(clientv3.OpDelete(serviceKey), clientv3.OpDelete(peerKey))
		if txnResp, err = txn.Commit(); err != nil {
			log.Errorw("service registry commit fail", "error", err)
			return true
		}
		if txnResp.Succeeded {
			log.Debugw("service registry unregister", "peer_key", peerKey, "create_revision", service.CreateRevision)
		} else {
			log.Warnw("service registry unregister", "peer_key", peerKey, "create_revision", service.CreateRevision)
		}
		return true
	})
	return nil
}

func (self *service_registry) NewServiceName(r *Registry, serviceName string, opt ...service_options.RegisterOption) string {
	opts := service_options.RegisterOptions{}
	for _, v := range opt {
		v.ConfigRegisterOption(&opts)
	}
	if opts.AsAppService {
		serviceName = fmt.Sprintf("%s/%d", serviceName, r.appId)
	}
	return serviceName
}

// 注册服务
func (self *service_registry) RegisterService(r *Registry, serviceFullName string, opt ...service_options.RegisterOption) (*gira.Peer, error) {
	serviceFullName = self.NewServiceName(r, serviceFullName, opt...)
	client := r.client
	serviceKey := fmt.Sprintf("%s%s", self.servicePrefix, serviceFullName)
	peerKey := fmt.Sprintf("%s%s", self.peerServicePrefix, serviceFullName)
	kv := clientv3.NewKV(client)
	var err error
	var txnResp *clientv3.TxnResponse
	txn := kv.Txn(self.ctx)
	// log.Debugw("service registry register", "service_name", serviceName, "peer_key", peerKey, "service_key", serviceKey)
	log.Debugw("service registry register", "service_key", serviceKey)
	value := r.appFullName

	var serviceTypeName string
	words := strings.Split(serviceFullName, "/")
	if len(words) == 2 {
		serviceFullName = fmt.Sprintf("%s/%s", words[0], words[1])
		serviceTypeName = words[0]
	} else if len(words) == 1 {
	} else {
		return nil, errors.ErrInvalidService
	}
	txn.If(clientv3.Compare(clientv3.CreateRevision(serviceKey), "=", 0)).
		Then(clientv3.OpPut(serviceKey, value), clientv3.OpPut(peerKey, value)).
		Else(clientv3.OpGet(serviceKey))
	if txnResp, err = txn.Commit(); err != nil {
		log.Errorw("service registry commit fail", "error", err)
		return nil, err
	}
	if txnResp.Succeeded {
		createRevision := txnResp.Responses[1].GetResponsePut().Header.Revision
		peer := r.GetPeer(value)
		service := &gira.ServiceName{
			IsSelf:          true,
			ServiceFullName: serviceFullName,
			ServiceTypeName: serviceTypeName,
			Peer:            peer,
			CreateRevision:  createRevision,
		}
		self.services.LoadOrStore(serviceFullName, service)
		self.prefixIndex.add(serviceFullName)
		self.selfServices.Store(serviceFullName, service)
		self.onServiceAdd(r, service)
		return nil, nil
	} else {
		log.Warnw("service registry register fail", "service_name", serviceFullName, "locked_by", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		appFullName := string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value)
		peer := r.GetPeer(appFullName)
		if peer == nil {
			return nil, errors.ErrServiceLocked
		}
		return peer, errors.ErrServiceLocked
	}
}

// 解锁服务
func (self *service_registry) UnregisterService(r *Registry, serviceName string) (*gira.Peer, error) {
	client := r.client
	serviceKey := fmt.Sprintf("%s%s", self.servicePrefix, serviceName)
	peerKey := fmt.Sprintf("%s%s", self.peerServicePrefix, serviceName)
	kv := clientv3.NewKV(client)
	var err error
	var txnResp *clientv3.TxnResponse
	txn := kv.Txn(self.ctx)
	log.Debugw("service registry", "peer_key", peerKey, "service_key", serviceKey)
	txn.If(clientv3.Compare(clientv3.Value(serviceKey), "=", r.appFullName), clientv3.Compare(clientv3.CreateRevision(serviceKey), "!=", 0)).
		Then(clientv3.OpDelete(peerKey), clientv3.OpDelete(serviceKey)).
		Else(clientv3.OpGet(serviceKey))
	if txnResp, err = txn.Commit(); err != nil {
		log.Errorw("service registry commit fail", "error", err)
		return nil, err
	}
	if txnResp.Succeeded {
		log.Debugw("service registry unregister", "service_name", serviceName)
		return nil, nil
	} else {
		var appFullName string
		if len(txnResp.Responses) > 0 && len(txnResp.Responses[0].GetResponseRange().Kvs) > 0 {
			appFullName = string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value)
		}
		log.Warnw("service registry unregister fail", "service_name", serviceName, "locked_by", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		peer := r.GetPeer(appFullName)
		if peer == nil {
			return nil, errors.ErrServiceLocked
		}
		return peer, errors.ErrServiceLocked
	}
}

// 查找服务位置
func (self *service_registry) WhereIsService(r *Registry, serviceName string, opt ...service_options.WhereOption) (peers []*gira.Peer, err error) {
	opts := service_options.WhereOptions{}
	for _, v := range opt {
		v.ConfigWhereOption(&opts)
	}
	if opts.Catalog || opts.Prefix {
		arr := self.prefixIndex.search(serviceName)
		peers = make([]*gira.Peer, 0)
		multicastCount := opts.MaxCount
		for _, name := range arr {
			if value, ok := self.services.Load(name); ok {
				service := value.(*gira.ServiceName)
				peers = append(peers, service.Peer)
				// 多播指定数量
				if multicastCount > 0 {
					multicastCount--
					if multicastCount <= 0 {
						break
					}
				}
			}
		}
		return
	} else {
		peers = make([]*gira.Peer, 0)
		if value, ok := self.services.Load(serviceName); ok {
			service := value.(*gira.ServiceName)
			peers = append(peers, service.Peer)
		}
		return
	}
}
