package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/boyter/gocodewalker"
)

type App interface {
	Name() string
	Color() string
	Success() *bool
	Start(context.Context)
	Wait()
}

type Msg struct {
	Text    string
	Time    time.Time
	Key     *string
	Loading *bool
	Success *bool

	App *App
}

func FindApps(path string, c chan Msg) ([]*App, error) {
	apps := []*App{}

	fileListQueue := make(chan *gocodewalker.File, 100)
	fileWalker := gocodewalker.NewFileWalker(path, fileListQueue)
	fileWalker.IncludeHidden = true

	errorHandler := func(e error) bool {
		fmt.Printf("Error: %s", e.Error())
		return true
	}
	fileWalker.SetErrorHandler(errorHandler)

	go fileWalker.Start()

	for f := range fileListQueue {
		dir := filepath.Dir(f.Location)

		switch f.Filename {
		case "buf.yaml", "buf.yml":
			buf, err := NewBuf(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app buf")
			apps = append(apps, buf)

		case "vite.config.js", "vite.config.ts":
			vite, err := NewVite(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app vite")
			apps = append(apps, vite)

		case "svelte.config.js", "svelte.config.ts":
			svelte, err := NewSvelte(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app svelte")
			apps = append(apps, svelte)

		case "revive.toml":
			revive, err := NewRevive(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app revive")
			apps = append(apps, revive)

		case "eslint.config.js", "eslint.config.ts":
			eslint, err := NewESLint(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app eslint")
			apps = append(apps, eslint)

		case ".prettierrc":
			prettier, err := NewPrettier(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app prettier")
			apps = append(apps, prettier)

		case "go.mod":
			golang, err := NewGolang(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app golang")
			apps = append(apps, golang)

		case "sqlc.yaml":
			sqlc, err := NewSQLc(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app sqlc")
			apps = append(apps, sqlc)

		case ".sqlfluff":
			fluff, err := NewSQLFluff(dir, c)
			if err != nil {
				fmt.Printf("found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}
			fmt.Println("found app fluff")
			apps = append(apps, fluff)
		}
	}

	return apps, nil
}
