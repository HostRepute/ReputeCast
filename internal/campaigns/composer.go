package campaigns

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// ComposerAudience says who receives a campaign: lists and segments, minus
// excluded lists and segments.
type ComposerAudience struct {
	Lists           []string `json:"lists"`
	Segments        []string `json:"segments"`
	ExcludeLists    []string `json:"excludeLists"`
	ExcludeSegments []string `json:"excludeSegments"`
}

// ComposerAudienceInput is an audience in requests (every part optional).
type ComposerAudienceInput struct {
	Lists           []string `json:"lists,omitempty" maxItems:"100"`
	Segments        []string `json:"segments,omitempty" maxItems:"100"`
	ExcludeLists    []string `json:"excludeLists,omitempty" maxItems:"100"`
	ExcludeSegments []string `json:"excludeSegments,omitempty" maxItems:"100"`
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
}

func audienceFromDB(raw json.RawMessage) ComposerAudience {
	var a dbAudience
	_ = json.Unmarshal(raw, &a)
	return ComposerAudience{Lists: nonNil(a.Lists), Segments: nonNil(a.Segments), ExcludeLists: nonNil(a.ExcludeLists), ExcludeSegments: nonNil(a.ExcludeSegments)}
}

func (a ComposerAudience) toDB(prev json.RawMessage) json.RawMessage {
	var d dbAudience
	_ = json.Unmarshal(prev, &d) // keeps exclude_tiers set elsewhere
	d.Lists, d.Segments, d.ExcludeLists, d.ExcludeSegments = a.Lists, a.Segments, a.ExcludeLists, a.ExcludeSegments
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
	out := &ComposerABTest{Kind: d.Kind, TestPercent: d.TestPercent, Metric: d.Metric, DurationHours: d.DurationHours, BestTime: d.BestTime, Winner: d.Winner, Variants: []ComposerABVariant{}}
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
	d := dbABTest{Kind: t.Kind, TestPercent: t.TestPercent, Metric: t.Metric, DurationHours: t.DurationHours, BestTime: t.BestTime, Winner: t.Winner}
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
	return d, nil
}

func (s *Service) access(ctx context.Context, ws *store.Workspace, p *authz.Principal, status string, started *time.Time) ComposerAccess {
	a := ComposerAccess{
		Editable: editableStatuses[status], CanEdit: p.Can(authz.CampaignsEdit), CanSend: p.Can(authz.CampaignsSend),
		CanManageRouting: p.Can(authz.ProvidersManage), CanSendTestToOthers: p.Can(authz.CampaignsSend),
		CanEditTemplates: p.Can(authz.TemplatesEdit), CanViewContacts: p.Can(authz.SubscribersView), SendHelpers: []string{},
	}
	if !a.Editable {
		switch status {
		case "sending", "paused":
			a.ReadOnlyReason = "This campaign is sending. Its content can't change mid-send; duplicate it to reuse the content."
		default:
			when := ""
			if started != nil {
				when = " on " + started.Format("Mon 2 Jan")
			}
			a.ReadOnlyReason = "This campaign was sent" + when + ". Duplicate it to reuse the content."
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
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Untitled campaign"
	}
	var subject, preheader, src, html, label string
	var ab json.RawMessage
	var audience json.RawMessage = []byte(`{}`)
	var count *int64
	var sender *uuid.UUID
	source := "create"
	if def, err := s.Q.ComposerDefaultSender(ctx, ws.ID); err == nil && def.SenderID != "" {
		if u, err := uuid.Parse(def.SenderID); err == nil {
			sender = &u
		}
	}
	if sender == nil {
		if senders, err := s.Q.ListComposerSenders(ctx, ws.ID); err == nil && len(senders) > 0 {
			sender = &senders[0].ID
		}
	}
	switch {
	case in.TemplateID != "":
		if s.Templates == nil {
			return ComposerCampaign{}, api.Invalid("body.templateId", "Templates aren't available.")
		}
		t, err := s.Templates.Get(ctx, ws, in.TemplateID, templates.WorkspaceAddress(ws))
		if err != nil {
			return ComposerCampaign{}, err
		}
		if t.Kind != "template" {
			return ComposerCampaign{}, api.Invalid("body.templateId", "That is a saved section; start from a template.")
		}
		src, html = t.MJML, t.HTML
		label = fmt.Sprintf("created from template “%s”", t.Name)
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
		audience, count = c.Audience, c.AudienceCount
		if c.SenderID != nil {
			sender = c.SenderID
		}
		if in.Name == "" {
			name = "Copy of " + c.Name
		}
		label = fmt.Sprintf("copied from “%s”", c.Name)
		source = "duplicate"
	}
	if src != "" && html == "" {
		html = mjml.Compile(src).HTML
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	actor := p.UserID
	var autosaved *time.Time
	if src != "" {
		autosaved = &now
	}
	plain := contentcheck.PlainText(html)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InsertComposerCampaign(ctx, store.InsertComposerCampaignParams{
			ID: id, WorkspaceID: ws.ID, Name: name, Subject: subject, Preheader: preheader, SenderID: sender, Audience: audience, AudienceCount: count,
			Mjml: src, Html: html, PlainText: plain, AbTest: ab, ContentHash: ContentHash(subject, preheader, src, html, ab),
			CreatedBy: &actor, Now: now, AutosavedAt: autosaved,
		}); err != nil {
			return err
		}
		if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: 1, Subject: subject, Preheader: preheader, Mjml: src, Html: html,
			PlainText: plain, AbTest: ab, SavedBy: &actor, SavedAt: now, Label: label, Source: source, SizeBytes: int32(len(html))}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "templateId": in.TemplateID, "fromCampaignId": in.FromCampaignID}})
	})
	if err != nil {
		return ComposerCampaign{}, err
	}
	return s.Detail(ctx, ws, p, id)
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
				return api.Invalid("body.name", "Give the campaign a name.")
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
					return api.Invalid("body.senderId", "Pick a sender from the list.")
				}
				snd, err := q.GetComposerSender(ctx, store.GetComposerSenderParams{WorkspaceID: ws.ID, ID: sid})
				if err != nil {
					return api.Invalid("body.senderId", "That sender doesn't exist in this workspace.")
				}
				if !p.Scope.AllowsSenderDomain(snd.Domain) {
					return api.Invalid("body.senderId", "Your access is limited to other sender domains.")
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
	return ComposerAudience{Lists: clean(a.Lists), Segments: clean(a.Segments), ExcludeLists: clean(a.ExcludeLists), ExcludeSegments: clean(a.ExcludeSegments)}
}

// checkAudience verifies the lists and segments exist and that a scoped
// member only targets lists in their scope.
func (s *Service) checkAudience(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, a ComposerAudience) error {
	lists, err := q.ComposerAudienceLists(ctx, ws.ID)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, l := range lists {
		known[l.ID.String()] = true
	}
	for _, id := range append(append([]string{}, a.Lists...), a.ExcludeLists...) {
		if !known[id] {
			return api.Invalid("body.audience.lists", "One of the lists no longer exists. Reload and pick again.")
		}
		u, _ := uuid.Parse(id)
		if !p.Scope.AllowsList(u) {
			return api.Invalid("body.audience.lists", "Your access is limited to other lists.")
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
				return api.Invalid("body.audience.segments", "One of the segments no longer exists. Reload and pick again.")
			}
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
