package gen_application

import (
	"io/ioutil"
	"path"

	"github.com/Lyndon-Zhang/gira/proj"
	"gopkg.in/yaml.v2"
)

type yaml_parser struct {
}

type applications_yaml_file struct {
	Application []struct {
		Name string `yaml:"name"`
	} `yaml:"application"`
}

func (p *yaml_parser) parse(state *gen_state) error {
	applicationFilePath := path.Join(proj.Dir.DocDir, "application.yaml")
	if data, err := ioutil.ReadFile(applicationFilePath); err != nil {
		return err
	} else {
		applicationsFile := applications_yaml_file{}
		if err := yaml.Unmarshal(data, &applicationsFile); err != nil {
			return err
		}
		for _, v := range applicationsFile.Application {
			state.applications = append(state.applications, application{
				ApplicationName: v.Name,
				ModuleName:      proj.Module,
			})
		}
	}
	return nil
}
