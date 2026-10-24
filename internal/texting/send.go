package texting

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsThresholds are a campaign's auto-pause thresholds, in percent.
type SmsThresholds struct {
	Failed   float64 `json:"failed" minimum:"0.1" maximum:"100"`
	Filtered float64 `json:"filtered" minimum:"0.1" maximum:"100"`
	OptOuts  float64 `json:"optOuts" minimum:"0.1" maximum:"100"`
}

// MinSentForPause: a rate counts once this many texts went out.
const MinSentForPause = 50

func thresholdsOf(raw json.RawMessage, r Rules) SmsThresholds {
	t := SmsThresholds{Failed: r.PauseFailed, Filtered: r.PauseFiltered, OptOuts: r.PauseOptOut}
	var c SmsThresholds
	if json.Unmarshal(raw, &c) == nil {
		if c.Failed > 0 {
			t.Failed = c.Failed
		}
		if c.Filtered > 0 {
			t.Filtered = c.Filtered
		}
		if c.OptOuts > 0 {
			t.OptOuts = c.OptOuts
		}
	}
	return t
}

// SmsZoneGroup is a group of recipients who get the text at the same
// moment or in one range: "08:00 UTC · Germany, Netherlands · 77".
type SmsZoneGroup struct {
	Countries  []string  `json:"countries"`
	Recipients int64     `json:"recipients"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	// LocalFrom and LocalTo: the local clock times for "Now" ("13:40").
	LocalFrom string `json:"localFrom"`
	LocalTo   string `json:"localTo"`
	// Held is true when quiet hours hold them; Until is when they go.
	Held  bool      `json:"held"`
	Until time.Time `json:"until"`
}

// SmsReviewRoute is a provider's share of the send.
type SmsReviewRoute struct {
	Provider   string  `json:"provider"`
	Adapter    string  `json:"adapter"`
	Sender     string  `json:"sender"`
	Recipients int64   `json:"recipients"`
	Share      float64 `json:"share"`
	Rate       float64 `json:"rate"`
	Cost       float64 `json:"cost"`
	// Seconds is how long its texts take at its throughput.
	Seconds float64 `json:"seconds"`
}

// SmsSchedule is when a campaign goes out.
type SmsSchedule struct {
	Mode      string     `json:"mode" enum:"now,at,local"`
	SendAt    *time.Time `json:"sendAt,omitempty"`
	LocalDate string     `json:"localDate,omitempty"`
	LocalTime string     `json:"localTime,omitempty"`
	// Window is the quiet-hours window ("09:00–20:00"); only times inside
	// it are offered for the local schedule.
	Window     string         `json:"window"`
	WindowFrom int            `json:"windowFrom"`
	WindowTo   int            `json:"windowTo"`
	Groups     []SmsZoneGroup `json:"groups"`
	// Now is the groups if the send started now (the "Now" option's note).
	Now []SmsZoneGroup `json:"now"`
	// End is when the last text goes, by the senders' throughput.
	End *time.Time `json:"end,omitempty"`
}

// SmsLarge is the typed-confirmation rule.
type SmsLarge struct {
	Recipients int64   `json:"recipients"`
	Cost       float64 `json:"cost"`
	Required   bool    `json:"required"`
}

// SmsReviewPolicy is what the approval note quotes.
type SmsReviewPolicy struct {
	AudienceMin     int64   `json:"audienceMin"`
	AudienceRule    bool    `json:"audienceRule"`
	CostRule        bool    `json:"costRule"`
	CostAmount      float64 `json:"costAmount"`
	AuthorLacksSend bool    `json:"authorLacksSend"`
}

// SmsSendView is a campaign's send, once scheduled.
type SmsSendView struct {
	Phase       string     `json:"phase" enum:"scheduled,sending,paused,completed,cancelled"`
	ScheduledAt time.Time  `json:"scheduledAt"`
	StartedBy   string     `json:"startedBy"`
	PausedKind  string     `json:"pausedKind" enum:"auto,person,outage,budget,"`
	PausedAt    *time.Time `json:"pausedAt,omitempty"`
}

// SmsReview is Review & send for an SMS campaign.
type SmsReview struct {
	Campaign      SmsReviewCampaign        `json:"campaign"`
	Verdict       string                   `json:"verdict" enum:"ready,warnings,blocked"`
	Passed        int                      `json:"passed"`
	Warnings      int                      `json:"warnings"`
	Blocks        int                      `json:"blocks"`
	CheckedAt     time.Time                `json:"checkedAt"`
	Checks        []SmsCheck               `json:"checks"`
	Funnel        SmsFunnel                `json:"funnel"`
	Lists         []string                 `json:"lists"`
	Routes        []SmsReviewRoute         `json:"routes"`
	Costs         []CountryLine            `json:"costs"`
	Texts         int64                    `json:"texts"`
	Segments      int64                    `json:"segments"`
	SegmentsEach  int                      `json:"segmentsEach"`
	Cost          float64                  `json:"cost"`
	PricedOn      *time.Time               `json:"pricedOn,omitempty"`
	Budget        SmsBudgetView            `json:"budget"`
	Schedule      SmsSchedule              `json:"schedule"`
	PendingPolicy string                   `json:"pendingPolicy" enum:"skip,wait"`
	Thresholds    SmsThresholds            `json:"thresholds"`
	Defaults      SmsThresholds            `json:"defaults"`
	Tests         []SmsTestNumber          `json:"tests"`
	Approval      approvals.ReviewApproval `json:"approval"`
	Policy        SmsReviewPolicy          `json:"policy"`
	Access        SmsCampaignAccess        `json:"access"`
	Large         SmsLarge                 `json:"large"`
	Fingerprint   string                   `json:"fingerprint"`
	Send          *SmsSendView             `json:"send,omitempty"`
	Preview       SmsPreview               `json:"preview"`
	Timezone      string                   `json:"timezone"`
}

// SmsReviewCampaign is the campaign as Review & send shows it.
type SmsReviewCampaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	Version     int        `json:"version"`
	AutosavedAt *time.Time `json:"autosavedAt,omitempty"`
	UpdatedBy   string     `json:"updatedBy"`
	Text        string     `json:"text"`
}

// Review builds Review & send.
func (s *Service) Review(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsReview, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsReview{}, err
	}
	rv, err := s.buildReview(ctx, ws, c)
	if err != nil {
		return SmsReview{}, err
	}
	return s.reviewView(ctx, ws, p, rv)
}

func (s *Service) reviewView(ctx context.Context, ws *store.Workspace, p *authz.Principal, rv *review) (SmsReview, error) {
	c, g := rv.c, rv.graph
	comp, err := s.composerView(ctx, ws, p, rv)
	if err != nil {
		return SmsReview{}, err
	}
	pass, warn, block := rv.counts()
	out := SmsReview{Campaign: SmsReviewCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, Version: int(c.Version), AutosavedAt: c.AutosavedAt,
		UpdatedBy: c.UpdatedByName, Text: rv.text.Full()},
		Verdict: rv.verdict, Passed: pass, Warnings: warn, Blocks: block, CheckedAt: rv.at, Checks: rv.checks, Funnel: comp.Funnel, Lists: rv.lists,
		Routes: []SmsReviewRoute{}, Costs: comp.Costs, Texts: comp.Texts, Segments: comp.Segments, SegmentsEach: rv.measured.Count.Segments, Cost: comp.Cost,
		PricedOn: comp.PricedOn, Budget: comp.Budget, PendingPolicy: c.PendingPolicy, Thresholds: thresholdsOf(c.Thresholds, g.rules),
		Defaults: SmsThresholds{Failed: g.rules.PauseFailed, Filtered: g.rules.PauseFiltered, OptOuts: g.rules.PauseOptOut}, Tests: comp.Tests,
		Access: comp.Access, Fingerprint: rv.fingerprint(), Timezone: ws.Timezone}
	out.Large = SmsLarge{Recipients: LargeRecipients, Cost: LargeCost, Required: rv.sent > LargeRecipients || rv.cost > LargeCost}
	// Providers' shares and how long their texts take.
	type acc struct {
		r     SmsReviewRoute
		order int
	}
	byProv := map[uuid.UUID]*acc{}
	for _, pl := range rv.plan {
		if pl.decision.Sender == nil {
			continue
		}
		sn := pl.decision.Sender
		a, ok := byProv[sn.ProviderID]
		if !ok {
			a = &acc{r: SmsReviewRoute{Provider: sn.ProviderName, Adapter: sn.ProviderAdapter, Sender: senderLabel(*sn), Rate: sn.Throughput}, order: len(byProv)}
			byProv[sn.ProviderID] = a
		}
		a.r.Recipients++
		a.r.Cost += pl.cost
	}
	routes := make([]*acc, 0, len(byProv))
	for _, a := range byProv {
		routes = append(routes, a)
	}
	// The provider with the most texts first (Twilio 659 · Vonage 195).
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].r.Recipients != routes[j].r.Recipients {
			return routes[i].r.Recipients > routes[j].r.Recipients
		}
		return routes[i].order < routes[j].order
	})
	for _, a := range routes {
		a.r.Cost = round2(a.r.Cost)
		if rv.sent > 0 {
			a.r.Share = math.Round(float64(a.r.Recipients)*1000/float64(rv.sent)) / 10
		}
		if a.r.Rate > 0 {
			a.r.Seconds = math.Ceil(float64(a.r.Recipients) / a.r.Rate)
		}
		out.Routes = append(out.Routes, a.r)
	}
	out.Schedule = s.schedule(rv)
	if out.Approval, err = s.approvalView(ctx, ws, p, c.ID); err != nil {
		return out, err
	}
	pol := approvals.ParsePolicy(ws.ApprovalPolicy)
	out.Policy = SmsReviewPolicy{AudienceMin: pol.AudienceMin.N, AudienceRule: pol.AudienceMin.Enabled, CostRule: pol.ChannelCost.Enabled,
		CostAmount: pol.ChannelCost.Amount, AuthorLacksSend: pol.AuthorLacksSend}
	if snd, err := s.Q.SmsSend(ctx, c.ID); err == nil {
		v := &SmsSendView{Phase: snd.Phase, ScheduledAt: snd.ScheduledAt, PausedKind: snd.PausedKind, PausedAt: snd.PausedAt}
		if snd.StartedBy != nil {
			if u, err := s.Q.GetUser(ctx, *snd.StartedBy); err == nil {
				v.StartedBy = u.Name
			}
		}
		out.Send = v
	}
	// The preview in the blocked opt-out state shows the text as it would go.
	first := ""
	for _, l := range comp.Costs {
		if l.Recipients > 0 {
			first = l.Country
			break
		}
	}
	if pv, err := s.Preview(ctx, ws, nil, c.ID, SmsPreviewInput{Country: first}); err == nil {
		out.Preview = pv
	}
	return out, nil
}

func (s *Service) approvalView(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (approvals.ReviewApproval, error) {
	if s.Approvals == nil {
		return approvals.ReviewApproval{Rules: []approvals.ApprovalRuleMatch{}, Approvers: []approvals.ApprovalPerson{}, PolicyRules: []string{}}, nil
	}
	return s.Approvals.ChannelReview(ctx, ws, p, id)
}

// schedule works out when each group of recipients gets the text.
func (s *Service) schedule(rv *review) SmsSchedule {
	c, w := rv.c, rv.graph.rules.Window
	out := SmsSchedule{Mode: c.ScheduleMode, SendAt: c.SendAt, Window: w.String(), WindowFrom: w.From, WindowTo: w.To, Groups: []SmsZoneGroup{}, Now: []SmsZoneGroup{}}
	if c.LocalDate != nil {
		out.LocalDate = c.LocalDate.Format("2006-01-02")
	}
	if c.LocalMinute != nil {
		out.LocalTime = channels.Clock(int(*c.LocalMinute))
	}
	now := s.now()
	times := map[int]time.Time{}
	for i, pl := range rv.plan {
		if pl.skip != "" {
			continue
		}
		times[i] = s.sendTime(c, w, pl.r, now)
	}
	out.Groups = groupTimes(rv.plan, times, w, false)
	nowTimes := map[int]time.Time{}
	for i, pl := range rv.plan {
		if pl.skip == "" {
			nowTimes[i] = now
		}
	}
	out.Now = groupTimes(rv.plan, nowTimes, w, true)
	// The end: each group starts at its time and sends at its senders'
	// throughput.
	var end time.Time
	for _, g := range out.Groups {
		rate := 1.0
		for _, pl := range rv.plan {
			if pl.decision.Sender != nil && slices.Contains(g.Countries, pl.r.Country) {
				rate = math.Max(rate, pl.decision.Sender.Throughput)
				break
			}
		}
		e := g.To.Add(time.Duration(float64(g.Recipients)/rate*float64(time.Second)) / time.Duration(max(1, len(g.Countries))))
		if e.After(end) {
			end = e
		}
	}
	if !end.IsZero() {
		out.End = &end
	}
	return out
}

// sendTime is when a recipient's text goes, quiet hours applied.
func (s *Service) sendTime(c store.SmsCampaignRow, w channels.Window, r Recipient, now time.Time) time.Time {
	loc := location(r.Zone)
	switch c.ScheduleMode {
	case "local":
		if c.LocalDate != nil && c.LocalMinute != nil {
			t, _ := w.LocalSend(*c.LocalDate, int(*c.LocalMinute), loc)
			if t.Before(now) {
				t = w.NextOpen(now, loc)
			}
			return w.NextOpen(t, loc)
		}
	case "at":
		if c.SendAt != nil {
			return w.NextOpen(*c.SendAt, loc)
		}
	}
	return w.NextOpen(now, loc)
}

// groupTimes groups recipients by country, then merges countries whose
// moments overlap (Germany and the Netherlands at 08:00 UTC; Canada and the
// US from 13:00 to 17:00 UTC). With now, the groups say who quiet hours hold.
func groupTimes(plan []planned, times map[int]time.Time, w channels.Window, now bool) []SmsZoneGroup {
	type cg struct {
		g        SmsZoneGroup
		localMin int
		localMax int
	}
	by := map[string]*cg{}
	for i, t := range times {
		pl := plan[i]
		key := pl.r.Country
		held := false
		if now {
			held = !w.Inside(t, location(pl.r.Zone))
			if held {
				key += "|held"
			}
		}
		x, ok := by[key]
		if !ok {
			x = &cg{g: SmsZoneGroup{Countries: []string{pl.r.Country}, From: t, To: t, Held: held}, localMin: 24 * 60, localMax: -1}
			by[key] = x
		}
		x.g.Recipients++
		if t.Before(x.g.From) {
			x.g.From = t
		}
		if t.After(x.g.To) {
			x.g.To = t
		}
		lt := t.In(location(pl.r.Zone))
		m := lt.Hour()*60 + lt.Minute()
		x.localMin, x.localMax = min(x.localMin, m), max(x.localMax, m)
		if held {
			u := w.NextOpen(t, location(pl.r.Zone))
			if x.g.Until.IsZero() || u.Before(x.g.Until) {
				x.g.Until = u
			}
		}
	}
	groups := make([]*cg, 0, len(by))
	for _, x := range by {
		groups = append(groups, x)
	}
	sort.Slice(groups, func(i, j int) bool {
		if !groups[i].g.From.Equal(groups[j].g.From) {
			return groups[i].g.From.Before(groups[j].g.From)
		}
		return groups[i].g.Countries[0] < groups[j].g.Countries[0]
	})
	merged := []*cg{}
	for _, x := range groups {
		if n := len(merged); n > 0 {
			last := merged[n-1]
			overlap := !x.g.From.After(last.g.To) && last.g.Held == x.g.Held
			if now {
				overlap = last.g.Held == x.g.Held && (!x.g.Held || last.localMin == x.localMin)
			}
			if overlap {
				last.g.Countries = append(last.g.Countries, x.g.Countries...)
				last.g.Recipients += x.g.Recipients
				if x.g.To.After(last.g.To) {
					last.g.To = x.g.To
				}
				last.localMin, last.localMax = min(last.localMin, x.localMin), max(last.localMax, x.localMax)
				continue
			}
		}
		merged = append(merged, x)
	}
	out := make([]SmsZoneGroup, 0, len(merged))
	for _, x := range merged {
		sort.Strings(x.g.Countries)
		x.g.LocalFrom, x.g.LocalTo = channels.Clock(x.localMin), channels.Clock(x.localMax)
		out = append(out, x.g)
	}
	return out
}

// SmsSendSettings changes when and how a campaign goes out.
type SmsSendSettings struct {
	Mode          string         `json:"mode,omitempty" enum:"now,at,local"`
	SendAt        *time.Time     `json:"sendAt,omitempty"`
	LocalDate     string         `json:"localDate,omitempty" pattern:"^\\d{4}-\\d{2}-\\d{2}$"`
	LocalTime     string         `json:"localTime,omitempty" pattern:"^([01]\\d|2[0-3]):[0-5]\\d$"`
	PendingPolicy string         `json:"pendingPolicy,omitempty" enum:"skip,wait"`
	Thresholds    *SmsThresholds `json:"thresholds,omitempty"`
}

// SaveSendSettings stores the schedule, the pending-sender choice and the
// thresholds. Only times inside the quiet-hours window are accepted.
func (s *Service) SaveSendSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsSendSettings) (SmsReview, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsReview{}, err
	}
	switch c.Status {
	case "draft", "approval", "approved":
	default:
		return SmsReview{}, api.Conflict(say(ctx, mSendSettingsLocked, nil))
	}
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return SmsReview{}, err
	}
	mode, sendAt, localDate, localMinute := c.ScheduleMode, c.SendAt, c.LocalDate, c.LocalMinute
	if in.Mode != "" {
		mode = in.Mode
	}
	switch mode {
	case "at":
		if in.SendAt != nil {
			sendAt = in.SendAt
		}
		if sendAt == nil || sendAt.Before(s.now()) {
			return SmsReview{}, api.Invalid("body.sendAt", say(ctx, mFutureTime, nil))
		}
	case "local":
		if in.LocalDate != "" {
			d, err := time.Parse("2006-01-02", in.LocalDate)
			if err != nil {
				return SmsReview{}, api.Invalid("body.localDate", say(ctx, mPickDate, nil))
			}
			localDate = &d
		}
		if in.LocalTime != "" {
			t, _ := time.Parse("15:04", in.LocalTime)
			m := int32(t.Hour()*60 + t.Minute())
			if int(m) < r.Window.From || int(m) >= r.Window.To {
				return SmsReview{}, api.Invalid("body.localTime", say(ctx, mInsideWindow, i18n.Args{"window": r.Window.String()}))
			}
			localMinute = &m
		}
		if localDate == nil || localMinute == nil {
			return SmsReview{}, api.Invalid("body.localDate", say(ctx, mDateAndTime, nil))
		}
	}
	policy := c.PendingPolicy
	if in.PendingPolicy != "" {
		policy = in.PendingPolicy
	}
	th := thresholdsOf(c.Thresholds, r)
	thRaw := c.Thresholds
	if in.Thresholds != nil {
		th = *in.Thresholds
		thRaw = jsonOf(th)
	}
	before := map[string]any{"mode": c.ScheduleMode, "pendingPolicy": c.PendingPolicy, "thresholds": c.Thresholds}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsCampaignScheduleSave(ctx, store.SmsCampaignScheduleSaveParams{ScheduleMode: mode, SendAt: sendAt, LocalDate: localDate, LocalMinute: localMinute,
			PendingPolicy: policy, Thresholds: thRaw, CampaignID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_settings", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: before, After: map[string]any{"mode": mode, "pendingPolicy": policy, "thresholds": th}, Metadata: map[string]any{"channel": "sms"}})
	})
	if err != nil {
		return SmsReview{}, err
	}
	if s.Changed != nil {
		s.Changed(ctx, ws, c.ID)
	}
	return s.Review(ctx, ws, p, c.ID)
}

// SmsSendInput sends or schedules a campaign as reviewed.
type SmsSendInput struct {
	Fingerprint string `json:"fingerprint" minLength:"1"`
	// Confirm is the typed recipient count, for sends above 1,000
	// recipients or $100.
	Confirm string `json:"confirm,omitempty"`
	// Acknowledged are the warnings' keys the person confirmed.
	Acknowledged []string `json:"acknowledged,omitempty"`
}

// SmsSendResult is what happened.
type SmsSendResult struct {
	Status      string    `json:"status" enum:"scheduled,sending"`
	Recipients  int64     `json:"recipients"`
	ScheduledAt time.Time `json:"scheduledAt"`
}

func digitsOf(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// Send schedules a campaign's texts as reviewed: each recipient's text is
// queued for its moment (quiet hours applied), skipped or held.
func (s *Service) Send(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsSendInput) (SmsSendResult, error) {
	if !p.Can(authz.SMSSend) {
		return SmsSendResult{}, api.Forbidden(authz.SMSSend, s.helpers(ctx, ws, authz.SMSSend))
	}
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsSendResult{}, err
	}
	switch c.Status {
	case "draft", "approval", "approved":
	case "scheduled":
		return SmsSendResult{}, api.Conflict(say(ctx, mAlreadyScheduled, nil))
	default:
		return SmsSendResult{}, api.Conflict(say(ctx, mAlreadyWentOut, nil))
	}
	rv, err := s.buildReview(ctx, ws, c)
	if err != nil {
		return SmsSendResult{}, err
	}
	if rv.fingerprint() != in.Fingerprint {
		return SmsSendResult{}, api.Err(http.StatusConflict, "review_changed", say(ctx, mReviewChanged, nil))
	}
	if rv.verdict == VerdictBlocked {
		_, _, b := rv.counts()
		return SmsSendResult{}, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mBlocked, i18n.Args{"count": b}))
	}
	missing := []string{}
	for _, ch := range rv.checks {
		if ch.Status == Warn && !slices.Contains(in.Acknowledged, ch.Key) {
			missing = append(missing, ch.Key)
		}
	}
	if len(missing) > 0 {
		return SmsSendResult{}, api.Invalid("body.acknowledged", say(ctx, mConfirmWarnings, i18n.Args{"warnings": strings.Join(missing, ", ")}))
	}
	if (rv.sent > LargeRecipients || rv.cost > LargeCost) && digitsOf(in.Confirm) != fmt.Sprint(rv.sent) {
		return SmsSendResult{}, api.Invalid("body.confirm", say(ctx, mTypeCount, i18n.Args{"count": fmt.Sprint(rv.sent)}))
	}
	var approval *store.ApprovalRequest
	if s.Approvals != nil {
		if approval, err = s.Approvals.ChannelGate(ctx, ws, p, c.ID); err != nil {
			return SmsSendResult{}, err
		}
	}
	return s.schedule2(ctx, ws, p, rv, approval)
}

// schedule2 writes the send: a message per recipient, the send row, the
// campaign's status, the audit entry.
func (s *Service) schedule2(ctx context.Context, ws *store.Workspace, p *authz.Principal, rv *review, approval *store.ApprovalRequest) (SmsSendResult, error) {
	c, w := rv.c, rv.graph.rules.Window
	now := s.now()
	domain := s.shortDomain(rv.graph.rules)
	campaignName := utmCampaign(c.Name)
	first := time.Time{}
	var skipped int64
	rows := [][]any{}
	links := [][]any{}
	for _, pl := range rv.plan {
		id := uuid.Must(uuid.NewV7())
		r := pl.r
		status, reason := "queued", ""
		var sendAfter, expires *time.Time
		var senderID, providerID *uuid.UUID
		sender := ""
		body := ""
		price := pl.decision.Price
		cost := pl.cost
		units := pl.segments
		switch {
		case pl.skip == "pending" && c.PendingPolicy == "wait":
			status, reason = "held", "pending_sender"
			e := now.AddDate(0, 0, 7)
			expires = &e
			if pl.decision.Route != nil {
				if sn, ok := rv.graph.senders[pl.decision.Route.SenderID]; ok {
					senderID, providerID, sender = &sn.ID, &sn.ProviderID, sn.Value
				}
			}
			cost = 0
		case pl.skip != "":
			status, reason = "skipped", pl.skip
			if pl.skip == "pending" || pl.skip == "rejected" || pl.skip == "not_registered" || pl.skip == "not_supported" {
				reason = "sender_" + pl.skip
			}
			cost, price = 0, 0
			skipped++
		default:
			t := s.sendTime(c, w, r, now)
			sendAfter = &t
			e := w.Closes(t, location(r.Zone))
			expires = &e
			if first.IsZero() || t.Before(first) {
				first = t
			}
			sn := pl.decision.Sender
			senderID, providerID, sender = &sn.ID, &sn.ProviderID, sn.Value
		}
		if status != "skipped" {
			b := rv.text.Body
			if domain != "" {
				b = withShortLinks(b, domain, func(i int, l Link) string {
					code := LinkCode(c.ID, r.ContactID, i)
					links = append(links, []any{code, ws.ID, id, c.ID, r.ContactID, l.URL, target(l.URL, campaignName), now, now.AddDate(1, 0, 0)})
					return code
				})
			}
			body = Text{Prefix: rv.text.Prefix, Body: render(b, r), OptOut: rv.text.OptOut}.Full()
			cnt := sms.CountText(body)
			units = max(cnt.Segments, 1)
		}
		var bodyPtr *string
		if body != "" {
			bodyPtr = &body
		}
		var pricePtr *float64
		if price > 0 {
			pricePtr = &price
		}
		rows = append(rows, []any{id, ws.ID, "sms", "campaign", c.ID, r.ContactID, r.ListID, r.Phone, r.Country, r.Zone, string(r.ZoneSource), providerID, senderID,
			sender, bodyPtr, int32(units), string(rv.measured.Count.Encoding), pricePtr, cost, status, reason, sendAfter, expires, now, now})
	}
	if first.IsZero() {
		first = now
	}
	status := "scheduled"
	if !first.After(now.Add(time.Minute)) {
		status = "sending"
	}
	leftOut := map[string]any{"base": rv.funnel.Base, "noConsent": rv.funnel.NoConsent, "suppressed": rv.funnel.Suppressed, "reasons": rv.funnel.SuppressedByReason,
		"shared": rv.funnel.Shared, "canReceive": rv.funnel.CanReceive, "cost": rv.cost}
	var approvedBy *uuid.UUID
	if approval != nil {
		approvedBy = approval.DecidedBy
	}
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		// The campaign is locked: a second send can't slip in.
		var cur string
		if err := tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id = $1 FOR UPDATE`, c.ID).Scan(&cur); err != nil {
			return err
		}
		if cur != c.Status {
			return api.Conflict(say(ctx, mChangedWhileSending, nil))
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"channel_messages"}, []string{"id", "workspace_id", "channel", "kind", "campaign_id", "contact_id", "list_id",
			"recipient", "country", "time_zone", "zone_source", "provider_id", "sender_id", "sender", "body", "units", "encoding", "price", "cost", "status", "reason",
			"send_after", "expires_at", "created_at", "updated_at"}, pgx.CopyFromRows(rows)); err != nil {
			return err
		}
		if len(links) > 0 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"sms_links"}, []string{"code", "workspace_id", "message_id", "campaign_id", "contact_id", "original", "target",
				"created_at", "expires_at"}, pgx.CopyFromRows(links)); err != nil {
				return err
			}
		}
		if err := q.SmsSendUpsert(ctx, store.SmsSendUpsertParams{CampaignID: c.ID, WorkspaceID: ws.ID, Phase: status, Recipients: int32(rv.sent), Skipped: int32(skipped),
			LeftOut: jsonOf(leftOut), StartedBy: uuidPtr(p.UserID), ApprovedBy: approvedBy, ScheduledAt: first, Thresholds: jsonOf(thresholdsOf(c.Thresholds, rv.graph.rules))}); err != nil {
			return err
		}
		var startedAt *time.Time
		if status == "sending" {
			startedAt = &now
		}
		if err := q.SmsCampaignSetStatus(ctx, store.SmsCampaignSetStatusParams{Status: status, ScheduledAt: &first, StartedAt: startedAt, StartedBy: uuidPtr(p.UserID),
			At: now, ID: c.ID}); err != nil {
			return err
		}
		action := "campaign.scheduled"
		if status == "sending" {
			action = "campaign.send_started"
		}
		meta := map[string]any{"channel": "sms", "recipients": rv.sent, "skipped": skipped, "cost": rv.cost, "schedule": scheduleLabel(c, location(ws.Timezone))}
		if approval != nil {
			meta["approval_id"] = approval.ID.String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": status, "scheduled_at": first}, Metadata: meta})
	})
	if err != nil {
		return SmsSendResult{}, err
	}
	s.kick(ctx)
	return SmsSendResult{Status: status, Recipients: rv.sent, ScheduledAt: first}, nil
}

func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission) []string {
	return access.Helpers(ctx, s.Q, ws.ID, perm)
}

// Unschedule takes a scheduled campaign back to a draft: nothing went yet.
func (s *Service) Unschedule(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	if c.Status != "scheduled" {
		return api.Conflict(say(ctx, mUnscheduleOnly, nil))
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var sent int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status NOT IN ('queued','held','skipped')`, c.ID).Scan(&sent); err != nil {
			return err
		}
		if sent > 0 {
			return api.Conflict(say(ctx, mTextsWentOut, nil))
		}
		if _, err := tx.Exec(ctx, `DELETE FROM sms_links WHERE campaign_id = $1`, c.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM channel_messages WHERE campaign_id = $1`, c.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM sms_sends WHERE campaign_id = $1`, c.ID); err != nil {
			return err
		}
		if err := q.SmsCampaignSetStatus(ctx, store.SmsCampaignSetStatusParams{Status: "draft", At: now, ID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.unscheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "draft"}, Metadata: map[string]any{"channel": "sms"}})
	})
}

// SmsSendAction is a person's action on a send: pause, resume or cancel.
type SmsSendAction struct {
	Action string `json:"action" enum:"pause,resume,cancel"`
	// Reason is optional ("Resume with changes" notes what changed).
	Reason string `json:"reason,omitempty" maxLength:"500"`
}

// Act pauses, resumes or cancels a send. Pausing needs campaigns:pause;
// resuming needs campaigns:send and sms:send; cancelling either.
func (s *Service) Act(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsSendAction) error {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	snd, err := s.Q.SmsSend(ctx, c.ID)
	if err != nil {
		return api.Conflict(say(ctx, mNotSent, nil))
	}
	now := s.now()
	switch in.Action {
	case "pause":
		if !p.Can(authz.CampaignsPause) {
			return api.Forbidden(authz.CampaignsPause, s.helpers(ctx, ws, authz.CampaignsPause))
		}
		if snd.Phase != "sending" && snd.Phase != "scheduled" {
			return api.Conflict(say(ctx, mPauseOnly, nil))
		}
		return s.setPhase(ctx, ws, c, snd, "paused", "person", map[string]any{"by": p.Name}, &audit.Event{Action: "campaign.paused", Reason: in.Reason})
	case "resume":
		if !canSend(p) {
			perm := authz.CampaignsSend
			if p.Can(authz.CampaignsSend) {
				perm = authz.SMSSend
			}
			return api.Forbidden(perm, s.helpers(ctx, ws, perm))
		}
		if snd.Phase != "paused" {
			return api.Conflict(say(ctx, mResumeOnly, nil))
		}
		// Texts held past their window wait for the next one.
		if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'queued', reason = '', updated_at = $2
			WHERE campaign_id = $1 AND status = 'held' AND reason IN ('paused','provider_down','budget')`, c.ID, now); err != nil {
			return err
		}
		phase := "sending"
		if err := s.setPhase(ctx, ws, c, snd, phase, "", map[string]any{}, &audit.Event{Action: "campaign.resumed", Reason: in.Reason}); err != nil {
			return err
		}
		s.kick(ctx)
		return nil
	case "cancel":
		if !p.Can(authz.CampaignsPause) && !canSend(p) {
			return api.Forbidden(authz.CampaignsPause, s.helpers(ctx, ws, authz.CampaignsPause))
		}
		if snd.Phase == "completed" || snd.Phase == "cancelled" {
			return api.Conflict(say(ctx, mSendOver, nil))
		}
		var held int64
		if err := s.Pool.QueryRow(ctx, `WITH x AS (UPDATE channel_messages SET status = 'cancelled', reason = 'cancelled', updated_at = $2
			WHERE campaign_id = $1 AND status IN ('queued','held') RETURNING 1) SELECT count(*) FROM x`, c.ID, now).Scan(&held); err != nil {
			return err
		}
		return s.setPhase(ctx, ws, c, snd, "cancelled", "", map[string]any{"cancelled": held}, &audit.Event{Action: "campaign.cancelled", Reason: in.Reason,
			Metadata: map[string]any{"cancelled_texts": held}})
	}
	return api.Invalid("body.action", say(ctx, mSendAction, nil))
}

// setPhase moves a send (and its campaign) to a phase, audited.
func (s *Service) setPhase(ctx context.Context, ws *store.Workspace, c store.SmsCampaignRow, snd store.SmsSend, phase, kind string, pause map[string]any, ev *audit.Event) error {
	now := s.now()
	status := map[string]string{"sending": "sending", "paused": "paused", "cancelled": "cancelled", "completed": "sent", "scheduled": "scheduled"}[phase]
	var pausedAt, completedAt *time.Time
	if phase == "paused" {
		pausedAt = &now
	}
	if phase == "cancelled" || phase == "completed" {
		completedAt = &now
	}
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if phase == "paused" {
			// The texts not sent yet wait: none goes while the send is paused.
			tag, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = 'paused', updated_at = $2 WHERE campaign_id = $1 AND status = 'queued'`, c.ID, now)
			if err != nil {
				return err
			}
			pause["held"] = tag.RowsAffected()
		}
		if err := q.SmsSendSetPhase(ctx, store.SmsSendSetPhaseParams{Phase: phase, PausedKind: kind, PausedAt: pausedAt, Pause: jsonOf(pause), CompletedAt: completedAt,
			CampaignID: c.ID}); err != nil {
			return err
		}
		if err := q.SmsCampaignSetStatus(ctx, store.SmsCampaignSetStatusParams{Status: status, ScheduledAt: &snd.ScheduledAt, CompletedAt: completedAt, At: now, ID: c.ID}); err != nil {
			return err
		}
		if ev == nil {
			return nil
		}
		e := *ev
		e.WorkspaceID, e.TargetType, e.TargetID, e.TargetName = &ws.ID, "campaign", c.ID.String(), c.Name
		e.Before, e.After = map[string]any{"phase": snd.Phase}, map[string]any{"phase": phase}
		if e.Metadata == nil {
			e.Metadata = map[string]any{}
		}
		e.Metadata["channel"] = "sms"
		return s.Audit.LogWith(ctx, q, e)
	})
}
