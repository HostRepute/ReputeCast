package automations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// TickArgs enrolls contacts from trigger events and moves due runs.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "automations.tick" }

func tickOpts() *river.InsertOpts {
	return &river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 2 * time.Second}}
}

// TickWorker runs Dispatch then Tick.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error {
	if _, err := w.S.Dispatch(ctx); err != nil {
		return fmt.Errorf("dispatch: %w", err)
	}
	if _, err := w.S.Tick(ctx); err != nil {
		return fmt.Errorf("tick: %w", err)
	}
	return nil
}

// Timeout keeps one tick from overlapping the next for long.
func (w *TickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return time.Minute }

// ScanArgs runs the periodic scan (goals, branch wake-ups, date triggers…).
type ScanArgs struct{}

// Kind implements river.JobArgs.
func (ScanArgs) Kind() string { return "automations.scan" }

// ScanWorker runs Scan.
type ScanWorker struct {
	river.WorkerDefaults[ScanArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ScanWorker) Work(ctx context.Context, _ *river.Job[ScanArgs]) error { return w.S.Scan(ctx) }

// WebhookArgs calls one webhook step for one run.
type WebhookArgs struct {
	RunID        uuid.UUID `json:"run_id" river:"unique"`
	AutomationID uuid.UUID `json:"automation_id"`
	Version      int       `json:"version"`
	NodeID       string    `json:"node_id" river:"unique"`
}

// Kind implements river.JobArgs.
func (WebhookArgs) Kind() string { return "automations.webhook" }

// webhookAttempts is how many times a call is tried before the contact
// moves on without it.
const webhookAttempts = 3

func webhookOpts() *river.InsertOpts {
	return &river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: webhookAttempts,
		UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// WebhookWorker calls webhook steps.
type WebhookWorker struct {
	river.WorkerDefaults[WebhookArgs]
	S *Service
}

// Work implements river.Worker.
func (w *WebhookWorker) Work(ctx context.Context, job *river.Job[WebhookArgs]) error {
	return w.S.RunWebhook(ctx, job.Args, job.Attempt, job.MaxAttempts)
}

// NextRetry backs off 30 s, 2 min, 8 min.
func (w *WebhookWorker) NextRetry(job *river.Job[WebhookArgs]) time.Time {
	return time.Now().Add(time.Duration(30<<(2*(job.Attempt-1))) * time.Second)
}

// RegisterJobs adds the engine's workers and schedules to the job registry.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	river.AddWorker(reg.Workers, &ScanWorker{S: s})
	river.AddWorker(reg.Workers, &WebhookWorker{S: s})
	reg.Every(10*time.Second, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, tickOpts() }, true)
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) {
		return ScanArgs{}, &river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Second}}
	}, true)
}

// webhookPayload is the JSON body a webhook step sends.
type webhookPayload struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	OccurredAt time.Time         `json:"occurredAt"`
	Automation webhookAutomation `json:"automation"`
	Step       webhookStep       `json:"step"`
	Contact    webhookContact    `json:"contact"`
}

type webhookAutomation struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type webhookStep struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type webhookContact struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	FirstName string         `json:"firstName"`
	LastName  string         `json:"lastName"`
	Tags      []string       `json:"tags"`
	Fields    map[string]any `json:"fields"`
}

// SignatureHeader carries the HMAC-SHA256 signature of a webhook body:
// "t=<unix seconds>,v1=<hex of HMAC(secret, t + "." + body)>".
const SignatureHeader = "X-ReputeCast-Signature"

// Sign computes the signature header value for a body.
func Sign(secret []byte, t time.Time, body []byte) string {
	ts := strconv.FormatInt(t.Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) httpClient() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return NewWebhookClient()
}

// RunWebhook makes one webhook call for a run waiting at a webhook step,
// logs it, and lets the contact move on: a failing webhook never blocks the
// flow. Transient failures (network errors, 429, 5xx) are retried by River
// up to webhookAttempts times first.
func (s *Service) RunWebhook(ctx context.Context, args WebhookArgs, attempt, maxAttempts int) error {
	run, err := s.Q.GetAutomationRunForUpdate(ctx, args.RunID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	st := parseState(run.State)
	if run.Status != "waiting" || run.NodeID != args.NodeID || st.Wait != "webhook" || st.Done {
		return nil // already handled
	}
	v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: run.AutomationID, Version: run.Version})
	if err != nil {
		return err
	}
	g := s.graphOf(v.Graph)
	node := g.Node(args.NodeID)
	if node == nil || node.Data.Webhook == nil {
		return s.finishWebhook(ctx, run.ID, args, 0, "the step was removed", false)
	}
	cfg := *node.Data.Webhook
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: run.WorkspaceID, ID: run.AutomationID})
	if err != nil {
		return err
	}
	c, err := s.Q.AutomationContact(ctx, run.ContactID)
	if err != nil {
		return err
	}
	fields := map[string]any{}
	_ = json.Unmarshal(c.Attributes, &fields)
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	now := s.now()
	body, _ := json.Marshal(webhookPayload{
		Type: "automation.step", ID: uuid.Must(uuid.NewV7()).String(), OccurredAt: now.UTC(),
		Automation: webhookAutomation{ID: a.ID.String(), Name: a.Name, Version: int(run.Version)},
		Step:       webhookStep{ID: node.ID, Name: NodeTitle(g, *node, nil)},
		Contact:    webhookContact{ID: c.ID.String(), Email: c.Email, FirstName: c.FirstName, LastName: c.LastName, Tags: tags, Fields: fields},
	})
	method := cfg.Method
	if method == "" {
		method = http.MethodPost
	}
	status, callErr, dur := 0, "", time.Duration(0)
	// The default client re-checks every address it dials; the URL check
	// here gives a clearer error for literal private hosts. A client set on
	// the service (tests) is trusted as-is.
	if cfg.Demo {
		// Demo data never leaves the instance: the step is recorded as
		// answered and nothing is sent to its fictional address.
		status, dur = http.StatusAccepted, 320*time.Millisecond
	} else if err := checkWebhookURL(cfg.URL); err != nil && s.HTTP == nil {
		callErr = "blocked: " + err.Error()
	} else {
		req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bytes.NewReader(body))
		if err != nil {
			callErr = err.Error()
		} else {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "ReputeCast-Webhooks/1.0")
			req.Header.Set("X-ReputeCast-Event", "automation.step")
			if enc, err := s.Q.GetAutomationStepSecret(ctx, store.GetAutomationStepSecretParams{AutomationID: a.ID, NodeID: node.ID}); err == nil {
				secret, err := s.Box.Open(enc, secretAD(a.ID, node.ID))
				if err == nil {
					req.Header.Set(SignatureHeader, Sign(secret, now, body))
				}
			}
			start := time.Now()
			resp, err := s.httpClient().Do(req)
			dur = time.Since(start)
			if err != nil {
				callErr = cleanErr(err)
			} else {
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
				_ = resp.Body.Close()
				status = resp.StatusCode
			}
		}
	}
	ok := callErr == "" && status >= 200 && status < 300
	if err := s.Q.InsertAutomationWebhookCall(ctx, store.InsertAutomationWebhookCallParams{
		ID: uuid.Must(uuid.NewV7()), AutomationID: a.ID, Version: run.Version, NodeID: node.ID, RunID: &run.ID,
		Url: cfg.URL, StatusCode: int32(status), Error: callErr, DurationMs: int32(dur.Milliseconds()), Attempt: int32(attempt), CalledAt: now,
	}); err != nil {
		return err
	}
	transient := !ok && !strings.HasPrefix(callErr, "blocked:") && (status == 0 || status == http.StatusTooManyRequests || status >= 500)
	if transient && attempt < maxAttempts {
		return fmt.Errorf("webhook %s: %s", hostOf(cfg.URL), describeFailure(status, callErr))
	}
	if !ok {
		detail, _ := json.Marshal(map[string]any{"url": cfg.URL, "status": status, "error": callErr, "name": cfg.Name})
		if err := s.Q.OpenAutomationAlert(ctx, store.OpenAutomationAlertParams{ID: uuid.Must(uuid.NewV7()), AutomationID: a.ID, Kind: "step_failing", NodeID: node.ID, Detail: detail, Failures: 1, At: now}); err != nil {
			return err
		}
	} else if _, err := s.Q.ResolveAutomationAlert(ctx, store.ResolveAutomationAlertParams{AutomationID: a.ID, Kind: "step_failing", NodeID: node.ID}); err != nil {
		return err
	}
	return s.finishWebhook(ctx, run.ID, args, status, callErr, ok)
}

func describeFailure(status int, err string) string {
	if err != "" {
		return err
	}
	return fmt.Sprintf("HTTP %d", status)
}

// cleanErr keeps network errors short and free of request details.
func cleanErr(err error) string {
	msg := err.Error()
	if errors.Is(err, ErrBlockedAddress) || strings.Contains(msg, ErrBlockedAddress.Error()) {
		return "blocked: " + ErrBlockedAddress.Error()
	}
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		msg = msg[i+2:]
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}

// finishWebhook records the outcome on the step and lets the run continue.
func (s *Service) finishWebhook(ctx context.Context, runID uuid.UUID, args WebhookArgs, status int, callErr string, ok bool) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		run, err := q.GetAutomationRunForUpdate(ctx, runID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		st := parseState(run.State)
		if run.Status != "waiting" || run.NodeID != args.NodeID || st.Wait != "webhook" || st.Done {
			return nil
		}
		now := s.now()
		b := statsBuf{}
		if !ok {
			b.at(run.AutomationID, run.Version, run.NodeID, now).Failed++
		}
		st.Done = true
		if err := q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{ID: run.ID, Version: run.Version, NodeID: run.NodeID, Status: "active", State: st.json(), NodeEnteredAt: run.NodeEnteredAt, NextAt: &now}); err != nil {
			return err
		}
		if err := b.flush(ctx, q); err != nil {
			return err
		}
		s.kick(ctx, tx)
		return nil
	})
	_ = status
	_ = callErr
	return err
}
