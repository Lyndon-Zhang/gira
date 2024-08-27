package config

import (
	"github.com/Lyndon-Zhang/gira"
	"gopkg.in/yaml.v3"
)

var Game *Config

type GameConfig struct {
	GatewayReportInterval int64 `yaml:"gateway-report-interval"`
	// 玩家数据保存间隔
	BgSaveInterval       int64 `yaml:"bgsave-interval"`
	RequestBufferSize    int   `yaml:"request-buffer-size"`
	ResponseBufferSize   int   `yaml:"response-buffer-size"`
	PushBufferSize       int   `yaml:"push-buffer-size"`
	SessionActorBuffSize int   `yaml:"session-actor-buffer-size"`
	TraceProtoDebugMsg   bool  `yaml:"trace-proto-debug-msg"`
}

type Config struct {
	gira.Config
	Framework struct {
		Game GameConfig `yaml:"game"`
	} `yaml:"framework"`
}

func (c *Config) OnConfigLoad(config *gira.Config) error {
	c.Config = *config
	if err := yaml.Unmarshal(config.Raw, c); err != nil {
		return err
	}
	Game = c
	return nil
}
