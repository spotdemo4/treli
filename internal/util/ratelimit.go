package util

import (
	"sync"
	"time"
)

type RateLimiter struct {
	time time.Time
	wait time.Duration
	mu   sync.Mutex
}

func NewRateLimiter(wait time.Duration) *RateLimiter {
	return &RateLimiter{
		time: time.Now().Add(-wait),
		wait: wait,
		mu:   sync.Mutex{},
	}
}

func (r *RateLimiter) Check() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.time.Add(r.wait).Before(time.Now()) {
		r.time = time.Now()
		return true
	}

	r.time = time.Now()
	return false
}

func (r *RateLimiter) Wait() {
	for {
		r.mu.Lock()
		if r.time.Add(r.wait).Before(time.Now()) {
			break
		}
		r.mu.Unlock()

		time.Sleep(r.wait)
	}

	r.mu.Unlock()
}
