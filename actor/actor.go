package actor

import (
	"time"
)

type Request interface {
	Next()
}

type Actor struct {
	__sync_ch__ chan Request
}

func NewActor(size int) *Actor {
	return &Actor{
		__sync_ch__: make(chan Request, size),
	}
}

// 返回收件箱chan
func (self *Actor) Inbox() chan Request {
	return self.__sync_ch__
}

// ======================== options =======================
// call选项
type CallOptions struct {
	TimeOut time.Duration
}

type CallOption interface {
	Config(opt *CallOptions)
}

type CallTimeOutOption time.Duration

// 设置超时时间
func WithCallTimeOut(timeout time.Duration) CallTimeOutOption {
	return CallTimeOutOption(timeout)
}

func (self CallTimeOutOption) Config(opt *CallOptions) {
	opt.TimeOut = time.Duration(self)
}
