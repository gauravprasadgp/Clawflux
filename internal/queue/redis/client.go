package redis

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

const (
	defaultPoolSize    = 8
	defaultTimeout     = 5 * time.Second
	defaultIdleTimeout = 60 * time.Second
)

// conn wraps a single TCP connection to Redis.
type conn struct {
	net.Conn
	reader    *bufio.Reader
	createdAt time.Time
}

// pool is a very small, mutex-protected connection pool.
type pool struct {
	addr        string
	password    string
	mu          sync.Mutex
	idle        []*conn
	size        int
	maxSize     int
	idleTimeout time.Duration
	dialTimeout time.Duration
}

func newPool(addr, password string, maxSize int) *pool {
	return &pool{
		addr:        addr,
		password:    password,
		maxSize:     maxSize,
		idleTimeout: defaultIdleTimeout,
		dialTimeout: defaultTimeout,
	}
}

func (p *pool) get(ctx context.Context) (*conn, error) {
	p.mu.Lock()
	// Return an idle connection if available and not stale.
	for len(p.idle) > 0 {
		c := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		p.mu.Unlock()
		if time.Since(c.createdAt) < p.idleTimeout {
			return c, nil
		}
		_ = c.Close()
		p.mu.Lock()
		p.size--
	}
	if p.size >= p.maxSize {
		p.mu.Unlock()
		return nil, fmt.Errorf("redis pool exhausted (max %d)", p.maxSize)
	}
	p.size++
	p.mu.Unlock()

	return p.dial(ctx)
}

func (p *pool) put(c *conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.idle = append(p.idle, c)
}

func (p *pool) discard(c *conn) {
	_ = c.Close()
	p.mu.Lock()
	p.size--
	p.mu.Unlock()
}

func (p *pool) dial(ctx context.Context) (*conn, error) {
	var d net.Dialer
	nc, err := d.DialContext(ctx, "tcp", p.addr)
	if err != nil {
		return nil, fmt.Errorf("redis dial %s: %w", p.addr, err)
	}
	_ = nc.SetDeadline(time.Now().Add(p.dialTimeout))
	c := &conn{Conn: nc, reader: bufio.NewReader(nc), createdAt: time.Now()}

	if p.password != "" {
		if err := sendCommand(c, "AUTH", p.password); err != nil {
			_ = nc.Close()
			return nil, fmt.Errorf("redis AUTH: %w", err)
		}
	}
	// Reset deadline — callers set their own per-operation deadline.
	_ = nc.SetDeadline(time.Time{})
	return c, nil
}

// sendCommand writes a simple inline command and reads the first response line.
func sendCommand(c *conn, args ...string) error {
	if _, err := c.Write([]byte(buildArray(args...))); err != nil {
		return err
	}
	line, err := readLine(c.reader)
	if err != nil {
		return err
	}
	if strings.HasPrefix(line, "-") {
		return fmt.Errorf("redis error: %s", strings.TrimPrefix(line, "-"))
	}
	return nil
}

// Client is a production Redis client backed by a small connection pool.
type Client struct {
	pool    *pool
	queue   string
	timeout time.Duration
}

func NewClient(addr, queue string) *Client {
	return NewClientWithPassword(addr, "", queue)
}

func NewClientWithPassword(addr, password, queue string) *Client {
	return &Client{
		pool:    newPool(addr, password, defaultPoolSize),
		queue:   queue,
		timeout: defaultTimeout,
	}
}

func (c *Client) Enqueue(ctx context.Context, job domain.Job) error {
	job.LeasedAt = time.Time{}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = time.Now().UTC()
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	rc, err := c.pool.get(ctx)
	if err != nil {
		return err
	}
	_ = rc.SetDeadline(time.Now().Add(c.timeout))
	defer func() { _ = rc.SetDeadline(time.Time{}) }()

	cmd := buildArray("RPUSH", c.queue, string(payload))
	if _, err := rc.Write([]byte(cmd)); err != nil {
		c.pool.discard(rc)
		return fmt.Errorf("redis RPUSH write: %w", err)
	}
	line, err := readLine(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return fmt.Errorf("redis RPUSH read: %w", err)
	}
	if strings.HasPrefix(line, "-") {
		c.pool.discard(rc)
		return fmt.Errorf("redis error: %s", strings.TrimPrefix(line, "-"))
	}
	c.pool.put(rc)
	return nil
}

func (c *Client) Dequeue(ctx context.Context) (domain.Job, error) {
	rc, err := c.pool.get(ctx)
	if err != nil {
		return domain.Job{}, err
	}
	// BRPOP blocks for up to 5 seconds; give it a 6-second deadline.
	_ = rc.SetDeadline(time.Now().Add(6 * time.Second))

	if _, err := rc.Write([]byte(buildArray("BRPOPLPUSH", c.queue, c.processingQueue(), "5"))); err != nil {
		c.pool.discard(rc)
		return domain.Job{}, fmt.Errorf("redis BRPOPLPUSH write: %w", err)
	}

	payload, err := readBulkString(rc.reader)
	if err != nil {
		if errors.Is(err, errNilBulkString) {
			_ = rc.SetDeadline(time.Time{})
			c.pool.put(rc)
			return domain.Job{}, context.DeadlineExceeded
		}
		// Timeout from BRPOPLPUSH — treat as "nothing to do" rather than fatal.
		c.pool.discard(rc)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return domain.Job{}, context.DeadlineExceeded
		}
		return domain.Job{}, fmt.Errorf("redis BRPOPLPUSH read: %w", err)
	}

	var job domain.Job
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		_ = rc.SetDeadline(time.Time{})
		c.pool.put(rc)
		return domain.Job{}, fmt.Errorf("unmarshal job: %w", err)
	}
	job.LeasedAt = time.Now().UTC()

	leasedPayload, err := json.Marshal(job)
	if err != nil {
		c.pool.discard(rc)
		return domain.Job{}, fmt.Errorf("marshal leased job: %w", err)
	}
	if string(leasedPayload) != payload {
		if err := writeExpectInteger(rc, "LPUSH", c.processingQueue(), string(leasedPayload)); err != nil {
			c.pool.discard(rc)
			return domain.Job{}, fmt.Errorf("redis LPUSH leased job: %w", err)
		}
		if err := writeExpectInteger(rc, "LREM", c.processingQueue(), "1", payload); err != nil {
			c.pool.discard(rc)
			return domain.Job{}, fmt.Errorf("redis LREM original leased job: %w", err)
		}
	}

	_ = rc.SetDeadline(time.Time{})
	c.pool.put(rc)
	return job, nil
}

func (c *Client) Ack(ctx context.Context, job domain.Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal ack job: %w", err)
	}
	return c.withIntegerCommand(ctx, "LREM", c.processingQueue(), "1", string(payload))
}

func (c *Client) EnqueueAfter(ctx context.Context, job domain.Job, delay time.Duration) error {
	job.LeasedAt = time.Time{}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.AvailableAt = time.Now().UTC().Add(delay)
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal delayed job: %w", err)
	}
	score := strconv.FormatInt(job.AvailableAt.UnixMilli(), 10)
	return c.withIntegerCommand(ctx, "ZADD", c.delayedQueue(), score, string(payload))
}

func (c *Client) PromoteDue(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	rc, err := c.pool.get(ctx)
	if err != nil {
		return 0, err
	}
	_ = rc.SetDeadline(time.Now().Add(c.timeout))
	defer func() { _ = rc.SetDeadline(time.Time{}) }()

	now := strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)
	if _, err := rc.Write([]byte(buildArray("ZRANGEBYSCORE", c.delayedQueue(), "-inf", now, "LIMIT", "0", strconv.Itoa(limit)))); err != nil {
		c.pool.discard(rc)
		return 0, fmt.Errorf("redis ZRANGEBYSCORE write: %w", err)
	}
	payloads, err := readBulkStringArray(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return 0, fmt.Errorf("redis ZRANGEBYSCORE read: %w", err)
	}

	promoted := 0
	for _, payload := range payloads {
		removed, err := writeIntegerReply(rc, "ZREM", c.delayedQueue(), payload)
		if err != nil {
			c.pool.discard(rc)
			return promoted, fmt.Errorf("redis ZREM delayed job: %w", err)
		}
		if removed == 0 {
			continue
		}
		if err := writeExpectInteger(rc, "RPUSH", c.queue, payload); err != nil {
			c.pool.discard(rc)
			return promoted, fmt.Errorf("redis RPUSH promoted job: %w", err)
		}
		promoted++
	}
	c.pool.put(rc)
	return promoted, nil
}

func (c *Client) ReclaimStale(ctx context.Context, staleAfter time.Duration, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	if staleAfter <= 0 {
		staleAfter = 2 * time.Minute
	}
	rc, err := c.pool.get(ctx)
	if err != nil {
		return 0, err
	}
	_ = rc.SetDeadline(time.Now().Add(c.timeout))
	defer func() { _ = rc.SetDeadline(time.Time{}) }()

	if _, err := rc.Write([]byte(buildArray("LRANGE", c.processingQueue(), "0", strconv.Itoa(limit-1)))); err != nil {
		c.pool.discard(rc)
		return 0, fmt.Errorf("redis LRANGE processing write: %w", err)
	}
	payloads, err := readBulkStringArray(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return 0, fmt.Errorf("redis LRANGE processing read: %w", err)
	}

	cutoff := time.Now().UTC().Add(-staleAfter)
	reclaimed := 0
	for _, payload := range payloads {
		var job domain.Job
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			continue
		}
		if !job.LeasedAt.IsZero() && job.LeasedAt.After(cutoff) {
			continue
		}
		removed, err := writeIntegerReply(rc, "LREM", c.processingQueue(), "1", payload)
		if err != nil {
			c.pool.discard(rc)
			return reclaimed, fmt.Errorf("redis LREM stale job: %w", err)
		}
		if removed == 0 {
			continue
		}
		job.LeasedAt = time.Time{}
		job.AvailableAt = time.Now().UTC()
		requeuedPayload, err := json.Marshal(job)
		if err != nil {
			c.pool.discard(rc)
			return reclaimed, fmt.Errorf("marshal reclaimed job: %w", err)
		}
		if err := writeExpectInteger(rc, "RPUSH", c.queue, string(requeuedPayload)); err != nil {
			c.pool.discard(rc)
			return reclaimed, fmt.Errorf("redis RPUSH reclaimed job: %w", err)
		}
		reclaimed++
	}
	c.pool.put(rc)
	return reclaimed, nil
}

func (c *Client) DeadLetter(ctx context.Context, item domain.DeadLetterJob) error {
	if item.FailedAt.IsZero() {
		item.FailedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal dead-letter job: %w", err)
	}
	return c.withIntegerCommand(ctx, "LPUSH", c.deadLetterQueue(), string(payload))
}

func (c *Client) ListDeadLetters(ctx context.Context, limit int) ([]domain.DeadLetterJob, error) {
	if limit <= 0 {
		limit = 50
	}
	payloads, err := c.readList(ctx, c.deadLetterQueue(), 0, limit-1)
	if err != nil {
		return nil, err
	}
	items := make([]domain.DeadLetterJob, 0, len(payloads))
	for _, payload := range payloads {
		var item domain.DeadLetterJob
		if err := json.Unmarshal([]byte(payload), &item); err != nil {
			return nil, fmt.Errorf("unmarshal dead-letter job: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *Client) ReplayDeadLetter(ctx context.Context, id string) (*domain.Job, error) {
	payloads, err := c.readList(ctx, c.deadLetterQueue(), 0, 499)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		var item domain.DeadLetterJob
		if err := json.Unmarshal([]byte(payload), &item); err != nil {
			continue
		}
		if item.ID != id {
			continue
		}
		if err := c.withIntegerCommand(ctx, "LREM", c.deadLetterQueue(), "1", payload); err != nil {
			return nil, err
		}
		job := item.Job
		job.Attempts = 0
		job.LastError = ""
		job.LeasedAt = time.Time{}
		job.AvailableAt = time.Now().UTC()
		if err := c.Enqueue(ctx, job); err != nil {
			return nil, err
		}
		return &job, nil
	}
	return nil, domain.ErrNotFound
}

func (c *Client) Stats(ctx context.Context) (*domain.QueueStats, error) {
	ready, err := c.integerReply(ctx, "LLEN", c.queue)
	if err != nil {
		return nil, err
	}
	processing, err := c.integerReply(ctx, "LLEN", c.processingQueue())
	if err != nil {
		return nil, err
	}
	delayed, err := c.integerReply(ctx, "ZCARD", c.delayedQueue())
	if err != nil {
		return nil, err
	}
	deadLetters, err := c.integerReply(ctx, "LLEN", c.deadLetterQueue())
	if err != nil {
		return nil, err
	}
	return &domain.QueueStats{
		Ready:       ready,
		Processing:  processing,
		Delayed:     delayed,
		DeadLetters: deadLetters,
	}, nil
}

func (c *Client) Check(ctx context.Context) error {
	rc, err := c.pool.get(ctx)
	if err != nil {
		return err
	}
	_ = rc.SetDeadline(time.Now().Add(2 * time.Second))
	defer func() { _ = rc.SetDeadline(time.Time{}) }()

	if _, err := rc.Write([]byte(buildArray("PING"))); err != nil {
		c.pool.discard(rc)
		return fmt.Errorf("redis PING write: %w", err)
	}
	line, err := readLine(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return fmt.Errorf("redis PING read: %w", err)
	}
	if line != "+PONG" {
		c.pool.discard(rc)
		return fmt.Errorf("unexpected ping response: %s", line)
	}
	c.pool.put(rc)
	return nil
}

func (c *Client) processingQueue() string {
	return c.queue + ":processing"
}

func (c *Client) delayedQueue() string {
	return c.queue + ":delayed"
}

func (c *Client) deadLetterQueue() string {
	return c.queue + ":dead"
}

func (c *Client) withIntegerCommand(ctx context.Context, args ...string) error {
	_, err := c.integerReply(ctx, args...)
	return err
}

func (c *Client) integerReply(ctx context.Context, args ...string) (int, error) {
	rc, err := c.pool.get(ctx)
	if err != nil {
		return 0, err
	}
	_ = rc.SetDeadline(time.Now().Add(c.timeout))
	defer func() { _ = rc.SetDeadline(time.Time{}) }()

	if _, err := rc.Write([]byte(buildArray(args...))); err != nil {
		c.pool.discard(rc)
		return 0, fmt.Errorf("redis %s write: %w", args[0], err)
	}
	value, err := readInteger(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return 0, fmt.Errorf("redis %s read: %w", args[0], err)
	}
	c.pool.put(rc)
	return value, nil
}

func (c *Client) readList(ctx context.Context, key string, start, stop int) ([]string, error) {
	rc, err := c.pool.get(ctx)
	if err != nil {
		return nil, err
	}
	_ = rc.SetDeadline(time.Now().Add(c.timeout))
	defer func() { _ = rc.SetDeadline(time.Time{}) }()

	if _, err := rc.Write([]byte(buildArray("LRANGE", key, strconv.Itoa(start), strconv.Itoa(stop)))); err != nil {
		c.pool.discard(rc)
		return nil, fmt.Errorf("redis LRANGE write: %w", err)
	}
	values, err := readBulkStringArray(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return nil, fmt.Errorf("redis LRANGE read: %w", err)
	}
	c.pool.put(rc)
	return values, nil
}

func writeExpectInteger(c *conn, args ...string) error {
	_, err := writeIntegerReply(c, args...)
	return err
}

func writeIntegerReply(c *conn, args ...string) (int, error) {
	if _, err := c.Write([]byte(buildArray(args...))); err != nil {
		return 0, err
	}
	return readInteger(c.reader)
}

// ── RESP helpers ──────────────────────────────────────────────────────────────

var errNilBulkString = errors.New("nil bulk string")

func buildArray(parts ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(parts))
	for _, part := range parts {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(part), part)
	}
	return b.String()
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func readBulkString(reader *bufio.Reader) (string, error) {
	header, err := readLine(reader)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(header, "$") {
		return "", fmt.Errorf("unexpected bulk string header: %s", header)
	}
	size, err := strconv.Atoi(strings.TrimPrefix(header, "$"))
	if err != nil {
		return "", fmt.Errorf("parse bulk string size: %w", err)
	}
	if size < 0 {
		return "", errNilBulkString
	}
	buf := make([]byte, size+2) // +2 for trailing \r\n
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", fmt.Errorf("read bulk string body: %w", err)
	}
	return string(buf[:size]), nil
}

func readBulkStringArray(reader *bufio.Reader) ([]string, error) {
	header, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(header, "*") {
		return nil, fmt.Errorf("unexpected array header: %s", header)
	}
	count, err := strconv.Atoi(strings.TrimPrefix(header, "*"))
	if err != nil {
		return nil, fmt.Errorf("parse array size: %w", err)
	}
	if count < 0 {
		return nil, nil
	}
	items := make([]string, 0, count)
	for i := 0; i < count; i++ {
		item, err := readBulkString(reader)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func readInteger(reader *bufio.Reader) (int, error) {
	line, err := readLine(reader)
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(line, "-") {
		return 0, fmt.Errorf("redis error: %s", strings.TrimPrefix(line, "-"))
	}
	if !strings.HasPrefix(line, ":") {
		return 0, fmt.Errorf("unexpected integer response: %s", line)
	}
	value, err := strconv.Atoi(strings.TrimPrefix(line, ":"))
	if err != nil {
		return 0, fmt.Errorf("parse integer response: %w", err)
	}
	return value, nil
}
