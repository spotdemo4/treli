package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spotdemo4/treli/internal/app"
	"github.com/spotdemo4/treli/internal/util"
)

type Runner struct {
	width  *int
	height *int

	header   *Header
	terminal *Terminal
	help     *Help
	spinner  spinner.Model

	msgChan chan app.Msg
	msgs    []app.Msg
	apps    []*app.App
}

func NewRunner(applications []*app.App, msgChan chan app.Msg) *Runner {
	mpl := 0

	for _, app := range applications {
		if len((*app).Name) > mpl {
			mpl = len((*app).Name)
		}
	}

	return &Runner{
		width:  nil,
		height: nil,

		header:   NewHeader(),
		terminal: NewTerminal(mpl + 1),
		help:     NewHelp(),
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")))),

		msgChan: msgChan,
		msgs:    []app.Msg{},
		apps:    applications,
	}
}

func (m Runner) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			return app.Msg(<-m.msgChan)
		},
	)
}

func (m Runner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case app.Msg:
		// Edit old message with the same key
		if msg.State == app.StateError || msg.State == app.StateSuccess {
			for i, prev := range m.msgs {
				if prev.Key == msg.Key && prev.State == app.StateLoading {
					m.msgs[i].State = app.StateIdle
					break
				}
			}
		}

		// Append new message
		m.msgs = append(m.msgs, msg)

		return m, func() tea.Msg {
			return app.Msg(<-m.msgChan)
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = util.IntPointer(msg.Width)
		m.height = util.IntPointer(msg.Height)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.help.keys.Help):
			m.help.Toggle()

		case key.Matches(msg, m.help.keys.Quit):
			return m, func() tea.Msg {
				for _, a := range m.apps {
					(*a).Stop()
				}

				return tea.QuitMsg{}
			}
		}

	case tea.QuitMsg:
		return m, tea.Quit

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelDown:
			m.terminal.Viewport.ScrollDown(1)

		case tea.MouseButtonWheelUp:
			m.terminal.Viewport.ScrollUp(1)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Runner) term() (rows []string) {
	if m.width == nil {
		return rows
	}

	for _, msg := range m.msgs {
		item := []string{}

		switch msg.State {
		case app.StateLoading:
			item = append(item, m.spinner.View())
		case app.StateSuccess:
			item = append(item, checkmark)
		case app.StateError:
			item = append(item, xmark)
		}

		item = append(item, msg.Text)
		itemStr := strings.Join(item, " ")

		// Render the row
		rows = append(rows, m.terminal.GenItem(msg.Time, msg.AppName, itemStr, msg.AppColor, *m.width))
	}

	return rows
}

func (m Runner) head() (items []string) {
	for _, a := range m.apps {
		item := []string{}

		switch a.State {
		case app.StateLoading:
			item = append(item, m.spinner.View())
		case app.StateSuccess:
			item = append(item, checkmark)
		case app.StateError:
			item = append(item, xmark)
		}

		item = append(item, (*a).Name)
		items = append(items, m.header.GenItem(strings.Join(item, " ")))
	}

	return items
}

func (m Runner) View() string {
	if m.width == nil || m.height == nil {
		return fmt.Sprintf("\n %s Loading...", m.spinner.View())
	}

	// Generate the UI
	header := m.header.Gen(*m.width, m.head()...)
	footer := m.help.Gen(*m.width)
	main := m.terminal.Gen(
		strings.Join(m.term(), "\n"),
		*m.width,
		*m.height,
		lipgloss.Height(header),
		lipgloss.Height(footer),
	)

	s := header
	s += main
	s += footer

	// Send the UI for rendering
	return s
}
