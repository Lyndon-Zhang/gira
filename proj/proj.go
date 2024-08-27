package proj

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	yaml "gopkg.in/yaml.v3"
)

/*
* 处理项目的目录结构
*
 */
const (
	project_config_file_name = "gira.yaml"
)

var (
	Version       string
	Module        string
	Env           string //当前环境 local|dev|qa|prd
	Zone          string // 当前区 wc|qq|gf|review
	Dir           *DirConfigs
	ProjectConfig *ProjectConfigs
	BuildConfig   *BuildConfigs
	TaskConfig    *TaskConfigs
)

type CommandConfig struct {
	WorkDir string   `yaml:"workdir"`
	Name    string   `yaml:"name"`
	Args    []string `yaml:"args"`
}

type BuildConfigs struct {
	Targets map[string]BuildTargetConfig
}

type BuildTargetConfig struct {
	Run         []string        `yaml:"run"`
	Dependency  []string        `yaml:"dependency"`
	Description string          `yaml:"description"`
	Command     []CommandConfig `yaml:"command"`
}

type TaskConfigs struct {
	Targets map[string][]string
}

type ProjectConfigs struct {
	Version         string `yaml:"version"`
	Module          string `yaml:"module"`
	GenResourceHash string `yaml:"gen_resource_hash"`
	Applications    map[string]struct {
	} `yaml:"application"`
}
type DirConfigs struct {
	ProjectDir           string
	DotEnvFilePath       string // .env
	ProjectConfFilePath  string // gira.yaml
	DocDir               string // doc
	LogDir               string // log
	ConfigDir            string // config
	EnvDir               string // env
	ResourceDir          string // resource
	GenDir               string // gen
	RunDir               string // run
	SrcTestDir           string // src/test
	GenModelDir          string // gen/model
	GenProtocolDir       string // gen/protocol
	SrcDir               string // src
	SrcDocDir            string // src/doc
	SrcGenDir            string // src/gen/
	SrcGenApplicationDir string // src/gen/application
	SrcGenModelDir       string // src/gen/model/
	SrcGenProtocolDir    string // src/gen/protocol/
	SrcGenConstDir       string // src/gen/const/
	SrcGenResourceDir    string // src/gen/resource/
	SrcGenBehaviorDir    string // src/gen/behavior/
	GenBehaviorDir       string // gen/behavior/
	GenResourceDir       string // gen/resource/
	ExcelDir             string // doc/resource/
	ConstDocFilePath     string // doc/const.yaml
	DocResourceFilePath  string // doc/resource.yaml
	DocProtocolFilePath  string // doc/protocol.yaml
	DocProtocolDir       string // doc/protocol/
	DocModelDir          string // doc/model/
	DocBehaviorDir       string // doc/behavior/
}

func init() {
	Dir = &DirConfigs{}
	if err := Dir.init(); err != nil {
		panic(err)
	}
	ProjectConfig = &ProjectConfigs{}
	if err := ProjectConfig.load(); err != nil {
		panic(err)
	}
	BuildConfig = &BuildConfigs{}
	if err := BuildConfig.load(); err != nil {
		panic(err)
	}
	TaskConfig = &TaskConfigs{}
	if err := TaskConfig.load(); err != nil {
		panic(err)
	}
}

func Update(key string, value interface{}) error {
	if data, err := ioutil.ReadFile(Dir.ProjectConfFilePath); err != nil {
		return err
	} else {
		result := make(map[string]interface{})
		if err := yaml.Unmarshal(data, result); err != nil {
			return err
		}
		result[key] = value
		if data, err := yaml.Marshal(result); err != nil {
			return err
		} else {
			if err := ioutil.WriteFile(Dir.ProjectConfFilePath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// 加载gira.yaml并初始化项目目录
func (c *DirConfigs) init() error {
	// 初始化
	// 向上查找gira.yaml文件
	if workDir, err := os.Getwd(); err != nil {
		return err
	} else {
		dir := workDir
		for {
			projectFilePath := filepath.Join(dir, "gira.yaml")
			if _, err := os.Stat(projectFilePath); err == nil {
				c.ProjectDir = dir
				break
			}
			dir = filepath.Dir(dir)
			if dir == "/" || dir == "" {
				break
			}
		}
	}
	c.ProjectConfFilePath = path.Join(c.ProjectDir, project_config_file_name)
	c.EnvDir = path.Join(c.ProjectDir, "env")
	c.ConfigDir = path.Join(c.ProjectDir, "config")
	c.RunDir = path.Join(c.ProjectDir, "run")
	c.LogDir = path.Join(c.ProjectDir, "log")
	c.DotEnvFilePath = path.Join(c.ConfigDir, ".env")
	c.DocDir = path.Join(c.ProjectDir, "doc")
	c.ResourceDir = path.Join(c.ProjectDir, "resource")
	c.GenDir = path.Join(c.ProjectDir, "gen")
	c.SrcTestDir = path.Join(c.ProjectDir, "src", "test")
	c.GenModelDir = path.Join(c.ProjectDir, "gen", "model")
	c.SrcDir = path.Join(c.ProjectDir, "src")
	c.SrcGenDir = path.Join(c.ProjectDir, "src", "gen")
	c.SrcDocDir = path.Join(c.ProjectDir, "src", "doc")
	c.SrcGenConstDir = path.Join(c.SrcGenDir, "const")
	c.SrcGenModelDir = path.Join(c.SrcGenDir, "model")
	c.SrcGenApplicationDir = path.Join(c.SrcGenDir, "application")
	c.SrcGenResourceDir = path.Join(c.SrcGenDir, "resource")
	c.SrcGenProtocolDir = path.Join(c.SrcGenDir, "protocol")
	c.SrcGenBehaviorDir = path.Join(c.SrcGenDir, "behavior")
	c.GenBehaviorDir = path.Join(c.GenDir, "behavior")
	c.GenResourceDir = path.Join(c.GenDir, "resource")
	c.ExcelDir = path.Join(c.DocDir, "resource")
	c.ConstDocFilePath = path.Join(c.DocDir, "const.yaml")
	c.DocResourceFilePath = path.Join(c.DocDir, "resource.yaml")
	c.DocProtocolFilePath = path.Join(c.DocDir, "protocol.yaml")
	c.DocProtocolDir = path.Join(c.DocDir, "protocol")
	c.DocModelDir = path.Join(c.DocDir, "model")
	c.DocBehaviorDir = path.Join(c.DocDir, "behavior")
	c.GenProtocolDir = path.Join(c.GenDir, "protocol")
	return nil
}

func (c *ProjectConfigs) load() error {
	if _, err := os.Stat(Dir.ProjectConfFilePath); err != nil && os.IsNotExist(err) {
		return err
	}
	data, err := ioutil.ReadFile(Dir.ProjectConfFilePath)
	if err != nil {
		return err
	}
	//使用yaml.Unmarshal将yaml文件中的信息反序列化给Config结构体
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	Version = c.Version
	Module = c.Module
	return nil
}

// 加载build.yaml
func (self *BuildConfigs) load() error {
	buildConfigFilePath := filepath.Join(Dir.ProjectDir, ".gira", "build.yaml")
	if _, err := os.Stat(buildConfigFilePath); err == nil {
		if data, err := ioutil.ReadFile(buildConfigFilePath); err != nil {
			return err
		} else {
			if err := yaml.Unmarshal(data, &self.Targets); err != nil {
				return err
			}
		}
	}
	return nil
}

// 加载tasks.yaml
func (self *TaskConfigs) load() error {
	taskConfigFilePath := filepath.Join(Dir.ProjectDir, ".gira", "tasks.yaml")
	if _, err := os.Stat(taskConfigFilePath); err == nil {
		if data, err := ioutil.ReadFile(taskConfigFilePath); err != nil {
			return err
		} else {
			if err := yaml.Unmarshal(data, &self.Targets); err != nil {
				return err
			}
		}
	}
	return nil
}

// 执行tasks.yaml中的命令
func Run(name string, args []string) error {
	if arr, ok := TaskConfig.Targets[name]; !ok {
		return nil
	} else {
		for _, line := range arr {
			// 替换命令中的变量
			for k, v := range args {
				line = strings.Replace(line, fmt.Sprintf("$(%d)", k+1), v, 1)
			}
			fmt.Println(line)
			if err := shell(line); err != nil {
				return err
			}
		}
	}
	return nil
}

// 执行build.yaml中的命令
func Build(name string) error {
	var buildFunc func(target string) error
	buildFunc = func(target string) error {
		if build, ok := BuildConfig.Targets[target]; !ok {
			return nil
		} else {
			if len(build.Dependency) > 0 {
				for _, v := range build.Dependency {
					if err := buildFunc(v); err != nil {
						log.Printf("[FAIL] build %s\n", v)
						return err
					} else {
						log.Printf("[OK] build %s\n", v)
					}
				}
			}
			log.Printf(build.Description)
			for _, v := range build.Run {
				if err := shell(v); err != nil {
					return err
				}
			}
			for _, v := range build.Command {
				if err := execCommand(v); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return buildFunc(name)
}

func shell(command string) error {
	log.Println(command)
	cmd := exec.Command("bash", "-c", command)
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

func execCommand(command CommandConfig) error {
	lastWd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		os.Chdir(lastWd)
	}()
	if command.WorkDir != "" {
		if err := os.Chdir(command.WorkDir); err != nil {
			return err
		}
	}
	line := fmt.Sprintf("%s %s", command.Name, strings.Join(command.Args, " "))
	log.Printf("%s", line)
	if err := execCommandArgv(command.Name, command.Args); err != nil {
		return err
	} else {
	}
	return nil
}

// func execCommandLine(line string) error {
// 	lastWd, err := os.Getwd()
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		os.Chdir(lastWd)
// 	}()
// 	arr := strings.Split(line, ";")
// 	for _, v := range arr {
// 		pats := strings.Fields(v)
// 		name := pats[0]
// 		args := pats[1:]
// 		switch name {
// 		case "cd":
// 			if len(args) > 0 {
// 				os.Chdir(args[0])
// 			} else {
// 				os.Chdir("")
// 			}
// 		default:
// 			log.Printf("%s", v)
// 			if err := execCommandArgv(name, args); err != nil {
// 				log.Fatalln(v)
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

func execCommandLineOutput(line string) (output string, err error) {
	pats := strings.Fields(line)
	name := pats[0]
	argv := pats[1:]
	cmd := exec.Command(name, argv...)
	var out strings.Builder
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return
	}
	output = out.String()
	return
}

func execCommandArgv(name string, argv []string) error {
	re := regexp.MustCompile(`\$\((.*?)\)`) // 匹配 $() 括号中的内容
	for index, arg := range argv {
		matchesArr := re.FindAllStringSubmatch(arg, -1)
		for _, matches := range matchesArr {
			if len(matches) > 1 {
				if v, err := execCommandLineOutput(matches[1]); err == nil {
					v = strings.Trim(v, "\r\n")
					arg = strings.Replace(arg, fmt.Sprintf("$(%s)", matches[1]), v, 1)
				}
			}
		}
		if len(matchesArr) > 0 {
			argv[index] = arg
		}
	}
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
