package automations

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

var (
	mAlertWebhookFails     = msg("alert.webhookFails")
	mAlertWebhookReturns   = msg("alert.webhookReturns")
	mAlertWebhookFailsWith = msg("alert.webhookFailsWith")
	mAlertWebhookDetail    = msg("alert.webhookDetail")
	mAlertOpenStep         = msg("alert.openStep")
	mAlertWebhookLog       = msg("alert.webhookLog")
	mAlertHeldTitle        = msg("alert.heldTitle")
	mAlertHeldDetail       = msg("alert.heldDetail")
	mAlertNoProvider       = msg("alert.noProvider")
	mAlertFallback         = msg("alert.fallback")
	mAlertAuthStatus       = msg("alert.authStatus")
	mAlertAuth             = msg("alert.auth")
	mAlertDownSince        = msg("alert.downSince")
	mAlertStatus           = msg("alert.status")
	mAlertStatusDetail     = msg("alert.statusDetail")
	mAlertFixToken         = msg("alert.fixToken")
	mAlertFixProvider      = msg("alert.fixProvider")
	mAlertAddFallback      = msg("alert.addFallback")
	mLabelRemovedProvider  = msg("label.removedProvider")
	mLabelRemovedShort     = msg("label.removedProviderShort")
)

// AutomationGoal is the list's goal column: "Clicked docs · 24%".
type AutomationGoal struct {
	Label   string   `json:"label" example:"Clicked docs"`
	Percent *float64 `json:"percent,omitempty" doc:"Contacts who reached the goal, % of those who entered in the window"`
}

// AutomationListItem is one row of the automations list.
type AutomationListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status" enum:"draft,live,paused"`
	LiveVersion *int   `json:"liveVersion,omitempty"`
	// DraftVersion is the version being edited (equals the live version when
	// there are no unpublished changes).
	DraftVersion int        `json:"draftVersion"`
	PublishedAt  *time.Time `json:"publishedAt,omitempty"`
	PausedAt     *time.Time `json:"pausedAt,omitempty"`
	PausedBy     string     `json:"pausedBy,omitempty"`
	PauseReason  string     `json:"pauseReason,omitempty"`
	// Trigger reads "Joined list · Weekly Digest".
	Trigger string `json:"trigger"`
	// Steps counts the steps of the live version (the draft for flows never published).
	Steps int `json:"steps"`
	// Stats over the list window; absent for flows that were never published.
	Entered    *int64          `json:"entered,omitempty"`
	InFlow     *int64          `json:"inFlow,omitempty"`
	Completed  *float64        `json:"completed,omitempty" doc:"% of entrants who finished the flow (goal or end)"`
	HumanOpens *float64        `json:"humanOpens,omitempty" doc:"Human opens, % of delivered"`
	Goal       *AutomationGoal `json:"goal,omitempty"`
	// UnpublishedChanges is true when a draft differs from the live version.
	UnpublishedChanges bool `json:"unpublishedChanges"`
	// Revenue is the store orders credited to the flow's emails over the
	// list window; absent when the workspace has no store.
	Revenue *AutomationRevenue `json:"revenue,omitempty"`
	// Channels are the channels the flow's steps send on (email, sms),
	// for the row's icons; ChannelCost is what its texts cost over the
	// list window (design expansion phase 8).
	Channels    []string `json:"channels"`
	ChannelCost *float64 `json:"channelCost,omitempty"`
}

// AutomationStatusCounts feeds the All · Live · Paused · Drafts filters.
type AutomationStatusCounts struct {
	All    int64 `json:"all"`
	Live   int64 `json:"live"`
	Paused int64 `json:"paused"`
	Drafts int64 `json:"drafts"`
}

// AutomationAlertAction is a button on an alert card.
type AutomationAlertAction struct {
	Label string `json:"label" example:"Open the step"`
	// Kind says where it goes: the step on the canvas, its delivery log, the
	// providers page or routing.
	Kind   string `json:"kind" enum:"open_step,step_log,providers,routing"`
	NodeID string `json:"nodeId,omitempty"`
}

// AutomationAlert is a flow problem shown above the list: sending held
// because no provider on the Automations route is healthy, or a step that
// keeps failing.
type AutomationAlert struct {
	ID           string                  `json:"id"`
	AutomationID string                  `json:"automationId"`
	Automation   string                  `json:"automation"`
	Kind         string                  `json:"kind" enum:"sending_held,step_failing"`
	NodeID       string                  `json:"nodeId,omitempty"`
	Title        string                  `json:"title"`
	Detail       string                  `json:"detail"`
	Since        time.Time               `json:"since"`
	Failures     int64                   `json:"failures"`
	Actions      []AutomationAlertAction `json:"actions"`
}

// AutomationList is the automations screen.
type AutomationList struct {
	Items  []AutomationListItem   `json:"items"`
	Counts AutomationStatusCounts `json:"counts"`
	Route  AutomationRoute        `json:"route"`
	Alerts []AutomationAlert      `json:"alerts"`
	// Days is the stats window of the list (30).
	Days int `json:"days"`
}

// ListDays is the stats window of the list and the canvas.
const ListDays = 30

// since returns the first day (UTC) of an N-day window ending today.
func (s *Service) since(days int) time.Time {
	t := s.now().UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
}

func pct(n, d int64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*1000) / 10
	return &v
}

// List returns every flow with its 30-day performance, filtered by status.
func (s *Service) List(ctx context.Context, ws *store.Workspace, status string) (AutomationList, error) {
	l := langOf(ctx)
	out := AutomationList{Items: []AutomationListItem{}, Alerts: []AutomationAlert{}, Days: ListDays}
	rows, err := s.Q.ListAutomations(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	refs, err := s.loadRefs(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
		out.Counts.All++
		switch r.Status {
		case StatusLive:
			out.Counts.Live++
		case StatusPaused:
			out.Counts.Paused++
		default:
			out.Counts.Drafts++
		}
	}
	stats, err := s.statsByAutomation(ctx, ids, ListDays)
	if err != nil {
		return out, err
	}
	inflow := map[uuid.UUID]int64{}
	counts, err := s.Q.CountAutomationInFlow(ctx, ids)
	if err != nil {
		return out, err
	}
	for _, c := range counts {
		inflow[c.AutomationID] = c.N
	}
	var revenue map[uuid.UUID]AutomationRevenue
	if s.StoreRevenue != nil {
		revenue = s.StoreRevenue(ctx, ws.ID, ids, s.now().AddDate(0, 0, -ListDays))
	}
	for _, r := range rows {
		if status != "" && status != "all" && !matchesFilter(r.Status, status) {
			continue
		}
		live := s.graphOf(r.LiveGraph)
		draft := s.graphOf(r.DraftGraph)
		shown := live
		if r.LiveVersion == nil {
			shown = draft
		}
		it := AutomationListItem{
			ID: r.ID.String(), Name: r.Name, Description: r.Description, Status: r.Status,
			LiveVersion: ptrInt(r.LiveVersion), DraftVersion: int(r.DraftVersion), PublishedAt: r.LivePublishedAt,
			PausedAt: r.PausedAt, PausedBy: r.PausedByName, PauseReason: r.PauseReason,
			Steps: len(shown.Nodes), Channels: graphChannels(shown),
		}
		var trig *AutomationTrigger
		if t := shown.Trigger(); t != nil {
			trig = t.Data.Trigger
		}
		it.Trigger = say(l, triggerLabel(trig, refs))
		if r.LiveVersion != nil && r.DraftVersion != *r.LiveVersion {
			it.UnpublishedChanges = Changes(Diff(live, draft, DiffContext{FromRefs: refs})) > 0
		}
		if goal := firstGoal(shown); goal != nil {
			it.Goal = &AutomationGoal{Label: say(l, goalLabel(*goal, refs))}
		}
		if r.LiveVersion != nil {
			st := stats[r.ID]
			entered := st.entered(shown)
			n := inflow[r.ID]
			it.Entered = &entered
			it.InFlow = &n
			it.Completed = pct(st.finished(shown), entered)
			it.HumanOpens = pct(st.total(func(x store.AutomationStepStatsSinceRow) int64 { return x.OpenedHuman }),
				st.total(func(x store.AutomationStepStatsSinceRow) int64 { return x.Delivered }))
			if it.Goal != nil {
				it.Goal.Percent = pct(st.goals(shown), entered)
			}
		}
		if rev, ok := revenue[r.ID]; ok {
			it.Revenue = &rev
		}
		out.Items = append(out.Items, it)
	}
	if err := s.channelCosts(ctx, ws.ID, out.Items, s.since(ListDays)); err != nil {
		return out, err
	}
	out.Route, err = s.route(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Alerts, err = s.alerts(ctx, ws, refs, out.Route, l)
	return out, err
}

func matchesFilter(status, filter string) bool {
	switch filter {
	case "live":
		return status == StatusLive
	case "paused":
		return status == StatusPaused
	case "drafts":
		return status == StatusDraft
	}
	return true
}

func firstGoal(g AutomationGraph) *AutomationNode {
	for _, id := range g.Order() {
		if n := g.Node(id); n != nil && n.Type == NodeGoal {
			return n
		}
	}
	return nil
}

// flowStats holds one flow's per-step totals over a window.
type flowStats map[string]store.AutomationStepStatsSinceRow

func (f flowStats) total(field func(store.AutomationStepStatsSinceRow) int64) int64 {
	var n int64
	for _, r := range f {
		n += field(r)
	}
	return n
}

// entered counts contacts who entered through the trigger.
func (f flowStats) entered(g AutomationGraph) int64 {
	if t := g.Trigger(); t != nil {
		return f[t.ID].Entered
	}
	return 0
}

// goals counts goals reached at any goal step.
func (f flowStats) goals(g AutomationGraph) int64 {
	var n int64
	for _, r := range f {
		n += r.GoalReached
	}
	_ = g
	return n
}

// finished counts contacts who reached a goal or the end of the flow.
func (f flowStats) finished(g AutomationGraph) int64 {
	var n int64
	for _, r := range f {
		n += r.GoalReached + r.Exited
	}
	_ = g
	return n
}

func (s *Service) statsByAutomation(ctx context.Context, ids []uuid.UUID, days int) (map[uuid.UUID]flowStats, error) {
	out := map[uuid.UUID]flowStats{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.AutomationStepStatsSince(ctx, store.AutomationStepStatsSinceParams{AutomationIds: ids, Since: s.since(days)})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if out[r.AutomationID] == nil {
			out[r.AutomationID] = flowStats{}
		}
		out[r.AutomationID][r.NodeID] = r
	}
	return out, nil
}

// alerts lists open problems, in l. A failing step that the draft already
// removes isn't shown: the fix is waiting to be published.
func (s *Service) alerts(ctx context.Context, ws *store.Workspace, refs *Refs, route AutomationRoute, l i18n.Lang) ([]AutomationAlert, error) {
	out := []AutomationAlert{}
	rows, err := s.Q.ListAutomationAlerts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	loc := workspaceLoc(ws)
	for _, a := range rows {
		al := AutomationAlert{
			ID: a.ID.String(), AutomationID: a.AutomationID.String(), Automation: a.AutomationName, Kind: a.Kind,
			NodeID: a.NodeID, Since: a.FirstAt, Failures: a.Failures, Actions: []AutomationAlertAction{},
		}
		var d struct {
			URL    string `json:"url"`
			Status int    `json:"status"`
			Error  string `json:"error"`
			Name   string `json:"name"`
			Queued int64  `json:"queued"`
		}
		_ = json.Unmarshal(a.Detail, &d)
		switch a.Kind {
		case "step_failing":
			if a.NodeID != "" {
				v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.AutomationID, Version: a.DraftVersion})
				if err == nil && s.graphOf(v.Graph).Node(a.NodeID) == nil {
					continue
				}
			}
			hook := d.Name
			if hook == "" {
				hook = hostOf(d.URL)
			}
			flow := alertFlowName(a.AutomationName)
			switch {
			case d.Status > 0:
				al.Title = say(l, txt(mAlertWebhookReturns, "flow", flow, "name", hook, "status", strconv.Itoa(d.Status)))
			case d.Error != "":
				al.Title = say(l, txt(mAlertWebhookFailsWith, "flow", flow, "name", hook, "error", d.Error))
			default:
				al.Title = say(l, txt(mAlertWebhookFails, "flow", flow, "name", hook))
			}
			al.Detail = say(l, txt(mAlertWebhookDetail, "count", a.Failures, "date", dayMonth(a.FirstAt, loc)))
			al.Actions = append(al.Actions,
				AutomationAlertAction{Label: say(l, txt(mAlertOpenStep)), Kind: "open_step", NodeID: a.NodeID},
				AutomationAlertAction{Label: say(l, txt(mAlertWebhookLog)), Kind: "step_log", NodeID: a.NodeID})
		case "sending_held":
			al.Title = say(l, txt(mAlertHeldTitle, "flow", alertFlowName(a.AutomationName)))
			al.Detail = say(l, heldDetailText(route, d.Queued, loc, s.now()))
			if fix := fixAction(route); !emptyText(fix) {
				al.Actions = append(al.Actions, AutomationAlertAction{Label: say(l, fix), Kind: "providers"})
			}
			if extra := spareProvider(ctx, s, ws, route); extra != "" {
				al.Actions = append(al.Actions, AutomationAlertAction{Label: say(l, txt(mAlertAddFallback, "provider", extra)), Kind: "routing"})
			}
		}
		out = append(out, al)
	}
	_ = refs
	return out, nil
}

// alertFlowName is how alerts name a flow: "Onboarding · new workspace" →
// "Onboarding".
func alertFlowName(name string) string {
	if i := strings.Index(name, " · "); i > 0 {
		return name[:i]
	}
	return name
}

// fixAction is the first thing the team can fix on the route: a rejected
// token ("Fix Postmark token") before a provider that is down on its side.
func fixAction(route AutomationRoute) i18n.Text {
	all := append(append([]AutomationRouteProvider{}, route.Primary...), derefRoute(route.Fallback)...)
	for _, p := range all {
		if !healthyStatus(p.Status) && p.DownReason == "auth" {
			return txt(mAlertFixToken, "provider", p.Short)
		}
	}
	for _, p := range all {
		if !healthyStatus(p.Status) {
			return txt(mAlertFixProvider, "provider", p.Short)
		}
	}
	return i18n.Text{}
}

func derefRoute(p *AutomationRouteProvider) []AutomationRouteProvider {
	if p == nil {
		return nil
	}
	return []AutomationRouteProvider{*p}
}

// providerProblem says what is wrong with a provider on the route: "SES has
// been Down since 09:02", "Postmark, the fallback, returned 401: its API key
// or token was rejected".
func providerProblem(p AutomationRouteProvider, name i18n.Text, loc *time.Location, now time.Time) i18n.Text {
	switch {
	case p.DownReason == "auth" && p.LastErrorStatus > 0:
		return txt(mAlertAuthStatus, "provider", name, "status", strconv.Itoa(p.LastErrorStatus))
	case p.DownReason == "auth":
		return txt(mAlertAuth, "provider", name)
	case p.DownSince != nil:
		since := p.DownSince.In(loc)
		when := clock(since, loc)
		if y, m, d := since.Date(); y != now.In(loc).Year() || m != now.In(loc).Month() || d != now.In(loc).Day() {
			when = dayMonth(since, loc)
		}
		return txt(mAlertDownSince, "provider", name, "when", when)
	}
	statusName := strings.ReplaceAll(p.Status, "_", " ")
	if p.Detail != "" {
		return txt(mAlertStatusDetail, "provider", name, "status", p.Status, "statusName", statusName, "detail", p.Detail)
	}
	return txt(mAlertStatus, "provider", name, "status", p.Status, "statusName", statusName)
}

// heldDetail explains why automation email is held and what happens next.
func heldDetail(route AutomationRoute, queued int64, loc *time.Location, now time.Time) string {
	return say(english, heldDetailText(route, queued, loc, now))
}

func heldDetailText(route AutomationRoute, queued int64, loc *time.Location, now time.Time) i18n.Text {
	var parts []i18n.Text
	for _, p := range route.Primary {
		parts = append(parts, providerProblem(p, name(p.Short), loc, now))
	}
	if f := route.Fallback; f != nil {
		parts = append(parts, providerProblem(*f, txt(mAlertFallback, "provider", f.Short), loc, now))
	}
	problems := andList(parts)
	if len(parts) == 0 {
		problems = txt(mAlertNoProvider)
	}
	return txt(mAlertHeldDetail, "problems", problems, "count", queued)
}

// spareProvider names a healthy provider that isn't on the Automations route.
func spareProvider(ctx context.Context, s *Service, ws *store.Workspace, route AutomationRoute) string {
	provs, err := s.Q.AutomationProviders(ctx, ws.ID)
	if err != nil {
		return ""
	}
	on := map[string]bool{}
	for _, p := range route.Primary {
		on[p.ID] = true
	}
	if route.Fallback != nil {
		on[route.Fallback.ID] = true
	}
	for _, p := range provs {
		if !on[p.ID.String()] && healthyStatus(p.Status) {
			return shortName(p.Name, p.Adapter)
		}
	}
	return ""
}

func workspaceLoc(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// AutomationProviderSends is how many automation emails a provider sent.
type AutomationProviderSends struct {
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider"`
	Short      string `json:"short"`
	Sent       int64  `json:"sent"`
	// Fallback is true for the Automations route's fallback provider.
	Fallback bool `json:"fallback"`
	// LastDay is the last day it sent in the window.
	LastDay string `json:"lastDay,omitempty" format:"date" example:"2026-10-02"`
}

// AutomationGoalCount is goals reached by one flow.
type AutomationGoalCount struct {
	Name  string `json:"name" example:"welcome"`
	Count int64  `json:"count"`
}

// AutomationSummary is the KPI row above the list.
type AutomationSummary struct {
	Days       int                       `json:"days"`
	InFlow     int64                     `json:"inFlow"`
	LiveCount  int64                     `json:"liveCount"`
	Sent       int64                     `json:"sent"`
	SentBy     []AutomationProviderSends `json:"sentBy"`
	HumanOpens *float64                  `json:"humanOpens,omitempty"`
	Clicks     *float64                  `json:"clicks,omitempty"`
	Goals      int64                     `json:"goals"`
	GoalsBy    []AutomationGoalCount     `json:"goalsBy"`
}

// Summary returns the KPI row: contacts in flows now and the last N days.
func (s *Service) Summary(ctx context.Context, ws *store.Workspace, days int) (AutomationSummary, error) {
	l := langOf(ctx)
	out := AutomationSummary{Days: days, SentBy: []AutomationProviderSends{}, GoalsBy: []AutomationGoalCount{}}
	rows, err := s.Q.ListAutomations(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	names := map[uuid.UUID]string{}
	for _, r := range rows {
		ids = append(ids, r.ID)
		names[r.ID] = r.Name
		if r.Status == StatusLive {
			out.LiveCount++
		}
	}
	counts, err := s.Q.CountAutomationInFlow(ctx, ids)
	if err != nil {
		return out, err
	}
	for _, c := range counts {
		out.InFlow += c.N
	}
	since := s.since(days)
	totals, err := s.Q.AutomationWorkspaceStatsSince(ctx, store.AutomationWorkspaceStatsSinceParams{WorkspaceID: ws.ID, Since: since})
	if err != nil {
		return out, err
	}
	var delivered, human, clicked int64
	for _, t := range totals {
		delivered += t.Delivered
		human += t.OpenedHuman
		clicked += t.Clicked
		out.Goals += t.GoalReached
		if t.GoalReached > 0 {
			out.GoalsBy = append(out.GoalsBy, AutomationGoalCount{Name: shortFlowName(names[t.AutomationID]), Count: t.GoalReached})
		}
	}
	sort.SliceStable(out.GoalsBy, func(i, j int) bool { return out.GoalsBy[i].Count > out.GoalsBy[j].Count })
	out.HumanOpens = pct(human, delivered)
	out.Clicks = pct(clicked, delivered)
	route, err := s.route(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	sends, err := s.Q.AutomationProviderSendsSince(ctx, store.AutomationProviderSendsSinceParams{WorkspaceID: ws.ID, Since: since})
	if err != nil {
		return out, err
	}
	provs, err := s.Q.AutomationProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byID := map[string]store.AutomationProvidersRow{}
	for _, p := range provs {
		byID[p.ID.String()] = p
	}
	for _, sd := range sends {
		ps := AutomationProviderSends{ProviderID: sd.ProviderID, Sent: sd.N}
		if p, ok := byID[sd.ProviderID]; ok {
			ps.Provider, ps.Short = p.Name, shortName(p.Name, p.Adapter)
		} else {
			ps.Provider, ps.Short = say(l, txt(mLabelRemovedProvider)), say(l, txt(mLabelRemovedShort))
		}
		ps.Fallback = route.Fallback != nil && route.Fallback.ID == sd.ProviderID
		ps.LastDay = sd.LastDay.Format(time.DateOnly)
		out.Sent += sd.N
		out.SentBy = append(out.SentBy, ps)
	}
	sort.SliceStable(out.SentBy, func(i, j int) bool { return out.SentBy[i].Sent > out.SentBy[j].Sent })
	return out, nil
}

// shortFlowName is the first word of a flow's name, lower case: "Welcome
// series" → "welcome", for the goals breakdown.
func shortFlowName(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return name
	}
	return strings.ToLower(strings.Trim(f[0], "·,:;"))
}
