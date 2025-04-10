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
	app     *App
	success *bool
	dir     string
	exts    []string
	msgs    chan Msg
	wg      *sync.WaitGroup
}

func NewVite(dir string, msgs chan Msg) (*App, error) {
	// Check if npx is installed
	_, err := exec.LookPath("npx")
	if err != nil {
		return nil, fmt.Errorf("'npx' not found in PATH")
	}

	// Create wait group
	wg := sync.WaitGroup{}

	vite := Vite{
		dir:  dir,
		exts: []string{"js", "ts", "tsx", "jsx", "vue", "svelte", "mdx", "md"},
		msgs: msgs,
		wg:   &wg,
	}

	app := App(&vite)
	vite.app = &app

	return &app, nil
}

func (v *Vite) Name() string {
	return "vite"
}

func (v *Vite) Color() string {
	return "#fab387"
}

func (v *Vite) Success() *bool {
	return v.success
}

func (v *Vite) Start(ctx context.Context) {
	v.wg.Wait()

	go v.dev(ctx)
}

func (v *Vite) Wait() {
	v.msg(Msg{
		Text:    "stopping",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("vite stop"),
	})

	v.wg.Wait()

	v.msg(Msg{
		Text:    "stopped",
		Loading: util.BoolPointer(false),
		Success: util.BoolPointer(true),
		Key:     util.StringPointer("vite stop"),
	})
}

func (v *Vite) msg(m Msg) {
	if m.Loading != nil {
		if *m.Loading {
			v.success = nil
		} else {
			v.success = m.Success
		}
	}
	m.Time = time.Now()
	m.App = v.app

	v.msgs <- m
}

func (v *Vite) dev(ctx context.Context) {
	v.wg.Add(1)
	defer v.wg.Done()
	key := v.Name() + "dev"

	v.msg(Msg{
		Text:    "starting",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	// Send good message if running for longer than 15 seconds
	go func() {
		select {
		case <-ctx.Done():
		case <-time.After(15 * time.Second):
			v.msg(Msg{
				Text:    fmt.Sprintf("started"),
				Loading: util.BoolPointer(false),
				Success: util.BoolPointer(true),
				Key:     &key,
			})
		}
	}()

	// Create cmd
	cmd := exec.Command("npx", "vite", "dev")
	cmd.Dir = v.dir

	// Stop cmd on exit
	v.wg.Add(1)
	go func() {
		defer v.wg.Done()
		<-ctx.Done()

		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			cmd.Process.Kill() // If the process is not responding to the interrupt signal, kill it
		}
	}()

	// Start cmd
	out, err := util.Run(cmd)
	if err != nil {
		v.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			v.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			v.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				v.msg(Msg{
					Text:    "dev stopped",
					Loading: util.BoolPointer(false),
					Success: util.BoolPointer(true),
					Key:     &key,
				})
			} else {
				v.msg(Msg{
					Text:    fmt.Sprintf("dev stopped"),
					Loading: util.BoolPointer(false),
					Success: util.BoolPointer(false),
					Key:     &key,
				})
			}
		}
	}
}
