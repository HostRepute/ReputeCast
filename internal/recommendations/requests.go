package recommendations

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/mailer"
	pp "github.com/hostrepute/reputecast/internal/recommendations/partnerproto"
	"github.com/hostrepute/reputecast/internal/store"
)

// Outcomes of a reader's Subscribe on our pages.
const (
	// OutcomeAccepted: the partner will email the reader to confirm.
	OutcomeAccepted = "accepted"
	// OutcomePassed: the partner didn't answer yet; the request is retried
	// ("We've passed your request to Ops Weekly…").
	OutcomePassed = "passed"
	// OutcomeLimit: the partner's daily limit is reached.
	OutcomeLimit = "limit"
	// OutcomeUnavailable: the partner link is paused, ended or invalid.
	OutcomeUnavailable = "unavailable"
)

func emailAD(request uuid.UUID) string { return "partner_request:" + request.String() + ":email" }

// backoff is the wait before a subscribe request's next attempt.
func backoff(attempt int) time.Duration {
	steps := []time.Duration{time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour}
	if attempt < len(steps) {
		return steps[attempt]
	}
	return 3 * time.Hour
}

// --- The recommending side -----------------------------------------------------------

// RequestSubscribe sends a reader's Subscribe to a partner: a signed
// request with only the address and our newsletter's name. It tries at once
// and, when the partner doesn't answer, keeps retrying for 24 hours.
func (s *Service) RequestSubscribe(ctx context.Context, ws *store.Workspace, recID uuid.UUID, email, origin string) (string, error) {
	rec, err := s.Q.RecGet(ctx, store.RecGetParams{WorkspaceID: ws.ID, ID: recID})
	if err != nil || rec.Kind != KindPartner || rec.PartnerLinkID == nil {
		return "", errNoLink
	}
	l, err := s.Q.PartnerLinkByID(ctx, *rec.PartnerLinkID)
	if err != nil {
		return "", errNoLink
	}
	now := s.now()
	if _, hidden := visibility(rec, &l, now); hidden {
		if l.LimitUntil != nil && l.LimitUntil.After(now) {
			return OutcomeLimit, nil
		}
		return OutcomeUnavailable, nil
	}
	addr, problem := emailaddr.Check(email)
	if problem != "" {
		return "", errBadEmail
	}
	id := uuid.Must(uuid.NewV7())
	sealed, err := s.Box.SealString(addr, emailAD(id))
	if err != nil {
		return "", err
	}
	if origin != "page" {
		origin = "confirm"
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerRequestOutInsert(ctx, store.PartnerRequestOutInsertParams{ID: id, WorkspaceID: ws.ID, LinkID: l.ID, RecommendationID: &rec.ID,
			Email: sealed, Origin: origin, Now: now}); err != nil {
			return err
		}
		return q.RecDailyBump(ctx, store.RecDailyBumpParams{RecommendationID: rec.ID, Day: dayOf(now, ws), Requests: 1})
	})
	if err != nil {
		return "", err
	}
	return s.deliver(ctx, id), nil
}

var errBadEmail = errors.New("recommendations: not an email address")

// deliver sends one queued subscribe request and records the answer.
func (s *Service) deliver(ctx context.Context, id uuid.UUID) string {
	r, err := s.Q.PartnerRequestOutGet(ctx, id)
	if err != nil || r.Status != "queued" || len(r.Email) == 0 {
		return OutcomePassed
	}
	l, err := s.Q.PartnerLinkByID(ctx, r.LinkID)
	if err != nil {
		return OutcomeUnavailable
	}
	ws, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
	if err != nil {
		return OutcomeUnavailable
	}
	now := s.now()
	finish := func(status string, next *time.Time, lastError string) {
		if err := s.Q.PartnerRequestOutResult(ctx, store.PartnerRequestOutResultParams{ID: id, Status: status, NextAttemptAt: next, LastError: lastError, Now: now}); err != nil {
			s.log().WarnContext(ctx, "partner request: record", "err", err)
		}
	}
	if l.Status != LinkConnected {
		finish("failed", nil, "The partner link is "+l.Status+".")
		return OutcomeUnavailable
	}
	email, err := s.Box.OpenString(r.Email, emailAD(id))
	if err != nil {
		finish("failed", nil, "The address couldn't be read.")
		return OutcomeUnavailable
	}
	me, _ := s.ownProfile(ctx, &ws)
	var out pp.SubscribeResponse
	ce := s.signedCall(ctx, l, "subscribe", pp.SubscribeRequest{Protocol: pp.Version, Request: id.String(), Email: email, Source: me.Name, At: r.CreatedAt}, &out)
	if ce != nil {
		s.linkTrouble(ctx, &ws, l, ce)
		if ce.Kind == "problem" {
			finish("rejected", nil, ce.Error())
			return OutcomeUnavailable
		}
		if now.Sub(r.CreatedAt) >= RequestRetryFor {
			finish("failed", nil, ce.Error())
			return OutcomePassed
		}
		next := now.Add(backoff(int(r.Attempts)))
		finish("queued", &next, ce.Error())
		return OutcomePassed
	}
	if l.FailingSince != nil {
		_ = s.Q.PartnerLinkFailing(ctx, store.PartnerLinkFailingParams{ID: l.ID, Failing: false, Now: now})
	}
	switch out.Status {
	case pp.StatusAccepted:
		finish("accepted", nil, "")
		return OutcomeAccepted
	case pp.StatusLimitReached:
		until := nextMidnightUTC(now)
		if out.ResetsAt != nil && out.ResetsAt.After(now) && out.ResetsAt.Before(now.Add(48*time.Hour)) {
			until = *out.ResetsAt
		}
		_ = s.Q.PartnerLinkLimit(ctx, store.PartnerLinkLimitParams{ID: l.ID, Until: &until, Now: now})
		finish("limit", nil, "")
		return OutcomeLimit
	case pp.StatusPaused:
		if l.Status == LinkConnected {
			_ = s.receiveState(ctx, &ws, l, pp.StatePaused)
		}
		finish("rejected", nil, "The partner paused the link.")
		return OutcomeUnavailable
	case pp.StatusEnded:
		_ = s.receiveState(ctx, &ws, l, pp.StateEnded)
		finish("rejected", nil, "The partner ended the partnership.")
		return OutcomeUnavailable
	}
	finish("failed", nil, "The partner's answer wasn't understood.")
	return OutcomeUnavailable
}

// SendDue sends the subscribe requests whose next attempt is due.
func (s *Service) SendDue(ctx context.Context) (int, error) {
	var ids []uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, err := store.New(tx).PartnerRequestsOutDue(ctx, store.PartnerRequestsOutDueParams{Now: s.now(), Lim: 100})
		if err != nil {
			return err
		}
		next := s.now().Add(time.Minute)
		for _, r := range rows {
			ids = append(ids, r.ID)
			// Claimed for a minute: another worker leaves them.
			if _, err := tx.Exec(ctx, `UPDATE partner_requests_out SET next_attempt_at = $2 WHERE id = $1`, r.ID, next); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		s.deliver(ctx, id)
	}
	return len(ids), nil
}

// receiveConfirmed counts a partner's confirmation of one of our requests.
func (s *Service) receiveConfirmed(ctx context.Context, ws *store.Workspace, l store.PartnerLink, in pp.ConfirmedNotice) error {
	id, err := uuid.Parse(in.Request)
	if err != nil {
		return nil //nolint:nilerr // not one of ours: nothing to count
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rec, err := q.PartnerRequestOutConfirmed(ctx, store.PartnerRequestOutConfirmedParams{ID: id, LinkID: l.ID, Now: now})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // counted already, or not ours
		}
		if err != nil || rec == nil {
			return err
		}
		return q.RecDailyBump(ctx, store.RecDailyBumpParams{RecommendationID: *rec, Day: dayOf(now, ws), Confirmed: 1})
	})
}

// --- The recommended side --------------------------------------------------------------

// receiveSubscribe handles a partner's subscribe request: the daily limit,
// then our own double opt-in. The answer is the same whether or not the
// address already gets the list, so the partner never learns who reads it.
func (s *Service) receiveSubscribe(ctx context.Context, ws *store.Workspace, l store.PartnerLink, in pp.SubscribeRequest) (pp.SubscribeResponse, error) {
	out := pp.SubscribeResponse{Protocol: pp.Version}
	now := s.now()
	if l.Status == LinkPaused {
		out.Status = pp.StatusPaused
		return out, nil
	}
	if l.Status != LinkConnected || l.ListID == nil {
		out.Status = pp.StatusEnded
		return out, nil
	}
	// A request id seen before gets the same answer (retries).
	var prev string
	if err := s.Pool.QueryRow(ctx, `SELECT status FROM partner_requests_in WHERE link_id = $1 AND request_id = $2`, l.ID, in.Request).Scan(&prev); err == nil {
		if prev == "limit" {
			out.Status, out.ResetsAt = pp.StatusLimitReached, ptr(nextMidnightUTC(now))
		} else {
			out.Status = pp.StatusAccepted
		}
		return out, nil
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// One link at a time, so two requests can't both take the last
		// place of the day.
		if _, err := q.PartnerLinkLock(ctx, l.ID); err != nil {
			return err
		}
		today, err := q.PartnerRequestsInToday(ctx, store.PartnerRequestsInTodayParams{LinkID: l.ID, Since: midnightUTC(now)})
		if err != nil {
			return err
		}
		if today >= int64(l.DailyLimit) {
			out.Status, out.ResetsAt = pp.StatusLimitReached, ptr(nextMidnightUTC(now))
			_, err := q.PartnerRequestInInsert(ctx, store.PartnerRequestInInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, LinkID: l.ID,
				RequestID: in.Request, Status: "limit", Now: now})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		out.Status = pp.StatusAccepted
		var contact *uuid.UUID
		asked := false
		if _, problem := emailaddr.Check(in.Email); problem == "" && s.Audience != nil {
			res, err := s.Audience.Signup(ctx, tx, ws, audience.SignupInput{Email: in.Email, Lists: []uuid.UUID{*l.ListID}, DoubleOptIn: true,
				Evidence: audience.PublicEvidence{Source: "partner:" + l.ID.String(), Note: "partner request from " + l.RemoteName, At: now}})
			var fe *audience.FieldError
			switch {
			case errors.As(err, &fe):
			case err != nil:
				return err
			default:
				contact = &res.ContactID
				asked = res.NeedsConfirmation() && slices.Contains(res.Pending, *l.ListID)
			}
		}
		id := uuid.Must(uuid.NewV7())
		if _, err := q.PartnerRequestInInsert(ctx, store.PartnerRequestInInsertParams{ID: id, WorkspaceID: ws.ID, LinkID: l.ID, RequestID: in.Request,
			Status: "accepted", ContactID: contact, Asked: asked, Now: now}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		if asked {
			_, err := tx.Exec(ctx, `UPDATE partner_requests_in SET optin_status = 'queued', optin_next_at = $2, source_name = $3 WHERE id = $1`,
				id, now, l.RemoteName)
			return err
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	if s.Jobs != nil {
		if j := s.Jobs(); j != nil {
			_, _ = j.Insert(ctx, SendArgs{}, nil)
		}
	}
	return out, nil
}

// SendOptIns sends the confirmation emails of partner signups on the System
// stream: our usual double opt-in email with "You asked for this on the
// Acme Weekly Digest's page."
func (s *Service) SendOptIns(ctx context.Context) (int, error) {
	now := s.now()
	rows, err := s.Pool.Query(ctx, `SELECT ri.id, ri.workspace_id, ri.contact_id, ri.source_name, ri.received_at, ri.optin_attempts, pl.list_id
		FROM partner_requests_in ri JOIN partner_links pl ON pl.id = ri.link_id
		WHERE ri.optin_status = 'queued' AND ri.optin_next_at <= $1 AND ri.contact_id IS NOT NULL
		ORDER BY ri.optin_next_at LIMIT 100`, now)
	if err != nil {
		return 0, err
	}
	type due struct {
		id, ws, contact uuid.UUID
		source          string
		received        time.Time
		attempts        int32
		list            *uuid.UUID
	}
	var items []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.ws, &d.contact, &d.source, &d.received, &d.attempts, &d.list); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, d)
	}
	rows.Close()
	sent := 0
	for _, d := range items {
		status, next, err := s.sendOptIn(ctx, d.ws, d.contact, d.list, d.source)
		if err != nil {
			s.log().WarnContext(ctx, "partner signup: confirmation email", "request", d.id, "err", err)
		}
		switch {
		case status == "sent":
			sent++
		case status == "queued" && now.Sub(d.received) >= OptInRetryFor:
			status, next = "failed", nil
		}
		if _, err := s.Pool.Exec(ctx, `UPDATE partner_requests_in SET optin_status = $2, optin_attempts = optin_attempts + 1, optin_next_at = $3,
			optin_sent_at = CASE WHEN $2 = 'sent' THEN $4::timestamptz END WHERE id = $1`, d.id, status, next, now); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

// sendOptIn sends one partner signup's confirmation email; "queued" with a
// next attempt when no System provider can send yet.
func (s *Service) sendOptIn(ctx context.Context, wsID, contactID uuid.UUID, list *uuid.UUID, source string) (string, *time.Time, error) {
	now := s.now()
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil || s.Forms == nil || s.Audience == nil || s.Mail == nil || s.Mail() == nil {
		return "failed", nil, err
	}
	m := s.Mail()
	if !m.SystemAvailable(ctx, &ws.ID) {
		return "queued", ptr(now.Add(10 * time.Minute)), nil
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, &ws, contactID)
	if err != nil {
		return "failed", nil, nil //nolint:nilerr // the contact is gone
	}
	if list == nil || pc.Lists[*list] != "pending" {
		return "failed", nil, nil // confirmed some other way, or left
	}
	set, err := s.Forms.LoadSettings(ctx, &ws)
	if err != nil {
		return "queued", ptr(now.Add(10 * time.Minute)), err
	}
	var listName string
	_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, *list).Scan(&listName)
	// The confirmation email limit applies to partner signups too (review
	// 2026-11 DP-3): held back, the request waits until the next may go.
	claimed, err := s.Audience.ClaimOptInSend(ctx, contactID)
	var cool *audience.OptInCooldown
	if errors.As(err, &cool) {
		return "queued", ptr(cool.Until), nil
	}
	if err != nil {
		return "queued", ptr(now.Add(5 * time.Minute)), err
	}
	msg := forms.PartnerConfirmationEmail(set.Branding, pc.FirstName, []string{listName}, s.Forms.ConfirmURL(&ws, contactID, uuid.Nil), source)
	msg.WorkspaceID, msg.To, msg.ToName = &ws.ID, pc.Email, pc.FirstName
	provider, err := m.SendSystem(ctx, msg)
	if err != nil {
		s.Audience.ReleaseOptInSend(ctx, contactID, claimed)
		if errors.Is(err, mailer.ErrNoSystemProvider) {
			return "queued", ptr(now.Add(10 * time.Minute)), nil
		}
		return "queued", ptr(now.Add(5 * time.Minute)), err
	}
	note := "confirmation sent for a partner request from " + source
	if provider != "" {
		note += " via " + provider + " (System stream)"
	}
	if err := s.Audience.RecordOptInSent(ctx, s.Q, contactID, []uuid.UUID{*list}, audience.PublicEvidence{Source: "system", Note: note, At: now}); err != nil {
		s.log().WarnContext(ctx, "partner signup: record the confirmation email", "err", err)
	}
	return "sent", nil, nil
}

// Hook tells partners when a reader they sent us confirmed
// (forms.SignupHook): the partner counts it, never learning who.
type Hook struct {
	S func() *Service
}

var _ forms.SignupHook = Hook{}

// DoubleOptIn implements forms.SignupHook (partner signups don't come
// through forms).
func (Hook) DoubleOptIn(context.Context, *store.Workspace, forms.Submission) bool { return false }

// Accepted implements forms.SignupHook.
func (Hook) Accepted(context.Context, pgx.Tx, *store.Workspace, forms.SignupEvent) error { return nil }

// Confirmed implements forms.SignupHook: the partner requests of the
// contact on the lists just confirmed queue their signed confirmation.
func (h Hook) Confirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e forms.ConfirmEvent) error {
	q := store.New(tx)
	rows, err := q.PartnerRequestsInPending(ctx, &e.ContactID)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.ListID == nil || !slices.Contains(e.Lists, *r.ListID) {
			continue
		}
		if err := q.PartnerRequestInConfirmed(ctx, store.PartnerRequestInConfirmedParams{ID: r.ID, Now: e.At}); err != nil {
			return err
		}
	}
	return nil
}

// SendNotices sends the partners' signed confirmations, retried for 24
// hours.
func (s *Service) SendNotices(ctx context.Context) (int, error) {
	now := s.now()
	var due []store.PartnerRequestsIn
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, err := store.New(tx).PartnerNoticesDue(ctx, store.PartnerNoticesDueParams{Now: now, Lim: 100})
		if err != nil {
			return err
		}
		due = rows
		for _, r := range rows {
			if _, err := tx.Exec(ctx, `UPDATE partner_requests_in SET notice_next_at = $2 WHERE id = $1`, r.ID, now.Add(time.Minute)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, r := range due {
		l, err := s.Q.PartnerLinkByID(ctx, r.LinkID)
		status, next := "sent", (*time.Time)(nil)
		switch {
		case err != nil || l.Status == LinkEnded || l.Status == LinkInvalid:
			status = "failed"
		default:
			at := now
			if r.ConfirmedAt != nil {
				at = *r.ConfirmedAt
			}
			if ce := s.signedCall(ctx, l, "confirmed", pp.ConfirmedNotice{Protocol: pp.Version, Request: r.RequestID, At: at}, nil); ce != nil {
				if ws, err := s.Q.GetWorkspace(ctx, l.WorkspaceID); err == nil {
					s.linkTrouble(ctx, &ws, l, ce)
				}
				status = "queued"
				if r.ConfirmedAt != nil && now.Sub(*r.ConfirmedAt) >= RequestRetryFor || ce.Kind == "problem" {
					status = "failed"
				} else {
					next = ptr(now.Add(backoff(int(r.NoticeAttempts))))
				}
			}
		}
		if status == "sent" {
			sent++
		}
		if err := s.Q.PartnerNoticeResult(ctx, store.PartnerNoticeResultParams{ID: r.ID, NoticeStatus: status, NoticeNextAt: next, Now: now}); err != nil {
			return sent, err
		}
	}
	return sent, nil
}

// --- The request log ---------------------------------------------------------------

// PartnerRequestItem is one row of a partner request log: no addresses.
type PartnerRequestItem struct {
	ID          string     `json:"id"`
	At          time.Time  `json:"at"`
	Status      string     `json:"status" enum:"queued,accepted,limit,failed,rejected,paused"`
	Origin      string     `json:"origin" enum:"confirm,page,partner"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"lastError"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
}

// PartnerRequestLog is a partner link's requests of the last 90 days.
type PartnerRequestLog struct {
	LinkID    string               `json:"linkId"`
	Partner   string               `json:"partner"`
	Direction string               `json:"direction" enum:"out,in"`
	Items     []PartnerRequestItem `json:"items"`
	Total     int64                `json:"total"`
}

// RequestLog is a partner link's request log, newest first.
func (s *Service) RequestLog(ctx context.Context, ws *store.Workspace, linkID uuid.UUID, page int) (PartnerRequestLog, error) {
	l, err := s.link(ctx, ws, linkID)
	if err != nil {
		return PartnerRequestLog{}, err
	}
	if page < 1 {
		page = 1
	}
	out := PartnerRequestLog{LinkID: l.ID.String(), Partner: l.RemoteName, Direction: l.Direction, Items: []PartnerRequestItem{}}
	var rows pgx.Rows
	if l.Direction == DirOut {
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM partner_requests_out WHERE link_id = $1`, l.ID).Scan(&out.Total)
		rows, err = s.Pool.Query(ctx, `SELECT id::text, created_at, status, origin, attempts, last_error, confirmed_at FROM partner_requests_out
			WHERE link_id = $1 ORDER BY created_at DESC LIMIT 50 OFFSET $2`, l.ID, (page-1)*50)
	} else {
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM partner_requests_in WHERE link_id = $1`, l.ID).Scan(&out.Total)
		rows, err = s.Pool.Query(ctx, `SELECT request_id, received_at, status, 'partner', 1, '', confirmed_at FROM partner_requests_in
			WHERE link_id = $1 ORDER BY received_at DESC LIMIT 50 OFFSET $2`, l.ID, (page-1)*50)
	}
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var it PartnerRequestItem
		var attempts int32
		if err := rows.Scan(&it.ID, &it.At, &it.Status, &it.Origin, &attempts, &it.LastError, &it.ConfirmedAt); err != nil {
			return out, err
		}
		it.Attempts = int(attempts)
		out.Items = append(out.Items, it)
	}
	return out, rows.Err()
}
