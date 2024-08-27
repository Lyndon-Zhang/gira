package gate

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lyndon-Zhang/gira"
)

var (
	ErrIllegalUID        = errors.New("illegal uid")
	sid           uint64 = 0
)

type Session struct {
	sync.RWMutex
	id       uint64
	uid      int64
	lastTime int64
	conn     *Conn
	data     map[string]interface{}
	secret   string
	userData interface{}
}

func newSession(agent *Conn) *Session {
	return &Session{
		id:       atomic.AddUint64(&sid, 1) + agent.server.sessionModifer,
		conn:     agent,
		data:     make(map[string]interface{}),
		lastTime: time.Now().Unix(),
		secret:   "",
	}
}

func (s *Session) getSecret() string {
	return s.secret
}

func (s *Session) setSecret(key string) {
	s.secret = key
}

func (s *Session) Push(route string, data []byte) error {
	return s.conn.Push(route, data)
}

func (s *Session) Recv(ctx context.Context) (gira.GatewayMessage, error) {
	return s.conn.Recv(ctx)
}

func (s *Session) Response(mid uint64, data []byte) error {
	return s.conn.Response(mid, data)
}

func (s *Session) Id() uint64 {
	return s.id
}

func (s *Session) UID() int64 {
	return atomic.LoadInt64(&s.uid)
}

func (s *Session) Bind(uid int64) error {
	if uid < 1 {
		return ErrIllegalUID
	}
	atomic.StoreInt64(&s.uid, uid)
	return nil
}

func (s *Session) Kick(reason string) {
	s.conn.Kick(reason)
	s.conn.Close()
}

func (s *Session) SendServerSuspend(reason string) {
	s.conn.SendServerSuspendPacket(reason)
}

func (s *Session) SendServerResume(reason string) {
	s.conn.SendServerResumePacket(reason)
}

func (s *Session) SendServerMaintain(reason string) {
	s.conn.SendServerMaintainPacket(reason)
}

func (s *Session) SendServerDown(reason string) {
	s.conn.SendServerDownPacket(reason)
}

func (s *Session) Close() error {
	return s.conn.Close()
}

func (s *Session) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *Session) Remove(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.data, key)
}

func (s *Session) SetUserData(value interface{}) {
	s.userData = value
}

func (s *Session) Set(key string, value interface{}) {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
}

func (s *Session) HasKey(key string) bool {
	s.RLock()
	defer s.RUnlock()
	_, has := s.data[key]
	return has
}

func (s *Session) Int(key string) int {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(int)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Int8(key string) int8 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(int8)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Int16(key string) int16 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(int16)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Int32(key string) int32 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(int32)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Int64(key string) int64 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(int64)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Uint(key string) uint {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(uint)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Uint8(key string) uint8 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(uint8)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Uint16(key string) uint16 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(uint16)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Uint32(key string) uint32 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(uint32)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Uint64(key string) uint64 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(uint64)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Float32(key string) float32 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(float32)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) Float64(key string) float64 {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return 0
	}
	value, ok := v.(float64)
	if !ok {
		return 0
	}
	return value
}

func (s *Session) String(key string) string {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return ""
	}
	value, ok := v.(string)
	if !ok {
		return ""
	}
	return value
}

func (s *Session) Value(key string) interface{} {
	s.RLock()
	defer s.RUnlock()
	return s.data[key]
}

func (s *Session) UserData() interface{} {
	return s.userData
}

func (s *Session) State() map[string]interface{} {
	s.RLock()
	defer s.RUnlock()
	return s.data
}

func (s *Session) Restore(data map[string]interface{}) {
	s.Lock()
	defer s.Unlock()
	s.data = data
}

func (s *Session) Clear() {
	s.Lock()
	defer s.Unlock()
	s.uid = 0
	s.data = map[string]interface{}{}
}
