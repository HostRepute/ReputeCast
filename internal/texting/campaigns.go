package texting

import (
	"context"
	"net/http"
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
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsAudienceRef is a list or segment in an SMS campaign's audience.
type SmsAudienceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind" enum:"list,segment"`
}

// SmsFunnel is how the audience narrows to who gets texts.
type SmsFunnel struct {
	Base       int64            `json:"base"`
	NoConsent  int64            `json:"noConsent"`
	Suppressed int64            `json:"suppressed"`
	Reasons    map[string]int64 `json:"reasons"`
	Shared     int64            `json:"shared"`
	CanReceive int64            `json:"canReceive"`
	// Skipped are recipients who can receive texts but get nothing from
	// this campaign: their country's sender is pending or has no route.
	Skipped   int64    `json:"skipped"`
	SkippedIn []string `json:"skippedIn"`
	ToSend    int64    `json:"toSend"`
	Assumed   int64    `json:"assumed"`
}

// SmsRouteLine is one part of the composer's route line: "US and Canada
// via Twilio +1 888 555 0142", "Ireland skipped (pending) · 20".
type SmsRouteLine struct {
	Countries []string `json:"countries"`
	Provider  string   `json:"provider"`
	Adapter   string   `json:"adapter"`
	Sender    string   `json:"sender"`
	// Skipped is why these countries get nothing (pending, no_route…).
	Skipped    string `json:"skipped,omitempty"`
	Recipients int64  `json:"recipients"`
}

// SmsCounter is the segment counter.
type SmsCounter struct {
	sms.Count
	// Base is the length without the personalized parts.
	Base int `json:"base"`
	// Fields are the contact fields that change the length.
	Fields []SmsField `json:"fields"`
	// Spread is how many recipients get each number of segments, when
	// fields change the length.
	Spread []SmsSpread `json:"spread"`
	// Longest is the longest personalized text: whose and how long.
	Longest *SmsLongest `json:"longest,omitempty"`
}

// SmsField is a personalization field in the text.
type SmsField struct {
	Key         string `json:"key"`
	Fallback    string `json:"fallback"`
	HasFallback bool   `json:"hasFallback"`
}

// SmsSpread is a number of segments and how many recipients get it.
type SmsSpread struct {
	Segments   int   `json:"segments"`
	Recipients int64 `json:"recipients"`
}

// SmsLongest is the recipient with the longest personalized text.
type SmsLongest struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Characters int    `json:"characters"`
	Segments   int    `json:"segments"`
	// LongFields counts recipients whose field pushes them past one segment.
	Recipients int64   `json:"recipients"`
	Cost       float64 `json:"cost"`
}

// SmsLinkView is a link of the text.
type SmsLinkView struct {
	URL       string `json:"url"`
	Host      string `json:"host"`
	Shortener bool   `json:"shortener"`
	// Length is what it counts once shortened (17 on go.acme.io).
	Length      int    `json:"length"`
	UTMCampaign string `json:"utmCampaign"`
}

// SmsBudgetView is the month's budget around a send.
type SmsBudgetView struct {
	Month    time.Time  `json:"month"`
	Budget   float64    `json:"budget"`
	AlertPct int        `json:"alertPct"`
	Spent    float64    `json:"spent"`
	After    float64    `json:"after"`
	SetBy    string     `json:"setBy,omitempty"`
	SetAt    *time.Time `json:"setAt,omitempty"`
}

// SmsTestNumber is a member's phone for test texts, with its last test.
type SmsTestNumber struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Role     string         `json:"role"`
	Color    string         `json:"color"`
	Phone    string         `json:"phone"`
	Display  string         `json:"display"`
	Country  string         `json:"country"`
	UserID   string         `json:"userId,omitempty"`
	LastTest *SmsTestResult `json:"lastTest,omitempty"`
}

// SmsTestResult is a test text and how it went.
type SmsTestResult struct {
	MessageID string     `json:"messageId"`
	Status    string     `json:"status" enum:"queued,held,skipped,sending,sent,delivered,undelivered,failed,cancelled"`
	At        time.Time  `json:"at"`
	Provider  string     `json:"provider"`
	Sender    string     `json:"sender"`
	SenderID  bool       `json:"senderId"`
	Seconds   *float64   `json:"seconds,omitempty"`
	Delivered *time.Time `json:"delivered,omitempty"`
}

// SmsCampaignAccess is what the viewer may do.
type SmsCampaignAccess struct {
	CanEdit bool `json:"canEdit"`
	// CanSend needs campaigns:send and sms:send; Missing names what the
	// viewer lacks, Helpers who can help.
	CanSend bool     `json:"canSend"`
	Missing []string `json:"missing"`
	Helpers []string `json:"helpers"`
	// CanManageRules is providers:manage (Routing & rules).
	CanManageRules  bool `json:"canManageRules"`
	CanViewContacts bool `json:"canViewContacts"`
}

// SmsCampaign is the SMS composer.
type SmsCampaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	Version     int        `json:"version"`
	Versions    int        `json:"versions"`
	AutosavedAt *time.Time `json:"autosavedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	UpdatedBy   string     `json:"updatedBy"`
	CreatedBy   string     `json:"createdBy"`
	// Body is the editable text between the locked parts.
	Body   string `json:"body"`
	Prefix string `json:"prefix"`
	OptOut string `json:"optOut"`
	// Text is the whole text as written (links not shortened yet).
	Text              string           `json:"text"`
	Audience          []SmsAudienceRef `json:"audience"`
	Funnel            SmsFunnel        `json:"funnel"`
	Routes            []SmsRouteLine   `json:"routes"`
	Counter           SmsCounter       `json:"counter"`
	Links             []SmsLinkView    `json:"links"`
	ShortDomain       string           `json:"shortDomain"`
	ShortDomainStatus string           `json:"shortDomainStatus" enum:"none,pending,verified,failed"`
	Costs             []CountryLine    `json:"costs"`
	Texts             int64            `json:"texts"`
	Segments          int64            `json:"segments"`
	Cost              float64          `json:"cost"`
	PricedOn          *time.Time       `json:"pricedOn,omitempty"`
	PriceProviders    []string         `json:"priceProviders"`
	Budget            SmsBudgetView    `json:"budget"`
	Tests             []SmsTestNumber  `json:"tests"`
	// Schedule is when it goes out (Review & send sets it).
	ScheduleMode string     `json:"scheduleMode" enum:"now,at,local"`
	SendAt       *time.Time `json:"sendAt,omitempty"`
	LocalDate    string     `json:"localDate,omitempty"`
	LocalTime    string     `json:"localTime,omitempty"`
	// Arrives is the first local send time, for the preview.
	Window        string            `json:"window"`
	RetentionDays int               `json:"retentionDays"`
	Warnings      int               `json:"warnings"`
	Blocks        int               `json:"blocks"`
	Access        SmsCampaignAccess `json:"access"`
	Timezone      string            `json:"timezone"`
}

// SmsCampaignCreate makes an SMS campaign.
type SmsCampaignCreate struct {
	Name  string   `json:"name" minLength:"1" maxLength:"200"`
	Lists []string `json:"lists,omitempty" maxItems:"20"`
	Body  string   `json:"body,omitempty" maxLength:"2000"`
}

// SmsCampaignUpdate is an autosave.
type SmsCampaignUpdate struct {
	Name     *string   `json:"name,omitempty" maxLength:"200"`
	Body     *string   `json:"body,omitempty" maxLength:"2000"`
	Lists    *[]string `json:"lists,omitempty" maxItems:"20"`
	Segments *[]string `json:"segments,omitempty" maxItems:"20"`
	// Version is the version the editor started from: a newer save answers
	// 409 version_conflict.
	Version int `json:"version" minimum:"1"`
}

func (s *Service) campaign(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.SmsCampaignRow, error) {
	c, err := s.Q.SmsCampaign(ctx, store.SmsCampaignParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return c, api.MapError(err, "SMS campaign")
	}
	return c, nil
}

// canSend reports whether p may send texts: campaigns:send and sms:send.
func canSend(p *authz.Principal) bool { return p.Can(authz.CampaignsSend) && p.Can(authz.SMSSend) }

func (s *Service) accessFor(ctx context.Context, ws *store.Workspace, p *authz.Principal) SmsCampaignAccess {
	a := SmsCampaignAccess{CanEdit: p.Can(authz.CampaignsEdit), CanSend: canSend(p), Missing: []string{}, Helpers: []string{},
		CanManageRules: p.Can(authz.ProvidersManage), CanViewContacts: p.Can(authz.SubscribersView)}
	if !p.Can(authz.CampaignsSend) {
		a.Missing = append(a.Missing, string(authz.CampaignsSend))
	}
	if !p.Can(authz.SMSSend) {
		a.Missing = append(a.Missing, string(authz.SMSSend))
	}
	if !a.CanSend || !a.CanEdit {
		perm := authz.SMSSend
		if !a.CanEdit {
			perm = authz.CampaignsEdit
		}
		a.Helpers = access.Helpers(ctx, s.Q, ws.ID, perm)
	}
	return a
}

// Create makes an SMS campaign (Regular only), audited like any campaign.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsCampaignCreate) (SmsCampaign, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return SmsCampaign{}, api.Invalid("body.name", "Give the campaign a name.")
	}
	for _, id := range in.Lists {
		if _, err := uuid.Parse(id); err != nil {
			return SmsCampaign{}, api.Invalid("body.lists", "Pick lists of this workspace.")
		}
	}
	return s.create(ctx, ws, p, name, Audience{Lists: in.Lists}, strings.TrimSpace(in.Body), nil)
}

// create inserts a draft with its first version; from is the campaign it
// duplicates.
func (s *Service) create(ctx context.Context, ws *store.Workspace, p *authz.Principal, name string, aud Audience, body string, from *uuid.UUID) (SmsCampaign, error) {
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return SmsCampaign{}, err
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	local := now.In(location(ws.Timezone)).AddDate(0, 0, 1)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	minute := int32(10 * 60)
	if minute < int32(r.Window.From) || minute >= int32(r.Window.To) {
		minute = int32(r.Window.From)
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsCampaignInsert(ctx, store.SmsCampaignInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Audience: jsonOf(aud),
			ContentHash: contentHash(textOf(body, r).Full()), By: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		if err := q.SmsCampaignDetailInsert(ctx, store.SmsCampaignDetailInsertParams{CampaignID: id, WorkspaceID: ws.ID, Body: body, ScheduleMode: "local",
			LocalDate: &day, LocalMinute: &minute}); err != nil {
			return err
		}
		if err := q.SmsCampaignVersionInsert(ctx, store.SmsCampaignVersionInsertParams{CampaignID: id, Version: 1, Body: body, SavedBy: uuidPtr(p.UserID), SavedAt: now}); err != nil {
			return err
		}
		after := map[string]any{"name": name, "channel": "sms", "lists": aud.Lists}
		if len(aud.Segments) > 0 {
			after["segments"] = aud.Segments
		}
		if from != nil {
			after["fromCampaignId"] = from.String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			After: after})
	})
	if err != nil {
		return SmsCampaign{}, err
	}
	return s.Get(ctx, ws, p, id)
}

// Get returns the composer's view of an SMS campaign.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsCampaign{}, err
	}
	rv, err := s.buildReview(ctx, ws, c)
	if err != nil {
		return SmsCampaign{}, err
	}
	return s.composerView(ctx, ws, p, rv)
}

func (s *Service) composerView(ctx context.Context, ws *store.Workspace, p *authz.Principal, rv *review) (SmsCampaign, error) {
	c, g := rv.c, rv.graph
	out := SmsCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, Version: int(c.Version), AutosavedAt: c.AutosavedAt, UpdatedAt: c.UpdatedAt,
		UpdatedBy: c.UpdatedByName, CreatedBy: c.CreatedByName, Body: c.Body, Prefix: rv.text.Prefix, OptOut: rv.text.OptOut, Text: rv.text.Full(),
		Audience: []SmsAudienceRef{}, Routes: []SmsRouteLine{}, Links: []SmsLinkView{}, Costs: rv.byCountry(), PriceProviders: []string{},
		ShortDomain: g.rules.ShortDomain, ShortDomainStatus: g.rules.ShortStatus, ScheduleMode: c.ScheduleMode, SendAt: c.SendAt,
		Window: g.rules.Window.String(), RetentionDays: g.rules.RetentionDays, Access: s.accessFor(ctx, ws, p), Timezone: ws.Timezone, Tests: []SmsTestNumber{}}
	if c.LocalDate != nil {
		out.LocalDate = c.LocalDate.Format("2006-01-02")
	}
	if c.LocalMinute != nil {
		out.LocalTime = channels.Clock(int(*c.LocalMinute))
	}
	_, out.Warnings, out.Blocks = rv.counts()
	if vs, err := s.Q.SmsCampaignVersions(ctx, c.ID); err == nil {
		out.Versions = len(vs)
	}
	aud := parseAudience(c.Audience)
	if ids := uuids(aud.Lists); len(ids) > 0 {
		names, err := s.listNames(ctx, ws, ids)
		if err != nil {
			return out, err
		}
		for _, id := range ids {
			if n, ok := names[id]; ok {
				out.Audience = append(out.Audience, SmsAudienceRef{ID: id.String(), Name: n, Kind: "list"})
			}
		}
	}
	for _, sid := range aud.Segments {
		var name string
		if err := s.Pool.QueryRow(ctx, `SELECT name FROM segments WHERE id = $1 AND workspace_id = $2`, sid, ws.ID).Scan(&name); err == nil {
			out.Audience = append(out.Audience, SmsAudienceRef{ID: sid, Name: name, Kind: "segment"})
		}
	}
	f := rv.funnel
	out.Funnel = SmsFunnel{Base: f.Base, NoConsent: f.NoConsent, Suppressed: f.Suppressed, Reasons: f.SuppressedByReason, Shared: f.Shared, CanReceive: f.CanReceive,
		ToSend: rv.sent, Assumed: f.Assumed, SkippedIn: []string{}}
	for _, l := range out.Costs {
		if l.Skipped > 0 {
			out.Funnel.Skipped += l.Skipped
			out.Funnel.SkippedIn = append(out.Funnel.SkippedIn, l.Country)
		}
		out.Texts += l.Recipients
		out.Segments += l.Recipients * int64(max(l.Segments, 1))
		if l.Provider != "" && l.Recipients > 0 && !slices.Contains(out.PriceProviders, l.Provider) {
			out.PriceProviders = append(out.PriceProviders, l.Provider)
		}
	}
	out.Cost = round2(rv.cost)
	// The route line: one part per route the recipients take.
	type key struct{ sender, skip string }
	parts := map[key]*SmsRouteLine{}
	order := []key{}
	for _, l := range out.Costs {
		k := key{l.Sender, l.SkipReason}
		if l.Recipients > 0 {
			k.skip = ""
		}
		pl, ok := parts[k]
		if !ok {
			pl = &SmsRouteLine{Countries: []string{}, Provider: l.Provider, Adapter: l.Adapter, Sender: l.Sender, Skipped: k.skip}
			parts[k] = pl
			order = append(order, k)
		}
		pl.Countries = append(pl.Countries, l.Country)
		if k.skip == "" {
			pl.Recipients += l.Recipients
		} else {
			pl.Recipients += l.Skipped
		}
	}
	for _, k := range order {
		out.Routes = append(out.Routes, *parts[k])
	}
	if len(out.Routes) == 0 {
		// No recipients yet: the workspace's routes.
		for _, r := range g.routes {
			if sn, ok := g.senders[r.SenderID]; ok {
				out.Routes = append(out.Routes, SmsRouteLine{Countries: r.Countries, Provider: sn.ProviderName, Adapter: sn.ProviderAdapter, Sender: senderLabel(sn)})
			}
		}
	}
	for _, d := range g.prices {
		for _, p := range d {
			if out.PricedOn == nil || p.AsOf.After(*out.PricedOn) {
				t := p.AsOf
				out.PricedOn = &t
			}
		}
	}
	// Counter and personalization.
	m := rv.measured
	out.Counter = SmsCounter{Count: m.Count, Base: m.Base, Fields: []SmsField{}, Spread: []SmsSpread{}}
	for _, u := range m.Fields {
		out.Counter.Fields = append(out.Counter.Fields, SmsField{Key: u.Key, Fallback: u.Fallback, HasFallback: u.HasFallback})
	}
	if len(m.Fields) > 0 && len(f.Recipients) > 0 {
		domain := s.shortDomain(g.rules)
		sp, longest, lc := spread(rv.text, domain, f.Recipients)
		var over int64
		for _, x := range sp {
			out.Counter.Spread = append(out.Counter.Spread, SmsSpread(x))
			if x.Segments > sp[0].Segments {
				over += x.Recipients
			}
		}
		if longest != nil {
			val := ""
			if len(m.Fields) > 0 {
				switch m.Fields[0].Key {
				case "first_name":
					val = longest.FirstName
				case "last_name":
					val = longest.LastName
				default:
					if v, ok := longest.Attributes[m.Fields[0].Key].(string); ok {
						val = v
					}
				}
			}
			out.Counter.Longest = &SmsLongest{Name: strings.TrimSpace(longest.FirstName + " " + longest.LastName), Value: val, Characters: lc.Characters,
				Segments: lc.Segments, Recipients: over, Cost: out.Cost}
		}
	}
	for _, l := range m.Links {
		v := SmsLinkView{URL: l.URL, Host: l.Host, Shortener: l.Shortener, Length: len(l.URL), UTMCampaign: utmCampaign(c.Name)}
		if !l.Shortener && g.rules.ShortDomain != "" {
			v.Length = shortLen(g.rules.ShortDomain)
		}
		out.Links = append(out.Links, v)
	}
	// Budget.
	loc := location(ws.Timezone)
	out.Budget = SmsBudgetView{Month: monthStart(s.now(), loc), Budget: g.rules.Budget, AlertPct: g.rules.BudgetAlertPct, Spent: round2(rv.spent),
		After: round2(rv.spent + rv.cost), SetAt: g.rules.BudgetSetAt}
	if c.Status == "sent" || c.Status == "sending" || c.Status == "paused" {
		// What the send cost is part of the month's spend already.
		var sent float64
		_ = s.Pool.QueryRow(ctx, `SELECT coalesce(round(sum(cost), 4), 0)::float8 FROM channel_messages WHERE campaign_id = $1`, c.ID).Scan(&sent)
		out.Budget.Spent = round2(rv.spent - sent)
		out.Budget.After = round2(rv.spent)
	}
	if g.rules.BudgetSetBy != nil {
		if u, err := s.Q.GetUser(ctx, *g.rules.BudgetSetBy); err == nil {
			out.Budget.SetBy = u.Name
		}
	}
	tests, err := s.testNumbers(ctx, ws, &c.ID)
	if err != nil {
		return out, err
	}
	out.Tests = tests
	return out, nil
}

// Update autosaves the composer: name, text and audience. A text change
// makes a new version.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsCampaignUpdate) (SmsCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsCampaign{}, err
	}
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
	default:
		return SmsCampaign{}, api.Conflict("This campaign already went out; duplicate it to send it again.")
	}
	if in.Version != int(c.Version) {
		return SmsCampaign{}, api.Err(http.StatusConflict, "version_conflict", "Someone saved a newer version of this text. Reload to see it.").
			WithMeta("version", c.Version, "by", c.UpdatedByName)
	}
	name, body := c.Name, c.Body
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			return SmsCampaign{}, api.Invalid("body.name", "Give the campaign a name.")
		}
		name = strings.TrimSpace(*in.Name)
	}
	if in.Body != nil {
		body = strings.TrimSpace(*in.Body)
	}
	aud := parseAudience(c.Audience)
	if in.Lists != nil {
		for _, l := range *in.Lists {
			if _, err := uuid.Parse(l); err != nil {
				return SmsCampaign{}, api.Invalid("body.lists", "Pick lists of this workspace.")
			}
		}
		aud.Lists = *in.Lists
	}
	if in.Segments != nil {
		for _, l := range *in.Segments {
			if _, err := uuid.Parse(l); err != nil {
				return SmsCampaign{}, api.Invalid("body.segments", "Pick segments of this workspace.")
			}
		}
		aud.Segments = *in.Segments
	}
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return SmsCampaign{}, err
	}
	version := c.Version
	if body != c.Body {
		version++
	}
	now := s.now()
	before := map[string]any{"name": c.Name, "body": c.Body, "audience": c.Audience}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsCampaignSave(ctx, store.SmsCampaignSaveParams{Name: name, Audience: jsonOf(aud), ContentHash: contentHash(textOf(body, r).Full()), Version: version,
			By: uuidPtr(p.UserID), At: now, ID: c.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		if err := q.SmsCampaignBodySave(ctx, store.SmsCampaignBodySaveParams{Body: body, CampaignID: c.ID}); err != nil {
			return err
		}
		if version != c.Version {
			if err := q.SmsCampaignVersionInsert(ctx, store.SmsCampaignVersionInsertParams{CampaignID: c.ID, Version: version, Body: body, SavedBy: uuidPtr(p.UserID), SavedAt: now}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: c.ID.String(), TargetName: name,
			Before: before, After: map[string]any{"name": name, "body": body, "audience": aud}, Metadata: map[string]any{"channel": "sms", "version": version}})
	})
	if err != nil {
		return SmsCampaign{}, err
	}
	if s.Changed != nil {
		s.Changed(ctx, ws, c.ID)
	}
	return s.Get(ctx, ws, p, c.ID)
}

// SmsCampaignVersion is a saved version of the text.
type SmsCampaignVersion struct {
	Version int       `json:"version"`
	Body    string    `json:"body"`
	SavedBy string    `json:"savedBy"`
	SavedAt time.Time `json:"savedAt"`
}

// Versions lists a campaign's versions, newest first.
func (s *Service) Versions(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]SmsCampaignVersion, error) {
	if _, err := s.campaign(ctx, ws, id); err != nil {
		return nil, err
	}
	vs, err := s.Q.SmsCampaignVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]SmsCampaignVersion, 0, len(vs))
	for _, v := range vs {
		out = append(out, SmsCampaignVersion{Version: int(v.Version), Body: v.Body, SavedBy: v.SavedByName, SavedAt: v.SavedAt})
	}
	return out, nil
}
