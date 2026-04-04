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
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type Client struct {
	addr    string
	queue   string
	timeout time.Duration
}

func NewClient(addr, queue string) *Client {
	return &Client{
		addr:    addr,
		queue:   queue,
		timeout: 5 * time.Second,
	}
}

func (c *Client) Enqueue(ctx context.Context, job domain.Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := buildArray("RPUSH", c.queue, string(payload))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	_, err = readLine(reader)
	return err
}

func (c *Client) Dequeue(ctx context.Context) (domain.Job, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return domain.Job{}, err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(buildArray("BRPOP", c.queue, "5"))); err != nil {
		return domain.Job{}, err
	}

	reader := bufio.NewReader(conn)
	line, err := readLine(reader)
	if err != nil {
		return domain.Job{}, err
	}
	if line == "$-1" || line == "*-1" {
		return domain.Job{}, context.DeadlineExceeded
	}
	if !strings.HasPrefix(line, "*") {
		return domain.Job{}, fmt.Errorf("unexpected redis response: %s", line)
	}

	items, err := strconv.Atoi(strings.TrimPrefix(line, "*"))
	if err != nil {
		return domain.Job{}, err
	}
	if items != 2 {
		return domain.Job{}, errors.New("unexpected BRPOP payload size")
	}

	if _, err := readBulkString(reader); err != nil {
		return domain.Job{}, err
	}
	payload, err := readBulkString(reader)
	if err != nil {
		return domain.Job{}, err
	}

	var job domain.Job
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.Job{}, err
	}
	return job, nil
}

func (c *Client) Check(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(buildArray("PING"))); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	line, err := readLine(reader)
	if err != nil {
		return err
	}
	if line != "+PONG" {
		return fmt.Errorf("unexpected ping response: %s", line)
	}
	return nil
}

func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	return conn, nil
}

func buildArray(parts ...string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, part := range parts {
		b.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(part), part))
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
		return "", err
	}
	buf := make([]byte, size+2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", err
	}
	return string(buf[:size]), nil
}
