package agency

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A workspace template copies a client's setup into another client:
//
//   - sender defaults: the From names and the footer text, not domains
//   - lists, empty
//   - automations, as drafts, with their triggers pointed at the new lists
//   - saved sections, campaign templates and SMS texts
//   - signup forms, paused until a sender is verified
//   - the approval policy's client sign-off, and custom roles
//   - branding, with the logo and colours left empty
//
// Never copied: providers and credentials, domains and senders, contacts,
// suppressions, consent, API keys, integrations and members. Applying adds
// what's missing and renames clashes ("Welcome series (template)"); it never
// overwrites or deletes anything.

// TemplateContent is what a workspace template sets up.
type TemplateContent struct {
	Lists       []TplList       `json:"lists"`
	Automations []TplAutomation `json:"automations"`
	// AutomationsNote names the automations in a few words on the card
	// ("welcome, booking reminder by SMS, win-back").
	AutomationsNote string    `json:"automationsNote,omitempty"`
	Sections        []TplItem `json:"sections"`
	Templates       []TplItem `json:"templates"`
	SMSTexts        []TplSMS  `json:"smsTexts"`
	Forms           []TplForm `json:"forms"`
	// ClientSignoff turns the approval policy's client sign-off on.
	ClientSignoff bool      `json:"clientSignoff"`
	Roles         []TplRole `json:"roles"`
	// Branding holds the public pages' branding without the logo and colours.
	Branding map[string]any `json:"branding,omitempty"`
	// SenderNames are the default senders' From names by stream, and Footer
	// the postal address under every email.
	SenderNames map[string]string `json:"senderNames,omitempty"`
	Footer      string            `json:"footer,omitempty"`
	// NextSteps are the setup steps a new client gets: provider, domain,
	// invite, and shopify when the template expects a store.
	NextSteps []string `json:"nextSteps"`
}

// TplList is a list, copied empty.
type TplList struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	DoubleOptIn bool   `json:"doubleOptIn"`
}

// TplAutomation is a flow; list, form and email references in its graph
// are names ("list:Newsletter", "form:Signup", "email:welcome-1").
type TplAutomation struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Graph       json.RawMessage `json:"graph"`
	Emails      []TplEmail      `json:"emails"`
}

// TplEmail is one of a flow's emails.
type TplEmail struct {
	Ref       string `json:"ref"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader,omitempty"`
	MJML      string `json:"mjml"`
	HTML      string `json:"html"`
	Text      string `json:"text,omitempty"`
}

// TplItem is a saved section or a campaign template.
type TplItem struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	MJML        string          `json:"mjml"`
	HTML        string          `json:"html,omitempty"`
	Thumb       json.RawMessage `json:"thumb,omitempty"`
}

// TplSMS is an SMS text.
type TplSMS struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

// TplForm is a signup form; its lists are names.
type TplForm struct {
	Name           string          `json:"name"`
	Kind           string          `json:"kind"`
	Lists          []string        `json:"lists"`
	DoubleOptIn    bool            `json:"doubleOptIn"`
	Fields         json.RawMessage `json:"fields"`
	Content        json.RawMessage `json:"content"`
	SpamProtection json.RawMessage `json:"spamProtection"`
	ConsentText    string          `json:"consentText"`
}

// TplRole is a custom role.
type TplRole struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

func parseContent(raw json.RawMessage) TemplateContent {
	var c TemplateContent
	_ = json.Unmarshal(raw, &c)
	return c
}

// brandingNeverCopied are the branding keys a template leaves empty: the
// logo and the colours.
var brandingNeverCopied = []string{"logo", "accent", "accentHover", "accentSoft", "accentText", "onAccent", "background", "footer_address", "address", "postal_address"}

// capture reads a workspace's setup into a template's content.
func (s *Service) capture(ctx context.Context, q *store.Queries, ws store.Workspace) (TemplateContent, error) {
	c := TemplateContent{Lists: []TplList{}, Automations: []TplAutomation{}, Sections: []TplItem{}, Templates: []TplItem{}, SMSTexts: []TplSMS{},
		Forms: []TplForm{}, Roles: []TplRole{}, NextSteps: []string{"provider", "domain", "invite"}}
	lists, err := q.AgencyCaptureLists(ctx, ws.ID)
	if err != nil {
		return c, err
	}
	listNames := map[string]string{}
	for _, l := range lists {
		c.Lists = append(c.Lists, TplList{Name: l.Name, Description: l.Description, DoubleOptIn: l.DoubleOptIn})
		listNames[l.ID.String()] = l.Name
	}
	fs, err := q.AgencyCaptureForms(ctx, ws.ID)
	if err != nil {
		return c, err
	}
	formNames := map[string]string{}
	for _, f := range fs {
		names := []string{}
		for _, id := range f.ListIds {
			if n, ok := listNames[id.String()]; ok {
				names = append(names, n)
			}
		}
		formNames[f.ID.String()] = f.Name
		c.Forms = append(c.Forms, TplForm{Name: f.Name, Kind: f.Kind, Lists: names, DoubleOptIn: f.DoubleOptIn, Fields: f.Fields, Content: f.Content,
			SpamProtection: f.SpamProtection, ConsentText: f.ConsentText})
	}
	flows, err := q.AgencyCaptureAutomations(ctx, ws.ID)
	if err != nil {
		return c, err
	}
	emails, err := q.AgencyCaptureAutomationEmails(ctx, ws.ID)
	if err != nil {
		return c, err
	}
	for _, f := range flows {
		g, err := automations.ParseGraph(f.Graph)
		if err != nil {
			continue
		}
		ta := TplAutomation{Name: f.Name, Description: f.Description, Emails: []TplEmail{}}
		refs := map[string]string{}
		for _, e := range emails {
			if e.AutomationID == nil || *e.AutomationID != f.ID {
				continue
			}
			ref := fmt.Sprintf("e%d", len(ta.Emails)+1)
			refs[e.ID.String()] = ref
			ta.Emails = append(ta.Emails, TplEmail{Ref: ref, Name: e.Name, Subject: e.Subject, Preheader: e.Preheader, MJML: e.Mjml, HTML: e.Html, Text: e.PlainText})
		}
		symbolize(&g, listNames, formNames, refs)
		ta.Graph = g.MustJSON()
		c.Automations = append(c.Automations, ta)
	}
	tpls, err := q.AgencyCaptureTemplates(ctx, ws.ID)
	if err != nil {
		return c, err
	}
	for _, t := range tpls {
		item := TplItem{Name: t.Name, Description: t.Description, MJML: t.Mjml, HTML: t.Html, Thumb: t.Thumb}
		if t.Kind == "section" {
			c.Sections = append(c.Sections, item)
		} else {
			c.Templates = append(c.Templates, item)
		}
	}
	roles, err := q.AgencyCaptureRoles(ctx, ws.ID)
	if err != nil {
		return c, err
	}
	for _, r := range roles {
		if r.Builtin {
			if authz.IsClientRole(r.Key) {
				c.Roles = append(c.Roles, TplRole{Name: r.Name, Permissions: nil})
			}
			continue
		}
		c.Roles = append(c.Roles, TplRole{Name: r.Name, Description: r.Description, Permissions: r.Permissions})
	}
	c.ClientSignoff = approvals.ParsePolicy(ws.ApprovalPolicy).ClientSignoff.Enabled
	var branding map[string]any
	_ = json.Unmarshal(ws.Branding, &branding)
	if branding != nil {
		if v, ok := branding["footer_address"].(string); ok {
			c.Footer = v
		}
		for _, k := range brandingNeverCopied {
			delete(branding, k)
		}
		if len(branding) > 0 {
			c.Branding = branding
		}
	}
	c.SenderNames = map[string]string{}
	for stream, id := range map[string]*uuid.UUID{"broadcast": ws.DefaultSenderID, "automation": ws.DefaultAutomationSenderID, "system": ws.DefaultSystemSenderID} {
		if id == nil {
			continue
		}
		if n, err := q.AgencySenderName(ctx, *id); err == nil && n != "" {
			c.SenderNames[stream] = n
		}
	}
	return c, nil
}

// symbolize replaces a captured graph's ids with names, so the graph can
// be pointed at another workspace's lists, forms and emails.
func symbolize(g *automations.AutomationGraph, lists, forms, emails map[string]string) {
	sym := func(prefix string, names map[string]string, id string) string {
		if id == "" {
			return ""
		}
		if n, ok := names[id]; ok {
			return prefix + n
		}
		return ""
	}
	for i := range g.Nodes {
		d := &g.Nodes[i].Data
		if t := d.Trigger; t != nil {
			t.ListID = sym("list:", lists, t.ListID)
			t.FormID = sym("form:", forms, t.FormID)
			t.Poll = ""
		}
		if e := d.Email; e != nil {
			e.EmailID = sym("email:", emails, e.EmailID)
			e.EmailVersion = 0
			e.Route = ""
			e.SkipListID = sym("list:", lists, e.SkipListID)
		}
		if c := d.Condition; c != nil {
			c.ListID = sym("list:", lists, c.ListID)
		}
		if a := d.Action; a != nil {
			a.ListID = sym("list:", lists, a.ListID)
		}
		if w := d.Webhook; w != nil {
			// The signing secret stays behind; the step needs a new one.
			w.HasSecret = false
			w.Demo = false
		}
	}
}

// resolve points a template's graph at a workspace's lists, forms and
// emails (by name); references it can't resolve are left empty, and the
// flow editor asks for them.
func resolve(g *automations.AutomationGraph, lists, forms, emails map[string]string) {
	res := func(prefix string, ids map[string]string, v string) string {
		if !strings.HasPrefix(v, prefix) {
			return ""
		}
		return ids[strings.TrimPrefix(v, prefix)]
	}
	for i := range g.Nodes {
		d := &g.Nodes[i].Data
		if t := d.Trigger; t != nil {
			t.ListID = res("list:", lists, t.ListID)
			t.FormID = res("form:", forms, t.FormID)
		}
		if e := d.Email; e != nil {
			e.EmailID = res("email:", emails, e.EmailID)
			e.SkipListID = res("list:", lists, e.SkipListID)
		}
		if c := d.Condition; c != nil {
			c.ListID = res("list:", lists, c.ListID)
		}
		if a := d.Action; a != nil {
			a.ListID = res("list:", lists, a.ListID)
		}
	}
}

// AgencyTemplate is a workspace template on the Library page.
type AgencyTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Default     bool      `json:"default"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   string    `json:"updatedBy,omitempty"`
	// SetsUp counts what it sets up.
	SetsUp AgencyTemplateSummary `json:"setsUp"`
	// UsedBy are the clients it set up or was applied to.
	UsedBy    []AgencyWorkspaceRef `json:"usedBy"`
	NextSteps []string             `json:"nextSteps"`
}

// AgencyTemplateSummary counts a template's content.
type AgencyTemplateSummary struct {
	Lists           int      `json:"lists"`
	ListNames       []string `json:"listNames"`
	Automations     int      `json:"automations"`
	AutomationNames []string `json:"automationNames"`
	AutomationsNote string   `json:"automationsNote,omitempty"`
	Sections        int      `json:"sections"`
	Templates       int      `json:"templates"`
	SMSTexts        int      `json:"smsTexts"`
	Forms           int      `json:"forms"`
	ClientSignoff   bool     `json:"clientSignoff"`
	Roles           int      `json:"roles"`
	ClientRoles     bool     `json:"clientRoles"`
	Branding        bool     `json:"branding"`
}

func summarize(c TemplateContent) AgencyTemplateSummary {
	out := AgencyTemplateSummary{Lists: len(c.Lists), ListNames: []string{}, Automations: len(c.Automations), AutomationNames: []string{},
		AutomationsNote: c.AutomationsNote, Sections: len(c.Sections), Templates: len(c.Templates), SMSTexts: len(c.SMSTexts), Forms: len(c.Forms),
		ClientSignoff: c.ClientSignoff, Branding: len(c.Branding) > 0}
	for _, l := range c.Lists {
		out.ListNames = append(out.ListNames, l.Name)
	}
	for _, a := range c.Automations {
		out.AutomationNames = append(out.AutomationNames, a.Name)
	}
	for _, r := range c.Roles {
		if r.Permissions == nil {
			out.ClientRoles = true
		} else {
			out.Roles++
		}
	}
	return out
}

// Templates lists the workspace templates.
func (s *Service) Templates(ctx context.Context) ([]AgencyTemplate, error) {
	me, st, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return nil, err
	}
	rows, err := s.Q.AgencyTemplates(ctx)
	if err != nil {
		return nil, err
	}
	uses, err := s.Q.AgencyTemplateUses(ctx)
	if err != nil {
		return nil, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return nil, err
	}
	out := make([]AgencyTemplate, 0, len(rows))
	for _, r := range rows {
		c := parseContent(r.Content)
		t := AgencyTemplate{ID: r.ID.String(), Name: r.Name, Description: r.Description, UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedByName,
			Default: st.DefaultTemplateID != nil && *st.DefaultTemplateID == r.ID, SetsUp: summarize(c), UsedBy: []AgencyWorkspaceRef{}, NextSteps: c.NextSteps}
		if t.NextSteps == nil {
			t.NextSteps = []string{}
		}
		for _, u := range uses {
			if u.TemplateID != r.ID {
				continue
			}
			if w, ok := set.find(u.WorkspaceID); ok {
				t.UsedBy = append(t.UsedBy, wsRef(w))
			} else {
				for _, a := range set.all {
					if a.ID == u.WorkspaceID {
						t.UsedBy = append(t.UsedBy, wsRef(a))
					}
				}
			}
		}
		out = append(out, t)
	}
	return out, nil
}

// AgencyTemplateFromWorkspace saves a client's setup as a template.
type AgencyTemplateFromWorkspace struct {
	Name        string `json:"name" minLength:"1" maxLength:"80"`
	Description string `json:"description,omitempty" maxLength:"300"`
}

// SaveAsTemplate saves a workspace's setup as a new workspace template
// ("Save as workspace template" in a client's Settings › General).
func (s *Service) SaveAsTemplate(ctx context.Context, ws *store.Workspace, in AgencyTemplateFromWorkspace) (AgencyTemplateRef, error) {
	me, _, err := s.require(ctx, authz.AgencyLibrary)
	if err != nil {
		return AgencyTemplateRef{}, err
	}
	l := i18n.LangFrom(ctx)
	if _, err := s.Q.AgencyWorkspace(ctx, ws.ID); err != nil {
		return AgencyTemplateRef{}, api.Conflict(mNotAgencyWorkspace.Text(l, i18n.Args{"workspace": ws.Name}))
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return AgencyTemplateRef{}, api.Invalid("body.name", mTemplateName.Text(l, nil))
	}
	if taken, err := s.Q.AgencyTemplateNameTaken(ctx, store.AgencyTemplateNameTakenParams{Name: name, ExcludeID: uuid.Nil}); err != nil {
		return AgencyTemplateRef{}, err
	} else if taken {
		return AgencyTemplateRef{}, api.Invalid("body.name", mTemplateNameTaken.Text(l, i18n.Args{"name": name}))
	}
	c, err := s.capture(ctx, s.Q, *ws)
	if err != nil {
		return AgencyTemplateRef{}, err
	}
	raw, _ := json.Marshal(c)
	id := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AgencyTemplateInsert(ctx, store.AgencyTemplateInsertParams{ID: id, Name: name, Description: strings.TrimSpace(in.Description), Content: raw,
			CreatedFrom: &ws.ID, CreatedBy: &me.UserID}); err != nil {
			return err
		}
		sum := summarize(c)
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.template_created", TargetType: "agency_template", TargetID: id.String(), TargetName: name,
			After: map[string]any{"from": ws.Name, "lists": sum.Lists, "automations": sum.Automations, "sections": sum.Sections, "templates": sum.Templates},
			Metadata: map[string]any{"permission": string(authz.AgencyLibrary), "workspace": ws.Name}})
	})
	return AgencyTemplateRef{ID: id.String(), Name: name}, err
}

// AgencyTemplateInput edits a template's name, description and what it
// includes.
type AgencyTemplateInput struct {
	Name        string `json:"name" minLength:"1" maxLength:"80"`
	Description string `json:"description,omitempty" maxLength:"300"`
	// Include switches the template's parts on or off (all on when empty):
	// lists, automations, sections, templates, smsTexts, forms,
	// clientSignoff, roles, branding, senderDefaults.
	Exclude []string `json:"exclude,omitempty" maxItems:"10"`
	// NextSteps are the setup steps new clients get.
	NextSteps []string `json:"nextSteps,omitempty" maxItems:"4"`
}

// UpdateTemplate renames a template, or leaves parts out of it.
func (s *Service) UpdateTemplate(ctx context.Context, id string, in AgencyTemplateInput) error {
	me, _, err := s.require(ctx, authz.AgencyLibrary)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	tid, err := uuid.Parse(id)
	if err != nil {
		return api.NotFound("Template")
	}
	t, err := s.Q.AgencyTemplate(ctx, tid)
	if err != nil {
		return api.MapError(err, "Template")
	}
	name := strings.TrimSpace(in.Name)
	if taken, err := s.Q.AgencyTemplateNameTaken(ctx, store.AgencyTemplateNameTakenParams{Name: name, ExcludeID: tid}); err != nil {
		return err
	} else if taken {
		return api.Invalid("body.name", mTemplateNameTaken.Text(l, i18n.Args{"name": name}))
	}
	c := parseContent(t.Content)
	for _, x := range in.Exclude {
		switch x {
		case "lists":
			c.Lists = []TplList{}
		case "automations":
			c.Automations = []TplAutomation{}
		case "sections":
			c.Sections = []TplItem{}
		case "templates":
			c.Templates = []TplItem{}
		case "smsTexts":
			c.SMSTexts = []TplSMS{}
		case "forms":
			c.Forms = []TplForm{}
		case "clientSignoff":
			c.ClientSignoff = false
		case "roles":
			c.Roles = []TplRole{}
		case "branding":
			c.Branding = nil
		case "senderDefaults":
			c.SenderNames, c.Footer = nil, ""
		}
	}
	if in.NextSteps != nil {
		steps := []string{}
		for _, k := range in.NextSteps {
			if slices.Contains([]string{"provider", "domain", "invite", "shopify"}, k) && !slices.Contains(steps, k) {
				steps = append(steps, k)
			}
		}
		c.NextSteps = steps
	}
	raw, _ := json.Marshal(c)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyTemplateUpdate(ctx, store.AgencyTemplateUpdateParams{ID: tid, Name: name, Description: strings.TrimSpace(in.Description), Content: raw, UpdatedBy: &me.UserID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.template_changed", TargetType: "agency_template", TargetID: tid.String(), TargetName: name,
			Before: map[string]any{"name": t.Name, "description": t.Description}, After: map[string]any{"name": name, "description": strings.TrimSpace(in.Description), "excluded": in.Exclude},
			Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
	})
}

// DeleteTemplate removes a template. The clients it set up keep everything.
func (s *Service) DeleteTemplate(ctx context.Context, id string) error {
	if _, _, err := s.require(ctx, authz.AgencyLibrary); err != nil {
		return err
	}
	tid, err := uuid.Parse(id)
	if err != nil {
		return api.NotFound("Template")
	}
	t, err := s.Q.AgencyTemplate(ctx, tid)
	if err != nil {
		return api.MapError(err, "Template")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyTemplateDelete(ctx, tid); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.template_deleted", TargetType: "agency_template", TargetID: tid.String(), TargetName: t.Name,
			Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
	})
}

// AgencyTemplatePlan is what applying a template to a client adds, and
// what it renames because the client has something with that name.
type AgencyTemplatePlan struct {
	Template  AgencyTemplateRef  `json:"template"`
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Lists       AgencyPlanPart `json:"lists"`
	Automations AgencyPlanPart `json:"automations"`
	Sections    AgencyPlanPart `json:"sections"`
	Templates   AgencyPlanPart `json:"templates"`
	SMSTexts    AgencyPlanPart `json:"smsTexts"`
	Forms       AgencyPlanPart `json:"forms"`
	Roles       AgencyPlanPart `json:"roles"`
	// ClientSignoff: on (turned on), already (already on), off (not in the
	// template) or skipped (you can't change the policy there).
	ClientSignoff string `json:"clientSignoff" enum:"on,already,off,skipped"`
	// ClientRoles: added or already there.
	ClientRoles string `json:"clientRoles" enum:"added,already"`
	Branding    bool   `json:"branding"`
	Senders     bool   `json:"senders"`
	NextSteps   []string `json:"nextSteps"`
	// Skipped names the parts you can't add there, with the permission.
	Skipped []AgencyPlanSkip `json:"skipped"`
	// AppliedAt and AppliedBy are set once it's applied.
	AppliedAt *time.Time `json:"appliedAt,omitempty"`
	AppliedBy string     `json:"appliedBy,omitempty"`
}

// AgencyPlanPart is one kind of content.
type AgencyPlanPart struct {
	Added   []string           `json:"added"`
	Renamed []AgencyPlanRename `json:"renamed"`
}

// AgencyPlanRename is a clash renamed on the way in.
type AgencyPlanRename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// AgencyPlanSkip is a part left out, and why.
type AgencyPlanSkip struct {
	Part       string `json:"part"`
	Permission string `json:"permission"`
}

func newPart() AgencyPlanPart {
	return AgencyPlanPart{Added: []string{}, Renamed: []AgencyPlanRename{}}
}

// renamedName is a clash's new name: "Welcome series (template)", then
// "Welcome series (template 2)".
func renamedName(name string, taken func(string) (bool, error)) (string, error) {
	cand := name + " (template)"
	for i := 2; i < 100; i++ {
		t, err := taken(cand)
		if err != nil {
			return "", err
		}
		if !t {
			return cand, nil
		}
		cand = fmt.Sprintf("%s (template %d)", name, i)
	}
	return cand, nil
}

// plan works out what applying a template to a workspace would do. With
// apply set it does it, in the caller's transaction.
func (s *Service) plan(ctx context.Context, q *store.Queries, tx pgx.Tx, tpl store.AgencyTemplateRow, w store.AgencyWorkspacesRow, perms authz.Set, by *uuid.UUID, apply bool) (AgencyTemplatePlan, error) {
	c := parseContent(tpl.Content)
	out := AgencyTemplatePlan{Template: AgencyTemplateRef{ID: tpl.ID.String(), Name: tpl.Name}, Workspace: wsRef(w),
		Lists: newPart(), Automations: newPart(), Sections: newPart(), Templates: newPart(), SMSTexts: newPart(), Forms: newPart(), Roles: newPart(),
		ClientSignoff: "off", ClientRoles: "already", NextSteps: c.NextSteps, Skipped: []AgencyPlanSkip{}}
	if out.NextSteps == nil {
		out.NextSteps = []string{}
	}
	ws, err := q.GetWorkspace(ctx, w.ID)
	if err != nil {
		return out, err
	}
	now := s.now()
	skip := func(part string, perm authz.Permission) bool {
		if perms.Has(perm) {
			return false
		}
		out.Skipped = append(out.Skipped, AgencyPlanSkip{Part: part, Permission: string(perm)})
		return true
	}
	// Lists first: automations and forms point at them.
	listIDs := map[string]string{}
	if len(c.Lists) > 0 && !skip("lists", authz.ListsManage) {
		existing, err := q.AgencyCaptureLists(ctx, w.ID)
		if err != nil {
			return out, err
		}
		for _, l := range c.Lists {
			name := l.Name
			taken := func(n string) (bool, error) {
				return q.ListNameTaken(ctx, store.ListNameTakenParams{WorkspaceID: w.ID, Name: n, ExceptID: uuid.Nil})
			}
			clash, err := taken(name)
			if err != nil {
				return out, err
			}
			if clash {
				if name, err = renamedName(l.Name, taken); err != nil {
					return out, err
				}
				out.Lists.Renamed = append(out.Lists.Renamed, AgencyPlanRename{From: l.Name, To: name})
			} else {
				out.Lists.Added = append(out.Lists.Added, l.Name)
			}
			if apply {
				id := uuid.Must(uuid.NewV7())
				if _, err := q.CreateList(ctx, store.CreateListParams{ID: id, WorkspaceID: w.ID, Name: name, Description: l.Description, DoubleOptIn: l.DoubleOptIn}); err != nil {
					return out, err
				}
				listIDs[l.Name] = id.String()
			}
		}
		_ = existing
	}
	// Signup forms, paused until a sender is verified.
	formIDs := map[string]string{}
	if len(c.Forms) > 0 && !skip("forms", authz.FormsEdit) {
		for _, f := range c.Forms {
			taken := func(n string) (bool, error) {
				return q.AgencyFormNameTaken(ctx, store.AgencyFormNameTakenParams{WorkspaceID: w.ID, Name: n})
			}
			name := f.Name
			clash, err := taken(name)
			if err != nil {
				return out, err
			}
			if clash {
				if name, err = renamedName(f.Name, taken); err != nil {
					return out, err
				}
				out.Forms.Renamed = append(out.Forms.Renamed, AgencyPlanRename{From: f.Name, To: name})
			} else {
				out.Forms.Added = append(out.Forms.Added, f.Name)
			}
			if apply {
				id, err := s.insertForm(ctx, q, w.ID, name, f, listIDs, by, now)
				if err != nil {
					return out, err
				}
				formIDs[f.Name] = id.String()
			}
		}
	}
	// Saved sections and campaign templates.
	for _, part := range []struct {
		kind  string
		items []TplItem
		into  *AgencyPlanPart
	}{{"section", c.Sections, &out.Sections}, {"template", c.Templates, &out.Templates}} {
		if len(part.items) == 0 || skip(part.kind+"s", authz.TemplatesEdit) {
			continue
		}
		for _, it := range part.items {
			kind := part.kind
			taken := func(n string) (bool, error) {
				return q.EmailTemplateNameTaken(ctx, store.EmailTemplateNameTakenParams{WorkspaceID: w.ID, Kind: kind, Name: n, ID: uuid.Nil})
			}
			name := it.Name
			clash, err := taken(name)
			if err != nil {
				return out, err
			}
			if clash {
				if name, err = renamedName(it.Name, taken); err != nil {
					return out, err
				}
				part.into.Renamed = append(part.into.Renamed, AgencyPlanRename{From: it.Name, To: name})
			} else {
				part.into.Added = append(part.into.Added, it.Name)
			}
			if apply {
				if _, err := insertTemplate(ctx, q, w.ID, kind, name, it, by, now); err != nil {
					return out, err
				}
			}
		}
	}
	// SMS texts wait for the SMS channel (X21): they're kept with the
	// template and listed, nothing is created yet.
	for _, t := range c.SMSTexts {
		out.SMSTexts.Added = append(out.SMSTexts.Added, t.Name)
	}
	// Automations, as drafts, pointed at the new lists.
	if len(c.Automations) > 0 && !skip("automations", authz.AutomationsEdit) {
		existing, err := q.AgencyCaptureLists(ctx, w.ID)
		if err != nil {
			return out, err
		}
		for _, l := range existing {
			if _, ok := listIDs[l.Name]; !ok {
				listIDs[l.Name] = l.ID.String()
			}
		}
		for _, a := range c.Automations {
			taken := func(n string) (bool, error) {
				return q.AgencyAutomationNameTaken(ctx, store.AgencyAutomationNameTakenParams{WorkspaceID: w.ID, Name: n})
			}
			name := a.Name
			clash, err := taken(name)
			if err != nil {
				return out, err
			}
			if clash {
				if name, err = renamedName(a.Name, taken); err != nil {
					return out, err
				}
				out.Automations.Renamed = append(out.Automations.Renamed, AgencyPlanRename{From: a.Name, To: name})
			} else {
				out.Automations.Added = append(out.Automations.Added, a.Name)
			}
			if apply {
				if _, err := s.insertAutomation(ctx, q, w.ID, name, a, listIDs, formIDs, by, "copied from the "+tpl.Name+" workspace template"); err != nil {
					return out, err
				}
			}
		}
	}
	// Custom roles; the client roles come with every client.
	if len(c.Roles) > 0 {
		for _, r := range c.Roles {
			if r.Permissions == nil {
				continue
			}
			if !perms.Has(authz.TeamManage) {
				out.Skipped = append(out.Skipped, AgencyPlanSkip{Part: "roles", Permission: string(authz.TeamManage)})
				break
			}
			if missing := perms.Missing(authz.ParseSet(r.Permissions)); len(missing) > 0 {
				out.Skipped = append(out.Skipped, AgencyPlanSkip{Part: "role:" + r.Name, Permission: string(missing[0])})
				continue
			}
			taken := func(n string) (bool, error) {
				return q.RolesNameTaken(ctx, store.RolesNameTakenParams{WorkspaceID: w.ID, Name: n, ExcludeID: uuid.Nil})
			}
			name := r.Name
			clash, err := taken(name)
			if err != nil {
				return out, err
			}
			if clash {
				if name, err = renamedName(r.Name, taken); err != nil {
					return out, err
				}
				out.Roles.Renamed = append(out.Roles.Renamed, AgencyPlanRename{From: r.Name, To: name})
			} else {
				out.Roles.Added = append(out.Roles.Added, r.Name)
			}
			if apply {
				if _, err := q.RolesCreate(ctx, store.RolesCreateParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: w.ID, Key: roleKey(name), Name: name,
					Description: r.Description, Permissions: authz.ParseSet(r.Permissions).Strings(), CreatedBy: by}); err != nil {
					return out, err
				}
			}
		}
	}
	if w.Kind == "client" {
		for _, r := range authz.ClientRoles {
			if _, err := q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: w.ID, Key: r.Key}); isNoRows(err) {
				out.ClientRoles = "added"
			} else if err != nil {
				return out, err
			}
		}
		if apply && out.ClientRoles == "added" {
			if err := s.ensureClientRoles(ctx, q, w.ID); err != nil {
				return out, err
			}
		}
	}
	// The approval policy's client sign-off: turned on, never off.
	if c.ClientSignoff && w.Kind == "client" {
		pol := approvals.ParsePolicy(ws.ApprovalPolicy)
		switch {
		case pol.ClientSignoff.Enabled:
			out.ClientSignoff = "already"
		case !perms.Has(authz.WorkspaceManage):
			out.ClientSignoff = "skipped"
			out.Skipped = append(out.Skipped, AgencyPlanSkip{Part: "policy", Permission: string(authz.WorkspaceManage)})
		default:
			out.ClientSignoff = "on"
			if apply {
				raw, err := approvals.EnableClientSignoff(ws.ApprovalPolicy, true)
				if err != nil {
					return out, err
				}
				if _, err := q.UpdateWorkspaceApprovalPolicy(ctx, store.UpdateWorkspaceApprovalPolicyParams{ID: w.ID, ApprovalPolicy: raw}); err != nil {
					return out, err
				}
			}
		}
	}
	// Branding without logo and colours, and the sender defaults: only the
	// keys the client doesn't have yet.
	if len(c.Branding) > 0 || c.Footer != "" {
		if !skip("branding", authz.FormsEdit) {
			out.Branding = true
			if apply {
				patch := map[string]any{}
				for k, v := range c.Branding {
					patch[k] = v
				}
				if c.Footer != "" {
					patch["footer_address"] = c.Footer
				}
				raw, _ := json.Marshal(patch)
				if err := q.AgencyAddBrandingKeys(ctx, store.AgencyAddBrandingKeysParams{ID: w.ID, Patch: raw}); err != nil {
					return out, err
				}
			}
		}
	}
	if len(c.SenderNames) > 0 && !skip("senders", authz.DomainsManage) {
		out.Senders = true
		if apply {
			raw, _ := json.Marshal(map[string]any{"from_names": c.SenderNames})
			if err := q.AgencyAddSendingDefaults(ctx, store.AgencyAddSendingDefaultsParams{ID: w.ID, Patch: raw}); err != nil {
				return out, err
			}
		}
	}
	return out, nil
}

// roleKey is a custom role's key from its name.
func roleKey(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	k := strings.Trim(b.String(), "-")
	if k == "" {
		k = "role"
	}
	if len(k) > 50 {
		k = k[:50]
	}
	return k + "-" + hex.EncodeToString(randBytes(2))
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// insertTemplate saves a section or campaign template in a workspace.
func insertTemplate(ctx context.Context, q *store.Queries, wsID uuid.UUID, kind, name string, it TplItem, by *uuid.UUID, now time.Time) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	thumb := it.Thumb
	if len(thumb) == 0 {
		thumb = json.RawMessage(`[]`)
	}
	html := it.HTML
	if kind == "section" {
		html = ""
	}
	return id, q.AgencyInsertTemplate(ctx, store.AgencyInsertTemplateParams{ID: id, WorkspaceID: wsID, Kind: kind, Name: name, Description: it.Description,
		Mjml: it.MJML, Html: html, Thumb: thumb, SizeBytes: int32(len(html)), CreatedBy: by, At: now})
}

// insertAutomation creates a draft flow with its emails from a template.
func (s *Service) insertAutomation(ctx context.Context, q *store.Queries, wsID uuid.UUID, name string, a TplAutomation, lists, forms map[string]string, by *uuid.UUID, summary string) (uuid.UUID, error) {
	g, err := automations.ParseGraph(a.Graph)
	if err != nil {
		g = automations.AutomationGraph{Nodes: []automations.AutomationNode{}, Edges: []automations.AutomationEdge{}}
	}
	// The graph names its emails ("email:e1"); keep them before resolving.
	refs := map[int]string{}
	for i, n := range g.Nodes {
		if e := n.Data.Email; e != nil {
			refs[i] = strings.TrimPrefix(e.EmailID, "email:")
		}
	}
	resolve(&g, lists, forms, nil)
	id := uuid.Must(uuid.NewV7())
	trig := json.RawMessage(`{}`)
	if t := g.Trigger(); t != nil && t.Data.Trigger != nil {
		trig, _ = json.Marshal(t.Data.Trigger)
	}
	if _, err := q.CreateAutomation(ctx, store.CreateAutomationParams{ID: id, WorkspaceID: wsID, Name: name, Description: a.Description, TemplateKey: "", Trigger: trig, CreatedBy: by}); err != nil {
		return id, err
	}
	emails := map[string]string{}
	for _, e := range a.Emails {
		eid := uuid.Must(uuid.NewV7())
		emails[e.Ref] = eid.String()
		if err := q.CreateAutomationEmail(ctx, store.CreateAutomationEmailParams{ID: eid, WorkspaceID: wsID, AutomationID: &id, Name: e.Name, CreatedBy: by}); err != nil {
			return id, err
		}
		if err := q.InsertAutomationEmailVersion(ctx, store.InsertAutomationEmailVersionParams{EmailID: eid, Version: 1, Subject: e.Subject, Preheader: e.Preheader,
			Mjml: e.MJML, Html: e.HTML, PlainText: e.Text, CreatedBy: by}); err != nil {
			return id, err
		}
	}
	for i, ref := range refs {
		g.Nodes[i].Data.Email.EmailID = emails[ref]
	}
	if err := q.InsertAutomationVersion(ctx, store.InsertAutomationVersionParams{AutomationID: id, Version: 1, Graph: g.MustJSON(), CreatedBy: by, Summary: summary}); err != nil {
		return id, err
	}
	return id, nil
}

// insertForm creates a paused signup form from a template.
func (s *Service) insertForm(ctx context.Context, q *store.Queries, wsID uuid.UUID, name string, f TplForm, lists map[string]string, by *uuid.UUID, now time.Time) (uuid.UUID, error) {
	ids := []uuid.UUID{}
	for _, n := range f.Lists {
		if v, ok := lists[n]; ok {
			ids = append(ids, uuid.MustParse(v))
		}
	}
	base := forms.Slugify(name, 50)
	if base == "" {
		base = "form"
	}
	slug := base
	for i := 2; ; i++ {
		taken, err := q.FormSlugTaken(ctx, store.FormSlugTakenParams{WorkspaceID: wsID, Slug: slug, ID: uuid.Nil})
		if err != nil {
			return uuid.Nil, err
		}
		if !taken && slug != "subscribe" && slug != "challenge" {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	pid := ""
	for range 8 {
		cand := "frm_" + hex.EncodeToString(randBytes(3))
		taken, err := q.FormPublicIDTaken(ctx, store.FormPublicIDTakenParams{WorkspaceID: wsID, PublicID: cand})
		if err != nil {
			return uuid.Nil, err
		}
		if !taken {
			pid = cand
			break
		}
	}
	kind := f.Kind
	if kind == "" {
		kind = forms.KindEmbedded
	}
	orEmpty := func(raw json.RawMessage, def string) json.RawMessage {
		if len(raw) == 0 {
			return json.RawMessage(def)
		}
		return raw
	}
	id := uuid.Must(uuid.NewV7())
	_, err := q.FormInsert(ctx, store.FormInsertParams{ID: id, WorkspaceID: wsID, Name: name, Kind: kind, Slug: slug, PublicID: pid,
		ListIds: ids, DefaultListIds: ids, ListChoice: false, DoubleOptIn: f.DoubleOptIn, Fields: orEmpty(f.Fields, `[]`), Content: orEmpty(f.Content, `{}`),
		SpamProtection: orEmpty(f.SpamProtection, `{}`), ConsentText: f.ConsentText, ConsentVersion: 1, AllowedOrigins: []string{}, AfterSubmit: "message",
		RedirectUrl: "", Status: "paused", CreatedBy: by, At: now})
	return id, err
}
