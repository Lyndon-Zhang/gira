package gate

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/gate/packet"
	"github.com/Lyndon-Zhang/gira/gate/ws"
	"github.com/Lyndon-Zhang/gira/log"
	"golang.org/x/sync/errgroup"
)

const (
	_ int32 = iota
	server_status_start
	server_status_maintain
	server_status_working
	server_status_closed
)

var heartbeat_packet []byte

func init() {
	var err error
	heartbeat_packet, err = packet.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

type Server struct {
	BindAddr string
	Host     string
	Port     int32

	ctx                context.Context
	cancelFunc         context.CancelFunc
	errCtx             context.Context
	errGroup           *errgroup.Group
	isWebsocket        bool
	tslCertificate     string
	tslKey             string
	handshakeValidator func([]byte) error
	heartbeat          time.Duration
	checkOrigin        func(*http.Request) bool
	debug              bool
	wsPath             string
	rsaPrivateKey      string
	sessionModifer     uint64
	handshakeTimeout   time.Duration
	sendBacklog        int
	recvBacklog        int
	recvBuffSize       int
	state              int32
	mu                 sync.RWMutex
	sessions           map[uint64]*Session
	handler            gira.GatewayHandler
	middlewareArr      []MiddleWareInterface
	listener           net.Listener
	httpServer         *http.Server
	Stat               Stat
}

type Stat struct {
	ActiveSessionCount        int64 // 当前会话数量
	CumulativeSessionCount    int64 // 累计会话数量
	CumulativeConnectionCount int64 // 累计连接数量
	ActiveConnectionCount     int64 // 当前连接数量
	HandshakeErrorCount       int64
}

func newDefaultServer() *Server {
	gate := &Server{
		state:              server_status_start,
		heartbeat:          30 * time.Second,
		debug:              false,
		checkOrigin:        func(_ *http.Request) bool { return true },
		handshakeValidator: func(_ []byte) error { return nil },
		middlewareArr:      make([]MiddleWareInterface, 0),
		sessions:           map[uint64]*Session{},
		handshakeTimeout:   2 * time.Second,
		sendBacklog:        16,
		recvBacklog:        16,
		recvBuffSize:       4096,
	}
	return gate
}

func NewConfigServer(ctx context.Context, config gira.GatewayConfig) (*Server, error) {
	opts := []Option{
		WithDebugMode(config.Debug),
		WithWSPath(config.WsPath),
		WithIsWebsocket(config.IsWebsocket),
		WithSessionModifer(uint64(facade.GetAppId()) << 48),
		WithRecvBuffSize(config.RecvBuffSize),
		WithSendBacklog(config.SendBacklog),
		WithRecvBacklog(config.RecvBacklog),
		WithHeartbeatInterval(config.Heartbeat),
		WithHandshakeTimeout(config.HandshakeTimeout),
	}
	if config.Ssl && len(config.CertFile) > 0 && len(config.KeyFile) > 0 {
		opts = append(opts, WithTSLConfig(config.CertFile, config.KeyFile))
	}
	var server *Server
	var err error
	if server, err = Listen(ctx, config.Bind, opts...); err != nil {
		return nil, err
	}
	corelog.Infow("gateway start", "bind", config.Bind, "path", config.WsPath)
	return server, nil
}

type Option func(gateway *Server)

func WithSessionModifer(v uint64) Option {
	return func(server *Server) {
		server.sessionModifer = v
	}
}

func WithHandshakeValidator(fn func([]byte) error) Option {
	return func(server *Server) {
		server.handshakeValidator = fn
	}
}

func WithDebugMode(v bool) Option {
	return func(server *Server) {
		server.debug = v
	}
}

func WithCheckOriginFunc(fn func(*http.Request) bool) Option {
	return func(server *Server) {
		server.checkOrigin = fn
	}
}

func WithHeartbeatInterval(d time.Duration) Option {
	return func(server *Server) {
		server.heartbeat = d
	}
}

func WithHandshakeTimeout(d time.Duration) Option {
	return func(server *Server) {
		server.handshakeTimeout = d
	}
}

func WithRecvBuffSize(v int) Option {
	return func(server *Server) {
		server.recvBuffSize = v
	}
}

func WithSendBacklog(v int) Option {
	return func(server *Server) {
		server.sendBacklog = v
	}
}

func WithRecvBacklog(v int) Option {
	return func(server *Server) {
		server.recvBacklog = v
	}
}

func WithDictionary(dict map[string]uint16) Option {
	return func(server *Server) {
	}
}

func WithWSPath(path string) Option {
	return func(server *Server) {
		server.wsPath = path
	}
}

func WithIsWebsocket(enableWs bool) Option {
	return func(server *Server) {
		server.isWebsocket = enableWs
	}
}

func WithTSLConfig(certificate, key string) Option {
	return func(server *Server) {
		server.tslCertificate = certificate
		server.tslKey = key
	}
}

func WithRSAPrivateKey(keyFile string) Option {
	data, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Error(err)
		return nil
	}
	return func(server *Server) {
		server.rsaPrivateKey = string(data)
	}
}

func Listen(ctx context.Context, addr string, opts ...Option) (*Server, error) {
	server := newDefaultServer()
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
	}
	server.ctx, server.cancelFunc = context.WithCancel(ctx)
	server.errGroup, server.errCtx = errgroup.WithContext(server.ctx)
	addrPat := strings.SplitN(addr, ":", 2)
	if server.isWebsocket {
		if len(server.tslCertificate) != 0 {
			server.Host = fmt.Sprintf("wss://%s", addrPat[0])
		} else {
			server.Host = fmt.Sprintf("ws://%s", addrPat[0])
		}
	} else {
		server.Host = addrPat[0]
	}
	if port, err := strconv.Atoi(addrPat[1]); err != nil {
		return nil, ErrInvalidAddress
	} else {
		server.Port = int32(port)
	}
	if len(addrPat) < 2 {
		return nil, ErrInvalidAddress
	}
	server.BindAddr = fmt.Sprintf(":%d", server.Port)
	return server, nil
}

func (server *Server) Serve(handler gira.GatewayHandler) error {
	server.handler = handler
	server.setStatus(server_status_working)
	if server.isWebsocket {
		if len(server.tslCertificate) != 0 {
			return server.listenAndServeWSTLS()
		} else {
			return server.listenAndServeWS()
		}
	} else {
		return server.listenAndServe()
	}
}

func (server *Server) UseMiddleware(m MiddleWareInterface) {
	server.middlewareArr = append(server.middlewareArr, m)
}

// 设置成维持状态,不接受新的连接
func (server *Server) Maintain(m bool) {
	status := server.status()
	if status != server_status_maintain && status != server_status_working {
		return
	}
	if m {
		server.setStatus(server_status_maintain)
	} else {
		server.setStatus(server_status_working)
	}
}

func (server *Server) Shutdown() {
	if server.status() == server_status_closed {
		return
	}
	server.setStatus(server_status_closed)
	if server.httpServer != nil {
		timeoutCtx, timeoutFunc := context.WithTimeout(server.ctx, 5*time.Second)
		defer timeoutFunc()
		// shutdown 会关闭listener,拒绝新的连接， 但并不会关闭websocket连接
		server.httpServer.Shutdown(timeoutCtx)
	} else if server.listener != nil {
		server.listener.Close()
	}
	server.Kick("shutdown")
	server.cancelFunc()
	err := server.errGroup.Wait()
	if server.debug {
		corelog.Debugw("gate shutdown", "error", err)
	}
}

func (server *Server) status() int32 {
	return atomic.LoadInt32(&server.state)
}

func (server *Server) setStatus(state int32) {
	atomic.StoreInt32(&server.state, state)
}

func (server *Server) Kick(reason string) {
	//断开已有的链接
	server.mu.RLock()
	for _, s := range server.sessions {
		s.Kick(reason)
	}
	server.mu.RUnlock()
	now := time.Now().Unix()
	for {
		if len(server.sessions) <= 0 {
			break
		}
		if time.Now().Unix()-now >= 120 {
			corelog.Infow("Waiting session to closed", "count", len(server.sessions))
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (server *Server) listenAndServe() error {
	listener, err := net.Listen("tcp", server.BindAddr)
	if err != nil {
		return err
	}
	server.listener = listener
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go server.handleConn(conn)
	}
}

func (server *Server) listenAndServeWS() error {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     server.checkOrigin,
	}
	http.HandleFunc("/"+strings.TrimPrefix(server.wsPath, "/"), func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			corelog.Errorw("Upgrade failure", "request_uri", r.RequestURI, "error", err)
			return
		}
		server.serveWsConn(conn)
	})
	httpServer := &http.Server{Addr: server.BindAddr}
	server.httpServer = httpServer
	if err := httpServer.ListenAndServe(); err == http.ErrServerClosed {
		return nil
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func (server *Server) listenAndServeWSTLS() error {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     server.checkOrigin,
	}
	http.HandleFunc("/"+strings.TrimPrefix(server.wsPath, "/"), func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			corelog.Errorw("Upgrade failure", "request_uri", r.RequestURI, "error", err)
			return
		}
		server.serveWsConn(conn)
	})
	httpServer := &http.Server{Addr: server.BindAddr}
	server.httpServer = httpServer
	if err := httpServer.ListenAndServeTLS(server.tslCertificate, server.tslKey); err == http.ErrServerClosed {
		return nil
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func (server *Server) handleConn(conn net.Conn) {
	if server.status() != server_status_working {
		corelog.Warn("gate is not working")
		conn.Close()
		return
	}
	atomic.AddInt64(&server.Stat.CumulativeConnectionCount, 1)
	atomic.AddInt64(&server.Stat.ActiveConnectionCount, 1)
	defer func() {
		atomic.AddInt64(&server.Stat.ActiveConnectionCount, -1)
	}()
	c := newConn(server)
	if server.debug {
		corelog.Debugw("accept a conn", "session_id", c.session.Id(), "remote_addr", conn.RemoteAddr())
	}
	err := c.serve(server.ctx, conn)
	if server.debug {
		corelog.Debugw("conn serve exit", "session_id", c.session.Id(), "error", err)
	}
}

func (server *Server) serveWsConn(conn *websocket.Conn) {
	if server.status() != server_status_working {
		corelog.Warn("gate is not working")
		conn.Close()
		return
	}
	c, err := ws.NewConn(conn)
	if err != nil {
		corelog.Info(err)
		return
	}
	server.errGroup.Go(func() error {
		server.handleConn(c)
		return nil
	})
}

func (server *Server) findSession(sid uint64) *Session {
	server.mu.RLock()
	s := server.sessions[sid]
	server.mu.RUnlock()
	return s
}

func (server *Server) storeSession(s *Session) {
	server.mu.Lock()
	server.sessions[s.Id()] = s
	server.mu.Unlock()
}

func (server *Server) sessionClosed(s *Session) error {
	server.mu.Lock()
	delete(server.sessions, s.Id())
	server.mu.Unlock()
	return nil
}
