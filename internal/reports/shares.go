package reports

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxActiveLinks caps the read-only links a campaign can have at once.
const MaxActiveLinks = 10

// tokenBytes is the entropy of a link token (128 bits).
const tokenBytes = 16

func newShareToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sealAD(id uuid.UUID) string { return "report_share:" + id.String() + ":token" }

// DefaultShareDays is how long a read-only link works unless the person
// who makes it chooses otherwise, and MaxShareDays the longest they can
// choose (review AN-16).
const (
	DefaultShareDays = 30
	MaxShareDays     = 90
)

// mShareDays is the problem of a link asked to work outside 1 to 90 days.
var mShareDays = msgs.M("problem.shareDays")

// ShareInput creates a read-only link. The numbers, charts and tables are
// always in; the cost per provider, the warm-up state and the email itself
// only when chosen (review AN-16). The AI summary and the Ask box are never
// shared.
type ShareInput struct {
	// ExpiresInDays makes the link stop working after that many days (30
	// when absent); every link expires.
	ExpiresInDays int  `json:"expiresInDays,omitempty" minimum:"1" maximum:"90" doc:"30 when absent"`
	IncludeCost   bool `json:"includeCost,omitempty" doc:"Show the cost per sending provider"`
	IncludeWarmup bool `json:"includeWarmup,omitempty" doc:"Show the warm-up day of each sending provider"`
	// IncludeEmail shows the email as sent in the click map.
	IncludeEmail bool `json:"includeEmail,omitempty" doc:"Show the email as sent, in the click map"`
}

// CreateShare makes a read-only link to a campaign's report.
func (s *Service) CreateShare(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, in ShareInput) (ReportShareLink, error) {
	days := in.ExpiresInDays
	if days == 0 {
		days = DefaultShareDays
	}
	// The API's schema holds the days to 1–90; agents (MCP) and other
	// callers come here without it, and a link mustn't outlive the limit.
	if days < 1 || days > MaxShareDays {
		return ReportShareLink{}, api.Invalid("body.expiresInDays", mShareDays.Text(i18n.LangFrom(ctx), nil))
	}
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return ReportShareLink{}, api.MapError(err, "Campaign")
	}
	if unsent[c.Status] {
		return ReportShareLink{}, api.Conflict(mNothingToShare.Text(i18n.LangFrom(ctx), i18n.Args{"campaign": c.Name}))
	}
	now := s.now()
	active, err := s.Q.ReportShareList(ctx, store.ReportShareListParams{CampaignID: c.ID, Now: &now})
	if err != nil {
		return ReportShareLink{}, err
	}
	if len(active) >= MaxActiveLinks {
		return ReportShareLink{}, api.Conflict(mTooManyLinks.Text(i18n.LangFrom(ctx), nil))
	}
	token, err := newShareToken()
	if err != nil {
		return ReportShareLink{}, err
	}
	id := uuid.Must(uuid.NewV7())
	sealed, err := s.Box.SealString(token, sealAD(id))
	if err != nil {
		return ReportShareLink{}, err
	}
	t := now.Add(time.Duration(days) * 24 * time.Hour)
	expires := &t
	p := api.PrincipalFrom(ctx)
	var by *uuid.UUID
	byName := ""
	if p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by, byName = &u, p.Name
	}
	var row store.ReportShare
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		row, err = q.ReportShareCreate(ctx, store.ReportShareCreateParams{ID: id, CampaignID: c.ID, TokenHash: authn.HashToken(token), TokenSealed: sealed,
			CreatedBy: by, CreatedAt: now, ExpiresAt: expires, IncludeCost: in.IncludeCost, IncludeWarmup: in.IncludeWarmup, IncludeEmail: in.IncludeEmail})
		if err != nil {
			return err
		}
		after := map[string]any{"campaign": c.Name, "expiresAt": expires, "includeCost": in.IncludeCost, "includeWarmup": in.IncludeWarmup, "includeEmail": in.IncludeEmail}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "report_share.create", TargetType: "report_share", TargetID: id.String(),
			TargetName: c.Name, After: after, Metadata: map[string]any{"campaignId": c.ID.String()}})
	})
	if err != nil {
		return ReportShareLink{}, err
	}
	return ReportShareLink{ID: row.ID.String(), URL: s.URLs.Report(token), CreatedAt: row.CreatedAt, CreatedByName: byName, ExpiresAt: row.ExpiresAt,
		Includes: ReportShareIncludes{Cost: row.IncludeCost, Warmup: row.IncludeWarmup, Email: row.IncludeEmail}}, nil
}

// Shares lists a campaign's active read-only links.
func (s *Service) Shares(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (ReportShareList, error) {
	if _, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: campaignID}); err != nil {
		return ReportShareList{}, api.MapError(err, "Campaign")
	}
	now := s.now()
	rows, err := s.Q.ReportShareList(ctx, store.ReportShareListParams{CampaignID: campaignID, Now: &now})
	if err != nil {
		return ReportShareList{}, err
	}
	out := ReportShareList{Items: []ReportShareLink{}}
	for _, r := range rows {
		l := ReportShareLink{ID: r.ID.String(), CreatedAt: r.CreatedAt, CreatedByName: r.CreatedByName, ExpiresAt: r.ExpiresAt, Views: r.Views, LastViewedAt: r.LastViewedAt,
			Includes: ReportShareIncludes{Cost: r.IncludeCost, Warmup: r.IncludeWarmup, Email: r.IncludeEmail}}
		if len(r.TokenSealed) > 0 {
			if token, err := s.Box.OpenString(r.TokenSealed, sealAD(r.ID)); err == nil {
				l.URL = s.URLs.Report(token)
			}
		}
		out.Items = append(out.Items, l)
	}
	return out, nil
}

// RevokeShare stops a read-only link from working.
func (s *Service) RevokeShare(ctx context.Context, ws *store.Workspace, campaignID, shareID uuid.UUID) error {
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return api.MapError(err, "Campaign")
	}
	now := s.now()
	var by *uuid.UUID
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by = &u
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.ReportShareRevoke(ctx, store.ReportShareRevokeParams{RevokedAt: &now, RevokedBy: by, ID: shareID, CampaignID: c.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Link")
		}
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "report_share.revoke", TargetType: "report_share", TargetID: row.ID.String(),
			TargetName: c.Name, Before: map[string]any{"createdAt": row.CreatedAt, "expiresAt": row.ExpiresAt, "views": row.Views},
			Metadata: map[string]any{"campaignId": c.ID.String()}})
	})
}

// Shared returns the report behind a read-only link, for anyone holding it.
// client identifies the caller for the rate limits (the client address).
func (s *Service) Shared(ctx context.Context, token, client string) (SharedCampaignReport, error) {
	now := s.now()
	lim := s.Limits
	if lim == nil {
		return SharedCampaignReport{}, errors.New("reports: no limiter configured")
	}
	if !lim.Allow(client, now) {
		return SharedCampaignReport{}, api.Err(http.StatusTooManyRequests, api.CodeRateLimited, mSharedLimited.Text(i18n.LangFrom(ctx), nil))
	}
	token = strings.TrimSpace(token)
	gone := api.Err(http.StatusNotFound, api.CodeNotFound, mSharedGone.Text(i18n.LangFrom(ctx), nil))
	if len(token) < 16 || len(token) > 64 {
		lim.Miss(client, now)
		return SharedCampaignReport{}, gone
	}
	link, err := s.Q.ReportShareByToken(ctx, authn.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (link.RevokedAt != nil || link.ExpiresAt != nil && !link.ExpiresAt.After(now)) {
		lim.Miss(client, now)
		return SharedCampaignReport{}, gone
	}
	if err != nil {
		return SharedCampaignReport{}, err
	}
	ws, err := s.Q.GetWorkspace(ctx, link.WorkspaceID)
	if err != nil {
		return SharedCampaignReport{}, err
	}
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: link.CampaignID})
	if err != nil {
		lim.Miss(client, now)
		return SharedCampaignReport{}, gone
	}
	rep, err := s.build(ctx, &ws, c)
	if err != nil {
		return SharedCampaignReport{}, err
	}
	redact(&rep)
	leaveOut(&rep, ReportShareIncludes{Cost: link.IncludeCost, Warmup: link.IncludeWarmup, Email: link.IncludeEmail})
	_ = s.Q.ReportShareTouch(ctx, store.ReportShareTouchParams{At: &now, ID: link.ID})
	return SharedCampaignReport{Workspace: ws.Name, ExpiresAt: link.ExpiresAt, Includes: ReportShareIncludes{Cost: link.IncludeCost, Warmup: link.IncludeWarmup,
		Email: link.IncludeEmail}, Report: rep}, nil
}

// leaveOut removes what a link doesn't share: the cost per provider, the
// warm-up state (in the providers' columns and in the insights' words),
// the email itself.
func leaveOut(r *CampaignReport, in ReportShareIncludes) {
	for i := range r.Providers.Columns {
		if !in.Cost {
			r.Providers.Columns[i].Cost = nil
		}
		if !in.Warmup {
			r.Providers.Columns[i].WarmupDay, r.Providers.Columns[i].EngagedFirst = nil, false
		}
	}
	if !in.Warmup {
		for i := range r.Insights {
			if d := r.Insights[i].detailNoWarmup; d != "" {
				r.Insights[i].Detail = d
			}
		}
	}
	if !in.Email {
		r.ClickMap.HTML, r.ClickMap.Available = "", false
		for i := range r.ClickMap.Variants {
			r.ClickMap.Variants[i].HTML, r.ClickMap.Variants[i].Available = "", false
		}
	}
}

// redact drops the ids of rows inside the workspace from a report that
// leaves through a link: only names and numbers remain.
func redact(r *CampaignReport) {
	r.Campaign.ID = ""
	r.NotSent = nil
	for i := range r.Headline.SentBy {
		r.Headline.SentBy[i].ProviderID = ""
	}
	for i := range r.Providers.Columns {
		r.Providers.Columns[i].ProviderID = ""
	}
	for i := range r.ClickMap.Links {
		r.ClickMap.Links[i].ID = ""
	}
	if r.Backlog != nil {
		for i := range r.Backlog.Providers {
			r.Backlog.Providers[i].ProviderID = ""
		}
	}
}
