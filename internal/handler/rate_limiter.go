package handler

import (
	"sync"
	"time"
)

// slidingWindowLimiter allows at most limit events per window for each key.
type slidingWindowLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	events map[int][]time.Time
	now    func() time.Time
}

func newSlidingWindowLimiter(limit int, window time.Duration) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		limit:  limit,
		window: window,
		events: make(map[int][]time.Time),
		now:    time.Now,
	}
}

// Allow reports whether another event is permitted for key and, if so,
// records it.
func (l *slidingWindowLimiter) Allow(key int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)
	kept := l.events[key][:0]
	for _, t := range l.events[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.events[key] = kept
		return false
	}
	l.events[key] = append(kept, now)
	return true
}
