package gen_macro

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	log "github.com/Lyndon-Zhang/gira/corelog"

	"github.com/Lyndon-Zhang/gira/proj"
)

/// 每个目录生成一个文件

type Arg struct {
	Name  string
	Type  string
	Ptr   string
	IsPtr bool
}

type Method struct {
	Line          string
	Declaration   string // 方法声明
	ReceiverName  string
	ReceiverIsPtr bool
	ReceiverPtr   string
	ReceiverType  string
	MethodName    string
	Args          []*Arg
	Returns       []*Arg

	ReturnsHead []*Arg
	ReturnsTail *Arg
	Arg0        *Arg
	Arg1        *Arg
}

type Macro struct {
	file       *os.File
	FilePath   string    // 所有的文件
	Type       MacroType //类型
	MacroFuncs []*MacroFunc
	Method     *Method
}

type MacroFunc struct {
	Line string
	Name string
	Args []string
	Arg0 string
	Arg1 string
	Arg2 string
	Arg3 string
	Arg4 string
}

func scanDirFiles(config *Config) map[string][]string {
	dirArr := make([]string, 0)
	dirArr = append(dirArr, proj.Dir.SrcDir)
	if config.SrcDirs != nil {
		dirArr = append(dirArr, config.SrcDirs...)
	}
	files := make(map[string][]string, 0)
	for _, dir := range dirArr {
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if strings.HasPrefix(path, proj.Dir.SrcGenDir) {
			} else if strings.HasPrefix(path, proj.Dir.SrcTestDir) {
			} else if !d.IsDir() && strings.HasSuffix(d.Name(), "_test.go") {
			} else if !d.IsDir() && strings.HasSuffix(d.Name(), "_macro.go") {
			} else if !d.IsDir() && strings.HasSuffix(d.Name(), ".macro.go") {
			} else if !d.IsDir() && strings.HasSuffix(d.Name(), ".pb.go") {
			} else if !d.IsDir() && !strings.HasSuffix(d.Name(), ".go") {
			} else {
				if d.IsDir() {
					if _, ok := files[path]; !ok {
						files[path] = make([]string, 0)
					}
				} else {
					d := filepath.Dir(path)
					arr, _ := files[d]
					arr = append(arr, path)
					files[d] = arr
				}
			}
			return nil
		})
	}
	return files
}

func parseReturns(line string) []*Arg {
	args := parseArgs(line)
	for k, v := range args {
		if v.Name == "" {
			v.Name = fmt.Sprintf("r%d", k)
		}
	}
	return args
}

func parseArgs(line string) []*Arg {
	line = strings.TrimSpace(line)
	args := make([]*Arg, 0)
	if len(line) == 0 {
		return args
	}
	re := regexp.MustCompile(`(\S+)\s*(\*?)\s*(\S*)`)

	pat1 := strings.Split(line, ",")
	for _, v := range pat1 {
		matches := re.FindStringSubmatch(v)
		if len(matches) > 0 {
			arg := &Arg{}
			arg.Name = matches[1]
			arg.Ptr = matches[2]
			if matches[2] == "" {
				arg.IsPtr = false
			} else {
				arg.IsPtr = true
			}
			arg.Type = matches[3]
			args = append(args, arg)
		}
	}
	for k, v := range args {
		if v.Type == "" {
			v.Type = v.Name
			v.Name = fmt.Sprintf("r%d", k)
		}
	}
	/*
		k := len(args) - 1
		for {
			if k <= 0 {
				break
			}
			if args[k-1].Type == "" {
				args[k-1].Type = args[k].Type
			}
			k = k - 1
		}*/
	return args
}

func scanMacros(files map[string][]string) (map[string][]*Macro, error) {
	results := make(map[string][]*Macro, 0)
	reMacroFunc := regexp.MustCompile(`// #(\w+)\(([^)]*)\)`)
	reFunc := regexp.MustCompile(`^func\s+\(([a-zA-Z0-9_]+)\s+([*]*)([a-zA-Z0-9_]+)\)\s+([a-zA-Z0-9_]+)\((.*?)\)\s*[\(]*(.*?)[\)]*\s*\{$`)
	for dir, arr := range files {
		for _, path := range arr {
			f, err := os.OpenFile(path, os.O_RDONLY, 0666)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			reader := bufio.NewReader(f)
			var lines []string
			// 按行处理txt
			for {
				line, _, err := reader.ReadLine()
				if err == io.EOF {
					break
				}
				lines = append(lines, string(line))
			}
			macro := &Macro{
				FilePath: path,
			}
			for _, v := range lines {
				if strings.HasPrefix(v, "// #") {
					// log.Info(v)
					matches := reMacroFunc.FindStringSubmatch(v)
					if len(matches) > 0 {
						macroName := matches[1]
						args := matches[2]
						// 处理 arg1 和 arg2
						// log.Info(macroName, args)
						macroFunc := &MacroFunc{}
						macroFunc.Name = macroName
						macroFunc.Line = v
						args = strings.TrimSpace(args)
						if len(args) > 0 {
							macroFunc.Args = strings.Split(args, ",")
						}
						// log.Info(mf)
						macro.MacroFuncs = append(macro.MacroFuncs, macroFunc)
					}
				} else if len(macro.MacroFuncs) > 0 {
					if strings.HasPrefix(v, "func") || strings.HasPrefix(v, "type") {
						// log.Info(v)
						match := reFunc.FindStringSubmatch(v)
						if len(match) == 7 {
							receiverName := match[1]
							receiverPtr := match[2]
							receiverType := match[3]
							methodName := match[4]
							args := match[5]
							returnType := match[6]
							// fmt.Printf("Receiver type: %s\n", receiverType)
							// fmt.Printf("Receiver ptr: %s\n", receiverPtr)
							// fmt.Printf("Receiver name: %s\n", receiverName)
							// fmt.Printf("Method name: %s\n", methodName)
							// fmt.Printf("Args: %s\n", args)
							// fmt.Printf("Return type: %s\n", returnType)
							declaration := strings.Replace(v, "{", "", 1)
							method := &Method{
								Line:         v,
								Declaration:  declaration,
								ReceiverName: receiverName,
								ReceiverType: receiverType,
								MethodName:   methodName,
							}
							if receiverPtr == "" {
								method.ReceiverIsPtr = false
							} else {
								method.ReceiverIsPtr = true
							}
							method.ReceiverPtr = receiverPtr
							method.Args = parseArgs(args)
							method.Returns = parseReturns(returnType)
							//log.Infof("%#v\n", method)
							macro.Method = method
						} else {
							fmt.Println("No match found")
							fmt.Println(v)
						}
						if arr, ok := results[dir]; !ok {
							arr = make([]*Macro, 0)
							arr = append(arr, macro)
							results[dir] = arr
						} else {
							arr = append(arr, macro)
							results[dir] = arr
						}
						macro = &Macro{
							FilePath: path,
						}
					}
				}
			}
		}
	}
	return results, nil
}

type GenState struct {
	Package string
}

var code = `// afafa
`

type Config struct {
	SrcDirs []string
}

func Gen(config *Config) error {
	log.Info("===============gen macro start===============")
	log.Info(config.SrcDirs)
	files := scanDirFiles(config)
	if fileMacros, err := scanMacros(files); err != nil {
		return err
	} else {
		// 补充一些字段
		for _, macros := range fileMacros {
			for _, macro := range macros {
				if macro.Method != nil {
					method := macro.Method
					macro.Type = MacroTypeMethod
					method.ReturnsHead = make([]*Arg, 0)
					for i := 0; i < len(method.Returns)-1; i++ {
						method.ReturnsHead = append(method.ReturnsHead, method.Returns[i])
					}
					if len(method.Returns) > 0 {
						method.ReturnsTail = method.Returns[len(method.Returns)-1]
					}

					if len(method.Args) > 0 {
						method.Arg0 = method.Args[0]
					}
					if len(method.Args) > 1 {
						method.Arg1 = method.Args[1]
					}
				}
				for _, m := range macro.MacroFuncs {
					if len(m.Args) > 0 {
						m.Arg0 = m.Args[0]
					}
					if len(m.Args) > 1 {
						m.Arg1 = m.Args[1]
					}
					if len(m.Args) > 2 {
						m.Arg2 = m.Args[2]
					}
					if len(m.Args) > 3 {
						m.Arg3 = m.Args[3]
					}
					if len(m.Args) > 4 {
						m.Arg4 = m.Args[4]
					}
				}
			}
		}
		// 找出包含宏的文件
		fs := make(map[string]*os.File)
		for _, macros := range fileMacros {
			for _, v := range macros {
				if f, ok := fs[v.FilePath]; !ok {
					f, err := os.OpenFile(v.FilePath, os.O_WRONLY|os.O_APPEND, 0644)
					if err != nil {
						return err
					}
					v.file = f
					fs[v.FilePath] = f
				} else {
					v.file = f
				}
			}
		}
		for path, f := range fs {
			all, err := ioutil.ReadFile(path)
			if err != nil {
				log.Info("b============", path)
				return err
			}
			var index int
			if index = bytes.Index(all, []byte(`
/// 宏展开的地方，不要在文件末尾添加代码============`)); index > 0 {
			} else {
			}
			if index == -1 {
				f.Seek(0, 2)
			} else {
				f.Truncate(int64(index))
				f.Seek(0, 2)
			}
			f.WriteString(`
/// 宏展开的地方，不要在文件末尾添加代码============`)
		}

		for _, f := range fs {
			tmpl := template.New("macro").Delims("<<", ">>")
			if tmpl, err := tmpl.Parse(code); err != nil {
				return err
			} else {
				if err := tmpl.Execute(f, nil); err != nil {
					return err
				}
			}
		}

		// 生成代码
		for dir, macros := range fileMacros {
			// 进行分组,方法宏,且接收类型一致的放在同一组
			methodMacro := make(map[string][]*Macro)
			log.Info(dir)
			for _, v := range macros {
				// log.Infof("%#v\n", v)
				// log.Infof("%#v\n", v.Method)
				if v.Method != nil {
					if _, ok := methodMacro[v.Method.ReceiverType]; !ok {
						methodMacro[v.Method.ReceiverType] = make([]*Macro, 0)
					}
					methodMacro[v.Method.ReceiverType] = append(methodMacro[v.Method.ReceiverType], v)
				}
			}
			for _, macro := range macros {
				sb := strings.Builder{}
				if dict, ok := builders[macro.Type]; !ok {
					log.Infof("macro %s not found", macro.Type)
				} else {
					for _, macroFunc := range macro.MacroFuncs {
						log.Println(macroFunc.Line)
						if handler, ok := dict[macroFunc.Name]; !ok {
							log.Infof("macro %s not found", macroFunc.Name)
						} else {
							if err := handler.Check(macroFunc, macro.Method); err != nil {
								return err
							} else {
								handler.Gen(macroFunc, macro.Method, &sb)
							}
						}
					}
					log.Println("方法宏", macro.Method.Declaration)
					log.Println()
				}
				macro.file.WriteString(sb.String())
			}
		}
		for _, f := range fs {
			f.WriteString(`
/// =============宏展开的地方，不要在文件末尾添加代码============`)
		}
	}
	log.Info("===============gen macro finished===============")
	return nil
}

type MacroType int

func (t MacroType) String() string {
	switch t {
	case MacroTypeMethod:
		return "method"
	}
	return fmt.Sprintf("%d", t)
}

const (
	MacroTypeMethod = 1
)

func LtChar() interface{} {
	return "<"
}

func JoinArgs(prefix string, args []*Arg) interface{} {
	sb := strings.Builder{}
	for k, v := range args {
		if k == 0 {
			sb.WriteString(fmt.Sprintf("%s%s", prefix, v.Name))
		} else {
			sb.WriteString(fmt.Sprintf(", %s%s", prefix, v.Name))
		}
	}
	return sb.String()
}

func JoinArgsTail(offset int, prefix string, args []*Arg) interface{} {
	newArgs := args[offset:]
	sb := strings.Builder{}
	for k, v := range newArgs {
		if k == 0 {
			sb.WriteString(fmt.Sprintf("%s%s", prefix, v.Name))
		} else {
			sb.WriteString(fmt.Sprintf(", %s%s", prefix, v.Name))
		}
	}
	return sb.String()
}

func JoinArgsWithType(args []*Arg) interface{} {
	sb := strings.Builder{}
	for k, v := range args {
		if k == 0 {
			sb.WriteString(fmt.Sprintf("%s %s%s", v.Name, v.Ptr, v.Type))
		} else {
			sb.WriteString(fmt.Sprintf(", %s %s%s", v.Name, v.Ptr, v.Type))
		}
	}
	return sb.String()
}

func JoinArgsTailWithType(offset int, args []*Arg) interface{} {
	newArgs := args[offset:]
	sb := strings.Builder{}
	for k, v := range newArgs {
		if k == 0 {
			sb.WriteString(fmt.Sprintf("%s %s%s", v.Name, v.Ptr, v.Type))
		} else {
			sb.WriteString(fmt.Sprintf(", %s %s%s", v.Name, v.Ptr, v.Type))
		}
	}
	return sb.String()
}

type MacroGenerator interface {
	Gen(f *MacroFunc, method *Method, sb *strings.Builder)
	Check(f *MacroFunc, method *Method) error
}

type MethodMacroGenerator struct {
}

func (self *MethodMacroGenerator) Check(macro *MacroFunc, method *Method) error {
	if len(macro.Args) != 1 {
		return fmt.Errorf("actor macro need 1 argument, %s", method.Declaration)
	}
	if macro.Arg0 != "call" && macro.Arg0 != "send" {
		return fmt.Errorf("actor macro argument 1 must call or send, %s", method.Declaration)
	}
	if macro.Arg0 == "send" {
		if len(method.Returns) != 1 {
			return fmt.Errorf("actor(send) macro need 1 return, %s", method.Declaration)
		}
		if method.Returns[0].Type != "error" {
			return fmt.Errorf("actor(send) macro need return 1 type must error, %s", method.Declaration)
		}
	} else if macro.Arg0 == "call" {
		if len(method.Args) <= 0 || method.Arg0.Type != "context.Context" {
			return fmt.Errorf("actor(call) macro first argument must context.Context, %s", method.Declaration)
		}
		if len(method.Returns) <= 0 {
			return fmt.Errorf("actor(call) macro need at least 1 return, %s", method.Declaration)
		}
		if method.ReturnsTail.Type != "error" {
			return fmt.Errorf("actor(call) macro last return type must error, %s", method.Declaration)
		}
	}
	return nil
}

func (self *MethodMacroGenerator) Gen(macro *MacroFunc, method *Method, sb *strings.Builder) {
	funcMap := template.FuncMap{
		"lt":                       LtChar,
		"join_args":                JoinArgs,
		"join_args_tail":           JoinArgsTail,
		"join_args_with_type":      JoinArgsWithType,
		"join_args_tail_with_type": JoinArgsTailWithType,
	}
	code = `
type <<.Method.ReceiverType>><<.Method.MethodName>>Argument struct {
	<< .Method.ReceiverName>> << .Method.ReceiverPtr>><<.Method.ReceiverType>>
	<<- range .Method.Args>>
	<<.Name>> <<.Ptr>><<.Type>>
	<<- end>>
	<<- range .Method.Returns>>
	<<.Name>> <<.Ptr>><<.Type>>
	<<- end>>
	<<- if eq .Macro.Arg0 "call">>
	__caller__ chan*<<.Method.ReceiverType>><<.Method.MethodName>>Argument
	<<- end>>
}

func (__arg__ *<<.Method.ReceiverType>><<.Method.MethodName>>Argument) Next() {
	<<join_args "__arg__." .Method.Returns>> = __arg__.<<.Method.ReceiverName>>.<<.Method.MethodName>>(<<join_args "__arg__." .Method.Args>>)
	<<- if eq .Macro.Arg0 "call">>
	__arg__.__caller__ <- __arg__
	<<- end>>
}

func (<<.Method.ReceiverName>> <<.Method.ReceiverPtr>><<.Method.ReceiverType>>) Call_<<.Method.MethodName>> (<<join_args_with_type .Method.Args>>, opts ...actor.CallOption) (<<join_args_with_type .Method.Returns>>){
	var __options__ actor.CallOptions
	for _, v := range opts {
		v.Config(&__options__)	
	}
	__arg__ := &<<.Method.ReceiverType>><<.Method.MethodName>>Argument {
		<< .Method.ReceiverName>>: << .Method.ReceiverName>>,
		<<- range .Method.Args>>
		<<.Name>>: <<.Name>>, 
		<<- end>>
	<<- if eq .Macro.Arg0 "call">>
		__caller__: make(chan*<<.Method.ReceiverType>><<.Method.MethodName>>Argument),
	<<- end>>
	}
	<<.Method.ReceiverName>>.Inbox() <- __arg__
	<<- if eq .Macro.Arg0 "call">>
	if __options__.TimeOut != 0 {
		__timer__ := time.NewTimer(__options__.TimeOut)
		defer __timer__.Stop()
		select {
		case resp :=<-__arg__.__caller__:
			return <<join_args "resp." .Method.Returns>>
		case <-<<.Method.Arg0.Name>>.Done():
			return <<- range .Method.ReturnsHead>> __arg__.<<.Name>>, <<- end>> <<.Method.Arg0.Name>>.Err()
		case <-__timer__.C:
	        log.Errorw("actor call time out", "func", "<<.Method.Declaration>>")
			return <<- range .Method.ReturnsHead>> __arg__.<<.Name>>, <<- end>> errors.ErrActorCallTimeOut
		}
	} else {
		select {
		case resp :=<-__arg__.__caller__:
			return <<join_args "resp." .Method.Returns>>
		case <-<<.Method.Arg0.Name>>.Done():
			return <<- range .Method.ReturnsHead>> __arg__.<<.Name>>, <<- end>> <<.Method.Arg0.Name>>.Err()
		}
	}
	<<- else >>
	return nil
	<<- end>>
}



	`
	tmpl := template.New("macro").Delims("<<", ">>")
	tmpl.Funcs(funcMap)
	if tmpl, err := tmpl.Parse(code); err != nil {
		log.Info(err)
		return
	} else {
		state := map[string]interface{}{
			"Method": method,
			"Macro":  macro,
		}
		if err := tmpl.Execute(sb, state); err != nil {
			log.Info(err)
			return
		}
	}
}

var builders = map[MacroType]map[string]MacroGenerator{
	MacroTypeMethod: {
		"actor": &MethodMacroGenerator{},
	},
}
