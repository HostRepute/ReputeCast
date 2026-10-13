package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// StageArgs reads an import's rows from its source into import_rows.
type StageArgs struct {
	ImportID uuid.UUID `json:"importId"`
}

// Kind implements river.JobArgs.
func (StageArgs) Kind() string { return "imports.stage" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (StageArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// StageWorker runs StageArgs.
type StageWorker struct {
	river.WorkerDefaults[StageArgs]
	Svc *Service
}

// Work implements river.Worker.
func (w *StageWorker) Work(ctx context.Context, job *river.Job[StageArgs]) error {
	return w.Svc.Stage(ctx, job.Args.ImportID)
}

// Timeout allows long API reads.
func (w *StageWorker) Timeout(*river.Job[StageArgs]) time.Duration { return time.Hour }

// RequestStage starts reading rows for the selected lists, unless they are
// already staged. It moves the draft to the field mapping step.
func (s *Service) RequestStage(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	if err := editable(imp); err != nil {
		return ImportDetail{}, err
	}
	opts := parseOptions(imp.Options)
	conn := parseConnection(imp.Connection)
	if len(opts.Lists) == 0 {
		return ImportDetail{}, api.Invalid("body.options.lists", "Choose at least one list to import.")
	}
	for i, l := range opts.Lists {
		if l.Target == "" {
			return ImportDetail{}, api.Invalid(fmt.Sprintf("body.options.lists[%d].target", i), "Pick the list each source list goes into.")
		}
	}
	if isFile(imp.Source) && len(imp.FileData) == 0 {
		return ImportDetail{}, api.Invalid("body", "Upload a file first.")
	}
	if !isFile(imp.Source) && len(imp.CredentialsEnc) == 0 {
		return ImportDetail{}, api.Invalid("body.apiKey", fmt.Sprintf("Connect %s with an API key first.", SourceName(imp.Source)))
	}
	if len(conn.BadRows) > 0 && len(opts.SkipRows) == 0 {
		return ImportDetail{}, api.Err(422, "file_columns", conn.Error.Message)
	}
	want := sourceIDs(opts.Lists)
	if imp.StagedAt != nil && sameLists(conn.StagedLists, want) {
		if imp.Step == "source" {
			if err := s.Q.SetImportStep(ctx, store.SetImportStepParams{ID: imp.ID, Step: "fields"}); err != nil {
				return ImportDetail{}, err
			}
		}
		return s.Get(ctx, ws, id)
	}
	if err := s.Q.SetImportStatus(ctx, store.SetImportStatusParams{ID: imp.ID, Status: "staging"}); err != nil {
		return ImportDetail{}, err
	}
	if err := s.Q.SetImportStep(ctx, store.SetImportStepParams{ID: imp.ID, Step: "fields"}); err != nil {
		return ImportDetail{}, err
	}
	if err := s.enqueue(ctx, StageArgs{ImportID: imp.ID}, nil); err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

// jobs returns the River client, or nil to run inline.
func (s *Service) jobs() *jobs.Client {
	if s.Inline || s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func (s *Service) enqueue(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	c := s.jobs()
	if c == nil {
		switch a := args.(type) {
		case StageArgs:
			return s.Stage(ctx, a.ImportID)
		case RunArgs:
			return s.Run(ctx, a.ImportID)
		}
		return fmt.Errorf("unknown job %T", args)
	}
	_, err := c.Insert(ctx, args, opts)
	return err
}

func sourceIDs(ls []ImportListChoice) []string {
	out := make([]string, 0, len(ls))
	for _, l := range ls {
		out = append(out, l.SourceID)
	}
	return out
}

func sameLists(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]bool{}
	for _, x := range a {
		m[x] = true
	}
	for _, x := range b {
		if !m[x] {
			return false
		}
	}
	return true
}

// retrySetter is implemented by the API sources, so retries show in the live log.
type retrySetter interface {
	setOnRetry(func(int, time.Duration, int))
}

func (m *Mailchimp) setOnRetry(f func(int, time.Duration, int))  { m.c.OnRetry = f }
func (b *Brevo) setOnRetry(f func(int, time.Duration, int))      { b.c.OnRetry = f }
func (m *MailerLite) setOnRetry(f func(int, time.Duration, int)) { m.c.OnRetry = f }
func (l *Listmonk) setOnRetry(f func(int, time.Duration, int))   { l.c.OnRetry = f }
func (k *Keila) setOnRetry(f func(int, time.Duration, int))      { k.c.OnRetry = f }

// Stage reads every row of the selected lists into import_rows.
func (s *Service) Stage(ctx context.Context, id uuid.UUID) error {
	imp, err := s.Q.GetImportByID(ctx, id)
	if err != nil {
		return err
	}
	if imp.Status != "staging" && imp.Status != "paused" {
		return nil
	}
	src, err := s.source(imp)
	if err != nil {
		return s.stageFailed(ctx, imp, err)
	}
	name := SourceName(imp.Source)
	if rs, ok := src.(retrySetter); ok {
		rs.setOnRetry(func(attempt int, wait time.Duration, status int) {
			msg := fmt.Sprintf("%s answered %d while reading contacts; retrying (attempt %d)", name, status, attempt)
			if status == 429 {
				msg = fmt.Sprintf("%s returned 429 Too Many Requests; waiting %s before retrying", name, wait.Round(time.Second))
			}
			_ = s.log(ctx, s.Q, imp.ID, "retry", msg)
		})
	}
	opts := parseOptions(imp.Options)
	lists := sourceIDs(opts.Lists)
	fields, err := src.Fields(ctx, lists)
	if err != nil {
		return s.stageFailed(ctx, imp, err)
	}
	if err := s.Q.DeleteImportRows(ctx, imp.ID); err != nil {
		return err
	}
	var buf [][]any
	var n int32
	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		_, err := s.Pool.CopyFrom(ctx, pgx.Identifier{"import_rows"},
			[]string{"import_id", "row_no", "email", "status", "list_ref", "data", "tags", "engagement"}, pgx.CopyFromRows(buf))
		buf = buf[:0]
		return err
	}
	err = src.Fetch(ctx, lists, func(r Row) error {
		n++
		data, _ := json.Marshal(r.Data)
		eng, _ := json.Marshal(r.Engagement)
		tags := r.Tags
		if tags == nil {
			tags = []string{}
		}
		buf = append(buf, []any{imp.ID, n, strings.TrimSpace(r.Email), r.Status, r.List, data, tags, eng})
		if len(buf) >= 2000 {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return s.stageFailed(ctx, imp, err)
	}
	conn := parseConnection(imp.Connection)
	conn.Fields = fields
	conn.StagedLists = lists
	conn.Error = nil
	if err := s.Q.SetImportStaged(ctx, store.SetImportStagedParams{ID: imp.ID, Connection: mustJSON(conn)}); err != nil {
		return err
	}
	return s.log(ctx, s.Q, imp.ID, "info", fmt.Sprintf("Read %s rows and %d fields from %s", thousands(int(n)), len(fields), name))
}

// stageFailed records why staging stopped. Rate limits pause and retry in
// a minute; other errors return the draft to the person with the reason.
func (s *Service) stageFailed(ctx context.Context, imp store.Import, err error) error {
	ie := toImportError(err)
	conn := parseConnection(imp.Connection)
	conn.Error = ie
	if _, e := s.Pool.Exec(ctx, `UPDATE imports SET connection = $2, updated_at = now() WHERE id = $1`, imp.ID, mustJSON(conn)); e != nil {
		return e
	}
	if se, ok := sourceError(err); ok && se.Code == CodeRateLimited {
		resume := s.now().Add(time.Minute)
		if err := s.Q.SetImportStatus(ctx, store.SetImportStatusParams{ID: imp.ID, Status: "paused", ResumeAt: &resume}); err != nil {
			return err
		}
		_ = s.log(ctx, s.Q, imp.ID, "retry", fmt.Sprintf("%s is rate-limiting us (429); paused, resuming at %s", SourceName(imp.Source), resume.UTC().Format("15:04:05 UTC")))
		if c := s.jobs(); c != nil {
			_, err := c.Insert(ctx, StageArgs{ImportID: imp.ID}, &river.InsertOpts{ScheduledAt: resume})
			return err
		}
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	_ = s.log(ctx, s.Q, imp.ID, "error", ie.Message)
	return s.Q.SetImportStatus(ctx, store.SetImportStatusParams{ID: imp.ID, Status: "draft"})
}

// Resume restarts a paused import now ("Resume now").
func (s *Service) Resume(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	if imp.Status != "paused" {
		return ImportDetail{}, api.Conflict("Only a paused import can be resumed.")
	}
	if err := s.Q.SetImportStatus(ctx, store.SetImportStatusParams{ID: imp.ID, Status: "staging"}); err != nil {
		return ImportDetail{}, err
	}
	if err := s.enqueue(ctx, StageArgs{ImportID: imp.ID}, nil); err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}
