package registry

/// 参考:
/// - https://pkg.go.dev/go.etcd.io/etcd/client/v3#section-readme
/// - https://www.cnblogs.com/hsyw/p/16026627.html

/// watch的版本如果已经compact,则watch会返回一个compacted error,且管道会被关闭

/// 如果etcd server关闭后再恢复，watch goroutine可以正常运行

/// key设置超时的话，如果etcd变成不可用，所有的key就会超时被删除，等etcd变得可用时，需要重新注册全部的key.

/// key不设置超时的话, 非正常关闭, key会一直保留着，但事实上服务已经变得不可用。玩家也会被锁着，无法登录到其他服。如果挂了的服有能力恢复玩家的数据的话，
/// 锁着玩家登录其他服，就可以防止玩家的数据丢失。这种情况下可以增加一个心跳功能，使服务被选中的优先级降低。

///

import (
	"context"
	"fmt"
	"time"

	"github.com/Lyndon-Zhang/gira"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

const (
	GRPC_KEY string = "grpc"
)

type Registry struct {
	config      gira.EtcdConfig
	appId       int32
	appFullName string // 节点全名
	name        string // 节点名
	client      *clientv3.Client
	ctx         context.Context
	cancelFunc  context.CancelFunc

	localPlayerWatchHandlers []gira.LocalPlayerWatchHandler
	peerWatchHandlers        []gira.PeerWatchHandler
	serviceWatchHandlers     []gira.ServiceWatchHandler

	peerRegistry    *peer_registry
	playerRegistry  *player_registry
	serviceRegistry *service_registry
	errCtx          context.Context
	errGroup        *errgroup.Group
	isNotify        int32
	peerResolver    *peer_resolver_builder
}

// 关闭，释放资源
func (r *Registry) Stop() error {
	log.Debug("registry stop")
	r.playerRegistry.stop(r)
	r.serviceRegistry.stop(r)
	r.peerRegistry.stop(r)
	return nil
}

func (r *Registry) notify() {
	r.isNotify = 1
	r.peerRegistry.notify(r)
	r.playerRegistry.notify(r)
	r.serviceRegistry.notify(r)
}

// 作为其中一个节点来启动
// 会将自己注册进去
func (r *Registry) StartAsMember() error {
	// 注册自己
	if err := r.peerRegistry.registerSelf(r); err != nil {
		return err
	}
	if err := r.peerRegistry.initPeers(r); err != nil {
		return err
	}
	if err := r.playerRegistry.recoverSelfPeerPlayers(r); err != nil {
		return err
	}
	if err := r.serviceRegistry.initServices(r); err != nil {
		return err
	}
	return nil
}

func (r *Registry) Watch(peerWatchHandlers []gira.PeerWatchHandler, localPlayerWatchHandlers []gira.LocalPlayerWatchHandler, serviceWatchHandlers []gira.ServiceWatchHandler) error {
	r.peerWatchHandlers = peerWatchHandlers
	r.localPlayerWatchHandlers = localPlayerWatchHandlers
	r.serviceWatchHandlers = serviceWatchHandlers
	r.errGroup, r.errCtx = errgroup.WithContext(r.ctx)
	r.errGroup.Go(func() error {
		// return r.peerRegistry.Serve(r)
		// 侦听伙伴信息
		if err := r.peerRegistry.watchPeers(r); err != nil {
			return err
		}
		return nil
	})
	// r.errGroup.Go(func() error {
	// return r.playerRegistry.Serve(r)
	// return r.playerRegistry.watchSelfPeerPlayers(r)
	// })
	r.errGroup.Go(func() error {
		// return r.serviceRegistry.Serve(r)
		return r.serviceRegistry.watchServices(r)
	})
	r.notify()
	return r.errGroup.Wait()
}

func (r *Registry) RangePeers(f func(k any, v any) bool) {
	r.peerRegistry.RangePeers(f)
}

// 根据app名查找节点
// 协程安全
func (r *Registry) GetPeer(fullName string) *gira.Peer {
	return r.peerRegistry.getPeer(r, fullName)
}

func (r *Registry) SelfPeer() *gira.Peer {
	return r.peerRegistry.SelfPeer
}

func NewConfigRegistry(ctx context.Context, config *gira.EtcdConfig) (*Registry, error) {
	r := &Registry{
		config:      *config,
		appFullName: facade.GetAppFullName(),
		appId:       facade.GetAppId(),
		name:        facade.GetAppType(),
	}
	r.ctx, r.cancelFunc = context.WithCancel(ctx)
	// 配置endpoints
	endpoints := make([]string, 0)
	for _, v := range r.config.Endpoints {
		endpoints = append(endpoints, fmt.Sprintf("http://%s:%d", v.Host, v.Port))
	}
	var client *clientv3.Client
	var err error
	c := clientv3.Config{
		Endpoints:   endpoints,                                         // 节点信息
		DialTimeout: time.Duration(r.config.DialTimeout) * time.Second, // 超时时间
		DialOptions: []grpc.DialOption{grpc.WithBlock()},               // 使用阻塞模式，确认启动时etcd是可用的
		Username:    r.config.Username,
		Password:    r.config.Password,
		Context:     r.ctx,
	}
	// 建立连接
	if client, err = clientv3.New(c); err != nil {
		log.Errorw("connect to etcd fail", "error", err)
		return nil, err
	}
	r.client = client
	log.Debugw("connect registry success", "endpoints", endpoints)
	if v, err := newConfigPeerRegistry(r); err != nil {
		return nil, err
	} else {
		r.peerRegistry = v
	}
	if v, err := newConfigPlayerRegistry(r); err != nil {
		return nil, err
	} else {
		r.playerRegistry = v
	}
	if v, err := newConfigServiceRegistry(r); err != nil {
		return nil, err
	} else {
		r.serviceRegistry = v
	}
	return r, nil
}

func (r *Registry) ListLocalUser() []string {
	return r.playerRegistry.ListLocalUser(r)
}

// 锁定玩家
func (r *Registry) LockLocalUser(userId string) (*gira.Peer, error) {
	return r.playerRegistry.LockLocalUser(r, userId)
}

// 解锁玩家
func (r *Registry) UnlockLocalUser(userId string) (*gira.Peer, error) {
	return r.playerRegistry.UnlockLocalUser(r, userId)
}

// 查找玩家
func (r *Registry) WhereIsUser(userId string) (*gira.Peer, error) {
	return r.playerRegistry.WhereIsUser(r, userId)
}

// 查找节点
func (r *Registry) WhereIsPeer(appFullName string) (*gira.Peer, error) {
	if p := r.peerRegistry.getPeer(r, appFullName); p != nil {
		return p, nil
	} else {
		return nil, errors.ErrPeerNotFound
	}
}

// 查找服务
func (r *Registry) WhereIsService(serviceName string, opt ...service_options.WhereOption) ([]*gira.Peer, error) {
	return r.serviceRegistry.WhereIsService(r, serviceName, opt...)
}

func (r *Registry) NewServiceName(serviceName string, opt ...service_options.RegisterOption) string {
	return r.serviceRegistry.NewServiceName(r, serviceName, opt...)
}

// 注册服务
func (r *Registry) RegisterService(serviceName string, opt ...service_options.RegisterOption) (*gira.Peer, error) {
	return r.serviceRegistry.RegisterService(r, serviceName, opt...)
}

// 反注册服务
func (r *Registry) UnregisterService(serviceName string) (*gira.Peer, error) {
	return r.serviceRegistry.UnregisterService(r, serviceName)
}

// 开启resolver
func (r *Registry) StartReslover() error {
	r.peerResolver = &peer_resolver_builder{
		r: r,
	}
	resolver.Register(r.peerResolver)
	return nil
}
