package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"golang.org/x/sync/errgroup"

	"github.com/gorilla/websocket"
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/gate/crypto"
	"github.com/Lyndon-Zhang/gira/gate/message"
	"github.com/Lyndon-Zhang/gira/gate/packet"
	"github.com/Lyndon-Zhang/gira/gate/ws"
)

const (
	_ int32 = iota
	client_status_start
	client_status_handshake
	client_status_working
	client_status_closed
)

// Errors that could be occurred during message handling.
var (
	ErrCloseClosedSession = errors.New("close closed session")
	ErrBrokenPipe         = errors.New("broken low-level pipe")
	ErrHandshake          = errors.New("handshake failed")
	ErrHandshakeAck       = errors.New("handshake ack failed")
	ErrHandshakeTimeout   = errors.New("handshake timeout failed")
	ErrHeartbeatTimeout   = errors.New("heartbeat timeout")
	ErrDialTimeout        = errors.New("dial timeout")
	ErrDialInterrupt      = errors.New("dial interrupt")
	ErrInvalidPacket      = errors.New("invalid packet")
)

type handshake_response struct {
	Sys struct {
		Heartbeat int    `json:"heartbeat"`
		Session   uint64 `json:"session"`
	} `json:"sys"`
	Code int `json:"code"`
}

type ClientConn struct {
	// options
	isWebsocket        bool
	tslInsecure        bool
	tslCertificate     string
	tslKey             string
	handshakeValidator func([]byte) error
	heartbeat          time.Duration
	debug              bool
	wsPath             string
	rsaPublicKey       string
	serverAddr         string
	heartbeatPacket    []byte
	sessionId          uint64
	conn               net.Conn
	decoder            *packet.Decoder
	secretKey          string
	dialTimeout        time.Duration
	responseRouter     sync.Map
	lastAt             int64
	chWrite            chan []byte
	state              int32
	chMessage          chan *message.Message
	ctx                context.Context
	cancelFunc         context.CancelFunc
	errCtx             context.Context
	sendBacklog        int
	recvBacklog        int
	lastErr            error
	recvBuffSize       int
	handshakeTimeout   time.Duration
}

func newClientConn() *ClientConn {
	self := &ClientConn{
		decoder:            packet.NewDecoder(),
		heartbeat:          30 * time.Second,
		debug:              false,
		handshakeValidator: func(_ []byte) error { return nil },
		rsaPublicKey:       "",
		sendBacklog:        16,
		recvBacklog:        16,
		lastAt:             time.Now().Unix(),
		state:              client_status_start,
		recvBuffSize:       4096,
		handshakeTimeout:   2 * time.Second,
	}
	return self
}

type Option func(conn *ClientConn)

func WithSendBacklog(v int) Option {
	return func(conn *ClientConn) {
		conn.sendBacklog = v
	}
}

func WithRecvBacklog(v int) Option {
	return func(conn *ClientConn) {
		conn.recvBacklog = v
	}
}

func WithHandshakeValidator(fn func([]byte) error) Option {
	return func(conn *ClientConn) {
		conn.handshakeValidator = fn
	}
}

func WithRecvBuffSize(v int) Option {
	return func(conn *ClientConn) {
		conn.recvBuffSize = v
	}
}

func WithHeartbeatInterval(d time.Duration) Option {
	return func(conn *ClientConn) {
		conn.heartbeat = d
	}
}

func WithDictionary(dict map[string]uint16) Option {
	return func(conn *ClientConn) {
	}
}

func WithWSPath(path string) Option {
	return func(conn *ClientConn) {
		conn.wsPath = path
	}
}

func WithServerAdd(addr string) Option {
	return func(conn *ClientConn) {
		conn.serverAddr = addr
	}
}

func WithTSLInsecure(insecureSkipVerify bool) Option {
	return func(conn *ClientConn) {
		conn.tslInsecure = insecureSkipVerify
	}
}
func WithTSLConfig(certificate, key string) Option {
	return func(conn *ClientConn) {
		conn.tslCertificate = certificate
		conn.tslKey = key
	}
}
func WithDialTimeout(timeout time.Duration) Option {
	return func(conn *ClientConn) {
		conn.dialTimeout = timeout
	}
}

func WithHandshakeTimeout(timeout time.Duration) Option {
	return func(conn *ClientConn) {
		conn.handshakeTimeout = timeout
	}
}

func WithContext(ctx context.Context) Option {
	return func(conn *ClientConn) {
		conn.ctx, conn.cancelFunc = context.WithCancel(ctx)
	}
}

func WithIsWebsocket(enableWs bool) Option {
	return func(conn *ClientConn) {
		conn.isWebsocket = enableWs
	}
}
func WithDebugMode(debug bool) Option {
	return func(conn *ClientConn) {
		conn.debug = debug
	}
}

func WithRSAPublicKey(keyFile string) Option {
	data, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Info(err)
		return nil
	}
	return func(conn *ClientConn) {
		conn.rsaPublicKey = string(data)
	}
}

func Dial(addr string, opts ...Option) (gira.GatewayClient, error) {
	conn := newClientConn()
	for _, v := range opts {
		v(conn)
	}
	if conn.ctx == nil {
		conn.ctx, conn.cancelFunc = context.WithCancel(context.Background())
	}

	if err := conn.dial(addr); err != nil {
		return nil, err
	}
	return conn, nil
}

func (conn *ClientConn) dial(addr string) error {
	var err error
	heartbeatPacket, err := packet.Encode(packet.Heartbeat, nil)
	if err != nil {
		return err
	}
	conn.heartbeatPacket = heartbeatPacket
	conn.serverAddr = addr
	var c net.Conn
	if conn.isWebsocket {
		if conn.tslInsecure {
			if c, err = conn.dialWSTLSInsecure(); err != nil {
				return err
			}
		} else if len(conn.tslCertificate) != 0 {
			if c, err = conn.dialWSTLS(); err != nil {
				return err
			}
		} else {
			if c, err = conn.dialWS(); err != nil {
				return err
			}
		}
	} else {
		if c, err = conn.dialTcp(); err != nil {
			return err
		}
	}
	conn.conn = c
	conn.setStatus(client_status_handshake)
	err = conn.sendHandshake()
	if err != nil {
		log.Errorw("client sendHandshake fail", "error", err)
		c.Close()
		return ErrHandshake
	}
	if err := conn.recvHandshakeAck(conn.ctx); err != nil {
		c.Close()
		return ErrHandshakeAck
	}
	conn.setStatus(client_status_working)
	conn.chWrite = make(chan []byte, conn.sendBacklog)
	conn.chMessage = make(chan *message.Message, conn.recvBacklog)
	go conn.serve()
	return nil
}

func (conn *ClientConn) serve() {
	defer func() {
		if conn.debug {
			log.Debugw("client serve exit", "session_id", conn.sessionId)
		}
	}()
	errGroup, errCtx := errgroup.WithContext(conn.ctx)
	// 写协程
	errGroup.Go(func() error {
		ticker := time.NewTicker(conn.heartbeat)
		defer func() {
			ticker.Stop()
			close(conn.chWrite)
			conn.conn.Close()
			if conn.debug {
				log.Debugw("client write goroutine exit", "session_id", conn.sessionId)
			}
		}()
		for {
			select {
			case <-ticker.C:
				deadline := time.Now().Add(-2 * conn.heartbeat).Unix()
				if atomic.LoadInt64(&conn.lastAt) < deadline {
					log.Debugw("client heartbeat timeout", "sessionid", conn.sessionId, "lastTime", atomic.LoadInt64(&conn.lastAt), "deadline", deadline)
					return ErrHeartbeatTimeout
				}
			case data := <-conn.chWrite:
				if _, err := conn.conn.Write(data); err != nil {
					log.Debugw("client write fail", "error", err)
					return err
				}
			case <-errCtx.Done():
				return errCtx.Err()
			}
		}
	})
	errGroup.Go(func() error {
		defer func() {
			close(conn.chMessage)
			if conn.debug {
				log.Debugw("client read goroutine exit", "session_id", conn.sessionId)
			}
		}()
		buf := make([]byte, conn.recvBuffSize)
		for {
			n, err := conn.conn.Read(buf)
			if err != nil {
				if conn.debug {
					log.Debugw("client read fail", "err", err, "session_id", conn.sessionId)
				}
				return err
			}
			packets, err := conn.decoder.Decode(buf[:n])
			if err != nil {
				return err
			}
			if len(packets) < 1 {
				continue
			}
			for _, p := range packets {
				if err := conn.processPacket(p); err != nil {
					return err
				}
			}
		}
	})
	err := errGroup.Wait()
	if conn.debug {
		log.Debugw("client wait group", "session_id", conn.sessionId, "error", err)
	}
}

func (conn *ClientConn) dialTcp() (net.Conn, error) {
	netDialer := net.Dialer{}
	if conn.dialTimeout != 0 {
		netDialer.Timeout = conn.dialTimeout
	}
	c, err := netDialer.DialContext(conn.ctx, "tcp", conn.serverAddr)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (conn *ClientConn) dialWS() (net.Conn, error) {
	netDialer := &net.Dialer{}
	if conn.dialTimeout != 0 {
		netDialer.Timeout = conn.dialTimeout
	}
	// websocket.DefaultDialer
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return netDialer.DialContext(conn.ctx, network, addr)
		},
	}
	u := url.URL{Scheme: "ws", Host: conn.serverAddr, Path: conn.wsPath}
	c, _, err := d.DialContext(conn.ctx, u.String(), nil)
	if err != nil {
		return nil, err
	}
	wsconn, err := ws.NewConn(c)
	if err != nil {
		return nil, err
	}
	return wsconn, nil
}

func (conn *ClientConn) dialWSTLS() (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(conn.tslCertificate, conn.tslKey)
	if err != nil {
		return nil, err
	}
	tlsDialer := &tls.Dialer{
		Config: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	netDialer := &net.Dialer{}
	if conn.dialTimeout != 0 {
		netDialer.Timeout = conn.dialTimeout
	}
	tlsDialer.NetDialer = netDialer
	// websocket.DefaultDialer
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return tlsDialer.DialContext(conn.ctx, network, addr)
		},
	}
	u := url.URL{Scheme: "wss", Host: conn.serverAddr, Path: conn.wsPath}
	c, _, err := d.DialContext(conn.ctx, u.String(), nil)
	if err != nil {
		return nil, err
	}
	wsconn, err := ws.NewConn(c)
	if err != nil {
		return nil, err
	}
	return wsconn, nil
}

func (conn *ClientConn) dialWSTLSInsecure() (net.Conn, error) {
	netDialer := &net.Dialer{}
	if conn.dialTimeout != 0 {
		netDialer.Timeout = conn.dialTimeout
	}
	// websocket.DefaultDialer
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return netDialer.DialContext(conn.ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	u := url.URL{Scheme: "wss", Host: conn.serverAddr, Path: conn.wsPath}
	c, _, err := d.DialContext(conn.ctx, u.String(), nil)
	if err != nil {
		return nil, err
	}
	wsconn, err := ws.NewConn(c)
	if err != nil {
		return nil, err
	}
	return wsconn, nil
}

func (conn *ClientConn) setSecretKey(key string) {
	conn.secretKey = key
}

func (conn *ClientConn) getSecretKey() string {
	return conn.secretKey
}

// 发送握手协议
func (conn *ClientConn) sendHandshake() error {
	tokenByte := make([]byte, 8)
	_, err := rand.Read(tokenByte)
	if err != nil {
		return err
	}
	token := ""
	if conn.rsaPublicKey != "" {
		token = base32.StdEncoding.EncodeToString(tokenByte)[:8]
		conn.setSecretKey(token)
		token, err = crypto.RsaEncryptWithSha1Base64(token, conn.rsaPublicKey)
		if err != nil {
			return err
		}
	}
	payload, err := json.Marshal(map[string]interface{}{
		"sys": map[string]interface{}{
			"type":    "go-websocket",
			"version": "0.0.1",
			"token":   token,
		},
		"user": map[string]interface{}{},
	})
	if err != nil {
		return err
	}
	data, err := packet.Encode(packet.Handshake, payload)
	if err != nil {
		return err
	}
	if _, err := conn.conn.Write(data); err != nil {
		return err
	}
	if conn.debug {
		log.Debugw("send handshake", "token", token)
	}
	return nil
}

// 等待握手确认
func (conn *ClientConn) recvHandshakeAck(ctx context.Context) (err error) {
	buf := make([]byte, conn.recvBuffSize)
	timeoutCtx, timeoutFunc := context.WithTimeout(ctx, conn.handshakeTimeout)
	defer timeoutFunc()
	go func() {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				log.Errorw("recv handshake timeout", "error", timeoutCtx.Err())
				// 关闭链接，使read解除阻塞
				conn.conn.Close()
				err = ErrHandshake
			}
		}
	}()
	for {
		var n int
		n, err = conn.conn.Read(buf)
		if err != nil {
			if conn.debug {
				log.Info(fmt.Sprintf("[agent] read message error: %s, session will be closed immediately, sessionid=%d", err.Error(), conn.sessionId))
			}
			return err
		}
		packets, err := conn.decoder.Decode(buf[:n])
		if err != nil {
			log.Info(err)
			return err
		}
		if len(packets) < 1 {
			continue
		}
		if len(packets) != 1 {
			return ErrInvalidPacket
		}
		p := packets[0]
		if p.Type != packet.Handshake {
			return ErrInvalidPacket
		}
		msg := &handshake_response{}
		err = json.Unmarshal(p.Data, msg)
		if err != nil {
			return err
		}
		if msg.Code != 200 {
			return ErrHandshake
		}
		payload, err := json.Marshal(map[string]interface{}{})
		if err != nil {
			return err
		}
		data, err := packet.Encode(packet.HandshakeAck, payload)
		if err != nil {
			return err
		}
		conn.sessionId = msg.Sys.Session
		conn.heartbeat = time.Duration(msg.Sys.Heartbeat) * time.Second
		if conn.debug {
			log.Debugw("handshake success", "session_id", conn.sessionId, "remote_addr", conn.conn.RemoteAddr())
		}
		if _, err := conn.conn.Write(data); err != nil {
			return err
		}
		return nil
	}
}

// 处理内部消息包
func (conn *ClientConn) processPacket(p *packet.Packet) error {
	if conn.debug {
		// log.Debugw("got packet", "type", p.Type)
	}
	switch p.Type {
	case packet.Data:
		if conn.status() < client_status_working {
			return fmt.Errorf("receive data on socket which not yet ACK, session will be closed immediately, sessionid=%d, remote=%s, status=%d",
				conn.sessionId, conn.conn.RemoteAddr().String(), conn.status())
		}
		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}
		conn.processMessage(msg)
	case packet.Heartbeat:
		conn.chWrite <- conn.heartbeatPacket
	case packet.Kick:
		log.Info("client recv kick packet", string(p.Data))
	case packet.ServerSuspend:
		log.Info("client recv server suspend packet")
	case packet.ServerResume:
		log.Info("client recv server resume packet")
	case packet.ServerDown:
		log.Info("client recv server down packet")
	case packet.ServerMaintain:
		log.Info("client recv server maintain packet")
	}
	conn.lastAt = time.Now().Unix()
	return nil
}

func (conn *ClientConn) processMessage(msg *message.Message) {
	if conn.debug {
		// log.Debugw("got message", "type", msg.Type)
	}
	switch msg.Type {
	case message.Response:
	case message.Push:
	default:
		log.Info("Invalid message type: " + msg.Type.String())
		return
	}
	if conn.getSecretKey() != "" {
		payload, err := crypto.DesDecrypt(msg.Data, conn.getSecretKey())
		if err != nil {
			log.Info(fmt.Sprintf("crypto.DesDecrypt failed: %+v (%v)", err, payload))
			return
		}
		msg.Data = payload
	}
	// WARN: 当前data是指向缓冲区的，如果交给其他协程处理，要复制一次
	data := msg.Data
	msg.Data = make([]byte, len(msg.Data))
	copy(msg.Data, data)
	conn.chMessage <- msg
}

func (conn *ClientConn) status() int32 {
	return atomic.LoadInt32(&conn.state)
}

func (conn *ClientConn) setStatus(state int32) {
	atomic.StoreInt32(&conn.state, state)
}

func (conn *ClientConn) Recv(ctx context.Context) (typ int, route string, reqId uint64, data []byte, err error) {
	select {
	case msg := <-conn.chMessage:
		if msg == nil {
			err = ErrBrokenPipe
			return
		}
		typ = int(msg.Type)
		switch msg.Type {
		case message.Response:
			reqId = msg.Id
			if s, ok := conn.responseRouter.Load(reqId); ok {
				route = s.(string)
				conn.responseRouter.Delete(reqId)
			}
		case message.Push:
			reqId = 0
		}
		data = msg.Data
		return
	case <-ctx.Done():
		err = ctx.Err()
		return
	}
}

func (conn *ClientConn) send(typ message.Type, reqId uint64, route string, data []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
		}
	}()
	if conn.status() == client_status_closed {
		err = ErrBrokenPipe
		return
	}
	// if len(self.chSend) >= self.sendBacklog {
	// 	return ErrBufferExceed
	// }
	if conn.debug {
		log.Debugw("send", "type", typ, "sessioni_id", conn.sessionId, "req_id", reqId, "route", route, "len", len(data))
	}
	if typ == message.Request {
		conn.responseRouter.Store(reqId, route)
	}
	m := &message.Message{
		Type:  typ,
		Data:  data,
		Route: route,
		Id:    reqId,
	}
	var em []byte
	var p []byte
	em, err = m.Encode()
	if err != nil {
		return
	}
	p, err = packet.Encode(packet.Data, em)
	if err != nil {
		return
	}
	conn.chWrite <- p
	// if _, err := self.conn.Write(p); err != nil {
	// 	return err
	// }
	return
}

// / 发送通知
func (conn *ClientConn) Notify(route string, data []byte) error {
	return conn.send(message.Notify, 0, route, data)
}

// / 发送请求
func (conn *ClientConn) Request(route string, reqId uint64, data []byte) error {
	return conn.send(message.Request, reqId, route, data)
}

// 关闭链接，使recv等阻塞操作立刻返回
func (conn *ClientConn) Close() error {
	if conn.status() == client_status_closed {
		return ErrCloseClosedSession
	} else {
		conn.setStatus(client_status_closed)
		conn.cancelFunc()
		return nil
	}
}
