package sponsors

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// statsFresh is how long a report still coming in is reused before it's
// counted again.
const statsFresh = 10 * time.Minute

// sponsorLinkKey is a link's address without its query: the sponsor's
// links are the campaign's links to the creative's address.
func sponsorLinkKey(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Host) + strings.TrimSuffix(u.EscapedPath(), "/")
}

// blockRule is the Visibility rule id of a campaign's Sponsor block, when
// it shows to some recipients only (internal/visibility's rc-visible).
func blockRule(src string) string {
	if !strings.Contains(src, BlockAttr) {
		return ""
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return ""
	}
	id := ""
	root.Walk(func(n *mjml.Node, _ []int) {
		if id != "" {
			return
		}
		if _, ok := n.Get(BlockAttr); !ok {
			return
		}
		raw, ok := n.Get("rc-visible")
		if !ok || strings.TrimSpace(raw) == "" {
			return
		}
		var r struct {
			ID   string `json:"id"`
			Show string `json:"show"`
		}
		if json.Unmarshal([]byte(raw), &r) == nil && r.ID != "" && r.Show != "everyone" {
			id = r.ID
		}
	})
	return id
}

// computeStats counts a sent booking's report from its campaign: the
// recipients who got the block, their human opens, and the unique clickers
// and clicks on the sponsor's links. Machine opens and bot clicks never
// reach these counts.
func (s *Service) computeStats(ctx context.Context, b store.SponsorBookingGetRow) (store.SponsorReportStatsUpsertParams, error) {
	out := store.SponsorReportStatsUpsertParams{BookingID: b.ID, Links: []byte(`[]`), ComputedAt: s.now()}
	if b.CampaignID == nil {
		return out, nil
	}
	cp, err := s.Q.SponsorCampaignBrief(ctx, store.SponsorCampaignBriefParams{ID: *b.CampaignID, WorkspaceID: b.WorkspaceID})
	if err != nil {
		return out, err
	}
	sent := firstTime(cp.StartedAt, b.SentAt)
	out.SentAt = sent
	rule := blockRule(cp.Mjml)
	if rule == "" {
		_ = s.Pool.QueryRow(ctx, `SELECT coalesce(sum(delivered), 0), coalesce(sum(opened_human), 0) FROM campaign_counters WHERE campaign_id = $1`, cp.ID).
			Scan(&out.Delivered, &out.HumanOpens)
	} else {
		// Only the recipients whose version had the block.
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE delivered_at IS NOT NULL), count(*) FILTER (WHERE human_opened_at IS NOT NULL)
			FROM messages WHERE campaign_id = $1 AND NOT (','||hidden_blocks||',' LIKE '%,'||$2||',%')`, cp.ID, rule).Scan(&out.Delivered, &out.HumanOpens)
		out.VisibilityNote = "rule"
	}
	c := parseCreative(b.Creative)
	key := sponsorLinkKey(WithUTM(c.Link, c.UTM))
	rows, err := s.Pool.Query(ctx, `SELECT position, url, label, clicks FROM campaign_links WHERE campaign_id = $1 ORDER BY position`, cp.ID)
	if err != nil {
		return out, err
	}
	type link struct {
		Kind   string `json:"kind"`
		Label  string `json:"label"`
		Clicks int64  `json:"clicks"`
	}
	var links []link
	var positions []int32
	for rows.Next() {
		var pos int32
		var u, label string
		var clicks int64
		if err := rows.Scan(&pos, &u, &label, &clicks); err != nil {
			rows.Close()
			return out, err
		}
		if key == "" || sponsorLinkKey(u) != key {
			continue
		}
		kind := "link"
		switch {
		case c.Button != "" && strings.EqualFold(strings.TrimSpace(label), c.Button):
			kind = "button"
		case strings.TrimSpace(label) == "" || strings.EqualFold(strings.TrimSpace(label), b.SponsorName):
			kind = "logo"
		}
		links = append(links, link{Kind: kind, Label: label, Clicks: clicks})
		positions = append(positions, pos)
		out.Clicks += clicks
	}
	rows.Close()
	if len(positions) > 0 {
		q := `SELECT count(DISTINCT t.message_id) FROM tracking_link_clicks t JOIN messages m ON m.id = t.message_id WHERE m.campaign_id = $1 AND t.link = ANY($2)`
		args := []any{cp.ID, positions}
		if rule != "" {
			q += ` AND NOT (','||m.hidden_blocks||',' LIKE '%,'||$3||',%')`
			args = append(args, rule)
		}
		_ = s.Pool.QueryRow(ctx, q, args...).Scan(&out.UniqueClickers)
	}
	if links == nil {
		links = []link{}
	}
	out.Links, _ = json.Marshal(links)
	out.Final = sent != nil && !s.now().Before(sent.Add(FinalAfter))
	return out, nil
}

// stats returns a sent booking's report numbers: frozen once final,
// otherwise counted again when older than a few minutes.
func (s *Service) stats(ctx context.Context, b store.SponsorBookingGetRow) (store.SponsorReportStat, error) {
	st, err := s.Q.SponsorReportStats(ctx, b.ID)
	if err == nil && (st.Final || s.now().Sub(st.ComputedAt) < statsFresh || b.CampaignStatus != "sent") {
		return st, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return st, err
	}
	if b.CampaignID == nil || b.CampaignStatus != "sent" {
		return st, err
	}
	p, err := s.computeStats(ctx, b)
	if err != nil {
		return store.SponsorReportStat{}, err
	}
	if err := s.Q.SponsorReportStatsUpsert(ctx, p); err != nil {
		return store.SponsorReportStat{}, err
	}
	return s.Q.SponsorReportStats(ctx, b.ID)
}

func ratio(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}

// report builds a sent booking's report.
func (s *Service) report(ctx context.Context, ws *store.Workspace, b store.SponsorBookingGetRow, slot store.SponsorSlotGetRow) (*SponsorReport, error) {
	status := b.Status
	if status == "booked" && b.CampaignStatus == "sent" {
		status = "sent"
	}
	if status != "sent" {
		return nil, nil
	}
	st, err := s.stats(ctx, b)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	c := parseCreative(b.Creative)
	r := &SponsorReport{BookingID: b.ID.String(), Issue: int(b.IssueNumber), IssueOn: noonOn(b.IssueOn, ws), ListName: slot.ListName, SlotName: slot.Name,
		Placement: slot.Placement, Sponsor: b.SponsorName, SponsorName: b.SponsorName, Initials: initials(b.SponsorName), Creative: c,
		Label: text(ctx, "block.labelBy", map[string]any{"sponsor": b.SponsorName}), Links: []SponsorReportLink{}, LinkCampaign: c.UTM.Campaign}
	if u, err := url.Parse(WithUTM(c.Link, c.UTM)); err == nil && u.Host != "" {
		r.LinkHost = u.Host + strings.TrimSuffix(u.EscapedPath(), "/")
	}
	sent := firstTime(st.SentAt, b.SentAt, b.CampaignStartedAt)
	r.SentAt = sent
	if sent != nil {
		fin := sent.Add(FinalAfter)
		r.FinalAt = &fin
	}
	r.Final = st.Final
	r.Delivered, r.HumanOpens, r.UniqueClickers, r.Clicks = st.Delivered, st.HumanOpens, st.UniqueClickers, st.Clicks
	r.HumanOpenRate = ratio(st.HumanOpens, st.Delivered)
	r.ClickRate = ratio(st.UniqueClickers, st.Delivered)
	r.ClickToOpen = ratio(st.UniqueClickers, st.HumanOpens)
	if st.VisibilityNote != "" {
		r.Visibility = st.VisibilityNote
	}
	var links []struct {
		Kind   string `json:"kind"`
		Label  string `json:"label"`
		Clicks int64  `json:"clicks"`
	}
	_ = json.Unmarshal(st.Links, &links)
	for _, l := range links {
		r.Links = append(r.Links, SponsorReportLink{Kind: l.Kind, Label: l.Label, Clicks: l.Clicks, Share: ratio(l.Clicks, st.Clicks)})
	}
	return r, nil
}

// Report is a sent booking's report.
func (s *Service) Report(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SponsorReport, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorReport{}, err
	}
	r, err := s.report(ctx, ws, b, slot)
	if err != nil {
		return SponsorReport{}, err
	}
	if r == nil {
		return SponsorReport{}, problem(ctx, 409, api.CodeConflict, "problem.notSentYet", map[string]any{"issue": b.IssueNumber})
	}
	return *r, nil
}

// ShareInput creates a read-only link to a sponsor report.
type ShareInput struct {
	ExpiresInDays int `json:"expiresInDays,omitempty" minimum:"0" maximum:"365" doc:"0 keeps the link until it's revoked."`
}

func sealAD(id uuid.UUID) string { return "sponsor_report_share:" + id.String() + ":token" }

func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateShare makes a read-only link to one booking's report, or to every
// booking of a sponsor.
func (s *Service) CreateShare(ctx context.Context, ws *store.Workspace, p *authz.Principal, booking, sponsor *uuid.UUID, in ShareInput) (SponsorShareLink, error) {
	target, name := "", ""
	switch {
	case booking != nil:
		b, slot, err := s.mustBooking(ctx, ws, *booking)
		if err != nil {
			return SponsorShareLink{}, err
		}
		if b.Status != "sent" && b.CampaignStatus != "sent" {
			return SponsorShareLink{}, problem(ctx, 409, api.CodeConflict, "problem.notSentYet", map[string]any{"issue": b.IssueNumber})
		}
		target, name = b.ID.String(), issueName(slot.ListName, int(b.IssueNumber), slot.Name)+" · "+b.SponsorName
	case sponsor != nil:
		sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: *sponsor, WorkspaceID: ws.ID})
		if err != nil {
			return SponsorShareLink{}, mapErr(ctx, err, "sponsor")
		}
		target, name = sp.ID.String(), sp.Name
	default:
		return SponsorShareLink{}, notFound(ctx, "booking")
	}
	token, err := newToken()
	if err != nil {
		return SponsorShareLink{}, err
	}
	id := newID()
	sealed, err := s.Box.SealString(token, sealAD(id))
	if err != nil {
		return SponsorShareLink{}, err
	}
	now := s.now()
	var expires *time.Time
	if in.ExpiresInDays > 0 {
		t := now.Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)
		expires = &t
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorShareInsert(ctx, store.SponsorShareInsertParams{ID: id, WorkspaceID: ws.ID, BookingID: booking, SponsorID: sponsor, TokenHash: authn.HashToken(token),
			TokenSealed: sealed, CreatedBy: userID(p), CreatedAt: now, ExpiresAt: expires}); err != nil {
			return err
		}
		if booking != nil {
			if err := s.event(ctx, q, *booking, p, "report_shared", map[string]any{"expiresAt": expires}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.report_link_created", TargetType: "sponsor_report_share", TargetID: id.String(),
			TargetName: name, After: map[string]any{"expiresAt": expires, "every": sponsor != nil}, Metadata: map[string]any{"target": target}})
	})
	if err != nil {
		return SponsorShareLink{}, err
	}
	return SponsorShareLink{ID: id.String(), URL: s.URLs.Report(token), CreatedAt: now, CreatedBy: actorName(p), ExpiresAt: expires, Every: sponsor != nil}, nil
}

// shares lists a booking's or a sponsor's active links.
func (s *Service) shares(ctx context.Context, ws *store.Workspace, booking, sponsor *uuid.UUID) ([]SponsorShareLink, error) {
	rows, err := s.Q.SponsorSharesOf(ctx, store.SponsorSharesOfParams{WorkspaceID: ws.ID, BookingID: booking, SponsorID: sponsor, Now: s.now()})
	if err != nil {
		return nil, err
	}
	out := []SponsorShareLink{}
	for _, r := range rows {
		l := SponsorShareLink{ID: r.ID.String(), CreatedAt: r.CreatedAt, CreatedBy: r.CreatedByName, ExpiresAt: r.ExpiresAt, Views: r.Views, LastViewed: r.LastViewedAt,
			Every: r.SponsorID != nil}
		if len(r.TokenSealed) > 0 {
			if token, err := s.Box.OpenString(r.TokenSealed, sealAD(r.ID)); err == nil {
				l.URL = s.URLs.Report(token)
			}
		}
		out = append(out, l)
	}
	return out, nil
}

// RevokeShare stops a read-only link from working.
func (s *Service) RevokeShare(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	sh, err := s.Q.SponsorShareGet(ctx, store.SponsorShareGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return mapErr(ctx, err, "link")
	}
	if sh.RevokedAt != nil {
		return nil
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorShareRevoke(ctx, store.SponsorShareRevokeParams{ID: id, WorkspaceID: ws.ID, At: s.now(), RevokedBy: userID(p)}); err != nil {
			return err
		}
		if sh.BookingID != nil {
			if err := s.event(ctx, q, *sh.BookingID, p, "report_link_revoked", nil); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.report_link_revoked", TargetType: "sponsor_report_share", TargetID: id.String(),
			Before: map[string]any{"createdAt": sh.CreatedAt, "expiresAt": sh.ExpiresAt, "views": sh.Views}})
	})
}

// SharedReport is what a read-only link shows: one booking's report, or
// every sent booking of a sponsor with the ones to come.
type SharedReport struct {
	Workspace store.Workspace
	ExpiresAt *time.Time
	Sponsor   string
	Reports   []SponsorReport
	// Upcoming are the sponsor's booked issues still to come.
	Upcoming []SponsorIssueRow
	Every    bool
}

// ErrShareGone is a link revoked, expired or unknown.
var ErrShareGone = errors.New("sponsors: link revoked or expired")

// Shared resolves a read-only link. It never returns contact data.
func (s *Service) Shared(ctx context.Context, token string) (SharedReport, error) {
	token = strings.TrimSpace(token)
	if len(token) < 16 || len(token) > 64 {
		return SharedReport{}, ErrShareGone
	}
	sh, err := s.Q.SponsorShareByToken(ctx, authn.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return SharedReport{}, ErrShareGone
	}
	if err != nil {
		return SharedReport{}, err
	}
	now := s.now()
	if sh.RevokedAt != nil || (sh.ExpiresAt != nil && !sh.ExpiresAt.After(now)) {
		return SharedReport{}, ErrShareGone
	}
	ws, err := s.Q.GetWorkspace(ctx, sh.WorkspaceID)
	if err != nil {
		return SharedReport{}, err
	}
	out := SharedReport{Workspace: ws, ExpiresAt: sh.ExpiresAt, Reports: []SponsorReport{}, Upcoming: []SponsorIssueRow{}}
	var bookings []uuid.UUID
	switch {
	case sh.BookingID != nil:
		bookings = append(bookings, *sh.BookingID)
	case sh.SponsorID != nil:
		out.Every = true
		sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: *sh.SponsorID, WorkspaceID: ws.ID})
		if err != nil {
			return SharedReport{}, ErrShareGone
		}
		out.Sponsor = sp.Name
		all, err := s.Q.SponsorBookingsOfWorkspace(ctx, ws.ID)
		if err != nil {
			return SharedReport{}, err
		}
		today := s.today(&ws)
		for _, b := range all {
			if b.SponsorID == nil || *b.SponsorID != sp.ID {
				continue
			}
			switch {
			case b.Status == "sent" || b.CampaignStatus == "sent":
				bookings = append(bookings, b.ID)
			case b.Status == "booked" && !civil(b.IssueOn).Before(today):
				out.Upcoming = append(out.Upcoming, SponsorIssueRow{Number: int(b.IssueNumber), IssueOn: noonOn(b.IssueOn, &ws)})
			}
		}
	}
	for _, id := range bookings {
		b, slot, err := s.mustBooking(ctx, &ws, id)
		if err != nil {
			continue
		}
		r, err := s.report(ctx, &ws, b, slot)
		if err != nil {
			return SharedReport{}, err
		}
		if r != nil {
			out.Reports = append(out.Reports, *r)
			if out.Sponsor == "" {
				out.Sponsor = r.Sponsor
			}
		}
	}
	if !out.Every && len(out.Reports) == 0 {
		return SharedReport{}, ErrShareGone
	}
	_ = s.Q.SponsorShareTouch(ctx, store.SponsorShareTouchParams{ID: sh.ID, At: now})
	return out, nil
}

// IsShareToken reports whether a /r/ token is a sponsor report's (the
// campaign reports' links share the path).
func (s *Service) IsShareToken(ctx context.Context, token string) bool {
	token = strings.TrimSpace(token)
	if len(token) < 16 || len(token) > 64 {
		return false
	}
	_, err := s.Q.SponsorShareByToken(ctx, authn.HashToken(token))
	return err == nil
}

var _ = http.StatusOK
