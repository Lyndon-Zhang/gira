package app

/*

实现gira.Application接口的服务端程序

*/

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	gruntime "runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Lyndon-Zhang/gira/behaviorlog"
	"github.com/Lyndon-Zhang/gira/codes"
	"github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/gins"
	"github.com/Lyndon-Zhang/gira/log"
	"github.com/Lyndon-Zhang/gira/service"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/db"
	"github.com/Lyndon-Zhang/gira/gate"
	"github.com/Lyndon-Zhang/gira/grpc"
	"github.com/Lyndon-Zhang/gira/platform"
	"github.com/Lyndon-Zhang/gira/proj"
	"github.com/Lyndon-Zhang/gira/registry"
	"github.com/Lyndon-Zhang/gira/registryclient"
	admin_service "github.com/Lyndon-Zhang/gira/service/admin"
	"github.com/Lyndon-Zhang/gira/service/admin/adminpb"
	channelz_service "github.com/Lyndon-Zhang/gira/service/channelz"
	peer_service "github.com/Lyndon-Zhang/gira/service/peer"

	_ "net/http/pprof"

	"github.com/robfig/cron"
	"golang.org/x/net/trace"
	"golang.org/x/sync/errgroup"
)

const (
	application_status_started = 1
	application_status_stopped = 2
)

// / @Component
type Runtime struct {
	zone               string // 区名 wc|qq|hw|quick
	env                string // dev|local|qa|prd
	appId              int32
	appType            string /// 服务类型
	appName            string /// 服务名
	appFullName        string /// 完整的服务名 Name_Id
	cancelFunc         context.CancelFunc
	ctx                context.Context
	errCtx             context.Context
	errGroup           *errgroup.Group
	resourceLoader     gira.ResourceLoader
	resourceSource     gira.ResourceSource
	config             *gira.Config
	chQuit             chan struct{}
	status             int64
	appVersion         string
	buildTime          int64
	upTime             int64
	projectFilePath    string /// 配置文件绝对路径, gira.yaml
	configDir          string /// config目录
	envDir             string /// env目录
	runConfigFilePath  string /// 运行时的配置文件
	workDir            string /// 工作目录
	logDir             string /// 日志目录
	runDir             string /// 运行目录
	application        gira.Application
	frameworks         []gira.Framework
	httpServer         *gins.HttpServer
	registry           *registry.Registry
	registryClient     *registryclient.RegistryClient
	dbClients          map[string]gira.DbClient
	gameDbClient       gira.DbClient
	logDbClient        gira.DbClient
	behaviorDbClient   gira.DbClient
	accountDbClient    gira.DbClient
	statDbClient       gira.DbClient
	resourceDbClient   gira.DbClient
	accountCacheClient gira.DbClient
	gameCacheClient    gira.DbClient
	adminCacheClient   gira.DbClient
	adminDbClient      gira.DbClient
	platformSdk        *platform.PlatformSdk
	gate               *gate.Server
	grpcServer         *grpc.Server
	serviceContainer   *service.ServiceContainer
	cron               *cron.Cron
}

func newRuntime(args gira.ApplicationArgs) *Runtime {
	ctx, cancelFunc := context.WithCancel(context.Background())
	errGroup, errCtx := errgroup.WithContext(ctx)
	runtime := &Runtime{
		appVersion:       args.AppVersion,
		buildTime:        args.BuildTime,
		appId:            args.AppId,
		application:      args.Application,
		frameworks:       make([]gira.Framework, 0),
		appType:          args.AppType,
		appName:          fmt.Sprintf("%s_%d", args.AppType, args.AppId),
		ctx:              ctx,
		cancelFunc:       cancelFunc,
		errCtx:           errCtx,
		errGroup:         errGroup,
		chQuit:           make(chan struct{}, 1),
		serviceContainer: service.NewContainer(ctx),
	}
	return runtime
}

func (runtime *Runtime) init() error {
	var err error
	application := runtime.application
	// 初始化
	rand.Seed(time.Now().UnixNano())
	runtime.upTime = time.Now().Unix()
	// 项目环境,目录初始化
	runtime.workDir = proj.Dir.ProjectDir
	if err := os.Chdir(runtime.workDir); err != nil {
		return err
	}
	runtime.projectFilePath = proj.Dir.ProjectConfFilePath
	if _, err := os.Stat(runtime.projectFilePath); err != nil {
		return err
	}
	runtime.envDir = proj.Dir.EnvDir
	runtime.configDir = proj.Dir.ConfigDir
	if _, err := os.Stat(runtime.configDir); err != nil {
		return err
	}
	runtime.runDir = proj.Dir.RunDir
	if _, err := os.Stat(runtime.runDir); err != nil {
		if err := os.Mkdir(runtime.runDir, 0755); err != nil {
			return err
		}
	}
	runtime.runConfigFilePath = filepath.Join(runtime.runDir, fmt.Sprintf("%s", runtime.appFullName))
	runtime.logDir = proj.Dir.LogDir
	// 初始化框架
	if f, ok := application.(gira.ApplicationFramework); ok {
		runtime.frameworks = f.OnFrameworkInit()
	}
	// 读应用配置文件
	if c, err := proj.LoadApplicationConfig(runtime.appType, runtime.appId); err != nil {
		return err
	} else {
		runtime.env = c.Env
		runtime.zone = c.Zone
		runtime.appFullName = gira.FormatAppFullName(runtime.appType, runtime.appId, runtime.zone, runtime.env)
		runtime.config = c
	}
	// 加载配置回调
	for _, fw := range runtime.frameworks {
		if err := fw.OnFrameworkConfigLoad(runtime.config); err != nil {
			return err
		}
	}
	if err := runtime.application.OnConfigLoad(runtime.config); err != nil {
		return err
	}
	// 初始化日志
	if runtime.config.CoreLog != nil {
		if err = corelog.Config(*runtime.config.CoreLog); err != nil {
			return err
		}
		if runtime.config.CoreLog.TraceErrorStack {
			codes.SetLogger(corelog.GetDefaultLogger())
		}
	}
	if runtime.config.BehaviorLog != nil {
		if err = behaviorlog.Config(*runtime.config.BehaviorLog); err != nil {
			return err
		}
	}
	if runtime.config.Log != nil {
		if err = log.Config(*runtime.config.Log); err != nil {
			return err
		}
	}
	gruntime.GOMAXPROCS(runtime.config.Thread)
	return nil
}

func (runtime *Runtime) start() (err error) {
	// 初始化
	if err = runtime.init(); err != nil {
		return
	}
	// 设置全局对象
	gira.OnApplicationCreate(runtime)
	if err = runtime.onCreate(); err != nil {
		return
	}
	if err = runtime.onStart(); err != nil {
		return
	}
	runtime.Go(func() error {
		quit := make(chan os.Signal, 1)
		defer close(quit)
		signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
		for {
			select {
			// 被动中断
			case s := <-quit:
				corelog.Infow("runtime recv signal", "signal", s)
				switch s {
				case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
					corelog.Infow("+++++++++++++++++++++++++++++")
					corelog.Infow("++    signal interrupt    +++", "signal", s)
					corelog.Infow("+++++++++++++++++++++++++++++")
					runtime.stop()
					corelog.Info("runtime interrupt end")
					return errors.ErrInterrupt
				case syscall.SIGUSR1:
				case syscall.SIGUSR2:
				default:
				}
			// 主动停止
			case <-runtime.chQuit:
				corelog.Info("runtime stop begin")
				runtime.stop()
				corelog.Info("runtime stop end")
				return nil
				// 有服务出错
				// case <-runtime.errCtx.Done():
				// 	corelog.Info("runtime errgroup", runtime.errCtx.Err())
				// 	runtime.stop()
				// 	return nil
			}

		}
	})
	runtime.status = application_status_started
	return nil
}

// 主动关闭, 启动成功后才可以调用
func (runtime *Runtime) Stop() error {
	if !atomic.CompareAndSwapInt64(&runtime.status, application_status_started, application_status_stopped) {
		return nil
	}
	runtime.chQuit <- struct{}{}
	return nil
}

func (runtime *Runtime) stop() {
	// runtime stop
	runtime.application.OnStop()
	// framework stop
	for _, fw := range runtime.frameworks {
		corelog.Info("framework on stop", "name")
		if err := fw.OnFrameworkStop(); err != nil {
			corelog.Warnw("framework on stop fail", "error", err)
		}
	}
	// service stop
	runtime.serviceContainer.Stop()
	if runtime.registry != nil {
		runtime.registry.Stop()
	}
	if runtime.grpcServer != nil {
		runtime.grpcServer.Stop()
	}
	if runtime.httpServer != nil {
		runtime.httpServer.Stop()
	}
	if runtime.gate != nil {
		runtime.gate.Shutdown()
	}
	runtime.cancelFunc()
}

func (runtime *Runtime) onStart() (err error) {
	application := runtime.application
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
		if err != nil {
			corelog.Errorw("+++++++++++++++++++++++++++++")
			corelog.Errorw("++         启动异常       +++", "error", err)
			corelog.Errorw("+++++++++++++++++++++++++++++")
			runtime.stop()
			runtime.errGroup.Wait()
		}
	}()
	// ==== cron ================
	runtime.cron.Start()

	// ==== registryClient ================
	if runtime.registryClient != nil {
		if err = runtime.registryClient.StartAsClient(); err != nil {
			return
		}
	}
	// ==== registry ================
	if runtime.registry != nil {
		if err = runtime.registry.StartAsMember(); err != nil {
			return
		}
	}
	// ==== grpc ================
	if runtime.grpcServer != nil {
		if err := runtime.grpcServer.Listen(); err != nil {
			return err
		}
		if runtime.registry != nil && runtime.config.Module.Grpc.Resolver {
			if err := runtime.registry.StartReslover(); err != nil {
				return err
			}
		}
	}
	// runtime.errGroup.Go(func() error {
	// 	return runtime.registry.Serve(runtime.ctx)
	// })
	// }
	// ==== http ================
	if runtime.httpServer != nil {
		runtime.errGroup.Go(func() error {
			return runtime.httpServer.Serve()
		})
	}
	// ==== service ================
	if runtime.grpcServer != nil {
		// ==== admin service ================
		{
			service := admin_service.NewService()
			if err = runtime.serviceContainer.StartService("admin", service); err != nil {
				return
			}
		}
		// ==== peer service ================
		{
			service := peer_service.NewService()
			if err = runtime.serviceContainer.StartService("peer", service); err != nil {
				return
			}
		}
		if runtime.config.Module.Grpc.Admin {
			service := channelz_service.NewService()
			if err = runtime.serviceContainer.StartService("channelz", service); err != nil {
				return
			}
		}
	}
	// ==== framework start ================
	for _, fw := range runtime.frameworks {
		if err = fw.OnFrameworkStart(); err != nil {
			return
		}
	}
	// ==== runtime start ================
	if err = runtime.application.OnStart(); err != nil {
		return
	}
	// ==== grpc ================
	if runtime.grpcServer != nil {
		runtime.errGroup.Go(func() error {
			err := runtime.grpcServer.Serve(runtime.ctx)
			return err
		})
	}
	// ==== registry ================
	if runtime.registry != nil {
		runtime.errGroup.Go(func() error {
			var peerWatchHandlers []gira.PeerWatchHandler
			var localPlayerWatchHandlers []gira.LocalPlayerWatchHandler
			var serviceWatchHandlers []gira.ServiceWatchHandler
			for _, fw := range runtime.frameworks {
				if handler, ok := fw.(gira.PeerWatchHandler); ok {
					peerWatchHandlers = append(peerWatchHandlers, handler)
				}
				if handler, ok := fw.(gira.LocalPlayerWatchHandler); ok {
					localPlayerWatchHandlers = append(localPlayerWatchHandlers, handler)
				}
				if handler, ok := fw.(gira.ServiceWatchHandler); ok {
					serviceWatchHandlers = append(serviceWatchHandlers, handler)
				}
			}
			if handler, ok := runtime.application.(gira.PeerWatchHandler); ok {
				peerWatchHandlers = append(peerWatchHandlers, handler)
			}
			if handler, ok := runtime.application.(gira.LocalPlayerWatchHandler); ok {
				localPlayerWatchHandlers = append(localPlayerWatchHandlers, handler)
			}
			if handler, ok := runtime.application.(gira.ServiceWatchHandler); ok {
				serviceWatchHandlers = append(serviceWatchHandlers, handler)
			}
			return runtime.registry.Watch(peerWatchHandlers, localPlayerWatchHandlers, serviceWatchHandlers)
		})
	}
	if runtime.gate != nil {
		runtime.errGroup.Go(func() error {
			var handler gira.GatewayHandler
			if h, ok := application.(gira.GatewayHandler); ok {
				handler = h
			} else {
				for _, fw := range runtime.frameworks {
					if h, ok = fw.(gira.GatewayHandler); ok {
						handler = h
						break
					}
				}
			}
			if handler == nil {
				return errors.ErrGateHandlerNotImplement
			}
			err := runtime.gate.Serve(handler)
			return err
		})
	}
	runtime.errGroup.Go(func() error {
		return runtime.serviceContainer.Serve()
	})
	return nil
}

func (runtime *Runtime) onCreate() error {
	// log.Info("application", app.FullName, "start")
	application := runtime.application

	// ==== pprof ================
	if runtime.config.Pprof.Port != 0 {
		go func() {
			corelog.Infof("pprof start at http://%s:%d/debug/pprof", runtime.config.Pprof.Bind, runtime.config.Pprof.Port)
			if runtime.config.Pprof.EnabledTrace {
				trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
					return true, true
				}
			}
			http.ListenAndServe(fmt.Sprintf("%s:%d", runtime.config.Pprof.Bind, runtime.config.Pprof.Port), nil)
		}()
	}
	// ==== cron ================
	runtime.cron = cron.New()
	// ==== registry ================
	if runtime.config.Module.Etcd != nil {
		if r, err := registry.NewConfigRegistry(runtime.ctx, runtime.config.Module.Etcd); err != nil {
			return err
		} else {
			runtime.registry = r
		}
	}
	// ==== registry client ================
	if runtime.config.Module.EtcdClient != nil {
		if r, err := registryclient.NewConfigRegistryClient(runtime.ctx, runtime.config.Module.EtcdClient, runtime.appId, runtime.appFullName); err != nil {
			return err
		} else {
			runtime.registryClient = r
		}
	}

	// ==== db ================
	runtime.dbClients = make(map[string]gira.DbClient)
	for name, c := range runtime.config.Db {
		if client, err := db.NewConfigDbClient(runtime.ctx, name, *c); err != nil {
			return err
		} else {
			runtime.dbClients[name] = client
			if name == gira.GAMEDB_NAME {
				runtime.gameDbClient = client
			} else if name == gira.RESOURCEDB_NAME {
				runtime.resourceDbClient = client
			} else if name == gira.STATDB_NAME {
				runtime.statDbClient = client
			} else if name == gira.ACCOUNTDB_NAME {
				runtime.accountDbClient = client
			} else if name == gira.LOGDB_NAME {
				runtime.logDbClient = client
			} else if name == gira.BEHAVIORDB_NAME {
				runtime.behaviorDbClient = client
			} else if name == gira.ACCOUNTCACHE_NAME {
				runtime.accountCacheClient = client
			} else if name == gira.ADMINCACHE_NAME {
				runtime.adminCacheClient = client
			} else if name == gira.ADMINDB_NAME {
				runtime.adminDbClient = client
			} else if name == gira.GAMECACHE_NAME {
				runtime.gameCacheClient = client
			}
		}
	}

	// ==== 加载resource ================
	if resourceComponent, ok := runtime.application.(gira.ResourceSource); ok {
		runtime.resourceSource = resourceComponent
		resourceLoader := resourceComponent.GetResourceLoader()
		if resourceLoader != nil {
			runtime.resourceLoader = resourceLoader
			if err := runtime.resourceLoader.LoadResource(runtime.ctx, runtime.resourceDbClient, path.Join("resource", "conf"), runtime.config.Resource.Compress); err != nil {
				return err
			} else {
				resourceComponent.OnResourcePostLoad(false)
			}
		}
	}

	// ==== grpc ================
	if runtime.config.Module.Grpc != nil {
		if s, err := grpc.NewConfigServer(*runtime.config.Module.Grpc); err != nil {
			return err
		} else {
			runtime.grpcServer = s
		}
	}

	// ==== sdk================
	if runtime.config.Module.Plat != nil {
		runtime.platformSdk = platform.NewConfigSdk(*runtime.config.Module.Plat)
	}

	// ==== http ================
	if runtime.config.Module.Http != nil {
		if handler, ok := application.(gira.HttpHandler); !ok {
			return errors.ErrHttpHandlerNotImplement
		} else {
			router := handler.HttpHandler()
			if httpServer, err := gins.NewConfigHttpServer(runtime.ctx, *runtime.config.Module.Http, router); err != nil {
				return err
			} else {
				runtime.httpServer = httpServer
			}
		}
	}

	// ==== gateway ================
	if runtime.config.Module.Gateway != nil {
		var handler gira.GatewayHandler
		if h, ok := application.(gira.GatewayHandler); ok {
			handler = h
		} else {
			for _, fw := range runtime.frameworks {
				if h, ok = fw.(gira.GatewayHandler); ok {
					handler = h
					break
				}
			}
		}
		if handler == nil {
			return errors.ErrGateHandlerNotImplement
		}
		if gate, err := gate.NewConfigServer(runtime.ctx, *runtime.config.Module.Gateway); err != nil {
			return err
		} else {
			runtime.gate = gate
		}
	}

	// ==== framework create ================
	for _, fw := range runtime.frameworks {
		if err := fw.OnFrameworkCreate(); err != nil {
			return err
		}
	}

	// ==== runtime create ================
	if err := application.OnCreate(); err != nil {
		return err
	}
	return nil
}

// 等待中断
func (runtime *Runtime) Wait() error {
	err := runtime.errGroup.Wait()
	corelog.Infow("runtime down", "full_name", runtime.appFullName, "error", err)
	return err
}

// 负载实现gira声明的接口和启动的模块

// ================== implement gira.Application ==================
// 返回配置
func (runtime *Runtime) GetConfig() *gira.Config {
	return runtime.config
}

// 返回构建版本
func (runtime *Runtime) GetAppVersion() string {
	return runtime.appVersion

}

// 返回构建时间
func (runtime *Runtime) GetBuildTime() int64 {
	return runtime.buildTime
}
func (runtime *Runtime) GetUpTime() int64 {
	return time.Now().Unix() - runtime.upTime
}

// 返回应用id
func (runtime *Runtime) GetAppId() int32 {
	return runtime.appId
}

// 返回应用类型
func (runtime *Runtime) GetAppType() string {
	return runtime.appType
}

// 返回应用名
func (runtime *Runtime) GetAppName() string {
	return runtime.appName
}

// 返回应用全名
func (runtime *Runtime) GetAppFullName() string {
	return runtime.appFullName
}

func (runtime *Runtime) GetWorkDir() string {
	return runtime.workDir
}

func (runtime *Runtime) GetLogDir() string {
	return runtime.logDir
}

func (runtime *Runtime) GetZone() string {
	return runtime.zone
}

func (runtime *Runtime) GetEnv() string {
	return runtime.env
}

// ================== context =========================

func (runtime *Runtime) Context() context.Context {
	return runtime.ctx
}

func (runtime *Runtime) Quit() {
	runtime.cancelFunc()
}

func (runtime *Runtime) Done() <-chan struct{} {
	return runtime.ctx.Done()
}

func (runtime *Runtime) Go(f func() error) {
	runtime.errGroup.Go(f)
}

func (runtime *Runtime) Err() error {
	return runtime.errCtx.Err()
}

// ================== application =========================
func (runtime *Runtime) Application() gira.Application {
	return runtime.application
}

// ================== framework =========================
// 返回框架列表
func (runtime *Runtime) Frameworks() []gira.Framework {
	return runtime.frameworks
}

// ================== gira.SdkComponent ==================
func (runtime *Runtime) GetPlatformSdk() gira.PlatformSdk {
	if runtime.platformSdk == nil {
		return nil
	} else {
		return runtime.platformSdk
	}
}

// ================== gira.RegistryComponent ==================
func (runtime *Runtime) GetRegistry() gira.Registry {
	if runtime.registry == nil {
		return nil
	} else {
		return runtime.registry
	}
}

func (runtime *Runtime) GetRegistryClient() gira.RegistryClient {
	if runtime.registryClient == nil {
		return nil
	} else {
		return runtime.registryClient
	}
}

// ================== gira.ServiceContainer ==================
func (runtime *Runtime) GetServiceContainer() gira.ServiceContainer {
	if runtime.serviceContainer == nil {
		return nil
	} else {
		return runtime.serviceContainer
	}
}

// ================== gira.GrpcServerComponent ==================
func (runtime *Runtime) GetGrpcServer() gira.GrpcServer {
	if runtime.grpcServer == nil {
		return nil
	} else {
		return runtime.grpcServer
	}
}

// ================== implement gira.ResourceComponent ==================
func (runtime *Runtime) GetResourceLoader() gira.ResourceLoader {
	return runtime.resourceLoader
}

// ================== implement gira.AdminClient ==================
// 重载配置
func (runtime *Runtime) BroadcastReloadResource(ctx context.Context, name string) (result gira.BroadcastReloadResourceResult, err error) {
	req := &adminpb.ReloadResourceRequest{
		Name: name,
	}
	result, err = adminpb.DefaultAdminClients.Broadcast().ReloadResource(ctx, req)
	return
}

// ================== implement gira.DbClientComponent ==================
func (runtime *Runtime) GetAccountDbClient() gira.DbClient {
	if runtime.accountDbClient == nil {
		return nil
	} else {
		return runtime.accountDbClient
	}
}

func (runtime *Runtime) GetGameDbClient() gira.DbClient {
	if runtime.gameDbClient == nil {
		return nil
	} else {
		return runtime.gameDbClient
	}
}

func (runtime *Runtime) GetLogDbClient() gira.DbClient {
	if runtime.logDbClient == nil {
		return nil
	} else {
		return runtime.logDbClient
	}
}

func (runtime *Runtime) GetBehaviorDbClient() gira.DbClient {
	if runtime.behaviorDbClient == nil {
		return nil
	} else {
		return runtime.behaviorDbClient
	}
}

func (runtime *Runtime) GetStatDbClient() gira.DbClient {
	if runtime.statDbClient == nil {
		return nil
	} else {
		return runtime.statDbClient
	}
}

func (runtime *Runtime) GetAccountCacheClient() gira.DbClient {
	if runtime.accountCacheClient == nil {
		return nil
	} else {
		return runtime.accountCacheClient
	}
}

func (runtime *Runtime) GetGameCacheClient() gira.DbClient {
	if runtime.gameCacheClient == nil {
		return nil
	} else {
		return runtime.gameCacheClient
	}
}

func (runtime *Runtime) GetAdminCacheClient() gira.DbClient {
	if runtime.adminCacheClient == nil {
		return nil
	} else {
		return runtime.adminCacheClient
	}
}

func (runtime *Runtime) GetResourceDbClient() gira.DbClient {
	if runtime.resourceDbClient == nil {
		return nil
	} else {
		return runtime.resourceDbClient
	}
}

func (runtime *Runtime) GetAdminDbClient() gira.DbClient {
	if runtime.adminDbClient == nil {
		return nil
	} else {
		return runtime.adminDbClient
	}
}

// ================== gira.Cron ==================
func (runtime *Runtime) GetCron() gira.Cron {
	if runtime.cron == nil {
		return nil
	} else {
		return runtime.cron
	}
}
