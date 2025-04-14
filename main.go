package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spotdemo4/treli/internal/app"
	"github.com/spotdemo4/treli/internal/model"
	"github.com/spotdemo4/treli/internal/settings"
)

func main() {
	path := os.Getenv("DIR")
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Printf("\nerror: %s", err.Error())
			os.Exit(1)
		}
	}

	// Get settings file
	sf, err := os.ReadFile(filepath.Join(path, "treli.yaml"))
	if err != nil {
		fmt.Println("No `treli.yaml` file found. Have you run `treli init`?")
		os.Exit(0)
	}

	// Get settings
	settings, err := settings.Get(sf)
	if err != nil {
		fmt.Printf("Error loading settings: %s", err.Error())
		os.Exit(1)
	}

	// Get env file if it exists
	err = godotenv.Load(filepath.Join(path, ".env"))
	if err == nil {
		fmt.Println("Using environment variables from .env")
	}

	// Create msg channel and context
	msgs := make(chan app.Msg, 100)
	ctx, cancel := context.WithCancel(context.Background())

	// Load apps
	apps := []*app.App{}
	for _, s := range settings {
		app := app.New(
			ctx,
			msgs,
			s.Name,
			s.Color,
			filepath.Join(path, s.Dir),
			s.Exts,
			s.InvertCheck,
			s.Check,
			s.Build,
			s.Start,
		)
		apps = append(apps, app)
	}
	if len(apps) == 0 {
		fmt.Println("no apps found")
		os.Exit(0)
		return
	}

	args := os.Args[1:]
	if len(args) == 0 {
		run(ctx, path, apps, msgs)
	} else {
		switch args[0] {
		case "run", "dev":
			run(ctx, path, apps, msgs)
		case "check":
			check(ctx, apps, msgs)
		default:
			fmt.Printf("option `%s` not found\n", args[0])
		}
	}

	cancel()
	for _, a := range apps {
		a.Wait()
	}
	close(msgs)
}

func run(ctx context.Context, path string, apps []*app.App, msgs chan app.Msg) {
	// Start apps
	for _, a := range apps {
		go a.Run()
	}

	// Start watching
	go app.Watch(ctx, path, apps)

	// Start tea
	p := tea.NewProgram(
		model.NewRunner(apps, msgs),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
	}
}

func check(ctx context.Context, apps []*app.App, msgs chan app.Msg) {
	go func() {
		for msg := range msgs {
			switch msg.State {
			case app.StateSuccess:
				log.Printf("%s: success", msg.AppName)

			case app.StateError:
				log.Printf("%s: failed", msg.AppName)
				os.Exit(1)

			default:
				log.Printf("%s: %s", msg.AppName, msg.Text)
			}
		}
	}()

	// Start checks
	wg := sync.WaitGroup{}
	for _, a := range apps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Check(ctx)
		}()
	}

	wg.Wait()
}
