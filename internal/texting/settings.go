package texting

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsRouteView is a route in Routing & rules.
type SmsRouteView struct {
	ID        string   `json:"id"`
	Countries []string `json:"countries"`
	SenderID  string   `json:"senderId"`
	Sender    string   `json:"sender"`
	SenderKind string  `json:"senderKind"`
	Provider  string   `json:"provider"`
	Adapter   string   `json:"adapter"`
	// Registration is the sender's status in the route's countries.
	Registration SmsRegistrationView `json:"registration"`
	FallbackID   string              `json:"fallbackId,omitempty"`
	Fallback     string              `json:"fallback,omitempty"`
	FallbackProvider string          `json:"fallbackProvider,omitempty"`
	// OnLists counts the people on SMS lists in these countries.
	OnLists int64 `json:"onLists"`
	// Skipped is true when the sender can't text these countries yet.
	Skipped bool `json:"skipped"`
}

// SmsRouteGap is a country whose route can't send.
type SmsRouteGap struct {
	Countries []string   `json:"countries"`
	Sender    string     `json:"sender"`
	Status    string     `json:"status"`
	Since     *time.Time `json:"since,omitempty"`
	People    int64      `json:"people"`
	// Held counts the opt-in confirmations waiting for it.
	Held int64 `json:"held"`
}

// SmsSenderOption is a sender a route can use.
type SmsSenderOption struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Kind      string   `json:"kind"`
	Provider  string   `json:"provider"`
	Adapter   string   `json:"adapter"`
	Countries []string `json:"countries"`
	// Sendable are the countries its registration lets it text.
	Sendable []string `json:"sendable"`
}

// SmsZoneNow is the local time in a zone, for the quiet-hours preview.
type SmsZoneNow struct {
	Zone   string `json:"zone"`
	Local  string `json:"local"`
	Inside bool   `json:"inside"`
	// Opens is when texts go next ("09:00").
	Opens string `json:"opens"`
}

// SmsReplyView is a keyword reply with its length.
type SmsReplyView struct {
	Text  string    `json:"text"`
	Count sms.Count `json:"count"`
}

// SmsShortDomain is the short-link domain's state.
type SmsShortDomain struct {
	Domain     string     `json:"domain"`
	Status     string     `json:"status" enum:"none,pending,verified,failed"`
	Target     string     `json:"target"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
	Detail     string     `json:"detail,omitempty"`
}

// SmsMemberOption is a member who can get test texts.
type SmsMemberOption struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Color  string `json:"color"`
}

// SmsRules is Routing & rules.
type SmsRules struct {
	Routes   []SmsRouteView    `json:"routes"`
	Other    string            `json:"other" enum:"dont_send,route"`
	OtherRouteID string        `json:"otherRouteId,omitempty"`
	OtherPeople  int64         `json:"otherPeople"`
	Gaps     []SmsRouteGap     `json:"gaps"`
	Senders  []SmsSenderOption `json:"senders"`
	// Quiet hours.
	QuietFrom string       `json:"quietFrom"`
	QuietTo   string       `json:"quietTo"`
	Assumed   int64        `json:"assumed"`
	Zones     []SmsZoneNow `json:"zones"`
	PushWindow     bool    `json:"pushWindow"`
	WhatsAppWindow bool    `json:"whatsAppWindow"`
	// Keywords and replies.
	OptOutKeywords []string     `json:"optOutKeywords"`
	OptInKeywords  []string     `json:"optInKeywords"`
	HelpKeywords   []string     `json:"helpKeywords"`
	HelpReply      SmsReplyView `json:"helpReply"`
	OptOutReply    SmsReplyView `json:"optOutReply"`
	OptInReply     SmsReplyView `json:"optInReply"`
	KeywordsBy     string       `json:"keywordsBy" enum:"reputecast,provider"`
	// ProviderAnswers names the connected providers that can answer
	// keywords themselves.
	ProviderAnswers []string `json:"providerAnswers"`
	// Budget and guards.
	Budget    float64       `json:"budget"`
	AlertPct  int           `json:"alertPct"`
	Spent     float64       `json:"spent"`
	Month     time.Time     `json:"month"`
	BudgetSetBy string      `json:"budgetSetBy,omitempty"`
	BudgetSetAt *time.Time  `json:"budgetSetAt,omitempty"`
	Thresholds SmsThresholds `json:"thresholds"`
	// Test numbers.
	TestNumbers []SmsTestNumber   `json:"testNumbers"`
	TestsSent   int64             `json:"testsSent"`
	TestsCost   float64           `json:"testsCost"`
	Members     []SmsMemberOption `json:"members"`
	RetentionDays int             `json:"retentionDays"`
	// Message rules.
	BrandPrefix    string       `json:"brandPrefix"`
	OptOutText     string       `json:"optOutText"`
	OptOutInSteps  bool         `json:"optOutInSteps"`
	ConfirmationOn bool         `json:"confirmationOn"`
	Confirmation   SmsReplyView `json:"confirmation"`
	RequireYes     bool         `json:"requireYes"`
	ShortDomain    SmsShortDomain `json:"shortDomain"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
	CanManage   bool       `json:"canManage"`
	Helpers     []string   `json:"helpers"`
	Timezone    string     `json:"timezone"`
}

// demoZones are the zones the quiet-hours preview shows: the workspace's,
// then where its readers are.
func previewZones(ws *store.Workspace, countries map[string]int64) []string {
	zones := []string{}
	add := func(z string) {
		if z != "" && !slices.Contains(zones, z) && len(zones) < 3 {
			zones = append(zones, z)
		}
	}
	for _, c := range []string{"GB", "US", "DE"} {
		if countries[c] > 0 {
			if z, _ := phone.TimeZone(countryProbe(c)); z != "" {
				if c == "US" {
					z = "America/Chicago"
				}
				add(z)
			}
		}
	}
	add(ws.Timezone)
	return zones
}

// Rules builds Routing & rules.
func (s *Service) Rules(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SmsRules, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsRules{}, err
	}
	r := g.rules
	now := s.now()
	out := SmsRules{Routes: []SmsRouteView{}, Other: r.OtherCountries, Gaps: []SmsRouteGap{}, Senders: []SmsSenderOption{}, QuietFrom: channels.Clock(r.Window.From),
		QuietTo: channels.Clock(r.Window.To), Zones: []SmsZoneNow{}, PushWindow: r.PushWindow, WhatsAppWindow: r.WhatsAppWindow,
		OptOutKeywords: r.Keywords.OptOut, OptInKeywords: r.Keywords.OptIn, HelpKeywords: r.Keywords.Help,
		HelpReply: SmsReplyView{Text: r.HelpReply, Count: sms.CountText(r.HelpReply)}, OptOutReply: SmsReplyView{Text: r.OptOutReply, Count: sms.CountText(r.OptOutReply)},
		OptInReply: SmsReplyView{Text: r.OptInReply, Count: sms.CountText(r.OptInReply)}, KeywordsBy: r.KeywordsBy, ProviderAnswers: []string{},
		Budget: r.Budget, AlertPct: r.BudgetAlertPct, Month: monthStart(now, location(ws.Timezone)), BudgetSetAt: r.BudgetSetAt,
		Thresholds: SmsThresholds{Failed: r.PauseFailed, Filtered: r.PauseFiltered, OptOuts: r.PauseOptOut}, Members: []SmsMemberOption{},
		RetentionDays: r.RetentionDays, BrandPrefix: r.BrandPrefix, OptOutText: r.OptOutText, OptOutInSteps: r.OptOutInSteps, ConfirmationOn: r.ConfirmationOn,
		Confirmation: SmsReplyView{Text: r.Confirmation, Count: sms.CountText(r.Confirmation)}, RequireYes: r.RequireYes,
		ShortDomain: SmsShortDomain{Domain: r.ShortDomain, Status: r.ShortStatus, Target: firstNonEmpty(r.ShortTarget, instanceHost(s.URLs.Base())), CheckedAt: r.ShortCheckedAt,
			VerifiedAt: r.ShortVerifiedAt, Detail: r.ShortDetail},
		CanManage: p.Can(authz.ProvidersManage), Helpers: []string{}, Timezone: ws.Timezone}
	if !out.CanManage {
		out.Helpers = access.Helpers(ctx, s.Q, ws.ID, authz.ProvidersManage)
	}
	if !r.UpdatedAt.IsZero() {
		t := r.UpdatedAt
		out.UpdatedAt = &t
		if r.UpdatedBy != nil {
			if u, err := s.Q.GetUser(ctx, *r.UpdatedBy); err == nil {
				out.UpdatedBy = u.Name
			}
		}
	}
	if r.BudgetSetBy != nil {
		if u, err := s.Q.GetUser(ctx, *r.BudgetSetBy); err == nil {
			out.BudgetSetBy = u.Name
		}
	}
	if r.OtherRouteID != nil {
		out.OtherRouteID = r.OtherRouteID.String()
	}
	counts, err := s.smsListsCounts(ctx, ws)
	if err != nil {
		return out, err
	}
	routed := map[string]bool{}
	for _, rt := range g.routes {
		sn, ok := g.senders[rt.SenderID]
		if !ok {
			continue
		}
		for _, c := range rt.Countries {
			routed[c] = true
		}
		// A route splits where its sender can't text some countries yet.
		ok1, held := []string{}, []string{}
		for _, c := range rt.Countries {
			if g.reg(sn, c).CanSend() {
				ok1 = append(ok1, c)
			} else {
				held = append(held, c)
			}
		}
		for _, part := range [][]string{ok1, held} {
			if len(part) == 0 {
				continue
			}
			v := SmsRouteView{ID: rt.ID.String(), Countries: part, SenderID: sn.ID.String(), Sender: senderLabel(sn), SenderKind: sn.Kind, Provider: sn.ProviderName,
				Adapter: sn.ProviderAdapter, Registration: regView(g.reg(sn, part[0]))}
			v.Registration.Country = part[0]
			v.Skipped = !g.reg(sn, part[0]).CanSend()
			if rt.FallbackSenderID != nil {
				if fb, ok := g.senders[*rt.FallbackSenderID]; ok {
					v.FallbackID, v.Fallback, v.FallbackProvider = fb.ID.String(), senderLabel(fb), fb.ProviderName
				}
			}
			for _, c := range part {
				v.OnLists += counts[c]
			}
			out.Routes = append(out.Routes, v)
			if v.Skipped {
				gap := SmsRouteGap{Countries: part, Sender: senderLabel(sn), Status: v.Registration.Status, Since: v.Registration.Since, People: v.OnLists}
				_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND kind = 'confirmation' AND status = 'held' AND country = ANY($2::text[])`,
					ws.ID, part).Scan(&gap.Held)
				out.Gaps = append(out.Gaps, gap)
			}
		}
	}
	for c, n := range counts {
		if !routed[c] {
			out.OtherPeople += n
		}
	}
	for _, id := range g.order {
		sn := g.senders[id]
		o := SmsSenderOption{ID: sn.ID.String(), Label: senderLabel(sn), Kind: sn.Kind, Provider: sn.ProviderName, Adapter: sn.ProviderAdapter, Countries: sn.Countries,
			Sendable: []string{}}
		for _, c := range sn.Countries {
			if g.reg(sn, c).CanSend() {
				o.Sendable = append(o.Sendable, c)
			}
		}
		out.Senders = append(out.Senders, o)
	}
	if out.Assumed, err = s.assumedZones(ctx, ws); err != nil {
		return out, err
	}
	for _, z := range previewZones(ws, counts) {
		loc := location(z)
		zn := SmsZoneNow{Zone: z, Local: now.In(loc).Format("15:04"), Inside: r.Window.Inside(now, loc)}
		zn.Opens = r.Window.NextOpen(now, loc).In(loc).Format("15:04")
		out.Zones = append(out.Zones, zn)
	}
	for _, pr := range g.providers {
		if pr.Adapter == "twilio" {
			out.ProviderAnswers = append(out.ProviderAnswers, pr.Name)
		}
	}
	if out.Spent, err = s.spentThisMonth(ctx, ws); err != nil {
		return out, err
	}
	out.Spent = round2(out.Spent)
	if out.TestNumbers, err = s.testNumbers(ctx, ws, nil); err != nil {
		return out, err
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*), coalesce(sum(cost), 0)::float8 FROM channel_messages WHERE workspace_id = $1 AND kind = 'test'
		AND coalesce(sent_at, created_at) >= $2 AND status IN ('sending','sent','delivered','undelivered','failed')`, ws.ID, out.Month).Scan(&out.TestsSent, &out.TestsCost)
	out.TestsCost = round2(out.TestsCost)
	rows, err := s.Pool.Query(ctx, `SELECT u.id, u.name, coalesce(r.name, ''), coalesce(u.avatar_color, '') FROM workspace_members m JOIN users u ON u.id = m.user_id
		LEFT JOIN roles r ON r.id = m.role_id WHERE m.workspace_id = $1 ORDER BY u.name`, ws.ID)
	if err == nil {
		for rows.Next() {
			var m SmsMemberOption
			var id uuid.UUID
			if rows.Scan(&id, &m.Name, &m.Role, &m.Color) == nil {
				m.UserID = id.String()
				out.Members = append(out.Members, m)
			}
		}
		rows.Close()
	}
	return out, nil
}

// instanceHost is the instance's host name (reputecast.acme.io): the
// short-link domain's CNAME target.
func instanceHost(base string) string {
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// SmsRulesInput saves Routing & rules (routes apart).
type SmsRulesInput struct {
	QuietFrom      string   `json:"quietFrom" pattern:"^([01]\\d|2[0-3]):[0-5]\\d$"`
	QuietTo        string   `json:"quietTo" pattern:"^([01]\\d|2[0-3]):[0-5]\\d$"`
	PushWindow     bool     `json:"pushWindow"`
	WhatsAppWindow bool     `json:"whatsAppWindow"`
	OptOutKeywords []string `json:"optOutKeywords" maxItems:"30"`
	OptInKeywords  []string `json:"optInKeywords" maxItems:"30"`
	HelpKeywords   []string `json:"helpKeywords" maxItems:"30"`
	HelpReply      string   `json:"helpReply" maxLength:"640"`
	OptOutReply    string   `json:"optOutReply" maxLength:"640"`
	OptInReply     string   `json:"optInReply" maxLength:"640"`
	KeywordsBy     string   `json:"keywordsBy" enum:"reputecast,provider"`
	Budget         float64  `json:"budget" minimum:"0"`
	AlertPct       int      `json:"alertPct" minimum:"1" maximum:"100"`
	Thresholds     SmsThresholds `json:"thresholds"`
	RetentionDays  int      `json:"retentionDays" enum:"0,7,30,60,90"`
	BrandPrefix    string   `json:"brandPrefix" maxLength:"20"`
	OptOutText     string   `json:"optOutText" maxLength:"60"`
	OptOutInSteps  bool     `json:"optOutInSteps"`
	ConfirmationOn bool     `json:"confirmationOn"`
	Confirmation   string   `json:"confirmation" maxLength:"640"`
	RequireYes     bool     `json:"requireYes"`
}

func minuteOf(hhmm string) int {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return -1
	}
	return t.Hour()*60 + t.Minute()
}

// SaveRules saves the rules. The window can be narrowed, never widened past
// 08:00–21:00; STOP and HELP stay; replies can't be empty. Applies to texts
// not yet sent. Audited.
func (s *Service) SaveRules(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsRulesInput) (SmsRules, error) {
	before, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return SmsRules{}, err
	}
	w := channels.Window{From: minuteOf(in.QuietFrom), To: minuteOf(in.QuietTo)}
	if err := w.Validate(); err != nil {
		switch {
		case errors.Is(err, channels.ErrWindowTooEarly):
			return SmsRules{}, api.Invalid("body.quietFrom", "Texts can't go out before 08:00 local time.")
		case errors.Is(err, channels.ErrWindowTooLate):
			return SmsRules{}, api.Invalid("body.quietTo", "Texts can't go out after 21:00 local time. The window can be narrowed, never widened past 08:00–21:00.")
		}
		return SmsRules{}, api.Invalid("body.quietTo", "The window ends after it starts.")
	}
	kw := sms.Keywords{OptOut: in.OptOutKeywords, OptIn: in.OptInKeywords, Help: in.HelpKeywords}
	if err := kw.Validate(); err != nil {
		msg := map[error]string{sms.ErrKeywordEmpty: "A keyword can't be empty.", sms.ErrKeywordFormat: "A keyword is one word of letters and digits, up to 20.",
			sms.ErrKeywordDuplicate: "A keyword can only mean one thing.", sms.ErrKeywordRequired: "STOP and HELP can't be removed: carriers require them."}[err]
		return SmsRules{}, api.Invalid("body.optOutKeywords", firstNonEmpty(msg, err.Error()))
	}
	kw = kw.Normalized()
	for field, v := range map[string]string{"helpReply": in.HelpReply, "optOutReply": in.OptOutReply, "optInReply": in.OptInReply} {
		if strings.TrimSpace(v) == "" {
			return SmsRules{}, api.Invalid("body."+field, "A reply can't be empty.")
		}
	}
	if in.ConfirmationOn && strings.TrimSpace(in.Confirmation) == "" {
		return SmsRules{}, api.Invalid("body.confirmation", "Write the confirmation text, or turn it off.")
	}
	for _, th := range []float64{in.Thresholds.Failed, in.Thresholds.Filtered, in.Thresholds.OptOuts} {
		if th < 0.1 || th > 100 {
			return SmsRules{}, api.Invalid("body.thresholds", "A threshold is a rate from 0.1% to 100%.")
		}
	}
	if in.KeywordsBy == "provider" {
		provs, _ := s.Q.SmsProviders(ctx, ws.ID)
		ok := false
		for _, pr := range provs {
			if pr.Adapter == "twilio" {
				ok = true
			}
		}
		if !ok {
			return SmsRules{}, api.Invalid("body.keywordsBy", "None of your SMS providers answers keywords itself.")
		}
	}
	if in.RequireYes && !in.ConfirmationOn {
		return SmsRules{}, api.Invalid("body.requireYes", "Asking for YES needs the opt-in confirmation text.")
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := s.ensureRules(ctx, q, ws); err != nil {
			return err
		}
		if err := q.SmsSettingsSave(ctx, store.SmsSettingsSaveParams{BrandPrefix: strings.TrimSpace(in.BrandPrefix), OptOutText: strings.TrimSpace(in.OptOutText),
			OptOutInSteps: in.OptOutInSteps, ConfirmationOn: in.ConfirmationOn, ConfirmationText: strings.TrimSpace(in.Confirmation), RequireYes: in.RequireYes,
			QuietFrom: int32(w.From), QuietTo: int32(w.To), PushSameWindow: in.PushWindow, WhatsappSameWindow: in.WhatsAppWindow, OptOutKeywords: kw.OptOut,
			OptInKeywords: kw.OptIn, HelpKeywords: kw.Help, HelpReply: strings.TrimSpace(in.HelpReply), OptOutReply: strings.TrimSpace(in.OptOutReply),
			OptInReply: strings.TrimSpace(in.OptInReply), KeywordsBy: in.KeywordsBy, Budget: round2(in.Budget), BudgetAlertPct: int32(in.AlertPct),
			UpdatedBy: uuidPtr(p.UserID), At: now, PauseFailedPct: in.Thresholds.Failed, PauseFilteredPct: in.Thresholds.Filtered, PauseOptOutPct: in.Thresholds.OptOuts,
			OtherCountries: before.OtherCountries, OtherRouteID: before.OtherRouteID, RetentionDays: int32(in.RetentionDays), WorkspaceID: ws.ID}); err != nil {
			return err
		}
		changed := map[string]any{}
		was := map[string]any{}
		note := func(k string, a, b any) {
			if fmt.Sprint(a) != fmt.Sprint(b) {
				was[k], changed[k] = a, b
			}
		}
		note("quiet_hours", before.Window.String(), w.String())
		note("budget", before.Budget, round2(in.Budget))
		note("budget_alert_pct", before.BudgetAlertPct, in.AlertPct)
		note("brand_prefix", before.BrandPrefix, strings.TrimSpace(in.BrandPrefix))
		note("opt_out_text", before.OptOutText, strings.TrimSpace(in.OptOutText))
		note("keywords", before.Keywords, kw)
		note("keywords_by", before.KeywordsBy, in.KeywordsBy)
		note("help_reply", before.HelpReply, strings.TrimSpace(in.HelpReply))
		note("opt_out_reply", before.OptOutReply, strings.TrimSpace(in.OptOutReply))
		note("opt_in_reply", before.OptInReply, strings.TrimSpace(in.OptInReply))
		note("confirmation", before.Confirmation, strings.TrimSpace(in.Confirmation))
		note("confirmation_on", before.ConfirmationOn, in.ConfirmationOn)
		note("require_yes", before.RequireYes, in.RequireYes)
		note("thresholds", []float64{before.PauseFailed, before.PauseFiltered, before.PauseOptOut}, []float64{in.Thresholds.Failed, in.Thresholds.Filtered, in.Thresholds.OptOuts})
		note("retention_days", before.RetentionDays, in.RetentionDays)
		note("push_window", before.PushWindow, in.PushWindow)
		note("whatsapp_window", before.WhatsAppWindow, in.WhatsAppWindow)
		note("opt_out_in_steps", before.OptOutInSteps, in.OptOutInSteps)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_rules.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: was, After: changed})
	})
	if err != nil {
		return SmsRules{}, err
	}
	if in.Budget > before.Budget || (before.Budget > 0 && in.Budget == 0) {
		// A raised budget lets waiting SMS steps go.
		s.releaseHeld(ctx, ws.ID, nil, "budget")
	}
	return s.Rules(ctx, ws, p)
}

// SmsRoutesInput replaces the routes.
type SmsRoutesInput struct {
	Routes []SmsRouteInput `json:"routes" maxItems:"100"`
	// Other is "every other country": dont_send, or the route at OtherRoute.
	Other      string `json:"other" enum:"dont_send,route"`
	OtherRoute int    `json:"otherRoute,omitempty" minimum:"0"`
}

// SmsRouteInput is one route.
type SmsRouteInput struct {
	Countries  []string `json:"countries" minItems:"1" maxItems:"250"`
	SenderID   string   `json:"senderId" format:"uuid"`
	FallbackID string   `json:"fallbackId,omitempty"`
}

// SaveRoutes replaces the routes: a country in one route only; a fallback
// must be registered for the route's countries. Audited.
func (s *Service) SaveRoutes(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsRoutesInput) (SmsRules, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsRules{}, err
	}
	seen := map[string]bool{}
	type route struct {
		id        uuid.UUID
		countries []string
		sender    uuid.UUID
		fallback  *uuid.UUID
	}
	routes := []route{}
	for i, r := range in.Routes {
		field := fmt.Sprintf("body.routes[%d]", i)
		sid, err := uuid.Parse(r.SenderID)
		if err != nil {
			return SmsRules{}, api.Invalid(field+".senderId", "Pick a sender.")
		}
		sn, ok := g.senders[sid]
		if !ok {
			return SmsRules{}, api.Invalid(field+".senderId", "Pick a sender of this workspace.")
		}
		rt := route{id: uuid.Must(uuid.NewV7()), sender: sid}
		for _, c := range r.Countries {
			if !validCountry(c) {
				return SmsRules{}, api.Invalid(field+".countries", "Pick countries by their two-letter code.")
			}
			if seen[c] {
				return SmsRules{}, api.Invalid(field+".countries", countryName(c)+" is in two routes. A country takes one route.")
			}
			if sn.Kind == "sender_id" && slices.Contains(nanp, c) {
				return SmsRules{}, api.Invalid(field+".countries", "Sender IDs can't be used in the US or Canada.")
			}
			seen[c] = true
			rt.countries = append(rt.countries, c)
		}
		if r.FallbackID != "" {
			fid, err := uuid.Parse(r.FallbackID)
			if err != nil {
				return SmsRules{}, api.Invalid(field+".fallbackId", "Pick a fallback sender, or none.")
			}
			fb, ok := g.senders[fid]
			if !ok || fid == sid {
				return SmsRules{}, api.Invalid(field+".fallbackId", "Pick another sender as the fallback.")
			}
			for _, c := range rt.countries {
				if !g.reg(fb, c).CanSend() {
					return SmsRules{}, api.Invalid(field+".fallbackId", "A fallback must be registered for the same countries: "+senderLabel(fb)+" can't text "+countryName(c)+".")
				}
			}
			rt.fallback = &fid
		}
		routes = append(routes, rt)
	}
	var other *uuid.UUID
	if in.Other == "route" {
		if in.OtherRoute >= len(routes) {
			return SmsRules{}, api.Invalid("body.otherRoute", "Pick the route every other country takes.")
		}
		other = &routes[in.OtherRoute].id
	}
	now := s.now()
	beforeRoutes := []map[string]any{}
	for _, r := range g.routes {
		beforeRoutes = append(beforeRoutes, map[string]any{"countries": r.Countries, "sender": senderOf(g, r.SenderID)})
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := s.ensureRules(ctx, q, ws); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE sms_settings SET other_route_id = NULL WHERE workspace_id = $1`, ws.ID); err != nil {
			return err
		}
		if err := q.SmsRoutesDelete(ctx, ws.ID); err != nil {
			return err
		}
		after := []map[string]any{}
		for i, r := range routes {
			if err := q.SmsRouteInsert(ctx, store.SmsRouteInsertParams{ID: r.id, WorkspaceID: ws.ID, Position: int32(i), Countries: r.countries, SenderID: r.sender,
				FallbackSenderID: r.fallback, At: now}); err != nil {
				return err
			}
			m := map[string]any{"countries": r.countries, "sender": senderOf(g, r.sender)}
			if r.fallback != nil {
				m["fallback"] = senderOf(g, *r.fallback)
			}
			after = append(after, m)
		}
		if _, err := tx.Exec(ctx, `UPDATE sms_settings SET other_countries = $2, other_route_id = $3, updated_by = $4, updated_at = $5 WHERE workspace_id = $1`,
			ws.ID, in.Other, other, uuidPtr(p.UserID), now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_routes.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"routes": beforeRoutes, "other": g.rules.OtherCountries}, After: map[string]any{"routes": after, "other": in.Other}})
	})
	if err != nil {
		return SmsRules{}, err
	}
	s.releasePending(ctx, ws)
	return s.Rules(ctx, ws, p)
}

func senderOf(g *graph, id uuid.UUID) string {
	if sn, ok := g.senders[id]; ok {
		return sn.ProviderName + " · " + senderLabel(sn)
	}
	return ""
}

// SmsTestNumberInput adds a member's phone for test texts.
type SmsTestNumberInput struct {
	UserID string `json:"userId,omitempty"`
	Phone  string `json:"phone" minLength:"3" maxLength:"40"`
}

// AddTestNumber adds a test number. Audited.
func (s *Service) AddTestNumber(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsTestNumberInput) (SmsRules, error) {
	n, err := parsePhone(in.Phone)
	if err != nil {
		return SmsRules{}, api.Invalid("body.phone", "That isn't a phone number. Use the international format, like +1 312 555 0123.")
	}
	var user *uuid.UUID
	name := ""
	if in.UserID != "" {
		uid, err := uuid.Parse(in.UserID)
		if err != nil {
			return SmsRules{}, api.Invalid("body.userId", "Pick a member of this workspace.")
		}
		if err := s.Pool.QueryRow(ctx, `SELECT u.name FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = $1 AND m.user_id = $2`,
			ws.ID, uid).Scan(&name); err != nil {
			return SmsRules{}, api.Invalid("body.userId", "Pick a member of this workspace.")
		}
		user = &uid
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var exists int64
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM sms_test_numbers WHERE workspace_id = $1 AND phone = $2`, ws.ID, n.E164).Scan(&exists)
		if exists > 0 {
			return api.Conflict(displayPhone(n.E164) + " is already a test number.")
		}
		if err := q.SmsTestNumberInsert(ctx, store.SmsTestNumberInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, UserID: user, Phone: n.E164,
			AddedBy: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_test_number.added", TargetType: "phone", TargetName: maskPhone(n.E164),
			After: map[string]any{"member": name, "phone": maskPhone(n.E164)}})
	})
	if err != nil {
		return SmsRules{}, err
	}
	return s.Rules(ctx, ws, p)
}

// RemoveTestNumber removes a test number. Audited.
func (s *Service) RemoveTestNumber(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsRules, error) {
	var ph string
	if err := s.Pool.QueryRow(ctx, `SELECT phone FROM sms_test_numbers WHERE id = $1 AND workspace_id = $2`, id, ws.ID).Scan(&ph); err != nil {
		return SmsRules{}, api.MapError(err, "Test number")
	}
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsTestNumberDelete(ctx, store.SmsTestNumberDeleteParams{ID: id, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_test_number.removed", TargetType: "phone", TargetName: maskPhone(ph),
			Before: map[string]any{"phone": maskPhone(ph)}})
	})
	if err != nil {
		return SmsRules{}, err
	}
	return s.Rules(ctx, ws, p)
}

// SmsRouteTestInput asks which sender would text a number.
type SmsRouteTestInput struct {
	Phone string `json:"phone" minLength:"3" maxLength:"40"`
	// Message is a campaign ("campaign:<id>") or a reply ("help", "opt_out")
	// to count its segments; empty is a one-segment text.
	Message string `json:"message,omitempty" maxLength:"60"`
}

// SmsRouteTest is the route tester's answer.
type SmsRouteTest struct {
	Number      string `json:"number"`
	Country     string `json:"country"`
	CallingCode string `json:"callingCode"`
	// Route is how the country is covered: route, other (every other
	// country) or none.
	Route        string              `json:"route" enum:"route,other,none"`
	RouteCountries []string          `json:"routeCountries"`
	Provider     string              `json:"provider"`
	Adapter      string              `json:"adapter"`
	Sender       string              `json:"sender"`
	SenderKind   string              `json:"senderKind"`
	CanReply     bool                `json:"canReply"`
	Registration SmsRegistrationView `json:"registration"`
	CanSend      bool                `json:"canSend"`
	Price        float64             `json:"price"`
	Segments     int                 `json:"segments"`
	Cost         float64             `json:"cost"`
	Suppressed   string              `json:"suppressed,omitempty"`
	Zone         string              `json:"zone"`
	ZoneSource   string              `json:"zoneSource"`
	Local        string              `json:"local"`
	Inside       bool                `json:"inside"`
	Opens        string              `json:"opens"`
	Window       string              `json:"window"`
	// HelpNumber is the number people can text for HELP when the sender is
	// an ID they can't reply to.
	HelpNumber string `json:"helpNumber,omitempty"`
}

// SmsMessageOption is a message the tester can count.
type SmsMessageOption struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Segments int    `json:"segments"`
}

// RouteTest answers "Which sender would text +44 7700 900456?".
func (s *Service) RouteTest(ctx context.Context, ws *store.Workspace, in SmsRouteTestInput) (SmsRouteTest, error) {
	n, err := parsePhone(in.Phone)
	if err != nil {
		return SmsRouteTest{}, api.Invalid("body.phone", "That isn't a phone number. Use the international format, like +44 7700 900456.")
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsRouteTest{}, err
	}
	out := SmsRouteTest{Number: displayPhone(n.E164), Country: n.Country, CallingCode: fmt.Sprintf("+%d", n.CallingCode), Route: "none", RouteCountries: []string{},
		Window: g.rules.Window.String()}
	text := Text{Prefix: g.rules.BrandPrefix, Body: "x", OptOut: g.rules.OptOutText}.Full()
	switch {
	case strings.HasPrefix(in.Message, "campaign:"):
		if id, err := uuid.Parse(strings.TrimPrefix(in.Message, "campaign:")); err == nil {
			if c, err := s.Q.SmsCampaign(ctx, store.SmsCampaignParams{ID: id, WorkspaceID: ws.ID}); err == nil {
				text = Text{Prefix: g.rules.BrandPrefix, Body: withShortLinks(c.Body, s.shortDomain(g.rules), placeholder), OptOut: g.rules.OptOutText}.Full()
			}
		}
	case in.Message == "help":
		text = g.rules.HelpReply
	case in.Message == "opt_out":
		text = g.rules.OptOutReply
	}
	out.Segments = max(sms.CountText(text).Segments, 1)
	d := g.decide(n.Country)
	if d.Route != nil {
		out.Route = "route"
		out.RouteCountries = d.Route.Countries
		if !slices.Contains(d.Route.Countries, n.Country) {
			out.Route = "other"
		}
		if sn, ok := g.senders[d.Route.SenderID]; ok {
			out.Provider, out.Adapter, out.Sender, out.SenderKind = sn.ProviderName, sn.ProviderAdapter, senderLabel(sn), sn.Kind
			out.CanReply = sn.Kind != "sender_id"
		}
		out.Registration = regView(d.Reg)
		out.Registration.Country = n.Country
		out.CanSend = d.Sender != nil
		out.Price = d.Price
		out.Cost = round4(d.Price * float64(out.Segments))
	}
	if !out.CanReply {
		for _, id := range g.order {
			sn := g.senders[id]
			if sn.Kind != "sender_id" {
				out.HelpNumber = senderLabel(sn)
				break
			}
		}
	}
	if sup, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "sms", Phone: n.E164}); err == nil {
		out.Suppressed = sup.Reason
	}
	zone, src := phone.TimeZone(n.E164)
	out.Zone, out.ZoneSource = zone, string(src)
	loc := location(zone)
	now := s.now()
	out.Local = now.In(loc).Format("15:04")
	out.Inside = g.rules.Window.Inside(now, loc)
	out.Opens = g.rules.Window.NextOpen(now, loc).In(loc).Format("15:04")
	return out, nil
}

// MessageOptions lists the messages the route tester can count: recent SMS
// campaigns and the replies.
func (s *Service) MessageOptions(ctx context.Context, ws *store.Workspace) ([]SmsMessageOption, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	out := []SmsMessageOption{}
	rows, err := s.Pool.Query(ctx, `SELECT c.id, c.name, s.body FROM campaigns c JOIN sms_campaigns s ON s.campaign_id = c.id
		WHERE c.workspace_id = $1 AND c.deleted_at IS NULL ORDER BY c.updated_at DESC LIMIT 10`, ws.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id uuid.UUID
		var name, body string
		if rows.Scan(&id, &name, &body) == nil {
			full := Text{Prefix: g.rules.BrandPrefix, Body: withShortLinks(body, s.shortDomain(g.rules), placeholder), OptOut: g.rules.OptOutText}.Full()
			out = append(out, SmsMessageOption{Key: "campaign:" + id.String(), Name: name, Segments: max(sms.CountText(full).Segments, 1)})
		}
	}
	rows.Close()
	out = append(out, SmsMessageOption{Key: "help", Name: "help", Segments: max(sms.CountText(g.rules.HelpReply).Segments, 1)},
		SmsMessageOption{Key: "opt_out", Name: "opt_out", Segments: max(sms.CountText(g.rules.OptOutReply).Segments, 1)})
	return out, nil
}

// SmsShortDomainInput sets the short-link domain.
type SmsShortDomainInput struct {
	Domain string `json:"domain" maxLength:"253"`
}

// SetShortDomain sets (or clears) the short-link domain; it's verified by
// its CNAME. Audited.
func (s *Service) SetShortDomain(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsShortDomainInput) (SmsRules, error) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Domain), "."))
	if domain != "" && (!strings.Contains(domain, ".") || strings.ContainsAny(domain, "/: ")) {
		return SmsRules{}, api.Invalid("body.domain", "Enter a domain name, like go.acme.io.")
	}
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return SmsRules{}, err
	}
	status := "pending"
	if domain == "" {
		status = "none"
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := s.ensureRules(ctx, q, ws); err != nil {
			return err
		}
		if err := q.SmsSettingsShortDomain(ctx, store.SmsSettingsShortDomainParams{ShortDomain: domain, ShortDomainStatus: status, ShortDomainTarget: instanceHost(s.URLs.Base()),
			WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_short_domain.set", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"domain": r.ShortDomain}, After: map[string]any{"domain": domain}})
	})
	if err != nil {
		return SmsRules{}, err
	}
	if domain != "" {
		_, _ = s.checkShortDomain(ctx, ws, p)
	}
	return s.Rules(ctx, ws, p)
}

// CheckShortDomain checks the short-link domain's CNAME now.
func (s *Service) CheckShortDomain(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SmsRules, error) {
	if _, err := s.checkShortDomain(ctx, ws, p); err != nil {
		return SmsRules{}, err
	}
	return s.Rules(ctx, ws, p)
}

func (s *Service) checkShortDomain(ctx context.Context, ws *store.Workspace, p *authz.Principal) (string, error) {
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return "", err
	}
	if r.ShortDomain == "" {
		return "none", nil
	}
	target := instanceHost(s.URLs.Base())
	ok, detail := false, "DNS checks aren't available on this server."
	if s.DNS != nil {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cname, err := s.DNS.CNAME(cctx, r.ShortDomain)
		cancel()
		switch {
		case err == nil && strings.EqualFold(strings.TrimSuffix(cname, "."), target):
			ok, detail = true, "CNAME points at "+target+"."
		case err == nil && cname != "":
			detail = fmt.Sprintf("%s is a CNAME for %s, not %s.", r.ShortDomain, strings.TrimSuffix(cname, "."), target)
		case err != nil && !dnscheck.IsNotFound(err):
			detail = "DNS didn't answer for " + r.ShortDomain + "."
		default:
			detail = "No CNAME found. DNS can take up to an hour."
		}
	}
	now := s.now()
	status, verifiedAt := "failed", (*time.Time)(nil)
	if ok {
		status, verifiedAt = "verified", r.ShortVerifiedAt
		if verifiedAt == nil || r.ShortStatus != "verified" {
			verifiedAt = &now
		}
	} else if r.ShortStatus == "pending" || r.ShortStatus == "none" {
		status = "pending"
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsSettingsShortDomain(ctx, store.SmsSettingsShortDomainParams{ShortDomain: r.ShortDomain, ShortDomainStatus: status, ShortDomainTarget: target,
			CheckedAt: &now, VerifiedAt: verifiedAt, ShortDomainDetail: detail, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		if status == r.ShortStatus {
			return nil
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "sms_short_domain.checked", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"status": r.ShortStatus}, After: map[string]any{"status": status, "domain": r.ShortDomain}}
		if p == nil {
			ev.Actor = systemActor
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	return status, err
}

func parsePhone(raw string) (phone.Number, error) { return phone.Parse(raw, "") }
