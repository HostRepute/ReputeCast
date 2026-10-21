package texting

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// The "Send SMS" automation step (brief §3.6, §5.8): the same message editor
// and counter as campaigns; "Outside quiet hours: wait for the window"
// (default) or skip; the list whose SMS consent it needs (a rule that can't
// be removed); "Skip if the contact got a text in the last N hours"; and
// branch conditions: has SMS consent, clicked the text's link, replied.

// SmsStep sends "Send SMS" steps (implements automations.ChannelStep).
type SmsStep struct{ S *Service }

var _ automations.ChannelStep = SmsStep{}

// Permission implements automations.ChannelStep: publishing a flow with an
// SMS step needs sms:send.
func (SmsStep) Permission() authz.Permission { return authz.SMSSend }

var _ automations.ChannelCoster = SmsStep{}

// CostSince implements automations.ChannelCoster: what each flow's texts
// cost since a time, for the automations list.
func (st SmsStep) CostSince(ctx context.Context, workspaceID uuid.UUID, automationIDs []uuid.UUID, since time.Time) (map[uuid.UUID]float64, error) {
	out := map[uuid.UUID]float64{}
	rows, err := st.S.Pool.Query(ctx, `SELECT automation_id, coalesce(sum(cost), 0)::float8 FROM channel_messages
		WHERE workspace_id = $1 AND automation_id = ANY($2) AND channel = 'sms' AND created_at >= $3
		  AND status IN ('sending','sent','delivered','undelivered','failed')
		GROUP BY automation_id`, workspaceID, automationIDs, since)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var cost float64
		if err := rows.Scan(&id, &cost); err != nil {
			return out, err
		}
		out[id] = round2(cost)
	}
	return out, rows.Err()
}

var _ automations.ChannelPublishChecker = SmsStep{}

// PublishChecks implements automations.ChannelPublishChecker (brief §6):
// quiet hours, and the senders' registration in the countries of the
// contacts with SMS consent on the steps' lists. A country whose sender
// can't send yet is a warning: the steps skip the texts there, as Send does.
func (st SmsStep) PublishChecks(ctx context.Context, workspaceID uuid.UUID, nodes []automations.AutomationNode) ([]automations.AutomationPublishCheck, error) {
	s := st.S
	ws, err := s.Q.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	g, err := s.graph(ctx, s.Q, &ws)
	if err != nil {
		return nil, err
	}
	out := []automations.AutomationPublishCheck{}
	steps, skips := 0, 0
	lists := []uuid.UUID{}
	for _, n := range nodes {
		cfg := n.Data.SMS
		if cfg == nil {
			continue
		}
		steps++
		if cfg.Outside == "skip" {
			skips++
		}
		if id, err := uuid.Parse(cfg.ConsentListID); err == nil && !slices.Contains(lists, id) {
			lists = append(lists, id)
		}
	}
	if steps == 0 {
		return out, nil
	}
	quiet := "Quiet hours: texts go out " + g.rules.Window.String() + " in each contact’s time zone; "
	switch skips {
	case 0:
		quiet += "outside it they wait"
	case steps:
		quiet += "outside it they’re skipped"
	default:
		quiet += fmt.Sprintf("outside it, %d of %d SMS steps skip the text", skips, steps)
	}
	out = append(out, automations.AutomationPublishCheck{Status: "ok", Label: quiet})
	if len(g.senders) == 0 {
		return append(out, automations.AutomationPublishCheck{Status: "warn", Label: "No SMS sender yet: the steps skip their texts until one is set up"}), nil
	}
	if len(lists) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.phone_country, count(DISTINCT c.id) FROM channel_consents cc JOIN contacts c ON c.id = cc.contact_id
		WHERE c.workspace_id = $1 AND cc.channel = 'sms' AND cc.status = 'subscribed' AND cc.list_id = ANY($2) AND c.phone <> '' AND c.phone_country <> ''
		  AND NOT EXISTS (SELECT 1 FROM channel_suppressions x WHERE x.workspace_id = c.workspace_id AND x.channel = 'sms' AND x.phone = c.phone AND x.removed_at IS NULL)
		GROUP BY 1 ORDER BY 1`, workspaceID, lists)
	if err != nil {
		return nil, err
	}
	type gap struct {
		sender, hold string
		countries    []string
		n            int64
	}
	var gaps []*gap
	var sending []string
	for rows.Next() {
		var country string
		var n int64
		if err := rows.Scan(&country, &n); err != nil {
			rows.Close()
			return nil, err
		}
		d := g.decide(country)
		if d.Sender != nil {
			if l := senderLabel(*d.Sender); !slices.Contains(sending, l) {
				sending = append(sending, l)
			}
			continue
		}
		sender := ""
		if d.Route != nil {
			if sn, ok := g.senders[d.Route.SenderID]; ok {
				sender = senderLabel(sn)
			}
		}
		i := slices.IndexFunc(gaps, func(x *gap) bool { return x.sender == sender && x.hold == d.Hold })
		if i < 0 {
			gaps = append(gaps, &gap{sender: sender, hold: d.Hold})
			i = len(gaps) - 1
		}
		gaps[i].countries = append(gaps[i].countries, countryName(country))
		gaps[i].n += n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, gp := range gaps {
		where, who := joinAnd(gp.countries), plural(gp.n, "contact there is skipped", "contacts there are skipped")
		var label string
		switch gp.hold {
		case "no_route":
			label = "No route covers " + where + ": " + who
		case "pending":
			label = gp.sender + " isn’t approved yet in " + where + ": " + who + " until it is"
		case "rejected":
			label = gp.sender + "’s registration was rejected in " + where + ": " + who
		case "not_supported":
			label = gp.sender + " can’t send in " + where + ": " + who
		default:
			label = gp.sender + " isn’t registered in " + where + ": " + who
		}
		out = append(out, automations.AutomationPublishCheck{Status: "warn", Label: label})
	}
	if len(sending) > 0 && len(gaps) == 0 {
		out = append(out, automations.AutomationPublishCheck{Status: "ok", Label: "Every country’s sender can text · " + strings.Join(sending, ", ")})
	}
	return out, nil
}

// joinAnd lists names in English: "Ireland", "Ireland and Germany",
// "Ireland, Germany and Spain".
func joinAnd(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

// Validate implements automations.ChannelStep.
func (SmsStep) Validate(n automations.AutomationNode) []automations.ChannelIssue {
	out := []automations.ChannelIssue{}
	if n.Data.SMS == nil {
		return out
	}
	for _, l := range findLinks(n.Data.SMS.Body) {
		if l.Shortener {
			out = append(out, automations.ChannelIssue{Code: "public_shortener", Message: l.Host + " is a public link shortener: carriers filter texts with them. Paste the full address; it's shortened on your own domain."})
			break
		}
	}
	return out
}

// stepText is a step's text with the rules' locked parts.
func stepText(cfg *automations.AutomationSMSStep, r Rules) Text {
	t := Text{Prefix: r.BrandPrefix, Body: cfg.Body}
	if r.OptOutInSteps && !cfg.NoOptOut {
		t.OptOut = r.OptOutText
	}
	return t
}

// skipped records a step's text that wasn't sent, for the step's stats.
func (st SmsStep) skipped(ctx context.Context, q *store.Queries, in automations.ChannelSend, ph, country, reason string) (automations.ChannelSent, error) {
	id := uuid.Must(uuid.NewV7())
	auto, run, contact := in.AutomationID, in.RunID, in.ContactID
	err := q.ChanMessageInsert(ctx, store.ChanMessageInsertParams{ID: id, WorkspaceID: in.WorkspaceID, Channel: "sms", Kind: "automation", AutomationID: &auto,
		AutomationStep: in.Node.ID, RunID: &run, ContactID: &contact, Recipient: ph, Country: country, Status: "skipped", Reason: reason, Units: 0, Encoding: "gsm7",
		CreatedAt: in.Now})
	return automations.ChannelSent{Skipped: reason}, err
}

// Send implements automations.ChannelStep: consent, the number, the
// suppression list and "Skip if" first; then quiet hours (wait or skip);
// then the route; then the text is queued.
func (st SmsStep) Send(ctx context.Context, tx pgx.Tx, in automations.ChannelSend) (automations.ChannelSent, error) {
	s := st.S
	cfg := in.Node.Data.SMS
	q := store.New(tx)
	if cfg == nil || strings.TrimSpace(cfg.Body) == "" {
		return automations.ChannelSent{Skipped: "no_text"}, nil
	}
	ws, err := q.GetWorkspace(ctx, in.WorkspaceID)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	var c phoneRow
	if err := tx.QueryRow(ctx, `SELECT id, phone, phone_country, phone_line_type, phone_line_source, timezone, first_name, last_name, email
		FROM contacts WHERE id = $1 AND workspace_id = $2`, in.ContactID, in.WorkspaceID).
		Scan(&c.ID, &c.Phone, &c.Country, &c.LineType, &c.LineSource, &c.Zone, &c.FirstName, &c.LastName, &c.Email); err != nil {
		return automations.ChannelSent{}, err
	}
	listID, err := uuid.Parse(cfg.ConsentListID)
	if err != nil {
		return st.skipped(ctx, q, in, c.Phone, c.Country, "no_consent")
	}
	// Consent is checked again when the step sends: a STOP while waiting
	// for the window counts.
	var status string
	_ = tx.QueryRow(ctx, `SELECT status FROM channel_consents WHERE list_id = $1 AND contact_id = $2 AND channel = 'sms'`, listID, c.ID).Scan(&status)
	if status != "subscribed" {
		return st.skipped(ctx, q, in, c.Phone, c.Country, "no_consent")
	}
	if c.Phone == "" {
		return st.skipped(ctx, q, in, "", "", "no_phone")
	}
	var supp int64
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM channel_suppressions WHERE workspace_id = $1 AND channel = 'sms' AND phone = $2 AND removed_at IS NULL`, ws.ID, c.Phone).Scan(&supp)
	if supp > 0 {
		return st.skipped(ctx, q, in, c.Phone, c.Country, "suppressed")
	}
	if cfg.SkipRecentHours > 0 {
		var recent int64
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND recipient = $2 AND channel = 'sms' AND kind IN ('campaign','automation')
			AND sent_at > $3`, ws.ID, c.Phone, in.Now.Add(-time.Duration(cfg.SkipRecentHours)*time.Hour)).Scan(&recent)
		if recent > 0 {
			return st.skipped(ctx, q, in, c.Phone, c.Country, "recent")
		}
	}
	g, err := s.graph(ctx, q, &ws)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	zone, _ := phone.RecipientZone(c.Zone, c.Phone)
	loc := location(zone)
	w := g.rules.Window
	if !w.Inside(in.Now, loc) {
		if cfg.Outside == "skip" {
			return st.skipped(ctx, q, in, c.Phone, c.Country, "quiet_hours")
		}
		next := w.NextOpen(in.Now, loc)
		return automations.ChannelSent{Wait: &next}, nil
	}
	country := c.Country
	if country == "" {
		if n, err := phone.Parse(c.Phone, ""); err == nil {
			country = n.Country
		}
	}
	if d := g.decide(country); d.Sender == nil {
		reason := "no_route"
		if d.Hold != "no_route" {
			reason = "sender_" + d.Hold
		}
		return st.skipped(ctx, q, in, c.Phone, country, reason)
	}
	name := ""
	_ = tx.QueryRow(ctx, `SELECT name FROM automations WHERE id = $1`, in.AutomationID).Scan(&name)
	auto, run, contact := in.AutomationID, in.RunID, in.ContactID
	r := Recipient{ContactID: c.ID, ListID: listID, Phone: c.Phone, Country: country, FirstName: c.FirstName, LastName: c.LastName, Email: c.Email, Zone: c.Zone}
	m, err := s.queueTextTx(ctx, q, tx, &ws, g, queuedText{kind: "automation", automationID: &auto, runID: &run, step: in.Node.ID, contactID: &contact,
		listID: &listID, recipient: r, text: stepText(cfg, g.rules), name: name})
	if err != nil {
		return automations.ChannelSent{}, err
	}
	if m.Status == "skipped" {
		return automations.ChannelSent{Skipped: m.Reason}, nil
	}
	return automations.ChannelSent{MessageID: m.ID}, nil
}

// Condition implements automations.ChannelStep.
func (st SmsStep) Condition(ctx context.Context, tx pgx.Tx, in automations.ChannelCheck) (bool, time.Time, error) {
	c := in.Condition
	switch c.Kind {
	case automations.CondSMSConsent:
		var n int64
		err := tx.QueryRow(ctx, `SELECT count(*) FROM channel_consents cc JOIN contacts ct ON ct.id = cc.contact_id
			WHERE cc.contact_id = $1 AND cc.channel = 'sms' AND cc.status = 'subscribed' AND ct.phone <> '' AND ($2 = '' OR cc.list_id::text = $2)
			AND NOT EXISTS (SELECT 1 FROM channel_suppressions s WHERE s.workspace_id = ct.workspace_id AND s.channel = 'sms' AND s.phone = ct.phone AND s.removed_at IS NULL)`,
			in.ContactID, c.ListID).Scan(&n)
		return n > 0, time.Time{}, err
	case automations.CondSMSClicked, automations.CondSMSReplied:
		// The step's text for this run (any SMS step of the run when the
		// condition names none).
		var msgID uuid.UUID
		var sentAt time.Time
		err := tx.QueryRow(ctx, `SELECT id, coalesce(sent_at, created_at) FROM channel_messages WHERE run_id = $1 AND kind = 'automation'
			AND status NOT IN ('skipped','cancelled') AND ($2 = '' OR automation_step = $2) ORDER BY created_at DESC LIMIT 1`, in.RunID, c.Step).Scan(&msgID, &sentAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, time.Time{}, nil
		}
		if err != nil {
			return false, time.Time{}, err
		}
		var n int64
		if c.Kind == automations.CondSMSClicked {
			err = tx.QueryRow(ctx, `SELECT count(*) FROM sms_link_clicks k JOIN sms_links l ON l.code = k.code WHERE k.message_id = $1 AND NOT k.machine
				AND ($2 = '' OR l.original ILIKE '%' || $2 || '%')`, msgID, c.URLContains).Scan(&n)
		} else {
			err = tx.QueryRow(ctx, `SELECT count(*) FROM sms_inbound WHERE message_id = $1 AND kind IN ('reply','help','yes')`, msgID).Scan(&n)
		}
		return n > 0, sentAt, err
	}
	return false, time.Time{}, nil
}

// SmsStepStats are a step's numbers over 30 days.
type SmsStepStats struct {
	NodeID    string  `json:"nodeId"`
	Sent      int64   `json:"sent"`
	Delivered int64   `json:"delivered"`
	Clicks    int64   `json:"clicks"`
	OptOuts   int64   `json:"optOuts"`
	Skipped   int64   `json:"skipped"`
	Cost      float64 `json:"cost"`
	Waiting   int64   `json:"waiting"`
	// SkippedBy counts the skipped texts by reason (no_consent,
	// sender_pending, quiet_hours…) and SkippedIn by country.
	SkippedBy map[string]int64 `json:"skippedBy"`
	SkippedIn map[string]int64 `json:"skippedIn"`
	FirstAt   *time.Time       `json:"firstAt,omitempty"`
	LastAt    *time.Time       `json:"lastAt,omitempty"`
}

// SmsStepStatsList is a flow's SMS steps' stats.
type SmsStepStatsList struct {
	Days  int            `json:"days"`
	Steps []SmsStepStats `json:"steps"`
}

// StepStats are a flow's SMS steps' numbers over 30 days.
func (s *Service) StepStats(ctx context.Context, ws *store.Workspace, automationID uuid.UUID) (SmsStepStatsList, error) {
	out := SmsStepStatsList{Days: 30, Steps: []SmsStepStats{}}
	var n int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM automations WHERE id = $1 AND workspace_id = $2`, automationID, ws.ID).Scan(&n); err != nil || n == 0 {
		return out, api.NotFound("Automation")
	}
	since := s.now().AddDate(0, 0, -30)
	rows, err := s.Pool.Query(ctx, `SELECT m.automation_step, count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'delivered'),
			count(*) FILTER (WHERE m.status = 'skipped'), count(*) FILTER (WHERE m.status IN ('queued','held')),
			coalesce(sum(m.cost) FILTER (WHERE m.status IN ('sending','sent','delivered','undelivered','failed')), 0)::float8,
			(SELECT count(DISTINCT k.contact_id) FROM sms_link_clicks k JOIN channel_messages x ON x.id = k.message_id
			 WHERE x.automation_id = $1 AND x.automation_step = m.automation_step AND NOT k.machine AND k.occurred_at >= $2),
			(SELECT count(*) FROM sms_inbound i JOIN channel_messages x ON x.id = i.message_id
			 WHERE x.automation_id = $1 AND x.automation_step = m.automation_step AND i.kind = 'opt_out' AND i.received_at >= $2),
			min(m.sent_at), max(m.sent_at)
		FROM channel_messages m WHERE m.automation_id = $1 AND m.created_at >= $2 GROUP BY m.automation_step ORDER BY m.automation_step`, automationID, since)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var st SmsStepStats
		if err := rows.Scan(&st.NodeID, &st.Sent, &st.Delivered, &st.Skipped, &st.Waiting, &st.Cost, &st.Clicks, &st.OptOuts, &st.FirstAt, &st.LastAt); err != nil {
			rows.Close()
			return out, err
		}
		st.Cost = round2(st.Cost)
		st.SkippedBy, st.SkippedIn = map[string]int64{}, map[string]int64{}
		out.Steps = append(out.Steps, st)
	}
	rows.Close()
	rows, err = s.Pool.Query(ctx, `SELECT automation_step, reason, country, count(*) FROM channel_messages WHERE automation_id = $1 AND created_at >= $2 AND status = 'skipped'
		GROUP BY 1, 2, 3`, automationID, since)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var node, reason, country string
		var k int64
		if rows.Scan(&node, &reason, &country, &k) != nil {
			continue
		}
		for i := range out.Steps {
			if out.Steps[i].NodeID == node {
				out.Steps[i].SkippedBy[reason] += k
				if country != "" && reason != "no_consent" {
					out.Steps[i].SkippedIn[country] += k
				}
			}
		}
	}
	return out, rows.Err()
}

// SmsStepEstimateInput asks what an SMS step would send.
type SmsStepEstimateInput struct {
	Body          string `json:"body" maxLength:"1000"`
	ConsentListID string `json:"consentListId,omitempty"`
	NoOptOut      bool   `json:"noOptOut,omitempty"`
}

// SmsStepEstimate is the step panel's counter and estimated cost.
type SmsStepEstimate struct {
	Prefix  string        `json:"prefix"`
	OptOut  string        `json:"optOut"`
	Counter SmsCounter    `json:"counter"`
	Links   []SmsLinkView `json:"links"`
	// Texts are the people with SMS consent on the list who'd get it now.
	Texts       int64         `json:"texts"`
	Costs       []CountryLine `json:"costs"`
	Cost        float64       `json:"cost"`
	Budget      SmsBudgetView `json:"budget"`
	Window      string        `json:"window"`
	Shorteners  []string      `json:"shorteners"`
	ShortDomain string        `json:"shortDomain"`
}

// StepEstimate counts an SMS step's text and estimates its cost for the
// people with SMS consent on its list.
func (s *Service) StepEstimate(ctx context.Context, ws *store.Workspace, in SmsStepEstimateInput) (SmsStepEstimate, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsStepEstimate{}, err
	}
	cfg := &automations.AutomationSMSStep{Body: in.Body, ConsentListID: in.ConsentListID, NoOptOut: in.NoOptOut}
	text := stepText(cfg, g.rules)
	domain := s.shortDomain(g.rules)
	m := measure(text, domain)
	out := SmsStepEstimate{Prefix: text.Prefix, OptOut: text.OptOut, Links: []SmsLinkView{}, Costs: []CountryLine{}, Window: g.rules.Window.String(),
		Shorteners: []string{}, ShortDomain: g.rules.ShortDomain}
	out.Counter = SmsCounter{Count: m.Count, Base: m.Base, Fields: []SmsField{}, Spread: []SmsSpread{}}
	for _, u := range m.Fields {
		out.Counter.Fields = append(out.Counter.Fields, SmsField{Key: u.Key, Fallback: u.Fallback, HasFallback: u.HasFallback})
	}
	for _, l := range m.Links {
		v := SmsLinkView{URL: l.URL, Host: l.Host, Shortener: l.Shortener, Length: len(l.URL)}
		if l.Shortener {
			out.Shorteners = append(out.Shorteners, l.Host)
		} else if g.rules.ShortDomain != "" {
			v.Length = shortLen(g.rules.ShortDomain)
		}
		out.Links = append(out.Links, v)
	}
	if in.ConsentListID != "" {
		if _, err := uuid.Parse(in.ConsentListID); err != nil {
			return out, api.Invalid("body.consentListId", "Pick a list of this workspace.")
		}
		f, err := s.funnel(ctx, ws, Audience{Lists: []string{in.ConsentListID}})
		if err != nil {
			return out, err
		}
		rv := &review{c: store.SmsCampaignRow{}, ws: ws, graph: g, text: text, measured: m, funnel: f}
		for _, r := range f.Recipients {
			d := g.decide(r.Country)
			p := planned{r: r, decision: d, segments: max(m.Count.Segments, 1)}
			if len(m.Fields) > 0 {
				body := render(withShortLinks(text.Body, domain, placeholder), r)
				p.segments = max(sms.CountText(Text{Prefix: text.Prefix, Body: body, OptOut: text.OptOut}.Full()).Segments, 1)
			}
			if d.Sender == nil {
				p.skip = d.Hold
			} else {
				p.cost = d.Price * float64(p.segments)
				rv.cost += p.cost
				rv.sent++
			}
			rv.plan = append(rv.plan, p)
		}
		out.Texts, out.Cost, out.Costs = rv.sent, round2(rv.cost), rv.byCountry()
	}
	spent, err := s.spentThisMonth(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Budget = SmsBudgetView{Month: monthStart(s.now(), location(ws.Timezone)), Budget: g.rules.Budget, AlertPct: g.rules.BudgetAlertPct, Spent: round2(spent),
		After: round2(spent + out.Cost)}
	return out, nil
}
