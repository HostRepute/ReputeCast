package audience

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/csvexport"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Audience actions too long for a request run in the background (review
// 2026-11 AU-4): an export of everything a filter matches, and a bulk
// "Resend opt-in". The job walks the contacts by id in batches, so a
// million rows take constant memory, records how far it got, and shows in
// the jobs tray. An export's file is kept for exportDays, in the file
// storage (the files folder or a bucket: object storage can't append, so
// each batch is a part object of its own, joined into the file at the end).

// Job kinds.
const (
	JobExport      = "export"
	JobResendOptIn = "resend_optin"
)

const (
	jobBatch   = 2000
	exportDays = 7
)

// AudienceJob is a background job as the API reports it.
type AudienceJob struct {
	ID     string `json:"id"`
	Kind   string `json:"kind" enum:"export,resend_optin"`
	Status string `json:"status" enum:"queued,running,done,failed"`
	// Total is how many contacts it covers; Done how many it went through.
	Total  int64             `json:"total"`
	Done   int64             `json:"done"`
	Result AudienceJobResult `json:"result"`
	Error  string            `json:"error,omitempty"`
	// DownloadURL fetches a finished export (its starter only) until
	// ExpiresAt.
	DownloadURL string     `json:"downloadUrl,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

// AudienceJobResult is what a job did: an export's rows; a resend's sent
// emails and the contacts it skipped, by reason.
type AudienceJobResult struct {
	Rows       int64 `json:"rows,omitempty"`
	Sent       int64 `json:"sent,omitempty"`
	Confirmed  int64 `json:"confirmed,omitempty" doc:"Skipped: confirmed already"`
	Suppressed int64 `json:"suppressed,omitempty" doc:"Skipped: on the suppression list"`
	Cooldown   int64 `json:"cooldown,omitempty" doc:"Skipped: got one in the last 10 minutes, or five today"`
	Failed     int64 `json:"failed,omitempty" doc:"The System provider refused the email"`
}

// jobActor is who started a job, as the job acts: their list scope and
// whether they may see paid status, so it reaches what they could.
type jobActor struct {
	Type    authz.ActorType `json:"type"`
	ID      uuid.UUID       `json:"id"`
	UserID  uuid.UUID       `json:"userId"`
	Name    string          `json:"name"`
	Scope   authz.Scope     `json:"scope"`
	SeePaid bool            `json:"seePaid"`
}

func (a jobActor) principal(ws uuid.UUID) *authz.Principal {
	p := &authz.Principal{Type: a.Type, ID: a.ID, UserID: a.UserID, Name: a.Name, WorkspaceID: ws, Scope: a.Scope, Permissions: authz.Set{authz.SubscribersView: {}}}
	if a.SeePaid {
		p.Permissions[authz.EarnView] = struct{}{}
	}
	return p
}

// StartJob queues a background export or resend for a selection.
func (s *Service) StartJob(ctx context.Context, ws *store.Workspace, kind string, sel Selection) (AudienceJob, error) {
	fs, err := s.selectionSQL(ctx, ws, sel)
	if err != nil {
		return AudienceJob{}, err
	}
	if kind == JobResendOptIn && (s.Mail == nil || !s.Mail.SystemAvailable(ctx, &ws.ID)) {
		return AudienceJob{}, api.Err(409, "no_system_provider", say(ctx, mNoSystemProvider, nil))
	}
	var total int64
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM contacts c WHERE "+fs.where, fs.args...).Scan(&total); err != nil {
		return AudienceJob{}, err
	}
	a := jobActor{SeePaid: canSeePaid(ctx)}
	if p := api.PrincipalFrom(ctx); p != nil {
		a.Type, a.ID, a.UserID, a.Name, a.Scope = p.Type, p.ID, p.UserID, p.Name, p.Scope
	}
	selJSON, _ := json.Marshal(sel)
	actorJSON, _ := json.Marshal(a)
	var row store.AudienceJob
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		row, err = q.AudienceJobInsert(ctx, store.AudienceJobInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Kind: kind, Selection: selJSON,
			Actor: actorJSON, CreatedBy: actorID(ctx), Total: total, At: s.now()})
		if err != nil {
			return err
		}
		if s.Jobs != nil && s.Jobs() != nil {
			if _, err := s.Jobs().InsertTx(ctx, tx, AudienceJobArgs{ID: row.ID}, nil); err != nil {
				return err
			}
		}
		action := map[string]string{JobExport: "contacts.export_started", JobResendOptIn: "contacts.resend_optin_started"}[kind]
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "audience_job", TargetID: row.ID.String(),
			Metadata: map[string]any{"contacts": total, "selection": selectionMeta(sel)}})
	})
	if err != nil {
		return AudienceJob{}, err
	}
	return s.jobView(ws, row), nil
}

// selectionSQL is a selection's contacts as a filter: everything matching,
// or the ids picked (within the caller's list scope either way).
func (s *Service) selectionSQL(ctx context.Context, ws *store.Workspace, sel Selection) (filterSQL, error) {
	if sel.AllMatching {
		return s.filterSQL(ctx, ws.ID, sel.Filter)
	}
	if len(sel.IDs) == 0 {
		return filterSQL{}, api.Invalid("body.ids", say(ctx, mSelectContacts, nil))
	}
	ids := make([]uuid.UUID, 0, len(sel.IDs))
	for _, raw := range sel.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return filterSQL{}, api.Invalid("body.ids", say(ctx, mBadIDs, nil))
		}
		ids = append(ids, id)
	}
	fs, err := s.filterSQL(ctx, ws.ID, ContactFilter{})
	if err != nil {
		return fs, err
	}
	fs.add("c.id = ANY (?)", ids)
	return fs, nil
}

// Job reads a job of the workspace.
func (s *Service) Job(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AudienceJob, error) {
	row, err := s.Q.AudienceJobGet(ctx, store.AudienceJobGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AudienceJob{}, api.MapError(err, "Job")
	}
	return s.jobView(ws, row), nil
}

func (s *Service) jobView(ws *store.Workspace, r store.AudienceJob) AudienceJob {
	out := AudienceJob{ID: r.ID.String(), Kind: r.Kind, Status: r.Status, Total: r.Total, Done: r.Done, Error: r.Error, CreatedAt: r.CreatedAt,
		FinishedAt: r.FinishedAt, ExpiresAt: r.ExpiresAt}
	_ = json.Unmarshal(r.Result, &out.Result)
	if r.Kind == JobExport && r.Status == "done" && r.FileKey != "" {
		out.DownloadURL = fmt.Sprintf("/api/v1/workspaces/%s/contacts/jobs/%s/download", ws.Slug, r.ID)
	}
	return out
}

// OpenExport opens a finished export's file for its starter, and logs the
// download.
func (s *Service) OpenExport(ctx context.Context, ws *store.Workspace, id uuid.UUID) (io.ReadCloser, int64, error) {
	row, err := s.Q.AudienceJobGet(ctx, store.AudienceJobGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return nil, 0, api.MapError(err, "Export")
	}
	if me := actorID(ctx); row.Kind != JobExport || row.FileKey == "" || me == nil || row.CreatedBy == nil || *me != *row.CreatedBy {
		return nil, 0, api.NotFound("Export")
	}
	info, err := s.Blobs.Stat(ctx, row.FileKey)
	if err != nil {
		return nil, 0, api.NotFound("Export")
	}
	f, err := s.Blobs.Open(ctx, row.FileKey)
	if err != nil {
		return nil, 0, api.NotFound("Export")
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "contacts.export_downloaded", TargetType: "audience_job", TargetID: id.String(),
		Metadata: map[string]any{"rows": row.Done}}); err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, info.Size, nil
}

// exportParts is where an export's batches are kept until they are
// joined: <file>.parts/<contacts before the batch, 12 digits>.
func exportParts(key string) string { return key + ".parts/" }

// AudienceJobArgs runs one audience job.
type AudienceJobArgs struct {
	ID uuid.UUID `json:"id"`
}

// Kind implements river.JobArgs.
func (AudienceJobArgs) Kind() string { return "audience.job" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (AudienceJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 5}
}

// AudienceJobWorker runs audience jobs.
type AudienceJobWorker struct {
	river.WorkerDefaults[AudienceJobArgs]
	Svc *Service
}

// Timeout lets a million-contact job finish.
func (w *AudienceJobWorker) Timeout(*river.Job[AudienceJobArgs]) time.Duration { return 6 * time.Hour }

func (w *AudienceJobWorker) Work(ctx context.Context, job *river.Job[AudienceJobArgs]) error {
	return w.Svc.RunJob(ctx, job.Args.ID)
}

// RunJob runs a job from where it stopped: in batches of jobBatch contacts
// by id, recording its progress after each.
func (s *Service) RunJob(ctx context.Context, id uuid.UUID) error {
	row, err := s.Q.AudienceJobForWork(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.Status == "done" || row.Status == "failed" {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, row.WorkspaceID)
	if err != nil {
		return err
	}
	var a jobActor
	_ = json.Unmarshal(row.Actor, &a)
	var sel Selection
	_ = json.Unmarshal(row.Selection, &sel)
	ctx = api.WithPrincipal(ctx, a.principal(ws.ID))
	ctx = i18n.WithLang(ctx, i18n.Default)
	if err := s.Q.AudienceJobStarted(ctx, store.AudienceJobStartedParams{ID: id, At: s.now()}); err != nil {
		return err
	}
	var res AudienceJobResult
	_ = json.Unmarshal(row.Result, &res)
	run := s.runResend
	if row.Kind == JobExport {
		run = s.runExport
	}
	done, err := run(ctx, &ws, row, sel, &res)
	if err != nil && ctx.Err() == nil {
		b, _ := json.Marshal(res)
		_ = s.Q.AudienceJobFinished(ctx, store.AudienceJobFinishedParams{ID: id, Status: "failed", Done: done, Result: b, Error: clipErr(err), At: s.now()})
		if row.Kind == JobExport {
			// Nobody downloads a failed export: its rows don't stay.
			s.dropExport(ctx, exportKey(ws.ID, row.ID), true)
		}
		return nil
	}
	return err
}

func clipErr(err error) string {
	msg := err.Error()
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return msg
}

// batches walks the selection's contact ids after row.AfterID, jobBatch at
// a time, calling fn with each batch; fn's error stops the walk.
func (s *Service) batches(ctx context.Context, ws *store.Workspace, row store.AudienceJob, sel Selection, fn func(ids []uuid.UUID) error) error {
	fs, err := s.selectionSQL(ctx, ws, sel)
	if err != nil {
		return err
	}
	after := row.AfterID
	for {
		args := fs.args
		where := fs.where
		if after != nil {
			args = append(args[:len(args):len(args)], *after)
			where += fmt.Sprintf(" AND c.id > $%d", len(args))
		}
		rows, err := s.Pool.Query(ctx, fmt.Sprintf("SELECT c.id FROM contacts c WHERE %s ORDER BY c.id LIMIT %d", where, jobBatch), args...)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := fn(ids); err != nil {
			return err
		}
		last := ids[len(ids)-1]
		after = &last
		if len(ids) < jobBatch {
			return nil
		}
	}
}

// runExport writes the contacts as the CSV export does, a batch at a time.
// Each batch is a part object named by the contacts done before it, so a
// batch run again after a failure replaces its part rather than adding
// its rows twice; once the walk ends, the parts are joined in order into
// the export's file.
func (s *Service) runExport(ctx context.Context, ws *store.Workspace, row store.AudienceJob, sel Selection, res *AudienceJobResult) (int64, error) {
	key := exportKey(ws.ID, row.ID)
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return row.Done, err
	}
	done := row.Done
	var buf bytes.Buffer
	err = s.batches(ctx, ws, row, sel, func(ids []uuid.UUID) error {
		buf.Reset()
		w := csvexport.NewWriter(&buf)
		if done == 0 {
			_ = w.Write(exportHeader(fields))
		}
		n, err := s.writeExportRows(ctx, w, fields, ids)
		if err != nil {
			return err
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return err
		}
		if err := s.Blobs.Put(ctx, exportParts(key)+partName(done), bytes.NewReader(buf.Bytes()), int64(buf.Len()), "text/csv; charset=utf-8"); err != nil {
			return err
		}
		done += int64(len(ids))
		res.Rows += int64(n)
		b, _ := json.Marshal(res)
		last := ids[len(ids)-1]
		return s.Q.AudienceJobProgress(ctx, store.AudienceJobProgressParams{ID: row.ID, Done: done, AfterID: &last, Result: b, FileKey: key})
	})
	if err != nil {
		return done, err
	}
	if err := s.joinExport(ctx, key, done); err != nil {
		return done, err
	}
	now := s.now()
	expires := now.AddDate(0, 0, exportDays)
	b, _ := json.Marshal(res)
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AudienceJobFinished(ctx, store.AudienceJobFinishedParams{ID: row.ID, Status: "done", Done: done, Result: b, At: now, ExpiresAt: &expires}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contacts.export", TargetType: "contact", Actor: jobAuditActor(row),
			Metadata: map[string]any{"rows": res.Rows, "selection": selectionMeta(sel), "job": row.ID.String()}})
	})
	if err != nil {
		return done, err
	}
	// Joined: the parts go (the sweep takes any left), and an export of
	// nothing keeps no file.
	s.dropExport(ctx, key, done == 0)
	return done, nil
}

// exportKey is an export's file.
func exportKey(ws, job uuid.UUID) string { return fmt.Sprintf("exports/%s/%s.csv", ws, job) }

// partName names the part of the batch that begins after done contacts.
func partName(done int64) string { return fmt.Sprintf("%012d", done) }

// joinExport writes an export's file from its parts, in order: those of
// the done contacts (a part written past them, by a batch whose progress
// wasn't recorded, is left out). An export of nothing has no file.
func (s *Service) joinExport(ctx context.Context, key string, done int64) error {
	if done == 0 {
		return nil
	}
	var parts []blobstore.Part
	prefix := exportParts(key)
	if err := s.Blobs.List(ctx, prefix, func(k string, info blobstore.Info) error {
		if from, err := strconv.ParseInt(strings.TrimPrefix(k, prefix), 10, 64); err == nil && from < done {
			parts = append(parts, blobstore.Part{Key: k, Size: info.Size})
		}
		return nil
	}); err != nil {
		return err
	}
	// Fixed-width names sort as their numbers.
	slices.SortFunc(parts, func(a, b blobstore.Part) int { return cmp.Compare(a.Key, b.Key) })
	if first := prefix + partName(0); len(parts) == 0 || parts[0].Key != first {
		// Begun by a release that appended to the file: what that file
		// holds is the export's start, its first part from now on.
		info, err := s.Blobs.Stat(ctx, key)
		if err != nil {
			return fmt.Errorf("audience: the export's first rows are missing: %w", err)
		}
		start := blobstore.Join(ctx, s.Blobs, []blobstore.Part{{Key: key, Size: info.Size}})
		err = s.Blobs.Put(ctx, first, start, info.Size, "text/csv; charset=utf-8")
		_ = start.Close()
		if err != nil {
			return err
		}
		parts = append([]blobstore.Part{{Key: first, Size: info.Size}}, parts...)
	}
	var size int64
	for _, p := range parts {
		size += p.Size
	}
	f := blobstore.Join(ctx, s.Blobs, parts)
	defer func() { _ = f.Close() }()
	return s.Blobs.Put(ctx, key, f, size, "text/csv; charset=utf-8")
}

// dropExport deletes an export's parts, and its file too when asked, even
// once the job's context has ended.
func (s *Service) dropExport(ctx context.Context, key string, file bool) {
	ctx = context.WithoutCancel(ctx)
	var keys []string
	_ = s.Blobs.List(ctx, exportParts(key), func(k string, _ blobstore.Info) error {
		keys = append(keys, k)
		return nil
	})
	if file {
		keys = append(keys, key)
	}
	for _, k := range keys {
		_ = s.Blobs.Delete(ctx, k)
	}
}

// jobAuditActor is the job's starter in the audit log.
func jobAuditActor(row store.AudienceJob) *audit.Actor {
	var a jobActor
	_ = json.Unmarshal(row.Actor, &a)
	if a.ID == uuid.Nil {
		return &audit.Actor{Type: authz.ActorSystem, Name: "System"}
	}
	id := a.ID
	return &audit.Actor{Type: a.Type, ID: &id, Name: a.Name}
}

// exportHeader is the export's first row: built-in columns, then each
// custom field's key.
func exportHeader(fields []store.CustomField) []string {
	header := []string{"email", "first_name", "last_name", "company", "country", "timezone", "language", "lists", "tier", "consent",
		"mailbox_provider", "source", "joined_at", "last_open_at", "last_click_at", "tags"}
	for _, f := range fields {
		header = append(header, f.Key)
	}
	return header
}

// writeExportRows writes the contacts ids as CSV rows, in id order.
func (s *Service) writeExportRows(ctx context.Context, w *csvexport.Writer, fields []store.CustomField, ids []uuid.UUID) (int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT c.email, c.first_name, c.last_name, c.company, c.country, c.timezone, c.language, c.tier, c.consent,
		c.mailbox_provider, c.source, c.created_at, c.last_open_at, c.last_click_at, c.tags, c.attributes,
		coalesce((SELECT string_agg(l.name, '; ' ORDER BY l.created_at) FROM list_memberships lm JOIN lists l ON l.id = lm.list_id
		          WHERE lm.contact_id = c.id AND lm.status = 'subscribed' AND l.deleted_at IS NULL), '')
		FROM contacts c WHERE c.id = ANY ($1) ORDER BY c.id`, ids)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var email, first, last, company, country, tz, lang, tier, consent, mbp, source, lists string
		var joined time.Time
		var lastOpen, lastClick *time.Time
		var tags []string
		var attrs json.RawMessage
		if err := rows.Scan(&email, &first, &last, &company, &country, &tz, &lang, &tier, &consent, &mbp, &source, &joined, &lastOpen, &lastClick, &tags, &attrs, &lists); err != nil {
			return n, err
		}
		var a map[string]any
		_ = json.Unmarshal(attrs, &a)
		rec := []string{email, first, last, company, country, tz, lang, lists, tier, consent, mbp, segments.SourceLabel(source),
			joined.UTC().Format(time.RFC3339), timeOrEmpty(lastOpen), timeOrEmpty(lastClick), strings.Join(tags, "; ")}
		for _, f := range fields {
			rec = append(rec, attrString(a[f.Key]))
		}
		if err := w.Write(rec); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

// runResend sends the opt-in email again to the selection's contacts still
// waiting for one: confirmed and suppressed contacts are skipped, and the
// per-address cooldown (DP-3) holds.
func (s *Service) runResend(ctx context.Context, ws *store.Workspace, row store.AudienceJob, sel Selection, res *AudienceJobResult) (int64, error) {
	note := kept(mNoteConfirmSent, nil)
	actor := jobAuditActor(row)
	var actorUser *uuid.UUID
	if actor.ID != nil && actor.Type == authz.ActorUser {
		actorUser = actor.ID
	}
	done := row.Done
	err := s.batches(ctx, ws, row, sel, func(ids []uuid.UUID) error {
		rows, err := s.Pool.Query(ctx, `SELECT c.*, EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email) AS suppressed
			FROM contacts c WHERE c.id = ANY ($1) ORDER BY c.id`, ids)
		if err != nil {
			return err
		}
		type contactRow struct {
			store.Contact
			Suppressed bool
		}
		list, err := pgx.CollectRows(rows, pgx.RowToStructByName[contactRow])
		if err != nil {
			return err
		}
		for _, c := range list {
			switch {
			case c.Suppressed:
				res.Suppressed++
				continue
			case c.Consent != "pending":
				res.Confirmed++
				continue
			}
			sent, _, err := s.sendConfirmation(ctx, ws, c.Contact)
			switch {
			case errors.Is(err, ErrOptInCooldown):
				res.Cooldown++
			case err != nil:
				return err
			case sent:
				res.Sent++
				_ = s.Q.InsertConsentEvent(ctx, store.InsertConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: c.ID, Action: "optin_sent", Source: "team",
					Note: note.English, Texts: KeptTexts(map[string]i18n.Text{"note": note.Text}), ActorID: actorUser, OccurredAt: s.now()})
			default:
				res.Failed++
			}
		}
		done += int64(len(ids))
		b, _ := json.Marshal(res)
		last := ids[len(ids)-1]
		return s.Q.AudienceJobProgress(ctx, store.AudienceJobProgressParams{ID: row.ID, Done: done, AfterID: &last, Result: b})
	})
	if err != nil {
		return done, err
	}
	b, _ := json.Marshal(res)
	return done, pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AudienceJobFinished(ctx, store.AudienceJobFinishedParams{ID: row.ID, Status: "done", Done: done, Result: b, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contacts.resend_optin", TargetType: "contact", Actor: actor,
			Metadata: map[string]any{"matched": done, "sent": res.Sent, "confirmed": res.Confirmed, "suppressed": res.Suppressed, "cooldown": res.Cooldown,
				"failed": res.Failed, "job": row.ID.String()}})
	})
}

// SweepJobFiles removes export files past their date, with any parts a
// finished export left.
func (s *Service) SweepJobFiles(ctx context.Context) error {
	rows, err := s.Q.AudienceJobsExpired(ctx, s.now())
	if err != nil {
		return err
	}
	for _, r := range rows {
		var keys []string
		if err := s.Blobs.List(ctx, exportParts(r.FileKey), func(k string, _ blobstore.Info) error {
			keys = append(keys, k)
			return nil
		}); err != nil {
			return err
		}
		for _, k := range append(keys, r.FileKey) {
			if err := s.Blobs.Delete(ctx, k); err != nil {
				return err
			}
		}
		if err := s.Q.AudienceJobFileGone(ctx, r.ID); err != nil {
			return err
		}
	}
	return nil
}

// TrayJob is a job as the jobs tray shows it, in the reader's language.
type TrayJob struct {
	ID, Kind, Title, Status, Detail, Href, DownloadURL string
	Progress                                           *int
	StartedAt, FinishedAt                              *time.Time
}

// TrayJobs lists the workspace's running jobs and those that ended in the
// last two hours, for the jobs tray. A finished export offers its
// download to its starter.
func (s *Service) TrayJobs(ctx context.Context, ws *store.Workspace) ([]TrayJob, error) {
	rows, err := s.Q.AudienceJobsRecent(ctx, store.AudienceJobsRecentParams{WorkspaceID: ws.ID, EndedAfter: s.now().Add(-2 * time.Hour)})
	if err != nil {
		return nil, err
	}
	l := i18n.LangFrom(ctx)
	me := actorID(ctx)
	out := make([]TrayJob, 0, len(rows))
	for _, r := range rows {
		v := s.jobView(ws, r)
		j := TrayJob{ID: v.ID, Kind: map[string]string{JobExport: "export", JobResendOptIn: "optin"}[r.Kind], StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
			Href: fmt.Sprintf("/w/%s/audience", ws.Slug)}
		args := i18n.Args{"count": r.Total}
		title := map[string]i18n.Msg{JobExport: mJobExportTitle, JobResendOptIn: mJobResendTitle}[r.Kind]
		j.Title = title.Text(l, args)
		switch r.Status {
		case "queued", "running":
			j.Status = r.Status
			if r.Total > 0 {
				pct := min(99, int(r.Done*100/r.Total))
				j.Progress = &pct
			}
		case "failed":
			j.Status, j.Detail = "failed", r.Error
		case "done":
			j.Status = "done"
			if r.Kind == JobExport {
				j.Detail = mJobExportDone.Text(l, i18n.Args{"count": v.Result.Rows})
				if me != nil && r.CreatedBy != nil && *me == *r.CreatedBy {
					j.DownloadURL = v.DownloadURL
				}
			} else {
				j.Detail = resendSummary(l, v.Result)
			}
		}
		out = append(out, j)
	}
	return out, nil
}

// resendSummary is a finished resend in words: "Sent to 3,904 · 41,118
// confirmed already · 2,710 suppressed · 481 got one in the last 10
// minutes".
func resendSummary(l i18n.Lang, r AudienceJobResult) string {
	parts := []string{mJobResendSent.Text(l, i18n.Args{"count": r.Sent})}
	for _, p := range []struct {
		n int64
		m i18n.Msg
	}{{r.Confirmed, mJobResendConfirmed}, {r.Suppressed, mJobResendSuppressed}, {r.Cooldown, mJobResendCooldown}, {r.Failed, mJobResendFailed}} {
		if p.n > 0 {
			parts = append(parts, p.m.Text(l, i18n.Args{"count": p.n}))
		}
	}
	return strings.Join(parts, " · ")
}
