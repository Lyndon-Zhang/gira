package gen_resource

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path"
	"strings"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/gen"
	"github.com/Lyndon-Zhang/gira/proj"
	"gopkg.in/yaml.v3"
)

type golang_parser struct {
}

type ResourceAnnotate struct {
	Excel  string
	Map    []string
	Object []string
}

type resources_annotate struct {
	Resources string
	Version   string
	Import    []string
}

func (p *golang_parser) extraResourceAnnotate(filePath string, fset *token.FileSet, name string, s *ast.StructType, comments []*ast.Comment) (*ResourceAnnotate, error) {
	values := make(map[string]interface{})
	lines := make([]string, 0)
	for _, comment := range comments {
		if !strings.HasPrefix(comment.Text, "// @") {
			continue
		}
		lines = append(lines, strings.Replace(comment.Text, "// @", "", 1))
	}
	str := strings.Join(lines, "\n")
	if err := yaml.Unmarshal([]byte(str), values); err != nil {
		log.Printf("\n%s\n", str)
		return nil, err
	}
	annotates := &ResourceAnnotate{}
	if v, ok := values["excel"]; !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s excel annotate not set", name))
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s excel annotate must string", name))
	} else {
		annotates.Excel = str
	}
	if v, ok := values["map"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, v2 := range arr {
				annotates.Map = append(annotates.Map, v2.(string))
			}
		}
	}
	if v, ok := values["object"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, v2 := range arr {
				annotates.Object = append(annotates.Object, v2.(string))
			}
		}
	}
	return annotates, nil
}

type BundleAnnotate struct {
	Format string
}

func (p *golang_parser) extraBundleAnnotate(filePath string, fset *token.FileSet, name string, s *ast.StructType, comments []*ast.Comment) (*BundleAnnotate, error) {
	values := make(map[string]interface{})
	lines := make([]string, 0)
	for _, comment := range comments {
		if !strings.HasPrefix(comment.Text, "// @") {
			continue
		}
		lines = append(lines, strings.Replace(comment.Text, "// @", "", 1))
	}
	str := strings.Join(lines, "\n")
	if err := yaml.Unmarshal([]byte(str), values); err != nil {
		log.Printf("\n%s\n", str)
		return nil, err
	}
	annotates := &BundleAnnotate{}
	if v, ok := values["format"]; !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s format annotate not set", name))
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s format annotate must string", name))
	} else {
		annotates.Format = str
	}
	return annotates, nil
}

type loaders_annotate struct {
	Loaders string
}

func (p *golang_parser) extraLoadersAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*loaders_annotate, error) {
	values := make(map[string]interface{})
	lines := make([]string, 0)
	for _, comment := range comments {
		if !strings.HasPrefix(comment.Text, "// @") {
			continue
		}
		lines = append(lines, strings.Replace(comment.Text, "// @", "", 1))
	}
	str := strings.Join(lines, "\n")
	if err := yaml.Unmarshal([]byte(str), values); err != nil {
		log.Printf("\n%s\n", str)
		return nil, err
	}
	annotates := &loaders_annotate{}

	if v, ok := values["loaders"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s loaders annotate must string", name))
	} else {
		annotates.Loaders = str
	}
	return annotates, nil
}

func (p *golang_parser) extraResourcesAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*resources_annotate, error) {
	values := make(map[string]interface{})
	lines := make([]string, 0)
	for _, comment := range comments {
		if !strings.HasPrefix(comment.Text, "// @") {
			continue
		}
		lines = append(lines, strings.Replace(comment.Text, "// @", "", 1))
	}
	str := strings.Join(lines, "\n")
	if err := yaml.Unmarshal([]byte(str), values); err != nil {
		log.Printf("\n%s\n", str)
		return nil, err
	}
	annotates := &resources_annotate{}

	if v, ok := values["resources"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s resources annotate must string", name))
	} else {
		annotates.Resources = str
	}

	if v, ok := values["version"]; !ok {
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s version annotate must string", name))
	} else {
		annotates.Version = str
	}
	if v, ok := values["import"]; ok {
		if arr, ok := v.([]interface{}); !ok {
			return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s import annotate must string array", name))
		} else {
			for _, str := range arr {
				annotates.Import = append(annotates.Import, str.(string))
			}
		}
	}
	return annotates, nil
}

type bundles_annotate struct {
	Bundles string
}

func (p *golang_parser) extraBundlesAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*bundles_annotate, error) {
	values := make(map[string]interface{})
	lines := make([]string, 0)
	for _, comment := range comments {
		if !strings.HasPrefix(comment.Text, "// @") {
			continue
		}
		lines = append(lines, strings.Replace(comment.Text, "// @", "", 1))
	}
	str := strings.Join(lines, "\n")
	if err := yaml.Unmarshal([]byte(str), values); err != nil {
		log.Printf("\n%s\n", str)
		return nil, err
	}
	annotates := &bundles_annotate{}

	if v, ok := values["bundles"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s bundles annotate must string", name))
	} else {
		annotates.Bundles = str
	}
	return annotates, nil
}

func (p *golang_parser) parseStruct(resource *Resource, fileContent []byte, s *ast.StructType) error {
	fileArr := make([]*Field, 0)
	for _, f := range s.Fields.List {
		var fieldName string
		var typeStr string

		fieldName = f.Names[0].Name
		typeStr = string(fileContent[f.Type.Pos()-1 : f.Type.End()-1])
		field := &Field{
			FieldName:       fieldName,
			StructFieldName: camelString(fieldName),
		}
		if typeValue, ok := type_name_dict[typeStr]; ok {
			field.Type = typeValue
			field.GoTypeName = go_type_name_dict[field.Type]
		} else {
			field.Type = field_type_struct
			field.GoTypeName = typeStr
		}
		if comments, err := gen.ExtraComment(f.Doc); err != nil {
			return err
		} else {
			field.Comment = strings.Join(comments, ",")
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
	resource.FieldArr = append(resource.FieldArr, fileArr...)
	return nil
}

func (p *golang_parser) parseBundleStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, name string, doc *ast.CommentGroup, s *ast.StructType) error {
	if annotates, err := p.extraBundleAnnotate(filePath, fset, name, s, doc.List); err != nil {
		return err
	} else {
		format := annotates.Format
		resources := make([]string, 0)
		for _, v2 := range s.Fields.List {
			resources = append(resources, v2.Names[0].Name)
		}
		// log.Println("parse bundle ", name, resources)
		bundle := &Bundle{
			BundleType:          format,
			BundleName:          name,
			BundleStructName:    name,
			CapBundleStructName: capLowerString(name),
			ResourceNameArr:     resources,
			ResourceArr:         make([]*Resource, 0),
		}
		state.BundleDict[name] = bundle
		state.BundleArr = append(state.BundleArr, bundle)
	}
	return nil
}

func (p *golang_parser) parseLoaderStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, name string, s *ast.StructType) error {
	bundles := make([]string, 0)
	for _, v2 := range s.Fields.List {
		bundles = append(bundles, v2.Names[0].Name)
	}
	// log.Println("parse loader ", name, bundles)
	loader := &Loader{
		LoaderName:        name,
		LoaderStructName:  name,
		HandlerStructName: "I" + name + "Handler",
		bundleNameArr:     bundles,
		BundleArr:         make([]*Bundle, 0),
	}
	state.LoaderArr = append(state.LoaderArr, loader)
	return nil
}

func (p *golang_parser) newAstError(fset *token.FileSet, filePath string, pos token.Pos, msg string) error {
	return fmt.Errorf("%s %v %s", filePath, fset.Position(pos), msg)
}

func (p *golang_parser) parseResourceStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, name string, doc *ast.CommentGroup, s *ast.StructType) error {
	// log.Println("parse resource ", name)
	if annotates, err := p.extraResourceAnnotate(filePath, fset, name, s, doc.List); err != nil {
		return err
	} else {
		filePath := annotates.Excel
		var keyArr []string
		// 类型
		var typ resource_type = resource_type_array
		// map类型
		if len(annotates.Map) > 0 {
			typ = resource_type_map
			keyArr = annotates.Map
		}
		// object类型
		if len(annotates.Object) > 0 {
			typ = resource_type_object
			keyArr = annotates.Object
		}
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
			MapKeyArr:      keyArr,
			ObjectKeyArr:   keyArr,
			KeyArr:         keyArr,
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
		// 解析struct, 附加或者覆盖字段
		if err := p.parseStruct(r, fileContent, s); err != nil {
			return err
		}
		// 处理不同的转换类型， 转map, 转object
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
	return nil
}

func (p *golang_parser) parse(state *gen_state) (err error) {
	filePath := path.Join(proj.Dir.DocDir, "resource.go")
	fset := token.NewFileSet()
	var f *ast.File
	var fileContent []byte
	fileContent, err = ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	f, err = parser.ParseFile(fset, "", fileContent, parser.ParseComments)
	if err != nil {
		err = fmt.Errorf("%s %s", filePath, err)
		return
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if err != nil {
			return false
		}
		switch x := n.(type) {
		// case *ast.TypeSpec:
		// 	if x.Name.Name == "Resource" {
		// 		if err = p.parseResourceStruct(state, filePath, fileContent, fset, x.Type.(*ast.StructType)); err != nil {
		// 			return false
		// 		}
		// 	} else if x.Name.Name == "Bundle" {
		// 		if err = p.parseBundleStruct(state, filePath, fileContent, fset, x.Type.(*ast.StructType)); err != nil {
		// 			return false
		// 		}
		// 	} else if x.Name.Name == "Loader" {
		// 		if err = p.parseLoaderStruct(state, x.Type.(*ast.StructType)); err != nil {
		// 			return false
		// 		}
		// 	}
		case *ast.GenDecl:
			for _, spec := range x.Specs {
				switch st := spec.(type) {
				case *ast.TypeSpec:
					if s, ok := st.Type.(*ast.StructType); ok {
						if err = p.parseResourcesStruct(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
						if err = p.parseBundlesStruct(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
						if err = p.parseLoadersStruct(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
					}
				}
			}
		}
		return true
	})
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
	return
}

func (p *golang_parser) parseResourcesStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
		return nil
	}
	if annotations, err := p.extraResourcesAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	} else {
		state.ResVersion = annotations.Version
		state.ImportArr = annotations.Import
	}
	for _, field := range s.Fields.List {
		if err := p.parseResourceStruct(state, filePath, fileContent, fset, field.Names[0].Name, field.Doc, field.Type.(*ast.StructType)); err != nil {
			return err
		}
	}
	return nil
}

func (p *golang_parser) parseLoadersStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
		return nil
	}
	if annotations, err := p.extraLoadersAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	}
	for _, field := range s.Fields.List {
		if err := p.parseLoaderStruct(state, filePath, fileContent, fset, field.Names[0].Name, field.Type.(*ast.StructType)); err != nil {
			return err
		}
	}
	return nil
}

func (p *golang_parser) parseBundlesStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
		return nil
	}
	if annotations, err := p.extraBundlesAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	}
	for _, field := range s.Fields.List {
		if err := p.parseBundleStruct(state, filePath, fileContent, fset, field.Names[0].Name, field.Doc, field.Type.(*ast.StructType)); err != nil {
			return err
		}
	}
	return nil
}
