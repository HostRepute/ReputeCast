// Package systemmail sends ReputeCast's own mail (password resets, email
// changes, team invitations, notifications, double opt-in confirmations,
// test messages) through the workspace's System-stream route, whose fallback
// takes over when the provider is down or refuses the message. It implements
// mailer.Sender (App.Mail).
//
// Messages without a workspace (a password reset, an email change) go
// through the recipient's first workspace, by membership date, that has a
// usable System route, and otherwise through the instance's oldest workspace
// with one. The From address is the workspace's System sender, else its
// default sender, else its first sender.
package systemmail

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// ErrNoSender means the workspace has no sender identity to send from. It
// counts as mailer.ErrNoSystemProvider, so callers fall back the way they do
// without a System provider (copy a link, the CLI command).
var ErrNoSender error = noSenderError{}

type noSenderError struct{}

func (noSenderError) Error() string {
	return "the workspace has no sender address for system mail: add one in Settings › Senders"
}

func (noSenderError) Is(target error) bool { return target == mailer.ErrNoSystemProvider }

// availabilityTTL is how long SystemAvailable answers from its cache.
const availabilityTTL = 30 * time.Second

// Mailer is the System-stream mailer.
type Mailer struct {
	Q       *store.Queries
	Deliver *Deliverer
	// Instance returns the instance name and public URL for the brand.
	Instance func(ctx context.Context) (name, publicURL string)
	Log      *slog.Logger
	Now      func() time.Time

	mu    sync.Mutex
	cache map[string]cachedAvailability
}

type cachedAvailability struct {
	ok    bool
	until time.Time
}

var _ mailer.Sender = (*Mailer)(nil)

func (m *Mailer) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m *Mailer) log() *slog.Logger {
	if m.Log != nil {
		return m.Log
	}
	return slog.Default()
}

// plan returns the providers to try for a recipient, in order: the System
// route's provider when it is available, then its fallback (the System
// stream fails over automatically unless someone changed its policy).
func (m *Mailer) plan(ctx context.Context, wsID uuid.UUID, to string) ([]store.Provider, error) {
	set, err := routing.Load(ctx, m.Q, wsID)
	if err != nil {
		return nil, err
	}
	d, ok := set.Resolve(routing.Recipient{Stream: providers.StreamSystem, Email: to})
	if !ok {
		return nil, nil
	}
	load := func(id uuid.UUID) (store.Provider, bool) {
		p, err := m.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: wsID, ID: id})
		if err != nil || !m.Deliver.Usable(p) {
			return store.Provider{}, false
		}
		return p, true
	}
	primary, primaryOK := load(d.ProviderID)
	var out []store.Provider
	if primaryOK && state.Available(primary) {
		out = append(out, primary)
		// A refused message moves on to the fallback when failover is automatic.
		if d.Fallback != nil && *d.Fallback != d.ProviderID && d.Policy == routing.Automatic {
			if fb, ok := load(*d.Fallback); ok && state.Available(fb) {
				out = append(out, fb)
			}
		}
		return out, nil
	}
	target, _ := routing.Failover(d.Policy, d.Fallback, func(id uuid.UUID) routing.ProviderState {
		p, ok := load(id)
		return routing.ProviderState{Available: ok && state.Available(p), WarmingUp: ok && p.Status == state.WarmingUp}
	})
	if target != nil {
		if p, ok := load(*target); ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// ready reports whether a workspace can send system mail: its System route
// has a provider to try and it has an address to send from.
func (m *Mailer) ready(ctx context.Context, wsID uuid.UUID) bool {
	plan, err := m.plan(ctx, wsID, "probe@example.com")
	if err != nil || len(plan) == 0 {
		return false
	}
	_, _, err = m.sender(ctx, wsID)
	return err == nil
}

// SystemAvailable implements mailer.Sender. It answers from a 30-second
// cache, so screens can ask on every load.
func (m *Mailer) SystemAvailable(ctx context.Context, workspaceID *uuid.UUID) bool {
	key := "*"
	if workspaceID != nil {
		key = workspaceID.String()
	}
	now := m.now()
	m.mu.Lock()
	if c, ok := m.cache[key]; ok && now.Before(c.until) {
		m.mu.Unlock()
		return c.ok
	}
	m.mu.Unlock()
	ok := false
	if workspaceID != nil {
		ok = m.ready(ctx, *workspaceID)
	} else if ids, err := m.Q.SysmailWorkspaces(ctx); err == nil {
		for _, id := range ids {
			if m.ready(ctx, id) {
				ok = true
				break
			}
		}
	}
	m.mu.Lock()
	if m.cache == nil {
		m.cache = map[string]cachedAvailability{}
	}
	m.cache[key] = cachedAvailability{ok: ok, until: now.Add(availabilityTTL)}
	m.mu.Unlock()
	return ok
}

// Forget drops the cached availability (after routing or providers changed).
func (m *Mailer) Forget() {
	m.mu.Lock()
	m.cache = nil
	m.mu.Unlock()
}

// SendSystem implements mailer.Sender: it returns the name of the provider
// that accepted the message.
func (m *Mailer) SendSystem(ctx context.Context, msg mailer.Message) (string, error) {
	to, err := mail.ParseAddress(strings.TrimSpace(msg.To))
	if err != nil {
		return "", errors.New("system mail: invalid recipient address")
	}
	kind := msg.Kind
	if kind == "" {
		kind = "system"
	}
	candidates := []uuid.UUID{}
	if msg.WorkspaceID != nil {
		candidates = append(candidates, *msg.WorkspaceID)
	} else {
		if ids, err := m.Q.SysmailWorkspacesForEmail(ctx, strings.ToLower(to.Address)); err == nil {
			candidates = append(candidates, ids...)
		}
		if ids, err := m.Q.SysmailWorkspaces(ctx); err == nil {
			for _, id := range ids {
				if !containsID(candidates, id) {
					candidates = append(candidates, id)
				}
			}
		}
	}
	lastErr := mailer.ErrNoSystemProvider
	for _, wsID := range candidates {
		plan, err := m.plan(ctx, wsID, to.Address)
		if err != nil {
			lastErr = err
			continue
		}
		if len(plan) == 0 {
			continue
		}
		from, replyTo, err := m.sender(ctx, wsID)
		if err != nil {
			lastErr = err
			continue
		}
		brand := m.brand(ctx, wsID, kind, msg.WorkspaceID == nil)
		rendered, err := Render(kind, brand, msg.Subject, msg.Text, msg.HTML)
		if err != nil {
			return "", err
		}
		for _, p := range plan {
			pm := providers.Message{
				ID: uuid.Must(uuid.NewV7()), Stream: providers.StreamSystem, From: from, ReplyTo: replyTo,
				To: mail.Address{Name: msg.ToName, Address: strings.ToLower(to.Address)}, Subject: msg.Subject, HTML: rendered.HTML, Text: rendered.Text,
				Headers: map[string]string{"Auto-Submitted": "auto-generated", "X-ReputeCast-Kind": kind},
				Tags:    map[string]string{"rc_kind": kind},
			}
			if _, err := m.Deliver.Send(ctx, p, pm, kind); err != nil {
				m.log().WarnContext(ctx, "system mail not accepted", "provider", p.Name, "kind", kind, "err", err)
				lastErr = err
				continue
			}
			return p.Name, nil
		}
		if msg.WorkspaceID != nil {
			break
		}
	}
	return "", lastErr
}

func containsID(xs []uuid.UUID, id uuid.UUID) bool {
	for _, x := range xs {
		if x == id {
			return true
		}
	}
	return false
}

// sender picks the From address of a workspace's system mail.
func (m *Mailer) sender(ctx context.Context, wsID uuid.UUID) (mail.Address, *mail.Address, error) {
	defs, err := m.Q.SenderWorkspaceDefaults(ctx, wsID)
	if err != nil {
		return mail.Address{}, nil, err
	}
	for _, id := range []*uuid.UUID{defs.DefaultSystemSenderID, defs.DefaultSenderID} {
		if id == nil {
			continue
		}
		if sn, err := m.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: wsID, ID: *id}); err == nil {
			return addresses(sn.Name, sn.Email, sn.ReplyTo)
		}
	}
	list, err := m.Q.SenderList(ctx, wsID)
	if err != nil {
		return mail.Address{}, nil, err
	}
	if len(list) == 0 {
		return mail.Address{}, nil, ErrNoSender
	}
	return addresses(list[0].Name, list[0].Email, list[0].ReplyTo)
}

func addresses(name, email, replyTo string) (mail.Address, *mail.Address, error) {
	from := mail.Address{Name: name, Address: email}
	if replyTo == "" || strings.EqualFold(replyTo, email) {
		return from, nil, nil
	}
	return from, &mail.Address{Address: replyTo}, nil
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)

// brand is ReputeCast's (the instance's) for team mail and the publisher's
// for subscriber mail.
func (m *Mailer) brand(ctx context.Context, wsID uuid.UUID, kind string, instanceLevel bool) Brand {
	name, public := "ReputeCast", ""
	if m.Instance != nil {
		if n, p := m.Instance(ctx); n != "" {
			name, public = n, p
		} else {
			public = p
		}
	}
	host := public
	if u, err := url.Parse(public); err == nil && u.Host != "" {
		host = u.Host
	}
	b := Brand{Name: name, Accent: "#0F766E", Footer: "Sent by " + name}
	if host != "" {
		b.Footer += " · " + host
	}
	if instanceLevel && !subscriberKinds[kind] {
		return b
	}
	ws, err := m.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return b
	}
	if !subscriberKinds[kind] {
		b.Footer = "Sent by " + name + " for " + ws.Name
		if host != "" {
			b.Footer += " · " + host
		}
		return b
	}
	var branding struct {
		Accent        string `json:"accent"`
		FooterAddress string `json:"footer_address"`
	}
	_ = json.Unmarshal(ws.Branding, &branding)
	b.Name, b.Accent, b.Footer = ws.Name, "#4F46E5", ws.Name
	if hexColor.MatchString(branding.Accent) {
		b.Accent = branding.Accent
	}
	if branding.FooterAddress != "" {
		b.Footer += " · " + branding.FooterAddress
	}
	return b
}
