// Package postmark is the Postmark adapter. Postmark's terms allow bulk mail
// only on a Broadcast message stream, which the adapter can create.
package postmark

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "postmark"

// DefaultBaseURL is Postmark's API.
const DefaultBaseURL = "https://api.postmarkapp.com"

// WebhookUser is the basic-auth user name ReputeCast registers on webhooks;
// the password is the provider's webhook secret.
const WebhookUser = "reputecast"

// MetaMessage is the metadata key carrying the ReputeCast message id.
const MetaMessage = "rc_message"

// BatchSize is Postmark's batch limit.
const BatchSize = 500

// Settings are the non-secret connection settings.
type Settings struct {
	Server          string `json:"server,omitempty"`
	ServerID        int64  `json:"server_id,omitempty"`
	MessageStream   string `json:"message_stream,omitempty"`
	BroadcastStream string `json:"broadcast_stream,omitempty"`
	// MonthlyLimit is the plan's monthly allowance, entered at connect time.
	MonthlyLimit int64 `json:"monthly_limit,omitempty"`
	// BaseURL overrides the API (tests, regional proxies).
	BaseURL string `json:"base_url,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	ServerToken  string `json:"server_token"`
	AccountToken string `json:"account_token,omitempty"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Postmark",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkBroadcastStreams,
			Webhooks:        "Registered",
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       BatchSize,
			DedicatedIPs:    "from 300k per month",
			Sandbox:         "test token",
			QuotaDiscovery:  true,
			Pricing:         "plan, from $15 per month",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{
			{Variant: "Transactional stream", Bulk: providers.BulkForbidden, Note: "Transactional streams are for one-to-one mail triggered by a user action. Marketing on them breaks Postmark's terms.",
				Streams: []providers.Stream{providers.StreamAutomation, providers.StreamSystem}},
			{Variant: "Broadcast stream", Bulk: providers.BulkAllowed, Note: "Requires an unsubscribe link in every message; Postmark adds one if missing.",
				Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation}},
		},
		Events:          "Webhooks",
		DomainSetup:     "API",
		SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "server_token", Label: "Server API token", Secret: true, Required: true, Help: "Postmark › Servers › your server › API Tokens."},
			{Key: "account_token", Label: "Account API token", Secret: true, Help: "Optional. Lets ReputeCast add your domain and fetch its DNS records."},
			{Key: "message_stream", Label: "Message stream", Required: true, Options: []string{"outbound"}, Help: "The transactional stream for System and Automations mail."},
			{Key: "broadcast_stream", Label: "Broadcast stream", Help: "Needed before Broadcasts can use Postmark. ReputeCast can create it."},
			{Key: "monthly_limit", Label: "Plan · emails per month", Help: "Used for the quota bar and cost per 1,000."},
		},
		New: New,
	}
}

// Provider talks to one Postmark server.
type Provider struct {
	cfg     providers.Config
	s       Settings
	c       Credentials
	api     *apiclient.Client
	account *apiclient.Client
}

var (
	_ providers.Provider           = (*Provider)(nil)
	_ providers.ReputationReporter = (*Provider)(nil)
)

// New builds the adapter from a connected provider's configuration.
func New(cfg providers.Config) (providers.Provider, error) {
	p := &Provider{cfg: cfg}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &p.s); err != nil {
			return nil, fmt.Errorf("postmark settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("postmark credentials: %w", err)
		}
	}
	if p.s.MessageStream == "" {
		p.s.MessageStream = "outbound"
	}
	base := p.s.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	p.api.Header.Set("X-Postmark-Server-Token", p.c.ServerToken)
	p.api.Decode = decodeError
	if p.c.AccountToken != "" {
		p.account = apiclient.New(base, cfg.HTTPClient)
		p.account.Header.Set("X-Postmark-Account-Token", p.c.AccountToken)
		p.account.Decode = decodeError
	}
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities {
	c := Descriptor().Capabilities
	if p.s.MonthlyLimit > 0 {
		c.Pricing = "plan, " + strconv.FormatInt(p.s.MonthlyLimit, 10) + " per month"
	}
	return c
}

func decodeError(_ int, body []byte) (string, string) {
	var e struct {
		ErrorCode int
		Message   string
	}
	if json.Unmarshal(body, &e) == nil && (e.ErrorCode != 0 || e.Message != "") {
		return "ErrorCode " + strconv.Itoa(e.ErrorCode), e.Message
	}
	return "", strings.TrimSpace(string(body))
}

type header struct {
	Name  string
	Value string
}

type outMessage struct {
	From          string
	To            string
	ReplyTo       string `json:",omitempty"`
	Subject       string
	HtmlBody      string            `json:",omitempty"` //nolint:revive // Postmark field name
	TextBody      string            `json:",omitempty"`
	Headers       []header          `json:",omitempty"`
	Metadata      map[string]string `json:",omitempty"`
	Tag           string            `json:",omitempty"`
	MessageStream string
	TrackOpens    bool
	TrackLinks    string
}

type sendResult struct {
	ErrorCode   int
	Message     string
	MessageID   string
	To          string
	SubmittedAt string
}

// StreamFor returns the Postmark message stream used for a ReputeCast stream.
func (p *Provider) StreamFor(s providers.Stream) string {
	if s == providers.StreamBroadcast && p.s.BroadcastStream != "" {
		return p.s.BroadcastStream
	}
	return p.s.MessageStream
}

// Send implements providers.Provider with POST /email/batch.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	if len(msgs) == 0 {
		return []providers.SendResult{}, nil
	}
	if len(msgs) > BatchSize {
		return nil, fmt.Errorf("postmark: batch of %d exceeds %d", len(msgs), BatchSize)
	}
	out := make([]outMessage, len(msgs))
	for i, m := range msgs {
		meta := map[string]string{MetaMessage: m.ID.String()}
		for k, v := range m.Tags {
			if len(meta) < 10 {
				meta[k] = v
			}
		}
		om := outMessage{
			From: mimemsg.Address(m.From), To: mimemsg.Address(m.To), Subject: m.Subject, HtmlBody: m.HTML, TextBody: m.Text,
			Metadata: meta, MessageStream: p.StreamFor(m.Stream), TrackOpens: false, TrackLinks: "None",
		}
		if m.ReplyTo != nil {
			om.ReplyTo = mimemsg.Address(*m.ReplyTo)
		}
		om.Headers = append(om.Headers, header{Name: "X-ReputeCast-Message", Value: m.ID.String()})
		for k, v := range m.Headers {
			om.Headers = append(om.Headers, header{Name: k, Value: v})
		}
		out[i] = om
	}
	var res []sendResult
	if err := p.api.Do(ctx, "POST /email/batch", http.MethodPost, "/email/batch", out, &res); err != nil {
		return nil, err
	}
	if len(res) != len(msgs) {
		return nil, &providers.CallError{Op: "POST /email/batch", Message: fmt.Sprintf("Postmark answered %d results for %d messages", len(res), len(msgs)), Temporary: true}
	}
	results := make([]providers.SendResult, len(msgs))
	for i, r := range res {
		results[i] = providers.SendResult{MessageID: msgs[i].ID, ProviderMessageID: r.MessageID}
		if r.ErrorCode != 0 {
			results[i].ProviderMessageID = ""
			results[i].Err = &providers.SendError{
				Temporary: !permanentCode(r.ErrorCode),
				Throttled: r.ErrorCode == 429,
				Code:      "ErrorCode " + strconv.Itoa(r.ErrorCode),
				Message:   r.Message,
			}
		}
	}
	return results, nil
}

// permanentCode lists per-message errors that retrying can't fix.
func permanentCode(code int) bool {
	switch code {
	case 300, 406: // invalid email request, inactive (suppressed) recipient
		return true
	}
	return false
}

// VerifyWebhook checks the basic-auth credentials ReputeCast registered.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, _ []byte) error {
	user, pass, ok := r.BasicAuth()
	if !ok || p.cfg.WebhookSecret == "" || user != WebhookUser ||
		subtle.ConstantTimeCompare([]byte(pass), []byte(p.cfg.WebhookSecret)) != 1 {
		return providers.ErrBadSignature
	}
	return nil
}

type record struct {
	RecordType        string
	ID                json.Number
	Type              string
	TypeCode          int
	Name              string
	MessageID         string
	MessageStream     string
	Recipient         string
	Email             string
	Description       string
	Details           string
	DeliveredAt       string
	BouncedAt         string
	ReceivedAt        string
	ChangedAt         string
	OriginalLink      string
	UserAgent         string
	SuppressSending   bool
	SuppressionReason string
	Metadata          map[string]string
}

// ParseWebhook implements providers.Provider. Postmark posts one record per request.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	var rec record
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&rec); err != nil {
		return nil, fmt.Errorf("postmark webhook: %w", err)
	}
	ev, ok := normalize(rec)
	if !ok {
		return []providers.Event{}, nil
	}
	ev.Raw = append(json.RawMessage(nil), body...)
	return []providers.Event{ev}, nil
}

func normalize(rec record) (providers.Event, bool) {
	ev := providers.Event{ProviderMessageID: rec.MessageID, Recipient: strings.ToLower(firstNonEmpty(rec.Recipient, rec.Email))}
	if id, err := uuid.Parse(rec.Metadata[MetaMessage]); err == nil {
		ev.MessageID = &id
	}
	switch rec.RecordType {
	case "Delivery":
		ev.Type = providers.EventDelivered
		ev.OccurredAt = parseTime(rec.DeliveredAt)
		ev.OriginalType = "Delivery"
		ev.OriginalCode, ev.Diagnostic = bounce.Split(rec.Details)
	case "Bounce":
		ev.OccurredAt = parseTime(rec.BouncedAt)
		ev.OriginalType = "Bounce · " + rec.Type
		code, rest := bounce.Split(strings.TrimPrefix(rec.Details, "smtp;"))
		ev.OriginalCode = code
		if ev.OriginalCode == "" {
			ev.OriginalCode = "Type " + strconv.Itoa(rec.TypeCode)
		}
		ev.Diagnostic = firstNonEmpty(rest, rec.Description)
		switch rec.Type {
		case "HardBounce", "BadEmailAddress", "ManuallyDeactivated":
			ev.Type = providers.EventHardBounce
		case "SpamNotification", "SpamComplaint":
			ev.Type = providers.EventComplaint
		case "AutoResponder", "Subscribe", "Unsubscribe", "AddressChange", "OpenRelayTest", "ChallengeVerification":
			return ev, false
		default: // Transient, SoftBounce, DnsError, InboxFull, Blocked, DMARCPolicy, SMTPApiError, Unknown…
			ev.Type = providers.EventSoftBounce
		}
		if ev.Type != providers.EventComplaint {
			ev.BounceClass = bounce.Classify(code, firstNonEmpty(rec.Details, rec.Description))
			if rec.Type == "InboxFull" {
				ev.BounceClass = bounce.MailboxFull
			}
			if rec.Type == "DnsError" {
				ev.BounceClass = bounce.DomainInvalid
			}
		}
	case "SpamComplaint":
		ev.Type = providers.EventComplaint
		ev.OccurredAt = parseTime(rec.BouncedAt)
		ev.OriginalType = "SpamComplaint"
		ev.OriginalCode = "Type " + strconv.Itoa(rec.TypeCode)
		ev.Diagnostic = firstNonEmpty(rec.Details, rec.Description)
	case "Open":
		ev.Type = providers.EventOpen
		ev.OccurredAt = parseTime(rec.ReceivedAt)
		ev.OriginalType = "Open"
		ev.UserAgent = rec.UserAgent
	case "Click":
		ev.Type = providers.EventClick
		ev.OccurredAt = parseTime(rec.ReceivedAt)
		ev.OriginalType = "Click"
		ev.URL = rec.OriginalLink
		ev.UserAgent = rec.UserAgent
	case "SubscriptionChange":
		if !rec.SuppressSending || rec.SuppressionReason != "ManualSuppression" {
			return ev, false
		}
		ev.Type = providers.EventUnsubscribe
		ev.OccurredAt = parseTime(rec.ChangedAt)
		ev.OriginalType = "SubscriptionChange · ManualSuppression"
	default:
		return ev, false
	}
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	return ev, true
}

func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.9999999Z07:00", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type serverInfo struct {
	ID         int64
	Name       string
	TrackOpens bool
	TrackLinks string
}

func (p *Provider) server(ctx context.Context) (serverInfo, error) {
	var s serverInfo
	err := p.api.Do(ctx, "GET /server", http.MethodGet, "/server", nil, &s)
	return s, err
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) {
	if _, err := p.server(ctx); err != nil {
		return providers.Quota{}, err
	}
	return p.Quota(ctx)
}

// Quota implements providers.QuotaReporter: emails sent this month on the
// server against the plan's monthly allowance.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	q := url.Values{"fromdate": {from.Format("2006-01-02")}, "todate": {now.Format("2006-01-02")}}
	var st struct{ Sent int64 }
	if err := p.api.Do(ctx, "GET /stats/outbound", http.MethodGet, "/stats/outbound?"+q.Encode(), nil, &st); err != nil {
		return providers.Quota{}, err
	}
	acct := "Approved"
	if p.s.MonthlyLimit > 0 {
		acct += " · " + humanCount(p.s.MonthlyLimit) + " per month plan"
	}
	return providers.Quota{Kind: "monthly_plan", Max: p.s.MonthlyLimit, Used: st.Sent, Account: acct}, nil
}

// Reputation implements providers.ReputationReporter: Postmark's own bounce
// and spam complaint rates over the last seven days, from its outbound stats.
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	now := time.Now().UTC()
	q := url.Values{"fromdate": {now.AddDate(0, 0, -6).Format("2006-01-02")}, "todate": {now.Format("2006-01-02")}}
	var st struct {
		BounceRate         float64
		SpamComplaintsRate float64
	}
	if err := p.api.Do(ctx, "GET /stats/outbound", http.MethodGet, "/stats/outbound?"+q.Encode(), nil, &st); err != nil {
		return providers.Reputation{}, err
	}
	return providers.Reputation{Bounce: st.BounceRate, Complaint: st.SpamComplaintsRate, Window: "7 d"}, nil
}

func humanCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ListSuppressions implements providers.SuppressionSyncer for every stream
// ReputeCast uses on this server.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	for _, stream := range p.streams() {
		q := url.Values{}
		if !since.IsZero() {
			q.Set("fromdate", since.UTC().Format("2006-01-02"))
		}
		var dump struct {
			Suppressions []struct {
				EmailAddress      string
				SuppressionReason string
				Origin            string
				CreatedAt         string
			}
		}
		path := apiclient.Path("/message-streams/%s/suppressions/dump", stream)
		if len(q) > 0 {
			path += "?" + q.Encode()
		}
		if err := p.api.Do(ctx, "GET /message-streams/suppressions/dump", http.MethodGet, path, nil, &dump); err != nil {
			return nil, err
		}
		for _, s := range dump.Suppressions {
			reason := "manual"
			switch s.SuppressionReason {
			case "HardBounce":
				reason = "hard_bounce"
			case "SpamComplaint":
				reason = "complaint"
			case "ManualSuppression":
				if s.Origin == "Recipient" {
					reason = "unsubscribe"
				}
			}
			out = append(out, providers.Suppression{Email: strings.ToLower(s.EmailAddress), Reason: reason, Code: s.SuppressionReason, CreatedAt: parseTime(s.CreatedAt)})
		}
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer (50 per request).
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	for _, stream := range p.streams() {
		for i := 0; i < len(emails); i += 50 {
			j := min(i+50, len(emails))
			type entry struct{ EmailAddress string }
			body := struct{ Suppressions []entry }{}
			for _, e := range emails[i:j] {
				body.Suppressions = append(body.Suppressions, entry{EmailAddress: e})
			}
			path := apiclient.Path("/message-streams/%s/suppressions", stream)
			if err := p.api.Do(ctx, "POST /message-streams/suppressions", http.MethodPost, path, body, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Provider) streams() []string {
	out := []string{p.s.MessageStream}
	if p.s.BroadcastStream != "" && p.s.BroadcastStream != p.s.MessageStream {
		out = append(out, p.s.BroadcastStream)
	}
	return out
}

type httpAuth struct {
	Username string
	Password string
}

type webhook struct {
	ID            int64  `json:"ID,omitempty"`
	URL           string `json:"Url"`
	MessageStream string
	HTTPAuth      *httpAuth `json:"HttpAuth,omitempty"`
	Triggers      map[string]map[string]bool
}

// RegisterWebhook implements providers.WebhookRegistrar: one webhook per
// stream, with basic auth. An existing webhook for the same URL is updated.
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	u, err := url.Parse(hookURL)
	if err != nil {
		return err
	}
	clean := *u
	clean.User = nil
	for _, stream := range p.streams() {
		existing, err := p.listWebhooks(ctx, stream)
		if err != nil {
			return err
		}
		w := webhook{URL: clean.String(), MessageStream: stream, Triggers: map[string]map[string]bool{
			"Open": {"Enabled": false}, "Click": {"Enabled": false}, "Delivery": {"Enabled": true},
			"Bounce": {"Enabled": true, "IncludeContent": false}, "SpamComplaint": {"Enabled": true, "IncludeContent": false},
			"SubscriptionChange": {"Enabled": true},
		}}
		w.HTTPAuth = &httpAuth{Username: WebhookUser, Password: p.cfg.WebhookSecret}
		var id int64
		for _, e := range existing {
			if sameURL(e.URL, clean.String()) {
				id = e.ID
			}
		}
		if id != 0 {
			err = p.api.Do(ctx, "PUT /webhooks", http.MethodPut, "/webhooks/"+strconv.FormatInt(id, 10), w, nil)
		} else {
			err = p.api.Do(ctx, "POST /webhooks", http.MethodPost, "/webhooks", w, nil)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) listWebhooks(ctx context.Context, stream string) ([]webhook, error) {
	var res struct{ Webhooks []webhook }
	err := p.api.Do(ctx, "GET /webhooks", http.MethodGet, "/webhooks?MessageStream="+url.QueryEscape(stream), nil, &res)
	return res.Webhooks, err
}

func sameURL(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

// InspectWebhook implements providers.WebhookInspector.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1}
	hooks, err := p.listWebhooks(ctx, p.s.MessageStream)
	if err != nil {
		return st, err
	}
	for _, h := range hooks {
		if sameURL(h.URL, hookURL) {
			st.Registered, st.ID, st.URL = true, strconv.FormatInt(h.ID, 10), h.URL
		}
	}
	return st, nil
}

// Tracking implements providers.TrackingManager from the server settings.
func (p *Provider) Tracking(ctx context.Context) (providers.Tracking, error) {
	s, err := p.server(ctx)
	if err != nil {
		return providers.Tracking{}, err
	}
	links := s.TrackLinks
	if links == "" {
		links = "None"
	}
	t := providers.Tracking{OpenPixel: s.TrackOpens, LinkRewrite: links != "None", Scope: "Server " + s.Name}
	onOff := "off"
	if s.TrackOpens {
		onOff = "on"
	}
	t.Detail = "Server setting TrackOpens is " + onOff + ", and TrackLinks is " + links + "."
	return t, nil
}

// DisableTracking implements providers.TrackingManager.
func (p *Provider) DisableTracking(ctx context.Context) error {
	body := map[string]any{"TrackOpens": false, "TrackLinks": "None"}
	return p.api.Do(ctx, "PUT /server", http.MethodPut, "/server", body, nil)
}

// CreateBroadcastStream implements providers.BroadcastStreamCreator.
func (p *Provider) CreateBroadcastStream(ctx context.Context, name string) (string, error) {
	if name == "" {
		name = "ReputeCast Broadcasts"
	}
	for _, id := range []string{"broadcast", "reputecast-broadcasts"} {
		body := map[string]any{"ID": id, "Name": name, "MessageStreamType": "Broadcasts", "Description": "Bulk mail sent by ReputeCast"}
		var res struct{ ID string }
		err := p.api.Do(ctx, "POST /message-streams", http.MethodPost, "/message-streams", body, &res)
		if err == nil {
			p.s.BroadcastStream = res.ID
			return res.ID, nil
		}
		var ce *providers.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusUnprocessableEntity && strings.Contains(strings.ToLower(ce.Message), "already") {
			continue
		}
		return "", err
	}
	return "", &providers.CallError{Op: "POST /message-streams", Message: "both stream ids are taken; create a Broadcast stream in Postmark and enter its id"}
}

// AddDomain implements providers.DomainManager (needs the account token).
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	if p.account == nil {
		return nil, providers.ErrNotSupported
	}
	body := map[string]any{"Name": domain, "ReturnPathDomain": "pm-bounces." + domain}
	var d domainInfo
	if err := p.account.Do(ctx, "POST /domains", http.MethodPost, "/domains", body, &d); err != nil {
		return nil, err
	}
	return d.records(domain), nil
}

// DomainStatus implements providers.DomainManager.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	if p.account == nil {
		return nil, providers.ErrNotSupported
	}
	var list struct {
		Domains []struct {
			ID   int64
			Name string
		}
	}
	if err := p.account.Do(ctx, "GET /domains", http.MethodGet, "/domains?count=500&offset=0", nil, &list); err != nil {
		return nil, err
	}
	for _, d := range list.Domains {
		if strings.EqualFold(d.Name, domain) {
			var info domainInfo
			if err := p.account.Do(ctx, "GET /domains/{id}", http.MethodGet, "/domains/"+strconv.FormatInt(d.ID, 10), nil, &info); err != nil {
				return nil, err
			}
			return info.records(domain), nil
		}
	}
	return nil, &providers.CallError{Op: "GET /domains", Status: http.StatusNotFound, Message: domain + " is not set up in Postmark"}
}

type domainInfo struct {
	DKIMVerified               bool
	DKIMHost                   string
	DKIMTextValue              string
	DKIMPendingHost            string
	DKIMPendingTextValue       string
	ReturnPathDomain           string
	ReturnPathDomainVerified   bool
	ReturnPathDomainCNAMEValue string
}

func (d domainInfo) records(domain string) []providers.DNSRecord {
	host, value := d.DKIMHost, d.DKIMTextValue
	if host == "" {
		host, value = d.DKIMPendingHost, d.DKIMPendingTextValue
	}
	status := func(ok bool) string {
		if ok {
			return "verified"
		}
		return "pending"
	}
	out := []providers.DNSRecord{{Purpose: "dkim", Type: "TXT", Name: host, Value: value, Status: status(d.DKIMVerified)}}
	if d.ReturnPathDomain != "" {
		cname := d.ReturnPathDomainCNAMEValue
		if cname == "" {
			cname = "pm.mtasv.net"
		}
		out = append(out, providers.DNSRecord{Purpose: "return_path", Type: "CNAME", Name: d.ReturnPathDomain, Value: cname, Status: status(d.ReturnPathDomainVerified)})
	}
	_ = domain
	return out
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "Postmark status", "https://status.postmarkapp.com"
}
