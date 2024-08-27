package admin

import (
	"context"

	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	"github.com/Lyndon-Zhang/gira/service/admin/adminpb"
)

type AdminService struct {
	ctx         context.Context
	adminServer *admin_server
}

type admin_server struct {
	adminpb.UnimplementedAdminServer
}

func (self *admin_server) ReloadResource(context.Context, *adminpb.ReloadResourceRequest) (*adminpb.ReloadResourceResponse, error) {
	resp := &adminpb.ReloadResourceResponse{}
	if err := facade.ReloadResource(); err != nil {
		return nil, err
	}
	return resp, nil
}

func NewService() *AdminService {
	return &AdminService{
		adminServer: &admin_server{},
	}
}

func (self *AdminService) OnStop() error {
	return nil
}

func (self *AdminService) Serve() error {
	<-self.ctx.Done()
	return nil
}

func (self *AdminService) OnStart(ctx context.Context) error {
	self.ctx = ctx
	// 注册服务名字
	if _, err := facade.RegisterServiceName(GetServiceName()); err != nil {
		return err
	}
	// 注册handler
	adminpb.RegisterAdminServer(facade.GrpcServer(), self.adminServer)
	return nil
}

func GetServiceName() string {
	return facade.NewServiceName(adminpb.AdminServerName, service_options.WithAsAppServiceOption())
}
