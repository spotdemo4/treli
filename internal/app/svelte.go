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

type Svelte struct {
	app     *App
	success *bool
	dir     string
	exts    []string
	msgs    chan Msg
	wg      *sync.WaitGroup
}

func NewSvelte(dir string, msgs chan Msg) (*App, error) {
	// Check if buf is installed
	_, err := exec.LookPath("npx")
	if err != nil {
		return nil, fmt.Errorf("'npx' not found in PATH")
	}

	// Create wait group
	wg := sync.WaitGroup{}

	svelte := Svelte{
		dir:  dir,
		exts: []string{"svelte"},
		msgs: msgs,
		wg:   &wg,
	}

	app := App(&svelte)
	svelte.app = &app

	return &app, nil
}

func (s *Svelte) Name() string {
	return "svelte"
}

func (s *Svelte) Color() string {
	return "#fab387"
}

func (s *Svelte) Success() *bool {
	return s.success
}

func (s *Svelte) Start(ctx context.Context) {
	s.wg.Wait()

	go s.check()
	go s.watch(ctx)
}

func (s *Svelte) Wait() {
	s.msg(Msg{
		Text:    "stopping",
		Loading: util.BoolPointer(true),
		Key:     util.StringPointer("svelte stop"),
	})

	s.wg.Wait()

	s.msg(Msg{
		Text:    "stopped",
		Loading: util.BoolPointer(false),
		Success: util.BoolPointer(true),
		Key:     util.StringPointer("svelte stop"),
	})
}

func (s *Svelte) msg(m Msg) {
	if m.Loading != nil {
		if *m.Loading {
			s.success = nil
		} else {
			s.success = m.Success
		}
	}
	m.Time = time.Now()
	m.App = s.app

	s.msgs <- m
}

func (s *Svelte) watch(ctx context.Context) {
	s.wg.Add(1)
	defer s.wg.Done()

	// Create new watcher
	watcher, err := util.Watch(s.dir, s.exts)
	if err != nil {
		s.msg(Msg{
			Text:    fmt.Sprintf("could not watch for changes: %s", err.Error()),
			Success: util.BoolPointer(false),
		})
	}
	defer watcher.Close()

	// Create new rate limit
	rl := util.NewRateLimiter(time.Second * 1)

	s.msg(Msg{
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
			if !slices.Contains(s.exts, util.ExtNoDot(filepath.Ext(event.Name))) {
				continue
			}

			// Rate limit
			ok = rl.Check("")
			if !ok {
				continue
			}

			go func() {
				rl.Wait("")

				s.msg(Msg{
					Text: "file changed: " + strings.TrimPrefix(event.Name, s.dir),
				})
				s.check()
			}()

		case err, ok := <-watcher.Errors:
			if !ok {
				break loop
			}

			s.msg(Msg{
				Text:    err.Error(),
				Success: util.BoolPointer(false),
			})
		}
	}

	s.msg(Msg{
		Text: "stopped watching for changes",
	})
}

func (s *Svelte) check() error {
	s.wg.Add(1)
	defer s.wg.Done()
	key := s.Name() + "check"

	s.msg(Msg{
		Text:    "checking",
		Loading: util.BoolPointer(true),
		Key:     &key,
	})

	// Run svelte-check
	cmd := exec.Command("npx", "svelte-check")
	cmd.Dir = s.dir
	out, err := util.Run(cmd)
	if err != nil {
		s.msg(Msg{
			Text:    err.Error(),
			Success: util.BoolPointer(false),
		})
		return err
	}

	// Watch for output
	for line := range out {
		switch line := line.(type) {
		case util.Stdout:
			s.msg(Msg{
				Text: string(line),
			})

		case util.Stderr:
			s.msg(Msg{
				Text:    string(line),
				Success: util.BoolPointer(false),
			})

		case util.ExitCode:
			if line == 0 {
				s.msg(Msg{
					Text:    "check successful",
					Success: util.BoolPointer(true),
					Loading: util.BoolPointer(false),
					Key:     &key,
				})

				return nil
			}

			s.msg(Msg{
				Text:    fmt.Sprintf("check failed"),
				Success: util.BoolPointer(false),
				Loading: util.BoolPointer(false),
				Key:     &key,
			})
		}
	}

	return fmt.Errorf("check failed")
}
