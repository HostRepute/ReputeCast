// Package telnyx is the Telnyx SMS adapter (API v2): texts with an API key,
// webhooks on a messaging profile ReputeCast sets up, signed by Telnyx with
// Ed25519 (checked with the account's public key, and a timestamp so old
// requests can't be replayed), the account's numbers, its opt-out list
// (read only), and 10DLC and toll-free verification statuses.
//
// Telnyx has no per-country price list in its API; each text's cost comes
// back with it, so prices are entered by hand and corrected from the texts.
package telnyx

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
)

// Adapter is the registry key.
const Adapter = "telnyx"

const apiHost = "https://api.telnyx.com"

// Signature headers.
const (
	SignatureHeader = "Telnyx-Signature-Ed25519"
	TimestampHeader = "Telnyx-Timestamp"
)

func init() { sms.Register(Descriptor()) }

// Descriptor is the adapter's manifest.
func Descriptor() sms.Descriptor {
	return sms.Descriptor{
		Manifest: sms.Manifest{
			Adapter: Adapter, Name: "Telnyx",
			Summary: "Pay as you go · its own carrier network in the US",
			Badges: []sms.Badge{
				{Capability: sms.CapTollFree, Support: sms.Yes}, {Capability: sms.Cap10DLC, Support: sms.Yes}, {Capability: sms.CapSenderIDs, Support: sms.Yes},
				{Capability: sms.CapReceipts, Support: sms.Yes}, {Capability: sms.CapInbound, Support: sms.Yes},
				{Capability: sms.CapOptOutSync, Support: sms.Partial, Note: "read only"},
				{Capability: sms.CapRegistration, Support: sms.Yes},
				{Capability: sms.CapPricing, Support: sms.Partial, Note: "each text's cost"},
			},
			Fields: []sms.Field{
				{Key: "api_key", Label: "API key", Secret: true, Required: true},
				{Key: "public_key", Label: "Public key", Required: true, Help: "Portal › Keys & Credentials › Public key. ReputeCast checks every webhook's signature with it."},
				{Key: "messaging_profile_id", Label: "Messaging profile ID", Help: "Leave empty and ReputeCast creates a messaging profile for its webhooks."},
			},
			Webhooks: "registered",
			Console: func(kind, _ string) string {
				switch kind {
				case "registration":
					return "https://portal.telnyx.com/#/app/messaging/10dlc/campaigns"
				case "numbers":
					return "https://portal.telnyx.com/#/app/numbers/my-numbers"
				}
				return "https://portal.telnyx.com/"
			},
			Version: "1.0.0", Maintainer: "ReputeCast",
		},
		New: New,
	}
}

// Provider is a Telnyx account.
type Provider struct {
	cfg     sms.Config
	key     string
	pub     ed25519.PublicKey
	profile string
	http    *http.Client
}

// New opens a Telnyx account.
func New(cfg sms.Config) (sms.Provider, error) {
	p := &Provider{cfg: cfg, key: cfg.Value("api_key"), profile: strings.TrimSpace(cfg.Value("messaging_profile_id")), http: cfg.Client()}
	if k := strings.TrimSpace(cfg.Value("public_key")); k != "" {
		b, err := base64.StdEncoding.DecodeString(k)
		if err != nil || len(b) != ed25519.PublicKeySize {
			return nil, errors.New("telnyx: the public key is the base64 key from Keys & Credentials")
		}
		p.pub = b
	}
	return p, nil
}

var (
	_ sms.SenderLister       = (*Provider)(nil)
	_ sms.WebhookRegistrar   = (*Provider)(nil)
	_ sms.OptOutReader       = (*Provider)(nil)
	_ sms.RegistrationReader = (*Provider)(nil)
)

// Adapter implements sms.Provider.
func (p *Provider) Adapter() string { return Adapter }

func (p *Provider) url(path string) string {
	if p.cfg.BaseURL != "" {
		return strings.TrimRight(p.cfg.BaseURL, "/") + path
	}
	return apiHost + path
}

type apiErr struct {
	Errors []struct {
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (p *Provider) call(ctx context.Context, op, method, path string, js, out any) ([]byte, error) {
	raw, err := sms.Do(ctx, p.http, sms.Call{Op: op, Method: method, URL: p.url(path), JSON: js, Header: http.Header{"Authorization": {"Bearer " + p.key}}}, out)
	var ce *sms.CallError
	if errors.As(err, &ce) {
		var e apiErr
		if json.Unmarshal(raw, &e) == nil && len(e.Errors) > 0 {
			ce.Code, ce.Message = e.Errors[0].Code, e.Errors[0].Title
		}
	}
	return raw, err
}

// Check implements sms.Provider.
func (p *Provider) Check(ctx context.Context) (sms.Account, error) {
	var b struct {
		Data struct {
			Balance  string `json:"balance"`
			Currency string `json:"currency"`
		} `json:"data"`
	}
	if _, err := p.call(ctx, "check", http.MethodGet, "/v2/balance", nil, &b); err != nil {
		return sms.Account{}, err
	}
	return sms.Account{ID: sms.Hint(p.key, 3, 4), Name: "Telnyx account", Hint: sms.Hint(p.key, 3, 4), Live: true}, nil
}

// codeReason maps Telnyx's message error codes.
func codeReason(code string) (sms.Reason, bool) {
	switch code {
	case "40002":
		return sms.ReasonFiltered, true // blocked as spam, temporary
	case "40003":
		return sms.ReasonFiltered, false
	case "40300":
		return sms.ReasonOptedOut, false // blocked due to STOP
	case "40310", "40012":
		return sms.ReasonInvalid, false
	case "40008":
		return sms.ReasonNotInService, false
	case "40005", "40006", "40011":
		return sms.ReasonUnreachable, true
	case "40001", "40004", "40301":
		return sms.ReasonRejected, false
	}
	return sms.ReasonOther, false
}

// Send implements sms.Provider.
func (p *Provider) Send(ctx context.Context, m sms.Message) (sms.SendResult, error) {
	in := map[string]any{"from": m.From, "to": m.To, "text": m.Body, "type": "SMS"}
	if m.StatusURL != "" {
		in["webhook_url"] = m.StatusURL
		in["use_profile_webhooks"] = false
	}
	if p.profile != "" && !strings.HasPrefix(m.From, "+") {
		in["messaging_profile_id"] = p.profile // an alphanumeric sender sends from a profile
	}
	var out struct {
		Data struct {
			ID    string `json:"id"`
			Parts int    `json:"parts"`
			Cost  *struct {
				Amount string `json:"amount"`
			} `json:"cost"`
		} `json:"data"`
	}
	raw, err := p.call(ctx, "send", http.MethodPost, "/v2/messages", in, &out)
	if err != nil {
		var ce *sms.CallError
		if errors.As(err, &ce) && (ce.Status == 400 || ce.Status == 422 || ce.Status == 429) {
			var e apiErr
			_ = json.Unmarshal(raw, &e)
			se := &sms.SendError{Code: strconv.Itoa(ce.Status), Message: "rejected"}
			if len(e.Errors) > 0 {
				se.Code, se.Message = e.Errors[0].Code, e.Errors[0].Title
			}
			se.Reason, se.Temporary = codeReason(se.Code)
			if ce.Status == 429 {
				se.Temporary, se.Throttled = true, true
			}
			return sms.SendResult{}, se
		}
		return sms.SendResult{}, err
	}
	res := sms.SendResult{ProviderMessageID: out.Data.ID, Segments: out.Data.Parts}
	if out.Data.Cost != nil {
		if v, err := strconv.ParseFloat(out.Data.Cost.Amount, 64); err == nil {
			res.Price = &v
		}
	}
	return res, nil
}

// VerifyWebhook implements sms.Provider: the Ed25519 signature of
// "timestamp|body", within 5 minutes.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, body []byte) error {
	if len(p.pub) == 0 {
		return sms.ErrBadSignature
	}
	sig, err := base64.StdEncoding.DecodeString(r.Header.Get(SignatureHeader))
	ts := r.Header.Get(TimestampHeader)
	if err != nil || len(sig) == 0 || ts == "" {
		return sms.ErrBadSignature
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return sms.ErrBadSignature
	}
	if d := p.cfg.Clock().Unix() - n; d > 300 || d < -300 {
		return sms.ErrBadSignature
	}
	if !ed25519.Verify(p.pub, []byte(ts+"|"+string(body)), sig) {
		return sms.ErrBadSignature
	}
	return nil
}

type hook struct {
	Data struct {
		EventType  string `json:"event_type"`
		OccurredAt string `json:"occurred_at"`
		Payload    struct {
			ID   string `json:"id"`
			Text string `json:"text"`
			From struct {
				PhoneNumber string `json:"phone_number"`
			} `json:"from"`
			To []struct {
				PhoneNumber string `json:"phone_number"`
				Status      string `json:"status"`
			} `json:"to"`
			Errors []struct {
				Code  string `json:"code"`
				Title string `json:"title"`
			} `json:"errors"`
			Parts int `json:"parts"`
			Cost  *struct {
				Amount string `json:"amount"`
			} `json:"cost"`
		} `json:"payload"`
	} `json:"data"`
}

// ParseWebhook implements sms.Provider: message.sent, message.finalized and
// message.received.
func (p *Provider) ParseWebhook(_ context.Context, _ string, body []byte) ([]sms.Event, error) {
	var h hook
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, err
	}
	at := p.cfg.Clock()
	if t, err := time.Parse(time.RFC3339, h.Data.OccurredAt); err == nil {
		at = t
	}
	pl := h.Data.Payload
	switch h.Data.EventType {
	case "message.received":
		to := ""
		if len(pl.To) > 0 {
			to = pl.To[0].PhoneNumber
		}
		return []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: pl.ID, From: pl.From.PhoneNumber, To: to, Body: pl.Text, OccurredAt: at,
			Summary: "message.received · text=" + sms.Quote(pl.Text)}}, nil
	case "message.sent", "message.finalized":
		if len(pl.To) == 0 {
			return nil, nil
		}
		st := pl.To[0].Status
		e := sms.Event{ProviderMessageID: pl.ID, To: pl.To[0].PhoneNumber, From: pl.From.PhoneNumber, OccurredAt: at, Segments: pl.Parts,
			Summary: h.Data.EventType + " · status=" + st}
		switch st {
		case "queued":
			e.Kind = sms.EventQueued
		case "sending", "sent", "delivery_unconfirmed":
			e.Kind = sms.EventSent
		case "delivered":
			e.Kind = sms.EventDelivered
		case "delivery_failed":
			e.Kind = sms.EventUndelivered
		case "sending_failed":
			e.Kind = sms.EventFailed
		default:
			return nil, nil
		}
		if len(pl.Errors) > 0 {
			e.Code = pl.Errors[0].Code
			e.Reason, _ = codeReason(e.Code)
			e.Summary += " · " + e.Code
		}
		if pl.Cost != nil {
			if v, err := strconv.ParseFloat(pl.Cost.Amount, 64); err == nil {
				e.Price = &v
			}
		}
		return []sms.Event{e}, nil
	}
	return nil, nil
}

type number struct {
	ID                 string `json:"id"`
	PhoneNumber        string `json:"phone_number"`
	PhoneNumberType    string `json:"phone_number_type"`
	MessagingProfileID string `json:"messaging_profile_id"`
}

func (p *Provider) numbers(ctx context.Context) ([]number, error) {
	var out struct {
		Data []number `json:"data"`
	}
	_, err := p.call(ctx, "numbers", http.MethodGet, "/v2/phone_numbers?page[size]=100", nil, &out)
	return out.Data, err
}

// Senders implements sms.SenderLister.
func (p *Provider) Senders(ctx context.Context) ([]sms.Sender, error) {
	ns, err := p.numbers(ctx)
	if err != nil {
		return nil, err
	}
	out := []sms.Sender{}
	for _, n := range ns {
		pn, err := phone.Parse(n.PhoneNumber, "")
		if err != nil {
			continue
		}
		s := sms.Sender{Value: n.PhoneNumber, ProviderRef: n.ID, Countries: []string{pn.Country}, Throughput: 1}
		switch {
		case n.PhoneNumberType == "toll_free":
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

// RegisterWebhooks implements sms.WebhookRegistrar: the messaging profile's
// webhook (a new profile when none was given) and each number on it.
func (p *Provider) RegisterWebhooks(ctx context.Context, hooks sms.Webhooks, senders []sms.Sender) (sms.WebhookSetup, error) {
	profile := p.profile
	if profile == "" {
		var out struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if _, err := p.call(ctx, "profile", http.MethodPost, "/v2/messaging_profiles", map[string]any{"name": "ReputeCast", "webhook_url": hooks.InboundURL, "webhook_api_version": "2"}, &out); err != nil {
			return sms.WebhookSetup{}, err
		}
		profile = out.Data.ID
	} else if _, err := p.call(ctx, "profile", http.MethodPatch, "/v2/messaging_profiles/"+url.PathEscape(profile), map[string]any{"webhook_url": hooks.InboundURL, "webhook_api_version": "2"}, nil); err != nil {
		return sms.WebhookSetup{}, err
	}
	refs := map[string]string{}
	if ns, err := p.numbers(ctx); err == nil {
		for _, n := range ns {
			refs[n.PhoneNumber] = n.ID
		}
	}
	for _, s := range senders {
		ref := s.ProviderRef
		if ref == "" {
			ref = refs[s.Value]
		}
		if ref == "" {
			continue
		}
		if _, err := p.call(ctx, "number_profile", http.MethodPatch, "/v2/phone_numbers/"+url.PathEscape(ref)+"/messaging", map[string]any{"messaging_profile_id": profile}, nil); err != nil {
			return sms.WebhookSetup{}, err
		}
	}
	return sms.WebhookSetup{}, nil
}

// OptOuts implements sms.OptOutReader: Telnyx's opt-out list.
func (p *Provider) OptOuts(ctx context.Context) ([]sms.OptOut, error) {
	var out struct {
		Data []struct {
			From      string `json:"from"`
			To        string `json:"to"`
			Keyword   string `json:"keyword"`
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	if _, err := p.call(ctx, "opt_outs", http.MethodGet, "/v2/messaging_optouts?page[size]=250", nil, &out); err != nil {
		return nil, err
	}
	list := []sms.OptOut{}
	for _, o := range out.Data {
		at, _ := time.Parse(time.RFC3339, o.CreatedAt)
		// "from" is the person who opted out of texts from "to".
		list = append(list, sms.OptOut{Phone: o.From, Keyword: strings.ToUpper(o.Keyword), At: at})
	}
	return list, nil
}

func tcrStatus(s string) string {
	switch strings.ToUpper(s) {
	case "TCR_ACCEPTED", "MNO_PROVISIONED", "ACTIVE", "VERIFIED", "VETTED_VERIFIED":
		return "approved"
	case "TCR_FAILED", "MNO_REJECTED", "REJECTED", "UNVERIFIED":
		return "rejected"
	case "TCR_PENDING", "MNO_PENDING", "PENDING", "SELF_DECLARED":
		return "pending"
	}
	return "not_registered"
}

// Registrations implements sms.RegistrationReader.
func (p *Provider) Registrations(ctx context.Context, senders []sms.Sender) ([]sms.Registration, error) {
	want := map[string]bool{}
	for _, s := range senders {
		want[s.Value] = true
	}
	out := []sms.Registration{}
	var tf struct {
		Records []struct {
			Status       string `json:"verificationStatus"`
			Reason       string `json:"reason"`
			UseCase      string `json:"useCase"`
			Volume       string `json:"messageVolume"`
			Updated      string `json:"updatedAt"`
			PhoneNumbers []struct {
				PhoneNumber string `json:"phoneNumber"`
			} `json:"phoneNumbers"`
		} `json:"records"`
	}
	if _, err := p.call(ctx, "tollfree_verifications", http.MethodGet, "/v2/messaging_tollfree/verification/requests?page=1&page_size=50", nil, &tf); err != nil {
		return nil, err
	}
	for _, r := range tf.Records {
		st := "pending"
		switch strings.ToLower(r.Status) {
		case "verified":
			st = "verified"
		case "rejected":
			st = "rejected"
		}
		var since *time.Time
		if t, err := time.Parse(time.RFC3339, r.Updated); err == nil {
			since = &t
		}
		for _, n := range r.PhoneNumbers {
			if want[n.PhoneNumber] {
				out = append(out, sms.Registration{Sender: n.PhoneNumber, Kind: "toll_free", Status: st, Reason: r.Reason, Since: since,
					Details: map[string]string{"useCase": r.UseCase, "volume": r.Volume}})
			}
		}
	}
	var assign struct {
		Records []struct {
			PhoneNumber string `json:"phoneNumber"`
			CampaignID  string `json:"campaignId"`
		} `json:"records"`
	}
	if _, err := p.call(ctx, "number_campaigns", http.MethodGet, "/10dlc/phone_number_campaigns?recordsPerPage=100", nil, &assign); err != nil {
		return nil, err
	}
	for _, a := range assign.Records {
		if !want[a.PhoneNumber] {
			continue
		}
		var c struct {
			BrandID     string `json:"brandId"`
			Description string `json:"description"`
			Status      string `json:"campaignStatus"`
			UseCase     string `json:"usecase"`
		}
		if _, err := p.call(ctx, "campaign", http.MethodGet, "/10dlc/campaign/"+url.PathEscape(a.CampaignID), nil, &c); err != nil {
			return nil, err
		}
		var b struct {
			DisplayName    string `json:"displayName"`
			IdentityStatus string `json:"identityStatus"`
		}
		if _, err := p.call(ctx, "brand", http.MethodGet, "/10dlc/brand/"+url.PathEscape(c.BrandID), nil, &b); err != nil {
			return nil, err
		}
		out = append(out,
			sms.Registration{Sender: a.PhoneNumber, Kind: "brand", Name: b.DisplayName, Status: tcrStatus(b.IdentityStatus)},
			sms.Registration{Sender: a.PhoneNumber, Kind: "campaign", Name: c.Description, Status: tcrStatus(c.Status), Details: map[string]string{"useCase": c.UseCase}})
	}
	return out, nil
}
