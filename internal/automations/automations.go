// Package automations implements visual flows (spec 7.10): the React Flow
// graph and its validation, versions with a diff and publish, the runs engine
// on River and per-step stats. The sending and approvals domains plug in
// through Mailer and PublishGate.
package automations

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Inserter enqueues River jobs; *jobs.Client satisfies it. It may be nil
// (OpenAPI generation, some tests): the periodic jobs pick work up anyway.
type Inserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Service holds the automations domain dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	// Jobs returns the River client used to wake the engine; nil or a nil
	// client means "wait for the next periodic tick".
	Jobs func() Inserter
	// Mailer queues automation emails (QueueOnlyMailer by default).
	Mailer Mailer
	// Gate decides whether publishing needs approval (NoApproval by default).
	Gate PublishGate
	// Checks runs content checks on emails before publishing.
	Checks ContentChecker
	// Accessibility checks the emails before publishing; nil skips it.
	Accessibility AccessibilityGate
	// HTTP calls webhook steps; nil uses NewWebhookClient (public addresses only).
	HTTP *http.Client
	Now  func() time.Time

	// Languages are the emails' language versions (internal/languages);
	// nil leaves them out.
	Languages EmailLanguages
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) mailer() Mailer {
	if s.Mailer != nil {
		return s.Mailer
	}
	return QueueOnlyMailer{}
}

func (s *Service) gate() PublishGate {
	if s.Gate != nil {
		return s.Gate
	}
	return NoApproval{}
}

func (s *Service) checks() ContentChecker {
	if s.Checks != nil {
		return s.Checks
	}
	return BasicContentChecks{}
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Service) inserter() Inserter {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

// kick wakes the engine soon after a change that made runs due.
func (s *Service) kick(ctx context.Context, tx pgx.Tx) {
	ins := s.inserter()
	if ins == nil {
		return
	}
	var err error
	if tx != nil {
		_, err = ins.InsertTx(ctx, tx, TickArgs{}, tickOpts())
	} else {
		_, err = ins.Insert(ctx, TickArgs{}, tickOpts())
	}
	if err != nil {
		s.log().WarnContext(ctx, "automations: could not wake the engine", "err", err)
	}
}

// Statuses of a flow.
const (
	StatusDraft  = "draft"
	StatusLive   = "live"
	StatusPaused = "paused"
)

// loadRefs loads what steps can point at in a workspace.
func (s *Service) loadRefs(ctx context.Context, q *store.Queries, wsID uuid.UUID) (*Refs, error) {
	r := &Refs{Lists: map[string]string{}, Forms: map[string]string{}, Emails: map[string]int{}, EmailNames: map[string]string{}, Fields: map[string]string{}, Providers: map[string]string{}}
	lists, err := q.AutomationLists(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, l := range lists {
		r.Lists[l.ID.String()] = l.Name
	}
	forms, err := q.AutomationForms(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, f := range forms {
		r.Forms[f.ID.String()] = f.Slug
	}
	emails, err := q.ListAutomationEmails(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, e := range emails {
		r.Emails[e.ID.String()] = int(e.LatestVersion)
		r.EmailNames[e.ID.String()] = e.Name
	}
	fields, err := q.AutomationFields(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		r.Fields[f.Key] = f.Type
	}
	provs, err := q.AutomationProviders(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, p := range provs {
		r.Providers[p.ID.String()] = p.Name
	}
	return r, nil
}

// AutomationRouteProvider is a sending provider on the Automations route.
type AutomationRouteProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name" example:"Amazon SES"`
	Short   string `json:"short" example:"SES"`
	Adapter string `json:"adapter"`
	Status  string `json:"status" doc:"healthy, degraded, down, sandbox, warming_up or paused"`
	Detail  string `json:"detail,omitempty"`
	Weight  int    `json:"weight,omitempty"`
	// DownSince and DownReason (outage or auth) are set while the provider is down.
	DownSince  *time.Time `json:"downSince,omitempty"`
	DownReason string     `json:"downReason,omitempty"`
	// LastErrorStatus is the HTTP status of its last failed call (401).
	LastErrorStatus int `json:"lastErrorStatus,omitempty"`
}

// providerHealth is the part of a provider's health (internal/providers/state)
// that the Automations route reports.
type providerHealth struct {
	Down *struct {
		Since  time.Time `json:"since"`
		Reason string    `json:"reason"`
	} `json:"down"`
	LastError *struct {
		Status int `json:"status"`
	} `json:"lastError"`
}

func routeProvider(p store.AutomationProvidersRow, weight int) AutomationRouteProvider {
	r := AutomationRouteProvider{ID: p.ID.String(), Name: p.Name, Short: shortName(p.Name, p.Adapter), Adapter: p.Adapter, Status: p.Status, Detail: p.StatusDetail, Weight: weight}
	var h providerHealth
	if p.Status == "down" && len(p.Health) > 0 && json.Unmarshal(p.Health, &h) == nil {
		if h.Down != nil {
			since := h.Down.Since
			r.DownSince, r.DownReason = &since, h.Down.Reason
		}
		if h.LastError != nil {
			r.LastErrorStatus = h.LastError.Status
		}
	}
	return r
}

// AutomationRoute summarizes the Automations route: "SES, failover Postmark".
type AutomationRoute struct {
	Label    string                    `json:"label" example:"SES, failover Postmark"`
	Healthy  bool                      `json:"healthy" doc:"A provider on the route can send right now"`
	Primary  []AutomationRouteProvider `json:"primary"`
	Fallback *AutomationRouteProvider  `json:"fallback,omitempty"`
}

// shortName is the badge label of a provider: "Amazon SES" → "SES".
func shortName(name, adapter string) string {
	switch adapter {
	case "ses":
		return "SES"
	case "azure":
		return "Azure"
	}
	if i := strings.Index(name, " ("); i > 0 {
		return name[:i]
	}
	return name
}

// healthyStatus says whether a provider in this status can send.
func healthyStatus(status string) bool {
	return status == "healthy" || status == "degraded" || status == "warming_up" || status == "sandbox"
}

func (s *Service) route(ctx context.Context, q *store.Queries, wsID uuid.UUID) (AutomationRoute, error) {
	out := AutomationRoute{Label: "not set up", Primary: []AutomationRouteProvider{}}
	provs, err := q.AutomationProviders(ctx, wsID)
	if err != nil {
		return out, err
	}
	byID := map[uuid.UUID]store.AutomationProvidersRow{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	rp := routeProvider
	rule, err := q.AutomationRoute(ctx, wsID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	var splits []struct {
		ProviderID uuid.UUID `json:"provider_id"`
		Weight     int       `json:"weight"`
	}
	_ = json.Unmarshal(rule.Splits, &splits)
	var parts []string
	for _, sp := range splits {
		p, ok := byID[sp.ProviderID]
		if !ok {
			continue
		}
		r := rp(p, sp.Weight)
		out.Primary = append(out.Primary, r)
		if healthyStatus(p.Status) {
			out.Healthy = true
		}
		if len(splits) > 1 {
			parts = append(parts, r.Short+" "+itoa(sp.Weight)+"%")
		} else {
			parts = append(parts, r.Short)
		}
	}
	if rule.FallbackProviderID != nil {
		if p, ok := byID[*rule.FallbackProviderID]; ok {
			r := rp(p, 0)
			out.Fallback = &r
			if healthyStatus(p.Status) {
				out.Healthy = true
			}
		}
	}
	if len(parts) > 0 {
		out.Label = strings.Join(parts, " / ")
		if out.Fallback != nil {
			out.Label += ", failover " + out.Fallback.Short
		}
	}
	return out, nil
}

func itoa(n int) string { return formatCount(int64(n)) }

// graphOf parses a stored graph, treating corrupt data as an empty flow.
func (s *Service) graphOf(raw []byte) AutomationGraph {
	g, err := ParseGraph(raw)
	if err != nil {
		s.log().Error("automations: unreadable graph", "err", err)
		return AutomationGraph{Nodes: []AutomationNode{}, Edges: []AutomationEdge{}}
	}
	return g
}

func triggerOf(g AutomationGraph) json.RawMessage {
	t := g.Trigger()
	if t == nil || t.Data.Trigger == nil {
		return json.RawMessage(`{}`)
	}
	b, _ := json.Marshal(t.Data.Trigger)
	return b
}

func ptrInt(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func uuidSet(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
