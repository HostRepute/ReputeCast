package accessibility

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service runs the accessibility checks for the workspace's emails, keeps
// the workspace policy and records the fixes applied from Review & send.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	// Versions lists an email's language versions around the one given
	// (the source until content languages exist). Nil: the email has one.
	Versions VersionSource
	// Language is the workspace's default content language ("en").
	Language func(ctx context.Context, ws *store.Workspace) string
	// Titles enqueues reading the titles of linked pages it doesn't know.
	Titles TitleQueue
	// Drafts lists automation drafts' emails (for the policy's impact).
	Drafts DraftSource

	cacheMu sync.Mutex
	cache   map[string]cachedReport
}

// VersionSource returns every language version of a campaign, with the
// given version in place of its stored content (the composer's unsaved
// edits).
type VersionSource func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, current Version) ([]Version, error)

// TitleQueue asks for linked pages' titles to be read in the background.
type TitleQueue interface {
	Enqueue(ctx context.Context, urls []string) error
}

// DraftEmail is an email of an automation draft.
type DraftEmail struct {
	AutomationID uuid.UUID
	Automation   string
	Version      int
	LiveVersion  *int
	// Step is how the flow names the step ("step 3 B").
	Step     string
	Versions []Version
}

// DraftSource lists the emails of the workspace's automation drafts.
type DraftSource func(ctx context.Context, ws *store.Workspace) ([]DraftEmail, error)

type cachedReport struct {
	report AccessibilityReport
	at     time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) language(ctx context.Context, ws *store.Workspace) string {
	if s.Language != nil {
		if l := s.Language(ctx, ws); l != "" {
			return l
		}
	}
	if l := ContentLanguage(ctx, ws); l != "" {
		return l
	}
	return "en"
}

// AccessibilityPolicy is the workspace's policy as Settings › General shows it.
type AccessibilityPolicy struct {
	OnSerious string `json:"onSerious" enum:"warn,block"`
	MinTextPx int    `json:"minTextPx" minimum:"12" maximum:"16"`
	// UpdatedBy and UpdatedAt say who changed it last (empty until someone does).
	UpdatedBy string     `json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	// CanManage is workspace:manage; Helpers are members who hold it.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// AccessibilityPolicyInput changes the policy.
type AccessibilityPolicyInput struct {
	OnSerious string `json:"onSerious" enum:"warn,block"`
	MinTextPx int    `json:"minTextPx" minimum:"12" maximum:"16"`
}

// policy reads the workspace's policy; the default warns at 12 px.
func (s *Service) policy(ctx context.Context, q *store.Queries, wsID uuid.UUID) (store.AccessibilityPolicyRow, bool, error) {
	row, err := q.AccessibilityPolicy(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.AccessibilityPolicyRow{OnSerious: PolicyWarn, MinTextPx: DefaultMinTextPx}, false, nil
	}
	return row, err == nil, err
}

// Settings returns the policy the checks apply: Warn or Block, the
// minimum size.
func (s *Service) Settings(ctx context.Context, wsID uuid.UUID) (string, int, error) {
	row, _, err := s.policy(ctx, s.Q, wsID)
	if err != nil {
		return PolicyWarn, DefaultMinTextPx, err
	}
	return row.OnSerious, int(row.MinTextPx), nil
}

// Policy returns the policy with who may change it.
func (s *Service) Policy(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AccessibilityPolicy, error) {
	row, saved, err := s.policy(ctx, s.Q, ws.ID)
	if err != nil {
		return AccessibilityPolicy{}, err
	}
	out := AccessibilityPolicy{OnSerious: row.OnSerious, MinTextPx: int(row.MinTextPx), CanManage: p.Can(authz.WorkspaceManage), Helpers: []string{}}
	if saved {
		at := row.UpdatedAt
		out.UpdatedAt, out.UpdatedBy = &at, row.UpdatedByName
	}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.WorkspaceManage)
	}
	return out, nil
}

// UpdatePolicy saves the policy and records the change.
func (s *Service) UpdatePolicy(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AccessibilityPolicyInput) (AccessibilityPolicy, error) {
	if in.OnSerious != PolicyWarn && in.OnSerious != PolicyBlock {
		return AccessibilityPolicy{}, api.Invalid("body.onSerious", "Choose Warn or Block.")
	}
	if in.MinTextPx < MinMinTextPx || in.MinTextPx > MaxMinTextPx {
		return AccessibilityPolicy{}, api.Invalid("body.minTextPx", "Choose a minimum text size from 12 to 16 px.")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, _, err := s.policy(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if cur.OnSerious == in.OnSerious && int(cur.MinTextPx) == in.MinTextPx {
			return nil
		}
		var by *uuid.UUID
		if p != nil && p.UserID != uuid.Nil {
			id := p.UserID
			by = &id
		}
		if err := q.AccessibilityUpsertPolicy(ctx, store.AccessibilityUpsertPolicyParams{WorkspaceID: ws.ID, OnSerious: in.OnSerious, MinTextPx: int32(in.MinTextPx),
			UpdatedBy: by, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.accessibility_policy", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"onSerious": cur.OnSerious, "minTextPx": cur.MinTextPx},
			After:  map[string]any{"onSerious": in.OnSerious, "minTextPx": in.MinTextPx}})
	})
	if err != nil {
		return AccessibilityPolicy{}, err
	}
	s.forget(ws.ID)
	return s.Policy(ctx, ws, p)
}

// helpers names up to two active members holding perm.
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, perm authz.Permission) []string {
	out := []string{}
	rows, err := s.Q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
			if len(out) == 2 {
				break
			}
		}
	}
	return out
}

// options are the check options for a workspace, with the linked pages'
// titles the link check knows.
func (s *Service) options(ctx context.Context, wsID uuid.UUID) (Options, error) {
	policy, min, err := s.Settings(ctx, wsID)
	if err != nil {
		return Options{}, err
	}
	return Options{Policy: policy, MinTextPx: min, Now: s.now}, nil
}

// run checks versions and fills in what the link check knows about the
// pages its vague links point at. Pages it hasn't read yet are queued; the
// next check suggests their titles.
func (s *Service) run(ctx context.Context, wsID uuid.UUID, versions []Version) (AccessibilityReport, error) {
	opts, err := s.options(ctx, wsID)
	if err != nil {
		return AccessibilityReport{}, err
	}
	r, err := Check(versions, opts)
	if err != nil {
		return r, err
	}
	var urls []string
	for _, is := range r.Issues {
		if is.Kind == KindLinkText && fetchable(is.Element.Href) {
			urls = append(urls, is.Element.Href)
		}
	}
	if len(urls) == 0 {
		return r, nil
	}
	titles, missing, err := s.titles(ctx, urls)
	if err != nil {
		return r, err
	}
	if len(titles) > 0 {
		opts.Titles = titles
		if r, err = Check(versions, opts); err != nil {
			return r, err
		}
	}
	if len(missing) > 0 && s.Titles != nil {
		if err := s.Titles.Enqueue(ctx, missing); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "queue link titles", "err", err)
		}
	}
	return r, nil
}

// titleTTL is how long a page's title is trusted.
const titleTTL = 7 * 24 * time.Hour

func (s *Service) titles(ctx context.Context, urls []string) (map[string]PageTitle, []string, error) {
	rows, err := s.Q.AccessibilityLinkTitles(ctx, urls)
	if err != nil {
		return nil, nil, err
	}
	out := map[string]PageTitle{}
	known := map[string]bool{}
	for _, r := range rows {
		if r.FetchedAt.After(s.now().Add(-titleTTL)) {
			known[r.Url] = true
		}
		if r.Title != "" {
			out[r.Url] = PageTitle{Title: r.Title, Lang: r.Lang}
		}
	}
	var missing []string
	seen := map[string]bool{}
	for _, u := range urls {
		if !known[u] && !seen[u] {
			seen[u] = true
			missing = append(missing, u)
		}
	}
	return out, missing, nil
}

// fetchable is a link whose page can be read: http or https, no Liquid.
func fetchable(href string) bool {
	h := strings.ToLower(strings.TrimSpace(href))
	return (strings.HasPrefix(h, "https://") || strings.HasPrefix(h, "http://")) && !strings.Contains(h, "{{") && !strings.Contains(h, "{%")
}

// CampaignContent is the content of a campaign to check.
type CampaignContent struct {
	ID     uuid.UUID
	Name   string
	Status string
	MJML   string
	HTML   string
	// Hash fingerprints the content (campaigns.content_hash).
	Hash string
}

// versionsOf returns a campaign's language versions with current in place
// of its stored content.
func (s *Service) versionsOf(ctx context.Context, ws *store.Workspace, id uuid.UUID, current Version) ([]Version, error) {
	if s.Versions != nil {
		return s.Versions(ctx, ws, id, current)
	}
	return []Version{current}, nil
}

// CampaignReport checks a campaign as it is saved.
func (s *Service) CampaignReport(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AccessibilityReport, error) {
	c, err := s.Q.AccessibilityCampaign(ctx, store.AccessibilityCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AccessibilityReport{}, api.MapError(err, "Campaign")
	}
	return s.Report(ctx, ws, CampaignContent{ID: c.ID, Name: c.Name, Status: c.Status, MJML: c.Mjml, HTML: c.Html, Hash: c.ContentHash})
}

// Report checks a campaign's content (all its language versions). Saved
// content is cached by its hash, so Review & send and the gallery don't
// compile the same email twice.
func (s *Service) Report(ctx context.Context, ws *store.Workspace, c CampaignContent) (AccessibilityReport, error) {
	versions, err := s.versionsOf(ctx, ws, c.ID, Version{Lang: s.language(ctx, ws), MJML: c.MJML, HTML: c.HTML, Source: true})
	if err != nil {
		return AccessibilityReport{}, err
	}
	key := ""
	if c.Hash != "" {
		key = ws.ID.String() + ":" + c.ID.String() + ":" + c.Hash + ":" + versionsKey(versions)
		if r, ok := s.cached(key); ok {
			return r, nil
		}
	}
	r, err := s.run(ctx, ws.ID, versions)
	if err != nil {
		return r, err
	}
	if key != "" && !s.hasMissingTitles(r) {
		s.remember(key, r)
	}
	return r, nil
}

// AccessibilityCheckRequest is the composer's content to check, unsaved
// edits included. Lang names the version on the canvas; the other versions
// are checked as saved.
type AccessibilityCheckRequest struct {
	Lang string `json:"lang,omitempty" maxLength:"35" doc:"The language version being edited (the source when empty)"`
	MJML string `json:"mjml,omitempty" maxLength:"2000000"`
	HTML string `json:"html,omitempty" maxLength:"2500000"`
}

// CheckCampaign checks the composer's current content.
func (s *Service) CheckCampaign(ctx context.Context, ws *store.Workspace, id uuid.UUID, in AccessibilityCheckRequest) (AccessibilityReport, error) {
	c, err := s.Q.AccessibilityCampaign(ctx, store.AccessibilityCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AccessibilityReport{}, api.MapError(err, "Campaign")
	}
	if strings.TrimSpace(in.MJML) == "" && strings.TrimSpace(in.HTML) == "" {
		return s.Report(ctx, ws, CampaignContent{ID: c.ID, Name: c.Name, Status: c.Status, MJML: c.Mjml, HTML: c.Html, Hash: c.ContentHash})
	}
	lang := in.Lang
	if lang == "" {
		lang = s.language(ctx, ws)
	}
	versions, err := s.versionsOf(ctx, ws, c.ID, Version{Lang: lang, MJML: in.MJML, HTML: in.HTML, Source: in.Lang == "" || BaseLanguage(in.Lang) == BaseLanguage(s.language(ctx, ws))})
	if err != nil {
		return AccessibilityReport{}, err
	}
	r, err := s.run(ctx, ws.ID, versions)
	if errors.Is(err, ErrCompile) {
		return r, api.Err(http.StatusUnprocessableEntity, "accessibility_unchecked", "Couldn't check the email: the MJML doesn't compile. Your edits are saved.")
	}
	return r, err
}

func versionsKey(vs []Version) string {
	h := sha256.New()
	for _, v := range vs {
		h.Write([]byte(v.Lang))
		h.Write([]byte{0})
		h.Write([]byte(v.MJML))
		h.Write([]byte{0})
		h.Write([]byte(v.HTML))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)[:12])
}

// cacheTTL bounds how long a report is reused (policy changes clear it).
const cacheTTL = 10 * time.Minute

func (s *Service) cached(key string) (AccessibilityReport, bool) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	c, ok := s.cache[key]
	if !ok || s.now().Sub(c.at) > cacheTTL {
		return AccessibilityReport{}, false
	}
	return c.report, true
}

func (s *Service) remember(key string, r AccessibilityReport) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cache == nil || len(s.cache) > 500 {
		s.cache = map[string]cachedReport{}
	}
	s.cache[key] = cachedReport{report: r, at: s.now()}
}

// forget drops a workspace's cached reports (its policy changed).
func (s *Service) forget(wsID uuid.UUID) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for k := range s.cache {
		if strings.HasPrefix(k, wsID.String()+":") {
			delete(s.cache, k)
		}
	}
}

// hasMissingTitles reports a link-text issue still waiting for its page's
// title: such a report isn't cached, so the suggestion appears once read.
func (s *Service) hasMissingTitles(r AccessibilityReport) bool {
	for _, is := range r.Issues {
		if is.Kind == KindLinkText && fetchable(is.Element.Href) && len(is.Versions) > 0 && is.Versions[0].Suggestion == "" {
			return true
		}
	}
	return false
}
