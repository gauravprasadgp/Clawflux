package memory

import (
	"context"
	"sync"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type Client struct {
	mu         sync.Mutex
	ready      []domain.Job
	processing []domain.Job
	delayed    []domain.Job
	dead       []domain.DeadLetterJob
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Enqueue(_ context.Context, job domain.Job) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	job.LeasedAt = time.Time{}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = time.Now().UTC()
	}
	c.ready = append(c.ready, job)
	return nil
}

func (c *Client) Dequeue(ctx context.Context) (domain.Job, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		c.mu.Lock()
		c.promoteDueLocked(100)
		if len(c.ready) > 0 {
			job := c.ready[0]
			c.ready = c.ready[1:]
			job.LeasedAt = time.Now().UTC()
			c.processing = append(c.processing, job)
			c.mu.Unlock()
			return job, nil
		}
		c.mu.Unlock()

		select {
		case <-ctx.Done():
			return domain.Job{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Client) Ack(_ context.Context, job domain.Job) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processing = removeJob(c.processing, job)
	return nil
}

func (c *Client) EnqueueAfter(_ context.Context, job domain.Job, delay time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	job.LeasedAt = time.Time{}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.AvailableAt = time.Now().UTC().Add(delay)
	c.delayed = append(c.delayed, job)
	return nil
}

func (c *Client) PromoteDue(_ context.Context, limit int) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.promoteDueLocked(limit), nil
}

func (c *Client) ReclaimStale(_ context.Context, staleAfter time.Duration, limit int) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	if staleAfter <= 0 {
		staleAfter = 10 * time.Minute
	}
	cutoff := time.Now().UTC().Add(-staleAfter)
	reclaimed := 0
	kept := c.processing[:0]
	for _, job := range c.processing {
		if reclaimed >= limit || (!job.LeasedAt.IsZero() && job.LeasedAt.After(cutoff)) {
			kept = append(kept, job)
			continue
		}
		job.LeasedAt = time.Time{}
		job.AvailableAt = time.Now().UTC()
		c.ready = append(c.ready, job)
		reclaimed++
	}
	c.processing = kept
	return reclaimed, nil
}

func (c *Client) DeadLetter(_ context.Context, item domain.DeadLetterJob) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item.FailedAt.IsZero() {
		item.FailedAt = time.Now().UTC()
	}
	c.dead = append([]domain.DeadLetterJob{item}, c.dead...)
	return nil
}

func (c *Client) ListDeadLetters(_ context.Context, limit int) ([]domain.DeadLetterJob, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if limit <= 0 || limit > len(c.dead) {
		limit = len(c.dead)
	}
	out := make([]domain.DeadLetterJob, limit)
	copy(out, c.dead[:limit])
	return out, nil
}

func (c *Client) ReplayDeadLetter(_ context.Context, id string) (*domain.Job, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, item := range c.dead {
		if item.ID != id {
			continue
		}
		c.dead = append(c.dead[:i], c.dead[i+1:]...)
		job := item.Job
		job.Attempts = 0
		job.LastError = ""
		job.LeasedAt = time.Time{}
		job.AvailableAt = time.Now().UTC()
		c.ready = append(c.ready, job)
		return &job, nil
	}
	return nil, domain.ErrNotFound
}

func (c *Client) Stats(_ context.Context) (*domain.QueueStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return &domain.QueueStats{
		Ready:       len(c.ready),
		Processing:  len(c.processing),
		Delayed:     len(c.delayed),
		DeadLetters: len(c.dead),
	}, nil
}

func (c *Client) Check(context.Context) error {
	return nil
}

func (c *Client) promoteDueLocked(limit int) int {
	if limit <= 0 {
		limit = 100
	}
	now := time.Now().UTC()
	promoted := 0
	kept := c.delayed[:0]
	for _, job := range c.delayed {
		if promoted < limit && (job.AvailableAt.IsZero() || !job.AvailableAt.After(now)) {
			c.ready = append(c.ready, job)
			promoted++
			continue
		}
		kept = append(kept, job)
	}
	c.delayed = kept
	return promoted
}

func removeJob(items []domain.Job, job domain.Job) []domain.Job {
	for i, item := range items {
		if item.ID == job.ID && item.LeasedAt.Equal(job.LeasedAt) {
			return append(items[:i], items[i+1:]...)
		}
	}
	for i, item := range items {
		if item.ID == job.ID {
			return append(items[:i], items[i+1:]...)
		}
	}
	return items
}
