package proc

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/boyter/gocodewalker"
	"github.com/fsnotify/fsnotify"
	"github.com/spotdemo4/treli/internal/util"
)

func Watch(ctx context.Context, path string, procs []*Proc) error {
	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Create walker
	fileListQueue := make(chan *gocodewalker.File, 100)
	fileWalker := gocodewalker.NewFileWalker(path, fileListQueue)

	// Create ratelimiter
	rl := util.NewRateLimiter(time.Second * 5)

	// Add extensions to walker
	exts := []string{}
	for _, app := range procs {
		for _, ext := range app.Exts {
			if !slices.Contains(exts, ext) {
				exts = append(exts, ext)
			}
		}
	}
	fileWalker.AllowListExtensions = exts

	// Walk path, add matching folders to watcher
	go fileWalker.Start()
	for f := range fileListQueue {
		dir := filepath.Dir(f.Location)

		if !slices.Contains(watcher.WatchList(), dir) {
			err := watcher.Add(dir)
			if err != nil {
				return err
			}
		}
	}

	// Start watching for changes
	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("could not watch for events")
			}

			for _, app := range procs {
				if !slices.Contains(app.Exts, extNoDot(filepath.Ext(event.Name))) {
					continue
				}

				// Rate limit calls
				ok := rl.Check(app.Name)
				if !ok {
					continue
				}

				go func() {
					// Wait for rate limiter to complete
					rl.Wait(app.Name)

					// Wait for proc to stop
					app.Stop()
					app.Wait()

					// Restart
					go app.Start(ctx)
				}()
			}
		}
	}
}

func extNoDot(s string) string {
	return strings.TrimPrefix(filepath.Ext(s), ".")
}
