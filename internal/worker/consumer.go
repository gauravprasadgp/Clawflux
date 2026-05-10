package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type Consumer struct {
	logger          *slog.Logger
	queue           domain.JobQueue
	handlers        map[domain.JobType]func(context.Context, domain.Job) error
	maxAttempts     int
	retryBackoff    time.Duration
	leaseTimeout    time.Duration
	lastMaintenance time.Time
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
		leaseTimeout: 10 * time.Minute,
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

		c.runMaintenance(ctx)

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
			if err := c.deadLetterJob(ctx, job, "no handler registered"); err != nil {
				c.logger.Error("dead-letter failed for unknown job", "job_id", job.ID, "job_type", job.Type, "error", err)
			}
			if err := c.ackJob(ctx, job); err != nil {
				c.logger.Error("job ack failed", "job_id", job.ID, "job_type", job.Type, "error", err)
			}
			continue
		}
		if err := handler(ctx, job); err != nil {
			c.logger.Error("job failed", "job_id", job.ID, "job_type", job.Type, "error", err)
			if retryErr := c.retryJob(ctx, job, err); retryErr != nil {
				c.logger.Error("job retry failed", "job_id", job.ID, "job_type", job.Type, "error", retryErr)
			}
			continue
		}
		if err := c.ackJob(ctx, job); err != nil {
			c.logger.Error("job ack failed", "job_id", job.ID, "job_type", job.Type, "error", err)
		}
	}
}

func (c *Consumer) retryJob(ctx context.Context, job domain.Job, cause error) error {
	leasedJob := job
	job.Attempts++
	job.LastError = cause.Error()

	if job.Attempts >= c.maxAttempts {
		c.logger.Warn("job exhausted retries", "job_id", job.ID, "job_type", job.Type, "attempts", job.Attempts)
		if err := c.deadLetterJob(ctx, job, cause.Error()); err != nil {
			return err
		}
		return c.ackJob(ctx, leasedJob)
	}

	delay := c.retryDelay(job.Attempts)
	if delayed, ok := c.queue.(domain.JobScheduler); ok {
		if err := delayed.EnqueueAfter(ctx, job, delay); err != nil {
			return err
		}
		if err := c.ackJob(ctx, leasedJob); err != nil {
			return err
		}
		c.logger.Info("job scheduled for retry", "job_id", job.ID, "job_type", job.Type, "attempts", job.Attempts, "retry_in", delay.String())
		return nil
	}

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
	if err := c.ackJob(ctx, leasedJob); err != nil {
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

func (c *Consumer) ackJob(ctx context.Context, job domain.Job) error {
	acker, ok := c.queue.(domain.JobAcker)
	if !ok {
		return nil
	}
	return acker.Ack(ctx, job)
}

func (c *Consumer) deadLetterJob(ctx context.Context, job domain.Job, reason string) error {
	deadLetters, ok := c.queue.(domain.DeadLetterQueue)
	if !ok {
		return nil
	}
	return deadLetters.DeadLetter(ctx, domain.DeadLetterJob{
		ID:           idgen.NewUUID(),
		Job:          job,
		Reason:       reason,
		FailedAt:     time.Now().UTC(),
		FinalAttempt: job.Attempts,
	})
}

func (c *Consumer) runMaintenance(ctx context.Context) {
	now := time.Now()
	if !c.lastMaintenance.IsZero() && now.Sub(c.lastMaintenance) < time.Second {
		return
	}
	c.lastMaintenance = now

	if scheduled, ok := c.queue.(domain.JobScheduler); ok {
		promoted, err := scheduled.PromoteDue(ctx, 100)
		if err != nil {
			c.logger.Warn("delayed job promotion failed", "error", err)
		} else if promoted > 0 {
			c.logger.Info("delayed jobs promoted", "count", promoted)
		}
	}
	if reclaimer, ok := c.queue.(domain.JobLeaseReclaimer); ok {
		reclaimed, err := reclaimer.ReclaimStale(ctx, c.leaseTimeout, 100)
		if err != nil {
			c.logger.Warn("stale job reclaim failed", "error", err)
		} else if reclaimed > 0 {
			c.logger.Warn("stale jobs reclaimed", "count", reclaimed)
		}
	}
}
