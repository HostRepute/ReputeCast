package imports

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// BatchSize is how many rows one batch imports.
const BatchSize = 500

// analyzeAfterRows is the import size after which the contact tables are
// analyzed again.
const analyzeAfterRows = 5000

// RunArgs imports the staged rows of an import.
type RunArgs struct {
	ImportID uuid.UUID `json:"importId"`
}

// Kind implements river.JobArgs.
func (RunArgs) Kind() string { return "imports.run" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (RunArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// RunWorker runs RunArgs.
type RunWorker struct {
	river.WorkerDefaults[RunArgs]
	Svc *Service
}

// Work implements river.Worker.
func (w *RunWorker) Work(ctx context.Context, job *river.Job[RunArgs]) error {
	return w.Svc.Run(ctx, job.Args.ImportID)
}

// Timeout allows large imports.
func (w *RunWorker) Timeout(*river.Job[RunArgs]) time.Duration { return 6 * time.Hour }

// Start begins the import of the staged rows.
func (s *Service) Start(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	if imp.Status != "draft" {
		return ImportDetail{}, api.Conflict("This import has already started.")
	}
	if imp.StagedAt == nil {
		return ImportDetail{}, api.Conflict("The rows aren't read from the source yet.")
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return ImportDetail{}, err
	}
	fm := s.effectiveMap(imp, custom)
	if e, _ := fieldSources(fm); e == "" {
		return ImportDetail{}, api.Invalid("body.fields", "Map a field to email before importing.")
	}
	total, err := s.Q.CountImportRows(ctx, imp.ID)
	if err != nil {
		return ImportDetail{}, err
	}
	batches := int((total + BatchSize - 1) / BatchSize)
	opts := parseOptions(imp.Options)
	cm := parseConsentMap(imp.ConsentMap, opts)
	conn := parseConnection(imp.Connection)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.StartImport(ctx, store.StartImportParams{ID: imp.ID, Progress: mustJSON(ImportProgress{Total: total, Batches: batches, BatchSize: BatchSize})}); err != nil {
			return err
		}
		if err := s.log(ctx, q, imp.ID, "start", s.startLine(ctx, ws, imp, conn, opts, cm, total)); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.start", TargetType: "import", TargetID: imp.ID.String(),
			TargetName: SourceName(imp.Source), Metadata: map[string]any{"rows": total}})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	if err := s.enqueue(ctx, RunArgs{ImportID: imp.ID}, nil); err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

// startLine is the first line of the live log: "Import started · 13,739
// rows · audience "Acme Customers" → list Product Updates · consent: trust
// double opt-in".
func (s *Service) startLine(ctx context.Context, ws *store.Workspace, imp store.Import, conn storedConnection, opts ImportOptions, cm ImportConsentMap, total int64) string {
	parts := []string{"Import started", thousands(int(total)) + " rows"}
	noun := map[string]string{"mailchimp": "audience", "mailerlite": "group"}[imp.Source]
	if noun == "" {
		noun = "list"
	}
	for _, l := range opts.Lists {
		target := strings.TrimPrefix(l.Target, "new:")
		if id, err := uuid.Parse(l.Target); err == nil {
			if lst, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id}); err == nil {
				target = lst.Name
			}
		}
		if isFile(imp.Source) {
			parts = append(parts, fmt.Sprintf("%s → list %s", imp.FileName, target))
		} else {
			parts = append(parts, fmt.Sprintf("%s %q → list %s", noun, listName(conn, l.SourceID), target))
		}
	}
	consent := "trust double opt-in"
	if cm.Subscribed == HandlePending {
		consent = "re-permission everyone"
	} else if !anyDoubleOptIn(conn) {
		consent = "keep source opt-in"
	}
	return strings.Join(parts, " · ") + " · consent: " + consent
}

func anyDoubleOptIn(conn storedConnection) bool {
	for _, l := range conn.Lists {
		if l.DoubleOptIn {
			return true
		}
	}
	return false
}

// Cancel stops an import; rows already imported stay.
func (s *Service) Cancel(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	switch imp.Status {
	case "done", "cancelled", "failed":
		return ImportDetail{}, api.Conflict("This import isn't running.")
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetImportStatus(ctx, store.SetImportStatusParams{ID: imp.ID, Status: "cancelled"}); err != nil {
			return err
		}
		if err := s.log(ctx, q, imp.ID, "info", "Import stopped; contacts already imported stay"); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.cancel", TargetType: "import", TargetID: imp.ID.String(), TargetName: SourceName(imp.Source)})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

// runState carries what the import job needs across batches.
type runState struct {
	ws        store.Workspace
	imp       store.Import
	fm        FieldMap
	opts      ImportOptions
	cm        ImportConsentMap
	conn      storedConnection
	targets   map[string]uuid.UUID // source list → ReputeCast list
	policy    audience.SunsetPolicy
	stats     ImportStats
	progress  ImportProgress
	newSupp   []string
	pending   []uuid.UUID
	report    *csv.Writer
	reportBuf bytes.Buffer
	skipped   *csv.Writer
	skipBuf   bytes.Buffer
	actor     *uuid.UUID
	actorName string
	hosting   map[string]string
}

// Run imports every staged row in batches, with a live log.
func (s *Service) Run(ctx context.Context, id uuid.UUID) error {
	imp, err := s.Q.GetImportByID(ctx, id)
	if err != nil {
		return err
	}
	if imp.Status != "running" {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, imp.WorkspaceID)
	if err != nil {
		return err
	}
	st := &runState{ws: ws, imp: imp, opts: parseOptions(imp.Options), conn: parseConnection(imp.Connection), targets: map[string]uuid.UUID{},
		policy: audience.ParsePolicy(ws.SunsetPolicy), actor: imp.CreatedBy, hosting: map[string]string{}}
	st.cm = parseConsentMap(imp.ConsentMap, st.opts)
	st.stats = ImportStats{SuppressedBy: map[string]int64{}, Reasons: map[string]int64{}, Lists: []ImportListAfter{}, PushedTo: []string{}}
	_ = json.Unmarshal(imp.Progress, &st.progress)
	st.report = csv.NewWriter(&st.reportBuf)
	st.skipped = csv.NewWriter(&st.skipBuf)
	_ = st.report.Write([]string{"row", "email", "source_status", "outcome", "reason"})
	_ = st.skipped.Write([]string{"row", "value", "source_status", "reason"})
	// Audit entries and consent history name the person who started the import.
	if imp.CreatedBy != nil {
		if u, err := s.Q.GetUser(ctx, *imp.CreatedBy); err == nil {
			st.actorName = u.Name
			ctx = api.WithPrincipal(ctx, &authz.Principal{Type: authz.ActorUser, ID: u.ID, UserID: u.ID, Name: u.Name, WorkspaceID: ws.ID})
		}
	}
	start := s.now()
	if st.opts.PullSuppressions && s.Audience != nil {
		if res, err := s.Audience.PullFromProviders(ctx, &ws, s.Box); err == nil {
			names := []string{}
			for _, p := range res.Providers {
				if p.Skipped == "" {
					names = append(names, p.Name)
				}
			}
			if len(names) > 0 {
				_ = s.log(ctx, s.Q, imp.ID, "sync", fmt.Sprintf("Pulled suppressions from %s: %d new", joinAnd(names), res.Added))
			}
		}
	}
	if err := s.prepareTargets(ctx, st); err != nil {
		return s.runFailed(ctx, st, err)
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return s.runFailed(ctx, st, err)
	}
	st.fm = s.effectiveMap(imp, custom)
	if err := s.createNewFields(ctx, st); err != nil {
		return s.runFailed(ctx, st, err)
	}
	var after int32
	for batchNo := 1; ; batchNo++ {
		cur, err := s.Q.GetImportByID(ctx, id)
		if err != nil {
			return err
		}
		if cur.Status == "cancelled" {
			return s.finish(ctx, st, "cancelled", start)
		}
		b, err := s.loadBatch(ctx, ws.ID, imp, st.fm, after, BatchSize)
		if err != nil {
			return s.runFailed(ctx, st, err)
		}
		if len(b.rows) == 0 {
			break
		}
		for d, r := range b.mx {
			if r.Hosting != "" {
				st.hosting[d] = r.Hosting
			}
		}
		before := st.progress
		if err := s.importBatch(ctx, st, b); err != nil {
			return s.runFailed(ctx, st, err)
		}
		after = b.rows[len(b.rows)-1].RowNo
		st.progress.Batch = batchNo
		st.progress.Processed += int64(len(b.rows))
		if err := s.Q.SetImportProgress(ctx, store.SetImportProgressParams{ID: imp.ID, Progress: mustJSON(st.progress)}); err != nil {
			return err
		}
		_ = s.log(ctx, s.Q, imp.ID, "done", fmt.Sprintf("Batch %d · %s rows: %s imported, %s merged, %s suppressed, %s skipped", batchNo, thousands(len(b.rows)),
			thousands(int(st.progress.Imported+st.progress.Pending-before.Imported-before.Pending)), thousands(int(st.progress.Merged-before.Merged)),
			thousands(int(st.progress.Suppressed-before.Suppressed)), thousands(int(st.progress.Skipped-before.Skipped))))
	}
	s.pushSuppressions(ctx, st)
	s.sendRepermission(ctx, st)
	s.importTemplates(ctx, st)
	return s.finish(ctx, st, "done", start)
}

// prepareTargets resolves each source list's target, creating new lists.
func (s *Service) prepareTargets(ctx context.Context, st *runState) error {
	for _, l := range st.opts.Lists {
		if name, ok := strings.CutPrefix(l.Target, "new:"); ok {
			created, err := s.Audience.CreateList(ctx, &st.ws, audience.AudienceListInput{Name: strings.TrimSpace(name), Description: "Imported from " + SourceName(st.imp.Source) + "."})
			if err != nil {
				var p *api.Problem
				if errors.As(err, &p) && p.Code == api.CodeConflict {
					// A list with that name exists already: import into it.
					rows, qerr := s.Q.ListListsWithCounts(ctx, st.ws.ID)
					if qerr != nil {
						return qerr
					}
					for _, existing := range rows {
						if strings.EqualFold(existing.Name, strings.TrimSpace(name)) {
							st.targets[l.SourceID] = existing.ID
						}
					}
					continue
				}
				return err
			}
			st.targets[l.SourceID] = uuid.MustParse(created.ID)
			continue
		}
		id, err := uuid.Parse(l.Target)
		if err != nil {
			return fmt.Errorf("list %s has no target", l.SourceID)
		}
		st.targets[l.SourceID] = id
	}
	return nil
}

// createNewFields creates the custom fields the mapping asked for.
func (s *Service) createNewFields(ctx context.Context, st *runState) error {
	for i, f := range st.fm.Fields {
		key, ok := strings.CutPrefix(f.Target, "new:")
		if !ok {
			continue
		}
		typ := f.Type
		if typ == "" {
			typ = "text"
		}
		if _, err := s.Audience.CreateField(ctx, &st.ws, audience.CustomFieldInput{Key: key, Label: key, Type: typ}); err != nil {
			var p *api.Problem
			if !errors.As(err, &p) || p.Code != api.CodeConflict {
				return err
			}
		}
		st.fm.Fields[i].Target = "custom:" + key
	}
	return nil
}

func (st *runState) target(list string) uuid.UUID {
	if id, ok := st.targets[list]; ok {
		return id
	}
	for _, id := range st.targets {
		return id
	}
	return uuid.Nil
}

// importBatch applies one batch in a transaction.
func (s *Service) importBatch(ctx context.Context, st *runState, b batch) error {
	now := s.now()
	type outcome struct {
		row             int32
		outcome, reason string
	}
	var outs []outcome
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		for _, r := range b.rows {
			v := classify(r, b, st.cm, now)
			oc, reason, err := s.applyRow(ctx, tx, st, r, v, now)
			if err != nil {
				return err
			}
			outs = append(outs, outcome{r.RowNo, oc, reason})
			email := r.Email
			if v.Class == classTrap {
				email = "redacted"
			}
			_ = st.report.Write([]string{fmt.Sprint(r.RowNo), formulaSafe(email), r.Status, oc, reason})
			if oc == "skipped" {
				raw := r.Raw
				if v.Class == classTrap {
					raw = "redacted"
				}
				_ = st.skipped.Write([]string{fmt.Sprint(r.RowNo), formulaSafe(raw), r.Status, reason})
			}
		}
		rows := make([]int32, len(outs))
		ocs := make([]string, len(outs))
		reasons := make([]string, len(outs))
		for i, o := range outs {
			rows[i], ocs[i], reasons[i] = o.row, o.outcome, o.reason
		}
		_, err := tx.Exec(ctx, `UPDATE import_rows r SET outcome = x.outcome, reason = x.reason
			FROM unnest($2::int[], $3::text[], $4::text[]) AS x(row_no, outcome, reason)
			WHERE r.import_id = $1 AND r.row_no = x.row_no`, st.imp.ID, rows, ocs, reasons)
		return err
	})
	return err
}

// applyRow imports, merges, suppresses or skips one row.
func (s *Service) applyRow(ctx context.Context, tx pgx.Tx, st *runState, r mappedRow, v verdict, now time.Time) (string, string, error) {
	q := store.New(tx)
	skip := func(reason string) (string, string, error) {
		st.progress.Skipped++
		st.stats.Skipped++
		st.stats.Reasons[reason]++
		return "skipped", reason, nil
	}
	switch v.Class {
	case classInvalid:
		return skip("invalid")
	case classSkipStatus:
		return skip("status_skipped")
	case classAlreadySuppressed:
		return skip("already_suppressed")
	case classSuppress:
		return s.suppressRow(ctx, tx, st, r, now)
	}
	email := r.Email
	role := false
	pending := st.cm.handling(r.Status) == HandlePending
	reason := ""
	if isRisky(v.Class) {
		action := riskyAction(st.opts.Risky, v.Class)
		if action == "skip" {
			return skip(v.Class)
		}
		switch v.Class {
		case classRole:
			role = action == "exclude_broadcasts"
			reason = "role"
		case classTypo:
			if action == "fix" {
				email = v.Fixed
				r.Tags = append(r.Tags, "review-typo")
				reason = "typo_fixed"
			}
		case classStale:
			if action == "pending" {
				pending = true
				reason = "stale"
			}
		}
		if reason != "" {
			st.stats.Reasons[reason]++
		}
	}
	if v.Class == classDuplicate {
		if err := s.mergeRow(ctx, q, tx, st, r, v.Contact, pending, now); err != nil {
			return "", "", err
		}
		st.progress.Merged++
		st.stats.Merged++
		return "merged", reason, nil
	}
	id, created, err := s.createRow(ctx, tx, st, r, email, role, pending, now)
	if err != nil {
		return "", "", err
	}
	if !created {
		st.progress.Merged++
		st.stats.Merged++
		return "merged", "duplicate in source", nil
	}
	if pending {
		st.pending = append(st.pending, id)
		st.progress.Pending++
		st.stats.Pending++
		return "pending", reason, nil
	}
	st.progress.Imported++
	st.stats.Imported++
	return "imported", reason, nil
}

func (s *Service) suppressRow(ctx context.Context, tx pgx.Tx, st *runState, r mappedRow, now time.Time) (string, string, error) {
	q := store.New(tx)
	reason, contactStatus, code := "unsubscribe", "unsubscribed", "unsubscribed in "+SourceName(st.imp.Source)
	switch st.cm.handling(r.Status) {
	case HandleSuppressBounce:
		reason, contactStatus, code = "hard_bounce", "bounced", "cleaned in "+SourceName(st.imp.Source)
	case HandleSuppressComplaint:
		reason, contactStatus, code = "complaint", "complained", "abuse complaint in "+SourceName(st.imp.Source)
	}
	if st.imp.Source == "brevo" {
		code = "blocklisted in Brevo"
	}
	impID := st.imp.ID
	_, err := q.InsertSuppression(ctx, store.InsertSuppressionParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: st.ws.ID, Email: r.Email, Reason: reason,
		Source: "import:" + st.imp.Source, OriginalCode: code, CreatedBy: st.actor, Scope: suppressionScope(reason), Context: "Migration", ImportID: &impID, CreatedAt: now})
	added := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return "", "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE contacts SET status = $3, updated_at = now() WHERE workspace_id = $1 AND email = $2 AND erased_at IS NULL`, st.ws.ID, r.Email, contactStatus); err != nil {
		return "", "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE list_memberships lm SET status = 'unsubscribed', unsubscribed_at = now() FROM contacts c
		WHERE c.id = lm.contact_id AND c.workspace_id = $1 AND c.email = $2 AND lm.status <> 'unsubscribed'`, st.ws.ID, r.Email); err != nil {
		return "", "", err
	}
	if added {
		st.newSupp = append(st.newSupp, r.Email)
	}
	st.progress.Suppressed++
	st.stats.Suppressed++
	st.stats.SuppressedBy[reason]++
	return "suppressed", reason, nil
}

func suppressionScope(reason string) string {
	if reason == "unsubscribe" {
		return "marketing"
	}
	return "all"
}

// contactValues reads the mapped built-in and custom values of a row.
func contactValues(r mappedRow, fm FieldMap) (map[string]string, map[string]string, map[string]string) {
	builtin := map[string]string{}
	custom := map[string]string{}
	types := map[string]string{}
	for _, f := range fm.Fields {
		v := strings.TrimSpace(r.Data[f.Source])
		switch {
		case v == "" || f.Target == TargetSkip || f.Target == TargetEmail || f.Target == TargetStatus || f.Target == TargetEvidence || f.Target == TargetTags:
		case f.Target == TargetName:
			first, last, _ := strings.Cut(v, " ")
			if builtin["first_name"] == "" {
				builtin["first_name"] = first
			}
			if builtin["last_name"] == "" {
				builtin["last_name"] = strings.TrimSpace(last)
			}
		case strings.HasPrefix(f.Target, "custom:"):
			key := strings.TrimPrefix(f.Target, "custom:")
			custom[key] = v
			types[key] = f.Type
		case f.Target == "language":
			// "Spanish", "es_MX" and "ES" become es and es-MX; values that
			// aren't languages are left empty (Map fields lists them).
			if code := importLanguage(v); code != "" {
				builtin["language"] = code
			}
		default:
			builtin[f.Target] = v
		}
	}
	return builtin, custom, types
}

// typedValue converts an imported text value for a custom field type.
func typedValue(v, typ string) any {
	switch typ {
	case "number":
		var f float64
		if _, err := fmt.Sscan(strings.ReplaceAll(v, ",", ""), &f); err == nil {
			return f
		}
		return nil
	case "boolean":
		switch strings.ToLower(v) {
		case "true", "yes", "1", "y":
			return true
		case "false", "no", "0", "n":
			return false
		}
		return nil
	case "date":
		if len(v) >= 10 && dateRe.MatchString(v) {
			return v[:10]
		}
		if t := parseTime(v); t != nil {
			return t.Format("2006-01-02")
		}
		return nil
	}
	return v
}

func (s *Service) customTypes(ctx context.Context, ws uuid.UUID) map[string]string {
	out := map[string]string{}
	if rows, err := s.Q.ListCustomFields(ctx, ws); err == nil {
		for _, f := range rows {
			out[f.Key] = f.Type
		}
	}
	return out
}

// createRow inserts a new contact with its membership and consent history.
// It reports false when the address was inserted by an earlier row.
func (s *Service) createRow(ctx context.Context, tx pgx.Tx, st *runState, r mappedRow, email string, role, pending bool, now time.Time) (uuid.UUID, bool, error) {
	builtin, custom, _ := contactValues(r, st.fm)
	types := s.customTypes(ctx, st.ws.ID)
	attrs := map[string]any{}
	for k, v := range custom {
		if tv := typedValue(v, types[k]); tv != nil {
			attrs[k] = tv
		}
	}
	tags := []string{}
	if st.opts.ImportTags {
		tags = cleanTags(r.Tags)
	} else if containsStr(r.Tags, "review-typo") {
		tags = []string{"review-typo"}
	}
	consent := "single"
	handling := st.cm.handling(r.Status)
	switch {
	case pending || handling == HandlePending:
		consent = "pending"
	case anyDoubleOptIn(st.conn) || r.Eng.OptinAt != nil:
		consent = "double"
	}
	evidence := map[string]any{}
	if r.Eng.OptinAt != nil && consent != "pending" {
		evidence["at"] = r.Eng.OptinAt.UTC().Format(time.RFC3339)
		evidence["source"] = SourceName(st.imp.Source) + " opt-in"
		if r.Eng.OptinIP != "" {
			evidence["ip"] = r.Eng.OptinIP
		}
	}
	created := now
	if r.Eng.SignupAt != nil && r.Eng.SignupAt.Before(now) {
		created = *r.Eng.SignupAt
	}
	var lastOpen, lastClick *time.Time
	if st.opts.ImportEngagement {
		lastOpen, lastClick = r.Eng.LastOpen, r.Eng.LastClick
	}
	tier := audience.TierFor(st.policy, lastOpen, lastOpen, lastClick, created, now)
	domain := emailaddr.Domain(email)
	impID := st.imp.ID
	id := uuid.Must(uuid.NewV7())
	var got uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name, last_name, company, country, timezone, language,
			mailbox_provider, mailbox_detail, consent, consent_evidence, source, tags, attributes, role_address, import_id, tier,
			last_open_at, last_human_open_at, last_click_at, last_activity_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$20,$21,$22,$23,$22)
		ON CONFLICT (workspace_id, email) DO NOTHING
		RETURNING id`,
		id, st.ws.ID, email, builtin["first_name"], builtin["last_name"], builtin["company"], strings.ToUpper(builtin["country"]), validTZ(builtin["timezone"]),
		builtin["language"], emailaddr.MailboxProvider(domain), st.hosting[domain], consent, mustJSON(evidence), "import:"+st.imp.Source, tags, mustJSON(attrs),
		role, &impID, tier, lastOpen, lastClick, now, created).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	if builtin["language"] != "" {
		if err := audience.RecordLanguageSource(ctx, store.New(tx), got, audience.LanguageFromImport, SourceName(st.imp.Source), now, nil); err != nil {
			return uuid.Nil, false, err
		}
	}
	status := "subscribed"
	if consent == "pending" {
		status = "pending"
	}
	list := st.target(r.List)
	subscribed := r.Eng.OptinAt
	if subscribed == nil {
		subscribed = &created
	}
	if _, err := tx.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
		VALUES ($1, $2, $3, $4, CASE WHEN $3 = 'subscribed' THEN $5::timestamptz END) ON CONFLICT DO NOTHING`, list, got, status, "import:"+st.imp.Source, subscribed); err != nil {
		return uuid.Nil, false, err
	}
	if st.opts.RecordHistory {
		if err := s.recordImport(ctx, tx, st, got, list, r, consent, now); err != nil {
			return uuid.Nil, false, err
		}
	}
	return got, true, nil
}

func validTZ(tz string) string {
	if tz == "" {
		return ""
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ""
	}
	return tz
}

// recordImport writes the consent history entry of an imported row.
func (s *Service) recordImport(ctx context.Context, tx pgx.Tx, st *runState, contact, list uuid.UUID, r mappedRow, consent string, now time.Time) error {
	detail := fmt.Sprintf("list %q · status %s", listName(st.conn, r.List), r.Status)
	if isFile(st.imp.Source) {
		detail = fmt.Sprintf("file %s · status %s", st.imp.FileName, r.Status)
	}
	note := "consent carried over"
	switch {
	case consent == "pending":
		note = `consent carried over as "pending" until re-permission`
	case r.Eng.OptinAt != nil:
		note = "evidence OPTIN_TIME " + r.Eng.OptinAt.UTC().Format("2006-01-02")
	}
	impID := st.imp.ID
	_, err := tx.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, detail, note, actor_id, import_id, occurred_at, ip)
		VALUES ($1, $2, $3, 'imported', $4, $5, $6, $7, $8, $9, $10)`,
		uuid.Must(uuid.NewV7()), contact, list, "import:"+st.imp.Source, detail, note, st.actor, &impID, now, optIP(r.Eng.OptinIP))
	return err
}

func optIP(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

// mergeRow updates an existing contact: fields per the conflict rule, new
// tags, the target list (consent kept) and the consent history.
func (s *Service) mergeRow(ctx context.Context, q *store.Queries, tx pgx.Tx, st *runState, r mappedRow, c *existingContact, pending bool, now time.Time) error {
	builtin, custom, _ := contactValues(r, st.fm)
	types := s.customTypes(ctx, st.ws.ID)
	cur := map[string]string{"first_name": c.FirstName, "last_name": c.LastName, "company": c.Company, "country": c.Country, "timezone": c.Timezone, "language": c.Language}
	apply := func(old, v string) string {
		switch st.opts.OnConflict {
		case "overwrite":
			if v != "" {
				return v
			}
		case "keep":
			return old
		default:
			if old == "" {
				return v
			}
		}
		return old
	}
	for k, v := range builtin {
		if k == "country" {
			v = strings.ToUpper(v)
		}
		if k == "timezone" {
			v = validTZ(v)
		}
		cur[k] = apply(cur[k], v)
	}
	attrs := c.Attributes
	for k, v := range custom {
		tv := typedValue(v, types[k])
		if tv == nil {
			continue
		}
		old := strings.TrimSpace(fmt.Sprint(valueOr(attrs[k], "")))
		if apply(old, fmt.Sprint(tv)) != old {
			attrs[k] = tv
		}
	}
	tags := c.Tags
	if st.opts.ImportTags {
		tags = cleanTags(append(append([]string{}, c.Tags...), r.Tags...))
	}
	if _, err := tx.Exec(ctx, `UPDATE contacts SET first_name = $2, last_name = $3, company = $4, country = $5, timezone = $6, language = $7,
		tags = $8, attributes = $9, updated_at = now() WHERE id = $1`,
		c.ID, cur["first_name"], cur["last_name"], cur["company"], cur["country"], cur["timezone"], cur["language"], tags, mustJSON(attrs)); err != nil {
		return err
	}
	if cur["language"] != c.Language {
		if err := audience.RecordLanguageSource(ctx, q, c.ID, audience.LanguageFromImport, SourceName(st.imp.Source), now, nil); err != nil {
			return err
		}
	}
	status := "subscribed"
	if pending || c.Consent == "pending" {
		status = "pending"
	}
	list := st.target(r.List)
	if _, err := tx.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
		VALUES ($1, $2, $3, $4, CASE WHEN $3 = 'subscribed' THEN now() END) ON CONFLICT DO NOTHING`, list, c.ID, status, "import:"+st.imp.Source); err != nil {
		return err
	}
	if st.opts.RecordHistory {
		return s.recordImport(ctx, tx, st, c.ID, list, r, c.Consent, now)
	}
	return nil
}

func cleanTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] || len(t) > 60 {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// pushSuppressions sends the new suppressions to providers that keep their
// own list (SES, Postmark, Postal adapters with a suppression API).
func (s *Service) pushSuppressions(ctx context.Context, st *runState) {
	if len(st.newSupp) == 0 {
		return
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, name, adapter, config, credentials_enc FROM providers WHERE workspace_id = $1 AND disconnected_at IS NULL ORDER BY connected_at`, st.ws.ID)
	if err != nil {
		return
	}
	type prov struct {
		id            uuid.UUID
		name, adapter string
		config        json.RawMessage
		creds         []byte
	}
	var provs []prov
	for rows.Next() {
		var p prov
		if rows.Scan(&p.id, &p.name, &p.adapter, &p.config, &p.creds) == nil {
			provs = append(provs, p)
		}
	}
	rows.Close()
	var pushed, waiting []string
	for _, p := range provs {
		desc, ok := providers.Lookup(p.adapter)
		if !ok || desc.New == nil {
			waiting = append(waiting, p.name)
			continue
		}
		var creds []byte
		if len(p.creds) > 0 && s.Box != nil {
			creds, _ = s.Box.Open(p.creds, "provider:"+p.id.String()+":credentials")
		}
		inst, err := desc.New(providers.Config{ProviderID: p.id, WorkspaceID: st.ws.ID, Settings: p.config, Credentials: creds})
		if err != nil {
			waiting = append(waiting, p.name)
			continue
		}
		syncer, ok := inst.(providers.SuppressionSyncer)
		if !ok {
			continue
		}
		if err := syncer.AddSuppressions(ctx, st.newSupp); err != nil {
			waiting = append(waiting, p.name)
			continue
		}
		_, _ = s.Pool.Exec(ctx, `INSERT INTO provider_suppression_sync (provider_id, mode, last_push_at, pushed) VALUES ($1, 'two_way', now(), $2)
			ON CONFLICT (provider_id) DO UPDATE SET last_push_at = now(), pushed = provider_suppression_sync.pushed + $2`, p.id, len(st.newSupp))
		pushed = append(pushed, p.name)
	}
	st.stats.PushedTo = pushed
	n := thousands(len(st.newSupp))
	if len(pushed) > 0 {
		_ = s.log(ctx, s.Q, st.imp.ID, "sync", fmt.Sprintf("Pushed %s new suppressions to %s", n, joinAnd(pushed)))
	}
	if len(waiting) > 0 {
		_ = s.log(ctx, s.Q, st.imp.ID, "sync", fmt.Sprintf("%s new suppressions reach %s with the next provider sync", n, joinAnd(waiting)))
	}
}

// sendRepermission emails pending contacts their confirmation link.
func (s *Service) sendRepermission(ctx context.Context, st *runState) {
	if len(st.pending) == 0 || s.Audience == nil {
		return
	}
	n, err := s.Audience.SendConfirmations(ctx, &st.ws, st.pending)
	st.stats.ConfirmsSent = int64(n)
	switch {
	case err != nil:
		_ = s.log(ctx, s.Q, st.imp.ID, "error", "Re-permission emails couldn't be sent: "+err.Error())
	case n > 0:
		_ = s.log(ctx, s.Q, st.imp.ID, "sync", fmt.Sprintf("Sent %s re-permission emails through the System stream", thousands(n)))
	default:
		_ = s.log(ctx, s.Q, st.imp.ID, "info", fmt.Sprintf("%s contacts wait for re-permission; connect a System provider to email them", thousands(len(st.pending))))
	}
}

func (s *Service) finish(ctx context.Context, st *runState, status string, start time.Time) error {
	st.stats.Rows = st.progress.Processed
	st.stats.DurationMs = s.now().Sub(start).Milliseconds()
	for _, id := range st.targets {
		if l, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: st.ws.ID, ID: id}); err == nil {
			var n int64
			_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM list_memberships WHERE list_id = $1 AND status = 'subscribed'`, id).Scan(&n)
			st.stats.Lists = append(st.stats.Lists, ImportListAfter{Name: l.Name, Contacts: n})
		}
	}
	st.report.Flush()
	st.skipped.Flush()
	step := "done"
	if status != "done" {
		step = "import"
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.FinishImport(ctx, store.FinishImportParams{ID: st.imp.ID, Status: status, Step: step, Stats: mustJSON(st.stats), Progress: mustJSON(st.progress),
			ReportCsv: st.reportBuf.Bytes(), SkippedCsv: st.skipBuf.Bytes()}); err != nil {
			return err
		}
		msg := fmt.Sprintf("Import finished · %s contacts · %s suppressions", thousands(int(st.stats.Imported+st.stats.Merged+st.stats.Pending)), thousands(int(st.stats.Suppressed)))
		if status == "cancelled" {
			msg = fmt.Sprintf("Import stopped after %s rows", thousands(int(st.progress.Processed)))
		}
		if err := s.log(ctx, q, st.imp.ID, "done", msg); err != nil {
			return err
		}
		if status == "done" {
			if err := q.DeleteImportRows(ctx, st.imp.ID); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &st.ws.ID, Action: "import.finish", TargetType: "import", TargetID: st.imp.ID.String(),
			TargetName: SourceName(st.imp.Source), Result: audit.Allowed, After: st.stats, Metadata: map[string]any{"status": status}})
	})
	if err != nil {
		return err
	}
	// A big import changes the tables' shape; fresh planner statistics keep
	// segment counts and List health fast before autovacuum gets there.
	if st.progress.Processed >= analyzeAfterRows {
		if _, err := s.Pool.Exec(ctx, `ANALYZE contacts, list_memberships, suppressions`); err != nil {
			_ = s.log(ctx, s.Q, st.imp.ID, "info", "Couldn't refresh the planner statistics: "+err.Error())
		}
	}
	if st.imp.CreatedBy != nil && status == "done" {
		wsID := st.ws.ID
		_ = notify.ToUser(ctx, s.Pool, *st.imp.CreatedBy, notify.Notification{
			WorkspaceID: &wsID, Kind: notify.KindJobFinished, Severity: "ok",
			Title: fmt.Sprintf("%s contacts imported from %s", thousands(int(st.stats.Imported+st.stats.Merged)), SourceName(st.imp.Source)),
			Body:  fmt.Sprintf("%s suppressed, %s skipped. The report lists every skipped row.", thousands(int(st.stats.Suppressed)), thousands(int(st.stats.Skipped))),
			Link:  fmt.Sprintf("/w/%s/import/%s/report", st.ws.Slug, st.imp.ID),
		})
	}
	return nil
}

func (s *Service) runFailed(ctx context.Context, st *runState, cause error) error {
	_ = s.log(ctx, s.Q, st.imp.ID, "error", "Import stopped: "+cause.Error())
	_ = s.Q.FinishImport(ctx, store.FinishImportParams{ID: st.imp.ID, Status: "failed", Step: "import", Stats: mustJSON(st.stats),
		Progress: mustJSON(st.progress), Error: cause.Error()})
	return cause
}
