// Package plivo is the Plivo SMS adapter: the Message API with the account's
// auth ID and token, status callbacks per text and inbound texts through a
// Plivo application on the account's numbers, both signed with Plivo's V3
// signature (HMAC-SHA256 with the auth token and a nonce), 10DLC and
// toll-free verification statuses, and outbound prices.
//
// Plivo keeps no opt-out list ReputeCast can read; keywords reach
// ReputeCast as inbound texts.
package plivo

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
)

// Adapter is the registry key.
const Adapter = "plivo"

const apiHost = "https://api.plivo.com"

// Signature headers.
const (
	SignatureHeader = "X-Plivo-Signature-V3"
	NonceHeader     = "X-Plivo-Signature-V3-Nonce"
)

func init() { sms.Register(Descriptor()) }

// Descriptor is the adapter's manifest.
func Descriptor() sms.Descriptor {
	return sms.Descriptor{
		Manifest: sms.Manifest{
			Adapter: Adapter, Name: "Plivo",
			Badges: []sms.Badge{
				{Capability: sms.CapTollFree, Support: sms.Yes}, {Capability: sms.Cap10DLC, Support: sms.Yes}, {Capability: sms.CapSenderIDs, Support: sms.Yes},
				{Capability: sms.CapReceipts, Support: sms.Yes}, {Capability: sms.CapInbound, Support: sms.Yes},
				{Capability: sms.CapOptOutSync, Support: sms.No},
				{Capability: sms.CapRegistration, Support: sms.Yes}, {Capability: sms.CapPricing, Support: sms.Yes},
			},
			Fields: []sms.Field{
				{Key: "auth_id", Required: true},
				{Key: "auth_token", Secret: true, Required: true},
			},
			Webhooks: "registered",
			Console: func(kind, _ string) string {
				switch kind {
				case "registration":
					return "https://console.plivo.com/messaging/10dlc/campaigns/"
				case "numbers":
					return "https://console.plivo.com/active-phone-numbers/"
				}
				return "https://console.plivo.com/dashboard/"
			},
			Version: "1.0.0", Maintainer: "ReputeCast",
			// The summary, the badges' notes and the fields' labels and hints
			// are in messages/en.json: WithMessages fills them.
		}.WithMessages(msgs),
		New: New,
	}
}

// Provider is a Plivo account.
type Provider struct {
	cfg       sms.Config
	id, token string
	http      *http.Client
}

// New opens a Plivo account.
func New(cfg sms.Config) (sms.Provider, error) {
	p := &Provider{cfg: cfg, id: strings.TrimSpace(cfg.Value("auth_id")), token: cfg.Value("auth_token"), http: cfg.Client()}
	if p.id == "" {
		return nil, errors.New("plivo: the auth ID is required")
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

func (p *Provider) url(path string) string {
	base := apiHost
	if p.cfg.BaseURL != "" {
		base = strings.TrimRight(p.cfg.BaseURL, "/")
	}
	return base + "/v1/Account/" + url.PathEscape(p.id) + path
}

func (p *Provider) call(ctx context.Context, op, method, path string, js, out any) ([]byte, error) {
	return sms.Do(ctx, p.http, sms.Call{Op: op, Method: method, URL: p.url(path), JSON: js, BasicUser: p.id, BasicPass: p.token}, out)
}

// Check implements sms.Provider.
func (p *Provider) Check(ctx context.Context) (sms.Account, error) {
	var a struct {
		Name   string `json:"name"`
		AuthID string `json:"auth_id"`
	}
	if _, err := p.call(ctx, "check", http.MethodGet, "/", nil, &a); err != nil {
		return sms.Account{}, err
	}
	return sms.Account{ID: a.AuthID, Name: a.Name, Hint: sms.Hint(a.AuthID, 2, 4), Live: true}, nil
}

// codeReason maps Plivo's message error codes.
func codeReason(code string) (sms.Reason, bool) {
	switch code {
	case "30", "100":
		return sms.ReasonFiltered, false // spam detected, prohibited by carrier
	case "50", "450":
		return sms.ReasonInvalid, false
	case "70":
		return sms.ReasonNotInService, false // permanently unavailable
	case "80", "20":
		return sms.ReasonUnreachable, true // temporarily unavailable, network error
	case "40", "90", "200":
		return sms.ReasonRejected, false
	}
	return sms.ReasonOther, false
}

// Send implements sms.Provider.
func (p *Provider) Send(ctx context.Context, m sms.Message) (sms.SendResult, error) {
	in := map[string]any{"src": m.From, "dst": sms.E164Digits(m.To), "text": m.Body, "type": "sms"}
	if strings.HasPrefix(m.From, "+") {
		in["src"] = sms.E164Digits(m.From)
	}
	if m.StatusURL != "" {
		in["url"], in["method"] = m.StatusURL, "POST"
	}
	var out struct {
		MessageUUID []string `json:"message_uuid"`
	}
	raw, err := p.call(ctx, "send", http.MethodPost, "/Message/", in, &out)
	if err != nil {
		var ce *sms.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusBadRequest {
			var e struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(raw, &e)
			reason := sms.ReasonRejected
			if strings.Contains(e.Error, "dst") {
				reason = sms.ReasonInvalid
			}
			return sms.SendResult{}, &sms.SendError{Code: "400", Message: e.Error, Reason: reason}
		}
		if errors.As(err, &ce) && ce.Status == http.StatusTooManyRequests {
			return sms.SendResult{}, &sms.SendError{Code: "429", Message: "Too many requests", Temporary: true, Throttled: true}
		}
		return sms.SendResult{}, err
	}
	if len(out.MessageUUID) == 0 {
		return sms.SendResult{}, &sms.SendError{Message: "Plivo returned no message id", Temporary: true}
	}
	return sms.SendResult{ProviderMessageID: out.MessageUUID[0], Segments: len(out.MessageUUID)}, nil
}

// sortedParams is Plivo's parameter string: each name and value, in name
// order.
func sortedParams(v url.Values) string { return sms.Concat(v) }

// Signature computes Plivo's V3 signature: the URL (its query sorted), for
// POST requests a dot and the sorted parameters, then a dot and the nonce,
// signed with HMAC-SHA256 and base64-encoded.
func Signature(token, method, fullURL, nonce string, form url.Values) string {
	u, err := url.Parse(fullURL)
	base := fullURL
	if err == nil {
		q := u.Query()
		u.RawQuery = ""
		base = u.String()
		if len(q) > 0 {
			keys := make([]string, 0, len(q))
			for k := range q {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := []string{}
			for _, k := range keys {
				parts = append(parts, k+"="+q.Get(k))
			}
			base += "?" + strings.Join(parts, "&")
		}
	}
	if method == http.MethodPost {
		base += "." + sortedParams(form)
	}
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write([]byte(base + "." + nonce))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyWebhook implements sms.Provider. The header may carry several
// signatures (during a token rotation); one has to match.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, body []byte) error {
	sigs, nonce := r.Header.Get(SignatureHeader), r.Header.Get(NonceHeader)
	if sigs == "" || nonce == "" || p.token == "" {
		return sms.ErrBadSignature
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return sms.ErrBadSignature
	}
	want := Signature(p.token, r.Method, r.URL.String(), nonce, form)
	for _, s := range strings.Split(sigs, ",") {
		if hmac.Equal([]byte(strings.TrimSpace(s)), []byte(want)) {
			return nil
		}
	}
	return sms.ErrBadSignature
}

func plus(n string) string {
	if n == "" || strings.HasPrefix(n, "+") {
		return n
	}
	if _, err := strconv.ParseUint(n, 10, 64); err != nil {
		return n
	}
	return "+" + n
}

// ParseWebhook implements sms.Provider.
func (p *Provider) ParseWebhook(_ context.Context, _ string, body []byte) ([]sms.Event, error) {
	v, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	now := p.cfg.Clock()
	if status := v.Get("Status"); status != "" {
		e := sms.Event{ProviderMessageID: v.Get("MessageUUID"), To: plus(v.Get("To")), From: plus(v.Get("From")), OccurredAt: now,
			Summary: sms.SortedParams(v, "Status", "ErrorCode")}
		switch status {
		case "queued":
			e.Kind = sms.EventQueued
		case "sent":
			e.Kind = sms.EventSent
		case "delivered":
			e.Kind = sms.EventDelivered
		case "undelivered":
			e.Kind = sms.EventUndelivered
		case "failed", "rejected":
			e.Kind = sms.EventFailed
		default:
			return nil, nil
		}
		if c := v.Get("ErrorCode"); c != "" && c != "000" && c != "0" {
			e.Code = c
			e.Reason, _ = codeReason(c)
		}
		if u, err := strconv.ParseInt(v.Get("Units"), 10, 32); err == nil {
			e.Segments = int(u)
		}
		if a, err := strconv.ParseFloat(v.Get("TotalAmount"), 64); err == nil {
			e.Price = &a
		}
		return []sms.Event{e}, nil
	}
	if v.Get("Text") != "" || v.Get("Type") == "sms" {
		return []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: v.Get("MessageUUID"), From: plus(v.Get("From")), To: plus(v.Get("To")), Body: v.Get("Text"),
			OccurredAt: now, Summary: "Inbound · Text=" + sms.Quote(v.Get("Text"))}}, nil
	}
	return nil, nil
}

type number struct {
	Number     string `json:"number"`
	SMSEnabled bool   `json:"sms_enabled"`
	Type       string `json:"type"`
}

func (p *Provider) numbers(ctx context.Context) ([]number, error) {
	var out struct {
		Objects []number `json:"objects"`
	}
	_, err := p.call(ctx, "numbers", http.MethodGet, "/Number/?limit=20", nil, &out)
	return out.Objects, err
}

// Senders implements sms.SenderLister.
func (p *Provider) Senders(ctx context.Context) ([]sms.Sender, error) {
	ns, err := p.numbers(ctx)
	if err != nil {
		return nil, err
	}
	out := []sms.Sender{}
	for _, n := range ns {
		if !n.SMSEnabled {
			continue
		}
		e := plus(n.Number)
		pn, err := phone.Parse(e, "")
		if err != nil {
			continue
		}
		s := sms.Sender{Value: e, Countries: []string{pn.Country}, Throughput: 1}
		switch {
		case n.Type == "tollfree":
			s.Kind, s.Throughput = "toll_free", 3
			if pn.Country == "US" {
				s.Countries = []string{"US", "CA"}
			}
		case pn.Country == "US":
			s.Kind = "10dlc"
		default:
			s.Kind = "mobile"
		}
		out = append(out, s)
	}
	return out, nil
}

// RegisterWebhooks implements sms.WebhookRegistrar: an application whose
// message URL is ReputeCast's, linked to each number. Status callbacks are
// set on each text. Plivo signs with the auth token ReputeCast already
// holds.
func (p *Provider) RegisterWebhooks(ctx context.Context, hooks sms.Webhooks, senders []sms.Sender) (sms.WebhookSetup, error) {
	var app struct {
		AppID string `json:"app_id"`
	}
	if _, err := p.call(ctx, "application", http.MethodPost, "/Application/", map[string]any{"app_name": "ReputeCast", "message_url": hooks.InboundURL, "message_method": "POST"}, &app); err != nil {
		return sms.WebhookSetup{}, err
	}
	for _, s := range senders {
		if !strings.HasPrefix(s.Value, "+") {
			continue
		}
		if _, err := p.call(ctx, "number_app", http.MethodPost, "/Number/"+url.PathEscape(sms.E164Digits(s.Value))+"/", map[string]any{"app_id": app.AppID}, nil); err != nil {
			return sms.WebhookSetup{}, err
		}
	}
	return sms.WebhookSetup{}, nil
}

func regStatus(s string) string {
	switch strings.ToUpper(s) {
	case "COMPLETED", "APPROVED", "VERIFIED", "ACTIVE":
		return "approved"
	case "FAILED", "REJECTED":
		return "rejected"
	case "PENDING", "PROCESSING", "SUBMITTED", "IN_PROGRESS":
		return "pending"
	}
	return "not_registered"
}

// Registrations implements sms.RegistrationReader: toll-free verification
// for toll-free numbers, the 10DLC brand and campaign of local US numbers.
func (p *Provider) Registrations(ctx context.Context, senders []sms.Sender) ([]sms.Registration, error) {
	want := map[string]bool{}
	for _, s := range senders {
		want[sms.E164Digits(s.Value)] = true
	}
	out := []sms.Registration{}
	var tf struct {
		Objects []struct {
			Number          string `json:"number"`
			Status          string `json:"status"`
			RejectionReason string `json:"rejection_reason"`
			UseCase         string `json:"usecase"`
			Volume          string `json:"volume"`
			Updated         string `json:"last_modified"`
		} `json:"objects"`
	}
	if _, err := p.call(ctx, "tollfree_verifications", http.MethodGet, "/TollfreeVerification/", nil, &tf); err != nil {
		return nil, err
	}
	for _, v := range tf.Objects {
		if !want[v.Number] {
			continue
		}
		st := regStatus(v.Status)
		if st == "approved" {
			st = "verified"
		}
		out = append(out, sms.Registration{Sender: plus(v.Number), Kind: "toll_free", Status: st, Reason: v.RejectionReason, Since: when(v.Updated),
			Details: map[string]string{"useCase": v.UseCase, "volume": v.Volume}})
	}
	var brands struct {
		Brands []struct {
			BrandID string `json:"brand_id"`
			Name    string `json:"brand_name"`
			Status  string `json:"registration_status"`
		} `json:"brands"`
	}
	if _, err := p.call(ctx, "brands", http.MethodGet, "/10dlc/Brand/", nil, &brands); err != nil {
		return nil, err
	}
	var camps struct {
		Campaigns []struct {
			CampaignID  string `json:"campaign_id"`
			BrandID     string `json:"brand_id"`
			Status      string `json:"registration_status"`
			Description string `json:"description"`
			UseCase     string `json:"usecase"`
		} `json:"campaigns"`
	}
	if _, err := p.call(ctx, "campaigns", http.MethodGet, "/10dlc/Campaign/", nil, &camps); err != nil {
		return nil, err
	}
	for _, c := range camps.Campaigns {
		var nums struct {
			PhoneNumbers []struct {
				Number string `json:"number"`
			} `json:"phone_numbers"`
		}
		if _, err := p.call(ctx, "campaign_numbers", http.MethodGet, "/10dlc/Campaign/"+url.PathEscape(c.CampaignID)+"/Number/", nil, &nums); err != nil {
			return nil, err
		}
		for _, n := range nums.PhoneNumbers {
			if !want[n.Number] {
				continue
			}
			for _, b := range brands.Brands {
				if b.BrandID == c.BrandID {
					out = append(out, sms.Registration{Sender: plus(n.Number), Kind: "brand", Name: b.Name, Status: regStatus(b.Status)})
				}
			}
			out = append(out, sms.Registration{Sender: plus(n.Number), Kind: "campaign", Name: c.Description, Status: regStatus(c.Status), Details: map[string]string{"useCase": c.UseCase}})
		}
	}
	return out, nil
}

func when(s string) *time.Time {
	for _, l := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(l, s); err == nil {
			return &t
		}
	}
	return nil
}

// Prices implements sms.PricingReader.
func (p *Provider) Prices(ctx context.Context, countries []string) (map[string]float64, error) {
	out := map[string]float64{}
	for _, c := range countries {
		var pr struct {
			Message struct {
				Outbound struct {
					Rate string `json:"rate"`
				} `json:"outbound"`
			} `json:"message"`
		}
		if _, err := p.call(ctx, "pricing", http.MethodGet, "/Pricing/?country_iso="+url.QueryEscape(strings.ToUpper(c)), nil, &pr); err != nil {
			return nil, err
		}
		if v, err := strconv.ParseFloat(pr.Message.Outbound.Rate, 64); err == nil && v > 0 {
			out[strings.ToUpper(c)] = v
		}
	}
	return out, nil
}
