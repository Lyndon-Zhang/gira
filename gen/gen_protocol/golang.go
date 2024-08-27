package gen_protocol

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/proj"
)

type golang_parser struct {
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

func (p *golang_parser) parse(state *gen_state) error {
	nameArr := make([]string, 0)
	if d, err := os.Open(proj.Dir.DocProtocolDir); err != nil {
		return err
	} else {
		if files, err := d.ReadDir(0); err != nil {
			return err
		} else {
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".go" {
					log.Info(file.Name())
					nameArr = append(nameArr, strings.Replace(file.Name(), ".go", "", 1))
				}
			}
		}
		d.Close()
	}

	for _, name := range nameArr {
		dir := filepath.Join(proj.Dir.DocProtocolDir, name)
		protocolFilePathArr := make([]string, 0)
		protocolFilePath := filepath.Join(proj.Dir.DocProtocolDir, fmt.Sprintf("%s.go", name))
		protocolFilePathArr = append(protocolFilePathArr, protocolFilePath)
		if _, err := os.Stat(dir); err == nil {
			filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				if filepath.Ext(path) == ".go" {
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
		for _, filePath := range protocolFilePathArr {
			if err := p.parseFile(state, proto, filePath); err != nil {
				return err
			}
		}
		state.protocols = append(state.protocols, proto)
	}
	return nil
}

func (p *golang_parser) parseFile(state *gen_state, proto *protocol, filePath string) (err error) {
	log.Info("处理文件", filePath)
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
				switch s := spec.(type) {
				case *ast.ValueSpec:
					if s.Names[0].Name == "Header" {
						if err = p.parseHeader(state, proto, filePath, fileContent, fset, s); err != nil {
							return false
						}
					}
				case *ast.TypeSpec:
					if typ, ok := s.Type.(*ast.StructType); ok {
						var packet *Packet
						if packet, err = p.parsePacket(state, proto, filePath, fileContent, s.Name.Name, fset, typ); err != nil {
							return false
						} else if packet == nil {
							// 忽略
						} else {
							if x.Doc != nil {
								var commentArr []string
								if commentArr, err = p.extraComment(x.Doc); err != nil {
									return false
								}
								for _, comment := range commentArr {
									if strings.HasPrefix(comment, "//") {
										packet.CommentArr = append(packet.CommentArr, strings.Replace(comment, "//", "", 1))
									}
								}
							}
							proto.PacketArr = append(proto.PacketArr, packet)
							proto.PacketDict[s.Name.Name] = packet
						}
					}
				}
			}
		}
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *golang_parser) parsePacket(state *gen_state, proto *protocol, filePath string, fileContent []byte, name string, fset *token.FileSet, s *ast.StructType) (*Packet, error) {
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
	var model *Message
	var request *Message
	var response *Message
	for _, f := range s.Fields.List {
		if f.Names[0].Name == "Model" {
			if m, ok := f.Type.(*ast.StructType); ok {
				model = &Message{}
				if err := p.parseMessage(state, model, filePath, fileContent, name, fset, m); err != nil {
					return nil, err
				}
			}
		} else if f.Names[0].Name == "MessageId" {
			if err := p.parseMessageId(state, packet, filePath, fileContent, fset, name, f); err != nil {
				return nil, err
			}
		} else if f.Names[0].Name == "Request" {
			if m, ok := f.Type.(*ast.StructType); ok {
				request = &Message{}
				if err := p.parseMessage(state, request, filePath, fileContent, name, fset, m); err != nil {
					return nil, err
				}
			}
		} else if f.Names[0].Name == "Response" {
			if m, ok := f.Type.(*ast.StructType); ok {
				response = &Message{}
				if err := p.parseMessage(state, response, filePath, fileContent, name, fset, m); err != nil {
					return nil, err
				}
			}
		}
	}
	if request != nil && response != nil {
		packet.Type = message_type_request
	} else if request != nil {
		packet.Type = message_type_push
	} else if response != nil {
		packet.Type = message_type_notify
	} else if model != nil {
		packet.Type = message_type_struct
	} else {
		return nil, nil
	}
	if packet.Type != message_type_struct {
		if packet.MessageId == 0 {
			return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s MessageId not set", name))
		}
	}
	if packet.Type == message_type_request {
		packet.Request = request
		packet.Response = response
		packet.Request.StructName = fmt.Sprintf("%sRequest", camelString(name))
		packet.Response.StructName = fmt.Sprintf("%sResponse", camelString(name))
	} else if packet.Type == message_type_notify {
		packet.Notify = response
		packet.Notify.StructName = fmt.Sprintf("%sNotify", camelString(name))
	} else if packet.Type == message_type_push {
		packet.Push = request
		packet.Push.StructName = fmt.Sprintf("%sPush", camelString(name))
	} else if packet.Type == message_type_struct {
		packet.Message = model
		packet.Message.StructName = camelString(name)
	}
	return packet, nil
}

func (p *golang_parser) newAstError(fset *token.FileSet, filePath string, pos token.Pos, msg string) error {
	return fmt.Errorf("%s %v %s", filePath, fset.Position(pos), msg)
}

func (p *golang_parser) parseMessageId(state *gen_state, packet *Packet, filePath string, fileContent []byte, fset *token.FileSet, name string, f *ast.Field) error {
	if f.Tag == nil {
		return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag is empty", name, f.Names[0].Name))
	}
	messageId := f.Tag.Value[1 : len(f.Tag.Value)-1]
	if v, err := strconv.Atoi(messageId); err != nil {
		return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag is invalid %s", name, f.Names[0].Name, err))
	} else {
		packet.MessageId = v
	}
	return nil
}

func (p *golang_parser) parseHeader(state *gen_state, proto *protocol, filePath string, fileContent []byte, fset *token.FileSet, s *ast.ValueSpec) error {
	if len(s.Values) <= 0 {
		return p.newAstError(fset, filePath, s.Pos(), "directive header is nil")
	}
	if a, ok := s.Values[0].(*ast.BasicLit); !ok {
		return p.newAstError(fset, filePath, s.Pos(), "directive header must string")
	} else if a.Kind != token.STRING {
		return p.newAstError(fset, filePath, s.Pos(), "directive header must string")
	} else {
		value := a.Value[1 : len(a.Value)-1]
		proto.Header = value
		return nil
	}
}

func (p *golang_parser) parseMessage(state *gen_state, m *Message, filePath string, fileContent []byte, structName string, fset *token.FileSet, s *ast.StructType) error {
	m.FieldDict = make(map[string]*Field)
	m.FieldArr = make([]*Field, 0)
	for _, f := range s.Fields.List {
		var tag int
		var err error
		var fieldName string
		var typeStr string
		var tagStr string
		if f.Tag == nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag is empty", structName, f.Names[0].Name))
		}
		tagStr = f.Tag.Value
		tagStr = strings.Replace(tagStr, "`", "", 2)
		if tag, err = strconv.Atoi(tagStr); err != nil {
			return p.newAstError(fset, filePath, f.Pos(), fmt.Sprintf("%s %s tag is invalid %s", structName, f.Names[0].Name, err))
		}
		// ast.Print(fset, s)
		fieldName = f.Names[0].Name
		// WARN: 兼容yaml做的处理, Type转为小写type
		if fieldName == "Type" {
			fieldName = "type"
		}
		typeStr = string(fileContent[f.Type.Pos()-1 : f.Type.End()-1])
		isArray := false
		if _, ok := f.Type.(*ast.ArrayType); ok {
			isArray = true
			typeStr = strings.Replace(typeStr, "[]", "", 1)
			typeStr = strings.TrimSpace(typeStr)
			if i := strings.Index(typeStr, "*"); i >= 0 {
				typeStr = typeStr[i+1:]
				typeStr = strings.TrimSpace(typeStr)
			}
		} else if i := strings.Index(typeStr, "*"); i >= 0 {
			typeStr = typeStr[i+1:]
			typeStr = strings.TrimSpace(typeStr)
		}
		field := &Field{
			Name:      fieldName,
			CamelName: camelString(fieldName),
			Tag:       tag,
			IsArray:   isArray,
			Message:   m,
		}
		if f.Doc != nil {
			if comments, err := p.extraComment(f.Doc); err == nil {
				for _, comment := range comments {
					if strings.HasPrefix(comment, "//") {
						field.CommentArr = append(field.CommentArr, strings.Replace(comment, "//", "", 1))
					}
				}
			}
		}
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
		m.FieldDict[fieldName] = field
		m.FieldArr = append(m.FieldArr, field)
	}
	return nil
}
