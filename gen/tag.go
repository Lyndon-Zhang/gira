package gen

import (
	"go/ast"
	"regexp"
	"strconv"
	"strings"

	"github.com/Lyndon-Zhang/gira/errors"
)

type TagList struct {
	Kv map[string]string
}

func (self *TagList) Int(k string) (int, error) {
	if v, ok := self.Kv[k]; !ok {
		return 0, errors.ErrTODO
	} else if v, err := strconv.Atoi(v); err != nil {
		return 0, err
	} else {
		return v, nil
	}
}

func ExplodeTag(lit *ast.BasicLit) (*TagList, error) {
	tags := &TagList{
		Kv: make(map[string]string),
	}
	str := lit.Value
	str = str[1 : len(str)-1]
	r := regexp.MustCompile(`(\w+)\s*:\s*"((?:\\.|[^"\\])*)"`)
	matches := r.FindAllStringSubmatch(str, -1)
	for _, v := range matches {
		if len(v) == 3 {
			tags.Kv[v[1]] = v[2]
		}
	}
	return tags, nil
}

func ExtraComment(commentGroup *ast.CommentGroup) ([]string, error) {
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
