// Package twilio is the Twilio SMS adapter: Programmable Messaging with an
// API key (never the auth token, so ReputeCast's access can be revoked on
// its own), status callbacks and inbound webhooks signed with a webhook
// shared key ReputeCast creates through Twilio's API, the account's numbers,
// toll-free verification and A2P 10DLC statuses, and outbound prices.
//
// Twilio keeps its own opt-out list (Advanced Opt-Out) without an API to
// read or write it; ReputeCast learns its opt-outs from the inbound
// webhook's OptOutType and from error 21610 when a text is refused.
package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Twilio's legacy signature scheme is HMAC-SHA1
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
)

// Adapter is the registry key.
const Adapter = "twilio"

// The APIs Twilio serves from separate hosts. With Config.BaseURL (tests,
// the demo) they're paths under one address.
const (
	apiHost       = "https://api.twilio.com"
	messagingHost = "https://messaging.twilio.com"
	pricingHost   = "https://pricing.twilio.com"
	webhooksHost  = "https://webhooks.twilio.com"
)

// SettingsHeader names the shared key and algorithm that signed a webhook.
const SettingsHeader = "X-Twilio-Webhooks-Signature-Settings"

// SignatureHeader carries the signature.
const SignatureHeader = "X-Twilio-Signature"

func init() { sms.Register(Descriptor()) }

// Descriptor is the adapter's manifest.
func Descriptor() sms.Descriptor {
	return sms.Descriptor{
		Manifest: sms.Manifest{
			Adapter: Adapter, Name: "Twilio",
			Badges: []sms.Badge{
				{Capability: sms.CapTollFree, Support: sms.Yes}, {Capability: sms.Cap10DLC, Support: sms.Yes},
				{Capability: sms.CapSenderIDs, Support: sms.Yes},
				{Capability: sms.CapReceipts, Support: sms.Yes}, {Capability: sms.CapInbound, Support: sms.Yes},
				{Capability: sms.CapOptOutSync, Support: sms.Partial},
				{Capability: sms.CapRegistration, Support: sms.Yes}, {Capability: sms.CapPricing, Support: sms.Yes},
			},
			Fields: []sms.Field{
				{Key: "account_sid", Required: true},
				{Key: "api_key_sid", Required: true},
				{Key: "api_key_secret", Secret: true, Required: true},
				{Key: "mode", Options: []string{"live", "test"}, Default: "live"},
			},
			Webhooks: "registered",
			Console: func(kind, _ string) string {
				switch kind {
				case "registration":
					return "https://console.twilio.com/us1/develop/sms/regulatory-compliance/a2p-10dlc-overview"
				case "tollfree":
					return "https://console.twilio.com/us1/develop/sms/regulatory-compliance/toll-free-verifications"
				case "numbers":
					return "https://console.twilio.com/us1/develop/phone-numbers/manage/incoming"
				case "webhooks":
					return "https://console.twilio.com/us1/monitor/logs/sms"
				}
				return "https://console.twilio.com"
			},
			Version: "1.0.0", Maintainer: "ReputeCast",
			// The summary, the badges' notes and the fields' labels and hints
			// are in messages/en.json: WithMessages fills them.
		}.WithMessages(msgs),
		New: New,
	}
}

// Provider is a Twilio account.
type Provider struct {
	cfg                  sms.Config
	sid, keySID, keySecr string
	http                 *http.Client
}

// New opens a Twilio account.
func New(cfg sms.Config) (sms.Provider, error) {
	p := &Provider{cfg: cfg, sid: strings.TrimSpace(cfg.Value("account_sid")), keySID: strings.TrimSpace(cfg.Value("api_key_sid")),
		keySecr: cfg.Value("api_key_secret"), http: cfg.Client()}
	if p.sid == "" || p.keySID == "" {
		return nil, errors.New("twilio: the account SID and the API key SID are required")
	}
	return p, nil
}

var (
	_ sms.SenderLister       = (*Provider)(nil)
	_ sms.WebhookRegistrar   = (*Provider)(nil)
	_ sms.RegistrationReader = (*Provider)(nil)
	_ sms.PricingReader      = (*Provider)(nil)
)

// Adapter implements sms.Provider.
func (p *Provider) Adapter() string { return Adapter }

func (p *Provider) host(h string) string {
	if p.cfg.BaseURL == "" {
		return h
	}
	b := strings.TrimRight(p.cfg.BaseURL, "/")
	switch h {
	case messagingHost:
		return b + "/messaging"
	case pricingHost:
		return b + "/pricing"
	case webhooksHost:
		return b + "/webhooks"
	}
	return b
}

func (p *Provider) call(ctx context.Context, op, method, u string, form url.Values, js any, out any) ([]byte, error) {
	raw, err := sms.Do(ctx, p.http, sms.Call{Op: op, Method: method, URL: u, Form: form, JSON: js, BasicUser: p.keySID, BasicPass: p.keySecr}, out)
	var ce *sms.CallError
	if errors.As(err, &ce) {
		var e twErr
		if json.Unmarshal(raw, &e) == nil && e.Code != 0 {
			ce.Code, ce.Message = strconv.Itoa(e.Code), e.Message
		}
	}
	return raw, err
}

type twErr struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

// Check implements sms.Provider: it reads the account.
func (p *Provider) Check(ctx context.Context) (sms.Account, error) {
	var a struct {
		SID          string `json:"sid"`
		FriendlyName string `json:"friendly_name"`
		Status       string `json:"status"`
		Type         string `json:"type"`
	}
	if _, err := p.call(ctx, "check", http.MethodGet, p.host(apiHost)+"/2010-04-01/Accounts/"+url.PathEscape(p.sid)+".json", nil, nil, &a); err != nil {
		return sms.Account{}, err
	}
	if a.Status != "" && a.Status != "active" {
		return sms.Account{}, fmt.Errorf("%w: the account is %s", sms.ErrCredentialsRejected, a.Status)
	}
	return sms.Account{ID: a.SID, Name: a.FriendlyName, Hint: sms.Hint(a.SID, 2, 3), Live: p.cfg.Value("mode") != "test"}, nil
}

// codeReason maps Twilio's error codes to what ReputeCast does about them.
func codeReason(code string) (sms.Reason, bool) {
	switch code {
	case "21211", "21401", "21407":
		return sms.ReasonInvalid, false
	case "21614", "30006":
		return sms.ReasonLandline, false
	case "21610", "30004":
		return sms.ReasonOptedOut, false
	case "30003":
		return sms.ReasonUnreachable, true
	case "30005":
		return sms.ReasonNotInService, false
	case "30007":
		return sms.ReasonFiltered, false
	case "21408", "21612", "30032", "30034", "21606", "21659":
		return sms.ReasonRejected, false
	case "20429", "14107":
		return sms.ReasonOther, true
	}
	return sms.ReasonOther, false
}

// Send implements sms.Provider.
func (p *Provider) Send(ctx context.Context, m sms.Message) (sms.SendResult, error) {
	form := url.Values{"To": {m.To}, "From": {m.From}, "Body": {m.Body}}
	if m.StatusURL != "" {
		form.Set("StatusCallback", m.StatusURL)
	}
	var out struct {
		SID         string  `json:"sid"`
		Status      string  `json:"status"`
		NumSegments string  `json:"num_segments"`
		Price       *string `json:"price"`
	}
	raw, err := p.call(ctx, "send", http.MethodPost, p.host(apiHost)+"/2010-04-01/Accounts/"+url.PathEscape(p.sid)+"/Messages.json", form, nil, &out)
	if err != nil {
		var ce *sms.CallError
		if errors.As(err, &ce) && ce.Status != http.StatusUnauthorized && ce.Status < 500 {
			var e twErr
			_ = json.Unmarshal(raw, &e)
			code := strconv.Itoa(e.Code)
			reason, temp := codeReason(code)
			return sms.SendResult{}, &sms.SendError{Code: code, Message: e.Message, Reason: reason, Temporary: temp || ce.Status == http.StatusTooManyRequests,
				Throttled: ce.Status == http.StatusTooManyRequests || code == "20429"}
		}
		return sms.SendResult{}, err
	}
	n, _ := strconv.Atoi(out.NumSegments)
	return sms.SendResult{ProviderMessageID: out.SID, Segments: n}, nil
}

// signature computes Twilio's request signature: the full URL followed by
// each POST parameter's name and value in name order, signed with HMAC and
// base64-encoded.
func signature(h func() hash.Hash, key, fullURL string, form url.Values) string {
	mac := hmac.New(h, []byte(key))
	mac.Write([]byte(fullURL + sms.Concat(form)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Sign signs a webhook the way Twilio does with a shared key (tests, the
// demo's fake).
func Sign(secret, fullURL string, form url.Values) string {
	return signature(sha256.New, secret, fullURL, form)
}

// SignLegacy signs with the account auth token (Twilio's default scheme).
func SignLegacy(authToken, fullURL string, form url.Values) string {
	return signature(sha1.New, authToken, fullURL, form)
}

// VerifyWebhook implements sms.Provider. ReputeCast registers a shared key
// for its webhooks; a request signed with the account's auth token (no
// settings header) can't be checked with an API key's credentials and is
// refused.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, body []byte) error {
	sig := r.Header.Get(SignatureHeader)
	settings := r.Header.Get(SettingsHeader)
	if sig == "" || settings == "" || p.cfg.WebhookSecret == "" {
		return sms.ErrBadSignature
	}
	algo := "HMAC_SHA1"
	for _, part := range strings.Split(settings, ",") {
		if k, v, ok := strings.Cut(strings.TrimSpace(part), "="); ok && k == "algorithm" {
			algo = v
		}
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return sms.ErrBadSignature
	}
	h := sha1.New
	if algo == "HMAC_SHA256" {
		h = sha256.New
	}
	want := signature(h, p.cfg.WebhookSecret, r.URL.String(), form)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return sms.ErrBadSignature
	}
	return nil
}

// ParseWebhook implements sms.Provider: a status callback or an inbound
// text.
func (p *Provider) ParseWebhook(_ context.Context, _ string, body []byte) ([]sms.Event, error) {
	v, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	now := p.cfg.Clock()
	if status := v.Get("MessageStatus"); status != "" {
		e := sms.Event{ProviderMessageID: v.Get("MessageSid"), To: v.Get("To"), From: v.Get("From"), OccurredAt: now,
			Summary: sms.SortedParams(v, "MessageStatus", "ErrorCode")}
		switch status {
		case "queued", "accepted", "scheduled":
			e.Kind = sms.EventQueued
		case "sending", "sent":
			e.Kind = sms.EventSent
		case "delivered", "read":
			e.Kind = sms.EventDelivered
		case "undelivered":
			e.Kind = sms.EventUndelivered
		case "failed", "canceled":
			e.Kind = sms.EventFailed
		default:
			return nil, nil
		}
		if code := v.Get("ErrorCode"); code != "" && code != "0" {
			e.Code = code
			e.Reason, _ = codeReason(code)
		}
		if n, err := strconv.Atoi(v.Get("NumSegments")); err == nil {
			e.Segments = n
		}
		return []sms.Event{e}, nil
	}
	if v.Get("Body") != "" || v.Get("SmsMessageSid") != "" || v.Get("MessageSid") != "" {
		e := sms.Event{Kind: sms.EventInbound, ProviderMessageID: v.Get("MessageSid"), From: v.Get("From"), To: v.Get("To"), Body: v.Get("Body"),
			OptOut: strings.ToUpper(v.Get("OptOutType")), OccurredAt: now, Summary: "Inbound · Body=" + sms.Quote(v.Get("Body"))}
		return []sms.Event{e}, nil
	}
	return nil, nil
}

type number struct {
	SID          string `json:"sid"`
	PhoneNumber  string `json:"phone_number"`
	Capabilities struct {
		SMS bool `json:"sms"`
	} `json:"capabilities"`
}

func (p *Provider) numbers(ctx context.Context) ([]number, error) {
	var out struct {
		Numbers []number `json:"incoming_phone_numbers"`
	}
	_, err := p.call(ctx, "numbers", http.MethodGet, p.host(apiHost)+"/2010-04-01/Accounts/"+url.PathEscape(p.sid)+"/IncomingPhoneNumbers.json?PageSize=100", nil, nil, &out)
	return out.Numbers, err
}

// tollFree reports a North American toll-free number.
func tollFree(e164 string) bool {
	for _, pre := range []string{"+1800", "+1833", "+1844", "+1855", "+1866", "+1877", "+1888"} {
		if strings.HasPrefix(e164, pre) {
			return true
		}
	}
	return false
}

// Senders implements sms.SenderLister: the account's numbers that can
// text. Toll-free numbers send 3 texts a second; local US numbers are
// 10DLC numbers whose throughput comes with their campaign.
func (p *Provider) Senders(ctx context.Context) ([]sms.Sender, error) {
	ns, err := p.numbers(ctx)
	if err != nil {
		return nil, err
	}
	out := []sms.Sender{}
	for _, n := range ns {
		if !n.Capabilities.SMS {
			continue
		}
		s := sms.Sender{Value: n.PhoneNumber, ProviderRef: n.SID, Throughput: 1}
		pn, err := phone.Parse(n.PhoneNumber, "")
		if err != nil {
			continue
		}
		switch {
		case tollFree(n.PhoneNumber):
			s.Kind, s.Countries, s.Throughput = "toll_free", []string{"US", "CA"}, 3
		case pn.Country == "US":
			s.Kind, s.Countries = "10dlc", []string{"US"}
		default:
			s.Kind, s.Countries = "mobile", []string{pn.Country}
		}
		out = append(out, s)
	}
	return out, nil
}

// RegisterWebhooks implements sms.WebhookRegistrar: a shared key signs
// ReputeCast's webhooks (a setting with HMAC-SHA256 and a rule for its
// address), and each number sends its inbound texts to ReputeCast. Status
// callbacks are set on each text.
func (p *Provider) RegisterWebhooks(ctx context.Context, hooks sms.Webhooks, senders []sms.Sender) (sms.WebhookSetup, error) {
	var key struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	if _, err := p.call(ctx, "shared_key", http.MethodPost, p.host(webhooksHost)+"/v1/Webhooks/SharedKeys", nil, map[string]any{"friendlyName": "ReputeCast"}, &key); err != nil {
		return sms.WebhookSetup{}, err
	}
	var setting struct {
		ID string `json:"id"`
	}
	if _, err := p.call(ctx, "webhook_setting", http.MethodPost, p.host(webhooksHost)+"/v1/Webhooks/Settings", nil, map[string]any{
		"friendlyName": "ReputeCast",
		"auth":         map[string]any{"source": "INLINE", "value": map[string]any{"scheme": "NONE"}},
		"signature":    map[string]any{"type": "SHARED_KEY", "sharedKeyId": key.ID, "algorithm": "HMAC_SHA256"},
	}, &setting); err != nil {
		return sms.WebhookSetup{}, err
	}
	base := hooks.StatusURL
	if u, err := url.Parse(hooks.StatusURL); err == nil {
		base = u.Scheme + "://" + u.Host + strings.TrimSuffix(u.Path, "/")
	}
	if _, err := p.call(ctx, "webhook_rule", http.MethodPost, p.host(webhooksHost)+"/v1/Webhooks/Rules", nil, map[string]any{
		"friendlyName": "ReputeCast", "matchType": "PREFIX", "match": base,
		"settings": []map[string]any{{"id": setting.ID, "percentage": 100}},
	}, nil); err != nil {
		return sms.WebhookSetup{}, err
	}
	refs := map[string]string{}
	if ns, err := p.numbers(ctx); err == nil {
		for _, n := range ns {
			refs[n.PhoneNumber] = n.SID
		}
	}
	for _, s := range senders {
		ref := s.ProviderRef
		if ref == "" {
			ref = refs[s.Value]
		}
		if ref == "" {
			continue // a sender ID: it can't receive texts
		}
		form := url.Values{"SmsUrl": {hooks.InboundURL}, "SmsMethod": {"POST"}}
		if _, err := p.call(ctx, "number_webhook", http.MethodPost, p.host(apiHost)+"/2010-04-01/Accounts/"+url.PathEscape(p.sid)+"/IncomingPhoneNumbers/"+url.PathEscape(ref)+".json", form, nil, nil); err != nil {
			return sms.WebhookSetup{}, err
		}
	}
	return sms.WebhookSetup{Secret: key.Secret}, nil
}

func tollFreeStatus(s string) string {
	switch s {
	case "TWILIO_APPROVED":
		return "verified"
	case "TWILIO_REJECTED":
		return "rejected"
	case "PENDING_REVIEW", "IN_REVIEW":
		return "pending"
	}
	return "not_registered"
}

func a2pStatus(s string) string {
	switch strings.ToUpper(s) {
	case "APPROVED", "VERIFIED":
		return "approved"
	case "FAILED", "REJECTED":
		return "rejected"
	case "PENDING", "IN_PROGRESS", "IN_REVIEW":
		return "pending"
	}
	return "not_registered"
}

func day(s string) *time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// Registrations implements sms.RegistrationReader: toll-free verifications
// for toll-free numbers, and the A2P brand and campaign of the messaging
// service a 10DLC number belongs to.
func (p *Provider) Registrations(ctx context.Context, senders []sms.Sender) ([]sms.Registration, error) {
	ns, err := p.numbers(ctx)
	if err != nil {
		return nil, err
	}
	bySID := map[string]string{}
	for _, n := range ns {
		bySID[n.SID] = n.PhoneNumber
	}
	want := map[string]bool{}
	for _, s := range senders {
		want[s.Value] = true
	}
	out := []sms.Registration{}
	var tf struct {
		Verifications []struct {
			SID                    string `json:"sid"`
			TollfreePhoneNumberSID string `json:"tollfree_phone_number_sid"`
			Status                 string `json:"status"`
			RejectionReason        string `json:"rejection_reason"`
			UseCaseSummary         string `json:"use_case_summary"`
			MessageVolume          string `json:"message_volume"`
			OptInType              string `json:"opt_in_type"`
			DateCreated            string `json:"date_created"`
			DateUpdated            string `json:"date_updated"`
		} `json:"verifications"`
	}
	if _, err := p.call(ctx, "tollfree_verifications", http.MethodGet, p.host(messagingHost)+"/v1/Tollfree/Verifications?PageSize=50", nil, nil, &tf); err != nil {
		return nil, err
	}
	for _, v := range tf.Verifications {
		num := bySID[v.TollfreePhoneNumberSID]
		if num == "" || !want[num] {
			continue
		}
		out = append(out, sms.Registration{Sender: num, Kind: "toll_free", Status: tollFreeStatus(v.Status), Reason: v.RejectionReason, Since: day(v.DateUpdated),
			Details: map[string]string{"useCase": v.UseCaseSummary, "volume": v.MessageVolume, "optIn": v.OptInType, "submitted": v.DateCreated}})
	}
	var brands struct {
		Data []struct {
			SID       string `json:"sid"`
			Status    string `json:"status"`
			Name      string `json:"brand_name"`
			Failure   string `json:"failure_reason"`
			Updated   string `json:"date_updated"`
			BrandType string `json:"brand_type"`
		} `json:"data"`
	}
	if _, err := p.call(ctx, "brands", http.MethodGet, p.host(messagingHost)+"/v1/a2p/BrandRegistrations", nil, nil, &brands); err != nil {
		return nil, err
	}
	var services struct {
		Services []struct {
			SID string `json:"sid"`
		} `json:"services"`
	}
	if _, err := p.call(ctx, "services", http.MethodGet, p.host(messagingHost)+"/v1/Services?PageSize=50", nil, nil, &services); err != nil {
		return nil, err
	}
	for _, svc := range services.Services {
		var pns struct {
			PhoneNumbers []struct {
				PhoneNumber string `json:"phone_number"`
			} `json:"phone_numbers"`
		}
		if _, err := p.call(ctx, "service_numbers", http.MethodGet, p.host(messagingHost)+"/v1/Services/"+url.PathEscape(svc.SID)+"/PhoneNumbers", nil, nil, &pns); err != nil {
			return nil, err
		}
		var camps struct {
			Compliance []struct {
				SID            string   `json:"sid"`
				Status         string   `json:"campaign_status"`
				Description    string   `json:"description"`
				BrandSID       string   `json:"brand_registration_sid"`
				Updated        string   `json:"date_updated"`
				FailureReason  string   `json:"failure_reason"`
				UseCase        string   `json:"us_app_to_person_usecase"`
				MessageFlow    string   `json:"message_flow"`
				MessageSamples []string `json:"message_samples"`
			} `json:"compliance"`
		}
		if _, err := p.call(ctx, "campaigns", http.MethodGet, p.host(messagingHost)+"/v1/Services/"+url.PathEscape(svc.SID)+"/Compliance/Usa2p", nil, nil, &camps); err != nil {
			return nil, err
		}
		for _, pn := range pns.PhoneNumbers {
			if !want[pn.PhoneNumber] {
				continue
			}
			for _, c := range camps.Compliance {
				for _, b := range brands.Data {
					if b.SID == c.BrandSID {
						out = append(out, sms.Registration{Sender: pn.PhoneNumber, Kind: "brand", Name: b.Name, Status: a2pStatus(b.Status), Reason: b.Failure, Since: day(b.Updated)})
					}
				}
				out = append(out, sms.Registration{Sender: pn.PhoneNumber, Kind: "campaign", Name: c.Description, Status: a2pStatus(c.Status), Reason: c.FailureReason,
					Since: day(c.Updated), Details: map[string]string{"useCase": c.UseCase, "optIn": c.MessageFlow}})
			}
		}
	}
	return out, nil
}

// Prices implements sms.PricingReader: the price of an outbound segment
// per country, as Twilio quotes it (carrier fees included where Twilio
// adds them).
func (p *Provider) Prices(ctx context.Context, countries []string) (map[string]float64, error) {
	out := map[string]float64{}
	for _, c := range countries {
		var pr struct {
			Outbound []struct {
				Prices []struct {
					NumberType   string `json:"number_type"`
					CurrentPrice string `json:"current_price"`
				} `json:"prices"`
			} `json:"outbound_sms_prices"`
		}
		if _, err := p.call(ctx, "pricing", http.MethodGet, p.host(pricingHost)+"/v1/Messaging/Countries/"+url.PathEscape(strings.ToUpper(c)), nil, nil, &pr); err != nil {
			var ce *sms.CallError
			if errors.As(err, &ce) && ce.Status == http.StatusNotFound {
				continue
			}
			return nil, err
		}
		best := 0.0
		for _, o := range pr.Outbound {
			for _, x := range o.Prices {
				if v, err := strconv.ParseFloat(x.CurrentPrice, 64); err == nil && v > best {
					best = v
				}
			}
		}
		if best > 0 {
			out[strings.ToUpper(c)] = best
		}
	}
	return out, nil
}
