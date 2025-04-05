package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/boyter/gocodewalker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spotdemo4/treli/internal/app"
	"github.com/spotdemo4/treli/internal/model"
)

func main() {
	path := os.Getenv("DIR")
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}

	// Create message channel
	c := make(chan app.Msg, 100)

	fmt.Printf("Searching for apps in %s\n", path)
	apps := findApps(path, c)
	if len(apps) == 0 {
		fmt.Println("\nNo apps found")
		return
	}

	fmt.Println("Starting UI")
	// Start tea
	p := tea.NewProgram(
		model.NewRunner(c, apps),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
	}

	// Stop apps
	fmt.Println()
	for _, a := range apps {
		fmt.Printf("Stopping %s\n", a.Name)
		a.Cancel()
		a.Wait()
	}

	close(c)
}

func findApps(path string, c chan app.Msg) []*app.App {
	apps := []*app.App{}

	fileListQueue := make(chan *gocodewalker.File, 100)
	fileWalker := gocodewalker.NewFileWalker(path, fileListQueue)

	errorHandler := func(e error) bool {
		fmt.Println("ERR", e.Error())
		return true
	}
	fileWalker.SetErrorHandler(errorHandler)

	go fileWalker.Start()

	for f := range fileListQueue {
		dir := filepath.Dir(f.Location)

		switch f.Filename {
		case "buf.yaml", "buf.yml":
			buf, err := app.NewBuf(dir, c)
			if err != nil {
				fmt.Printf("Found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}

			apps = append(apps, buf)

		case "vite.config.js", "vite.config.ts":
			vite, err := app.NewVite(dir, c)
			if err != nil {
				fmt.Printf("Found %s but could not add it: %s\n", f.Filename, err.Error())
				continue
			}

			apps = append(apps, vite)
		}
	}

	return apps
}
