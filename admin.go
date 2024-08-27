package gira

import (
	"context"
)

type AdminService interface {
	ResourceReload() error
}

type BroadcastReloadResourceResult interface {
	SuccessPeer(index int) *Peer
	ErrorPeer(index int) *Peer
	PeerCount() int
	SuccessCount() int
	ErrorCount() int
	Errors(index int) error
	Error() error
}

type AdminClient interface {
	// 广播所有节点重载资源
	BroadcastReloadResource(ctx context.Context, name string) (result BroadcastReloadResourceResult, err error)
}
