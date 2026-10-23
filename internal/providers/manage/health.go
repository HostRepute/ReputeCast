package manage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// Health thresholds.
const (
	// SilentAfter is how long webhooks may be quiet while mail goes out.
	SilentAfter = 6 * time.Hour
	// CheckEvery is how often a healthy provider's API is checked.
	CheckEvery = 5 * time.Minute
	// OutageCalls and OutageRate: at least this many calls in three
	// minutes with this share failing marks the provider down.
	OutageCalls = 5
	OutageRate  = 0.5
	// SelfTestHeader marks the webhook endpoint self-test.
	SelfTestHeader = "X-ReputeCast-Self-Test"
	// AwaitingRows is how many messages without an event the webhooks-silent
	// view lists (newest first) before the last one that had its event.
	AwaitingRows = 3
)

// IsDemo reports whether a provider is sample data (config "demo": true):
// jobs never call its API, so the seeded state stays as designed.
func IsDemo(p store.Provider) bool {
	var c struct {
		Demo bool `json:"demo"`
	}
	_ = json.Unmarshal(p.Config, &c)
	return c.Demo
}

// opName is the call a health check makes, for "Last successful call".
func opName(adapter string) string {
	switch adapter {
	case "ses":
		return "GetAccount"
	case "postmark":
		return "GET /server"
	case "postal":
		return "POST /api/v1/messages/message"
	case "smtp":
		return "SMTP AUTH"
	case "resend":
		return "GET /domains"
	case "sendgrid":
		return "GET /v3/user/credits"
	case "mailgun":
		return "GET /v5/accounts/limit/custom/monthly"
	case "sparkpost":
		return "GET /api/v1/account"
	case "mailjet":
		return "GET /v3/REST/apikey"
	case "brevo":
		return "GET /v3/account"
	case "azureacs":
		return "GET /emails/operations/{id}"
	case "httpgeneric":
		return "credential check"
	}
	return "health check"
}

// HealthArgs is the periodic provider health job.
type HealthArgs struct{}

// Kind implements river.JobArgs.
func (HealthArgs) Kind() string { return "providers.health" }

// InsertOpts keeps one run per minute.
func (HealthArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// HealthWorker checks every connected provider.
type HealthWorker struct {
	river.WorkerDefaults[HealthArgs]
	S *Service
}

// Timeout bounds one sweep.
func (w *HealthWorker) Timeout(*river.Job[HealthArgs]) time.Duration { return 4 * time.Minute }

// Work implements river.Worker.
func (w *HealthWorker) Work(ctx context.Context, _ *river.Job[HealthArgs]) error {
	provs, err := w.S.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	now := w.S.now()
	var errs []error
	for _, p := range provs {
		if IsDemo(p) {
			continue
		}
		probe := p.LastCheckedAt == nil || now.Sub(*p.LastCheckedAt) >= CheckEvery || p.Status == state.Down || p.Status == state.Degraded
		if _, err := w.S.Check(ctx, p, probe); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
		}
	}
	if _, err := w.S.Q.ProvPurgeCallWindows(ctx, now.Add(-48*time.Hour)); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// TrackingArgs is the daily provider-side tracking check.
type TrackingArgs struct{}

// Kind implements river.JobArgs.
func (TrackingArgs) Kind() string { return "providers.tracking_check" }

// InsertOpts runs one check at a time.
func (TrackingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// TrackingWorker refreshes every provider's tracking state.
type TrackingWorker struct {
	river.WorkerDefaults[TrackingArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TrackingWorker) Work(ctx context.Context, _ *river.Job[TrackingArgs]) error {
	provs, err := w.S.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	for _, p := range provs {
		if IsDemo(p) {
			continue
		}
		_, _ = w.S.refreshTracking(ctx, p)
	}
	return nil
}

// Check runs a provider's health check: an API probe (when probe is set),
// the three-minute error rate, outage and recovery transitions, webhook
// freshness and the follow-ups (failover, held broadcasts, incidents,
// notifications, audit).
func (s *Service) Check(ctx context.Context, p store.Provider, probe bool) (store.Provider, error) {
	now := s.now()
	var callErr error
	var quota providers.Quota
	var adapter providers.Provider
	var reputation *state.Reputation
	probed := false
	if probe {
		var err error
		adapter, err = s.Factory.Open(ctx, p)
		if err != nil {
			ce := &providers.CallError{Op: "configuration", Message: "The stored settings can't be read: " + err.Error(), Auth: true}
			ce.Text = mErrorSettings.Keep(i18n.Args{"error": err.Error()})
			callErr = ce
		} else {
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			switch a := adapter.(type) {
			case providers.QuotaReporter:
				quota, callErr = a.Quota(cctx)
				probed = true
			case providers.Verifier:
				quota, callErr = a.Verify(cctx)
				probed = true
			}
			// The provider's own bounce and complaint rates, when it reports
			// them; a failure here doesn't make the provider unhealthy.
			if rr, ok := adapter.(providers.ReputationReporter); ok && callErr == nil {
				if r, err := rr.Reputation(cctx); err == nil {
					reputation = &state.Reputation{Bounce: r.Bounce, Complaint: r.Complaint, Window: r.Window, CheckedAt: now}
				}
			}
			cancel()
		}
		if probed || callErr != nil {
			if err := RecordCall(ctx, s.Pool, p, opName(p.Adapter), callErr, 0, now); err != nil {
				return p, err
			}
		}
	}
	// Re-read: RecordCall and the sending pipeline also write health.
	fresh, err := s.Q.ProvGetAny(ctx, p.ID)
	if err != nil {
		return p, err
	}
	p = fresh
	h := state.HealthOf(p)
	before := h
	if probed || callErr != nil {
		if callErr == nil {
			h.ConsecutiveFailures = 0
			t := now
			h.LastSuccessAt, h.LastSuccessOp = &t, opName(p.Adapter)
			if quota.Kind != "" {
				h.Quota = &state.Quota{Kind: quota.Kind, Max: quota.Max, Used: quota.Used}
			}
			if quota.MaxRate > 0 {
				if h.Rate == nil {
					h.Rate = &state.Rate{}
				}
				h.Rate.Max = quota.MaxRate
			}
			if quota.Account != "" {
				h.Account, h.AccountText = quota.Account, quota.AccountText
			}
			h.Sandbox = quota.Sandbox
			h.VDM = quota.VDM
			if reputation != nil {
				h.Reputation = reputation
			}
		} else {
			h.ConsecutiveFailures++
			rec := CallRecordOf(opName(p.Adapter), callErr, now)
			h.LastError = &rec
		}
	}
	wins, err := s.Q.ProvCallWindowTotals(ctx, store.ProvCallWindowTotalsParams{ProviderIds: []uuid.UUID{p.ID}, Since: now.Add(-3 * time.Minute)})
	if err != nil {
		return p, err
	}
	var calls, failed int64
	if len(wins) == 1 {
		calls, failed = wins[0].Calls, wins[0].Errors
	}
	auth := callErr != nil && errors.Is(callErr, providers.ErrUnauthorized)
	failing := callErr != nil && h.ConsecutiveFailures >= 2 || calls >= OutageCalls && float64(failed)/float64(calls) >= OutageRate
	var went, recovered bool
	switch {
	case auth:
		if h.Down == nil || h.Down.Reason != "auth" {
			h.Down = &state.DownState{Since: now, Reason: "auth"}
			went = true
		}
		h.Down.RecoveringSince = nil
	case failing:
		if h.Down == nil {
			h.Down = &state.DownState{Since: streakStart(h, now), Reason: "outage"}
			went = true
		}
		h.Down.RecoveringSince = nil
	case h.Down != nil && (!probe || callErr == nil) && (calls == 0 || float64(failed)/float64(calls) < 0.1):
		if h.Down.RecoveringSince == nil {
			t := now
			h.Down.RecoveringSince = &t
		} else if now.Sub(*h.Down.RecoveringSince) >= SwitchBackMinutes*time.Minute || h.Down.Reason == "auth" {
			recovered = true
		}
	}
	if went && h.Down.Reason == "outage" && adapter != nil {
		if sr, ok := adapter.(providers.StatusReporter); ok {
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if st, err := sr.ServiceStatus(cctx); err == nil {
				h.Down.ProviderStatus = &state.ProviderStatus{Incident: st.Incident, Summary: st.Summary, SummaryText: st.SummaryText, CheckedAt: now}
			}
			cancel()
		}
	}
	downState := h.Down
	if recovered {
		h.Down = nil
	}
	// Webhook freshness, only while the provider is up.
	silentNow, resumed := false, false
	caps := state.Caps(p)
	if h.Down == nil && caps.Webhooks != "" && caps.Webhooks != "Bounce mailbox" && p.LastEventAt != nil {
		if h.Webhooks != nil && h.Webhooks.SilentSince != nil && p.LastEventAt.After(*h.Webhooks.SilentSince) {
			resumed = true
			h.Webhooks = &state.WebhookState{}
		} else if now.Sub(*p.LastEventAt) >= SilentAfter {
			rows, err := s.Q.ProvSentSince(ctx, store.ProvSentSinceParams{ProviderIds: []uuid.UUID{p.ID}, Since: p.LastEventAt})
			if err != nil {
				return p, err
			}
			if len(rows) == 1 && rows[0].SinceLastEvent > 0 {
				if h.Webhooks == nil || h.Webhooks.SilentSince == nil {
					silentNow = true
					t := *p.LastEventAt
					h.Webhooks = &state.WebhookState{SilentSince: &t}
				}
				if h.Webhooks.CheckedAt == nil || now.Sub(*h.Webhooks.CheckedAt) >= time.Hour {
					s.diagnose(ctx, p, adapter, callErr, &h)
				}
			}
		}
	}
	status := s.baseStatus(ctx, p, h)
	detail := ""
	switch {
	case h.Down != nil:
		status = state.Down
		detail = state.DetailDown
		if h.Down.Reason == "auth" {
			detail = state.DetailDownAuth
		}
	case h.Webhooks != nil && h.Webhooks.SilentSince != nil:
		status, detail = state.Degraded, state.DetailWebhooksSilent
	}
	if probed && callErr == nil && adapter != nil {
		snap := state.Snapshot{Capabilities: adapter.Capabilities(), BroadcastStream: caps.BroadcastStream}
		if b, err := json.Marshal(snap); err == nil {
			_ = s.Q.ProvUpdateCapabilities(ctx, store.ProvUpdateCapabilitiesParams{ID: p.ID, Capabilities: b})
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		hb, _ := json.Marshal(h)
		// Keep the recent errors RecordCall appended.
		if _, err := tx.Exec(ctx, `UPDATE providers SET status = $2, status_detail = $3, last_checked_at = $4,
			health = $5::jsonb || jsonb_build_object('recentErrors', coalesce(health->'recentErrors', '[]'::jsonb))
			WHERE id = $1`, p.ID, status, detail, now, hb); err != nil {
			return err
		}
		ws, err := q.GetWorkspace(ctx, p.WorkspaceID)
		if err != nil {
			return err
		}
		switch {
		case went:
			if err := s.onDown(ctx, tx, ws, p, h, now); err != nil {
				return err
			}
		case recovered:
			if err := s.onRecovered(ctx, tx, ws, p, downState, now); err != nil {
				return err
			}
		}
		if silentNow {
			if err := s.onSilent(ctx, tx, ws, p, h, now); err != nil {
				return err
			}
		}
		if resumed {
			if err := s.onResumed(ctx, tx, ws, p, before, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return p, err
	}
	return s.Q.ProvGetAny(ctx, p.ID)
}

// streakStart is when the current run of failures began.
func streakStart(h state.Health, now time.Time) time.Time {
	start := now
	for _, e := range h.RecentErrors {
		if now.Sub(e.At) > 15*time.Minute {
			break
		}
		start = e.At
	}
	return start
}

// baseStatus is a provider's status when nothing is wrong.
func (s *Service) baseStatus(ctx context.Context, p store.Provider, h state.Health) string {
	if plan, err := s.Q.WarmupActiveForProvider(ctx, p.ID); err == nil && plan.Status != "done" {
		return state.WarmingUp
	}
	if h.Sandbox {
		return state.Sandbox
	}
	return state.Healthy
}

// diagnose runs the webhooks-silent checks.
func (s *Service) diagnose(ctx context.Context, p store.Provider, adapter providers.Provider, probeErr error, h *state.Health) {
	now := s.now()
	yes, no := true, false
	checks := []state.WebhookCheck{}
	// "GetAccount responds", "AUTH accepted" (SMTP)
	api := state.WebhookCheck{Key: "api", At: &now}
	if p.Adapter == "smtp" {
		api.Detail, api.DetailText = keep(mCheckAuthAccepted, nil)
	} else {
		op := map[string]string{"postal": "/api/v1/messages", "ses": "GetAccount", "postmark": "GET /server"}[p.Adapter]
		if op == "" {
			op = opName(p.Adapter)
		}
		api.Detail, api.DetailText = keep(mCheckResponds, i18n.Args{"op": op})
	}
	if probeErr == nil {
		api.OK = &yes
	} else {
		api.OK = &no
		api.Detail, api.DetailText = probeErr.Error(), i18n.Text{}
	}
	checks = append(checks, api)
	url := WebhookURL(s.URLs, p)
	reg := state.WebhookCheck{Key: "registered", URL: url}
	queued := int64(-1)
	if wi, ok := adapter.(providers.WebhookInspector); ok && adapter != nil {
		if st, err := wi.InspectWebhook(ctx, url); err == nil {
			reg.OK = &st.Registered
			reg.ID = st.ID
			if st.Queued >= 0 {
				queued = st.Queued
			}
		}
	}
	checks = append(checks, reg)
	code, err := s.selfTest(ctx, p)
	end := state.WebhookCheck{Key: "endpoint"}
	if err == nil && code == http.StatusOK {
		end.OK = &yes
		end.Detail, end.DetailText = keep(mCheckSelfTestOK, nil)
	} else {
		end.OK = &no
		if err != nil {
			end.Detail = err.Error()
		} else {
			end.Detail, end.DetailText = keep(mCheckSelfTest, i18n.Args{"status": strconv.Itoa(code)})
		}
	}
	checks = append(checks, end)
	var got int64
	if h.Webhooks != nil && h.Webhooks.SilentSince != nil {
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1 AND received_at > $2 AND source = 'webhook'`, p.ID, *h.Webhooks.SilentSince).Scan(&got)
	}
	logCheck := state.WebhookCheck{Key: "provider_log", Count: &got}
	if queued >= 0 {
		logCheck.Queued = &queued
	}
	checks = append(checks, logCheck)
	var badSigs int64
	if h.Webhooks != nil && h.Webhooks.SilentSince != nil {
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1 AND received_at > $2 AND NOT signature_ok`, p.ID, *h.Webhooks.SilentSince).Scan(&badSigs)
	}
	causes := []string{}
	selfHosted := false
	if d, ok := providers.Lookup(p.Adapter); ok {
		selfHosted = d.Kind == "self_hosted"
	}
	switch {
	case badSigs > 0:
		causes = append(causes, "secret_rotated", "firewall")
	case end.OK != nil && !*end.OK:
		causes = append(causes, "endpoint_unreachable", "firewall", "secret_rotated")
	case reg.OK != nil && !*reg.OK:
		causes = append(causes, "webhook_removed", "firewall", "secret_rotated")
	case selfHosted:
		causes = append(causes, "worker_stuck", "firewall", "secret_rotated")
	default:
		causes = append(causes, "webhook_removed", "firewall", "secret_rotated")
	}
	if h.Webhooks == nil {
		h.Webhooks = &state.WebhookState{}
	}
	h.Webhooks.Checks, h.Webhooks.Causes, h.Webhooks.CheckedAt = checks, causes, &now
}

// SelfTestSignature authenticates a self-test request for a provider.
func SelfTestSignature(token string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = fmt.Fprintf(mac, "self-test:%d", ts)
	return strconv.FormatInt(ts, 10) + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifySelfTest checks a self-test header (five-minute window).
func VerifySelfTest(token, header string, now time.Time) bool {
	ts, _, ok := strings.Cut(header, ".")
	if !ok {
		return false
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || now.Sub(time.Unix(n, 0)).Abs() > 5*time.Minute {
		return false
	}
	return hmac.Equal([]byte(SelfTestSignature(token, n)), []byte(header))
}

// selfTest posts to the provider's public webhook URL, through whatever
// proxy and TLS the instance sits behind, and expects 200.
func (s *Service) selfTest(ctx context.Context, p store.Provider) (int, error) {
	url := WebhookURL(s.URLs, p)
	hdr := http.Header{}
	hdr.Set(SelfTestHeader, SelfTestSignature(p.WebhookToken, s.now().Unix()))
	hdr.Set("Content-Type", "application/json")
	if s.SelfTest != nil {
		return s.SelfTest(ctx, url, hdr)
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, url, strings.NewReader(`{"selfTest":true}`))
	if err != nil {
		return 0, err
	}
	req.Header = hdr
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

func streamLabel(s string) string {
	switch providers.Stream(s) {
	case providers.StreamBroadcast:
		return "Broadcasts"
	case providers.StreamAutomation:
		return "Automations"
	case routing.StreamTransactional:
		return "Transactional"
	}
	return "System"
}

// incident records an entry of the incident timeline: its English title,
// and the message it comes from for readers in other languages.
func (s *Service) incident(ctx context.Context, q *store.Queries, wsID uuid.UUID, at time.Time, kind, severity, title string, text i18n.Text, provider, campaign *uuid.UUID) error {
	var texts json.RawMessage
	if !text.IsZero() {
		texts, _ = json.Marshal(map[string]i18n.Text{"title": text})
	}
	return q.ProvInsertIncident(ctx, store.ProvInsertIncidentParams{ID: newID(), WorkspaceID: wsID, OccurredAt: at, Kind: kind, Severity: severity, Title: title,
		CampaignID: campaign, ProviderID: provider, Texts: texts})
}

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// onDown fails streams over, holds broadcasts and tells the team.
func (s *Service) onDown(ctx context.Context, tx pgx.Tx, ws store.Workspace, p store.Provider, h state.Health, now time.Time) error {
	q := store.New(tx)
	set, err := routing.Load(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	provs, err := q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, x := range provs {
		byID[x.ID] = x
	}
	short := state.Short(p.Name, p.Adapter)
	// "503 ServiceUnavailable on SendEmail", "API errors" without one.
	errText := "API errors"
	failed := i18n.Args{"name": p.Name, "hasError": "no", "error": "", "hasOp": "no", "op": ""}
	if h.LastError != nil {
		errText = strings.TrimSpace(strings.Join([]string{strconv.Itoa(h.LastError.Status), h.LastError.Code}, " "))
		if h.LastError.Status == 0 {
			errText = h.LastError.Code
		}
		if errText == "" {
			errText = h.LastError.Message
		}
		failed["hasError"], failed["error"] = "yes", errText
		if h.LastError.Op != "" {
			errText += " on " + h.LastError.Op
			failed["hasOp"], failed["op"] = "yes", h.LastError.Op
		}
	}
	title, titleText := p.Name+": "+errText+" · health check failed.", mIncidentDown.Keep(failed)
	if h.Down.Reason == "auth" {
		title, titleText = p.Name+" rejected the credentials: "+errText+".", mIncidentAuthDown.Keep(failed)
	}
	if err := s.incident(ctx, q, ws.ID, h.Down.Since, "provider_down", "bad", title, titleText, &p.ID, nil); err != nil {
		return err
	}
	var auto, held []string
	for _, stream := range routing.Streams {
		def, ok := set.Default(stream)
		if !ok || !inSplit(def, p.ID) {
			continue
		}
		target, hold := routing.Failover(set.Policy(stream), def.Fallback, func(id uuid.UUID) routing.ProviderState {
			fp, ok := byID[id]
			if !ok {
				return routing.ProviderState{}
			}
			wu, _ := q.WarmupActiveForProvider(ctx, id)
			return routing.ProviderState{Available: state.Available(fp) && id != p.ID, WarmingUp: wu.Status != "" && wu.Status != "done"}
		})
		mode := "automatic"
		if hold {
			mode, target = "held", nil
		}
		if err := q.ProvStartFailover(ctx, store.ProvStartFailoverParams{ID: newID(), WorkspaceID: ws.ID, Stream: string(stream), ProviderID: p.ID,
			FallbackProviderID: target, Mode: mode, StartedAt: now, Detail: json.RawMessage(`{}`)}); err != nil {
			return err
		}
		if !hold {
			fb := byID[*target]
			auto = append(auto, streamLabel(string(stream)))
			failures := max(h.ConsecutiveFailures, 1)
			if err := s.incident(ctx, q, ws.ID, now, "failover", "ok",
				fmt.Sprintf("%s: switched to %s after %d failed health checks.", streamLabel(string(stream)), state.Short(fb.Name, fb.Adapter), failures),
				mIncidentFailover.Keep(i18n.Args{"stream": string(stream), "fallback": state.Short(fb.Name, fb.Adapter), "count": failures}), &p.ID, nil); err != nil {
				return err
			}
			continue
		}
		held = append(held, streamLabel(string(stream)))
		if stream != providers.StreamBroadcast {
			continue
		}
		camps, err := q.ProvSendingCampaignsOn(ctx, store.ProvSendingCampaignsOnParams{WorkspaceID: ws.ID, ProviderID: p.ID})
		if err != nil {
			return err
		}
		for _, c := range camps {
			detail, _ := json.Marshal(map[string]string{"reason": "provider_down", "provider": p.Name})
			if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: c.ID, ProviderID: p.ID, Status: "held", StatusDetail: string(detail)}); err != nil {
				return err
			}
			var planned, sent int64
			_ = tx.QueryRow(ctx, `SELECT cr.planned, coalesce((SELECT sum(cc.sent) FROM campaign_counters cc WHERE cc.campaign_id = cr.campaign_id AND cc.provider_id = cr.provider_id), 0)
				FROM campaign_routes cr WHERE cr.campaign_id = $1 AND cr.provider_id = $2`, c.ID, p.ID).Scan(&planned, &sent)
			heldN := max(planned-sent, 0)
			if _, err := tx.Exec(ctx, `INSERT INTO campaign_incidents (id, campaign_id, kind, scope, provider_id, metric, held, detail, tripped_at)
				VALUES ($1, $2, 'provider_outage', 'route', $3, 'api_errors', $4, $5, $6)`, newID(), c.ID, p.ID, heldN, json.RawMessage(`{}`), now); err != nil {
				return err
			}
			cid := c.ID
			if err := s.incident(ctx, q, ws.ID, now, "broadcast_held", "warn",
				fmt.Sprintf("Broadcasts: %q %s share held, %d recipients. Decision requested.", c.Name, short, heldN),
				mIncidentHeld.Keep(i18n.Args{"campaign": fmt.Sprintf("%q", c.Name), "short": short, "count": heldN}), &p.ID, &cid); err != nil {
				return err
			}
			// Each member reads it in their own language (notify's kept text).
			title, titleText := keep(mNotifyHeldTitle, i18n.Args{"campaign": fmt.Sprintf("%q", c.Name)})
			body, bodyText := keep(mNotifyHeldBody, i18n.Args{"name": p.Name, "count": heldN, "countText": strconv.FormatInt(heldN, 10)})
			if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.CampaignsSend, notify.Notification{
				Kind: notify.KindSendPaused, Severity: "bad", Title: title, TitleText: titleText, Body: body, BodyText: bodyText,
				Link: "/w/" + ws.Slug + "/providers", Data: map[string]any{"campaignId": c.ID.String(), "providerId": p.ID.String()},
			}); err != nil {
				return err
			}
		}
	}
	// "503 ServiceUnavailable on SendEmail since 14:02. Automations failed
	// over automatically. Broadcasts wait for you."
	body, bodyText := keep(mNotifyDownBody, i18n.Args{"hasError": failed["hasError"], "error": failed["error"], "hasOp": failed["hasOp"], "op": failed["op"],
		"since": h.Down.Since.In(state.Location(ws.Timezone)), "auto": strings.Join(auto, " and "), "autoCount": len(auto),
		"held": strings.Join(held, " and "), "heldCount": len(held)})
	ntitle, ntitleText := keep(mNotifyDownTitle, i18n.Args{"name": p.Name})
	if h.Down.Reason == "auth" {
		ntitle, ntitleText = keep(mNotifyAuthTitle, i18n.Args{"name": p.Name})
	}
	if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.ProvidersManage, notify.Notification{
		Kind: notify.KindProviderDown, Severity: "bad", Title: ntitle, TitleText: ntitleText, Body: body, BodyText: bodyText, Link: "/w/" + ws.Slug + "/providers",
		Data: map[string]any{"providerId": p.ID.String(), "reason": h.Down.Reason},
	}); err != nil {
		return err
	}
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.down", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
		Metadata: map[string]any{"reason": h.Down.Reason, "error": errText, "failed_over": auto, "held": held}})
}

func inSplit(def routing.Rule, id uuid.UUID) bool {
	for _, sp := range def.Splits {
		if sp.ProviderID == id && sp.Weight > 0 {
			return true
		}
	}
	return false
}

// onRecovered switches streams back and resumes broadcasts set to wait.
func (s *Service) onRecovered(ctx context.Context, tx pgx.Tx, ws store.Workspace, p store.Provider, down *state.DownState, now time.Time) error {
	q := store.New(tx)
	ended, err := q.ProvEndFailovers(ctx, store.ProvEndFailoversParams{ProviderID: p.ID, EndedAt: &now, EndReason: "recovered"})
	if err != nil {
		return err
	}
	short := state.Short(p.Name, p.Adapter)
	for _, f := range ended {
		if f.Mode == "automatic" {
			if err := s.incident(ctx, q, ws.ID, now, "failover_ended", "ok",
				fmt.Sprintf("%s: back on %s after %d healthy minutes.", streamLabel(f.Stream), short, SwitchBackMinutes),
				mIncidentFailoverEnded.Keep(i18n.Args{"stream": f.Stream, "short": short, "minutes": SwitchBackMinutes}), &p.ID, nil); err != nil {
				return err
			}
		}
	}
	held, err := q.ProvHeldRoutes(ctx, store.ProvHeldRoutesParams{WorkspaceID: ws.ID, ProviderID: p.ID})
	if err != nil {
		return err
	}
	for _, r := range held {
		var d struct {
			Decision string `json:"decision"`
		}
		_ = json.Unmarshal([]byte(r.StatusDetail), &d)
		if d.Decision != "wait" {
			continue
		}
		if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: r.CampaignID, ProviderID: p.ID, Status: "sending", StatusDetail: `{"decision":"wait","resumed":true}`}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaign_incidents SET resolved_at = $3, resolution = 'auto_recovered'
			WHERE campaign_id = $1 AND provider_id = $2 AND kind = 'provider_outage' AND resolved_at IS NULL`, r.CampaignID, p.ID, now); err != nil {
			return err
		}
	}
	if err := q.ProvResolveIncidents(ctx, store.ProvResolveIncidentsParams{ProviderID: &p.ID, Kinds: []string{"provider_down"}, ResolvedAt: &now}); err != nil {
		return err
	}
	minutes := 0
	if down != nil {
		minutes = int(now.Sub(down.Since).Minutes())
	}
	if err := s.incident(ctx, q, ws.ID, now, "provider_recovered", "ok", fmt.Sprintf("%s recovered after %d min.", p.Name, minutes),
		mIncidentRecovered.Keep(i18n.Args{"name": p.Name, "minutes": minutes}), &p.ID, nil); err != nil {
		return err
	}
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.recovered", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
		Metadata: map[string]any{"down_minutes": minutes}})
}

// onSilent holds the provider's Broadcast share and tells the team.
func (s *Service) onSilent(ctx context.Context, tx pgx.Tx, ws store.Workspace, p store.Provider, h state.Health, now time.Time) error {
	q := store.New(tx)
	set, err := routing.Load(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	if def, ok := set.Default(providers.StreamBroadcast); ok && inSplit(def, p.ID) {
		if err := q.ProvStartFailover(ctx, store.ProvStartFailoverParams{ID: newID(), WorkspaceID: ws.ID, Stream: string(providers.StreamBroadcast), ProviderID: p.ID,
			Mode: "held", StartedAt: now, Detail: json.RawMessage(`{"reason":"webhooks_silent"}`)}); err != nil {
			return err
		}
	}
	sinceAt := h.Webhooks.SilentSince.In(state.Location(ws.Timezone))
	since := sinceAt.Format("15:04")
	if err := s.incident(ctx, q, ws.ID, now, "webhooks_silent", "warn", p.Name+": no webhook events since "+since+" while mail went out.",
		mIncidentSilent.Keep(i18n.Args{"name": p.Name, "since": sinceAt}), &p.ID, nil); err != nil {
		return err
	}
	title, titleText := keep(mNotifySilentTitle, i18n.Args{"name": p.Name})
	body, bodyText := keep(mNotifySilentBody, i18n.Args{"since": sinceAt})
	if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.ProvidersManage, notify.Notification{
		Kind: notify.KindWebhooksSilent, Severity: "warn", Title: title, TitleText: titleText, Body: body, BodyText: bodyText,
		Link: "/w/" + ws.Slug + "/providers", Data: map[string]any{"providerId": p.ID.String()},
	}); err != nil {
		return err
	}
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.webhooks_silent", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
		Metadata: map[string]any{"silent_since": h.Webhooks.SilentSince}})
}

// onResumed lifts the hold once events flow again.
func (s *Service) onResumed(ctx context.Context, tx pgx.Tx, ws store.Workspace, p store.Provider, before state.Health, now time.Time) error {
	q := store.New(tx)
	if _, err := q.ProvEndFailovers(ctx, store.ProvEndFailoversParams{ProviderID: p.ID, EndedAt: &now, EndReason: "webhooks_resumed"}); err != nil {
		return err
	}
	if err := q.ProvResolveIncidents(ctx, store.ProvResolveIncidentsParams{ProviderID: &p.ID, Kinds: []string{"webhooks_silent"}, ResolvedAt: &now}); err != nil {
		return err
	}
	if err := s.incident(ctx, q, ws.ID, now, "webhooks_resumed", "ok", p.Name+": webhook events are arriving again.",
		mIncidentResumed.Keep(i18n.Args{"name": p.Name}), &p.ID, nil); err != nil {
		return err
	}
	wsID := ws.ID
	meta := map[string]any{}
	if before.Webhooks != nil {
		meta["silent_since"] = before.Webhooks.SilentSince
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.webhooks_resumed", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name, Metadata: meta})
}

// refreshTracking reads provider-side tracking and stores it.
func (s *Service) refreshTracking(ctx context.Context, p store.Provider) (state.Tracking, error) {
	cur := state.TrackingOf(p)
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		return cur, err
	}
	tm, ok := adapter.(providers.TrackingManager)
	if !ok {
		return cur, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	t, err := tm.Tracking(cctx)
	now := s.now()
	cur.CheckedAt = &now
	if err != nil {
		cur.Error = err.Error()
	} else {
		cur.OpenPixel, cur.LinkRewrite, cur.Detail, cur.Scope, cur.Error = t.OpenPixel, t.LinkRewrite, t.Detail, t.Scope, ""
		cur.DetailText, cur.ScopeText = t.DetailText, t.ScopeText
	}
	b, _ := json.Marshal(cur)
	return cur, s.Q.ProvUpdateTracking(ctx, store.ProvUpdateTrackingParams{ID: p.ID, Tracking: b})
}
