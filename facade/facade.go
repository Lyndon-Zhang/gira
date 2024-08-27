package facade

import (
	"context"
	"path"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/options/service_options"
)

// 返回app配置
func GetConfig() *gira.Config {
	return gira.GetRuntime().GetConfig()
}

// 返回app context
func Context() context.Context {
	return gira.GetRuntime().Context()
}

// 返回app构建版本
func GetAppVersion() string {
	return gira.GetRuntime().GetAppVersion()
}

// 返回app构建时间
func GetBuildTime() int64 {
	return gira.GetRuntime().GetBuildTime()
}

// 返回resource版本
func GetResVersion() string {
	application := gira.GetApplication()
	if c, ok := application.(gira.ResourceSource); !ok {
		return ""
	} else if r := c.GetResourceLoader(); r == nil {
		return ""
	} else {
		return r.GetResVersion()
	}
}

// 返回resource loader版本
func GetLoaderVersion() string {
	application := gira.GetApplication()
	if c, ok := application.(gira.ResourceSource); !ok {
		return ""
	} else if r := c.GetResourceLoader(); r == nil {
		return ""
	} else {
		return r.GetLoaderVersion()
	}
}

// 返回启动时间
func GetUpTime() int64 {
	return gira.GetRuntime().GetUpTime()
}

// 返回app id
func GetAppId() int32 {
	return gira.GetRuntime().GetAppId()
}

// 返回app全名
func GetAppFullName() string {
	return gira.GetRuntime().GetAppFullName()
}

// 返回当前所在的区
func GetZone() string {
	return gira.GetRuntime().GetZone()
}

// 返回当前环境
func GetEnv() string {
	return gira.GetRuntime().GetEnv()
}

// 是否开发环境
func IsDevEnv() bool {
	return gira.GetRuntime().GetEnv() == gira.Env_DEV
}

// 是否测试环境
func IsQaEnv() bool {
	return gira.GetRuntime().GetEnv() == gira.Env_QA
}

// 是否生产环境
func IsPrdEnv() bool {
	return gira.GetRuntime().GetEnv() == gira.Env_prd
}

// 返回app类型
func GetAppType() string {
	return gira.GetRuntime().GetAppType()
}

func Go(f func() error) {
	gira.GetRuntime().Go(f)
}

func Done() <-chan struct{} {
	return gira.GetRuntime().Done()
}

func GetLogDir() string {
	return gira.GetRuntime().GetLogDir()
}

func GetWorkDir() string {
	return gira.GetRuntime().GetWorkDir()
}

func Stop() error {
	return gira.GetRuntime().Stop()
}

func Wait() error {
	return gira.GetRuntime().Wait()
}

// ================= resource component =============================
// 重载配置
func ReloadResource() error {
	application := gira.GetApplication()
	runtime := gira.GetRuntime()
	if s, ok := application.(gira.ResourceSource); !ok {
		return errors.ErrResourceManagerNotImplement
	} else if r := s.GetResourceLoader(); r == nil {
		return errors.ErrResourceManagerNotImplement
	} else {
		s.OnResourcePreLoad(true)
		if err := r.ReloadResource(runtime.Context(), GetResourceDbClient(), path.Join("resource", "conf"), GetConfig().Resource.Compress); err != nil {
			return err
		} else {
			s.OnResourcePostLoad(true)
			return nil
		}
	}
}

// 广播重载配置
func BroadcastReloadResource(ctx context.Context, name string) (result gira.BroadcastReloadResourceResult, err error) {
	application := gira.GetRuntime()
	if h, ok := application.(gira.AdminClient); !ok {
		err = errors.ErrAdminClientNotImplement
		return
	} else {
		result, err = h.BroadcastReloadResource(ctx, name)
		return
	}
}

// ================= registry =============================
// 解锁user
func UnlockLocalUser(userId string) (*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r == nil {
		return nil, errors.ErrRegistryNOtImplement
	} else {
		return r.UnlockLocalUser(userId)
	}
}

// 锁定user
func LockLocalUser(userId string) (*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r == nil {
		return nil, errors.ErrRegistryNOtImplement
	} else {
		return r.LockLocalUser(userId)
	}
}

// 查找user所在的节点
func WhereIsUser(userId string) (*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r != nil {
		return r.WhereIsUser(userId)
	} else if r := application.GetRegistryClient(); r != nil {
		return r.WhereIsUser(userId)
	} else {
		return nil, errors.ErrRegistryNOtImplement
	}
}

func ListLocalUser() []string {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r != nil {
		return r.ListLocalUser()
	} else {
		return nil
	}
}

func SelfPeer(userId string) *gira.Peer {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r == nil {
		return nil
	} else {
		return r.SelfPeer()
	}
}

// 查找节点位置
func WhereIsPeer(appFullName string) (*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r != nil {
		return r.WhereIsPeer(appFullName)
	} else if r := application.GetRegistryClient(); r != nil {
		return r.WhereIsPeer(appFullName)
	} else {
		return nil, errors.ErrRegistryNOtImplement
	}
}

func ListPeerKvs() (peers map[string]string, err error) {
	application := gira.GetRuntime()
	if r := application.GetRegistryClient(); r == nil {
		err = errors.ErrRegistryNOtImplement
		return
	} else {
		peers, err = r.ListPeerKvs()
		return
	}
}

// 遍历节点
func RangePeers(f func(k any, v any) bool) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r == nil {
		return
	} else {
		r.RangePeers(f)
	}
}

func ListServiceKvs() (peers map[string][]string, err error) {
	application := gira.GetRuntime()
	if r := application.GetRegistryClient(); r == nil {
		err = errors.ErrRegistryNOtImplement
		return
	} else {
		peers, err = r.ListServiceKvs()
		return
	}
}

// 构造服务名
func NewServiceName(serviceName string, opt ...service_options.RegisterOption) string {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r != nil {
		return r.NewServiceName(serviceName, opt...)
	} else if r := application.GetRegistryClient(); r != nil {
		return r.NewServiceName(serviceName, opt...)
	} else {
		return ""
	}
}

// 注册服务名
func RegisterServiceName(serviceName string, opt ...service_options.RegisterOption) (*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r == nil {
		return nil, errors.ErrRegistryNOtImplement
	} else {
		return r.RegisterService(serviceName, opt...)
	}
}

// 反注册服务名
func UnregisterServiceName(serviceName string) (*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r == nil {
		return nil, errors.ErrRegistryNOtImplement
	} else {
		return r.UnregisterService(serviceName)
	}
}

// 查找服务
func WhereIsServiceName(serviceName string, opt ...service_options.WhereOption) ([]*gira.Peer, error) {
	application := gira.GetRuntime()
	if r := application.GetRegistry(); r != nil {
		return r.WhereIsService(serviceName, opt...)
	} else if r := application.GetRegistryClient(); r != nil {
		return r.WhereIsService(serviceName, opt...)
	} else {
		return nil, errors.ErrRegistryNOtImplement
	}
}

func UnregisterPeer(appFullName string) error {
	application := gira.GetRuntime()
	if r := application.GetRegistryClient(); r != nil {
		return r.UnregisterPeer(appFullName)
	} else {
		return errors.ErrRegistryNOtImplement
	}
}

// ================= db client component =============================
// 返回预定义的admindb client
func GetAdminDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetAdminDbClient()
	} else {
		return nil
	}
}

// 返回预定义的resourcedb client
func GetResourceDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetResourceDbClient()
	} else {
		return nil
	}
}

// 返回预定义的statdb client
func GetStatDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetStatDbClient()
	} else {
		return nil
	}
}

// 返回预定义的accountdb client
func GetAccountDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetAccountDbClient()
	} else {
		return nil
	}
}

// 返回预定义的logdb client
func GetLogDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetLogDbClient()
	} else {
		return nil
	}
}

// 返回预定义的behaviordb client
func GetBehaviorDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetBehaviorDbClient()
	} else {
		return nil
	}
}

// 返回预定义的admincache client
func GetAdminCacheClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetAdminCacheClient()
	} else {
		return nil
	}
}

// 返回预定义的accountcache client
func GetAccountCacheClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetAccountCacheClient()
	} else {
		return nil
	}
}

// 返回预定义的gamecache client
func GetGameCacheClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetGameCacheClient()
	} else {
		return nil
	}
}

// 返回预定义的gamedb client
func GetGameDbClient() gira.DbClient {
	application := gira.GetRuntime()
	if c, ok := application.(gira.DbClientComponent); ok {
		return c.GetGameDbClient()
	} else {
		return nil
	}
}

// ================= grpc server component =============================
func GrpcServer() gira.GrpcServer {
	application := gira.GetRuntime()
	return application.GetGrpcServer()
}

func IsEnableResolver() bool {
	application := gira.GetRuntime()
	if cfg := application.GetConfig().Module.Grpc; cfg != nil {
		return cfg.Resolver
	} else {
		return false
	}
}

// 查看grpc server
func WhereIsServer(name string) (svr interface{}, ok bool) {
	application := gira.GetRuntime()
	if s := application.GetGrpcServer(); s == nil {
		return nil, false
	} else {
		return s.GetServer(name)
	}
}

// ================= sdk =============================
// 登录sdk
func SdkLogin(accountPlat string, openId string, token string, authUrl string, appId string, appSecret string) (*gira.SdkAccount, error) {
	application := gira.GetRuntime()
	if s := application.GetPlatformSdk(); s == nil {
		return nil, errors.ErrSdkComponentNotImplement
	} else {
		return s.Login(accountPlat, openId, token, authUrl, appId, appSecret)
	}
}

// 验证sdk订单
func SdkPayOrderCheck(accountPlat string, args map[string]interface{}, paySecret string) (*gira.SdkPayOrder, error) {
	application := gira.GetRuntime()
	if s := application.GetPlatformSdk(); s == nil {
		return nil, errors.ErrSdkComponentNotImplement
	} else {
		return s.PayOrderCheck(accountPlat, args, paySecret)
	}
}

// ================= service =============================
// 停止服务
func StopService(service gira.Service) error {
	application := gira.GetRuntime()
	if s := application.GetServiceContainer(); s == nil {
		return errors.ErrServiceContainerNotImplement
	} else {
		return s.StopService(service)
	}
}

// 启动服务
func StartService(name string, service gira.Service) error {
	application := gira.GetRuntime()
	if s := application.GetServiceContainer(); s == nil {
		return errors.ErrServiceContainerNotImplement
	} else {
		return s.StartService(name, service)
	}
}

// ================= cron =============================
// 设置定时调度函数
// 并发不安全, github.com/robfig/cron 中的AddFunc函数会对同一个切片进行修改
func Cron(spec string, cmd func()) error {
	application := gira.GetRuntime()
	if s := application.GetCron(); s == nil {
		return errors.ErrCronNotImplement
	} else {
		return s.AddFunc(spec, cmd)
	}
}
