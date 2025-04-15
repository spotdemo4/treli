package app

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

func Watch(ctx context.Context, dir string, apps []*App) error {
	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Create walker
	fileListQueue := make(chan *gocodewalker.File, 100)
	fileWalker := gocodewalker.NewFileWalker(dir, fileListQueue)

	// Create ratelimiter
	rl := util.NewRateLimiter(time.Second * 5)

	// Add extensions to walker
	exts := []string{}
	for _, app := range apps {
		for _, ext := range app.Exts {
			if !slices.Contains(exts, ext) {
				exts = append(exts, ext)
			}
		}
	}
	fileWalker.AllowListExtensions = exts

	// Walk dir, add matching folders to watcher
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

			for _, app := range apps {
				if !slices.Contains(app.Exts, extNoDot(filepath.Ext(event.Name))) {
					continue
				}

				// Rate limit calls
				ok := rl.Check(app.Name)
				if !ok {
					continue
				}

				go func() {
					rl.Wait(app.Name)

					app.Stop()
					app.Run(app.OnChange)
				}()
			}
		}
	}
}

func extNoDot(s string) string {
	return strings.TrimPrefix(filepath.Ext(s), ".")
}
