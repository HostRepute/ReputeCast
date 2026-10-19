package verification

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/time/rate"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// VerificationTarget is what a bulk use checks.
type VerificationTarget struct {
	Kind      string   `json:"kind" enum:"list,segment,tier,selection,contact,import"`
	ListID    string   `json:"listId,omitempty"`
	SegmentID string   `json:"segmentId,omitempty"`
	Tiers     []string `json:"tiers,omitempty" maxItems:"4"`
	// ContactIDs are chosen contacts (Contacts' selection), Filter every
	// contact matching a Contacts filter or saved view.
	ContactIDs []string                `json:"contactIds,omitempty" maxItems:"10000"`
	Filter     *audience.ContactFilter `json:"filter,omitempty"`
	ContactID  string                  `json:"contactId,omitempty"`
	ImportID   string                  `json:"importId,omitempty"`
	// Label names the selection or saved view ("Beta applicants from Typeform").
	Label string `json:"label,omitempty" maxLength:"200"`
}

// VerificationServiceChoice is a connected service in the dialog's select.
type VerificationServiceChoice struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Balance     *int64  `json:"balance,omitempty"`
	Price       float64 `json:"price"`
	Default     bool    `json:"default"`
	RegionLabel string  `json:"regionLabel"`
}

// VerificationEstimate is what a use would send and cost, for the dialog
// and its notice.
type VerificationEstimate struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	// Contacts in the target; Skipped of them checked in the last 90 days.
	Contacts int64 `json:"contacts"`
	Skipped  int64 `json:"skipped"`
	// SkippedService and SkippedOn describe the skipped ones' last check
	// ("Skips 3,615 checked with ZeroBounce on 14 Oct").
	SkippedService string     `json:"skippedService,omitempty"`
	SkippedOn      *time.Time `json:"skippedOn,omitempty"`
	ToCheck        int64      `json:"toCheck"`
	SkipRecent     bool       `json:"skipRecent"`
	// The service it would use (the bulk default unless another is picked).
	Service     string     `json:"service,omitempty"`
	ServiceName string     `json:"serviceName,omitempty"`
	RegionLabel string     `json:"regionLabel,omitempty"`
	Price       float64    `json:"price"`
	Cost        float64    `json:"cost"`
	Balance     *int64     `json:"balance,omitempty"`
	BalanceAt   *time.Time `json:"balanceAt,omitempty"`
	After       *int64     `json:"after,omitempty"`
	// Enough is false when the balance doesn't cover the checks (the job
	// still starts and stops when the credits run out).
	Enough   bool                        `json:"enough"`
	Services []VerificationServiceChoice `json:"services"`
	// Connected: any service is connected.
	Connected     bool     `json:"connected"`
	CanVerify     bool     `json:"canVerify"`
	VerifyHelpers []string `json:"verifyHelpers"`
	CanManage     bool     `json:"canManage"`
	ManageHelpers []string `json:"manageHelpers"`
}

// VerificationEstimateInput asks for an estimate.
type VerificationEstimateInput struct {
	Target     VerificationTarget `json:"target"`
	Service    string             `json:"service,omitempty" enum:"zerobounce,kickbox,neverbounce,bouncer,emailable,"`
	SkipRecent *bool              `json:"skipRecent,omitempty" doc:"Skip addresses checked in the last 90 days (default true)"`
}

// resolved is a target turned into addresses.
type resolved struct {
	kind, label string
	ids         []uuid.UUID // contacts (empty for an import)
	emails      []string    // in order, one per id for contacts
	importID    *uuid.UUID
	listID      *uuid.UUID
}

var tierNames = map[string]string{"active": "Active", "cooling": "Cooling", "at_risk": "At risk", "sunset": "Sunset"}

func parseID(raw, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return id, api.Invalid("body.target."+field, "That isn't a valid id.")
	}
	return id, nil
}

// resolve finds the addresses of a target.
func (s *Service) resolve(ctx context.Context, ws *store.Workspace, t VerificationTarget) (resolved, error) {
	r := resolved{kind: t.Kind, label: strings.TrimSpace(t.Label)}
	sel := Selection{}
	switch t.Kind {
	case "list":
		id, err := parseID(t.ListID, "listId")
		if err != nil {
			return r, err
		}
		l, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return r, api.MapError(err, "List")
		}
		r.label, r.listID, sel.ListID = l.Name, &id, &id
	case "segment":
		id, err := parseID(t.SegmentID, "segmentId")
		if err != nil {
			return r, err
		}
		name := ""
		if err := s.Pool.QueryRow(ctx, `SELECT name FROM segments WHERE workspace_id = $1 AND id = $2`, ws.ID, id).Scan(&name); err != nil {
			return r, api.MapError(err, "Segment")
		}
		r.label, sel.SegmentID = name, &id
	case "tier":
		if len(t.Tiers) == 0 {
			return r, api.Invalid("body.target.tiers", "Pick a tier.")
		}
		names := []string{}
		for _, tr := range t.Tiers {
			n, ok := tierNames[tr]
			if !ok {
				return r, api.Invalid("body.target.tiers", "Tiers are active, cooling, at_risk and sunset.")
			}
			names = append(names, n)
		}
		r.label = strings.Join(names, " and ")
		sel.Tiers = t.Tiers
	case "selection":
		if t.Filter != nil {
			b, _ := json.Marshal(t.Filter)
			sel.Filter = b
		} else {
			if len(t.ContactIDs) == 0 {
				return r, api.Invalid("body.target.contactIds", "Select at least one contact.")
			}
			for _, raw := range t.ContactIDs {
				id, err := parseID(raw, "contactIds")
				if err != nil {
					return r, err
				}
				sel.ContactIDs = append(sel.ContactIDs, id)
			}
		}
	case "contact":
		id, err := parseID(t.ContactID, "contactId")
		if err != nil {
			return r, err
		}
		sel.ContactIDs = []uuid.UUID{id}
	case "import":
		id, err := parseID(t.ImportID, "importId")
		if err != nil {
			return r, err
		}
		if s.Imports == nil {
			return r, api.NotFound("Import")
		}
		label, emails, err := s.Imports.NewAddresses(ctx, ws, id)
		if err != nil {
			return r, err
		}
		r.label, r.emails, r.importID = label, emails, &id
		return r, nil
	default:
		return r, api.Invalid("body.target.kind", "Pick what to check.")
	}
	if s.Audience == nil {
		return r, errors.New("verification: no audience")
	}
	ids, err := s.Audience.SelectionIDs(ctx, ws, sel)
	if err != nil {
		return r, err
	}
	if len(ids) == 0 {
		r.ids, r.emails = []uuid.UUID{}, []string{}
	} else {
		rows, err := s.Pool.Query(ctx, `SELECT id, email FROM contacts WHERE workspace_id = $1 AND id = ANY($2) AND erased_at IS NULL ORDER BY id`, ws.ID, ids)
		if err != nil {
			return r, err
		}
		type row struct {
			ID    uuid.UUID
			Email string
		}
		list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
		if err != nil {
			return r, err
		}
		for _, c := range list {
			r.ids = append(r.ids, c.ID)
			r.emails = append(r.emails, c.Email)
		}
	}
	if t.Kind == "contact" {
		if len(r.ids) == 0 {
			return r, api.Err(http.StatusNotFound, api.CodeNotFound, mContactErased.Text(i18n.LangFrom(ctx), nil))
		}
		r.label = r.emails[0]
	}
	if t.Kind == "selection" && r.label == "" {
		r.label = fmt.Sprintf("%d selected contacts", len(r.ids))
		if len(r.ids) == 1 {
			r.label = r.emails[0]
		}
	}
	return r, nil
}

// skip splits the resolved contacts into those to check and those checked
// in the last 90 days.
func (s *Service) skip(ctx context.Context, ws *store.Workspace, r resolved, skipRecent bool) (check []int, skipped int64, service string, on *time.Time, err error) {
	if !skipRecent || len(r.ids) == 0 {
		for i := range r.emails {
			check = append(check, i)
		}
		return check, 0, "", nil, nil
	}
	rows, err := s.Q.VerifCheckedSince(ctx, store.VerifCheckedSinceParams{WorkspaceID: ws.ID, ContactIds: r.ids, Since: s.now().Add(-SkipWithin)})
	if err != nil {
		return nil, 0, "", nil, err
	}
	recent := make(map[uuid.UUID]bool, len(rows))
	services := map[string]int{}
	var latest time.Time
	for _, c := range rows {
		recent[c.ContactID] = true
		services[c.Service]++
		if c.CheckedAt.After(latest) {
			latest = c.CheckedAt
		}
	}
	for i, id := range r.ids {
		if recent[id] {
			skipped++
			continue
		}
		check = append(check, i)
	}
	most := 0
	for k, n := range services {
		if n > most {
			service, most = k, n
		}
	}
	if skipped > 0 {
		on = &latest
	}
	return check, skipped, service, on, nil
}

// pick returns the service to use: the one asked for, else the bulk
// default, else the only one; and every connected service.
func (s *Service) pick(ctx context.Context, ws *store.Workspace, key string) (*store.VerificationService, []store.VerificationService, error) {
	rows, err := s.Q.VerifServiceList(ctx, ws.ID)
	if err != nil {
		return nil, nil, err
	}
	all := make([]store.VerificationService, len(rows))
	var chosen *store.VerificationService
	for i := range rows {
		all[i] = rows[i].VerificationService
	}
	for i := range all {
		if key != "" && all[i].Service == key || key == "" && all[i].DefaultBulk {
			chosen = &all[i]
		}
	}
	if chosen == nil && key == "" && len(all) > 0 {
		chosen = &all[0]
	}
	return chosen, all, nil
}

// Estimate says what a use would check and cost.
func (s *Service) Estimate(ctx context.Context, ws *store.Workspace, in VerificationEstimateInput) (VerificationEstimate, error) {
	p := api.PrincipalFrom(ctx)
	skipRecent := in.SkipRecent == nil || *in.SkipRecent
	out := VerificationEstimate{Kind: in.Target.Kind, SkipRecent: skipRecent, Services: []VerificationServiceChoice{},
		CanVerify: p.Can(authz.SubscribersVerify), VerifyHelpers: Helpers(ctx, s.Q, ws.ID, authz.SubscribersVerify),
		CanManage: p.Can(authz.IntegrationsManage), ManageHelpers: Helpers(ctx, s.Q, ws.ID, authz.IntegrationsManage)}
	chosen, rows, err := s.pick(ctx, ws, in.Service)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Services = append(out.Services, VerificationServiceChoice{Key: r.Service, Name: ServiceName(r.Service), Balance: r.Balance,
			Price: Money(r.PriceMicros).Dollars(), Default: r.DefaultBulk, RegionLabel: RegionLabel(r.Service, r.Region)})
	}
	out.Connected = len(rows) > 0
	r, err := s.resolve(ctx, ws, in.Target)
	if err != nil {
		return out, err
	}
	out.Label = r.label
	out.Contacts = int64(len(r.emails))
	check, skipped, svc, on, err := s.skip(ctx, ws, r, skipRecent)
	if err != nil {
		return out, err
	}
	out.Skipped, out.ToCheck = skipped, int64(len(check))
	if svc != "" {
		out.SkippedService, out.SkippedOn = ServiceName(svc), on
	}
	if chosen != nil {
		out.Service, out.ServiceName = chosen.Service, ServiceName(chosen.Service)
		out.RegionLabel = RegionLabel(chosen.Service, chosen.Region)
		out.Price = Money(chosen.PriceMicros).Dollars()
		out.Cost = cost(out.ToCheck, chosen.PriceMicros)
		out.Balance, out.BalanceAt = chosen.Balance, chosen.BalanceAt
		if chosen.Balance != nil {
			after := *chosen.Balance - out.ToCheck
			out.After = &after
			out.Enough = after >= 0
		} else {
			out.Enough = true
		}
	}
	return out, nil
}

// VerificationStartInput starts a use.
type VerificationStartInput struct {
	Target     VerificationTarget `json:"target"`
	Service    string             `json:"service,omitempty" enum:"zerobounce,kickbox,neverbounce,bouncer,emailable,"`
	SkipRecent *bool              `json:"skipRecent,omitempty"`
	// Origin is the screen it started from (list_health, readiness,
	// contacts, profile, import, segments, growth).
	Origin     string `json:"origin,omitempty" enum:"list_health,readiness,contacts,profile,import,segments,growth,"`
	CampaignID string `json:"campaignId,omitempty"`
}

// VerificationJobView is a use's progress and results.
type VerificationJobView struct {
	ID          string `json:"id"`
	Kind        string `json:"kind" enum:"import,list,segment,tier,selection,contact"`
	Label       string `json:"label"`
	Origin      string `json:"origin"`
	Service     string `json:"service"`
	ServiceName string `json:"serviceName"`
	RegionLabel string `json:"regionLabel"`
	Status      string `json:"status" enum:"queued,running,paused,stopped,cancelled,finished"`
	StopReason  string `json:"stopReason" enum:",no_credits,service_down,key_rejected,cancelled"`
	Total       int64  `json:"total"`
	Skipped     int64  `json:"skipped"`
	Checked     int64  `json:"checked"`
	Valid       int64  `json:"valid"`
	Invalid     int64  `json:"invalid"`
	Risky       int64  `json:"risky"`
	Unknown     int64  `json:"unknown"`
	Suppressed  int64  `json:"suppressed"`
	// Reasons counts the invalid and risky answers by reason
	// (invalid_address, spam_trap, catch_all, role, disposable…).
	Reasons       map[string]int64 `json:"reasons"`
	Price         float64          `json:"price"`
	Cost          float64          `json:"cost"`
	Estimated     float64          `json:"estimated"`
	BalanceBefore *int64           `json:"balanceBefore,omitempty"`
	BalanceAfter  *int64           `json:"balanceAfter,omitempty"`
	StartedBy     string           `json:"startedBy"`
	StoppedBy     string           `json:"stoppedBy,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	StartedAt     *time.Time       `json:"startedAt,omitempty"`
	FinishedAt    *time.Time       `json:"finishedAt,omitempty"`
	// ETASeconds while running, at the service's pace.
	ETASeconds *int64 `json:"etaSeconds,omitempty"`
	// RecheckFrom: unknown results can be checked again from then.
	RecheckFrom *time.Time `json:"recheckFrom,omitempty"`
	ErrorSince  *time.Time `json:"errorSince,omitempty"`
	NextTryAt   *time.Time `json:"nextTryAt,omitempty"`
	RetryUntil  *time.Time `json:"retryUntil,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	ListID      string     `json:"listId,omitempty"`
	ImportID    string     `json:"importId,omitempty"`
	CampaignID  string     `json:"campaignId,omitempty"`
	Tiers       []string   `json:"tiers"`
	CanVerify   bool       `json:"canVerify"`
	CanExport   bool       `json:"canExport"`
}

type jobTarget struct {
	ListID    string                  `json:"listId,omitempty"`
	SegmentID string                  `json:"segmentId,omitempty"`
	Tiers     []string                `json:"tiers,omitempty"`
	Filter    *audience.ContactFilter `json:"filter,omitempty"`
	Count     int                     `json:"count,omitempty"`
}

func (s *Service) jobView(ctx context.Context, j store.VerificationJob, startedBy, stoppedBy string) VerificationJobView {
	p := api.PrincipalFrom(ctx)
	v := VerificationJobView{ID: j.ID.String(), Kind: j.Kind, Label: j.Label, Origin: j.Origin, Service: j.Service, ServiceName: ServiceName(j.Service),
		Status: j.Status, StopReason: j.StopReason, Total: int64(j.Total), Skipped: int64(j.Skipped), Checked: int64(j.Checked),
		Valid: int64(j.Valid), Invalid: int64(j.Invalid), Risky: int64(j.Risky), Unknown: int64(j.Unknown), Suppressed: int64(j.Suppressed),
		Reasons: map[string]int64{}, Price: Money(j.PriceMicros).Dollars(), Cost: cost(int64(j.Checked), j.PriceMicros),
		Estimated: cost(int64(j.Total), j.PriceMicros), BalanceBefore: j.BalanceBefore, BalanceAfter: j.BalanceAfter,
		StartedBy: startedBy, StoppedBy: stoppedBy, CreatedAt: j.CreatedAt, StartedAt: j.StartedAt, FinishedAt: j.FinishedAt,
		ErrorSince: j.ErrorSince, NextTryAt: j.NextTryAt, RetryUntil: j.RetryUntil, LastError: j.LastError, Tiers: []string{},
		CanVerify: p.Can(authz.SubscribersVerify), CanExport: p.Can(authz.SubscribersExport)}
	_ = json.Unmarshal(j.Reasons, &v.Reasons)
	var t jobTarget
	_ = json.Unmarshal(j.Target, &t)
	v.ListID = t.ListID
	if t.Tiers != nil {
		v.Tiers = t.Tiers
	}
	if j.ImportID != nil {
		v.ImportID = j.ImportID.String()
	}
	if j.CampaignID != nil {
		v.CampaignID = j.CampaignID.String()
	}
	if j.Unknown > 0 && j.FinishedAt != nil {
		r := j.FinishedAt.Add(RecheckAfter)
		v.RecheckFrom = &r
	}
	if j.Status == "running" || j.Status == "queued" {
		left := int64(j.Total - j.Checked)
		pace := 10.0
		if d, ok := adapter.Lookup(j.Service); ok && d.PerSecond > 0 {
			pace = d.PerSecond
		}
		eta := int64(float64(left)/pace) + 1
		v.ETASeconds = &eta
	}
	if svc, err := s.Q.VerifServiceByKey(ctx, store.VerifServiceByKeyParams{WorkspaceID: j.WorkspaceID, Service: j.Service}); err == nil {
		v.RegionLabel = RegionLabel(j.Service, svc.Region)
	} else {
		v.RegionLabel = RegionLabel(j.Service, "")
	}
	return v
}

// Job returns a use.
func (s *Service) Job(ctx context.Context, ws *store.Workspace, id uuid.UUID) (VerificationJobView, error) {
	j, err := s.Q.VerifJobGet(ctx, store.VerifJobGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return VerificationJobView{}, api.MapError(err, "Verification job")
	}
	return s.jobView(ctx, j.VerificationJob, j.StartedByName, j.StoppedByName), nil
}

// Start creates a job and queues it. Every use is audited: who, which
// service, what, how many addresses and the estimated cost.
func (s *Service) Start(ctx context.Context, ws *store.Workspace, in VerificationStartInput) (VerificationJobView, error) {
	return s.startJob(ctx, ws, in, false)
}

// startJob creates a job; it's queued unless the caller runs it itself.
func (s *Service) startJob(ctx context.Context, ws *store.Workspace, in VerificationStartInput, runHere bool) (VerificationJobView, error) {
	l := i18n.LangFrom(ctx)
	chosen, _, err := s.pick(ctx, ws, in.Service)
	if err != nil {
		return VerificationJobView{}, err
	}
	if chosen == nil {
		h := Helpers(ctx, s.Q, ws.ID, authz.IntegrationsManage)
		msg := mNoService.Text(l, nil)
		if !api.PrincipalFrom(ctx).Can(authz.IntegrationsManage) && len(h) > 0 {
			msg = mNoServiceAsk.Text(l, i18n.Args{"helpers": i18n.OrList(l.Locale, h)})
		}
		return VerificationJobView{}, api.Err(http.StatusConflict, "no_verification_service", msg)
	}
	skipRecent := in.SkipRecent == nil || *in.SkipRecent
	r, err := s.resolve(ctx, ws, in.Target)
	if err != nil {
		return VerificationJobView{}, err
	}
	check, skipped, _, _, err := s.skip(ctx, ws, r, skipRecent)
	if err != nil {
		return VerificationJobView{}, err
	}
	if len(check) == 0 {
		return VerificationJobView{}, api.Err(http.StatusConflict, "nothing_to_check", mNothingToCheck.Text(l, nil))
	}
	var campaign *uuid.UUID
	if in.CampaignID != "" {
		if id, err := uuid.Parse(in.CampaignID); err == nil {
			campaign = &id
		}
	}
	target := jobTarget{ListID: in.Target.ListID, SegmentID: in.Target.SegmentID, Tiers: in.Target.Tiers, Filter: in.Target.Filter, Count: len(check)}
	tb, _ := json.Marshal(target)
	id := uuid.Must(uuid.NewV7())
	emails := make([]string, len(check))
	ids := make([]uuid.UUID, len(check))
	pos := make([]int32, len(check))
	for i, k := range check {
		emails[i] = strings.ToLower(r.emails[k])
		if k < len(r.ids) {
			ids[i] = r.ids[k]
		}
		pos[i] = int32(i)
	}
	now := s.now()
	estimated := cost(int64(len(check)), chosen.PriceMicros)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifJobInsert(ctx, store.VerifJobInsertParams{ID: id, WorkspaceID: ws.ID, ServiceID: &chosen.ID, Service: chosen.Service,
			Kind: r.kind, Target: tb, Label: r.label, Origin: in.Origin, CampaignID: campaign, ImportID: r.importID,
			SkipRecent: skipRecent, Total: int32(len(check)), Skipped: int32(skipped), PriceMicros: chosen.PriceMicros,
			BalanceBefore: chosen.Balance, StartedBy: actor(ctx), CreatedAt: now}); err != nil {
			return err
		}
		if err := q.VerifJobItemsInsert(ctx, store.VerifJobItemsInsertParams{JobID: id, Emails: emails, ContactIds: ids, Positions: pos}); err != nil {
			return err
		}
		if err := q.VerifServiceUsed(ctx, store.VerifServiceUsedParams{ID: chosen.ID, At: &now, By: actor(ctx), What: useWhat(r.kind, r.label)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.job_start", TargetType: "verification_job",
			TargetID: id.String(), TargetName: r.label,
			Metadata: map[string]any{
				"summary": fmt.Sprintf("Verify %d addresses with %s", len(check), ServiceName(chosen.Service)),
				"service": chosen.Service, "what": r.kind, "label": r.label, "addresses": len(check), "skipped": skipped,
				"estimatedCost": estimated, "region": RegionLabel(chosen.Service, chosen.Region), "origin": in.Origin,
			}})
	})
	if err != nil {
		return VerificationJobView{}, err
	}
	if !runHere {
		if err := s.enqueue(ctx, id); err != nil {
			return VerificationJobView{}, err
		}
	}
	return s.Job(ctx, ws, id)
}

// useWhat words a use for the service's "last used" line.
func useWhat(kind, label string) string {
	switch kind {
	case "import":
		return "import preview"
	case "tier":
		return label + " tier"
	case "contact":
		return "one contact"
	}
	return label
}

// JobArgs works on a verification job.
type JobArgs struct {
	JobID uuid.UUID `json:"jobId"`
}

// Kind implements river.JobArgs.
func (JobArgs) Kind() string { return "verification.job" }

// JobWorker runs JobArgs.
type JobWorker struct {
	river.WorkerDefaults[JobArgs]
	Svc *Service
}

// Work implements river.Worker: a slice of up to four minutes, then the
// job snoozes until it continues (at once, or a pause's next try), so large
// jobs never hit the job timeout. Two runs never work on one job at once
// (Run claims it).
func (w *JobWorker) Work(ctx context.Context, job *river.Job[JobArgs]) error {
	next, err := w.Svc.Run(ctx, job.Args.JobID, 4*time.Minute)
	if err != nil {
		return err
	}
	if next != nil {
		return river.JobSnooze(max(time.Until(*next), time.Second))
	}
	return nil
}

// Timeout allows a slice plus the last batch.
func (w *JobWorker) Timeout(*river.Job[JobArgs]) time.Duration { return 8 * time.Minute }

// SweepArgs requeues jobs that are due (after a restart, or a pause's next try).
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "verification.sweep" }

// SweepWorker runs SweepArgs.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	Svc *Service
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	jobs, err := w.Svc.Q.VerifJobsRunnable(ctx, ptrTime(w.Svc.now()))
	if err != nil {
		return err
	}
	for _, j := range jobs {
		_ = w.Svc.enqueue(ctx, j.ID)
	}
	return nil
}

// BalanceArgs reads every service's balance (hourly).
type BalanceArgs struct{}

// Kind implements river.JobArgs.
func (BalanceArgs) Kind() string { return "verification.balances" }

// BalanceWorker runs BalanceArgs.
type BalanceWorker struct {
	river.WorkerDefaults[BalanceArgs]
	Svc *Service
}

// Work implements river.Worker.
func (w *BalanceWorker) Work(ctx context.Context, _ *river.Job[BalanceArgs]) error {
	return w.Svc.RefreshBalances(ctx)
}

func (s *Service) client() bool { return !s.Inline && s.Jobs != nil && s.Jobs() != nil }

func (s *Service) enqueue(ctx context.Context, id uuid.UUID) error {
	if !s.client() {
		// Tests and the CLI: run the whole job now.
		for {
			next, err := s.Run(context.WithoutCancel(ctx), id, time.Hour)
			if err != nil || next == nil {
				return err
			}
			if d := time.Until(*next); d > 0 {
				return nil // a pause: the sweep picks it up when it's due
			}
		}
	}
	_, err := s.Jobs().Insert(ctx, JobArgs{JobID: id}, nil)
	return err
}

// counts is a job's running totals.
type counts struct {
	checked, valid, invalid, risky, unknown, suppressed int32
	reasons                                             map[string]int64
}

// Run works on a job for up to budget. It returns when to continue (nil
// when the job ended).
func (s *Service) Run(ctx context.Context, id uuid.UUID, budget time.Duration) (*time.Time, error) {
	j, err := s.Q.VerifJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	switch j.Status {
	case "queued", "running":
	case "paused":
		if j.NextTryAt != nil && j.NextTryAt.After(s.now()) {
			t := *j.NextTryAt
			return &t, nil
		}
	default:
		return nil, nil
	}
	claimed, err := s.Q.VerifJobClaim(ctx, store.VerifJobClaimParams{ID: j.ID, Until: ptrTime(time.Now().Add(budget + 5*time.Minute)), Now: ptrTime(time.Now())})
	if err != nil {
		return nil, err
	}
	if claimed == 0 {
		return nil, nil // another run is on it
	}
	defer func() { _ = s.Q.VerifJobRelease(context.WithoutCancel(ctx), j.ID) }()
	if j.ServiceID == nil {
		return nil, s.endJob(ctx, j, "cancelled", "cancelled", "")
	}
	svc, err := s.Q.VerifServiceByID(ctx, *j.ServiceID)
	if err != nil {
		return nil, s.endJob(ctx, j, "cancelled", "cancelled", "")
	}
	a, d, err := s.open(svc)
	if err != nil {
		return nil, err
	}
	ws, err := s.Q.GetWorkspace(ctx, j.WorkspaceID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := s.Q.VerifJobSetStatus(ctx, store.VerifJobSetStatusParams{ID: j.ID, Status: "running", StartedAt: &now}); err != nil {
		return nil, err
	}
	c := counts{checked: j.Checked, valid: j.Valid, invalid: j.Invalid, risky: j.Risky, unknown: j.Unknown, suppressed: j.Suppressed, reasons: map[string]int64{}}
	_ = json.Unmarshal(j.Reasons, &c.reasons)
	batch := max(d.MaxBatch, 1)
	lim := rate.NewLimiter(rate.Limit(max(d.PerSecond, 0.1)), batch)
	deadline := time.Now().Add(budget)
	for {
		if time.Now().After(deadline) {
			t := s.now()
			return &t, nil
		}
		cur, err := s.Q.VerifJobByID(ctx, j.ID)
		if err != nil {
			return nil, err
		}
		if cur.Status == "cancelled" {
			s.ended(ctx, &ws, cur)
			return nil, nil
		}
		pending, err := s.Q.VerifJobPending(ctx, store.VerifJobPendingParams{JobID: j.ID, Lim: int32(batch)})
		if err != nil {
			return nil, err
		}
		if len(pending) == 0 {
			return nil, s.finish(ctx, &ws, j, svc, a, c)
		}
		if err := lim.WaitN(ctx, len(pending)); err != nil {
			return nil, err
		}
		emails := make([]string, len(pending))
		for i, p := range pending {
			emails[i] = p.Email
		}
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		answers, callErr := a.VerifyBatch(cctx, emails)
		cancel()
		if callErr != nil && errors.Is(callErr, adapter.ErrRefused) && len(answers) < len(emails) {
			// A request the service refuses keeps no result: unknown, with its words.
			answers = append(answers, adapter.Answer{Email: emails[len(answers)], Result: adapter.Unknown, Code: "refused: " + callErr.Error()})
			callErr = nil
		}
		if len(answers) > 0 {
			if err := s.apply(ctx, &ws, j, svc, pending[:len(answers)], answers, &c); err != nil {
				return nil, err
			}
		}
		at := s.now()
		if callErr == nil {
			_ = s.Q.VerifServiceAnswered(ctx, store.VerifServiceAnsweredParams{ID: svc.ID, At: &at, What: useWhat(j.Kind, j.Label) + " job"})
			if cur.ErrorSince != nil {
				_ = s.Q.VerifJobSetStatus(ctx, store.VerifJobSetStatusParams{ID: j.ID, Status: "running"})
			}
			continue
		}
		_ = s.Q.VerifServiceFailed(ctx, store.VerifServiceFailedParams{ID: svc.ID, Error: callErr.Error(), At: &at})
		switch {
		case errors.Is(callErr, adapter.ErrNoCredits):
			zero := int64(0)
			_ = s.Q.VerifServiceSetBalance(ctx, store.VerifServiceSetBalanceParams{ID: svc.ID, Balance: &zero, BalanceAt: &at})
			if err := s.endJob(ctx, j, "stopped", "no_credits", callErr.Error()); err != nil {
				return nil, err
			}
			s.checkLow(ctx, &ws, svc.ID)
			s.endedByID(ctx, &ws, j.ID)
			return nil, nil
		case errors.Is(callErr, adapter.ErrAuth):
			if err := s.endJob(ctx, j, "stopped", "key_rejected", callErr.Error()); err != nil {
				return nil, err
			}
			s.endedByID(ctx, &ws, j.ID)
			return nil, nil
		case adapter.Temporary(callErr):
			since := at
			if cur.ErrorSince != nil {
				since = *cur.ErrorSince
			}
			until := at.Add(time.Hour)
			if cur.RetryUntil != nil {
				until = *cur.RetryUntil
			}
			if at.After(until) {
				if err := s.endJob(ctx, j, "stopped", "service_down", callErr.Error()); err != nil {
					return nil, err
				}
				s.endedByID(ctx, &ws, j.ID)
				return nil, nil
			}
			wait := 5 * time.Minute
			var ce *adapter.CallError
			if errors.As(callErr, &ce) && ce.RetryAfter > 0 && ce.RetryAfter < wait {
				wait = ce.RetryAfter
			} else if errors.Is(callErr, adapter.ErrRateLimited) {
				wait = 30 * time.Second
			}
			next := at.Add(wait)
			if err := s.Q.VerifJobSetStatus(ctx, store.VerifJobSetStatusParams{ID: j.ID, Status: "paused", LastError: callErr.Error(),
				ErrorSince: &since, NextTryAt: &next, RetryUntil: &until}); err != nil {
				return nil, err
			}
			return &next, nil
		default:
			return nil, callErr
		}
	}
}

// apply records a batch of answers in one transaction: the items, each
// contact's latest result, the verify:… tags, and suppressions for the
// invalid ones.
func (s *Service) apply(ctx context.Context, ws *store.Workspace, j store.VerificationJob, svc store.VerificationService, items []store.VerifJobPendingRow, answers []adapter.Answer, c *counts) error {
	at := s.now()
	name := ServiceName(svc.Service)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		emails := make([]string, len(answers))
		results := make([]string, len(answers))
		reasons := make([]string, len(answers))
		codes := make([]string, len(answers))
		var cids []uuid.UUID
		var cres, creas, ccodes, ctags []string
		var invalid []int
		for i, a := range answers {
			emails[i], results[i], reasons[i], codes[i] = items[i].Email, string(a.Result), a.Reason, a.Code
			c.checked++
			switch a.Result {
			case adapter.Valid:
				c.valid++
			case adapter.Invalid:
				c.invalid++
				c.reasons[a.Reason]++
				invalid = append(invalid, i)
			case adapter.Risky:
				c.risky++
				c.reasons[a.Reason]++
			default:
				c.unknown++
			}
			if items[i].ContactID != nil {
				cids = append(cids, *items[i].ContactID)
				cres = append(cres, string(a.Result))
				creas = append(creas, a.Reason)
				ccodes = append(ccodes, a.Code)
				ctags = append(ctags, a.Tag())
			}
		}
		if err := q.VerifJobItemsDone(ctx, store.VerifJobItemsDoneParams{JobID: j.ID, Emails: emails, Results: results, Reasons: reasons, Codes: codes, CheckedAt: &at}); err != nil {
			return err
		}
		if len(cids) > 0 {
			if err := q.VerifContactChecksUpsert(ctx, store.VerifContactChecksUpsertParams{WorkspaceID: ws.ID, Service: svc.Service, JobID: &j.ID,
				CheckedAt: at, ContactIds: cids, Results: cres, Reasons: creas, Codes: ccodes}); err != nil {
				return err
			}
			if err := q.VerifSetTags(ctx, store.VerifSetTagsParams{WorkspaceID: ws.ID, ContactIds: cids, Tags: ctags}); err != nil {
				return err
			}
		}
		if len(invalid) > 0 && j.Kind != "import" {
			n, err := suppress(ctx, tx, q, ws, j, name, svc.Service, items, answers, invalid, at)
			if err != nil {
				return err
			}
			c.suppressed += int32(n)
		}
		rb, _ := json.Marshal(c.reasons)
		return q.VerifJobSaveCounts(ctx, store.VerifJobSaveCountsParams{ID: j.ID, Checked: c.checked, Valid: c.valid, Invalid: c.invalid,
			Risky: c.risky, Unknown: c.unknown, Suppressed: c.suppressed, Reasons: rb})
	})
}

// suppress puts invalid addresses on the shared suppression list for all
// mail, with the service as the source and its code as the original code,
// and suppresses their contacts. It returns how many were added.
func suppress(ctx context.Context, tx pgx.Tx, q *store.Queries, ws *store.Workspace, j store.VerificationJob, name, service string,
	items []store.VerifJobPendingRow, answers []adapter.Answer, invalid []int, at time.Time) (int, error) {
	added := 0
	var cids, evIDs []uuid.UUID
	var notes []string
	for _, i := range invalid {
		a := answers[i]
		reason := "invalid"
		if a.Reason == adapter.ReasonSpamTrap {
			reason = "spam_trap"
		}
		code := name + ": " + a.Code
		_, err := q.InsertSuppression(ctx, store.InsertSuppressionParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: items[i].Email,
			Reason: reason, Source: SuppressionSource(service), OriginalCode: code, Detail: "Verification · " + j.Label,
			CreatedBy: j.StartedBy, Scope: "all", CampaignID: j.CampaignID, Context: "Verification · " + jobContext(j), CreatedAt: at})
		if errors.Is(err, pgx.ErrNoRows) {
			continue // already suppressed
		}
		if err != nil {
			return added, err
		}
		added++
		if items[i].ContactID != nil {
			cids = append(cids, *items[i].ContactID)
			evIDs = append(evIDs, uuid.Must(uuid.NewV7()))
			notes = append(notes, code)
		}
	}
	if len(cids) > 0 {
		if _, err := q.VerifSuppressContacts(ctx, store.VerifSuppressContactsParams{WorkspaceID: ws.ID, Ids: cids}); err != nil {
			return added, err
		}
		if err := q.VerifConsentEvents(ctx, store.VerifConsentEventsParams{Ids: evIDs, ContactIds: cids, Notes: notes, Actor: j.StartedBy, At: at}); err != nil {
			return added, err
		}
	}
	return added, nil
}

// jobContext names a job's target the way the suppression list's "When ·
// What" column shows it ("Sunset tier").
func jobContext(j store.VerificationJob) string {
	switch j.Kind {
	case "tier":
		return j.Label + " tier"
	case "contact":
		return "one contact"
	}
	return j.Label
}

// finish ends a job whose addresses are all checked: the balance is read
// again, the job is audited with its results, and the starter is told.
func (s *Service) finish(ctx context.Context, ws *store.Workspace, j store.VerificationJob, svc store.VerificationService, a adapter.Adapter, c counts) error {
	at := s.now()
	var after *int64
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	if n, err := a.Balance(cctx); err == nil {
		after = &n
		_ = s.Q.VerifServiceSetBalance(ctx, store.VerifServiceSetBalanceParams{ID: svc.ID, Balance: &n, BalanceAt: &at})
	}
	cancel()
	rb, _ := json.Marshal(c.reasons)
	if err := s.Q.VerifJobSaveCounts(ctx, store.VerifJobSaveCountsParams{ID: j.ID, Checked: c.checked, Valid: c.valid, Invalid: c.invalid,
		Risky: c.risky, Unknown: c.unknown, Suppressed: c.suppressed, Reasons: rb, BalanceAfter: after}); err != nil {
		return err
	}
	if err := s.endJob(ctx, j, "finished", "", ""); err != nil {
		return err
	}
	s.checkLow(ctx, ws, svc.ID)
	s.endedByID(ctx, ws, j.ID)
	return nil
}

// endJob sets a job's final status and audits its results as the system.
func (s *Service) endJob(ctx context.Context, j store.VerificationJob, status, reason, lastError string) error {
	at := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifJobSetStatus(ctx, store.VerifJobSetStatusParams{ID: j.ID, Status: status, StopReason: reason, LastError: lastError, FinishedAt: &at}); err != nil {
			return err
		}
		cur, err := q.VerifJobByID(ctx, j.ID)
		if err != nil {
			return err
		}
		action := "verification.job_finish"
		if status == "stopped" {
			action = "verification.job_stop"
		}
		if status == "cancelled" && reason == "cancelled" && cur.StoppedBy == nil {
			action = "verification.job_cancel"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &j.WorkspaceID, Action: action, TargetType: "verification_job",
			TargetID: j.ID.String(), TargetName: j.Label, Reason: reason,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "ReputeCast"},
			Metadata: map[string]any{"service": j.Service, "checked": cur.Checked, "valid": cur.Valid, "invalid": cur.Invalid,
				"risky": cur.Risky, "unknown": cur.Unknown, "suppressed": cur.Suppressed, "cost": cost(int64(cur.Checked), cur.PriceMicros),
				"summary": fmt.Sprintf("%s checked %d of %d addresses: %d suppressed", ServiceName(j.Service), cur.Checked, cur.Total, cur.Suppressed)}})
	})
}

func (s *Service) endedByID(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
	if j, err := s.Q.VerifJobByID(ctx, id); err == nil {
		s.ended(ctx, ws, j)
	}
}

func (s *Service) ended(ctx context.Context, ws *store.Workspace, j store.VerificationJob) {
	if s.Notify != nil && j.Kind != "contact" {
		s.Notify.JobEnded(ctx, ws, j)
	}
}

// Cancel stops a job; the results so far are kept.
func (s *Service) Cancel(ctx context.Context, ws *store.Workspace, id uuid.UUID) (VerificationJobView, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.VerifJobCancel(ctx, store.VerifJobCancelParams{ID: id, WorkspaceID: ws.ID, StoppedBy: actor(ctx), At: ptrTime(s.now())})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Conflict(mJobFinished.Text(i18n.LangFrom(ctx), nil))
		}
		j, err := q.VerifJobByID(ctx, id)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.job_cancel", TargetType: "verification_job",
			TargetID: id.String(), TargetName: j.Label,
			Metadata: map[string]any{"service": j.Service, "checked": j.Checked, "total": j.Total, "suppressed": j.Suppressed,
				"summary": fmt.Sprintf("Stopped at %d of %d; results kept", j.Checked, j.Total)}})
	})
	if err != nil {
		return VerificationJobView{}, err
	}
	return s.Job(ctx, ws, id)
}

func ptrTime(t time.Time) *time.Time { return &t }

// Resume continues a job that stopped for credits or an outage.
func (s *Service) Resume(ctx context.Context, ws *store.Workspace, id uuid.UUID) (VerificationJobView, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.VerifJobResume(ctx, store.VerifJobResumeParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Conflict(mJobNotStopped.Text(i18n.LangFrom(ctx), nil))
		}
		j, err := q.VerifJobByID(ctx, id)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.job_resume", TargetType: "verification_job",
			TargetID: id.String(), TargetName: j.Label,
			Metadata: map[string]any{"service": j.Service, "left": j.Total - j.Checked, "estimatedCost": cost(int64(j.Total-j.Checked), j.PriceMicros)}})
	})
	if err != nil {
		return VerificationJobView{}, err
	}
	if err := s.enqueue(ctx, id); err != nil {
		return VerificationJobView{}, err
	}
	return s.Job(ctx, ws, id)
}

// ResultsCSV writes a job's results: addresses, so it needs
// subscribers:export and is audited like an export.
func (s *Service) ResultsCSV(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	jr, err := s.Q.VerifJobGet(ctx, store.VerifJobGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return nil, "", api.MapError(err, "Verification job")
	}
	j := jr.VerificationJob
	rows, err := s.Q.VerifJobItemsAll(ctx, id)
	if err != nil {
		return nil, "", err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"email", "result", "reason", "service_code", "checked_at"})
	for _, r := range rows {
		at := ""
		if r.CheckedAt != nil {
			at = r.CheckedAt.UTC().Format(time.RFC3339)
		}
		_ = w.Write([]string{r.Email, r.Result, r.Reason, ServiceName(j.Service) + ": " + r.Code, at})
	}
	w.Flush()
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "verification.results_export", TargetType: "verification_job",
		TargetID: id.String(), TargetName: j.Label, Metadata: map[string]any{"rows": len(rows), "service": j.Service}}); err != nil {
		return nil, "", err
	}
	name := "verification-" + strings.NewReplacer(" ", "-", "/", "-", "“", "", "”", "", ":", "").Replace(strings.ToLower(j.Label)) + ".csv"
	return []byte(b.String()), name, nil
}

// ImportResults returns the answers of an import's latest check, by address.
func (s *Service) ImportResults(ctx context.Context, ws *store.Workspace, importID uuid.UUID) (*store.VerificationJob, map[string]adapter.Answer, error) {
	j, err := s.Q.VerifLatestImportJob(ctx, store.VerifLatestImportJobParams{WorkspaceID: ws.ID, ImportID: &importID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.Q.VerifJobResults(ctx, j.ID)
	if err != nil {
		return nil, nil, err
	}
	out := make(map[string]adapter.Answer, len(rows))
	for _, r := range rows {
		out[r.Email] = adapter.Answer{Email: r.Email, Result: adapter.Result(r.Result), Reason: r.Reason, Code: r.Code}
	}
	return &j, out, nil
}
