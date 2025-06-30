package settings

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/goccy/go-yaml"
)

func GetYaml(path string) (*Settings, error) {
	// Read settings file
	sf, err := os.ReadFile(path)
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

func FindYaml(path string) (string, error) {
	r, _ := regexp.Compile(`^(\.)?treli.y(a)?ml$`)

	var file string
	err := filepath.WalkDir(path, func(p string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if r.MatchString(info.Name()) {
			file = p
			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return file, nil
}

func CreateYaml(path string, settings *Settings) error {
	// Turn into yaml
	bytes, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	// Save to .treli.yaml
	err = os.WriteFile(filepath.Join(path, ".treli.yaml"), bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
