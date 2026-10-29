package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProvisionArgs runs a setup's provisioning steps (SES: SNS topic,
// subscription handshake, configuration sets) one after another, saving
// progress after each so the connection step shows it live.
type ProvisionArgs struct {
	SetupID uuid.UUID `json:"setup_id"`
}

// Kind implements river.JobArgs.
func (ProvisionArgs) Kind() string { return "connect.provision" }

// InsertOpts keeps one run per setup at a time; a finished run doesn't
// block the next one (retry, region change).
func (ProvisionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true,
		ByState: []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateScheduled}}}
}

// ProvisionWorker runs ProvisionArgs.
type ProvisionWorker struct {
	river.WorkerDefaults[ProvisionArgs]
	S *Service
}

// Timeout bounds one run.
func (w *ProvisionWorker) Timeout(*river.Job[ProvisionArgs]) time.Duration { return 5 * time.Minute }

// Work implements river.Worker.
func (w *ProvisionWorker) Work(ctx context.Context, job *river.Job[ProvisionArgs]) error {
	again, err := w.S.RunProvisioning(ctx, job.Args.SetupID)
	if err != nil {
		return err
	}
	if again {
		// Waiting for the SNS handshake: look again in five seconds.
		return river.JobSnooze(5 * time.Second)
	}
	return nil
}

// maxConfirmAttempts bounds the wait for the SNS handshake (5 s apart).
const maxConfirmAttempts = 60

// startProvisioning lists the provisioning rows after a successful
// connection: the credential check and the account read (done), then the
// adapter's steps.
func (s *Service) startProvisioning(ctx context.Context, p store.Provider, st *State, quota providers.Quota, now time.Time) {
	verify := ProvisionState{Key: "verify", Status: "done", At: &now}
	verify.Label = verify.say("label", mProvisionVerify, nil)
	verify.Text = verify.say("text", mProvisionVerified, nil)
	account := ProvisionState{Key: "account", Status: "done", At: &now}
	account.Label = account.say("label", mProvisionAccount, nil)
	var text i18n.Text
	account.Text, text = accountText(quota)
	account.keep("text", text)
	rows := []ProvisionState{verify, account}
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		st.Connection.Provisioning = rows
		return
	}
	pv, ok := adapter.(providers.Provisioner)
	if !ok {
		st.Connection.Provisioning = rows
		return
	}
	suffix := s.provisionSuffix(ctx, p)
	st.Connection.ProvisionSuffix = suffix
	in := s.provisionInput(ctx, p, *st)
	for _, step := range pv.ProvisionSteps(ctx, in) {
		row := ProvisionState{Key: step.Key, Label: step.Label, Status: "waiting"}
		row.keep("label", step.LabelText)
		rows = append(rows, row)
	}
	// The setup is done once a real event came back (PR-3).
	rows = append(rows, eventRows(adapter)...)
	// No provider can reach a private public URL: say so before creating
	// anything (events through a queue need no public URL).
	if in.Events == providers.EventsQueue {
		st.Connection.Provisioning = rows
		return
	}
	if reach := s.reachable(ctx, in.WebhookURL, p); !reach.OK && reach.Private {
		for i := range rows {
			if rows[i].Status == "waiting" {
				rows[i].Status, rows[i].At = "failed", &now
				rows[i].unreachable(reach)
				break
			}
		}
		st.Connection.Provisioning = rows
		return
	}
	st.Connection.Provisioning = rows
}

// provisionInput is what the provider's provisioning steps need: where
// events go (the webhook or a queue, PR-4) and the workspace's tenant
// (PR-11, named after the workspace).
func (s *Service) provisionInput(ctx context.Context, p store.Provider, st State) providers.ProvisionInput {
	in := providers.ProvisionInput{WebhookURL: s.webhookURL(p, ""), Suffix: st.Connection.ProvisionSuffix, Events: st.Connection.Events}
	if ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID); err == nil {
		in.Tenant = ws.Slug
	}
	return in
}

// queueProvisioning queues provisioning and the region scan of a setup.
func (s *Service) queueProvisioning(ctx context.Context, setupID uuid.UUID) {
	c := s.jobs()
	if c == nil {
		return
	}
	if _, err := c.Insert(ctx, ProvisionArgs{SetupID: setupID}, nil); err != nil {
		s.log().WarnContext(ctx, "queue provisioning", "setup", setupID, "err", err)
	}
	if _, err := c.Insert(ctx, ScanArgs{SetupID: setupID}, nil); err != nil {
		s.log().WarnContext(ctx, "queue region scan", "setup", setupID, "err", err)
	}
}

// accountText is the "Read account" row once done ("Read account:
// production access, 50,000 per 24 h, 14 per second"), with its message.
func accountText(q providers.Quota) (string, i18n.Text) {
	if q.Kind == "" && q.Account == "" {
		return say(mProvisionAccount, nil)
	}
	mode := "none"
	if q.Sandbox {
		mode = "sandbox"
	} else if q.Kind != "" {
		mode = "production"
	}
	hasMax, hasRate := q.Max > 0, q.MaxRate > 0
	if mode == "none" && !hasMax && !hasRate {
		return say(mProvisionAccountReadAs, i18n.Args{"account": strings.ToLower(q.Account)})
	}
	per := "day"
	if q.Kind == "monthly_plan" {
		per = "month"
	}
	// One decimal at most, as the provider's console shows it.
	rate, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", q.MaxRate), 64)
	return say(mProvisionAccountRead, i18n.Args{"mode": mode, "hasMax": yesNo(hasMax), "max": q.Max, "per": per, "maxSep": yesNo(mode != "none" && hasMax),
		"hasRate": yesNo(hasRate), "rate": rate, "rateSep": yesNo(hasRate && (mode != "none" || hasMax))})
}

// say sets one of a row's texts from a message (field: label, text, error,
// hint) and returns its English.
func (r *ProvisionState) say(field string, m i18n.Msg, args i18n.Args) string {
	s, t := say(m, args)
	r.keep(field, t)
	return s
}

// keep sets the message of one of a row's texts; a zero message clears it,
// so the row reads in English (an adapter's own text without a message).
func (r *ProvisionState) keep(field string, t i18n.Text) {
	if t.IsZero() {
		delete(r.Texts, field)
		return
	}
	if r.Texts == nil {
		r.Texts = map[string]i18n.Text{}
	}
	r.Texts[field] = t
}

// clearProblem forgets a row's error and hint.
func (r *ProvisionState) clearProblem() {
	r.Error, r.Hint = "", ""
	for _, k := range []string{"error", "reason", "hint"} {
		delete(r.Texts, k)
	}
}

// unreachable fails a row on a public URL no provider can reach: the
// self-test's code and reason, and what to do.
func (r *ProvisionState) unreachable(reach SelfTestState) {
	r.clearProblem()
	r.Error = reach.Code + ": " + reach.Reason
	r.keep("reason", reach.ReasonText)
	r.Hint = r.say("hint", mProvisionHintPublicURL, nil)
}

// provisionSuffix avoids clashing resource names when another connection
// of this instance uses the same account and region.
func (s *Service) provisionSuffix(ctx context.Context, p store.Provider) string {
	var me struct {
		Region    string `json:"region"`
		AccountID string `json:"account_id"`
	}
	_ = json.Unmarshal(p.Config, &me)
	others, err := s.Q.ConnectSameAdapter(ctx, store.ConnectSameAdapterParams{Adapter: p.Adapter, ID: p.ID})
	if err != nil {
		return ""
	}
	for _, o := range others {
		var them struct {
			Demo      bool   `json:"demo"`
			Region    string `json:"region"`
			AccountID string `json:"account_id"`
		}
		_ = json.Unmarshal(o.Config, &them)
		if !them.Demo && them.AccountID != "" && them.AccountID == me.AccountID && them.Region == me.Region {
			ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
			if err == nil {
				return ws.Slug
			}
			return p.ID.String()[:8]
		}
	}
	return ""
}

var deniedAction = regexp.MustCompile(`perform:\s*([a-z0-9-]+:[A-Za-z0-9*]+)`)

// failureText is the failed row's error: the provider's code and the
// action it refused ("AccessDenied: ses:CreateConfigurationSet").
func failureText(err error) (string, string) {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return err.Error(), ""
	}
	code := ce.Code
	if code == "" {
		code = ce.Op
	}
	action := ce.Op
	if m := deniedAction.FindStringSubmatch(ce.Message); m != nil {
		action = m[1]
	}
	if ce.Auth {
		return code + ": " + action, "auth"
	}
	return code + ": " + ce.Message, ""
}

// RunProvisioning runs the pending provisioning steps of a setup. It
// returns true when a step waits for something (the SNS handshake) and the
// job should run again in a few seconds.
func (s *Service) RunProvisioning(ctx context.Context, setupID uuid.UUID) (bool, error) {
	for {
		st, err := s.Q.SetupGetAny(ctx, setupID)
		if err != nil {
			return false, err
		}
		if st.ProviderID == nil || st.Status == "abandoned" {
			return false, nil
		}
		state := decodeState(st.State)
		idx := -1
		for i, r := range state.Connection.Provisioning {
			// A failed step waits for a retry; the event check waits for a
			// verified domain.
			if r.Status == "failed" || r.Status == statusDeferred {
				return false, nil
			}
			if r.Status == "waiting" || r.Status == "running" {
				idx = i
				break
			}
		}
		if idx < 0 {
			if hasEventRows(state.Connection.Provisioning) {
				// The resources were recorded before the event check ran.
				return false, nil
			}
			return false, s.finishProvisioning(ctx, st, state)
		}
		p, err := s.Q.ProvGetAny(ctx, *st.ProviderID)
		if err != nil {
			return false, err
		}
		row := state.Connection.Provisioning[idx]
		if row.Status != "running" {
			if err := s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) { r.Status = "running" }); err != nil {
				return false, err
			}
		}
		if row.Key == eventSendKey || row.Key == eventCheckKey {
			if row.Key == eventSendKey && row.Attempts == 0 {
				// Every resource exists: record them before the event check,
				// which may wait for a verified domain.
				if err := s.finishProvisioning(ctx, st, state); err != nil {
					return false, err
				}
				if err := s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) { r.Attempts = 1 }); err != nil {
					return false, err
				}
			}
			again, stop, err := s.runEventStep(ctx, st, state, p, row)
			if err != nil || again || stop {
				return again, err
			}
			continue
		}
		if manage.IsDemo(p) {
			// Demo providers have no account to provision.
			now := s.now()
			if err := s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) {
				r.Status, r.At, r.Text = "done", &now, strings.Replace(r.Label, "Create", "Created", 1)
				// Other languages read the step's own words.
				r.keep("text", r.Texts["label"])
			}); err != nil {
				return false, err
			}
			continue
		}
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return false, err
		}
		pv, ok := adapter.(providers.Provisioner)
		if !ok {
			return false, nil
		}
		in := s.provisionInput(ctx, p, state)
		cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		res, perr := pv.Provision(cctx, row.Key, in)
		cancel()
		now := s.now()
		if perr != nil {
			msg, kind := failureText(perr)
			_ = manage.RecordCall(ctx, s.Pool, p, row.Key, perr, 0, now)
			return false, s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) {
				r.clearProblem()
				r.Status, r.At, r.Error = "failed", &now, msg
				if kind == "auth" {
					hint := mProvisionHintIAMUser
					if state.Connection.Access == "role" {
						hint = mProvisionHintRolePolicy
					}
					r.Hint = r.say("hint", hint, nil)
				}
			})
		}
		if len(res.Settings) > 0 {
			cfg := currentSettings(&p)
			for k, v := range res.Settings {
				cfg[k] = v
			}
			b, _ := json.Marshal(cfg)
			if err := s.Q.ConnectSetProviderConfig(ctx, store.ConnectSetProviderConfigParams{ID: p.ID, Config: b}); err != nil {
				return false, err
			}
		}
		if res.Pending {
			attempts := row.Attempts + 1
			if attempts >= maxConfirmAttempts {
				return false, s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) {
					r.clearProblem()
					r.Status, r.At, r.Attempts = "failed", &now, attempts
					r.Error = r.say("error", mProvisionNoConfirmation, nil)
					r.Hint = r.say("hint", mProvisionHintSNS, i18n.Args{"url": in.WebhookURL})
				})
			}
			return true, s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) { r.Attempts = attempts })
		}
		if err := s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) {
			r.clearProblem()
			r.Status, r.At, r.Text, r.Note = "done", &now, res.Done, res.Note
			r.keep("text", res.DoneText)
			r.keep("note", res.NoteText)
		}); err != nil {
			return false, err
		}
	}
}

// setRow changes one provisioning row and saves the setup.
func (s *Service) setRow(ctx context.Context, setupID uuid.UUID, key string, fn func(*ProvisionState)) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st, err := q.SetupLock(ctx, setupID)
		if err != nil {
			return err
		}
		state := decodeState(st.State)
		for i := range state.Connection.Provisioning {
			if state.Connection.Provisioning[i].Key == key {
				fn(&state.Connection.Provisioning[i])
			}
		}
		_, err = s.save(ctx, q, st, state)
		return err
	})
}

// finishProvisioning records the result once every step is done: the
// provider's tracking and capability snapshot, and an audit entry.
func (s *Service) finishProvisioning(ctx context.Context, st store.ProviderSetup, setupState State) error {
	if st.ProviderID == nil {
		return nil
	}
	p, err := s.Q.ProvGetAny(ctx, *st.ProviderID)
	if err != nil {
		return err
	}
	keys := []string{}
	for _, r := range setupState.Connection.Provisioning {
		keys = append(keys, r.Key)
	}
	t := state.TrackingOf(p)
	t.OpenPixel, t.LinkRewrite = false, false
	now := s.now()
	t.CheckedAt = &now
	t.Detail, t.DetailText = say(mTrackingConfigSets, nil)
	tb, _ := json.Marshal(t)
	wsID := p.WorkspaceID
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ProvUpdateTracking(ctx, store.ProvUpdateTrackingParams{ID: p.ID, Tracking: tb}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "provider.provision", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
			Actor: &audit.Actor{Type: "system", Name: "System"}, After: map[string]any{"steps": keys}})
	})
}

// RetryProvisioning sets failed steps back to waiting and runs them again.
func (s *Service) RetryProvisioning(ctx context.Context, ws *store.Workspace, rawID string) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	st, err := s.update(ctx, ws, cur.ID, "provider.provision_retry", map[string]any{"provider": p.Name}, func(_ *store.ProviderSetup, state *State) error {
		n := 0
		rows := state.Connection.Provisioning
		for i := range rows {
			r := &rows[i]
			if r.Status != "failed" {
				continue
			}
			r.clearProblem()
			r.Status, r.Attempts = "waiting", 0
			n++
			switch r.Key {
			case eventSendKey:
				// The resources were recorded already.
				r.Attempts = 1
			case eventCheckKey:
				// No event came back: the check sends a new message.
				for j := range rows {
					if rows[j].Key == eventSendKey {
						rows[j].clearProblem()
						rows[j].Status, rows[j].MessageID, rows[j].Attempts = "waiting", "", 1
					}
				}
			}
		}
		if n == 0 {
			return conflict(ctx, mProblemNothingToRetry, nil)
		}
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	// The public URL may still be private (events through a queue need none).
	state := decodeState(st.State)
	if reach := s.reachable(ctx, s.webhookURL(p, ""), p); state.Connection.Events != providers.EventsQueue && !reach.OK && reach.Private {
		now := s.now()
		for i := range state.Connection.Provisioning {
			if state.Connection.Provisioning[i].Status == "waiting" {
				r := &state.Connection.Provisioning[i]
				r.Status, r.At = "failed", &now
				r.unreachable(reach)
				break
			}
		}
		if st, err = s.saveSetup(ctx, ws, st, state, "", nil); err != nil {
			return ConnectSetup{}, err
		}
		return s.View(ctx, ws, st)
	}
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, ProvisionArgs{SetupID: st.ID}, nil); err != nil {
			return ConnectSetup{}, err
		}
	}
	return s.View(ctx, ws, st)
}

// ConnectRegionInput picks the SES region.
type ConnectRegionInput struct {
	Region string `json:"region" minLength:"6" maxLength:"32"`
}

// SetRegion moves an SES connection to another region: the account is read
// there and provisioning runs again in it.
func (s *Service) SetRegion(ctx context.Context, ws *store.Workspace, rawID string, in ConnectRegionInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	d, err := descriptor(ctx, cur.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	l := i18n.LangFrom(ctx)
	found := false
	for _, f := range d.Fields {
		if f.Key == "region" && contains(f.Options, in.Region) {
			found = true
		}
	}
	if !found {
		return ConnectSetup{}, api.Invalid("body.region", mProblemPickRegion.Text(l, nil))
	}
	cfg := currentSettings(&p)
	before := stringOf(cfg["region"])
	cfg["region"] = in.Region
	delete(cfg, "configuration_sets")
	delete(cfg, "configuration_set")
	b := built{settings: cfg, creds: s.currentCreds(&p), label: in.Region}
	quota := providers.Quota{}
	var adapter providers.Provider
	if !manage.IsDemo(p) {
		if adapter, err = newAdapter(d, p.ID, ws.ID, b, s.Factory.WebhookSecret(p), s.httpClient()); err != nil {
			return ConnectSetup{}, err
		}
		if v, ok := adapter.(providers.Verifier); ok {
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			quota, err = v.Verify(cctx)
			cancel()
			if err != nil {
				return ConnectSetup{}, api.Err(422, api.CodeProviderUnavailable, mProblemSESNoAnswer.Text(l, i18n.Args{"region": in.Region, "error": providers.MessageIn(err, l)}))
			}
		}
	}
	cb, _ := json.Marshal(cfg)
	statusWord := state.Healthy
	if quota.Sandbox {
		statusWord = state.Sandbox
	}
	h := state.HealthOf(p)
	h.Account, h.AccountText, h.Sandbox = quota.Account, quota.AccountText, quota.Sandbox
	if quota.Kind != "" {
		h.Quota = &state.Quota{Kind: quota.Kind, Max: quota.Max, Used: quota.Used}
	}
	if quota.MaxRate > 0 {
		h.Rate = &state.Rate{Max: quota.MaxRate}
	}
	hb, _ := json.Marshal(h)
	now := s.now()
	if _, err := s.Q.ConnectUpdateProvider(ctx, store.ConnectUpdateProviderParams{ID: p.ID, Label: in.Region, Config: cb, CredentialsEnc: p.CredentialsEnc,
		Status: statusWord, StatusDetail: quota.Account, Health: hb, CheckedAt: &now}); err != nil {
		return ConnectSetup{}, err
	}
	p.Config = cb
	// Dedicated IPs belong to a region: the new one's are read again.
	p = s.refreshIPs(ctx, p, adapter)
	st, err := s.update(ctx, ws, cur.ID, "provider.region_change", map[string]any{"before": before, "after": in.Region}, func(_ *store.ProviderSetup, state *State) error {
		state.Connection.Region = in.Region
		state.Connection.Summary = in.Region + " · " + map[bool]string{true: "sandbox", false: "production"}[quota.Sandbox]
		rows := []ProvisionState{}
		for _, r := range state.Connection.Provisioning {
			switch r.Key {
			case "verify":
			case "account":
				var text i18n.Text
				r.Text, text = accountText(quota)
				r.keep("text", text)
				r.At = &now
			default:
				r.clearProblem()
				r.Status, r.Text, r.At, r.Attempts = "waiting", "", nil, 0
				r.keep("text", i18n.Text{})
			}
			rows = append(rows, r)
		}
		state.Connection.Provisioning = rows
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, ProvisionArgs{SetupID: st.ID}, nil); err != nil {
			return ConnectSetup{}, err
		}
	}
	return s.View(ctx, ws, st)
}

// ConnectEventsInput picks how SES events reach ReputeCast (PR-4).
type ConnectEventsInput struct {
	Events string `json:"events" enum:"webhook,queue"`
}

// SetEvents switches an SES connection between SNS calling the webhook and
// an SQS queue ReputeCast polls: the event path is set up again, and the
// event check runs again through it.
func (s *Service) SetEvents(ctx context.Context, ws *store.Workspace, rawID string, in ConnectEventsInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	if cur.Status == "abandoned" {
		return ConnectSetup{}, conflict(ctx, mProblemCancelled, nil)
	}
	d, err := descriptor(ctx, cur.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	if flowOf(d) != "ses" {
		return ConnectSetup{}, conflict(ctx, mProblemEventsMode, i18n.Args{"name": d.Name})
	}
	before := decodeState(cur.State).Connection.Events
	if before == "" {
		before = "webhook"
	}
	if cur.ProviderID == nil {
		// Not connected yet: the choice applies when it connects.
		out, err := s.update(ctx, ws, cur.ID, "provider.events_mode", map[string]any{"before": before, "after": in.Events}, func(_ *store.ProviderSetup, state *State) error {
			state.Connection.Events = in.Events
			return nil
		})
		if err != nil {
			return ConnectSetup{}, err
		}
		return s.View(ctx, ws, out)
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	cfg := currentSettings(&p)
	if in.Events == providers.EventsQueue {
		cfg["events"] = providers.EventsQueue
	} else {
		delete(cfg, "events")
		delete(cfg, "queue_url")
		delete(cfg, "queue_arn")
	}
	cb, _ := json.Marshal(cfg)
	p.Config = cb
	st, err := s.updateWith(ctx, ws, cur.ID, "provider.events_mode", map[string]any{"before": before, "after": in.Events}, func(q *store.Queries, _ *store.ProviderSetup, state *State) error {
		// The provider's settings change with the setup and its audit entry.
		if err := q.ConnectSetProviderConfig(ctx, store.ConnectSetProviderConfigParams{ID: p.ID, Config: cb}); err != nil {
			return err
		}
		state.Connection.Events = in.Events
		var adapter providers.Provider
		if a, err := s.Factory.Open(ctx, p); err == nil {
			adapter = a
		}
		pv, ok := adapter.(providers.Provisioner)
		if !ok && !manage.IsDemo(p) {
			return nil
		}
		// Keep the access and account rows; the rest runs again through the
		// new path, and the event check with it.
		rows := []ProvisionState{}
		for _, r := range state.Connection.Provisioning {
			if r.Key == "verify" || r.Key == "account" {
				rows = append(rows, r)
			}
		}
		in := s.provisionInput(ctx, p, *state)
		if ok {
			for _, step := range pv.ProvisionSteps(ctx, in) {
				row := ProvisionState{Key: step.Key, Label: step.Label, Status: "waiting"}
				row.keep("label", step.LabelText)
				rows = append(rows, row)
			}
			rows = append(rows, eventRows(adapter)...)
		} else {
			for _, r := range state.Connection.Provisioning {
				if r.Key != "verify" && r.Key != "account" {
					r.clearProblem()
					r.Status, r.Text, r.Note, r.At, r.Attempts, r.MessageID = "waiting", "", "", nil, 0, ""
					r.keep("text", i18n.Text{})
					r.keep("note", i18n.Text{})
					rows = append(rows, r)
				}
			}
		}
		for i := range rows {
			if rows[i].Key == eventSendKey {
				// The resources were recorded already.
				rows[i].Attempts = 1
			}
		}
		state.Connection.Provisioning = rows
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	if c := s.jobs(); c != nil {
		if _, err := c.Insert(ctx, ProvisionArgs{SetupID: st.ID}, nil); err != nil {
			return ConnectSetup{}, err
		}
	}
	return s.View(ctx, ws, st)
}

// ConnectProductionAccessInput files a production-access request.
type ConnectProductionAccessInput struct {
	Region      string `json:"region" minLength:"6" maxLength:"32"`
	Website     string `json:"website" format:"uri" maxLength:"500"`
	MailType    string `json:"mailType" enum:"marketing,transactional"`
	UseCase     string `json:"useCase" minLength:"20" maxLength:"5000"`
	DailyVolume int64  `json:"dailyVolume" minimum:"1" maximum:"100000000"`
	Language    string `json:"language" enum:"en,ja"`
}

// RequestProductionAccess asks the provider to lift its sandbox in a region.
func (s *Service) RequestProductionAccess(ctx context.Context, ws *store.Workspace, rawID string, in ConnectProductionAccessInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	if !manage.IsDemo(p) {
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return ConnectSetup{}, err
		}
		pr, ok := adapter.(providers.ProductionAccessRequester)
		if !ok {
			return ConnectSetup{}, conflict(ctx, mProblemNoProductionAccess, i18n.Args{"name": p.Name})
		}
		var contacts []string
		if u := principalUser(ctx); u != nil {
			if usr, err := s.Q.GetUser(ctx, *u); err == nil {
				contacts = []string{usr.Email}
			}
		}
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = pr.RequestProductionAccess(cctx, providers.ProductionAccessRequest{Region: in.Region, WebsiteURL: in.Website, MailType: in.MailType, UseCase: in.UseCase,
			Language: in.Language, DailyVolume: in.DailyVolume, ContactEmails: contacts})
		cancel()
		if err != nil {
			return ConnectSetup{}, api.Err(502, api.CodeProviderUnavailable, mProblemAWSRejected.Text(i18n.LangFrom(ctx), i18n.Args{"error": providers.MessageIn(err, i18n.LangFrom(ctx))}))
		}
	}
	now := s.now()
	st, err := s.update(ctx, ws, cur.ID, "provider.production_access_request", map[string]any{"region": in.Region, "mail_type": in.MailType, "daily_volume": in.DailyVolume},
		func(_ *store.ProviderSetup, state *State) error {
			state.Connection.ProductionAccess = &ProductionAccess{Region: in.Region, RequestedAt: now, Website: in.Website, MailType: in.MailType, UseCase: in.UseCase,
				Language: in.Language, DailyVolume: in.DailyVolume}
			return nil
		})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, st)
}

// ScanArgs reads an SES account in every region for the region select.
type ScanArgs struct {
	SetupID uuid.UUID `json:"setup_id"`
}

// Kind implements river.JobArgs.
func (ScanArgs) Kind() string { return "connect.region_scan" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ScanArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 2}
}

// ScanWorker runs ScanArgs.
type ScanWorker struct {
	river.WorkerDefaults[ScanArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ScanWorker) Work(ctx context.Context, job *river.Job[ScanArgs]) error {
	return w.S.ScanRegions(ctx, job.Args.SetupID)
}

// ScanRegions records the account's standing in every region.
func (s *Service) ScanRegions(ctx context.Context, setupID uuid.UUID) error {
	st, err := s.Q.SetupGetAny(ctx, setupID)
	if err != nil || st.ProviderID == nil {
		return err
	}
	p, err := s.Q.ProvGetAny(ctx, *st.ProviderID)
	if err != nil || manage.IsDemo(p) {
		return err
	}
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		return err
	}
	rs, ok := adapter.(providers.RegionScanner)
	if !ok {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	found, err := rs.ScanRegions(cctx)
	cancel()
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SetupLock(ctx, setupID)
		if err != nil {
			return err
		}
		state := decodeState(cur.State)
		state.Connection.Regions = make([]RegionState, 0, len(found))
		for _, r := range found {
			state.Connection.Regions = append(state.Connection.Regions, RegionState{Region: r.Region, Name: r.Name, Production: r.Production, Max24h: r.Max24h, MaxRate: r.MaxRate, Err: r.Err})
		}
		_, err = s.save(ctx, q, cur, state)
		return err
	})
}

// hasEventRows reports whether a setup has the event check (PR-3).
func hasEventRows(rows []ProvisionState) bool {
	for _, r := range rows {
		if r.Key == eventSendKey || r.Key == eventCheckKey {
			return true
		}
	}
	return false
}
