package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
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
// adapter's steps, and queues the job. It also scans regions.
func (s *Service) startProvisioning(ctx context.Context, p store.Provider, st *State, quota providers.Quota, now time.Time) {
	rows := []ProvisionState{
		{Key: "verify", Label: "Verify credentials and permissions", Text: "Verified credentials and permissions", Status: "done", At: &now},
		{Key: "account", Label: "Read the account", Text: accountText(quota), Status: "done", At: &now},
	}
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
	in := providers.ProvisionInput{WebhookURL: s.webhookURL(p, ""), Suffix: suffix}
	for _, step := range pv.ProvisionSteps(ctx, in) {
		rows = append(rows, ProvisionState{Key: step.Key, Label: step.Label, Status: "waiting"})
	}
	// No provider can reach a private public URL: say so before creating anything.
	if reach := s.reachable(ctx, in.WebhookURL, p); !reach.OK && reach.Private {
		for i := range rows {
			if rows[i].Status == "waiting" {
				rows[i].Status, rows[i].At = "failed", &now
				rows[i].Error = reach.Code + ": " + reach.Reason
				rows[i].Hint = "AWS has to reach ReputeCast's public URL to deliver events. Set PUBLIC_URL to the HTTPS address your reverse proxy exposes, then retry."
				break
			}
		}
		st.Connection.Provisioning = rows
		return
	}
	st.Connection.Provisioning = rows
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

// accountText is the "Read account" row.
func accountText(q providers.Quota) string {
	if q.Kind == "" && q.Account == "" {
		return "Read the account"
	}
	parts := []string{}
	if q.Sandbox {
		parts = append(parts, "sandbox")
	} else if q.Kind != "" {
		parts = append(parts, "production access")
	}
	if q.Max > 0 {
		per := "per 24 h"
		if q.Kind == "monthly_plan" {
			per = "per month"
		}
		parts = append(parts, thousands(int(q.Max))+" "+per)
	}
	if q.MaxRate > 0 {
		parts = append(parts, strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", q.MaxRate), "0"), ".")+" per second")
	}
	if len(parts) == 0 {
		return "Read account: " + strings.ToLower(q.Account)
	}
	return "Read account: " + strings.Join(parts, ", ")
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
			if r.Status == "failed" {
				return false, nil
			}
			if r.Status == "waiting" || r.Status == "running" {
				idx = i
				break
			}
		}
		if idx < 0 {
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
		if manage.IsDemo(p) {
			// Demo providers have no account to provision.
			now := s.now()
			if err := s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) {
				r.Status, r.At, r.Text = "done", &now, strings.Replace(r.Label, "Create", "Created", 1)
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
		in := providers.ProvisionInput{WebhookURL: s.webhookURL(p, ""), Suffix: state.Connection.ProvisionSuffix}
		cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		res, perr := pv.Provision(cctx, row.Key, in)
		cancel()
		now := s.now()
		if perr != nil {
			msg, kind := failureText(perr)
			hint := ""
			if kind == "auth" {
				hint = "the IAM user's policy doesn't allow it"
				if state.Connection.Access == "role" {
					hint = "the role policy was edited after the stack ran"
				}
			}
			_ = manage.RecordCall(ctx, s.Pool, p, row.Key, perr, 0, now)
			return false, s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) { r.Status, r.At, r.Error, r.Hint = "failed", &now, msg, hint })
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
					r.Status, r.At, r.Attempts = "failed", &now, attempts
					r.Error = "No confirmation after 5 minutes"
					r.Hint = "SNS couldn't reach " + in.WebhookURL + ". Check that the public URL is reachable from the internet, then retry."
				})
			}
			return true, s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) { r.Attempts = attempts })
		}
		if err := s.setRow(ctx, setupID, row.Key, func(r *ProvisionState) { r.Status, r.At, r.Text, r.Error, r.Hint = "done", &now, res.Done, "", "" }); err != nil {
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
	t.Detail = "Configuration sets publish delivery events only: open and click tracking off."
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
		for i := range state.Connection.Provisioning {
			r := &state.Connection.Provisioning[i]
			if r.Status == "failed" {
				r.Status, r.Error, r.Hint, r.Attempts = "waiting", "", "", 0
				n++
			}
		}
		if n == 0 {
			return api.Conflict("Nothing to retry: no provisioning step failed.")
		}
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	// The public URL may still be private.
	state := decodeState(st.State)
	if reach := s.reachable(ctx, s.webhookURL(p, ""), p); !reach.OK && reach.Private {
		now := s.now()
		for i := range state.Connection.Provisioning {
			if state.Connection.Provisioning[i].Status == "waiting" {
				r := &state.Connection.Provisioning[i]
				r.Status, r.At, r.Error = "failed", &now, reach.Code+": "+reach.Reason
				r.Hint = "AWS has to reach ReputeCast's public URL to deliver events. Set PUBLIC_URL to the HTTPS address your reverse proxy exposes, then retry."
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
	d, err := descriptor(cur.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	found := false
	for _, f := range d.Fields {
		if f.Key == "region" && contains(f.Options, in.Region) {
			found = true
		}
	}
	if !found {
		return ConnectSetup{}, api.Invalid("body.region", "Pick an SES region from the list.")
	}
	cfg := currentSettings(&p)
	before := stringOf(cfg["region"])
	cfg["region"] = in.Region
	delete(cfg, "configuration_sets")
	delete(cfg, "configuration_set")
	b := built{settings: cfg, creds: s.currentCreds(&p), label: in.Region}
	quota := providers.Quota{}
	if !manage.IsDemo(p) {
		adapter, err := newAdapter(d, p.ID, ws.ID, b, s.Factory.WebhookSecret(p), s.httpClient())
		if err != nil {
			return ConnectSetup{}, err
		}
		if v, ok := adapter.(providers.Verifier); ok {
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			quota, err = v.Verify(cctx)
			cancel()
			if err != nil {
				return ConnectSetup{}, api.Err(422, api.CodeProviderUnavailable, "SES didn't answer in "+in.Region+": "+callMessage(err))
			}
		}
	}
	cb, _ := json.Marshal(cfg)
	statusWord := state.Healthy
	if quota.Sandbox {
		statusWord = state.Sandbox
	}
	h := state.HealthOf(p)
	h.Account, h.Sandbox = quota.Account, quota.Sandbox
	if quota.Kind != "" {
		h.Quota = &state.Quota{Kind: quota.Kind, Max: quota.Max, Used: quota.Used}
	}
	if quota.MaxRate > 0 {
		h.Rate = &state.Rate{Max: quota.MaxRate}
	}
	hb, _ := json.Marshal(h)
	now := s.now()
	if _, err := s.Q.ConnectUpdateProvider(ctx, store.ConnectUpdateProviderParams{ID: p.ID, Label: in.Region, Config: cb, CredentialsEnc: p.CredentialsEnc,
		Status: statusWord, StatusDetail: quota.Account, IpMode: p.IpMode, Ips: p.Ips, Health: hb, CheckedAt: &now}); err != nil {
		return ConnectSetup{}, err
	}
	p.Config = cb
	st, err := s.update(ctx, ws, cur.ID, "provider.region_change", map[string]any{"before": before, "after": in.Region}, func(_ *store.ProviderSetup, state *State) error {
		state.Connection.Region = in.Region
		state.Connection.Summary = in.Region + " · " + map[bool]string{true: "sandbox", false: "production"}[quota.Sandbox]
		rows := []ProvisionState{}
		for _, r := range state.Connection.Provisioning {
			switch r.Key {
			case "verify":
			case "account":
				r.Text, r.At = accountText(quota), &now
			default:
				r.Status, r.Text, r.Error, r.Hint, r.At, r.Attempts = "waiting", "", "", "", nil, 0
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
			return ConnectSetup{}, api.Conflict(p.Name + " has no production-access request.")
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
			return ConnectSetup{}, api.Err(502, api.CodeProviderUnavailable, "AWS didn't take the request: "+callMessage(err)+". Nothing was filed.")
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
