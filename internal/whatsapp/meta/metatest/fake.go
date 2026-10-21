// Package metatest is an in-process Graph API for WhatsApp, with recorded
// payloads: the phone number, the business account, message templates,
// sends and the webhook bodies Meta would post back. Tests and the demo use
// it, so nothing reaches Meta.
package metatest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// Sent is one message the fake accepted.
type Sent struct {
	At       time.Time
	ID       string
	To       string
	Template string
	Language string
	Params   []string
	Text     string
	// Body is the request as the client sent it.
	Body []byte
}

// Fake is the Graph API for one WhatsApp Business Account.
type Fake struct {
	mu        sync.Mutex
	phone     meta.PhoneNumber
	account   meta.BusinessAccount
	templates []meta.Template
	token     string
	expired   bool
	failures  map[string]int
	sent      []Sent
	calls     []string
	next      int
	// Subscribed is true once the app subscribed to the account's webhooks.
	Subscribed bool
}

// New makes a Graph API for an account with the given ids and token.
func New(businessID, phoneID, token string) *Fake {
	return &Fake{
		phone:    meta.PhoneNumber{ID: phoneID, DisplayPhoneNumber: "+1 312-555-0110", VerifiedName: "Acme", NameStatus: "APPROVED", QualityRating: "GREEN", MessagingLimitTier: "TIER_1K", Status: "CONNECTED"},
		account:  meta.BusinessAccount{ID: businessID, Name: "Acme Inc", Currency: "USD"},
		token:    token,
		failures: map[string]int{},
	}
}

// SetPhone replaces the number's details (quality rating, limit, name).
func (f *Fake) SetPhone(p meta.PhoneNumber) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p.ID = f.phone.ID
	f.phone = p
}

// SetTemplates replaces the account's templates.
func (f *Fake) SetTemplates(ts []meta.Template) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.templates = append([]meta.Template(nil), ts...)
}

// ExpireToken makes every call answer as Meta does for an expired token
// (190, subcode 463); a new token passed to Renew works again.
func (f *Fake) ExpireToken() {
	f.mu.Lock()
	f.expired = true
	f.mu.Unlock()
}

// Renew accepts a new token.
func (f *Fake) Renew(token string) {
	f.mu.Lock()
	f.token, f.expired = token, false
	f.mu.Unlock()
}

// Fail makes sends to a number fail with Meta's error code (131026 for a
// number that isn't on WhatsApp, 131047 for free text outside the window).
func (f *Fake) Fail(number string, code int) {
	f.mu.Lock()
	f.failures[strings.TrimPrefix(number, "+")] = code
	f.mu.Unlock()
}

// SentMessages returns what was sent, oldest first.
func (f *Fake) SentMessages() []Sent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Sent(nil), f.sent...)
}

// Calls lists "METHOD path" of every request, oldest first.
func (f *Fake) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// Client sends every request to the fake, whatever its host.
func (f *Fake) Client() *http.Client {
	return &http.Client{Transport: rt{f}, Timeout: 10 * time.Second}
}

type rt struct{ f *Fake }

func (t rt) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.f.ServeHTTP(rec, r)
	resp := rec.Result()
	resp.Request = r
	return resp, nil
}

func graphError(w http.ResponseWriter, status, code, sub int, typ, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": msg, "type": typ, "code": code, "error_subcode": sub, "fbtrace_id": "AbCdEfGh123"}})
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// ServeHTTP answers the calls the client makes.
func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	token, expired := f.token, f.expired
	f.mu.Unlock()
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	switch {
	case expired && got == token:
		graphError(w, http.StatusUnauthorized, 190, 463, "OAuthException", "Error validating access token: Session has expired.")
		return
	case got != token:
		graphError(w, http.StatusUnauthorized, 190, 467, "OAuthException", "Invalid OAuth access token - Cannot parse access token")
		return
	}
	// /v23.0/{id}[/{edge}]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		graphError(w, http.StatusBadRequest, 100, 0, "GraphMethodException", "Unsupported request")
		return
	}
	id, edge := parts[1], ""
	if len(parts) > 2 {
		edge = parts[2]
	}
	f.mu.Lock()
	phone, account := f.phone, f.account
	f.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && id == phone.ID && edge == "":
		reply(w, phone)
	case r.Method == http.MethodGet && id == account.ID && edge == "":
		reply(w, account)
	case r.Method == http.MethodGet && id == account.ID && edge == "message_templates":
		f.listTemplates(w, r)
	case r.Method == http.MethodPost && id == account.ID && edge == "subscribed_apps":
		f.mu.Lock()
		f.Subscribed = true
		f.mu.Unlock()
		reply(w, map[string]bool{"success": true})
	case r.Method == http.MethodPost && id == phone.ID && edge == "messages":
		f.send(w, r)
	default:
		graphError(w, http.StatusBadRequest, 100, 33, "GraphMethodException",
			fmt.Sprintf("Unsupported %s request. Object with ID '%s' does not exist", r.Method, id))
	}
}

func (f *Fake) listTemplates(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 25
	}
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))
	f.mu.Lock()
	all := append([]meta.Template(nil), f.templates...)
	f.mu.Unlock()
	if after > len(all) {
		after = len(all)
	}
	end := min(after+limit, len(all))
	page := map[string]any{"data": all[after:end]}
	if end < len(all) {
		page["paging"] = map[string]any{"cursors": map[string]string{"after": strconv.Itoa(end)}, "next": "https://graph.facebook.com/next"}
	}
	reply(w, page)
}

func (f *Fake) send(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	var req struct {
		Product  string `json:"messaging_product"`
		To       string `json:"to"`
		Type     string `json:"type"`
		Template *struct {
			Name     string `json:"name"`
			Language struct {
				Code string `json:"code"`
			} `json:"language"`
			Components []struct {
				Type       string `json:"type"`
				Parameters []struct {
					Text string `json:"text"`
				} `json:"parameters"`
			} `json:"components"`
		} `json:"template"`
		Text *struct {
			Body string `json:"body"`
		} `json:"text"`
	}
	if json.Unmarshal(body, &req) != nil || req.Product != "whatsapp" || req.To == "" {
		graphError(w, http.StatusBadRequest, 100, 0, "OAuthException", "(#100) Invalid parameter")
		return
	}
	f.mu.Lock()
	code := f.failures[req.To]
	f.mu.Unlock()
	switch code {
	case 0:
	case 131047:
		graphError(w, http.StatusBadRequest, 131047, 2494010, "OAuthException",
			"Re-engagement message: more than 24 hours have passed since the recipient last replied to the sender number.")
		return
	default:
		graphError(w, http.StatusBadRequest, code, 0, "OAuthException", fmt.Sprintf("(#%d) %s", code, meta.ReasonOf(code)))
		return
	}
	s := Sent{At: time.Now(), To: "+" + req.To, Body: body}
	switch req.Type {
	case "template":
		if req.Template == nil {
			graphError(w, http.StatusBadRequest, 100, 0, "OAuthException", "(#100) The parameter template is required.")
			return
		}
		if !f.knows(req.Template.Name, req.Template.Language.Code) {
			graphError(w, http.StatusNotFound, 132001, 2494073, "OAuthException", "(#132001) Template name does not exist in the translation")
			return
		}
		s.Template, s.Language = req.Template.Name, req.Template.Language.Code
		for _, c := range req.Template.Components {
			for _, p := range c.Parameters {
				s.Params = append(s.Params, p.Text)
			}
		}
	case "text":
		if req.Text == nil || req.Text.Body == "" {
			graphError(w, http.StatusBadRequest, 100, 0, "OAuthException", "(#100) The parameter text['body'] is required.")
			return
		}
		s.Text = req.Text.Body
	default:
		graphError(w, http.StatusBadRequest, 131051, 0, "OAuthException", "(#131051) Unsupported message type")
		return
	}
	f.mu.Lock()
	f.next++
	// Ids are unique across fakes and restarts, as Meta's are.
	s.ID = fmt.Sprintf("wamid.HBgL%s%06d", strconv.FormatInt(time.Now().UnixNano(), 36), f.next)
	f.sent = append(f.sent, s)
	f.mu.Unlock()
	reply(w, map[string]any{"messaging_product": "whatsapp", "contacts": []map[string]string{{"input": req.To, "wa_id": req.To}},
		"messages": []map[string]string{{"id": s.ID, "message_status": "accepted"}}})
}

// knows reports whether an approved template exists in that language.
func (f *Fake) knows(name, lang string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.templates {
		if t.Name == name && (lang == "" || t.Language == lang) {
			return strings.EqualFold(t.Status, "APPROVED")
		}
	}
	return false
}

// webhookBody wraps a change in a webhook body.
func webhookBody(businessID, field string, value any) []byte {
	b, _ := json.Marshal(map[string]any{"object": "whatsapp_business_account", "entry": []any{map[string]any{"id": businessID,
		"changes": []any{map[string]any{"field": field, "value": value}}}}})
	return b
}

// StatusWebhook is the body Meta posts when a message is sent, delivered,
// read or failed (errCode for failed), and its signature.
func StatusWebhook(appSecret, businessID, phoneID, messageID, recipient, status string, at time.Time, errCode int) ([]byte, string) {
	st := map[string]any{"id": messageID, "status": status, "timestamp": strconv.FormatInt(at.Unix(), 10), "recipient_id": strings.TrimPrefix(recipient, "+")}
	if status == "sent" || status == "delivered" {
		st["pricing"] = map[string]any{"billable": true, "pricing_model": "PMP", "category": "utility", "type": "regular"}
	}
	if errCode != 0 {
		st["errors"] = []any{map[string]any{"code": errCode, "title": meta.ReasonOf(errCode), "message": meta.ReasonOf(errCode),
			"error_data": map[string]string{"details": "Message failed to send because of an unknown error."}}}
	}
	b := webhookBody(businessID, "messages", map[string]any{"messaging_product": "whatsapp",
		"metadata": map[string]string{"display_phone_number": "13125550110", "phone_number_id": phoneID}, "statuses": []any{st}})
	return b, meta.Sign(appSecret, b)
}

// MessageWebhook is the body Meta posts when a person writes.
func MessageWebhook(appSecret, businessID, phoneID, from, name, text string, at time.Time) ([]byte, string) {
	waid := strings.TrimPrefix(from, "+")
	b := webhookBody(businessID, "messages", map[string]any{"messaging_product": "whatsapp",
		"metadata": map[string]string{"display_phone_number": "13125550110", "phone_number_id": phoneID},
		"contacts": []any{map[string]any{"profile": map[string]string{"name": name}, "wa_id": waid}},
		"messages": []any{map[string]any{"from": waid, "id": fmt.Sprintf("wamid.in.%d", at.UnixNano()), "timestamp": strconv.FormatInt(at.Unix(), 10),
			"type": "text", "text": map[string]string{"body": text}}}})
	return b, meta.Sign(appSecret, b)
}

// ButtonWebhook is the body Meta posts when a person taps a template's
// quick reply ("Stop promotions").
func ButtonWebhook(appSecret, businessID, phoneID, from, contextID, text string, at time.Time) ([]byte, string) {
	waid := strings.TrimPrefix(from, "+")
	b := webhookBody(businessID, "messages", map[string]any{"messaging_product": "whatsapp",
		"metadata": map[string]string{"display_phone_number": "13125550110", "phone_number_id": phoneID},
		"messages": []any{map[string]any{"from": waid, "id": fmt.Sprintf("wamid.btn.%d", at.UnixNano()), "timestamp": strconv.FormatInt(at.Unix(), 10),
			"type": "button", "button": map[string]string{"text": text, "payload": text}, "context": map[string]string{"id": contextID}}}})
	return b, meta.Sign(appSecret, b)
}

// TemplateWebhook is the body Meta posts when a template's status changes.
func TemplateWebhook(appSecret, businessID, templateID, name, language, event, reason, description string) ([]byte, string) {
	v := map[string]any{"event": event, "message_template_id": templateID, "message_template_name": name, "message_template_language": language, "reason": reason}
	if description != "" {
		v["other_info"] = map[string]string{"title": "Template rejected", "description": description}
	}
	b := webhookBody(businessID, "message_template_status_update", v)
	return b, meta.Sign(appSecret, b)
}

// QualityWebhook is the body Meta posts when the number's quality or limit
// changes.
func QualityWebhook(appSecret, businessID, event, current, old string) ([]byte, string) {
	b := webhookBody(businessID, "phone_number_quality_update", map[string]any{"display_phone_number": "13125550110", "event": event,
		"current_limit": current, "old_limit": old})
	return b, meta.Sign(appSecret, b)
}
