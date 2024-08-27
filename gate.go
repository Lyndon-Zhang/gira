package gira

import "context"

// 服务端conn接口
type GatewayConn interface {
	Id() uint64
	Close() error
	Kick(reason string)
	SendServerSuspend(reason string)
	SendServerResume(reason string)
	SendServerMaintain(reason string)
	SendServerDown(reason string)
	Recv(ctx context.Context) (GatewayMessage, error)
	Push(route string, data []byte) error
	Response(mid uint64, data []byte) error
	Value(key string) interface{}
	Set(key string, value interface{})
	Uint64(key string) uint64
	Int32(key string) int32
	SetUserData(value interface{})
	UserData() interface{}
}

// client端接口
type GatewayClient interface {
	Recv(ctx context.Context) (typ int, route string, reqId uint64, data []byte, err error)
	Notify(route string, data []byte) error
	Request(route string, reqId uint64, data []byte) error
	Close() error
}

type GatewayHandler interface {
	ServeClientStream(conn GatewayConn)
}

type GatewayMessage interface {
	Response(data []byte) error
	Payload() []byte
	ReqId() uint64
	Session() GatewayConn
	Push(route string, data []byte) error
}

const (
	GatewayMessageType_RESPONSE = 2
	GatewayMessageType_PUSH     = 3
)
