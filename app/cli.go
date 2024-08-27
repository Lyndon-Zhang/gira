package app

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/log"
	"github.com/Lyndon-Zhang/gira/proj"
	"github.com/Lyndon-Zhang/gira/service/admin/adminpb"
	peer_service "github.com/Lyndon-Zhang/gira/service/peer"
	"github.com/Lyndon-Zhang/gira/service/peer/peerpb"
	"gopkg.in/yaml.v3"

	"github.com/Lyndon-Zhang/gira"
	"github.com/urfave/cli/v2"
)

type ClientApplication struct {
}

func (s *ClientApplication) OnConfigLoad(c *gira.Config) error {
	return nil
}
func (s *ClientApplication) OnCreate() error {
	return nil
}
func (s *ClientApplication) OnStart() error {
	return nil
}
func (s *ClientApplication) OnStop() error {
	return nil
}

// 需要两个系参数 xx -id 1 start|stop|restart
func Cli(name string, appVersion string, buildTime string, application gira.Application) error {
	app := &cli.App{
		Name: "gira service",
		// Authors:     []*cli.Author{&cli.Author{Name: "lujingwei", Email: "lujingwei002@qq.com"}},
		Description: "gira service",
		Flags:       []cli.Flag{},
		Action:      runAction,
		Metadata: map[string]interface{}{
			"application": application,
			"name":        name,
			"buildTime":   buildTime,
			"appVersion":  appVersion,
		},
		Commands: []*cli.Command{
			{
				Name: "start",
				//Aliases: []string{"start"},
				Usage:  "Start service",
				Action: startAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "service id",
						Required: true,
					},
				},
			},
			{
				Name:   "status",
				Usage:  "Display status",
				Action: statusAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "service id",
						Required: true,
					},
				},
			},
			{
				Name:   "unregister",
				Usage:  "unregister service",
				Action: unregisterAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "service id",
						Required: true,
					},
				},
			},
			{
				Name:   "stop",
				Usage:  "Stop service",
				Action: stopAction,
			},
			{
				Name:   "restart",
				Usage:  "Restart service",
				Action: restartAction,
			},
			{
				Name:   "reload",
				Usage:  "Reload config",
				Action: reloadAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "service id",
						Required: true,
					},
				},
			},
			{
				Name:   "version",
				Usage:  "Build version",
				Action: versionAction,
			},
			{
				Name:   "time",
				Usage:  "Build time",
				Action: timeAction,
			},
			{
				Name:   "env",
				Usage:  "Print env",
				Action: envAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "service id",
						Required: true,
					},
				},
			},
			{
				Name:   "config",
				Usage:  "Print config",
				Action: configAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "service id",
						Required: true,
					},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		return err
	}
	return nil
}

func StartAsClient(application gira.Application, appId int32, appType string) error {
	r := newRuntime(gira.ApplicationArgs{
		AppType:     appType,
		AppId:       appId,
		Application: application,
	})
	return r.start()
}

func StartAsServer(application gira.Application, appId int32, appType string) error {
	r := newRuntime(gira.ApplicationArgs{
		AppType:     appType,
		AppId:       appId,
		Application: application,
	})
	return r.start()
}

// 启动应用
func startAction(args *cli.Context) error {
	appId := int32(args.Int("id"))
	application, _ := args.App.Metadata["application"].(gira.Application)
	appType, _ := args.App.Metadata["name"].(string)
	appVersion, _ := args.App.Metadata["appVersion"].(string)
	var buildTime int64
	if v, ok := args.App.Metadata["buildTime"].(string); ok {
		if t, err := strconv.Atoi(v); err != nil {
			buildTime = 0
		} else {
			buildTime = int64(t)
		}
	}
	log.Println("build version:", appVersion)
	log.Println("build time:", buildTime)
	log.Infof("%s %d starting...", appType, appId)
	runtime := newRuntime(gira.ApplicationArgs{
		AppType:     appType,
		AppId:       appId,
		AppVersion:  appVersion,
		BuildTime:   buildTime,
		Application: application,
	})

	if err := runtime.start(); err != nil {
		return err
	}
	if err := runtime.Wait(); err != nil {
		return err
	}
	return nil
}

// 打印应用构建版本
func versionAction(args *cli.Context) error {
	appVersion, _ := args.App.Metadata["appVersion"].(string)
	fmt.Println(appVersion)
	return nil
}

// 打印应用构建时间
func timeAction(args *cli.Context) error {
	buildTime, _ := args.App.Metadata["buildTime"].(string)
	fmt.Println(buildTime)
	return nil
}

// 打印应用环境变量
func envAction(args *cli.Context) error {
	appId := int32(args.Int("id"))
	appType, _ := args.App.Metadata["name"].(string)
	if _, err := proj.LoadApplicationConfig(appType, appId); err != nil {
		return err
	} else {
		for _, k := range os.Environ() {
			fmt.Println(k, os.Getenv(k))
		}
	}
	return nil
}

// 打印应用配置
func configAction(args *cli.Context) error {
	appId := int32(args.Int("id"))
	appType, _ := args.App.Metadata["name"].(string)
	if c, err := proj.LoadApplicationConfig(appType, appId); err != nil {
		return err
	} else {
		c.Raw = nil
		if s, err := yaml.Marshal(c); err != nil {
			return err
		} else {
			fmt.Println(string(s))
		}
	}
	return nil
}

func stopAction(args *cli.Context) error {
	return nil
}

func restartAction(args *cli.Context) error {
	return nil
}

func statusAction(args *cli.Context) error {
	appId := int32(args.Int("id"))

	if err := StartAsClient(&ClientApplication{}, appId, "cli"); err != nil {
		return err
	}
	ctx := facade.Context()
	serviceName := peer_service.GetServiceName()
	if _, err := peerpb.DefaultPeerClients.Unicast().Where(serviceName).HealthCheck(ctx, &peerpb.HealthCheckRequest{}); err != nil {
		log.Println(err)
		log.Println("dead")
		return nil
	} else {
		log.Println("alive")
		return nil
	}
}

func unregisterAction(args *cli.Context) error {
	appId := int32(args.Int("id"))
	appType, _ := args.App.Metadata["name"].(string)
	if err := StartAsClient(&ClientApplication{}, appId, "cli"); err != nil {
		return err
	}
	appFullName := gira.FormatAppFullName(appType, appId, facade.GetZone(), facade.GetEnv())
	if err := facade.UnregisterPeer(appFullName); err != nil {
		log.Println(err)
		return nil
	} else {
		log.Println("OK")
		return nil
	}
}

func reloadAction(args *cli.Context) error {
	appId := int32(args.Int("id"))
	if err := StartAsClient(&ClientApplication{}, appId, "cli"); err != nil {
		return err
	}
	ctx := facade.Context()
	serviceName := peer_service.GetServiceName()
	if _, err := adminpb.DefaultAdminClients.Unicast().Where(serviceName).ReloadResource(ctx, &adminpb.ReloadResourceRequest{}); err != nil {
		log.Println(err)
		return nil
	} else {
		log.Println("OK")
		return nil
	}
}

func runAction(args *cli.Context) error {
	return nil
}
