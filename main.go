package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spotdemo4/treli/internal/model"
	"github.com/spotdemo4/treli/internal/proc"
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
			fmt.Printf("Error getting current path: %v\n", err)
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
		fmt.Printf("Could not search for config yaml: %v\n", err)
		os.Exit(1)
	}
	if yamlPath == "" {
		fmt.Println("Config file not found")
		os.Exit(1)
	}

	// Load settings from yaml config
	s, err = settings.GetYaml(yamlPath)
	if err != nil {
		fmt.Printf("Could not load config file: %v\n", err)
		os.Exit(1)
	}

	// If there's no apps we can't do anything, so just exit
	if len(s.Procs) == 0 {
		fmt.Println("No procs found")
		os.Exit(0)
	}

	// Create msg channel and context
	onchange := make(chan int, 100)
	ctx, cancel := context.WithCancel(context.Background())

	// Create apps
	procs := []*proc.Proc{}
	for name, p := range s.Procs {
		proc, err := proc.New(
			name,
			p.Exts,
			p.AutoStart,
			p.AutoRestart,
			p.Shell,
			p.Cwd,
			sh,
			onchange,
		)
		if err != nil {
			fmt.Printf("Cannot load proc %s: %s\n", name, err.Error())
			os.Exit(1)
		}

		procs = append(procs, proc)
	}

	// Start apps
	for _, proc := range procs {
		if proc.AutoStart {
			go proc.Start(ctx)
		}
	}

	// Start watching
	go proc.Watch(ctx, path, procs)

	// Gracefully shutdown on SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Received signal %s, closing\n", sig)

		cancel()
		for _, p := range procs {
			p.Wait()
		}
		close(onchange)
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
