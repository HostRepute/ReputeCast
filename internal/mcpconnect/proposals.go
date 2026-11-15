package mcpconnect

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/website/render"
)

type MCPProposedEmail struct {
	Subject   string `json:"subject" minLength:"1" maxLength:"250"`
	Preheader string `json:"preheader" maxLength:"500"`
	MJML      string `json:"mjml" minLength:"1" maxLength:"100000"`
}

type MCPProposalInput struct {
	Key    string           `json:"key" jsonschema:"idempotency key: 1–80 ASCII letters, numbers, hyphens or underscores"`
	Kind   string           `json:"kind" enum:"campaign,post"`
	Name   string           `json:"name" minLength:"1" maxLength:"200"`
	Source string           `json:"source,omitempty" jsonschema:"selected source handle; omit for a new blank draft"`
	Change bool             `json:"change" jsonschema:"true proposes an exact-revision change; false creates a new draft copy"`
	Email  MCPProposedEmail `json:"email"`
}

type proposalContext struct {
	Draft  sourceContext
	Source *sourceReference
}

// Receipts contain no restored text, links, target IDs or reviewer identity.
type MCPProposalReceipt struct {
	ID        uuid.UUID `json:"id"`
	Decision  string    `json:"decision"`
	ExpiresAt time.Time `json:"expiresAt"`
}

var proposalKey = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

func emailShape(kind string, e MCPProposedEmail) bool {
	subject, preheader := 250, 500
	if kind == "post" {
		subject, preheader = 200, 200
	}
	return strings.TrimSpace(e.Subject) != "" && utf8.ValidString(e.Subject+e.Preheader+e.MJML) &&
		utf8.RuneCountInString(e.Subject) <= subject && utf8.RuneCountInString(e.Preheader) <= preheader && len(e.MJML) > 0 && len(e.MJML) <= 100_000
}

func checkSavedEmail(ctx context.Context, q *store.Queries, ws uuid.UUID, c sourceContext, kind string, e MCPProposedEmail) error {
	if !emailShape(kind, e) || mjml.CheckGenerated(editorial.Marker.ReplaceAllString(e.MJML, ""), c.FinalPolicy) != nil {
		return errors.New("invalid_content")
	}
	fields, err := q.ComposerCustomFields(ctx, ws)
	if err != nil {
		return err
	}
	custom := make([]liquid.CustomField, 0, len(fields))
	for _, field := range fields {
		custom = append(custom, liquid.CustomField{Key: field.Key})
	}
	keys := []string{}
	for _, token := range liquid.Catalog(custom) {
		keys = append(keys, token.Key)
	}
	for _, text := range []string{e.Subject, e.Preheader, e.MJML} {
		if len(liquid.Check(text, keys)) != 0 {
			return errors.New("invalid_content")
		}
	}
	return nil
}

func receipt(p store.McpConnectProposal) MCPProposalReceipt {
	return MCPProposalReceipt{ID: p.ID, Decision: p.Decision, ExpiresAt: p.ExpiresAt}
}

// A purged proposal still owns its idempotency key. The caller holds the
// grant lock through both this check and the insertion.
func (s *Service) proposalRetry(ctx context.Context, q *store.Queries, c *connectionCall, key string, digest [32]byte) (*MCPProposalReceipt, error) {
	if !proposalKey.MatchString(key) {
		return nil, errRequest
	}
	prior, err := q.MCPConnectProposalByKey(ctx, store.MCPConnectProposalByKeyParams{GrantID: c.Grant.ID, RequestKey: key})
	if err == nil {
		if !bytes.Equal(prior.RequestHash, digest[:]) {
			return nil, errors.New("idempotency_conflict")
		}
		var local proposalContext
		if err := json.Unmarshal(prior.Context, &local); err != nil {
			return nil, err
		}
		if err := proposalAccess(ctx, q, c.Principal, prior, local, true); err != nil {
			return nil, errors.New("insufficient_scope")
		}
		out := receipt(prior)
		return &out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	n, err := q.MCPConnectProposalCount(ctx, c.Grant.ID)
	if err != nil {
		return nil, err
	}
	if n >= 200 {
		return nil, errors.New("proposal_limit")
	}
	return nil, nil
}

// The caller holds the grant lock. Deduplication, staging and its audit all
// commit together, with no product draft or in-app AI request created here.
func (s *Service) propose(ctx context.Context, q *store.Queries, c *connectionCall, in MCPProposalInput) (MCPProposalReceipt, error) {
	var out MCPProposalReceipt
	if (in.Kind != "campaign" && in.Kind != "post") || kindAccess(c.Principal, in.Kind, true) != nil ||
		strings.TrimSpace(in.Name) == "" || utf8.RuneCountInString(in.Name) > 200 || !emailShape(in.Kind, in.Email) {
		return out, errRequest
	}
	raw, _ := json.Marshal(in)
	digest := sha256.Sum256(raw)
	prior, err := s.proposalRetry(ctx, q, c, in.Key, digest)
	if err != nil {
		return out, err
	}
	if prior != nil {
		return *prior, nil
	}
	local := proposalContext{Draft: sourceContext{Name: in.Name}}
	var target *uuid.UUID
	var revision int64
	if in.Source != "" {
		if !containsScope(c.Grant.Scopes, "editorial:read") {
			return out, errors.New("insufficient_scope")
		}
		source, err := q.MCPConnectSource(ctx, store.MCPConnectSourceParams{Handle: in.Source, GrantID: c.Grant.ID, Now: s.now()})
		if err != nil || (source.Kind != in.Kind && (source.Kind != "template" || in.Change)) || checkSource(ctx, q, c.Principal, sourceRef(source), true) != nil {
			return out, errors.New("source_changed")
		}
		ref := sourceRef(source)
		local.Source = &ref
		if err := json.Unmarshal(source.Context, &local.Draft); err != nil {
			return out, err
		}
		if in.Change {
			_, _, editable, err := sourceRead(ctx, q, c.Grant.WorkspaceID, c.Principal, in.Kind, source.TargetID)
			if err != nil || !editable {
				return out, errors.New("source_changed")
			}
			target, revision = &source.TargetID, source.BaseRevision
		}
	} else {
		if in.Change {
			return out, errRequest
		}
		starter, _ := templates.StarterMJML("blank", "")
		if _, err := local.Draft.Project(starter, nil, nil, 20_000); err != nil {
			return out, err
		}
	}
	if !in.Change {
		local.Draft.Name = in.Name
	}
	if _, ok := editorial.Text(in.Name); !ok || editorial.URL.MatchString(in.Name) || editorial.Liquid.MatchString(in.Name) {
		return out, errRequest
	}
	if err := local.Draft.Restore((*editorial.Email)(&in.Email), local.Draft.Before); err != nil {
		return out, errors.New("invalid_content")
	}
	local.Draft.FinalPolicy = editorial.SavedPolicy(in.Email.MJML)
	if err := checkSavedEmail(ctx, q, c.Grant.WorkspaceID, local.Draft, in.Kind, in.Email); err != nil {
		return out, err
	}
	html, _, issues := mjml.Checked(in.Email.MJML, "")
	if issues != nil {
		return out, errors.New("invalid_content")
	}
	if in.Kind == "post" {
		web, err := render.Render(in.Email.MJML, render.Options{Title: in.Email.Subject})
		if err != nil {
			return out, errors.New("invalid_content")
		}
		html = web.Body
	}
	content, _ := json.Marshal(in.Email)
	bindings, _ := json.Marshal(local)
	return s.saveProposal(ctx, q, c, store.MCPConnectProposalInsertParams{RequestKey: in.Key, RequestHash: digest[:], Kind: in.Kind,
		TargetID: target, BaseRevision: revision, Context: bindings, Content: content, Preview: html})
}

func (s *Service) saveProposal(ctx context.Context, q *store.Queries, c *connectionCall, in store.MCPConnectProposalInsertParams) (MCPProposalReceipt, error) {
	var out MCPProposalReceipt
	ws, err := q.GetWorkspace(ctx, c.Grant.WorkspaceID)
	if err != nil {
		return out, err
	}
	settings := struct {
		Hours int `json:"approval_expiry_hours"`
	}{Hours: 2}
	if err := json.Unmarshal(ws.McpSettings, &settings); err != nil {
		return out, err
	}
	now := s.now()
	expires := now.Add(time.Duration(max(1, min(settings.Hours, 168))) * time.Hour)
	if c.Grant.ExpiresAt.Before(expires) {
		expires = c.Grant.ExpiresAt
	}
	id := uuid.Must(uuid.NewV7())
	in.ID, in.GrantID, in.WorkspaceID, in.UserID = id, c.Grant.ID, c.Grant.WorkspaceID, c.Grant.UserID
	in.CreatedAt, in.ExpiresAt, in.PurgeAt = now, expires, expires.Add(7*24*time.Hour)
	row, err := q.MCPConnectProposalInsert(ctx, in)
	if err != nil {
		return out, err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_connect.proposed", TargetType: "external_proposal", TargetID: id.String(),
		Metadata: map[string]any{"via": "external_mcp", "grant_id": c.Grant.ID.String(), "kind": in.Kind, "base_revision": in.BaseRevision}}); err != nil {
		return out, err
	}
	if err := notify.ToUserWith(ctx, q, c.Grant.UserID, notify.Notification{
		WorkspaceID: &ws.ID, Kind: notify.KindExternalProposal,
		TitleText: msgs.M("notify.title").Keep(nil), BodyText: msgs.M("notify.body").Keep(nil),
		Link: "/w/" + ws.Slug + "/proposals/" + id.String(),
		Data: map[string]any{"proposal_id": id.String(), "grant_id": c.Grant.ID.String()},
	}); err != nil {
		return out, err
	}
	return receipt(row), nil
}

type MCPProposalView struct {
	ID           uuid.UUID           `json:"id"`
	Kind         string              `json:"kind"`
	Name         string              `json:"name"`
	ClientName   string              `json:"clientName"`
	Verified     bool                `json:"verified"`
	UserName     string              `json:"userName"`
	CreatedAt    time.Time           `json:"createdAt"`
	CanDecide    bool                `json:"canDecide"`
	BaseRevision int64               `json:"baseRevision"`
	TargetID     *uuid.UUID          `json:"targetId,omitempty"`
	Before       *MCPProposedEmail   `json:"before,omitempty"`
	Email        MCPProposedEmail    `json:"email"`
	HTML         string              `json:"html"`
	Decision     string              `json:"decision"`
	Blocked      string              `json:"blocked" enum:",expired,revoked,changed,purged,permission"`
	ExpiresAt    time.Time           `json:"expiresAt"`
	ResultID     *uuid.UUID          `json:"resultId,omitempty"`
	Workflow     *MCPWorkflowContent `json:"workflow,omitempty"`
	BeforeFlow   *MCPProposedFlow    `json:"beforeFlow,omitempty"`
	TriggerName  string              `json:"triggerName,omitempty"`
}

type MCPProposalDecision struct {
	Decision string `json:"decision" enum:"kept,discarded,accepted,rejected"`
}

type MCPProposalListItem struct {
	ID         uuid.UUID `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	ClientName string    `json:"clientName"`
	Verified   bool      `json:"verified"`
	Change     bool      `json:"change"`
	Decision   string    `json:"decision"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Purged     bool      `json:"purged"`
	Revoked    bool      `json:"revoked"`
}

type MCPProposalList struct {
	Items      []MCPProposalListItem `json:"items"`
	NextCursor string                `json:"nextCursor,omitempty"`
}

func (s *Service) Proposals(ctx context.Context, ws *store.Workspace, p *authz.Principal, cursor string) (MCPProposalList, error) {
	out := MCPProposalList{Items: []MCPProposalListItem{}}
	var before *uuid.UUID
	if cursor != "" {
		id, err := uuid.Parse(cursor)
		if err != nil {
			return out, api.Invalid("cursor", tr(ctx, "problem.cursor"))
		}
		before = &id
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := s.human(ctx, q, ws.ID, p)
		if err != nil {
			return api.NotFound("Proposals")
		}
		rows, err := q.MCPConnectProposals(ctx, store.MCPConnectProposalsParams{WorkspaceID: ws.ID, UserID: fresh.UserID, Manager: fresh.Can(authz.MCPManage), BeforeID: before})
		if err != nil {
			return err
		}
		if len(rows) > 50 {
			rows = rows[:50]
			out.NextCursor = rows[49].ID.String()
		}
		for _, row := range rows {
			var local proposalContext
			if err := json.Unmarshal(row.Context, &local); err != nil {
				return err
			}
			a := store.McpConnectProposal{Kind: row.Kind, WorkspaceID: ws.ID, ResultID: row.ResultID}
			if proposalAccess(ctx, q, fresh, a, local, false) != nil {
				continue
			}
			purged := row.PurgedAt != nil || !row.PurgeAt.After(s.now())
			if purged {
				row.Name = ""
			}
			out.Items = append(out.Items, MCPProposalListItem{ID: row.ID, Kind: row.Kind, Name: row.Name, ClientName: row.ClientName,
				Verified: row.Verified, Change: row.TargetID != nil, Decision: row.Decision, ExpiresAt: row.ExpiresAt, Purged: purged, Revoked: row.RevokedAt != nil})
		}
		return nil
	})
	return out, err
}

// Source access is checked for both people and the still-authorized grant.
// Revocation does not prevent permitted people from recovering local content.
func proposalAccess(ctx context.Context, q *store.Queries, p *authz.Principal, row store.McpConnectProposal, local proposalContext, edit bool) error {
	if err := kindAccess(p, row.Kind, edit); err != nil {
		return err
	}
	if local.Source != nil {
		if local.Source.CreatesSegment && kindAccess(p, "segment", edit) != nil {
			return errors.New("insufficient_scope")
		}
		if err := checkSource(ctx, q, p, *local.Source, false); err != nil {
			return err
		}
	}
	if row.ResultID != nil {
		_, _, _, err := sourceRead(ctx, q, row.WorkspaceID, p, row.Kind, *row.ResultID)
		return err
	}
	return nil
}

func (s *Service) proposalBlocked(ctx context.Context, q *store.Queries, g store.McpConnectGrant, row store.McpConnectProposal, local proposalContext) string {
	if row.PurgedAt != nil || !row.PurgeAt.After(s.now()) {
		return "purged"
	}
	if row.Decision != "" {
		return ""
	}
	if !row.ExpiresAt.After(s.now()) {
		return "expired"
	}
	p, err := s.grantAuthority(ctx, q, g)
	if err != nil || !containsScope(g.Scopes, "drafts:propose") {
		return "revoked"
	}
	if proposalAccess(ctx, q, p, row, local, true) != nil {
		return "permission"
	}
	if local.Source != nil && checkSource(ctx, q, p, *local.Source, true) != nil {
		return "changed"
	}
	return ""
}

func (s *Service) Proposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (MCPProposalView, error) {
	var out MCPProposalView
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.MCPConnectProposal(ctx, store.MCPConnectProposalParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.NotFound("Proposal")
		}
		g, err := q.MCPConnectGrantRead(ctx, row.GrantID)
		if err != nil {
			return err
		}
		fresh, err := s.human(ctx, q, ws.ID, p)
		if err != nil || (row.UserID != p.UserID && !fresh.Can(authz.MCPManage)) {
			return api.NotFound("Proposal")
		}
		var local proposalContext
		if err := json.Unmarshal(row.Context, &local); err != nil {
			return err
		}
		if err := proposalAccess(ctx, q, fresh, row, local, false); err != nil {
			return api.NotFound("Proposal")
		}
		client, err := q.MCPConnectClient(ctx, g.ClientID)
		if err != nil {
			return err
		}
		user, err := q.GetUser(ctx, row.UserID)
		if err != nil {
			return err
		}
		out = MCPProposalView{ID: row.ID, Kind: row.Kind, ClientName: client.Name, Verified: client.Verified,
			UserName: user.Name, CreatedAt: row.CreatedAt, CanDecide: proposalAccess(ctx, q, fresh, row, local, true) == nil,
			BaseRevision: row.BaseRevision, TargetID: row.TargetID, Decision: row.Decision, ExpiresAt: row.ExpiresAt, ResultID: row.ResultID,
			Blocked: s.proposalBlocked(ctx, q, g, row, local)}
		if out.Blocked != "purged" {
			if row.Kind == "segment" || row.Kind == "automation" {
				out.Workflow = &MCPWorkflowContent{}
				out.TriggerName = local.Draft.TriggerName
				if err := json.Unmarshal(row.Content, out.Workflow); err != nil {
					return err
				}
				if row.TargetID != nil {
					out.BeforeFlow = local.Draft.Flow
				}
			} else {
				if err := json.Unmarshal(row.Content, &out.Email); err != nil {
					return err
				}
			}
			out.Name = local.Draft.Name
			out.Before, out.HTML = (*MCPProposedEmail)(local.Draft.Before), row.Preview
		}
		return nil
	})
	return out, err
}

func (s *Service) DecideProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in MCPProposalDecision) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// All writers lock grant before proposal, including replay and revoke.
		row, err := q.MCPConnectProposal(ctx, store.MCPConnectProposalParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.NotFound("Proposal")
		}
		g, err := q.MCPConnectGrantLock(ctx, row.GrantID)
		if err != nil {
			return err
		}
		row, err = q.MCPConnectProposalLock(ctx, store.MCPConnectProposalLockParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		fresh, err := s.human(ctx, q, ws.ID, p)
		if err != nil || (row.UserID != p.UserID && !fresh.Can(authz.MCPManage)) {
			return api.NotFound("Proposal")
		}
		ctx = audit.WithPrincipal(ctx, fresh)
		var local proposalContext
		if err := json.Unmarshal(row.Context, &local); err != nil {
			return err
		}
		if err := proposalAccess(ctx, q, fresh, row, local, true); err != nil {
			return api.NotFound("Proposal")
		}
		valid := (row.TargetID == nil && (in.Decision == "kept" || in.Decision == "discarded")) || (row.TargetID != nil && (in.Decision == "accepted" || in.Decision == "rejected"))
		if !valid {
			return api.Conflict(tr(ctx, "problem.decisionKind"))
		}
		if row.Decision != "" {
			if row.Decision == in.Decision {
				return nil
			}
			return api.Conflict(tr(ctx, "problem.decided"))
		}
		if in.Decision == "kept" || in.Decision == "accepted" {
			if blocked := s.proposalBlocked(ctx, q, g, row, local); blocked != "" {
				return api.Err(409, "mcp_proposal_"+blocked, tr(ctx, "problem.blocked"))
			}
			var email MCPProposedEmail
			if row.Kind == "campaign" || row.Kind == "post" {
				if err := json.Unmarshal(row.Content, &email); err != nil || checkSavedEmail(ctx, q, ws.ID, local.Draft, row.Kind, email) != nil {
					return api.Conflict(tr(ctx, "problem.content"))
				}
			}
			origin := audit.ProposalOrigin{ProposalID: id, GrantID: &g.ID}
			var result uuid.UUID
			var revision int64
			switch row.Kind {
			case "campaign":
				result, revision, err = s.Campaigns.AcceptDraftIn(ctx, tx, ws, fresh, row.TargetID, row.BaseRevision, local.Draft.Name, email.Subject, email.Preheader, email.MJML, origin)
			case "post":
				if row.TargetID == nil {
					result, err = s.Website.KeepDraftIn(ctx, tx, ws, fresh, email.Subject, email.Preheader, email.MJML, origin)
				} else {
					result, err = s.Website.AcceptDraftIn(ctx, tx, ws, fresh, *row.TargetID, row.BaseRevision, local.Draft.Snapshot, email.Subject, email.Preheader, email.MJML, origin)
				}
				revision = row.BaseRevision + 1
			case "automation", "segment":
				result, revision, err = s.keepWorkflow(ctx, tx, ws, fresh, row, local, origin)
			}
			if err != nil {
				return err
			}
			row.ResultID, row.ResultRevision = &result, &revision
			// Validators may outlive the deadline; authority must still be
			// valid when the application transaction commits.
			if !row.ExpiresAt.After(s.now()) || !g.ExpiresAt.After(s.now()) {
				return api.Conflict(tr(ctx, "problem.expired"))
			}
		}
		now := s.now()
		if err := q.MCPConnectProposalDecide(ctx, store.MCPConnectProposalDecideParams{ID: id, Decision: in.Decision, DecidedBy: &fresh.UserID,
			Now: &now, ResultID: row.ResultID, ResultRevision: row.ResultRevision, PurgeAt: now.Add(7 * 24 * time.Hour)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_connect." + in.Decision, TargetType: "external_proposal", TargetID: id.String(),
			Metadata: audit.ProposalOrigin{ProposalID: id, GrantID: &g.ID}.Metadata()})
	})
}
