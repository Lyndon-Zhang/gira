package gen

import (
	"go/ast"
	"log"
	"strings"

	"gopkg.in/yaml.v2"
)

type Annotate struct {
	Name   string
	Values []string
}

func (a *Annotate) String() string {
	if a.Values == nil {
		return ""
	} else {
		return strings.Join(a.Values, ",")
	}
}
func ExtraAnnotate(comments []*ast.Comment) (map[string]*Annotate, error) {
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
	annotates := make(map[string]*Annotate)
	for k, v := range values {
		switch a := v.(type) {
		case []interface{}:
			annotate := &Annotate{
				Name: k,
			}
			for _, v1 := range a {
				annotate.Values = append(annotate.Values, v1.(string))
			}
			annotates[k] = annotate
		case interface{}:
			annotate := &Annotate{
				Name: k,
			}
			annotate.Values = append(annotate.Values, a.(string))
			annotates[k] = annotate
		}
	}
	return annotates, nil
}
