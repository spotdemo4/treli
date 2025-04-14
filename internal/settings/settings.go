package settings

import (
	"errors"

	"github.com/goccy/go-yaml"
)

type Setting struct {
	Name        string
	Color       string
	Dir         string
	Exts        []string
	InvertCheck bool

	Check string
	Build string
	Start string
}

func Get(yml []byte) ([]Setting, error) {
	settings := []Setting{}

	tmp := map[string]any{}
	if err := yaml.Unmarshal(yml, &tmp); err != nil {
		return nil, err
	}

	tmpApps, ok := tmp["apps"]
	if !ok {
		return nil, errors.New("no apps found")
	}

	tmpAppsList := tmpApps.([]any)
	for _, tmpApp := range tmpAppsList {
		tmpAppMap := tmpApp.(map[string]any)
		s := Setting{}

		for k, v := range tmpAppMap {
			s.Name = k
			values := v.(map[string]any)

			if color, ok := values["color"]; ok {
				s.Color = color.(string)
			}

			if dir, ok := values["dir"]; ok {
				s.Dir = dir.(string)
			}

			if exts, ok := values["exts"]; ok {
				tempext := exts.([]any)
				for _, ext := range tempext {
					s.Exts = append(s.Exts, ext.(string))
				}
			}

			if invertcheck, ok := values["invertcheck"]; ok {
				s.InvertCheck = invertcheck.(bool)
			}

			if check, ok := values["check"]; ok {
				s.Check = check.(string)
			}

			if build, ok := values["build"]; ok {
				s.Build = build.(string)
			}

			if start, ok := values["start"]; ok {
				s.Start = start.(string)
			}
		}

		settings = append(settings, s)
	}

	return settings, nil
}
