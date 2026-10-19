package website

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// WebsiteAccessList is a list as Website › Access offers it on sign-up.
type WebsiteAccessList struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Public  bool   `json:"public"`
	Offered bool   `json:"offered"`
	Checked bool   `json:"checked"`
	// Consent is the consent text shown with it (kept with every sign-up).
	Consent string `json:"consent"`
}

// WebsiteTierOption is a paid tier as the paywall card can offer it.
type WebsiteTierOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Label string `json:"label"`
	// Monthly and Yearly are the current prices in cents (0: none), with
	// their currency; TrialDays the free days.
	Monthly   int64  `json:"monthly"`
	Yearly    int64  `json:"yearly"`
	Currency  string `json:"currency"`
	TrialDays int    `json:"trialDays"`
	// Includes names the lower tier whose content it includes ("Insider").
	Includes string `json:"includes,omitempty"`
	Public   bool   `json:"public"`
}

// WebsiteNeverFreePost is a post left out of the meter.
type WebsiteNeverFreePost struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Visibility  string     `json:"visibility"`
	Tiers       []string   `json:"tiers"`
	Status      string     `json:"status"`
	PublishAt   *time.Time `json:"publishAt,omitempty"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

// WebsitePayments is the payment connection the paid tiers sell through.
type WebsitePayments struct {
	Connected bool   `json:"connected"`
	Mode      string `json:"mode" enum:"live,test,"`
	Account   string `json:"account"`
}

// WebsiteAccess is Website › Access.
type WebsiteAccess struct {
	SigninOn    bool `json:"signinOn"`
	LinkMinutes int  `json:"linkMinutes"`
	SessionDays int  `json:"sessionDays"`
	// The sign-in email's sender and stream (the workspace default, System).
	SenderName  string `json:"senderName"`
	SenderEmail string `json:"senderEmail"`
	// SystemProvider names the System stream's provider ("Postmark").
	SystemProvider     string                 `json:"systemProvider"`
	SignupOn           bool                   `json:"signupOn"`
	Lists              []WebsiteAccessList    `json:"lists"`
	DefaultVisibility  string                 `json:"defaultVisibility" enum:"public,members,paid"`
	CampaignVisibility string                 `json:"campaignVisibility" enum:"audience,public,members,paid"`
	MeterVisitorsOn    bool                   `json:"meterVisitorsOn"`
	MeterVisitors      int                    `json:"meterVisitors"`
	MeterMembersOn     bool                   `json:"meterMembersOn"`
	MeterMembers       int                    `json:"meterMembers"`
	NeverFree          []WebsiteNeverFreePost `json:"neverFree"`
	Paywall            WebsitePaywall         `json:"paywall"`
	Tiers              []WebsiteTierOption    `json:"tiers"`
	Payments           WebsitePayments        `json:"payments"`
	Members            WebsiteMembersFacts    `json:"members"`
	// Gated are the published members-only and paid posts (turning
	// sign-in off makes them unreadable on the web).
	MembersOnlyPosts int64     `json:"membersOnlyPosts"`
	PaidPosts        int64     `json:"paidPosts"`
	Since            time.Time `json:"since"`
	UpdatedAt        time.Time `json:"updatedAt"`
	UpdatedBy        string    `json:"updatedBy,omitempty"`
}

// WebsiteAccessInput saves Website › Access.
type WebsiteAccessInput struct {
	SigninOn           bool                `json:"signinOn"`
	LinkMinutes        int                 `json:"linkMinutes" enum:"10,15,30,60"`
	SessionDays        int                 `json:"sessionDays" minimum:"7" maximum:"365"`
	SignupOn           bool                `json:"signupOn"`
	Lists              []WebsiteSignupList `json:"lists" maxItems:"50"`
	DefaultVisibility  string              `json:"defaultVisibility" enum:"public,members,paid"`
	CampaignVisibility string              `json:"campaignVisibility" enum:"audience,public,members,paid"`
	MeterVisitorsOn    bool                `json:"meterVisitorsOn"`
	MeterVisitors      int                 `json:"meterVisitors" minimum:"0" maximum:"100"`
	MeterMembersOn     bool                `json:"meterMembersOn"`
	MeterMembers       int                 `json:"meterMembers" minimum:"0" maximum:"100"`
	// NeverFree are the posts left out of the meter.
	NeverFree []string       `json:"neverFree" maxItems:"200"`
	Paywall   WebsitePaywall `json:"paywall"`
}

// accessLists are the workspace's lists with what sign-up offers.
func (s *Service) accessLists(ctx context.Context, ws *store.Workspace, c *config) ([]WebsiteAccessList, error) {
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	out := make([]WebsiteAccessList, 0, len(lists))
	for _, l := range lists {
		v := WebsiteAccessList{ID: l.ID.String(), Name: l.Name, Public: l.Public}
		for _, sl := range c.SignupLists {
			if sl.ListID == v.ID {
				v.Offered, v.Checked, v.Consent = true, sl.Checked, sl.Consent
			}
		}
		if v.Consent == "" {
			v.Consent = defaultConsent(l.Name)
		}
		out = append(out, v)
	}
	return out, nil
}

// defaultConsent is a list's consent text until someone writes one.
func defaultConsent(list string) string {
	return "Send me " + list + ". Unsubscribe any time."
}

// tierOptions are the paid tiers with their current prices.
func (s *Service) tierOptions(ctx context.Context, q *store.Queries, ws *store.Workspace) ([]WebsiteTierOption, error) {
	rows, err := q.WebsiteTierPrices(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []WebsiteTierOption{}
	idx := map[uuid.UUID]int{}
	for _, r := range rows {
		i, ok := idx[r.ID]
		if !ok {
			out = append(out, WebsiteTierOption{ID: r.ID.String(), Name: r.Name, Label: TierLabel(r.Name), TrialDays: int(r.TrialDays), Currency: r.Currency,
				Public: r.Visibility == "public"})
			i = len(out) - 1
			idx[r.ID] = i
		}
		switch r.Period {
		case "month":
			out[i].Monthly = r.AmountCents
		case "year":
			out[i].Yearly = r.AmountCents
		}
	}
	// A higher tier includes the content of the tier below it.
	tiers, err := s.tiers(ctx, q, ws)
	if err != nil {
		return nil, err
	}
	for i := range out {
		var me *gate.Tier
		for j := range tiers {
			if tiers[j].ID.String() == out[i].ID {
				me = &tiers[j]
			}
		}
		if me == nil {
			continue
		}
		var below *gate.Tier
		for j := range tiers {
			if tiers[j].Rank < me.Rank && (below == nil || tiers[j].Rank > below.Rank) {
				below = &tiers[j]
			}
		}
		if below != nil {
			out[i].Includes = below.Label
		}
	}
	return out, nil
}

// Access returns Website › Access.
func (s *Service) Access(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WebsiteAccess, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteAccess{}, err
	}
	out := WebsiteAccess{SigninOn: c.row.SigninOn, LinkMinutes: int(c.row.LinkMinutes), SessionDays: int(c.row.SessionDays), SignupOn: c.row.SignupOn,
		DefaultVisibility: c.row.DefaultVisibility, CampaignVisibility: c.row.CampaignVisibility, MeterVisitorsOn: c.row.MeterVisitorsOn,
		MeterVisitors: int(c.row.MeterVisitors), MeterMembersOn: c.row.MeterMembersOn, MeterMembers: int(c.row.MeterMembers), Paywall: c.Paywall,
		NeverFree: []WebsiteNeverFreePost{}, UpdatedAt: c.row.UpdatedAt}
	if by, err := s.Q.WebsiteSettingsUpdatedBy(ctx, ws.ID); err == nil {
		out.UpdatedBy = by
	}
	out.SenderName, out.SenderEmail = s.Forms.DefaultSender(ctx, ws)
	out.SystemProvider = s.Forms.SystemProviderName(ctx, ws)
	if out.Lists, err = s.accessLists(ctx, ws, c); err != nil {
		return out, err
	}
	if out.Tiers, err = s.tierOptions(ctx, s.Q, ws); err != nil {
		return out, err
	}
	pay, err := s.Q.WebsitePayments(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Payments = WebsitePayments{Connected: pay.Mode != "", Mode: pay.Mode, Account: pay.Account}
	if out.Members, err = s.membersFacts(ctx, ws, c); err != nil {
		return out, err
	}
	if c.row.TurnedOnAt != nil {
		out.Since = *c.row.TurnedOnAt
	}
	counts, err := s.counts(ctx, ws)
	if err != nil {
		return out, err
	}
	out.MembersOnlyPosts, out.PaidPosts = counts.MembersOnly, counts.Paid
	rows, err := s.Pool.Query(ctx, `SELECT id, title, visibility, tiers, status, publish_at, published_at FROM website_posts
		WHERE workspace_id = $1 AND deleted_at IS NULL AND never_free ORDER BY coalesce(published_at, publish_at, created_at) DESC`, ws.ID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	tiers, _ := s.tiers(ctx, s.Q, ws)
	for rows.Next() {
		var n WebsiteNeverFreePost
		var id uuid.UUID
		var tids []uuid.UUID
		if err := rows.Scan(&id, &n.Title, &n.Visibility, &tids, &n.Status, &n.PublishAt, &n.PublishedAt); err != nil {
			return out, err
		}
		n.ID = id.String()
		n.Tiers = []string{}
		for _, t := range tierRefs(tids, tiers) {
			n.Tiers = append(n.Tiers, t.Name)
		}
		out.NeverFree = append(out.NeverFree, n)
	}
	return out, rows.Err()
}

// SaveAccess saves Website › Access, with an audit entry per part changed.
func (s *Service) SaveAccess(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsiteAccessInput) (WebsiteAccess, error) {
	if in.MeterVisitorsOn && (in.MeterVisitors < 1 || in.MeterVisitors > 20) {
		return WebsiteAccess{}, api.Invalid("body.meterVisitors", tr(ctx, "meter.range", nil))
	}
	if in.MeterMembersOn && (in.MeterMembers < 1 || in.MeterMembers > 20) {
		return WebsiteAccess{}, api.Invalid("body.meterMembers", tr(ctx, "meter.range", nil))
	}
	if !gate.ValidLevel(in.DefaultVisibility) {
		return WebsiteAccess{}, api.Invalid("body.defaultVisibility", tr(ctx, "post.visibilityInvalid", nil))
	}
	switch in.LinkMinutes {
	case 10, 15, 30, 60:
	default:
		return WebsiteAccess{}, api.Invalid("body.linkMinutes", tr(ctx, "access.linkMinutes", nil))
	}
	if in.SessionDays < 7 || in.SessionDays > 365 {
		return WebsiteAccess{}, api.Invalid("body.sessionDays", tr(ctx, "access.sessionDays", nil))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.lock(ctx, q, ws)
		if err != nil {
			return err
		}
		before := *c
		lists, err := s.Audience.PublicLists(ctx, q, ws)
		if err != nil {
			return err
		}
		next := []WebsiteSignupList{}
		for _, l := range in.Lists {
			known := false
			for _, x := range lists {
				if x.ID.String() == l.ListID {
					known = true
				}
			}
			if !known {
				return api.Invalid("body.lists", tr(ctx, "access.listUnknown", nil))
			}
			l.Consent = strings.TrimSpace(l.Consent)
			next = append(next, l)
		}
		if in.Paywall.Heading = strings.TrimSpace(in.Paywall.Heading); in.Paywall.Heading == "" {
			return api.Invalid("body.paywall.heading", tr(ctx, "paywall.headingRequired", nil))
		}
		if in.Paywall.Tiers == nil {
			in.Paywall.Tiers = []string{}
		}
		tiers, err := s.tiers(ctx, q, ws)
		if err != nil {
			return err
		}
		for _, t := range in.Paywall.Tiers {
			if !slices.ContainsFunc(tiers, func(x gate.Tier) bool { return x.ID.String() == t }) {
				return api.Invalid("body.paywall.tiers", tr(ctx, "post.tierUnknown", nil))
			}
		}
		c.row.SigninOn, c.row.LinkMinutes, c.row.SessionDays = in.SigninOn, int32(in.LinkMinutes), int32(in.SessionDays)
		c.row.SignupOn, c.SignupLists = in.SignupOn, next
		c.row.DefaultVisibility, c.row.CampaignVisibility = in.DefaultVisibility, in.CampaignVisibility
		c.row.MeterVisitorsOn, c.row.MeterMembersOn = in.MeterVisitorsOn, in.MeterMembersOn
		if in.MeterVisitors >= 1 && in.MeterVisitors <= 20 {
			c.row.MeterVisitors = int32(in.MeterVisitors)
		}
		if in.MeterMembers >= 1 && in.MeterMembers <= 20 {
			c.row.MeterMembers = int32(in.MeterMembers)
		}
		c.Paywall = in.Paywall
		if err := s.save(ctx, q, ws, c, userID(p)); err != nil {
			return err
		}
		// Never free: the posts left out of the meter.
		ids := []uuid.UUID{}
		for _, raw := range in.NeverFree {
			id, err := uuid.Parse(raw)
			if err != nil {
				return api.Invalid("body.neverFree", tr(ctx, "post.notFound", nil))
			}
			ids = append(ids, id)
		}
		tag, err := tx.Exec(ctx, `UPDATE website_posts SET never_free = (id = ANY ($2) AND visibility <> 'public')
			WHERE workspace_id = $1 AND deleted_at IS NULL AND never_free <> (id = ANY ($2) AND visibility <> 'public')`, ws.ID, ids)
		if err != nil {
			return err
		}
		return s.auditAccess(ctx, q, ws, &before, c, tag.RowsAffected())
	})
	if err != nil {
		return WebsiteAccess{}, err
	}
	return s.Access(ctx, ws, p)
}

// auditAccess writes what changed in Website › Access: sign-in and sign-up,
// the meter, the paywall card.
func (s *Service) auditAccess(ctx context.Context, q *store.Queries, ws *store.Workspace, before, after *config, neverFree int64) error {
	log := func(action string, b, a map[string]any) error {
		if fmt.Sprint(b) == fmt.Sprint(a) {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "website", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: b, After: a})
	}
	lists := func(c *config) string { b, _ := json.Marshal(c.SignupLists); return string(b) }
	if err := log("website.access_updated",
		map[string]any{"signin_on": before.row.SigninOn, "link_minutes": before.row.LinkMinutes, "session_days": before.row.SessionDays, "signup_on": before.row.SignupOn,
			"signup_lists": lists(before), "default_visibility": before.row.DefaultVisibility, "campaign_visibility": before.row.CampaignVisibility},
		map[string]any{"signin_on": after.row.SigninOn, "link_minutes": after.row.LinkMinutes, "session_days": after.row.SessionDays, "signup_on": after.row.SignupOn,
			"signup_lists": lists(after), "default_visibility": after.row.DefaultVisibility, "campaign_visibility": after.row.CampaignVisibility}); err != nil {
		return err
	}
	meterB := map[string]any{"visitors_on": before.row.MeterVisitorsOn, "visitors": before.row.MeterVisitors, "members_on": before.row.MeterMembersOn, "members": before.row.MeterMembers}
	meterA := map[string]any{"visitors_on": after.row.MeterVisitorsOn, "visitors": after.row.MeterVisitors, "members_on": after.row.MeterMembersOn, "members": after.row.MeterMembers}
	if neverFree > 0 {
		meterA["never_free_changed"] = neverFree
	}
	if err := log("website.meter_updated", meterB, meterA); err != nil {
		return err
	}
	pw := func(c *config) map[string]any {
		return map[string]any{"heading": c.Paywall.Heading, "text": c.Paywall.Text, "tiers": c.Paywall.Tiers, "upgrade": c.Paywall.Upgrade,
			"signup": c.Paywall.Signup, "free": c.Paywall.Free}
	}
	return log("website.paywall_updated", pw(before), pw(after))
}
