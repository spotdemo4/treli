package util

import (
	"sync"
)

type Broadcast struct {
	chans []chan int
	mu    sync.Mutex
}

func NewBroadcast() *Broadcast {
	return &Broadcast{
		chans: []chan int{},
		mu:    sync.Mutex{},
	}
}

func (b *Broadcast) Notify() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range b.chans {
		select {
		case c <- 1:
		default:
		}
		close(c)
	}
	b.chans = nil
}

func (b *Broadcast) Wait() chan int {
	b.mu.Lock()
	c := make(chan int)
	b.chans = append(b.chans, c)
	b.mu.Unlock()

	return c
}
