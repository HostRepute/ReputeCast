package auditlog

// Instance admin › Audit log (add-on spec 3.10 and 5.14, the AuditInstance
// board): every workspace's entries and the instance-level ones in one list.
// The instance retention applies to the instance-level entries; each
// workspace keeps its own.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// InstanceScope is the workspace filter value for instance-level entries.
const InstanceScope = "instance"

// InstanceFilter narrows the instance view.
type InstanceFilter struct {
	Filter
	// Workspace is a workspace slug, or InstanceScope for the entries that
	// belong to no workspace.
	Workspace string
}

func (s *Service) instanceParams(ctx context.Context, f InstanceFilter) (store.InstanceAuditListParams, error) {
	base, err := f.params(uuid.Nil, s.now())
	if err != nil {
		return store.InstanceAuditListParams{}, err
	}
	p := store.InstanceAuditListParams{ActorType: base.ActorType, Result: base.Result, Since: base.Since, Until: base.Until, Search: base.Search,
		CursorAt: base.CursorAt, CursorID: base.CursorID, Lim: base.Lim, Actions: base.Actions}
	switch f.Workspace {
	case "":
	case InstanceScope:
		p.InstanceOnly = true
	default:
		ws, err := s.Q.GetWorkspaceBySlug(ctx, f.Workspace)
		if err != nil {
			return p, api.Invalid("query.workspace", "Unknown workspace.")
		}
		p.WorkspaceID = &ws.ID
	}
	return p, nil
}

// InstanceAuditItem is one row of the instance view.
type InstanceAuditItem struct {
	AuditEventItem
	// Workspace is the entry's workspace; empty for instance-level entries.
	Workspace     string `json:"workspace,omitempty"`
	WorkspaceSlug string `json:"workspaceSlug,omitempty"`
}

// AuditWorkspaceRef names a workspace for the workspace filter.
type AuditWorkspaceRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// InstanceAuditList is one page of the instance view.
type InstanceAuditList struct {
	Items      []InstanceAuditItem `json:"items"`
	NextCursor string              `json:"nextCursor,omitempty"`
	Workspaces []AuditWorkspaceRef `json:"workspaces"`
	// RetentionDays is the retention of the instance-level entries.
	RetentionDays int32 `json:"retentionDays"`
}

// InstanceList returns one page of every entry on the instance.
func (s *Service) InstanceList(ctx context.Context, f InstanceFilter) (InstanceAuditList, error) {
	p, err := s.instanceParams(ctx, f)
	if err != nil {
		return InstanceAuditList{}, err
	}
	p.Lim++ // one extra row tells whether there is a next page
	rows, err := s.Q.InstanceAuditList(ctx, p)
	if err != nil {
		return InstanceAuditList{}, err
	}
	out := InstanceAuditList{Items: []InstanceAuditItem{}, Workspaces: []AuditWorkspaceRef{}}
	limit := int(p.Lim) - 1
	for i, r := range rows {
		if i == limit {
			last := rows[i-1]
			out.NextCursor = encodeCursor(last.OccurredAt, last.ID)
			break
		}
		it := instanceItem(item(r.ID, r.OccurredAt, r.ActorType, r.ActorID, r.ActorName, r.ActorColor, "", r.Action, r.TargetType, r.TargetID, r.TargetName,
			r.Result, r.Reason, ipString(r.Ip), r.RequestID, r.Metadata, nil, nil), r.Metadata, ipString(r.Ip))
		it.Changes = r.HasChanges
		out.Items = append(out.Items, InstanceAuditItem{AuditEventItem: it, Workspace: r.WorkspaceName, WorkspaceSlug: r.WorkspaceSlug})
	}
	wss, err := s.Q.ListWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	for _, w := range wss {
		out.Workspaces = append(out.Workspaces, AuditWorkspaceRef{Slug: w.Slug, Name: w.Name})
	}
	st, err := s.Q.InstanceAuditStreamState(ctx)
	if err != nil {
		return out, err
	}
	out.RetentionDays = st.AuditRetentionDays
	return out, nil
}

// SignInMeta is what sign-in entries keep in their metadata.
type SignInMeta struct {
	Method      string `json:"method"`
	Provider    string `json:"provider"`
	AMR         string `json:"amr"`
	Code        string `json:"code"`
	Reference   string `json:"reference"`
	BreakGlass  bool   `json:"break_glass"`
	Workspace   string `json:"workspace"`
	Failures    int    `json:"failures"`
	MaxAttempts int    `json:"max_attempts"`
	// FailedBefore counts the failed attempts before a successful one.
	FailedBefore int `json:"failed_before"`
}

// instanceItem fills what instance-level sign-ins don't store: their
// summary ("Google Workspace SSO · amr pwd+mfa") and the auth channel.
func instanceItem(it AuditEventItem, raw json.RawMessage, ip string) AuditEventItem {
	if !strings.HasPrefix(it.Action, "signin.") {
		return it
	}
	if parseMeta(raw).Source == "" {
		it.Source = AuditSource{Channel: "auth", Detail: ip}
	}
	if parseMeta(raw).Summary != "" {
		return it
	}
	var m SignInMeta
	_ = json.Unmarshal(raw, &m)
	it.Summary = SignInSummary(it.Action, it.Reason, m)
	return it
}

// SignInSummary is the details line of a sign-in entry, as on the boards:
// "Password + TOTP · break-glass Owner", "wrong password · attempt 3 of
// 10", "Google Workspace SSO · email domain not allowed · reference auth_…".
func SignInSummary(action, reason string, m SignInMeta) string {
	parts := []string{}
	switch action {
	case "signin.succeeded":
		parts = append(parts, SignInMethod(m.Method, m.Provider))
		if m.AMR != "" {
			parts = append(parts, "amr "+m.AMR)
		}
		if m.BreakGlass {
			parts = append(parts, "break-glass Owner")
		}
		if m.Workspace != "" {
			parts = append(parts, "workspace "+m.Workspace)
		}
		if m.FailedBefore > 0 {
			parts = append(parts, fmt.Sprintf("after %d failed %s", m.FailedBefore, map[bool]string{true: "attempt", false: "attempts"}[m.FailedBefore == 1]))
		}
	case "signin.failed":
		why := reason
		if strings.HasPrefix(why, "wrong password") {
			why = "wrong password"
		}
		parts = append(parts, why)
		switch {
		case m.Failures > 0 && m.MaxAttempts > 0:
			parts = append(parts, fmt.Sprintf("attempt %d of %d", m.Failures, m.MaxAttempts))
		case m.Failures > 0:
			parts = append(parts, fmt.Sprintf("attempt %d", m.Failures))
		}
	default:
		if m.Method != "" {
			parts = append(parts, SignInMethod(m.Method, m.Provider))
		}
		why := reason
		if m.Provider != "" {
			why = strings.TrimSuffix(why, " for "+m.Provider+" SSO")
		}
		if why != "" {
			parts = append(parts, why)
		}
		if m.Reference != "" {
			parts = append(parts, "reference "+m.Reference)
		}
	}
	return strings.Join(parts, " · ")
}

// SignInMethod names a sign-in method like the boards: "Password + TOTP",
// "Google Workspace SSO", "Passkey".
func SignInMethod(method, provider string) string {
	parts := strings.Split(method, "+")
	names := make([]string, 0, len(parts))
	for i, p := range parts {
		switch p {
		case "password":
			names = append(names, "Password")
		case "totp":
			names = append(names, "TOTP")
		case "passkey":
			if i == 0 {
				names = append(names, "Passkey")
			} else {
				names = append(names, "passkey")
			}
		case "recovery":
			names = append(names, "recovery code")
		case "sso":
			if provider == "" {
				provider = "Single sign-on"
			}
			names = append(names, provider+" SSO")
		case "google":
			names = append(names, "Google")
		case "microsoft":
			names = append(names, "Microsoft")
		case "github":
			names = append(names, "GitHub")
		case "":
		default:
			names = append(names, p)
		}
	}
	if len(names) == 0 {
		return "Sign-in"
	}
	return strings.Join(names, " + ")
}

// SignInRejection says in a few words why a sign-in was refused.
func SignInRejection(code, reason string) string {
	switch code {
	case "domain_not_allowed":
		return "domain not allowed"
	case "no_account":
		return "no account"
	case "account_deactivated":
		return "deactivated"
	case "not_in_org":
		return "not in the organization"
	case "email_not_verified":
		return "email not verified"
	case "":
	default:
		return strings.ReplaceAll(code, "_", " ")
	}
	switch {
	case strings.Contains(reason, "wrong password"):
		return "wrong password"
	case strings.Contains(reason, "domain not allowed"):
		return "domain not allowed"
	case strings.Contains(reason, "passkey"):
		return "passkey failed"
	}
	if len(reason) > 40 {
		return reason[:40] + "…"
	}
	return reason
}

// InstanceAuditDetail is the detail of an entry in the instance view.
type InstanceAuditDetail struct {
	AuditEventDetail
	WorkspaceSlug string `json:"workspaceSlug,omitempty"`
	// AdminPath is the Instance admin screen of an instance-level target,
	// relative to /w/{workspace}/admin/ ("sso/0190…", "users?q=tom@acme.io").
	AdminPath string `json:"adminPath,omitempty"`
}

// InstanceGet returns any entry on the instance.
func (s *Service) InstanceGet(ctx context.Context, id uuid.UUID) (InstanceAuditDetail, error) {
	e, err := s.Q.InstanceAuditGet(ctx, id)
	if err != nil {
		return InstanceAuditDetail{}, api.MapError(err, "Audit entry")
	}
	m := parseMeta(e.Metadata)
	it := instanceItem(item(e.ID, e.OccurredAt, e.ActorType, e.ActorID, e.ActorName, e.ActorColor, "", e.Action, e.TargetType, e.TargetID, e.TargetName,
		e.Result, e.Reason, ipString(e.Ip), e.RequestID, e.Metadata, e.Before, e.After), e.Metadata, ipString(e.Ip))
	out := InstanceAuditDetail{AuditEventDetail: AuditEventDetail{
		Event: it, Workspace: e.WorkspaceName, IP: ipString(e.Ip), UserAgent: e.UserAgent, Client: Client(e.UserAgent),
		Permission: m.Perm, Before: nonEmpty(e.Before), After: nonEmpty(e.After), Diff: Diff(e.Before, e.After), Related: []string{},
	}, WorkspaceSlug: e.WorkspaceSlug}
	if out.Diff == nil {
		out.Diff = []AuditDiffLine{}
	}
	if out.Workspace == "" {
		out.Workspace = "Instance level"
	}
	out.What = m.What
	if out.What == "" {
		out.What = instanceWhat(e.Action, e.TargetType, e.TargetName)
	}
	tz := e.WorkspaceTimezone
	if m.Related != "" {
		out.Related = append(out.Related, m.Related)
	} else if rel, err := s.Q.InstanceAuditRelated(ctx, store.InstanceAuditRelatedParams{ID: e.ID, WorkspaceID: e.WorkspaceID, TargetID: e.TargetID,
		TargetType: e.TargetType, TargetName: e.TargetName, Since: e.OccurredAt.Add(-24 * time.Hour), Until: e.OccurredAt.Add(24 * time.Hour)}); err == nil {
		for _, r := range rel {
			rm := parseMeta(r.Metadata)
			line := r.Action + " by " + r.ActorName + " at " + r.OccurredAt.In(loc(tz)).Format("15:04")
			if rm.Summary != "" {
				line = rm.Summary + " (" + line + ")"
			}
			out.Related = append(out.Related, line)
		}
	}
	if e.WorkspaceSlug != "" {
		out.Link, out.LinkLabel = link(e.WorkspaceSlug, e.TargetType, e.TargetID, m.LinkPath)
	} else {
		out.AdminPath, out.LinkLabel = adminPath(e.TargetType, e.TargetID, e.TargetName)
	}
	full, _ := json.Marshal(map[string]any{
		"id": e.ID, "occurred_at": e.OccurredAt, "workspace": nilIfEmpty(e.WorkspaceSlug), "actor": map[string]any{"type": e.ActorType, "id": e.ActorID, "name": e.ActorName},
		"action": e.Action, "target": map[string]any{"type": e.TargetType, "id": e.TargetID, "name": e.TargetName},
		"result": e.Result, "reason": e.Reason, "ip": out.IP, "user_agent": e.UserAgent, "request_id": e.RequestID,
		"before": nonEmpty(e.Before), "after": nonEmpty(e.After), "metadata": nonEmpty(e.Metadata),
	})
	out.JSON = full
	return out, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// instanceWhat is the detail panel's sentence for an entry.
func instanceWhat(action, targetType, targetName string) string {
	switch action {
	case "signin.succeeded":
		return "Signed in as " + targetName
	case "signin.failed":
		return "Sign-in failed for " + targetName
	case "signin.rejected":
		return "Sign-in rejected for " + targetName
	}
	// "sso.provider.updated" reads as "sso_provider.updated".
	if n := strings.Count(action, "."); n > 1 {
		action = strings.Replace(action, ".", "_", n-1)
	}
	return Describe(action, targetType, targetName)
}

// adminPath maps an instance-level target to its Instance admin screen.
func adminPath(targetType, targetID, targetName string) (string, string) {
	switch targetType {
	case "user":
		email := targetName
		if i := strings.LastIndex(email, " · "); i >= 0 {
			email = email[i+len(" · "):]
		}
		return "users?q=" + url.QueryEscape(email), "Open user"
	case "sso_provider":
		if targetID != "" {
			return "sso/" + targetID, "Open provider"
		}
		return "sso", "Open SSO providers"
	case "social_provider", "instance_policy":
		return "authentication", "Open authentication"
	case "workspace":
		return "workspaces", "Open workspaces"
	case "instance":
		return "instance", "Open instance settings"
	}
	return "", ""
}

// InstanceExport writes the matching entries as CSV or JSON, page by page.
func (s *Service) InstanceExport(ctx context.Context, f InstanceFilter, format string, w io.Writer) error {
	p, err := s.instanceParams(ctx, f)
	if err != nil {
		return err
	}
	p.Lim = 1000
	p.CursorAt, p.CursorID = nil, nil
	bw := bufio.NewWriterSize(w, 64<<10)
	var cw *csv.Writer
	if format == "csv" {
		cw = csv.NewWriter(bw)
		_ = cw.Write([]string{"occurred_at", "workspace", "actor_type", "actor_name", "actor_id", "action", "target_type", "target_id", "target_name", "result", "reason", "summary", "ip", "user_agent", "request_id", "before", "after"})
	} else {
		_, _ = bw.WriteString("[\n")
	}
	n := 0
	for {
		rows, err := s.Q.InstanceAuditExport(ctx, store.InstanceAuditExportParams(p))
		if err != nil {
			return err
		}
		for _, e := range rows {
			row := toExport(store.AuditEvent{ID: e.ID, WorkspaceID: e.WorkspaceID, OccurredAt: e.OccurredAt, ActorType: e.ActorType, ActorID: e.ActorID, ActorName: e.ActorName,
				Action: e.Action, TargetType: e.TargetType, TargetID: e.TargetID, TargetName: e.TargetName, Result: e.Result, Reason: e.Reason, Ip: e.Ip,
				UserAgent: e.UserAgent, RequestID: e.RequestID, Before: e.Before, After: e.After, Metadata: e.Metadata})
			if cw != nil {
				it := instanceItem(AuditEventItem{Action: e.Action, Reason: e.Reason, Summary: parseMeta(e.Metadata).Summary}, e.Metadata, row.IP)
				_ = cw.Write([]string{row.OccurredAt.UTC().Format(time.RFC3339), e.WorkspaceSlug, row.ActorType, row.ActorName, row.ActorID, row.Action, row.TargetType, row.TargetID,
					row.TargetName, row.Result, row.Reason, it.Summary, row.IP, row.UserAgent, row.RequestID, string(compactOr(row.Before)), string(compactOr(row.After))})
			} else {
				b, _ := json.Marshal(struct {
					Workspace string `json:"workspace,omitempty"`
					exportRow
				}{e.WorkspaceSlug, row})
				if n > 0 {
					_, _ = bw.WriteString(",\n")
				}
				_, _ = bw.Write(b)
			}
			n++
		}
		if len(rows) < int(p.Lim) {
			break
		}
		last := rows[len(rows)-1]
		p.CursorAt, p.CursorID = &last.OccurredAt, &last.ID
		if cw != nil {
			cw.Flush()
		}
		if err := bw.Flush(); err != nil {
			return err
		}
	}
	if cw != nil {
		cw.Flush()
	} else {
		_, _ = bw.WriteString("\n]\n")
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	scope := "all workspaces and the instance"
	switch f.Workspace {
	case "":
	case InstanceScope:
		scope = "instance-level entries"
	default:
		scope = "workspace " + f.Workspace
	}
	return s.Audit.Log(ctx, audit.Event{Action: "audit.exported", TargetType: "audit_log", TargetName: "Instance audit log",
		Metadata: map[string]any{"summary": fmt.Sprintf("%d entries as %s · %s", n, map[string]string{"csv": "CSV", "json": "JSON"}[format], scope)}})
}

// InstanceWorkspaceStream is a workspace that streams its own log.
type InstanceWorkspaceStream struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Failing bool   `json:"failing"`
}

// InstanceAuditSettings is the retention and stream of the instance-level log.
type InstanceAuditSettings struct {
	RetentionDays int32             `json:"retentionDays"`
	Stream        AuditStreamStatus `json:"stream"`
	// Total is the number of instance-level entries kept.
	Total            int64                     `json:"total"`
	WorkspaceStreams []InstanceWorkspaceStream `json:"workspaceStreams"`
}

// InstanceSettings returns the instance-level retention and stream, and
// which workspaces stream their own log.
func (s *Service) InstanceSettings(ctx context.Context) (InstanceAuditSettings, error) {
	st, err := s.Q.InstanceAuditStreamState(ctx)
	if err != nil {
		return InstanceAuditSettings{}, err
	}
	out := InstanceAuditSettings{RetentionDays: st.AuditRetentionDays, WorkspaceStreams: []InstanceWorkspaceStream{},
		Stream: streamView(st.AuditStreamEnabled, st.AuditStreamUrl, st.AuditStreamSecretEnc, st.AuditStreamStatus)}
	if out.Total, err = s.Q.InstanceAuditCount(ctx); err != nil {
		return out, err
	}
	wss, err := s.Q.InstanceAuditWorkspaceStreams(ctx)
	if err != nil {
		return out, err
	}
	for _, w := range wss {
		v := streamView(true, w.AuditStreamUrl, nil, w.AuditStreamStatus)
		out.WorkspaceStreams = append(out.WorkspaceStreams, InstanceWorkspaceStream{Slug: w.Slug, Name: w.Name, Host: v.Host, Failing: v.FailingSince != nil})
	}
	return out, nil
}

func streamView(enabled bool, u *string, secret []byte, status json.RawMessage) AuditStreamStatus {
	st := AuditStreamStatus{Enabled: enabled, HasSecret: len(secret) > 0}
	if u != nil {
		st.URL = *u
		if pu, err := url.Parse(*u); err == nil {
			st.Host = pu.Hostname()
		}
	}
	var state streamState
	_ = json.Unmarshal(status, &state)
	st.LastOKAt, st.FailingSince, st.LastError, st.Delivered = state.LastOKAt, state.FailingSince, state.LastError, state.Delivered
	return st
}

// InstanceAuditSettingsResult carries a new stream secret, shown once.
type InstanceAuditSettingsResult struct {
	Settings InstanceAuditSettings `json:"settings"`
	Secret   string                `json:"secret,omitempty" doc:"New signing secret, shown only in this response"`
}

const instanceStreamAD = "instance:audit_stream_secret"

// UpdateInstanceSettings changes the instance retention or stream.
func (s *Service) UpdateInstanceSettings(ctx context.Context, in AuditSettingsInput) (InstanceAuditSettingsResult, error) {
	var res InstanceAuditSettingsResult
	st, err := s.Q.InstanceAuditStreamState(ctx)
	if err != nil {
		return res, err
	}
	before := map[string]any{"retention_days": st.AuditRetentionDays, "stream_enabled": st.AuditStreamEnabled, "stream_url": st.AuditStreamUrl}
	after := map[string]any{}
	changes := []string{}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	q := store.New(tx)
	if in.RetentionDays != nil && *in.RetentionDays != st.AuditRetentionDays {
		if err := q.InstanceAuditSetRetention(ctx, *in.RetentionDays); err != nil {
			return res, err
		}
		after["retention_days"] = *in.RetentionDays
		changes = append(changes, fmt.Sprintf("retention %d days (was %d)", *in.RetentionDays, st.AuditRetentionDays))
	}
	if in.Stream != nil {
		u := strings.TrimSpace(in.Stream.URL)
		if u == "" && st.AuditStreamUrl != nil {
			u = *st.AuditStreamUrl
		}
		if in.Stream.Enabled && u == "" {
			return res, api.Invalid("body.stream.url", "Add the URL that receives the audit stream.")
		}
		if u != "" {
			pu, err := s.policy().ValidateURL(u)
			if err != nil {
				msg := err.Error()
				return res, api.Invalid("body.stream.url", strings.ToUpper(msg[:1])+msg[1:]+".")
			}
			u = pu.String()
		}
		var enc []byte
		if in.Stream.RotateSecret || (in.Stream.Enabled && len(st.AuditStreamSecretEnc) == 0) {
			res.Secret = newSecret()
			if enc, err = s.Box.SealString(res.Secret, instanceStreamAD); err != nil {
				return res, err
			}
		}
		var urlPtr *string
		if u != "" {
			urlPtr = &u
		}
		reset := in.Stream.Enabled && (!st.AuditStreamEnabled || st.AuditStreamUrl == nil || *st.AuditStreamUrl != u)
		if err := q.InstanceAuditStreamConfigure(ctx, store.InstanceAuditStreamConfigureParams{Enabled: in.Stream.Enabled, Url: urlPtr, SecretEnc: enc, ResetCursor: reset}); err != nil {
			return res, err
		}
		after["stream_enabled"] = in.Stream.Enabled
		after["stream_url"] = u
		switch {
		case in.Stream.Enabled && !st.AuditStreamEnabled:
			changes = append(changes, "instance events streamed to "+hostOf(u))
		case !in.Stream.Enabled && st.AuditStreamEnabled:
			changes = append(changes, "instance events no longer streamed")
		case in.Stream.Enabled && reset:
			changes = append(changes, "stream moved to "+hostOf(u))
		}
		if res.Secret != "" {
			after["stream_secret"] = "rotated"
			changes = append(changes, "signing secret rotated")
		}
	}
	if len(changes) > 0 {
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "audit.settings_updated", TargetType: "audit_log", TargetName: "Instance audit log",
			Before: before, After: after, Metadata: map[string]any{"summary": strings.Join(changes, " · ")}}); err != nil {
			return res, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	res.Settings, err = s.InstanceSettings(ctx)
	return res, err
}

func hostOf(u string) string {
	if pu, err := url.Parse(u); err == nil && pu.Hostname() != "" {
		return pu.Hostname()
	}
	return u
}

// InstanceStreamArgs pushes new instance-level entries to the instance stream.
type InstanceStreamArgs struct{}

// Kind implements river.JobArgs.
func (InstanceStreamArgs) Kind() string { return "auditlog.instance_stream" }

// InsertOpts keeps one run at a time.
func (InstanceStreamArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 20 * time.Second}}
}

// InstanceStreamWorker delivers instance-level entries in order, 500 per
// request, signed like outgoing webhooks. The cursor only moves after a 2xx.
type InstanceStreamWorker struct {
	river.WorkerDefaults[InstanceStreamArgs]
	S   *Service
	Log *slog.Logger
}

// Work implements river.Worker.
func (w *InstanceStreamWorker) Work(ctx context.Context, _ *river.Job[InstanceStreamArgs]) error {
	if err := w.S.streamInstanceOnce(ctx); err != nil {
		w.Log.WarnContext(ctx, "instance audit stream", "err", err)
	}
	return nil
}

func (s *Service) streamInstanceOnce(ctx context.Context) error {
	st, err := s.Q.InstanceAuditStreamState(ctx)
	if err != nil || !st.AuditStreamEnabled || st.AuditStreamUrl == nil || *st.AuditStreamUrl == "" {
		return err
	}
	var state streamState
	_ = json.Unmarshal(st.AuditStreamStatus, &state)
	rows, err := s.Q.InstanceAuditStreamBatch(ctx, store.InstanceAuditStreamBatchParams{CursorAt: st.AuditStreamCursorAt, CursorID: st.AuditStreamCursorID, Lim: 500})
	if err != nil || len(rows) == 0 {
		return err
	}
	events := make([]exportRow, len(rows))
	for i, r := range rows {
		events[i] = toExport(r)
	}
	body, err := json.Marshal(map[string]any{"scope": "instance", "sent_at": s.now().UTC(), "events": events})
	if err != nil {
		return err
	}
	secret := ""
	if len(st.AuditStreamSecretEnc) > 0 {
		if secret, err = s.Box.OpenString(st.AuditStreamSecretEnc, instanceStreamAD); err != nil {
			return err
		}
	}
	fail := func(msg string) error {
		now := s.now()
		if state.FailingSince == nil {
			state.FailingSince = &now
		}
		state.LastError = msg
		b, _ := json.Marshal(state)
		return s.Q.InstanceAuditStreamSetStatus(ctx, b)
	}
	ts := s.now().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *st.AuditStreamUrl, bytes.NewReader(body))
	if err != nil {
		return fail("Invalid URL")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ReputeCast-Audit-Stream/"+version.Get().Version)
	req.Header.Set(webhooks.HeaderTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(webhooks.HeaderEvent, "audit.batch")
	if secret != "" {
		req.Header.Set(webhooks.HeaderSignature, webhooks.Sign(secret, ts, body))
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return fail(netguard.Describe(ctx, err))
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fail(fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)))
	}
	now := s.now()
	state.LastOKAt, state.FailingSince, state.LastError = &now, nil, ""
	state.Delivered += int64(len(rows))
	b, _ := json.Marshal(state)
	last := rows[len(rows)-1]
	id := last.ID
	return s.Q.InstanceAuditStreamAdvance(ctx, store.InstanceAuditStreamAdvanceParams{CursorAt: &last.OccurredAt, CursorID: &id, Status: b})
}

// RegisterInstanceJobs adds the instance stream worker, run every 30 seconds.
func RegisterInstanceJobs(reg *jobs.Registry, s *Service, log *slog.Logger) {
	river.AddWorker(reg.Workers, &InstanceStreamWorker{S: s, Log: log})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return InstanceStreamArgs{}, nil }, false)
}

// InstanceAuditFilterInput is the query of the instance view and its export.
type InstanceAuditFilterInput struct {
	Search    string    `query:"q" maxLength:"200" doc:"Account, target, request id or sign-in reference"`
	Workspace string    `query:"workspace" maxLength:"64" doc:"Workspace slug, or instance for instance-level entries only"`
	ActorType string    `query:"actorType" enum:"user,api_key,mcp_agent,system"`
	Result    string    `query:"result" enum:"allowed,denied,error"`
	Range     string    `query:"range" enum:"24h,7d,30d,90d,all" default:"30d"`
	Since     time.Time `query:"since" doc:"Overrides range"`
	Until     time.Time `query:"until"`
	// Category narrows to a group of actions, as in the workspace view
	// (agency: the agency's actions across the instance).
	Category string `query:"category" enum:"sending,routing,audience,team,settings,integrations,transactional,scoring,digests,polls,ai,earn,verification,placement,website,agency"`
}

func (in *InstanceAuditFilterInput) filter() InstanceFilter {
	f := InstanceFilter{Filter: Filter{Search: in.Search, ActorType: in.ActorType, Result: in.Result, Range: in.Range, Category: in.Category}, Workspace: in.Workspace}
	if !in.Since.IsZero() {
		since := in.Since
		f.Since = &since
	}
	if !in.Until.IsZero() {
		until := in.Until
		f.Until = &until
	}
	return f
}

// RegisterInstance adds the instance-level audit operations (instance admins only).
func RegisterInstance(h huma.API, s *Service) {
	tags := []string{"Instance admin · Audit log"}
	admin := api.InstanceAdmin
	api.Get(h, api.Op{ID: "list-instance-audit-events", Path: "/admin/audit", Summary: "Every audit entry on the instance, newest first", Tags: tags, Access: admin},
		func(ctx context.Context, in *struct {
			InstanceAuditFilterInput
			Cursor string `query:"cursor" maxLength:"200"`
			Limit  int    `query:"limit" minimum:"1" maximum:"200" default:"50"`
		}) (*api.Body[InstanceAuditList], error) {
			f := in.filter()
			f.Cursor, f.Limit = in.Cursor, in.Limit
			out, err := s.InstanceList(ctx, f)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-instance-audit-event", Path: "/admin/audit/events/{eventId}", Summary: "One audit entry of any workspace or of the instance", Tags: tags, Access: admin},
		func(ctx context.Context, in *struct {
			EventID string `path:"eventId" format:"uuid"`
		}) (*api.Body[InstanceAuditDetail], error) {
			id, err := uuid.Parse(in.EventID)
			if err != nil {
				return nil, api.NotFound("Audit entry")
			}
			out, err := s.InstanceGet(ctx, id)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "export-instance-audit-events", Path: "/admin/audit/export", Summary: "Export matching entries as CSV or JSON (streamed)", Tags: tags, Access: admin},
		func(ctx context.Context, in *struct {
			InstanceAuditFilterInput
			Format string `query:"format" enum:"csv,json" default:"csv"`
		}) (*huma.StreamResponse, error) {
			f := in.filter()
			if _, err := s.instanceParams(ctx, f); err != nil {
				return nil, err
			}
			format := in.Format
			return &huma.StreamResponse{Body: func(hctx huma.Context) {
				ext, ct := "csv", "text/csv; charset=utf-8"
				if format == "json" {
					ext, ct = "json", "application/json"
				}
				hctx.SetHeader("Content-Type", ct)
				hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-instance-%s.%s"`, time.Now().UTC().Format("2006-01-02"), ext))
				hctx.SetHeader("Cache-Control", "no-store")
				hctx.SetStatus(200)
				_ = s.InstanceExport(ctx, f, format, hctx.BodyWriter())
			}}, nil
		})

	api.Get(h, api.Op{ID: "get-instance-audit-settings", Path: "/admin/audit/settings", Summary: "Retention and stream of the instance-level entries", Tags: tags, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstanceAuditSettings], error) {
			out, err := s.InstanceSettings(ctx)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-instance-audit-settings", Path: "/admin/audit/settings", Summary: "Change the instance retention or stream instance events to a webhook", Tags: tags, Access: admin},
		func(ctx context.Context, in *struct{ Body AuditSettingsInput }) (*api.Body[InstanceAuditSettingsResult], error) {
			out, err := s.UpdateInstanceSettings(ctx, in.Body)
			return api.OK(out), err
		})
}
