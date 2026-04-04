package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type Consumer struct {
	logger   *slog.Logger
	queue    domain.JobQueue
	handlers map[domain.JobType]func(context.Context, domain.Job) error
}

func NewConsumer(logger *slog.Logger, queue domain.JobQueue) *Consumer {
	return &Consumer{
		logger:   logger,
		queue:    queue,
		handlers: map[domain.JobType]func(context.Context, domain.Job) error{},
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
		}
	}
}
