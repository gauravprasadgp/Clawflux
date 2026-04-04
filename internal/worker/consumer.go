package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type Consumer struct {
	logger       *slog.Logger
	queue        domain.JobQueue
	handlers     map[domain.JobType]func(context.Context, domain.Job) error
	maxAttempts  int
	retryBackoff time.Duration
}

func NewConsumer(logger *slog.Logger, queue domain.JobQueue, maxAttempts int, retryBackoff time.Duration) *Consumer {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if retryBackoff <= 0 {
		retryBackoff = 2 * time.Second
	}
	return &Consumer{
		logger:       logger,
		queue:        queue,
		handlers:     map[domain.JobType]func(context.Context, domain.Job) error{},
		maxAttempts:  maxAttempts,
		retryBackoff: retryBackoff,
	}
}

func (c *Consumer) Register(jobType domain.JobType, handler func(context.Context, domain.Job) error) {
	c.handlers[jobType] = handler
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		job, err := c.queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return err
		}

		handler, ok := c.handlers[job.Type]
		if !ok {
			c.logger.Warn("no handler registered", "job_type", job.Type)
			continue
		}
		if err := handler(ctx, job); err != nil {
			c.logger.Error("job failed", "job_id", job.ID, "job_type", job.Type, "error", err)
			if retryErr := c.retryJob(ctx, job); retryErr != nil {
				c.logger.Error("job retry failed", "job_id", job.ID, "job_type", job.Type, "error", retryErr)
			}
		}
	}
}

func (c *Consumer) retryJob(ctx context.Context, job domain.Job) error {
	if job.Attempts+1 >= c.maxAttempts {
		c.logger.Warn("job exhausted retries", "job_id", job.ID, "job_type", job.Type, "attempts", job.Attempts+1)
		return nil
	}

	job.Attempts++
	delay := c.retryDelay(job.Attempts)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	if err := c.queue.Enqueue(ctx, job); err != nil {
		return err
	}
	c.logger.Info("job requeued", "job_id", job.ID, "job_type", job.Type, "attempts", job.Attempts, "retry_in", delay.String())
	return nil
}

func (c *Consumer) retryDelay(attempt int) time.Duration {
	delay := c.retryBackoff
	for i := 1; i < attempt; i++ {
		if delay >= 30*time.Second {
			return 30 * time.Second
		}
		delay *= 2
	}
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}
