package gen_const

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path"
	"strings"

	"github.com/Lyndon-Zhang/gira/proj"
	yaml "gopkg.in/yaml.v3"
)

// 注解
type struct_annotate struct {
	Excel   string
	Key     string
	Value   string
	Comment *string
}

func (p *golang_parser) newAstError(fset *token.FileSet, filePath string, pos token.Pos, msg string) error {
	return fmt.Errorf("%s %v %s", filePath, fset.Position(pos), msg)
}

func (p *golang_parser) extraStructAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*struct_annotate, error) {
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
	annotates := &struct_annotate{}

	if v, ok := values["excel"]; !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s excel annotate not set", name))
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s excel annotate must string", name))
	} else {
		annotates.Excel = str
	}

	if v, ok := values["key"]; !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s key annotate not set", name))
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s key annotate must string", name))
	} else {
		annotates.Key = str
	}
	if v, ok := values["value"]; !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s value annotate not set", name))
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s value annotate must string", name))
	} else {
		annotates.Value = str
	}
	if v, ok := values["comment"]; ok {
		if str, ok := v.(string); ok {
			annotates.Comment = &str
		}
	}
	return annotates, nil
}

type consts_annotate struct {
	Consts string
}

func (p *golang_parser) extraConstsAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*consts_annotate, error) {
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
	annotates := &consts_annotate{}

	if v, ok := values["consts"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s consts annotate must string", name))
	} else {
		annotates.Consts = str
	}
	return annotates, nil
}

type golang_parser struct {
}

func (p *golang_parser) parseConstStruct(state *const_state, fset *token.FileSet, filePath string, s *ast.StructType) error {
	for _, field := range s.Fields.List {
		name := field.Names[0].Name
		if annotates, err := p.extraStructAnnotate(fset, filePath, s, name, field.Doc.List); err != nil {
			return err
		} else {
			filePath := annotates.Excel
			keyArr := []string{annotates.Key, annotates.Value}
			if annotates.Comment != nil {
				keyArr = append(keyArr, *annotates.Comment)
			}
			descriptor := &Descriptor{
				Name:     name,
				filePath: filePath,
				keyArr:   keyArr,
			}
			state.constFile.descriptorDict[name] = descriptor
		}
	}
	return nil
}

func (p *golang_parser) parse(state *const_state) (err error) {
	filePath := path.Join(proj.Dir.DocDir, "const.go")
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
					if s, ok := st.Type.(*ast.StructType); ok {
						if err = p.parseConstsStruct(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
					}
				}
			}
		case *ast.TypeSpec:
			if x.Name.Name == "Const" {
				if err = p.parseConstStruct(state, fset, filePath, x.Type.(*ast.StructType)); err != nil {
					return false
				}
			}
		}
		return true
	})
	return
}

func (p *golang_parser) parseConstsStruct(state *const_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
		return nil
	}
	if annotations, err := p.extraConstsAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	}
	for _, field := range s.Fields.List {
		if err := p.parseConstStruct(state, fset, filePath, field.Type.(*ast.StructType)); err != nil {
			return err
		}
	}
	return nil
}
