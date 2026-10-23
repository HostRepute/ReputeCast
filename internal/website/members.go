package website

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// WebsiteMeterUse is how much of the meter a member used this month.
type WebsiteMeterUse struct {
	Used  int64 `json:"used"`
	Limit int   `json:"limit"`
	// Month is the first day of the calendar month.
	Month time.Time `json:"month"`
}

// WebsiteMemberEvent is one of a member's website events (the contact's
// timeline).
type WebsiteMemberEvent struct {
	ID   string    `json:"id"`
	Kind string    `json:"kind" enum:"signed_in,link_refused,signed_out_everywhere,joined,email_changed"`
	At   time.Time `json:"at"`
	// Method is how they signed in (link, code, signup).
	Method      string `json:"method,omitempty"`
	SameBrowser *bool  `json:"sameBrowser,omitempty"`
	// Actor is the person who signed them out everywhere.
	Actor  string `json:"actor,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// WebsiteContactCard is the Website card of a contact's profile.
type WebsiteContactCard struct {
	SiteOn bool   `json:"siteOn"`
	Host   string `json:"host"`
	// Member: they signed in to the website at least once.
	Member bool       `json:"member"`
	Since  *time.Time `json:"since,omitempty"`
	// Plan is free, or the paid tiers they hold.
	Paid       bool       `json:"paid"`
	Tiers      []string   `json:"tiers"`
	LastSignIn *time.Time `json:"lastSignIn,omitempty"`
	LastMethod string     `json:"lastMethod,omitempty" enum:"link,code,signup,"`
	SignIns    int        `json:"signIns"`
	// Sessions are active; SessionEnds is when the latest one ends.
	Sessions    int        `json:"sessions"`
	SessionEnds *time.Time `json:"sessionEnds,omitempty"`
	// Meter is the paid posts a free member read for free this month
	// (when that meter is on).
	Meter  *WebsiteMeterUse     `json:"meter,omitempty"`
	Events []WebsiteMemberEvent `json:"events"`
	// CanSignOut: holds subscribers:edit; Reason says why not.
	CanSignOut bool   `json:"canSignOut"`
	Reason     string `json:"reason,omitempty"`
}

// ContactCard returns a contact's Website card.
func (s *Service) ContactCard(ctx context.Context, ws *store.Workspace, p *authz.Principal, contactID uuid.UUID) (WebsiteContactCard, error) {
	out := WebsiteContactCard{Tiers: []string{}, Events: []WebsiteMemberEvent{}, CanSignOut: p.Can(authz.SubscribersEdit)}
	if !out.CanSignOut {
		out.Reason = authz.DeniedMessage(authz.SubscribersEdit, s.helpers(ctx, ws, authz.SubscribersEdit))
	}
	var exists bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL)`, contactID, ws.ID).Scan(&exists); err != nil {
		return out, err
	}
	if !exists {
		return out, notFound(ctx, "contact")
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	out.SiteOn = c.On()
	out.Host = s.address(ctx, ws, false).Host
	m, err := s.Q.WebsiteMemberGet(ctx, store.WebsiteMemberGetParams{ContactID: contactID, WorkspaceID: ws.ID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return out, err
	default:
		since, last := m.FirstSignedIn, m.LastSignedIn
		out.Member, out.Since, out.LastSignIn, out.LastMethod, out.SignIns = true, &since, &last, m.LastMethod, int(m.SignIns)
	}
	tiers, err := s.Q.WebsiteReaderTiers(ctx, store.WebsiteReaderTiersParams{WorkspaceID: ws.ID, ContactID: contactID})
	if err != nil {
		return out, err
	}
	for _, t := range tiers {
		out.Tiers = append(out.Tiers, t.Name)
	}
	out.Paid = len(tiers) > 0
	sessions, err := s.Q.WebsiteSessionsActive(ctx, store.WebsiteSessionsActiveParams{WorkspaceID: ws.ID, ContactID: contactID, Now: s.now()})
	if err != nil {
		return out, err
	}
	out.Sessions = len(sessions)
	for _, se := range sessions {
		if out.SessionEnds == nil || se.ExpiresAt.After(*out.SessionEnds) {
			t := se.ExpiresAt
			out.SessionEnds = &t
		}
	}
	if out.Member && !out.Paid && c.row.MeterMembersOn {
		month := monthOf(s.now(), location(ws))
		st, err := s.Q.WebsiteMeterState(ctx, store.WebsiteMeterStateParams{WorkspaceID: ws.ID, Month: month, Reader: gate.Reader{Member: &gate.Member{ContactID: contactID}}.Key(),
			Rule: gate.RuleMember, PostID: uuid.Nil})
		if err != nil {
			return out, err
		}
		out.Meter = &WebsiteMeterUse{Used: st.Used, Limit: int(c.row.MeterMembers), Month: month}
	}
	events, err := s.Q.WebsiteMemberEvents(ctx, store.WebsiteMemberEventsParams{WorkspaceID: ws.ID, ContactID: contactID, Lim: 20})
	if err != nil {
		return out, err
	}
	for _, e := range events {
		out.Events = append(out.Events, WebsiteMemberEvent{ID: e.ID.String(), Kind: e.Kind, At: e.OccurredAt, Method: e.Method, SameBrowser: e.SameBrowser,
			Actor: e.ActorName, Detail: memberEventDetailIn(lang(ctx), e.Detail)})
	}
	return out, nil
}

// WebsiteSignedOut is how many sessions signing out everywhere ended.
type WebsiteSignedOut struct {
	Sessions int64 `json:"sessions"`
}

// SignOutEverywhere ends every session of a member, from their contact
// profile (subscribers:edit). It is audited and on their timeline.
func (s *Service) SignOutEverywhere(ctx context.Context, ws *store.Workspace, p *authz.Principal, contactID uuid.UUID) (WebsiteSignedOut, error) {
	var email string
	err := s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, contactID, ws.ID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebsiteSignedOut{}, notFound(ctx, "contact")
	}
	if err != nil {
		return WebsiteSignedOut{}, err
	}
	out := WebsiteSignedOut{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		n, err := q.WebsiteSessionsEndAll(ctx, store.WebsiteSessionsEndAllParams{WorkspaceID: ws.ID, ContactID: contactID, At: &now, Reason: "everywhere"})
		if err != nil {
			return err
		}
		out.Sessions = n
		if err := q.WebsiteMemberEventInsert(ctx, store.WebsiteMemberEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: contactID, Kind: "signed_out_everywhere",
			ActorID: userID(p), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.member_signed_out", TargetType: "contact", TargetID: contactID.String(),
			TargetName: email, After: map[string]any{"sessions_ended": n}})
	})
	return out, err
}

// MemberCount is how many contacts signed in to the website (the contacts
// filter "Signed in to the website").
func (s *Service) MemberCount(ctx context.Context, ws *store.Workspace) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM website_members m JOIN contacts c ON c.id = m.contact_id AND c.erased_at IS NULL
		WHERE m.workspace_id = $1`, ws.ID).Scan(&n)
	return n, err
}
