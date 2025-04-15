package model

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type Terminal struct {
	style        lipgloss.Style
	Viewport     *viewport.Model
	maxPrefixLen int
}

func NewTerminal(maxPrefixLen int) *Terminal {
	s := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#45475a")).
		BorderTop(true).
		BorderBottom(true).
		Margin(1, 0, 0)

	return &Terminal{
		style:        s,
		Viewport:     nil,
		maxPrefixLen: maxPrefixLen,
	}
}

func (c *Terminal) Gen(text string, width int, height int, mtop int, mbottom int) string {
	if c.Viewport == nil {
		vp := viewport.New(width, height-(mtop+mbottom)+2)
		c.Viewport = &vp
		c.Viewport.YPosition = mtop
		c.Viewport.Style = c.style
	} else {
		c.Viewport.Width = width
		c.Viewport.Height = height - (mtop + mbottom) + 2
		c.Viewport.YPosition = mtop
	}

	atBottom := c.Viewport.AtBottom()
	c.Viewport.SetContent(text)

	if atBottom {
		c.Viewport.GotoBottom()
	}

	return c.Viewport.View()
}

func (c *Terminal) GenItem(ti time.Time, prefix string, text string, color string, width int) string {
	t := lipgloss.NewStyle().
		Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color("#a6adc8")).
		Render(ti.Format(time.Kitchen))

	p := lipgloss.NewStyle().
		Padding(0, 1, 0, 0).
		Width(c.maxPrefixLen).
		Foreground(lipgloss.Color(color)).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(lipgloss.Color(color))

	m := lipgloss.NewStyle().
		Padding(0, 1, 0, 1).
		Foreground(lipgloss.Color("#cdd6f4")).
		Width(width - lipgloss.Width(t) - lipgloss.Width(p.Render(prefix))).
		Render(text)

	p = p.Height(lipgloss.Height(m))

	combine := lipgloss.JoinHorizontal(lipgloss.Top, t, p.Render(prefix), m)

	return lipgloss.NewStyle().
		Width(width).
		Render(combine)
}
