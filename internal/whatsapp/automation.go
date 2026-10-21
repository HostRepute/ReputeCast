package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Send WhatsApp step (brief §3.9, §5.8): an approved template (works
// any time), or free text, sent only inside the 24-hour customer service
// window and otherwise skipped or replaced by a chosen template. It sends
// only to contacts with WhatsApp consent on the step's list; the rule
// can't be removed. Quiet hours wait for the window (default) or skip.

// Step sends the "Send WhatsApp" steps (automations.ChannelStep).
type Step struct{ S *Service }

// Permission is what publishing a flow with a WhatsApp step needs.
func (Step) Permission() authz.Permission { return authz.WhatsAppSend }

// Validate checks the step beyond the engine's own checks.
func (Step) Validate(n automations.AutomationNode) []automations.ChannelIssue {
	st := n.Data.WhatsApp
	out := []automations.ChannelIssue{}
	if st == nil {
		return out
	}
	switch st.Mode {
	case "text":
		if strings.TrimSpace(st.Text) == "" {
			out = append(out, automations.ChannelIssue{Code: "missing_text", Message: "Write the message, or pick an approved template."})
		}
		if st.OutsideWindow == "template" && strings.TrimSpace(st.FallbackTemplate) == "" {
			out = append(out, automations.ChannelIssue{Code: "missing_template", Message: "Pick the template sent outside the 24-hour window."})
		}
	default:
		if strings.TrimSpace(st.Template) == "" {
			out = append(out, automations.ChannelIssue{Code: "missing_template", Message: "Pick an approved template."})
		}
	}
	if _, err := uuid.Parse(st.ConsentListID); err != nil {
		out = append(out, automations.ChannelIssue{Code: "missing_consent_list", Message: "Pick the list whose WhatsApp consent the step needs."})
	}
	return out
}

// Condition answers nothing: WhatsApp has no branch conditions of its own.
func (Step) Condition(context.Context, pgx.Tx, automations.ChannelCheck) (bool, time.Time, error) {
	return false, time.Time{}, nil
}

// listOf reads a step's consent list.
func listOf(raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	return id, err == nil
}

// approvedByName finds an approved template by name (and language).
func approvedByName(ctx context.Context, q *store.Queries, accountID uuid.UUID, name, lang string) (store.WhatsappTemplate, bool, error) {
	t, err := q.WhatsAppTemplateByName(ctx, store.WhatsAppTemplateByNameParams{AccountID: accountID, Name: name, Language: lang})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return t, false, nil
	case err != nil:
		return t, false, err
	}
	return t, t.Status == "approved", nil
}

// varsOf turns a step's variables into a campaign's.
func varsOf(in []automations.AutomationWhatsAppVar) []WhatsAppVariable {
	out := make([]WhatsAppVariable, 0, len(in))
	for _, v := range in {
		out = append(out, WhatsAppVariable{Key: v.Key, Source: v.Source, Field: v.Field, Text: v.Text, Fallback: v.Fallback})
	}
	return out
}

// Send queues the step's message for the run's contact.
func (st Step) Send(ctx context.Context, tx pgx.Tx, in automations.ChannelSend) (automations.ChannelSent, error) {
	s := st.S
	step := in.Node.Data.WhatsApp
	if step == nil {
		return automations.ChannelSent{Skipped: "no_content"}, nil
	}
	q := store.New(tx)
	ws, err := q.GetWorkspace(ctx, in.WorkspaceID)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	a, err := s.account(ctx, q, &ws)
	if errors.Is(err, errNoAccount) {
		return automations.ChannelSent{Skipped: "whatsapp_off"}, nil
	}
	if err != nil {
		return automations.ChannelSent{}, err
	}
	var r recipient
	var attrs []byte
	if err := tx.QueryRow(ctx, `SELECT id, phone, phone_country, timezone, first_name, last_name, email, attributes FROM contacts
		WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, in.ContactID, ws.ID).
		Scan(&r.ContactID, &r.Phone, &r.Country, &r.Zone, &r.FirstName, &r.LastName, &r.Email, &attrs); errors.Is(err, pgx.ErrNoRows) {
		return automations.ChannelSent{Skipped: "no_contact"}, nil
	} else if err != nil {
		return automations.ChannelSent{}, err
	}
	if r.Phone == "" {
		return automations.ChannelSent{Skipped: "no_phone"}, nil
	}
	_ = json.Unmarshal(attrs, &r.Attributes)
	listID, ok := listOf(step.ConsentListID)
	if !ok {
		// A step without its list is refused at publication: nobody
		// passes the consent rule.
		return automations.ChannelSent{Skipped: "no_consent"}, nil
	}
	r.ListID = listID
	var consent bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM channel_consents WHERE list_id = $1 AND contact_id = $2 AND channel = 'whatsapp' AND status = 'subscribed')`,
		listID, r.ContactID).Scan(&consent); err != nil {
		return automations.ChannelSent{}, err
	}
	if !consent {
		return automations.ChannelSent{Skipped: "no_consent"}, nil
	}
	if _, err := q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "whatsapp", Phone: r.Phone}); err == nil {
		return automations.ChannelSent{Skipped: "suppressed"}, nil
	}
	zone, src := phone.RecipientZone(r.Zone, r.Phone)
	r.Zone, r.ZoneSource = zone, string(src)
	rl := s.rules(ctx, q, &ws)
	loc, _ := zoneOf(r, &ws)
	if rl.Quiet && !in.Waited && !rl.Window.Inside(in.Now, loc) {
		if step.Outside == "skip" {
			return automations.ChannelSent{Skipped: "quiet_hours"}, nil
		}
		next := rl.Window.NextOpen(in.Now, loc)
		return automations.ChannelSent{Wait: &next}, nil
	}
	o := outgoing{id: uuid.Must(uuid.NewV7()), kind: "automation", automationID: &in.AutomationID, step: in.Node.ID, runID: &in.RunID, contactID: &r.ContactID,
		listID: &listID, recipient: r.Phone, country: r.Country, timeZone: r.Zone, zoneSource: r.ZoneSource, status: "queued", after: &in.Now}
	templateName := step.Template
	if step.Mode == "text" {
		open, _, _ := s.windowOpen(ctx, q, &ws, r.Phone, in.Now)
		switch {
		case open:
			o.body, templateName = strings.TrimSpace(step.Text), ""
		case step.OutsideWindow == "template" && step.FallbackTemplate != "":
			templateName = step.FallbackTemplate
		default:
			return automations.ChannelSent{Skipped: "window_closed"}, nil
		}
	}
	if templateName != "" {
		t, found, err := approvedByName(ctx, q, a.ID, templateName, step.Language)
		if err != nil {
			return automations.ChannelSent{}, err
		}
		if !found {
			return automations.ChannelSent{Skipped: "template_not_approved"}, nil
		}
		params := paramsFor(variablesOf(t.Body), varsOf(step.Variables), r)
		price := s.rateCard(ctx, q, &ws).price(r.Country, t.Category)
		o.template, o.language, o.category, o.params, o.price = t.Name, t.Language, t.Category, params, &price
		o.body = render(t.Body, params)
		// Meta's messaging limit: over it, the message waits for a slot.
		if lp, err := s.limitPlan(ctx, &ws, int(a.MessagingLimit), in.Now); err == nil && lp.Left <= 0 {
			at := lp.later(0, in.Now)
			if rl.Quiet {
				at = rl.Window.NextOpen(at, loc)
			}
			o.after, o.reason = &at, "messaging_limit"
		}
	}
	if a.Status == "token_rejected" {
		o.status, o.reason = "held", "token_rejected"
	}
	if err := insertMessage(ctx, q, &ws, a.ID, o, in.Now); err != nil {
		return automations.ChannelSent{}, err
	}
	s.kick(ctx)
	return automations.ChannelSent{MessageID: o.id}, nil
}

// WhatsAppEstimate is who a WhatsApp step or campaign on a list reaches
// and what it costs.
type WhatsAppEstimate struct {
	Consent   int64        `json:"consent"`
	Cost      WhatsAppCost `json:"cost"`
	Connected bool         `json:"connected"`
	Quiet     bool         `json:"quiet"`
	Window    string       `json:"window"`
	// Open is how many of them can get free text now (they wrote in the
	// last 24 hours).
	Open int64 `json:"open"`
}

// Estimate counts a list's people with WhatsApp consent and prices one
// template to them.
func (s *Service) Estimate(ctx context.Context, ws *store.Workspace, listID, templateID string) (WhatsAppEstimate, error) {
	out := WhatsAppEstimate{Cost: WhatsAppCost{ByCountry: []WhatsAppCostCountry{}}}
	_, err := s.account(ctx, s.Q, ws)
	out.Connected = err == nil
	rl := s.rules(ctx, s.Q, ws)
	out.Quiet, out.Window = rl.Quiet, rl.Window.String()
	lid, err := uuid.Parse(listID)
	if err != nil {
		return out, api.Invalid("list", "Pick a list of this workspace.")
	}
	aud, err := s.audience(ctx, ws, []uuid.UUID{lid}, nil)
	if err != nil {
		return out, err
	}
	out.Consent = int64(len(aud.list))
	category := "utility"
	if tid, err := uuid.Parse(templateID); err == nil {
		if t, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: tid, WorkspaceID: ws.ID}); err == nil {
			category = t.Category
		}
	}
	out.Cost = s.rateCard(ctx, s.Q, ws).estimate(category, aud.countries)
	if len(aud.list) > 0 {
		phones := make([]string, 0, len(aud.list))
		for _, r := range aud.list {
			phones = append(phones, r.Phone)
		}
		_ = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT phone) FROM whatsapp_inbound WHERE workspace_id = $1 AND phone = ANY($2)
			AND received_at > $3::timestamptz - interval '24 hours'`, ws.ID, phones, s.now()).Scan(&out.Open)
	}
	return out, nil
}

// WhatsAppStepEstimateInput is the Send WhatsApp step the panel shows.
type WhatsAppStepEstimateInput struct {
	ConsentListID string                              `json:"consentListId,omitempty" maxLength:"40"`
	Mode          string                              `json:"mode,omitempty" enum:"template,text,"`
	Template      string                              `json:"template,omitempty" maxLength:"512"`
	Language      string                              `json:"language,omitempty" maxLength:"16"`
	Variables     []automations.AutomationWhatsAppVar `json:"variables,omitempty" maxItems:"20"`
	// FallbackTemplate is the template free text falls back to outside the window.
	FallbackTemplate string `json:"fallbackTemplate,omitempty" maxLength:"512"`
}

// WhatsAppWindowExample is someone the step's free text reaches now, and
// until when.
type WhatsAppWindowExample struct {
	Name  string    `json:"name"`
	Until time.Time `json:"until"`
}

// WhatsAppStepEstimate is what the Send WhatsApp step's panel and its node
// on the canvas show (AutomationChannels): the account, the templates to
// pick, the fields variables take, who the step reaches on its list, what
// a template to them costs, a preview as one of them, and someone inside
// the customer service window.
type WhatsAppStepEstimate struct {
	Connected bool               `json:"connected"`
	Business  *WhatsAppBusiness  `json:"business,omitempty"`
	Templates []WhatsAppTemplate `json:"templates"`
	Fields    []WhatsAppField    `json:"fields"`
	// Consent counts the list's people with WhatsApp consent, a phone and
	// no suppression; Open those who wrote in the last 24 hours.
	Consent int64        `json:"consent"`
	Open    int64        `json:"open"`
	Cost    WhatsAppCost `json:"cost"`
	Quiet   bool         `json:"quiet"`
	Window  string       `json:"window"`
	// Preview is the step's template as the list's most recent writer reads it.
	Preview       *WhatsAppPreview       `json:"preview,omitempty"`
	WindowExample *WhatsAppWindowExample `json:"windowExample,omitempty"`
}

// StepEstimate answers the Send WhatsApp step's panel. A list or template
// that isn't picked yet leaves its parts empty.
func (s *Service) StepEstimate(ctx context.Context, ws *store.Workspace, in WhatsAppStepEstimateInput) (WhatsAppStepEstimate, error) {
	out := WhatsAppStepEstimate{Templates: []WhatsAppTemplate{}, Fields: s.fields(ctx, ws), Cost: WhatsAppCost{ByCountry: []WhatsAppCostCountry{}}}
	rl := s.rules(ctx, s.Q, ws)
	out.Quiet, out.Window = rl.Quiet, rl.Window.String()
	rc := s.rateCard(ctx, s.Q, ws)
	out.Cost.RatesAsOf = rc.asOf
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Connected = true
	out.Business = &WhatsAppBusiness{DisplayName: a.DisplayName, Number: phone.Display(a.PhoneNumber)}
	ts, err := s.Q.WhatsAppTemplates(ctx, a.ID)
	if err != nil {
		return out, err
	}
	for _, t := range ts {
		out.Templates = append(out.Templates, templateView(t))
	}
	listID, ok := listOf(in.ConsentListID)
	if !ok {
		return out, nil
	}
	aud, err := s.audience(ctx, ws, []uuid.UUID{listID}, nil)
	if err != nil {
		return out, err
	}
	out.Consent = int64(len(aud.list))
	name := in.Template
	if in.Mode == "text" {
		name = in.FallbackTemplate
	}
	var tpl *store.WhatsappTemplate
	if name != "" {
		if t, found, err := approvedByName(ctx, s.Q, a.ID, name, in.Language); err != nil {
			return out, err
		} else if found {
			tpl = &t
		}
	}
	category := "utility"
	if tpl != nil {
		category = tpl.Category
	}
	out.Cost = rc.estimate(category, aud.countries)
	if len(aud.list) == 0 {
		return out, nil
	}
	phones := make([]string, 0, len(aud.list))
	for _, r := range aud.list {
		phones = append(phones, r.Phone)
	}
	now := s.now()
	_ = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT phone) FROM whatsapp_inbound WHERE workspace_id = $1 AND phone = ANY($2)
		AND received_at > $3::timestamptz - interval '24 hours'`, ws.ID, phones, now).Scan(&out.Open)
	// The most recent writer: the preview reads as them, and they show what
	// the window means.
	var ph string
	var at time.Time
	if err := s.Pool.QueryRow(ctx, `SELECT phone, received_at FROM whatsapp_inbound WHERE workspace_id = $1 AND phone = ANY($2)
		ORDER BY received_at DESC LIMIT 1`, ws.ID, phones).Scan(&ph, &at); err == nil {
		for _, r := range aud.list {
			if r.Phone == ph {
				who := fullName(r.FirstName, r.LastName)
				if who == "" {
					who = r.Email
				}
				out.WindowExample = &WhatsAppWindowExample{Name: who, Until: at.Add(24 * time.Hour)}
				break
			}
		}
	}
	if tpl != nil && in.Mode != "text" {
		out.Preview = s.preview(ctx, ws, *tpl, varsOf(in.Variables), aud, uuid.Nil)
	}
	return out, nil
}

// WhatsAppStepStats is one WhatsApp step's last 30 days.
type WhatsAppStepStats struct {
	NodeID    string           `json:"nodeId"`
	Sent      int64            `json:"sent"`
	Delivered int64            `json:"delivered"`
	Read      int64            `json:"read"`
	Failed    int64            `json:"failed"`
	Waiting   int64            `json:"waiting"`
	Skipped   map[string]int64 `json:"skipped"`
	Cost      float64          `json:"cost"`
}

// WhatsAppStepStatsList is a flow's WhatsApp steps.
type WhatsAppStepStatsList struct {
	Steps []WhatsAppStepStats `json:"steps"`
}

// StepStats are a flow's WhatsApp steps over 30 days. Skips come from the
// engine's run log (no consent, no phone, the window closed…).
func (s *Service) StepStats(ctx context.Context, ws *store.Workspace, automationID uuid.UUID) (WhatsAppStepStatsList, error) {
	out := WhatsAppStepStatsList{Steps: []WhatsAppStepStats{}}
	rows, err := s.Pool.Query(ctx, `SELECT m.automation_step, count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'delivered'),
			count(*) FILTER (WHERE w.read_at IS NOT NULL), count(*) FILTER (WHERE m.status IN ('failed','undelivered')),
			count(*) FILTER (WHERE m.status IN ('queued','held','sending')), coalesce(sum(m.cost), 0)
		FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id
		WHERE m.workspace_id = $1 AND m.automation_id = $2 AND m.channel = 'whatsapp' AND m.created_at > $3::timestamptz - interval '30 days'
		GROUP BY 1 ORDER BY 1`, ws.ID, automationID, s.now())
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		st := WhatsAppStepStats{Skipped: map[string]int64{}}
		if err := rows.Scan(&st.NodeID, &st.Sent, &st.Delivered, &st.Read, &st.Failed, &st.Waiting, &st.Cost); err != nil {
			return out, err
		}
		st.Cost = float64(int64(st.Cost*100+0.5)) / 100
		out.Steps = append(out.Steps, st)
	}
	return out, rows.Err()
}
