package gen_const

import (
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	log "github.com/Lyndon-Zhang/gira/corelog"

	"github.com/Lyndon-Zhang/gira/proj"
	excelize "github.com/xuri/excelize/v2"
)

var const_template = `
// Code generated by github.com/Lyndon-Zhang/gira. DO NOT EDIT.
// Code generated by github.com/Lyndon-Zhang/gira. DO NOT EDIT.
// Code generated by github.com/Lyndon-Zhang/gira. DO NOT EDIT.

package <<.Descriptor.Name>>

<<- if eq .Descriptor.Name "ec">>

import (
	"github.com/Lyndon-Zhang/gira/codes"
)

<<- $commentField := .CommentField >>
<<- $keyField := .KeyField >>
<<- $valueField := .ValueField >>

const (
<<- range $row := .ValueArr >>
	<<- $key := index $row $keyField.Tag >>
	<<- $value := index $row $valueField.Tag >>
	// << index $row $commentField.Tag >>
	Code<<camelString $key>> = <<$value>>
<<- end>>
)

const (
<<- range $row := .ValueArr >>
	<<- $key := index $row $keyField.Tag >>
	<<- $value := index $row $valueField.Tag >>
	// << index $row $commentField.Tag >>
	msg_<<camelString $key>> = "<<index $row $commentField.Tag>>"
<<- end>>
)
var (
<<- range $row := .ValueArr >>
	<<- $key := index $row $keyField.Tag >>
	<<- $value := index $row $valueField.Tag >>
	// << index $row $commentField.Tag >>
	Err<<camelString $key>> = codes.New(Code<<camelString $key>>, msg_<<camelString $key>>)
<<- end>>
)


<<- range $row := .ValueArr >>
<<- $key := index $row $keyField.Tag >>
<<- $value := index $row $valueField.Tag >>
// << index $row $commentField.Tag >>
func TraceErr<<camelString $key>>(values ...interface{}) *codes.TraceError {
	return Err<<camelString $key>>.TraceWithSkip(1, values...)
}
<<- end>>
<<- else >>

<<- $commentField := .CommentField >>
<<- $keyField := .KeyField >>
<<- $valueField := .ValueField >>
<<- range $row := .ValueArr >>
	<<- $key := index $row $keyField.Tag >>
	<<- $value := index $row $valueField.Tag >>
	<<- if $commentField >>
// << index $row $commentField.Tag >>
const << camelString $key>> = <<$value>>
	<<- else>>
const << camelString $key>> = "<<$value>>"
	<<- end>>
<<- end>>

<<- end>>
`

// 字段类型
type field_type int

const (
	field_type_int field_type = iota
	field_type_string
	field_type_json
)

var type_name_dict = map[string]field_type{
	"int":    field_type_int,
	"string": field_type_string,
	"json":   field_type_json,
}

var go_type_name_dict = map[field_type]string{
	field_type_int:    "int",
	field_type_string: "string",
	field_type_json:   "interface{}",
}

// 字段结构
type Field struct {
	Tag      int
	name     string     // 字段名
	Type     field_type // 字段类型
	typeName string
}

type excel_data struct {
	FieldDict    map[string]*Field // 字段信息
	FieldArr     []*Field          // 字段信息
	ValueArr     [][]interface{}   // 字段值
	KeyField     *Field
	ValueField   *Field
	CommentField *Field
	Descriptor   *Descriptor
}

type Descriptor struct {
	fieldDict map[string]*Field // 字段信息
	filePath  string
	Name      string
	keyArr    []string
}

type const_file struct {
	descriptorDict map[string]*Descriptor
}

// 生成协议的状态
type const_state struct {
	constFile     const_file
	excelDataDict map[string]*excel_data
}

type Parser interface {
	parse(constState *const_state) error
}

func capUpperString(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func upperString(s string) string {
	return strings.ToUpper(s)
}

func camelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

func genConstFile(constState *const_state, descriptor *Descriptor, data *excel_data) error {

	log.Info("gen const ", descriptor.Name)
	dir := path.Join(proj.Dir.SrcGenConstDir, descriptor.Name)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	filePath := path.Join(dir, fmt.Sprintf("%s.go", descriptor.Name))
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	file.Truncate(0)
	defer file.Close()

	funcMap := template.FuncMap{
		"join":        strings.Join,
		"capUpper":    capUpperString,
		"upper":       upperString,
		"camelString": camelString,
	}

	tmpl := template.New("const").Delims("<<", ">>")
	tmpl.Funcs(funcMap)
	if tmpl, err := tmpl.Parse(const_template); err != nil {
		return err
	} else {
		if err := tmpl.Execute(file, data); err != nil {
			return err
		}
	}

	return nil
}

func (r *excel_data) read(name string, descriptor *Descriptor, filePath string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Info(err)
		return err
	}
	// 获取 Sheet1 上所有单元格
	rows, err := f.GetRows("Sheet1")
	typeRow := rows[4]
	nameRow := rows[3]
	// 字段名
	for index, v := range nameRow {
		if v != "" {
			typeName := typeRow[index]
			// 字段类型
			if realType, ok := type_name_dict[typeRow[index]]; ok {
				field := &Field{
					name:     v,
					Type:     realType,
					Tag:      index,
					typeName: go_type_name_dict[realType],
				}
				r.FieldArr = append(r.FieldArr, field)
				r.FieldDict[v] = field
			} else {
				return fmt.Errorf("invalid type %s", typeName)
			}
		}
	}
	descriptor.fieldDict = r.FieldDict
	// 值
	for index, row := range rows {
		if index <= 4 {
			continue
		}
		valueArr := make([]interface{}, 0)
		for _, field := range r.FieldArr {
			var v interface{}
			if len(row) > field.Tag {
				v = row[field.Tag]
			} else {
				v = ""
			}
			if field.Type == field_type_string {
			} else if field.Type == field_type_json {
			} else {
				if v == "" {
					v = 0
				}
			}
			valueArr = append(valueArr, v)
		}
		r.ValueArr = append(r.ValueArr, valueArr)
	}

	var ok bool
	var keyField *Field
	var valueField *Field
	var commentField *Field = nil
	keyField, ok = r.FieldDict[descriptor.keyArr[0]]
	if !ok {
		return fmt.Errorf("descriptor %s key field %s not found", descriptor.Name, descriptor.keyArr[0])
	}
	valueField, ok = r.FieldDict[descriptor.keyArr[1]]
	if !ok {
		return fmt.Errorf("descriptor %s value field %s not found", descriptor.Name, descriptor.keyArr[1])
	}
	if len(descriptor.keyArr) == 3 {
		commentField, ok = r.FieldDict[descriptor.keyArr[2]]
		if !ok {
			return fmt.Errorf("descriptor %s comment field %s not found", descriptor.Name, descriptor.keyArr[1])
		}
	}
	r.KeyField = keyField
	r.ValueField = valueField
	r.CommentField = commentField
	return nil
}

func parse(constState *const_state) error {
	var p Parser
	if true {
		p = &golang_parser{}
	} else {
		p = &yaml_parser{}
	}
	if err := p.parse(constState); err != nil {
		return err
	}
	// 读取excel文件
	for name, v := range constState.constFile.descriptorDict {
		filePath := path.Join(proj.Dir.ExcelDir, v.filePath)
		resource := &excel_data{
			Descriptor: v,
			FieldDict:  make(map[string]*Field, 0),
			FieldArr:   make([]*Field, 0),
			ValueArr:   make([][]interface{}, 0),
		}
		if err := resource.read(name, v, filePath); err != nil {
			return err
		}
		constState.excelDataDict[name] = resource
	}
	return nil
}

func genCode(constState *const_state) error {
	log.Info("生成go文件")
	// 生成cost文件夹
	dir := proj.Dir.SrcGenConstDir
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	// 生成代码
	for name, descriptor := range constState.constFile.descriptorDict {
		if resourceData, ok := constState.excelDataDict[name]; ok {
			if err := genConstFile(constState, descriptor, resourceData); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%s not found", name)
		}
	}
	return nil
}

// 生成协议
func Gen() error {
	log.Info("===============gen const start===============")
	// 初始化
	constState := &const_state{
		excelDataDict: make(map[string]*excel_data),
		constFile: const_file{
			descriptorDict: make(map[string]*Descriptor, 0),
		},
	}
	if err := parse(constState); err != nil {
		return err
	}
	if err := genCode(constState); err != nil {
		return err
	}
	log.Info("===============gen const finished===============")
	return nil
}
