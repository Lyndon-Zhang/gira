package config

import (
	"github.com/Lyndon-Zhang/gira"
	"gopkg.in/yaml.v3"
)

var Gateway *Config

type GatewayConfig struct {
	// 登录超时
	WaitLoginTimeout int64 `yaml:"wait-login-timeout"`
	// 最大会话数量
	MaxSessionCount int64 `yaml:"max-session-count"`
	Upstream        struct {
		HeartbeatInvertal int64  `yaml:"heartbeat-interval"`
		MinId             int32  `yaml:"min-id"`
		MaxId             int32  `yaml:"max-id"`
		Name              string `yaml:"name"`
	} `yaml:"upstream"`
}

type Config struct {
	gira.Config
	Framework struct {
		Gateway GatewayConfig `yaml:"gateway"`
	} `yaml:"framework"`
}

func (c *Config) OnConfigLoad(config *gira.Config) error {
	c.Config = *config
	if err := yaml.Unmarshal(config.Raw, c); err != nil {
		return err
	}
	Gateway = c
	return nil
}
