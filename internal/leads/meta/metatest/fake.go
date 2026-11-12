// Package metatest is an in-process Graph API for lead ads, with Meta's
// payloads: the Pages a system user's token reaches and their tokens, the
// app's and the Pages' webhook subscriptions, lead forms, leads and the
// webhook bodies Meta posts. Tests and the demo use it, so nothing reaches
// Meta. Like Meta, subscribing the app calls the callback URL's
// verification request first when Callback is set.
package metatest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/leads/meta"
)

// Fake is the Graph API for one Meta app and its system user.
type Fake struct {
	mu        sync.Mutex
	appID     string
	appSecret string
	token     string
	expired   bool
	limited   bool
	pages     []*page
	leads     map[string]*lead
	calls     []string
	next      int
	// Callback answers the verification request Meta makes when the app
	// subscribes (the app's router in tests); nil accepts any address.
	Callback http.Handler
	// AppSubscribed, CallbackURL and VerifyToken are the app's webhook
	// subscription once made.
	AppSubscribed bool
	CallbackURL   string
	VerifyToken   string
	// Permissive accepts any token and secret (the demo's stand-in holds
	// credentials it never issued).
	Permissive bool
}

type page struct {
	meta.Page
	forms      []meta.Form
	subscribed bool
}

type lead struct {
	pageID string
	meta.Lead
}

// New makes a Graph API for an app and a system user's token.
func New(appID, appSecret, token string) *Fake {
	return &Fake{appID: appID, appSecret: appSecret, token: token, leads: map[string]*lead{}}
}

// AddPage adds a Page the token reaches, with its token.
func (f *Fake) AddPage(id, name, token string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pages = append(f.pages, &page{Page: meta.Page{ID: id, Name: name, AccessToken: token, Tasks: []string{"ADVERTISE", "MANAGE"}}})
}

// AddForm adds a lead form to a Page.
func (f *Fake) AddForm(pageID string, form meta.Form) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p := f.page(pageID); p != nil {
		p.forms = append(p.forms, form)
	}
}

// SetCheckboxes replaces a form's consent checkboxes.
func (f *Fake) SetCheckboxes(formID string, cbs []meta.Checkbox) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.pages {
		for i := range p.forms {
			if p.forms[i].ID == formID {
				p.forms[i].Checkboxes = cbs
			}
		}
	}
}

// AddLead adds a lead to a form and returns its id. Answers are question
// key → value; checked are the consent checkboxes ticked.
func (f *Fake) AddLead(pageID, formID string, at time.Time, answers map[string]string, checked ...string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	id := strconv.Itoa(910000000000 + f.next)
	keys := make([]string, 0, len(answers))
	for k := range answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fields := make([]meta.Answer, 0, len(keys))
	for _, k := range keys {
		fields = append(fields, meta.Answer{Name: k, Values: []string{answers[k]}})
	}
	l := &lead{pageID: pageID, Lead: meta.Lead{ID: id, FormID: formID, AdID: "238500000001", CreatedTime: at.UTC().Truncate(time.Second), Fields: fields, Checked: map[string]bool{}}}
	for _, form := range f.formsOf(pageID) {
		if form.ID != formID {
			continue
		}
		for _, cb := range form.Checkboxes {
			l.Checked[cb.Key] = false
		}
	}
	for _, k := range checked {
		l.Checked[k] = true
	}
	f.leads[id] = l
	return id
}

// ExpireToken makes calls with the system user's and Pages' tokens answer
// as Meta does for an expired token (190, subcode 463).
func (f *Fake) ExpireToken() {
	f.mu.Lock()
	f.expired = true
	f.mu.Unlock()
}

// Renew accepts a new system user token.
func (f *Fake) Renew(token string) {
	f.mu.Lock()
	f.token, f.expired = token, false
	f.mu.Unlock()
}

// RateLimit makes every call answer with the lead ads rate limit (80005).
func (f *Fake) RateLimit(on bool) {
	f.mu.Lock()
	f.limited = on
	f.mu.Unlock()
}

// Subscribed reports whether the app is installed on a Page.
func (f *Fake) Subscribed(pageID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.page(pageID)
	return p != nil && p.subscribed
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

func (f *Fake) page(id string) *page {
	for _, p := range f.pages {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (f *Fake) formsOf(pageID string) []meta.Form {
	if p := f.page(pageID); p != nil {
		return p.forms
	}
	return nil
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

// who is the token's owner: the system user, a Page (its id) or the app.
func (f *Fake) who(token string) (kind, pageID string) {
	if f.Permissive {
		if strings.Contains(token, "|") {
			return "app", ""
		}
		for _, p := range f.pages {
			if p.AccessToken == token {
				return "page", p.ID
			}
		}
		return "user", ""
	}
	switch token {
	case f.appID + "|" + f.appSecret:
		return "app", ""
	case f.token:
		return "user", ""
	}
	for _, p := range f.pages {
		if p.AccessToken == token {
			return "page", p.ID
		}
	}
	return "", ""
}

// ServeHTTP answers the calls the client makes.
func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	expired, limited := f.expired, f.limited
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	kind, pageID := f.who(token)
	secret := f.appSecret
	f.mu.Unlock()
	q := r.URL.Query()
	switch {
	case kind == "":
		graphError(w, http.StatusUnauthorized, 190, 467, "OAuthException", "Invalid OAuth access token - Cannot parse access token")
		return
	case expired && kind != "app":
		graphError(w, http.StatusUnauthorized, 190, 463, "OAuthException", "Error validating access token: Session has expired.")
		return
	case limited:
		graphError(w, http.StatusBadRequest, 80005, 0, "OAuthException", "There have been too many leadgen api calls to this Page account. Wait a bit and try again.")
		return
	case kind != "app" && !f.Permissive && q.Get("appsecret_proof") != meta.Proof(secret, token):
		graphError(w, http.StatusBadRequest, 100, 0, "GraphMethodException", "Invalid appsecret_proof provided in the API argument")
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
	switch {
	case r.Method == http.MethodGet && id == "me" && edge == "" && kind == "user":
		reply(w, map[string]string{"id": "61550000000001", "name": "ReputeCast leads"})
	case r.Method == http.MethodGet && id == "me" && edge == "accounts" && kind == "user":
		f.mu.Lock()
		data := make([]meta.Page, 0, len(f.pages))
		for _, p := range f.pages {
			data = append(data, p.Page)
		}
		f.mu.Unlock()
		reply(w, map[string]any{"data": data, "paging": map[string]any{"cursors": map[string]string{"before": "QVFI", "after": "QVFJ"}}})
	case r.Method == http.MethodPost && edge == "subscriptions" && kind == "app":
		f.subscribeApp(w, r, id)
	case edge == "subscribed_apps" && kind == "page" && pageID == id:
		f.mu.Lock()
		if p := f.page(id); p != nil {
			p.subscribed = r.Method == http.MethodPost && q.Get("subscribed_fields") == "leadgen"
		}
		f.mu.Unlock()
		reply(w, map[string]bool{"success": true})
	case r.Method == http.MethodGet && edge == "leadgen_forms" && kind == "page" && pageID == id:
		f.mu.Lock()
		data := []map[string]any{}
		for _, form := range f.formsOf(id) {
			data = append(data, rawForm(form))
		}
		f.mu.Unlock()
		reply(w, map[string]any{"data": data, "paging": map[string]any{"cursors": map[string]string{"before": "QVFI", "after": "QVFJ"}}})
	case r.Method == http.MethodGet && edge == "leads" && kind == "page":
		f.formLeads(w, q, pageID, id)
	case r.Method == http.MethodGet && edge == "" && kind == "app" && (id == f.appID || f.Permissive):
		reply(w, map[string]string{"id": id, "name": "Acme lead ads"})
	case r.Method == http.MethodGet && edge == "" && kind == "page":
		f.node(w, pageID, id)
	default:
		graphError(w, http.StatusBadRequest, 100, 33, "GraphMethodException",
			fmt.Sprintf("Unsupported %s request. Object with ID '%s' does not exist, cannot be loaded due to missing permissions, or does not support this operation", r.Method, id))
	}
}

// subscribeApp subscribes the app, after Meta's verification request.
func (f *Fake) subscribeApp(w http.ResponseWriter, r *http.Request, appID string) {
	q := r.URL.Query()
	if appID != f.appID && !f.Permissive {
		graphError(w, http.StatusBadRequest, 100, 33, "GraphMethodException", "Unsupported post request.")
		return
	}
	if q.Get("object") != "page" || q.Get("fields") != "leadgen" {
		graphError(w, http.StatusBadRequest, 100, 0, "OAuthException", "(#100) Invalid parameter")
		return
	}
	cb, verify := q.Get("callback_url"), q.Get("verify_token")
	if f.Callback != nil {
		u, err := url.Parse(cb)
		if err != nil {
			graphError(w, http.StatusBadRequest, 2200, 0, "OAuthException", "(#2200) callback verification failed: invalid URL")
			return
		}
		vq := url.Values{"hub.mode": {"subscribe"}, "hub.challenge": {"1158201444"}, "hub.verify_token": {verify}}
		// Meta calls from outside: none of the caller's request values.
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, u.Path+"?"+vq.Encode(), nil)
		rec := httptest.NewRecorder()
		f.Callback.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "1158201444" {
			graphError(w, http.StatusBadRequest, 2200, 0, "OAuthException", fmt.Sprintf("(#2200) callback verification failed: Received HTTP code %d", rec.Code))
			return
		}
	}
	f.mu.Lock()
	f.AppSubscribed, f.CallbackURL, f.VerifyToken = true, cb, verify
	f.mu.Unlock()
	reply(w, map[string]bool{"success": true})
}

// node answers GET /{id}: a lead or a form of the token's Page.
func (f *Fake) node(w http.ResponseWriter, pageID, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if l, ok := f.leads[id]; ok && l.pageID == pageID {
		reply(w, rawLead(l.Lead))
		return
	}
	for _, form := range f.formsOf(pageID) {
		if form.ID == id {
			reply(w, rawForm(form))
			return
		}
	}
	graphError(w, http.StatusBadRequest, 100, 33, "GraphMethodException", fmt.Sprintf("Unsupported get request. Object with ID '%s' does not exist", id))
}

// formLeads answers GET /{form}/leads with the filtering on time_created.
func (f *Fake) formLeads(w http.ResponseWriter, q url.Values, pageID, formID string) {
	var since int64
	var filter []struct {
		Field    string `json:"field"`
		Operator string `json:"operator"`
		Value    int64  `json:"value"`
	}
	if raw := q.Get("filtering"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &filter)
	}
	for _, ft := range filter {
		if ft.Field == "time_created" && ft.Operator == "GREATER_THAN" {
			since = ft.Value
		}
	}
	f.mu.Lock()
	ls := []*lead{}
	for _, l := range f.leads {
		if l.pageID == pageID && l.FormID == formID && l.CreatedTime.Unix() > since {
			ls = append(ls, l)
		}
	}
	f.mu.Unlock()
	sort.Slice(ls, func(i, j int) bool { return ls[i].CreatedTime.After(ls[j].CreatedTime) })
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 25
	}
	start, _ := strconv.Atoi(q.Get("after"))
	if start > len(ls) {
		start = len(ls)
	}
	end := min(start+limit, len(ls))
	data := make([]map[string]any, 0, end-start)
	for _, l := range ls[start:end] {
		data = append(data, rawLead(l.Lead))
	}
	pg := map[string]any{"cursors": map[string]string{"before": strconv.Itoa(start), "after": strconv.Itoa(end)}}
	if end < len(ls) {
		pg["next"] = "https://graph.facebook.com/next"
	}
	reply(w, map[string]any{"data": data, "paging": pg})
}

// rawForm is a form as the Graph API returns it.
func rawForm(form meta.Form) map[string]any {
	cbs := make([]map[string]any, 0, len(form.Checkboxes))
	for _, c := range form.Checkboxes {
		cbs = append(cbs, map[string]any{"key": c.Key, "text": c.Text, "is_required": c.Required})
	}
	out := map[string]any{"id": form.ID, "name": form.Name, "status": form.Status, "locale": form.Locale, "questions": form.Questions}
	if form.Disclaimer != "" || len(cbs) > 0 {
		out["legal_content"] = map[string]any{"id": form.ID + "1", "custom_disclaimer": map[string]any{"title": "Consent", "body": map[string]string{"text": form.Disclaimer}, "checkboxes": cbs}}
	}
	return out
}

// rawLead is a lead as the Graph API returns it.
func rawLead(l meta.Lead) map[string]any {
	resp := []map[string]string{}
	keys := make([]string, 0, len(l.Checked))
	for k := range l.Checked {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := ""
		if l.Checked[k] {
			v = "1"
		}
		resp = append(resp, map[string]string{"checkbox_key": k, "is_checked": v})
	}
	return map[string]any{"id": l.ID, "created_time": l.CreatedTime.UTC().Format("2006-01-02T15:04:05-0700"), "ad_id": l.AdID, "form_id": l.FormID,
		"field_data": l.Fields, "custom_disclaimer_responses": resp}
}

// LeadWebhook is the body Meta posts for a new lead, and its signature.
func LeadWebhook(appSecret, pageID, formID, leadgenID string, at time.Time) ([]byte, string) {
	body, _ := json.Marshal(map[string]any{
		"object": "page",
		"entry": []map[string]any{{"id": pageID, "time": at.Unix(), "changes": []map[string]any{{
			"field": "leadgen",
			"value": map[string]any{"leadgen_id": leadgenID, "page_id": pageID, "form_id": formID, "adgroup_id": "238500000002", "ad_id": "238500000001", "created_time": at.Unix()},
		}}}},
	})
	return body, meta.Sign(appSecret, body)
}
