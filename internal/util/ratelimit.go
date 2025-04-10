package util

import (
	"sync"
	"time"
)

type RateLimiter struct {
	ratelimits map[string]time.Time
	wait       time.Duration
	mu         sync.Mutex
}

func NewRateLimiter(wait time.Duration) RateLimiter {
	return RateLimiter{
		ratelimits: make(map[string]time.Time),
		wait:       wait,
		mu:         sync.Mutex{},
	}
}

func (r *RateLimiter) Check(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	last, ok := r.ratelimits[key]
	if !ok {
		r.ratelimits[key] = time.Now()
		return true
	}

	if last.Add(r.wait).Before(time.Now()) {
		r.ratelimits[key] = time.Now()
		return true
	}

	r.ratelimits[key] = time.Now()
	return false
}

func (r *RateLimiter) Wait(key string) {
	for {
		r.mu.Lock()

		last, ok := r.ratelimits[key]
		if !ok {
			break
		}
		if last.Add(r.wait).Before(time.Now()) {
			delete(r.ratelimits, key)
			break
		}

		w := r.wait
		r.mu.Unlock()

		time.Sleep(w)
	}

	r.mu.Unlock()
}
