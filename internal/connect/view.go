package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// ConnectOption is one choice of a select.
type ConnectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ConnectField is one input of the generic connection form, from the
// adapter's Descriptor.
type ConnectField struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Secret   bool            `json:"secret"`
	Required bool            `json:"required"`
	Options  []ConnectOption `json:"options"`
	Help     string          `json:"help,omitempty"`
	// Type is "number" for a whole number (sent as text, stored as a JSON
	// number), else "text".
	Type string `json:"type" enum:"text,number"`
	// Value is the current value of a non-secret field.
	Value string `json:"value"`
	// Set is true when a secret is stored (shown masked with Replace).
	Set bool `json:"set"`
}

// ConnectAdapterInfo is how the setup talks about its adapter.
type ConnectAdapterInfo struct {
	Adapter        string `json:"adapter"`
	Name           string `json:"name"`
	Short          string `json:"short"`
	Step1          string `json:"step1"`
	Title          string `json:"title"`
	Heading        string `json:"heading"`
	Lead           string `json:"lead"`
	Section        string `json:"section"`
	Flow           string `json:"flow" enum:"ses,smtp,api"`
	ConnectionHint string `json:"connectionHint"`
	DomainHint     string `json:"domainHint"`
	KeyHint        string `json:"keyHint"`
	KeysURL        string `json:"keysUrl"`
	VerifyCall     string `json:"verifyCall"`
	WebhookEvents  string `json:"webhookEvents"`
	// Webhooks is the adapter's event channel: SNS, Registered, Bounce mailbox or none.
	Webhooks string `json:"webhooks"`
	// CanRegister: ReputeCast registers the webhook itself.
	CanRegister bool `json:"canRegister"`
	// CanPoll: the provider has an events API ReputeCast can poll instead.
	CanPoll bool `json:"canPoll"`
	// BulkAllowed is false when the terms exclude bulk marketing.
	BulkAllowed bool           `json:"bulkAllowed"`
	Fields      []ConnectField `json:"fields"`
}

// ConnectSummaries are the stepper's lines.
type ConnectSummaries struct {
	Provider    string `json:"provider"`
	Connection  string `json:"connection"`
	Domain      string `json:"domain"`
	DomainShort string `json:"domainShort"`
	Test        string `json:"test"`
	Streams     string `json:"streams"`
}

// ConnectError is why the connection failed.
type ConnectError struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// ConnectRegion is a region of the region select.
type ConnectRegion struct {
	Region     string  `json:"region"`
	Name       string  `json:"name"`
	Production bool    `json:"production"`
	Max24h     int64   `json:"max24h"`
	MaxRate    float64 `json:"maxRate"`
	// Label is the option text ("us-east-1 · N. Virginia · production access, quota 50,000 per day").
	Label string `json:"label"`
	Err   string `json:"err,omitempty"`
}

// ConnectProvisionRow is one provisioning row.
type ConnectProvisionRow struct {
	Key string `json:"key"`
	// Text is the label or, once done, what was done; `backticks` mark code.
	Text string `json:"text"`
	// Status deferred: the event check waits for a verified sending domain.
	Status string     `json:"status" enum:"waiting,running,done,failed,deferred"`
	At     *time.Time `json:"at,omitempty"`
	Error  string     `json:"error,omitempty"`
	Hint   string     `json:"hint,omitempty"`
	// Note qualifies what was done ("the SES mailbox simulator; it doesn't
	// affect your bounce or complaint rates").
	Note string `json:"note,omitempty"`
}

// ConnectHost is where ReputeCast runs, which decides the SES access it
// recommends (PR-5).
type ConnectHost struct {
	OnAWS bool `json:"onAws"`
	// Kind is ec2, ecs, eks or lambda on AWS.
	Kind string `json:"kind"`
	// BaseCredentials: this server has AWS credentials of its own to assume
	// the stack's role with.
	BaseCredentials bool      `json:"baseCredentials"`
	CheckedAt       time.Time `json:"checkedAt"`
	Recommended     string    `json:"recommended" enum:"role,keys"`
}

// ConnectWebhookTest is the provider's test webhook (PR-9): ReputeCast asks
// the provider to call the endpoint and waits a minute for it.
type ConnectWebhookTest struct {
	// Status: waiting (asked, within the minute), arrived, missing (nothing
	// within the minute) or refused (the provider wouldn't send one).
	Status      string     `json:"status" enum:"waiting,arrived,missing,refused"`
	RequestedAt time.Time  `json:"requestedAt"`
	Until       time.Time  `json:"until"`
	ArrivedAt   *time.Time `json:"arrivedAt,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// ConnectSandbox is the sandbox card and its production-access request.
type ConnectSandbox struct {
	Region string `json:"region"`
	Name   string `json:"name"`
	// Selected is true when the selected region itself is in the sandbox.
	Selected    bool       `json:"selected"`
	Website     string     `json:"website"`
	MailType    string     `json:"mailType" enum:"marketing,transactional"`
	UseCase     string     `json:"useCase"`
	DailyVolume int64      `json:"dailyVolume"`
	Language    string     `json:"language" enum:"en,ja"`
	RequestedAt *time.Time `json:"requestedAt,omitempty"`
}

// ConnectAccess is the CloudFormation path.
type ConnectAccess struct {
	LaunchURL string `json:"launchUrl"`
	Policy    string `json:"policy"`
	Principal string `json:"principal"`
	RoleName  string `json:"roleName"`
}

// ConnectProbe is a connection or mailbox test result.
type ConnectProbe struct {
	OK      bool      `json:"ok"`
	At      time.Time `json:"at"`
	Summary string    `json:"summary"`
	// Transcript is the protocol exchange joined with " · ".
	Transcript string `json:"transcript"`
	Error      string `json:"error,omitempty"`
}

// ConnectSMTPForm is the SMTP form's current values (never the passwords).
type ConnectSMTPForm struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	TLS             string `json:"tls" enum:"starttls,tls,none"`
	Username        string `json:"username"`
	PasswordSet     bool   `json:"passwordSet"`
	MaxRate         string `json:"maxRate"`
	DailyCap        string `json:"dailyCap"`
	ReturnPath      string `json:"returnPath"`
	Protocol        string `json:"protocol" enum:"imap,pop3"`
	MailboxHost     string `json:"mailboxHost"`
	MailboxPort     int    `json:"mailboxPort"`
	MailboxTLS      string `json:"mailboxTls" enum:"tls,starttls,none"`
	MailboxUser     string `json:"mailboxUser"`
	MailboxPassSet  bool   `json:"mailboxPasswordSet"`
	Folder          string `json:"folder"`
	PollEvery       string `json:"pollEvery" enum:"1m,5m,15m"`
	AfterProcessing string `json:"afterProcessing" enum:"move,delete,leave"`
}

// ConnectSelfTest is the webhook address check: localhost, private
// addresses and names that don't resolve fail it. ReputeCast never calls
// its own URL; only the provider calling it proves it reachable (PR-9).
type ConnectSelfTest struct {
	OK      bool      `json:"ok"`
	At      time.Time `json:"at"`
	Code    string    `json:"code,omitempty"`
	Reason  string    `json:"reason,omitempty"`
	Private bool      `json:"private"`
}

// ConnectRegistration is the webhook registration at the provider.
type ConnectRegistration struct {
	Status string    `json:"status" enum:"done,skipped,failed,manual"`
	Detail string    `json:"detail,omitempty"`
	At     time.Time `json:"at"`
}

// ConnectWebhook is how events come back from the provider.
type ConnectWebhook struct {
	Mode      string `json:"mode" enum:"webhook,polling,mailbox,none"`
	URL       string `json:"url"`
	PublicURL string `json:"publicUrl"`
	// NeedsSecret: the webhook is set up by hand with the secret ReputeCast
	// generated (shown on request), sent in SecretHeader or used to sign.
	NeedsSecret  bool                 `json:"needsSecret"`
	SecretHeader string               `json:"secretHeader,omitempty"`
	SelfTest     *ConnectSelfTest     `json:"selfTest,omitempty"`
	Registration *ConnectRegistration `json:"registration,omitempty"`
	// CanTest: the provider sends a test webhook on request; without it the
	// test send's first event confirms the endpoint.
	CanTest bool                `json:"canTest"`
	Test    *ConnectWebhookTest `json:"test,omitempty"`
}

// ConnectConnection is step 2.
type ConnectConnection struct {
	Status    string        `json:"status" enum:"none,connected,failed"`
	At        *time.Time    `json:"at,omitempty"`
	Account   string        `json:"account"`
	AccountID string        `json:"accountId"`
	Summary   string        `json:"summary"`
	Error     *ConnectError `json:"error,omitempty"`
	// Fields are the generic form (api flow).
	Fields []ConnectField `json:"fields"`
	// SES
	Access string `json:"access" enum:"role,keys"`
	// AccessKey is the saved access key id, masked ("AKIA••••••••7QXP");
	// IAMUser the IAM user it belongs to.
	AccessKey string       `json:"accessKey"`
	IAMUser   string       `json:"iamUser"`
	Host      *ConnectHost `json:"host,omitempty"`
	// Events is how SES events come back (PR-4); EventsURL the webhook SNS
	// calls in the webhook mode.
	Events    string `json:"events" enum:"webhook,queue"`
	EventsURL string `json:"eventsUrl"`
	// EventsCheck is the setup's event check (PR-3): none (no check for
	// this setup), waiting, deferred (waits for a verified domain), verified
	// (EventsAt is when the event arrived) or failed.
	EventsCheck    string                `json:"eventsCheck" enum:"none,waiting,deferred,verified,failed"`
	EventsAt       *time.Time            `json:"eventsAt,omitempty"`
	ExternalID     string                `json:"externalId"`
	RoleARN        string                `json:"roleArn"`
	Region         string                `json:"region"`
	Regions        []ConnectRegion       `json:"regions"`
	Template       *ConnectAccess        `json:"template,omitempty"`
	Provisioning   []ConnectProvisionRow `json:"provisioning"`
	ProvisionDone  int                   `json:"provisionDone"`
	ProvisionTotal int                   `json:"provisionTotal"`
	// Working is true while provisioning runs (poll for progress).
	Working bool            `json:"working"`
	Sandbox *ConnectSandbox `json:"sandbox,omitempty"`
	// SMTP
	SMTPForm *ConnectSMTPForm `json:"smtpForm,omitempty"`
	SMTP     *ConnectProbe    `json:"smtp,omitempty"`
	Mailbox  *ConnectProbe    `json:"mailbox,omitempty"`
	// Events
	Webhook *ConnectWebhook `json:"webhook,omitempty"`
	// CanContinue: the provider is connected and its events can come back.
	CanContinue bool `json:"canContinue"`
}

// ConnectDomainChoice is the domain of step 3.
type ConnectDomainChoice struct {
	DomainID string `json:"domainId"`
	Name     string `json:"name"`
	Skipped  bool   `json:"skipped"`
	// Suggested is the domain to propose when none is chosen yet.
	Suggested string `json:"suggested"`
}

// ConnectSetup is a setup with everything its steps show.
type ConnectSetup struct {
	ID         string              `json:"id"`
	Adapter    string              `json:"adapter"`
	Info       ConnectAdapterInfo  `json:"info"`
	Step       string              `json:"step" enum:"connection,domain,test,streams,contacts,done"`
	Status     string              `json:"status" enum:"open,finished,abandoned"`
	ProviderID string              `json:"providerId"`
	Provider   string              `json:"provider"`
	Summaries  ConnectSummaries    `json:"summaries"`
	Connection ConnectConnection   `json:"connection"`
	Domain     ConnectDomainChoice `json:"domain"`
	Streams    ConnectStreams      `json:"streams"`
	Done       *ConnectDone        `json:"done,omitempty"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
	FinishedAt *time.Time          `json:"finishedAt,omitempty"`
	CanManage  bool                `json:"canManage"`
	// Helpers names who can manage providers when the caller can't.
	Helpers []string `json:"helpers"`
}

// View builds the setup for its screens.
func (s *Service) View(ctx context.Context, ws *store.Workspace, st store.ProviderSetup) (ConnectSetup, error) {
	d, err := descriptor(ctx, st.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	state := decodeState(st.State)
	l := i18n.LangFrom(ctx)
	c := copyFor(l, d)
	out := ConnectSetup{ID: st.ID.String(), Adapter: st.Adapter, Step: st.Step, Status: st.Status, CreatedAt: st.CreatedAt, UpdatedAt: st.UpdatedAt,
		FinishedAt: st.FinishedAt, CanManage: canManage(ctx), Helpers: s.helpers(ctx, ws.ID)}
	var prov *store.Provider
	if st.ProviderID != nil {
		out.ProviderID = st.ProviderID.String()
		if p, err := s.Q.ProvGetAny(ctx, *st.ProviderID); err == nil {
			prov = &p
			out.Provider = p.Name
		}
	}
	// The stepper's domain line counts what DNS says now.
	if state.Domain != nil && state.Domain.DomainID != "" && prov != nil {
		if id, err := uuid.Parse(state.Domain.DomainID); err == nil {
			state.Domain.Verified, state.Domain.Total = s.domainCounts(ctx, id, prov.ID)
		}
	}
	out.Info = s.adapterInfo(ctx, l, d, c, prov, state)
	out.Connection = s.connectionView(ctx, l, ws, d, state, prov)
	out.Summaries = summaries(l, d, c, state)
	out.Domain = ConnectDomainChoice{Suggested: s.suggestedDomain(ctx, ws)}
	if state.Domain != nil {
		out.Domain.DomainID, out.Domain.Name, out.Domain.Skipped = state.Domain.DomainID, state.Domain.Name, state.Domain.Skipped
	}
	if prov != nil {
		if out.Streams, err = s.streamsView(ctx, l, ws, d, c, *prov, state); err != nil {
			return ConnectSetup{}, err
		}
	} else {
		out.Streams = ConnectStreams{Streams: []ConnectStreamView{}}
	}
	if st.Status == "finished" {
		done := s.doneView(ctx, l, ws, st, d, c, state, prov)
		out.Done = &done
	}
	return out, nil
}

// adapterInfo describes the adapter and its form.
func (s *Service) adapterInfo(ctx context.Context, l i18n.Lang, d providers.Descriptor, c adapterCopy, prov *store.Provider, state State) ConnectAdapterInfo {
	info := ConnectAdapterInfo{Adapter: d.Adapter, Name: c.Display, Short: c.Short, Step1: c.Step1, Title: c.Title, Heading: c.Heading, Lead: c.Lead,
		Section: c.Section, Flow: flowOf(d), ConnectionHint: c.ConnectionHint, DomainHint: c.DomainHint, KeyHint: c.KeyHint, KeysURL: c.KeysURL,
		VerifyCall: c.VerifyCall, WebhookEvents: c.WebhookEvents, Webhooks: d.Capabilities.Webhooks, BulkAllowed: d.Capabilities.Bulk != providers.BulkForbidden,
		Fields: []ConnectField{}}
	// What the adapter can do with events, from an instance of it.
	if a := s.probeAdapter(ctx, d, prov); a != nil {
		_, info.CanRegister = a.(providers.WebhookRegistrar)
		_, info.CanPoll = a.(providers.EventBackfiller)
	}
	if c.EventsAPI {
		info.CanPoll = true
	}
	settings := map[string]any{}
	if prov != nil {
		_ = json.Unmarshal(prov.Config, &settings)
	}
	creds := map[string]bool{}
	if prov != nil && s.Factory != nil && !manage.IsDemo(*prov) {
		if raw, err := s.Factory.Credentials(*prov); err == nil {
			m := map[string]any{}
			_ = json.Unmarshal(raw, &m)
			for k, v := range m {
				creds[k] = fmt.Sprint(v) != ""
			}
		}
	} else if prov != nil {
		for _, f := range d.Fields {
			creds[f.Key] = f.Secret
		}
	}
	for _, f := range d.Fields {
		words := f.In(l)
		cf := ConnectField{Key: f.Key, Label: words.Label, Secret: f.Secret, Required: f.Required, Help: words.Help, Options: []ConnectOption{}, Type: fieldType(f)}
		for _, o := range f.Options {
			cf.Options = append(cf.Options, ConnectOption{Value: o, Label: optionLabel(d.Adapter, f.Key, o)})
		}
		if f.Secret {
			cf.Set = creds[f.Key]
		} else {
			if v, ok := settings[f.Key]; ok && v != nil {
				cf.Value = fmt.Sprint(v)
			} else if v, ok := state.Connection.Settings[f.Key]; ok {
				cf.Value = v
			} else if len(cf.Options) > 0 {
				cf.Value = cf.Options[0].Value
			}
		}
		info.Fields = append(info.Fields, cf)
	}
	return info
}

// optionLabel names a select option the way the design does
// ("US · api.mailgun.net").
func optionLabel(adapter, key, v string) string {
	if adapter == "mailgun" && key == "region" {
		switch strings.ToLower(v) {
		case "us":
			return "US · api.mailgun.net"
		case "eu":
			return "EU · api.eu.mailgun.net"
		}
	}
	return v
}

// probeAdapter builds an instance of the adapter (the connected provider's
// or one with placeholder settings) to see which optional interfaces it has.
func (s *Service) probeAdapter(ctx context.Context, d providers.Descriptor, prov *store.Provider) providers.Provider {
	if prov != nil && s.Factory != nil && !manage.IsDemo(*prov) {
		if a, err := s.Factory.Open(ctx, *prov); err == nil {
			return a
		}
	}
	settings := map[string]any{}
	creds := map[string]any{}
	for _, f := range d.Fields {
		switch {
		case f.Secret:
			creds[f.Key] = "placeholder"
		case len(f.Options) > 0:
			settings[f.Key] = f.Options[0]
		case f.Required:
			settings[f.Key] = "https://example.com"
		}
	}
	if d.Adapter == "ses" {
		settings["access"], creds["access_key_id"], creds["secret_access_key"] = "keys", "AKIAPLACEHOLDER", "placeholder"
	}
	sb, _ := json.Marshal(settings)
	cb, _ := json.Marshal(creds)
	a, err := d.New(providers.Config{Settings: sb, Credentials: cb})
	if err != nil {
		return nil
	}
	return a
}

// summaries are the stepper's lines.
func summaries(l i18n.Lang, d providers.Descriptor, c adapterCopy, state State) ConnectSummaries {
	out := ConnectSummaries{Provider: c.Step1, Connection: connectionSummaryIn(l, flowOf(d), state.Connection)}
	if state.Domain != nil {
		out.DomainShort = state.Domain.Name
		out.Domain = state.Domain.Name
		if state.Domain.Total > 0 {
			out.Domain = mSummaryDomain.Text(l, i18n.Args{"name": state.Domain.Name, "verified": state.Domain.Verified, "total": state.Domain.Total})
		}
		if state.Domain.Skipped {
			skipped := mSummarySkipped.Text(l, nil)
			out.Domain, out.DomainShort = skipped, skipped
		}
	}
	if state.Test != nil {
		switch {
		case state.Test.RoundTripMs > 0:
			out.Test = mSummaryDelivered.Text(l, i18n.Args{"seconds": seconds(l, float64(state.Test.RoundTripMs)/1000)})
		default:
			out.Test = mSummarySentTo.Text(l, i18n.Args{"to": state.Test.To})
		}
	}
	if state.Streams != nil && state.Streams.Saved {
		names := []string{}
		for _, st := range []string{"broadcast", "automation", "system"} {
			if state.Streams.Streams[st].Enabled {
				names = append(names, streamName(l, st))
			}
		}
		out.Streams = strings.Join(names, ", ")
	}
	return out
}

// seconds writes a duration in seconds with one decimal ("2.4").
func seconds(l i18n.Lang, s float64) string {
	return l.Region.Decimal(s, 1)
}

// connectionSummaryIn is the stepper's line under Connection for a reader:
// the line written at connect time, written again in the reader's language
// from what the setup kept. A line that can't be (a row written before the
// account's message was kept) reads as it was written.
func connectionSummaryIn(l i18n.Lang, flow string, cn Connection) string {
	if english(l) || cn.Summary == "" {
		return cn.Summary
	}
	switch flow {
	case "ses":
		region, mode, ok := strings.Cut(cn.Summary, " · ")
		if !ok || mode != "sandbox" && mode != "production" {
			return cn.Summary
		}
		return mSummarySES.Text(l, i18n.Args{"region": region, "sandbox": yesNo(mode == "sandbox")})
	case "smtp":
		return cn.Summary
	}
	label := labelOf(func(k string) string { return cn.Settings[k] })
	if apiSummary(i18n.Default, label, cn.Account) != cn.Summary || cn.Account != "" && cn.AccountText.IsZero() {
		return cn.Summary
	}
	return apiSummary(l, label, textIn(cn.Account, cn.AccountText, l))
}

// apiSummary is the stepper's line for an API provider: its region or
// server, and the first words of the account's standing ("us · production").
func apiSummary(l i18n.Lang, label, account string) string {
	parts := []string{}
	if label != "" {
		parts = append(parts, label)
	}
	if account != "" {
		parts = append(parts, strings.ToLower(strings.SplitN(account, " · ", 2)[0]))
	}
	if len(parts) == 0 {
		return mSummaryConnected.Text(l, nil)
	}
	return strings.Join(parts, " · ")
}

// labelOf names a connection by its region or server, from its settings.
func labelOf(setting func(key string) string) string {
	for _, key := range []string{"region", "server", "message_stream", "server_url"} {
		if v := setting(key); v != "" {
			if u, err := url.Parse(v); err == nil && u.Host != "" {
				v = u.Host
			}
			return v
		}
	}
	return ""
}

// suggestedDomain is the domain step's first proposal: the domain of the
// workspace's default sender, else its first sending domain.
func (s *Service) suggestedDomain(ctx context.Context, ws *store.Workspace) string {
	if ws.DefaultSenderID != nil {
		if sn, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: *ws.DefaultSenderID}); err == nil {
			return sn.DomainName
		}
	}
	if doms, err := s.Q.DomList(ctx, ws.ID); err == nil && len(doms) > 0 {
		return doms[0].Domain
	}
	return ""
}

// regionOrder is how the region select lists regions: the most used first.
var regionOrder = []string{"us-east-1", "eu-west-1", "us-west-2", "eu-central-1", "us-east-2", "eu-west-2", "ap-southeast-2", "ap-northeast-1", "ca-central-1",
	"ap-south-1", "eu-west-3", "eu-north-1", "ap-southeast-1", "ap-northeast-2", "sa-east-1", "us-west-1", "ap-northeast-3", "eu-south-1", "me-south-1", "af-south-1", "il-central-1"}

func regionRank(r string) int {
	if i := slices.Index(regionOrder, r); i >= 0 {
		return i
	}
	return len(regionOrder)
}

func regionLabel(l i18n.Lang, r RegionState) string {
	args := i18n.Args{"region": r.Region, "hasName": yesNo(r.Name != ""), "name": r.Name, "state": "sandbox", "hasQuota": yesNo(r.Max24h > 0), "quota": r.Max24h}
	switch {
	case r.Err != "":
		args["state"] = "unavailable"
	case r.Production:
		args["state"] = "production"
	}
	return mRegionLabel.Text(l, args)
}

// connectionView is step 2.
func (s *Service) connectionView(ctx context.Context, l i18n.Lang, ws *store.Workspace, d providers.Descriptor, state State, prov *store.Provider) ConnectConnection {
	cn := state.Connection
	out := ConnectConnection{Status: "none", At: cn.At, Account: textIn(cn.Account, cn.AccountText, l), AccountID: cn.AccountID,
		Summary: connectionSummaryIn(l, flowOf(d), cn), Fields: []ConnectField{},
		Access: cn.Access, ExternalID: cn.ExternalID, RoleARN: cn.RoleARN, Region: cn.Region, Regions: []ConnectRegion{}, Provisioning: []ConnectProvisionRow{},
		Events: "webhook", EventsCheck: "none"}
	switch cn.Status {
	case "connected":
		out.Status = "connected"
	case "failed":
		out.Status = "failed"
	}
	if cn.Error != nil {
		out.Error = &ConnectError{Field: cn.Error.Field, Code: cn.Error.Code, Message: textsIn(cn.Error.Message, cn.Error.Texts, l), Detail: cn.Error.Detail}
	}
	flow := flowOf(d)
	if flow == "ses" {
		h := s.host(ctx)
		out.Host = &ConnectHost{OnAWS: h.OnAWS, Kind: h.Kind, BaseCredentials: h.BaseCredentials, CheckedAt: h.CheckedAt, Recommended: h.RecommendedAccess()}
		if out.Access == "" {
			out.Access = h.RecommendedAccess()
		}
		if cn.Events == providers.EventsQueue {
			out.Events = providers.EventsQueue
		}
		if prov != nil {
			out.EventsURL = s.webhookURL(*prov, "")
		} else {
			out.EventsURL = s.webhookBase("") + "/hooks/" + d.Adapter + "/…"
		}
		out.AccessKey, out.IAMUser = cn.Settings["access_key_hint"], cn.Settings["iam_user"]
		out.EventsCheck, out.EventsAt = eventsCheckOf(cn.Provisioning)
		regions := slices.Clone(cn.Regions)
		if len(regions) == 0 && cn.Region != "" {
			regions = []RegionState{{Region: cn.Region, Name: regionName(cn.Region), Production: !strings.Contains(strings.ToLower(cn.Account), "sandbox") && cn.Status == "connected"}}
		}
		slices.SortStableFunc(regions, func(a, b RegionState) int {
			ka, kb := regionKey(a, cn.Region), regionKey(b, cn.Region)
			if ka != kb {
				return ka - kb
			}
			return regionRank(a.Region) - regionRank(b.Region)
		})
		for _, r := range regions {
			out.Regions = append(out.Regions, ConnectRegion{Region: r.Region, Name: r.Name, Production: r.Production, Max24h: r.Max24h, MaxRate: r.MaxRate, Label: regionLabel(l, r), Err: r.Err})
		}
		if t, err := s.accessTemplate(ctx, d, cn); err == nil {
			out.Template = &t
		}
		for _, p := range cn.Provisioning {
			row := ConnectProvisionRow{Key: p.Key, Text: p.in(l, "label", p.Label), Status: p.Status, At: p.At, Error: p.errorIn(l), Hint: p.in(l, "hint", p.Hint)}
			if p.Status == "done" {
				out.ProvisionDone++
				if p.Text != "" {
					row.Text = p.in(l, "text", p.Text)
				}
				if p.Note != "" {
					row.Note = p.in(l, "note", p.Note)
				}
			}
			if p.Status == statusDeferred && p.Text != "" {
				row.Text = p.in(l, "text", p.Text)
			}
			if p.Status == "running" || p.Status == "waiting" {
				out.Working = true
			}
			out.Provisioning = append(out.Provisioning, row)
		}
		out.ProvisionTotal = len(cn.Provisioning)
		if hasFailed(cn.Provisioning) {
			out.Working = false
		}
		out.Sandbox = s.sandboxView(ctx, ws, cn)
	}
	if flow == "smtp" {
		out.SMTPForm = smtpForm(prov, cn)
		if cn.SMTP != nil {
			out.SMTP = probeView(l, cn.SMTP)
		}
		if cn.Mailbox != nil {
			out.Mailbox = probeView(l, cn.Mailbox)
		}
	}
	if cn.Webhook != nil {
		w := cn.Webhook
		out.Webhook = &ConnectWebhook{Mode: w.Mode, URL: w.URL, PublicURL: w.PublicURL, NeedsSecret: w.NeedsSecret, SecretHeader: w.SecretHeader, CanTest: w.CanTest}
		if w.SelfTest != nil {
			out.Webhook.SelfTest = &ConnectSelfTest{OK: w.SelfTest.OK, At: w.SelfTest.At, Code: w.SelfTest.Code, Reason: textIn(w.SelfTest.Reason, w.SelfTest.ReasonText, l),
				Private: w.SelfTest.Private}
		}
		if w.Registration != nil {
			out.Webhook.Registration = &ConnectRegistration{Status: w.Registration.Status, Detail: textIn(w.Registration.Detail, w.Registration.DetailText, l), At: w.Registration.At}
		}
		if prov != nil && w.Mode == "webhook" {
			// The test webhook arrived when the provider delivered anything
			// since it was asked for one.
			s.webhookTestArrival(ctx, *prov, w)
		}
		if w.TestEventAt != nil {
			out.Webhook.Test = webhookTestView(w, s.now())
			if out.Webhook.Test.Status == "waiting" {
				out.Working = true
			}
		}
	}
	out.CanContinue = canContinue(flow, cn, prov)
	return out
}

// eventsCheckOf is the state of a setup's event check (PR-3).
func eventsCheckOf(rows []ProvisionState) (string, *time.Time) {
	for _, r := range rows {
		if r.Key == eventSendKey && r.Status == statusDeferred {
			return "deferred", nil
		}
		if r.Key != eventCheckKey {
			continue
		}
		switch r.Status {
		case "done":
			return "verified", r.At
		case "failed":
			return "failed", nil
		}
		return "waiting", nil
	}
	for _, r := range rows {
		if r.Key == eventSendKey && r.Status == "failed" {
			return "failed", nil
		}
	}
	return "none", nil
}

// webhookTestWait is how long the provider's test webhook may take.
const webhookTestWait = 60 * time.Second

// webhookTestView is the provider's test webhook as the wizard shows it.
func webhookTestView(w *WebhookState, now time.Time) *ConnectWebhookTest {
	t := &ConnectWebhookTest{RequestedAt: *w.TestEventAt, Until: w.TestEventAt.Add(webhookTestWait), Status: "waiting"}
	switch {
	case w.TestEventErr != "":
		t.Status, t.Error, t.Until = "refused", w.TestEventErr, *w.TestEventAt
	case w.TestArrivedAt != nil:
		t.Status, t.ArrivedAt = "arrived", w.TestArrivedAt
	case !now.Before(t.Until):
		t.Status = "missing"
	}
	return t
}

// webhookTestArrival notes when the provider's first event after the test
// request arrived (in w; the caller saves it when it should last).
func (s *Service) webhookTestArrival(ctx context.Context, p store.Provider, w *WebhookState) {
	if w.TestEventAt == nil || w.TestArrivedAt != nil || w.TestEventErr != "" {
		return
	}
	// A little slack before the request: the provider may answer while
	// ReputeCast still waits for the request's own reply.
	ev, err := s.Q.ConnectFirstEventSince(ctx, store.ConnectFirstEventSinceParams{ProviderID: p.ID, Since: w.TestEventAt.Add(-10 * time.Second)})
	if err != nil {
		return
	}
	at := ev.ReceivedAt
	w.TestArrivedAt = &at
}

func regionKey(r RegionState, selected string) int {
	switch {
	case r.Region == selected:
		return 0
	case r.Err != "":
		return 3
	case r.Production:
		return 1
	}
	return 2
}

func regionName(region string) string {
	names := map[string]string{"us-east-1": "N. Virginia", "us-east-2": "Ohio", "us-west-1": "N. California", "us-west-2": "Oregon", "eu-west-1": "Ireland",
		"eu-west-2": "London", "eu-west-3": "Paris", "eu-central-1": "Frankfurt", "eu-north-1": "Stockholm", "ap-southeast-2": "Sydney", "ap-northeast-1": "Tokyo"}
	return names[region]
}

func hasFailed(rows []ProvisionState) bool {
	for _, r := range rows {
		if r.Status == "failed" {
			return true
		}
	}
	return false
}

func probeView(l i18n.Lang, p *ProbeState) *ConnectProbe {
	return &ConnectProbe{OK: p.OK, At: p.At, Summary: textIn(p.Summary, p.SummaryText, l), Transcript: strings.Join(p.Transcript, " · "),
		Error: textIn(p.Error, p.ErrorText, l)}
}

// in is one of a provisioning row's texts for a reader.
func (p ProvisionState) in(l i18n.Lang, field, english string) string {
	return textIn(english, p.Texts[field], l)
}

// errorIn is a provisioning row's error for a reader: the provider's own
// error as it is, ReputeCast's in the reader's language.
func (p ProvisionState) errorIn(l i18n.Lang) string {
	if t, ok := p.Texts["reason"]; ok && !english(l) {
		if code, _, ok := strings.Cut(p.Error, ": "); ok {
			if s := t.In(l); s != "" {
				return code + ": " + s
			}
		}
		return p.Error
	}
	return p.in(l, "error", p.Error)
}

// canContinue: connected, and events can come back (webhook reachable and
// registered, polling chosen, or a bounce mailbox that works).
func canContinue(flow string, cn Connection, prov *store.Provider) bool {
	if prov == nil || cn.Status != "connected" {
		return false
	}
	switch flow {
	case "ses":
		// Events must be verified (PR-3); a check that waits for a verified
		// domain lets the setup go on to the domain step. Setups from before
		// the check have no event rows.
		switch check, _ := eventsCheckOf(cn.Provisioning); check {
		case "verified", "deferred", "none":
			return true
		}
		return false
	case "smtp":
		return cn.SMTP != nil && cn.SMTP.OK && cn.Mailbox != nil && cn.Mailbox.OK
	}
	w := cn.Webhook
	if w == nil {
		return true
	}
	switch w.Mode {
	case "polling", "none", "mailbox":
		return true
	}
	if w.Registration == nil || w.Registration.Status == "failed" || w.Registration.Status == "skipped" {
		return false
	}
	if w.CanTest {
		// The provider proved it reaches the endpoint (PR-9).
		return w.TestArrivedAt != nil
	}
	// Without a test on request, the test send's first event confirms the
	// endpoint; an address no provider can reach stops here.
	return w.SelfTest == nil || w.SelfTest.OK
}

// sandboxView is the sandbox card: the selected region when it is in the
// sandbox, else the first region of the select that is.
func (s *Service) sandboxView(ctx context.Context, ws *store.Workspace, cn Connection) *ConnectSandbox {
	var target *RegionState
	sorted := slices.Clone(cn.Regions)
	slices.SortStableFunc(sorted, func(a, b RegionState) int { return regionRank(a.Region) - regionRank(b.Region) })
	for i := range sorted {
		r := sorted[i]
		if r.Err != "" || r.Production {
			continue
		}
		if r.Region == cn.Region {
			target = &sorted[i]
			break
		}
		if target == nil {
			target = &sorted[i]
		}
	}
	if target == nil {
		return nil
	}
	out := &ConnectSandbox{Region: target.Region, Name: target.Name, Selected: target.Region == cn.Region, MailType: "marketing", Language: "en"}
	if pa := cn.ProductionAccess; pa != nil && pa.Region == target.Region {
		t := pa.RequestedAt
		out.RequestedAt = &t
		out.Website, out.MailType, out.UseCase, out.DailyVolume, out.Language = pa.Website, pa.MailType, pa.UseCase, pa.DailyVolume, pa.Language
		return out
	}
	out.Website, out.UseCase = s.useCase(ctx, ws)
	for _, r := range cn.Regions {
		if r.Production && r.Max24h > out.DailyVolume {
			out.DailyVolume = r.Max24h
		}
	}
	if out.DailyVolume == 0 {
		out.DailyVolume = 10000
	}
	return out
}

// useCase prefills the production-access request from the workspace: its
// website (the default sender's organizational domain), lists and
// subscribers.
func (s *Service) useCase(ctx context.Context, ws *store.Workspace) (string, string) {
	website := ""
	if dom := s.suggestedDomain(ctx, ws); dom != "" {
		website = "https://" + dnscheck.OrgDomain(dom)
	}
	lists, _ := s.Q.ConnectListNames(ctx, ws.ID)
	subs, _ := s.Q.ConnectActiveContacts(ctx, ws.ID)
	what := ws.Name + " newsletters"
	if len(lists) > 0 {
		what = joinAnd(lists)
	}
	n := subs
	if n >= 1000 {
		n = int64(math.Floor(float64(n)/1000) * 1000)
	}
	text := what
	if n > 0 {
		text += fmt.Sprintf(" for %s opted-in subscribers (double opt-in)", thousands(int(n)))
	}
	text += ". Bounces and complaints are processed automatically via SNS and suppressed across all sending providers. One-click unsubscribe headers on every message."
	return website, text
}

// thousands writes a count with commas ("50,000"): the request to AWS is
// in English.
func thousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func joinAnd(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

// smtpMailbox is the bounce mailbox in an SMTP provider's settings.
type smtpMailbox struct {
	Protocol        string `json:"protocol"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	TLS             string `json:"tls"`
	Username        string `json:"username"`
	Folder          string `json:"folder"`
	PollEvery       string `json:"poll_every"`
	AfterProcessing string `json:"after_processing"`
}

// smtpForm returns the SMTP form's values from the provider's settings.
func smtpForm(prov *store.Provider, cn Connection) *ConnectSMTPForm {
	f := &ConnectSMTPForm{Port: 587, TLS: "starttls", Protocol: "imap", MailboxPort: 993, MailboxTLS: "tls", Folder: "INBOX", PollEvery: "1m", AfterProcessing: "move"}
	if prov == nil {
		if v := cn.Settings["host"]; v != "" {
			f.Host = v
		}
		return f
	}
	var c struct {
		Host       string       `json:"host"`
		Port       int          `json:"port"`
		TLS        string       `json:"tls"`
		Username   string       `json:"username"`
		ReturnPath string       `json:"return_path"`
		MaxRate    string       `json:"max_rate"`
		DailyCap   string       `json:"daily_cap"`
		Mailbox    *smtpMailbox `json:"bounce_mailbox"`
	}
	_ = json.Unmarshal(prov.Config, &c)
	f.Host, f.Username, f.ReturnPath, f.MaxRate, f.DailyCap = c.Host, c.Username, c.ReturnPath, c.MaxRate, c.DailyCap
	if c.Port != 0 {
		f.Port = c.Port
	}
	if c.TLS != "" {
		f.TLS = c.TLS
	}
	f.PasswordSet = len(prov.CredentialsEnc) > 0
	// The mailbox is saved with its password once a test passes; after a
	// failed test the form shows what was tried (the password isn't kept).
	f.MailboxPassSet = c.Mailbox != nil
	if t := cn.MailboxTried; t != nil && (c.Mailbox == nil || cn.Mailbox == nil || !cn.Mailbox.OK) {
		c.Mailbox = &smtpMailbox{Protocol: t.Protocol, Host: t.Host, Port: t.Port, TLS: t.TLS, Username: t.Username, Folder: t.Folder,
			PollEvery: t.PollEvery, AfterProcessing: t.AfterProcessing}
		if t.Address != "" {
			f.ReturnPath = t.Address
		}
	}
	if m := c.Mailbox; m != nil {
		f.MailboxHost, f.MailboxUser = m.Host, m.Username
		if m.Protocol != "" {
			f.Protocol = m.Protocol
		}
		if m.Port != 0 {
			f.MailboxPort = m.Port
		}
		if m.TLS != "" {
			f.MailboxTLS = m.TLS
		}
		if m.Folder != "" {
			f.Folder = m.Folder
		}
		if m.PollEvery != "" {
			f.PollEvery = m.PollEvery
		}
		if m.AfterProcessing != "" {
			f.AfterProcessing = m.AfterProcessing
		}
	}
	return f
}

// accessTemplate is the CloudFormation path of SES.
func (s *Service) accessTemplate(ctx context.Context, d providers.Descriptor, cn Connection) (ConnectAccess, error) {
	region := cn.Region
	if region == "" {
		region = "us-east-1"
	}
	sb, _ := json.Marshal(map[string]any{"region": region, "access": "keys"})
	a, err := d.New(providers.Config{Settings: sb, Credentials: json.RawMessage(`{"access_key_id":"AKIAPLACEHOLDER","secret_access_key":"placeholder"}`)})
	if err != nil {
		return ConnectAccess{}, err
	}
	at, ok := a.(providers.AccessTemplater)
	if !ok {
		return ConnectAccess{}, fmt.Errorf("%s has no access template", d.Name)
	}
	cctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	t, err := at.AccessTemplate(cctx, providers.AccessTemplateInput{Region: region, ExternalID: cn.ExternalID, TemplateURL: s.TemplateURL})
	if err != nil {
		return ConnectAccess{}, err
	}
	return ConnectAccess{LaunchURL: t.LaunchURL, Policy: t.Policy, Principal: t.Principal, RoleName: t.RoleName}, nil
}
