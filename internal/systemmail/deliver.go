package systemmail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// ErrDemoProvider is returned outside development for the demo's sample
// providers: they have no real account behind them.
var ErrDemoProvider = errors.New("this is a sample provider from the demo data and can't send real mail; connect a sending provider and route the System stream to it")

// Deliverer sends one message through a connected provider and records it
// like any other send: a messages row (stream system, with its kind), the
// provider's call statistics and, when the provider reports no delivery
// events of its own (SMTP), the delivered event inferred from its 250 reply.
//
// Demo providers (providers.config "demo": true) never reach a provider
// API: in development their mail goes to DevSMTP (Mailpit) and is recorded
// as delivered by that provider; elsewhere Send refuses with ErrDemoProvider.
type Deliverer struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Factory *manage.Factory
	// ApplyEvent records a normalized event through the ingest pipeline
	// (message status, provider stats, event log).
	ApplyEvent func(ctx context.Context, tx pgx.Tx, p store.Provider, ev providers.Event) error
	// DevSMTP is the development mail catcher ("localhost:1025"); empty
	// outside development.
	DevSMTP string
	Log     *slog.Logger
	Now     func() time.Time
}

// Delivery is the outcome of one message.
type Delivery struct {
	ProviderMessageID string
	QueuedAt          time.Time
	SentAt            time.Time
	// Via names where a demo provider's mail went ("Mailpit").
	Via string
}

func (d *Deliverer) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d *Deliverer) log() *slog.Logger {
	if d.Log != nil {
		return d.Log
	}
	return slog.Default()
}

// Usable reports whether Send can deliver through p at all (demo providers
// only in development).
func (d *Deliverer) Usable(p store.Provider) bool {
	return !manage.IsDemo(p) || d.DevSMTP != ""
}

// adapter returns what actually sends for p.
func (d *Deliverer) adapter(ctx context.Context, p store.Provider) (providers.Provider, string, error) {
	if !manage.IsDemo(p) {
		a, err := d.Factory.Open(ctx, p)
		return a, "", err
	}
	if d.DevSMTP == "" {
		return nil, "", ErrDemoProvider
	}
	desc, ok := providers.Lookup("smtp")
	if !ok || desc.New == nil {
		return nil, "", errors.New("the SMTP adapter isn't in this build, so demo mail can't reach Mailpit")
	}
	host, port, err := net.SplitHostPort(d.DevSMTP)
	if err != nil {
		return nil, "", fmt.Errorf("mail.dev_smtp %q: %w", d.DevSMTP, err)
	}
	n, _ := strconv.Atoi(port)
	settings, _ := json.Marshal(map[string]any{"host": host, "port": n, "tls": "none", "auth": "none", "helo": "reputecast.local"})
	a, err := desc.New(providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: settings, Credentials: json.RawMessage(`{}`)})
	return a, "Mailpit", err
}

// Send delivers msg through p and records it. kind tags the message
// ("password_reset", "invite", "test").
func (d *Deliverer) Send(ctx context.Context, p store.Provider, msg providers.Message, kind string) (Delivery, error) {
	out := Delivery{QueuedAt: d.now()}
	adapter, via, err := d.adapter(ctx, p)
	if err != nil {
		return out, err
	}
	out.Via = via
	to := strings.ToLower(msg.To.Address)
	stream := msg.Stream
	if stream == "" {
		stream = providers.StreamSystem
	}
	if err := d.Q.SysmailInsertMessage(ctx, store.SysmailInsertMessageParams{ID: msg.ID, WorkspaceID: p.WorkspaceID, Stream: string(stream), Email: to, ProviderID: &p.ID,
		MailboxProvider: routing.MailboxProvider(to), QueuedAt: out.QueuedAt, Kind: kind}); err != nil {
		return out, err
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	results, sendErr := adapter.Send(cctx, []providers.Message{msg})
	cancel()
	out.SentAt = d.now()
	if err := manage.RecordSend(ctx, d.Pool, p, results, sendErr, out.SentAt); err != nil {
		d.log().WarnContext(ctx, "record system send", "provider", p.ID, "err", err)
	}
	if sendErr == nil && (len(results) == 0 || results[0].Err != nil) {
		if len(results) > 0 {
			sendErr = results[0].Err
		} else {
			sendErr = errors.New("the provider returned no result")
		}
	}
	if sendErr != nil {
		_ = d.Q.SysmailMarkSent(ctx, store.SysmailMarkSentParams{ID: msg.ID, Status: "failed", SentAt: &out.SentAt, Error: clip(sendErr.Error(), 500)})
		return out, fmt.Errorf("%s: %w", p.Name, sendErr)
	}
	out.ProviderMessageID = results[0].ProviderMessageID
	pmid := out.ProviderMessageID
	var pmidPtr *string
	if pmid != "" {
		pmidPtr = &pmid
	}
	if err := d.Q.SysmailMarkSent(ctx, store.SysmailMarkSentParams{ID: msg.ID, Status: "sent", ProviderMessageID: pmidPtr, SentAt: &out.SentAt}); err != nil {
		return out, err
	}
	// Providers that report no delivery events (SMTP) and demo mail caught
	// by Mailpit count as delivered on the server's 250 reply.
	caps := state.Caps(p)
	if d.ApplyEvent != nil && (via != "" || caps.Webhooks == "" || caps.Webhooks == "Bounce mailbox") {
		ev := providers.Event{Type: providers.EventDelivered, ProviderMessageID: pmid, MessageID: &msg.ID, Recipient: to, OccurredAt: out.SentAt,
			OriginalType: "SMTP 250 reply", OriginalCode: "250", Diagnostic: "2.0.0 accepted by " + smtpHost(p)}
		if via != "" {
			ev.OriginalType, ev.Diagnostic = "Delivered to "+via+" (development)", "2.0.0 Ok: queued · smtpResponse from "+d.DevSMTP
		}
		err := db.Tx(ctx, d.Pool, func(tx pgx.Tx) error { return d.ApplyEvent(ctx, tx, p, ev) })
		if err != nil {
			d.log().WarnContext(ctx, "record inferred delivery", "provider", p.ID, "err", err)
		}
	}
	return out, nil
}

func smtpHost(p store.Provider) string {
	var c struct {
		Host string `json:"host"`
	}
	_ = json.Unmarshal(p.Config, &c)
	if c.Host == "" {
		return p.Name
	}
	return c.Host
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
