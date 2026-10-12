package audience

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// Suppression reasons shown in the UI, in the order groups list them.
var reasonOrder = []string{"hard_bounce", "complaint", "unsubscribe", "manual", "imported", "role", "spam_trap", "invalid"}

func reasonName(r string) string {
	switch r {
	case "hard_bounce":
		return "Hard bounce"
	case "complaint":
		return "Complaint"
	case "unsubscribe":
		return "Unsubscribed"
	case "manual":
		return "Manual"
	case "imported":
		return "Imported"
	case "role":
		return "Role address"
	case "spam_trap":
		return "Spam trap"
	case "invalid":
		return "Invalid"
	}
	return r
}

// scopeFor is the streams a suppression blocks: unsubscribes keep System
// mail (password resets, receipts); everything else blocks every stream.
func scopeFor(reason string) string {
	if reason == "unsubscribe" {
		return "marketing"
	}
	return "all"
}

// SuppressionSource says where a suppression came from.
type SuppressionSource struct {
	// Kind: provider, preference_center, one_click, manual, api,
	// bounce_mailbox, import, provider_sync, csv.
	Kind  string `json:"kind"`
	Label string `json:"label" doc:"Amazon SES, Preference center, Linh Nguyen, Brevo import…"`
	// Adapter is set for providers (ses, postmark, postal…), for the badge colour.
	Adapter string `json:"adapter,omitempty"`
}

// SuppressionItem is one row of the suppression list, or a group of rows
// that came in with one import.
type SuppressionItem struct {
	ID string `json:"id"`
	// Address is masked (r.w···@outlook.com) except role addresses; for an
	// import group it's the source file name.
	Address string `json:"address"`
	Group   bool   `json:"group"`
	// Rows in an import group.
	Rows     int64             `json:"rows"`
	ImportID string            `json:"importId,omitempty"`
	Reasons  []string          `json:"reasons"`
	Source   SuppressionSource `json:"source"`
	// OriginalCode is the provider's code ("550 5.1.1 user unknown") or a note.
	OriginalCode string    `json:"originalCode"`
	At           time.Time `json:"at"`
	// Context is the campaign or flow ("Welcome series · step 1") or a note
	// ("Migration"); empty when there is none.
	Context    string `json:"context"`
	CampaignID string `json:"campaignId,omitempty"`
	Scope      string `json:"scope" enum:"all,marketing"`
}

// SuppressionCountsView is the stat row above the list.
type SuppressionCountsView struct {
	Total       int64 `json:"total"`
	HardBounce  int64 `json:"hardBounce"`
	Complaint   int64 `json:"complaint"`
	Unsubscribe int64 `json:"unsubscribe"`
	Manual      int64 `json:"manual"`
	Other       int64 `json:"other"`
}

// ProviderSync is a sending provider's suppression sync state (kept by the
// providers domain; shown here).
type ProviderSync struct {
	ProviderID string     `json:"providerId"`
	Name       string     `json:"name"`
	Adapter    string     `json:"adapter"`
	Status     string     `json:"status" enum:"in_sync,syncing,error,off"`
	At         *time.Time `json:"at,omitempty"`
	Pulled     int64      `json:"pulled"`
	Pushed     int64      `json:"pushed"`
	// PulledNote explains what "pulled" covers for this provider.
	PulledNote string `json:"pulledNote"`
	Detail     string `json:"detail"`
}

// FilterOption is a value of a filter select.
type FilterOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// SuppressionList is one page of the suppression list.
type SuppressionList struct {
	Items    []SuppressionItem     `json:"items"`
	Total    int64                 `json:"total" doc:"Rows matching, groups counted once"`
	Matching int64                 `json:"matching" doc:"Addresses matching the filter"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Counts   SuppressionCountsView `json:"counts"`
	Sync     []ProviderSync        `json:"sync"`
	Sources  []FilterOption        `json:"sources"`
}

// SuppressionFilter narrows the list.
type SuppressionFilter struct {
	Q      string
	Reason string
	Source string // provider:<id>, import:<id>, manual, preference_center, one_click, api, bounce_mailbox
	Since  string // 7d, 30d
	Import string // show the rows of one import group
}

// ListSuppressions returns one page of the shared suppression list. Rows
// that came in with an import are grouped unless the person searches or
// opens the group.
func (s *Service) ListSuppressions(ctx context.Context, ws *store.Workspace, f SuppressionFilter, page, pageSize int) (SuppressionList, error) {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	out := SuppressionList{Items: []SuppressionItem{}, Page: page, PageSize: pageSize, Sync: []ProviderSync{}, Sources: []FilterOption{}}
	fs := filterSQL{where: "s.workspace_id = $1", args: []any{ws.ID}}
	if q := strings.TrimSpace(strings.ToLower(f.Q)); q != "" {
		fs.add("s.email LIKE ?", "%"+escapeLike(q)+"%")
	}
	if f.Reason != "" {
		fs.add("s.reason = ?", f.Reason)
	}
	switch {
	case strings.HasPrefix(f.Source, "provider:"):
		id, err := uuid.Parse(strings.TrimPrefix(f.Source, "provider:"))
		if err != nil {
			return out, api.Invalid("source", "Pick a source from the list.")
		}
		fs.add("s.source_provider_id = ?", id)
	case strings.HasPrefix(f.Source, "import:"):
		id, err := uuid.Parse(strings.TrimPrefix(f.Source, "import:"))
		if err != nil {
			return out, api.Invalid("source", "Pick a source from the list.")
		}
		fs.add("s.import_id = ?", id)
	case f.Source != "":
		fs.add("s.source = ? AND s.source_provider_id IS NULL", f.Source)
	}
	switch f.Since {
	case "7d":
		fs.add("s.created_at >= ?", s.now().AddDate(0, 0, -7))
	case "30d":
		fs.add("s.created_at >= ?", s.now().AddDate(0, 0, -30))
	}
	if f.Import != "" {
		id, err := uuid.Parse(f.Import)
		if err != nil {
			return out, api.Invalid("import", "Pick an import group.")
		}
		fs.add("s.import_id = ?", id)
	}
	group := strings.TrimSpace(f.Q) == "" && f.Import == "" && !strings.HasPrefix(f.Source, "import:")
	cte := `WITH f AS (SELECT s.* FROM suppressions s WHERE ` + fs.where + `),
	r AS (
		SELECT false AS grp, f.id, f.email, ARRAY[f.reason] AS reasons, f.source, f.source_provider_id, f.original_code, f.created_at,
		       f.campaign_id, f.automation_id, f.automation_step, f.context, f.scope, f.import_id, 1::bigint AS n, f.created_by
		FROM f WHERE NOT ` + fmt.Sprint(group) + ` OR f.import_id IS NULL
		UNION ALL
		SELECT true, (array_agg(f.id ORDER BY f.created_at, f.id))[1], '', array_agg(DISTINCT f.reason), 'import', NULL, max(f.original_code),
		       max(f.created_at), NULL, NULL, '', max(f.context), CASE WHEN bool_or(f.scope = 'all') THEN 'all' ELSE 'marketing' END,
		       f.import_id, count(*)::bigint, NULL
		FROM f WHERE ` + fmt.Sprint(group) + ` AND f.import_id IS NOT NULL GROUP BY f.import_id
	)`
	if err := s.Pool.QueryRow(ctx, cte+` SELECT count(*), coalesce(sum(n), 0)::bigint FROM r`, fs.args...).Scan(&out.Total, &out.Matching); err != nil {
		return out, err
	}
	args := append(fs.args, pageSize, (page-1)*pageSize)
	rows, err := s.Pool.Query(ctx, cte+fmt.Sprintf(`
		SELECT r.grp, r.id, r.email, r.reasons, r.source, r.original_code, r.created_at, r.campaign_id, r.automation_step,
		       r.context, r.scope, r.import_id, r.n,
		       coalesce(p.name, ''), coalesce(p.adapter, ''), coalesce(u.name, ''), coalesce(i.source, ''), coalesce(i.file_name, ''),
		       coalesce(cp.name, ''), coalesce(a.name, '')
		FROM r
		LEFT JOIN providers p ON p.id = r.source_provider_id
		LEFT JOIN users u ON u.id = r.created_by
		LEFT JOIN imports i ON i.id = r.import_id
		LEFT JOIN campaigns cp ON cp.id = r.campaign_id
		LEFT JOIN automations a ON a.id = r.automation_id
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var it SuppressionItem
		var id uuid.UUID
		var campaignID, importID *uuid.UUID
		var email, source, step, provName, adapter, userName, impSource, fileName, campaign, automation string
		if err := rows.Scan(&it.Group, &id, &email, &it.Reasons, &source, &it.OriginalCode, &it.At, &campaignID, &step,
			&it.Context, &it.Scope, &importID, &it.Rows, &provName, &adapter, &userName, &impSource, &fileName, &campaign, &automation); err != nil {
			return out, err
		}
		it.ID = id.String()
		it.Reasons = sortReasons(it.Reasons)
		if importID != nil {
			it.ImportID = importID.String()
		}
		if campaignID != nil {
			it.CampaignID = campaignID.String()
		}
		switch {
		case it.Group:
			it.Address = fileName
			if it.Address == "" {
				it.Address = importerLabel(impSource) + " export"
			}
		case emailaddr.IsRole(email):
			it.Address = email
		default:
			it.Address = emailaddr.Mask(email)
		}
		it.Source = suppressionSource(source, provName, adapter, userName, impSource)
		switch {
		case campaign != "":
			it.Context = campaign
		case automation != "":
			it.Context = automation
			if step != "" {
				it.Context += " · " + step
			}
		}
		out.Items = append(out.Items, it)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	counts, err := s.Q.SuppressionCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Counts = SuppressionCountsView{Total: counts.Total, HardBounce: counts.HardBounce, Complaint: counts.Complaint, Unsubscribe: counts.Unsubscribe, Manual: counts.Manual, Other: counts.Other}
	if out.Sync, err = s.syncStates(ctx, ws); err != nil {
		return out, err
	}
	out.Sources, err = s.suppressionSources(ctx, ws)
	return out, err
}

func sortReasons(in []string) []string {
	out := make([]string, 0, len(in))
	for _, r := range reasonOrder {
		for _, x := range in {
			if x == r {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

func importerLabel(src string) string {
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
	case "csv":
		return "CSV"
	case "xlsx":
		return "XLSX"
	}
	return "Import"
}

func suppressionSource(source, provider, adapter, user, importSource string) SuppressionSource {
	if provider != "" {
		return SuppressionSource{Kind: "provider", Label: provider, Adapter: adapter}
	}
	switch source {
	case "preference_center":
		return SuppressionSource{Kind: source, Label: "Preference center"}
	case "one_click":
		return SuppressionSource{Kind: source, Label: "List-Unsubscribe one-click"}
	case "bounce_mailbox":
		return SuppressionSource{Kind: source, Label: "SMTP bounce mailbox"}
	case "api":
		return SuppressionSource{Kind: source, Label: "API"}
	case "import":
		return SuppressionSource{Kind: source, Label: importerLabel(importSource) + " import"}
	case "csv":
		return SuppressionSource{Kind: source, Label: "CSV upload"}
	case "manual":
		if user != "" {
			return SuppressionSource{Kind: source, Label: user}
		}
		return SuppressionSource{Kind: source, Label: "Manual"}
	}
	return SuppressionSource{Kind: source, Label: source}
}

func (s *Service) syncStates(ctx context.Context, ws *store.Workspace) ([]ProviderSync, error) {
	rows, err := s.Q.SuppressionSyncStates(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]ProviderSync, 0, len(rows))
	for _, r := range rows {
		ps := ProviderSync{ProviderID: r.ID.String(), Name: r.Name, Adapter: r.Adapter, Pulled: r.Pulled, Pushed: r.Pushed, Detail: r.StatusDetail}
		switch r.Adapter {
		case "ses":
			ps.PulledNote = "pulled from the account-level list"
		case "postmark":
			ps.PulledNote = "pulled (bounces, complaints, manual)"
		default:
			ps.PulledNote = "pulled"
		}
		at := r.LastPushAt
		if r.LastPullAt != nil && (at == nil || r.LastPullAt.After(*at)) {
			at = r.LastPullAt
		}
		ps.At = at
		switch {
		case !r.HasSync || r.Mode == "off":
			ps.Status = "off"
		case r.Status == "error":
			ps.Status = "error"
		case r.Pending > 0:
			ps.Status = "syncing"
		default:
			ps.Status = "in_sync"
		}
		out = append(out, ps)
	}
	return out, nil
}

func (s *Service) suppressionSources(ctx context.Context, ws *store.Workspace) ([]FilterOption, error) {
	out := []FilterOption{}
	provs, err := s.Q.WorkspaceProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, p := range provs {
		out = append(out, FilterOption{Value: "provider:" + p.ID.String(), Label: p.Name})
	}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT i.id, i.source, i.created_at FROM suppressions s JOIN imports i ON i.id = s.import_id
		WHERE s.workspace_id = $1 ORDER BY i.created_at DESC LIMIT 10`, ws.ID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var src string
		var at time.Time
		if err := rows.Scan(&id, &src, &at); err != nil {
			return out, err
		}
		out = append(out, FilterOption{Value: "import:" + id.String(), Label: importerLabel(src) + " import"})
	}
	out = append(out, FilterOption{Value: "manual", Label: "Manual"}, FilterOption{Value: "preference_center", Label: "Preference center"},
		FilterOption{Value: "one_click", Label: "List-Unsubscribe one-click"})
	return out, rows.Err()
}

// AddResult reports how many addresses were added.
type AddResult struct {
	Added   int      `json:"added"`
	Already int      `json:"already" doc:"Already on the list"`
	Invalid []string `json:"invalid" doc:"Values that aren't addresses"`
}

// AddSuppressionsInput adds addresses by hand.
type AddSuppressionsInput struct {
	Emails []string `json:"emails" minItems:"1" maxItems:"1000"`
	Note   string   `json:"note,omitempty" maxLength:"300" doc:"Why, shown as the original code"`
}

// AddSuppressions adds addresses to the shared suppression list by hand.
func (s *Service) AddSuppressions(ctx context.Context, ws *store.Workspace, in AddSuppressionsInput) (AddResult, error) {
	return s.addSuppressions(ctx, ws, in.Emails, "manual", strings.TrimSpace(in.Note))
}

func (s *Service) addSuppressions(ctx context.Context, ws *store.Workspace, emails []string, source, note string) (AddResult, error) {
	res := AddResult{Invalid: []string{}}
	valid := []string{}
	seen := map[string]bool{}
	for _, raw := range emails {
		e, problem := emailaddr.Check(raw)
		if problem != "" {
			res.Invalid = append(res.Invalid, raw)
			continue
		}
		if !seen[e] {
			seen[e] = true
			valid = append(valid, e)
		}
	}
	actor := actorID(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		added := []string{}
		for _, e := range valid {
			_, err := q.InsertSuppression(ctx, store.InsertSuppressionParams{
				ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: e, Reason: "manual", Source: source, OriginalCode: note,
				CreatedBy: actor, Scope: "all", CreatedAt: s.now(),
			})
			if errors.Is(err, pgx.ErrNoRows) {
				res.Already++
				continue
			}
			if err != nil {
				return err
			}
			added = append(added, e)
		}
		res.Added = len(added)
		if len(added) == 0 {
			return nil
		}
		rows, err := tx.Query(ctx, `UPDATE contacts SET status = 'suppressed', updated_at = now() WHERE workspace_id = $1 AND email = ANY ($2) AND erased_at IS NULL RETURNING id`, ws.ID, added)
		if err != nil {
			return err
		}
		contacts, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		if err := insertConsentEvents(ctx, tx, contacts, nil, "suppressed", "team", note, actor, s.now()); err != nil {
			return err
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "suppression.add", TargetType: "suppression", TargetName: strings.Join(first(added, 3), ", "),
			Reason: note, Metadata: map[string]any{"count": len(added), "emails": first(added, 50), "source": source}}
		if len(added) == 1 {
			ev.TargetName = added[0]
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	return res, err
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

// RemoveSuppressionInput asks for a reason, which goes to the audit log.
type RemoveSuppressionInput struct {
	Reason string `json:"reason" minLength:"3" maxLength:"300"`
}

// RemoveSuppression takes one address off the list. Addresses that
// bounced or complained can only be mailed again after a new opt-in, so
// their contact goes back to pending.
func (s *Service) RemoveSuppression(ctx context.Context, ws *store.Workspace, id uuid.UUID, in RemoveSuppressionInput) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		sup, err := q.GetSuppression(ctx, store.GetSuppressionParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Suppression")
		}
		if _, err := q.DeleteSuppression(ctx, store.DeleteSuppressionParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		switch sup.Reason {
		case "hard_bounce", "complaint", "spam_trap", "invalid":
			if _, err := tx.Exec(ctx, `UPDATE contacts SET consent = 'pending', status = 'active', updated_at = now() WHERE workspace_id = $1 AND email = $2 AND erased_at IS NULL`, ws.ID, sup.Email); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE list_memberships lm SET status = 'pending' FROM contacts c
				WHERE c.id = lm.contact_id AND c.workspace_id = $1 AND c.email = $2 AND lm.status = 'subscribed'`, ws.ID, sup.Email); err != nil {
				return err
			}
		default:
			if _, err := tx.Exec(ctx, `UPDATE contacts SET status = 'active', updated_at = now() WHERE workspace_id = $1 AND email = $2 AND erased_at IS NULL AND status = 'suppressed'`, ws.ID, sup.Email); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "suppression.remove", TargetType: "suppression", TargetID: id.String(),
			TargetName: sup.Email, Reason: in.Reason,
			Before: map[string]any{"email": sup.Email, "reason": sup.Reason, "source": sup.Source, "originalCode": sup.OriginalCode, "createdAt": sup.CreatedAt}})
	})
}

// ExportSuppressions writes the whole list as CSV. Every export is logged.
func (s *Service) ExportSuppressions(ctx context.Context, ws *store.Workspace) ([]byte, int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT s.email, s.reason, s.scope, s.source, coalesce(p.name, ''), s.original_code,
		coalesce(cp.name, a.name || CASE WHEN s.automation_step <> '' THEN ' · ' || s.automation_step ELSE '' END, s.context, ''), s.created_at
		FROM suppressions s
		LEFT JOIN providers p ON p.id = s.source_provider_id
		LEFT JOIN campaigns cp ON cp.id = s.campaign_id
		LEFT JOIN automations a ON a.id = s.automation_id
		WHERE s.workspace_id = $1 ORDER BY s.created_at DESC, s.id`, ws.ID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"email", "reason", "scope", "source", "source_provider", "original_code", "campaign_or_flow", "suppressed_at"})
	n := 0
	for rows.Next() {
		var email, reason, scope, source, prov, code, ctxName string
		var at time.Time
		if err := rows.Scan(&email, &reason, &scope, &source, &prov, &code, &ctxName, &at); err != nil {
			return nil, 0, err
		}
		_ = w.Write(csvSafe(email, reason, scope, source, prov, code, ctxName, at.UTC().Format(time.RFC3339)))
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	w.Flush()
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "suppressions.export", TargetType: "suppression_list", Metadata: map[string]any{"rows": n}})
	return buf.Bytes(), n, err
}

// csvSafe neutralizes values a spreadsheet would run as a formula.
func csvSafe(vals ...string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
			v = "'" + v
		}
		out[i] = v
	}
	return out
}

// UploadResult reports a CSV upload of addresses.
type UploadResult struct {
	AddResult
	Rows int `json:"rows"`
}

// UploadSuppressions reads addresses from a CSV (an "email" column, or the
// first column) and adds them with the given reason.
func (s *Service) UploadSuppressions(ctx context.Context, ws *store.Workspace, body []byte, fileName, reason, note string) (UploadResult, error) {
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	col := 0
	var emails []string
	header := true
	rows := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return UploadResult{}, api.Invalid("body", fmt.Sprintf("%s isn't a readable CSV: %v.", fileName, err))
		}
		if header {
			header = false
			found := false
			for i, h := range rec {
				if strings.EqualFold(strings.TrimSpace(h), "email") || strings.EqualFold(strings.TrimSpace(h), "email address") {
					col, found = i, true
				}
			}
			if found || len(rec) == 0 || !emailaddr.Valid(rec[0]) {
				continue
			}
		}
		if col < len(rec) {
			emails = append(emails, rec[col])
			rows++
		}
		if rows > 100000 {
			return UploadResult{}, api.Invalid("body", "Upload at most 100,000 addresses at a time.")
		}
	}
	if len(emails) == 0 {
		return UploadResult{}, api.Invalid("body", "The file has no addresses. Put them in a column named email.")
	}
	if reason == "" {
		reason = "manual"
	}
	res, err := s.addUploaded(ctx, ws, emails, reason, note, fileName)
	return UploadResult{AddResult: res, Rows: rows}, err
}

func (s *Service) addUploaded(ctx context.Context, ws *store.Workspace, emails []string, reason, note, fileName string) (AddResult, error) {
	res := AddResult{Invalid: []string{}}
	actor := actorID(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, raw := range emails {
			e, problem := emailaddr.Check(raw)
			if problem != "" {
				if len(res.Invalid) < 50 {
					res.Invalid = append(res.Invalid, raw)
				}
				continue
			}
			_, err := q.InsertSuppression(ctx, store.InsertSuppressionParams{
				ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: e, Reason: reason, Source: "csv", OriginalCode: note,
				CreatedBy: actor, Scope: scopeFor(reason), Context: "CSV upload · " + fileName, CreatedAt: s.now(),
			})
			if errors.Is(err, pgx.ErrNoRows) {
				res.Already++
				continue
			}
			if err != nil {
				return err
			}
			res.Added++
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "suppression.upload", TargetType: "suppression_list", TargetName: fileName,
			Reason: note, Metadata: map[string]any{"added": res.Added, "already": res.Already, "reason": reason}})
	})
	return res, err
}

// PullResult reports suppressions pulled from sending providers.
type PullResult struct {
	Providers []PullProvider `json:"providers"`
	Added     int            `json:"added"`
}

// PullProvider is one provider's part of a pull.
type PullProvider struct {
	Name    string `json:"name"`
	Pulled  int    `json:"pulled"`
	Added   int    `json:"added"`
	Skipped string `json:"skipped,omitempty" doc:"Why nothing was pulled"`
}

// Box decrypts provider credentials for pulls.
type Box interface {
	Open(ciphertext []byte, ad string) ([]byte, error)
}

// PullFromProviders reads each connected provider's own suppression list
// (adapters that implement providers.SuppressionSyncer) and adds what the
// shared list doesn't have yet. Regular two-way sync is the providers
// domain's job; this is the one-off pull offered by the importer and the
// suppression list.
func (s *Service) PullFromProviders(ctx context.Context, ws *store.Workspace, box Box) (PullResult, error) {
	out := PullResult{Providers: []PullProvider{}}
	rows, err := s.Pool.Query(ctx, `SELECT id, name, adapter, config, credentials_enc FROM providers WHERE workspace_id = $1 AND disconnected_at IS NULL ORDER BY connected_at`, ws.ID)
	if err != nil {
		return out, err
	}
	type prov struct {
		id      uuid.UUID
		name    string
		adapter string
		config  json.RawMessage
		creds   []byte
	}
	var provs []prov
	for rows.Next() {
		var p prov
		if err := rows.Scan(&p.id, &p.name, &p.adapter, &p.config, &p.creds); err != nil {
			rows.Close()
			return out, err
		}
		provs = append(provs, p)
	}
	rows.Close()
	for _, p := range provs {
		pp := PullProvider{Name: p.name}
		desc, ok := providers.Lookup(p.adapter)
		if !ok || desc.New == nil {
			pp.Skipped = "adapter not installed"
			out.Providers = append(out.Providers, pp)
			continue
		}
		var creds []byte
		if len(p.creds) > 0 && box != nil {
			if creds, err = box.Open(p.creds, "provider:"+p.id.String()+":credentials"); err != nil {
				pp.Skipped = "credentials can't be read"
				out.Providers = append(out.Providers, pp)
				continue
			}
		}
		inst, err := desc.New(providers.Config{ProviderID: p.id, WorkspaceID: ws.ID, Settings: p.config, Credentials: creds})
		if err != nil {
			pp.Skipped = err.Error()
			out.Providers = append(out.Providers, pp)
			continue
		}
		syncer, ok := inst.(providers.SuppressionSyncer)
		if !ok {
			pp.Skipped = "no suppression list API"
			out.Providers = append(out.Providers, pp)
			continue
		}
		list, err := syncer.ListSuppressions(ctx, time.Time{})
		if err != nil {
			pp.Skipped = "the provider didn't answer: " + err.Error()
			out.Providers = append(out.Providers, pp)
			continue
		}
		pp.Pulled = len(list)
		pid := p.id
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, sup := range list {
				e, problem := emailaddr.Check(sup.Email)
				if problem != "" {
					continue
				}
				reason := sup.Reason
				switch reason {
				case "hard_bounce", "complaint", "unsubscribe", "manual":
				default:
					reason = "manual"
				}
				at := sup.CreatedAt
				if at.IsZero() {
					at = s.now()
				}
				_, err := q.InsertSuppression(ctx, store.InsertSuppressionParams{
					ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: e, Reason: reason, Source: "provider_sync", SourceProviderID: &pid,
					OriginalCode: sup.Code, Scope: scopeFor(reason), CreatedAt: at,
				})
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				if err != nil {
					return err
				}
				pp.Added++
			}
			if _, err := tx.Exec(ctx, `INSERT INTO provider_suppression_sync (provider_id, mode, last_pull_at, pulled) VALUES ($1, 'two_way', now(), $2)
				ON CONFLICT (provider_id) DO UPDATE SET last_pull_at = now(), pulled = provider_suppression_sync.pulled + $2`, pid, pp.Added); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "suppression.pull", TargetType: "provider", TargetID: pid.String(), TargetName: p.name,
				Metadata: map[string]any{"pulled": pp.Pulled, "added": pp.Added}})
		})
		if err != nil {
			return out, err
		}
		out.Added += pp.Added
		out.Providers = append(out.Providers, pp)
	}
	return out, nil
}
