package settings

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type app struct {
	Color string   `yaml:"color"`
	Dir   string   `yaml:"dir"`
	Exts  []string `yaml:"exts"`

	OnStart  string `yaml:"onstart"`
	OnChange string `yaml:"onchange"`
}

type Settings struct {
	Apps map[string]app `yaml:"apps"`
}

func Get(path string) (*Settings, error) {
	// Get settings file
	sf, err := os.ReadFile(filepath.Join(path, "treli.yaml"))
	if err != nil {
		return nil, err
	}

	// Load settings file
	settings := Settings{}
	if err := yaml.Unmarshal(sf, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func Create() {

}
