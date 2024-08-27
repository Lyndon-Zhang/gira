package registryclient

import (
	"context"
	"fmt"
	"time"

	"github.com/Lyndon-Zhang/gira"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

const GRPC_KEY string = "grpc"

type RegistryClient struct {
	config      gira.EtcdClientConfig
	appId       int32
	appFullName string // 节点全名
	client      *clientv3.Client
	ctx         context.Context
	cancelFunc  context.CancelFunc

	peerRegistry    *peer_registry
	playerRegistry  *player_registry
	serviceRegistry *service_registry
}

func (r *RegistryClient) StartAsClient() error {
	return nil
}

func NewConfigRegistryClient(ctx context.Context, config *gira.EtcdClientConfig, appId int32, appFullName string) (*RegistryClient, error) {
	r := &RegistryClient{
		config:      *config,
		appFullName: appFullName,
		appId:       appId,
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

// 查找玩家
func (r *RegistryClient) WhereIsUser(userId string) (*gira.Peer, error) {
	return r.playerRegistry.WhereIsUser(r, userId)
}

// 查找服务
func (r *RegistryClient) WhereIsService(serviceName string, opt ...service_options.WhereOption) ([]*gira.Peer, error) {
	return r.serviceRegistry.WhereIsService(r, serviceName, opt...)
}

func (r *RegistryClient) NewServiceName(serviceName string, opt ...service_options.RegisterOption) string {
	return r.serviceRegistry.NewServiceName(r, serviceName, opt...)
}

// 查找节点位置
func (r *RegistryClient) GetPeer(fullName string) *gira.Peer {
	return r.peerRegistry.GetPeer(r, fullName)
}

func (r *RegistryClient) UnregisterPeer(appFullName string) error {
	return r.peerRegistry.UnregisterPeer(r, appFullName)
}

func (r *RegistryClient) ListPeerKvs() (peers map[string]string, err error) {
	peers, err = r.peerRegistry.ListPeerKvs(r)
	return
}

func (r *RegistryClient) ListServiceKvs() (services map[string][]string, err error) {
	services, err = r.serviceRegistry.ListServiceKvs(r)
	return
}

// 查找节点
func (r *RegistryClient) WhereIsPeer(appFullName string) (*gira.Peer, error) {
	if p := r.peerRegistry.GetPeer(r, appFullName); p != nil {
		return p, nil
	} else {
		return nil, errors.ErrPeerNotFound
	}
}
