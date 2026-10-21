package website

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// The member's account page (brief §3.2): their profile (email changed by
// a link to the new address, their name), their plan, their newsletters
// (the preference center's), and their sessions. Later phases add their
// own sections (purchases, gifts, referrals) with RegisterAccountSection.

// AccountList is a newsletter on the account page.
type AccountList struct {
	ID          uuid.UUID
	Name        string
	Description string
	Subscribed  bool
	Pending     bool
}

// AccountSession is one of the member's sessions.
type AccountSession struct {
	ID        uuid.UUID
	Current   bool
	Method    string
	CreatedAt time.Time
	LastSeen  time.Time
	ExpiresAt time.Time
}

// AccountMeter is what the member meter used this month.
type AccountMeter struct {
	Used  int
	Limit int
	// Reads are the posts it counted ("the runbook, 13 Oct").
	Reads []AccountRead
	Next  time.Time
}

// AccountRead is a post the meter counted.
type AccountRead struct {
	Title string
	At    time.Time
}

// Account is the member's account page.
type Account struct {
	ContactID   uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	Since       time.Time
	Tiers       []string
	Plan        *earn.PreferencesPaid
	Meter       *AccountMeter
	Lists       []AccountList
	Frequency   string
	PausedUntil *time.Time
	Sessions    []AccountSession
	SessionDays int
	// Sections are the later phases' sections, in their order.
	Sections []AccountSection
}

// AccountSection is a section another part of ReputeCast adds to the
// account page: its id, title and body (HTML the section draws itself).
type AccountSection struct {
	ID    string
	Title string
	HTML  string
}

// AccountSectionFunc draws a section for a member (ok false: none).
type AccountSectionFunc func(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, l i18n.Lang) (AccountSection, bool, error)

var (
	accountMu       sync.RWMutex
	accountSections []AccountSectionFunc
)

type accountQueriesKey struct{}

// AccountQueries are the queries an account section reads its data with
// (nil outside Account).
func AccountQueries(ctx context.Context) *store.Queries {
	q, _ := ctx.Value(accountQueriesKey{}).(*store.Queries)
	return q
}

// RegisterAccountSection adds a section to the account page (Purchases,
// Gifts, Referrals…), after the built-in ones, in registration order.
func RegisterAccountSection(f AccountSectionFunc) {
	accountMu.Lock()
	defer accountMu.Unlock()
	accountSections = append(accountSections, f)
}

// Account builds the account page of a signed-in member.
func (s *Service) Account(ctx context.Context, ws *store.Workspace, sess *Session, l i18n.Lang) (Account, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return Account{}, err
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, ws, sess.ContactID)
	if err != nil {
		return Account{}, err
	}
	out := Account{ContactID: pc.ID, Email: pc.Email, FirstName: pc.FirstName, LastName: pc.LastName, Frequency: pc.Frequency, PausedUntil: pc.PausedUntil,
		SessionDays: int(c.row.SessionDays), Tiers: []string{}, Lists: []AccountList{}, Sessions: []AccountSession{}, Sections: []AccountSection{}}
	if m, err := s.Q.WebsiteMemberGet(ctx, store.WebsiteMemberGetParams{ContactID: pc.ID, WorkspaceID: ws.ID}); err == nil {
		out.Since = m.FirstSignedIn
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	} else {
		out.Since = sess.CreatedAt
	}
	tiers, err := s.Q.WebsiteReaderTiers(ctx, store.WebsiteReaderTiersParams{WorkspaceID: ws.ID, ContactID: pc.ID})
	if err != nil {
		return out, err
	}
	for _, t := range tiers {
		out.Tiers = append(out.Tiers, t.Name)
	}
	if e := s.earn(); e != nil {
		if plan, err := e.PreferencesPaid(ctx, ws, pc.ID, l); err == nil {
			out.Plan = plan
		} else {
			s.log().WarnContext(ctx, "account plan", "err", err)
		}
	}
	if len(tiers) == 0 && c.row.MeterMembersOn {
		loc := location(ws)
		month := monthOf(s.now(), loc)
		reads, err := s.Q.WebsiteMeterReads(ctx, store.WebsiteMeterReadsParams{WorkspaceID: ws.ID, Month: month,
			Reader: gate.Reader{Member: &gate.Member{ContactID: pc.ID}}.Key(), Rule: gate.RuleMember})
		if err != nil {
			return out, err
		}
		m := &AccountMeter{Used: len(reads), Limit: int(c.row.MeterMembers), Next: month.AddDate(0, 1, 0)}
		for _, r := range reads {
			m.Reads = append(m.Reads, AccountRead{Title: r.Title, At: r.ReadAt})
		}
		out.Meter = m
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	for _, li := range lists {
		st := pc.Lists[li.ID]
		if !li.Public && st != "subscribed" && st != "pending" {
			continue
		}
		out.Lists = append(out.Lists, AccountList{ID: li.ID, Name: li.Name, Description: li.Description, Subscribed: pc.Subscribed(li.ID),
			Pending: st == "pending"})
	}
	sessions, err := s.Q.WebsiteSessionsActive(ctx, store.WebsiteSessionsActiveParams{WorkspaceID: ws.ID, ContactID: pc.ID, Now: s.now()})
	if err != nil {
		return out, err
	}
	for _, se := range sessions {
		out.Sessions = append(out.Sessions, AccountSession{ID: se.ID, Current: se.ID == sess.ID, Method: se.Method, CreatedAt: se.CreatedAt,
			LastSeen: se.LastSeenAt, ExpiresAt: se.ExpiresAt})
	}
	slices.SortStableFunc(out.Sessions, func(a, b AccountSession) int {
		switch {
		case a.Current:
			return -1
		case b.Current:
			return 1
		}
		return b.LastSeen.Compare(a.LastSeen)
	})
	accountMu.RLock()
	extra := slices.Clone(accountSections)
	accountMu.RUnlock()
	// Sections read their data with this site's queries (AccountQueries).
	ctx = context.WithValue(ctx, accountQueriesKey{}, s.Q)
	for _, f := range extra {
		sec, ok, err := f(ctx, ws, pc.ID, l)
		if err != nil {
			s.log().WarnContext(ctx, "account section", "err", err)
			continue
		}
		if ok {
			out.Sections = append(out.Sections, sec)
		}
	}
	return out, nil
}

// SaveName changes the member's name.
func (s *Service) SaveName(ctx context.Context, ws *store.Workspace, sess *Session, first, last string) error {
	first, last = clip(strings.TrimSpace(first), 100), clip(strings.TrimSpace(last), 100)
	_, err := s.Pool.Exec(ctx, `UPDATE contacts SET first_name = $3, last_name = $4, updated_at = $5 WHERE id = $1 AND workspace_id = $2`,
		sess.ContactID, ws.ID, first, last, s.now())
	return err
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// ChangeEmail sends a confirmation link to a member's new address; the
// change happens when it is opened. A taken or suppressed address gets no
// link, and the answer is the same.
func (s *Service) ChangeEmail(ctx context.Context, ws *store.Workspace, sess *Session, newEmail, browser string) (SigninOutcome, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return SigninOutcome{}, err
	}
	out := SigninOutcome{Minutes: int(c.row.LinkMinutes)}
	email, problem := emailaddr.Check(newEmail)
	if problem != "" {
		out.Status, out.Message = SigninInvalid, tr(ctx, "signin.emailInvalid", nil)
		return out, nil
	}
	out.Email, out.Status = email, SigninSent
	if strings.EqualFold(email, sess.Email) {
		return out, nil
	}
	if _, taken, err := s.Audience.FindPublicContact(ctx, s.Q, ws, email); err != nil || taken {
		return out, err
	}
	if sup, err := s.Q.SignupSuppression(ctx, store.SignupSuppressionParams{WorkspaceID: ws.ID, Email: email}); err == nil && sup.Reason != "unsubscribe" {
		return out, nil
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, ws, sess.ContactID)
	if err != nil {
		return out, err
	}
	if wait, err := s.addressLimit(ctx, pc.ID); err != nil || wait > 0 {
		if wait > 0 {
			out.Status, out.RetryIn = SigninLimited, wait
		}
		return out, err
	}
	err = s.sendLink(ctx, ws, c, pc, linkRequest{purpose: PurposeEmail, browser: browser, returnTo: "/account", newEmail: email,
		expires: time.Duration(c.row.LinkMinutes) * time.Minute})
	return out, err
}

// SaveNewsletters applies the account page's newsletters, as the
// preference center does: joining needs no confirmation (the member's
// session proves the address), leaving is immediate.
func (s *Service) SaveNewsletters(ctx context.Context, ws *store.Workspace, sess *Session, in audience.PreferencesInput, ev audience.PublicEvidence) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		_, err := s.Audience.SavePreferences(ctx, tx, ws, sess.ContactID, in, ev)
		return err
	})
}

// UnsubscribeAll takes the member off every newsletter; they stay a member.
func (s *Service) UnsubscribeAll(ctx context.Context, ws *store.Workspace, sess *Session, ev audience.PublicEvidence) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		_, err := s.Audience.Unsubscribe(ctx, tx, ws, sess.ContactID, nil, ev)
		return err
	})
}

// SignOutEverywhereSelf ends every session of the member, from their
// account page.
func (s *Service) SignOutEverywhereSelf(ctx context.Context, ws *store.Workspace, sess *Session) error {
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.WebsiteSessionsEndAll(ctx, store.WebsiteSessionsEndAllParams{At: &now, Reason: "everywhere", WorkspaceID: ws.ID, ContactID: sess.ContactID}); err != nil {
			return err
		}
		return q.WebsiteMemberEventInsert(ctx, store.WebsiteMemberEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: sess.ContactID,
			Kind: "signed_out_everywhere", Detail: "from their account page", At: now})
	})
}
