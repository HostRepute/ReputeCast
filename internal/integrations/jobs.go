package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// ContactEventsArgs turns queued contact events (contact.subscribed,
// .unsubscribed, .updated) into webhook deliveries.
type ContactEventsArgs struct{}

// Kind implements river.JobArgs.
func (ContactEventsArgs) Kind() string { return "integrations.contact_events" }

// InsertOpts keeps one run at a time on the events queue.
func (ContactEventsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Second}}
}

// ContactEventsWorker delivers queued contact events.
type ContactEventsWorker struct {
	river.WorkerDefaults[ContactEventsArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ContactEventsWorker) Work(ctx context.Context, _ *river.Job[ContactEventsArgs]) error {
	for range 20 {
		n, err := w.S.DeliverContactEvents(ctx, 500)
		if err != nil || n < 500 {
			return err
		}
	}
	return nil
}

// DeliverContactEvents queues webhook deliveries for up to limit contact
// events, with the contact as it is now. It returns how many it took.
func (s *Service) DeliverContactEvents(ctx context.Context, limit int) (int, error) {
	n := 0
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		evs, err := q.WebhookContactEventsTake(ctx, int32(limit))
		if err != nil || len(evs) == 0 {
			return err
		}
		n = len(evs)
		ids := make([]uuid.UUID, 0, len(evs))
		for _, e := range evs {
			ids = append(ids, e.ContactID)
		}
		rows, err := q.IntegrationEventContacts(ctx, ids)
		if err != nil {
			return err
		}
		contacts := map[uuid.UUID]store.IntegrationEventContactsRow{}
		for _, c := range rows {
			contacts[c.ID] = c
		}
		slugs := map[uuid.UUID]string{}
		listNames := map[uuid.UUID]map[uuid.UUID]string{}
		for _, e := range evs {
			c, ok := contacts[e.ContactID]
			if !ok || c.ErasedAt != nil {
				continue
			}
			slug, ok := slugs[e.WorkspaceID]
			if !ok {
				ws, err := q.GetWorkspace(ctx, e.WorkspaceID)
				if err != nil {
					continue
				}
				slug = ws.Slug
				slugs[e.WorkspaceID] = slug
				names := map[uuid.UUID]string{}
				if ls, err := q.IntegrationListNames(ctx, e.WorkspaceID); err == nil {
					for _, l := range ls {
						names[l.ID] = l.Name
					}
				}
				listNames[e.WorkspaceID] = names
			}
			data := ContactEventData(c)
			var extra map[string]any
			_ = json.Unmarshal(e.Data, &extra)
			if e.ListID != nil {
				data["list_id"] = e.ListID.String()
				data["list_name"] = listNames[e.WorkspaceID][*e.ListID]
			}
			for k, v := range extra {
				data[k] = v
			}
			if _, err := webhooks.Emit(ctx, tx, e.WorkspaceID, slug, e.Event, data); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

// ContactEventData is a contact as contact events carry it.
func ContactEventData(c store.IntegrationEventContactsRow) map[string]any {
	var fields map[string]any
	_ = json.Unmarshal(c.Attributes, &fields)
	if fields == nil {
		fields = map[string]any{}
	}
	var lists []map[string]any
	_ = json.Unmarshal(c.Lists, &lists)
	if lists == nil {
		lists = []map[string]any{}
	}
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{
		"contact_id": c.ID.String(), "email": c.Email, "first_name": c.FirstName, "last_name": c.LastName, "company": c.Company,
		"country": c.Country, "timezone": c.Timezone, "language": c.Language, "tags": tags, "fields": fields, "status": c.Status, "lists": lists,
	}
}

// MaintenanceArgs drops calls older than 7 days, workflows not seen for 30,
// and the bodies of calls about contacts erased since; then recomputes
// which integrations need attention.
type MaintenanceArgs struct{}

// Kind implements river.JobArgs.
func (MaintenanceArgs) Kind() string { return "integrations.maintenance" }

// InsertOpts runs it on the maintenance queue, once at a time.
func (MaintenanceArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Minute}}
}

// MaintenanceWorker runs MaintenanceArgs.
type MaintenanceWorker struct {
	river.WorkerDefaults[MaintenanceArgs]
	S *Service
}

// Work implements river.Worker.
func (w *MaintenanceWorker) Work(ctx context.Context, _ *river.Job[MaintenanceArgs]) error {
	now := w.S.now()
	if _, err := w.S.Q.IntegrationCallsPurge(ctx, now.Add(-CallRetention)); err != nil {
		return err
	}
	if _, err := w.S.Q.IntegrationWorkflowsPurge(ctx, now.Add(-WorkflowRetention)); err != nil {
		return err
	}
	if _, err := w.S.Q.IntegrationCallsForgetErased(ctx); err != nil {
		return err
	}
	return w.S.CheckHealth(ctx)
}

// PluginReleaseArgs fetches the latest WordPress plugin version with the
// daily update check (Instance admin › Instance & updates). With the check
// off, nothing is fetched and pages compare with the version this release
// knows.
type PluginReleaseArgs struct{}

// Kind implements river.JobArgs.
func (PluginReleaseArgs) Kind() string { return "integrations.plugin_release" }

// InsertOpts runs it on the maintenance queue, at most once an hour.
func (PluginReleaseArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// PluginReleaseWorker runs PluginReleaseArgs.
type PluginReleaseWorker struct {
	river.WorkerDefaults[PluginReleaseArgs]
	S *Service
	// URL overrides the WordPress plugin directory's API (tests).
	URL string
}

// PluginInfoURL is the WordPress plugin directory's answer about the plugin.
const PluginInfoURL = "https://api.wordpress.org/plugins/info/1.2/?action=plugin_information&request%5Bslug%5D=reputecast"

// Work implements river.Worker.
func (w *PluginReleaseWorker) Work(ctx context.Context, _ *river.Job[PluginReleaseArgs]) error {
	if w.S.Cfg != nil && !w.S.Cfg.Updates.Check {
		return nil
	}
	st, err := w.S.Q.GetInstanceSettings(ctx)
	if err != nil || !st.UpdateCheck {
		return err
	}
	if r, err := w.S.Q.IntegrationReleaseGet(ctx, KindWordPress); err == nil && w.S.now().Sub(r.CheckedAt) < 20*time.Hour {
		return nil
	}
	u := w.URL
	if u == "" {
		u = PluginInfoURL
	}
	v, err := w.S.fetchPluginVersion(ctx, u)
	if err != nil {
		w.S.log().InfoContext(ctx, "WordPress plugin release check", "err", err)
		return nil
	}
	return w.S.Q.IntegrationReleaseSet(ctx, store.IntegrationReleaseSetParams{Kind: KindWordPress, Version: v, Source: "wordpress.org"})
}

// fetchPluginVersion reads the version from the plugin directory's answer:
// one plain GET that sends nothing about the instance.
func (s *Service) fetchPluginVersion(ctx context.Context, u string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ReputeCast-update-check")
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("plugin directory answered %d", resp.StatusCode)
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&body); err != nil {
		return "", err
	}
	v := strings.TrimSpace(body.Version)
	if v == "" || len(v) > 20 {
		return "", fmt.Errorf("no version in the answer")
	}
	return v, nil
}

// RegisterJobs adds the hub's workers and periodic jobs.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &ContactEventsWorker{S: s})
	river.AddWorker(reg.Workers, &MaintenanceWorker{S: s})
	river.AddWorker(reg.Workers, &PluginReleaseWorker{S: s})
	reg.Every(10*time.Second, func() (river.JobArgs, *river.InsertOpts) { return ContactEventsArgs{}, nil }, true)
	reg.Every(10*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return MaintenanceArgs{}, nil }, true)
	reg.Every(6*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return PluginReleaseArgs{}, nil }, true)
}
