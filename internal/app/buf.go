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
	app     *App
	success *bool
	dir     string
	exts    []string
	msgs    chan Msg
	wg      *sync.WaitGroup
}

func NewBuf(dir string, msgs chan Msg) (*App, error) {
	// Check if buf is installed
	_, err := exec.LookPath("buf")
	if err != nil {
		return nil, fmt.Errorf("'buf' not found in PATH")
	}

	// Create wait group
	wg := sync.WaitGroup{}

	buf := Buf{
		dir:  dir,
		exts: []string{"proto"},
		msgs: msgs,
		wg:   &wg,
	}

	app := App(&buf)
	buf.app = &app

	return &app, nil
}

func (b *Buf) Name() string {
	return "buf"
}

func (b *Buf) Color() string {
	return "#cba6f7"
}

func (b *Buf) Success() *bool {
	return b.success
}

func (b *Buf) Start(ctx context.Context) {
	b.wg.Wait()

	go b.lint()
	go b.watch(ctx)
}

func (b *Buf) Wait() {
	b.msg(Msg{
		Text:    "stopping",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("buf stop"),
	})

	b.wg.Wait()

	b.msg(Msg{
		Text:    "stopped",
		Loading: util.BoolPointer(false),
		Success: util.BoolPointer(true),
		Key:     util.StringPointer("buf stop"),
	})
}

func (b *Buf) msg(m Msg) {
	if m.Loading != nil {
		if *m.Loading {
			b.success = nil
		} else {
			b.success = m.Success
		}
	}

	m.Time = time.Now()
	m.App = b.app
	b.msgs <- m
}

func (b *Buf) watch(ctx context.Context) {
	b.wg.Add(1)
	defer b.wg.Done()

	// Create new watcher
	watcher, err := util.Watch(b.dir, b.exts)
	if err != nil {
		b.msg(Msg{
			Text:    fmt.Sprintf("could not watch for changes: %s", err.Error()),
			Success: util.BoolPointer(false),
		})
	}
	defer watcher.Close()

	// Create new rate limiter
	rl := util.NewRateLimiter(time.Second * 1)

	b.msg(Msg{
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
			if !slices.Contains(b.exts, util.ExtNoDot(filepath.Ext(event.Name))) {
				continue
			}

			// Rate limit
			ok = rl.Check("")
			if !ok {
				continue
			}

			go func() {
				rl.Wait("")

				b.msg(Msg{
					Text: "file changed: " + strings.TrimPrefix(event.Name, b.dir),
				})

				err = b.lint()
				if err != nil {
					return
				}

				b.generate()
			}()

		case err, ok := <-watcher.Errors:
			if !ok {
				break loop
			}

			b.msg(Msg{
				Text:    err.Error(),
				Success: util.BoolPointer(false),
			})
		}
	}

	b.msg(Msg{
		Text: "stopped watching for changes",
	})
}

func (b *Buf) lint() error {
	b.wg.Add(1)
	defer b.wg.Done()

	b.msg(Msg{
		Text:    "linting",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("buf lint"),
	})

	// Run buf lint
	cmd := exec.Command("buf", "lint")
	cmd.Dir = b.dir
	out, err := util.Run(cmd)
	if err != nil {
		b.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return err
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			b.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			b.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				b.msg(Msg{
					Text:    "lint successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     util.StringPointer("buf lint"),
				})

				return nil
			}

			b.msg(Msg{
				Text:    fmt.Sprintf("lint failed"),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     util.StringPointer("buf lint"),
			})
		}
	}

	return fmt.Errorf("lint failed")
}

func (b *Buf) generate() error {
	b.wg.Add(1)
	defer b.wg.Done()
	key := b.Name() + "gen"

	b.msg(Msg{
		Text:    "generating proto files",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	// Run buf gen
	cmd := exec.Command("buf", "generate")
	cmd.Dir = b.dir
	out, err := util.Run(cmd)
	if err != nil {
		b.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return err
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			b.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			b.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				b.msg(Msg{
					Text:    "generate successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     &key,
				})

				return nil
			}

			b.msg(Msg{
				Text:    fmt.Sprintf("generate failed"),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     &key,
			})
		}
	}

	return fmt.Errorf("generate failed")
}
