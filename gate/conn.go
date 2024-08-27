package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	log "github.com/Lyndon-Zhang/gira/corelog"

	"github.com/Lyndon-Zhang/gira/gate/crypto"
	"github.com/Lyndon-Zhang/gira/gate/message"
	"github.com/Lyndon-Zhang/gira/gate/packet"
	"golang.org/x/sync/errgroup"
)

const (
	_ int32 = iota
	conn_status_start
	conn_status_handshake
	conn_status_working
	conn_status_closed
)

type Conn struct {
	session    *Session
	conn       net.Conn
	state      int32
	lastAt     int64
	decoder    *packet.Decoder
	server     *Server
	chSend     chan []byte
	chMessage  chan *Message
	ctx        context.Context
	cancelFunc context.CancelFunc
	errGroup   *errgroup.Group
	errCtx     context.Context
}

type handShake_request struct {
	Sys struct {
		Token   string `json:"token"`
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"sys"`
}

func newConn(server *Server) *Conn {
	self := &Conn{
		server:  server,
		state:   conn_status_start,
		lastAt:  time.Now().Unix(),
		decoder: packet.NewDecoder(),
	}
	self.session = newSession(self)
	return self
}

// 如果链接已关闭，则返回ErrBrokenPipe
func (a *Conn) send(data []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
		}
	}()
	a.chSend <- data
	return
}

// 如果链接已关闭, 则返回ErrBrokenPipe
func (self *Conn) Push(route string, data []byte) error {
	if self.server.debug {
		// log.Debugw("conn push", "session_id", self.session.Id(), "route", route, "len", len(data))
	}
	var err error
	payload, err := self.serialize(data)
	if err != nil {
		log.Errorw("conn push fail", "session_id", self.session.Id(), "route", route, "error", err)
	}
	msg := &message.Message{
		Type:  message.Push,
		Data:  payload,
		Route: "",
		Id:    0,
	}
	em, err := msg.Encode()
	if err != nil {
		return err
	}
	p, err := packet.Encode(packet.Data, em)
	if err != nil {
		return err
	}
	return self.send(p)
}

// 如果链接已关闭,则返回ErrBrokenPipe
func (self *Conn) Response(mid uint64, data []byte) error {
	if self.server.debug {
		log.Debugw("conn response", "session_id", self.session.Id(), "req_id", mid, "len", len(data))
	}
	if mid <= 0 {
		return ErrSessionOnNotify
	}
	var err error
	payload, err := self.serialize(data)
	if err != nil {
		log.Errorw("conn response fail", "session_id", self.session.Id(), "req_id", mid, "error", err)
	}
	msg := &message.Message{
		Type:  message.Response,
		Data:  payload,
		Route: "",
		Id:    mid,
	}
	em, err := msg.Encode()
	if err != nil {
		return err
	}
	p, err := packet.Encode(packet.Data, em)
	if err != nil {
		return err
	}
	return self.send(p)
}

func (self *Conn) Kick(reason string) error {
	data, err := packet.Encode(packet.Kick, []byte(reason))
	if err != nil {
		return err
	}
	return self.send(data)
}

func (self *Conn) SendServerSuspendPacket(reason string) error {
	data, err := packet.Encode(packet.ServerSuspend, []byte(reason))
	if err != nil {
		return err
	}
	return self.send(data)
}

func (self *Conn) SendServerResumePacket(reason string) error {
	data, err := packet.Encode(packet.ServerResume, []byte(reason))
	if err != nil {
		return err
	}
	return self.send(data)
}

func (self *Conn) SendServerDownPacket(reason string) error {
	data, err := packet.Encode(packet.ServerDown, []byte(reason))
	if err != nil {
		return err
	}
	return self.send(data)
}

func (self *Conn) SendServerMaintainPacket(reason string) error {
	data, err := packet.Encode(packet.ServerMaintain, []byte(reason))
	if err != nil {
		return err
	}
	return self.send(data)
}

func (self *Conn) Close() error {
	if self.status() == conn_status_closed {
		return nil
	}
	self.setStatus(conn_status_closed)
	if self.server.debug {
		log.Infow("close conn", "session_id", self.session.Id(), "remote_addr", self.conn.RemoteAddr())
	}
	self.cancelFunc()
	return nil
}

func (self *Conn) RemoteAddr() net.Addr {
	return self.conn.RemoteAddr()
}

func (self *Conn) String() string {
	return fmt.Sprintf("[gate conn] remote=%s, lastTime=%d", self.conn.RemoteAddr().String(), atomic.LoadInt64(&self.lastAt))
}

func (self *Conn) status() int32 {
	return atomic.LoadInt32(&self.state)
}

func (self *Conn) setStatus(state int32) {
	atomic.StoreInt32(&self.state, state)
}

func (self *Conn) serialize(data []byte) ([]byte, error) {
	var session = self.session
	if session.getSecret() != "" {
		var err error
		data, err = crypto.DesEncrypt(data, session.getSecret())
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (self *Conn) recvHandshake(ctx context.Context) error {
	buf := make([]byte, 2048)
	timeoutCtx, timeoutFunc := context.WithTimeout(ctx, self.server.handshakeTimeout)
	defer timeoutFunc()
	go func() {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				log.Errorw("recv handshake timeout", "error", timeoutCtx.Err())
				// 关闭链接，便read返回
				self.conn.Close()
			}
		}
	}()
	for {
		n, err := self.conn.Read(buf)
		if err != nil {
			if self.server.debug {
				log.Debugw("conn read fail", "err", err, "session_id", self.session.Id())
			}
			return err
		}
		packets, err := self.decoder.Decode(buf[:n])
		if err != nil {
			return err
		}
		if len(packets) < 1 {
			continue
		}
		if len(packets) != 1 {
			return ErrHandShakeAck
		}
		p := packets[0]
		if p.Type != packet.Handshake {
			return ErrInvalidPacket
		}
		msg := &handShake_request{}
		err = json.Unmarshal(p.Data, msg)
		if err != nil {
			return err
		}
		if self.server.rsaPrivateKey != "" {
			token, err := crypto.RsaDecryptWithSha1Base64(msg.Sys.Token, self.server.rsaPrivateKey)
			if err != nil {
				return err
			}
			self.session.setSecret(token)
		}
		if err := self.server.handshakeValidator(p.Data); err != nil {
			return err
		}
		data, err := json.Marshal(map[string]interface{}{
			"code": 200,
			"sys": map[string]interface{}{
				"heartbeat": self.server.heartbeat.Seconds(),
				"session":   self.session.Id(),
			},
		})
		if err != nil {
			return err
		}
		handsharkResponse, err := packet.Encode(packet.Handshake, data)
		if err != nil {
			return err
		}
		if _, err := self.conn.Write(handsharkResponse); err != nil {
			return err
		}
		if self.server.debug {
			log.Debugw("handshake success", "session_id", self.session.Id(), "remote_addr", self.conn.RemoteAddr(), "secret", self.session.getSecret())
		}
		return nil
	}
}

func (self *Conn) recvHandshakeAck(ctx context.Context) ([]*packet.Packet, error) {
	cancelCtx, cancelFunc := context.WithTimeout(ctx, self.server.handshakeTimeout)
	defer cancelFunc()
	go func() {
		select {
		case <-cancelCtx.Done():
			if cancelCtx.Err() == context.DeadlineExceeded {
				log.Info("recv handshake ack timeout", cancelCtx.Err())
				// 关闭链接，便read返回
				self.conn.Close()
			}
		}
	}()
	buf := make([]byte, 2048)
	for {
		n, err := self.conn.Read(buf)
		if err != nil {
			if self.server.debug {
				log.Debugw("conn read fail", "err", err, "session_id", self.session.Id())
			}
			return nil, err
		}
		packets, err := self.decoder.Decode(buf[:n])
		if err != nil {
			return nil, err
		}
		if len(packets) < 1 {
			continue
		}
		p := packets[0]
		if p.Type != packet.HandshakeAck {
			return nil, ErrInvalidPacket
		}
		self.setStatus(conn_status_working)
		if self.server.debug {
			log.Debugw("recv handshake ack success", "session_id", self.session.Id(), "remote_addr", self.conn.RemoteAddr())
		}
		return packets[1:], nil
	}
}

// / serve函数负责关闭传递过来的conn
func (self *Conn) serve(ctx context.Context, conn net.Conn) error {
	var err error
	var packets []*packet.Packet
	self.conn = conn
	sessionId := self.session.Id()
	if self.server.debug {
		log.Debugw("conn established", "session_id", sessionId)
	}
	defer func() {
		if self.server.debug {
			log.Debugw("conn closed", "session_id", sessionId)
		}
		// 关闭链接， self.ctx还是空的，不用关闭ctx
		self.setStatus(conn_status_closed)
		self.conn.Close()
	}()
	if self.server.handler == nil {
		return ErrInvalidHandler
	}
	self.ctx, self.cancelFunc = context.WithCancel(ctx)
	defer self.cancelFunc()
	if err := self.recvHandshake(self.ctx); err != nil {
		atomic.AddInt64(&self.server.Stat.HandshakeErrorCount, 1)
		return err
	}
	self.setStatus(conn_status_handshake)
	if packets, err = self.recvHandshakeAck(self.ctx); err != nil {
		atomic.AddInt64(&self.server.Stat.HandshakeErrorCount, 1)
		return err
	}
	self.setStatus(conn_status_working)

	// 握手成功，开始收发消息
	self.chMessage = make(chan *Message, self.server.recvBacklog)
	self.chSend = make(chan []byte, self.server.sendBacklog)

	errGroup, errCtx := errgroup.WithContext(self.ctx)
	self.errGroup, self.errCtx = errGroup, errCtx

	// 开启读协程
	// 退出时不需要主动关闭链接，由于写协程还需要发送缓冲区剩下的消息
	errGroup.Go(func() (err error) {
		defer func() {
			close(self.chMessage)
			if self.server.debug {
				log.Debugw("conn recv goroutine exit", "sessionid", sessionId)
			}
		}()
		// 处理一些多接收到的消息
		for i := range packets {
			if err = self.processPacket(packets[i]); err != nil {
				return
			}
		}
		var n int
		// 持续读数据
		buf := make([]byte, self.server.recvBuffSize)
		var packets []*packet.Packet
		for {
			n, err = self.conn.Read(buf)
			if err != nil {
				if self.server.debug {
					log.Debugw("conn read fail", "err", err, "session_id", self.session.Id())
				}
				return
			}
			// 使read goroutine尽快结束
			if self.status() == conn_status_closed {
				return
			}
			packets, err = self.decoder.Decode(buf[:n])
			if err != nil {
				return err
			}
			if len(packets) < 1 {
				continue
			}
			for i := range packets {
				if err = self.processPacket(packets[i]); err != nil {
					return
				}
			}
		}
	})
	//开启写协程
	errGroup.Go(func() (err error) {
		defer func() {
			// 发送完数据后，可以关闭socket了，使recv可以解除阻塞
			self.conn.Close()
			if self.server.debug {
				log.Debugw("conn send goroutine exit", "session_id", sessionId)
			}
		}()
		err = func() (err error) {
			ticker := time.NewTicker(self.server.heartbeat)
			defer func() {
				ticker.Stop()
			}()
			for {
				select {
				case <-ticker.C:
					deadline := time.Now().Add(-2 * self.server.heartbeat).Unix()
					if atomic.LoadInt64(&self.lastAt) < deadline {
						log.Infof("gate connection heartbeat timeout, sessionid=%d, lastTime=%d, deadline=%d\n", sessionId, atomic.LoadInt64(&self.lastAt), deadline)
						err = ErrHeartbeatTimeout
						return
					}
					self.chSend <- heartbeat_packet
				case data := <-self.chSend:
					if data == nil {
						// unexpect
						err = ErrBrokenPipe
						return
					} else {
						if _, err = self.conn.Write(data); err != nil {
							log.Infof("gate connection write failed, sessionid=%d, error:%s\n", sessionId, err.Error())
							return
						}
					}
				case <-errCtx.Done():
					ticker.Stop()
					close(self.chSend)
					err = errCtx.Err()
					return
				}
			}
		}()
		// 发送完剩下的数据
		for {
			select {
			case data := <-self.chSend:
				if data == nil {
					// unexpect
					err = ErrBrokenPipe
					return
				} else {
					if _, err = self.conn.Write(data); err != nil {
						log.Infof("gate connection write failed, sessionid=%d, error:%s\n", sessionId, err.Error())
						return
					}
				}
			}
		}
	})
	errGroup.Go(func() (err error) {
		select {
		case <-errCtx.Done():
			err = errCtx.Err()
			return
		}
	})
	atomic.AddInt64(&self.server.Stat.ActiveSessionCount, 1)
	atomic.AddInt64(&self.server.Stat.CumulativeSessionCount, 1)

	self.server.storeSession(self.session)
	defer self.server.sessionClosed(self.session)
	// middleware
	for _, middleware := range self.server.middlewareArr {
		middleware.OnSessionOpen(self.session)
	}
	defer func() {
		for _, middleware := range self.server.middlewareArr {
			middleware.OnSessionClose(self.session)
		}
	}()

	if self.server.handler != nil {
		//处理消息
		self.server.handler.ServeClientStream(self.session)
	}
	self.setStatus(conn_status_closed)
	self.cancelFunc()
	err = errGroup.Wait()
	if self.server.debug {
		log.Debugw("conn wait group exit", "error", err)
	}
	atomic.AddInt64(&self.server.Stat.ActiveSessionCount, -1)
	return err
}

func (self *Conn) processPacket(p *packet.Packet) error {
	self.lastAt = time.Now().Unix()
	switch p.Type {
	case packet.Data:
		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}
		return self.processMessage(msg)
	case packet.Heartbeat:
		// expected
	default:
		return ErrInvalidPacket
	}
	return nil
}

func (self *Conn) processMessage(msg *message.Message) (err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Errorw("process message panic", "error", e)
			err = ErrBrokenPipe
		}
	}()
	var reqId uint64
	switch msg.Type {
	case message.Request:
		reqId = msg.Id
	case message.Notify:
		reqId = 0
	default:
		log.Warnw("recv invalid message type", "type", msg.Type.String())
		err = ErrInvalidMessage
		return
	}
	var session = self.session
	// WARN: 当前data指向缓冲区，要复制出来
	var payload = make([]byte, len(msg.Data))
	copy(payload, msg.Data)
	if session.getSecret() != "" {
		payload, err = crypto.DesDecrypt(payload, session.getSecret())
		if err != nil {
			log.Errorw("des decrypt fail", "error", err)
			return
		}
	}
	if self.server.handler == nil {
		log.Warnw("handler not found", "route", msg.Route)
		err = ErrInvalidHandler
	}
	r := &Message{
		session: session,
		route:   msg.Route,
		payload: payload,
		reqId:   reqId,
	}
	// 处理request
	for _, middleware := range self.server.middlewareArr {
		middleware.ServeMessage(r)
	}
	self.chMessage <- r
	return
}

// 返回接收到的消息
// 即使链接已经关闭,也会返回已经接收到的消息,直到没有可处理的消息为止,则返回ErrBrokenPipe
// Returns:
// msg - 返回接收到的消息
// err - 如果连接已关闭, 且没有消息了, 则返回ErrBrokenPipe
func (self *Conn) Recv(ctx context.Context) (msg *Message, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
			return
		}
	}()
	select {
	case msg = <-self.chMessage:
		if msg == nil {
			err = ErrBrokenPipe
			return
		}
		return
	case <-ctx.Done():
		err = ctx.Err()
		return
	}
}
