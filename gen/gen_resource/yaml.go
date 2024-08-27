package gen_resource

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/proj"
	"gopkg.in/yaml.v3"
)

type sort_field_by_name []*Field

func (self sort_field_by_name) Len() int           { return len(self) }
func (self sort_field_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_field_by_name) Less(i, j int) bool { return self[i].Tag < self[j].Tag }

type sort_resource_by_name []*Resource

func (self sort_resource_by_name) Len() int           { return len(self) }
func (self sort_resource_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_resource_by_name) Less(i, j int) bool { return self[i].StructName < self[j].StructName }

type sort_bundle_by_name []*Bundle

func (self sort_bundle_by_name) Len() int           { return len(self) }
func (self sort_bundle_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_bundle_by_name) Less(i, j int) bool { return self[i].BundleName < self[j].BundleName }

type sort_loader_by_name []*Loader

func (self sort_loader_by_name) Len() int           { return len(self) }
func (self sort_loader_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_loader_by_name) Less(i, j int) bool { return self[i].LoaderName < self[j].LoaderName }

type yaml_parser struct {
}

func (p *yaml_parser) parseStruct(resource *Resource, attrs map[string]interface{}) error {
	spaceRegexp := regexp.MustCompile("[^\\s]+")
	equalRegexp := regexp.MustCompile("[^=]+")
	fileArr := make([]*Field, 0)

	for valueStr, v := range attrs {
		var tag int
		var err error
		var fieldName string
		var typeStr string
		var tagStr string
		var optionArr []interface{}
		switch v.(type) {
		case nil:
			break
		case []interface{}:
			optionArr = v.([]interface{})
		default:
			return fmt.Errorf("%+v invalid11", v)
		}
		args := equalRegexp.FindAllString(valueStr, -1)
		if len(args) != 2 {
			return fmt.Errorf("%s invalid", valueStr)
		}
		tagStr = strings.TrimSpace(args[1])
		if tag, err = strconv.Atoi(tagStr); err != nil {
			return err
		}
		args = spaceRegexp.FindAllString(args[0], -1)
		if len(args) != 2 {
			return fmt.Errorf("%s invalid", valueStr)
		}
		fieldName = args[0]
		typeStr = args[1]
		field := &Field{
			FieldName:       fieldName,
			StructFieldName: camelString(fieldName),
			Tag:             tag,
		}
		if typeValue, ok := type_name_dict[typeStr]; ok {
			field.Type = typeValue
			field.GoTypeName = go_type_name_dict[field.Type]
		} else {
			field.Type = field_type_struct
			field.GoTypeName = typeStr
		}
		for _, option := range optionArr {
			log.Println(option)
			optionDict := option.(map[string]interface{})
			if comment, ok := optionDict["comment"]; ok {
				field.Comment = comment.(string)
			}
		}
		if last, ok := resource.FieldDict[fieldName]; ok {
			resource.FieldDict[fieldName] = field
			// 替换之前的
			for k, v := range resource.FieldArr {
				if v == last {
					resource.FieldArr[k] = field
					break
				}
			}
		} else {
			resource.FieldDict[fieldName] = field
			fileArr = append(fileArr, field)

		}
	}
	sort.Sort(sort_field_by_name(fileArr))
	resource.FieldArr = append(resource.FieldArr, fileArr...)
	return nil
}

func (p *yaml_parser) parse(state *gen_state) error {
	// srcFilePathArr := make([]string, 0)
	// srcFilePathArr = append(srcFilePathArr, proj.Config.DocResourceFilePath)

	if data, err := ioutil.ReadFile(proj.Dir.DocResourceFilePath); err != nil {
		return err
	} else {
		kv := make(map[string]interface{})
		if err := yaml.Unmarshal(data, kv); err != nil {
			return err
		}
		structDict := make(map[string]*struct_type)
		for name, v := range kv {
			// log.Println(name)
			if name == "$import" {
				if vdata, err := yaml.Marshal(v); err != nil {
					return err
				} else {
					var importType import_type
					if err := yaml.Unmarshal(vdata, &importType); err != nil {
						return err
					} else {
						for _, v := range importType {
							state.ImportArr = append(state.ImportArr, v)
						}
					}
				}
			} else {
				if vdata, err := yaml.Marshal(v); err != nil {
					return err
				} else {
					structType := &struct_type{}
					if err := yaml.Unmarshal(vdata, &structType); err != nil {
						return err
					} else {
						structDict[name] = structType
					}
				}
			}
		}
		// if err := yaml.Unmarshal(data, &structDict); err != nil {
		// 	return err
		// }
		for name, v := range structDict {
			// bundle
			if len(v.Resources) > 0 {
				bundle := &Bundle{
					BundleType:          v.Format,
					BundleName:          name,
					BundleStructName:    name,
					CapBundleStructName: capLowerString(name),
					ResourceNameArr:     v.Resources,
					ResourceArr:         make([]*Resource, 0),
				}
				state.BundleDict[name] = bundle
				state.BundleArr = append(state.BundleArr, bundle)

			} else if len(v.Bundles) > 0 {
				// loader
				loader := &Loader{
					LoaderName:        name,
					LoaderStructName:  name,
					HandlerStructName: "I" + name + "Handler",
					bundleNameArr:     v.Bundles,
					BundleArr:         make([]*Bundle, 0),
				}
				state.LoaderArr = append(state.LoaderArr, loader)

			} else if v.Excel != "" {
				var filePath string
				filePath = v.Excel
				// 类型
				var typ resource_type = resource_type_array
				if len(v.Map) > 0 {
					typ = resource_type_map
				}
				if len(v.Object) > 0 {
					typ = resource_type_object
				}
				// 解析key arr
				// xlsx替换成yaml
				//yamlFileName := strings.Replace(filePath, filepath.Ext(filePath), ".yaml", -1)
				yamlFileName := fmt.Sprintf("%s.yaml", name)
				camelName := camelString(name)
				r := &Resource{
					FieldDict:      make(map[string]*Field, 0),
					FieldArr:       make([]*Field, 0),
					ValueArr:       make([][]interface{}, 0),
					Type:           typ,
					ResourceName:   name,
					StructName:     camelName,
					TableName:      name,
					CapStructName:  capLowerString(camelName),
					ArrTypeName:    camelName + "Arr",
					MapTypeName:    camelName + "Map",
					ObjectTypeName: camelName + "Object",
					FilePath:       filePath,
					YamlFileName:   yamlFileName,
					MapKeyArr:      v.Map,
					ObjectKeyArr:   v.Object,
				}
				if typ == resource_type_array {
					r.WrapStructName = r.ArrTypeName
				} else if typ == resource_type_map {
					r.WrapStructName = r.MapTypeName
				} else if typ == resource_type_object {
					r.WrapStructName = r.ObjectTypeName
				}
				// 解析excel
				// filePath := path.Join(proj.Config.ExcelDir, v.FilePath)
				if err := r.readExcel(path.Join(proj.Dir.ExcelDir, filePath)); err != nil {
					return err
				}
				// 解析struct
				if len(v.Struct) > 0 {
					if err := p.parseStruct(r, v.Struct); err != nil {
						return err
					}
				}
				// 处理不同的转换类型， 转map, 转object
				//GoMapTypeName
				r.GoMapTypeName = ""
				for _, v := range r.MapKeyArr {
					if field, ok := r.FieldDict[v]; ok {
						r.GoMapTypeName = r.GoMapTypeName + fmt.Sprintf(`map[%s]`, field.GoTypeName)
					} else {
						return fmt.Errorf("resource %s key %s not found\n", r.StructName, v)
					}
				}
				r.GoMapTypeName = r.GoMapTypeName + fmt.Sprintf(" *%s", r.StructName)
				// GoObjectTypeName
				r.GoObjectTypeName = ""
				for _, v := range r.ObjectKeyArr {
					if field, ok := r.FieldDict[v]; ok {
						r.GoObjectTypeName = r.GoObjectTypeName + fmt.Sprintf(`map[%s]`, field.GoTypeName)
					} else {
						return fmt.Errorf("resource %s key %s not found\n", r.StructName, v)
					}
				}
				r.GoObjectTypeName = r.GoObjectTypeName + fmt.Sprintf(" *%s", r.StructName)
				// WrapTypeName
				if r.Type == resource_type_map {
					r.WrapTypeName = r.GoMapTypeName
				} else if r.Type == resource_type_array {
					r.WrapTypeName = fmt.Sprintf("[]* %s", r.StructName)
				} else if r.Type == resource_type_object {
					for k, v := range r.FieldArr {
						if v.FieldName == r.ObjectKeyArr[0] {
							r.ObjectKeyIndex = k
							break
						}
					}
				}
				// make key时用的
				mapKeyGoTypeNameArr := make([]string, 0)
				for _, k := range r.MapKeyArr {
					f, _ := r.FieldDict[k]
					mapKeyGoTypeNameArr = append(mapKeyGoTypeNameArr, f.GoTypeName)
				}
				r.MapKeyGoTypeNameArr = mapKeyGoTypeNameArr

				// make key时用的
				objectKeyGoTypeNameArr := make([]string, 0)
				for _, k := range r.ObjectKeyArr {
					f, _ := r.FieldDict[k]
					objectKeyGoTypeNameArr = append(objectKeyGoTypeNameArr, f.GoTypeName)
				}
				r.ObjectKeyGoTypeNameArr = objectKeyGoTypeNameArr
				state.ResourceDict[name] = r
				state.ResourceArr = append(state.ResourceArr, r)
			}
		}
		for _, v := range state.BundleArr {
			for _, name := range v.ResourceNameArr {
				if r, ok := state.ResourceDict[name]; ok {
					v.ResourceArr = append(v.ResourceArr, r)
				}
			}
		}
		for _, v := range state.LoaderArr {
			for _, name := range v.bundleNameArr {
				if r, ok := state.BundleDict[name]; ok {
					v.BundleArr = append(v.BundleArr, r)
				}
			}
		}
		// 排序
		sort.Sort(sort_resource_by_name(state.ResourceArr))
		sort.Sort(sort_bundle_by_name(state.BundleArr))
		sort.Sort(sort_loader_by_name(state.LoaderArr))
	}
	return nil
}
