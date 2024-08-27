package gate

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/Lyndon-Zhang/gira/corelog"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/gate/client"
	"golang.org/x/sync/errgroup"
)

func setup() {

}
func teardown() {
}

func TestMain(m *testing.M) {
	setup()
	err := m.Run()
	teardown()
	os.Exit(err)
}

type GateHandler_TestManyClient struct {
}

func (self *GateHandler_TestManyClient) ServeClientStream(s gira.GatewayConn) {
	//var req gira.GateRequest
	for {
		_, err := s.Recv(context.TODO())
		if err != nil {
			// log.Infow("recv", "error", err)
			break
		} else {
			// log.Infow("recv", "data", string(req.Payload()))
		}
	}
}

// 模拟很多个客户端连接, 达到指定的链接数量，然后发送随机的消息，然后退出
func TestManyClient(t *testing.T) {
	var err error
	var gateway *Server
	gateway, err = Listen(context.TODO(), ":1234",
		// WithDebugMode(true),
		WithIsWebsocket(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelFunc := context.WithCancel(context.TODO())
	defer cancelFunc()
	errGroup, errCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		go func() {
			gateway.Serve(&GateHandler_TestManyClient{})
		}()
		select {
		case <-errCtx.Done():
			time.Sleep(1 * time.Second)
			log.Infow("gate stat", "ActiveSessionCount", gateway.Stat.ActiveSessionCount)
			log.Infow("gate stat", "CumulativeSessionCount", gateway.Stat.CumulativeSessionCount)
			log.Infow("gate stat", "CumulativeConnectionCount", gateway.Stat.CumulativeConnectionCount)
			log.Infow("gate stat", "HandshakeErrorCount", gateway.Stat.HandshakeErrorCount)
			gateway.Shutdown()
		}
		return nil
	})
	var activeUserCount int64 = 0
	var totalUserCount int64 = 10000
	var offlineUserCount int64 = 0
	var i int64 = 0
	for i = 0; i < totalUserCount; i++ {
		// userId := i
		errGroup.Go(func() error {
			//	log.Info("login", userId)
			atomic.AddInt64(&activeUserCount, 1)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			msgCount := rand.Intn(50)
			defer func() {
				atomic.AddInt64(&activeUserCount, -1)
				atomic.AddInt64(&offlineUserCount, 1)
				// log.Infow("logout", "user_id", userId, "sent", msgCount, "active_user_count", offlineUserCount)
				if offlineUserCount >= totalUserCount {
					cancelFunc()
				}
			}()
			switch rand.Intn(8) {
			case 0:
				// 不握手，不关连接
				netDialer := &net.Dialer{}
				netDialer.Timeout = 10 * time.Second
				// websocket.DefaultDialer
				d := &websocket.Dialer{
					Proxy:            http.ProxyFromEnvironment,
					HandshakeTimeout: 45 * time.Second,
					NetDial: func(network, addr string) (net.Conn, error) {
						return netDialer.DialContext(context.TODO(), network, addr)
					},
				}
				u := url.URL{Scheme: "ws", Host: "127.0.0.1:1234", Path: "/"}
				conn, _, err := d.DialContext(context.TODO(), u.String(), nil)
				if err != nil {
					return err
				}
				t, n, err := conn.ReadMessage()
				log.Info(t, n, err)
			default:
				var conn gira.GatewayClient
				conn, err = client.Dial("127.0.0.1:1234",
					//client.WithDebugMode(),
					client.WithIsWebsocket(true),
				)
				if err != nil {
					log.Warnw("client dial fail", "error", err)
					return nil
				}
				// 发2次消息
				for i := 0; i < msgCount; i++ {
					err = conn.Request("", 1, []byte("hello"))
					if err != nil {
						return err
					}
					time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				}
				conn.Close()
			}
			return nil
		})
	}
	go func() {
		for {
			var memStats runtime.MemStats
			// 获取内存占用情况
			runtime.ReadMemStats(&memStats)
			// 打印内存占用信息
			log.Infow("memory usage",
				"alloc", memStats.Alloc,
				"total_alloc", memStats.TotalAlloc,
				"sys", memStats.Sys,
				"mallocs", memStats.Mallocs,
				"frees", memStats.Frees,
				"heap_alloc", memStats.HeapAlloc,
				"heap_sys", memStats.HeapSys,
				"heap_idle", memStats.HeapIdle,
				"heap_inuse", memStats.HeapInuse,
				"heap_released", memStats.HeapReleased,
				"heap_objects", memStats.HeapObjects,
			)
			time.Sleep(1 * time.Second)
		}
	}()
	err = errGroup.Wait()
	if err != nil {
		log.Infow("errGroup", "error", err)
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Logf("Allocated memory: %v bytes", mem.Alloc)
	return
}

type GateHandler_TestOnlineClient struct {
}

func (self *GateHandler_TestOnlineClient) ServeClientStream(s gira.GatewayConn) {
	// var req gira.GatewayMessage
	var err error
	for {
		_, err = s.Recv(context.TODO())
		if err != nil {
			// log.Infow("recv", "error", err)
			break
		} else {
			// log.Infow("recv", "data", string(req.Payload()))
		}
	}
}

// 模拟很多个客户端连接, 维护指定数量的在线人数
func TestOnlineClient(t *testing.T) {
	var err error
	var gateway *Server
	gateway, err = Listen(context.TODO(), ":1234",
		// WithDebugMode(true),
		WithIsWebsocket(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelFunc := context.WithCancel(context.TODO())
	defer cancelFunc()
	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		gateway.Serve(&GateHandler_TestOnlineClient{})
		return nil
	})
	var activeUserCount int64 = 0
	var totalUserCount int64 = 5000
	errGroup.Go(func() error {
		for {
			var memStats runtime.MemStats
			// 获取内存占用情况
			runtime.ReadMemStats(&memStats)
			// 打印内存占用信息
			log.Infow("memory usage",
				"alloc", memStats.Alloc,
				"total_alloc", memStats.TotalAlloc,
				"sys", memStats.Sys,
				"mallocs", memStats.Mallocs,
				"frees", memStats.Frees,
				"heap_alloc", memStats.HeapAlloc,
				"heap_sys", memStats.HeapSys,
				"heap_idle", memStats.HeapIdle,
				"heap_inuse", memStats.HeapInuse,
				"heap_released", memStats.HeapReleased,
				"heap_objects", memStats.HeapObjects,
			)
			log.Infow("user",
				"active_user_count", activeUserCount,
				"ActiveConnectionCount:", gateway.Stat.ActiveConnectionCount,
				"CumulativeSessionCount:", gateway.Stat.CumulativeSessionCount,
			)
			time.Sleep(1 * time.Second)
		}
	})

	for {
		if activeUserCount >= totalUserCount {
			time.Sleep(1 * time.Second)
			continue
		}
		// userId := i
		errGroup.Go(func() error {
			//	log.Info("login", userId)
			atomic.AddInt64(&activeUserCount, 1)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			msgCount := rand.Intn(50)
			defer func() {
				atomic.AddInt64(&activeUserCount, -1)
				// log.Infow("logout", "user_id", userId, "sent", msgCount, "active_user_count", offlineUserCount)
			}()
			switch rand.Intn(8) {
			case -1:
				// 不握手，不关连接
				netDialer := &net.Dialer{}
				netDialer.Timeout = 10 * time.Second
				// websocket.DefaultDialer
				d := &websocket.Dialer{
					Proxy:            http.ProxyFromEnvironment,
					HandshakeTimeout: 45 * time.Second,
					NetDial: func(network, addr string) (net.Conn, error) {
						return netDialer.DialContext(context.TODO(), network, addr)
					},
				}
				u := url.URL{Scheme: "ws", Host: "127.0.0.1:1234", Path: "/"}
				conn, _, err := d.DialContext(context.TODO(), u.String(), nil)
				if err != nil {
					return err
				}
				t, n, err := conn.ReadMessage()
				log.Info(t, n, err)
			default:
				var conn gira.GatewayClient
				conn, err = client.Dial("127.0.0.1:1234",
					//client.WithDebugMode(),
					client.WithIsWebsocket(true),
				)
				if err != nil {
					log.Warnw("client dial fail", "error", err)
					return nil
				}
				// 发2次消息
				for i := 0; i < msgCount; i++ {
					err = conn.Request("", 1, []byte("hello"))
					if err != nil {
						return err
					}
					time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				}
				conn.Close()
			}
			return nil
		})
	}
}

type GateHandler_TestClientClose1 struct {
	RecvMessageCount       int64
	ExpectRecvMessageCount int64
}

func (self *GateHandler_TestClientClose1) ServeClientStream(s gira.GatewayConn) {
	// var req gira.GatewayMessage
	var err error
	for {
		_, err = s.Recv(context.TODO())
		if err != nil {
			// log.Infow("recv", "error", err)
			break
		} else {
			self.RecvMessageCount++
			// log.Infow("recv", "data", string(req.Payload()))
		}
	}
}

// 客户端连接上服务器后，每隔1秒发送1次消息，然后主动关闭
func TestClientClose1(t *testing.T) {
	var err error
	var gateway *Server
	gateway, err = Listen(context.TODO(), ":1234",
		WithDebugMode(false),
		WithIsWebsocket(true))
	if err != nil {
		t.Fatal(err)
	}
	handler := &GateHandler_TestClientClose1{
		ExpectRecvMessageCount: 100,
	}
	errGroup, errCtx := errgroup.WithContext(context.TODO())
	errGroup.Go(func() error {
		go func() {
			gateway.Serve(handler)
		}()
		select {
		case <-errCtx.Done():
			for {
				if gateway.Stat.ActiveSessionCount != 0 {
					time.Sleep(1 * time.Millisecond)
				} else {
					break
				}
			}
			if handler.RecvMessageCount != handler.ExpectRecvMessageCount {
				t.Fatal("recv message fail")
			}
			if gateway.Stat.CumulativeSessionCount != 1 {
				t.Fatal("CumulativeSessionCount fail")
			}
			if gateway.Stat.ActiveSessionCount != 0 {
				t.Fatal("ActiveSessionCount fail")
			}
			if gateway.Stat.CumulativeConnectionCount != 1 {
				t.Fatal("CumulativeConnectionCount fail")
			}
			gateway.Shutdown()
		}
		return nil
	})
	errGroup.Go(func() error {
		time.Sleep(1 * time.Millisecond)
		var conn gira.GatewayClient
		conn, err = client.Dial("127.0.0.1:1234",
			client.WithDebugMode(false),
			client.WithIsWebsocket(true))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < int(handler.ExpectRecvMessageCount); i++ {
			err = conn.Request("", 1, []byte("hello"))
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(1 * time.Millisecond)
		}
		conn.Close()
		return io.EOF
	})
	err = errGroup.Wait()
	if err != nil {
		log.Infow("errGroup", "error", err)
	}
	return
}

type GateHandler2 struct {
}

func (self *GateHandler2) ServeClientStream(s gira.GatewayConn) {
	var req gira.GatewayMessage
	var err error
	req, err = s.Recv(context.TODO())
	if err != nil {
		log.Infow("recv", "error", err)

	} else {
		log.Infow("recv", "data", string(req.Payload()))

	}
	s.Close()
}

// 客户端连接上服务器后，每隔1秒发送1次消息，然后由服务器主动关闭链接
func TestClientClose2(t *testing.T) {
	var err error
	var gateway *Server
	gateway, err = Listen(context.TODO(), ":1234",
		// WithDebugMode(true),
		WithIsWebsocket(true))
	if err != nil {
		t.Fatal(err)
	}
	errGroup, errCtx := errgroup.WithContext(context.TODO())
	errGroup.Go(func() error {
		go func() {
			gateway.Serve(&GateHandler2{})
		}()
		select {
		case <-errCtx.Done():
			gateway.Shutdown()
		}
		return nil
	})
	errGroup.Go(func() error {
		time.Sleep(1 * time.Millisecond)
		var conn gira.GatewayClient
		conn, err = client.Dial("127.0.0.1:1234",
			//  client.WithDebugMode(),
			client.WithIsWebsocket(true))
		if err != nil {
			t.Fatal(err)
		}
		for {
			err = conn.Request("", 1, []byte("hello"))
			if err != nil {
				log.Infow("request", "error", err)
				break
			}
			time.Sleep(1 * time.Second)
		}
		conn.Close()
		return io.EOF
	})
	err = errGroup.Wait()
	if err != nil {
		log.Infow("errGroup", "error", err)

	}
	return
}

type GateHandler_TestServerClose struct {
}

func (self *GateHandler_TestServerClose) ServeClientStream(s gira.GatewayConn) {
	var req gira.GatewayMessage
	var err error
	req, err = s.Recv(context.TODO())
	if err != nil {
		log.Infow("recv", "error", err)
	} else {
		log.Infow("recv", "data", string(req.Payload()))
	}
}

// 客户端连接上服务器后，每隔1秒发送1次消息，然后由服务器主动关闭链接（通过函数返回的方式）
func TestServerClose(t *testing.T) {
	var err error
	var gateway *Server
	gateway, err = Listen(context.TODO(), ":1234",
		WithDebugMode(false),
		WithIsWebsocket(true))
	if err != nil {
		t.Fatal(err)
	}
	errGroup, errCtx := errgroup.WithContext(context.TODO())
	errGroup.Go(func() error {
		go func() {
			gateway.Serve(&GateHandler_TestServerClose{})

		}()
		select {
		case <-errCtx.Done():
			gateway.Shutdown()
		}
		return nil
	})
	errGroup.Go(func() error {
		time.Sleep(1 * time.Millisecond)
		var conn gira.GatewayClient
		conn, err = client.Dial("127.0.0.1:1234",
			client.WithDebugMode(false),
			client.WithIsWebsocket(true))
		if err != nil {
			t.Fatal(err)
		}
		for {
			err = conn.Request("", 1, []byte("hello"))
			if err != nil {
				log.Infow("request", "error", err)
				break
			}
			time.Sleep(1 * time.Millisecond)
		}
		conn.Close()
		return io.EOF
	})
	err = errGroup.Wait()
	if err != nil {
		log.Infow("errGroup", "error", err)

	}
	return
}

type GateHandler_TestClientClose4 struct {
}

func (self *GateHandler_TestClientClose4) ServeClientStream(s gira.GatewayConn) {
	var req gira.GatewayMessage
	var err error
	req, err = s.Recv(context.TODO())
	if err != nil {
		log.Infow("recv", "error", err)

	} else {
		log.Infow("recv", "data", string(req.Payload()))

	}
}

// 客户端连接上服务器后，不进行握手
func TestClientClose4(t *testing.T) {
	var err error
	var gateway *Server
	gateway, err = Listen(context.TODO(), ":1234",
		WithDebugMode(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	errGroup, errCtx := errgroup.WithContext(context.TODO())
	errGroup.Go(func() error {
		go func() {
			gateway.Serve(&GateHandler_TestClientClose4{})
		}()
		select {
		case <-errCtx.Done():
			time.Sleep(1 * time.Second)
			for {
				if gateway.Stat.CumulativeConnectionCount == 1 && gateway.Stat.ActiveConnectionCount == 0 {
					break
				}
			}
			if gateway.Stat.ActiveConnectionCount != 0 {
				t.Fatal("ActiveConnectionCount fail")
			}
			if gateway.Stat.CumulativeConnectionCount != 1 {
				t.Fatal("CumulativeConnectionCount fail")
			}
			if gateway.Stat.CumulativeSessionCount != 0 {
				t.Fatal("CumulativeSessionCount fail")
			}
			if gateway.Stat.HandshakeErrorCount != 1 {
				t.Fatal("HandshakeErrorCount fail")
			}
			gateway.Shutdown()
		}
		return nil
	})
	errGroup.Go(func() error {
		time.Sleep(1 * time.Millisecond)
		var conn net.Conn
		conn, err = net.Dial("tcp", "127.0.0.1:1234")
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		log.Info(n, err)
		return io.EOF
	})
	err = errGroup.Wait()
	if err != nil {
		log.Infow("errGroup", "error", err)
	}
	return
}

type GateHandler_TestServeKick struct {
	ExpectSendMessageCount int64
}

// 发送100个消息马上关闭
func (self *GateHandler_TestServeKick) ServeClientStream(s gira.GatewayConn) {
	//var req gira.GateRequest
	var i int64
	for i = 1; i <= self.ExpectSendMessageCount; i++ {
		s.Push("hello", []byte(fmt.Sprintf("world%d", i)))
	}
	s.Kick("hi")
}

// 客户端连接上服务器后，每隔1秒发送1次消息，然后主动关闭
func TestServeKick(t *testing.T) {
	var err error
	var gateway *Server
	ctx := context.TODO()
	gateway, err = Listen(ctx, ":1234",
		//  WithDebugMode(true),
		WithIsWebsocket(true))
	if err != nil {
		t.Fatal(err)
	}
	handler := &GateHandler_TestServeKick{
		ExpectSendMessageCount: 100,
	}
	var recvMessageCount int64
	errGroup, errCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		go func() {
			gateway.Serve(handler)
		}()
		select {
		case <-errCtx.Done():
			for {
				if gateway.Stat.ActiveSessionCount == 0 && gateway.Stat.CumulativeSessionCount == 1 {
					break
				}
				time.Sleep(1 * time.Millisecond)
			}
			if recvMessageCount != handler.ExpectSendMessageCount {
				t.Fatal("recv count fail", recvMessageCount)
			}
			gateway.Shutdown()
		}
		return nil
	})
	errGroup.Go(func() error {
		time.Sleep(1 * time.Millisecond)
		var conn gira.GatewayClient
		conn, err = client.Dial("127.0.0.1:1234",
			// client.WithDebugMode(),
			client.WithIsWebsocket(true))
		if err != nil {
			t.Fatal(err)
		}
		for {
			_, _, _, _, err := conn.Recv(ctx)
			// typ, route, reqId, data, err := conn.Recv(ctx)
			// log.Infow("recv", "type", typ, "route", route, "req_id", reqId, "data", string(data), "error", err)
			if err != nil {
				break
			} else {
				recvMessageCount++
			}
		}
		conn.Close()

		return io.EOF
	})
	err = errGroup.Wait()
	if err != nil {
		log.Infow("errGroup", "error", err)
	}
	return
}

type GateHandler_TestClientHandshakeTimeout struct {
}

func (self *GateHandler_TestClientHandshakeTimeout) ServeClientStream(s gira.GatewayConn) {
	// var req gira.GatewayMessage
	var err error
	for {
		_, err = s.Recv(context.TODO())
		if err != nil {
			// log.Infow("recv", "error", err)
			break
		} else {
			// log.Infow("recv", "data", string(req.Payload()))
		}
	}
}

// 客户端连接上服务器后，每隔1秒发送1次消息，然后主动关闭
func TestClientHandshakeTimeout(t *testing.T) {
	gateway, err := Listen(context.TODO(), ":1234",
		WithDebugMode(false),
		WithIsWebsocket(true))
	if err != nil {
		t.Fatal(err)
	}
	handler := &GateHandler_TestClientHandshakeTimeout{}
	go func() {
		gateway.Serve(handler)
	}()
	time.Sleep(1 * time.Millisecond)
	_, err = client.Dial("127.0.0.1:1234",
		// client.WithDebugMode(),
		client.WithHandshakeTimeout(time.Duration(0)*time.Second),
		client.WithIsWebsocket(true))
	if err == nil {
		t.Fatal("dail success")
	} else {
		log.Infow("dail fail", "error", err)
	}
	return
}
