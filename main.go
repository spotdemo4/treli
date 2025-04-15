package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spotdemo4/treli/internal/app"
	"github.com/spotdemo4/treli/internal/model"
	"github.com/spotdemo4/treli/internal/settings"
	"github.com/twpayne/go-shell"
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

	// Get shell
	sh, ok := shell.CurrentUserShell()
	if !ok {
		sh = shell.DefaultShell()
		fmt.Printf("Could not get current shell, defaulting to %s", sh)
	}

	// Get settings
	s, err := settings.Get(path)
	if err != nil {
		fmt.Printf("Error getting settings: %s", err.Error())
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
	for name, s := range s.Apps {
		app := app.New(
			ctx,
			sh,
			msgs,
			name,
			s.Color,
			filepath.Join(path, s.Dir),
			s.Exts,
			s.OnStart,
			s.OnChange,
		)
		apps = append(apps, app)
	}
	if len(apps) == 0 {
		fmt.Println("no apps found")
		os.Exit(0)
		return
	}

	// Start apps
	for _, app := range apps {
		go func() {
			app.Run(app.OnStart)
		}()
	}

	// Start watching
	go app.Watch(ctx, path, apps)

	// Gracefully shutdown on SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Received signal: %s", sig)

		cancel()
		for _, a := range apps {
			a.Wait()
		}
		close(msgs)
	}()

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
