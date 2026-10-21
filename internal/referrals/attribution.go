package referrals

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Hook records referrals from signups and confirmations (forms.SignupHook).
type Hook struct {
	S func() *Service
}

var _ forms.SignupHook = Hook{}

// CodeOf reads the referral code and the channel of a signup's page: the
// invite page (/join/K7Q2MX) or any page opened with ?ref=K7Q2MX; share
// buttons add ?via=.
func CodeOf(page string) (code, via string) {
	if page == "" {
		return "", ""
	}
	u, err := url.Parse(page)
	if err != nil {
		return "", ""
	}
	q := u.Query()
	via = ChannelOf(q.Get("via"))
	if c, ok := NormalizeCode(q.Get("ref")); ok {
		return c, via
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := len(parts) - 2; i >= 0; i-- {
		if parts[i] == "join" {
			if c, ok := NormalizeCode(parts[i+1]); ok {
				return c, via
			}
		}
	}
	return "", ""
}

// activeLink is a referral link that attributes signups: the program is
// on, the link isn't paused and its referrer is still on the list.
type activeLink struct {
	link    store.ReferralLinkByCodeRow
	program store.ReferralProgram
}

func (s *Service) activeLink(ctx context.Context, q *store.Queries, ws *store.Workspace, code string) (*activeLink, error) {
	l, err := q.ReferralLinkByCode(ctx, store.ReferralLinkByCodeParams{WorkspaceID: ws.ID, Code: code})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p, err := q.ReferralProgramByID(ctx, l.ProgramID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.Status != "on" || l.Status != "active" {
		return nil, nil
	}
	subscribed, err := q.ReferralContactOnList(ctx, store.ReferralContactOnListParams{ListID: p.ListID, ContactID: l.ContactID})
	if err != nil {
		return nil, err
	}
	if !subscribed {
		return nil, nil
	}
	return &activeLink{link: l, program: p}, nil
}

// DoubleOptIn implements forms.SignupHook: a signup through a working
// referral link confirms by email.
func (h Hook) DoubleOptIn(ctx context.Context, ws *store.Workspace, sub forms.Submission) bool {
	code, _ := CodeOf(sub.Page)
	if code == "" {
		return false
	}
	s := h.S()
	al, err := s.activeLink(ctx, s.Q, ws, code)
	return err == nil && al != nil
}

// Accepted implements forms.SignupHook.
func (h Hook) Accepted(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e forms.SignupEvent) error {
	return h.S().recordSignup(ctx, tx, ws, e)
}

// Confirmed implements forms.SignupHook.
func (h Hook) Confirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e forms.ConfirmEvent) error {
	return h.S().recordConfirmed(ctx, tx, ws, e)
}

// recordSignup records a referral when a signup came through a working
// link and asked for the program's list. Only new subscribers can be
// referrals: an address that is or ever was on the list isn't counted.
func (s *Service) recordSignup(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e forms.SignupEvent) error {
	code, via := CodeOf(e.Page)
	if code == "" {
		return nil
	}
	q := store.New(tx)
	al, err := s.activeLink(ctx, q, ws, code)
	if err != nil || al == nil {
		return err
	}
	p := al.program
	if !slices.Contains(e.Lists, p.ListID) {
		return nil
	}
	if _, err := q.ReferralOfContact(ctx, store.ReferralOfContactParams{ProgramID: p.ID, ContactID: &e.ContactID}); err == nil {
		return nil // one referrer per signup: the first link wins
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	status, reason := StatusAwaiting, ""
	switch {
	case e.ContactID == al.link.ContactID || SameMailbox(e.Email, al.link.ContactEmail):
		status, reason = StatusNotCounted, ReasonSelf
	case slices.Contains(e.Already, p.ListID):
		status, reason = StatusNotCounted, ReasonAlready
	default:
		var before bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM consent_events WHERE contact_id = $1 AND list_id = $2 AND occurred_at < $3)
			OR EXISTS (SELECT 1 FROM list_memberships WHERE contact_id = $1 AND list_id = $2 AND created_at < $3 - interval '1 second')`,
			e.ContactID, p.ListID, e.At).Scan(&before); err != nil {
			return err
		}
		if before {
			status, reason = StatusNotCounted, ReasonAlready
		}
	}
	id := uuid.Must(uuid.NewV7())
	var ip *netip.Addr
	network := ""
	if e.IP != nil && e.IP.IsValid() {
		a := e.IP.Unmap()
		ip, network = &a, NetworkOf(a)
	}
	var notCounted *time.Time
	if status == StatusNotCounted {
		notCounted = &e.At
	}
	linkID, referrer := al.link.ID, al.link.ContactID
	if err := q.ReferralInsert(ctx, store.ReferralInsertParams{ID: id, WorkspaceID: ws.ID, ProgramID: p.ID, LinkID: &linkID, ReferrerID: &referrer,
		ContactID: &e.ContactID, Status: status, Reason: reason, Via: via, Ip: ip, IpNetwork: network, Now: e.At, NotCountedAt: notCounted}); err != nil {
		return err
	}
	if err := q.ReferralLinkBump(ctx, store.ReferralLinkBumpParams{ID: linkID, Signups: 1}); err != nil {
		return err
	}
	if err := q.ReferralVisitBump(ctx, store.ReferralVisitBumpParams{LinkID: linkID, ProgramID: p.ID, Day: dayOf(e.At, workspaceLocation(ws)), Via: via, Signups: 1}); err != nil {
		return err
	}
	contact := e.ContactID
	if err := writeLog(ctx, q, ws.ID, logEntry{program: p.ID, referral: &id, link: &linkID, contact: &contact, kind: EventSignedUp, at: e.At,
		detail: map[string]any{"referrer": referrer.String(), "via": via, "status": status, "reason": reason}}); err != nil {
		return err
	}
	// A contact this signup created came through the link: the contacts
	// list reads "Referral · Sam Rivera".
	if e.Created {
		if _, err := tx.Exec(ctx, `UPDATE contacts SET source = 'referral' WHERE id = $1`, e.ContactID); err != nil {
			return err
		}
	}
	if status == StatusNotCounted {
		r, err := q.ReferralGet(ctx, store.ReferralGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		return emitHook(ctx, tx, ws, HookNotCounted, map[string]any{"program": p.ID.String(), "referral": referralHook(r),
			"referrer": hookContact{ContactID: referrer.String(), Email: al.link.ContactEmail}})
	}
	return nil
}

// recordConfirmed starts the hold of the contact's referrals on the lists
// they confirmed, after the exclusions and the suspicion rules; and gives
// a new confirmed subscriber of a program's list their own link.
func (s *Service) recordConfirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e forms.ConfirmEvent) error {
	q := store.New(tx)
	rows, err := q.ReferralAwaitingOfContact(ctx, &e.ContactID)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if !slices.Contains(e.Lists, r.ListID) {
			continue
		}
		if err := s.startHold(ctx, tx, ws, r, e.At); err != nil {
			return err
		}
	}
	// New subscribers get their own link while a program is on.
	progs, err := q.ReferralProgramsOf(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, p := range progs {
		if p.Status != "on" || !slices.Contains(e.Lists, p.ListID) {
			continue
		}
		if _, err := insertLinks(ctx, tx, store.ReferralProgram{ID: p.ID, WorkspaceID: p.WorkspaceID, ListID: p.ListID}, []uuid.UUID{e.ContactID}, e.At); err != nil {
			return err
		}
	}
	return nil
}

// startHold checks a referral that just confirmed: excluded, held for
// review, or into its hold (counted at once with a hold of 0 days).
func (s *Service) startHold(ctx context.Context, tx pgx.Tx, ws *store.Workspace, r store.ReferralAwaitingOfContactRow, at time.Time) error {
	q := store.New(tx)
	email := ""
	if r.ContactID != nil {
		if err := tx.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1`, *r.ContactID).Scan(&email); err != nil {
			return err
		}
	}
	reason := ""
	if r.RuleSameIp && r.Ip != nil && r.LinkID != nil && r.ReferrerID != nil {
		match, err := q.ReferralReferrerIPMatch(ctx, store.ReferralReferrerIPMatchParams{LinkID: *r.LinkID, Addr: *r.Ip, ReferrerID: *r.ReferrerID,
			Since: at.Add(-SameIPWindow)})
		if err != nil {
			return err
		}
		if match {
			reason = ReasonSameIP
		}
	}
	if reason == "" && r.RuleDisposable && emailaddr.IsDisposable(emailaddr.Domain(email)) {
		reason = ReasonDisposable
	}
	var signals []string
	var also []uuid.UUID // referrals in hold that the same rule now holds
	if reason == "" && r.LinkID != nil {
		if r.BurstOn {
			recent, err := q.ReferralRecentConfirmed(ctx, store.ReferralRecentConfirmedParams{LinkID: r.LinkID, Since: at.Add(-time.Hour)})
			if err != nil {
				return err
			}
			if len(recent)+1 > int(r.BurstThreshold) {
				signals = append(signals, SignalBurst)
				for _, x := range recent {
					also = append(also, x.ID)
				}
			}
		}
		if r.SharedIpOn && r.Ip != nil {
			same, err := q.ReferralRecentFromIP(ctx, store.ReferralRecentFromIPParams{LinkID: r.LinkID, Addr: *r.Ip, Since: at.Add(-24 * time.Hour)})
			if err != nil {
				return err
			}
			// This referral is among them (awaiting).
			if len(same) >= int(r.SharedIpThreshold) {
				signals = append(signals, SignalSharedIP)
				for _, x := range same {
					if x.ID != r.ID {
						also = append(also, x.ID)
					}
				}
			}
		}
		if r.NumberedOn {
			if key := NumberedKey(email); key != "" {
				peers, err := q.ReferralNumberedPeers(ctx, store.ReferralNumberedPeersParams{LinkID: r.LinkID, ID: r.ID, Since: at.Add(-30 * 24 * time.Hour)})
				if err != nil {
					return err
				}
				matched := false
				for _, pr := range peers {
					if NumberedKey(pr.Email) == key && !SameMailbox(pr.Email, email) {
						matched = true
						also = append(also, pr.ID)
					}
				}
				if matched {
					signals = append(signals, SignalNumbered)
				}
			}
		}
	}
	status := StatusHold
	var holdUntil *time.Time
	switch {
	case reason != "":
		status = StatusNotCounted
	case len(signals) > 0:
		status = StatusReview
	default:
		h := at.Add(time.Duration(r.HoldDays) * 24 * time.Hour)
		holdUntil = &h
	}
	if len(signals) == 0 {
		signals = []string{}
	}
	if err := q.ReferralSetConfirmed(ctx, store.ReferralSetConfirmedParams{ID: r.ID, ConfirmedAt: at, Status: status, Reason: reason, Signals: signals,
		HoldUntil: holdUntil}); err != nil {
		return err
	}
	if len(signals) > 0 && len(also) > 0 && r.LinkID != nil {
		// The earlier referrals that make the burst, share the address or
		// the number pattern wait for a person too.
		if _, err := q.ReferralHoldToReview(ctx, store.ReferralHoldToReviewParams{LinkID: r.LinkID, Ids: also, Signals: signals, Now: at}); err != nil {
			return err
		}
		if err := q.ReferralAddSignals(ctx, store.ReferralAddSignalsParams{Ids: also, Signals: signals, Now: at}); err != nil {
			return err
		}
	}
	kind := EventConfirmed
	switch status {
	case StatusNotCounted:
		kind = EventNotCounted
	case StatusReview:
		kind = EventHeld
	}
	if err := writeLog(ctx, q, ws.ID, logEntry{program: r.ProgramID, referral: &r.ID, link: r.LinkID, contact: r.ContactID, kind: kind, at: at,
		detail: map[string]any{"reason": reason, "signals": signals, "countsOn": holdUntil}}); err != nil {
		return err
	}
	ref, err := q.ReferralGet(ctx, store.ReferralGetParams{WorkspaceID: ws.ID, ID: r.ID})
	if err != nil {
		return err
	}
	referrer := hookContact{}
	if r.ReferrerID != nil {
		referrer.ContactID = r.ReferrerID.String()
	}
	switch status {
	case StatusNotCounted:
		return emitHook(ctx, tx, ws, HookNotCounted, map[string]any{"program": r.ProgramID.String(), "referral": referralHook(ref), "referrer": referrer})
	case StatusReview:
		return emitHook(ctx, tx, ws, HookHeld, map[string]any{"program": r.ProgramID.String(), "referral": referralHook(ref), "referrer": referrer})
	}
	if r.HoldDays == 0 {
		return s.countReferral(ctx, tx, ws, r.ID, at)
	}
	return nil
}

// --- Visits --------------------------------------------------------------------

// botAgents are crawlers, link previews and scanners: their visits are left
// out (a shared link is fetched by the app it was shared in).
var botAgents = []string{"bot", "crawler", "spider", "preview", "facebookexternalhit", "whatsapp", "telegram", "slack", "discord", "linkedin",
	"twitter", "embedly", "skype", "googleimageproxy", "yahoomailproxy", "barracuda", "mimecast", "proofpoint", "safelinks", "curl/", "wget/",
	"python-requests", "go-http-client", "headless"}

// IsBot reports whether a visit came from a bot, a link preview or a scanner.
func IsBot(method, userAgent string) bool {
	if strings.EqualFold(method, "HEAD") {
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return true
	}
	for _, b := range botAgents {
		if strings.Contains(ua, b) {
			return true
		}
	}
	return false
}

// RecordVisit counts a visit to a link's invite page: once per network and
// browser a day, without cookies; bots are left out.
func (s *Service) RecordVisit(ctx context.Context, ws *store.Workspace, linkID, programID uuid.UUID, via, method, userAgent string, ip *netip.Addr) error {
	if IsBot(method, userAgent) {
		return nil
	}
	now := s.now()
	day := dayOf(now, workspaceLocation(ws))
	network := ""
	if ip != nil {
		network = NetworkOf(*ip)
	}
	key := linkID[:]
	if s.Links != nil {
		key = s.Links.Digest(linkID[:], []byte(network), []byte(userAgent), []byte(day.Format("2006-01-02")))
	}
	if _, err := s.Q.ReferralVisitKey(ctx, store.ReferralVisitKeyParams{Key: key, Now: now}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already counted today
		}
		return err
	}
	if err := s.Q.ReferralVisitBump(ctx, store.ReferralVisitBumpParams{LinkID: linkID, ProgramID: programID, Day: day, Via: ChannelOf(via), Visits: 1}); err != nil {
		return err
	}
	return s.Q.ReferralLinkBump(ctx, store.ReferralLinkBumpParams{ID: linkID, Visits: 1})
}

// SeenReferrer records an address a referrer used (their hub), for the
// "same IP address as the referrer" rule.
func (s *Service) SeenReferrer(ctx context.Context, linkID uuid.UUID, ip *netip.Addr) {
	if ip == nil || !ip.IsValid() {
		return
	}
	if err := s.Q.ReferralLinkSeenIP(ctx, store.ReferralLinkSeenIPParams{LinkID: linkID, Ip: ip.Unmap(), Now: s.now()}); err != nil {
		s.log().WarnContext(ctx, "referrals: record the referrer's address", "err", err)
	}
}
