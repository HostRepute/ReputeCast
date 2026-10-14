package connect

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ConnectSESInput is the AWS access of the SES path.
type ConnectSESInput struct {
	Access string `json:"access" enum:"role,keys"`
	// RoleARN is the stack's RoleArn output (CloudFormation path).
	RoleARN         string `json:"roleArn,omitempty" maxLength:"2048"`
	AccessKeyID     string `json:"accessKeyId,omitempty" maxLength:"128"`
	SecretAccessKey string `json:"secretAccessKey,omitempty" maxLength:"256"`
	Region          string `json:"region,omitempty" maxLength:"32"`
}

// ConnectSMTPInput is the outgoing server of the SMTP path.
type ConnectSMTPInput struct {
	Host     string `json:"host" minLength:"1" maxLength:"253"`
	Port     int    `json:"port" minimum:"1" maximum:"65535"`
	TLS      string `json:"tls" enum:"starttls,tls,none"`
	Username string `json:"username,omitempty" maxLength:"254"`
	// Password is kept as stored when empty.
	Password string `json:"password,omitempty" maxLength:"1024"`
	MaxRate  string `json:"maxRate,omitempty" maxLength:"40" doc:"Messages per second the server allows, e.g. 5 per second"`
	DailyCap string `json:"dailyCap,omitempty" maxLength:"40" doc:"Messages per day, e.g. 20,000"`
}

// ConnectMailboxInput is the bounce mailbox of the SMTP path.
type ConnectMailboxInput struct {
	Address  string `json:"address" maxLength:"254" doc:"The return-path address bounces go to"`
	Protocol string `json:"protocol" enum:"imap,pop3"`
	Host     string `json:"host" minLength:"1" maxLength:"253"`
	Port     int    `json:"port" minimum:"1" maximum:"65535"`
	TLS      string `json:"tls" enum:"tls,starttls,none"`
	Username string `json:"username" maxLength:"254"`
	// Password is kept as stored when empty.
	Password        string `json:"password,omitempty" maxLength:"1024"`
	Folder          string `json:"folder,omitempty" maxLength:"200"`
	PollEvery       string `json:"pollEvery,omitempty" enum:"1m,5m,15m"`
	AfterProcessing string `json:"afterProcessing,omitempty" enum:"move,delete,leave"`
}

// ConnectInput is the connection step's form.
type ConnectInput struct {
	// Values are the generic form's fields by key (settings and credentials).
	Values map[string]string `json:"values,omitempty"`
	// PublicURL is a tunnel address the provider calls instead of the
	// instance's public URL (testing on a laptop or a staging box).
	PublicURL string            `json:"publicUrl,omitempty" maxLength:"2048"`
	SES       *ConnectSESInput  `json:"ses,omitempty"`
	SMTP      *ConnectSMTPInput `json:"smtp,omitempty"`
}

// built is a candidate configuration for a provider.
type built struct {
	settings map[string]any
	creds    map[string]any
	label    string
	// token is the webhook token reserved on the setup for a new provider.
	token string
}

func (s *Service) currentCreds(prov *store.Provider) map[string]any {
	out := map[string]any{}
	if prov == nil || s.Factory == nil {
		return out
	}
	if raw, err := s.Factory.Credentials(*prov); err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func currentSettings(prov *store.Provider) map[string]any {
	out := map[string]any{}
	if prov != nil {
		_ = json.Unmarshal(prov.Config, &out)
	}
	return out
}

// Connect checks the credentials with the provider, then creates (or
// updates) the provider, sealed credentials included, and sets up how its
// events come back: SES provisioning, webhook registration behind a
// reachability test of the public URL, or the SMTP bounce mailbox. A
// rejected key is recorded on the setup (the error state), not saved.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, rawID string, in ConnectInput) (ConnectSetup, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	if st.Status == "abandoned" {
		return ConnectSetup{}, api.Conflict("This setup was cancelled. Start again from the catalog.")
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	c := copyFor(d)
	var prov *store.Provider
	if st.ProviderID != nil {
		if p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: *st.ProviderID}); err == nil {
			prov = &p
		}
	}
	state := decodeState(st.State)
	flow := flowOf(d)
	var b built
	switch flow {
	case "ses":
		if in.SES == nil {
			return ConnectSetup{}, api.Invalid("body.ses", "Choose how ReputeCast gets AWS access.")
		}
		b, err = s.sesConfig(prov, state, *in.SES)
	case "smtp":
		if in.SMTP == nil {
			return ConnectSetup{}, api.Invalid("body.smtp", "Enter the SMTP server.")
		}
		b, err = s.smtpConfig(prov, *in.SMTP)
	default:
		b, err = s.apiConfig(d, prov, in.Values)
	}
	if err != nil {
		return ConnectSetup{}, err
	}
	providerID := uuid.Must(uuid.NewV7())
	if prov != nil {
		providerID = prov.ID
	}
	if flow == "api" && d.Capabilities.Webhooks != "" {
		if state.Connection.Webhook == nil {
			state.Connection.Webhook = &WebhookState{Mode: "webhook"}
		}
		w := state.Connection.Webhook
		if w.Token == "" {
			w.Token = randomToken(16)
		}
		b.token = w.Token
		if in.PublicURL != "" {
			tunnel, err := tunnelURL(in.PublicURL)
			if err != nil {
				return ConnectSetup{}, err
			}
			w.PublicURL = tunnel
		}
		if w.PublicURL != "" {
			b.settings["webhook_base_url"] = w.PublicURL
		}
		if w.Mode == "polling" {
			b.settings["events"] = "polling"
		}
	}
	webhookSecret := randomToken(24)
	if prov != nil && s.Factory != nil {
		if sec := s.Factory.WebhookSecret(*prov); sec != "" {
			webhookSecret = sec
		}
	}
	adapter, err := newAdapter(d, providerID, ws.ID, b, webhookSecret, s.httpClient())
	if err != nil {
		return s.connectFailed(ctx, ws, st, state, b, &StepError{Code: "invalid_settings", Message: "These settings can't be used: " + err.Error()})
	}
	now := s.now()
	quota := providers.Quota{}
	accountID := ""
	var probe *ProbeState
	switch flow {
	case "smtp":
		ct, ok := adapter.(providers.ConnectionTester)
		if !ok {
			break
		}
		if err := s.checkHost(ctx, stringOf(b.settings["host"])); err != nil {
			return s.connectFailed(ctx, ws, st, state, b, hostBlocked("host", err))
		}
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		rep, err := ct.TestConnection(cctx)
		cancel()
		probe = &ProbeState{OK: err == nil, At: now, Summary: rep.Summary, Transcript: rep.Transcript}
		if err != nil {
			probe.Error = callMessage(err)
			state.Connection.SMTP = probe
			return s.connectFailed(ctx, ws, st, state, b, smtpError(err))
		}
		// Sends use the login method the server just accepted.
		if a := stringOf(b.settings["auth"]); rep.Auth != "" && (a == "" || a == "auto") {
			b.settings["auth"] = rep.Auth
		}
	default:
		if v, ok := adapter.(providers.Verifier); ok {
			cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			quota, err = v.Verify(cctx)
			cancel()
			if err != nil {
				return s.connectFailed(ctx, ws, st, state, b, keyError(d, c, b, err))
			}
		}
		if ai, ok := adapter.(providers.AccountIdentifier); ok {
			cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			accountID, _ = ai.AccountID(cctx)
			cancel()
		}
	}
	if accountID != "" {
		b.settings["account_id"] = accountID
	}
	p, err := s.saveProvider(ctx, ws, d, c, prov, providerID, b, webhookSecret, quota, now)
	if err != nil {
		return ConnectSetup{}, err
	}
	state.Connection.Status, state.Connection.At, state.Connection.Error = "connected", &now, nil
	state.Connection.Account, state.Connection.AccountID = quota.Account, accountID
	state.Connection.Settings = map[string]string{}
	for k, v := range b.settings {
		if s, ok := v.(string); ok {
			state.Connection.Settings[k] = s
		}
	}
	switch flow {
	case "ses":
		state.Connection.Access, state.Connection.Region = in.SES.Access, stringOf(b.settings["region"])
		state.Connection.RoleARN = stringOf(b.settings["role_arn"])
		state.Connection.Summary = state.Connection.Region + " · " + map[bool]string{true: "sandbox", false: "production"}[quota.Sandbox]
		s.startProvisioning(ctx, p, &state, quota, now)
	case "smtp":
		state.Connection.SMTP = probe
		state.Connection.Summary = stringOf(b.settings["host"])
		state.Connection.Webhook = &WebhookState{Mode: "mailbox"}
	default:
		state.Connection.Summary = connectionSummary(quota, b)
		s.setupEvents(ctx, d, c, p, &state, now)
	}
	st.ProviderID = &p.ID
	out, err := s.saveSetup(ctx, ws, st, state, "provider.connect_check", map[string]any{"adapter": d.Adapter, "result": "connected"})
	if err != nil {
		return ConnectSetup{}, err
	}
	if flow == "ses" {
		s.queueProvisioning(ctx, out.ID)
	}
	return s.View(ctx, ws, out)
}

func stringOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// connectionSummary is the stepper's line for an API provider.
func connectionSummary(q providers.Quota, b built) string {
	parts := []string{}
	if b.label != "" {
		parts = append(parts, b.label)
	}
	if q.Account != "" {
		parts = append(parts, strings.ToLower(strings.SplitN(q.Account, " · ", 2)[0]))
	}
	if len(parts) == 0 {
		return "connected"
	}
	return strings.Join(parts, " · ")
}

// saveSetup writes the setup with an audit entry.
func (s *Service) saveSetup(ctx context.Context, ws *store.Workspace, st store.ProviderSetup, state State, action string, meta map[string]any) (store.ProviderSetup, error) {
	var out store.ProviderSetup
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if out, err = s.save(ctx, q, st, state); err != nil {
			return err
		}
		if action == "" {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "provider_setup", TargetID: st.ID.String(), TargetName: st.Adapter, Metadata: meta})
	})
	return out, err
}

// connectFailed records a rejected connection on the setup (the error
// state) and returns the setup.
func (s *Service) connectFailed(ctx context.Context, ws *store.Workspace, st store.ProviderSetup, state State, b built, e *StepError) (ConnectSetup, error) {
	now := s.now()
	state.Connection.Status, state.Connection.At, state.Connection.Error = "failed", &now, e
	// Whether the provider could reach the webhook endpoint is known before
	// the key works: a private or local public URL fails up front.
	if w := state.Connection.Webhook; w != nil && w.Mode == "webhook" && w.Token != "" && st.ProviderID == nil {
		w.URL = s.webhookBase(w.PublicURL) + "/hooks/" + st.Adapter + "/" + w.Token
		w.SelfTest = Preflight(ctx, w.URL, now, s.AllowPrivateURL)
		w.Registration = nil
		if w.SelfTest != nil {
			w.Registration = &Registration{Status: "skipped", At: now, Detail: "after the endpoint is reachable"}
		}
	}
	if state.Connection.Settings == nil {
		state.Connection.Settings = map[string]string{}
	}
	for k, v := range b.settings {
		if s, ok := v.(string); ok {
			state.Connection.Settings[k] = s
		}
	}
	wsID := ws.ID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := s.save(ctx, q, st, state); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "provider.connect_check", TargetType: "provider_setup", TargetID: st.ID.String(), TargetName: st.Adapter,
			Result: audit.Failed, Reason: e.Message, Metadata: map[string]any{"code": e.Code}})
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	cur, err := s.Q.SetupGetAny(ctx, st.ID)
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, cur)
}

// httpClient is the client provider calls go through: the factory's, which
// the SSRF guard protects (private networks only when the instance allows
// them; link-local and cloud metadata addresses never).
func (s *Service) httpClient() *http.Client {
	if s.Factory != nil && s.Factory.HTTP != nil {
		return s.Factory.HTTP
	}
	return netguard.Policy{}.Client(30 * time.Second)
}

// urlField is the form field that holds the provider's address, if any.
func urlField(d providers.Descriptor) string {
	for _, f := range d.Fields {
		if !f.Secret && (strings.Contains(f.Key, "url") || strings.Contains(f.Key, "endpoint") || strings.Contains(f.Key, "host")) {
			return f.Key
		}
	}
	return ""
}

// newAdapter builds the adapter with the candidate configuration. Form
// values arrive as strings; when the adapter wants numbers, a second try
// passes digit-only values as numbers.
func newAdapter(d providers.Descriptor, providerID, wsID uuid.UUID, b built, webhookSecret string, client *http.Client) (providers.Provider, error) {
	try := func(settings map[string]any) (providers.Provider, error) {
		sb, _ := json.Marshal(settings)
		cb, _ := json.Marshal(b.creds)
		return d.New(providers.Config{ProviderID: providerID, WorkspaceID: wsID, Settings: sb, Credentials: cb, WebhookSecret: webhookSecret, HTTPClient: client})
	}
	a, err := try(b.settings)
	if err == nil {
		return a, nil
	}
	numeric := map[string]any{}
	for k, v := range b.settings {
		if s, ok := v.(string); ok {
			if n, err := strconv.ParseInt(strings.ReplaceAll(s, ",", ""), 10, 64); err == nil {
				numeric[k] = n
				continue
			}
		}
		numeric[k] = v
	}
	if a2, err2 := try(numeric); err2 == nil {
		for k, v := range numeric {
			b.settings[k] = v
		}
		return a2, nil
	}
	return nil, err
}

// apiConfig splits the generic form into settings and credentials, keeping
// stored secrets the person didn't replace.
func (s *Service) apiConfig(d providers.Descriptor, prov *store.Provider, values map[string]string) (built, error) {
	values = clean(values)
	b := built{settings: currentSettings(prov), creds: s.currentCreds(prov)}
	delete(b.settings, "demo")
	for _, f := range d.Fields {
		v := values[f.Key]
		switch {
		case f.Secret && v != "":
			b.creds[f.Key] = v
		case f.Secret:
			if _, ok := b.creds[f.Key]; !ok && f.Required {
				return b, api.Invalid("body.values."+f.Key, "Enter the "+strings.ToLower(f.Label)+".")
			}
		case v != "" && fieldType(f) == fieldNumber:
			n, ok := wholeNumber(v)
			if !ok {
				return b, api.Invalid("body.values."+f.Key, "Enter a whole number, like 10000.")
			}
			b.settings[f.Key] = n
		case v != "":
			b.settings[f.Key] = v
		case f.Required:
			if _, ok := b.settings[f.Key]; !ok {
				return b, api.Invalid("body.values."+f.Key, "Enter the "+strings.ToLower(f.Label)+".")
			}
		}
		if !f.Secret && len(f.Options) > 0 && v != "" && !contains(f.Options, v) {
			return b, api.Invalid("body.values."+f.Key, "Pick one of the listed values.")
		}
	}
	for _, key := range []string{"region", "server", "message_stream", "server_url"} {
		if v := stringOf(b.settings[key]); v != "" {
			if u, err := url.Parse(v); err == nil && u.Host != "" {
				v = u.Host
			}
			b.label = v
			break
		}
	}
	return b, nil
}

// fieldNumber is the form type of a field that holds a whole number.
const fieldNumber = "number"

// fieldType is a form field's type: "number" for the fields adapters read
// as whole numbers (ports, allowances and rate limits, which the flex
// settings types also accept as text), else "text".
func fieldType(f providers.Field) string {
	if f.Secret || len(f.Options) > 0 {
		return "text"
	}
	k := f.Key
	if k == "port" || strings.HasSuffix(k, "_limit") || strings.HasPrefix(k, "per_") || strings.HasSuffix(k, "_per_day") ||
		strings.HasSuffix(k, "_per_hour") || strings.HasSuffix(k, "_per_minute") || strings.HasSuffix(k, "_cap") {
		return fieldNumber
	}
	return "text"
}

// wholeNumber reads "10000", "10,000" or "10 000" as a number.
func wholeNumber(v string) (int64, bool) {
	clean := strings.NewReplacer(",", "", "_", "", " ", "", "\u00a0", "").Replace(strings.TrimSpace(v))
	n, err := strconv.ParseInt(clean, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

var roleARN = regexp.MustCompile(`^arn:aws[a-z-]*:iam::\d{12}:role/[\w+=,.@/-]+$`)

// sesConfig is the SES path: role (from the stack's output) or keys.
func (s *Service) sesConfig(prov *store.Provider, state State, in ConnectSESInput) (built, error) {
	b := built{settings: currentSettings(prov), creds: map[string]any{}}
	delete(b.settings, "demo")
	region := strings.TrimSpace(in.Region)
	if region == "" {
		region = state.Connection.Region
	}
	if region == "" {
		region = "us-east-1"
	}
	b.settings["region"], b.settings["access"] = region, in.Access
	b.label = region
	switch in.Access {
	case "role":
		arn := strings.TrimSpace(in.RoleARN)
		if arn == "" {
			arn = state.Connection.RoleARN
		}
		if !roleARN.MatchString(arn) {
			return b, api.Invalid("body.ses.roleArn", "Paste the RoleArn output of the stack, like arn:aws:iam::123456789012:role/ReputeCastSES.")
		}
		b.settings["role_arn"], b.settings["external_id"] = arn, state.Connection.ExternalID
	default:
		cur := s.currentCreds(prov)
		id, secret := strings.TrimSpace(in.AccessKeyID), strings.TrimSpace(in.SecretAccessKey)
		if id == "" {
			id = stringOf(cur["access_key_id"])
		}
		if secret == "" {
			secret = stringOf(cur["secret_access_key"])
		}
		if !strings.HasPrefix(id, "AKIA") && !strings.HasPrefix(id, "ASIA") || len(id) < 16 {
			return b, api.Invalid("body.ses.accessKeyId", "Paste an access key id; it starts with AKIA.")
		}
		if secret == "" {
			return b, api.Invalid("body.ses.secretAccessKey", "Paste the secret access key.")
		}
		delete(b.settings, "role_arn")
		delete(b.settings, "external_id")
		b.creds["access_key_id"], b.creds["secret_access_key"] = id, secret
	}
	return b, nil
}

// numberIn reads the first number of "5 per second" or "20,000".
func numberIn(s string) float64 {
	m := regexp.MustCompile(`[0-9][0-9,]*(\.[0-9]+)?`).FindString(s)
	f, _ := strconv.ParseFloat(strings.ReplaceAll(m, ",", ""), 64)
	return f
}

// smtpConfig is the SMTP path's outgoing server.
func (s *Service) smtpConfig(prov *store.Provider, in ConnectSMTPInput) (built, error) {
	b := built{settings: currentSettings(prov), creds: s.currentCreds(prov)}
	delete(b.settings, "demo")
	host := strings.ToLower(strings.TrimSpace(in.Host))
	if strings.ContainsAny(host, " /:@") || host == "" {
		return b, api.Invalid("body.smtp.host", "Enter the server's host name, like smtp.example.com.")
	}
	b.settings["host"], b.settings["port"], b.settings["tls"] = host, in.Port, in.TLS
	b.settings["username"] = strings.TrimSpace(in.Username)
	if in.MaxRate != "" {
		b.settings["max_rate"] = in.MaxRate
	}
	if in.DailyCap != "" {
		b.settings["daily_cap"] = in.DailyCap
	}
	if in.Password != "" {
		b.creds["password"] = in.Password
	}
	if b.settings["helo"] == nil {
		if u, err := url.Parse(s.URLs.Base()); err == nil && u.Hostname() != "" && u.Hostname() != "localhost" {
			b.settings["helo"] = u.Hostname()
		}
	}
	b.label = host
	return b, nil
}

// keyError explains a rejected credential check.
func keyError(d providers.Descriptor, c adapterCopy, b built, err error) *StepError {
	field := ""
	for _, f := range d.Fields {
		if f.Secret {
			field = f.Key
			break
		}
	}
	var ce *providers.CallError
	detail := err.Error()
	if errors.As(err, &ce) {
		parts := []string{}
		if ce.Status != 0 {
			parts = append(parts, strconv.Itoa(ce.Status)+" "+http.StatusText(ce.Status))
		}
		if ce.Code != "" {
			parts = append(parts, ce.Code)
		}
		if len(parts) == 0 && ce.Message != "" {
			parts = append(parts, ce.Message)
		}
		detail = strings.Join(parts, " · ")
	}
	if d.Adapter == "ses" {
		if stringOf(b.settings["access"]) == "role" {
			field = "role_arn"
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "ec2 imds") || strings.Contains(low, "no valid credential") || strings.Contains(low, "failed to refresh cached credentials") ||
				strings.Contains(low, "get identity") || strings.Contains(low, "could not find") {
				return &StepError{Field: field, Code: "no_host_identity", Detail: detail,
					Message: "ReputeCast couldn't assume the role: this server has no AWS credentials of its own to assume it with. Run ReputeCast with an AWS identity (an instance role or AWS_* variables), or use IAM access keys."}
			}
			return &StepError{Field: field, Code: "role_rejected", Detail: detail,
				Message: "AWS didn't let ReputeCast assume this role. Check that the stack finished, that the ARN is its RoleArn output, and that the external id wasn't changed."}
		}
		field = "access_key_id"
	}
	if netguard.IsBlocked(err) {
		why := netguard.Describe(context.Background(), err)
		return &StepError{Field: urlField(d), Code: "address_blocked", Detail: why,
			Message: "ReputeCast won't connect to that address: `" + strings.ReplaceAll(why, "`", "'") + "`. Provider URLs can't point at private, loopback or cloud metadata addresses unless an instance admin turns on providers.allow_private_networks."}
	}
	if errors.Is(err, providers.ErrUnauthorized) || ce != nil && ce.Auth {
		// "Mailgun rejected this key: `401 Unauthorized · Forbidden`. Check…"
		msg := d.Name + " rejected this key"
		if detail != "" {
			msg += ": `" + strings.ReplaceAll(detail, "`", "'") + "`"
		}
		msg += "."
		if c.KeyHint != "" {
			msg += " " + c.KeyHint
		}
		return &StepError{Field: field, Code: "invalid_credentials", Detail: detail, Message: msg}
	}
	return &StepError{Field: field, Code: "provider_unavailable", Detail: detail,
		Message: d.Name + " didn't answer the check. Nothing was saved; try again in a moment."}
}

// checkHost refuses a mail server host that is, or resolves to, a loopback,
// private, link-local or cloud metadata address, unless an instance admin
// allowed private networks: the connection tests show what a server
// answers, so they must not reach internal services. (A name that doesn't
// resolve is left to the test, which says so.)
func (s *Service) checkHost(ctx context.Context, host string) error {
	policy := netguard.Policy{AllowPrivate: s.AllowPrivateNetworks}
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		return policy.Check(ip.Unmap())
	}
	low := strings.ToLower(host)
	if low == "localhost" || strings.HasSuffix(low, ".localhost") || strings.HasSuffix(low, ".internal") {
		if !policy.AllowPrivate {
			return fmt.Errorf("%w: %s is a local host name", netguard.ErrBlocked, host)
		}
		return nil
	}
	lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupNetIP(lctx, "ip", host)
	if err != nil {
		return nil
	}
	for _, ip := range ips {
		if err := policy.Check(ip.Unmap()); err != nil {
			return err
		}
	}
	return nil
}

// hostBlocked explains a mail server address the policy refuses.
func hostBlocked(field string, err error) *StepError {
	why := strings.ReplaceAll(err.Error(), "`", "'")
	return &StepError{Field: field, Code: "address_blocked", Detail: why,
		Message: "ReputeCast won't connect to that address: `" + why + "`. Mail servers can't be on private, loopback or cloud metadata addresses unless an instance admin turns on providers.allow_private_networks."}
}

func smtpError(err error) *StepError {
	var ce *providers.CallError
	field := "host"
	if errors.As(err, &ce) && ce.Auth {
		field = "password"
	}
	return &StepError{Field: field, Code: "smtp_failed", Message: callMessage(err), Detail: callDetail(err)}
}

func callMessage(err error) string {
	var ce *providers.CallError
	if errors.As(err, &ce) && ce.Message != "" {
		return ce.Message
	}
	return err.Error()
}

func callDetail(err error) string {
	var ce *providers.CallError
	if errors.As(err, &ce) {
		if ce.Status != 0 {
			return ce.Op + " · " + strconv.Itoa(ce.Status)
		}
		return ce.Op
	}
	return ""
}

// pricingOf turns the adapter's pricing summary into the provider's
// pricing ("$0.10 per 1,000" → metered).
func pricingOf(d providers.Descriptor) state.Pricing {
	p := strings.ToLower(d.Capabilities.Pricing)
	switch {
	case d.Kind == "self_hosted" || d.Kind == "smtp" || strings.Contains(p, "self-hosted") || strings.Contains(p, "your server"):
		return state.Pricing{Model: "self_hosted"}
	case strings.Contains(p, "plan"):
		return state.Pricing{Model: "plan"}
	}
	if m := regexp.MustCompile(`\$([0-9.]+) per 1,000`).FindStringSubmatch(p); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		return state.Pricing{Model: "metered", Per1000: v, Source: d.Name + " pricing"}
	}
	return state.Pricing{Model: "metered"}
}

// saveProvider creates or updates the provider with sealed credentials.
func (s *Service) saveProvider(ctx context.Context, ws *store.Workspace, d providers.Descriptor, c adapterCopy, prov *store.Provider, id uuid.UUID,
	b built, webhookSecret string, quota providers.Quota, now time.Time) (store.Provider, error) {
	ref := store.Provider{ID: id}
	credsJSON, _ := json.Marshal(b.creds)
	var sealed []byte
	if len(b.creds) > 0 {
		var err error
		if sealed, err = s.Box.Seal(credsJSON, manage.CredentialsAD(ref)); err != nil {
			return store.Provider{}, err
		}
	}
	cfg, _ := json.Marshal(b.settings)
	statusWord, detail := state.Healthy, quota.Account
	if quota.Sandbox {
		statusWord = state.Sandbox
	}
	ok := now
	h := state.Health{Account: quota.Account, Sandbox: quota.Sandbox, LastSuccessAt: &ok, LastSuccessOp: c.VerifyCall}
	if quota.Kind != "" {
		h.Quota = &state.Quota{Kind: quota.Kind, Max: quota.Max, Used: quota.Used}
	}
	if quota.MaxRate > 0 {
		h.Rate = &state.Rate{Max: quota.MaxRate}
	}
	if flowOf(d) == "smtp" {
		if capN := numberIn(stringOf(b.settings["daily_cap"])); capN > 0 {
			h.Quota = &state.Quota{Kind: "rolling_24h", Max: int64(capN)}
		}
		if rate := numberIn(stringOf(b.settings["max_rate"])); rate > 0 {
			h.Rate = &state.Rate{Max: rate, Label: "set manually"}
		}
		if detail == "" {
			detail = stringOf(b.settings["host"]) + " · accepting mail"
		}
	}
	health, _ := json.Marshal(h)
	var out store.Provider
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if prov != nil {
			var err error
			out, err = q.ConnectUpdateProvider(ctx, store.ConnectUpdateProviderParams{ID: id, Label: b.label, Config: cfg, CredentialsEnc: sealed, Status: statusWord,
				StatusDetail: detail, IpMode: prov.IpMode, Ips: prov.Ips, Health: health, CheckedAt: &now})
			if err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.credentials_replace", TargetType: "provider", TargetID: id.String(), TargetName: prov.Name,
				Metadata: map[string]any{"source": "setup", "keys": keysOf(b.creds)}})
		}
		secretSealed, err := s.Box.SealString(webhookSecret, manage.WebhookSecretAD(ref))
		if err != nil {
			return err
		}
		caps, _ := json.Marshal(state.Snapshot{Capabilities: d.Capabilities})
		limits := state.Limits{Bounce: state.Limit{Limit: 5, Label: c.Short}, Complaint: state.Limit{Limit: 0.1, Label: c.Short}}
		if d.Adapter == "ses" {
			limits = state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}}
		}
		lb, _ := json.Marshal(limits)
		pb, _ := json.Marshal(pricingOf(d))
		ipMode := "shared"
		if d.Kind == "self_hosted" {
			ipMode = "dedicated"
		}
		name := s.providerName(ctx, ws.ID, d, b.label)
		out, err = q.ConnectInsertProvider(ctx, store.ConnectInsertProviderParams{ID: id, WorkspaceID: ws.ID, Adapter: d.Adapter, Name: name, Label: b.label, Config: cfg,
			CredentialsEnc: sealed, WebhookToken: cmp.Or(b.token, randomToken(16)), WebhookSecretEnc: secretSealed, Status: statusWord, StatusDetail: detail, IpMode: ipMode, Ips: []string{},
			Capabilities: caps, Limits: lb, Health: health, Pricing: pb, ConnectedBy: principalUser(ctx), ConnectedAt: now})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.connect", TargetType: "provider", TargetID: id.String(), TargetName: name,
			After: map[string]any{"adapter": d.Adapter, "label": b.label, "status": statusWord}})
	})
	return out, err
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// webhookURL is where a provider posts events: the instance's public URL,
// or the setup's tunnel URL while testing.
func (s *Service) webhookURL(p store.Provider, override string) string {
	base := strings.TrimRight(override, "/")
	if base == "" {
		var cfg struct {
			Base string `json:"webhook_base_url"`
		}
		_ = json.Unmarshal(p.Config, &cfg)
		base = strings.TrimRight(cfg.Base, "/")
	}
	if base == "" {
		base = s.webhookBase("")
	}
	return base + "/hooks/" + p.Adapter + "/" + p.WebhookToken
}

// webhookBase is the address providers call: the override (a tunnel) or
// the instance's public URL.
func (s *Service) webhookBase(override string) string {
	if base := strings.TrimRight(override, "/"); base != "" {
		return base
	}
	return s.URLs.Base()
}

// setupEvents tests the webhook endpoint the way the provider would reach
// it and, when reachable, registers it at the provider.
func (s *Service) setupEvents(ctx context.Context, d providers.Descriptor, c adapterCopy, p store.Provider, st *State, now time.Time) {
	w := st.Connection.Webhook
	if w == nil {
		w = &WebhookState{Mode: "webhook"}
	}
	if d.Capabilities.Webhooks == "" {
		w.Mode = "none"
		st.Connection.Webhook = w
		return
	}
	if w.Mode == "polling" {
		st.Connection.Webhook = w
		return
	}
	w.Mode = "webhook"
	w.URL = s.webhookURL(p, w.PublicURL)
	reach := s.reachable(ctx, w.URL, p)
	w.SelfTest = &reach
	if !reach.OK {
		w.Registration = &Registration{Status: "skipped", At: now, Detail: "after the endpoint is reachable"}
		st.Connection.Webhook = w
		return
	}
	if manage.IsDemo(p) {
		w.Registration = &Registration{Status: "done", At: now}
		st.Connection.Webhook = w
		return
	}
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		w.Registration = &Registration{Status: "failed", At: now, Detail: err.Error()}
		st.Connection.Webhook = w
		return
	}
	wr, ok := adapter.(providers.WebhookRegistrar)
	if !ok {
		w.Registration = &Registration{Status: "manual", At: now, Detail: "Add this URL as a webhook in " + d.Name + ": " + w.URL}
		w.NeedsSecret, w.SecretHeader = manualSecret(adapter, s.currentCreds(&p))
		st.Connection.Webhook = w
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err = wr.RegisterWebhook(cctx, w.URL)
	cancel()
	if err != nil {
		w.Registration = &Registration{Status: "failed", At: now, Detail: callMessage(err)}
	} else {
		w.Registration = &Registration{Status: "done", At: now, Detail: c.WebhookEvents}
	}
	st.Connection.Webhook = w
}

// manualSecret says whether a provider whose webhook is set up by hand
// authenticates it with the secret ReputeCast generated (it has no signing
// key of its own among the credentials), and the header that carries it.
func manualSecret(a providers.Provider, creds map[string]any) (bool, string) {
	if _, ok := a.(providers.WebhookVerifier); !ok {
		return false, ""
	}
	for k, v := range creds {
		if strings.Contains(k, "webhook") && fmt.Sprint(v) != "" {
			return false, ""
		}
	}
	header := ""
	if ws, ok := a.(providers.WebhookSecrets); ok {
		if hs := ws.WebhookSecretHeaders(); len(hs) > 0 {
			header = hs[0]
		}
	}
	return true, header
}

// ConnectWebhookSecret is the secret a provider set up by hand sends with
// its webhooks, shown to the person who configures it.
type ConnectWebhookSecret struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
	// Header carries the secret as it is ("X-ReputeCast-Secret"); empty when
	// the provider signs its requests with it instead.
	Header string `json:"header,omitempty"`
}

// WebhookSecret issues a new webhook secret for a provider whose webhook is
// set up by hand, and returns it this once: like API keys, a stored secret is
// never read back. The new secret replaces the previous one, in the audit log.
func (s *Service) WebhookSecret(ctx context.Context, ws *store.Workspace, rawID string) (ConnectWebhookSecret, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectWebhookSecret{}, err
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectWebhookSecret{}, err
	}
	w := decodeState(st.State).Connection.Webhook
	if w == nil || !w.NeedsSecret {
		return ConnectWebhookSecret{}, api.Conflict("This provider's webhooks don't use a secret from ReputeCast.")
	}
	secret := randomToken(24)
	sealed, err := s.Box.SealString(secret, manage.WebhookSecretAD(p))
	if err != nil {
		return ConnectWebhookSecret{}, err
	}
	wsID := ws.ID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ConnectSetWebhookSecret(ctx, store.ConnectSetWebhookSecretParams{ID: p.ID, WebhookSecretEnc: sealed}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "provider.webhook_secret_rotate", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
			Metadata: map[string]any{"reason": "issued for the webhook set up in the provider"}})
	})
	if err != nil {
		return ConnectWebhookSecret{}, err
	}
	return ConnectWebhookSecret{URL: w.URL, Secret: secret, Header: w.SecretHeader}, nil
}

// reachable runs the endpoint self-test (tests replace it).
func (s *Service) reachable(ctx context.Context, url string, p store.Provider) SelfTestState {
	if s.Reachable != nil {
		return s.Reachable(ctx, url, p)
	}
	return SelfTest(ctx, url, p.WebhookToken, s.now(), s.AllowPrivateURL)
}

// SelfTest checks that a webhook URL can be reached from the internet:
// localhost and private addresses fail up front (no provider can reach
// them), then the endpoint must answer a signed self-test with 200.
func SelfTest(ctx context.Context, rawURL, token string, now time.Time, allowPrivate bool) SelfTestState {
	out := SelfTestState{At: now}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		out.Code, out.Reason = "EINVAL", "the public URL isn't a valid URL"
		return out
	}
	if allowPrivate {
		return selfTestPost(ctx, out, u, rawURL, token, now)
	}
	if bad, ok := unreachable(ctx, u, now); ok {
		return bad
	}
	return selfTestPost(ctx, out, u, rawURL, token, now)
}

// Preflight checks, before anything is connected, whether providers could
// reach a webhook URL at all: localhost, private addresses and names that
// don't resolve fail up front. It returns nil when the URL may be
// reachable (the self-test proper needs the connected provider).
func Preflight(ctx context.Context, rawURL string, now time.Time, allowPrivate bool) *SelfTestState {
	if allowPrivate {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return &SelfTestState{At: now, Code: "EINVAL", Reason: "the public URL isn't a valid URL"}
	}
	if bad, ok := unreachable(ctx, u, now); ok {
		return &bad
	}
	return nil
}

// unreachable reports a URL no provider can reach from the internet.
func unreachable(ctx context.Context, u *url.URL, now time.Time) (SelfTestState, bool) {
	out := SelfTestState{At: now}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".lan") {
		out.Code, out.Reason, out.Private = "ECONNREFUSED", host+" only works on this machine", true
		return out, true
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if !manage.PublicIP(ip) {
			out.Code, out.Reason, out.Private = "ECONNREFUSED", ip.String()+" is a private address", true
			return out, true
		}
		return out, false
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	addrs, err := net.DefaultResolver.LookupNetIP(cctx, "ip", host)
	cancel()
	if err != nil || len(addrs) == 0 {
		out.Code, out.Reason = "ENOTFOUND", host+" doesn't resolve in public DNS"
		return out, true
	}
	for _, a := range addrs {
		if manage.PublicIP(a) {
			return out, false
		}
	}
	out.Code, out.Reason, out.Private = "ECONNREFUSED", "DNS resolves to "+addrs[0].Unmap().String()+", a private address", true
	return out, true
}

// selfTestPost posts a signed self-test to the endpoint and expects 200.
func selfTestPost(ctx context.Context, out SelfTestState, u *url.URL, rawURL, token string, now time.Time) SelfTestState {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, rawURL, strings.NewReader(`{"selfTest":true}`))
	if err != nil {
		out.Code, out.Reason = "EINVAL", err.Error()
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(manage.SelfTestHeader, manage.SelfTestSignature(token, now.Unix()))
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		low := strings.ToLower(err.Error())
		switch {
		case strings.Contains(low, "refused"):
			out.Code, out.Reason = "ECONNREFUSED", "nothing answers at "+u.Host
		case strings.Contains(low, "timeout") || strings.Contains(low, "deadline"):
			out.Code, out.Reason = "ETIMEDOUT", "no answer within 10 s (a firewall may block it)"
		case strings.Contains(low, "certificate") || strings.Contains(low, "tls"):
			out.Code, out.Reason = "TLS", "the HTTPS certificate isn't valid for "+u.Host
		default:
			out.Code, out.Reason = "ERROR", err.Error()
		}
		return out
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		out.Code, out.Reason = "HTTP "+strconv.Itoa(resp.StatusCode), "the reverse proxy doesn't pass /hooks/ to ReputeCast"
		return out
	}
	out.OK = true
	return out
}

// ConnectWebhookInput re-tests the webhook endpoint, optionally through a
// tunnel URL.
type ConnectWebhookInput struct {
	// PublicURL replaces the instance's public URL for this provider's
	// webhooks (a Cloudflare Tunnel or ngrok address while testing).
	PublicURL string `json:"publicUrl,omitempty" maxLength:"2048"`
}

// tunnelURL checks a tunnel address: https, no path. Empty stays empty.
func tunnelURL(raw string) (string, error) {
	tunnel := strings.TrimRight(strings.TrimSpace(raw), "/")
	if tunnel == "" {
		return "", nil
	}
	u, err := url.Parse(tunnel)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" && u.Path != "/" {
		return "", api.Invalid("body.publicUrl", "Paste the tunnel's https:// address, without a path.")
	}
	return tunnel, nil
}

// RetestWebhook tests the endpoint again and registers it when reachable.
func (s *Service) RetestWebhook(ctx context.Context, ws *store.Workspace, rawID string, in ConnectWebhookInput) (ConnectSetup, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	if st.ProviderID == nil {
		// Not connected yet: check that providers could reach the endpoint
		// (through the tunnel, when given); the self-test proper follows the
		// connection.
		tunnel, err := tunnelURL(in.PublicURL)
		if err != nil {
			return ConnectSetup{}, err
		}
		out, err := s.update(ctx, ws, st.ID, "provider.webhook_test", map[string]any{"tunnel": tunnel != "", "connected": false}, func(st *store.ProviderSetup, state *State) error {
			if state.Connection.Webhook == nil {
				state.Connection.Webhook = &WebhookState{}
			}
			w := state.Connection.Webhook
			if w.Token == "" {
				w.Token = randomToken(16)
			}
			now := s.now()
			w.Mode, w.PublicURL = "webhook", tunnel
			w.URL = s.webhookBase(w.PublicURL) + "/hooks/" + st.Adapter + "/" + w.Token
			w.SelfTest, w.Registration = Preflight(ctx, w.URL, now, s.AllowPrivateURL), nil
			if w.SelfTest != nil {
				w.Registration = &Registration{Status: "skipped", At: now, Detail: "after the endpoint is reachable"}
			}
			return nil
		})
		if err != nil {
			return ConnectSetup{}, err
		}
		return s.View(ctx, ws, out)
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectSetup{}, err
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	state := decodeState(st.State)
	if state.Connection.Webhook == nil {
		state.Connection.Webhook = &WebhookState{Mode: "webhook"}
	}
	tunnel, err := tunnelURL(in.PublicURL)
	if err != nil {
		return ConnectSetup{}, err
	}
	if tunnel != "" {
		cfg := currentSettings(&p)
		cfg["webhook_base_url"] = tunnel
		b, _ := json.Marshal(cfg)
		if err := s.Q.ConnectSetProviderConfig(ctx, store.ConnectSetProviderConfigParams{ID: p.ID, Config: b}); err != nil {
			return ConnectSetup{}, err
		}
		p.Config = b
	}
	state.Connection.Webhook.PublicURL = tunnel
	state.Connection.Webhook.Mode = "webhook"
	s.setupEvents(ctx, d, copyFor(d), p, &state, s.now())
	out, err := s.saveSetup(ctx, ws, st, state, "provider.webhook_test", map[string]any{"reachable": state.Connection.Webhook.SelfTest != nil && state.Connection.Webhook.SelfTest.OK,
		"tunnel": tunnel != ""})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, out)
}

// UsePolling switches the provider to polling its events API instead of
// webhooks.
func (s *Service) UsePolling(ctx context.Context, ws *store.Workspace, rawID string) (ConnectSetup, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	if st.ProviderID == nil {
		// Not connected yet: the connection uses polling from the start.
		if !copyFor(d).EventsAPI {
			return ConnectSetup{}, api.Conflict(d.Name + " has no events API to poll. Make the webhook endpoint reachable instead.")
		}
		out, err := s.update(ctx, ws, st.ID, "provider.events_polling", map[string]any{"adapter": d.Adapter, "events": "polling"}, func(_ *store.ProviderSetup, state *State) error {
			if state.Connection.Webhook == nil {
				state.Connection.Webhook = &WebhookState{}
			}
			w := state.Connection.Webhook
			w.Mode, w.SelfTest, w.Registration = "polling", nil, nil
			return nil
		})
		if err != nil {
			return ConnectSetup{}, err
		}
		return s.View(ctx, ws, out)
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectSetup{}, err
	}
	if a := s.probeAdapter(ctx, d, &p); a != nil {
		if _, ok := a.(providers.EventBackfiller); !ok && !copyFor(d).EventsAPI {
			return ConnectSetup{}, api.Conflict(d.Name + " has no events API to poll. Make the webhook endpoint reachable instead.")
		}
	}
	cfg := currentSettings(&p)
	cfg["events"] = "polling"
	b, _ := json.Marshal(cfg)
	state := decodeState(st.State)
	if state.Connection.Webhook == nil {
		state.Connection.Webhook = &WebhookState{}
	}
	state.Connection.Webhook.Mode = "polling"
	var out store.ProviderSetup
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ConnectSetProviderConfig(ctx, store.ConnectSetProviderConfigParams{ID: p.ID, Config: b}); err != nil {
			return err
		}
		var err error
		if out, err = s.save(ctx, q, st, state); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.events_polling", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
			After: map[string]any{"events": "polling"}})
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, out)
}

// ConnectMailbox sets up and tests the SMTP bounce mailbox.
func (s *Service) ConnectMailbox(ctx context.Context, ws *store.Workspace, rawID string, in ConnectMailboxInput) (ConnectSetup, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectSetup{}, api.Conflict("Test the outgoing server first: the bounce mailbox belongs to it.")
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	cfg := currentSettings(&p)
	creds := s.currentCreds(&p)
	folder := strings.TrimSpace(in.Folder)
	if folder == "" {
		folder = "INBOX"
	}
	address := strings.ToLower(strings.TrimSpace(in.Address))
	mb := map[string]any{"protocol": in.Protocol, "host": strings.ToLower(strings.TrimSpace(in.Host)), "port": in.Port, "tls": in.TLS,
		"username": strings.TrimSpace(in.Username), "folder": folder, "address": address, "poll_every": in.PollEvery, "after_processing": in.AfterProcessing}
	cfg["bounce_mailbox"] = mb
	if address != "" {
		cfg["return_path"] = address
	}
	if in.Password != "" {
		creds["bounce_password"] = in.Password
	}
	if _, ok := creds["bounce_password"]; !ok && in.Username != "" {
		return ConnectSetup{}, api.Invalid("body.password", "Enter the mailbox password.")
	}
	b := built{settings: cfg, creds: creds}
	adapter, err := newAdapter(d, p.ID, ws.ID, b, s.Factory.WebhookSecret(p), s.httpClient())
	if err != nil {
		return ConnectSetup{}, api.Invalid("body", "These settings can't be used: "+err.Error())
	}
	now := s.now()
	state := decodeState(st.State)
	probe := &ProbeState{At: now}
	if err := s.checkHost(ctx, stringOf(mb["host"])); err != nil {
		probe.Error = hostBlocked("host", err).Message
	} else if mt, ok := adapter.(providers.BounceMailboxTester); ok {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		rep, err := mt.TestBounceMailbox(cctx)
		cancel()
		if err != nil {
			probe.Error = callMessage(err)
		} else {
			probe.OK = true
			probe.Summary = fmt.Sprintf("%d %s in %s, %d %s like DSNs", rep.Messages, plural(rep.Messages, "message", "messages"), rep.Folder, rep.Reports,
				plural(rep.Reports, "looks", "look"))
		}
	} else {
		probe.Error = d.Name + " can't test a bounce mailbox."
	}
	state.Connection.Mailbox = probe
	state.Connection.MailboxTried = &MailboxTried{Address: address, Protocol: in.Protocol, Host: stringOf(mb["host"]), Port: in.Port, TLS: in.TLS,
		Username: strings.TrimSpace(in.Username), Folder: folder, PollEvery: in.PollEvery, AfterProcessing: in.AfterProcessing}
	cb, _ := json.Marshal(cfg)
	credJSON, _ := json.Marshal(creds)
	sealed, err := s.Box.Seal(credJSON, manage.CredentialsAD(p))
	if err != nil {
		return ConnectSetup{}, err
	}
	var out store.ProviderSetup
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if probe.OK {
			if err := q.ConnectSetProviderConfig(ctx, store.ConnectSetProviderConfigParams{ID: p.ID, Config: cb}); err != nil {
				return err
			}
			if err := q.ProvUpdateCredentials(ctx, store.ProvUpdateCredentialsParams{ID: p.ID, CredentialsEnc: sealed}); err != nil {
				return err
			}
		}
		var err error
		if out, err = s.save(ctx, q, st, state); err != nil {
			return err
		}
		res := audit.Allowed
		if !probe.OK {
			res = audit.Failed
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.bounce_mailbox", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
			Result: res, After: map[string]any{"protocol": in.Protocol, "host": in.Host, "folder": folder}})
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, out)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
