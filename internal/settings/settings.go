package settings

import (
	"os"
	"path/filepath"
	"strings"
)

type app struct {
	Color string   `yaml:"color"`
	Dir   string   `yaml:"dir,omitempty"`
	Exts  []string `yaml:"exts,omitempty"`

	OnStart  string `yaml:"onstart,omitempty"`
	OnChange string `yaml:"onchange,omitempty"`
}

type Settings struct {
	Apps map[string]app `yaml:"apps"`
}

func Get(path string) (*Settings, error) {
	apps := map[string]app{}

	// Find setting files
	err := filepath.WalkDir(path, func(p string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		switch info.Name() {
		case "buf.yaml", "buf.yml":
			app := app{
				Color:    "#cba6f7",
				Dir:      strings.TrimPrefix(filepath.Dir(p), path),
				Exts:     []string{"proto"},
				OnStart:  "buf lint",
				OnChange: "buf lint && buf generate",
			}
			apps["buf"] = app

		case "eslint.config.js", "eslint.config.ts":
			app := app{
				Color: "#fab387",
				Dir:   strings.TrimPrefix(filepath.Dir(p), path),
				Exts: []string{
					"js",
					"ts",
					"jsx",
					"tsx",
					"vue",
					"svelte",
				},
				OnStart:  "npx eslint .",
				OnChange: "npx eslint .",
			}
			apps["eslint"] = app

		case "go.mod":
			app := app{
				Color:    "#89dceb",
				Dir:      strings.TrimPrefix(filepath.Dir(p), path),
				Exts:     []string{"go"},
				OnStart:  "go build -o ./tmp/app && ./tmp/app",
				OnChange: "go build -o ./tmp/app && ./tmp/app",
			}
			apps["golang"] = app

		case ".prettierrc":
			app := app{
				Color: "#fab387",
				Dir:   strings.TrimPrefix(filepath.Dir(p), path),
				Exts: []string{
					"js",
					"ts",
					"jsx",
					"tsx",
					"vue",
					"svelte",
				},
				OnStart:  "npx prettier --check .",
				OnChange: "npx prettier --check . || npx prettier --write .",
			}
			apps["prettier"] = app

		case "revive.toml":
			app := app{
				Color:    "#89dceb",
				Dir:      strings.TrimPrefix(filepath.Dir(p), path),
				Exts:     []string{"go"},
				OnStart:  "revive -config revive.toml -set_exit_status ./...",
				OnChange: "revive -config revive.toml -set_exit_status ./...",
			}
			apps["revive"] = app

		case "sqlc.yaml", "sqlc.yml":
			app := app{
				Color:    "#a6e3a1",
				Dir:      strings.TrimPrefix(filepath.Dir(p), path),
				Exts:     []string{"sql"},
				OnStart:  "sqlc vet",
				OnChange: "sqlc vet && sqlc generate",
			}
			apps["sqlc"] = app

		case ".sqlfluff":
			app := app{
				Color:    "#a6e3a1",
				Dir:      strings.TrimPrefix(filepath.Dir(p), path),
				Exts:     []string{"sql"},
				OnStart:  "sqlfluff lint",
				OnChange: "sqlfluff lint",
			}
			apps["sqlfluff"] = app

		case "svelte.config.js", "svelte.config.ts":
			app := app{
				Color:    "#fab387",
				Dir:      strings.TrimPrefix(filepath.Dir(p), path),
				Exts:     []string{"svelte"},
				OnStart:  "npx svelte-check",
				OnChange: "npx svelte-check",
			}
			apps["svelte"] = app

		case "vite.config.js", "vite.config.ts":
			app := app{
				Color:   "#fab387",
				Dir:     strings.TrimPrefix(filepath.Dir(p), path),
				OnStart: "npx vite dev",
			}
			apps["vite"] = app
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Settings{
		Apps: apps,
	}, nil
}
