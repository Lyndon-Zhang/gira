package gira

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Lyndon-Zhang/gira/errors"
)

// 环境枚举
const (
	Env_DEV = "dev"
	Env_QA  = "qa"
	Env_prd = "prd"
)

/*
	生命周期

OnFrameworkInit

	|

OnFrameworkConfigLoad

	|

OnConfigLoad

	|

OnFrameworkCreate

	|

OnCreate

	|

OnFrameworkStart

	|

OnStart

	|

OnStop

	|

OnFrameworkStop
*/

type Framework interface {
	OnFrameworkConfigLoad(c *Config) error
	OnFrameworkCreate() error
	OnFrameworkStart() error
	OnFrameworkStop() error
}

type Application interface {
	// ======= 生命周期回调 ===========
	// 配置加载完成后接收通知
	OnConfigLoad(c *Config) error
	OnCreate() error
	OnStart() error
	OnStop() error
}

type ApplicationFramework interface {
	OnFrameworkInit() []Framework
}
type ApplicationArgs struct {
	AppType             string /// 服务名
	AppId               int32  /// 服务id
	BuildTime           int64
	AppVersion          string
	RespositoryVersion1 string
	Application         Application
}

type Runtime interface {
	// ======= 状态数据 ===========
	GetConfig() *Config
	GetAppType() string
	GetAppName() string
	GetAppFullName() string
	GetEnv() string
	GetZone() string
	GetAppId() int32
	GetLogDir() string
	GetWorkDir() string
	GetAppVersion() string
	GetBuildTime() int64
	GetUpTime() int64

	// ======= 同步接口 ===========
	Wait() error
	Stop() error
	Context() context.Context
	Go(f func() error)
	Done() <-chan struct{}

	Frameworks() []Framework
	Application() Application

	// ======= 组件 ===========
	GetServiceContainer() ServiceContainer
	GetPlatformSdk() PlatformSdk
	GetCron() Cron
	GetGrpcServer() GrpcServer
	GetRegistry() Registry
	GetRegistryClient() RegistryClient
}

// 当前正在运行的应用
var runtime Runtime

func GetRuntime() Runtime {
	return runtime
}
func GetApplication() Application {
	return runtime.Application()
}

// 创建完成时回调
func OnApplicationCreate(r Runtime) {
	runtime = r
}

func FormatAppFullName(appType string, appId int32, zone string, env string) string {
	return fmt.Sprintf("%s_%s_%s_%d", appType, zone, env, appId)
}

func ParseAppFullName(fullName string) (name string, id int32, err error) {
	pats := strings.Split(string(fullName), "_")
	if len(pats) != 4 {
		err = errors.New("invalid app full name", "full_name", fullName)
		return
	}
	name = pats[0]
	var v int
	if v, err = strconv.Atoi(pats[3]); err == nil {
		id = int32(v)
	}
	return
}
