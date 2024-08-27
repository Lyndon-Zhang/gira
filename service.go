package gira

import "context"

// 服务
type Service interface {
	// 服务启动
	OnStart(ctx context.Context) error
	// 服务函数
	Serve() error
	// 服务停止
	OnStop() error
}

// 服务容器
type ServiceContainer interface {
	// 启动服务
	StartService(name string, service Service) error
	// 停止服务
	StopService(service Service) error
}
