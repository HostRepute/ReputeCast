package campaigns

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/retryaudience"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// ComposerAudience says who receives a campaign: lists and segments, minus
// excluded lists and segments, and who on them (Earn: everyone, paid
// members only at a tier and above, or free subscribers only).
type ComposerAudience struct {
	Lists           []string `json:"lists"`
	Segments        []string `json:"segments"`
	ExcludeLists    []string `json:"excludeLists"`
	ExcludeSegments []string `json:"excludeSegments"`
	// Paid is "paid" for paid members only, "free" for free subscribers
	// only; empty for everyone. PaidTier is the lowest tier of paid
	// members ("Insider and above"); empty for any paid tier.
	Paid     string `json:"paid,omitempty" enum:"paid,free,"`
	PaidTier string `json:"paidTier,omitempty"`
}

// ComposerAudienceInput is an audience in requests (every part optional).
type ComposerAudienceInput struct {
	Lists           []string `json:"lists,omitempty" maxItems:"100"`
	Segments        []string `json:"segments,omitempty" maxItems:"100"`
	ExcludeLists    []string `json:"excludeLists,omitempty" maxItems:"100"`
	ExcludeSegments []string `json:"excludeSegments,omitempty" maxItems:"100"`
	Paid            string   `json:"paid,omitempty" enum:"paid,free," doc:"paid: paid members only; free: free subscribers only; empty: everyone on the lists"`
	PaidTier        string   `json:"paidTier,omitempty" doc:"With paid: the lowest tier (the tier and above); empty for any paid tier"`
}

func (in ComposerAudienceInput) def() ComposerAudience {
	return ComposerAudience(in)
}

// dbAudience is the stored shape of campaigns.audience.
type dbAudience struct {
	Lists           []string `json:"lists,omitempty"`
	Segments        []string `json:"segments,omitempty"`
	ExcludeLists    []string `json:"exclude_lists,omitempty"`
	ExcludeSegments []string `json:"exclude_segments,omitempty"`
	ExcludeTiers    []string `json:"exclude_tiers,omitempty"`
	Paid            string   `json:"paid,omitempty"`
	PaidTier        string   `json:"paid_tier,omitempty"`
	// IncludeSunset and IncludeUnengaged are set on Review & send (the
	// sunset policy's override, "Include them"); a save keeps them.
	IncludeSunset    bool  `json:"include_sunset,omitempty"`
	IncludeUnengaged *bool `json:"include_unengaged,omitempty"`
	// RetryOf makes the campaign a retry of an issue (retry.go): a save
	// keeps it, so the retry still goes only to the blocked addresses.
	RetryOf string `json:"retry_of,omitempty"`
}

func audienceFromDB(raw json.RawMessage) ComposerAudience {
	var a dbAudience
	_ = json.Unmarshal(raw, &a)
	p := paidaccess.Parse(raw)
	return ComposerAudience{Lists: nonNil(a.Lists), Segments: nonNil(a.Segments), ExcludeLists: nonNil(a.ExcludeLists), ExcludeSegments: nonNil(a.ExcludeSegments),
		Paid: p.Who, PaidTier: p.Tier}
}

func (a ComposerAudience) toDB(prev json.RawMessage) json.RawMessage {
	var d dbAudience
	_ = json.Unmarshal(prev, &d) // keeps exclude_tiers set elsewhere
	d.Lists, d.Segments, d.ExcludeLists, d.ExcludeSegments, d.Paid, d.PaidTier = a.Lists, a.Segments, a.ExcludeLists, a.ExcludeSegments, a.Paid, a.PaidTier
	b, _ := json.Marshal(d)
	return b
}

func (a ComposerAudience) empty() bool { return len(a.Lists) == 0 && len(a.Segments) == 0 }

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ComposerABVariant is one variant of an A/B test.
type ComposerABVariant struct {
	Key       string `json:"key" enum:"A,B,C" doc:"Variant A is the campaign itself"`
	Subject   string `json:"subject,omitempty" maxLength:"998"`
	Preheader string `json:"preheader,omitempty" maxLength:"500"`
	// MJML and HTML are set for content tests (variants B and C).
	MJML string `json:"mjml,omitempty" maxLength:"2000000"`
	HTML string `json:"html,omitempty" maxLength:"2000000"`
	// SendTime is the local send time for send-time tests ("14:00").
	SendTime string `json:"sendTime,omitempty" pattern:"^([01]\\d|2[0-3]):[0-5]\\d$"`
}

// ComposerABTest compares variants on a sample, then sends the winner to the rest.
type ComposerABTest struct {
	Kind        string              `json:"kind" enum:"subject,content,send_time"`
	Variants    []ComposerABVariant `json:"variants" minItems:"2" maxItems:"3"`
	TestPercent int                 `json:"testPercent" minimum:"5" maximum:"50" doc:"Share of the audience in the test"`
	Metric      string              `json:"metric" enum:"human_opens,clicks,manual"`
	// DurationHours is how long the test runs before the winner is sent.
	DurationHours int `json:"durationHours" minimum:"1" maximum:"72"`
	// BestTime sends each recipient at their usual open time (send-time tests).
	BestTime bool   `json:"bestTime,omitempty"`
	Winner   string `json:"winner,omitempty"`
	// The decision rule (review AN-1): a variant wins only when every
	// variant reached MinSample recipients (500 when unset) and its lead is
	// significant at Confidence (0.95 when unset); otherwise the rest get
	// NoWinner: "default", the first variant (A).
	MinSample  int     `json:"minSample,omitempty" minimum:"0" maximum:"1000000" doc:"Recipients each variant needs before a winner can be picked (default 500)"`
	Confidence float64 `json:"confidence,omitempty" minimum:"0" maximum:"0.999" doc:"How sure the test must be (default 0.95)"`
	NoWinner   string  `json:"noWinner,omitempty" enum:"default" doc:"What the rest get without a clear winner: default, the first variant"`
}

// dbABTest is the stored shape of campaigns.ab_test.
type dbABTest struct {
	Kind          string          `json:"kind"`
	Variants      []dbABVariant   `json:"variants"`
	TestPercent   int             `json:"test_percent"`
	Metric        string          `json:"metric"`
	DurationHours int             `json:"duration_hours"`
	BestTime      bool            `json:"best_time,omitempty"`
	Winner        string          `json:"winner,omitempty"`
	MinSample     int             `json:"min_sample,omitempty"`
	Confidence    float64         `json:"confidence,omitempty"`
	NoWinner      string          `json:"no_winner,omitempty"`
	Extra         json.RawMessage `json:"-"`
}

type dbABVariant struct {
	Key       string `json:"key"`
	Subject   string `json:"subject,omitempty"`
	Preheader string `json:"preheader,omitempty"`
	MJML      string `json:"mjml,omitempty"`
	HTML      string `json:"html,omitempty"`
	SendTime  string `json:"send_time,omitempty"`
}

func abFromDB(raw json.RawMessage) *ComposerABTest {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var d dbABTest
	if json.Unmarshal(raw, &d) != nil || d.Kind == "" {
		return nil
	}
	out := &ComposerABTest{Kind: d.Kind, TestPercent: d.TestPercent, Metric: d.Metric, DurationHours: d.DurationHours, BestTime: d.BestTime, Winner: d.Winner,
		MinSample: d.MinSample, Confidence: d.Confidence, NoWinner: d.NoWinner, Variants: []ComposerABVariant{}}
	if out.Metric == "" {
		out.Metric = "human_opens"
	}
	for _, v := range d.Variants {
		out.Variants = append(out.Variants, ComposerABVariant(v))
	}
	return out
}

func (t *ComposerABTest) toDB() json.RawMessage {
	if t == nil {
		return nil
	}
	d := dbABTest{Kind: t.Kind, TestPercent: t.TestPercent, Metric: t.Metric, DurationHours: t.DurationHours, BestTime: t.BestTime, Winner: t.Winner,
		MinSample: t.MinSample, Confidence: t.Confidence, NoWinner: t.NoWinner}
	for _, v := range t.Variants {
		d.Variants = append(d.Variants, dbABVariant(v))
	}
	b, _ := json.Marshal(d)
	return b
}

// ComposerContentSettings are composer options applied to the stored HTML.
type ComposerContentSettings struct {
	// Minify shrinks the HTML (whitespace, inline CSS) to stay under Gmail's clipping limit.
	Minify bool `json:"minify"`
}

// ComposerSenderRef is the "From" identity.
type ComposerSenderRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ComposerRoute is the sending route of a campaign: its planned split, or
// the Broadcasts rule when it has none yet.
type ComposerRoute struct {
	Source string       `json:"source" enum:"campaign,rule,none"`
	Shares []RouteShare `json:"shares"`
}

// ComposerAccess tells the composer what the caller may do.
type ComposerAccess struct {
	// Editable is false once a campaign started sending.
	Editable       bool   `json:"editable"`
	ReadOnlyReason string `json:"readOnlyReason,omitempty"`
	CanEdit        bool   `json:"canEdit"`
	// CanSend is campaigns:send. Without it "Review & send" becomes "Request approval".
	CanSend     bool     `json:"canSend"`
	SendHelpers []string `json:"sendHelpers" doc:"Members who hold campaigns:send"`
	// EditHelpers are members who hold campaigns:edit, named to people who
	// can't edit (the Accessibility panel's read-only note).
	EditHelpers []string `json:"editHelpers,omitempty" doc:"Members who hold campaigns:edit (when the caller can't edit)"`
	// CanManageRouting is providers:manage; others see the route read-only.
	CanManageRouting bool `json:"canManageRouting"`
	// CanSendTestToOthers lets test emails go to other addresses (campaigns:send).
	CanSendTestToOthers bool `json:"canSendTestToOthers"`
	CanEditTemplates    bool `json:"canEditTemplates"`
	CanViewContacts     bool `json:"canViewContacts"`
}

// ComposerCampaign is everything the composer loads.
type ComposerCampaign struct {
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	Status        string                  `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	Subject       string                  `json:"subject"`
	Preheader     string                  `json:"preheader"`
	Sender        *ComposerSenderRef      `json:"sender,omitempty"`
	ReplyTo       string                  `json:"replyTo"`
	Audience      ComposerAudience        `json:"audience"`
	AudienceLabel string                  `json:"audienceLabel"`
	AudienceCount *int64                  `json:"audienceCount,omitempty"`
	EditorMode    string                  `json:"editorMode" enum:"visual,mjml,html"`
	MJML          string                  `json:"mjml"`
	HTML          string                  `json:"html"`
	PlainText     string                  `json:"plainText"`
	Version       int                     `json:"version"`
	VersionCount  int                     `json:"versionCount"`
	ContentHash   string                  `json:"contentHash"`
	ABTest        *ComposerABTest         `json:"abTest,omitempty"`
	Settings      ComposerContentSettings `json:"settings"`
	ScheduledAt   *time.Time              `json:"scheduledAt,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
	UpdatedBy     string                  `json:"updatedBy"`
	AutosavedAt   *time.Time              `json:"autosavedAt,omitempty"`
	Route         ComposerRoute           `json:"route"`
	Readiness     *ReadinessSummary       `json:"readiness,omitempty"`
	Access        ComposerAccess          `json:"access"`
	// Language is the language of the content returned: the source's, or
	// the language version asked for with ?language=. SourceLanguage is the
	// language the campaign is written in. Both are empty without content
	// languages.
	Language       string `json:"language"`
	SourceLanguage string `json:"sourceLanguage"`
	// DigestID is set for a recurring digest's template (DigestRole
	// "template"), its next issue ("next") and its issues ("issue"): the
	// composer opens in digest mode (internal/digests).
	DigestID   string `json:"digestId,omitempty"`
	DigestRole string `json:"digestRole,omitempty" enum:"template,next,issue,"`
	// Approval is the campaign's pending or approved request: editing
	// cancels it, so the composer asks first.
	Approval *ComposerApproval `json:"approval,omitempty"`
}

var editableStatuses = map[string]bool{"draft": true, "approval": true, "approved": true, "scheduled": true}

// ContentHash fingerprints what an approval covers in the content: subject,
// preheader, MJML, HTML and A/B variants.
func ContentHash(subject, preheader, mjmlSrc, html string, ab json.RawMessage) string {
	h := sha256.New()
	for _, part := range []string{subject, preheader, mjmlSrc, html, string(ab)} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Detail loads a campaign for the composer.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (ComposerCampaign, error) {
	return s.DetailIn(ctx, ws, p, id, "")
}

// DetailIn loads a campaign for the composer with the content of one
// language version ("" or the source language: the source).
func (s *Service) DetailIn(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string) (ComposerCampaign, error) {
	r, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerCampaign{}, api.MapError(err, "Campaign")
	}
	d := ComposerCampaign{
		ID: r.ID.String(), Name: r.Name, Status: r.Status, Subject: r.Subject, Preheader: r.Preheader, ReplyTo: r.ReplyTo,
		Audience: audienceFromDB(r.Audience), AudienceCount: r.AudienceCount, EditorMode: r.EditorMode, MJML: r.Mjml, HTML: r.Html, PlainText: r.PlainText,
		Version: int(r.Version), VersionCount: int(r.VersionCount), ContentHash: r.ContentHash, ABTest: abFromDB(r.AbTest), ScheduledAt: r.ScheduledAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedByName, AutosavedAt: r.AutosavedAt,
	}
	if r.DigestID != nil {
		d.DigestID, d.DigestRole = r.DigestID.String(), r.DigestRole
	}
	_ = json.Unmarshal(r.ContentSettings, &d.Settings)
	if r.SenderID != nil {
		d.Sender = &ComposerSenderRef{ID: r.SenderID.String(), Name: r.SenderName, Email: r.SenderEmail}
	}
	labels, err := s.audienceLabeler(ctx, ws.ID)
	if err != nil {
		return ComposerCampaign{}, err
	}
	d.AudienceLabel = labels(r.Audience)
	if d.Route, err = s.route(ctx, ws.ID, r.ID); err != nil {
		return ComposerCampaign{}, err
	}
	if run, err := s.Q.ComposerLatestReadiness(ctx, r.ID); err == nil {
		d.Readiness = &ReadinessSummary{Verdict: run.Verdict, Issues: run.Issues}
	}
	d.Access = s.access(ctx, ws, p, r.Status, r.StartedAt)
	if d.Approval, err = s.openApproval(ctx, s.Q, id); err != nil {
		return ComposerCampaign{}, err
	}
	if src := s.sourceLanguage(ctx, ws, id); src != "" {
		d.SourceLanguage, d.Language = src, src
		if lang != "" && lang != src {
			t, err := s.Translations.Content(ctx, ws, id, lang)
			if err != nil {
				return ComposerCampaign{}, err
			}
			d.Language, d.Subject, d.Preheader, d.MJML, d.HTML, d.PlainText = t.Language, t.Subject, t.Preheader, t.MJML, t.HTML, t.PlainText
			d.Version, d.VersionCount, d.ContentHash, d.ABTest = t.Version, t.VersionCount, t.ContentHash, abFromDB(t.ABTest)
			d.UpdatedAt, d.UpdatedBy, d.AutosavedAt = t.UpdatedAt, t.UpdatedBy, t.AutosavedAt
		}
	}
	return d, nil
}

func (s *Service) access(ctx context.Context, ws *store.Workspace, p *authz.Principal, status string, started *time.Time) ComposerAccess {
	a := ComposerAccess{
		Editable: editableStatuses[status], CanEdit: p.Can(authz.CampaignsEdit), CanSend: p.Can(authz.CampaignsSend),
		CanManageRouting: p.Can(authz.ProvidersManage), CanSendTestToOthers: p.Can(authz.CampaignsSend),
		CanEditTemplates: p.Can(authz.TemplatesEdit), CanViewContacts: p.Can(authz.SubscribersView), SendHelpers: []string{},
	}
	if !a.CanEdit {
		a.EditHelpers = s.helpers(ctx, ws.ID, authz.CampaignsEdit)
	}
	if !a.Editable {
		l := i18n.LangFrom(ctx)
		switch {
		case status == "sending" || status == "paused":
			a.ReadOnlyReason = mProblemReadOnlySending.Text(l, nil)
		case started != nil:
			a.ReadOnlyReason = mProblemReadOnlySentOn.Text(l, i18n.Args{"date": l.Region.ShortDate(*started)})
		default:
			a.ReadOnlyReason = mProblemReadOnlySent.Text(l, nil)
		}
	}
	if !a.CanSend {
		a.SendHelpers = s.helpers(ctx, ws.ID, authz.CampaignsSend)
	}
	return a
}

// helpers names up to two active members holding perm (admins first).
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

// route returns the campaign's planned split, or the Broadcasts default rule.
func (s *Service) route(ctx context.Context, wsID, id uuid.UUID) (ComposerRoute, error) {
	shares, err := s.routes(ctx, wsID, []uuid.UUID{id})
	if err != nil {
		return ComposerRoute{}, err
	}
	if len(shares[id]) > 0 {
		return ComposerRoute{Source: "campaign", Shares: shares[id]}, nil
	}
	return s.broadcastRoute(ctx, wsID)
}

func (s *Service) broadcastRoute(ctx context.Context, wsID uuid.UUID) (ComposerRoute, error) {
	raw, err := s.Q.BroadcastDefaultSplit(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ComposerRoute{Source: "none", Shares: []RouteShare{}}, nil
	}
	if err != nil {
		return ComposerRoute{}, err
	}
	var splits []struct {
		ProviderID uuid.UUID `json:"provider_id"`
		Weight     float64   `json:"weight"`
	}
	_ = json.Unmarshal(raw, &splits)
	provs, err := s.Q.ProviderNames(ctx, wsID)
	if err != nil {
		return ComposerRoute{}, err
	}
	byID := map[uuid.UUID]store.ProviderNamesRow{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	total := 0.0
	for _, sp := range splits {
		total += sp.Weight
	}
	out := ComposerRoute{Source: "rule", Shares: []RouteShare{}}
	for _, sp := range splits {
		p, ok := byID[sp.ProviderID]
		if !ok || total == 0 {
			continue
		}
		out.Shares = append(out.Shares, RouteShare{ProviderID: p.ID.String(), Provider: p.Name, Short: ShortName(p.Name, p.Adapter), Adapter: p.Adapter, Percent: math.Round(sp.Weight/total*1000) / 10})
	}
	return out, nil
}

// ComposerCreateCampaign starts a draft, blank or from a template or another campaign.
type ComposerCreateCampaign struct {
	Name           string `json:"name,omitempty" maxLength:"200"`
	TemplateID     string `json:"templateId,omitempty" maxLength:"64" doc:"Start from a template (starter:<key> or a saved template id)"`
	FromCampaignID string `json:"fromCampaignId,omitempty" format:"uuid" doc:"Start from a copy of another campaign's content"`
}

// Create makes a draft and returns it.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in ComposerCreateCampaign) (ComposerCampaign, error) {
	l := i18n.LangFrom(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		// A new draft's default name is written in its author's language.
		name = mLabelUntitled.Text(l, nil)
	}
	var subject, preheader, src, html, label string
	var ab json.RawMessage
	var audience json.RawMessage = []byte(`{}`)
	var count *int64
	sender := s.defaultSender(ctx, ws)
	source := "create"
	switch {
	case in.TemplateID != "":
		if s.Templates == nil {
			return ComposerCampaign{}, api.Invalid("body.templateId", mProblemNoTemplates.Text(l, nil))
		}
		t, err := s.Templates.Get(ctx, ws, in.TemplateID, templates.WorkspaceAddress(ws))
		if err != nil {
			return ComposerCampaign{}, err
		}
		if t.Kind != "template" {
			return ComposerCampaign{}, api.Invalid("body.templateId", mProblemSavedSection.Text(l, nil))
		}
		src, html = t.MJML, t.HTML
		label = mVersionFromTemplate.Text(i18n.Default, i18n.Args{"name": t.Name})
		source = "template"
	case in.FromCampaignID != "":
		from, err := uuid.Parse(in.FromCampaignID)
		if err != nil {
			return ComposerCampaign{}, api.NotFound("Campaign")
		}
		c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: from})
		if err != nil {
			return ComposerCampaign{}, api.MapError(err, "Campaign")
		}
		subject, preheader, src, html, ab = c.Subject, c.Preheader, c.Mjml, c.Html, c.AbTest
		if abt := abFromDB(ab); abt != nil {
			abt.Winner = ""
			ab = abt.toDB()
		}
		// A copy of a retry is a campaign of its own, to the whole audience.
		audience, count = retryaudience.Without(c.Audience), c.AudienceCount
		if _, retry := retryaudience.Of(c.Audience); retry {
			count = nil
		}
		// A scoped member's copy keeps the audience only if they could pick it.
		if len(p.Scope.Lists) > 0 && s.checkAudience(ctx, s.Q, ws, p, audienceFromDB(c.Audience)) != nil {
			audience, count = []byte(`{}`), nil
		}
		if c.SenderID != nil {
			sender = c.SenderID
		}
		if in.Name == "" {
			name = mLabelCopyOf.Text(l, i18n.Args{"name": c.Name})
		}
		label = mVersionCopiedFrom.Text(i18n.Default, i18n.Args{"name": c.Name})
		source = "duplicate"
	}
	if src != "" {
		// A template's or another campaign's HTML is stored only when it shows
		// and does what its MJML says (CO-3).
		checked, _, errs := mjml.Checked(src, html)
		if errs != nil {
			checked = ""
		}
		html = checked
	}
	html = accessibility.FinalizeStored(ctx, s.Q, ws.ID, accessibility.OwnerCampaign, uuid.Nil, 0, html)
	id, err := s.insertDraft(ctx, ws, p, draftRow{name: name, subject: subject, preheader: preheader, mjml: src, html: html, label: label, source: source,
		ab: ab, audience: audience, count: count, sender: sender},
		&audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetName: name,
			After: map[string]any{"name": name, "templateId": in.TemplateID, "fromCampaignId": in.FromCampaignID}})
	if err != nil {
		return ComposerCampaign{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// defaultSender is the sender a new draft starts with: the workspace's
// default, or its first sender.
func (s *Service) defaultSender(ctx context.Context, ws *store.Workspace) *uuid.UUID {
	if def, err := s.Q.ComposerDefaultSender(ctx, ws.ID); err == nil && def.SenderID != "" {
		if u, err := uuid.Parse(def.SenderID); err == nil {
			return &u
		}
	}
	if senders, err := s.Q.ListComposerSenders(ctx, ws.ID); err == nil && len(senders) > 0 {
		return &senders[0].ID
	}
	return nil
}

// draftRow is a new draft as it's first stored: its content, audience and
// sender, and its first version's label and source.
type draftRow struct {
	name, subject, preheader, mjml, html, label, source string
	ab, audience                                        json.RawMessage
	count                                               *int64
	sender                                              *uuid.UUID
}

// insertDraft stores a new draft and its first version. event, when set, is
// the creation's audit entry (its target is filled in); without it the
// caller writes one.
func (s *Service) insertDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, d draftRow, event *audit.Event) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	actor := userRef(p)
	var autosaved *time.Time
	if d.mjml != "" {
		autosaved = &now
	}
	plain := contentcheck.PlainText(d.html)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InsertComposerCampaign(ctx, store.InsertComposerCampaignParams{
			ID: id, WorkspaceID: ws.ID, Name: d.name, Subject: d.subject, Preheader: d.preheader, SenderID: d.sender, Audience: d.audience, AudienceCount: d.count,
			Mjml: d.mjml, Html: d.html, PlainText: plain, AbTest: d.ab, ContentHash: ContentHash(d.subject, d.preheader, d.mjml, d.html, d.ab),
			CreatedBy: actor, Now: now, AutosavedAt: autosaved,
		}); err != nil {
			return err
		}
		if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: 1, Subject: d.subject, Preheader: d.preheader, Mjml: d.mjml, Html: d.html,
			PlainText: plain, AbTest: d.ab, SavedBy: actor, SavedAt: now, Label: d.label, Source: d.source, SizeBytes: int32(len(d.html))}); err != nil {
			return err
		}
		if event == nil {
			return nil
		}
		e := *event
		e.TargetID = id.String()
		return s.Audit.LogWith(ctx, q, e)
	})
	return id, err
}

// userRef is the person behind an action, for the columns that name one
// (nil for a key nobody created any more).
func userRef(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// ComposerUpdateCampaign changes what isn't versioned content: name, sender,
// reply-to, audience and content settings. Omitted fields stay.
type ComposerUpdateCampaign struct {
	Name     *string                  `json:"name,omitempty" minLength:"1" maxLength:"200"`
	SenderID *string                  `json:"senderId,omitempty" doc:"A sender id, or empty to clear"`
	ReplyTo  *string                  `json:"replyTo,omitempty" maxLength:"254"`
	Audience *ComposerAudienceInput   `json:"audience,omitempty"`
	Settings *ComposerContentSettings `json:"settings,omitempty"`
}

// Update applies an UpdateCampaign.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerUpdateCampaign) (ComposerCampaign, error) {
	l := i18n.LangFrom(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Campaign")
		}
		if !editableStatuses[cur.Status] {
			return api.Conflict(s.access(ctx, ws, p, cur.Status, cur.StartedAt).ReadOnlyReason)
		}
		before, after := map[string]any{}, map[string]any{}
		name, sender, replyTo, audience, count, settings := cur.Name, cur.SenderID, cur.ReplyTo, cur.Audience, cur.AudienceCount, cur.ContentSettings
		if in.Name != nil {
			n := strings.TrimSpace(*in.Name)
			if n == "" {
				return api.Invalid("body.name", mProblemName.Text(l, nil))
			}
			if n != name {
				before["name"], after["name"] = name, n
				name = n
			}
		}
		if in.SenderID != nil {
			var next *uuid.UUID
			if *in.SenderID != "" {
				sid, err := uuid.Parse(*in.SenderID)
				if err != nil {
					return api.Invalid("body.senderId", mProblemPickSender.Text(l, nil))
				}
				snd, err := q.GetComposerSender(ctx, store.GetComposerSenderParams{WorkspaceID: ws.ID, ID: sid})
				if err != nil {
					return api.Invalid("body.senderId", mProblemUnknownSender.Text(l, nil))
				}
				if !p.Scope.AllowsSenderDomain(snd.Domain) {
					return api.Invalid("body.senderId", mProblemSenderScope.Text(l, nil))
				}
				next = &sid
			}
			if !sameID(next, sender) {
				before["senderId"], after["senderId"] = idString(sender), idString(next)
				sender = next
			}
		}
		if in.ReplyTo != nil && strings.TrimSpace(*in.ReplyTo) != replyTo {
			before["replyTo"], after["replyTo"] = replyTo, strings.TrimSpace(*in.ReplyTo)
			replyTo = strings.TrimSpace(*in.ReplyTo)
		}
		if in.Audience != nil {
			def := normalizeAudience(in.Audience.def())
			if err := s.checkAudience(ctx, q, ws, p, def); err != nil {
				return err
			}
			nextRaw := def.toDB(audience)
			if string(nextRaw) != string(audience) {
				n, err := s.countAudience(ctx, q, ws.ID, def)
				if err != nil {
					return err
				}
				before["audience"], after["audience"] = audienceFromDB(audience), def
				before["audienceCount"], after["audienceCount"] = count, n.Count
				audience, count = nextRaw, &n.Count
			}
		}
		if in.Settings != nil {
			b, _ := json.Marshal(in.Settings)
			if string(b) != string(settings) {
				before["settings"], after["settings"] = settings, in.Settings
				settings = b
			}
		}
		if len(after) == 0 {
			return nil
		}
		if err := q.ComposerUpdateCampaignMeta(ctx, store.ComposerUpdateCampaignMetaParams{WorkspaceID: ws.ID, ID: id, Name: name, SenderID: sender, ReplyTo: replyTo,
			Audience: audience, AudienceCount: count, ContentSettings: settings, UpdatedBy: &p.UserID, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: id.String(), TargetName: name, Before: before, After: after})
	})
	if err != nil {
		return ComposerCampaign{}, err
	}
	s.changed(ctx, ws, id)
	return s.Detail(ctx, ws, p, id)
}

func normalizeAudience(a ComposerAudience) ComposerAudience {
	clean := func(ids []string) []string {
		out := []string{}
		seen := map[string]bool{}
		for _, id := range ids {
			if u, err := uuid.Parse(strings.TrimSpace(id)); err == nil && !seen[u.String()] {
				seen[u.String()] = true
				out = append(out, u.String())
			}
		}
		return out
	}
	out := ComposerAudience{Lists: clean(a.Lists), Segments: clean(a.Segments), ExcludeLists: clean(a.ExcludeLists), ExcludeSegments: clean(a.ExcludeSegments)}
	switch a.Paid {
	case paidaccess.Paid:
		out.Paid = a.Paid
		if u, err := uuid.Parse(strings.TrimSpace(a.PaidTier)); err == nil {
			out.PaidTier = u.String()
		}
	case paidaccess.Free:
		out.Paid = a.Paid
	}
	return out
}

// checkAudience verifies the lists and segments exist and that a scoped
// member only targets lists in their scope.
func (s *Service) checkAudience(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, a ComposerAudience) error {
	l := i18n.LangFrom(ctx)
	lists, err := q.ComposerListIDs(ctx, ws.ID)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, id := range lists {
		known[id.String()] = true
	}
	for _, id := range append(append([]string{}, a.Lists...), a.ExcludeLists...) {
		if !known[id] {
			return api.Invalid("body.audience.lists", mProblemListGone.Text(l, nil))
		}
		u, _ := uuid.Parse(id)
		if !p.Scope.AllowsList(u) {
			return api.Invalid("body.audience.lists", mProblemListScope.Text(l, nil))
		}
	}
	if len(a.Segments)+len(a.ExcludeSegments) > 0 {
		segs, err := q.ComposerAudienceSegments(ctx, ws.ID)
		if err != nil {
			return err
		}
		knownSeg := map[string]bool{}
		for _, sg := range segs {
			knownSeg[sg.ID.String()] = true
		}
		for _, id := range append(append([]string{}, a.Segments...), a.ExcludeSegments...) {
			if !knownSeg[id] {
				return api.Invalid("body.audience.segments", mProblemSegmentGone.Text(l, nil))
			}
		}
		// Excluding a segment only narrows the audience.
		if !access.AudienceInScope(p.Scope, nil, len(a.Segments)) {
			return api.Invalid("body.audience.segments", access.AudienceScopeProblem(l))
		}
	}
	if a.PaidTier != "" {
		tiers, err := paidaccess.Tiers(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		found := false
		for _, t := range tiers {
			found = found || t.ID.String() == a.PaidTier
		}
		if !found {
			return api.Invalid("body.audience.paidTier", mProblemTierGone.Text(l, nil))
		}
	}
	return nil
}

func sameID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func idString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
