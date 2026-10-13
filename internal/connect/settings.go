package connect

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// The workspace-wide sending defaults live in two JSON columns:
// workspaces.send_thresholds keeps the auto-pause thresholds read by the
// sending engine (hard_bounce, complaint, scope, min_sample) and gains
// mailbox_complaint (per mailbox provider), provider_error_rate (% of API
// calls over 3 minutes) and confirm_over (recipients above which a send
// needs a typed confirmation); workspaces.sending_defaults gains throttles
// (messages per second per mailbox provider, 0 = no limit),
// suppression_sync ("15m" or "1h"), webhook_secrets_rotated_at and
// seed_list (test addresses per mailbox provider).

// sendThresholds is workspaces.send_thresholds.
type sendThresholds struct {
	HardBounce        float64 `json:"hard_bounce"`
	Complaint         float64 `json:"complaint"`
	MailboxComplaint  float64 `json:"mailbox_complaint"`
	ProviderErrorRate float64 `json:"provider_error_rate"`
	Scope             string  `json:"scope"`
	MinSample         int64   `json:"min_sample"`
	ConfirmOver       int64   `json:"confirm_over"`
}

// SendingDefaults is what the defaults cards show and edit.
type SendingDefaults struct {
	Bounce           float64            `json:"bounce" doc:"Auto-pause at this hard-bounce rate, percent"`
	Complaint        float64            `json:"complaint" doc:"Auto-pause at this complaint rate, percent"`
	MailboxComplaint float64            `json:"mailboxComplaint" doc:"Auto-pause one mailbox provider at this complaint rate, percent"`
	ProviderError    float64            `json:"providerError" doc:"Treat a provider as failing at this API error rate over 3 minutes, percent"`
	PauseScope       string             `json:"pauseScope" enum:"route,campaign"`
	ConfirmOver      int64              `json:"confirmOver" doc:"Sends to more recipients need the count typed to confirm"`
	Throttles        map[string]float64 `json:"throttles" doc:"Messages per second per mailbox provider (gmail, outlook, yahoo, icloud); 0 = no limit"`
	SuppressionSync  string             `json:"suppressionSync" enum:"15m,1h"`
	// SecretsRotatedAt is when the webhook signing secrets were last rotated.
	SecretsRotatedAt *time.Time `json:"secretsRotatedAt,omitempty"`
}

// defaultThrottles are the safe per-mailbox-provider starting points.
var defaultThrottles = map[string]float64{"gmail": 0, "outlook": 5, "yahoo": 4, "icloud": 3}

func (s *Service) sendingDefaults(ctx context.Context, ws *store.Workspace) (SendingDefaults, error) {
	row, err := s.Q.SenderWorkspaceDefaults(ctx, ws.ID)
	if err != nil {
		return SendingDefaults{}, err
	}
	th := sendThresholds{}
	_ = json.Unmarshal(workspaces.DefaultSendThresholds, &th)
	_ = json.Unmarshal(row.SendThresholds, &th)
	var sd struct {
		Throttles        map[string]float64 `json:"throttles"`
		SuppressionSync  string             `json:"suppression_sync"`
		SecretsRotatedAt *time.Time         `json:"webhook_secrets_rotated_at"`
	}
	_ = json.Unmarshal(row.SendingDefaults, &sd)
	out := SendingDefaults{Bounce: th.HardBounce, Complaint: th.Complaint, MailboxComplaint: th.MailboxComplaint, ProviderError: th.ProviderErrorRate,
		PauseScope: th.Scope, ConfirmOver: th.ConfirmOver, Throttles: map[string]float64{}, SuppressionSync: sd.SuppressionSync, SecretsRotatedAt: sd.SecretsRotatedAt}
	if out.MailboxComplaint == 0 {
		out.MailboxComplaint = out.Complaint
	}
	if out.ProviderError == 0 {
		out.ProviderError = 5
	}
	if out.PauseScope == "" {
		out.PauseScope = "route"
	}
	if out.ConfirmOver == 0 {
		out.ConfirmOver = 5000
	}
	if out.SuppressionSync == "" {
		out.SuppressionSync = "15m"
	}
	for k, v := range defaultThrottles {
		out.Throttles[k] = v
	}
	for k, v := range sd.Throttles {
		out.Throttles[k] = v
	}
	return out, nil
}

// SettingsProviderCost is a provider's cost this month.
type SettingsProviderCost struct {
	Basis   string  `json:"basis" enum:"metered,plan,self_hosted"`
	ToDate  float64 `json:"toDate"`
	Monthly float64 `json:"monthly"`
}

// SettingsProviderCard is one provider of Settings › Providers.
type SettingsProviderCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status" enum:"healthy,degraded,down,sandbox,warming_up,paused"`
	StatusLabel string `json:"statusLabel"`
	StatusTone  string `json:"statusTone" enum:"ok,info,warn,bad,neutral"`
	// Streams says what it carries ("Broadcasts 90% · Automations").
	Streams string `json:"streams"`
	// IdentityLabel and Identity name the account or server
	// ("Region · account", "us-east-1 · 123456789012").
	IdentityLabel string               `json:"identityLabel"`
	Identity      string               `json:"identity"`
	ConnectedAt   time.Time            `json:"connectedAt"`
	ConnectedVia  string               `json:"connectedVia"`
	Cost          SettingsProviderCost `json:"cost"`
}

// SettingsProviders is Settings › Providers.
type SettingsProviders struct {
	Providers []SettingsProviderCard `json:"providers"`
	Defaults  SendingDefaults        `json:"defaults"`
	// MonthStart is the first day of this month (the cost period).
	MonthStart time.Time `json:"monthStart"`
	CanManage  bool      `json:"canManage"`
}

// SettingsPage returns Settings › Providers.
func (s *Service) SettingsPage(ctx context.Context, ws *store.Workspace) (SettingsProviders, error) {
	out := SettingsProviders{Providers: []SettingsProviderCard{}, CanManage: canManage(ctx)}
	def, err := s.sendingDefaults(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Defaults = def
	ov, err := s.Providers.Overview(ctx, ws)
	if err != nil {
		return out, err
	}
	out.MonthStart = ov.Summary.MonthStart
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byID := map[string]store.Provider{}
	for _, p := range provs {
		byID[p.ID.String()] = p
	}
	for _, c := range ov.Providers {
		p := byID[c.ID]
		card := SettingsProviderCard{ID: c.ID, Name: c.Name, Status: c.Status, ConnectedAt: c.ConnectedAt, Streams: streamsText(c.Roles),
			Cost: SettingsProviderCost{Basis: c.Cost.Basis, ToDate: c.Cost.ToDate, Monthly: c.Cost.Monthly}}
		card.StatusLabel, card.StatusTone = statusLabel(c)
		card.IdentityLabel, card.Identity, card.ConnectedVia = identity(p)
		out.Providers = append(out.Providers, card)
	}
	return out, nil
}

func statusLabel(c manage.ProviderCard) (string, string) {
	switch c.Status {
	case "healthy":
		return "Healthy", "ok"
	case "warming_up":
		if c.Warmup != nil {
			return "Warming up · day " + strconv.Itoa(c.Warmup.Day), "info"
		}
		return "Warming up", "info"
	case "degraded":
		return "Degraded", "warn"
	case "down":
		return "Outage", "bad"
	case "sandbox":
		return "Sandbox", "warn"
	}
	return "Paused", "neutral"
}

// streamsText says what a provider carries: the streams it sends first
// ("Broadcasts 90% · Automations"), then the ones it backs up ("System ·
// Automations fallback"), then streams it gets only through rules.
func streamsText(roles []manage.ProviderStreamRole) string {
	parts := []string{}
	seen := map[string]bool{}
	for _, pass := range []string{"carries", "fallback", "condition"} {
		for _, r := range roles {
			if seen[r.Stream] {
				continue
			}
			name := streamName(r.Stream)
			switch {
			case pass == "carries" && r.Role == "split":
				parts = append(parts, name+" "+strconv.Itoa(r.Percent)+"%")
			case pass == "carries" && r.Role == "primary":
				parts = append(parts, name)
			case pass == "fallback" && r.Role == "fallback":
				parts = append(parts, name+" fallback")
			case pass == "condition" && r.Role == "condition":
				parts = append(parts, name+" (rules)")
			default:
				continue
			}
			seen[r.Stream] = true
		}
	}
	if len(parts) == 0 {
		return "Not routed yet"
	}
	return strings.Join(parts, " · ")
}

// identity names the account or server of a connection and how it was
// connected.
func identity(p store.Provider) (string, string, string) {
	var c struct {
		Region        string `json:"region"`
		AccountID     string `json:"account_id"`
		RoleARN       string `json:"role_arn"`
		Access        string `json:"access"`
		Server        string `json:"server"`
		MessageStream string `json:"message_stream"`
		ServerURL     string `json:"server_url"`
		Host          string `json:"host"`
		Port          int    `json:"port"`
	}
	_ = json.Unmarshal(p.Config, &c)
	switch p.Adapter {
	case "ses":
		account := c.AccountID
		if account == "" {
			if parts := strings.Split(c.RoleARN, ":"); len(parts) > 4 {
				account = parts[4]
			}
		}
		via := "IAM keys"
		if c.Access == "role" {
			via = "CloudFormation role"
		}
		return "Region · account", strings.Trim(c.Region+" · "+account, " ·"), via
	case "postmark":
		stream := "Transactional"
		if c.MessageStream != "" && c.MessageStream != "outbound" {
			stream = c.MessageStream
		}
		server := c.Server
		if server == "" {
			server = p.Label
		}
		return "Server · stream", strings.Trim(server+" · "+stream, " ·"), "server token"
	case "postal":
		host := c.ServerURL
		if u, err := url.Parse(c.ServerURL); err == nil && u.Host != "" {
			host = u.Host
		}
		ip := ""
		if len(p.Ips) > 0 {
			ip = p.Ips[0]
		}
		return "Server · IP", strings.Trim(host+" · "+ip, " ·"), "API key"
	case "smtp":
		server := c.Host
		if c.Port != 0 {
			server += ":" + strconv.Itoa(c.Port)
		}
		return "Server", server, "SMTP login"
	}
	label := p.Label
	if label == "" {
		label = p.StatusDetail
	}
	return "Account", label, "API key"
}

// SettingsProvidersInput saves the defaults cards.
type SettingsProvidersInput struct {
	Bounce           float64            `json:"bounce" minimum:"0.1" maximum:"50"`
	Complaint        float64            `json:"complaint" minimum:"0.01" maximum:"5"`
	MailboxComplaint float64            `json:"mailboxComplaint" minimum:"0.01" maximum:"5"`
	ProviderError    float64            `json:"providerError" minimum:"0.5" maximum:"100"`
	PauseScope       string             `json:"pauseScope" enum:"route,campaign"`
	ConfirmOver      int64              `json:"confirmOver" minimum:"1" maximum:"100000000"`
	Throttles        map[string]float64 `json:"throttles"`
	SuppressionSync  string             `json:"suppressionSync" enum:"15m,1h"`
}

// SaveSettings saves the workspace-wide sending defaults.
func (s *Service) SaveSettings(ctx context.Context, ws *store.Workspace, in SettingsProvidersInput) (SettingsProviders, error) {
	for k, v := range in.Throttles {
		if _, ok := defaultThrottles[k]; !ok {
			return SettingsProviders{}, api.Invalid("body.throttles."+k, "Throttles are set for gmail, outlook, yahoo and icloud.")
		}
		if v < 0 || v > 10000 {
			return SettingsProviders{}, api.Invalid("body.throttles."+k, "Enter messages per second, or 0 for no limit.")
		}
	}
	before, err := s.sendingDefaults(ctx, ws)
	if err != nil {
		return SettingsProviders{}, err
	}
	row, err := s.Q.SenderWorkspaceDefaults(ctx, ws.ID)
	if err != nil {
		return SettingsProviders{}, err
	}
	th := map[string]any{}
	_ = json.Unmarshal(workspaces.DefaultSendThresholds, &th)
	_ = json.Unmarshal(row.SendThresholds, &th)
	th["hard_bounce"], th["complaint"], th["mailbox_complaint"] = in.Bounce, in.Complaint, in.MailboxComplaint
	th["provider_error_rate"], th["scope"], th["confirm_over"] = in.ProviderError, in.PauseScope, in.ConfirmOver
	sd := map[string]any{}
	_ = json.Unmarshal(row.SendingDefaults, &sd)
	throttles := map[string]float64{}
	for k, v := range before.Throttles {
		throttles[k] = v
	}
	for k, v := range in.Throttles {
		throttles[k] = v
	}
	sd["throttles"], sd["suppression_sync"] = throttles, in.SuppressionSync
	thb, _ := json.Marshal(th)
	sdb, _ := json.Marshal(sd)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ConnectSetSendThresholds(ctx, store.ConnectSetSendThresholdsParams{ID: ws.ID, SendThresholds: thb, UpdatedAt: s.now()}); err != nil {
			return err
		}
		if err := q.SenderSetSendingDefaults(ctx, store.SenderSetSendingDefaultsParams{ID: ws.ID, SendingDefaults: sdb, UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sending.defaults_update", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: before, After: in})
	})
	if err != nil {
		return SettingsProviders{}, err
	}
	return s.SettingsPage(ctx, ws)
}

// SettingsRotationResult says what a rotation changed.
type SettingsRotationResult struct {
	Rotated int               `json:"rotated"`
	Skipped []string          `json:"skipped"`
	Page    SettingsProviders `json:"page"`
}

// RotateSecrets gives every provider a new webhook signing secret and
// registers the webhooks again where the provider holds the secret, so
// events keep verifying.
func (s *Service) RotateSecrets(ctx context.Context, ws *store.Workspace) (SettingsRotationResult, error) {
	provs, err := s.Q.ConnectProvidersWithSecrets(ctx, ws.ID)
	if err != nil {
		return SettingsRotationResult{}, err
	}
	out := SettingsRotationResult{Skipped: []string{}}
	now := s.now()
	for _, p := range provs {
		secret := randomToken(24)
		if !manage.IsDemo(p) {
			if d, ok := providers.Lookup(p.Adapter); ok && d.New != nil {
				creds, err := s.Factory.Credentials(p)
				if err != nil {
					out.Skipped = append(out.Skipped, p.Name)
					continue
				}
				a, err := d.New(providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: p.Config, Credentials: creds, WebhookSecret: secret,
					HTTPClient: s.httpClient()})
				if err != nil {
					out.Skipped = append(out.Skipped, p.Name)
					continue
				}
				if wr, ok := a.(providers.WebhookRegistrar); ok {
					cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
					err = wr.RegisterWebhook(cctx, s.webhookURL(p, ""))
					cancel()
					if err != nil {
						out.Skipped = append(out.Skipped, p.Name)
						continue
					}
				}
			}
		}
		sealed, err := s.Box.SealString(secret, manage.WebhookSecretAD(p))
		if err != nil {
			return out, err
		}
		if err := s.Q.ConnectSetWebhookSecret(ctx, store.ConnectSetWebhookSecretParams{ID: p.ID, WebhookSecretEnc: sealed}); err != nil {
			return out, err
		}
		out.Rotated++
	}
	row, err := s.Q.SenderWorkspaceDefaults(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	sd := map[string]any{}
	_ = json.Unmarshal(row.SendingDefaults, &sd)
	sd["webhook_secrets_rotated_at"] = now
	sdb, _ := json.Marshal(sd)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SenderSetSendingDefaults(ctx, store.SenderSetSendingDefaultsParams{ID: ws.ID, SendingDefaults: sdb, UpdatedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.webhook_secrets_rotate", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Metadata: map[string]any{"rotated": out.Rotated, "skipped": out.Skipped}})
	})
	if err != nil {
		return out, err
	}
	out.Page, err = s.SettingsPage(ctx, ws)
	return out, err
}
