package instance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Instance admin › System (review 2026-11 OP-10): the background jobs that
// failed, with retry, cancel and delete; the queues and their lag; the
// database's size and largest tables; and the file storage.

// SystemView is the System tab.
type SystemView struct {
	Jobs     SystemJobs     `json:"jobs"`
	Queues   []SystemQueue  `json:"queues"`
	Database SystemDatabase `json:"database"`
	Files    SystemFiles    `json:"files"`
}

// SystemJobs counts the jobs in every queue.
type SystemJobs struct {
	Waiting   int64 `json:"waiting" doc:"Due and waiting for a worker"`
	Scheduled int64 `json:"scheduled" doc:"Due later"`
	Running   int64 `json:"running"`
	Retrying  int64 `json:"retrying" doc:"Failed, waiting for their next attempt"`
	GivenUp   int64 `json:"givenUp" doc:"Out of attempts, kept 7 days"`
	// LagSeconds is how long the oldest due job has waited (0 when none).
	LagSeconds float64 `json:"lagSeconds"`
}

// SystemQueue is one queue.
type SystemQueue struct {
	Name       string   `json:"name"`
	Waiting    int64    `json:"waiting"`
	Scheduled  int64    `json:"scheduled"`
	Running    int64    `json:"running"`
	LagSeconds *float64 `json:"lagSeconds,omitempty" doc:"How long the oldest due job has waited; absent when none waits"`
}

// SystemDatabase is the database's size and schema.
type SystemDatabase struct {
	Version        string        `json:"version"`
	Bytes          int64         `json:"bytes"`
	Connections    int64         `json:"connections" doc:"Open connections to this database, from every process"`
	MaxConnections int64         `json:"maxConnections"`
	SchemaVersion  int64         `json:"schemaVersion"`
	SchemaCurrent  bool          `json:"schemaCurrent"`
	Tables         []SystemTable `json:"tables" doc:"The 10 largest tables, partitions added up"`
}

// SystemTable is one table's size.
type SystemTable struct {
	Name       string `json:"name"`
	Rows       int64  `json:"rows" doc:"Postgres's estimate"`
	Bytes      int64  `json:"bytes" doc:"With its indexes and TOAST"`
	IndexBytes int64  `json:"indexBytes"`
}

// SystemFiles is the file storage: the local files folder or an
// S3-compatible bucket (storage.backend). Never its keys.
type SystemFiles struct {
	Backend string `json:"backend" enum:"local,s3"`
	// Dir is the local folder (local).
	Dir string `json:"dir,omitempty"`
	// Bucket, Endpoint, Region and Prefix are the bucket's (s3); Endpoint
	// is empty for AWS.
	Bucket    string    `json:"bucket,omitempty"`
	Endpoint  string    `json:"endpoint,omitempty"`
	Region    string    `json:"region,omitempty"`
	Prefix    string    `json:"prefix,omitempty"`
	Files     int64     `json:"files"`
	Bytes     int64     `json:"bytes"`
	CountedAt time.Time `json:"countedAt"`
	// Partial is true when the count stopped early (a very large folder
	// or bucket).
	Partial bool   `json:"partial,omitempty"`
	Missing bool   `json:"missing,omitempty" doc:"The local folder doesn't exist yet (nothing uploaded)"`
	Error   string `json:"error,omitempty"`
}

// SystemJob is a failed job.
type SystemJob struct {
	ID          int64            `json:"id"`
	Kind        string           `json:"kind"`
	Queue       string           `json:"queue"`
	State       string           `json:"state" enum:"retrying,given_up"`
	Attempt     int              `json:"attempt"`
	MaxAttempts int              `json:"maxAttempts"`
	Error       string           `json:"error,omitempty" doc:"The last attempt's error, cut to 500 characters"`
	FailedAt    *time.Time       `json:"failedAt,omitempty"`
	NextAt      *time.Time       `json:"nextAt,omitempty" doc:"A retrying job's next attempt"`
	Workspace   *SystemWorkspace `json:"workspace,omitempty"`
}

// SystemWorkspace names a job's workspace.
type SystemWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// SystemJobList is a page of failed jobs.
type SystemJobList struct {
	Items      []SystemJob `json:"items"`
	NextCursor string      `json:"nextCursor,omitempty"`
}

// SystemJobResult is a job after an action.
type SystemJobResult struct {
	ID    int64  `json:"id"`
	State string `json:"state" enum:"available,cancelled,deleted"`
}

// System reads the System tab.
func (s *Service) System(ctx context.Context) (SystemView, error) {
	var out SystemView
	rows, err := s.Pool.Query(ctx, `
		SELECT queue, state::text, count(*),
		       coalesce(extract(epoch FROM now() - min(scheduled_at) FILTER (WHERE state = 'available' AND scheduled_at <= now())), -1)::float8
		FROM river_job
		WHERE state IN ('available', 'scheduled', 'running', 'retryable', 'discarded')
		GROUP BY 1, 2`)
	if err != nil {
		return out, err
	}
	queues := map[string]*SystemQueue{}
	for _, q := range []string{"sending", "events", "default", "system", "maintenance"} {
		queues[q] = &SystemQueue{Name: q}
	}
	for rows.Next() {
		var queue, state string
		var n int64
		var lag float64
		if err := rows.Scan(&queue, &state, &n, &lag); err != nil {
			rows.Close()
			return out, err
		}
		q := queues[queue]
		if q == nil {
			q = &SystemQueue{Name: queue}
			queues[queue] = q
		}
		switch state {
		case "available":
			q.Waiting += n
			out.Jobs.Waiting += n
			if lag >= 0 {
				l := lag
				q.LagSeconds = &l
				out.Jobs.LagSeconds = max(out.Jobs.LagSeconds, lag)
			}
		case "scheduled":
			q.Scheduled += n
			out.Jobs.Scheduled += n
		case "running":
			q.Running += n
			out.Jobs.Running += n
		case "retryable":
			out.Jobs.Retrying += n
		case "discarded":
			out.Jobs.GivenUp += n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	for _, name := range []string{"sending", "events", "default", "system", "maintenance"} {
		out.Queues = append(out.Queues, *queues[name])
		delete(queues, name)
	}
	for _, q := range queues { // queues a newer release added
		out.Queues = append(out.Queues, *q)
	}

	d := &out.Database
	if err := s.Pool.QueryRow(ctx, `
		SELECT current_setting('server_version'), pg_database_size(current_database()),
		       (SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()),
		       current_setting('max_connections')::bigint`).Scan(&d.Version, &d.Bytes, &d.Connections, &d.MaxConnections); err != nil {
		return out, err
	}
	if st, err := db.Schema(ctx, s.Pool); err == nil {
		d.SchemaVersion, d.SchemaCurrent = st.Applied, st.Current()
	}
	rows, err = s.Pool.Query(ctx, `
		WITH t AS (
			SELECT c.oid, c.relname, c.relkind, c.reltuples FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition),
		s AS (
			-- A partitioned table adds up its partitions.
			SELECT t.relname,
			       CASE WHEN t.relkind = 'p' THEN (SELECT coalesce(sum(greatest(p.reltuples, 0)), 0) FROM pg_partition_tree(t.oid) pt JOIN pg_class p ON p.oid = pt.relid WHERE pt.isleaf)
			            ELSE greatest(t.reltuples, 0) END::bigint AS rows,
			       CASE WHEN t.relkind = 'p' THEN (SELECT coalesce(sum(pg_total_relation_size(pt.relid)), 0) FROM pg_partition_tree(t.oid) pt WHERE pt.isleaf)
			            ELSE pg_total_relation_size(t.oid) END::bigint AS bytes,
			       CASE WHEN t.relkind = 'p' THEN (SELECT coalesce(sum(pg_indexes_size(pt.relid)), 0) FROM pg_partition_tree(t.oid) pt WHERE pt.isleaf)
			            ELSE pg_indexes_size(t.oid) END::bigint AS index_bytes
			FROM t)
		SELECT relname, rows, bytes, index_bytes FROM s ORDER BY bytes DESC, relname LIMIT 10`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var t SystemTable
		if err := rows.Scan(&t.Name, &t.Rows, &t.Bytes, &t.IndexBytes); err != nil {
			rows.Close()
			return out, err
		}
		d.Tables = append(d.Tables, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Files = s.filesUsage(ctx)
	return out, nil
}

// The file storage is counted at most every 5 minutes, and for at most 5
// seconds or 200,000 files.
var filesCache struct {
	sync.Mutex
	where string
	at    time.Time
	v     SystemFiles
}

const (
	filesTTL     = 5 * time.Minute
	filesBudget  = 5 * time.Second
	filesEntries = 200_000
)

// errEnough stops a count at filesEntries.
var errEnough = errors.New("instance: enough files counted")

func (s *Service) filesUsage(ctx context.Context) SystemFiles {
	v := SystemFiles{Backend: config.StorageLocal}
	switch st := s.Blobs.(type) {
	case blobstore.Local:
		v.Dir = st.Dir
	case *blobstore.S3:
		c := s.Cfg.Storage.S3
		v.Backend, v.Bucket, v.Region, v.Prefix, v.Endpoint = config.StorageS3, st.Bucket(), c.Region, st.Prefix(), safeEndpoint(c.Endpoint)
	}
	where := v.Backend + "|" + v.Dir + "|" + v.Endpoint + "|" + v.Bucket + "|" + v.Prefix
	filesCache.Lock()
	defer filesCache.Unlock()
	now := s.now()
	if filesCache.where == where && now.Sub(filesCache.at) < filesTTL {
		return filesCache.v
	}
	v.CountedAt = now
	if v.Dir != "" {
		if _, err := os.Stat(v.Dir); errors.Is(err, os.ErrNotExist) {
			v.Missing = true
		}
	}
	if !v.Missing && s.Blobs != nil {
		cctx, cancel := context.WithTimeout(ctx, filesBudget)
		err := s.Blobs.List(cctx, "", func(_ string, info blobstore.Info) error {
			v.Files++
			v.Bytes += info.Size
			if v.Files >= filesEntries {
				return errEnough
			}
			return nil
		})
		cancel()
		switch {
		case errors.Is(err, errEnough), errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
			v.Partial = true
		case err != nil:
			v.Error = err.Error()
		}
	}
	filesCache.where, filesCache.at, filesCache.v = where, now, v
	return v
}

// safeEndpoint is an endpoint URL without what it may carry beyond the
// address (a user and password, a query).
func safeEndpoint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + u.EscapedPath()
}

// jobError is one entry of River's errors column.
type jobError struct {
	At    time.Time `json:"at"`
	Error string    `json:"error"`
}

// FailedJobs lists the failed jobs, newest first: state is retrying,
// given_up or empty for both.
func (s *Service) FailedJobs(ctx context.Context, state, cursor string) (SystemJobList, error) {
	states := []string{"retryable", "discarded"}
	switch state {
	case "retrying":
		states = []string{"retryable"}
	case "given_up":
		states = []string{"discarded"}
	}
	before := int64(1<<63 - 1)
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil || n <= 0 {
			return SystemJobList{}, api.Invalid("query.cursor", say(ctx, "system.cursor", nil))
		}
		before = n
	}
	const page = 50
	rows, err := s.Pool.Query(ctx, `
		SELECT j.id, j.kind, j.queue, j.state::text, j.attempt, j.max_attempts, j.errors, j.scheduled_at,
		       w.id::text, w.name, w.slug
		FROM river_job j
		LEFT JOIN workspaces w ON w.id = CASE WHEN j.args->>'workspaceId' ~ '^[0-9a-f-]{36}$' THEN (j.args->>'workspaceId')::uuid END
		WHERE j.state::text = ANY($1) AND j.id < $2
		ORDER BY j.id DESC LIMIT $3`, states, before, page+1)
	if err != nil {
		return SystemJobList{}, err
	}
	defer rows.Close()
	out := SystemJobList{Items: []SystemJob{}}
	for rows.Next() {
		var j SystemJob
		var errs [][]byte
		var scheduled time.Time
		var wsID, wsName, wsSlug *string
		if err := rows.Scan(&j.ID, &j.Kind, &j.Queue, &j.State, &j.Attempt, &j.MaxAttempts, &errs, &scheduled, &wsID, &wsName, &wsSlug); err != nil {
			return SystemJobList{}, err
		}
		if j.State == "retryable" {
			j.State = "retrying"
			j.NextAt = &scheduled
		} else {
			j.State = "given_up"
		}
		if n := len(errs); n > 0 {
			var e jobError
			if json.Unmarshal(errs[n-1], &e) == nil {
				at := e.At
				j.FailedAt = &at
				j.Error = cut(e.Error, 500)
			}
		}
		if wsID != nil {
			j.Workspace = &SystemWorkspace{ID: *wsID, Name: deref(wsName), Slug: deref(wsSlug)}
		}
		out.Items = append(out.Items, j)
	}
	if err := rows.Err(); err != nil {
		return SystemJobList{}, err
	}
	if len(out.Items) > page {
		out.Items = out.Items[:page]
		out.NextCursor = strconv.FormatInt(out.Items[page-1].ID, 10)
	}
	return out, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// failedJob loads a failed job for an action, or says it isn't one.
func (s *Service) failedJob(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	if s.Jobs == nil || s.Jobs() == nil {
		return nil, api.Err(http.StatusServiceUnavailable, "unavailable", say(ctx, "system.noJobs", nil))
	}
	j, err := s.Jobs().JobGet(ctx, id)
	if errors.Is(err, river.ErrNotFound) {
		return nil, api.Err(http.StatusNotFound, "not_found", say(ctx, "system.jobGone", nil))
	}
	if err != nil {
		return nil, err
	}
	if j.State != rivertype.JobStateRetryable && j.State != rivertype.JobStateDiscarded {
		return nil, api.Err(http.StatusConflict, "not_failed", say(ctx, "system.notFailed", i18n.Args{"kind": j.Kind}))
	}
	return j, nil
}

// RetryJob runs a failed job again now. Audited.
func (s *Service) RetryJob(ctx context.Context, id int64) (SystemJobResult, error) {
	j, err := s.failedJob(ctx, id)
	if err != nil {
		return SystemJobResult{}, err
	}
	if _, err := s.Jobs().JobRetry(ctx, id); err != nil {
		return SystemJobResult{}, err
	}
	return SystemJobResult{ID: id, State: "available"}, s.auditJob(ctx, "instance.job_retried", j, "Retried")
}

// DiscardJob stops a failed job: a retrying job is cancelled (it stays in
// the list of given-up jobs for 7 days), a given-up one is deleted.
// Audited.
func (s *Service) DiscardJob(ctx context.Context, id int64) (SystemJobResult, error) {
	j, err := s.failedJob(ctx, id)
	if err != nil {
		return SystemJobResult{}, err
	}
	if j.State == rivertype.JobStateRetryable {
		if _, err := s.Jobs().JobCancel(ctx, id); err != nil {
			return SystemJobResult{}, err
		}
		return SystemJobResult{ID: id, State: "cancelled"}, s.auditJob(ctx, "instance.job_cancelled", j, "Cancelled")
	}
	if _, err := s.Jobs().JobDelete(ctx, id); err != nil {
		return SystemJobResult{}, err
	}
	return SystemJobResult{ID: id, State: "deleted"}, s.auditJob(ctx, "instance.job_deleted", j, "Deleted")
}

func (s *Service) auditJob(ctx context.Context, action string, j *rivertype.JobRow, verb string) error {
	last := ""
	if n := len(j.Errors); n > 0 {
		last = cut(j.Errors[n-1].Error, 300)
	}
	return s.Audit.Log(ctx, audit.Event{Action: action, TargetType: "job", TargetID: strconv.FormatInt(j.ID, 10), TargetName: j.Kind,
		Before: map[string]any{"state": string(j.State), "attempt": j.Attempt},
		Metadata: map[string]any{"kind": j.Kind, "queue": j.Queue, "error": last,
			"summary": verb + " job " + j.Kind + " #" + strconv.FormatInt(j.ID, 10)}})
}

// SystemJobPath names a job.
type SystemJobPath struct {
	JobID int64 `path:"jobId" minimum:"1"`
}

// registerSystem adds the System tab's operations.
func registerSystem(h huma.API, s *Service) {
	tags := []string{"Instance admin · System"}
	admin := api.InstanceAdmin
	api.Get(h, api.Op{ID: "get-instance-system", Path: "/admin/system", Summary: "Background jobs, queues, the database's size and largest tables, and the file storage",
		Tags: tags, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[SystemView], error) {
			out, err := s.System(ctx)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-instance-failed-jobs", Path: "/admin/system/jobs", Summary: "Failed background jobs, newest first: retrying, and given up (kept 7 days)",
		Tags: tags, Access: admin},
		func(ctx context.Context, in *struct {
			State  string `query:"state" enum:"retrying,given_up"`
			Cursor string `query:"cursor" maxLength:"20"`
		}) (*api.Body[SystemJobList], error) {
			out, err := s.FailedJobs(ctx, in.State, in.Cursor)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "retry-instance-job", Path: "/admin/system/jobs/{jobId}/retry", Summary: "Run a failed job again now",
		Description: "Audited as instance.job_retried.", Tags: tags, Access: admin},
		func(ctx context.Context, in *SystemJobPath) (*api.Body[SystemJobResult], error) {
			out, err := s.RetryJob(ctx, in.JobID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "discard-instance-job", Path: "/admin/system/jobs/{jobId}/discard", Summary: "Cancel a retrying job, or delete a job that gave up",
		Description: "Audited as instance.job_cancelled or instance.job_deleted.", Tags: tags, Access: admin},
		func(ctx context.Context, in *SystemJobPath) (*api.Body[SystemJobResult], error) {
			out, err := s.DiscardJob(ctx, in.JobID)
			return api.OK(out), err
		})
}
