package main

import (
	"bufio"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/log"
	"github.com/urfave/cli/v2"

	"github.com/Lyndon-Zhang/gira/gen/gen_application"
	"github.com/Lyndon-Zhang/gira/gen/gen_behavior"
	"github.com/Lyndon-Zhang/gira/gen/gen_const"
	"github.com/Lyndon-Zhang/gira/gen/gen_macro"
	"github.com/Lyndon-Zhang/gira/gen/gen_model"
	"github.com/Lyndon-Zhang/gira/gen/gen_protocol"
	"github.com/Lyndon-Zhang/gira/gen/gen_resource"
	"github.com/Lyndon-Zhang/gira/proj"
)

func main() {
	app := &cli.App{
		Name: "gira-cli",
		Authors: []*cli.Author{{
			Name:  "lujingwei",
			Email: "lujingwei@xx.org",
		}},
		Description: "gira-cli",
		Flags:       []cli.Flag{},
		//log.SetFlags(log.LstdFlags | log.Lshortfile)
		Before: beforeAction,
		Commands: []*cli.Command{
			{
				Name:   "gen",
				Usage:  "gen [resource|model|protocol|const]",
				Before: beforeAction1,
				Subcommands: []*cli.Command{
					{
						Name:   "all",
						Usage:  "gen all",
						Action: genAllAction,
					},
					{
						Name:   "resource",
						Usage:  "gen resource",
						Action: genResourceAction,
					},
					{
						Name:   "model",
						Usage:  "gen model",
						Action: genModelAction,
					},
					{
						Name:   "protocol",
						Usage:  "gen protocol",
						Action: genProtocolAction,
					},
					{
						Name:   "const",
						Usage:  "gen const",
						Action: genConstAction,
					},
					{
						Name:   "application",
						Usage:  "gen application",
						Action: genApplicationAction,
					},
					{
						Name:   "behavior",
						Usage:  "gen behavior",
						Action: genBehaviorAction,
					},
				},
			},
			{
				Name:   "resource",
				Usage:  "resource [compress|push]",
				Before: beforeAction1,
				Subcommands: []*cli.Command{
					{
						Name:   "compress",
						Usage:  "compress yaml to binary",
						Action: resourceCompressAction,
					},
					{
						Name:   "push",
						Usage:  "push table to database",
						Action: resourcePushAction,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:     "drop",
								Value:    false,
								Usage:    "drop collection",
								Required: false,
							},
							&cli.BoolFlag{
								Name:     "force",
								Value:    false,
								Usage:    "force overwrite collection",
								Required: false,
							},
						},
					},
					{
						Name:   "reload",
						Usage:  "reload",
						Action: resourceReloadAction,
					},
				},
			},
			{
				Name:   "migrate",
				Usage:  "migrate dbname",
				Before: beforeAction1,
				Action: migrateAction,
				Flags:  []cli.Flag{},
			},
			{
				Name:   "env",
				Usage:  "env",
				Before: beforeAction1,
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "list env name",
						Action: envListAction,
						Flags:  []cli.Flag{},
					},
					{
						Name:   "switch",
						Usage:  "switch env name",
						Action: envSwitchAction,
					},
				},
			},
			{
				Name:   "macro",
				Usage:  "macro code",
				Action: macroAction,
				Before: beforeAction1,
			},
			{
				Name: "start",
				//Aliases: []string{"start"},
				Usage:  "Start minigame",
				Action: startAction,
			},
			{
				Name:   "status",
				Usage:  "Display status",
				Action: statusAction,
			},
			{
				Name:   "stop",
				Usage:  "Stop minigame",
				Action: stopAction,
			},
			{
				Name:   "restart",
				Usage:  "Restart minigame",
				Action: restartAction,
			},
			{
				Name:   "reload",
				Usage:  "Reload config",
				Action: reloadAction,
			},
			{
				Name:   "run",
				Usage:  "run command",
				Action: runAction,
			},
			{
				Name:    "build",
				Usage:   "build command",
				Aliases: []string{"b"},
				Action:  buildAction,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Info(err)
	}
}

func beforeAction1(args *cli.Context) error {
	return nil
}

func command(name string, argv []string) error {
	cmd := exec.Command(name, argv...)
	// 获取命令的标准输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	// 启动命令
	if err := cmd.Start(); err != nil {
		return err
	}
	// 创建一个channel，用于接收信号
	c := make(chan os.Signal, 1)
	// 监听SIGINT信号
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	defer func() {
		signal.Reset(os.Interrupt, syscall.SIGINT)
	}()
	// 创建一个 Scanner 对象，对命令的标准输出和标准错误输出进行扫描
	scanner1 := bufio.NewScanner(stdout)
	go func() {
		for scanner1.Scan() {
			// 输出命令的标准输出
			log.Println(scanner1.Text())
		}
	}()
	scanner2 := bufio.NewScanner(stderr)
	go func() {
		for scanner2.Scan() {
			// 输出命令的标准错误输出
			fmt.Fprintln(os.Stderr, scanner2.Text())
		}
	}()
	go func() {
		// 等待信号
		<-c
	}()
	// 等待命令执行完成
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func migrateAction(c *cli.Context) error {
	if c.Args().Len() < 2 {
		cli.ShowAppHelp(c)
		return nil
	}
	name := c.Args().Get(0)
	db := c.Args().Get(1)
	bin := fmt.Sprintf("bin/migrate-%s", name)
	if _, err := os.Stat(bin); err != nil {
		return err
	}
	if config, err := proj.LoadCliConfig(); err != nil {
		return err
	} else {
		if dbConfig, ok := config.Db[db]; !ok {
			log.Errorw("db config not found", "db", db)
			return nil
		} else {
			uri := dbConfig.Uri()
			argv := []string{"migrate", "--uri", uri}
			argv = append(argv, c.Args().Tail()...)
			return command(bin, argv)
		}
	}
}

func resourceCompressAction(args *cli.Context) error {
	bin := "bin/resource"
	argv := []string{"compress"}
	return command(bin, argv)
}

func resourcePushAction(c *cli.Context) error {
	if config, err := proj.LoadCliConfig(); err != nil {
		return err
	} else if dbConfig, ok := config.Db[gira.RESOURCEDB_NAME]; !ok {
		fmt.Printf("%s config not found\n", gira.RESOURCEDB_NAME)
		return nil
	} else {
		uri := dbConfig.Uri()
		bin := "bin/resource"
		argv := []string{"push", "--uri", uri}
		if c.Bool("force") {
			argv = append(argv, "--force")
		}
		if c.Bool("drop") {
			argv = append(argv, "--drop")
		}
		return command(bin, argv)
	}
}

func resourceReloadAction(args *cli.Context) error {
	return nil
}

// 切换环境
func envSwitchAction(args *cli.Context) error {
	if args.NArg() < 1 {
		cli.ShowAppHelp(args)
		return nil
	}
	expectName := args.Args().Get(0)
	envDir := proj.Dir.EnvDir
	found := false
	if d, err := os.Open(envDir); err != nil {
		return err
	} else {
		defer d.Close()
		if files, err := d.ReadDir(0); err != nil {
			return err
		} else {
			for _, file := range files {
				if file.IsDir() {
					envName := file.Name()
					if envName == expectName {
						found = true
					}
				}
			}
		}
	}
	if !found {
		log.Println("env not found, you can select one of below env")
		return envListAction(args)
	}
	os.Remove(filepath.Join(proj.Dir.EnvDir, ".env"))
	os.Remove(filepath.Join(proj.Dir.EnvDir, "config.yaml"))
	if err := os.Symlink(filepath.Join(proj.Dir.EnvDir, expectName, ".env"), filepath.Join(proj.Dir.EnvDir, ".env")); err != nil {
		return err
	}
	if err := os.Symlink(filepath.Join(proj.Dir.EnvDir, expectName, "config.yaml"), filepath.Join(proj.Dir.EnvDir, "config.yaml")); err != nil {
		return err
	}
	log.Printf("switch %s success", expectName)
	return envListAction(args)
}

// 环境列表
func envListAction(c *cli.Context) error {
	if config, err := proj.LoadCliConfig(); err != nil {
		return err
	} else {
		envDir := proj.Dir.EnvDir
		if d, err := os.Open(envDir); err != nil {
			return err
		} else {
			defer d.Close()
			if files, err := d.ReadDir(0); err != nil {
				return err
			} else {
				for _, file := range files {
					if file.IsDir() {
						envName := file.Name()
						if envName == config.Env {
							log.Info("*", file.Name())
						} else {
							log.Info(file.Name())
						}
					}
				}
			}
		}
	}
	return nil
}

func beforeAction(args *cli.Context) error {
	if err := log.ConfigAsCli(proj.Dir.LogDir); err != nil {
		return err
	}
	log.Println()
	log.Println("*************", strings.Join(os.Args, " "), "***************")
	return nil
}

func genResourceAction(args *cli.Context) error {
	if err := gen_resource.Gen(gen_resource.Config{
		Force: true,
	}); err != nil {
		return err
	}
	return nil
}

func genProtocolAction(args *cli.Context) error {
	if err := gen_protocol.Gen(); err != nil {
		return err
	}
	return nil
}

func genModelAction(args *cli.Context) error {
	if err := gen_model.Gen(); err != nil {
		return err
	}
	return nil
}

func genApplicationAction(args *cli.Context) error {
	if err := gen_application.Gen(); err != nil {
		return err
	}
	return nil
}

func genConstAction(args *cli.Context) error {
	if err := gen_const.Gen(); err != nil {
		return err
	}
	return nil
}

func genBehaviorAction(args *cli.Context) error {
	if err := gen_behavior.Gen(); err != nil {
		return err
	}
	return nil
}

func genAllAction(args *cli.Context) error {
	if err := gen_protocol.Gen(); err != nil {
		return err
	}
	if err := gen_application.Gen(); err != nil {
		return err
	}
	if err := gen_resource.Gen(gen_resource.Config{
		Force: true,
	}); err != nil {
		return err
	}
	if err := gen_const.Gen(); err != nil {
		return err
	}
	if err := gen_model.Gen(); err != nil {
		return err
	}
	return nil
}

func resourceAction(args *cli.Context) error {
	log.Println("resourceAction")
	return nil
}

func macroAction(args *cli.Context) error {
	log.Println("macroAction")
	if err := gen_macro.Gen(&gen_macro.Config{}); err != nil {
		return err
	}
	return nil
}

func startAction(args *cli.Context) error {
	return nil
}

func stopAction(args *cli.Context) error {
	return nil
}

func restartAction(args *cli.Context) error {
	return nil
}

func statusAction(args *cli.Context) error {
	/*projectDir, err := getProjectDir()
	if err != nil {
		return err
	}
	configPath := fmt.Sprintf("%s/supervisord/supervisord.conf", projectDir)
	execCommand("supervisorctl", []string{"-c", configPath, "status"})*/
	return nil
}

func reloadAction(args *cli.Context) error {
	return nil
}

func runAction(c *cli.Context) error {
	if c.Args().Len() <= 0 {
		return nil
	}
	name := c.Args().First()
	args := c.Args().Tail()
	return proj.Run(name, args)
}

func buildAction(c *cli.Context) error {
	if c.Args().Len() <= 0 {
		name := "default"
		return proj.Build(name)
	} else {
		name := c.Args().First()
		return proj.Build(name)
	}
}
