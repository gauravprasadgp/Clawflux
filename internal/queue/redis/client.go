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

	if _, err := rc.Write([]byte(buildArray("BRPOP", c.queue, "5"))); err != nil {
		c.pool.discard(rc)
		return domain.Job{}, fmt.Errorf("redis BRPOP write: %w", err)
	}

	line, err := readLine(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		// Timeout from BRPOP — treat as "nothing to do" rather than fatal.
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return domain.Job{}, context.DeadlineExceeded
		}
		return domain.Job{}, fmt.Errorf("redis BRPOP read: %w", err)
	}

	if line == "$-1" || line == "*-1" {
		c.pool.put(rc)
		return domain.Job{}, context.DeadlineExceeded
	}
	if !strings.HasPrefix(line, "*") {
		c.pool.discard(rc)
		return domain.Job{}, fmt.Errorf("unexpected redis response: %s", line)
	}

	items, err := strconv.Atoi(strings.TrimPrefix(line, "*"))
	if err != nil {
		c.pool.discard(rc)
		return domain.Job{}, fmt.Errorf("parse array length: %w", err)
	}
	if items != 2 {
		c.pool.discard(rc)
		return domain.Job{}, fmt.Errorf("unexpected BRPOP payload size: %d", items)
	}

	// First bulk string is the queue name — discard it.
	if _, err := readBulkString(rc.reader); err != nil {
		c.pool.discard(rc)
		return domain.Job{}, err
	}
	payload, err := readBulkString(rc.reader)
	if err != nil {
		c.pool.discard(rc)
		return domain.Job{}, err
	}

	_ = rc.SetDeadline(time.Time{})
	c.pool.put(rc)

	var job domain.Job
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.Job{}, fmt.Errorf("unmarshal job: %w", err)
	}
	return job, nil
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

// ── RESP helpers ──────────────────────────────────────────────────────────────

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
	buf := make([]byte, size+2) // +2 for trailing \r\n
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", fmt.Errorf("read bulk string body: %w", err)
	}
	return string(buf[:size]), nil
}
