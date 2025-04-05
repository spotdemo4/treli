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

type Buf struct {
	App *App
}

func NewBuf(dir string, c chan Msg) (*App, error) {
	// Check if buf is installed
	_, err := exec.LookPath("buf")
	if err != nil {
		return nil, fmt.Errorf("'buf' not found in PATH")
	}

	// Create new context
	ctx, cancel := context.WithCancel(context.Background())

	// Create wait group
	wg := sync.WaitGroup{}

	app := App{
		Name:   "buf",
		Color:  "#89dceb",
		Cancel: cancel,
		Wait:   wg.Wait,

		dir:     dir,
		exts:    []string{"proto"},
		msgChan: c,
		ctx:     ctx,
		wg:      &wg,
	}
	proto := Buf{
		App: &app,
	}

	// Start watching
	go proto.watch()

	return &app, nil
}

func (p *Buf) msg(m Msg) {
	m.Time = time.Now()
	m.App = p.App
	p.App.msgChan <- m
}

func (p *Buf) watch() {
	p.App.wg.Add(1)
	defer p.App.wg.Done()

	// Create new watcher
	watcher, err := util.Watch(p.App.dir, p.App.exts)
	if err != nil {
		p.msg(Msg{
			Text:    fmt.Sprintf("could not watch for changes: %s", err.Error()),
			Success: util.BoolPointer(false),
		})
	}
	defer watcher.Close()

	// Create new rate limit map
	rateLimit := make(map[string]time.Time)

	p.lint()

	p.msg(Msg{
		Text: "watching for changes...",
	})

loop:
	for {
		select {
		case <-p.App.ctx.Done():
			break loop

		case event, ok := <-watcher.Events:
			if !ok {
				break loop
			}

			// Validate ext
			if !slices.Contains(p.App.exts, filepath.Ext(event.Name)) {
				continue
			}

			// Rate limit
			rl, ok := rateLimit[event.Name]
			if ok && time.Since(rl) < 1*time.Second {
				continue
			}
			rateLimit[event.Name] = time.Now()

			p.msg(Msg{
				Text: "file changed: " + strings.TrimPrefix(event.Name, p.App.dir),
			})

			ok, _ = p.lint()
			if !ok {
				continue
			}

			p.generate()

		case err, ok := <-watcher.Errors:
			if !ok {
				break loop
			}

			p.msg(Msg{
				Text:    err.Error(),
				Success: util.BoolPointer(false),
			})
		}
	}

	p.msg(Msg{
		Text: "stopped watching for changes",
	})
}

func (p *Buf) lint() (bool, error) {
	p.msg(Msg{
		Text:    "linting",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("buf lint"),
	})

	// Run buf lint
	cmd := exec.Command("buf", "lint")
	cmd.Dir = p.App.dir
	out, err := util.Run(cmd)
	if err != nil {
		p.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return false, err
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			p.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			p.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				p.msg(Msg{
					Text:    "lint successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     util.StringPointer("buf lint"),
				})

				return true, nil
			}

			p.msg(Msg{
				Text:    fmt.Sprintf("lint failed, exit code %d", out),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     util.StringPointer("buf lint"),
			})

			return false, fmt.Errorf("lint failed, exit code %d", line)
		}
	}

	return false, fmt.Errorf("lint failed")
}

func (p *Buf) generate() error {
	p.msg(Msg{
		Text:    "generating proto files",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("buf generate"),
	})

	// Run buf gen
	cmd := exec.Command("buf", "generate")
	cmd.Dir = p.App.dir
	out, err := util.Run(cmd)
	if err != nil {
		p.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return err
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			p.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			p.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				p.msg(Msg{
					Text:    "generate successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     util.StringPointer("buf generate"),
				})

				return nil
			}

			p.msg(Msg{
				Text:    fmt.Sprintf("generate failed with exit code %d", out),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     util.StringPointer("buf generate"),
			})

			return fmt.Errorf("generate failed with exit code %d", line)
		}
	}

	return fmt.Errorf("generate failed")
}
