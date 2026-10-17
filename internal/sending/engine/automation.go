package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// AutomationMailer is the automations engine's "Send email" step: it routes
// the contact on the Automations stream (automatic failover), records the
// message and wakes the outbox, all inside the engine's transaction.
type AutomationMailer struct{ E *Engine }

var _ automations.Mailer = AutomationMailer{}

// QueueAutomationEmail implements automations.Mailer.
func (m AutomationMailer) QueueAutomationEmail(ctx context.Context, tx pgx.Tx, o automations.OutgoingEmail) (automations.QueuedEmail, error) {
	q := store.New(tx)
	mbp := o.MailboxProvider
	contactLanguage := ""
	if c, err := q.SendingContactForAutomation(ctx, o.ContactID); err == nil {
		if c.MailboxProvider != "" && mbp == "" {
			mbp = c.MailboxProvider
		}
		contactLanguage = c.Language
	}
	if mbp == "" {
		mbp = routing.MailboxProvider(o.To)
	}
	pid, err := m.E.automationRoute(ctx, q, o.WorkspaceID, o.Route, o.To, mbp)
	if err != nil {
		return automations.QueuedEmail{}, err
	}
	out := automations.QueuedEmail{MessageID: newID()}
	status := "queued"
	var provider *uuid.UUID
	if pid == uuid.Nil {
		out.Held, status = true, "held"
	} else {
		out.ProviderID, provider = pid, &pid
	}
	version := o.EmailVersion
	if version <= 0 {
		if v, err := q.SendingAutomationEmailLatest(ctx, o.EmailID); err == nil {
			version = int(v)
		}
	}
	ml, err := m.E.automationLanguage(ctx, o.WorkspaceID, o.EmailID, version, contactLanguage)
	if err != nil {
		return out, err
	}
	cr := contentRef{Email: o.EmailID, Version: version, Subject: o.Subject}
	if ml.Language != "" && ml.Language != ml.source {
		cr.Language = ml.Language
	}
	ref, _ := json.Marshal(cr)
	node := o.NodeID
	// Milestone emails (Grow) are automation emails outside any flow.
	var automation *uuid.UUID
	if o.AutomationID != uuid.Nil {
		automation = &o.AutomationID
	}
	if err := q.SendingInsertAutomationMessage(ctx, store.SendingInsertAutomationMessageParams{
		ID: out.MessageID, WorkspaceID: o.WorkspaceID, AutomationID: automation, AutomationNode: &node, ContactID: &o.ContactID,
		Email: o.To, ProviderID: provider, MailboxProvider: mbp, Status: status, QueuedAt: m.E.now(), ContentRef: ref,
		Language: ml.Language, ContactLanguage: ml.Contact, LanguageReason: ml.Reason,
	}); err != nil {
		return out, err
	}
	if !out.Held {
		if err := m.E.insertTx(ctx, tx, OutboxArgs{}, nil); err != nil {
			return out, err
		}
	}
	return out, nil
}

// automationRoute picks the provider of an automation email: the step's
// override when it can send, else the Automations rules, failing over
// automatically when the provider is down; uuid.Nil when nobody can send.
func (e *Engine) automationRoute(ctx context.Context, q *store.Queries, ws uuid.UUID, override, to, mbp string) (uuid.UUID, error) {
	provs, err := q.SendingWorkspaceProviders(ctx, ws)
	if err != nil {
		return uuid.Nil, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	if override != "" {
		if id, err := uuid.Parse(override); err == nil {
			if p, ok := byID[id]; ok && state.Available(p) {
				return id, nil
			}
		}
	}
	set, err := routing.Load(ctx, q, ws)
	if err != nil {
		return uuid.Nil, err
	}
	d, ok := set.Resolve(routing.Recipient{Stream: providers.StreamAutomation, Email: to, MailboxProvider: mbp})
	if !ok {
		return uuid.Nil, nil
	}
	if p, ok := byID[d.ProviderID]; ok && state.Available(p) {
		return d.ProviderID, nil
	}
	target, hold := routing.Failover(set.Policy(providers.StreamAutomation), d.Fallback, func(id uuid.UUID) routing.ProviderState {
		p, ok := byID[id]
		if !ok {
			return routing.ProviderState{}
		}
		return routing.ProviderState{Available: state.Available(p), WarmingUp: p.Status == state.WarmingUp}
	})
	if hold || target == nil {
		return uuid.Nil, nil
	}
	return *target, nil
}

// outboxTemplate caches an automation email version's template.
type outboxTemplate struct {
	tpl     *Template
	sender  store.SendingAutomationSenderRow
	name    string
	missing string
	// source is set when the language version queued is no longer sent:
	// the source language goes instead.
	source string
}

// RunOutbox sends queued automation emails per provider, within each
// provider's shared rate, for up to budget.
func (e *Engine) RunOutbox(ctx context.Context, budget time.Duration) (time.Duration, bool, error) {
	start := e.now()
	cache := map[string]*outboxTemplate{}
	transports := map[uuid.UUID]*runRoute{}
	for {
		now := e.now()
		if now.Sub(start) >= budget {
			return 0, false, nil
		}
		ids, err := e.Q.SendingOutboxProviders(ctx)
		if err != nil {
			return 0, false, err
		}
		if len(ids) == 0 {
			return 0, true, nil
		}
		sent := 0
		for _, pid := range ids {
			n, err := e.outboxTick(ctx, pid, transports, cache, now)
			if err != nil {
				return 0, false, err
			}
			sent += n
		}
		wait := time.Until(now.Truncate(time.Second).Add(time.Second))
		if sent == 0 {
			wait = 2 * time.Second
		}
		select {
		case <-ctx.Done():
			return 0, false, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// OutboxStep sends one second's automation emails (tests and fast-forward).
func (e *Engine) OutboxStep(ctx context.Context, now time.Time) (int, error) {
	ids, err := e.Q.SendingOutboxProviders(ctx)
	if err != nil {
		return 0, err
	}
	sent := 0
	transports := map[uuid.UUID]*runRoute{}
	cache := map[string]*outboxTemplate{}
	for _, pid := range ids {
		n, err := e.outboxTick(ctx, pid, transports, cache, now)
		if err != nil {
			return sent, err
		}
		sent += n
	}
	return sent, nil
}

func (e *Engine) outboxTick(ctx context.Context, pid uuid.UUID, transports map[uuid.UUID]*runRoute, cache map[string]*outboxTemplate, now time.Time) (int, error) {
	r := transports[pid]
	if r == nil {
		p, err := e.Q.ProvGetAny(ctx, pid)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil // the provider was deleted; its mail fails over or waits
		}
		if err != nil {
			return 0, err
		}
		r = &runRoute{provider: p, batch: batchSize(p), tracked: state.TrackingOf(p).Mode != "provider"}
		r.transport, _, r.openErr = e.transport(ctx, p, "", time.Time{})
		if r.openErr != nil {
			e.log().WarnContext(ctx, "open a provider for automation mail", "provider", p.ID, "err", r.openErr)
		}
		if s, ok := r.transport.(*simulator.Provider); ok {
			r.sim = s
		}
		r.rate = e.rateOf(ctx, e.Q, p, time.UTC, now)
		if pr := state.PricingOf(p); pr.Model == "metered" {
			r.micros = int64(math.Round(pr.Per1000 * 1000))
		}
		transports[pid] = r
	}
	if !state.Available(r.provider) {
		// Automations fail over on their own.
		return 0, e.failOverOutbox(ctx, r.provider)
	}
	// A provider whose adapter can't be opened (logged above) keeps its
	// mail queued, as does one cooling down after a failed call.
	if r.transport == nil || r.cooldown.After(now) {
		return 0, nil
	}
	want := max(int(math.Floor(r.rate.PerSecond)), 1)
	granted, err := e.reserveAfterTransactional(ctx, pid, now.Unix(), want, int(math.Ceil(r.rate.PerSecond)), now)
	if err != nil || granted <= 0 {
		return 0, err
	}
	rows, err := e.Q.SendingOutboxClaim(ctx, store.SendingOutboxClaimParams{ProviderID: &pid, Stale: now.Add(-staleClaim), Now: now, Lim: int32(granted)})
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	var out outcome
	type pending struct {
		row store.SendingOutboxClaimRow
		msg providers.Message
	}
	var batch []pending
	for _, row := range rows {
		switch {
		case row.Gone:
			out.drop(row.ID, "cancelled", "The contact was erased, unsubscribed or suppressed before the email went out.")
			continue
		case row.Suppressed:
			out.drop(row.ID, "cancelled", "On the shared suppression list.")
			continue
		}
		msg, err := e.composeAutomation(ctx, r, row, cache)
		if err != nil {
			out.drop(row.ID, "failed", err.Error())
			continue
		}
		batch = append(batch, pending{row, msg})
	}
	key := routeKey{uuid.Nil, pid}
	for len(batch) > 0 {
		n := min(len(batch), max(r.batch, 1))
		chunk := batch[:n]
		batch = batch[n:]
		msgs := make([]providers.Message, len(chunk))
		for i, p := range chunk {
			msgs[i] = p.msg
		}
		cctx, cancel := context.WithTimeout(simulator.WithNow(ctx, now), 30*time.Second)
		res, callErr := r.transport.Send(cctx, msgs)
		cancel()
		if callErr == nil && len(res) != len(msgs) {
			callErr = &providers.CallError{Op: "Send", Message: "the provider didn't answer for every message", Temporary: true}
		}
		e.recordCall(key, now, callErr)
		if err := manage.RecordSend(ctx, e.Pool, r.provider, res, callErr, now); err != nil {
			e.log().WarnContext(ctx, "record provider call", "provider", pid, "err", err)
		}
		if callErr != nil {
			for _, p := range chunk {
				out.retry(p.row.ID, int(p.row.Attempts), now, callText(callErr))
			}
			for _, p := range batch {
				out.release = append(out.release, p.row.ID)
			}
			batch = nil
			r.cooldown = now.Add(backoffMin)
			break
		}
		for i, rr := range res {
			p := chunk[i]
			switch {
			case rr.Err == nil:
				out.sentIDs = append(out.sentIDs, p.row.ID)
				out.sentPMID = append(out.sentPMID, rr.ProviderMessageID)
				if r.sim != nil {
					out.sims = append(out.sims, simRows(p.row.WorkspaceID, nil, p.row.ID, pid, now, r.sim.Plan(p.msg))...)
				}
			case rr.Err.Temporary:
				out.retry(p.row.ID, int(p.row.Attempts), now, rr.Err.Error())
			default:
				out.drop(p.row.ID, "rejected", rr.Err.Error())
			}
		}
	}
	err = db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		return e.record(ctx, store.New(tx), tx, uuid.Nil, pid, &out, now)
	})
	if err == nil && len(out.sims) > 0 {
		e.insert(ctx, SimArgs{}, nil)
	}
	return len(out.sentIDs), err
}

// failOverOutbox moves a down provider's queued automation emails to the
// Automations fallback.
func (e *Engine) failOverOutbox(ctx context.Context, p store.Provider) error {
	set, err := routing.Load(ctx, e.Q, p.WorkspaceID)
	if err != nil {
		return err
	}
	def, ok := set.Default(providers.StreamAutomation)
	if !ok || def.Fallback == nil || *def.Fallback == p.ID {
		return nil
	}
	fb, err := e.Q.ProvGetAny(ctx, *def.Fallback)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // no fallback any more: the mail waits for its provider
	}
	if err != nil {
		return err
	}
	if !state.Available(fb) {
		return nil
	}
	_, err = e.Pool.Exec(ctx, `UPDATE messages SET provider_id = $2, claimed_at = NULL
		WHERE provider_id = $1 AND status = 'queued' AND campaign_id IS NULL AND stream = 'automation'`, p.ID, fb.ID)
	return err
}

// composeAutomation renders an automation email for one contact.
func (e *Engine) composeAutomation(ctx context.Context, r *runRoute, row store.SendingOutboxClaimRow, cache map[string]*outboxTemplate) (providers.Message, error) {
	var ref contentRef
	_ = json.Unmarshal(row.ContentRef, &ref)
	if ref.Email == uuid.Nil {
		return providers.Message{}, errors.New("the flow step has no email")
	}
	k := ref.Email.String() + ":" + fmt.Sprint(ref.Version) + ":" + ref.Subject + ":" + ref.Language
	ot := cache[k]
	if ot == nil {
		ot = &outboxTemplate{}
		cache[k] = ot
		em, err := e.Q.SendingAutomationEmail(ctx, store.SendingAutomationEmailParams{EmailID: ref.Email, Version: int32(ref.Version)})
		if err != nil {
			ot.missing = "the email version this step sends no longer exists"
		} else {
			subject := em.Subject
			if strings.TrimSpace(ref.Subject) != "" {
				subject = ref.Subject
			}
			c := Content{Subject: subject, Preheader: em.Preheader, HTML: media.Absolutize(em.Html, e.URLs.Base()), Text: em.PlainText}
			lang := ref.Language
			if lang != "" {
				// A language version has its own subject: the step's subject
				// override is the source's.
				lc, source, err := e.automationTranslation(ctx, em.WorkspaceID, ref.Email, ref.Version, lang)
				switch {
				case err != nil:
					ot.missing = err.Error()
				case lc == nil:
					// The version stopped being sent after this email was
					// queued (its language was removed): the source goes.
					ot.source, lang = source, ""
				default:
					c = *lc
				}
			}
			tpl, err := Prepare(c, Tracking{Opens: true, Clicks: true})
			if err != nil {
				ot.missing = err.Error()
			} else {
				ot.tpl, ot.name = tpl, em.Name
				if n, err := e.Q.SendingAutomationLinkCount(ctx, store.SendingAutomationLinkCountParams{EmailID: ref.Email, Version: int32(ref.Version), Language: lang}); err == nil && n == 0 {
					for _, l := range tpl.Links {
						_ = e.Q.SendingInsertAutomationLink(ctx, store.SendingInsertAutomationLinkParams{EmailID: ref.Email, Version: int32(ref.Version), Language: lang,
							Position: int32(l.Position), Url: l.URL, Label: l.Label})
					}
				}
			}
			ot.sender, _ = e.Q.SendingAutomationSender(ctx, store.SendingAutomationSenderParams{WorkspaceID: em.WorkspaceID, EmailID: ref.Email})
			if ot.sender.Email == "" && ot.missing == "" {
				ot.missing = "the workspace has no default sender; set one in Settings › General"
			}
		}
	}
	if ot.missing != "" {
		return providers.Message{}, errors.New(capitalize(ot.missing) + ".")
	}
	if ot.source != "" {
		if err := e.Q.SendingMessageToSource(ctx, store.SendingMessageToSourceParams{ID: row.ID, Language: ot.source}); err != nil {
			return providers.Message{}, err
		}
	}
	var contact uuid.UUID
	if row.ContactID != nil {
		contact = *row.ContactID
	}
	sign := func(kind linktoken.Kind, link uint32) string {
		return e.Links.Sign(linktoken.Claims{Kind: kind, WorkspaceID: row.WorkspaceID, MessageID: row.ID, ContactID: contact, Link: link})
	}
	slug := ot.sender.Slug
	unsub := sign(linktoken.Unsubscribe, 0)
	links := map[string]string{
		"unsubscribe_url":     e.URLs.Unsubscribe(slug, unsub),
		"preferences_url":     e.URLs.Preferences(slug, sign(linktoken.Preferences, 0)),
		"view_in_browser_url": e.URLs.ViewOnline(slug, sign(linktoken.ViewOnline, 0)),
	}
	tracked := r.tracked
	if tracked {
		links[OpenVar] = e.URLs.Open(sign(linktoken.Open, 0))
		for _, l := range ot.tpl.Links {
			links[LinkVar(l.Position)] = e.URLs.Click(sign(linktoken.Click, uint32(l.Position)))
		}
	}
	var attrs map[string]any
	_ = json.Unmarshal(row.Attributes, &attrs)
	out, err := ot.tpl.Render(liquid.Vars(row.Email, row.FirstName, row.LastName, attrs, links), tracked)
	if err != nil {
		return providers.Message{}, err
	}
	auto := ""
	if row.AutomationID != nil {
		auto = row.AutomationID.String()
	}
	msg := providers.Message{
		ID: row.ID, Stream: providers.StreamAutomation,
		From: mail.Address{Name: ot.sender.Name, Address: ot.sender.Email}, To: mail.Address{Name: strings.TrimSpace(row.FirstName + " " + row.LastName), Address: row.Email},
		Subject: out.Subject, HTML: out.HTML, Text: out.Text,
		Headers: bulkHeaders(links["unsubscribe_url"], unsub, firstNonEmpty(ot.sender.ReplyTo, ot.sender.Email), feedbackID(auto, slug, "automation")),
		Tags:    map[string]string{"automation": auto, "workspace": slug},
	}
	if ot.sender.ReplyTo != "" {
		if a, err := mail.ParseAddress(ot.sender.ReplyTo); err == nil {
			msg.ReplyTo = a
		}
	}
	if r.sim != nil {
		msg.Tags[simulator.TagMailbox] = row.MailboxProvider
		msg.Tags[simulator.TagTier] = row.Tier
		msg.Tags[simulator.TagSource] = row.Source
		msg.Tags[simulator.TagEngaged] = fmt.Sprint(row.Engaged)
	}
	return msg, nil
}
