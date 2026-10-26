package connect

import (
	"context"
	"math"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// The event check (review PR-3): an SES setup counts as connected only
// once a real event made it back. ReputeCast sends a message to the
// provider's mailbox simulator (success@simulator.amazonses.com), which
// accepts it and makes the provider report a delivery without touching the
// account's rates, and waits for that event to arrive the way the setup
// chose (SNS calling the webhook, or the queue ReputeCast polls). The
// provider only sends from a verified identity: without one yet, the check
// waits for domain verification (the next step) and the setup can go on.

const (
	eventSendKey  = "event_send"
	eventCheckKey = "event_check"
	// maxEventAttempts bounds the wait for the event: 5 s apart, 10 min.
	maxEventAttempts = 120
	// statusDeferred is a row waiting for a verified sending domain.
	statusDeferred = "deferred"
)

// eventRows are the event check's provisioning rows, for an adapter with
// a simulator address.
func eventRows(adapter providers.Provider) []ProvisionState {
	ec, ok := adapter.(providers.EventChecker)
	if !ok || ec.EventCheckAddress() == "" {
		return nil
	}
	send := ProvisionState{Key: eventSendKey, Status: "waiting"}
	send.Label = send.say("label", mProvisionEventSend, i18n.Args{"address": ec.EventCheckAddress()})
	check := ProvisionState{Key: eventCheckKey, Status: "waiting"}
	check.Label = check.say("label", mProvisionEventCheck, nil)
	return []ProvisionState{send, check}
}

// eventsVerified reports whether the event check found its event; ok is
// false for setups without one (other providers, setups from before).
func eventsVerified(rows []ProvisionState) (verified, ok bool) {
	for _, r := range rows {
		if r.Key == eventCheckKey {
			return r.Status == "done", true
		}
	}
	return false, false
}

// eventsDeferred reports whether the event check waits for a verified
// sending domain.
func eventsDeferred(rows []ProvisionState) bool {
	for _, r := range rows {
		if r.Key == eventSendKey && r.Status == statusDeferred {
			return true
		}
	}
	return false
}

// eventFound is the event the check found: when it arrived, how (webhook
// or queue) and when its message went out.
type eventFound struct {
	at     time.Time
	source string
	sent   time.Time
}

// runEventStep runs one of the event check's rows. again: look again in
// a few seconds; stop: provisioning waits (for a verified domain).
func (s *Service) runEventStep(ctx context.Context, st store.ProviderSetup, setupState State, p store.Provider, row ProvisionState) (again, stop bool, err error) {
	now := s.now()
	demo := manage.IsDemo(p)
	switch row.Key {
	case eventSendKey:
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return false, false, err
		}
		ec, ok := adapter.(providers.EventChecker)
		if !ok {
			return false, true, nil
		}
		address := ec.EventCheckAddress()
		short := state.Short(p.Name, p.Adapter)
		if demo {
			// A demo provider has no account to send through.
			return false, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At, r.MessageID = "done", &now, ""
				r.Text = r.say("text", mProvisionEventSent, i18n.Args{"address": address})
				r.Note = r.say("note", mProvisionEventSentNote, i18n.Args{"short": short})
			})
		}
		if s.Deliver == nil {
			return false, true, nil
		}
		from, err := ec.EventCheckFrom(ctx, s.senderCandidates(ctx, p.WorkspaceID))
		if err != nil {
			msg, _ := failureText(err)
			return false, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At, r.Error = "failed", &now, msg
			})
		}
		if from == "" {
			// Nothing verified at the provider yet: the check goes out once
			// domain verification (the next step) has verified one.
			return false, true, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At = statusDeferred, &now
				r.Text = r.say("text", mProvisionEventDeferred, i18n.Args{"address": address})
			})
		}
		id := uuid.Must(uuid.NewV7())
		subject, _ := say(mEventCheckSubject, nil)
		body, _ := say(mEventCheckBody, i18n.Args{"name": p.Name})
		msg := providers.Message{ID: id, Stream: providers.StreamSystem, From: mail.Address{Name: "ReputeCast", Address: from}, To: mail.Address{Address: address},
			Subject: subject, Text: body + "\n", HTML: "<p>" + htmlEscape(body) + "</p>",
			Headers: map[string]string{"X-ReputeCast-Kind": "event_check"}, Tags: map[string]string{"rc_kind": "event_check"}}
		if _, err := s.Deliver.Send(ctx, p, msg, "event_check"); err != nil {
			return false, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At, r.Error = "failed", &now, providers.MessageIn(err, i18n.Default)
			})
		}
		return false, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
			r.clearProblem()
			r.Status, r.At, r.MessageID = "done", &now, id.String()
			r.Text = r.say("text", mProvisionEventSent, i18n.Args{"address": address})
			r.Note = r.say("note", mProvisionEventSentNote, i18n.Args{"short": short})
		})
	case eventCheckKey:
		queue := setupState.Connection.Events == providers.EventsQueue
		if demo {
			return false, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At = "done", &now
				r.Text = r.say("text", mProvisionEventReceived, i18n.Args{"time": now})
				r.Note = r.say("note", mProvisionEventVia, i18n.Args{"via": viaKey(queue), "seconds": 2})
			})
		}
		found, ok := s.eventFor(ctx, p, setupState.Connection.Provisioning)
		if ok {
			if err := s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				at := found.at
				r.Status, r.At = "done", &at
				r.Text = r.say("text", mProvisionEventReceived, i18n.Args{"time": found.at})
				r.Note = r.say("note", mProvisionEventVia, i18n.Args{"via": viaKey(found.source == "queue"),
					"seconds": max(int(math.Round(found.at.Sub(found.sent).Seconds())), 1)})
			}); err != nil {
				return false, false, err
			}
			wsID := p.WorkspaceID
			return false, false, s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "provider.events_verified", TargetType: "provider", TargetID: p.ID.String(),
				TargetName: p.Name, Actor: &audit.Actor{Type: "system", Name: "System"}, After: map[string]any{"via": found.source, "received_at": found.at}})
		}
		attempts := row.Attempts + 1
		if attempts >= maxEventAttempts {
			hookURL := s.webhookURL(p, "")
			return false, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At, r.Attempts = "failed", &now, attempts
				r.Error = r.say("error", mProvisionNoEvent, i18n.Args{"minutes": maxEventAttempts * 5 / 60})
				if queue {
					r.Hint = r.say("hint", mProvisionHintEventQueue, nil)
				} else {
					r.Hint = r.say("hint", mProvisionHintEventWebhook, i18n.Args{"url": hookURL})
				}
			})
		}
		return true, false, s.setRow(ctx, st.ID, row.Key, func(r *ProvisionState) { r.Attempts = attempts })
	}
	return false, false, nil
}

func viaKey(queue bool) string {
	if queue {
		return "queue"
	}
	return "webhook"
}

// eventFor finds the provider's first event about the check's message.
func (s *Service) eventFor(ctx context.Context, p store.Provider, rows []ProvisionState) (eventFound, bool) {
	id := ""
	for _, r := range rows {
		if r.Key == eventSendKey {
			id = r.MessageID
		}
	}
	m, ok := s.testMessage(ctx, p.WorkspaceID, id)
	if !ok || m.SentAt == nil || m.ProviderMessageID == nil {
		return eventFound{}, false
	}
	pmid := strings.Trim(strings.TrimSpace(*m.ProviderMessageID), "<>")
	if pmid == "" {
		return eventFound{}, false
	}
	hook, err := s.Q.ConnectFirstWebhook(ctx, store.ConnectFirstWebhookParams{ProviderID: p.ID, Since: m.SentAt.Add(-time.Minute), Until: m.SentAt.Add(30 * time.Minute),
		ProviderMessageID: pmid})
	if err != nil {
		return eventFound{}, false
	}
	return eventFound{at: hook.ReceivedAt, source: hook.Source, sent: *m.SentAt}, true
}

// senderCandidates are the workspace's sender addresses, the default first.
func (s *Service) senderCandidates(ctx context.Context, wsID uuid.UUID) []string {
	out := []string{}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	senders, serr := s.Q.SenderList(ctx, wsID)
	if serr != nil {
		return out
	}
	for _, sn := range senders {
		if err == nil && ws.DefaultSenderID != nil && *ws.DefaultSenderID == sn.ID {
			out = append([]string{sn.Email}, out...)
			continue
		}
		out = append(out, sn.Email)
	}
	return out
}

// resumeEventCheck sends the deferred event check again once a domain
// may be verified (the person went on from the domain step).
func (s *Service) resumeEventCheck(ctx context.Context, setupID uuid.UUID) {
	resumed := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st, err := q.SetupLock(ctx, setupID)
		if err != nil {
			return err
		}
		setupState := decodeState(st.State)
		for i := range setupState.Connection.Provisioning {
			r := &setupState.Connection.Provisioning[i]
			if r.Key == eventSendKey && r.Status == statusDeferred {
				r.Status, r.Text = "waiting", ""
				delete(r.Texts, "text")
				resumed = true
			}
		}
		if !resumed {
			return nil
		}
		_, err = s.save(ctx, q, st, setupState)
		return err
	})
	if err != nil {
		s.log().WarnContext(ctx, "resume event check", "setup", setupID, "err", err)
		return
	}
	if resumed {
		if c := s.jobs(); c != nil {
			if _, err := c.Insert(ctx, ProvisionArgs{SetupID: setupID}, nil); err != nil {
				s.log().WarnContext(ctx, "queue event check", "setup", setupID, "err", err)
			}
		}
	}
}

// htmlEscape escapes the few characters the check's plain sentence may hold.
func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
