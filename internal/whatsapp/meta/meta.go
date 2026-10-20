// Package meta talks to Meta's WhatsApp Cloud API over plain HTTP (design
// expansion phase 8, X22): the phone number and its quality rating and
// messaging limit, the business account, message templates, sending
// template and text messages, the webhook's verification handshake and its
// X-Hub-Signature-256, and the webhook payloads.
//
// It's the only WhatsApp provider in this phase (brief §3.9). It knows
// nothing about workspaces: internal/whatsapp keeps the account, its sealed
// credentials, the templates and the message log. metatest is an
// in-process Graph API with recorded payloads, for tests and the demo.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// DefaultBaseURL is the Graph API.
const DefaultBaseURL = "https://graph.facebook.com"

// DefaultVersion is the Graph API version the client calls.
const DefaultVersion = "v23.0"

// Credentials are what connecting asks for, all write-only.
type Credentials struct {
	// BusinessAccountID is the WhatsApp Business Account (WABA) id.
	BusinessAccountID string `json:"businessAccountId"`
	// PhoneNumberID is the Cloud API's id of the sending number.
	PhoneNumberID string `json:"phoneNumberId"`
	// AccessToken is a system user's permanent token.
	AccessToken string `json:"accessToken"`
	// AppSecret signs the webhook's requests (X-Hub-Signature-256).
	AppSecret string `json:"appSecret"`
}

// Client calls the Graph API for one account.
type Client struct {
	Creds Credentials
	// HTTP is the SSRF-guarded client in production, the in-process Graph
	// API in tests and the demo.
	HTTP *http.Client
	// BaseURL and Version default to the Graph API and DefaultVersion.
	BaseURL string
	Version string
}

func (c *Client) base() string {
	b := c.BaseURL
	if b == "" {
		b = DefaultBaseURL
	}
	v := c.Version
	if v == "" {
		v = DefaultVersion
	}
	return strings.TrimRight(b, "/") + "/" + v
}

// Error is a Graph API error (an "error" object), or a transport failure.
type Error struct {
	// Status is the HTTP status, 0 when nothing answered.
	Status int
	// Code and Subcode are Meta's: 190 for an access token that expired or
	// was revoked (subcode 463 expired), 131047 for a free-form message
	// outside the customer service window, 132001 for a template that
	// doesn't exist…
	Code    int
	Subcode int
	Type    string
	Message string
	// Details is error_data.details, Meta's longer explanation.
	Details string
	TraceID string
}

func (e *Error) Error() string {
	if e.Code == 0 {
		return "whatsapp: " + e.Message
	}
	return fmt.Sprintf("whatsapp: Meta error %d: %s", e.Code, e.Message)
}

// TokenRejected reports whether Meta refused the access token: it expired,
// was revoked, or lacks a permission. Sends are held until it's replaced.
func (e *Error) TokenRejected() bool { return e.Code == 190 || e.Code == 102 || e.Status == http.StatusUnauthorized }

// Expired reports whether the token expired (code 190, subcode 463).
func (e *Error) Expired() bool { return e.Code == 190 && e.Subcode == 463 }

// RateLimited reports whether Meta asks to slow down.
func (e *Error) RateLimited() bool {
	switch e.Code {
	case 4, 80007, 130429, 131056:
		return true
	}
	return e.Status == http.StatusTooManyRequests
}

// Unavailable reports a failure worth retrying: no answer or a 5xx.
func (e *Error) Unavailable() bool { return e.Status == 0 || e.Status >= 500 }

// ErrUnreachable wraps a transport failure.
var ErrUnreachable = errors.New("whatsapp: Meta's Graph API didn't answer")

type graphError struct {
	Error *struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		FbtraceID    string `json:"fbtrace_id"`
		ErrorData    *struct {
			Details string `json:"details"`
		} `json:"error_data"`
	} `json:"error"`
}

// call makes one Graph API request and decodes the answer into out.
func (c *Client) call(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u := c.base() + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Creds.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return &Error{Message: "Meta's Graph API didn't answer: " + clip(err.Error(), 160)}
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return &Error{Status: resp.StatusCode, Message: "the answer was cut off"}
	}
	if resp.StatusCode >= 300 {
		var ge graphError
		if json.Unmarshal(b, &ge) == nil && ge.Error != nil {
			e := &Error{Status: resp.StatusCode, Code: ge.Error.Code, Subcode: ge.Error.ErrorSubcode, Type: ge.Error.Type,
				Message: ge.Error.Message, TraceID: ge.Error.FbtraceID}
			if ge.Error.ErrorData != nil {
				e.Details = ge.Error.ErrorData.Details
			}
			return e
		}
		return &Error{Status: resp.StatusCode, Message: fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return &Error{Status: resp.StatusCode, Message: "the answer wasn't JSON"}
	}
	return nil
}

// PhoneNumber is the sending number as Meta reports it.
type PhoneNumber struct {
	ID string `json:"id"`
	// DisplayPhoneNumber is "+1 312-555-0110".
	DisplayPhoneNumber string `json:"display_phone_number"`
	// VerifiedName is the display name ("Acme").
	VerifiedName string `json:"verified_name"`
	// NameStatus is the display name's review: APPROVED, PENDING_REVIEW,
	// DECLINED, AVAILABLE_WITHOUT_REVIEW, NONE.
	NameStatus string `json:"name_status"`
	// QualityRating is GREEN (High), YELLOW (Medium), RED (Low) or UNKNOWN.
	QualityRating string `json:"quality_rating"`
	// MessagingLimitTier is TIER_50, TIER_250, TIER_1K, TIER_10K,
	// TIER_100K or TIER_UNLIMITED: unique people a business can start
	// conversations with in a rolling 24 hours.
	MessagingLimitTier string `json:"messaging_limit_tier"`
	// Status is CONNECTED, PENDING, FLAGGED, RESTRICTED, DISCONNECTED…
	Status string `json:"status"`
}

// Phone reads the sending number.
func (c *Client) Phone(ctx context.Context) (PhoneNumber, error) {
	var p PhoneNumber
	q := url.Values{"fields": {"id,display_phone_number,verified_name,name_status,quality_rating,messaging_limit_tier,status"}}
	err := c.call(ctx, http.MethodGet, "/"+url.PathEscape(c.Creds.PhoneNumberID), q, nil, &p)
	return p, err
}

// BusinessAccount is the WhatsApp Business Account.
type BusinessAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

// Account reads the business account.
func (c *Client) Account(ctx context.Context) (BusinessAccount, error) {
	var a BusinessAccount
	err := c.call(ctx, http.MethodGet, "/"+url.PathEscape(c.Creds.BusinessAccountID), url.Values{"fields": {"id,name,currency"}}, nil, &a)
	return a, err
}

// SubscribeApp subscribes the app to the business account's webhooks, so
// Meta sends message statuses, replies and template updates.
func (c *Client) SubscribeApp(ctx context.Context) error {
	var out struct {
		Success bool `json:"success"`
	}
	if err := c.call(ctx, http.MethodPost, "/"+url.PathEscape(c.Creds.BusinessAccountID)+"/subscribed_apps", nil, nil, &out); err != nil {
		return err
	}
	if !out.Success {
		return &Error{Message: "Meta didn't subscribe the app to the account's webhooks"}
	}
	return nil
}

// LimitOf is the number of people a messaging limit tier allows a day; 0
// for TIER_UNLIMITED or an unknown tier.
func LimitOf(tier string) int {
	switch strings.ToUpper(tier) {
	case "TIER_50":
		return 50
	case "TIER_250":
		return 250
	case "TIER_1K":
		return 1000
	case "TIER_2K":
		return 2000
	case "TIER_10K":
		return 10000
	case "TIER_100K":
		return 100000
	}
	return 0
}

// Quality is a quality rating in the app's words: high, medium, low or
// unknown.
func Quality(rating string) string {
	switch strings.ToUpper(rating) {
	case "GREEN", "HIGH":
		return "high"
	case "YELLOW", "MEDIUM":
		return "medium"
	case "RED", "LOW":
		return "low"
	}
	return "unknown"
}

// Template is a message template as Meta lists it.
type Template struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Language string `json:"language"`
	// RejectedReason is Meta's code (INVALID_FORMAT, ABUSIVE_CONTENT,
	// INCORRECT_CATEGORY, TAG_CONTENT_MISMATCH, SCAM, NONE).
	RejectedReason string              `json:"rejected_reason"`
	Components     []TemplateComponent `json:"components"`
}

// TemplateComponent is a template's header, body, footer or buttons.
type TemplateComponent struct {
	Type    string           `json:"type"`
	Format  string           `json:"format,omitempty"`
	Text    string           `json:"text,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

// TemplateButton is a quick reply, URL or phone number button.
type TemplateButton struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// Body is the template's body text with its {{n}} variables.
func (t Template) Body() string {
	for _, c := range t.Components {
		if strings.EqualFold(c.Type, "BODY") {
			return c.Text
		}
	}
	return ""
}

// Buttons are the template's buttons.
func (t Template) Buttons() []TemplateButton {
	for _, c := range t.Components {
		if strings.EqualFold(c.Type, "BUTTONS") {
			return c.Buttons
		}
	}
	return nil
}

// Templates lists every message template of the business account, page by
// page.
func (c *Client) Templates(ctx context.Context) ([]Template, error) {
	out := []Template{}
	q := url.Values{"fields": {"id,name,status,category,language,rejected_reason,components"}, "limit": {"100"}}
	for page := 0; page < 50; page++ {
		var resp struct {
			Data   []Template `json:"data"`
			Paging struct {
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := c.call(ctx, http.MethodGet, "/"+url.PathEscape(c.Creds.BusinessAccountID)+"/message_templates", q, nil, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Data...)
		if resp.Paging.Next == "" || resp.Paging.Cursors.After == "" {
			return out, nil
		}
		q.Set("after", resp.Paging.Cursors.After)
	}
	return out, nil
}

// Message is one message to send: a template with its variables, or free
// text (inside the customer service window only).
type Message struct {
	// To is the recipient's number in E.164 ("+13125550123").
	To string
	// Template, Language and Params send a template; Params fill {{1}},
	// {{2}}… of its body in order.
	Template string
	Language string
	Params   []string
	// Text sends a free-form message.
	Text string
}

// Sent is Meta's answer to a send: the message id (wamid…) that its status
// webhooks carry.
type Sent struct {
	ID string
	// WAID is the recipient's WhatsApp id as Meta resolved it.
	WAID string
}

// Send sends one message.
func (c *Client) Send(ctx context.Context, m Message) (Sent, error) {
	to := strings.TrimPrefix(strings.TrimSpace(m.To), "+")
	body := map[string]any{"messaging_product": "whatsapp", "recipient_type": "individual", "to": to}
	switch {
	case m.Template != "":
		lang := m.Language
		if lang == "" {
			lang = "en_US"
		}
		tpl := map[string]any{"name": m.Template, "language": map[string]string{"code": lang}}
		if len(m.Params) > 0 {
			params := make([]map[string]string, 0, len(m.Params))
			for _, p := range m.Params {
				params = append(params, map[string]string{"type": "text", "text": p})
			}
			tpl["components"] = []map[string]any{{"type": "body", "parameters": params}}
		}
		body["type"] = "template"
		body["template"] = tpl
	case m.Text != "":
		body["type"] = "text"
		body["text"] = map[string]any{"preview_url": strings.Contains(m.Text, "://"), "body": m.Text}
	default:
		return Sent{}, errors.New("whatsapp: a message needs a template or text")
	}
	var resp struct {
		Contacts []struct {
			WAID string `json:"wa_id"`
		} `json:"contacts"`
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := c.call(ctx, http.MethodPost, "/"+url.PathEscape(c.Creds.PhoneNumberID)+"/messages", nil, body, &resp); err != nil {
		return Sent{}, err
	}
	if len(resp.Messages) == 0 || resp.Messages[0].ID == "" {
		return Sent{}, &Error{Status: 200, Message: "Meta answered without a message id"}
	}
	out := Sent{ID: resp.Messages[0].ID}
	if len(resp.Contacts) > 0 {
		out.WAID = resp.Contacts[0].WAID
	}
	return out, nil
}

// Errors Meta names when a message fails (in the send answer or a
// "failed" status), in the report's words.
var errorReasons = map[int]string{
	131026: "undeliverable",     // not a WhatsApp user, or an old app version
	131047: "window_closed",     // free-form outside the customer service window
	131049: "marketing_limited", // Meta limits marketing messages per person
	131050: "opted_out",         // the person stopped marketing messages
	131051: "unsupported",
	131052: "media_failed",
	131053: "media_failed",
	131021: "same_number",
	131056: "rate_limited",
	130429: "rate_limited",
	131048: "spam_limited",
	132000: "template_params",
	132001: "template_missing",
	132005: "template_text_too_long",
	132007: "template_policy",
	132012: "template_params",
	132015: "template_paused",
	132016: "template_disabled",
	133010: "number_not_registered",
	190:    "token_rejected",
	368:    "account_restricted",
	131031: "account_locked",
}

// ReasonOf is the report's reason for a Meta error code ("other" when it
// isn't one ReputeCast knows).
func ReasonOf(code int) string {
	if r, ok := errorReasons[code]; ok {
		return r
	}
	return "other"
}

// Permanent reports whether a failure won't go away by retrying the same
// message.
func Permanent(code int) bool {
	switch code {
	case 130429, 131056, 4, 80007, 131000, 131016:
		return false
	}
	return code != 0
}

// ParsePhoneDisplay normalizes "+1 312-555-0110" to "+13125550110".
func ParsePhoneDisplay(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "+" + b.String()
}

// Unix reads Meta's string timestamps ("1697450520").
func Unix(s string) time.Time {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
