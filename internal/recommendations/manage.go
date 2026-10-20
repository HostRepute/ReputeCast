package recommendations

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

// RecImage is a logo from the media library.
type RecImage struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

// RecommendationItem is a row of "You recommend".
type RecommendationItem struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind" enum:"link,partner"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Initials    string    `json:"initials"`
	Logo        *RecImage `json:"logo,omitempty"`
	// URL is a link's subscribe page; Host what a member recognizes
	// ("fabrikam.example/newsletter", "reputecast.opsweekly.example").
	URL         string `json:"url"`
	Host        string `json:"host"`
	ShowConfirm bool   `json:"showConfirm"`
	ShowBlock   bool   `json:"showBlock"`
	Position    int    `json:"position"`
	// State: a link shows or is paused; a partner is connected, paused by
	// us or by them, invalid (signature rejected or ended by them), at its
	// daily limit, or unreachable (requests failing, still shown).
	State  string `json:"state" enum:"showing,paused,connected,partner_paused,invalid,limit,unreachable,ended"`
	Hidden bool   `json:"hidden"`
	// Since: connected, paused, invalid or failing since.
	Since         *time.Time `json:"since,omitempty"`
	PausedBy      string     `json:"pausedBy"`
	InvalidReason string     `json:"invalidReason" enum:"signature_rejected,ended,unknown_link,"`
	LimitUntil    *time.Time `json:"limitUntil,omitempty"`
	// Today and DailyLimit: partner requests accepted today (UTC) of the
	// partner's limit for our readers.
	Today      int64 `json:"today"`
	DailyLimit int   `json:"dailyLimit"`
	// Waiting requests are retried (an unreachable partner).
	Waiting     int64      `json:"waiting"`
	LinkID      string     `json:"linkId"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty"`
	ConnectedBy string     `json:"connectedBy"`
	// Numbers for the last 30 days.
	Shown        int64 `json:"shown"`
	ShownConfirm int64 `json:"shownConfirm"`
	ShownEmail   int64 `json:"shownEmail"`
	Clicks       int64 `json:"clicks"`
	Requests     int64 `json:"requests"`
	Confirmed    int64 `json:"confirmed"`
}

// RecommendingYouItem is a partner that recommends this workspace.
type RecommendingYouItem struct {
	LinkID      string    `json:"linkId"`
	Name        string    `json:"name"`
	Instance    string    `json:"instance"`
	Label       string    `json:"label"`
	ListName    string    `json:"listName"`
	Since       time.Time `json:"since"`
	ConnectedBy string    `json:"connectedBy"`
	// Requests and Confirmed in the last 30 days; Today since 00:00 UTC.
	Requests   int64      `json:"requests"`
	Confirmed  int64      `json:"confirmed"`
	Today      int64      `json:"today"`
	DailyLimit int        `json:"dailyLimit"`
	State      string     `json:"state" enum:"connected,paused,partner_paused,invalid"`
	PausedBy   string     `json:"pausedBy"`
	PausedAt   *time.Time `json:"pausedAt,omitempty"`
}

// PartnerCodeItem is a partner code not used yet.
type PartnerCodeItem struct {
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	ListName   string    `json:"listName"`
	DailyLimit int       `json:"dailyLimit"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// RecBlockUse is an email with the Recommendations block.
type RecBlockUse struct {
	EmailID string `json:"emailId"`
	// Automation names the flow ("Welcome series"), Name the email ("You're in").
	Automation   string `json:"automation"`
	AutomationID string `json:"automationId"`
	Name         string `json:"name"`
}

// RecProfile is the partner profile: what partners see and show.
type RecProfile struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Initials        string    `json:"initials"`
	Logo            *RecImage `json:"logo,omitempty"`
	ListID          string    `json:"listId"`
	ListName        string    `json:"listName"`
	ListSubscribers int64     `json:"listSubscribers"`
	Instance        string    `json:"instance"`
}

// RecListOption is a list a partner code can offer.
type RecListOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Subscribers int64  `json:"subscribers"`
}

// RecommendationsView is Grow › Recommendations.
type RecommendationsView struct {
	Items           []RecommendationItem  `json:"items"`
	RecommendingYou []RecommendingYouItem `json:"recommendingYou"`
	Codes           []PartnerCodeItem     `json:"codes"`
	// ConfirmPage is the confirmation page's switch; ConfirmShows the
	// names it shows now (up to 3).
	ConfirmPage  bool          `json:"confirmPage"`
	ConfirmShows []string      `json:"confirmShows"`
	Blocks       []RecBlockUse `json:"blocks"`
	// BlockShows is how many recommendations the block shows now.
	BlockShows int                      `json:"blockShows"`
	PageURL    string                   `json:"pageUrl"`
	Profile    RecProfile               `json:"profile"`
	Lists      []RecListOption          `json:"lists"`
	Helpers    referrals.GrowHelperSets `json:"helpers"`
}

// visibility is a recommendation's state on readers' pages now.
func visibility(r store.Recommendation, l *store.PartnerLink, now time.Time) (state string, hidden bool) {
	if r.Kind == KindLink {
		if r.Status == "paused" {
			return "paused", true
		}
		return "showing", false
	}
	if l == nil {
		return "ended", true
	}
	switch l.Status {
	case LinkPaused:
		if l.PausedSide == "remote" {
			return "partner_paused", true
		}
		return "paused", true
	case LinkInvalid:
		return "invalid", true
	case LinkEnded:
		return "ended", true
	}
	if r.Status == "paused" {
		return "paused", true
	}
	if l.LimitUntil != nil && l.LimitUntil.After(now) {
		return "limit", true
	}
	if l.FailingSince != nil {
		return "unreachable", false
	}
	return "connected", false
}

// View is Grow › Recommendations: what this workspace recommends, who
// recommends it, where recommendations show and the partner profile.
func (s *Service) View(ctx context.Context, ws *store.Workspace) (RecommendationsView, error) {
	now := s.now()
	out := RecommendationsView{Items: []RecommendationItem{}, RecommendingYou: []RecommendingYouItem{}, Codes: []PartnerCodeItem{}, ConfirmShows: []string{},
		Blocks: []RecBlockUse{}, Lists: []RecListOption{}, Helpers: referrals.GrowHelperSets{
			Edit: referrals.GrowHelpers(ctx, s.Q, ws.ID, authz.GrowEdit), Connect: referrals.GrowHelpers(ctx, s.Q, ws.ID, authz.GrowConnect),
			Review: []string{}, Export: []string{}, Earn: []string{}}}
	items, err := s.items(ctx, ws, now)
	if err != nil {
		return out, err
	}
	out.Items = items
	set, err := s.settings(ctx, ws)
	if err != nil {
		return out, err
	}
	out.ConfirmPage = set.ConfirmPage
	for _, it := range items {
		if !it.Hidden && it.ShowConfirm && len(out.ConfirmShows) < ConfirmPageMax {
			out.ConfirmShows = append(out.ConfirmShows, it.Name)
		}
		if !it.Hidden && it.ShowBlock {
			out.BlockShows++
		}
	}
	if !set.ConfirmPage {
		out.ConfirmShows = []string{}
	}
	if out.RecommendingYou, err = s.recommendingYou(ctx, ws, now); err != nil {
		return out, err
	}
	codes, err := s.Q.PartnerCodesOpen(ctx, store.PartnerCodesOpenParams{WorkspaceID: ws.ID, Now: now})
	if err != nil {
		return out, err
	}
	for _, c := range codes {
		out.Codes = append(out.Codes, PartnerCodeItem{ID: c.ID.String(), Label: c.Label, ListName: c.ListName, DailyLimit: int(c.DailyLimit),
			CreatedAt: c.CreatedAt, CreatedBy: c.CreatedByName, ExpiresAt: c.ExpiresAt})
	}
	if out.Blocks, err = s.blockUses(ctx, ws); err != nil {
		return out, err
	}
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return out, err
	}
	out.PageURL = strings.TrimPrefix(strings.TrimPrefix(pages.Recommends(""), "https://"), "http://")
	if out.Profile, err = s.profile(ctx, ws, set); err != nil {
		return out, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT l.id, l.name, (SELECT count(*) FROM list_memberships m JOIN contacts c ON c.id = m.contact_id AND c.erased_at IS NULL
			WHERE m.list_id = l.id AND m.status = 'subscribed')
		FROM lists l WHERE l.workspace_id = $1 AND l.deleted_at IS NULL ORDER BY l.created_at, l.name`, ws.ID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var o RecListOption
		var id uuid.UUID
		if err := rows.Scan(&id, &o.Name, &o.Subscribers); err != nil {
			return out, err
		}
		o.ID = id.String()
		out.Lists = append(out.Lists, o)
	}
	return out, rows.Err()
}

func (s *Service) settings(ctx context.Context, ws *store.Workspace) (store.RecommendationSetting, error) {
	set, err := s.Q.RecSettingsGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.RecommendationSetting{WorkspaceID: ws.ID, ConfirmPage: true}, nil
	}
	return set, err
}

func (s *Service) image(ctx context.Context, ws *store.Workspace, id *uuid.UUID) *RecImage {
	if id == nil {
		return nil
	}
	a, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: *id})
	if err != nil {
		return nil
	}
	return &RecImage{ID: a.ID.String(), URL: media.URL(a.ID, a.FileName), Name: a.FileName}
}

// items are the recommendations with their state and 30-day numbers.
func (s *Service) items(ctx context.Context, ws *store.Workspace, now time.Time) ([]RecommendationItem, error) {
	recs, err := s.Q.RecList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []RecommendationItem{}
	if len(recs) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, len(recs))
	for i, r := range recs {
		ids[i] = r.ID
	}
	totals, err := s.Q.RecTotals(ctx, store.RecTotalsParams{Ids: ids, Since: dayOf(now.AddDate(0, 0, -29), ws)})
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]store.RecTotalsRow{}
	for _, t := range totals {
		byID[t.RecommendationID] = t
	}
	for i, r := range recs {
		it := RecommendationItem{ID: r.ID.String(), Kind: r.Kind, Name: r.Name, Description: r.Description, Initials: initials(r.Name),
			Logo: s.image(ctx, ws, r.LogoID), URL: r.Url, ShowConfirm: r.ShowConfirm, ShowBlock: r.ShowBlock, Position: i + 1}
		var link *store.PartnerLink
		if r.PartnerLinkID != nil {
			if l, err := s.Q.PartnerLinkByID(ctx, *r.PartnerLinkID); err == nil {
				link = &l
			}
		}
		it.State, it.Hidden = visibility(r, link, now)
		switch {
		case r.Kind == KindLink:
			it.Host = hostOf(r.Url, true)
			if r.Status == "paused" {
				it.Since = r.PausedAt
				it.PausedBy = s.userName(ctx, r.PausedBy)
			}
		case link != nil:
			it.Host = hostOf(link.RemoteUrl, false)
			it.LinkID, it.DailyLimit, it.ConnectedAt = link.ID.String(), int(link.DailyLimit), &link.ConnectedAt
			it.ConnectedBy = s.userName(ctx, link.ConnectedBy)
			it.Since = &link.ConnectedAt
			switch it.State {
			case "paused", "partner_paused":
				it.Since, it.PausedBy = link.PausedAt, s.userName(ctx, link.PausedBy)
			case "invalid":
				it.Since, it.InvalidReason = link.InvalidSince, link.InvalidReason
			case "limit":
				it.LimitUntil = link.LimitUntil
			case "unreachable":
				it.Since = link.FailingSince
			}
			_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'accepted' AND sent_at >= $2), count(*) FILTER (WHERE status = 'queued')
				FROM partner_requests_out WHERE link_id = $1`, link.ID, midnightUTC(now)).Scan(&it.Today, &it.Waiting)
		}
		if t, ok := byID[r.ID]; ok {
			it.ShownConfirm, it.ShownEmail, it.Clicks, it.Requests, it.Confirmed = t.ShownConfirm, t.ShownEmail, t.Clicks, t.Requests, t.Confirmed
			it.Shown = t.ShownConfirm + t.ShownEmail
		}
		out = append(out, it)
	}
	return out, nil
}

func (s *Service) userName(ctx context.Context, id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	var name string
	_ = s.Pool.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, *id).Scan(&name)
	return name
}

// recommendingYou are the partners that recommend this workspace.
func (s *Service) recommendingYou(ctx context.Context, ws *store.Workspace, now time.Time) ([]RecommendingYouItem, error) {
	links, err := s.Q.PartnerLinks(ctx, store.PartnerLinksParams{WorkspaceID: ws.ID, Direction: DirIn})
	if err != nil {
		return nil, err
	}
	out := []RecommendingYouItem{}
	for _, l := range links {
		it := RecommendingYouItem{LinkID: l.ID.String(), Name: l.RemoteName, Instance: hostOf(l.RemoteUrl, false), Label: l.Label, ListName: l.ListName,
			Since: l.ConnectedAt, ConnectedBy: l.ConnectedByName, DailyLimit: int(l.DailyLimit), State: l.Status, PausedBy: l.PausedByName, PausedAt: l.PausedAt}
		if l.Status == LinkPaused && l.PausedSide == "remote" {
			it.State = "partner_paused"
		}
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'accepted' AND received_at >= $2),
				count(*) FILTER (WHERE confirmed_at IS NOT NULL AND received_at >= $2), count(*) FILTER (WHERE status = 'accepted' AND received_at >= $3)
			FROM partner_requests_in WHERE link_id = $1`, l.ID, now.AddDate(0, 0, -30), midnightUTC(now)).Scan(&it.Requests, &it.Confirmed, &it.Today); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

// blockUses are the emails whose saved version has the Recommendations
// block (the welcome emails, usually).
func (s *Service) blockUses(ctx context.Context, ws *store.Workspace) ([]RecBlockUse, error) {
	rows, err := s.Pool.Query(ctx, `SELECT e.id, e.name, coalesce(a.name, ''), coalesce(a.id::text, '')
		FROM automation_emails e
		JOIN automation_email_versions v ON v.email_id = e.id AND v.version = e.latest_version
		LEFT JOIN automations a ON a.id = e.automation_id
		WHERE e.workspace_id = $1 AND e.deleted_at IS NULL AND v.html LIKE '%rc:recommendations%'
		ORDER BY a.name, e.created_at LIMIT 10`, ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RecBlockUse{}
	for rows.Next() {
		var u RecBlockUse
		var id uuid.UUID
		if err := rows.Scan(&id, &u.Name, &u.Automation, &u.AutomationID); err != nil {
			return nil, err
		}
		u.EmailID = id.String()
		out = append(out, u)
	}
	return out, rows.Err()
}

// profile is the partner profile, with its defaults: the publisher's name
// and the first list.
func (s *Service) profile(ctx context.Context, ws *store.Workspace, set store.RecommendationSetting) (RecProfile, error) {
	p := RecProfile{Name: set.ProfileName, Description: set.ProfileDescription, Logo: s.image(ctx, ws, set.ProfileLogoID), Instance: hostOf(s.instance(), false)}
	if p.Name == "" {
		p.Name = s.publisher(ctx, ws)
	}
	p.Initials = initials(p.Name)
	listID := set.ProfileListID
	if listID == nil {
		var id uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL ORDER BY created_at LIMIT 1`, ws.ID).Scan(&id); err == nil {
			listID = &id
		}
	}
	if listID != nil {
		p.ListID = listID.String()
		_ = s.Pool.QueryRow(ctx, `SELECT l.name, (SELECT count(*) FROM list_memberships m JOIN contacts c ON c.id = m.contact_id AND c.erased_at IS NULL
			WHERE m.list_id = l.id AND m.status = 'subscribed') FROM lists l WHERE l.id = $1`, *listID).Scan(&p.ListName, &p.ListSubscribers)
	}
	return p, nil
}

// publisher is the workspace's publisher name (its public pages' branding).
func (s *Service) publisher(ctx context.Context, ws *store.Workspace) string {
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil && strings.TrimSpace(set.Branding.PublisherName) != "" {
			return set.Branding.PublisherName
		}
	}
	return ws.Name
}

// --- Changes -------------------------------------------------------------------

// RecLinkInput adds or changes a link recommendation (a partner's only
// changes where it shows).
type RecLinkInput struct {
	Name        string `json:"name" maxLength:"120"`
	Description string `json:"description" maxLength:"120"`
	LogoID      string `json:"logoId,omitempty" format:"uuid"`
	URL         string `json:"url" maxLength:"2000"`
	ShowConfirm bool   `json:"showConfirm"`
	ShowBlock   bool   `json:"showBlock"`
}

var (
	mNameRequired = msgs.M("rec.nameRequired")
	mDescLong     = msgs.M("rec.descriptionLong")
	mURL          = msgs.M("rec.url")
	mWhere        = msgs.M("rec.where")
	mLogo         = msgs.M("rec.logo")
)

func (s *Service) checkLink(ctx context.Context, ws *store.Workspace, in *RecLinkInput, partner bool) error {
	in.Name, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if !in.ShowConfirm && !in.ShowBlock {
		return api.Invalid("body.showConfirm", s.text(ctx, mWhere, nil))
	}
	if partner {
		return nil
	}
	if in.Name == "" {
		return api.Invalid("body.name", s.text(ctx, mNameRequired, nil))
	}
	if utf8.RuneCountInString(in.Description) > 120 {
		return api.Invalid("body.description", s.text(ctx, mDescLong, nil))
	}
	u, err := s.guard().ValidateURL(in.URL)
	if err != nil || (u.Scheme != "https" && !s.allowPrivate()) {
		return api.Invalid("body.url", s.text(ctx, mURL, nil))
	}
	in.URL = u.String()
	if in.LogoID != "" {
		if s.image(ctx, ws, uuidOrNil(in.LogoID)) == nil {
			return api.Invalid("body.logoId", s.text(ctx, mLogo, nil))
		}
	}
	return nil
}

func actor(ctx context.Context) *uuid.UUID {
	p := api.PrincipalFrom(ctx)
	if p == nil || p.Type != authz.ActorUser {
		return nil
	}
	id := p.ID
	return &id
}

// AddLink adds a link recommendation, last in the order.
func (s *Service) AddLink(ctx context.Context, ws *store.Workspace, in RecLinkInput) (RecommendationItem, error) {
	if err := s.checkLink(ctx, ws, &in, false); err != nil {
		return RecommendationItem{}, err
	}
	id := uuid.Must(uuid.NewV7())
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RecInsert(ctx, store.RecInsertParams{ID: id, WorkspaceID: ws.ID, Kind: KindLink, Name: in.Name, Description: in.Description,
			LogoID: uuidOrNil(in.LogoID), Url: in.URL, ShowConfirm: in.ShowConfirm, ShowBlock: in.ShowBlock, CreatedBy: actor(ctx), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "recommendation.added", TargetType: "recommendation", TargetID: id.String(),
			TargetName: in.Name, After: map[string]any{"kind": KindLink, "url": in.URL, "description": in.Description, "showConfirm": in.ShowConfirm, "showBlock": in.ShowBlock}})
	})
	if err != nil {
		return RecommendationItem{}, err
	}
	return s.Item(ctx, ws, id)
}

// Item is one recommendation as the tab shows it.
func (s *Service) Item(ctx context.Context, ws *store.Workspace, id uuid.UUID) (RecommendationItem, error) {
	items, err := s.items(ctx, ws, s.now())
	if err != nil {
		return RecommendationItem{}, err
	}
	for _, it := range items {
		if it.ID == id.String() {
			return it, nil
		}
	}
	return RecommendationItem{}, api.NotFound("Recommendation")
}

// Update changes a recommendation: a link's name, description, logo,
// address and where it shows; a partner's where it shows.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, id uuid.UUID, in RecLinkInput) (RecommendationItem, error) {
	cur, err := s.Q.RecGet(ctx, store.RecGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return RecommendationItem{}, api.MapError(err, "Recommendation")
	}
	partner := cur.Kind == KindPartner
	if err := s.checkLink(ctx, ws, &in, partner); err != nil {
		return RecommendationItem{}, err
	}
	if partner {
		in.Name, in.Description, in.URL, in.LogoID = cur.Name, cur.Description, cur.Url, ""
		if cur.LogoID != nil {
			in.LogoID = cur.LogoID.String()
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RecUpdate(ctx, store.RecUpdateParams{ID: id, Name: in.Name, Description: in.Description, LogoID: uuidOrNil(in.LogoID), Url: in.URL,
			ShowConfirm: in.ShowConfirm, ShowBlock: in.ShowBlock, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "recommendation.updated", TargetType: "recommendation", TargetID: id.String(),
			TargetName: in.Name,
			Before:     map[string]any{"name": cur.Name, "description": cur.Description, "url": cur.Url, "showConfirm": cur.ShowConfirm, "showBlock": cur.ShowBlock},
			After:      map[string]any{"name": in.Name, "description": in.Description, "url": in.URL, "showConfirm": in.ShowConfirm, "showBlock": in.ShowBlock}})
	})
	if err != nil {
		return RecommendationItem{}, err
	}
	return s.Item(ctx, ws, id)
}

var mPartnerPause = msgs.M("rec.partnerPause")

// SetPaused pauses or resumes a link recommendation. A partner is paused
// with its partner link (PausePartner).
func (s *Service) SetPaused(ctx context.Context, ws *store.Workspace, id uuid.UUID, paused bool) (RecommendationItem, error) {
	cur, err := s.Q.RecGet(ctx, store.RecGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return RecommendationItem{}, api.MapError(err, "Recommendation")
	}
	if cur.Kind == KindPartner {
		return RecommendationItem{}, api.Err(http.StatusConflict, "partner_link", s.text(ctx, mPartnerPause, nil))
	}
	status, action := "showing", "recommendation.resumed"
	if paused {
		status, action = "paused", "recommendation.paused"
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RecSetStatus(ctx, store.RecSetStatusParams{ID: id, Status: status, Now: s.now(), By: actor(ctx)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "recommendation", TargetID: id.String(), TargetName: cur.Name,
			Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": status}})
	})
	if err != nil {
		return RecommendationItem{}, err
	}
	return s.Item(ctx, ws, id)
}

// Remove takes a link recommendation off every page; its numbers stay in
// the results. A partner is removed with its partner link (EndPartner).
func (s *Service) Remove(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	cur, err := s.Q.RecGet(ctx, store.RecGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Recommendation")
	}
	if cur.Kind == KindPartner && cur.PartnerLinkID != nil {
		if l, err := s.Q.PartnerLinkByID(ctx, *cur.PartnerLinkID); err == nil && l.Status != LinkEnded {
			p := api.PrincipalFrom(ctx)
			if p == nil || !p.Can(authz.GrowConnect) {
				return api.Forbidden(authz.GrowConnect, referrals.GrowHelpers(ctx, s.Q, ws.ID, authz.GrowConnect))
			}
			if err := s.EndLink(ctx, ws, l.ID); err != nil {
				return err
			}
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RecRemove(ctx, store.RecRemoveParams{ID: id, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "recommendation.removed", TargetType: "recommendation", TargetID: id.String(),
			TargetName: cur.Name, Before: map[string]any{"kind": cur.Kind, "url": cur.Url}})
	})
}

// RecOrderInput is the recommendations in the order readers see them.
type RecOrderInput struct {
	IDs []string `json:"ids" maxItems:"100"`
}

var mOrder = msgs.M("rec.order")

// Reorder puts the recommendations in a new order.
func (s *Service) Reorder(ctx context.Context, ws *store.Workspace, in RecOrderInput) error {
	recs, err := s.Q.RecList(ctx, ws.ID)
	if err != nil {
		return err
	}
	have := map[string]string{}
	before := []string{}
	for _, r := range recs {
		have[r.ID.String()] = r.Name
		before = append(before, r.Name)
	}
	if len(in.IDs) != len(recs) {
		return api.Invalid("body.ids", s.text(ctx, mOrder, nil))
	}
	after := []string{}
	for _, id := range in.IDs {
		name, ok := have[id]
		if !ok {
			return api.Invalid("body.ids", s.text(ctx, mOrder, nil))
		}
		delete(have, id)
		after = append(after, name)
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for i, id := range in.IDs {
			if err := q.RecSetPosition(ctx, store.RecSetPositionParams{ID: uuid.MustParse(id), WorkspaceID: ws.ID, Position: int32(i), Now: s.now()}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "recommendation.reordered", TargetType: "recommendations", TargetID: ws.ID.String(),
			TargetName: "Recommendations", Before: map[string]any{"order": before}, After: map[string]any{"order": after}})
	})
}

// RecSettingsInput is where recommendations show: the confirmation page.
type RecSettingsInput struct {
	ConfirmPage bool `json:"confirmPage"`
}

// SetConfirmPage turns the confirmation page's "While you're here" on or off.
func (s *Service) SetConfirmPage(ctx context.Context, ws *store.Workspace, on bool) error {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.RecSettingsUpsert(ctx, store.RecSettingsUpsertParams{WorkspaceID: ws.ID, ConfirmPage: on, ProfileName: set.ProfileName,
			ProfileDescription: set.ProfileDescription, ProfileLogoID: set.ProfileLogoID, ProfileListID: set.ProfileListID, UpdatedBy: actor(ctx), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "recommendation.settings_changed", TargetType: "recommendations",
			TargetID: ws.ID.String(), TargetName: "Confirmation page", Before: map[string]any{"confirmPage": set.ConfirmPage}, After: map[string]any{"confirmPage": on}})
	})
}

// RecProfileInput changes the partner profile.
type RecProfileInput struct {
	Name        string `json:"name" maxLength:"120"`
	Description string `json:"description" maxLength:"120"`
	LogoID      string `json:"logoId,omitempty" format:"uuid"`
	ListID      string `json:"listId" format:"uuid"`
}

var mListUnknown = msgs.M("rec.listUnknown")

// UpdateProfile changes what partners see and show; partners that
// recommend this workspace get the new profile over the signed channel.
func (s *Service) UpdateProfile(ctx context.Context, ws *store.Workspace, in RecProfileInput) (RecProfile, error) {
	in.Name, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if in.Name == "" {
		return RecProfile{}, api.Invalid("body.name", s.text(ctx, mNameRequired, nil))
	}
	if utf8.RuneCountInString(in.Description) > 120 {
		return RecProfile{}, api.Invalid("body.description", s.text(ctx, mDescLong, nil))
	}
	list := uuidOrNil(in.ListID)
	if list == nil {
		return RecProfile{}, api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
	}
	if _, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: *list}); err != nil {
		return RecProfile{}, api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
	}
	if in.LogoID != "" && s.image(ctx, ws, uuidOrNil(in.LogoID)) == nil {
		return RecProfile{}, api.Invalid("body.logoId", s.text(ctx, mLogo, nil))
	}
	set, err := s.settings(ctx, ws)
	if err != nil {
		return RecProfile{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.RecSettingsUpsert(ctx, store.RecSettingsUpsertParams{WorkspaceID: ws.ID, ConfirmPage: set.ConfirmPage, ProfileName: in.Name,
			ProfileDescription: in.Description, ProfileLogoID: uuidOrNil(in.LogoID), ProfileListID: list, UpdatedBy: actor(ctx), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "recommendation.profile_updated", TargetType: "recommendations",
			TargetID: ws.ID.String(), TargetName: "Partner profile",
			Before: map[string]any{"name": set.ProfileName, "description": set.ProfileDescription},
			After:  map[string]any{"name": in.Name, "description": in.Description, "list": in.ListID}})
	})
	if err != nil {
		return RecProfile{}, err
	}
	set, err = s.settings(ctx, ws)
	if err != nil {
		return RecProfile{}, err
	}
	s.announceProfile(ctx, ws)
	return s.profile(ctx, ws, set)
}
