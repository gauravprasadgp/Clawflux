package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type testQueue struct {
	mu        sync.Mutex
	jobs      []domain.Job
	enqueued  []domain.Job
	onEnqueue func()
}

func (q *testQueue) Enqueue(_ context.Context, job domain.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueued = append(q.enqueued, job)
	if q.onEnqueue != nil {
		q.onEnqueue()
	}
	return nil
}

func (q *testQueue) Dequeue(ctx context.Context) (domain.Job, error) {
	q.mu.Lock()
	if len(q.jobs) > 0 {
		job := q.jobs[0]
		q.jobs = q.jobs[1:]
		q.mu.Unlock()
		return job, nil
	}
	q.mu.Unlock()

	<-ctx.Done()
	return domain.Job{}, ctx.Err()
}

func TestConsumerRequeuesFailedJobs(t *testing.T) {
	var cancel context.CancelFunc
	queue := &testQueue{
		jobs: []domain.Job{{
			ID:       "job_1",
			Type:     domain.JobTypeDeploymentCreate,
			Attempts: 0,
		}},
		onEnqueue: func() {
			cancel()
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewConsumer(logger, queue, 3, time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	consumer.Register(domain.JobTypeDeploymentCreate, func(context.Context, domain.Job) error {
		return errors.New("boom")
	})

	err := consumer.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}

	if len(queue.enqueued) != 1 {
		t.Fatalf("expected 1 requeued job, got %d", len(queue.enqueued))
	}
	if queue.enqueued[0].Attempts != 1 {
		t.Fatalf("expected requeued attempts to be 1, got %d", queue.enqueued[0].Attempts)
	}
}

func TestConsumerStopsAfterMaxAttempts(t *testing.T) {
	queue := &testQueue{
		jobs: []domain.Job{{
			ID:       "job_2",
			Type:     domain.JobTypeDeploymentCreate,
			Attempts: 2,
		}},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewConsumer(logger, queue, 3, time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	consumer.Register(domain.JobTypeDeploymentCreate, func(context.Context, domain.Job) error {
		cancel()
		return errors.New("boom")
	})

	err := consumer.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}

	if len(queue.enqueued) != 0 {
		t.Fatalf("expected no requeued jobs, got %d", len(queue.enqueued))
	}
}
