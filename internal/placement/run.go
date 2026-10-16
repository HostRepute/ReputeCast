package placement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// PlacementContentRef names what a test sends.
type PlacementContentRef struct {
	Kind string `json:"kind" enum:"campaign,template,transactional"`
	ID   string `json:"id" doc:"A campaign, template or transactional template id"`
}

// PlacementRunInput is the Run a placement test dialog's choices.
type PlacementRunInput struct {
	Content PlacementContentRef `json:"content"`
	// Variant is the A/B variant, Language the language version ("" for
	// the source), TestData a transactional template's test data set
	// ("" for the fallbacks).
	Variant  string `json:"variant,omitempty" maxLength:"8"`
	Language string `json:"language,omitempty" maxLength:"16"`
	TestData string `json:"testData,omitempty" maxLength:"120"`
	// Through: the route as configured, one provider, or several compared.
	Through   string   `json:"through" enum:"route,provider,compare"`
	Providers []string `json:"providers,omitempty" maxItems:"6"`
	SenderID  string   `json:"senderId,omitempty"`
	// Seeds are the chosen seeds; empty means every connected seed.
	Seeds  []string `json:"seeds,omitempty" maxItems:"100"`
	Origin string   `json:"origin,omitempty" enum:"placement,review,composer,warmup,template,"`
}

// PlacementVariant is an A/B variant of the email.
type PlacementVariant struct {
	Key     string `json:"key"`
	Subject string `json:"subject"`
}

// PlacementLanguage is a language version of the email.
type PlacementLanguage struct {
	Code   string `json:"code"`
	Source bool   `json:"source"`
}

// PlacementContentView is the email the dialog shows under Content.
type PlacementContentView struct {
	Kind        string     `json:"kind" enum:"campaign,template,transactional"`
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status,omitempty"`
	Version     int        `json:"version,omitempty"`
	Key         string     `json:"key,omitempty"`
	Category    string     `json:"category,omitempty"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	Audience    string     `json:"audience,omitempty"`
	Recipients  *int64     `json:"recipients,omitempty"`
	// Variants, languages and test data sets the email has.
	Variants  []PlacementVariant  `json:"variants"`
	Languages []PlacementLanguage `json:"languages"`
	TestData  []string            `json:"testData"`
	// Sensitive variables carry a placeholder in copies (reset_url);
	// TestValues are the chosen set's other values (first_name "Sam").
	Sensitive  []string             `json:"sensitive"`
	TestValues []PlacementTestValue `json:"testValues"`
	// FallbackGreeting is the fallbacks' greeting ("Hi there").
	FallbackGreeting string `json:"fallbackGreeting,omitempty"`
}

// PlacementTestValue is a variable of the chosen test data set.
type PlacementTestValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PlacementProviderOption is a provider the dialog offers.
type PlacementProviderOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Short    string `json:"short"`
	Adapter  string `json:"adapter"`
	Color    string `json:"color"`
	Detail   string `json:"detail" doc:"us-east-1, mta.acme.io, the Transactional stream"`
	Status   string `json:"status" enum:"healthy,degraded,down,sandbox,warming_up,paused"`
	WarmDay  int    `json:"warmDay,omitempty"`
	WarmDays int    `json:"warmDays,omitempty"`
	// DownSince and DownReason explain an outage.
	DownSince  *time.Time `json:"downSince,omitempty"`
	DownReason string     `json:"downReason,omitempty"`
	// StreamOnly: the provider carries only another stream (Postmark's
	// Transactional stream) and can't send this email.
	StreamOnly string `json:"streamOnly,omitempty"`
	Available  bool   `json:"available"`
	// CostPerCopy in dollars (metered pricing; 0 for plans and self-hosted).
	CostPerCopy float64 `json:"costPerCopy"`
	// Pricing is the provider's model; PlanMonthly a plan's monthly allowance.
	Pricing     string `json:"pricing" enum:"metered,plan,self_hosted"`
	PlanMonthly int64  `json:"planMonthly,omitempty"`
	// TodayCap and SentToday: a warming-up provider's allowance.
	TodayCap  int64 `json:"todayCap,omitempty"`
	SentToday int64 `json:"sentToday,omitempty"`
}

// PlacementRouteShare is the route as configured, for one provider.
type PlacementRouteShare struct {
	ProviderID string   `json:"providerId"`
	Seeds      []string `json:"seeds"`
	// Reasons say why: "condition:outlook", "split:90", "default", "fallback".
	Reasons []string `json:"reasons"`
	// Fallback is the provider the route falls back to.
	Fallback string `json:"fallback,omitempty"`
}

// PlacementSenderOption is a sender of the stream.
type PlacementSenderOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Domain   string `json:"domain"`
	Verified bool   `json:"verified" doc:"Verified on every provider the test uses"`
	Default  bool   `json:"default"`
}

// PlacementSeedChoice is a seed in the dialog.
type PlacementSeedChoice struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Provider     string     `json:"provider"`
	ProviderName string     `json:"providerName"`
	Status       string     `json:"status"`
	StatusAt     *time.Time `json:"statusAt,omitempty"`
	Selected     bool       `json:"selected"`
}

// PlacementCopyCost is a provider's copies and their cost.
type PlacementCopyCost struct {
	ProviderID string  `json:"providerId"`
	Short      string  `json:"short"`
	Copies     int     `json:"copies"`
	Cost       float64 `json:"cost"`
	// Pricing and a plan's monthly allowance ("Inside Postmark's plan of 10,000 a month").
	Pricing     string `json:"pricing" enum:"metered,plan,self_hosted"`
	PlanMonthly int64  `json:"planMonthly,omitempty"`
	// Warm-up: today's cap and what's sent, when the provider warms up.
	WarmDay   int   `json:"warmDay,omitempty"`
	TodayCap  int64 `json:"todayCap,omitempty"`
	SentToday int64 `json:"sentToday,omitempty"`
}

// PlacementPlan is what a test would send, for the dialog.
type PlacementPlan struct {
	Content   PlacementContentView      `json:"content"`
	Stream    string                    `json:"stream" enum:"broadcast,automation,system,transactional"`
	Providers []PlacementProviderOption `json:"providers"`
	// Chosen are the providers the test would use (route: those the route picks).
	Chosen  []string                `json:"chosen"`
	Route   []PlacementRouteShare   `json:"route"`
	Senders []PlacementSenderOption `json:"senders"`
	Sender  string                  `json:"sender,omitempty"`
	// SenderProblem: no sender of the stream is verified on the chosen
	// providers; SenderMissing names the ones without one, and
	// SenderAlternatives the other providers that have one.
	SenderProblem      string                `json:"senderProblem,omitempty"`
	SenderDomain       string                `json:"senderDomain,omitempty"`
	SenderMissing      []string              `json:"senderMissing,omitempty"`
	SenderAlternatives []string              `json:"senderAlternatives,omitempty"`
	Seeds              []PlacementSeedChoice `json:"seeds"`
	Copies             int                   `json:"copies"`
	Costs              []PlacementCopyCost   `json:"costs"`
	// Problems keep the test from starting; CompareProblem explains a
	// comparison with fewer than two usable providers.
	Problems       []string `json:"problems"`
	CompareProblem string   `json:"compareProblem,omitempty"`
	// Alone is the one provider a comparison can still test.
	Alone    string `json:"alone,omitempty"`
	CanStart bool   `json:"canStart"`
	CanRun   bool   `json:"canRun"`
	// RunHelpers hold placement:run.
	RunHelpers []string `json:"runHelpers"`
	// Earlier tests through a warming-up provider ("day 2, day 5 and day 9").
	WarmupDays []int `json:"warmupDays"`
}

// content is a loaded email.
type content struct {
	view     PlacementContentView
	stream   providers.Stream
	subject  string
	html     string
	text     string
	senderID *uuid.UUID
	replyTo  string
	tracking engine.Tracking
	campaign *uuid.UUID
	category string
	note     string
	at       *time.Time
	template string
	rendered bool // transactional: already rendered with its test data
	langUsed string
}

func (s *Service) loadContent(ctx context.Context, ws *store.Workspace, in PlacementRunInput) (content, error) {
	l := i18n.LangFrom(ctx)
	c := content{view: PlacementContentView{Kind: in.Content.Kind, ID: in.Content.ID, Variants: []PlacementVariant{}, Languages: []PlacementLanguage{},
		TestData: []string{}, Sensitive: []string{}, TestValues: []PlacementTestValue{}}}
	id, err := uuid.Parse(in.Content.ID)
	if err != nil {
		return c, api.Invalid("body.content.id", mTestNoContent.Text(l, nil))
	}
	switch in.Content.Kind {
	case "campaign":
		r, err := s.Q.SendingCampaignContext(ctx, id)
		if err != nil || r.WorkspaceID != ws.ID {
			return c, api.NotFound("Campaign")
		}
		c.view.Name, c.view.Status, c.view.Version = r.Name, r.Status, int(r.Version)
		c.view.ScheduledAt, c.view.SentAt, c.view.Recipients = r.ScheduledAt, r.StartedAt, r.AudienceCount
		c.view.Audience = s.audienceLabel(ctx, ws, r.Audience)
		c.stream, c.campaign, c.senderID = providers.StreamBroadcast, &r.ID, r.SenderID
		c.subject, c.html, c.text, c.replyTo = r.Subject, r.Html, r.PlainText, firstNonEmpty(r.ReplyTo, r.SenderReplyTo)
		c.tracking = engine.Tracking{Opens: true, Clicks: true}
		if len(r.Tracking) > 0 {
			_ = json.Unmarshal(r.Tracking, &c.tracking)
		}
		var ab struct {
			Variants []struct {
				Key, Subject, Preheader, HTML string
			} `json:"variants"`
		}
		_ = json.Unmarshal(r.AbTest, &ab)
		for _, v := range ab.Variants {
			subj := firstNonEmpty(v.Subject, r.Subject)
			c.view.Variants = append(c.view.Variants, PlacementVariant{Key: v.Key, Subject: subj})
			if v.Key == in.Variant {
				c.subject = subj
				if strings.TrimSpace(v.HTML) != "" {
					c.html = v.HTML
				}
			}
		}
		c.view.Languages = s.campaignLanguages(ctx, ws, r.ID)
		if in.Language != "" && in.Language != sourceLanguage(c.view.Languages) {
			t, err := s.Q.PlacementCampaignTranslation(ctx, store.PlacementCampaignTranslationParams{OwnerID: r.ID, Language: in.Language})
			if err == nil && strings.TrimSpace(t.Html) != "" {
				c.subject, c.html, c.text, c.langUsed = firstNonEmpty(t.Subject, c.subject), t.Html, t.PlainText, in.Language
			}
		}
		if c.langUsed == "" {
			c.langUsed = sourceLanguage(c.view.Languages)
		}
		if strings.TrimSpace(c.html) == "" {
			return c, api.Conflict(mTestContentEmpty.Text(l, i18n.Args{"name": r.Name}))
		}
		c.html = media.Absolutize(c.html, s.URLs.Base())
		c.view.FallbackGreeting = greeting(c.html)
	case "template":
		t, err := s.Q.PlacementTemplate(ctx, store.PlacementTemplateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return c, api.MapError(err, "Template")
		}
		c.view.Name, c.stream, c.subject, c.html = t.Name, providers.StreamBroadcast, t.Name, t.Html
		c.tracking = engine.Tracking{Opens: true, Clicks: true}
		if strings.TrimSpace(t.Html) == "" {
			return c, api.Conflict(mTestContentEmpty.Text(l, i18n.Args{"name": t.Name}))
		}
		c.html = media.Absolutize(c.html, s.URLs.Base())
		c.view.FallbackGreeting = greeting(c.html)
	case "transactional":
		t, err := s.Q.PlacementTransactional(ctx, store.PlacementTransactionalParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return c, api.MapError(err, "Transactional template")
		}
		c.view.Name, c.view.Key, c.view.Category, c.template = t.Name, t.Key, t.CategoryName, t.Key
		if t.LiveVersion == nil {
			return c, api.Conflict(mTestNotPublished.Text(l, i18n.Args{"name": t.Name}))
		}
		c.view.Version = int(*t.LiveVersion)
		c.stream = routing.StreamTransactional
		v, err := s.Q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: id, Version: *t.LiveVersion})
		if err != nil {
			return c, err
		}
		c.senderID, c.replyTo = v.SenderID, v.ReplyTo
		c.tracking = engine.Tracking{Opens: v.TrackOpens, Clicks: v.TrackClicks}
		var vars []struct {
			Name      string `json:"name"`
			Sample    string `json:"sample"`
			Default   string `json:"default"`
			Sensitive bool   `json:"sensitive"`
		}
		_ = json.Unmarshal(v.Variables, &vars)
		var sets []struct {
			Name string         `json:"name"`
			Data map[string]any `json:"data"`
		}
		_ = json.Unmarshal(v.TestData, &sets)
		// The fallbacks are the variables' defaults (or samples); a test
		// data set fills in the rest. Sensitive variables (reset_url)
		// carry a placeholder: copies never hold a real secret.
		data := map[string]any{}
		for _, x := range vars {
			if val := firstNonEmpty(x.Default, x.Sample); val != "" {
				data[x.Name] = val
			}
		}
		for _, d := range sets {
			c.view.TestData = append(c.view.TestData, d.Name)
			if d.Name == in.TestData {
				for k, val := range d.Data {
					data[k] = val
				}
			}
		}
		for _, x := range vars {
			if x.Sensitive {
				c.view.Sensitive = append(c.view.Sensitive, x.Name)
				data[x.Name] = strings.TrimRight(s.URLs.Base(), "/") + "/"
				continue
			}
			if val, ok := data[x.Name]; ok && len(c.view.TestValues) < 4 {
				c.view.TestValues = append(c.view.TestValues, PlacementTestValue{Name: x.Name, Value: clip(fmt.Sprint(val), 80)})
			}
		}
		if s.Transactional == nil {
			return c, api.Conflict(mTestContentEmpty.Text(l, i18n.Args{"name": t.Name}))
		}
		subj, html, text, _, err := s.Transactional.RenderLive(ctx, ws, t.Key, data, "")
		if err != nil {
			return c, err
		}
		c.subject, c.html, c.text, c.rendered = subj, html, text, true
	default:
		return c, api.Invalid("body.content.kind", mTestNoContent.Text(l, nil))
	}
	c.note = s.contentNote(&c)
	return c, nil
}

// greeting finds the fallbacks' greeting ("Hi there") in an email's Liquid.
func greeting(html string) string {
	i := strings.Index(html, "first_name")
	if i < 0 {
		return ""
	}
	seg := html[i:min(len(html), i+80)]
	j := strings.Index(seg, `default:`)
	if j < 0 {
		return ""
	}
	rest := strings.TrimSpace(seg[j+len("default:"):])
	if len(rest) < 2 {
		return ""
	}
	q := rest[0]
	if q != '"' && q != '\'' {
		return ""
	}
	if k := strings.IndexByte(rest[1:], q); k > 0 {
		return "Hi " + rest[1:1+k]
	}
	return ""
}

// campaignLanguages are a campaign's source language and its translated versions.
func (s *Service) campaignLanguages(ctx context.Context, ws *store.Workspace, id uuid.UUID) []PlacementLanguage {
	src, err := s.Q.PlacementSourceLanguage(ctx, store.PlacementSourceLanguageParams{OwnerID: id, WorkspaceID: ws.ID})
	if err != nil || src == "" {
		src = "en"
	}
	out := []PlacementLanguage{{Code: src, Source: true}}
	codes, _ := s.Q.PlacementCampaignLanguages(ctx, id)
	for _, code := range codes {
		if code != src {
			out = append(out, PlacementLanguage{Code: code})
		}
	}
	return out
}

func sourceLanguage(ls []PlacementLanguage) string {
	for _, l := range ls {
		if l.Source {
			return l.Code
		}
	}
	return ""
}

// audienceLabel names a campaign's lists ("Weekly Digest list").
func (s *Service) audienceLabel(ctx context.Context, ws *store.Workspace, raw json.RawMessage) string {
	var a struct {
		Lists []string `json:"lists"`
	}
	_ = json.Unmarshal(raw, &a)
	ids := []uuid.UUID{}
	for _, x := range a.Lists {
		if id, err := uuid.Parse(x); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ""
	}
	names, err := s.Q.PlacementListNames(ctx, store.PlacementListNamesParams{WorkspaceID: ws.ID, Ids: ids})
	if err != nil {
		return ""
	}
	return strings.Join(names, " + ")
}

// contentNote is how the content stood: draft, scheduled or sent for a
// campaign (with when, in content_at), template, or transactional.
func (s *Service) contentNote(c *content) string {
	switch c.view.Kind {
	case "campaign":
		switch c.view.Status {
		case "scheduled", "approved":
			if c.view.ScheduledAt != nil {
				c.at = c.view.ScheduledAt
				return "scheduled"
			}
		case "sent", "sending", "paused", "cancelled":
			if c.view.SentAt != nil {
				c.at = c.view.SentAt
				return "sent"
			}
		}
		return "draft"
	case "transactional":
		return "transactional"
	}
	return "template"
}

func feedbackID(id, workspace, stream string) string {
	if len(id) > 8 {
		id = id[:8]
	}
	return id + ":" + workspace + ":" + stream + ":reputecast"
}

// routingMailbox is a seed's mailbox provider as routing conditions know it.
func routingMailbox(provider string) string {
	switch provider {
	case seedbox.Gmail, seedbox.GoogleWorkspace:
		return routing.Gmail
	case seedbox.Outlook, seedbox.Microsoft365:
		return routing.Outlook
	case seedbox.Yahoo, seedbox.AOL:
		return routing.Yahoo
	case seedbox.ICloud:
		return routing.ICloud
	}
	return routing.Other
}

// providerOption describes a provider for the dialog.
func (s *Service) providerOption(ctx context.Context, ws *store.Workspace, p store.Provider, stream providers.Stream) PlacementProviderOption {
	o := PlacementProviderOption{ID: p.ID.String(), Name: p.Name, Short: state.Short(p.Name, p.Adapter), Adapter: p.Adapter, Color: state.Color(p.Adapter),
		Status: p.Status, Available: state.Available(p)}
	if o.Status == "" {
		o.Status = state.Healthy
	}
	o.Detail = providerDetail(p)
	if h := state.HealthOf(p); h.Down != nil {
		since := h.Down.Since
		o.DownSince, o.DownReason = &since, firstNonEmpty(p.StatusDetail, h.Down.Reason)
	}
	pr := state.PricingOf(p)
	o.Pricing = pr.Model
	if pr.Model == "metered" {
		o.CostPerCopy = pr.Per1000 / 1000
	}
	if q := state.HealthOf(p).Quota; q != nil && q.Kind == "monthly_plan" {
		o.PlanMonthly = q.Max
	}
	if a, ok, err := warmup.AllowanceFor(ctx, s.Q, p.ID, s.now(), state.Location(ws.Timezone)); err == nil && ok {
		o.WarmDay, o.WarmDays, o.TodayCap, o.SentToday = a.Day, a.Days, a.Cap, a.Sent
		if o.Status == state.Healthy {
			o.Status = state.WarmingUp
		}
	}
	if stream == providers.StreamBroadcast && !state.Caps(p).BulkAllowed() {
		o.StreamOnly = "transactional"
		o.Available = false
	}
	return o
}

// providerDetail is the line under a provider's name: its region or host.
func providerDetail(p store.Provider) string {
	var c struct {
		Region    string `json:"region"`
		Host      string `json:"host"`
		URL       string `json:"url"`
		ServerURL string `json:"server_url"`
	}
	_ = json.Unmarshal(p.Config, &c)
	switch {
	case c.Region != "":
		return c.Region
	case c.Host != "":
		return c.Host
	case c.URL != "" || c.ServerURL != "":
		if u, err := url.Parse(firstNonEmpty(c.ServerURL, c.URL)); err == nil && u.Host != "" {
			return u.Hostname()
		}
	}
	// Postmark's stream ("Transactional stream").
	return p.Label
}

// planned is a plan with what Start needs: the chosen seeds, the seeds
// left out because their sign-in failed (shown as Not checked), who gets
// which provider, and each seed's row.
type planned struct {
	plan    PlacementPlan
	c       content
	seeds   []store.PlacementSeed
	skipped []store.PlacementSeed
	// assign maps a provider to its seeds; skipAssign a skipped seed to
	// the providers it would have had.
	assign     map[uuid.UUID][]uuid.UUID
	skipAssign map[uuid.UUID][]uuid.UUID
	position   map[uuid.UUID]int
}

// plan works out a test's providers, senders, seeds and copies.
func (s *Service) plan(ctx context.Context, ws *store.Workspace, in PlacementRunInput) (planned, error) {
	l := i18n.LangFrom(ctx)
	p := api.PrincipalFrom(ctx)
	out := PlacementPlan{Providers: []PlacementProviderOption{}, Chosen: []string{}, Route: []PlacementRouteShare{}, Senders: []PlacementSenderOption{},
		Seeds: []PlacementSeedChoice{}, Costs: []PlacementCopyCost{}, Problems: []string{}, WarmupDays: []int{},
		CanRun: p.Can(authz.PlacementRun), RunHelpers: Helpers(ctx, s.Q, ws.ID, authz.PlacementRun)}
	res := planned{assign: map[uuid.UUID][]uuid.UUID{}, skipAssign: map[uuid.UUID][]uuid.UUID{}, position: map[uuid.UUID]int{}}
	c, err := s.loadContent(ctx, ws, in)
	if err != nil {
		res.plan = out
		return res, err
	}
	out.Content, out.Stream = c.view, string(c.stream)
	res.c = c

	// Seeds: every connected seed unless some are picked. Seeds whose
	// sign-in failed are left out, and shown as Not checked.
	rows, err := s.Q.PlacementSeeds(ctx, ws.ID)
	if err != nil {
		return res, err
	}
	picked := map[string]bool{}
	for _, id := range in.Seeds {
		picked[id] = true
	}
	for i, r := range rows {
		sd := r.PlacementSeed
		res.position[sd.ID] = i + 1
		sel := sd.Status == "connected" && (len(picked) == 0 || picked[sd.ID.String()])
		pv, _ := seedbox.ProviderOf(sd.Provider)
		out.Seeds = append(out.Seeds, PlacementSeedChoice{ID: sd.ID.String(), Email: sd.Email, Provider: sd.Provider, ProviderName: pv.Name,
			Status: sd.Status, StatusAt: sd.StatusAt, Selected: sel})
		switch {
		case sel:
			res.seeds = append(res.seeds, sd)
		case len(picked) == 0 && (sd.Status == "failed" || sd.Status == "expired"):
			res.skipped = append(res.skipped, sd)
		}
	}
	if len(rows) == 0 {
		out.Problems = append(out.Problems, mTestNoSeeds.Text(l, nil))
	} else if len(res.seeds) == 0 {
		out.Problems = append(out.Problems, mTestNoConnectedSeeds.Text(l, nil))
	}

	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return res, err
	}
	opts := map[uuid.UUID]PlacementProviderOption{}
	for _, pr := range provs {
		o := s.providerOption(ctx, ws, pr, c.stream)
		opts[pr.ID] = o
		out.Providers = append(out.Providers, o)
	}

	// Who gets which provider.
	switch in.Through {
	case "route":
		set, err := routing.Load(ctx, s.Q, ws.ID)
		if err != nil {
			return res, err
		}
		shares := map[uuid.UUID]*PlacementRouteShare{}
		order := []uuid.UUID{}
		pick := func(sd store.PlacementSeed) (uuid.UUID, routing.Decision, string, bool) {
			d, ok := set.Resolve(routing.Recipient{Stream: c.stream, Email: sd.Email, MailboxProvider: routingMailbox(sd.Provider), Category: c.category})
			if !ok {
				return uuid.Nil, d, "", false
			}
			pid, reason := d.ProviderID, routeReason(d)
			if o, ok := opts[pid]; ok && !o.Available && d.Fallback != nil {
				if fb, ok := opts[*d.Fallback]; ok && fb.Available {
					pid, reason = *d.Fallback, "fallback"
				}
			}
			return pid, d, reason, true
		}
		for _, sd := range res.seeds {
			pid, d, reason, ok := pick(sd)
			if !ok {
				out.Problems = append(out.Problems, mTestNoRoute.Text(l, i18n.Args{"stream": streamName(c.stream)}))
				break
			}
			if shares[pid] == nil {
				shares[pid] = &PlacementRouteShare{ProviderID: pid.String(), Seeds: []string{}, Reasons: []string{}}
				if d.Fallback != nil && *d.Fallback != pid {
					shares[pid].Fallback = d.Fallback.String()
				}
				order = append(order, pid)
			}
			shares[pid].Seeds = append(shares[pid].Seeds, sd.ID.String())
			if !slices.Contains(shares[pid].Reasons, reason) {
				shares[pid].Reasons = append(shares[pid].Reasons, reason)
			}
			res.assign[pid] = append(res.assign[pid], sd.ID)
		}
		for _, sd := range res.skipped {
			if pid, _, _, ok := pick(sd); ok && shares[pid] != nil {
				res.skipAssign[sd.ID] = []uuid.UUID{pid}
			}
		}
		for _, pid := range order {
			out.Route = append(out.Route, *shares[pid])
			out.Chosen = append(out.Chosen, pid.String())
		}
	case "provider", "compare":
		usable := []uuid.UUID{}
		for _, raw := range in.Providers {
			pid, err := uuid.Parse(raw)
			if err != nil {
				continue
			}
			o, ok := opts[pid]
			if !ok {
				continue
			}
			if !o.Available {
				if o.StreamOnly != "" {
					out.Problems = append(out.Problems, mTestWrongStream.Text(l, i18n.Args{"provider": o.Name}))
				} else {
					out.Problems = append(out.Problems, mTestProviderDown.Text(l, i18n.Args{"provider": o.Name}))
				}
				continue
			}
			usable = append(usable, pid)
			if in.Through == "provider" {
				break
			}
		}
		if in.Through == "compare" && len(usable) < 2 {
			out.CompareProblem = mTestCompareNeedsTwo.Text(l, nil)
			if len(usable) == 1 {
				out.Alone = usable[0].String()
			}
			out.Problems = append(out.Problems, out.CompareProblem)
		}
		if len(usable) == 0 {
			out.Problems = append(out.Problems, mTestNoProviders.Text(l, nil))
		}
		for _, pid := range usable {
			out.Chosen = append(out.Chosen, pid.String())
			for _, sd := range res.seeds {
				res.assign[pid] = append(res.assign[pid], sd.ID)
			}
		}
		for _, sd := range res.skipped {
			res.skipAssign[sd.ID] = usable
		}
	}

	// Senders of the stream, verified on every chosen provider.
	if err := s.senders(ctx, ws, c, in.SenderID, out.Chosen, out.Providers, &out); err != nil {
		return res, err
	}

	// Copies, cost and warm-up caps.
	for _, raw := range out.Chosen {
		pid := uuid.MustParse(raw)
		o := opts[pid]
		n := len(res.assign[pid])
		out.Copies += n
		cc := PlacementCopyCost{ProviderID: raw, Short: o.Short, Copies: n, Cost: math.Round(float64(n)*o.CostPerCopy*1000) / 1000,
			Pricing: o.Pricing, PlanMonthly: o.PlanMonthly}
		if o.TodayCap > 0 {
			cc.WarmDay, cc.TodayCap, cc.SentToday = o.WarmDay, o.TodayCap, o.SentToday
			if left := o.TodayCap - o.SentToday; int64(n) > left {
				out.Problems = append(out.Problems, mTestWarmupCap.Text(l, i18n.Args{"provider": o.Name, "left": max(left, 0), "cap": o.TodayCap, "needed": n}))
			}
			if days, err := s.warmupDays(ctx, ws, pid); err == nil {
				out.WarmupDays = days
			}
		}
		out.Costs = append(out.Costs, cc)
	}
	out.CanStart = len(out.Problems) == 0 && out.SenderProblem == "" && out.Copies > 0 && out.Sender != ""
	res.plan = out
	return res, nil
}

func streamName(st providers.Stream) string {
	switch st {
	case routing.StreamTransactional:
		return "Transactional"
	case providers.StreamAutomation:
		return "Automations"
	case providers.StreamSystem:
		return "System"
	}
	return "Broadcasts"
}

// routeReason says why the route picked a provider for a seed.
func routeReason(d routing.Decision) string {
	switch {
	case d.Matched != "":
		return "condition:" + d.MatchedValue
	case d.Split:
		for _, sp := range d.Rule.Splits {
			if sp.ProviderID == d.ProviderID {
				return fmt.Sprintf("split:%d", sp.Weight)
			}
		}
		return "split"
	}
	return "default"
}

// warmupDays are the warm-up days of earlier tests through a provider.
func (s *Service) warmupDays(ctx context.Context, ws *store.Workspace, pid uuid.UUID) ([]int, error) {
	tests, err := s.Q.PlacementTestsForProvider(ctx, store.PlacementTestsForProviderParams{WorkspaceID: ws.ID, ProviderID: pid, Since: s.now().AddDate(0, 0, -60)})
	if err != nil {
		return nil, err
	}
	out := []int{}
	for _, t := range tests {
		var m map[string]providerSnapshot
		_ = json.Unmarshal(t.Providers, &m)
		if ps, ok := m[pid.String()]; ok && ps.WarmDay > 0 && !slices.Contains(out, ps.WarmDay) {
			out = append(out, ps.WarmDay)
		}
	}
	return out, nil
}

// senders lists the stream's senders, says which are verified on every
// chosen provider, and picks one: the one asked for, the email's own, or
// the first verified. When none is, it names the providers that lack one
// and the others that have one.
func (s *Service) senders(ctx context.Context, ws *store.Workspace, c content, want string, chosen []string, opts []PlacementProviderOption, out *PlacementPlan) error {
	l := i18n.LangFrom(ctx)
	type sender struct {
		id                  uuid.UUID
		name, email, domain string
		verification        map[string]string
		transactional       bool
	}
	rows, err := s.Q.TxnSenders(ctx, ws.ID)
	if err != nil {
		return err
	}
	// Senders of the stream: transactional domains carry the Transactional
	// stream, the others the rest; a workspace with one kind uses them all.
	var all, list []sender
	for _, r := range rows {
		var v map[string]string
		_ = json.Unmarshal(r.Verification, &v)
		all = append(all, sender{id: r.ID, name: r.Name, email: r.Email, domain: r.Domain, verification: v, transactional: r.Transactional})
	}
	wantTxn := c.stream == routing.StreamTransactional
	for _, sd := range all {
		if sd.transactional == wantTxn {
			list = append(list, sd)
		}
	}
	if len(list) == 0 {
		list = all
	}
	def := ""
	if c.senderID != nil {
		def = c.senderID.String()
	}
	verifiedOn := func(sd sender, pids []string) bool {
		if len(pids) == 0 {
			return false
		}
		for _, pid := range pids {
			if sd.verification[pid] != "verified" {
				return false
			}
		}
		return true
	}
	for _, sd := range list {
		out.Senders = append(out.Senders, PlacementSenderOption{ID: sd.id.String(), Name: sd.name, Email: sd.email, Domain: sd.domain,
			Verified: verifiedOn(sd, chosen), Default: sd.id.String() == def})
	}
	pick := func(id string) bool {
		for _, o := range out.Senders {
			if o.ID == id && o.Verified {
				out.Sender = id
				return true
			}
		}
		return false
	}
	if (want == "" || !pick(want)) && (def == "" || !pick(def)) {
		for _, o := range out.Senders {
			if o.Verified {
				out.Sender = o.ID
				break
			}
		}
	}
	if out.Sender != "" || len(chosen) == 0 {
		return nil
	}
	for _, o := range out.Senders {
		if o.ID == def || out.SenderDomain == "" {
			out.SenderDomain = o.Domain
		}
	}
	out.SenderMissing, out.SenderAlternatives = []string{}, []string{}
	for _, o := range opts {
		has := false
		for _, sd := range list {
			if verifiedOn(sd, []string{o.ID}) {
				has = true
			}
		}
		switch {
		case slices.Contains(chosen, o.ID) && !has:
			out.SenderMissing = append(out.SenderMissing, o.Name)
		case !slices.Contains(chosen, o.ID) && has && o.Available:
			out.SenderAlternatives = append(out.SenderAlternatives, o.Name)
		}
	}
	out.SenderProblem = mTestNoSender.Text(l, i18n.Args{"stream": streamName(c.stream)})
	return nil
}

// Plan says what a test would send, for the dialog.
func (s *Service) Plan(ctx context.Context, ws *store.Workspace, in PlacementRunInput) (PlacementPlan, error) {
	res, err := s.plan(ctx, ws, in)
	return res.plan, err
}

// providerSnapshot is a provider as it was when a test started.
type providerSnapshot struct {
	Name     string  `json:"name"`
	Short    string  `json:"short"`
	Adapter  string  `json:"adapter"`
	Detail   string  `json:"detail"`
	IP       string  `json:"ip,omitempty"`
	Status   string  `json:"status"`
	WarmDay  int     `json:"warmDay,omitempty"`
	WarmDays int     `json:"warmDays,omitempty"`
	Cost     float64 `json:"cost"`
}

// Start creates a test and its copies and starts sending (audited).
func (s *Service) Start(ctx context.Context, ws *store.Workspace, in PlacementRunInput) (PlacementTestDetail, error) {
	res, err := s.plan(ctx, ws, in)
	if err != nil {
		return PlacementTestDetail{}, err
	}
	plan, c := res.plan, res.c
	if !plan.CanStart {
		msg := firstNonEmpty(append(plan.Problems, plan.SenderProblem)...)
		return PlacementTestDetail{}, api.Err(http.StatusConflict, "placement_not_ready", msg).WithMeta("problems", plan.Problems)
	}
	sender := uuid.MustParse(plan.Sender)
	var fromName, fromEmail string
	for _, o := range plan.Senders {
		if o.ID == plan.Sender {
			fromName, fromEmail = o.Name, o.Email
		}
	}
	snap := map[string]providerSnapshot{}
	shorts := []string{}
	pids := []uuid.UUID{}
	for _, raw := range plan.Chosen {
		for _, o := range plan.Providers {
			if o.ID != raw {
				continue
			}
			pid := uuid.MustParse(o.ID)
			pids = append(pids, pid)
			ps := providerSnapshot{Name: o.Name, Short: o.Short, Adapter: o.Adapter, Detail: o.Detail, Status: o.Status, WarmDay: o.WarmDay, WarmDays: o.WarmDays, Cost: o.CostPerCopy}
			if pr, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: pid}); err == nil && len(pr.Ips) > 0 {
				ps.IP = pr.Ips[0]
			}
			snap[o.ID] = ps
			shorts = append(shorts, o.Short)
		}
	}
	name := c.view.Name
	switch {
	case in.Through == "compare" || len(shorts) > 1:
		name += " · " + strings.Join(shorts, " vs ")
	case len(shorts) == 1:
		name += " · " + shorts[0]
	}
	// Copy order: in compare mode one round per provider, 2 minutes apart,
	// alternating which provider goes first.
	now := s.now()
	type cp struct {
		seed   store.PlacementSeed
		pid    uuid.UUID
		round  int
		at     time.Time
		status string
		reason string
	}
	var copies []cp
	if in.Through == "compare" {
		for i, sd := range res.seeds {
			for r := range pids {
				pid := pids[(i+r)%len(pids)]
				copies = append(copies, cp{seed: sd, pid: pid, round: r + 1, at: now.Add(time.Duration(r) * CompareGap), status: "queued"})
			}
		}
	} else {
		for _, pid := range pids {
			for _, sid := range res.assign[pid] {
				for _, sd := range res.seeds {
					if sd.ID == sid {
						copies = append(copies, cp{seed: sd, pid: pid, round: 1, at: now, status: "queued"})
					}
				}
			}
		}
	}
	last := now
	for _, x := range copies {
		if x.at.After(last) {
			last = x.at
		}
	}
	sent := len(copies)
	for _, sd := range res.skipped {
		reason := ReasonSignInFailed
		if sd.Status == "expired" {
			reason = ReasonSignInExpired
		}
		for r, pid := range res.skipAssign[sd.ID] {
			copies = append(copies, cp{seed: sd, pid: pid, round: r + 1, at: now, status: "not_checked", reason: reason})
		}
	}
	tpl, err := engine.Prepare(engine.Content{Subject: c.subject, HTML: c.html, Text: c.text}, c.tracking)
	if err != nil {
		return PlacementTestDetail{}, api.Invalid("body.content", err.Error())
	}
	links := tpl.Links
	if links == nil {
		links = []engine.TrackedLink{}
	}
	lb, _ := json.Marshal(links)
	sb, _ := json.Marshal(snap)
	deleteAfter, err := s.Q.PlacementSettings(ctx, ws.ID)
	if err != nil {
		return PlacementTestDetail{}, err
	}
	id := uuid.Must(uuid.NewV7())
	origin := firstNonEmpty(in.Origin, "placement")
	var contentID *uuid.UUID
	if cid, err := uuid.Parse(in.Content.ID); err == nil {
		contentID = &cid
	}
	wait := last.Add(WaitFor)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PlacementTestInsert(ctx, store.PlacementTestInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, ContentKind: in.Content.Kind,
			ContentID: contentID, ContentName: c.view.Name, ContentNote: c.note, ContentKey: c.view.Key, ContentVersion: int32(c.view.Version), ContentAt: c.at,
			Variant: in.Variant, Language: c.langUsed, Personalization: in.TestData, Through: in.Through, Stream: string(c.stream), ProviderIds: pids,
			SenderID: &sender, FromName: fromName, FromEmail: fromEmail, Origin: origin, Subject: c.subject, Html: c.html, TextBody: c.text, Links: lb,
			Providers: sb, TrackOpens: c.tracking.Opens, TrackClicks: c.tracking.Clicks, StartedBy: actor(ctx), StartedAt: now, WaitUntil: &wait,
			DeleteAfter: deleteAfter}); err != nil {
			return err
		}
		for _, x := range copies {
			if err := q.PlacementCopyInsert(ctx, store.PlacementCopyInsertParams{ID: uuid.Must(uuid.NewV7()), TestID: id, WorkspaceID: ws.ID, SeedID: &x.seed.ID,
				SeedEmail: x.seed.Email, SeedProvider: x.seed.Provider, Position: int32(res.position[x.seed.ID]), ProviderID: &x.pid,
				ProviderName: plan.providerName(x.pid), Token: newToken(), Round: int32(x.round), SendAt: x.at, Status: x.status, NotChecked: x.reason}); err != nil {
				return err
			}
		}
		seedIDs := make([]uuid.UUID, len(res.seeds))
		for i, sd := range res.seeds {
			seedIDs[i] = sd.ID
		}
		if err := q.PlacementSeedUsed(ctx, store.PlacementSeedUsedParams{At: &now, TestID: &id, Label: testLabel(c), Ids: seedIDs}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.test_start", TargetType: "placement_test", TargetID: id.String(),
			TargetName: name, Metadata: map[string]any{"content": c.view.Name, "contentKind": in.Content.Kind, "providers": shorts, "through": in.Through,
				"seeds": len(res.seeds), "copies": sent, "origin": origin, "summary": fmt.Sprintf("%d copies to %d seeds", sent, len(res.seeds))}})
	})
	if err != nil {
		return PlacementTestDetail{}, err
	}
	if err := s.enqueueSend(ctx, id); err != nil {
		return PlacementTestDetail{}, err
	}
	return s.Test(ctx, ws, id)
}

func (p PlacementPlan) providerName(id uuid.UUID) string {
	for _, o := range p.Providers {
		if o.ID == id.String() {
			return o.Name
		}
	}
	return ""
}

// testLabel names a test for the seeds' "last used" line ("password-reset test").
func testLabel(c content) string {
	if c.view.Kind == "transactional" {
		return c.view.Key + " test"
	}
	// "Flash sale: 48 hours only" reads as "Flash sale test".
	name, _, _ := strings.Cut(c.view.Name, ": ")
	return name + " test"
}

// SendArgs hands a test's due copies to their providers.
type SendArgs struct {
	TestID uuid.UUID `json:"testId"`
}

// Kind implements river.JobArgs.
func (SendArgs) Kind() string { return "placement.send" }

// SendWorker runs SendArgs.
type SendWorker struct {
	river.WorkerDefaults[SendArgs]
	Svc *Service
}

// Work implements river.Worker; a comparison's later rounds snooze until due.
func (w *SendWorker) Work(ctx context.Context, job *river.Job[SendArgs]) error {
	next, err := w.Svc.SendDue(ctx, job.Args.TestID)
	if err != nil {
		return err
	}
	if next != nil {
		return river.JobSnooze(max(time.Until(*next), time.Second))
	}
	return nil
}

func (s *Service) enqueueSend(ctx context.Context, id uuid.UUID) error {
	if !s.client() {
		_, err := s.SendDue(context.WithoutCancel(ctx), id)
		return err
	}
	_, err := s.Jobs().Insert(ctx, SendArgs{TestID: id}, nil)
	return err
}

// SendDue sends the copies whose time has come and returns when the next
// round is due (nil when every copy is out).
func (s *Service) SendDue(ctx context.Context, id uuid.UUID) (*time.Time, error) {
	t, err := s.Q.PlacementTestByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t.Status != "sending" {
		return nil, nil
	}
	ws, err := s.Q.GetWorkspace(ctx, t.WorkspaceID)
	if err != nil {
		return nil, err
	}
	var links []engine.TrackedLink
	_ = json.Unmarshal(t.Links, &links)
	tpl, err := engine.Prepare(engine.Content{Subject: t.Subject, HTML: t.Html, Text: t.TextBody}, engine.Tracking{Opens: t.TrackOpens, Clicks: t.TrackClicks})
	if err != nil {
		return nil, err
	}
	due, err := s.Q.PlacementCopiesDue(ctx, store.PlacementCopiesDueParams{TestID: id, Now: s.now()})
	if err != nil {
		return nil, err
	}
	provs := map[uuid.UUID]store.Provider{}
	for _, c := range due {
		if c.ProviderID == nil {
			continue
		}
		p, ok := provs[*c.ProviderID]
		if !ok {
			p, err = s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: *c.ProviderID})
			if err != nil {
				_ = s.Q.PlacementCopySent(ctx, store.PlacementCopySentParams{ID: c.ID, Status: "failed", Error: "the provider was removed"})
				continue
			}
			provs[p.ID] = p
		}
		msg := s.compose(&ws, t, c, tpl)
		at := s.now()
		d, err := s.Deliver.Send(ctx, p, msg, Kind)
		status, handed, errText := "sent", ptrTime(d.SentAt), ""
		if d.SentAt.IsZero() {
			handed = &at
		}
		if err != nil {
			status, handed, errText = "failed", nil, clip(err.Error(), 400)
		}
		if err := s.Q.PlacementCopySent(ctx, store.PlacementCopySentParams{ID: c.ID, Status: status, HandedAt: handed, ProviderMessageID: d.ProviderMessageID, Error: errText}); err != nil {
			return nil, err
		}
	}
	// What's left, and when.
	all, err := s.Q.PlacementCopies(ctx, id)
	if err != nil {
		return nil, err
	}
	var next *time.Time
	var last time.Time
	for _, c := range all {
		if c.Status == "queued" && (next == nil || c.SendAt.Before(*next)) {
			at := c.SendAt
			next = &at
		}
		if c.HandedAt != nil && c.HandedAt.After(last) {
			last = *c.HandedAt
		}
	}
	if next != nil {
		return next, nil
	}
	if last.IsZero() {
		last = s.now()
	}
	wait, late := last.Add(WaitFor), last.Add(LateWithin)
	if err := s.Q.PlacementTestSetStatus(ctx, store.PlacementTestSetStatusParams{ID: id, Status: "waiting", LastSentAt: &last, WaitUntil: &wait, LateUntil: &late}); err != nil {
		return nil, err
	}
	if !s.client() {
		_ = s.Check(context.WithoutCancel(ctx), id)
	}
	return nil, nil
}

// compose builds a copy: the email with the fallbacks, link tracking on
// (opens and clicks are never counted), the bulk headers a real send has,
// and the test token.
func (s *Service) compose(ws *store.Workspace, t store.PlacementTest, c store.PlacementCopy, tpl *engine.Template) providers.Message {
	sign := func(k linktoken.Kind, link uint32) string {
		return s.Links.Sign(linktoken.Claims{Kind: k, WorkspaceID: ws.ID, MessageID: c.ID, Link: link})
	}
	unsub := sign(linktoken.Unsubscribe, 0)
	links := map[string]string{
		"unsubscribe_url":     s.URLs.Unsubscribe(ws.Slug, unsub),
		"preferences_url":     s.URLs.Preferences(ws.Slug, sign(linktoken.Preferences, 0)),
		"view_in_browser_url": s.URLs.ViewOnline(ws.Slug, sign(linktoken.ViewOnline, 0)),
	}
	if tpl.Opens {
		links[engine.OpenVar] = s.URLs.Open(sign(linktoken.Open, 0))
	}
	for _, l := range tpl.Links {
		links[engine.LinkVar(l.Position)] = s.URLs.Click(sign(linktoken.Click, uint32(l.Position)))
	}
	vars := liquid.Vars(c.SeedEmail, "", "", nil, links)
	r, err := tpl.Render(vars, true)
	if err != nil {
		r = engine.Rendered{Subject: t.Subject, HTML: t.Html, Text: t.TextBody}
	}
	stream := providers.Stream(t.Stream)
	feedback := feedbackID(t.ID.String(), ws.Slug, string(stream))
	if t.ContentID != nil {
		feedback = feedbackID(t.ContentID.String(), ws.Slug, string(stream))
	}
	h := map[string]string{
		"List-Unsubscribe":      "<" + links["unsubscribe_url"] + ">, <mailto:" + t.FromEmail + "?subject=" + url.PathEscape("unsubscribe "+unsub) + ">",
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		"Feedback-ID":           feedback,
		seedbox.TestHeader:      c.Token,
	}
	return providers.Message{ID: c.ID, Stream: stream, From: mail.Address{Name: t.FromName, Address: t.FromEmail}, To: mail.Address{Address: c.SeedEmail},
		Subject: r.Subject, HTML: r.HTML, Text: r.Text, Headers: h, Tags: map[string]string{"placement": t.ID.String(), "workspace": ws.Slug}}
}

// TrackedLink resolves a copy's tracked link to the test's own link at
// that position (tracking redirects clicks of copies without counting them).
func (s *Service) TrackedLink(ctx context.Context, copyID uuid.UUID, position int) (string, bool) {
	c, err := s.Q.PlacementCopyByID(ctx, copyID)
	if err != nil {
		return "", false
	}
	t, err := s.Q.PlacementTestByID(ctx, c.TestID)
	if err != nil {
		return "", false
	}
	var links []engine.TrackedLink
	if json.Unmarshal(t.Links, &links) != nil {
		return "", false
	}
	for _, l := range links {
		if l.Position == position {
			return l.URL, true
		}
	}
	return "", false
}
