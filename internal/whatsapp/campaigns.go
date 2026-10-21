package whatsapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// WhatsApp campaigns (brief §3.9, board WhatsAppChannel › A WhatsApp
// campaign): an approved template, its variables mapped to contact fields
// with fallbacks, a chat preview as a real contact, the audience with
// WhatsApp consent, the schedule and quiet hours, and the cost.

// WhatsAppAudience is a campaign's people.
type WhatsAppAudience struct {
	Lists []WhatsAppListConsent `json:"lists"`
	// Consent is everyone with WhatsApp consent on the lists, once each.
	Consent int64 `json:"consent"`
	// Without are the lists' people without it: they get nothing.
	Without int64 `json:"without"`
	// Suppressed opted out on WhatsApp; Final is who it goes to.
	Suppressed int64 `json:"suppressed"`
	Final      int64 `json:"final"`
	// Countries are the recipients by country, for the cost.
	Countries map[string]int64 `json:"countries"`
}

// WhatsAppSchedule is when a campaign goes.
type WhatsAppSchedule struct {
	// Mode: now; at one moment; local, a time in each recipient's zone.
	Mode   string     `json:"mode" enum:"now,at,local"`
	SendAt *time.Time `json:"sendAt,omitempty"`
	// LocalDate (YYYY-MM-DD) and LocalMinute (540 = 09:00) for local.
	LocalDate   string `json:"localDate,omitempty"`
	LocalMinute *int   `json:"localMinute,omitempty"`
	Label       string `json:"label"`
}

// WhatsAppPreview is the chat preview as one contact.
type WhatsAppPreview struct {
	ContactID string                   `json:"contactId,omitempty"`
	Contact   string                   `json:"contact"`
	FirstName string                   `json:"firstName,omitempty"`
	Text      string                   `json:"text"`
	Buttons   []WhatsAppTemplateButton `json:"buttons"`
	// Fallbacks are the variables that used their fallback for this
	// contact, and what someone without the field reads.
	Fallbacks []WhatsAppFallback `json:"fallbacks"`
}

// WhatsAppFallback is a variable's fallback as the preview explains it.
type WhatsAppFallback struct {
	Key      string `json:"key"`
	Field    string `json:"field"`
	Fallback string `json:"fallback"`
}

// WhatsAppField is a contact field a variable can take.
type WhatsAppField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// WhatsAppAccess is what the viewer may do with a campaign.
type WhatsAppAccess struct {
	CanEdit bool     `json:"canEdit"`
	CanSend bool     `json:"canSend"`
	Missing []string `json:"missing"`
	Helpers []string `json:"helpers"`
}

// WhatsAppCampaign is the composer.
type WhatsAppCampaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	Version     int        `json:"version"`
	AutosavedAt *time.Time `json:"autosavedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	UpdatedBy   string     `json:"updatedBy"`
	CreatedBy   string     `json:"createdBy"`

	TemplateID string            `json:"templateId,omitempty"`
	Template   *WhatsAppTemplate `json:"template,omitempty"`
	// Templates are every template: the picker offers the approved ones.
	Templates   []WhatsAppTemplate    `json:"templates"`
	Variables   []WhatsAppVariable    `json:"variables"`
	ListOptions []WhatsAppListConsent `json:"listOptions"`
	Segments    []string              `json:"segments"`
	Audience    WhatsAppAudience      `json:"audience"`
	Schedule    WhatsAppSchedule      `json:"schedule"`
	OverLimit   string                `json:"overLimit" enum:"next_day,skip"`
	Quiet       bool                  `json:"quiet"`
	Window      string                `json:"window"`
	Preview     *WhatsAppPreview      `json:"preview,omitempty"`
	Cost        WhatsAppCost          `json:"cost"`
	Business    *WhatsAppBusiness     `json:"business,omitempty"`
	Fields      []WhatsAppField       `json:"fields"`
	Access      WhatsAppAccess        `json:"access"`
	Connected   bool                  `json:"connected"`
	// AccountStatus is waiting, connected or token_rejected.
	AccountStatus string              `json:"accountStatus"`
	Test          *WhatsAppTestResult `json:"test,omitempty"`
}

// WhatsAppCampaignCreate makes a campaign.
type WhatsAppCampaignCreate struct {
	Name       string   `json:"name" minLength:"1" maxLength:"200"`
	TemplateID string   `json:"templateId,omitempty" maxLength:"40"`
	Lists      []string `json:"lists,omitempty" maxItems:"50"`
}

// WhatsAppCampaignUpdate is an autosave from the composer.
type WhatsAppCampaignUpdate struct {
	Name        string             `json:"name" minLength:"1" maxLength:"200"`
	TemplateID  string             `json:"templateId" maxLength:"40"`
	Variables   []WhatsAppVariable `json:"variables" maxItems:"20"`
	Lists       []string           `json:"lists" maxItems:"50"`
	Segments    []string           `json:"segments,omitempty" maxItems:"50"`
	Mode        string             `json:"mode" enum:"now,at,local"`
	SendAt      *time.Time         `json:"sendAt,omitempty"`
	LocalDate   string             `json:"localDate,omitempty" pattern:"^(\\d{4}-\\d{2}-\\d{2})?$"`
	LocalMinute *int               `json:"localMinute,omitempty" minimum:"0" maximum:"1439"`
	OverLimit   string             `json:"overLimit,omitempty" enum:"next_day,skip,"`
	// Version is the version the composer edited; a save from an older one
	// is refused.
	Version int `json:"version" minimum:"0"`
}

func audienceJSON(lists, segments []uuid.UUID) json.RawMessage {
	str := func(ids []uuid.UUID) []string {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, id.String())
		}
		return out
	}
	b, _ := json.Marshal(map[string]any{"lists": str(lists), "segments": str(segments)})
	return b
}

// audienceOf reads a campaign's lists and segments.
func audienceOf(raw json.RawMessage) (lists, segments []uuid.UUID) {
	var a struct {
		Lists    []string `json:"lists"`
		Segments []string `json:"segments"`
	}
	_ = json.Unmarshal(raw, &a)
	for _, v := range a.Lists {
		if id, err := uuid.Parse(v); err == nil {
			lists = append(lists, id)
		}
	}
	for _, v := range a.Segments {
		if id, err := uuid.Parse(v); err == nil {
			segments = append(segments, id)
		}
	}
	return lists, segments
}

func parseIDs(raw []string, field, what string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(raw))
	for _, r := range raw {
		id, err := uuid.Parse(strings.TrimSpace(r))
		if err != nil {
			return nil, api.Invalid(field, "Pick "+what+" of this workspace.")
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// contentHash is what approvals compare: the template and what fills it.
func contentHash(templateID *uuid.UUID, variables json.RawMessage) string {
	h := sha256.New()
	fmt.Fprintf(h, "%v\x00%s", templateID, variables)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// audienceSet is a campaign's people after consent and suppressions.
type audienceSet struct {
	list       []recipient
	consent    int64
	without    int64
	suppressed int64
	countries  map[string]int64
}

// audience is everyone with WhatsApp consent on the lists (narrowed by
// segments), with a phone, not suppressed on WhatsApp, once each.
func (s *Service) audience(ctx context.Context, ws *store.Workspace, lists, segments []uuid.UUID) (audienceSet, error) {
	out := audienceSet{countries: map[string]int64{}}
	if len(lists) == 0 {
		return out, nil
	}
	narrow := len(segments) > 0
	var only []uuid.UUID
	if narrow && s.Segments != nil {
		seen := map[uuid.UUID]bool{}
		for _, seg := range segments {
			ids, err := s.Segments.ContactIDs(ctx, ws.ID, seg)
			if err != nil {
				return out, err
			}
			for _, id := range ids {
				if !seen[id] {
					seen[id] = true
					only = append(only, id)
				}
			}
		}
	}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (c.id) c.id, cc.list_id, c.phone, c.phone_country, c.timezone, c.first_name, c.last_name, c.email,
			c.attributes, (sup.id IS NOT NULL)
		FROM channel_consents cc
		JOIN contacts c ON c.id = cc.contact_id AND c.erased_at IS NULL AND c.phone <> ''
		LEFT JOIN channel_suppressions sup ON sup.workspace_id = c.workspace_id AND sup.channel = 'whatsapp' AND sup.phone = c.phone AND sup.removed_at IS NULL
		WHERE cc.workspace_id = $1 AND cc.channel = 'whatsapp' AND cc.status = 'subscribed' AND cc.list_id = ANY($2)
		  AND (NOT $3::boolean OR c.id = ANY($4::uuid[]))
		ORDER BY c.id, cc.changed_at`, ws.ID, lists, narrow, only)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r recipient
		var attrs []byte
		var suppressed bool
		if err := rows.Scan(&r.ContactID, &r.ListID, &r.Phone, &r.Country, &r.Zone, &r.FirstName, &r.LastName, &r.Email, &attrs, &suppressed); err != nil {
			return out, err
		}
		out.consent++
		if suppressed {
			out.suppressed++
			continue
		}
		_ = json.Unmarshal(attrs, &r.Attributes)
		zone, src := phone.RecipientZone(r.Zone, r.Phone)
		r.Zone, r.ZoneSource = zone, string(src)
		if r.Country == "" {
			if n, err := phone.Parse(r.Phone, ""); err == nil {
				r.Country = n.Country
			}
		}
		out.countries[r.Country]++
		out.list = append(out.list, r)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	rows.Close()
	err = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT lm.contact_id) FROM list_memberships lm JOIN contacts c ON c.id = lm.contact_id AND c.erased_at IS NULL
		WHERE lm.list_id = ANY($1) AND lm.status = 'subscribed'
		  AND NOT EXISTS (SELECT 1 FROM channel_consents cc WHERE cc.contact_id = lm.contact_id AND cc.list_id = ANY($1) AND cc.channel = 'whatsapp'
		      AND cc.status = 'subscribed')`, lists).Scan(&out.without)
	return out, err
}

// campaign loads a WhatsApp campaign of the workspace.
func (s *Service) campaign(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.WhatsAppCampaignRow, error) {
	c, err := s.Q.WhatsAppCampaign(ctx, store.WhatsAppCampaignParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return c, api.MapError(err, "WhatsApp campaign")
	}
	return c, nil
}

// approvedTemplate checks a template id for a campaign or a step.
func (s *Service) approvedTemplate(ctx context.Context, ws *store.Workspace, raw, field string) (*store.WhatsappTemplate, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, api.Invalid(field, "Pick one of the account's approved templates.")
	}
	t, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: id, WorkspaceID: ws.ID})
	if err != nil || t.RemovedAt != nil {
		return nil, api.Invalid(field, "Pick one of the account's approved templates.")
	}
	if t.Status != "approved" {
		return nil, api.Invalid(field, "Only approved templates can be picked: "+t.Name+" is "+t.Status+".")
	}
	return &t, nil
}

// CreateCampaign makes a WhatsApp campaign, audited like any campaign.
func (s *Service) CreateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WhatsAppCampaignCreate) (WhatsAppCampaign, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return WhatsAppCampaign{}, api.Invalid("body.name", "Give the campaign a name.")
	}
	tpl, err := s.approvedTemplate(ctx, ws, in.TemplateID, "body.templateId")
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	lists, err := parseIDs(in.Lists, "body.lists", "lists")
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	if err := s.checkLists(ctx, ws, lists); err != nil {
		return WhatsAppCampaign{}, err
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	var tid *uuid.UUID
	tname, tlang := "", ""
	vars := []WhatsAppVariable{}
	if tpl != nil {
		tid, tname, tlang = &tpl.ID, tpl.Name, tpl.Language
		for _, k := range variablesOf(tpl.Body) {
			vars = append(vars, WhatsAppVariable{Key: k, Source: "field"})
		}
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.WhatsAppCampaignInsert(ctx, store.WhatsAppCampaignInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Audience: audienceJSON(lists, nil),
			ContentHash: contentHash(tid, jsonOf(vars)), By: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		if err := q.WhatsAppCampaignDetailInsert(ctx, store.WhatsAppCampaignDetailInsertParams{CampaignID: id, WorkspaceID: ws.ID, TemplateID: tid, TemplateName: tname,
			TemplateLanguage: tlang, Variables: jsonOf(vars), ScheduleMode: "now", OverLimit: "next_day"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "channel": "whatsapp", "template": tname}})
	})
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	return s.Campaign(ctx, ws, p, id, uuid.Nil)
}

// checkLists refuses lists of other workspaces.
func (s *Service) checkLists(ctx context.Context, ws *store.Workspace, lists []uuid.UUID) error {
	if len(lists) == 0 {
		return nil
	}
	var n int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL AND id = ANY($2)`, ws.ID, lists).Scan(&n); err != nil {
		return err
	}
	if n != len(lists) {
		return api.Invalid("body.lists", "Pick lists of this workspace.")
	}
	return nil
}

// fields are the contact fields a variable can take.
func (s *Service) fields(ctx context.Context, ws *store.Workspace) []WhatsAppField {
	out := []WhatsAppField{{"first_name", "First name"}, {"last_name", "Last name"}, {"full_name", "Full name"}, {"email", "Email"}, {"phone", "Phone"}}
	rows, err := s.Pool.Query(ctx, `SELECT key, label FROM custom_fields WHERE workspace_id = $1 ORDER BY label`, ws.ID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var f WhatsAppField
		if rows.Scan(&f.Key, &f.Label) == nil {
			out = append(out, f)
		}
	}
	return out
}

// Campaign is the composer's view; preview names the contact the preview
// shows (uuid.Nil picks one of the audience).
func (s *Service) Campaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id, preview uuid.UUID) (WhatsAppCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	out := WhatsAppCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, Version: int(c.Version), AutosavedAt: c.AutosavedAt, UpdatedAt: c.UpdatedAt,
		UpdatedBy: c.UpdatedByName, CreatedBy: c.CreatedByName, Templates: []WhatsAppTemplate{}, Variables: []WhatsAppVariable{}, Segments: []string{},
		OverLimit: c.OverLimit, Fields: s.fields(ctx, ws), Access: s.accessFor(ctx, ws, p)}
	_ = json.Unmarshal(c.Variables, &out.Variables)
	if out.Variables == nil {
		out.Variables = []WhatsAppVariable{}
	}
	a, aerr := s.account(ctx, s.Q, ws)
	out.Connected = aerr == nil
	var tpl *store.WhatsappTemplate
	if aerr == nil {
		out.AccountStatus = a.Status
		out.Business = &WhatsAppBusiness{DisplayName: a.DisplayName, Number: phone.Display(a.PhoneNumber)}
		ts, err := s.Q.WhatsAppTemplates(ctx, a.ID)
		if err != nil {
			return out, err
		}
		for _, t := range ts {
			out.Templates = append(out.Templates, templateView(t))
		}
	}
	if c.TemplateID != nil {
		if t, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: *c.TemplateID, WorkspaceID: ws.ID}); err == nil {
			tpl = &t
			v := templateView(t)
			out.TemplateID, out.Template = t.ID.String(), &v
		}
	}
	lists, segments := audienceOf(c.Audience)
	for _, sid := range segments {
		out.Segments = append(out.Segments, sid.String())
	}
	opts, _, err := s.listConsent(ctx, ws)
	if err != nil {
		return out, err
	}
	out.ListOptions = opts
	aud, err := s.audience(ctx, ws, lists, segments)
	if err != nil {
		return out, err
	}
	out.Audience = summary(aud, opts, lists)
	rl := s.rules(ctx, s.Q, ws)
	out.Quiet, out.Window = rl.Quiet, rl.Window.String()
	out.Schedule = scheduleOf(c, ws)
	if tpl != nil {
		out.Cost = s.rateCard(ctx, s.Q, ws).estimate(tpl.Category, aud.countries)
		out.Preview = s.preview(ctx, ws, *tpl, out.Variables, aud, preview)
	} else {
		out.Cost = WhatsAppCost{ByCountry: []WhatsAppCostCountry{}, RatesAsOf: s.rateCard(ctx, s.Q, ws).asOf}
	}
	out.Test = s.lastTest(ctx, ws, c.ID)
	return out, nil
}

// summary is the audience as the composer shows it.
func summary(aud audienceSet, opts []WhatsAppListConsent, lists []uuid.UUID) WhatsAppAudience {
	out := WhatsAppAudience{Lists: []WhatsAppListConsent{}, Consent: aud.consent, Without: aud.without, Suppressed: aud.suppressed,
		Final: int64(len(aud.list)), Countries: aud.countries}
	for _, id := range lists {
		for _, o := range opts {
			if o.ListID == id.String() {
				out.Lists = append(out.Lists, o)
			}
		}
	}
	return out
}

// scheduleOf is a campaign's schedule in words: "21 Oct, 09:00 in each
// recipient's time zone".
func scheduleOf(c store.WhatsAppCampaignRow, ws *store.Workspace) WhatsAppSchedule {
	out := WhatsAppSchedule{Mode: c.ScheduleMode, SendAt: c.SendAt, Label: "Now"}
	switch c.ScheduleMode {
	case "at":
		if c.SendAt != nil {
			loc := wsLocation(ws)
			out.Label = c.SendAt.In(loc).Format("2 Jan, 15:04") + " " + tzName(ws)
		}
	case "local":
		if c.LocalDate != nil && c.LocalMinute != nil {
			m := int(*c.LocalMinute)
			out.LocalDate, out.LocalMinute = c.LocalDate.Format("2006-01-02"), &m
			out.Label = c.LocalDate.Format("2 Jan") + fmt.Sprintf(", %02d:%02d in each recipient's time zone", m/60, m%60)
		}
	}
	return out
}

func tzName(ws *store.Workspace) string {
	if ws.Timezone == "" {
		return "UTC"
	}
	return ws.Timezone
}

// preview fills the template for one contact: the one asked for, else the
// audience's most recent writer, else its first.
func (s *Service) preview(ctx context.Context, ws *store.Workspace, tpl store.WhatsappTemplate, vars []WhatsAppVariable, aud audienceSet,
	want uuid.UUID) *WhatsAppPreview {
	if len(aud.list) == 0 {
		return nil
	}
	pick := aud.list[0]
	found := false
	if want != uuid.Nil {
		for _, r := range aud.list {
			if r.ContactID == want {
				pick, found = r, true
			}
		}
	}
	if !found {
		phones := make([]string, 0, len(aud.list))
		for _, r := range aud.list {
			phones = append(phones, r.Phone)
		}
		var ph string
		if err := s.Pool.QueryRow(ctx, `SELECT phone FROM whatsapp_inbound WHERE workspace_id = $1 AND phone = ANY($2) ORDER BY received_at DESC LIMIT 1`,
			ws.ID, phones).Scan(&ph); err == nil {
			for _, r := range aud.list {
				if r.Phone == ph {
					pick = r
					break
				}
			}
		}
	}
	keys := variablesOf(tpl.Body)
	params := paramsFor(keys, vars, pick)
	out := &WhatsAppPreview{ContactID: pick.ContactID.String(), Contact: fullName(pick.FirstName, pick.LastName), FirstName: pick.FirstName,
		Text: render(tpl.Body, params), Buttons: templateView(tpl).Buttons, Fallbacks: []WhatsAppFallback{}}
	if out.Contact == "" {
		out.Contact = pick.Email
	}
	for _, v := range vars {
		if v.Source == "field" && v.Fallback != "" {
			out.Fallbacks = append(out.Fallbacks, WhatsAppFallback{Key: v.Key, Field: v.Field, Fallback: v.Fallback})
		}
	}
	return out
}

func (s *Service) accessFor(ctx context.Context, ws *store.Workspace, p *authz.Principal) WhatsAppAccess {
	a := WhatsAppAccess{CanEdit: p.Can(authz.CampaignsEdit), CanSend: canSend(p), Missing: []string{}, Helpers: []string{}}
	if !p.Can(authz.CampaignsSend) {
		a.Missing = append(a.Missing, string(authz.CampaignsSend))
	}
	if !p.Can(authz.WhatsAppSend) {
		a.Missing = append(a.Missing, string(authz.WhatsAppSend))
	}
	if !a.CanSend {
		a.Helpers = s.helpers(ctx, ws, authz.WhatsAppSend)
	}
	return a
}

// lastTest is the campaign's last test.
func (s *Service) lastTest(ctx context.Context, ws *store.Workspace, id uuid.UUID) *WhatsAppTestResult {
	var mid uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT id FROM channel_messages WHERE campaign_id = $1 AND kind = 'test' AND workspace_id = $2
		ORDER BY created_at DESC LIMIT 1`, id, ws.ID).Scan(&mid); err != nil {
		return nil
	}
	t, err := s.TestStatus(ctx, ws, mid)
	if err != nil {
		return nil
	}
	return &t
}

// UpdateCampaign autosaves the composer: the template, its variables, the
// audience and the schedule.
func (s *Service) UpdateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WhatsAppCampaignUpdate) (WhatsAppCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	switch c.Status {
	case "draft", "approval", "approved":
	default:
		return WhatsAppCampaign{}, api.Conflict("This campaign is " + c.Status + ": it can't be edited any more.")
	}
	if in.Version > 0 && in.Version != int(c.Version) {
		return WhatsAppCampaign{}, api.Err(409, "stale_version", fmt.Sprintf("Someone saved a newer version (%d). Reload to see it.", c.Version))
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return WhatsAppCampaign{}, api.Invalid("body.name", "Give the campaign a name.")
	}
	tpl, err := s.approvedTemplate(ctx, ws, in.TemplateID, "body.templateId")
	if err != nil {
		// A template that stopped being approved can stay on a draft.
		if c.TemplateID == nil || in.TemplateID != c.TemplateID.String() {
			return WhatsAppCampaign{}, err
		}
		t, terr := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: *c.TemplateID, WorkspaceID: ws.ID})
		if terr != nil {
			return WhatsAppCampaign{}, err
		}
		tpl = &t
	}
	lists, err := parseIDs(in.Lists, "body.lists", "lists")
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	if err := s.checkLists(ctx, ws, lists); err != nil {
		return WhatsAppCampaign{}, err
	}
	segments, err := parseIDs(in.Segments, "body.segments", "segments")
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	vars, err := s.checkVariables(ctx, ws, tpl, in.Variables)
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	mode := in.Mode
	var sendAt, localDate *time.Time
	var localMinute *int32
	switch mode {
	case "at":
		if in.SendAt == nil || !in.SendAt.After(s.now()) {
			return WhatsAppCampaign{}, api.Invalid("body.sendAt", "Pick a time in the future.")
		}
		sendAt = in.SendAt
	case "local":
		d, err := time.Parse("2006-01-02", in.LocalDate)
		if err != nil || in.LocalMinute == nil {
			return WhatsAppCampaign{}, api.Invalid("body.localDate", "Pick a day and a time.")
		}
		rl := s.rules(ctx, s.Q, ws)
		if rl.Quiet && (*in.LocalMinute < rl.Window.From || *in.LocalMinute >= rl.Window.To) {
			return WhatsAppCampaign{}, api.Invalid("body.localMinute", "Pick a time inside the quiet-hours window, "+rl.Window.String()+".")
		}
		localDate, localMinute = &d, ptr(int32(*in.LocalMinute))
	default:
		mode = "now"
	}
	over := in.OverLimit
	if over == "" {
		over = c.OverLimit
	}
	var tid *uuid.UUID
	tname, tlang := "", ""
	if tpl != nil {
		tid, tname, tlang = &tpl.ID, tpl.Name, tpl.Language
	}
	hash := contentHash(tid, jsonOf(vars))
	aud := audienceJSON(lists, segments)
	changed := hash != c.ContentHash || string(aud) != string(c.Audience) || name != c.Name || mode != c.ScheduleMode ||
		!sameTime(sendAt, c.SendAt) || !sameDate(localDate, c.LocalDate) || !sameInt(localMinute, c.LocalMinute) || over != c.OverLimit
	version := c.Version
	if changed {
		version++
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.WhatsAppCampaignSave(ctx, store.WhatsAppCampaignSaveParams{Name: name, Audience: aud, ContentHash: hash, Version: version, By: uuidPtr(p.UserID),
			At: now, ID: id, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		if err := q.WhatsAppCampaignDetailSave(ctx, store.WhatsAppCampaignDetailSaveParams{TemplateID: tid, TemplateName: tname, TemplateLanguage: tlang,
			Variables: jsonOf(vars), ScheduleMode: mode, SendAt: sendAt, LocalDate: localDate, LocalMinute: localMinute, OverLimit: over, CampaignID: id}); err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			Before: map[string]any{"version": c.Version, "template": c.TemplateName}, After: map[string]any{"version": version, "template": tname},
			Metadata: map[string]any{"channel": "whatsapp", "autosave": true}})
	})
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	if changed && s.Changed != nil {
		s.Changed(ctx, ws, id)
	}
	return s.Campaign(ctx, ws, p, id, uuid.Nil)
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

func sameInt(a, b *int32) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// checkVariables keeps a variable per {{n}} of the template, each a known
// field or a text.
func (s *Service) checkVariables(ctx context.Context, ws *store.Workspace, tpl *store.WhatsappTemplate, in []WhatsAppVariable) ([]WhatsAppVariable, error) {
	if tpl == nil {
		return []WhatsAppVariable{}, nil
	}
	known := map[string]bool{}
	for _, f := range s.fields(ctx, ws) {
		known[f.Key] = true
	}
	byKey := map[string]WhatsAppVariable{}
	for _, v := range in {
		byKey[v.Key] = v
	}
	out := []WhatsAppVariable{}
	for _, k := range variablesOf(tpl.Body) {
		v, ok := byKey[k]
		if !ok {
			v = WhatsAppVariable{Key: k, Source: "field"}
		}
		v.Key = k
		switch v.Source {
		case "text":
			v.Field = ""
		default:
			v.Source, v.Text = "field", ""
			if v.Field != "" && !known[v.Field] {
				return nil, api.Invalid("body.variables", "{{"+k+"}} takes a contact field this workspace doesn't have: "+v.Field+".")
			}
		}
		v.Fallback = strings.TrimSpace(v.Fallback)
		out = append(out, v)
	}
	return out, nil
}

// DuplicateCampaign copies a campaign as a draft (logged).
func (s *Service) DuplicateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WhatsAppCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	nid := uuid.Must(uuid.NewV7())
	now := s.now()
	name := clip("Copy of "+c.Name, 200)
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.WhatsAppCampaignInsert(ctx, store.WhatsAppCampaignInsertParams{ID: nid, WorkspaceID: ws.ID, Name: name, Audience: c.Audience,
			ContentHash: c.ContentHash, By: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		if err := q.WhatsAppCampaignDetailInsert(ctx, store.WhatsAppCampaignDetailInsertParams{CampaignID: nid, WorkspaceID: ws.ID, TemplateID: c.TemplateID,
			TemplateName: c.TemplateName, TemplateLanguage: c.TemplateLanguage, Variables: c.Variables, ScheduleMode: "now", OverLimit: c.OverLimit}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: nid.String(), TargetName: name,
			After: map[string]any{"name": name, "channel": "whatsapp", "copied_from": c.ID.String()}})
	})
	if err != nil {
		return WhatsAppCampaign{}, err
	}
	return s.Campaign(ctx, ws, p, nid, uuid.Nil)
}

// WhatsAppCampaignTestInput sends a campaign to a member's WhatsApp.
type WhatsAppCampaignTestInput struct {
	Phone string `json:"phone" minLength:"1" maxLength:"40"`
}

// SendCampaignTest sends the campaign's template to a member's WhatsApp
// (logged), its variables filled with the fixed texts and fallbacks.
func (s *Service) SendCampaignTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WhatsAppCampaignTestInput) (WhatsAppTestResult, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppTestResult{}, err
	}
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return WhatsAppTestResult{}, api.Conflict("Connect WhatsApp first.")
	}
	if err != nil {
		return WhatsAppTestResult{}, err
	}
	if c.TemplateID == nil {
		return WhatsAppTestResult{}, api.Invalid("body", "Pick a template first.")
	}
	tpl, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: *c.TemplateID, WorkspaceID: ws.ID})
	if err != nil || tpl.Status != "approved" {
		return WhatsAppTestResult{}, api.Conflict("The campaign's template isn't approved: pick an approved one.")
	}
	n, err := phone.Parse(in.Phone, "")
	if err != nil {
		return WhatsAppTestResult{}, api.Invalid("body.phone", "Enter the number with its country code, like +1 312 555 0123.")
	}
	var vars []WhatsAppVariable
	_ = json.Unmarshal(c.Variables, &vars)
	me := recipient{Phone: n.E164}
	if u, err := s.Q.GetUser(ctx, p.UserID); err == nil {
		parts := strings.Fields(u.Name)
		if len(parts) > 0 {
			me.FirstName = parts[0]
			me.LastName = strings.Join(parts[1:], " ")
		}
		me.Email = u.Email
	}
	params := paramsFor(variablesOf(tpl.Body), vars, me)
	return s.queueTest(ctx, ws, p, a, &c.ID, tpl, params, n, "campaign.test_sent", "campaign", c.ID.String(), c.Name)
}
