package grpc

import (
	"context"
	"net"
	"sync"

	log "github.com/Lyndon-Zhang/gira/corelog"

	"github.com/Lyndon-Zhang/gira"
	"google.golang.org/grpc"
)

// http://wzmmmmj.com/2020/09/06/grpc-stream/

type Server struct {
	config     gira.GrpcConfig
	server     *grpc.Server
	ctx        context.Context
	cancelFunc context.CancelFunc
	servers    map[string]interface{}
	listener   net.Listener
	mu         sync.Mutex
}

func NewConfigServer(config gira.GrpcConfig) (*Server, error) {
	opts := []grpc.ServerOption{grpc.NumStreamWorkers(config.Workers)}
	self := &Server{
		config:  config,
		server:  grpc.NewServer(opts...),
		servers: make(map[string]interface{}),
	}
	if config.EnabledTrace {
		grpc.EnableTracing = true
	}
	return self, nil
}

// 实现接口 grpc.ServiceRegistrar
// 将impl保存起来，转发的时候要用到
func (self *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	self.server.RegisterService(desc, impl)
	self.mu.Lock()
	self.servers[desc.ServiceName] = impl
	self.mu.Unlock()
}

func (self *Server) GetServer(name string) (svr interface{}, ok bool) {
	self.mu.Lock()
	svr, ok = self.servers[name]
	self.mu.Unlock()
	return
}

func (self *Server) Listen() error {
	if listener, err := net.Listen("tcp", self.config.Address); err != nil {
		return err
	} else {
		self.listener = listener
		return nil
	}
}

func (self *Server) Serve(ctx context.Context) error {
	log.Debugw("grpc server started", "addr", self.config.Address)
	self.ctx, self.cancelFunc = context.WithCancel(ctx)
	go func() {
		<-self.ctx.Done()
		self.server.GracefulStop()
	}()

	err := self.server.Serve(self.listener)
	log.Debugw("gpc server on stop2", "error", err)
	return err
}

func (self *Server) Stop() error {
	if self.ctx == nil {
		return nil
	}
	log.Debugw("gpc server on stop1")
	self.cancelFunc()
	return nil
}
