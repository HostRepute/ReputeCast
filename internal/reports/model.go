package reports

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// ModelSnapshot is a fixed, coarse view of a completed campaign. It contains
// no free text, identifiers, rates, per-provider dimensions or links. A nil
// metric means withheld, not zero. Rounded values need not add up.
type ModelSnapshot struct {
	Status      string           `json:"status"`
	AsOf        time.Time        `json:"asOf"`
	SentFrom    time.Time        `json:"sentFrom"`
	SentThrough time.Time        `json:"sentThrough"`
	Rounding    int64            `json:"rounding"`
	Sent        *int64           `json:"sent,omitempty"`
	Metrics     map[string]int64 `json:"metrics"`
}

// ModelCampaign rechecks current scope even when a snapshot already exists.
// Callers hold current membership/role locks and use one transaction for the
// access check and snapshot. The returned name is human-only metadata.
func ModelCampaign(ctx context.Context, q *store.Queries, p *authz.Principal, id uuid.UUID) (store.ReportModelCampaignRow, error) {
	if p == nil || !p.Can(authz.ReportsView) || !p.Can(authz.CampaignsView) {
		return store.ReportModelCampaignRow{}, errors.New("report access required")
	}
	domains := make([]string, 0, len(p.Scope.SenderDomains))
	for _, domain := range p.Scope.SenderDomains {
		domains = append(domains, strings.ToLower(domain))
	}
	return q.ReportModelCampaign(ctx, store.ReportModelCampaignParams{WorkspaceID: p.WorkspaceID, ID: id, ScopeLists: access.ScopeLists(p), ScopeDomains: domains})
}

// ReadModelSnapshot never refreshes a released snapshot. Reconnects, different
// users and new runs all see the same numbers, preventing temporal differencing
// within a campaign. This does not claim differential privacy across campaigns.
func ReadModelSnapshot(ctx context.Context, q *store.Queries, p *authz.Principal, id uuid.UUID, now time.Time) (ModelSnapshot, error) {
	var out ModelSnapshot
	c, err := ModelCampaign(ctx, q, p, id)
	if err != nil {
		return out, err
	}
	if c.CompletedAt.After(now.Add(-FinalAfter)) {
		return out, errors.New("report not final")
	}
	row, err := q.ReportModelSnapshot(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		cells, readErr := q.ReportCounterCells(ctx, id)
		if readErr != nil {
			return out, readErr
		}
		head, _ := headline(cells, "")
		out = ModelSnapshot{Status: "withheld", AsOf: now.UTC(), SentFrom: c.StartedAt.UTC().Truncate(24 * time.Hour), SentThrough: c.CompletedAt.UTC().Truncate(24 * time.Hour), Rounding: 10, Metrics: map[string]int64{}}
		if head.Sent >= 100 {
			out.Status = "rounded"
			sent := head.Sent / out.Rounding * out.Rounding
			out.Sent = &sent
			// Suppress both a small metric and its complement. Keep opens as
			// one metric: publishing a partition of small buckets would reveal
			// suppressed cells by subtraction. No rates undo the rounding.
			for key, n := range map[string]int64{"delivered": head.Delivered, "humanOpens": head.HumanOpens, "humanClicks": head.Clicks, "unsubscribes": head.Unsubscribes, "bounces": head.Bounces} {
				if n >= 20 && head.Sent-n >= 20 {
					out.Metrics[key] = n / out.Rounding * out.Rounding
				}
			}
		}
		raw, marshalErr := json.Marshal(out)
		if marshalErr != nil {
			return out, marshalErr
		}
		if err := q.ReportModelSnapshotInsert(ctx, store.ReportModelSnapshotInsertParams{CampaignID: id, Content: raw, CreatedAt: now}); err != nil {
			return out, err
		}
		// Read the winner even if another caller inserted first.
		row, err = q.ReportModelSnapshot(ctx, id)
	}
	if err != nil {
		return out, err
	}
	if row.Content == nil || !now.Before(row.CreatedAt.Add(30*24*time.Hour)) {
		return ModelSnapshot{}, errors.New("report snapshot expired")
	}
	err = json.Unmarshal(row.Content, &out)
	return out, err
}

type ModelRetentionArgs struct{}

func (ModelRetentionArgs) Kind() string { return "reports.model_retention" }
func (ModelRetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

type ModelRetentionWorker struct {
	river.WorkerDefaults[ModelRetentionArgs]
	S *Service
}

func (w *ModelRetentionWorker) Work(ctx context.Context, _ *river.Job[ModelRetentionArgs]) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for range 10 {
		n, err := w.S.Q.ReportModelSnapshotsPurge(ctx, w.S.now().Add(-30*24*time.Hour))
		if err != nil {
			return err
		}
		if n < 500 {
			break
		}
	}
	return nil
}
