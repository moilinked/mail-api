// Package mailqueue 串行消化发信队列，并执行发送配额。
package mailqueue

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	window      = 100 * time.Second
	windowLimit = 150
	dailyLimit  = 4500
	retryDelay  = 5 * time.Second
)

// ErrClosed 表示队列已停止接收。
var ErrClosed = errors.New("mail queue is closed")

// Queue 保存待发邮件。只有一个消费协程会取出邮件，避免并发发送造成丢失。
type Queue struct {
	mu          sync.Mutex
	items       []string
	sent        []time.Time
	closed      bool
	wake        chan struct{}
	now         func() time.Time
	sleep       func(context.Context, time.Duration) error
	window      time.Duration
	windowLimit int
	dailyLimit  int
	retryDelay  time.Duration
}

// New 创建一个按 100 秒 150 封、每个自然日 4500 封限制的队列。
func New() *Queue {
	return &Queue{
		wake:        make(chan struct{}, 1),
		now:         time.Now,
		sleep:       sleepContext,
		window:      window,
		windowLimit: windowLimit,
		dailyLimit:  dailyLimit,
		retryDelay:  retryDelay,
	}
}

// Enqueue 把收件人放进队尾。
func (q *Queue) Enqueue(ctx context.Context, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	q.items = append(q.items, to)
	q.signalLocked()
	return nil
}

// Close 拒绝新的入队请求。已经在队列里的邮件仍可由消费协程处理。
func (q *Queue) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
}

// Len 返回尚未取出的邮件数量。
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// Run 由唯一的消费协程调用，直到 ctx 取消。取消时会把正在发送的邮件放回队头。
func (q *Queue) Run(ctx context.Context, send func(context.Context, string) error) {
	for {
		to, ok := q.pop(ctx)
		if !ok {
			slog.Info("mail queue stopped", "pending", q.Len())
			return
		}
		if err := q.deliver(ctx, to, send); err != nil {
			q.pushFront(to)
			slog.Info("mail queue stopped", "pending", q.Len())
			return
		}
	}
}

func (q *Queue) deliver(ctx context.Context, to string, send func(context.Context, string) error) error {
	for {
		if err := q.waitTurn(ctx); err != nil {
			return err
		}
		err := send(ctx, to)
		if err == nil {
			q.markSent(q.now())
			slog.Info("email sent", "to", to)
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slog.Error("send queued email", "err", err, "to", to)
		if err := q.sleep(ctx, q.retryDelay); err != nil {
			return err
		}
	}
}

func (q *Queue) waitTurn(ctx context.Context) error {
	for {
		delay := q.delay(q.now())
		if delay <= 0 {
			return nil
		}
		if err := q.sleep(ctx, delay); err != nil {
			return err
		}
	}
}

func (q *Queue) pop(ctx context.Context) (string, bool) {
	for {
		if err := ctx.Err(); err != nil {
			return "", false
		}

		q.mu.Lock()
		if len(q.items) > 0 {
			to := q.items[0]
			q.items[0] = ""
			q.items = q.items[1:]
			q.mu.Unlock()
			return to, true
		}
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return "", false
		case <-q.wake:
		}
	}
}

func (q *Queue) pushFront(to string) {
	q.mu.Lock()
	q.items = append([]string{to}, q.items...)
	q.signalLocked()
	q.mu.Unlock()
}

func (q *Queue) markSent(at time.Time) {
	q.mu.Lock()
	q.sent = append(q.sent, at)
	q.mu.Unlock()
}

func (q *Queue) delay(now time.Time) time.Duration {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prune(now)

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowStart := now.Add(-q.window)
	dayCount := 0
	windowCount := 0
	var oldestWindow time.Time
	for _, sentAt := range q.sent {
		if !sentAt.Before(dayStart) {
			dayCount++
		}
		if sentAt.After(windowStart) {
			windowCount++
			if oldestWindow.IsZero() || sentAt.Before(oldestWindow) {
				oldestWindow = sentAt
			}
		}
	}

	var wait time.Duration
	if dayCount >= q.dailyLimit {
		wait = dayStart.Add(24 * time.Hour).Sub(now)
	}
	if windowCount >= q.windowLimit {
		windowWait := oldestWindow.Add(q.window).Sub(now)
		if windowWait > wait {
			wait = windowWait
		}
	}
	if wait < 0 {
		return 0
	}
	return wait
}

func (q *Queue) prune(now time.Time) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := dayStart
	if windowStart := now.Add(-q.window); windowStart.Before(cutoff) {
		cutoff = windowStart
	}

	kept := q.sent[:0]
	for _, sentAt := range q.sent {
		if !sentAt.Before(cutoff) {
			kept = append(kept, sentAt)
		}
	}
	q.sent = kept
}

func (q *Queue) signalLocked() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
