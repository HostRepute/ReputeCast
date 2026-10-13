package bouncebox

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// pop3Box is a minimal RFC 1939 client: USER/PASS, STAT, RETR, DELE, QUIT.
// Processed messages are deleted when the session ends (POP3 has no flags).
type pop3Box struct {
	conn    net.Conn
	r       *bufio.Reader
	cfg     Config
	deleted map[string]bool
}

func openPOP3(ctx context.Context, cfg Config) (Box, error) {
	d := cfg.dialer()
	tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	var err error
	if cfg.TLS == "tls" {
		conn, err = (&tls.Dialer{NetDialer: d, Config: tlsCfg}).DialContext(ctx, "tcp", cfg.addr())
	} else {
		conn, err = d.DialContext(ctx, "tcp", cfg.addr())
	}
	if err != nil {
		return nil, err
	}
	b := &pop3Box{conn: conn, r: bufio.NewReader(conn), cfg: cfg, deleted: map[string]bool{}}
	b.deadline()
	if _, err := b.status(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("pop3 greeting: %w", err)
	}
	if cfg.TLS == "starttls" {
		if _, err := b.cmd("STLS"); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("pop3 STLS: %w", err)
		}
		tc := tls.Client(conn, tlsCfg)
		if err := tc.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}
		b.conn, b.r = tc, bufio.NewReader(tc)
	}
	if _, err := b.cmd("USER " + cfg.Username); err != nil {
		_ = b.conn.Close()
		return nil, ErrAuth
	}
	if _, err := b.cmd("PASS " + cfg.Password); err != nil {
		_ = b.conn.Close()
		return nil, ErrAuth
	}
	return b, nil
}

func (b *pop3Box) deadline() { _ = b.conn.SetDeadline(time.Now().Add(b.cfg.Timeout)) }

// status reads a one-line +OK/-ERR response.
func (b *pop3Box) status() (string, error) {
	line, err := b.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if strings.HasPrefix(line, "+OK") {
		return strings.TrimSpace(strings.TrimPrefix(line, "+OK")), nil
	}
	return "", errors.New(strings.TrimSpace(strings.TrimPrefix(line, "-ERR")))
}

func (b *pop3Box) cmd(c string) (string, error) {
	b.deadline()
	if _, err := io.WriteString(b.conn, c+"\r\n"); err != nil {
		return "", err
	}
	return b.status()
}

// multiline reads a dot-terminated response, undoing dot-stuffing.
func (b *pop3Box) multiline() ([]byte, error) {
	var buf bytes.Buffer
	for {
		line, err := b.r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			return buf.Bytes(), nil
		}
		if strings.HasPrefix(trimmed, "..") {
			trimmed = trimmed[1:]
		}
		buf.WriteString(trimmed)
		buf.WriteString("\r\n")
		if buf.Len() > 16<<20 {
			return nil, errors.New("pop3: message too large")
		}
	}
}

func (b *pop3Box) Fetch(_ context.Context, max int) ([]Message, error) {
	st, err := b.cmd("STAT")
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(st)
	if len(fields) == 0 {
		return []Message{}, nil
	}
	n, _ := strconv.Atoi(fields[0])
	out := []Message{}
	for i := 1; i <= n && (max <= 0 || len(out) < max); i++ {
		id := strconv.Itoa(i)
		if b.deleted[id] {
			continue
		}
		if _, err := b.cmd("RETR " + id); err != nil {
			continue
		}
		raw, err := b.multiline()
		if err != nil {
			return out, err
		}
		out = append(out, Message{ID: id, Raw: raw})
	}
	return out, nil
}

func (b *pop3Box) Ack(_ context.Context, ids []string) error {
	for _, id := range ids {
		if _, err := b.cmd("DELE " + id); err != nil {
			return err
		}
		b.deleted[id] = true
	}
	return nil
}

func (b *pop3Box) Close() error {
	_, _ = b.cmd("QUIT")
	return b.conn.Close()
}
