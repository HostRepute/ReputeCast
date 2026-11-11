// Package meta reads Facebook and Instagram lead ads over Meta's Graph API
// (M66): the Pages a system user's token reaches and their own tokens, the
// app's and the Pages' leadgen webhook subscriptions, the Pages' lead forms
// with their questions and consent checkboxes, one lead by its id, and a
// form's leads since a moment (polling and importing past leads).
//
// It reuses the WhatsApp Graph client (internal/whatsapp/meta) for the
// transport, its errors and the webhook's verification handshake and
// X-Hub-Signature-256. Every call made with a system user's or a Page's
// token carries appsecret_proof, so an app that requires it keeps working
// and a token on its own is useless to whoever reads it. It knows nothing
// about workspaces: internal/leads keeps the connection and its sealed
// credentials. metatest is an in-process Graph API for tests and the demo.
package meta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	wa "github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// DefaultBaseURL and DefaultVersion are the WhatsApp client's: one Graph
// API version for every Meta call ReputeCast makes.
const (
	DefaultBaseURL = wa.DefaultBaseURL
	DefaultVersion = wa.DefaultVersion
)

// Permissions are what the system user's token needs, in the order the
// connect dialog lists them.
var Permissions = []string{"leads_retrieval", "pages_show_list", "pages_read_engagement", "pages_manage_metadata", "ads_management"}

// Credentials are what connecting asks for, all write-only but the app id.
type Credentials struct {
	AppID     string `json:"appId"`
	AppSecret string `json:"appSecret"`
	// Token is a system user's token for the app (no expiry).
	Token string `json:"token"`
}

// Client calls the Graph API for one connection.
type Client struct {
	Creds Credentials
	// HTTP is the SSRF-guarded client in production, the in-process Graph
	// API in tests and the demo.
	HTTP *http.Client
	// BaseURL and Version default to the Graph API and DefaultVersion.
	BaseURL string
	Version string
}

// Error is a Graph API error, or a transport failure.
type Error = wa.Error

// AsError returns the Graph API error inside err.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// TokenRejected reports whether Meta refused the token: it expired, was
// revoked or lost a permission.
func TokenRejected(err error) bool {
	e, ok := AsError(err)
	return ok && e.TokenRejected()
}

// Expired reports whether the token expired.
func Expired(err error) bool {
	e, ok := AsError(err)
	return ok && e.Expired()
}

// PermissionMissing reports whether the token lacks a permission (10, 200
// to 299) or lead access (Leads Access Manager answers with 100/33 or 200).
func PermissionMissing(err error) bool {
	e, ok := AsError(err)
	return ok && (e.Code == 10 || (e.Code >= 200 && e.Code <= 299))
}

// RateLimited reports whether Meta asks to slow down: the app, user, Page
// or lead ads rate limits.
func RateLimited(err error) bool {
	e, ok := AsError(err)
	if !ok {
		return false
	}
	switch e.Code {
	case 4, 17, 32, 613, 80001, 80005:
		return true
	}
	return e.RateLimited()
}

// Unavailable reports a failure worth retrying: no answer or a 5xx.
func Unavailable(err error) bool {
	e, ok := AsError(err)
	return ok && e.Unavailable()
}

// graph is the transport for one token.
func (c *Client) graph(token string) *wa.Client {
	return &wa.Client{Creds: wa.Credentials{AccessToken: token}, HTTP: c.HTTP, BaseURL: c.BaseURL, Version: c.Version}
}

// appToken is the app's own token, for the app's webhook subscription.
func (c *Client) appToken() string { return c.Creds.AppID + "|" + c.Creds.AppSecret }

// Proof is appsecret_proof for a token: the hex HMAC-SHA256 of the token
// with the app secret.
func Proof(appSecret, token string) string {
	m := hmac.New(sha256.New, []byte(appSecret))
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}

// call makes one request with a system user's or a Page's token.
func (c *Client) call(ctx context.Context, token, method, path string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	if c.Creds.AppSecret != "" && token != "" {
		q.Set("appsecret_proof", Proof(c.Creds.AppSecret, token))
	}
	return c.graph(token).Call(ctx, method, path, q, nil, out)
}

// Me checks the system user's token: its user's id and name.
func (c *Client) Me(ctx context.Context) (id, name string, err error) {
	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err = c.call(ctx, c.Creds.Token, http.MethodGet, "/me", url.Values{"fields": {"id,name"}}, &out)
	return out.ID, out.Name, err
}

// Page is a Facebook Page the token reaches, with its own token.
type Page struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	AccessToken string   `json:"access_token"`
	Tasks       []string `json:"tasks"`
}

type paging struct {
	Cursors struct {
		After string `json:"after"`
	} `json:"cursors"`
	Next string `json:"next"`
}

// App checks the app id and secret together: the app's name, read with the
// app's own token.
func (c *Client) App(ctx context.Context) (string, error) {
	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := c.graph(c.appToken()).Call(ctx, http.MethodGet, "/"+url.PathEscape(c.Creds.AppID), url.Values{"fields": {"id,name"}}, nil, &out)
	return out.Name, err
}

// maxPages bounds a listing, so a token with thousands of Pages can't keep
// a request busy.
const maxPages = 20

// Pages lists every Page the system user's token reaches, with its token.
func (c *Client) Pages(ctx context.Context) ([]Page, error) {
	out := []Page{}
	after := ""
	for range maxPages {
		q := url.Values{"fields": {"id,name,access_token,tasks"}, "limit": {"100"}}
		if after != "" {
			q.Set("after", after)
		}
		var page struct {
			Data   []Page `json:"data"`
			Paging paging `json:"paging"`
		}
		if err := c.call(ctx, c.Creds.Token, http.MethodGet, "/me/accounts", q, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Data...)
		if page.Paging.Next == "" || page.Paging.Cursors.After == "" {
			break
		}
		after = page.Paging.Cursors.After
	}
	return out, nil
}

type success struct {
	Success bool `json:"success"`
}

// SubscribeApp subscribes the app to Pages' leadgen webhooks at
// callbackURL. Meta checks the address at once with a verification
// request carrying verifyToken, so the connection must already be stored.
func (c *Client) SubscribeApp(ctx context.Context, callbackURL, verifyToken string) error {
	q := url.Values{"object": {"page"}, "callback_url": {callbackURL}, "fields": {"leadgen"}, "verify_token": {verifyToken}, "include_values": {"true"}}
	var out success
	if err := c.graph(c.appToken()).Call(ctx, http.MethodPost, "/"+url.PathEscape(c.Creds.AppID)+"/subscriptions", q, nil, &out); err != nil {
		return err
	}
	if !out.Success {
		return &Error{Message: "Meta didn't subscribe the app to Pages' leads"}
	}
	return nil
}

// Subscribe installs the app on a Page for its leadgen webhook.
func (c *Client) Subscribe(ctx context.Context, p Page) error {
	var out success
	if err := c.call(ctx, p.AccessToken, http.MethodPost, "/"+url.PathEscape(p.ID)+"/subscribed_apps", url.Values{"subscribed_fields": {"leadgen"}}, &out); err != nil {
		return err
	}
	if !out.Success {
		return &Error{Message: "Meta didn't subscribe the app to the Page's leads"}
	}
	return nil
}

// Unsubscribe removes the app from a Page.
func (c *Client) Unsubscribe(ctx context.Context, p Page) error {
	var out success
	return c.call(ctx, p.AccessToken, http.MethodDelete, "/"+url.PathEscape(p.ID)+"/subscribed_apps", nil, &out)
}

// Question is a lead form's question.
type Question struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Type is Meta's: EMAIL, FULL_NAME, FIRST_NAME, LAST_NAME, PHONE,
	// COMPANY_NAME, JOB_TITLE, CUSTOM…
	Type string `json:"type"`
}

// Checkbox is one of the custom disclaimer's checkboxes.
type Checkbox struct {
	Key      string `json:"key"`
	Text     string `json:"text"`
	Required bool   `json:"required"`
}

// Form is a lead form as Meta lists it.
type Form struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Locale string `json:"locale"`
	// Questions and the custom disclaimer: its text and checkboxes.
	Questions  []Question `json:"questions"`
	Disclaimer string     `json:"disclaimer"`
	Checkboxes []Checkbox `json:"checkboxes"`
}

// formFields are the fields a form is read with.
const formFields = "id,name,status,locale,questions{key,label,type},legal_content{custom_disclaimer{title,body{text},checkboxes{key,text,is_required}}}"

type rawForm struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	Locale       string     `json:"locale"`
	Questions    []Question `json:"questions"`
	LegalContent *struct {
		CustomDisclaimer *struct {
			Title string `json:"title"`
			Body  *struct {
				Text string `json:"text"`
			} `json:"body"`
			Checkboxes []struct {
				Key        string `json:"key"`
				Text       string `json:"text"`
				IsRequired *bool  `json:"is_required"`
			} `json:"checkboxes"`
		} `json:"custom_disclaimer"`
	} `json:"legal_content"`
}

func (r rawForm) form() Form {
	f := Form{ID: r.ID, Name: r.Name, Status: r.Status, Locale: r.Locale, Questions: r.Questions, Checkboxes: []Checkbox{}}
	if f.Questions == nil {
		f.Questions = []Question{}
	}
	if r.LegalContent != nil && r.LegalContent.CustomDisclaimer != nil {
		d := r.LegalContent.CustomDisclaimer
		if d.Body != nil {
			f.Disclaimer = strings.TrimSpace(d.Body.Text)
		}
		for _, cb := range d.Checkboxes {
			// Meta's default is required.
			req := cb.IsRequired == nil || *cb.IsRequired
			f.Checkboxes = append(f.Checkboxes, Checkbox{Key: cb.Key, Text: strings.TrimSpace(cb.Text), Required: req})
		}
	}
	return f
}

// Forms lists a Page's lead forms with their questions and checkboxes.
func (c *Client) Forms(ctx context.Context, p Page) ([]Form, error) {
	out := []Form{}
	after := ""
	for range maxPages {
		q := url.Values{"fields": {formFields}, "limit": {"100"}}
		if after != "" {
			q.Set("after", after)
		}
		var page struct {
			Data   []rawForm `json:"data"`
			Paging paging    `json:"paging"`
		}
		if err := c.call(ctx, p.AccessToken, http.MethodGet, "/"+url.PathEscape(p.ID)+"/leadgen_forms", q, &page); err != nil {
			return nil, err
		}
		for _, f := range page.Data {
			out = append(out, f.form())
		}
		if page.Paging.Next == "" || page.Paging.Cursors.After == "" {
			break
		}
		after = page.Paging.Cursors.After
	}
	return out, nil
}

// Form reads one lead form.
func (c *Client) Form(ctx context.Context, p Page, formID string) (Form, error) {
	var r rawForm
	err := c.call(ctx, p.AccessToken, http.MethodGet, "/"+url.PathEscape(formID), url.Values{"fields": {formFields}}, &r)
	return r.form(), err
}

// Answer is one field of a lead: the question's key and its values.
type Answer struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Lead is a person's answers to a lead form.
type Lead struct {
	ID          string
	FormID      string
	AdID        string
	CreatedTime time.Time
	Fields      []Answer
	// Checked holds each custom disclaimer checkbox's answer by key.
	Checked map[string]bool
}

// Value is the first answer to a question ("" when there's none).
func (l Lead) Value(key string) string {
	for _, a := range l.Fields {
		if a.Name == key && len(a.Values) > 0 {
			return strings.TrimSpace(a.Values[0])
		}
	}
	return ""
}

// leadFields are the fields a lead is read with.
const leadFields = "id,created_time,ad_id,form_id,field_data,custom_disclaimer_responses"

type rawLead struct {
	ID          string   `json:"id"`
	CreatedTime string   `json:"created_time"`
	AdID        string   `json:"ad_id"`
	FormID      string   `json:"form_id"`
	FieldData   []Answer `json:"field_data"`
	Responses   []struct {
		Key       string `json:"checkbox_key"`
		IsChecked any    `json:"is_checked"`
	} `json:"custom_disclaimer_responses"`
}

func (r rawLead) lead() Lead {
	l := Lead{ID: r.ID, FormID: r.FormID, AdID: r.AdID, Fields: r.FieldData, Checked: map[string]bool{}}
	if l.Fields == nil {
		l.Fields = []Answer{}
	}
	l.CreatedTime = graphTime(r.CreatedTime)
	for _, resp := range r.Responses {
		l.Checked[resp.Key] = truthy(resp.IsChecked)
	}
	return l
}

// truthy reads Meta's "1", "", true or 1.
func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	case float64:
		return t != 0
	}
	return false
}

// graphTime reads "2026-10-15T14:08:00+0000".
func graphTime(s string) time.Time {
	for _, layout := range []string{"2006-01-02T15:04:05-0700", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// GetLead reads one lead with the Page's token.
func (c *Client) GetLead(ctx context.Context, p Page, leadgenID string) (Lead, error) {
	var r rawLead
	if err := c.call(ctx, p.AccessToken, http.MethodGet, "/"+url.PathEscape(leadgenID), url.Values{"fields": {leadFields}}, &r); err != nil {
		return Lead{}, err
	}
	return r.lead(), nil
}

// LeadsPage is one page of a form's leads.
type LeadsPage struct {
	Leads []Lead
	// Next is the cursor of the next page, "" at the end.
	Next string
}

// FormLeads lists a form's leads created after since, a page at a time.
func (c *Client) FormLeads(ctx context.Context, p Page, formID string, since time.Time, after string) (LeadsPage, error) {
	filter, _ := json.Marshal([]map[string]any{{"field": "time_created", "operator": "GREATER_THAN", "value": since.Unix()}})
	q := url.Values{"fields": {leadFields}, "limit": {"100"}, "filtering": {string(filter)}}
	if after != "" {
		q.Set("after", after)
	}
	var page struct {
		Data   []rawLead `json:"data"`
		Paging paging    `json:"paging"`
	}
	if err := c.call(ctx, p.AccessToken, http.MethodGet, "/"+url.PathEscape(formID)+"/leads", q, &page); err != nil {
		return LeadsPage{}, err
	}
	out := LeadsPage{Leads: make([]Lead, 0, len(page.Data))}
	for _, r := range page.Data {
		out.Leads = append(out.Leads, r.lead())
	}
	if page.Paging.Next != "" {
		out.Next = page.Paging.Cursors.After
	}
	return out, nil
}

// LeadChange is one new lead a webhook delivery names: ids only, never
// answers.
type LeadChange struct {
	LeadgenID   string
	PageID      string
	FormID      string
	AdID        string
	CreatedTime time.Time
}

// flexID reads an id Meta sends as a string or a number.
type flexID string

func (f *flexID) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexID(n.String())
	return nil
}

// ErrNotLeads is a delivery that isn't a Page's webhook.
var ErrNotLeads = errors.New("meta: not a Page webhook")

// ParseWebhook reads the leads a delivery names; changes to other fields
// are skipped.
func ParseWebhook(body []byte) ([]LeadChange, error) {
	var w struct {
		Object string `json:"object"`
		Entry  []struct {
			ID      flexID `json:"id"`
			Changes []struct {
				Field string `json:"field"`
				Value struct {
					LeadgenID   flexID `json:"leadgen_id"`
					PageID      flexID `json:"page_id"`
					FormID      flexID `json:"form_id"`
					AdID        flexID `json:"ad_id"`
					CreatedTime flexID `json:"created_time"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}
	if w.Object != "page" {
		return nil, ErrNotLeads
	}
	out := []LeadChange{}
	for _, e := range w.Entry {
		for _, ch := range e.Changes {
			if ch.Field != "leadgen" || ch.Value.LeadgenID == "" {
				continue
			}
			page := string(ch.Value.PageID)
			if page == "" {
				page = string(e.ID)
			}
			var at time.Time
			if n, err := strconv.ParseInt(string(ch.Value.CreatedTime), 10, 64); err == nil {
				at = time.Unix(n, 0).UTC()
			}
			out = append(out, LeadChange{LeadgenID: string(ch.Value.LeadgenID), PageID: page, FormID: string(ch.Value.FormID),
				AdID: string(ch.Value.AdID), CreatedTime: at})
		}
	}
	return out, nil
}

// VerifyChallenge answers Meta's verification request (hub.challenge when
// hub.verify_token is ours, compared in constant time).
func VerifyChallenge(q url.Values, verifyToken string) (string, bool) {
	return wa.VerifyChallenge(q, verifyToken)
}

// VerifySignature checks X-Hub-Signature-256 over the raw body.
func VerifySignature(appSecret string, body []byte, header string) bool {
	return wa.VerifySignature(appSecret, body, header)
}

// SignatureShape reports whether a header can be a signature at all.
func SignatureShape(header string) bool { return wa.SignatureShape(header) }

// Sign is the X-Hub-Signature-256 of a body.
func Sign(appSecret string, body []byte) string { return wa.Sign(appSecret, body) }

// testEmails are the addresses Meta's Lead Ads Testing Tool fills in.
var testEmails = map[string]bool{"test@meta.com": true, "test@fb.com": true}

// IsTest reports whether a lead came from Meta's Lead Ads Testing Tool: its
// email is Meta's test address, or an answer is the tool's placeholder
// ("<test lead: dummy data for full_name>").
func (l Lead) IsTest() bool {
	for _, a := range l.Fields {
		for _, v := range a.Values {
			v = strings.TrimSpace(v)
			if testEmails[strings.ToLower(v)] || strings.HasPrefix(v, "<test lead:") {
				return true
			}
		}
	}
	return false
}
