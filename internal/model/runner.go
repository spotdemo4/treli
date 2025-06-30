package model

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spotdemo4/treli/internal/proc"
	"github.com/spotdemo4/treli/internal/util"
)

type Runner struct {
	ctx    context.Context
	width  *int
	height *int

	header   *Header
	terminal *Terminal
	help     *Help
	spinner  spinner.Model

	procs    []*proc.Proc
	onchange chan int
	selected int
}

func NewRunner(ctx context.Context, procs []*proc.Proc, onchange chan int) *Runner {
	mpl := 0

	for _, app := range procs {
		if len((*app).Name) > mpl {
			mpl = len((*app).Name)
		}
	}

	return &Runner{
		ctx:    ctx,
		width:  nil,
		height: nil,

		header:   NewHeader(),
		terminal: NewTerminal(mpl + 1),
		help:     NewHelp(),
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")))),

		procs:    procs,
		onchange: onchange,
	}
}

func (m Runner) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			return <-m.onchange
		},
	)
}

func (m Runner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case int:
		return m, func() tea.Msg {
			return <-m.onchange
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
				for _, p := range m.procs {
					(*p).Stop()
				}

				return tea.QuitMsg{}
			}

		case key.Matches(msg, m.help.keys.Up):
			m.terminal.Viewport.ScrollUp(1)

		case key.Matches(msg, m.help.keys.Down):
			m.terminal.Viewport.ScrollDown(1)

		case key.Matches(msg, m.help.keys.Start):
			p := m.procs[m.selected]
			if p.State() == proc.StateRunning {
				(*m.procs[m.selected]).Stop()
			} else {
				(*m.procs[m.selected]).Start(m.ctx)
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
		case app.StatePause:
			item = append(item, pause)
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

	os.WriteFile("/tmp/teatest", []byte(s), 0644)

	// Send the UI for rendering
	return s
}
