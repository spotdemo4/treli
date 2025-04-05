package util

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/boyter/gocodewalker"
	"github.com/fsnotify/fsnotify"
)

func ExtNoDot(s string) string {
	return strings.TrimPrefix(filepath.Ext(s), ".")
}

func Watch(dir string, exts []string) (*fsnotify.Watcher, error) {
	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Find files
	fileListQueue := make(chan *gocodewalker.File, 100)
	fileWalker := gocodewalker.NewFileWalker(dir, fileListQueue)
	fileWalker.AllowListExtensions = exts

	// Add files to watcher
	go fileWalker.Start()
	for f := range fileListQueue {
		dir := filepath.Dir(f.Location)
		if !slices.Contains(watcher.WatchList(), dir) {
			err := watcher.Add(dir)
			if err != nil {
				return nil, err
			}
		}
	}

	return watcher, nil
}
