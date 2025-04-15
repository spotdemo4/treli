package model

import "github.com/charmbracelet/lipgloss"

var prefix = lipgloss.NewStyle().
	Padding(0, 1, 0, 1).
	Margin(0, 1, 0, 1).
	Background(lipgloss.Color("#89dceb")).
	Foreground(lipgloss.Color("#11111b"))

var checkmark = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#a6e3a1")).
	Bold(true).
	Render("✓")

var xmark = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f38ba8")).
	Bold(true).
	Render("✕")

var pause = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f9e2af")).
	Bold(true).
	Render("⏸")
