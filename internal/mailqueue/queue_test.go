package mailqueue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEnqueuePreservesEveryMessage(t *testing.T) {
	q := New()
	q.windowLimit = 1000
	q.dailyLimit = 1000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	got := make(map[string]int)
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run(ctx, func(_ context.Context, to string) error {
			mu.Lock()
			got[to]++
			mu.Unlock()
			return nil
		})
	}()

	const total = 200
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := q.Enqueue(context.Background(), fmt.Sprintf("user%d@example.com", i)); err != nil {
				t.Errorf("enqueue: %v", err)
			}
		}(i)
	}
	wg.Wait()
	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == total
	})
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < total; i++ {
		addr := fmt.Sprintf("user%d@example.com", i)
		if got[addr] != 1 {
			t.Fatalf("%s sent %d times", addr, got[addr])
		}
	}
}

func TestWorkerKeepsOrderAndRetries(t *testing.T) {
	start := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	clock := newManualClock(start)
	q := New()
	q.now = clock.Now
	q.sleep = clock.Sleep
	q.windowLimit = 10
	q.dailyLimit = 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var got []string
	attempts := make(map[string]int)
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run(ctx, func(_ context.Context, to string) error {
			mu.Lock()
			attempts[to]++
			fail := to == "b@example.com" && attempts[to] == 1
			mu.Unlock()
			if fail {
				return errors.New("smtp down")
			}
			mu.Lock()
			got = append(got, to)
			mu.Unlock()
			return nil
		})
	}()

	for _, to := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if err := q.Enqueue(context.Background(), to); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == 3
	})
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if got[0] != "a@example.com" || got[1] != "b@example.com" || got[2] != "c@example.com" {
		t.Fatalf("order = %v", got)
	}
	if attempts["b@example.com"] != 2 {
		t.Fatalf("retry attempts = %d", attempts["b@example.com"])
	}
	if len(q.sent) != 3 {
		t.Fatalf("recorded sends = %d", len(q.sent))
	}
}

func TestWindowAndDailyLimits(t *testing.T) {
	start := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	q := New()
	for i := 0; i < windowLimit; i++ {
		q.markSent(start)
	}
	if got := q.delay(start); got != window {
		t.Fatalf("window delay = %s", got)
	}
	if got := q.delay(start.Add(window)); got != 0 {
		t.Fatalf("delay after window = %s", got)
	}

	q.sent = nil
	for i := 0; i < dailyLimit; i++ {
		q.markSent(start)
	}
	if got := q.delay(start); got != 12*time.Hour {
		t.Fatalf("daily delay = %s", got)
	}
	if got := q.delay(start.Add(12 * time.Hour)); got != 0 {
		t.Fatalf("delay next day = %s", got)
	}
}

func TestWorkerReturnsInFlightMail(t *testing.T) {
	q := New()
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run(ctx, func(sendCtx context.Context, _ string) error {
			close(started)
			<-sendCtx.Done()
			return sendCtx.Err()
		})
	}()

	if err := q.Enqueue(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	<-started
	cancel()
	<-done
	if q.Len() != 1 {
		t.Fatalf("pending = %d", q.Len())
	}
}

type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func newManualClock(now time.Time) *manualClock {
	return &manualClock{now: now}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Sleep(_ context.Context, delay time.Duration) error {
	c.mu.Lock()
	c.now = c.now.Add(delay)
	c.mu.Unlock()
	return nil
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met")
}
