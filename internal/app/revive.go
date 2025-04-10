package app

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spotdemo4/treli/internal/util"
)

type Revive struct {
	app     *App
	success *bool
	dir     string
	exts    []string
	msgs    chan Msg
	wg      *sync.WaitGroup
}

func NewRevive(dir string, msgs chan Msg) (*App, error) {
	// Check if buf is installed
	_, err := exec.LookPath("revive")
	if err != nil {
		return nil, fmt.Errorf("'revive' not found in PATH")
	}

	// Create wait group
	wg := sync.WaitGroup{}

	revive := Revive{
		dir:  dir,
		exts: []string{"go"},
		msgs: msgs,
		wg:   &wg,
	}

	app := App(&revive)
	revive.app = &app

	return &app, nil
}

func (a *Revive) Name() string {
	return "revive"
}

func (a *Revive) Color() string {
	return "#89dceb"
}

func (a *Revive) Success() *bool {
	return a.success
}

func (a *Revive) Start(ctx context.Context) {
	a.wg.Wait()

	go a.lint()
	go a.watch(ctx)
}

func (a *Revive) Wait() {
	a.msg(Msg{
		Text:    "Stopping",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("buf stop"),
	})

	a.wg.Wait()

	a.msg(Msg{
		Text:    "Stopped",
		Loading: util.BoolPointer(false),
		Success: util.BoolPointer(true),
		Key:     util.StringPointer("buf stop"),
	})
}

func (a *Revive) msg(m Msg) {
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

func (a *Revive) watch(ctx context.Context) {
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

				a.msg(Msg{
					Text: "file changed: " + strings.TrimPrefix(event.Name, a.dir),
				})
				a.lint()
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

func (a *Revive) lint() (bool, error) {
	a.wg.Add(1)
	defer a.wg.Done()
	key := a.Name() + "lint"

	a.msg(Msg{
		Text:    "linting",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	// Run revive
	cmd := exec.Command("revive", "-config", "revive.toml", "-set_exit_status", "./...")
	cmd.Dir = a.dir
	out, err := util.Run(cmd)
	if err != nil {
		a.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return false, err
	}

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
					Text:    "lint successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     &key,
				})

				return true, nil
			}

			a.msg(Msg{
				Text:    fmt.Sprintf("lint failed, exit code %d", out),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     &key,
			})

			return false, fmt.Errorf("lint failed, exit code %d", line)
		}
	}

	return false, fmt.Errorf("lint failed")
}
