package proj

import (
	"fmt"
	"path"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/config"
)

var (
	Config *gira.Config
)

// 读取应该配置
func LoadCliConfig() (*gira.Config, error) {
	configFilePath := path.Join(Dir.ConfigDir, "cli.yaml")
	dotEnvFilePath := path.Join(Dir.EnvDir, ".env")
	if c, err := config.Load(Dir.ProjectDir, configFilePath, dotEnvFilePath, "cli", 0); err != nil {
		return nil, err
	} else {
		Config = c
		return c, nil
	}
}

// 读取测试配置
func LoadTestConfig() (*gira.Config, error) {
	configFilePath := path.Join(Dir.ConfigDir, "test.yaml")
	dotEnvFilePath := path.Join(Dir.EnvDir, ".env")
	if c, err := config.Load(Dir.ProjectDir, configFilePath, dotEnvFilePath, "test", 0); err != nil {
		return nil, err
	} else {
		Config = c
		return c, nil
	}
}

// 读取应用配置
func LoadApplicationConfig(appType string, appId int32) (*gira.Config, error) {
	configFilePath := path.Join(Dir.ConfigDir, fmt.Sprintf("%s.yaml", appType))
	dotEnvFilePath := path.Join(Dir.EnvDir, ".env")
	if c, err := config.Load(Dir.ProjectDir, configFilePath, dotEnvFilePath, appType, appId); err != nil {
		return nil, err
	} else {
		Config = c
		return c, nil
	}
}
