package app

import (
	"context"
	"sync"
	"time"
)

type Msg struct {
	Text    string
	Time    time.Time
	Key     *string
	Loading *bool
	Success *bool

	App *App
}

type App struct {
	Name    string
	Color   string
	Loading *bool
	Cancel  context.CancelFunc
	Wait    func()

	dir     string
	exts    []string
	msgChan chan Msg
	ctx     context.Context
	wg      *sync.WaitGroup
}
