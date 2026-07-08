package handler

import (
	"testing"
	"time"
)

func TestSlidingWindowLimiter(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	l := newSlidingWindowLimiter(5, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 5; i++ {
		if !l.Allow(1) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if l.Allow(1) {
		t.Error("6th request within window should be denied")
	}

	// A different key has its own budget.
	if !l.Allow(2) {
		t.Error("different key should be allowed")
	}

	// Still denied just before the window slides.
	now = now.Add(59 * time.Second)
	if l.Allow(1) {
		t.Error("request at 59s should still be denied")
	}

	// Once the first events age out, requests are allowed again.
	now = now.Add(2 * time.Second)
	if !l.Allow(1) {
		t.Error("request after window elapsed should be allowed")
	}
}

func TestSlidingWindowLimiterPrunes(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	l := newSlidingWindowLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	l.Allow(1)
	l.Allow(1)
	now = now.Add(2 * time.Minute)
	if !l.Allow(1) {
		t.Fatal("expected allow after all events expired")
	}
	if got := len(l.events[1]); got != 1 {
		t.Errorf("events kept = %d, want 1 (expired entries pruned)", got)
	}
}
