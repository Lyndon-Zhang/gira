package gen_model

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

type sort_field_by_name []*Field

func (self sort_field_by_name) Len() int           { return len(self) }
func (self sort_field_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_field_by_name) Less(i, j int) bool { return self[i].Tag < self[j].Tag }

type sort_index_key_by_name []*Key

func (self sort_index_key_by_name) Len() int           { return len(self) }
func (self sort_index_key_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_index_key_by_name) Less(i, j int) bool { return self[i].Key < self[j].Key }

type sort_index_by_name []*Index

func (self sort_index_by_name) Len() int           { return len(self) }
func (self sort_index_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_index_by_name) Less(i, j int) bool { return self[i].Tag < self[j].Tag }

type sort_collection_by_name []*Collection

func (self sort_collection_by_name) Len() int           { return len(self) }
func (self sort_collection_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_collection_by_name) Less(i, j int) bool { return self[i].CollName < self[j].CollName }

type yaml_parser struct {
}

func (p *yaml_parser) parse(state *gen_state) error {
	filePathArr := make([]string, 0)
	filepath.WalkDir(proj.Dir.DocModelDir, func(path string, d os.DirEntry, err error) error {
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
	})
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
	}
	result := make(map[string]interface{})
	if err := yaml.Unmarshal(data, result); err != nil {
		return err
	}
	for k, v := range result {
		if k == "$driver" {
			database.Driver = v.(string)
		} else if k == "$import" {
			for _, v := range v.([]interface{}) {
				database.ImportArr = append(database.ImportArr, v.(string))
			}
		} else {
			collName := k
			coll := &Collection{
				CollName:           collName,
				StructName:         camelString(collName),
				PbStructName:       fmt.Sprintf("%sPb", camelString(collName)),
				ArrStructName:      fmt.Sprintf("%sArr", camelString(collName)),
				DataStructName:     fmt.Sprintf("%sData", camelString(collName)),
				MongoDaoStructName: fmt.Sprintf("%sMongoDao", camelString(collName)),
				RedisDaoStructName: fmt.Sprintf("%sRedisDao", camelString(collName)),
			}
			if err := p.Unmarshal(state, coll, v); err != nil {
				return err
			}
			database.CollectionArr = append(database.CollectionArr, coll)
		}
	}
	sort.Sort(sort_collection_by_name(database.CollectionArr))
	state.databaseArr = append(state.databaseArr, database)
	return nil
}

func (p *yaml_parser) Unmarshal(genState *gen_state, coll *Collection, v interface{}) error {
	var derive string
	keyArr := make([]string, 0)
	row := v.(map[string]interface{})
	if v, ok := row["derive"]; ok {
		derive = v.(string)
	}
	// } else {
	// 	return fmt.Errorf("collection %s derive part not found", coll.CollName)
	// }
	if _, ok := row["struct"]; !ok {
		return fmt.Errorf("collection %s struct part not found", coll.CollName)
	}
	structPart := row["struct"]
	if _, ok := structPart.(map[string]interface{}); !ok {
		return fmt.Errorf("collection %s struct part not map", coll.CollName)
	}
	if _, ok := row["key"]; ok {
		keyPart := row["key"]
		if _, ok := keyPart.([]interface{}); !ok {
			return fmt.Errorf("collection %s key part not array", coll.CollName)
		}
		for _, v := range keyPart.([]interface{}) {
			keyArr = append(keyArr, v.(string))
		}
	}
	// } else {
	// 	return fmt.Errorf("collection %s key part not found", coll.CollName)

	// }
	coll.Derive = derive
	coll.KeyArr = keyArr
	if err := p.parseStruct(coll, row["struct"].(map[string]interface{})); err != nil {
		return err
	}
	// 解析index
	if v, ok := row["index"]; !ok {
	} else if v2, ok := v.(map[string]interface{}); !ok {
	} else if err := p.parseIndex(coll, v2); err != nil {
		return err
	}
	if coll.Derive == "userarr" {
		if len(keyArr) != 2 {
			return fmt.Errorf("collection %s derive userarr need 2 key", coll.CollName)
		}
		primaryKey := keyArr[0]
		secondaryKey := keyArr[1]
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
			return fmt.Errorf("collection %s derive userarr, but primary key %s not found", coll.CollName, primaryKey)
		}
		if field, ok := coll.FieldDict[secondaryKey]; ok {
			coll.SecondaryKeyField = field
			field.IsSecondaryKey = true
		} else {
			return fmt.Errorf("collection %s derive userarr, but secondary key %s not found", coll.CollName, secondaryKey)
		}
	}
	if v, ok := row["capped"]; ok {
		coll.Capped = int64(v.(int))
	}
	return nil
}

func (p *yaml_parser) parseStruct(descriptor *Collection, attrs map[string]interface{}) error {
	descriptor.FieldDict = make(map[string]*Field)
	descriptor.FieldArr = make([]*Field, 0)
	//commaRegexp := regexp.MustCompile("[^,]+")
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
			Coll:      descriptor,
			Name:      fieldName,
			CamelName: camelString(fieldName),
			Array:     false,
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
			field.ProtobufTypeName = protobuf_type_name_dict[field.Type]
		} else {
			field.Type = field_type_struct
			field.GoTypeName = typeStr
			field.ProtobufTypeName = "bytes"
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

		descriptor.FieldDict[fieldName] = field
		descriptor.FieldArr = append(descriptor.FieldArr, field)
	}
	sort.Sort(sort_field_by_name(descriptor.FieldArr))
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
		indexName = strings.TrimSpace(indexName)
		index := &Index{
			Name:     indexName,
			Tag:      tag,
			FullName: indexName,
		}
		for _, option := range optionArr {
			optionDict := option.(map[string]interface{})
			if keyArr, ok := optionDict["key"]; ok {
				// var fullName string
				index.KeyDict = make(map[string]*Key)
				if keyArr2, ok := keyArr.([]interface{}); ok {
					for _, keyObject := range keyArr2 {
						if keyObject2, ok := keyObject.(map[string]interface{}); ok {
							for k, v := range keyObject2 {
								indexKey := &Key{Key: k, Value: v}
								index.KeyDict[k] = indexKey
								index.KeyArr = append(index.KeyArr, indexKey)
							}
						}
					}
				}
				// sort.Sort(SortIndexKeyByName(index.KeyArr))
				// for _, v := range index.KeyArr {
				// 	if len(index.KeyArr) == 1 {
				// 		log.Println("qqqqqqqqqqqqqqq")
				// 		fullName = fmt.Sprintf("%s_%v", v.Key, v.Value)
				// 	} else {
				// 		fullName = fmt.Sprintf("%s_%s_%v", fullName, v.Key, v.Value)
				// 	}
				// }
				// index.FullName = fullName
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
