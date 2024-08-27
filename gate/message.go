package gate

import "github.com/Lyndon-Zhang/gira"

type Message struct {
	session *Session
	route   string
	payload []byte
	reqId   uint64
}

func (r *Message) Session() gira.GatewayConn {
	return r.session
}
func (r *Message) Payload() []byte {
	return r.payload
}

func (r *Message) ReqId() uint64 {
	return r.reqId
}

func (r *Message) Response(data []byte) error {
	return r.session.Response(r.reqId, data)
}

func (r *Message) Push(route string, data []byte) error {
	return r.session.Push(route, data)
}
