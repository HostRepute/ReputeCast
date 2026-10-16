package importer

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// recordedFS holds answers recorded from each provider's API (the Acme
// sample accounts). The tests and the demo replay them through
// RecordedTransport, so neither ever reaches the network.
//
//go:embed recorded
var recordedFS embed.FS

// RecordedKeys are the keys the recorded accounts accept.
var RecordedKeys = map[string]string{
	"postmark": "pm-0f3c-recorded-8a21",
	"sendgrid": "SG.recorded",
	"mailgun":  "key-recorded",
	"notifuse": "nf-recorded",
}

// RecordedNotifuse is the sample Notifuse instance.
const RecordedNotifuse = "https://notifuse.acme.io"

// RecordedTransport answers requests to Postmark, SendGrid, Mailgun and
// the sample Notifuse instance from the recorded answers, checking keys
// like the providers do.
func RecordedTransport() http.RoundTripper { return recordedTransport{} }

// SampleTransport answers the sample accounts (their recorded keys, the
// sample Notifuse instance) from the recordings and sends every other
// request to next. The development server uses it, so the demo's sample
// accounts import without a network.
func SampleTransport(next http.RoundTripper) http.RoundTripper { return sampleTransport{next: next} }

type sampleTransport struct{ next http.RoundTripper }

func (t sampleTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if IsSample(r) {
		return recordedTransport{}.RoundTrip(r)
	}
	return t.next.RoundTrip(r)
}

// IsSample reports whether a request is for one of the sample accounts.
func IsSample(r *http.Request) bool {
	if r.URL.Host == strings.TrimPrefix(RecordedNotifuse, "https://") {
		return true
	}
	if r.Header.Get("X-Postmark-Server-Token") == RecordedKeys["postmark"] || r.Header.Get("Authorization") == "Bearer "+RecordedKeys["sendgrid"] {
		return true
	}
	_, pass, ok := r.BasicAuth()
	return ok && pass == RecordedKeys["mailgun"]
}

type recordedTransport struct{}

func (recordedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	switch r.URL.Host {
	case "api.postmarkapp.com":
		return postmarkAnswer(r)
	case "api.sendgrid.com":
		return sendgridAnswer(r)
	case "api.mailgun.net", "api.eu.mailgun.net":
		return mailgunAnswer(r)
	case strings.TrimPrefix(RecordedNotifuse, "https://"):
		return notifuseAnswer(r)
	}
	return nil, fmt.Errorf("dial tcp: lookup %s: no such host", r.URL.Host)
}

func answer(r *http.Request, status int, body []byte) *http.Response {
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(bytes.NewReader(body)), Request: r, ContentLength: int64(len(body))}
}

func recorded(name string) ([]byte, bool) {
	b, err := fs.ReadFile(recordedFS, "recorded/"+name)
	return b, err == nil
}

func fileOr404(r *http.Request, name string) (*http.Response, error) {
	if b, ok := recorded(name); ok {
		return answer(r, 200, b), nil
	}
	return answer(r, 404, []byte(`{"message":"Not found"}`)), nil
}

func postmarkAnswer(r *http.Request) (*http.Response, error) {
	if r.Header.Get("X-Postmark-Server-Token") != RecordedKeys["postmark"] {
		return answer(r, 401, []byte(`{"ErrorCode":10,"Message":"The Server Token you provided in the X-Postmark-Server-Token request header was invalid."}`)), nil
	}
	p := r.URL.Path
	switch {
	case p == "/server":
		return fileOr404(r, "postmark/server.json")
	case p == "/templates":
		return fileOr404(r, "postmark/templates.json")
	case strings.HasPrefix(p, "/templates/"):
		alias := path.Base(p)
		meta, ok := recorded("postmark/templates/" + alias + ".json")
		if !ok {
			return answer(r, 422, []byte(`{"ErrorCode":1101,"Message":"The template was not found."}`)), nil
		}
		var t map[string]any
		if err := json.Unmarshal(meta, &t); err != nil {
			return nil, err
		}
		html, _ := recorded("postmark/templates/" + alias + ".html")
		text, _ := recorded("postmark/templates/" + alias + ".txt")
		t["HtmlBody"], t["TextBody"] = string(html), string(text)
		b, _ := json.Marshal(t)
		return answer(r, 200, b), nil
	}
	return answer(r, 404, []byte(`{"ErrorCode":404,"Message":"Not found."}`)), nil
}

func sendgridAnswer(r *http.Request) (*http.Response, error) {
	if r.Header.Get("Authorization") != "Bearer "+RecordedKeys["sendgrid"] {
		return answer(r, 401, []byte(`{"errors":[{"field":null,"message":"authorization required"}]}`)), nil
	}
	p := r.URL.Path
	switch {
	case p == "/v3/templates":
		return fileOr404(r, "sendgrid/templates.json")
	case p == "/v3/user/profile":
		return fileOr404(r, "sendgrid/profile.json")
	case strings.HasPrefix(p, "/v3/templates/"):
		return fileOr404(r, "sendgrid/"+path.Base(p)+".json")
	}
	return answer(r, 404, []byte(`{"errors":[{"message":"not found"}]}`)), nil
}

func mailgunAnswer(r *http.Request) (*http.Response, error) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "api" || pass != RecordedKeys["mailgun"] {
		return answer(r, 401, []byte(`Forbidden`)), nil
	}
	p := r.URL.Path
	switch {
	case p == "/v4/domains":
		return fileOr404(r, "mailgun/domains.json")
	case p == "/v3/mg.acme.io/templates":
		if r.URL.Query().Get("page") == "next" {
			return answer(r, 200, []byte(`{"items":[],"paging":{}}`)), nil
		}
		return fileOr404(r, "mailgun/templates.json")
	case strings.HasPrefix(p, "/v3/mg.acme.io/templates/"):
		return fileOr404(r, "mailgun/"+path.Base(p)+".json")
	case strings.HasSuffix(p, "/templates"):
		return answer(r, 404, []byte(`{"message":"domain not found"}`)), nil
	}
	return answer(r, 404, []byte(`{"message":"not found"}`)), nil
}

func notifuseAnswer(r *http.Request) (*http.Response, error) {
	if r.Header.Get("Authorization") != "Bearer "+RecordedKeys["notifuse"] {
		return answer(r, 401, []byte(`{"error":"Unauthorized"}`)), nil
	}
	p := r.URL.Path
	if p != "/api/workspaces.list" && r.URL.Query().Get("workspace_id") != "acmecloud" {
		return answer(r, 400, []byte(`{"error":"workspace_id is required"}`)), nil
	}
	q := r.URL.Query()
	switch p {
	case "/api/workspaces.list":
		return fileOr404(r, "notifuse/workspaces.json")
	case "/api/templates.list":
		return fileOr404(r, "notifuse/templates.json")
	case "/api/templates.get":
		return fileOr404(r, "notifuse/"+path.Base(q.Get("id"))+".json")
	case "/api/lists.stats":
		return fileOr404(r, "notifuse/lists.stats-"+path.Base(q.Get("list_id"))+".json")
	case "/api/contacts.list":
		name := "notifuse/contacts.list-" + path.Base(q.Get("list_id"))
		if c := q.Get("cursor"); c != "" {
			name += "-" + path.Base(c)
		}
		return fileOr404(r, name+".json")
	}
	if b, ok := recorded("notifuse/" + strings.TrimPrefix(p, "/api/") + ".json"); ok {
		return answer(r, 200, b), nil
	}
	return answer(r, 404, []byte(`{"error":"not found"}`)), nil
}
