package gate

type MiddleWareInterface interface {
	ServeMessage(r *Message)
	OnSessionOpen(s *Session)
	OnSessionClose(s *Session)
}
