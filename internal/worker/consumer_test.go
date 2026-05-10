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

type reliableTestQueue struct {
	testQueue
	delayed     []domain.Job
	delay       time.Duration
	acked       []domain.Job
	deadLetters []domain.DeadLetterJob
	onDelayed   func()
	onDead      func()
}

func (q *reliableTestQueue) EnqueueAfter(_ context.Context, job domain.Job, delay time.Duration) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.delayed = append(q.delayed, job)
	q.delay = delay
	if q.onDelayed != nil {
		q.onDelayed()
	}
	return nil
}

func (q *reliableTestQueue) PromoteDue(context.Context, int) (int, error) { return 0, nil }

func (q *reliableTestQueue) Ack(_ context.Context, job domain.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.acked = append(q.acked, job)
	return nil
}

func (q *reliableTestQueue) DeadLetter(_ context.Context, item domain.DeadLetterJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.deadLetters = append(q.deadLetters, item)
	if q.onDead != nil {
		q.onDead()
	}
	return nil
}

func (q *reliableTestQueue) ListDeadLetters(context.Context, int) ([]domain.DeadLetterJob, error) {
	return q.deadLetters, nil
}

func (q *reliableTestQueue) ReplayDeadLetter(context.Context, string) (*domain.Job, error) {
	return nil, domain.ErrNotFound
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

func TestConsumerSchedulesDelayedRetryAndAcksOriginalLease(t *testing.T) {
	leasedAt := time.Now().UTC()
	var cancel context.CancelFunc
	queue := &reliableTestQueue{
		testQueue: testQueue{
			jobs: []domain.Job{{
				ID:       "job_3",
				Type:     domain.JobTypeDeploymentCreate,
				Attempts: 0,
				LeasedAt: leasedAt,
			}},
		},
		onDelayed: func() {
			cancel()
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewConsumer(logger, queue, 3, time.Millisecond)

	ctx, stop := context.WithCancel(context.Background())
	cancel = stop
	consumer.Register(domain.JobTypeDeploymentCreate, func(context.Context, domain.Job) error {
		return errors.New("boom")
	})

	err := consumer.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}

	if len(queue.delayed) != 1 {
		t.Fatalf("expected 1 delayed retry, got %d", len(queue.delayed))
	}
	if queue.delayed[0].Attempts != 1 {
		t.Fatalf("expected delayed attempts to be 1, got %d", queue.delayed[0].Attempts)
	}
	if queue.delayed[0].LastError == "" {
		t.Fatalf("expected delayed job to include last error")
	}
	if len(queue.acked) != 1 {
		t.Fatalf("expected 1 acked lease, got %d", len(queue.acked))
	}
	if queue.acked[0].Attempts != 0 {
		t.Fatalf("expected acked lease to keep original attempts, got %d", queue.acked[0].Attempts)
	}
	if !queue.acked[0].LeasedAt.Equal(leasedAt) {
		t.Fatalf("expected acked lease timestamp to be preserved")
	}
}

func TestConsumerDeadLettersExhaustedReliableJobs(t *testing.T) {
	var cancel context.CancelFunc
	queue := &reliableTestQueue{
		testQueue: testQueue{
			jobs: []domain.Job{{
				ID:       "job_4",
				Type:     domain.JobTypeDeploymentCreate,
				Attempts: 2,
				LeasedAt: time.Now().UTC(),
			}},
		},
		onDead: func() {
			cancel()
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewConsumer(logger, queue, 3, time.Millisecond)

	ctx, stop := context.WithCancel(context.Background())
	cancel = stop
	consumer.Register(domain.JobTypeDeploymentCreate, func(context.Context, domain.Job) error {
		return errors.New("boom")
	})

	err := consumer.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}

	if len(queue.delayed) != 0 {
		t.Fatalf("expected no delayed retries, got %d", len(queue.delayed))
	}
	if len(queue.deadLetters) != 1 {
		t.Fatalf("expected 1 dead-letter job, got %d", len(queue.deadLetters))
	}
	if queue.deadLetters[0].FinalAttempt != 3 {
		t.Fatalf("expected final attempt 3, got %d", queue.deadLetters[0].FinalAttempt)
	}
	if len(queue.acked) != 1 {
		t.Fatalf("expected exhausted job lease to be acked")
	}
	if queue.acked[0].Attempts != 2 {
		t.Fatalf("expected acked exhausted lease to keep original attempts, got %d", queue.acked[0].Attempts)
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
