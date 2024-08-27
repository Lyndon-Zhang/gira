package gen_model

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

type models_annotate struct {
	Models string
	Driver []string
	Import []string
}

func (p *golang_parser) extraModelsAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*models_annotate, error) {
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
	annotates := &models_annotate{}

	if v, ok := values["models"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s models annotate must string", name))
	} else {
		annotates.Models = str
	}
	if v, ok := values["driver"]; !ok {
		log.Println(values)
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s driver annotate not set", name))
	} else if arr, ok := v.([]interface{}); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s driver annotate must string array", name))
	} else {
		for _, str := range arr {
			annotates.Driver = append(annotates.Driver, str.(string))
		}
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

type index_annotate struct {
	Unique bool
}

func (p *golang_parser) extraIndexAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, modelName string, indexName string, comments []*ast.Comment) (*index_annotate, error) {
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
	annotates := &index_annotate{}
	if v, ok := values["unique"]; ok {
		if b, ok := v.(bool); !ok {
			return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s %s unique annotate must bool", modelName, indexName))
		} else {
			annotates.Unique = b
		}
	}
	return annotates, nil
}

type golang_parser struct {
}

func (p *golang_parser) newAstError(fset *token.FileSet, filePath string, pos token.Pos, msg string) error {
	return fmt.Errorf("%s %v %s", filePath, fset.Position(pos), msg)
}

func (p *golang_parser) parse(state *gen_state) error {
	filePathArr := make([]string, 0)
	filepath.WalkDir(proj.Dir.DocModelDir, func(path string, d os.DirEntry, err error) error {
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
	})
	sort.Strings(filePathArr)
	for _, filePath := range filePathArr {
		if err := p.parseFile(state, filePath); err != nil {
			return err
		}
	}
	return nil
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
				// case *ast.ValueSpec:
				// 	if s.Names[0].Name == "Driver" {
				// 		if err = p.parseDriver(state, database, filePath, fileContent, fset, s); err != nil {
				// 			return false
				// 		}
				// 	} else if s.Names[0].Name == "Import" {
				// 		if err = p.parseImport(state, database, filePath, fileContent, fset, s); err != nil {
				// 			return false
				// 		}
				// 	}
				case *ast.TypeSpec:
					// var comment string
					// if x.Doc != nil {
					// 	if comments, err := gen.ExtraComment(x.Doc); err == nil {
					// 		comment = strings.Join(comments, ",")
					// 	}
					// }
					// if typ, ok := s.Type.(*ast.StructType); ok {
					// 	var coll *Collection
					// 	if coll, err = p.parseCollection(state, database, fileContent, s.Name.Name, fset, typ); err != nil {
					// 		return false
					// 	} else {
					// 		coll.Comment = comment
					// 		database.CollectionArr = append(database.CollectionArr, coll)
					// 	}
					// }
					// if s.Name.Name == "Models" {
					if s, ok := st.Type.(*ast.StructType); ok {
						if err = p.parseModelsStruct(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
					}
					// }
				}
			}
		}
		return true
	})
	return
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

func (p *golang_parser) parseModelsStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
		return nil
	}
	var database *Database
	if annotations, err := p.extraModelsAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	} else {
		dbName := annotations.Models
		if v, ok := state.databasDict[dbName]; !ok {
			database = &Database{
				Module:              proj.Module,
				CollectionArr:       make([]*Collection, 0),
				GenBinFilePath:      path.Join(proj.Dir.SrcGenModelDir, dbName, "bin", fmt.Sprintf("%s.gen.go", dbName)),
				GenBinDir:           path.Join(proj.Dir.SrcGenModelDir, dbName, "bin"),
				GenModelDir:         path.Join(proj.Dir.SrcGenModelDir, dbName),
				GenModelFilePath:    path.Join(proj.Dir.SrcGenModelDir, dbName, fmt.Sprintf("%s.gen.go", dbName)),
				GenProtobufFilePath: path.Join(proj.Dir.GenModelDir, dbName, fmt.Sprintf("%s.gen.proto", dbName)),
				DbName:              dbName,
				DbStructName:        camelString(dbName),
				MongoDaoStructName:  fmt.Sprintf("%sMongoDao", camelString(dbName)),
				RedisDaoStructName:  fmt.Sprintf("%sRedisDao", camelString(dbName)),
				DaoInterfaceName:    fmt.Sprintf("%sDao", camelString(dbName)),
				Driver:              annotations.Driver[0],
				ImportArr:           annotations.Import,
			}
			state.databaseArr = append(state.databaseArr, database)
			state.databasDict[dbName] = database
		} else {
			database = v
			database.ImportArr = append(database.ImportArr, annotations.Import...)
		}
	}
	for _, f := range s.Fields.List {
		name := f.Names[0].Name
		var err error
		var coll *Collection
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

// func (p *golang_parser) parseDriver(state *gen_state, database *Database, filePath string, fileContent []byte, fset *token.FileSet, s *ast.ValueSpec) error {
// 	if values0, ok := s.Values[0].(*ast.CompositeLit); !ok {
// 		return p.reportError(fset, filePath, s.Pos(), "driver必须是string数组")
// 	} else if typ, ok := values0.Type.(*ast.ArrayType); !ok {
// 		return p.reportError(fset, filePath, values0.Pos(), "driver必须是string数组")
// 	} else if elt, ok := typ.Elt.(*ast.Ident); !ok {
// 		return p.reportError(fset, filePath, values0.Pos(), "driver必须是string数组")
// 	} else if elt.Name != "string" {
// 		return p.reportError(fset, filePath, values0.Pos(), "driver必须是string数组")
// 	} else if len(values0.Elts) <= 0 {
// 		return p.reportError(fset, filePath, values0.Pos(), "至少一个元素")
// 	} else {
// 		if v, ok := values0.Elts[0].(*ast.BasicLit); !ok {
// 			return p.reportError(fset, filePath, values0.Pos(), "至少一个元素")
// 		} else {
// 			database.Driver = v.Value[1 : len(v.Value)-1]
// 			return nil
// 		}
// 	}
// }

// func (p *golang_parser) parseImport(state *gen_state, database *Database, filePath string, fileContent []byte, fset *token.FileSet, s *ast.ValueSpec) error {
// 	if values0, ok := s.Values[0].(*ast.CompositeLit); !ok {
// 		return p.reportError(fset, filePath, s.Pos(), "driver必须是string数组")
// 	} else if typ, ok := values0.Type.(*ast.ArrayType); !ok {
// 		return p.reportError(fset, filePath, values0.Pos(), "driver必须是string数组")
// 	} else if elt, ok := typ.Elt.(*ast.Ident); !ok {
// 		return p.reportError(fset, filePath, values0.Pos(), "driver必须是string数组")
// 	} else if elt.Name != "string" {
// 		return p.reportError(fset, filePath, values0.Pos(), "driver必须是string数组")
// 	} else if len(values0.Elts) <= 0 {
// 		return p.reportError(fset, filePath, values0.Pos(), "至少一个元素")
// 	} else {
// 		if v, ok := values0.Elts[0].(*ast.BasicLit); !ok {
// 			return p.reportError(fset, filePath, values0.Pos(), "至少一个元素")
// 		} else {
// 			database.ImportArr = append(database.ImportArr, v.Value[1:len(v.Value)-1])
// 			return nil
// 		}
// 	}
// }

func (p *golang_parser) parseCollection(state *gen_state, database *Database, filePath string, fileContent []byte, collName string, fset *token.FileSet, s *ast.StructType) (*Collection, error) {
	coll := &Collection{
		CollName:           collName,
		StructName:         camelString(collName),
		PbStructName:       fmt.Sprintf("%sPb", camelString(collName)),
		ArrStructName:      fmt.Sprintf("%sArr", camelString(collName)),
		DataStructName:     fmt.Sprintf("%sData", camelString(collName)),
		MongoDaoStructName: fmt.Sprintf("%sMongoDao", camelString(collName)),
		RedisDaoStructName: fmt.Sprintf("%sRedisDao", camelString(collName)),
	}
	for _, f := range s.Fields.List {
		if f.Names[0].Name == "Model" {
			if model, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseModelStruct(state, coll, filePath, fileContent, fset, collName, model); err != nil {
					return nil, err
				}
			}
		} else if f.Names[0].Name == "Index" {
			if index, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseIndexStruct(state, coll, filePath, fileContent, fset, collName, index); err != nil {
					return nil, err
				}
			}
		} else if f.Names[0].Name == "Dao" {
			if dao, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseDaoStruct(state, coll, fileContent, fset, dao); err != nil {
					return nil, err
				}
			}
		} else if f.Names[0].Name == "Capped" {
			capped := strings.Replace(f.Tag.Value, "`", "", 2)
			if v, err := strconv.Atoi(capped); err != nil {
				return nil, err
			} else {
				coll.Capped = int64(v)
			}
		}
	}
	if coll.Derive == "userarr" {
		if len(coll.KeyArr) != 2 {
			return nil, fmt.Errorf("collection %s derive userarr need 2 key", coll.CollName)
		}
		primaryKey := coll.KeyArr[0]
		secondaryKey := coll.KeyArr[1]
		coll.PrimaryKey = primaryKey
		coll.CamelPrimaryKey = camelString(primaryKey)
		coll.CapCamelPrimaryKey = capLowerString(coll.CamelPrimaryKey)
		coll.SecondaryKey = secondaryKey
		coll.CamelSecondaryKey = camelString(secondaryKey)
		coll.CapCamelSecondaryKey = capLowerString(coll.CamelSecondaryKey)
		if field, ok := coll.FieldDict[primaryKey]; ok {
			coll.PrimaryKeyField = field
			field.IsPrimaryKey = true
		} else {
			return nil, fmt.Errorf("collection %s derive userarr, but primary key %s not found", coll.CollName, primaryKey)
		}
		if field, ok := coll.FieldDict[secondaryKey]; ok {
			coll.SecondaryKeyField = field
			field.IsSecondaryKey = true
		} else {
			return nil, fmt.Errorf("collection %s derive userarr, but secondary key %s not found", coll.CollName, secondaryKey)
		}
	}
	return coll, nil
}

func (p *golang_parser) parseDaoStruct(state *gen_state, coll *Collection, fileContent []byte, fset *token.FileSet, s *ast.StructType) error {
	for _, f := range s.Fields.List {
		if f.Names[0].Name == "User" {
			coll.Derive = "user"
			if keys, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseDeriveKeys(state, coll, fileContent, fset, keys); err != nil {
					return err
				}
			}
		} else if f.Names[0].Name == "UserArr" {
			coll.Derive = "userarr"
			if keys, ok := f.Type.(*ast.StructType); ok {
				if err := p.parseDeriveKeys(state, coll, fileContent, fset, keys); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *golang_parser) parseDeriveKeys(state *gen_state, coll *Collection, fileContent []byte, fset *token.FileSet, s *ast.StructType) error {
	for _, f := range s.Fields.List {
		// log.Println(f.Names[0].Name)
		coll.KeyArr = append(coll.KeyArr, f.Names[0].Name)
	}
	return nil
}

func (p *golang_parser) parseModelStruct(state *gen_state, coll *Collection, filePath string, fileContent []byte, fset *token.FileSet, structName string, s *ast.StructType) error {
	coll.FieldDict = make(map[string]*Field)
	coll.FieldArr = make([]*Field, 0)
	for _, f := range s.Fields.List {
		var tag int
		var err error
		var fieldName string
		var typeStr string
		var tagStr string
		if f.Tag == nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag not set", structName, fieldName))
		}
		tagStr = f.Tag.Value
		tagStr = strings.Replace(tagStr, "`", "", 2)
		if tag, err = strconv.Atoi(tagStr); err != nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag invalid %s", structName, fieldName, err))
		}
		// ast.Print(fset, s)
		fieldName = f.Names[0].Name
		// WARN: 兼容type
		if fieldName == "Type" {
			fieldName = "type"
		}
		typeStr = string(fileContent[f.Type.Pos()-1 : f.Type.End()-1])
		field := &Field{
			Coll:      coll,
			Name:      fieldName,
			CamelName: camelString(fieldName),
			Array:     false,
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
			field.ProtobufTypeName = protobuf_type_name_dict[field.Type]
		} else {
			field.Type = field_type_struct
			field.GoTypeName = typeStr
			field.ProtobufTypeName = "bytes"
		}
		coll.FieldDict[fieldName] = field
		coll.FieldArr = append(coll.FieldArr, field)
	}
	sort.Sort(sort_field_by_name(coll.FieldArr))
	return nil
}

func (p *golang_parser) parseIndexStruct(state *gen_state, coll *Collection, filePath string, fileContent []byte, fset *token.FileSet, structName string, s *ast.StructType) error {
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
			Name:     indexName,
			Tag:      tag,
			FullName: indexName,
		}
		if st, ok := f.Type.(*ast.StructType); ok {
			var fullName string
			index.KeyDict = make(map[string]*Key)
			if f.Doc != nil {
				if annotations, err := p.extraIndexAnnotate(fset, filePath, st, structName, indexName, f.Doc.List); err != nil {
					return err
				} else {
					index.Unique = annotations.Unique
				}
			}
			for _, f1 := range st.Fields.List {
				k := f1.Names[0].Name
				if f1.Tag == nil {
					return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s %s tag not set", structName, indexName, k))
				}
				v := f1.Tag.Value
				v = strings.Replace(v, "`", "", 2)
				indexKey := &Key{Key: k, Value: v}
				if i, err := strconv.Atoi(v); err == nil {
					indexKey.Value = i
				}
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
	sort.Sort(sort_index_by_name(coll.IndexArr))
	return nil
}
