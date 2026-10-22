package automations

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// OutgoingEmail is one "Send email" step for one contact.
type OutgoingEmail struct {
	WorkspaceID  uuid.UUID
	AutomationID uuid.UUID
	Version      int
	NodeID       string
	RunID        uuid.UUID
	ContactID    uuid.UUID
	// To is the contact's address; MailboxProvider their mailbox provider key.
	To              string
	MailboxProvider string
	EmailID         uuid.UUID
	EmailVersion    int
	Subject         string
	// Route is empty for the Automations route, or a provider id that
	// overrides it for this step.
	Route string
	// Context is what the trigger passed to the flow's emails (a store's
	// checkout or order): the sending engine keeps it with the message.
	Context map[string]string
}

// QueuedEmail is what the mailer did with an email.
type QueuedEmail struct {
	MessageID uuid.UUID
	// ProviderID is the sending provider chosen (uuid.Nil when held).
	ProviderID uuid.UUID
	// Held is true when no provider on the route can send right now: the
	// message waits and goes out once one is healthy.
	Held bool
}

// Mailer queues automation emails. QueueAutomationEmail runs inside the
// engine's transaction and must not do network I/O: implementations record
// the message (a messages row with stream "automation", automation_id and
// automation_node set) and schedule their own delivery. internal/app wires in
// the sending engine's implementation (automationMailer).
type Mailer interface {
	QueueAutomationEmail(ctx context.Context, tx pgx.Tx, m OutgoingEmail) (QueuedEmail, error)
}

// QueueOnlyMailer records the message as queued on a provider picked from
// the Automations route, or as held when no provider can send. Nothing is
// delivered until the sending domain replaces it.
type QueueOnlyMailer struct{}

// QueueAutomationEmail implements Mailer.
func (QueueOnlyMailer) QueueAutomationEmail(ctx context.Context, tx pgx.Tx, m OutgoingEmail) (QueuedEmail, error) {
	q := store.New(tx)
	provider, err := pickProvider(ctx, q, m.WorkspaceID, m.Route, m.RunID)
	if err != nil {
		return QueuedEmail{}, err
	}
	out := QueuedEmail{MessageID: uuid.Must(uuid.NewV7()), ProviderID: provider}
	status := "queued"
	var pid *uuid.UUID
	if provider == uuid.Nil {
		out.Held = true
		status = "held"
	} else {
		pid = &provider
	}
	node := m.NodeID
	mbp := m.MailboxProvider
	if mbp == "" {
		mbp = "other"
	}
	_, err = q.InsertAutomationMessage(ctx, store.InsertAutomationMessageParams{
		ID: out.MessageID, WorkspaceID: m.WorkspaceID, AutomationID: &m.AutomationID, AutomationNode: &node,
		ContactID: &m.ContactID, Email: m.To, ProviderID: pid, MailboxProvider: mbp, Status: status, QueuedAt: nowUTC(ctx),
	})
	return out, err
}

type ctxNowKey struct{}

// withClock passes the engine's clock to code it calls (tests travel in time).
func withClock(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, ctxNowKey{}, t)
}

// nowUTC returns the engine's clock when set, else the wall clock.
func nowUTC(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ctxNowKey{}).(time.Time); ok {
		return t
	}
	return time.Now()
}

// pickProvider chooses who sends: the override when it can send, otherwise a
// weighted pick on the Automations route, otherwise its fallback; uuid.Nil
// when nobody can send.
func pickProvider(ctx context.Context, q *store.Queries, wsID uuid.UUID, override string, key uuid.UUID) (uuid.UUID, error) {
	provs, err := q.AutomationProviders(ctx, wsID)
	if err != nil {
		return uuid.Nil, err
	}
	status := map[uuid.UUID]string{}
	for _, p := range provs {
		status[p.ID] = p.Status
	}
	if override != "" {
		if id, err := uuid.Parse(override); err == nil && healthyStatus(status[id]) {
			return id, nil
		}
	}
	rule, err := q.AutomationRoute(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		// No Automations route: any healthy provider will do.
		for _, p := range provs {
			if healthyStatus(p.Status) {
				return p.ID, nil
			}
		}
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	var splits []struct {
		ProviderID uuid.UUID `json:"provider_id"`
		Weight     int       `json:"weight"`
	}
	_ = json.Unmarshal(rule.Splits, &splits)
	total := 0
	for _, sp := range splits {
		if healthyStatus(status[sp.ProviderID]) {
			total += sp.Weight
		}
	}
	if total > 0 {
		h := fnv.New32a()
		_, _ = h.Write(key[:])
		pick := int(h.Sum32() % uint32(total))
		for _, sp := range splits {
			if !healthyStatus(status[sp.ProviderID]) {
				continue
			}
			if pick < sp.Weight {
				return sp.ProviderID, nil
			}
			pick -= sp.Weight
		}
	}
	if rule.FallbackProviderID != nil && healthyStatus(status[*rule.FallbackProviderID]) {
		return *rule.FallbackProviderID, nil
	}
	return uuid.Nil, nil
}

// ContentChecker runs the content checks of Send Readiness on an email. It
// returns "" when the email passes, otherwise the first problem in a few
// words ("no unsubscribe link").
type ContentChecker interface {
	Check(c EmailContent) string
}

// BasicContentChecks covers what every automation email needs.
type BasicContentChecks struct{}

var (
	reImg    = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	reImgAlt = regexp.MustCompile(`(?is)\balt\s*=\s*("[^"]+"|'[^']+')`)
)

var (
	mCheckNoSubject     = msg("check.noSubject")
	mCheckEmpty         = msg("check.empty")
	mCheckNoUnsubscribe = msg("check.noUnsubscribe")
	mCheckClipped       = msg("check.clipped")
	mCheckNoAlt         = msg("check.noAlt")
)

// Check implements ContentChecker.
func (BasicContentChecks) Check(c EmailContent) string { return say(english, basicCheck(c)) }

// basicCheck is the first problem BasicContentChecks finds, or nothing.
func basicCheck(c EmailContent) i18n.Text {
	switch {
	case strings.TrimSpace(c.Subject) == "":
		return txt(mCheckNoSubject)
	case strings.TrimSpace(c.HTML) == "" && strings.TrimSpace(c.MJML) == "":
		return txt(mCheckEmpty)
	case !strings.Contains(c.HTML+c.MJML, "unsubscribe_url") && !strings.Contains(strings.ToLower(c.HTML), "unsubscribe"):
		return txt(mCheckNoUnsubscribe)
	case len(c.HTML) > 102*1024:
		return txt(mCheckClipped)
	}
	for _, img := range reImg.FindAllString(c.HTML, -1) {
		if !reImgAlt.MatchString(img) {
			return txt(mCheckNoAlt)
		}
	}
	return i18n.Text{}
}
