package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spotdemo4/treli/internal/app"
	"github.com/spotdemo4/treli/internal/model"
	"github.com/spotdemo4/treli/internal/settings"
	"github.com/twpayne/go-shell"
)

func main() {
	// Get current path
	path := os.Getenv("DIR")
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current path: %s\n", err.Error())
			os.Exit(1)
		}
	}

	// Get shell
	sh, ok := shell.CurrentUserShell()
	if !ok {
		sh = shell.DefaultShell()
		fmt.Printf("Could not get current shell, defaulting to %s\n", sh)
	}

	// Get settings
	var s *settings.Settings
	yamlPath, err := settings.FindYaml(path)
	if err != nil {
		fmt.Printf("Error searching for treli.yaml: %s\n", err.Error())
		os.Exit(1)
	}
	if yamlPath == "" {
		fmt.Println("Settings file not found")

		// Try to find apps via their config files
		s, err = settings.Get(path)
		if err != nil {
			fmt.Printf("Error searching for apps: %s\n", err.Error())
			os.Exit(1)
		}
		if len(s.Apps) == 0 {
			fmt.Println("No apps found")
			os.Exit(0)
		}

		fmt.Printf("Found %d app(s). Would you like to create a config file?\n", len(s.Apps))
		fmt.Print("[y/n]: ")

		// Create reader for reading y/n response
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Could not create reader: %s\n", err.Error())
			os.Exit(1)
		}
		response = strings.ToLower(strings.TrimSpace(response))

		// Create .treli.yaml
		if response == "y" || response == "yes" {
			err = settings.CreateYaml(path, s)
			if err != nil {
				fmt.Printf("Could not create .treli.yaml: %s\n", err.Error())
				os.Exit(1)
			}
		}
	} else {
		// Load settings from yaml config
		s, err = settings.GetYaml(yamlPath)
		if err != nil {
			fmt.Printf("Could not load config file: %s\n", err.Error())
			os.Exit(1)
		}
	}
	// If there's no apps we can't do anything, so just exit
	if len(s.Apps) == 0 {
		fmt.Println("no apps found")
		os.Exit(0)
	}

	// Create msg channel and context
	msgs := make(chan app.Msg, 100)
	ctx, cancel := context.WithCancel(context.Background())

	// Create apps
	apps := []*app.App{}
	for name, s := range s.Apps {
		app, err := app.New(
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
		if err != nil {
			fmt.Printf("Cannot use app %s: %s\n", name, err.Error())
			os.Exit(1)
		}

		apps = append(apps, app)
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
		fmt.Printf("Received signal %s, closing\n", sig)

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
		fmt.Printf("Error running tea: %v\n", err)
	}
}
