package imports_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeDNS answers every domain with an MX record, except *.invalid.
type fakeDNS struct{}

func (fakeDNS) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	if strings.HasSuffix(name, ".invalid") {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	return []*net.MX{{Host: "mx." + name, Pref: 10}}, nil
}

func (fakeDNS) LookupHost(_ context.Context, host string) ([]string, error) {
	return []string{"192.0.2.1"}, nil
}

// service is the importer as the worker runs it, inline, for the jobs the
// test server's River client only enqueues.
func service(env *apptest.Env, bases map[string]string) *imports.Service {
	aud := &audience.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Mail: env.App.Mail, Links: env.App.Links, URLs: env.App.URLs}
	return &imports.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Audience: aud, Mail: env.App.Mail,
		MX: &imports.MXChecker{Q: env.App.Q, Resolver: fakeDNS{}}, HTTP: http.DefaultClient, BaseURLs: bases, Inline: true}
}

func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND result = 'allowed'`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type detail struct {
	ID, Source, Status, Step string
	HasCredentials           bool
	StagedRows               int64
	Options                  map[string]any
	Error                    *struct {
		Code    string
		BadRows []struct{ Line, Columns int }
	}
	Stats *struct {
		Imported, Merged, Suppressed, Pending, Skipped int64
		Reasons                                        map[string]int64
	}
}

func TestCSVImport(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Importers", owner.ID)
	c := env.Client()
	c.SignIn("owner@example.com", "")
	viewerUser := env.CreateUser(apptest.UserOpts{Email: "viewer@example.com", Name: "Vic Viewer"})
	env.AddMember(ws, viewerUser.ID, "viewer")
	viewer := env.Client()
	viewer.SignIn("viewer@example.com", "")
	base := "/workspaces/" + ws.Slug

	var list struct{ ID string }
	c.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Newsletter"}).MustStatus(t, 201).JSON(t, &list)
	c.Do(http.MethodPost, base+"/suppressions", map[string]any{"emails": []string{"gone@gmail.com"}}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/contacts", map[string]any{"email": "cara@outlook.com", "lists": []string{list.ID}, "consent": "single"}).MustStatus(t, 201)

	viewer.Do(http.MethodPost, base+"/imports", map[string]any{"source": "csv"}).MustStatus(t, 403)
	var d detail
	c.Do(http.MethodPost, base+"/imports", map[string]any{"source": "csv"}).MustStatus(t, 201).JSON(t, &d)
	id := d.ID

	file := strings.Join([]string{
		"email,first_name,status,company,tags",
		"ana@gmail.com,Ana,subscribed,Acme,customer",
		"ben@gmail.com,Ben,unsubscribed,,",
		"not-an-address,Bad,subscribed,,",
		`"bad,comma@",Comma,subscribed,,`,
		"gone@gmail.com,Gone,subscribed,,",
		"cara@outlook.com,Cara,subscribed,Cara Co,",
		"info@gmail.com,Info,subscribed,,",
		"temp@mailinator.com,Temp,subscribed,,",
		`dan@yahoo.com,Dan,subscribed,"Dan, Inc",`,
		"eve@gmail.com,Eve,subscribed,Eve,x,extra",
	}, "\n")
	// The browser posts the file as is; the service gets the same bytes.
	svc := service(env, nil)
	if _, err := svc.Upload(ctx, &ws, uuid.MustParse(id), "people.csv", []byte(file)); err != nil {
		t.Fatalf("upload: %v", err)
	}
	c.Do(http.MethodGet, base+"/imports/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Error == nil || d.Error.Code != "file_columns" || len(d.Error.BadRows) != 1 || d.Error.BadRows[0].Line != 11 {
		t.Fatalf("broken row not reported: %+v", d.Error)
	}
	d = detail{} // decode into a fresh value: an omitted error must read as none
	c.Do(http.MethodPost, base+"/imports/"+id+"/skip-bad-rows", nil).MustStatus(t, 200).JSON(t, &d)
	if d.Error != nil {
		t.Fatalf("error after skipping: %+v", d.Error)
	}

	opts := d.Options
	opts["lists"] = []map[string]string{{"sourceId": "file", "target": list.ID}}
	opts["pullSuppressions"] = false
	c.Do(http.MethodPatch, base+"/imports/"+id, map[string]any{"options": opts}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+id+"/stage", nil).MustStatus(t, 200)
	if err := svc.Stage(ctx, uuid.MustParse(id)); err != nil {
		t.Fatalf("stage: %v", err)
	}
	c.Do(http.MethodGet, base+"/imports/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	if d.StagedRows != 9 {
		t.Fatalf("staged %d rows, want 9 (the broken one is left out)", d.StagedRows)
	}

	var fields struct {
		Ready  bool
		Fields []struct{ Source, Target, Status string }
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/fields", nil).MustStatus(t, 200).JSON(t, &fields)
	targets := map[string]string{}
	choices := []map[string]string{}
	for _, f := range fields.Fields {
		targets[f.Source] = f.Target
		choices = append(choices, map[string]string{"source": f.Source, "target": f.Target})
	}
	if !fields.Ready || targets["email"] != "email" || targets["status"] != "status" || targets["tags"] != "tags" {
		t.Fatalf("guesses: %+v", targets)
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/fields?guesses=true", nil).MustStatus(t, 200)
	c.Do(http.MethodPut, base+"/imports/"+id+"/fields", map[string]any{"fields": []map[string]string{{"source": "first_name", "target": "first_name"}}}).MustStatus(t, 422)
	c.Do(http.MethodPut, base+"/imports/"+id+"/fields", map[string]any{"fields": choices}).MustStatus(t, 200)

	var consent struct {
		Statuses []struct {
			Status   string
			Contacts int64
			Locked   bool
		}
		SuppressCount int64
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/consent", nil).MustStatus(t, 200).JSON(t, &consent)
	if consent.SuppressCount != 1 {
		t.Fatalf("consent: %+v", consent)
	}
	c.Do(http.MethodPut, base+"/imports/"+id+"/consent", map[string]any{
		"map": map[string]string{"subscribed": "confirmed", "pending": "pending", "transactional": "skip"}, "evidence": "trust", "pullSuppressions": false, "recordHistory": true,
	}).MustStatus(t, 200)

	var p struct {
		Rows, Invalid, Suppressed, Risky, ImportCount int64
		RiskyGroups                                   []struct {
			Key  string
			Rows int64
		}
	}
	c.Do(http.MethodPost, base+"/imports/"+id+"/preview", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Rows != 9 || p.Invalid != 2 || p.Suppressed != 1 || p.Risky != 2 {
		t.Fatalf("preview: %+v", p)
	}

	c.Do(http.MethodPost, base+"/imports/"+id+"/start", nil).MustStatus(t, 200)
	if err := svc.Run(ctx, uuid.MustParse(id)); err != nil {
		t.Fatalf("run: %v", err)
	}
	c.Do(http.MethodGet, base+"/imports/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "done" || d.Stats == nil {
		t.Fatalf("after the run: %s %+v", d.Status, d.Stats)
	}
	s := d.Stats
	if s.Merged != 1 || s.Suppressed != 1 || s.Reasons["invalid"] != 2 || s.Reasons["disposable"] != 1 || s.Reasons["already_suppressed"] != 1 || s.Reasons["role"] != 1 {
		t.Fatalf("stats: %+v", *s)
	}
	if s.Imported != 3 { // ana, info (no Broadcasts), dan
		t.Fatalf("imported %d, want 3", s.Imported)
	}

	q := env.App.Q
	cara, err := q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: "cara@outlook.com"})
	if err != nil {
		t.Fatal(err)
	}
	if cara.Company != "Cara Co" {
		t.Errorf("merge didn't fill the empty company: %q", cara.Company)
	}
	var src string
	if err := env.App.Pool.QueryRow(ctx, `SELECT source FROM suppressions WHERE workspace_id = $1 AND email = 'ben@gmail.com'`, ws.ID).Scan(&src); err != nil {
		t.Fatalf("ben isn't suppressed: %v", err)
	}
	if src != "import:csv" {
		t.Errorf("suppression source %q, want import:csv", src)
	}
	var role bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT role_address FROM contacts WHERE workspace_id = $1 AND email = 'info@gmail.com'`, ws.ID).Scan(&role); err != nil || !role {
		t.Errorf("info@ should be imported as a role address (excluded from Broadcasts): %v", err)
	}
	var gone int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND email IN ('gone@gmail.com', 'not-an-address', 'temp@mailinator.com')`, ws.ID).Scan(&gone)
	if gone != 0 {
		t.Errorf("%d skipped rows became contacts", gone)
	}

	for _, a := range []string{"import.create", "import.upload", "import.skip_rows", "import.update", "import.stage", "import.map_fields", "import.map_consent", "import.start", "import.finish"} {
		if audits(t, env, a) == 0 {
			t.Errorf("no audit entry for %s", a)
		}
	}
	rep := c.Do(http.MethodGet, base+"/imports/"+id+"/files/report", nil).MustStatus(t, 200)
	if !strings.HasPrefix(rep.Header.Get("Content-Type"), "text/csv") || audits(t, env, "import.report_download") != 1 {
		t.Errorf("report download: %s, audit %d", rep.Header.Get("Content-Type"), audits(t, env, "import.report_download"))
	}
	viewer.Do(http.MethodGet, base+"/imports/"+id+"/files/report", nil).MustStatus(t, 403)
	// The skipped-rows file reads back with any CSV reader: each value once.
	skipped := c.Do(http.MethodGet, base+"/imports/"+id+"/files/skipped", nil).MustStatus(t, 200)
	recs, err := csv.NewReader(bytes.NewReader(skipped.Body)).ReadAll()
	if err != nil {
		t.Fatalf("skipped rows CSV: %v", err)
	}
	values := map[string]bool{}
	for _, rec := range recs[1:] {
		values[rec[1]] = true
	}
	if !values["bad,comma@"] || !values["not-an-address"] {
		t.Errorf("skipped values %v", values)
	}

	var jobs struct {
		Items []struct{ ID, Status, Href string }
	}
	c.Do(http.MethodGet, base+"/jobs", nil).MustStatus(t, 200).JSON(t, &jobs)
	if len(jobs.Items) != 1 || jobs.Items[0].Status != "done" || !strings.HasSuffix(jobs.Items[0].Href, "/report") {
		t.Errorf("jobs tray: %+v", jobs.Items)
	}
	c.Do(http.MethodPatch, base+"/imports/"+id, map[string]any{"step": "fields"}).MustStatus(t, 409)
}

// fakeMailchimp serves the parts of the Mailchimp API the importer reads.
func fakeMailchimp(t *testing.T, key string) *httptest.Server {
	t.Helper()
	members := `{"members":[
		{"email_address":"ana@gmail.com","status":"subscribed","merge_fields":{"FNAME":"Ana"},"tags":[{"name":"customer"}],"timestamp_opt":"2025-03-02T14:11:00+00:00","ip_opt":"203.0.113.9","member_rating":4,"last_changed":"2026-09-01T10:00:00+00:00","location":{"timezone":"Europe/Berlin"}},
		{"email_address":"ben@gmail.com","status":"unsubscribed","merge_fields":{"FNAME":"Ben"},"tags":[],"member_rating":1,"last_changed":"2026-09-01T10:00:00+00:00","location":{}}
	],"total_items":2}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pass, _ := r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		if pass != key {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"title":"API Key Invalid","status":401,"detail":"Your API key may be invalid, or you've attempted to access the wrong datacenter."}`))
			return
		}
		switch p := r.URL.Path; {
		case p == "/" || p == "":
			_, _ = w.Write([]byte(`{"account_name":"Acme Inc"}`))
		case p == "/lists":
			_, _ = w.Write([]byte(`{"lists":[{"id":"3f9a2c","name":"Acme Customers","double_optin":true,"date_created":"2024-02-01T00:00:00+00:00","stats":{"member_count":1,"unsubscribe_count":1,"cleaned_count":0}}]}`))
		case strings.HasSuffix(p, "/merge-fields"):
			_, _ = w.Write([]byte(`{"merge_fields":[{"tag":"FNAME","name":"First name","type":"text"}]}`))
		case strings.HasSuffix(p, "/segments"):
			_, _ = w.Write([]byte(`{"segments":[{"name":"customer","member_count":1}]}`))
		case strings.HasSuffix(p, "/abuse-reports"):
			_, _ = w.Write([]byte(`{"abuse_reports":[],"total_items":0}`))
		case strings.HasSuffix(p, "/members") && r.URL.Query().Get("count") == "1":
			_, _ = w.Write([]byte(`{"total_items":0}`))
		case strings.HasSuffix(p, "/members"):
			_, _ = w.Write([]byte(members))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestMailchimpConnect(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Importers", owner.ID)
	const key = "5f0c9e3a7b2d4e6f8a1b3c5d7e9f0a2b-us21"
	srv := fakeMailchimp(t, key)
	defer srv.Close()
	svc := service(env, map[string]string{"mailchimp": srv.URL})

	imp, err := svc.Create(ctx, &ws, imports.ImportCreateInput{Source: "mailchimp"})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(imp.ID)

	_, err = svc.Connect(ctx, &ws, id, imports.ImportConnectInput{APIKey: "0000-us21"})
	if err == nil || !strings.Contains(err.Error(), `"API Key Invalid"`) {
		t.Fatalf("a wrong key: %v", err)
	}
	got, err := svc.Get(ctx, &ws, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Error == nil || got.Error.Code != imports.CodeAuthFailed || got.HasCredentials {
		t.Fatalf("the failure isn't kept for the error state: %+v", got.Error)
	}

	got, err = svc.Connect(ctx, &ws, id, imports.ImportConnectInput{APIKey: key})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if got.Connection == nil || got.Connection.Account != "Acme Inc" || len(got.Connection.Lists) != 1 || !got.HasCredentials || got.Error != nil {
		t.Fatalf("connection: %+v", got)
	}
	b, _ := json.Marshal(got)
	if bytes.Contains(b, []byte(key)) {
		t.Fatal("the API key is returned")
	}
	var sealed []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT credentials_enc FROM imports WHERE id = $1`, id).Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte(key)) {
		t.Fatal("the API key is stored in plain text")
	}

	// A rate-limited read is resumed slower: 3 requests a second from now on.
	opts := got.Options
	opts.Lists = []imports.ImportListChoice{{SourceID: "3f9a2c", Target: "new:Acme Customers"}}
	if _, err := svc.Update(ctx, &ws, id, imports.ImportUpdateInput{Options: &opts}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE imports SET status = 'paused' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	got, err = svc.Resume(ctx, &ws, id, true)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got.Options.Pace != 3 || got.StagedRows != 2 || got.Status != "draft" {
		t.Fatalf("after resuming slower: pace %d, %d rows, %s", got.Options.Pace, got.StagedRows, got.Status)
	}
	if audits(t, env, "import.resume") != 1 {
		t.Fatal("resume wrote no audit entry")
	}
	if _, err := svc.Resume(ctx, &ws, id, false); err == nil {
		t.Fatal("resuming an import that isn't paused should fail")
	}
}
