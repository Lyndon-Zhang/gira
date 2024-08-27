package gira

import "context"

// 协议接口
// 包括消息的编码，解码和消息路由功能

type ProtoRequest interface {
	GetRequestName() string
}

type ProtoPush interface {
	GetPushName() string
}

type ProtoResponse interface {
	SetErrorCode(v int32)
	SetErrorMsg(v string)
	SetDebugMsg(v string)
}

type ProtoHandler interface {
	HasRoute(route string) bool
	// 将push路由到handler的相应方法
	PushDispatch(ctx context.Context, receiver interface{}, route string, r interface{}) (err error)
	// 将request路由到handler的相应方法
	RequestDispatch(ctx context.Context, receiver interface{}, route string, r interface{}) (resp interface{}, push []ProtoPush, err error)
}

// 协议
type Proto interface {

	// request解码
	RequestEncode(name string, session int32, req interface{}) (data []byte, err error)
	// response编码
	ResponseEncode(name string, session int32, response interface{}) (data []byte, err error)
	// push解码
	PushEncode(req ProtoPush) (data []byte, err error)
	// struct编码
	StructEncode(req interface{}) (data []byte, err error)

	// request解码
	RequestDecode(packed []byte) (route string, session int32, resp ProtoRequest, err error)
	// response解码
	ResponseDecode(packed []byte) (route string, session int32, resp ProtoResponse, err error)
	// push解码
	PushDecode(packed []byte) (route string, session int32, resp ProtoPush, err error)
	// struct编码
	StructDecode(data []byte, req interface{}) error

	// 创建response
	NewResponse(req ProtoRequest) (resp ProtoResponse, err error)
	// 生成handler
	RegisterHandler(handler interface{}) ProtoHandler
	// 将request路由到handler的相应方法
	RequestDispatch(ctx context.Context, handler ProtoHandler, receiver interface{}, route string, session int32, req interface{}, traceDebugMsg bool) (dataResp []byte, pushArr []ProtoPush, err error)
	// 将push路由到handler的相应方法
	PushDispatch(ctx context.Context, handler ProtoHandler, receiver interface{}, route string, req ProtoPush) error
}
