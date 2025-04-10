package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spotdemo4/treli/internal/util"
)

type Golang struct {
	app     *App
	success *bool
	dir     string
	exts    []string
	msgs    chan Msg
	wg      *sync.WaitGroup

	runWg *sync.WaitGroup
}

func NewGolang(dir string, msgs chan Msg) (*App, error) {
	// Check if buf is installed
	_, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("'go' not found in PATH")
	}

	// Create wait groups
	wg := sync.WaitGroup{}
	runWg := sync.WaitGroup{}

	golang := Golang{
		dir:  dir,
		exts: []string{"go"},
		msgs: msgs,
		wg:   &wg,

		runWg: &runWg,
	}

	app := App(&golang)
	golang.app = &app

	return &app, nil
}

func (a *Golang) Name() string {
	return "go"
}

func (a *Golang) Color() string {
	return "#89dceb"
}

func (a *Golang) Success() *bool {
	return a.success
}

func (a *Golang) Start(ctx context.Context) {
	a.wg.Wait()

	runCtx, runCancel := context.WithCancel(ctx)

	go func() {
		a.build()
		a.run(runCtx)
	}()
	go a.watch(ctx, runCancel)
}

func (a *Golang) Wait() {
	key := a.Name() + "stop"

	a.msg(Msg{
		Text:    "Stopping",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	a.wg.Wait()

	a.msg(Msg{
		Text:    "Stopped",
		Loading: util.BoolPointer(false),
		Success: util.BoolPointer(true),
		Key:     &key,
	})
}

func (a *Golang) msg(m Msg) {
	if m.Loading != nil {
		if *m.Loading {
			a.success = nil
		} else {
			a.success = m.Success
		}
	}

	m.Time = time.Now()
	m.App = a.app
	a.msgs <- m
}

func (a *Golang) watch(ctx context.Context, runCancel context.CancelFunc) {
	a.wg.Add(1)
	defer a.wg.Done()

	// Create new watcher
	watcher, err := util.Watch(a.dir, a.exts)
	if err != nil {
		a.msg(Msg{
			Text:    fmt.Sprintf("could not watch for changes: %s", err.Error()),
			Success: util.BoolPointer(false),
		})
	}
	defer watcher.Close()

	// Create new rate limiter
	rl := util.NewRateLimiter(time.Second * 1)

	a.msg(Msg{
		Text: "watching for changes...",
	})

loop:
	for {
		select {
		case <-ctx.Done():
			break loop

		case event, ok := <-watcher.Events:
			if !ok {
				break loop
			}

			// Validate ext
			if !slices.Contains(a.exts, util.ExtNoDot(filepath.Ext(event.Name))) {
				continue
			}

			// Rate limit
			ok = rl.Check("")
			if !ok {
				continue
			}

			go func() {
				rl.Wait("")

				runCancel()
				a.runWg.Wait()

				var runCtx context.Context
				runCtx, runCancel = context.WithCancel(ctx)

				a.msg(Msg{
					Text: "file changed: " + strings.TrimPrefix(event.Name, a.dir),
				})

				a.build()
				a.run(runCtx)
			}()

		case err, ok := <-watcher.Errors:
			if !ok {
				break loop
			}

			a.msg(Msg{
				Text:    err.Error(),
				Success: util.BoolPointer(false),
			})
		}
	}

	a.msg(Msg{
		Text: "stopped watching for changes",
	})
}

func (a *Golang) run(ctx context.Context) {
	a.wg.Add(1)
	defer a.wg.Done()

	// Create cmd
	cmd := exec.Command("./tmp/app")
	cmd.Dir = a.dir

	// Start cmd
	out, err := util.Run(cmd)
	if err != nil {
		a.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
	}

	// Stop cmd on exit
	a.wg.Add(1)
	a.runWg.Add(1)
	go func() {
		defer a.wg.Done()
		defer a.runWg.Done()
		<-ctx.Done()

		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			cmd.Process.Kill() // If the process is not responding to the interrupt signal, kill it
		}
	}()

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			a.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			a.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				a.msg(Msg{
					Text: fmt.Sprintf("stopped"),
				})
			} else {
				a.msg(Msg{
					Text: fmt.Sprintf("stopped"),
				})
			}
		}
	}

	return
}

func (v *Golang) build() error {
	v.wg.Add(1)
	defer v.wg.Done()
	key := v.Name() + "build"

	v.msg(Msg{
		Text:    "building",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	// Run vite build
	cmd := exec.Command("go", "build", "-o", "./tmp/app", "-tags", "dev")
	cmd.Dir = v.dir
	out, err := util.Run(cmd)
	if err != nil {
		v.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return err
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
					Text:    "build successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     &key,
				})

				return nil
			}

			v.msg(Msg{
				Text:    fmt.Sprintf("build failed with exit code %d", out),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     &key,
			})

			return fmt.Errorf("build failed with exit code %d", line)
		}
	}

	return fmt.Errorf("build failed")
}
