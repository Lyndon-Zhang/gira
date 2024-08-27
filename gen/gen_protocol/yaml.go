package gen_protocol

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/proj"
	yaml "gopkg.in/yaml.v3"
)

type sort_field_by_name []*Field

func (self sort_field_by_name) Len() int           { return len(self) }
func (self sort_field_by_name) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }
func (self sort_field_by_name) Less(i, j int) bool { return self[i].Tag < self[j].Tag }

type yaml_parser struct {
}

func (p *yaml_parser) parse(state *gen_state) error {
	nameArr := make([]string, 0)
	if d, err := os.Open(proj.Dir.DocProtocolDir); err != nil {
		return err
	} else {
		if files, err := d.ReadDir(0); err != nil {
			return err
		} else {
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
					log.Info(file.Name())
					nameArr = append(nameArr, strings.Replace(file.Name(), ".yaml", "", 1))
				}
			}
		}
		d.Close()
	}

	for _, name := range nameArr {
		dir := filepath.Join(proj.Dir.DocProtocolDir, name)
		protocolFilePathArr := make([]string, 0)
		protocolFilePath := filepath.Join(proj.Dir.DocProtocolDir, fmt.Sprintf("%s.yaml", name))
		protocolFilePathArr = append(protocolFilePathArr, protocolFilePath)
		if _, err := os.Stat(dir); err == nil {
			filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				if filepath.Ext(path) == ".yaml" {
					protocolFilePathArr = append(protocolFilePathArr, path)
				}
				return nil
			})
		}
		proto := &protocol{
			Module:     name,
			PacketDict: make(map[string]*Packet),
			PacketArr:  make([]*Packet, 0),
		}
		if err := p.parseFile(state, proto, protocolFilePath, protocolFilePathArr); err != nil {
			log.Info(err)
			return err
		} else {
			state.protocols = append(state.protocols, proto)
		}
	}
	return nil
}

func (p *yaml_parser) parseFile(genState *gen_state, proto *protocol, mainFilePath string, filePathArr []string) error {
	// 主文件的注释
	f, err := os.Open(mainFilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)

	var results []string
	// 按行处理txt
	for {
		b, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		line := string(b)
		if strings.HasPrefix(line, "### ") {
			line = strings.Replace(line, "### ", "", 1)
			results = append(results, line)
		}
	}
	proto.Header = strings.Join(results, "\n")

	// 提取 request, response, push, notify
	spaceRegexp := regexp.MustCompile("[^\\s]+")
	for _, filePath := range filePathArr {
		log.Info("处理文件", filePath)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		result := make(map[string]interface{})
		if err := yaml.Unmarshal(data, result); err != nil {
			return err
		}
		for k, v := range result {
			var dict map[string]interface{}
			var ok bool
			if dict, ok = v.(map[string]interface{}); !ok {
				return fmt.Errorf("%s invalid111", k)
			}
			args := spaceRegexp.FindAllString(k, -1)
			name := args[0]
			packet := &Packet{
				Name:       name,
				FullName:   name,
				StructName: camelString(name),
				Message:    &Message{},
				Request:    &Message{},
				Response:   &Message{},
				Push:       &Message{},
				Notify:     &Message{},
			}
			if c, ok := dict["comment"]; ok {
				packet.CommentArr = append(packet.CommentArr, c.(string))
			}
			if len(args) == 1 {
				log.Info("读取协议", name)
				packet.Type = message_type_struct
				if _, ok := dict["struct"]; ok {
					if _, ok := dict["struct"].(map[string]interface{}); ok {
						if err := p.parseMessage(packet.Message, dict["struct"].(map[string]interface{})); err != nil {
							return err
						}
					}
				}
				packet.Message.StructName = fmt.Sprintf("%s", camelString(name))
				proto.PacketDict[name] = packet
			} else if len(args) == 2 {
				if message_id, err := strconv.Atoi(args[1]); err != nil {
					return err
				} else {
					packet.MessageId = message_id
				}
				log.Info("读取协议", name)
				hasRequest := false
				hasResponse := false
				if v, ok := dict["request"]; ok {
					if v == nil {
					} else if _, ok := v.(map[string]interface{}); !ok {
						return fmt.Errorf("%s invalid66 %#v", k, v)
					}
					hasRequest = true
				}
				if v, ok := dict["response"]; ok {
					if v == nil {
					} else if _, ok := v.(map[string]interface{}); !ok {
						return fmt.Errorf("%s invalid77", k)
					}
					hasResponse = true
				}
				if hasRequest && hasResponse {
					packet.Type = message_type_request
				} else if hasRequest {
					packet.Type = message_type_push
				} else if hasResponse {
					packet.Type = message_type_notify
				} else {
					return fmt.Errorf("%s request and response not found", name)
				}
				if packet.Type == message_type_request {
					if v, ok := dict["request"].(map[string]interface{}); ok {
						if err := p.parseMessage(packet.Request, v); err != nil {
							return err
						}
					}
					if v, ok := dict["response"].(map[string]interface{}); ok {
						if err := p.parseMessage(packet.Response, v); err != nil {
							return err
						}
					}
					packet.Request.StructName = fmt.Sprintf("%sRequest", camelString(name))
					packet.Response.StructName = fmt.Sprintf("%sResponse", camelString(name))
				} else if packet.Type == message_type_notify {
					if v, ok := dict["response"].(map[string]interface{}); ok {
						if err := p.parseMessage(packet.Notify, v); err != nil {
							return err
						}
					}
					packet.Notify.StructName = fmt.Sprintf("%sNotify", camelString(name))
				} else if packet.Type == message_type_push {
					if v, ok := dict["request"].(map[string]interface{}); ok {
						if err := p.parseMessage(packet.Push, v); err != nil {
							return err
						}
					}
					packet.Push.StructName = fmt.Sprintf("%sPush", camelString(name))
				}
				proto.PacketDict[name] = packet
			}
		}
	}
	// 对packetarr排序
	for _, filePath := range filePathArr {
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()
		reader := bufio.NewReader(f)
		// 按行处理txt
		for {
			b, _, err := reader.ReadLine()
			if err == io.EOF {
				break
			}
			line := string(b)
			if len(line) > 0 {
				if !unicode.IsSpace([]rune(line)[0]) && !strings.HasPrefix(line, "#") {
					args := spaceRegexp.FindAllString(line, -1)
					name := args[0]
					name = strings.Replace(name, ":", "", 1)
					if p, ok := proto.PacketDict[name]; ok {
						proto.PacketArr = append(proto.PacketArr, p)
					}
				}
			}
		}
	}
	return nil
}

func (p *yaml_parser) parseMessage(m *Message, attrs map[string]interface{}) error {
	m.FieldDict = make(map[string]*Field)
	m.FieldArr = make([]*Field, 0)
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
			return fmt.Errorf("%+v invalid11 %s", v, valueStr)
		}
		args := equalRegexp.FindAllString(valueStr, -1)
		if len(args) != 2 {
			return fmt.Errorf("%s invalid22", valueStr)
		}
		tagStr = strings.TrimSpace(args[1])
		if tag, err = strconv.Atoi(tagStr); err != nil {
			return err
		}
		args = spaceRegexp.FindAllString(args[0], -1)
		if len(args) != 2 {
			return fmt.Errorf("%s invalid33", valueStr)
		}
		typeStr = args[1]
		isArray := false
		fieldName = args[0]
		if strings.HasPrefix(typeStr, "[]") {
			isArray = true
			typeStr = strings.Replace(typeStr, "[]", "", 1)
			typeStr = strings.TrimSpace(typeStr)
		}

		field := &Field{
			Name:      fieldName,
			CamelName: camelString(fieldName),
			Tag:       tag,
			IsArray:   isArray,
			Message:   m,
		}
		//args = commaRegexp.FindAllString(v.(string), -1)
		//if len(args) <= 0 {
		//	fmt.Println(args)
		//	return fmt.Errorf("field args wrong %s %s %s", "name", k, v)
		//}
		field.TypeName = typeStr
		if typeValue, ok := type_name_dict[typeStr]; ok {
			field.Type = typeValue
		} else {
			field.Type = field_type_struct
		}
		if typeName, ok := sp_type_name_dict[field.Type]; ok {
			field.SpTypeName = typeName
		} else {
			field.SpTypeName = typeStr
		}
		if typeName, ok := sp_type_name_dict[field.Type]; ok {
			field.SpWireTypeName = typeName
		} else {
			field.SpWireTypeName = "struct"
		}
		if typeName, ok := go_type_name_dict[field.Type]; ok {
			field.GoTypeName = typeName
		} else {
			field.GoTypeName = fmt.Sprintf("*%s", typeStr)
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
		m.FieldDict[fieldName] = field
		m.FieldArr = append(m.FieldArr, field)
	}
	sort.Sort(sort_field_by_name(m.FieldArr))
	return nil
}
