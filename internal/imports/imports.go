// Package imports is the migration importer (spec 5.8, 7.11): CSV and XLSX
// uploads and API importers for Brevo, Mailchimp, MailerLite, listmonk and
// Keila. A River job stages the rows; after field and consent mapping and a
// dry run, a second job imports them in batches. Unsubscribed, bounced and
// complained contacts become suppressions and are never re-subscribed.
package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sealer encrypts and decrypts importer credentials.
type Sealer interface {
	Seal(plaintext []byte, ad string) ([]byte, error)
	Open(ciphertext []byte, ad string) ([]byte, error)
}

type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Box      Sealer
	Audience *audience.Service
	Mail     mailer.Sender
	// Jobs returns the River client (created after the services are wired).
	Jobs func() *jobs.Client
	MX   *MXChecker
	// HTTP is used by the API importers; BaseURLs overrides their API base
	// per source (tests). Guard decides which addresses a self-hosted
	// server (listmonk, Keila) may have: the client refuses the same ones
	// at connect time.
	HTTP     *http.Client
	Guard    netguard.Policy
	BaseURLs map[string]string
	Now      func() time.Time
	// Inline runs staging and imports in the request instead of River (tests).
	Inline bool
	// Templates imports a source's email templates (Notifuse); nil skips
	// the step.
	Templates TemplateImporter
	// Texts records numbers and SMS consent (phase 8); nil imports neither.
	Texts Texts
	// JobSources add other domains' jobs to the background-jobs tray.
	JobSources []JobSource
	// Checks gives a verification service's answers for an import's new
	// addresses (X14); nil skips them.
	Checks AddressChecks
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Sources the importer supports, in the order of the source grid.
var Sources = []string{"brevo", "mailchimp", "mailerlite", "listmonk", "keila", "notifuse", "csv"}

// SourceName is the display name of a source.
func SourceName(src string) string {
	switch src {
	case "brevo":
		return "Brevo"
	case "mailchimp":
		return "Mailchimp"
	case "mailerlite":
		return "MailerLite"
	case "listmonk":
		return "listmonk"
	case "keila":
		return "Keila"
	case "notifuse":
		return "Notifuse"
	case "csv":
		return "CSV"
	case "xlsx":
		return "XLSX"
	}
	return src
}

func isFile(src string) bool { return src == "csv" || src == "xlsx" }

// ImportOptions are the choices made across the wizard.
type ImportOptions struct {
	// Lists are the source lists to import and where each goes: a list id
	// or "new:<name>" to create one.
	Lists            []ImportListChoice `json:"lists"`
	ImportTags       bool               `json:"importTags"`
	ImportEngagement bool               `json:"importEngagement"`
	PullSuppressions bool               `json:"pullSuppressions"`
	RecordHistory    bool               `json:"recordHistory"`
	// Evidence: trust (keep the source's opt-in as evidence) or
	// repermission (import as pending and send a confirmation email).
	Evidence string `json:"evidence" enum:"trust,repermission"`
	// OnConflict for contacts that already exist: fill (empty fields only),
	// overwrite, or keep (ReputeCast values win).
	OnConflict string             `json:"onConflict" enum:"fill,overwrite,keep"`
	Risky      ImportRiskyActions `json:"risky"`
	// SkipRows are file lines left out (rows with the wrong column count).
	SkipRows []int `json:"skipRows"`
	// Pace limits requests to the source, per second (0: as fast as it
	// answers); "Slow down" after a rate limit sets 3.
	Pace int `json:"pace,omitempty" minimum:"0" maximum:"50"`
	// ImportTemplates brings the source's email templates along (Notifuse):
	// transactional ones as drafts, the others as saved templates.
	ImportTemplates bool `json:"importTemplates"`
}

// ImportListChoice maps a source list to a ReputeCast list.
type ImportListChoice struct {
	SourceID string `json:"sourceId"`
	Target   string `json:"target" doc:"List id, or new:<name>"`
}

// ImportRiskyActions say what happens to each risky group.
type ImportRiskyActions struct {
	Role       string `json:"role" enum:"exclude_broadcasts,skip,import"`
	Disposable string `json:"disposable" enum:"skip,import"`
	Typo       string `json:"typo" enum:"fix,skip,import"`
	Stale      string `json:"stale" enum:"pending,skip,import"`
	Trap       string `json:"trap" enum:"skip,import"`
	// A verification service's risky rows (X14); empty imports them, tagged.
	CatchAll string `json:"catch_all,omitempty" enum:"import,skip,"`
	Abuse    string `json:"abuse,omitempty" enum:"import,skip,"`
	Unknown  string `json:"unknown,omitempty" enum:"import,skip,"`
}

func defaultOptions() ImportOptions {
	return ImportOptions{Lists: []ImportListChoice{}, ImportTags: true, ImportEngagement: true, PullSuppressions: true, RecordHistory: true,
		ImportTemplates: true, Evidence: "trust", OnConflict: "fill", SkipRows: []int{},
		Risky: ImportRiskyActions{Role: "exclude_broadcasts", Disposable: "skip", Typo: "fix", Stale: "pending", Trap: "skip"}}
}

func parseOptions(raw json.RawMessage) ImportOptions {
	o := defaultOptions()
	_ = json.Unmarshal(raw, &o)
	if o.Lists == nil {
		o.Lists = []ImportListChoice{}
	}
	if o.SkipRows == nil {
		o.SkipRows = []int{}
	}
	d := defaultOptions()
	if o.Evidence == "" {
		o.Evidence = d.Evidence
	}
	if o.OnConflict == "" {
		o.OnConflict = d.OnConflict
	}
	if o.Risky.Role == "" {
		o.Risky.Role = d.Risky.Role
	}
	if o.Risky.Disposable == "" {
		o.Risky.Disposable = d.Risky.Disposable
	}
	if o.Risky.Typo == "" {
		o.Risky.Typo = d.Risky.Typo
	}
	if o.Risky.Stale == "" {
		o.Risky.Stale = d.Risky.Stale
	}
	if o.Risky.Trap == "" {
		o.Risky.Trap = d.Risky.Trap
	}
	return o
}

// storedConnection is imports.connection.
type storedConnection struct {
	ImportConnection
	Fields []SourceField `json:"fields"`
	// BadRows are file rows with the wrong column count.
	BadRows []ImportBadRow `json:"badRows,omitempty"`
	Header  int            `json:"header,omitempty"`
	// Staged remembers which lists the staged rows came from.
	StagedLists []string `json:"stagedLists,omitempty"`
	// Error is the last failure talking to the source.
	Error *ImportError `json:"error,omitempty"`
}

func parseConnection(raw json.RawMessage) storedConnection {
	var c storedConnection
	_ = json.Unmarshal(raw, &c)
	if c.Lists == nil {
		c.Lists = []ImportSourceList{}
	}
	if c.Fields == nil {
		c.Fields = []SourceField{}
	}
	return c
}

// credentials are sealed with the import id as associated data.
type credentials struct {
	APIKey  string `json:"apiKey"`
	User    string `json:"user,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"`
}

func credentialsAD(id uuid.UUID) string { return "import:" + id.String() + ":credentials" }

// ImportError explains why the source or the file can't be used.
type ImportError struct {
	Code    string         `json:"code" doc:"source_auth_failed, source_rate_limited, source_unavailable, file_columns, file_unreadable"`
	Message string         `json:"message"`
	Status  int            `json:"status,omitempty" doc:"HTTP status from the source (401, 429)"`
	BadRows []ImportBadRow `json:"badRows,omitempty"`
	Header  int            `json:"header,omitempty"`
	RetryAt *time.Time     `json:"retryAt,omitempty"`
	// Row is where a paused import stopped.
	Row int64 `json:"row,omitempty"`
}

// ImportLogEntry is one line of the live log.
type ImportLogEntry struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind" enum:"start,running,done,sync,retry,error,info"`
	Message string    `json:"message"`
}

// ImportProgress is the state of a running import.
type ImportProgress struct {
	Total      int64 `json:"total"`
	Processed  int64 `json:"processed"`
	Batch      int   `json:"batch"`
	Batches    int   `json:"batches"`
	BatchSize  int   `json:"batchSize"`
	Imported   int64 `json:"imported"`
	Merged     int64 `json:"merged"`
	Suppressed int64 `json:"suppressed"`
	Pending    int64 `json:"pending"`
	Skipped    int64 `json:"skipped"`
}

// ImportStats is the summary of a finished import.
type ImportStats struct {
	Rows         int64            `json:"rows"`
	Imported     int64            `json:"imported"`
	Merged       int64            `json:"merged"`
	Suppressed   int64            `json:"suppressed"`
	SuppressedBy map[string]int64 `json:"suppressedBy" doc:"unsubscribe, hard_bounce, complaint"`
	Pending      int64            `json:"pending"`
	Skipped      int64            `json:"skipped"`
	// Reasons counts skipped and changed rows: invalid, disposable, trap,
	// already_suppressed, typo_fixed, role.
	Reasons map[string]int64 `json:"reasons"`
	// Lists are the target lists and their size after the import.
	Lists        []ImportListAfter `json:"lists"`
	DurationMs   int64             `json:"durationMs"`
	PushedTo     []string          `json:"pushedTo" doc:"Providers the new suppressions were pushed to"`
	ConfirmsSent int64             `json:"confirmsSent" doc:"Re-permission emails sent through the System stream"`
	// Templates the import brought along (Notifuse).
	Templates *ImportTemplateStats `json:"templates,omitempty"`
	// Texts counts the rows' numbers and SMS consent (phase 8): number,
	// consent, no_evidence (SMS consent without its date, source and text:
	// Not subscribed) and invalid_phone.
	Texts map[string]int64 `json:"texts,omitempty"`
}

// ImportListAfter is a target list and its size after the import.
type ImportListAfter struct {
	Name     string `json:"name"`
	Contacts int64  `json:"contacts"`
}

// ImportDetail is one import for the wizard.
type ImportDetail struct {
	ID             string            `json:"id"`
	Source         string            `json:"source"`
	SourceName     string            `json:"sourceName"`
	Status         string            `json:"status" enum:"draft,staging,mapping,previewing,running,paused,done,failed,cancelled"`
	Step           string            `json:"step" enum:"source,fields,consent,preview,import,done"`
	FileName       string            `json:"fileName"`
	HasCredentials bool              `json:"hasCredentials" doc:"An API key is stored (it is never returned)"`
	Connection     *ImportConnection `json:"connection,omitempty"`
	Options        ImportOptions     `json:"options"`
	StagedRows     int64             `json:"stagedRows"`
	StagedAt       *time.Time        `json:"stagedAt,omitempty"`
	Progress       ImportProgress    `json:"progress"`
	Stats          *ImportStats      `json:"stats,omitempty"`
	Error          *ImportError      `json:"error,omitempty"`
	Log            []ImportLogEntry  `json:"log"`
	CreatedAt      time.Time         `json:"createdAt"`
	StartedAt      *time.Time        `json:"startedAt,omitempty"`
	FinishedAt     *time.Time        `json:"finishedAt,omitempty"`
	CreatedBy      string            `json:"createdBy"`
	// Fields found in the source (merge fields, attributes, columns).
	FieldCount int `json:"fieldCount"`
	// Previous is the latest finished import, for "Previous imports".
	Previous *ImportSummary `json:"previous,omitempty"`
	// ProviderSync summarizes suppressions already pulled from providers
	// ("SES 612, Postmark 188, Postal 41").
	ProviderSync []ImportProviderPulled `json:"providerSync"`
}

// ImportProviderPulled is a provider and how many suppressions came from it.
type ImportProviderPulled struct {
	Name   string `json:"name"`
	Short  string `json:"short"`
	Pulled int64  `json:"pulled"`
}

// ImportSummary is one import in the history.
type ImportSummary struct {
	ID         string     `json:"id"`
	Source     string     `json:"source"`
	SourceName string     `json:"sourceName"`
	Status     string     `json:"status"`
	Step       string     `json:"step"`
	FileName   string     `json:"fileName"`
	Contacts   int64      `json:"contacts" doc:"Contacts imported, merged or imported as pending"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	CreatedBy  string     `json:"createdBy"`
}

// ImportList is the import history.
type ImportList struct {
	Items []ImportSummary `json:"items"`
	// Active is the draft or running import the wizard resumes.
	Active string `json:"active,omitempty"`
}

// List returns the imports of a workspace, newest first.
func (s *Service) List(ctx context.Context, ws *store.Workspace) (ImportList, error) {
	rows, err := s.Q.ListImports(ctx, ws.ID)
	if err != nil {
		return ImportList{}, err
	}
	out := ImportList{Items: make([]ImportSummary, 0, len(rows))}
	for _, r := range rows {
		var st ImportStats
		_ = json.Unmarshal(r.Stats, &st)
		out.Items = append(out.Items, ImportSummary{ID: r.ID.String(), Source: r.Source, SourceName: SourceName(r.Source), Status: r.Status, Step: r.Step,
			FileName: r.FileName, Contacts: st.Imported + st.Merged + st.Pending, CreatedAt: r.CreatedAt, FinishedAt: r.FinishedAt, CreatedBy: r.CreatedByName})
		if out.Active == "" && (r.Status == "draft" || r.Status == "staging" || r.Status == "running" || r.Status == "paused") {
			out.Active = r.ID.String()
		}
	}
	return out, nil
}

// get loads an import of the workspace.
func (s *Service) get(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.Import, error) {
	imp, err := s.Q.GetImport(ctx, store.GetImportParams{WorkspaceID: ws.ID, ID: id})
	return imp, api.MapError(err, "Import")
}

// Get returns one import with its connection, progress and log.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	return s.detail(ctx, ws, imp)
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, imp store.Import) (ImportDetail, error) {
	conn := parseConnection(imp.Connection)
	out := ImportDetail{ID: imp.ID.String(), Source: imp.Source, SourceName: SourceName(imp.Source), Status: imp.Status, Step: imp.Step,
		FileName: imp.FileName, HasCredentials: len(imp.CredentialsEnc) > 0, Options: parseOptions(imp.Options), StagedAt: imp.StagedAt,
		CreatedAt: imp.CreatedAt, StartedAt: imp.StartedAt, FinishedAt: imp.FinishedAt, Log: []ImportLogEntry{}, ProviderSync: []ImportProviderPulled{},
		FieldCount: len(conn.Fields), Error: conn.Error}
	if conn.Account != "" || len(conn.Lists) > 0 {
		c := conn.ImportConnection
		out.Connection = &c
	}
	_ = json.Unmarshal(imp.Progress, &out.Progress)
	if len(imp.Stats) > 2 {
		var st ImportStats
		if json.Unmarshal(imp.Stats, &st) == nil {
			out.Stats = &st
		}
	}
	if imp.Error != "" && out.Error == nil {
		out.Error = &ImportError{Code: "import_failed", Message: imp.Error}
	}
	if imp.Status == "paused" && out.Error != nil {
		out.Error.RetryAt = imp.ResumeAt
	}
	var err error
	if out.StagedRows, err = s.Q.CountImportRows(ctx, imp.ID); err != nil {
		return out, err
	}
	logs, err := s.Q.ListImportLog(ctx, imp.ID)
	if err != nil {
		return out, err
	}
	for _, l := range logs {
		out.Log = append(out.Log, ImportLogEntry{At: l.OccurredAt, Kind: l.Kind, Message: l.Message})
	}
	if imp.CreatedBy != nil {
		if u, err := s.Q.GetUser(ctx, *imp.CreatedBy); err == nil {
			out.CreatedBy = u.Name
		}
	}
	if prev, err := s.Q.LatestFinishedImport(ctx, ws.ID); err == nil && prev.ID != imp.ID {
		var st ImportStats
		_ = json.Unmarshal(prev.Stats, &st)
		out.Previous = &ImportSummary{ID: prev.ID.String(), Source: prev.Source, SourceName: SourceName(prev.Source), Status: prev.Status,
			Step: prev.Step, FileName: prev.FileName, Contacts: st.Imported + st.Merged + st.Pending, CreatedAt: prev.CreatedAt, FinishedAt: prev.FinishedAt}
	}
	syncs, err := s.Q.SuppressionSyncStates(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, p := range syncs {
		if p.Pulled > 0 {
			out.ProviderSync = append(out.ProviderSync, ImportProviderPulled{Name: p.Name, Short: shortProvider(p.Name, p.Adapter), Pulled: p.Pulled})
		}
	}
	return out, nil
}

func shortProvider(name, adapter string) string {
	if adapter == "ses" {
		return "SES"
	}
	if i := strings.Index(name, " ("); i > 0 {
		return name[:i]
	}
	return name
}

// ImportCreateInput starts an import.
type ImportCreateInput struct {
	Source string `json:"source" enum:"brevo,mailchimp,mailerlite,listmonk,keila,notifuse,csv,xlsx"`
}

// Create makes a draft import.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, in ImportCreateInput) (ImportDetail, error) {
	opts, _ := json.Marshal(defaultOptions())
	var imp store.Import
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		imp, err = q.CreateImport(ctx, store.CreateImportParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Source: in.Source, Options: opts, CreatedBy: actorID(ctx)})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.create", TargetType: "import", TargetID: imp.ID.String(), TargetName: SourceName(in.Source)})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	return s.detail(ctx, ws, imp)
}

// Delete discards a draft import and its staged rows.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		imp, err := q.GetImport(ctx, store.GetImportParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Import")
		}
		n, err := q.DeleteImport(ctx, store.DeleteImportParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Conflict("Only drafts and stopped imports can be discarded. Finished imports stay in the history.")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.discard", TargetType: "import", TargetID: id.String(), TargetName: SourceName(imp.Source)})
	})
}

// ImportUpdateInput changes the draft: its source (before connecting), options
// and the step the person is on.
type ImportUpdateInput struct {
	Source  string         `json:"source,omitempty" enum:"brevo,mailchimp,mailerlite,listmonk,keila,notifuse,csv,xlsx,"`
	Options *ImportOptions `json:"options,omitempty"`
	Step    string         `json:"step,omitempty" enum:"source,fields,consent,preview,"`
}

// Update saves changes to a draft.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ImportUpdateInput) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	if err := editable(imp); err != nil {
		return ImportDetail{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if in.Source != "" && in.Source != imp.Source {
			if _, err := tx.Exec(ctx, `UPDATE imports SET source = $2, connection = '{}', credentials_enc = NULL, file_name = '', file_data = NULL,
				staged_at = NULL, field_map = '{}', step = 'source', updated_at = now() WHERE id = $1`, imp.ID, in.Source); err != nil {
				return err
			}
			if err := q.DeleteImportRows(ctx, imp.ID); err != nil {
				return err
			}
		}
		if in.Options != nil {
			o := *in.Options
			if err := validateOptions(ctx, q, ws, &o); err != nil {
				return err
			}
			raw, _ := json.Marshal(o)
			if err := q.SetImportOptions(ctx, store.SetImportOptionsParams{ID: imp.ID, Options: raw}); err != nil {
				return err
			}
		}
		if in.Step != "" {
			if err := q.SetImportStep(ctx, store.SetImportStepParams{ID: imp.ID, Step: in.Step}); err != nil {
				return err
			}
		}
		meta := map[string]any{}
		name := SourceName(imp.Source)
		if in.Source != "" && in.Source != imp.Source {
			meta["source"] = map[string]string{"from": imp.Source, "to": in.Source}
			name = SourceName(in.Source)
		}
		if in.Options != nil {
			meta["options"] = true
		}
		if in.Step != "" {
			meta["step"] = in.Step
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.update", TargetType: "import", TargetID: imp.ID.String(),
			TargetName: name, Metadata: meta})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

func editable(imp store.Import) error {
	switch imp.Status {
	case "running":
		return api.Conflict("This import is running. Stop it first or wait for the summary.")
	case "done":
		return api.Conflict("This import is finished. Start a new import to bring in more contacts.")
	}
	return nil
}

func validateOptions(ctx context.Context, q *store.Queries, ws *store.Workspace, o *ImportOptions) error {
	seen := map[string]bool{}
	for i, l := range o.Lists {
		if seen[l.SourceID] {
			return api.Invalid(fmt.Sprintf("body.options.lists[%d]", i), "Each source list can only be imported once.")
		}
		seen[l.SourceID] = true
		if name, ok := strings.CutPrefix(l.Target, "new:"); ok {
			if strings.TrimSpace(name) == "" || len(name) > 100 {
				return api.Invalid(fmt.Sprintf("body.options.lists[%d].target", i), "Name the new list.")
			}
			continue
		}
		id, err := uuid.Parse(l.Target)
		if err != nil {
			return api.Invalid(fmt.Sprintf("body.options.lists[%d].target", i), "Pick a list to import into.")
		}
		if _, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return api.Invalid(fmt.Sprintf("body.options.lists[%d].target", i), "That list doesn't exist in this workspace.")
		}
	}
	if o.SkipRows == nil {
		o.SkipRows = []int{}
	}
	return nil
}

func actorID(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	return nil
}

func (s *Service) log(ctx context.Context, q *store.Queries, id uuid.UUID, kind, msg string) error {
	return q.InsertImportLog(ctx, store.InsertImportLogParams{ID: uuid.Must(uuid.NewV7()), ImportID: id, OccurredAt: s.now(), Kind: kind, Message: msg})
}

// sourceError is errors.As for SourceError.
func sourceError(err error) (*SourceError, bool) {
	var se *SourceError
	ok := errors.As(err, &se)
	return se, ok
}
