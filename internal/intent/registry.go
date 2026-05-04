package intent

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type CommandDef struct {
	Name    string   `yaml:"name"`
	Aliases []string `yaml:"aliases"`
	Exec    string   `yaml:"exec"`
}

type Registry struct {
	Commands []CommandDef
}

func NewRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Read registry: %w", err)
	}

	var cmds []CommandDef

	if err := yaml.Unmarshal(data, &cmds); err != nil {
		return nil, fmt.Errorf("Parse registry: %w", err)
	}

	return &Registry{Commands: cmds}, nil
}
