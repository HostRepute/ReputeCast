package automations

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/store"
)

// AutomationDetail is a flow as the editor needs it: its settings, the
// draft graph and what the canvas marks on it.
type AutomationDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status" enum:"draft,live,paused"`
	TemplateKey string `json:"templateKey,omitempty"`
	LiveVersion *int   `json:"liveVersion,omitempty"`
	// DraftVersion is the version the canvas edits; Revision guards
	// concurrent edits (send it back with each save).
	DraftVersion int             `json:"draftVersion"`
	Revision     int64           `json:"revision"`
	PublishedAt  *time.Time      `json:"publishedAt,omitempty"`
	PublishedBy  string          `json:"publishedBy,omitempty"`
	PausedAt     *time.Time      `json:"pausedAt,omitempty"`
	PausedBy     string          `json:"pausedBy,omitempty"`
	PauseReason  string          `json:"pauseReason,omitempty"`
	Graph        AutomationGraph `json:"graph"`
	// LiveNodeIDs are the steps of the live version (others are "new in vN").
	LiveNodeIDs []string `json:"liveNodeIds"`
	// PublishedNodeIDs are steps that were in any published version (others
	// have "no stats until published").
	PublishedNodeIDs []string               `json:"publishedNodeIds"`
	Changes          int                    `json:"changes" doc:"Unpublished changes, as counted by the version diff"`
	Issues           []AutomationGraphIssue `json:"issues"`
	Route            AutomationRoute        `json:"route"`
	InFlow           int64                  `json:"inFlow"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// Get returns a flow for the editor.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AutomationDetail, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationDetail{}, api.MapError(err, "Automation")
	}
	refs, err := s.loadRefs(ctx, s.Q, ws.ID)
	if err != nil {
		return AutomationDetail{}, err
	}
	draftV, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: a.DraftVersion})
	if err != nil {
		return AutomationDetail{}, err
	}
	draft := s.graphOf(draftV.Graph)
	if err := s.markSecrets(ctx, s.Q, a.ID, &draft); err != nil {
		return AutomationDetail{}, err
	}
	out := AutomationDetail{
		ID: a.ID.String(), Name: a.Name, Description: a.Description, Status: a.Status, TemplateKey: a.TemplateKey,
		LiveVersion: ptrInt(a.LiveVersion), DraftVersion: int(a.DraftVersion), Revision: a.DraftRevision,
		PausedAt: a.PausedAt, PausedBy: a.PausedByName, PauseReason: a.PauseReason,
		Graph: draft, LiveNodeIDs: []string{}, PublishedNodeIDs: []string{}, UpdatedAt: a.UpdatedAt,
	}
	out.Issues = Validate(draft, refs)
	published, err := s.Q.ListAutomationPublishedGraphs(ctx, a.ID)
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, p := range published {
		for _, n := range s.graphOf(p.Graph).Nodes {
			seen[n.ID] = true
		}
	}
	out.PublishedNodeIDs = sortedKeys(seen)
	if a.LiveVersion != nil {
		lv, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: *a.LiveVersion})
		if err != nil {
			return out, err
		}
		live := s.graphOf(lv.Graph)
		for _, n := range live.Nodes {
			out.LiveNodeIDs = append(out.LiveNodeIDs, n.ID)
		}
		out.PublishedAt, out.PublishedBy = lv.PublishedAt, lv.PublishedByName
		if *a.LiveVersion != a.DraftVersion {
			out.Changes = Changes(Diff(live, draft, DiffContext{FromRefs: refs, ToVersion: int(a.DraftVersion)}))
		}
	}
	counts, err := s.Q.CountAutomationInFlow(ctx, []uuid.UUID{a.ID})
	if err != nil {
		return out, err
	}
	for _, c := range counts {
		out.InFlow += c.N
	}
	out.Route, err = s.route(ctx, s.Q, ws.ID)
	return out, err
}

// markSecrets sets HasSecret on webhook steps whose signing secret is stored.
func (s *Service) markSecrets(ctx context.Context, q *store.Queries, automationID uuid.UUID, g *AutomationGraph) error {
	nodes, err := q.AutomationStepSecretNodes(ctx, automationID)
	if err != nil {
		return err
	}
	has := uuidSet(nodes)
	for i := range g.Nodes {
		if w := g.Nodes[i].Data.Webhook; w != nil {
			w.HasSecret = has[g.Nodes[i].ID]
		}
	}
	return nil
}

// AutomationPathCount is how many contacts took one path out of a branch or split.
type AutomationPathCount struct {
	Path  string `json:"path" example:"yes"`
	Count int64  `json:"count"`
}

// AutomationStepStats is one step's numbers over the stats window.
type AutomationStepStats struct {
	NodeID string `json:"nodeId"`
	// Entered counts arrivals; InFlowNow counts contacts at the step right now.
	Entered      int64                     `json:"entered"`
	InFlowNow    int64                     `json:"inFlowNow"`
	Completed    int64                     `json:"completed" doc:"Moved on to the next step"`
	Exited       int64                     `json:"exited" doc:"Finished the flow at this step"`
	Dropped      int64                     `json:"dropped" doc:"Left early: filtered out, unsubscribed or removed"`
	GoalReached  int64                     `json:"goalReached"`
	Skipped      int64                     `json:"skipped" doc:"Emails skipped by a rule"`
	Failed       int64                     `json:"failed" doc:"Webhook calls that failed"`
	Sent         int64                     `json:"sent"`
	Delivered    int64                     `json:"delivered"`
	Bounced      int64                     `json:"bounced"`
	Complained   int64                     `json:"complained"`
	Opened       int64                     `json:"opened"`
	OpenedHuman  int64                     `json:"openedHuman"`
	Clicked      int64                     `json:"clicked"`
	Unsubscribed int64                     `json:"unsubscribed"`
	Paths        []AutomationPathCount     `json:"paths"`
	Providers    []AutomationProviderSends `json:"providers"`
}

// AutomationStats is the live stats of a flow, per step.
type AutomationStats struct {
	Days  int       `json:"days"`
	Since time.Time `json:"since"`
	// Entered counts contacts who entered through the trigger in the window.
	Entered int64                 `json:"entered"`
	InFlow  int64                 `json:"inFlow"`
	Steps   []AutomationStepStats `json:"steps"`
}

// Stats returns per-step stats for the last N days, across versions (steps
// keep their ids, so a step's numbers carry over when a new version is
// published).
func (s *Service) Stats(ctx context.Context, ws *store.Workspace, id uuid.UUID, days int) (AutomationStats, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationStats{}, api.MapError(err, "Automation")
	}
	since := s.since(days)
	out := AutomationStats{Days: days, Since: since, Steps: []AutomationStepStats{}}
	ids := []uuid.UUID{a.ID}
	rows, err := s.Q.AutomationStepStatsSince(ctx, store.AutomationStepStatsSinceParams{AutomationIds: ids, Since: since})
	if err != nil {
		return out, err
	}
	paths, err := s.Q.AutomationStepPathsSince(ctx, store.AutomationStepPathsSinceParams{AutomationIds: ids, Since: since})
	if err != nil {
		return out, err
	}
	provRows, err := s.Q.AutomationStepProvidersSince(ctx, store.AutomationStepProvidersSinceParams{AutomationIds: ids, Since: since})
	if err != nil {
		return out, err
	}
	now, err := s.Q.CountAutomationInFlowByNode(ctx, a.ID)
	if err != nil {
		return out, err
	}
	route, err := s.route(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	provs, err := s.Q.AutomationProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	names := map[string]store.AutomationProvidersRow{}
	for _, p := range provs {
		names[p.ID.String()] = p
	}
	byNode := map[string]*AutomationStepStats{}
	get := func(node string) *AutomationStepStats {
		if st, ok := byNode[node]; ok {
			return st
		}
		st := &AutomationStepStats{NodeID: node, Paths: []AutomationPathCount{}, Providers: []AutomationProviderSends{}}
		byNode[node] = st
		return st
	}
	for _, r := range rows {
		st := get(r.NodeID)
		st.Entered, st.Completed, st.Exited, st.Dropped = r.Entered, r.Completed, r.Exited, r.Dropped
		st.GoalReached, st.Skipped, st.Failed, st.Sent = r.GoalReached, r.Skipped, r.Failed, r.Sent
		st.Delivered, st.Bounced, st.Complained, st.Opened = r.Delivered, r.Bounced, r.Complained, r.Opened
		st.OpenedHuman, st.Clicked, st.Unsubscribed = r.OpenedHuman, r.Clicked, r.Unsubscribed
	}
	for _, p := range paths {
		st := get(p.NodeID)
		st.Paths = append(st.Paths, AutomationPathCount{Path: p.Path, Count: p.N})
	}
	for _, p := range provRows {
		st := get(p.NodeID)
		ps := AutomationProviderSends{ProviderID: p.ProviderID, Sent: p.N}
		if pr, ok := names[p.ProviderID]; ok {
			ps.Provider, ps.Short = pr.Name, shortName(pr.Name, pr.Adapter)
		} else {
			ps.Provider, ps.Short = "Removed provider", "Removed"
		}
		ps.Fallback = route.Fallback != nil && route.Fallback.ID == p.ProviderID
		ps.LastDay = p.LastDay.Format(time.DateOnly)
		st.Providers = append(st.Providers, ps)
	}
	for _, n := range now {
		get(n.NodeID).InFlowNow += n.N
		out.InFlow += n.N
	}
	for _, st := range byNode {
		sort.Slice(st.Paths, func(i, j int) bool { return st.Paths[i].Path < st.Paths[j].Path })
		sort.Slice(st.Providers, func(i, j int) bool { return st.Providers[i].Sent > st.Providers[j].Sent })
		out.Steps = append(out.Steps, *st)
	}
	sort.Slice(out.Steps, func(i, j int) bool { return out.Steps[i].NodeID < out.Steps[j].NodeID })
	// Entered = the trigger of the version being shown (draft or live).
	v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: a.DraftVersion})
	if err == nil {
		if t := s.graphOf(v.Graph).Trigger(); t != nil {
			if st, ok := byNode[t.ID]; ok {
				out.Entered = st.Entered
			}
		}
	}
	return out, nil
}

// AutomationOptionItem is a choice in a step's settings.
type AutomationOptionItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AutomationFormOption is a signup form a trigger can listen to.
type AutomationFormOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// AutomationFieldOption is a contact field a condition or date trigger can use.
type AutomationFieldOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// AutomationEmailOption is an email a send step can use.
type AutomationEmailOption struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	LatestVersion int       `json:"latestVersion"`
	Subject       string    `json:"subject"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// AutomationOptions are the choices the step settings offer.
type AutomationOptions struct {
	Lists     []AutomationOptionItem    `json:"lists"`
	Forms     []AutomationFormOption    `json:"forms"`
	Fields    []AutomationFieldOption   `json:"fields"`
	Emails    []AutomationEmailOption   `json:"emails"`
	Providers []AutomationRouteProvider `json:"providers"`
	Tags      []string                  `json:"tags"`
	Events    []string                  `json:"events"`
	// ScoreBands are the applied score thresholds, for the "Score
	// crosses…" trigger; null while the workspace has no scoring (X13).
	ScoreBands *scoring.ScoringBands `json:"scoreBands,omitempty"`
	// Products are the digital products, for "Bought a product" (X19):
	// published first, then the others that ever sold.
	Products []AutomationOptionItem `json:"products"`
	// Milestones are the referral milestones the "Reached a milestone"
	// trigger can start on (Grow).
	Milestones []AutomationMilestoneOption `json:"milestones"`
}

// AutomationMilestoneOption is a referral milestone a "Reached a
// milestone" trigger can start on.
type AutomationMilestoneOption struct {
	ID        string `json:"id"`
	ProgramID string `json:"programId"`
	ListName  string `json:"listName" example:"Weekly Digest" doc:"The list of the milestone's referral program"`
	Count     int    `json:"count" example:"3"`
	Reward    string `json:"reward" example:"The Acme Field Guide"`
	Name      string `json:"name" example:"3 referrals · The Acme Field Guide"`
}

// Options lists lists, forms, fields, emails, providers, tags and API events.
func (s *Service) Options(ctx context.Context, ws *store.Workspace) (AutomationOptions, error) {
	out := AutomationOptions{Lists: []AutomationOptionItem{}, Forms: []AutomationFormOption{}, Fields: []AutomationFieldOption{}, Emails: []AutomationEmailOption{}, Providers: []AutomationRouteProvider{}, Tags: []string{}, Events: []string{}, Milestones: []AutomationMilestoneOption{},
		Products: []AutomationOptionItem{}}
	if prods, err := s.Q.SegmentProductOptions(ctx, ws.ID); err == nil {
		for _, p := range prods {
			out.Products = append(out.Products, AutomationOptionItem{ID: p.ID.String(), Name: p.Name})
		}
	} else {
		return out, err
	}
	lists, err := s.Q.AutomationLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, AutomationOptionItem{ID: l.ID.String(), Name: l.Name})
	}
	forms, err := s.Q.AutomationForms(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range forms {
		out.Forms = append(out.Forms, AutomationFormOption{ID: f.ID.String(), Name: f.Name, Slug: f.Slug})
	}
	fields, err := s.Q.AutomationFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range fields {
		out.Fields = append(out.Fields, AutomationFieldOption{Key: f.Key, Label: f.Label, Type: f.Type})
	}
	emails, err := s.Q.ListAutomationEmails(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, e := range emails {
		out.Emails = append(out.Emails, AutomationEmailOption{ID: e.ID.String(), Name: e.Name, LatestVersion: int(e.LatestVersion), Subject: e.Subject, UpdatedAt: e.UpdatedAt})
	}
	provs, err := s.Q.AutomationProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, p := range provs {
		out.Providers = append(out.Providers, AutomationRouteProvider{ID: p.ID.String(), Name: p.Name, Short: shortName(p.Name, p.Adapter), Adapter: p.Adapter, Status: p.Status, Detail: p.StatusDetail})
	}
	tags, err := s.Q.AutomationTopTags(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, t := range tags {
		out.Tags = append(out.Tags, t.Tag)
	}
	if bands, ok, err := scoring.AppliedBands(ctx, s.Q, ws.ID); err != nil {
		return out, err
	} else if ok {
		out.ScoreBands = &bands
	}
	events, err := s.Q.AutomationEventNames(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Events = append(out.Events, events...)
	ms, err := s.Q.AutomationMilestones(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, m := range ms {
		out.Milestones = append(out.Milestones, AutomationMilestoneOption{ID: m.ID.String(), ProgramID: m.ProgramID.String(), ListName: m.ListName,
			Count: int(m.Count), Reward: m.RewardName, Name: milestoneName(m.Count, m.RewardName)})
	}
	return out, nil
}

// AutomationVersionItem is one row of the version history.
type AutomationVersionItem struct {
	Version     int        `json:"version"`
	Status      string     `json:"status" enum:"draft,live,retiring,retired" doc:"retiring: replaced, but contacts are still finishing on it"`
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	PublishedBy string     `json:"publishedBy,omitempty"`
	RetiredAt   *time.Time `json:"retiredAt,omitempty"`
	Summary     string     `json:"summary"`
	InFlow      int64      `json:"inFlow"`
}

// AutomationVersions is the version history of a flow.
type AutomationVersions struct {
	Name     string                  `json:"name"`
	Versions []AutomationVersionItem `json:"versions"`
}

// Versions lists every version, newest first.
func (s *Service) Versions(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AutomationVersions, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationVersions{}, api.MapError(err, "Automation")
	}
	out := AutomationVersions{Name: a.Name, Versions: []AutomationVersionItem{}}
	rows, err := s.Q.ListAutomationVersions(ctx, a.ID)
	if err != nil {
		return out, err
	}
	nodes, err := s.Q.CountAutomationInFlowByNode(ctx, a.ID)
	if err != nil {
		return out, err
	}
	inflow := map[int32]int64{}
	for _, n := range nodes {
		inflow[n.Version] += n.N
	}
	for _, r := range rows {
		it := AutomationVersionItem{
			Version: int(r.Version), CreatedAt: r.CreatedAt, CreatedBy: r.CreatedByName,
			PublishedAt: r.PublishedAt, PublishedBy: r.PublishedByName, RetiredAt: r.RetiredAt,
			Summary: r.Summary, InFlow: inflow[r.Version],
		}
		switch {
		case r.PublishedAt == nil:
			it.Status = "draft"
		case a.LiveVersion != nil && r.Version == *a.LiveVersion:
			it.Status = "live"
		case r.RetiredAt == nil:
			it.Status = "retiring"
		default:
			it.Status = "retired"
		}
		out.Versions = append(out.Versions, it)
	}
	return out, nil
}

// AutomationVersionGraph is one version's graph (read-only).
type AutomationVersionGraph struct {
	Version int             `json:"version"`
	Graph   AutomationGraph `json:"graph"`
}

// Version returns the graph of one version.
func (s *Service) Version(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int) (AutomationVersionGraph, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationVersionGraph{}, api.MapError(err, "Automation")
	}
	v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: int32(version)})
	if err != nil {
		return AutomationVersionGraph{}, api.MapError(err, "Version")
	}
	g := s.graphOf(v.Graph)
	if err := s.markSecrets(ctx, s.Q, a.ID, &g); err != nil {
		return AutomationVersionGraph{}, err
	}
	return AutomationVersionGraph{Version: version, Graph: g}, nil
}

// AutomationWebhookCall is one entry of a webhook step's delivery log.
type AutomationWebhookCall struct {
	ID         string    `json:"id"`
	Version    int       `json:"version"`
	URL        string    `json:"url"`
	StatusCode int       `json:"statusCode"`
	Error      string    `json:"error,omitempty"`
	DurationMs int       `json:"durationMs"`
	Attempt    int       `json:"attempt"`
	CalledAt   time.Time `json:"calledAt"`
}

// WebhookCalls returns the latest calls of a webhook step.
func (s *Service) WebhookCalls(ctx context.Context, ws *store.Workspace, id uuid.UUID, node string, limit int) ([]AutomationWebhookCall, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return nil, api.MapError(err, "Automation")
	}
	rows, err := s.Q.ListAutomationWebhookCalls(ctx, store.ListAutomationWebhookCallsParams{AutomationID: a.ID, NodeID: node, Lim: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]AutomationWebhookCall, 0, len(rows))
	for _, r := range rows {
		out = append(out, AutomationWebhookCall{ID: r.ID.String(), Version: int(r.Version), URL: r.Url, StatusCode: int(r.StatusCode), Error: r.Error, DurationMs: int(r.DurationMs), Attempt: int(r.Attempt), CalledAt: r.CalledAt})
	}
	return out, nil
}
