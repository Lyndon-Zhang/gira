package config

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/errors"
	"gopkg.in/yaml.v2"
)

const (
	config_instruction_type_none = iota
	config_instruction_type_include
	config_instruction_type_template
)

// 读取cli工具配置
// func application_field(reader *config_reader, env map[string]interface{}, key string) interface{} {
// 	if v, ok := env["application"]; ok {
// 		if applications, ok := v.(map[string]interface{}); ok {
// 			if v, ok := applications[reader.appName]; ok {
// 				if application, ok := v.(map[string]interface{}); ok {
// 					if value, ok := application[key]; ok {
// 						return value
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return ""
// }

// func other_application_field(reader *config_reader, otherAppType string, otherAppId int32, env map[string]interface{}, key string) interface{} {
// 	otherAppName := fmt.Sprintf("%s_%d", otherAppType, reader.appId+otherAppId)
// 	if v, ok := env["application"]; ok {
// 		if applications, ok := v.(map[string]interface{}); ok {
// 			if v, ok := applications[otherAppName]; ok {
// 				if application, ok := v.(map[string]interface{}); ok {
// 					if value, ok := application[key]; ok {
// 						return value
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return ""
// }

// func host_field(reader *config_reader, env map[string]interface{}, key string) interface{} {
// 	var hostName string
// 	var hostFound bool = false
// 	if v, ok := env["application"]; ok {
// 		if applications, ok := v.(map[string]interface{}); ok {
// 			if v, ok := applications[reader.appName]; ok {
// 				if application, ok := v.(map[string]interface{}); ok {
// 					hostName = application["host"].(string)
// 					hostFound = true
// 				}
// 			}
// 		}
// 	}
// 	if !hostFound {
// 		return ""
// 	}
// 	if v, ok := env["host"]; ok {
// 		if hosts, ok := v.(map[string]interface{}); ok {
// 			if v, ok := hosts[hostName]; ok {
// 				if host, ok := v.(map[string]interface{}); ok {
// 					if value, ok := host[key]; ok {
// 						return value
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return ""
// }

// func other_host_field(reader *config_reader, otherAppType string, otherAppId int32, env map[string]interface{}, key string) interface{} {
// 	appName := fmt.Sprintf("%s_%d", otherAppType, reader.appId+otherAppId)
// 	var hostName string
// 	var hostFound bool = false
// 	if v, ok := env["application"]; ok {
// 		if applications, ok := v.(map[string]interface{}); ok {
// 			if v, ok := applications[appName]; ok {
// 				if application, ok := v.(map[string]interface{}); ok {
// 					hostName = application["host"].(string)
// 					hostFound = true
// 				}
// 			}
// 		}
// 	}
// 	if !hostFound {
// 		return ""
// 	}
// 	if v, ok := env["host"]; ok {
// 		if hosts, ok := v.(map[string]interface{}); ok {
// 			if v, ok := hosts[hostName]; ok {
// 				if host, ok := v.(map[string]interface{}); ok {
// 					if value, ok := host[key]; ok {
// 						return value
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return ""

// }

// func printLines(str string) {
// 	lines := strings.Split(str, "\n")
// 	for num, line := range lines {
// 		fmt.Println(num+1, ":", line)
// 	}
// }

func newDefaultConfig() *gira.Config {
	return &gira.Config{
		Thread:  1,
		Sandbox: 0,
	}
}

// 读取配置
func Load(projectDir string, configFilePath string, dotEnvFilePath string, appType string, appId int32) (*gira.Config, error) {
	c := newDefaultConfig()
	reader := config_reader{
		projectDir: projectDir,
		appType:    appType,
		appId:      appId,
		appName:    fmt.Sprintf("%s_%d", appType, appId),
	}
	if data, err := reader.read(configFilePath, dotEnvFilePath); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(data, c); err != nil {
		return nil, errors.NewSyntaxError(err.Error(), configFilePath, string(data))
	} else {
		c.Raw = data
		return c, nil
	}
}

type config_reader struct {
	appId      int32
	appType    string
	appName    string
	projectDir string
}

type config_file struct {
	builder   *strings.Builder
	templates []string // 模板
}

func (f *config_file) WriteString(s string) (int, error) {
	return f.builder.WriteString(s)
}

func (f *config_file) String() string {
	return f.builder.String()
}

// dotEnvFilePath 环境变量文件路径 .env
// 优先级 命令行 > 文件中appName指定变量 > 文件中变量
func (c *config_reader) readDotEnv(dotEnvFilePath string) error {
	fileEnv := make(map[string]string)
	appPrefixEnv := make(map[string]string)
	appNamePrefix := c.appName + "."
	if _, err := os.Stat(dotEnvFilePath); err == nil {
		if dict, err := godotenv.Read(dotEnvFilePath); err != nil {
			return errors.NewSyntaxError(err.Error(), dotEnvFilePath, "")
		} else {
			for k, v := range dict {
				if strings.HasPrefix(k, appNamePrefix) {
					appPrefixEnv[strings.Replace(k, appNamePrefix, "", 1)] = v
				} else {
					fileEnv[k] = v
				}
			}
		}
	} else {
		return errors.NewSyntaxError(err.Error(), dotEnvFilePath, "")
	}
	// 读文件中的环境变量到os.env中
	// if _, err := os.Stat(dotEnvFilePath); err == nil {
	// 	if err := godotenv.Load(dotEnvFilePath); err != nil && err != os.ErrNotExist {
	// 		return err
	// 	}
	// }
	for k, v := range appPrefixEnv {
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
		}
	}
	for k, v := range fileEnv {
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
		}
	}
	return nil
}

func (c *config_reader) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"app_id": func() interface{} {
			return c.appId
		},
		"other_ap_id": func(id int32) interface{} {
			return c.appId + id
		},
		"app_type": func() interface{} {
			return c.appType
		},
		"app_name": func() interface{} {
			return c.appName
		},
		"work_dir": func() interface{} {
			return c.projectDir
		},
		"e": func(key string) interface{} {
			return os.Getenv(key)
		},
	}
}

// 加载配置
// 1.加载环境变量
// 2.加载模板变量
// 3.处理模板
// 4.yaml decode
func (c *config_reader) read(configFilePath string, dotEnvFilePath string) ([]byte, error) {
	var err error
	// 加载.env环境变量
	if err := c.readDotEnv(dotEnvFilePath); err != nil {
		return nil, err
	}
	file := &config_file{
		builder: &strings.Builder{},
	}
	// 预处理
	if err := c.preprocess(file, "", configFilePath); err != nil {
		return nil, err
	}
	// 处理模板变量
	if len(file.templates) > 0 {
		envData := make(map[string]interface{})
		for _, filePath := range file.templates {
			if envData, err = c.readTemplateData(filePath); err != nil {
				return nil, err
			}
		}
		// 替换环境变量
		t := template.New("config").Delims("<<", ">>")
		t.Funcs(c.templateFuncs())
		t, err = t.Parse(file.String())
		if err != nil {
			return nil, err
		}
		out := strings.Builder{}
		if err := t.Execute(&out, envData); err != nil {
			return nil, err
		}
		// log.Infof("替换环境变量后\n%v\n", out.String())
		return []byte(out.String()), nil
	} else {
		// 替换环境变量
		t := template.New("config").Delims("<<", ">>")
		t.Funcs(c.templateFuncs())
		t, err = t.Parse(file.String())
		if err != nil {
			return nil, err
		}
		out := strings.Builder{}
		if err := t.Execute(&out, nil); err != nil {
			return nil, err
		}
		// log.Infof("替换环境变量后\n%v\n", out.String())
		return []byte(out.String()), nil
	}
}

// 加载模板变量并返回
func (c *config_reader) readTemplateData(filePath string) (map[string]interface{}, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, errors.NewSyntaxError(err.Error(), filePath, "")
	}
	file := &config_file{
		builder: &strings.Builder{},
	}
	if err := c.preprocess(file, "", filePath); err != nil {
		return nil, err
	}
	var err error
	// 替换环境变量
	t := template.New("config").Delims("<<", ">>")
	t.Funcs(c.templateFuncs())
	t, err = t.Parse(file.String())
	if err != nil {
		return nil, errors.NewSyntaxError(err.Error(), filePath, file.String())
	}
	out := strings.Builder{}
	if err := t.Execute(&out, nil); err != nil {
		return nil, errors.NewSyntaxError(err.Error(), filePath, file.String())
	}
	envData := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(out.String()), envData); err != nil {
		return nil, errors.NewSyntaxError(err.Error(), filePath, file.String())
	}
	return envData, nil
}

// 预处理文件filePath,输出到file中
// include指令
// template指令
// 替换${var}成环境变量
func (c *config_reader) preprocess(file *config_file, indent string, filePath string) error {
	dir := path.Dir(filePath)
	lines, err := os.Open(filePath)
	if err != nil {
		return errors.NewSyntaxError(err.Error(), filePath, "")
	}
	// 正则表达式，用于匹配形如${VAR_NAME}的环境变量
	re := regexp.MustCompile(`\${\w+}`)
	scanner := bufio.NewScanner(lines)
	scanner.Split(bufio.ScanLines)
	instruction := config_instruction_type_none
	for scanner.Scan() {
		line := scanner.Text()
		sline := strings.TrimSpace(line)
		if instruction == config_instruction_type_none {
			if strings.HasPrefix(sline, `$include:`) {
				instruction = config_instruction_type_include
			} else if strings.HasPrefix(sline, `$template:`) {
				instruction = config_instruction_type_template
			} else {
				// 循环匹配所有环境变量
				for _, match := range re.FindAllString(line, -1) {
					// 获取环境变量的值
					envName := match[2 : len(match)-1]
					envValue := os.Getenv(envName)
					// 将环境变量替换为其值
					line = strings.Replace(line, match, envValue, 1)
				}
				// 循环匹配所有环境变量
				file.WriteString(indent)
				file.WriteString(line)
				file.WriteString("\n")
			}
		}
		if instruction == config_instruction_type_include {
			var includeFilePath string
			var found bool = false
			if strings.HasPrefix(sline, `$include:`) {
				pats := strings.SplitN(line, ":", 2)
				if len(pats) == 2 {
					includeFilePath = strings.TrimSpace(pats[1])
				}
				found = true
			} else {
				str := strings.TrimSpace(line)
				if strings.HasPrefix(str, "-") {
					str = strings.Replace(line, `-`, "", 1)
					includeFilePath = strings.TrimSpace(str)
					found = true
				}
			}
			if len(includeFilePath) > 0 {
				indent2 := strings.Replace(line, sline, "", 1) + indent
				if err := c.preprocess(file, indent2, path.Join(dir, includeFilePath)); err != nil {
					return err
				}
			}
			if !found {
				instruction = config_instruction_type_none
				file.WriteString(indent)
				file.WriteString(line)
				file.WriteString("\n")
			}
		}
		if instruction == config_instruction_type_template {
			var templateFilePath string
			var found bool = false
			if strings.HasPrefix(sline, `$template:`) {
				pats := strings.SplitN(line, ":", 2)
				if len(pats) == 2 {
					templateFilePath = strings.TrimSpace(pats[1])
				}
				found = true
			} else {
				str := strings.TrimSpace(line)
				if strings.HasPrefix(str, "-") {
					str = strings.Replace(line, `-`, "", 1)
					templateFilePath = strings.TrimSpace(str)
					found = true
				}
			}
			if len(templateFilePath) > 0 {
				file.templates = append(file.templates, path.Join(dir, templateFilePath))
			}
			if !found {
				instruction = config_instruction_type_none
				file.WriteString(indent)
				file.WriteString(line)
				file.WriteString("\n")
			}
		}
	}
	lines.Close()
	return nil
}
