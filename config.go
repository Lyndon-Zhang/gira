package gira

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Lyndon-Zhang/gira/errors"
)

// 日志配置
type LogConfig struct {
	Console         bool   `yaml:"console"`
	Db              bool   `yaml:"db"`
	Level           string `yaml:"level"` // ERROR|WARN|INFO|DEBUG
	DbLevel         string `yaml:"db-level"`
	TraceErrorStack bool   `yaml:"trace-error-stack"`
	// 输出到文件
	Files []struct {
		Level      string `yaml:"level"`  // 大于level的都会被输出
		Filter     string `yaml:"filter"` // 等于filter的才会被输出, filter比level优先级更高
		MaxSize    int    `yaml:"max-size"`
		MaxBackups int    `yaml:"max-backups"`
		MaxAge     int    `yaml:"max-age"`
		Compress   bool   `yaml:"compress"`
		Path       string `yaml:"path"`
		Format     string `yaml:"format"` // json|console
	} `yaml:"files"`
}

// jwt配置
type JwtConfig struct {
	Secret            string `yaml:"secret"`
	Expiretime        int64  `yaml:"expiretime"`
	RefreshExpiretime int64  `yaml:"refresh-expiretime"`
}

type DbConfig struct {
	Driver          string        `yaml:"driver"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Db              string        `yaml:"db"`
	Query           string        `yaml:"query"`
	MaxOpenConns    int           `yaml:"max-open-conns"`
	MaxIdleConns    int           `yaml:"max-idle-conns"`
	ConnMaxIdleTime time.Duration `yaml:"conn-max-idle-time"`
	ConnMaxLifetime time.Duration `yaml:"conn-max-lifetime"`
	ConnnectTimeout time.Duration `yaml:"connect-timeout"`
}

type BehaviorConfig struct {
	SyncInterval int64 `yaml:"sync-interval"`
	BatchInsert  int   `yaml:"batch-insert"`
}

// 完整的地址，包括path部分
func (self DbConfig) Uri() string {
	switch self.Driver {
	case MONGODB_NAME:
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/?db=%s&%s", self.User, self.Password, self.Host, self.Port, self.Db, self.Query)
	case REDIS_NAME:
		return fmt.Sprintf("redis://%s:%s@%s:%d?%s", self.User, self.Password, self.Host, self.Port, self.Query)
	case MYSQL_NAME:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&%s", self.User, self.Password, self.Host, self.Port, self.Db, self.Query)
	default:
		return fmt.Sprintf("%s not support", self.Driver)
	}
	//return fmt.Sprintf("%s://%s:%s@%s:%d/%s", self.Driver, self.User, self.Password, self.Host, self.Port, self.Db)
}

func (self DbConfig) GormUri() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", self.User, self.Password, self.Host, self.Port, self.Db)
}

func (self *DbConfig) Parse(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case MONGODB_NAME:
		self.Driver = u.Scheme
	case REDIS_NAME:
		self.Driver = u.Scheme
	case MYSQL_NAME:
		self.Driver = u.Scheme
	default:
		return errors.ErrDbNotSupport
	}
	host2 := strings.Split(u.Host, ":")
	if len(host2) == 2 {
		self.Host = host2[0]
		if v, err := strconv.Atoi(host2[1]); err != nil {
			return err
		} else {
			self.Port = v
		}
	} else if len(host2) == 1 {
		switch u.Scheme {
		case MONGODB_NAME:
			self.Port = 27017
		case REDIS_NAME:
			self.Port = 6379
		case MYSQL_NAME:
			self.Port = 3306
		}
	}
	self.User = u.User.Username()
	if v, set := u.User.Password(); set {
		self.Password = v
	} else {
		self.Password = ""
	}
	switch u.Scheme {
	case MONGODB_NAME:
		self.Db = u.Query().Get("db")
		query := u.Query()
		query.Del("db")
		self.Query = query.Encode()
	default:
		path := strings.TrimPrefix(u.Path, "/")
		self.Db = path
		self.Query = u.Query().Encode()
	}
	return nil
}

// registry配置
type EtcdConfig struct {
	Endpoints []struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"endpoints"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DialTimeout  int    `yaml:"dial-timeout"`
	LeaseTimeout int64  `yaml:"lease-timeout"`
	Address      string `yaml:"address"`
	Advertise    []struct {
		Name  string `yaml:"name"`
		Value string `yaml:"value"`
	} `yaml:"advertise"`
}

// registry配置
type EtcdClientConfig struct {
	Endpoints []struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"endpoints"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DialTimeout  int    `yaml:"dial-timeout"`
	LeaseTimeout int64  `yaml:"lease-timeout"`
	Address      string `yaml:"address"`
}

// http模块配置
type HttpConfig struct {
	Addr         string `yaml:"addr"`
	ReadTimeout  int64  `yaml:"read-timeout"`
	WriteTimeout int64  `yaml:"write-timeout"`
	Ssl          bool   `yaml:"ssl"`
	CertFile     string `yaml:"cert-file"`
	KeyFile      string `yaml:"key-file"`
}

// 网关模块配置
type GatewayConfig struct {
	IsWebsocket      bool          `yaml:"is-websocket"`
	Bind             string        `yaml:"bind"`
	Address          string        `yaml:"address"`
	Debug            bool          `yaml:"debug"`
	Ssl              bool          `yaml:"ssl"`
	CertFile         string        `yaml:"cert-file"`
	KeyFile          string        `yaml:"key-file"`
	WsPath           string        `yaml:"ws-path"`
	RecvBuffSize     int           `yaml:"recv-buff-size"`
	RecvBacklog      int           `yaml:"recv-backlog"`
	SendBacklog      int           `yaml:"send-backlog"`
	HandshakeTimeout time.Duration `yaml:"handshake-timeout"`
	Heartbeat        time.Duration `yaml:"heartbeat"`
}

type TestPlatformConfig struct {
	Secret string `yaml:"secret"`
}

type PwdPlatformConfig struct {
	Secret string `yaml:"secret"`
}

type UltraPlatformConfig struct {
	Secret string `yaml:"secret"`
}

type DouyinPlatformConfig struct {
	Secret string `yaml:"secret"`
}

type WeixinPlatformConfig struct {
	Secret string `yaml:"secret"`
}

type KuaishouPlatformConfig struct {
	Secret string `yaml:"secret"`
}
type A4399PlatformConfig struct {
	Secret string `yaml:"secret"`
}
type PlatformConfig struct {
	Test     *TestPlatformConfig     `yaml:"test"`
	Pwd      *PwdPlatformConfig      `yaml:"pwd"`
	Ultra    *UltraPlatformConfig    `yaml:"ultra"`
	Douyin   *DouyinPlatformConfig   `yaml:"douyin"`
	Weixin   *WeixinPlatformConfig   `yaml:"weixin"`
	Kuaishou *KuaishouPlatformConfig `yaml:"kuaishou"`
	A4399	 *A4399PlatformConfig 	 `yaml:"a4399"`
}

type GrpcConfig struct {
	Address      string `yaml:"address"`
	Workers      uint32 `yaml:"workers"`
	Resolver     bool   `yaml:"resolver"` // 是否开启resolver
	Admin        bool   `yaml:"admin"`
	EnabledTrace bool   `yaml:"enabled-trace"`
}

type PprofConfig struct {
	Port         int    `yaml:"port"`
	Bind         string `yaml:"bind"`
	EnabledTrace bool   `yaml:"enabled-trace"`
}

type ResourceConfig struct {
	Compress bool `yaml:"compress"`
}

type AdminConfig struct {
	None string `yaml:"none"`
}

type Config struct {
	Raw         []byte
	Thread      int         `yaml:"thread"`
	Env         string      `yaml:"env"`
	Zone        string      `yaml:"zone"`
	Log         *LogConfig  `yaml:"log"`
	CoreLog     *LogConfig  `yaml:"core-log"`
	BehaviorLog *LogConfig  `yaml:"behavior-log"`
	Pprof       PprofConfig `yaml:"pprof"`
	Sandbox     int         `yaml:"sandbox"`
	Db          map[string]*DbConfig
	Resource    ResourceConfig `yaml:"resource"`
	Module      struct {
		Behavior   *BehaviorConfig   `yaml:"behavior"`
		Http       *HttpConfig       `yaml:"http,omitempty"`
		Etcd       *EtcdConfig       `yaml:"etcd"`
		EtcdClient *EtcdClientConfig `yaml:"etcd-client"`
		Grpc       *GrpcConfig       `yaml:"grpc"`
		Plat       *PlatformConfig   `yaml:"plat"`
		Jwt        *JwtConfig        `yaml:"jwt"`
		Gateway    *GatewayConfig    `yaml:"gateway"`
		Admin      *AdminConfig      `yaml:"admin"`
	} `yaml:"module"`
}
