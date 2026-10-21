package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Meta's webhook (brief §3.9): a GET handshake when the address is saved in
// Meta's app settings, then POSTs signed with the app secret.

// VerifyChallenge answers Meta's verification request: when hub.mode is
// "subscribe" and hub.verify_token is ours, the answer is hub.challenge.
func VerifyChallenge(q url.Values, verifyToken string) (string, bool) {
	if q.Get("hub.mode") != "subscribe" || verifyToken == "" {
		return "", false
	}
	if !hmac.Equal([]byte(q.Get("hub.verify_token")), []byte(verifyToken)) {
		return "", false
	}
	c := q.Get("hub.challenge")
	if c == "" || len(c) > 200 {
		return "", false
	}
	return c, true
}

// Sign is the X-Hub-Signature-256 value of a body: "sha256=" and the hex
// HMAC-SHA256 with the app secret.
func Sign(appSecret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(appSecret))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

// VerifySignature checks X-Hub-Signature-256 over the raw body, in constant
// time.
func VerifySignature(appSecret string, body []byte, header string) bool {
	if appSecret == "" {
		return false
	}
	got, ok := strings.CutPrefix(strings.TrimSpace(header), "sha256=")
	if !ok {
		return false
	}
	want := Sign(appSecret, body)[len("sha256="):]
	return hmac.Equal([]byte(strings.ToLower(got)), []byte(want))
}

// Event kinds of a webhook.
const (
	EventStatus   = "status"   // a message we sent: sent, delivered, read, failed
	EventMessage  = "message"  // a message from a person
	EventTemplate = "template" // a template's status changed
	EventQuality  = "quality"  // the number's quality or messaging limit changed
	EventAccount  = "account"  // other account updates
)

// EventError is one of Meta's errors on a failed message.
type EventError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// Event is one normalized item of a webhook.
type Event struct {
	Kind          string
	BusinessID    string
	PhoneNumberID string
	At            time.Time

	// Status of a message we sent.
	MessageID string
	Status    string // sent, delivered, read, failed
	Recipient string // E.164
	Errors    []EventError
	Billable  bool
	Category  string // utility, marketing, authentication, service

	// A person's message.
	From          string // E.164
	Name          string // their WhatsApp profile name
	Type          string // text, button, interactive, image…
	Text          string
	ButtonText    string
	ButtonPayload string
	// ContextID is the message a reply quotes or a button belongs to.
	ContextID string

	// A template's status.
	TemplateID       string
	TemplateName     string
	TemplateLanguage string
	TemplateEvent    string // APPROVED, REJECTED, PENDING, PAUSED, DISABLED, FLAGGED, REINSTATED
	Reason           string // Meta's code
	ReasonText       string // Meta's explanation, when it gives one

	// The number's quality or limit.
	QualityEvent string // DOWNGRADE, UPGRADE, FLAGGED, UNFLAGGED
	CurrentLimit string
	OldLimit     string
}

// ErrWebhook is a body that isn't a WhatsApp webhook.
var ErrWebhook = errors.New("whatsapp: not a WhatsApp Business Account webhook")

type webhook struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string          `json:"field"`
			Value json.RawMessage `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type messagesValue struct {
	Metadata struct {
		PhoneNumberID string `json:"phone_number_id"`
	} `json:"metadata"`
	Contacts []struct {
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
		WAID string `json:"wa_id"`
	} `json:"contacts"`
	Statuses []struct {
		ID          string `json:"id"`
		Status      string `json:"status"`
		Timestamp   string `json:"timestamp"`
		RecipientID string `json:"recipient_id"`
		Errors      []struct {
			Code      int    `json:"code"`
			Title     string `json:"title"`
			Message   string `json:"message"`
			ErrorData struct {
				Details string `json:"details"`
			} `json:"error_data"`
		} `json:"errors"`
		Pricing *struct {
			Billable bool   `json:"billable"`
			Category string `json:"category"`
		} `json:"pricing"`
	} `json:"statuses"`
	Messages []struct {
		From      string `json:"from"`
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"`
		Text      *struct {
			Body string `json:"body"`
		} `json:"text"`
		Button *struct {
			Text    string `json:"text"`
			Payload string `json:"payload"`
		} `json:"button"`
		Interactive *struct {
			ButtonReply *struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"button_reply"`
			ListReply *struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"list_reply"`
		} `json:"interactive"`
		Context *struct {
			ID string `json:"id"`
		} `json:"context"`
	} `json:"messages"`
}

type templateValue struct {
	Event      string          `json:"event"`
	TemplateID json.RawMessage `json:"message_template_id"`
	Name       string          `json:"message_template_name"`
	Language   string          `json:"message_template_language"`
	Reason     string          `json:"reason"`
	OtherInfo  *struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"other_info"`
}

type qualityValue struct {
	Event        string `json:"event"`
	CurrentLimit string `json:"current_limit"`
	OldLimit     string `json:"old_limit"`
}

// ParseWebhook reads a webhook body into events, in their order.
func ParseWebhook(body []byte) ([]Event, error) {
	var w webhook
	if err := json.Unmarshal(body, &w); err != nil || w.Object != "whatsapp_business_account" {
		return nil, ErrWebhook
	}
	out := []Event{}
	for _, e := range w.Entry {
		for _, ch := range e.Changes {
			switch ch.Field {
			case "messages":
				var v messagesValue
				if err := json.Unmarshal(ch.Value, &v); err != nil {
					return nil, ErrWebhook
				}
				names := map[string]string{}
				for _, c := range v.Contacts {
					names[c.WAID] = c.Profile.Name
				}
				for _, s := range v.Statuses {
					ev := Event{Kind: EventStatus, BusinessID: e.ID, PhoneNumberID: v.Metadata.PhoneNumberID, MessageID: s.ID, Status: s.Status,
						Recipient: plus(s.RecipientID), At: Unix(s.Timestamp)}
					for _, er := range s.Errors {
						ev.Errors = append(ev.Errors, EventError{Code: er.Code, Title: er.Title, Message: er.Message, Details: er.ErrorData.Details})
					}
					if s.Pricing != nil {
						ev.Billable, ev.Category = s.Pricing.Billable, s.Pricing.Category
					}
					out = append(out, ev)
				}
				for _, m := range v.Messages {
					ev := Event{Kind: EventMessage, BusinessID: e.ID, PhoneNumberID: v.Metadata.PhoneNumberID, MessageID: m.ID, From: plus(m.From),
						Name: names[m.From], Type: m.Type, At: Unix(m.Timestamp)}
					if m.Text != nil {
						ev.Text = m.Text.Body
					}
					if m.Button != nil {
						ev.ButtonText, ev.ButtonPayload = m.Button.Text, m.Button.Payload
					}
					if m.Interactive != nil {
						switch {
						case m.Interactive.ButtonReply != nil:
							ev.ButtonText, ev.ButtonPayload = m.Interactive.ButtonReply.Title, m.Interactive.ButtonReply.ID
						case m.Interactive.ListReply != nil:
							ev.ButtonText, ev.ButtonPayload = m.Interactive.ListReply.Title, m.Interactive.ListReply.ID
						}
					}
					if m.Context != nil {
						ev.ContextID = m.Context.ID
					}
					out = append(out, ev)
				}
			case "message_template_status_update":
				var v templateValue
				if err := json.Unmarshal(ch.Value, &v); err != nil {
					return nil, ErrWebhook
				}
				ev := Event{Kind: EventTemplate, BusinessID: e.ID, TemplateID: strings.Trim(string(v.TemplateID), `"`), TemplateName: v.Name,
					TemplateLanguage: v.Language, TemplateEvent: strings.ToUpper(v.Event), Reason: v.Reason}
				if v.OtherInfo != nil {
					ev.ReasonText = strings.TrimSpace(v.OtherInfo.Description)
				}
				out = append(out, ev)
			case "phone_number_quality_update":
				var v qualityValue
				if err := json.Unmarshal(ch.Value, &v); err != nil {
					return nil, ErrWebhook
				}
				out = append(out, Event{Kind: EventQuality, BusinessID: e.ID, QualityEvent: strings.ToUpper(v.Event), CurrentLimit: v.CurrentLimit, OldLimit: v.OldLimit})
			default:
				out = append(out, Event{Kind: EventAccount, BusinessID: e.ID, Type: ch.Field})
			}
		}
	}
	return out, nil
}

// plus writes a WhatsApp id as E.164.
func plus(waid string) string {
	waid = strings.TrimSpace(waid)
	if waid == "" || strings.HasPrefix(waid, "+") {
		return waid
	}
	if _, err := strconv.ParseUint(waid, 10, 64); err != nil {
		return waid
	}
	return "+" + waid
}
