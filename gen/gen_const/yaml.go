package gen_const

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/proj"
	yaml "gopkg.in/yaml.v3"
)

type yaml_parser struct {
}

func (p *yaml_parser) parse(constState *const_state) error {
	filePath := path.Join(proj.Dir.DocDir, "const.yaml")
	if data, err := ioutil.ReadFile(filePath); err != nil {
		return err
	} else {
		result := make(map[string]interface{})
		if err := yaml.Unmarshal(data, result); err != nil {
			return err
		}
		if descriptors, ok := result["descriptor"]; !ok {
			return errors.ErrTODO.Trace()
		} else {
			for _, row := range descriptors.([]interface{}) {
				arr := row.([]interface{})
				if len(arr) < 3 {
					return fmt.Errorf("descriptor %+v at least need 3 argument", row)
				}
				var ok bool
				var filePath string
				var name string
				filePath, ok = arr[0].(string)
				if !ok {
					return fmt.Errorf("descriptor %+v arg1 not string", row)
				}
				name, ok = arr[1].(string)
				if !ok {
					return fmt.Errorf("descriptor %+v arg2 not string", row)
				}
				keyArr := make([]string, 0)
				if _, ok := arr[2].([]interface{}); !ok {
					return fmt.Errorf("descriptor %+v arg3 not a array", name)
				}
				for _, v := range arr[2].([]interface{}) {
					vv, ok := v.(string)
					if !ok {
						return fmt.Errorf("descriptor %+v arg3 not a string array", name)
					}
					keyArr = append(keyArr, vv)
				}
				if len(keyArr) != 2 && len(keyArr) != 3 {
					return fmt.Errorf("descriptor %+v arg3 array length must equal 2 or 3", name)
				}
				item := &Descriptor{
					Name:     name,
					filePath: filePath,
					keyArr:   keyArr,
				}
				constState.constFile.descriptorDict[name] = item
			}
		}
	}
	return nil
}
