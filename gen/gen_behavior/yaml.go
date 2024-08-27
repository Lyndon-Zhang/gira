package gen_behavior

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/proj"
	"gopkg.in/yaml.v3"
)

type yaml_parser struct {
}

type sort_collection_by_name []*Collection

func (self sort_collection_by_name) Len() int           { return len(self) }
func (self sort_collection_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_collection_by_name) Less(i, j int) bool { return self[i].CollName < self[j].CollName }

type sort_field_by_name []*Field

func (self sort_field_by_name) Len() int           { return len(self) }
func (self sort_field_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_field_by_name) Less(i, j int) bool { return self[i].Tag < self[j].Tag }

type sort_index_by_name []*Index

func (self sort_index_by_name) Len() int           { return len(self) }
func (self sort_index_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_index_by_name) Less(i, j int) bool { return self[i].Tag < self[j].Tag }

func (p *yaml_parser) parse(state *gen_state) error {
	filePathArr := make([]string, 0)
	if err := filepath.WalkDir(proj.Dir.DocBehaviorDir, func(path string, d os.DirEntry, err error) error {
		if d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) == ".yaml" {
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

func (p *yaml_parser) parseFile(state *gen_state, filePath string) error {
	log.Info("处理文件", filePath)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileName := path.Base(filePath)
	dbName := strings.Replace(fileName, ".yaml", "", 1)
	database := &Database{
		Module:              proj.Module,
		CollectionArr:       make([]*Collection, 0),
		SrcGenModelDir:      path.Join(proj.Dir.SrcGenBehaviorDir, dbName),
		SrcGenModelFilePath: path.Join(proj.Dir.SrcGenBehaviorDir, dbName, fmt.Sprintf("%s.gen.go", dbName)),
		SrcGenBinFilePath:   path.Join(proj.Dir.SrcGenBehaviorDir, dbName, "bin", fmt.Sprintf("%s.gen.go", dbName)),
		SrcGenBinDir:        path.Join(proj.Dir.SrcGenBehaviorDir, dbName, "bin"),
		DbName:              dbName,
		DbStructName:        camelString(dbName),
		MongoDaoStructName:  fmt.Sprintf("%sMongoDao", camelString(dbName)),
		DaoInterfaceName:    fmt.Sprintf("%Dao", camelString(dbName)),
	}
	result := make(map[string]interface{})
	if err := yaml.Unmarshal(data, result); err != nil {
		return err
	}
	for k, v := range result {
		if k == "$driver" {
			database.Driver = v.(string)
		} else {
			collName := k
			coll := &Collection{
				CollName:           collName,
				StructName:         camelString(collName),
				MongoDaoStructName: fmt.Sprintf("%sMongoDao", camelString(collName)),
			}
			if err := p.collUnmarshal(state, coll, v); err != nil {
				return err
			}
			database.CollectionArr = append(database.CollectionArr, coll)
		}
	}
	sort.Sort(sort_collection_by_name(database.CollectionArr))
	state.databaseArr = append(state.databaseArr, database)
	return nil
}

func (p *yaml_parser) collUnmarshal(genState *gen_state, coll *Collection, v interface{}) error {
	var derive string
	row := v.(map[string]interface{})
	if _, ok := row["struct"]; !ok {
		return fmt.Errorf("collection %s struct part not found", coll.CollName)
	}
	structPart := row["struct"]
	if _, ok := structPart.(map[string]interface{}); !ok {
		return fmt.Errorf("collection %s struct part not map", coll.CollName)
	}
	coll.Derive = derive
	if err := p.parseStruct(coll, row["struct"].(map[string]interface{})); err != nil {
		return err
	}
	// 解析index
	if v, ok := row["index"]; !ok {
	} else if v2, ok := v.(map[string]interface{}); !ok {
	} else if err := p.parseIndex(coll, v2); err != nil {
		return err
	}
	if v, ok := row["comment"]; ok {
		coll.CommentArr = append(coll.CommentArr, v.(string))
	}
	return nil
}

func (p *yaml_parser) parseStruct(coll *Collection, attrs map[string]interface{}) error {
	coll.FieldDict = make(map[string]*Field)
	coll.FieldArr = make([]*Field, 0)
	spaceRegexp := regexp.MustCompile("[^\\s]+")
	equalRegexp := regexp.MustCompile("[^=]+")

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
		typeStr = args[1]
		fieldName = args[0]
		field := &Field{
			Coll:      coll,
			Name:      fieldName,
			CamelName: camelString(fieldName),
			Tag:       tag,
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
		// fmt.Println(optionArr)
		for _, option := range optionArr {
			optionDict := option.(map[string]interface{})
			if defaultVal, ok := optionDict["default"]; ok {
				field.Default = defaultVal
			}
			if comment, ok := optionDict["comment"]; ok {
				field.CommentArr = append(field.CommentArr, comment.(string))
			}
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

func (p *yaml_parser) parseIndex(coll *Collection, attrs map[string]interface{}) error {
	coll.IndexDict = make(map[string]*Index)
	coll.IndexArr = make([]*Index, 0)
	equalRegexp := regexp.MustCompile("[^=]+")

	for valueStr, v := range attrs {
		var indexName string
		var tagStr string
		var tag int
		var err error
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
		indexName = args[0]
		index := &Index{
			Name: indexName,
			Tag:  tag,
		}
		for _, option := range optionArr {
			optionDict := option.(map[string]interface{})
			if keyArr, ok := optionDict["key"]; ok {
				var fullName string
				index.KeyDict = make(map[string]*Key)
				if keyArr2, ok := keyArr.([]interface{}); ok {
					for _, keyObject := range keyArr2 {
						if keyObject2, ok := keyObject.(map[string]interface{}); ok {
							for k, v := range keyObject2 {
								indexKey := &Key{Key: k, Value: v}
								index.KeyDict[k] = indexKey
								index.KeyArr = append(index.KeyArr, indexKey)
								if len(index.KeyDict) == 1 {
									fullName = fmt.Sprintf("%s_%v", k, v)
								} else {
									fullName = fmt.Sprintf("%s_%s_%v", fullName, k, v)
								}
							}
						}
					}
				}
				index.FullName = fullName
			}
		}
		// log.Println(index.KeyDict)
		// log.Println(index.FullName)
		coll.IndexDict[indexName] = index
		coll.IndexArr = append(coll.IndexArr, index)
	}
	sort.Sort(sort_index_by_name(coll.IndexArr))
	return nil
}
