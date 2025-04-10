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

type ESLint struct {
	app     *App
	success *bool
	dir     string
	exts    []string
	msgs    chan Msg
	wg      *sync.WaitGroup
}

func NewESLint(dir string, msgs chan Msg) (*App, error) {
	// Check if buf is installed
	_, err := exec.LookPath("npx")
	if err != nil {
		return nil, fmt.Errorf("'npx' not found in PATH")
	}

	// Create wait group
	wg := sync.WaitGroup{}

	eslint := ESLint{
		dir:  dir,
		exts: []string{"js", "ts", "tsx", "jsx", "vue", "svelte", "mdx", "md"},
		msgs: msgs,
		wg:   &wg,
	}

	app := App(&eslint)
	eslint.app = &app

	return &app, nil
}

func (a *ESLint) Name() string {
	return "eslint"
}

func (a *ESLint) Color() string {
	return "#fab387"
}

func (a *ESLint) Success() *bool {
	return a.success
}

func (a *ESLint) Start(ctx context.Context) {
	a.wg.Wait()

	go a.lint()
	go a.watch(ctx)
}

func (a *ESLint) Wait() {
	a.msg(Msg{
		Text:    "stopping",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer(a.Name() + "stop"),
	})

	a.wg.Wait()

	a.msg(Msg{
		Text:    "stopped",
		Loading: util.BoolPointer(false),
		Success: util.BoolPointer(true),
		Key:     util.StringPointer(a.Name() + "stop"),
	})
}

func (a *ESLint) msg(m Msg) {
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

func (a *ESLint) watch(ctx context.Context) {
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

func (a *ESLint) lint() error {
	a.wg.Add(1)
	defer a.wg.Done()
	key := a.Name() + "lint"

	a.msg(Msg{
		Text:    "linting",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	// Run revive
	cmd := exec.Command("npx", "eslint", ".")
	cmd.Dir = a.dir
	out, err := util.Run(cmd)
	if err != nil {
		a.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return err
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

				return nil
			}

			a.msg(Msg{
				Text:    fmt.Sprintf("lint failed"),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     &key,
			})
		}
	}

	return fmt.Errorf("lint failed")
}
