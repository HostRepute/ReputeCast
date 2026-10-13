package auditlog

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

func newSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "whsec_" + base64.RawURLEncoding.EncodeToString(b)
}

// exportRow is one entry in an export.
type exportRow struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	ActorType  string          `json:"actor_type"`
	ActorID    string          `json:"actor_id,omitempty"`
	ActorName  string          `json:"actor_name"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	TargetName string          `json:"target_name"`
	Result     string          `json:"result"`
	Reason     string          `json:"reason,omitempty"`
	IP         string          `json:"ip,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

func toExport(e store.AuditEvent) exportRow {
	r := exportRow{ID: e.ID.String(), OccurredAt: e.OccurredAt, ActorType: e.ActorType, ActorName: e.ActorName, Action: e.Action, TargetType: e.TargetType,
		TargetID: e.TargetID, TargetName: e.TargetName, Result: e.Result, Reason: e.Reason, IP: ipString(e.Ip), UserAgent: e.UserAgent, RequestID: e.RequestID,
		Before: nonEmpty(e.Before), After: nonEmpty(e.After), Metadata: nonEmpty(e.Metadata)}
	if e.ActorID != nil {
		r.ActorID = e.ActorID.String()
	}
	return r
}

// Export writes the entries matching the filter as CSV or JSON, page by page,
// so large logs stream without being held in memory. The export itself is
// recorded in the log.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, f Filter, format string, w io.Writer) error {
	p, err := f.params(ws.ID, s.now())
	if err != nil {
		return err
	}
	p.Lim = 1000
	p.CursorAt, p.CursorID = nil, nil
	bw := bufio.NewWriterSize(w, 64<<10)
	var cw *csv.Writer
	if format == "csv" {
		cw = csv.NewWriter(bw)
		_ = cw.Write([]string{"occurred_at", "actor_type", "actor_name", "actor_id", "action", "target_type", "target_id", "target_name", "result", "reason", "summary", "ip", "user_agent", "request_id", "before", "after"})
	} else {
		_, _ = bw.WriteString("[\n")
	}
	n := 0
	for {
		rows, err := s.Q.AuditLogExport(ctx, store.AuditLogExportParams(p))
		if err != nil {
			return err
		}
		for _, e := range rows {
			row := toExport(store.AuditEvent{ID: e.ID, WorkspaceID: e.WorkspaceID, OccurredAt: e.OccurredAt, ActorType: e.ActorType, ActorID: e.ActorID, ActorName: e.ActorName,
				Action: e.Action, TargetType: e.TargetType, TargetID: e.TargetID, TargetName: e.TargetName, Result: e.Result, Reason: e.Reason, Ip: e.Ip,
				UserAgent: e.UserAgent, RequestID: e.RequestID, Before: e.Before, After: e.After, Metadata: e.Metadata})
			if cw != nil {
				m := parseMeta(e.Metadata)
				_ = cw.Write([]string{row.OccurredAt.UTC().Format(time.RFC3339), row.ActorType, row.ActorName, row.ActorID, row.Action, row.TargetType, row.TargetID,
					row.TargetName, row.Result, row.Reason, m.Summary, row.IP, row.UserAgent, row.RequestID, string(compactOr(row.Before)), string(compactOr(row.After))})
			} else {
				b, _ := json.Marshal(row)
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
	return s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "audit.exported", TargetType: "audit_log", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"summary": fmt.Sprintf("%d entries as %s", n, map[string]string{"csv": "CSV", "json": "JSON"}[format]), "permission": "audit:view"}})
}

func compactOr(v json.RawMessage) []byte {
	if len(v) == 0 {
		return nil
	}
	var b bytes.Buffer
	if json.Compact(&b, v) != nil {
		return v
	}
	return b.Bytes()
}

// StreamArgs is the periodic job that pushes new entries to each
// workspace's audit stream endpoint.
type StreamArgs struct{}

// Kind implements river.JobArgs.
func (StreamArgs) Kind() string { return "auditlog.stream" }

// InsertOpts keeps one run at a time.
func (StreamArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 20 * time.Second}}
}

// StreamWorker delivers audit entries in order, 500 per request, signed like
// outgoing webhooks. The cursor only moves after a 2xx.
type StreamWorker struct {
	river.WorkerDefaults[StreamArgs]
	S   *Service
	Log *slog.Logger
}

// Work implements river.Worker.
func (w *StreamWorker) Work(ctx context.Context, _ *river.Job[StreamArgs]) error {
	wss, err := w.S.Q.AuditStreamWorkspaces(ctx)
	if err != nil {
		return err
	}
	for _, ws := range wss {
		if err := w.S.streamOnce(ctx, ws); err != nil {
			w.Log.WarnContext(ctx, "audit stream", "workspace", ws.Slug, "err", err)
		}
	}
	return nil
}

func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return s.policy().Client(15 * time.Second)
}

// streamOnce sends the next batch for one workspace.
func (s *Service) streamOnce(ctx context.Context, ws store.AuditStreamWorkspacesRow) error {
	var state streamState
	_ = json.Unmarshal(ws.AuditStreamStatus, &state)
	rows, err := s.Q.AuditStreamBatch(ctx, store.AuditStreamBatchParams{WorkspaceID: &ws.ID, CursorAt: ws.AuditStreamCursorAt, CursorID: ws.AuditStreamCursorID, Lim: 500})
	if err != nil || len(rows) == 0 {
		return err
	}
	events := make([]exportRow, len(rows))
	for i, r := range rows {
		events[i] = toExport(r)
	}
	body, err := json.Marshal(map[string]any{"workspace": ws.Slug, "sent_at": s.now().UTC(), "events": events})
	if err != nil {
		return err
	}
	secret := ""
	if len(ws.AuditStreamSecretEnc) > 0 {
		if secret, err = s.Box.OpenString(ws.AuditStreamSecretEnc, streamAD(ws.ID)); err != nil {
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
		return s.Q.AuditStreamSetStatus(ctx, store.AuditStreamSetStatusParams{ID: ws.ID, Status: b})
	}
	ts := s.now().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *ws.AuditStreamUrl, bytes.NewReader(body))
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
	return s.Q.AuditStreamAdvance(ctx, store.AuditStreamAdvanceParams{ID: ws.ID, CursorAt: &last.OccurredAt, CursorID: &id, Status: b})
}

// RegisterJobs adds the stream worker, run every 30 seconds.
func RegisterJobs(reg *jobs.Registry, s *Service, log *slog.Logger) {
	river.AddWorker(reg.Workers, &StreamWorker{S: s, Log: log})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return StreamArgs{}, nil }, false)
}

var _ = uuid.Nil
