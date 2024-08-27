package gira

import (
	service_options "github.com/Lyndon-Zhang/gira/options/service_options"
)

type Registry interface {
	// 如果失败，则返回当前所在的节点
	LockLocalUser(userId string) (*Peer, error)
	UnlockLocalUser(userId string) (*Peer, error)
	ListLocalUser() []string
	WhereIsUser(userId string) (*Peer, error)
	RangePeers(f func(k any, v any) bool)
	NewServiceName(serviceName string, opt ...service_options.RegisterOption) string
	// 注册服务
	RegisterService(serviceName string, opt ...service_options.RegisterOption) (*Peer, error)
	// 反注册服务
	UnregisterService(serviceName string) (*Peer, error)
	// 查找服务
	WhereIsService(serviceName string, opt ...service_options.WhereOption) ([]*Peer, error)
	// 查找节点
	WhereIsPeer(appFullName string) (*Peer, error)
	// 自身节点
	SelfPeer() *Peer
}

type RegistryClient interface {
	NewServiceName(serviceName string, opt ...service_options.RegisterOption) string
	WhereIsUser(userId string) (*Peer, error)
	// 查找服务
	WhereIsService(serviceName string, opt ...service_options.WhereOption) ([]*Peer, error)
	UnregisterPeer(appFullName string) error
	ListPeerKvs() (peers map[string]string, err error)
	ListServiceKvs() (services map[string][]string, err error)
	// 查找节点
	WhereIsPeer(appFullName string) (*Peer, error)
}

// 伙伴节点
type Peer struct {
	Name     string // 服务类型
	Id       int32  // 服务id
	FullName string // 服务全名
	Address  string // grpc地址
	Url      string
	Metadata map[string]string // /server/account_1/ 下的键
}

// 玩家位置
type LocalPlayer struct {
	UserId         string
	LoginTime      int64
	CreateRevision int64
}

// 服务名
type ServiceName struct {
	// <<GroupName>>/<<ShortName>>
	Peer            *Peer
	ServiceFullName string
	ServiceTypeName string
	IsSelf          bool
	CreateRevision  int64
}

// 侦听伙伴节点
type PeerWatchHandler interface {
	OnPeerAdd(peer *Peer)
	OnPeerDelete(peer *Peer)
	OnPeerUpdate(peer *Peer)
}

// 侦听玩家位置
type LocalPlayerWatchHandler interface {
	OnLocalPlayerAdd(player *LocalPlayer)
	OnLocalPlayerDelete(player *LocalPlayer)
	OnLocalPlayerUpdate(player *LocalPlayer)
}

// 侦听服务状态
type ServiceWatchHandler interface {
	OnServiceAdd(service *ServiceName)
	OnServiceDelete(service *ServiceName)
	OnServiceUpdate(service *ServiceName)
}
