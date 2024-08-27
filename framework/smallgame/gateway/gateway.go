package gateway

import (
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gateway/config"
)

type LoginRequest interface {
	GetMemberId() string
	GetToken() string
}

// 需要实现的接口
type GatewayHandler interface {
}

type GatewayFramework interface {
	gira.Framework
	// 配置
	GetConfig() *config.GatewayConfig
	// 当前会话的数量
	SessionCount() int64
	// 当前连接的数量
	ConnectionCount() int64
}
