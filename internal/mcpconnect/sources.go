package mcpconnect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// Selecting a source is a human API action, never an external tool that can
// enumerate raw resource names or guess workspace IDs. Discovery returns only
// these explicitly selected, revision-bound handles.
type MCPSourceSelection struct {
	SharingReviewed bool      `json:"sharingReviewed" doc:"Confirm review of the selected content, residual privacy risk and external client data policy before sharing"`
	Kind            string    `json:"kind" enum:"campaign,post,template,voice,list,segment,automation,report"`
	ID              uuid.UUID `json:"id"`
}

type MCPSourceHandle struct {
	Handle    string    `json:"handle"`
	Kind      string    `json:"kind"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Human-only selection receipts; never returned to an external client.
type MCPSelectedSource struct {
	MCPSourceHandle
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type MCPSelectedSources struct {
	Items []MCPSelectedSource `json:"items"`
}

type sourceContext struct {
	editorial.EmailContext
	Name        string
	Before      *editorial.Email
	Voice       *brandkit.BrandVoiceContent
	Snapshot    string
	Segment     *segments.SegmentDefinition
	Literals    map[string]string
	Flow        *MCPProposedFlow
	Emails      map[string]sourceContext
	TriggerName string
	Report      *reports.ModelSnapshot
}

type sourceProjection struct {
	editorial.Source
	Voice   *brandkit.BrandVoiceContent
	Segment *segments.SegmentDefinition
	Flow    *MCPFlowSource
	Report  *reports.ModelSnapshot
}

// This content-free identity survives proposal cleanup so recovery metadata
// cannot bypass a later resource-scope restriction.
type sourceReference struct {
	Kind           string
	TargetID       uuid.UUID
	BaseRevision   int64
	Snapshot       string
	Related        []sourceReference
	CreatesSegment bool
}

func sourceRef(row store.McpConnectSource) sourceReference {
	return sourceReference{Kind: row.Kind, TargetID: row.TargetID, BaseRevision: row.BaseRevision, Snapshot: row.Snapshot}
}

func kindAccess(p *authz.Principal, kind string, edit bool) error {
	var view, write authz.Permission
	switch kind {
	case "campaign":
		view, write = authz.CampaignsView, authz.CampaignsEdit
	case "report":
		view = authz.ReportsView
		if !p.Can(authz.CampaignsView) {
			return errors.New("insufficient_scope")
		}
	case "post":
		view, write = authz.WebsiteView, authz.WebsiteEdit
	case "template":
		view = authz.TemplatesView
	case "voice":
		view = authz.CampaignsView
	case "list":
		view = authz.SubscribersView
	case "segment":
		view, write = authz.SubscribersView, authz.SegmentsEdit
		if len(p.Scope.Lists) != 0 {
			return errors.New("insufficient_scope")
		}
	case "automation":
		view, write = authz.AutomationsView, authz.AutomationsEdit
	default:
		return errRequest
	}
	if !p.Can(view) || (edit && (write == "" || !p.Can(write))) {
		return errors.New("insufficient_scope")
	}
	return nil
}

func (s *Service) human(ctx context.Context, q *store.Queries, ws uuid.UUID, p *authz.Principal) (*authz.Principal, error) {
	if p == nil || !p.IsHuman() || p.WorkspaceID != ws {
		return nil, errGrant
	}
	a, err := s.membership(ctx, q, ws, p.UserID)
	if err != nil {
		return nil, err
	}
	fresh := *p
	fresh.Permissions = p.Permissions.Intersect(authz.Effective(a.RoleKey, a.Builtin, a.Permissions))
	fresh.Scope = p.Scope.Intersect(access.ParseScope(a.Scope))
	return &fresh, nil
}

func sourceRevision(value any) string {
	raw, _ := json.Marshal(value)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func sourceRead(ctx context.Context, q *store.Queries, ws uuid.UUID, p *authz.Principal, kind string, target uuid.UUID) (sourceContext, int64, bool, error) {
	var c sourceContext
	if err := kindAccess(p, kind, false); err != nil {
		return c, 0, false, err
	}
	switch kind {
	case "report":
		r, err := reports.ModelCampaign(ctx, q, p, target)
		if err != nil {
			return c, 0, false, err
		}
		c.Name = r.Name
		c.Snapshot = sourceRevision([]any{r.StartedAt, r.CompletedAt})
		return c, 0, false, nil
	case "campaign":
		domains := make([]string, 0, len(p.Scope.SenderDomains))
		for _, d := range p.Scope.SenderDomains {
			domains = append(domains, strings.ToLower(d))
		}
		r, err := q.AIDraftCampaign(ctx, store.AIDraftCampaignParams{WorkspaceID: ws, ID: target, ScopeLists: access.ScopeLists(p), ScopeDomains: domains})
		if err != nil {
			return c, 0, false, err
		}
		c.Name, c.Snapshot = r.Name, sourceRevision(r)
		c.Before = &editorial.Email{Subject: r.Subject, Preheader: r.Preheader, MJML: r.Mjml}
		return c, int64(r.Version), r.Status == "draft", nil
	case "post":
		r, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws, ID: target})
		if err != nil {
			return c, 0, false, err
		}
		// Campaign-backed posts use their campaign's scope; they cannot
		// bypass that boundary by being selected through the website.
		if r.Kind != website.KindWeb {
			return c, 0, false, errRequest
		}
		c.Name, c.Snapshot = r.Title, website.ProposalRevision(r)
		c.Before = &editorial.Email{Subject: r.Title, Preheader: r.Excerpt, MJML: r.Mjml}
		return c, int64(r.Version), r.Status == website.PostDraft && r.FirstPublishedAt == nil, nil
	case "template":
		r, err := q.MCPConnectTemplate(ctx, store.MCPConnectTemplateParams{WorkspaceID: ws, ID: target})
		if err != nil {
			return c, 0, false, err
		}
		c.Name, c.Snapshot = r.Name, sourceRevision(r)
		c.Before = &editorial.Email{MJML: r.Mjml}
		return c, 0, false, nil // A template is a reference, never an edit target.
	case "voice":
		if target != ws {
			return c, 0, false, errRequest
		}
		r, err := q.BrandVoiceLatest(ctx, ws)
		if err != nil {
			return c, 0, false, err
		}
		c.Snapshot = sourceRevision(r.Content)
		c.Voice = &brandkit.BrandVoiceContent{}
		if err := json.Unmarshal(r.Content, c.Voice); err != nil {
			return c, 0, false, err
		}
		return c, int64(r.Revision), false, nil
	case "list", "segment", "automation":
		return workflowSource(ctx, q, ws, p, kind, target)
	}
	return c, 0, false, errRequest
}

func (c *sourceContext) project(ctx context.Context) (sourceProjection, error) {
	out := sourceProjection{Source: editorial.Source{Links: []string{}, Images: []string{}, Tokens: []string{}}}
	if c.Report != nil {
		out.Report = c.Report
		return out, nil
	}
	if c.Segment != nil {
		definition, bindings := segments.ProjectDefinition(*c.Segment)
		c.Literals, out.Segment = bindings, &definition
		return out, nil
	}
	if c.Flow != nil {
		flow, err := c.projectFlow(ctx)
		out.Flow = &flow
		return out, err
	}
	if c.Voice != nil {
		voice := *c.Voice
		if err := voice.Validate(ctx); err != nil {
			return out, editorial.ErrPrivacy
		}
		for _, field := range []*string{&voice.Tone, &voice.Address, &voice.SentenceLength, &voice.PreferredWords, &voice.AvoidedWords, &voice.Examples} {
			text, safe := editorial.Text(*field)
			if !safe || editorial.Liquid.MatchString(text) || editorial.Reserved.MatchString(text) {
				return out, editorial.ErrPrivacy
			}
			*field = text
		}
		out.Voice = &voice
		return out, nil
	}
	if c.Before == nil {
		return out, nil // A selected list grants trigger context, not content.
	}
	var err error
	out.Source, err = c.Project(c.Before.MJML, nil, c.Before, 20_000)
	return out, err
}

func (s *Service) SelectSource(ctx context.Context, ws *store.Workspace, p *authz.Principal, grant uuid.UUID, in MCPSourceSelection) (MCPSourceHandle, error) {
	var out MCPSourceHandle
	if !in.SharingReviewed {
		return out, api.Invalid("body.sharingReviewed", tr(ctx, "problem.sharingReview"))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		g, err := q.MCPConnectGrantLock(ctx, grant)
		if err != nil || g.WorkspaceID != ws.ID {
			return api.NotFound("Connection")
		}
		human, err := s.human(ctx, q, ws.ID, p)
		if err != nil || (g.UserID != p.UserID && !human.Can(authz.MCPManage)) {
			return api.NotFound("Connection")
		}
		external, err := s.grantAuthority(ctx, q, g)
		if err != nil || !containsScope(g.Scopes, sourceScope(in.Kind)) {
			return api.Conflict(tr(ctx, "problem.sourcePermission"))
		}
		if _, _, _, err := sourceRead(ctx, q, ws.ID, human, in.Kind, in.ID); err != nil {
			return api.NotFound("Source")
		}
		c, revision, _, err := sourceRead(ctx, q, ws.ID, external, in.Kind, in.ID)
		if err != nil {
			return api.NotFound("Source")
		}
		// Repeating the same human selection, including after a lost
		// response, returns its saved handle without extending access.
		selected, err := q.MCPConnectSelectedSource(ctx, store.MCPConnectSelectedSourceParams{GrantID: g.ID, Kind: in.Kind, TargetID: in.ID, Snapshot: c.Snapshot, Now: s.now()})
		if err == nil {
			out = MCPSourceHandle{Handle: selected.Handle, Kind: selected.Kind, ExpiresAt: selected.ExpiresAt}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		count, err := q.MCPConnectSourceCount(ctx, g.ID)
		if err != nil {
			return err
		}
		if count >= 50 {
			return api.Conflict(tr(ctx, "problem.sourceLimit"))
		}
		handle, _ := token("rc_source_")
		c.BindingPrefix = handle
		if in.Kind == "report" {
			report, err := reports.ReadModelSnapshot(ctx, q, external, in.ID, s.now())
			if err != nil {
				return api.Conflict(tr(ctx, "problem.reportUnavailable"))
			}
			c.Report = &report
		}
		projected, err := c.project(ctx)
		if err != nil {
			return api.Invalid("body.id", tr(ctx, "problem.privateSource"))
		}
		local, _ := json.Marshal(c)
		raw, _ := json.Marshal(projected)
		now, expires := s.now(), s.now().Add(2*time.Hour)
		if g.ExpiresAt.Before(expires) {
			expires = g.ExpiresAt
		}
		if c.Report != nil && c.Report.AsOf.Add(30*24*time.Hour).Before(expires) {
			expires = c.Report.AsOf.Add(30 * 24 * time.Hour)
		}
		if err := q.MCPConnectSourceInsert(ctx, store.MCPConnectSourceInsertParams{Handle: handle, GrantID: g.ID, Kind: in.Kind, TargetID: in.ID,
			BaseRevision: revision, Snapshot: c.Snapshot, Context: local, Projected: raw, CreatedAt: now, ExpiresAt: expires}); err != nil {
			return err
		}
		out = MCPSourceHandle{Handle: handle, Kind: in.Kind, ExpiresAt: expires}
		return s.Audit.LogWith(audit.WithPrincipal(ctx, human), q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_connect.source_selected", TargetType: in.Kind, TargetID: in.ID.String(),
			Metadata: map[string]any{"via": "external_mcp", "grant_id": grant.String(), "revision": revision, "sharing_policy": editorial.SharingPolicy}})
	})
	return out, err
}

func (s *Service) SelectedSources(ctx context.Context, ws *store.Workspace, p *authz.Principal, grant uuid.UUID) (MCPSelectedSources, error) {
	out := MCPSelectedSources{Items: []MCPSelectedSource{}}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		g, err := q.MCPConnectGrantLock(ctx, grant)
		if err != nil || g.WorkspaceID != ws.ID {
			return api.NotFound("Connection")
		}
		fresh, err := s.human(ctx, q, ws.ID, p)
		if err != nil || (g.UserID != p.UserID && !fresh.Can(authz.MCPManage)) {
			return api.NotFound("Connection")
		}
		external, err := s.grantAuthority(ctx, q, g)
		if err != nil {
			return api.Conflict(tr(ctx, "problem.sourcePermission"))
		}
		rows, err := q.MCPConnectSources(ctx, store.MCPConnectSourcesParams{GrantID: g.ID, Now: s.now()})
		if err != nil {
			return err
		}
		for _, row := range rows {
			if !containsScope(g.Scopes, sourceScope(row.Kind)) {
				continue
			}
			if checkSource(ctx, q, fresh, sourceRef(row), true) != nil || checkSource(ctx, q, external, sourceRef(row), true) != nil {
				continue
			}
			var local sourceContext
			if err := json.Unmarshal(row.Context, &local); err != nil {
				return err
			}
			out.Items = append(out.Items, MCPSelectedSource{MCPSourceHandle: MCPSourceHandle{Handle: row.Handle, Kind: row.Kind, ExpiresAt: row.ExpiresAt}, ID: row.TargetID, Name: local.Name})
		}
		return nil
	})
	return out, err
}

func sourceScope(kind string) string {
	if kind == "report" {
		return "reports:read"
	}
	return "editorial:read"
}

func checkSource(ctx context.Context, q *store.Queries, p *authz.Principal, row sourceReference, exact bool) error {
	current, revision, _, err := sourceRead(ctx, q, p.WorkspaceID, p, row.Kind, row.TargetID)
	if err != nil {
		return err
	}
	if exact && (revision != row.BaseRevision || current.Snapshot != row.Snapshot) {
		return errors.New("source_changed")
	}
	for _, related := range row.Related {
		if err := checkSource(ctx, q, p, related, exact); err != nil {
			return err
		}
	}
	return nil
}
