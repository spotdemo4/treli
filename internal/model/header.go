package model

import "github.com/charmbracelet/lipgloss"

type Header struct {
	style lipgloss.Style
}

func NewHeader() *Header {
	s := lipgloss.NewStyle().
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Bottom).
		MarginTop(1)

	return &Header{
		style: s,
	}
}

func (h *Header) Gen(width int, items ...string) string {
	s := h.style.Width(width)

	pp := lipgloss.JoinHorizontal(lipgloss.Center, items...)

	return s.Render(pp)
}

func (h *Header) GenItem(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")).
		Margin(0, 1).
		Render(text)
}
