package gen_application

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path"
	"strings"

	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/proj"
	"gopkg.in/yaml.v2"
)

type applications_annotate struct {
	Applications string
}

func (p *golang_parser) extraApplicationsAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*applications_annotate, error) {
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
	annotates := &applications_annotate{}

	if v, ok := values["applications"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s applications annotate must string", name))
	} else {
		annotates.Applications = str
	}
	return annotates, nil
}

type application_annotate struct {
	Module  string
	Version string
}

func (p *golang_parser) extraApplicationAnnotate(fset *token.FileSet, filePath string, s *ast.StructType, name string, comments []*ast.Comment) (*application_annotate, error) {
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
	annotates := &application_annotate{}

	if v, ok := values["module"]; !ok {
		return nil, nil
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s module annotate must string", name))
	} else {
		annotates.Module = str
	}
	if v, ok := values["version"]; !ok {
	} else if str, ok := v.(string); !ok {
		return nil, p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s version annotate must string", name))
	} else {
		annotates.Version = str
	}
	return annotates, nil
}

type golang_parser struct {
}

func (p *golang_parser) newAstError(fset *token.FileSet, filePath string, pos token.Pos, msg string) error {
	return fmt.Errorf("%s %v %s", filePath, fset.Position(pos), msg)
}

func (p *golang_parser) parseApplicationStruct(state *gen_state, filePath string, fset *token.FileSet, name string, doc *ast.CommentGroup, s *ast.StructType) error {
	if doc == nil {
		return p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("struct %s module annotate must set", name))
	}
	if annotations, err := p.extraApplicationAnnotate(fset, filePath, s, name, doc.List); err != nil {
		return err
	} else {
		// var version string
		// for _, v := range s.Fields.List {
		// 	switch v.Names[0].Name {
		// 	case "Version":
		// 		if v.Tag == nil {
		// 			return p.newAstError(fset, filePath, s.Pos(), fmt.Sprintf("%s.%s is empty", name, v.Names[0].Name))
		// 		} else {
		// 			version = strings.Replace(v.Tag.Value, "`", "", -1)
		// 		}
		// 	}
		// }
		state.applications = append(state.applications, application{
			ApplicationName: name,
			ModuleName:      annotations.Module,
			Version:         annotations.Version,
		})
	}
	return nil
}

func (p *golang_parser) parseApplicationsStruct(state *gen_state, filePath string, fileContent []byte, fset *token.FileSet, g *ast.GenDecl, t *ast.TypeSpec, s *ast.StructType) error {
	if g.Doc == nil {
		// return p.newAstError(fset, filePath, g.Pos(), fmt.Sprintf("%s annotate not set", t.Name.Name))
		return nil
	}
	if annotations, err := p.extraApplicationsAnnotate(fset, filePath, s, t.Name.Name, g.Doc.List); err != nil {
		return err
	} else if annotations == nil {
		return nil
	}
	for _, field := range s.Fields.List {
		if err := p.parseApplicationStruct(state, filePath, fset, field.Names[0].Name, field.Doc, field.Type.(*ast.StructType)); err != nil {
			return err
		}
	}
	return nil
}

func (p *golang_parser) parse(state *gen_state) (err error) {
	filePath := path.Join(proj.Dir.DocDir, "application.go")
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
					if s, ok := st.Type.(*ast.StructType); ok {
						if err = p.parseApplicationsStruct(state, filePath, fileContent, fset, x, st, s); err != nil {
							return false
						}
					}
				}
			}
		}
		return true
	})
	return
}
