package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// AICampaignBriefInput names only explicit editorial context. No audience,
// recipient preview, automatic history search or remote fetch is available.
type AICampaignBriefInput struct {
	SharingReviewed bool       `json:"sharingReviewed" doc:"Confirm review of the brief, selected content, residual personal-data risk and provider data policy before sharing"`
	Brief           string     `json:"brief" minLength:"1" maxLength:"4000"`
	Name            string     `json:"name" minLength:"1" maxLength:"200"`
	Language        string     `json:"language" enum:"en,vi,de,es,fr"`
	TemplateID      string     `json:"templateId,omitempty" maxLength:"100"`
	CampaignID      *uuid.UUID `json:"campaignId,omitempty"`
	BaseRevision    int64      `json:"baseRevision,omitempty" minimum:"0"`
	Links           []string   `json:"links,omitempty" maxItems:"10"`
	VoiceRevision   *int32     `json:"voiceRevision,omitempty" minimum:"0"`
}

// AIDraftEmail is unrendered content; HTML is always compiled on the server.
type AIDraftEmail struct {
	Subject   string `json:"subject" minLength:"1" maxLength:"250"`
	Preheader string `json:"preheader" maxLength:"500"`
	MJML      string `json:"mjml" minLength:"1" maxLength:"200000"`
}

type AICampaignProposal struct {
	ID           uuid.UUID     `json:"id"`
	Name         string        `json:"name"`
	CampaignID   *uuid.UUID    `json:"campaignId,omitempty"`
	BaseRevision int64         `json:"baseRevision"`
	Before       *AIDraftEmail `json:"before,omitempty"`
	Content      AIDraftEmail  `json:"content"`
	HTML         string        `json:"html"`
	SubjectCheck AISubjectIdea `json:"subjectCheck"`
	Sent         AISent        `json:"sent"`
	ExpiresAt    time.Time     `json:"expiresAt"`
	Decision     string        `json:"decision" enum:",accepted,rejected"`
}

type AIDraftDecision struct {
	Decision string `json:"decision" enum:"accepted,rejected"`
}

type AIDraftDecided struct {
	Decision string     `json:"decision" enum:"accepted,rejected"`
	ID       *uuid.UUID `json:"id,omitempty"`
	Revision *int64     `json:"revision,omitempty"`
}

// No contact records are read. Selected editorial prose may contain personal
// information that detectors miss. URL and token bindings remain local.
type draftContext struct {
	editorial.EmailContext
	Name         string
	Language     string
	TemplateID   string
	TemplateHash [32]byte
	Before       *AIDraftEmail
	Sent         AISent
	// Agent checkpoints keep the validated preview so polling never
	// recompiles an unchanged email. It stays local, outside model context.
	PreviewHTML string `json:",omitempty"`
}

type campaignDraftTask struct {
	Brief, Language, Source string
	Subject, Preheader      string
	Links, Images, Tokens   []string
}

func draftProblem(ctx context.Context, status int, key string) error {
	return problem(ctx, status, "ai_draft_"+key, msgs.M("draft."+key), nil)
}

// runDraft allows two repairs, with immutable projected context, voice and
// connection. Only a fixed validation code is returned to the model: neither
// rejected output nor locally restored bindings may become repair context.
// draftWriteTimeout lets a draft's answer reach the person after all three
// attempts of runDraft, each bounded by the draft timeout.
const draftWriteTimeout = 3*longestTimeout + time.Minute

func (s *Service) runDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, t task, validate func(llm.Response) (string, error)) (result, error) {
	c, err := s.usable(ctx, ws, t.feature)
	if err != nil {
		return result{}, err
	}
	if err := s.withVoice(ctx, ws, &t); err != nil {
		return result{}, err
	}
	base := t.req.System
	var input, output int
	var cost int64
	for attempt := range 3 {
		r, err := s.send(ctx, ws, p, c, t)
		if err != nil {
			return result{}, err
		}
		if r.Refusal != "" {
			return result{}, draftProblem(ctx, 422, "refused")
		}
		input += r.Response.InputTokens
		output += r.Response.OutputTokens
		cost += r.CostMicros
		code := "schema"
		if r.Response.StopReason == llm.StopEnd {
			code, err = validate(r.Response)
		}
		if err != nil {
			s.finish(ctx, ws, p, r.RequestID, OutcomeError, "validation unavailable")
			return result{}, err
		}
		if code == "" {
			r.Response.InputTokens, r.Response.OutputTokens, r.CostMicros = input, output, cost
			return r, nil
		}
		s.finish(ctx, ws, p, r.RequestID, OutcomeError, "invalid draft")
		if attempt < 2 {
			t.req.System = base + "\nThe previous attempt failed the local " + code + " check. Generate a complete replacement from the original brief, preserving all selected constraints and bindings."
		}
	}
	return result{}, draftProblem(ctx, 502, "invalid")
}

func draftActor(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, use bool, permission authz.Permission) (*authz.Principal, error) {
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return nil, err
	}
	if use && !fresh.Can(authz.AIUse) {
		return nil, api.Forbidden(authz.AIUse, nil)
	}
	if !fresh.Can(permission) {
		return nil, api.Forbidden(permission, nil)
	}
	return fresh, nil
}

func draftCampaign(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (store.AIDraftCampaignRow, error) {
	domains := make([]string, 0, len(p.Scope.SenderDomains))
	for _, domain := range p.Scope.SenderDomains {
		domains = append(domains, strings.ToLower(domain))
	}
	c, err := q.AIDraftCampaign(ctx, store.AIDraftCampaignParams{WorkspaceID: ws.ID, ID: id, ScopeLists: access.ScopeLists(p), ScopeDomains: domains})
	if err != nil {
		return c, api.MapError(err, "Campaign")
	}
	return c, nil
}

func (s *Service) GenerateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AICampaignBriefInput) (AICampaignProposal, error) {
	var out AICampaignProposal
	if !in.SharingReviewed {
		return out, draftProblem(ctx, 422, "sharing_review")
	}
	p, err := draftActor(ctx, s.Q, ws, p, true, authz.CampaignsEdit)
	if err != nil {
		return out, err
	}
	if err := validateDraftJSON(in); err != nil || (in.CampaignID == nil) != (in.BaseRevision == 0) || (in.CampaignID != nil && in.TemplateID != "") {
		return out, draftProblem(ctx, 422, "input")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || draftReserved.MatchString(in.Name+in.Brief) || voiceLiquid.MatchString(in.Brief) {
		return out, draftProblem(ctx, 422, "input")
	}
	// A brief is editorial text, not an implicit URL capability. Link
	// destinations must be selected explicitly and bound before serialization.
	if voiceURL.MatchString(in.Brief) {
		return out, draftProblem(ctx, 422, "input")
	}
	brief, ok := voiceText(in.Brief)
	if !ok || strings.TrimSpace(brief) == "" {
		return out, draftProblem(ctx, 422, "privacy")
	}
	c := draftContext{Name: in.Name, Language: in.Language, TemplateID: in.TemplateID, EmailContext: editorial.EmailContext{Bindings: []editorial.Binding{}}}
	var source string
	if in.CampaignID != nil {
		current, err := draftCampaign(ctx, s.Q, ws, p, *in.CampaignID)
		if err != nil {
			return out, err
		}
		if int64(current.Version) != in.BaseRevision || current.Status != "draft" {
			return out, draftProblem(ctx, 409, "changed")
		}
		c.Name = current.Name // A content proposal never renames its target.
		c.Before = &AIDraftEmail{Subject: current.Subject, Preheader: current.Preheader, MJML: current.Mjml}
		source = current.Mjml
	} else {
		source, _ = templates.StarterMJML("blank", "")
		if in.TemplateID != "" {
			if !p.Can(authz.TemplatesView) {
				return out, api.Forbidden(authz.TemplatesView, nil)
			}
			tpl, err := s.Campaigns.Templates.Get(ctx, ws, in.TemplateID, "")
			if err != nil {
				return out, err
			}
			source = tpl.MJML
			c.TemplateHash = sha256.Sum256([]byte(source))
		}
	}
	t, err := c.projectSource(ctx, source, in.Links)
	if err != nil {
		return out, err
	}
	t.Brief, t.Language = brief, languageName(in.Language)
	checkSource := func(ctx context.Context, q *store.Queries, fresh *authz.Principal) error {
		if !fresh.Can(authz.CampaignsEdit) {
			return api.Forbidden(authz.CampaignsEdit, nil)
		}
		if in.CampaignID != nil {
			current, err := draftCampaign(ctx, q, ws, fresh, *in.CampaignID)
			if err != nil {
				return err
			}
			if int64(current.Version) != in.BaseRevision || current.Status != "draft" {
				return draftProblem(ctx, 409, "changed")
			}
		}
		return c.checkTemplate(ctx, q, ws, fresh, true)
	}
	var email AIDraftEmail
	res, err := s.runDraft(ctx, ws, p, task{feature: FeatureDraft, targetKind: "campaign", targetID: in.CampaignID, req: campaignDraftPrompt(t), selectedVoice: in.VoiceRevision, checkSource: checkSource}, func(r llm.Response) (string, error) {
		email = AIDraftEmail{}
		if decodeDraft([]byte(r.Text), &email) != nil {
			return "schema", nil //nolint:nilerr // A safe validation code requests bounded repair; an error aborts generation.
		}
		if c.restoreEmail(&email) != nil {
			return "email_bindings", nil //nolint:nilerr // Bound values stay local; the model receives only a repair code.
		}
		if in.CampaignID == nil {
			email.MJML = s.BrandKit.Starter(ctx, ws, email.MJML)
		}
		c.FinalPolicy = savedDraftPolicy(email.MJML)
		if err := c.validateSaved(ctx, s.Q, ws, email); err != nil {
			if errors.Is(err, errDraftContent) {
				return "email_content", nil
			}
			return "", err
		}
		if _, _, errs := mjml.Checked(email.MJML, ""); errs != nil {
			return "mjml", nil
		}
		return "", nil
	})
	if err != nil {
		return out, err
	}
	c.Sent = res.sent(chars(brief) + chars(t.Source) + chars(t.Subject) + chars(t.Preheader))
	now := s.now()
	content, err := json.Marshal(email)
	if err != nil {
		return out, err
	}
	contextJSON, err := json.Marshal(c)
	if err != nil {
		return out, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, true, authz.CampaignsEdit)
		if err != nil {
			return err
		}
		if err := checkSource(ctx, q, fresh); err != nil {
			return err
		}
		conn, err := s.current(ctx, q, ws.ID, res.conn)
		if err != nil {
			return err
		}
		if !conn.connected() || !conn.featureOn(FeatureDraft) {
			return draftProblem(ctx, 409, "unavailable")
		}
		if err := q.AIDraftInsert(ctx, store.AIDraftInsertParams{ID: res.RequestID, WorkspaceID: ws.ID, UserID: p.UserID, Kind: "campaign", TargetID: in.CampaignID,
			BaseRevision: in.BaseRevision, Context: contextJSON, Content: content, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.draft.propose", TargetType: "ai_request", TargetID: res.RequestID.String(), Metadata: map[string]any{"kind": "campaign", "base_revision": in.BaseRevision, "sharing_policy": editorial.SharingPolicy}})
	})
	if err != nil {
		s.finish(ctx, ws, p, res.RequestID, OutcomeDiscarded, "access changed")
		return out, err
	}
	return s.CampaignProposal(ctx, ws, p, res.RequestID)
}

// projectSource keeps markup and destinations local. Both campaign and flow
// emails give the model only editorial text and opaque capabilities.
func (c *draftContext) projectSource(ctx context.Context, source string, links []string) (campaignDraftTask, error) {
	t, err := c.Project(source, links, (*editorial.Email)(c.Before), maxEmailChars)
	if err != nil {
		key := "source"
		if errors.Is(err, editorial.ErrPrivacy) {
			key = "privacy"
		} else if errors.Is(err, editorial.ErrInput) {
			key = "input"
		}
		return campaignDraftTask{}, draftProblem(ctx, 422, key)
	}
	return campaignDraftTask{Source: t.Source, Subject: t.Subject, Preheader: t.Preheader, Links: t.Links, Images: t.Images, Tokens: t.Tokens}, nil
}

var savedDraftPolicy = editorial.SavedPolicy

func (c draftContext) checkTemplate(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, exact bool) error {
	if c.TemplateID == "" {
		return nil
	}
	if !p.Can(authz.TemplatesView) {
		return api.Forbidden(authz.TemplatesView, nil)
	}
	if strings.HasPrefix(c.TemplateID, templates.StarterPrefix) {
		return nil // Built-in templates are immutable; brand styles were snapshotted.
	}
	id, err := uuid.Parse(c.TemplateID)
	if err != nil {
		return api.NotFound("Template")
	}
	source, err := q.AIDraftTemplate(ctx, store.AIDraftTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Template")
	}
	if exact && sha256.Sum256([]byte(source)) != c.TemplateHash {
		return draftProblem(ctx, 409, "changed")
	}
	return nil
}

var errDraftContent = errors.New("invalid draft content")

func (c draftContext) validateSaved(ctx context.Context, q *store.Queries, ws *store.Workspace, email AIDraftEmail) error {
	if err := validateDraftJSON(email); err != nil {
		return errDraftContent
	}
	if err := mjml.CheckGenerated(draftMarker.ReplaceAllString(email.MJML, ""), c.FinalPolicy); err != nil {
		return errDraftContent
	}
	fields, err := q.ComposerCustomFields(ctx, ws.ID)
	if err != nil {
		return err
	}
	custom := make([]liquid.CustomField, 0, len(fields))
	for _, f := range fields {
		custom = append(custom, liquid.CustomField{Key: f.Key})
	}
	keys := []string{}
	for _, token := range liquid.Catalog(custom) {
		keys = append(keys, token.Key)
	}
	for _, src := range []string{email.Subject, email.Preheader, email.MJML} {
		if len(liquid.Check(src, keys)) != 0 {
			return errDraftContent
		}
	}
	return nil
}

var draftReserved = editorial.Reserved
var draftMarker = editorial.Marker

// Huma's generated schema also validates model output. json.Unmarshal alone
// silently accepts missing/unknown fields and out-of-range array sizes.
func validateDraftJSON[T any](v T) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return validateDraftRaw[T](raw)
}

func validateDraftRaw[T any](raw []byte) error {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	r := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	schema := huma.SchemaFromType(r, reflect.TypeFor[T]())
	result := &huma.ValidateResult{}
	huma.Validate(r, schema, huma.NewPathBuffer(nil, 0), huma.ModeWriteToServer, data, result)
	if len(result.Errors) > 0 {
		return errors.New("draft violates schema")
	}
	return nil
}

func decodeDraft[T any](raw []byte, out *T) error {
	if err := validateDraftRaw[T](raw); err != nil {
		return err
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	return nil
}

func campaignDraftPrompt(t campaignDraftTask) llm.Request {
	data, _ := json.Marshal(t)
	req := userRequest(systemPrompt, textPart(
		"Draft one complete email in the requested language, using the brief and source only. Links, images and tokens are opaque bindings: preserve every handle exactly, links in href and images in src. Use token handles in the text. Do not add URLs or Liquid. Do not write a compliance footer; the server adds the unchanged footer locally.",
		`Return {"subject":"…","preheader":"…","mjml":"<mjml><mj-body>…</mj-body></mjml>"}. Subject at most 250 characters; preheader at most 500. Use ordinary mj-section, mj-column, mj-text, mj-button, mj-image, mj-divider and mj-spacer. No raw HTML blocks, CSS stylesheets, comments, includes, conditional blocks or event attributes. Do not invent facts, offers, dates or audiences.`,
		tag("draft_context", string(data)),
	))
	req.SchemaName = "campaign_draft"
	req.Schema = map[string]any{"type": "object", "properties": map[string]any{"subject": map[string]any{"type": "string"}, "preheader": map[string]any{"type": "string"}, "mjml": map[string]any{"type": "string"}, "refusal": map[string]any{"type": "string"}}}
	req.MaxTokens, req.Task = 8000, t
	return req
}

func (c draftContext) restoreEmail(email *AIDraftEmail) error {
	return c.Restore((*editorial.Email)(email), (*editorial.Email)(c.Before))
}

func (s *Service) CampaignProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AICampaignProposal, error) {
	out, err := s.campaignProposal(ctx, ws, p, id)
	if err == nil {
		out.HTML = mjml.Compile(out.Content.MJML).HTML
	}
	return out, err
}

func (s *Service) campaignProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AICampaignProposal, error) {
	var out AICampaignProposal
	var c draftContext
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, false, authz.CampaignsView)
		if err != nil {
			return err
		}
		r, err := q.AIDraftGet(ctx, store.AIDraftGetParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Now: s.now()})
		if err != nil {
			return api.MapError(err, "Proposal")
		}
		if r.Kind != "campaign" {
			return api.NotFound("Proposal")
		}
		target := r.TargetID
		if target == nil {
			target = r.ResultID
		}
		if target != nil {
			if _, err := draftCampaign(ctx, q, ws, fresh, *target); err != nil {
				return err
			}
		}
		if err := json.Unmarshal(r.Context, &c); err != nil {
			return err
		}
		if err := c.checkTemplate(ctx, q, ws, fresh, false); err != nil {
			return err
		}
		if err := json.Unmarshal(r.Content, &out.Content); err != nil {
			return err
		}
		out.ID, out.Name, out.CampaignID, out.BaseRevision, out.Before = r.ID, c.Name, r.TargetID, r.BaseRevision, c.Before
		out.Sent, out.ExpiresAt, out.Decision = c.Sent, r.ExpiresAt, r.Decision
		return nil
	})
	if err != nil {
		return out, err
	}
	out.SubjectCheck = checkSubject(out.Content.Subject, c.Language)
	return out, nil
}

type AIPendingDraft struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AIPendingDrafts struct {
	Items []AIPendingDraft `json:"items"`
	Next  *uuid.UUID       `json:"next,omitempty"`
}

type pendingDraftInput struct {
	api.WorkspaceInput
	Target string `query:"target" format:"uuid" maxLength:"36"`
	Before string `query:"before" format:"uuid" maxLength:"36"`
	All    bool   `query:"all"`
}

// Recovery uses the same current-source authorization as opening a proposal,
// without compiling email previews or making model requests while listing.
func (s *Service) pendingDrafts(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, in *pendingDraftInput) (AIPendingDrafts, error) {
	out := AIPendingDrafts{Items: []AIPendingDraft{}}
	var target, before *uuid.UUID
	for key, value := range map[string]string{"target": in.Target, "before": in.Before} {
		if value == "" {
			continue
		}
		id, err := uuid.Parse(value)
		if err != nil {
			return out, draftProblem(ctx, 422, "input")
		}
		if key == "target" {
			target = &id
		} else {
			before = &id
		}
	}
	permission := authz.CampaignsView
	if kind == "automation" {
		permission = authz.AutomationsView
	}
	fresh, err := draftActor(ctx, s.Q, ws, p, false, permission)
	if err != nil {
		return out, err
	}
	ids, err := s.Q.AIDraftPending(ctx, store.AIDraftPendingParams{WorkspaceID: ws.ID, UserID: fresh.UserID, Kind: kind, TargetID: target, BeforeID: before, AllTargets: in.All, Now: s.now()})
	if err != nil {
		return out, err
	}
	for _, id := range ids {
		var row AIPendingDraft
		var err error
		if kind == "campaign" {
			proposal, readErr := s.campaignProposal(ctx, ws, fresh, id)
			row, err = AIPendingDraft{ID: proposal.ID, Name: proposal.Name, ExpiresAt: proposal.ExpiresAt}, readErr
		} else {
			proposal, readErr := s.automationProposal(ctx, ws, fresh, id)
			row, err = AIPendingDraft{ID: proposal.ID, Name: proposal.Name, ExpiresAt: proposal.ExpiresAt}, readErr
		}
		if err != nil {
			var status huma.StatusError
			if errors.As(err, &status) && (status.GetStatus() == 403 || status.GetStatus() == 404 || status.GetStatus() == 409) {
				continue
			}
			return out, err
		}
		out.Items = append(out.Items, row)
	}
	if len(ids) == 20 {
		out.Next = &ids[len(ids)-1]
	}
	return out, nil
}

func (s *Service) DecideCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIDraftDecision) (AIDraftDecided, error) {
	var out AIDraftDecided
	if in.Decision != "accepted" && in.Decision != "rejected" {
		return out, draftProblem(ctx, 422, "input")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, false, authz.CampaignsEdit)
		if err != nil {
			return err
		}
		r, err := q.AIDraftLock(ctx, store.AIDraftLockParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Now: s.now()})
		if err != nil {
			return api.MapError(err, "Proposal")
		}
		if r.Kind != "campaign" {
			return api.NotFound("Proposal")
		}
		target := r.TargetID
		if target == nil {
			target = r.ResultID
		}
		if target != nil {
			if _, err := draftCampaign(ctx, q, ws, fresh, *target); err != nil {
				return err
			}
		}
		if r.Decision != "" {
			if r.Decision != in.Decision {
				return draftProblem(ctx, 409, "decided")
			}
			out = AIDraftDecided{Decision: r.Decision, ID: r.ResultID, Revision: r.ResultRevision}
			return nil
		}
		out.Decision = in.Decision
		if in.Decision == "accepted" {
			var c draftContext
			var email AIDraftEmail
			if err := json.Unmarshal(r.Context, &c); err != nil {
				return err
			}
			if err := decodeDraft(r.Content, &email); err != nil {
				return draftProblem(ctx, 422, "invalid")
			}
			if err := c.checkTemplate(ctx, q, ws, fresh, true); err != nil {
				return err
			}
			if err := c.validateSaved(ctx, q, ws, email); err != nil {
				return draftProblem(ctx, 422, "invalid")
			}
			result, revision, err := s.Campaigns.AcceptDraftIn(ctx, tx, ws, fresh, r.TargetID, r.BaseRevision, c.Name, email.Subject, email.Preheader, email.MJML, audit.ProposalOrigin{ProposalID: id})
			if err != nil {
				return err
			}
			out.ID, out.Revision = &result, &revision
		} else if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.draft.reject", TargetType: "ai_request", TargetID: id.String()}); err != nil {
			return err
		}
		if err := q.AIDraftDecide(ctx, store.AIDraftDecideParams{ID: id, Decision: in.Decision, ResultID: out.ID, ResultRevision: out.Revision}); err != nil {
			return err
		}
		_, err = q.AIRequestSetOutcome(ctx, store.AIRequestSetOutcomeParams{ID: id, WorkspaceID: ws.ID, UserID: &p.UserID, Outcome: in.Decision, DecidedAt: ptr(s.now())})
		return err
	})
	return out, err
}

func registerDrafts(h huma.API, s *Service) {
	base := "/workspaces/{workspace}/ai/drafts/campaigns"
	tags := []string{"AI assistant"}
	api.Get(h, api.Op{ID: "list-campaign-draft-proposals", Path: base, Tags: tags, Summary: "Recover your pending proposals for this draft or new campaigns", Access: api.WorkspaceHuman(authz.CampaignsView)},
		func(ctx context.Context, in *pendingDraftInput) (*api.Body[AIPendingDrafts], error) {
			out, err := s.pendingDrafts(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), "campaign", in)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "generate-campaign-draft", Path: base, Tags: tags, Summary: "Propose a campaign from an explicit brief, without saving a campaign", Access: api.WorkspaceHuman(authz.AIUse), WriteTimeout: draftWriteTimeout},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body AICampaignBriefInput
		}) (*api.Body[AICampaignProposal], error) {
			out, err := s.GenerateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-draft-proposal", Path: base + "/{proposal}", Tags: tags, Summary: "Read your unexpired campaign proposal", Access: api.WorkspaceHuman(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Proposal uuid.UUID `path:"proposal"`
		}) (*api.Body[AICampaignProposal], error) {
			out, err := s.CampaignProposal(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Proposal)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decide-campaign-draft-proposal", Path: base + "/{proposal}/decision", Tags: tags, Summary: "Accept exactly this proposal or reject it; never send or schedule", Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Proposal uuid.UUID `path:"proposal"`
			Body     AIDraftDecision
		}) (*api.Body[AIDraftDecided], error) {
			out, err := s.DecideCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Proposal, in.Body)
			return api.OK(out), err
		})
}
