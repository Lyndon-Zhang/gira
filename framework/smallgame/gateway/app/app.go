package app

import (
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gateway"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gateway/config"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gateway/server"
)

type Framework struct {
	server *server.Server
	config *config.Config
	proto  gira.Proto // 使用的协议，当中必须包括名为Login的协议
}

func NewFramework(proto gira.Proto) gateway.GatewayFramework {
	return &Framework{
		proto: proto,
	}
}

// region implement gira.Framework
func (framework *Framework) OnFrameworkConfigLoad(c *gira.Config) error {
	framework.config = &config.Config{}
	return framework.config.OnConfigLoad(c)
}

func (framework *Framework) OnFrameworkCreate() error {
	framework.server = server.NewServer(framework.proto)
	if err := framework.server.OnCreate(); err != nil {
		return err
	}
	return nil
}

func (framework *Framework) OnFrameworkStart() error {
	return nil
}

func (framework *Framework) OnFrameworkStop() error {
	return nil
}

// endregion

// region implement gira.GateHandler
func (framework *Framework) ServeClientStream(conn gira.GatewayConn) {
	framework.server.ServeClientStream(conn)
}

// endregion

func (framework *Framework) OnPeerAdd(peer *gira.Peer) {
	framework.server.OnPeerAdd(peer)
}

func (framework *Framework) OnPeerDelete(peer *gira.Peer) {
	framework.server.OnPeerDelete(peer)
}

func (framework *Framework) OnPeerUpdate(peer *gira.Peer) {
	framework.server.OnPeerUpdate(peer)
}

// region implement gateway.GatewayFramework
// 当前会话的数量
func (framework *Framework) SessionCount() int64 {
	return framework.server.SessionCount
}

// 当前连接的数量
func (framework *Framework) ConnectionCount() int64 {
	return framework.server.ConnectionCount
}

func (framework *Framework) GetConfig() *config.GatewayConfig {
	return &framework.config.Framework.Gateway
}

// endregion
