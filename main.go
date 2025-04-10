package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spotdemo4/treli/internal/app"
	"github.com/spotdemo4/treli/internal/model"
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

	// Get env file if it exists
	err := godotenv.Load(filepath.Join(path, ".env"))
	if err == nil {
		fmt.Println("found env file")
	}

	// Create msg channel
	msgs := make(chan app.Msg, 100)

	// Find apps
	fmt.Printf("searching for apps in %s\n", path)
	apps, err := app.FindApps(path, msgs)
	if err != nil {
		fmt.Printf("\nerror: %s\n", err.Error())
		os.Exit(1)
	}
	if len(apps) == 0 {
		fmt.Println("\nno apps found")
		os.Exit(0)
		return
	}

	// Start tea
	p := tea.NewProgram(
		model.NewRunner(apps, msgs),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v", err)
	}

	fmt.Println("\ndone")

	close(msgs)
}
