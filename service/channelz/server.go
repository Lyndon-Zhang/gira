package channelz

import (
	"context"

	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	"github.com/Lyndon-Zhang/gira/service/channelz/grpc_channelz"
	"google.golang.org/grpc/admin"
)

type ChannelzService struct {
	ctx         context.Context
	cleanupFunc func()
}

func NewService() *ChannelzService {
	return &ChannelzService{}
}

func (self *ChannelzService) OnStop() error {
	self.cleanupFunc()
	return nil
}

func (self *ChannelzService) Serve() error {
	<-self.ctx.Done()
	return nil
}

func (self *ChannelzService) OnStart(ctx context.Context) error {
	self.ctx = ctx

	// 注册服务名字
	if _, err := facade.RegisterServiceName(GetServiceName()); err != nil {
		return err
	}
	// 注册handler
	if cleanup, err := admin.Register(facade.GrpcServer()); err != nil {
		return err
	} else {
		self.cleanupFunc = cleanup
	}
	return nil
}

func GetServiceName() string {
	return facade.NewServiceName(grpc_channelz.ChannelzServerName, service_options.WithAsAppServiceOption())
}
