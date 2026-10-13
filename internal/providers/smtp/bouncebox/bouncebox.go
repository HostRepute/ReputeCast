// Package bouncebox reads the bounce mailbox of an SMTP provider over IMAP
// (github.com/emersion/go-imap/v2, isolated here because it is still in beta)
// or POP3 (a small client in this package). Plain SMTP has no event
// webhooks, so bounces and abuse reports reach ReputeCast by email.
package bouncebox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Config describes a bounce mailbox.
type Config struct {
	Protocol string `json:"protocol"` // "imap" (default) or "pop3"
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	// TLS is "tls" (implicit, default), "starttls" or "none".
	TLS      string `json:"tls,omitempty"`
	Username string `json:"username"`
	// Folder is the IMAP folder to read (INBOX by default).
	Folder string `json:"folder,omitempty"`
	// Address is the mailbox's email address, shown in the event log
	// ("Parsed from bounces@news.acme.io via IMAP").
	Address string `json:"address,omitempty"`
	// Password comes from the provider's encrypted credentials.
	Password string `json:"-"`
	Timeout  time.Duration
	// Dialer, when set, dials through the provider address guard.
	Dialer *net.Dialer `json:"-"`
}

// dialer is Dialer (with the timeout) or a plain one.
func (c Config) dialer() *net.Dialer {
	if c.Dialer == nil {
		return &net.Dialer{Timeout: c.Timeout}
	}
	d := *c.Dialer
	if c.Timeout > 0 {
		d.Timeout = c.Timeout
	}
	return &d
}

// Message is one email in the mailbox.
type Message struct {
	ID  string
	Raw []byte
}

// Box is an open mailbox.
type Box interface {
	// Fetch returns up to max messages that have not been processed yet.
	Fetch(ctx context.Context, max int) ([]Message, error)
	// Ack marks messages as processed (IMAP: \Seen; POP3: deleted on Close).
	Ack(ctx context.Context, ids []string) error
	Close() error
}

// ErrAuth means the mailbox rejected the user name or password.
var ErrAuth = errors.New("the bounce mailbox rejected the user name or password")

// Open connects and authenticates.
func Open(ctx context.Context, c Config) (Box, error) {
	if c.Host == "" {
		return nil, errors.New("bounce mailbox: host is required")
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.TLS == "" {
		c.TLS = "tls"
	}
	switch strings.ToLower(c.Protocol) {
	case "", "imap":
		if c.Port == 0 {
			c.Port = map[string]int{"tls": 993, "starttls": 143, "none": 143}[c.TLS]
		}
		return openIMAP(ctx, c)
	case "pop3", "pop":
		if c.Port == 0 {
			c.Port = map[string]int{"tls": 995, "starttls": 110, "none": 110}[c.TLS]
		}
		return openPOP3(ctx, c)
	}
	return nil, fmt.Errorf("bounce mailbox: unknown protocol %q", c.Protocol)
}

func (c Config) addr() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.Port)) }

// Label is the protocol name for messages ("IMAP", "POP3").
func (c Config) Label() string {
	if strings.HasPrefix(strings.ToLower(c.Protocol), "pop") {
		return "POP3"
	}
	return "IMAP"
}
