package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/spotdemo4/treli/internal/util"
)

type Vite struct {
	App *App
}

func NewVite(dir string, c chan Msg) (*App, error) {
	// Check if npx is installed
	_, err := exec.LookPath("npx")
	if err != nil {
		return nil, fmt.Errorf("'npx' not found in PATH")
	}

	// Create new context
	ctx, cancel := context.WithCancel(context.Background())

	// Create wait group
	wg := sync.WaitGroup{}

	app := App{
		Name:   "vite",
		Color:  "#fab387",
		Cancel: cancel,
		Wait:   wg.Wait,

		dir:     dir,
		exts:    []string{"js", "ts", "tsx", "jsx", "vue", "svelte", "mdx", "md"},
		msgChan: c,
		ctx:     ctx,
		wg:      &wg,
	}
	node := Vite{
		App: &app,
	}

	// Start dev
	go node.dev()

	return &app, nil
}

func (n *Vite) msg(m Msg) {
	m.Time = time.Now()
	m.App = n.App
	n.App.msgChan <- m
}

func (n *Vite) dev() {
	n.App.wg.Add(1)
	defer n.App.wg.Done()

	// Create cmd
	cmd := exec.Command("npx", "vite", "dev")
	cmd.Dir = n.App.dir

	// Stop cmd on exit
	n.App.wg.Add(1)
	go func() {
		defer n.App.wg.Done()
		<-n.App.ctx.Done()

		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			cmd.Process.Kill() // If the process is not responding to the interrupt signal, kill it
		}
	}()

	// Start cmd
	out, err := util.Run(cmd)
	if err != nil {
		n.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			n.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			n.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				n.msg(Msg{
					Text:    "Node stopped",
					Success: util.BoolPointer(true),
				})
			} else {
				n.msg(Msg{
					Text:    fmt.Sprintf("Node failed with exit code %d", out),
					Success: util.BoolPointer(false),
				})
			}
		}
	}
}
