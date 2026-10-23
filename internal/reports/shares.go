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

// ShareInput creates a read-only link.
type ShareInput struct {
	// ExpiresInDays makes the link stop working after that many days; 0 or
	// absent keeps it until it is revoked.
	ExpiresInDays int `json:"expiresInDays,omitempty" minimum:"0" maximum:"365" doc:"0 keeps the link until it is revoked"`
}

// CreateShare makes a read-only link to a campaign's report.
func (s *Service) CreateShare(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, in ShareInput) (ReportShareLink, error) {
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
	var expires *time.Time
	if in.ExpiresInDays > 0 {
		t := now.Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)
		expires = &t
	}
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
			CreatedBy: by, CreatedAt: now, ExpiresAt: expires})
		if err != nil {
			return err
		}
		after := map[string]any{"campaign": c.Name, "expiresAt": expires}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "report_share.create", TargetType: "report_share", TargetID: id.String(),
			TargetName: c.Name, After: after, Metadata: map[string]any{"campaignId": c.ID.String()}})
	})
	if err != nil {
		return ReportShareLink{}, err
	}
	return ReportShareLink{ID: row.ID.String(), URL: s.URLs.Report(token), CreatedAt: row.CreatedAt, CreatedByName: byName, ExpiresAt: row.ExpiresAt}, nil
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
		l := ReportShareLink{ID: r.ID.String(), CreatedAt: r.CreatedAt, CreatedByName: r.CreatedByName, ExpiresAt: r.ExpiresAt, Views: r.Views, LastViewedAt: r.LastViewedAt}
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
	_ = s.Q.ReportShareTouch(ctx, store.ReportShareTouchParams{At: &now, ID: link.ID})
	return SharedCampaignReport{Workspace: ws.Name, ExpiresAt: link.ExpiresAt, Report: rep}, nil
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
