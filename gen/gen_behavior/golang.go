package gen_behavior

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/proj"
	"gopkg.in/yaml.v3"
)

// 注解
type behaviors_annotate struct {
	Behaviors string
	Driver    []string
}

func (p *golang_parser) extraBehaviorsAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*behaviors_annotate, error) {
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
	annotates := &behaviors_annotate{}

	if v, ok := values["behaviors"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s behaviors annotate must string", name))
	} else {
		annotates.Behaviors = str
	}
	if v, ok := values["driver"]; !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s driver annotate not set", name))
	} else if arr, ok := v.([]interface{}); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s driver annotate must string array", name))
	} else {
		for _, str := range arr {
			annotates.Driver = append(annotates.Driver, str.(string))
		}
	}
	return annotates, nil
}

type golang_parser struct {
}

func (p *golang_parser) parse(state *gen_state) error {
	filePathArr := make([]string, 0)
	if err := filepath.WalkDir(path.Join(proj.Dir.DocDir, "behavior"), func(path string, d os.DirEntry, err error) error {
		if d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) == ".go" {
			filePathArr = append(filePathArr, path)
		}
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(filePathArr)
	for _, filePath := range filePathArr {
		if err := p.parseFile(state, filePath); err != nil {
			return err
		}
	}
	return nil
}

func (p *golang_parser) newAstError(fset *token.FileSet, filePath string, pos token.Pos, msg string) error {
	return fmt.Errorf("%s %v %s", filePath, fset.Position(pos), msg)
}

func (p *golang_parser) extraComment(commentGroup *ast.CommentGroup) ([]string, error) {
	results := make([]string, 0)
	if commentGroup == nil {
		return results, nil
	}
	for _, c := range commentGroup.List {
		if !strings.HasPrefix(c.Text, "// @") {
			results = append(results, c.Text)
		}
	}
	return results, nil
}

func (p *golang_parser) parseFile(state *gen_state, filePath string) (err error) {
	log.Info("处理文件", filePath)
	// fileName := path.Base(filePath)
	// dbName := strings.Replace(fileName, ".go", "", 1)

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
		case *ast.GenDecl:
			for _, spec := range x.Specs {
				switch st := spec.(type) {
				case *ast.TypeSpec:
					// 	if s.Names[0].Name == "Driver" {
					// 		if err = p.parseDriver(state, database, filePath, fileContent, fset, s); err != nil {
					// 			return false
					// 		}
					// 	}
					// case *ast.TypeSpec:
					// 	if s.Name.Name == "Behavior" {
					// 		if err = p.parseBehaviors(state, database, filePath, fileContent, fset, x, s, s.Type.(*ast.StructType)); err != nil {
					// 			return false
					// 		}
					// 	}
					// }
					if s, ok := st.Type.(*ast.StructType); ok {
						if err = p.parseBehaviors(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
					}
				}
			}
		}
		return true
	})
	return nil
}

func (p *golang_parser) parseDriver(state *gen_state, database *Database, filePath string, fileContent []byte, fset *token.FileSet, v *ast.ValueSpec) error {
	if values0, ok := v.Values[0].(*ast.CompositeLit); !ok {
		return p.newAstError(fset, filePath, v.Pos(), "driver必须是string数组")
	} else if typ, ok := values0.Type.(*ast.ArrayType); !ok {
		return p.newAstError(fset, filePath, values0.Pos(), "driver必须是string数组")
	} else if elt, ok := typ.Elt.(*ast.Ident); !ok {
		return p.newAstError(fset, filePath, values0.Pos(), "driver必须是string数组")
	} else if elt.Name != "string" {
		return p.newAstError(fset, filePath, values0.Pos(), "driver必须是string数组")
	} else if len(values0.Elts) <= 0 {
		return p.newAstError(fset, filePath, values0.Pos(), "至少一个元素")
	} else {
		if v, ok := values0.Elts[0].(*ast.BasicLit); !ok {
			return p.newAstError(fset, filePath, values0.Pos(), "至少一个元素")
		} else {
			database.Driver = v.Value
			return nil
		}
	}
}

func (p *golang_parser) parseBehaviors(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		return nil
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
	}
	var database *Database
	if annotations, err := p.extraBehaviorsAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	} else {
		dbName := annotations.Behaviors
		if v, ok := state.databasDict[dbName]; !ok {
			database = &Database{
				Module:              proj.Module,
				CollectionArr:       make([]*Collection, 0),
				GenXmlDir:           path.Join(proj.Dir.GenBehaviorDir, dbName),
				GenXmlFilePath:      path.Join(proj.Dir.GenBehaviorDir, dbName, fmt.Sprintf("%s.xml", dbName)),
				SrcGenModelDir:      path.Join(proj.Dir.SrcGenBehaviorDir, dbName),
				SrcGenModelFilePath: path.Join(proj.Dir.SrcGenBehaviorDir, dbName, fmt.Sprintf("%s.gen.go", dbName)),
				SrcGenBinFilePath:   path.Join(proj.Dir.SrcGenBehaviorDir, dbName, "bin", fmt.Sprintf("%s.gen.go", dbName)),
				SrcGenBinDir:        path.Join(proj.Dir.SrcGenBehaviorDir, dbName, "bin"),
				DbName:              dbName,
				DbStructName:        camelString(dbName),
				MongoDaoStructName:  fmt.Sprintf("%sMongoDao", camelString(dbName)),
				DaoInterfaceName:    fmt.Sprintf("%sDao", camelString(dbName)),
			}
			state.databaseArr = append(state.databaseArr, database)
			state.databasDict[dbName] = database
		} else {
			database = v
			database.Driver = annotations.Driver[0]
		}
	}
	for _, f := range s.Fields.List {
		name := f.Names[0].Name
		var coll *Collection
		var err error
		if coll, err = p.parseCollection(state, database, filePath, fileContent, name, fset, f.Type.(*ast.StructType)); err != nil {
			return err
		} else {
			var commentArr []string
			if commentArr, err = p.extraComment(f.Doc); err != nil {
				return err
			}
			for _, comment := range commentArr {
				if strings.HasPrefix(comment, "//") {
					coll.CommentArr = append(coll.CommentArr, strings.Replace(comment, "//", "", 1))
				}
			}
			database.CollectionArr = append(database.CollectionArr, coll)
		}
	}
	return nil
}

func (p *golang_parser) parseCollection(state *gen_state, database *Database, filePath string, fileContent []byte, collName string, fset *token.FileSet, s *ast.StructType) (*Collection, error) {
	coll := &Collection{
		CollName:           collName,
		StructName:         camelString(collName),
		MongoDaoStructName: fmt.Sprintf("%sMongoDao", camelString(collName)),
	}
	for _, f := range s.Fields.List {
		if f.Names[0].Name == "Model" {
			if model, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseStruct(state, coll, filePath, fileContent, fset, collName, model); err != nil {
					return nil, err
				}
			}
		} else if f.Names[0].Name == "Index" {
			if index, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseIndex(state, coll, filePath, fileContent, fset, collName, index); err != nil {
					return nil, err
				}
			}
		}
	}
	return coll, nil
}

func (p *golang_parser) parseStruct(state *gen_state, coll *Collection, filePath string, fileContent []byte, fset *token.FileSet, structName string, s *ast.StructType) error {
	coll.FieldDict = make(map[string]*Field)
	coll.FieldArr = make([]*Field, 0)
	for _, f := range s.Fields.List {
		var tag int
		var err error
		var fieldName string
		var typeStr string
		var tagStr string
		fieldName = f.Names[0].Name
		if f.Tag == nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag not set", structName, fieldName))
		}
		tagStr = f.Tag.Value
		tagStr = strings.Replace(tagStr, "`", "", 2)
		if tag, err = strconv.Atoi(tagStr); err != nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag invalid %s", structName, fieldName, err))
		}
		typeStr = string(fileContent[f.Type.Pos()-1 : f.Type.End()-1])
		field := &Field{
			Coll:      coll,
			Name:      fieldName,
			CamelName: camelString(fieldName),
			Tag:       tag,
		}
		if f.Doc != nil {
			var commentArr []string
			if commentArr, err = p.extraComment(f.Doc); err != nil {
				return err
			}
			for _, comment := range commentArr {
				if strings.HasPrefix(comment, "//") {
					field.CommentArr = append(field.CommentArr, strings.Replace(comment, "//", "", 1))
				}
			}
		}
		if fieldName == "id" {
			field.Name = "_id"
			field.CamelName = "Id"
		} else {
			field.Name = fieldName
			field.CamelName = camelString(fieldName)
		}
		field.TypeName = typeStr
		if typeValue, ok := type_name_dict[typeStr]; ok {
			field.Type = typeValue
			field.GoTypeName = go_type_name_dict[field.Type]
		} else {
			field.Type = field_type_struct
			field.GoTypeName = typeStr
		}
		coll.FieldDict[fieldName] = field
		coll.FieldArr = append(coll.FieldArr, field)
	}

	for _, f := range coll.FieldArr {
		if f.Name == "log_time" && f.Type == field_type_int64 {
			f.Coll.HasLogTimeField = true
		}
		if f.Name == "create_time" && f.Type == field_type_int64 {
			f.Coll.HasCreateTimeField = true
		}
	}
	sort.Sort(sort_field_by_name(coll.FieldArr))
	return nil
}

func (p *golang_parser) parseIndex(state *gen_state, coll *Collection, filePath string, fileContent []byte, fset *token.FileSet, structName string, s *ast.StructType) error {
	coll.IndexDict = make(map[string]*Index)
	coll.IndexArr = make([]*Index, 0)
	for _, f := range s.Fields.List {
		var indexName string
		var tagStr string
		var tag int
		var err error
		indexName = f.Names[0].Name
		if f.Tag == nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag not set", structName, indexName))
		}
		tagStr = f.Tag.Value
		tagStr = strings.Replace(tagStr, "`", "", 2)
		if tag, err = strconv.Atoi(tagStr); err != nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag invalid %s", structName, indexName, err))
		}
		index := &Index{
			Name: indexName,
			Tag:  tag,
		}
		// ast.Print(fset, s)
		if st, ok := f.Type.(*ast.StructType); ok {
			var fullName string
			index.KeyDict = make(map[string]*Key)
			for _, f1 := range st.Fields.List {
				k := f1.Names[0].Name
				if f1.Tag == nil {
					return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s %s tag not set", structName, indexName, k))
				}
				v := f1.Tag.Value
				v = strings.Replace(v, "`", "", 2)
				if _, err = strconv.Atoi(v); err != nil {
					return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s %s tag invalid %s", structName, indexName, k, err))
				}
				indexKey := &Key{Key: k, Value: v}
				index.KeyDict[k] = indexKey
				index.KeyArr = append(index.KeyArr, indexKey)
				if len(index.KeyDict) == 1 {
					fullName = fmt.Sprintf("%s_%v", k, v)
				} else {
					fullName = fmt.Sprintf("%s_%s_%v", fullName, k, v)
				}
			}
			index.FullName = fullName
		}
		coll.IndexDict[indexName] = index
		coll.IndexArr = append(coll.IndexArr, index)
	}
	return nil
}
