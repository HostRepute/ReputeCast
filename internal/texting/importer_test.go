package texting_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/texting"
)

// anyMX answers every domain with an MX record.
type anyMX struct{}

func (anyMX) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	return []*net.MX{{Host: "mx." + name, Pref: 10}}, nil
}

func (anyMX) LookupHost(context.Context, string) ([]string, error) { return []string{"192.0.2.1"}, nil }

// An import's numbers and SMS consent (brief §3.3): SMS consent comes in
// only with its date, source and text, recorded as imported; without them
// the number comes in and SMS consent stays Not subscribed. Nobody is texted.
func TestSMSConsentFromAnImport(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	sms, _, _ := service(env, time.Now)
	aud := &audience.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Mail: env.App.Mail, Links: env.App.Links, URLs: env.App.URLs}
	svc := &imports.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Audience: aud, Mail: env.App.Mail,
		MX: &imports.MXChecker{Q: env.App.Q, Resolver: anyMX{}}, HTTP: http.DefaultClient, Inline: true, Texts: texting.ImportTexts{S: sms}}
	summit := one[string](t, env, `SELECT id::text FROM lists WHERE name = 'Acme Summit 2026'`)

	var d struct {
		ID      string
		Status  string
		Options map[string]any
		Stats   *struct{ Texts map[string]int64 }
	}
	linh.Do(http.MethodPost, ws+"/imports", map[string]any{"source": "csv"}).MustStatus(t, 201).JSON(t, &d)
	id := uuid.MustParse(d.ID)
	file := strings.Join([]string{
		"email,first_name,mobile,sms_consent,sms_consent_date,sms_consent_source,sms_consent_text",
		"pat.import@gmail.com,Pat,+1 312 555 0181,yes,2026-09-30,Summit booth sign-up sheet,Text me Acme Summit reminders. Msg & data rates may apply. Reply STOP to cancel.",
		"quinn.import@gmail.com,Quinn,(312) 555-0182,yes,,,",
		"ray.import@gmail.com,Ray,07700 900183,no,,,",
	}, "\n")
	if _, err := svc.Upload(ctx, &w, id, "summit-booth.csv", []byte(file)); err != nil {
		t.Fatalf("upload: %v", err)
	}
	linh.Do(http.MethodGet, ws+"/imports/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	opts := d.Options
	opts["lists"] = []map[string]string{{"sourceId": "file", "target": summit}}
	opts["pullSuppressions"] = false
	linh.Do(http.MethodPatch, ws+"/imports/"+d.ID, map[string]any{"options": opts}).MustStatus(t, 200)
	linh.Do(http.MethodPost, ws+"/imports/"+d.ID+"/stage", nil).MustStatus(t, 200)
	if err := svc.Stage(ctx, id); err != nil {
		t.Fatalf("stage: %v", err)
	}
	var fields struct {
		Fields []struct {
			Source, Target string
			Options        []struct{ Value string }
		}
	}
	linh.Do(http.MethodGet, ws+"/imports/"+d.ID+"/fields", nil).MustStatus(t, 200).JSON(t, &fields)
	choices := []map[string]string{}
	for _, f := range fields.Fields {
		target := f.Target
		if f.Source == "mobile" {
			target = imports.TargetPhone
		}
		choices = append(choices, map[string]string{"source": f.Source, "target": target})
		if f.Source == "sms_consent_date" && f.Target != imports.TargetSmsConsentAt {
			t.Errorf("sms_consent_date guessed %q", f.Target)
		}
	}
	linh.Do(http.MethodPut, ws+"/imports/"+d.ID+"/fields", map[string]any{"fields": choices}).MustStatus(t, 200)
	var consent struct {
		Sms *struct {
			ConsentMapped, EvidenceMapped bool
			Missing                       []string
		}
	}
	linh.Do(http.MethodGet, ws+"/imports/"+d.ID+"/consent", nil).MustStatus(t, 200).JSON(t, &consent)
	if consent.Sms == nil || !consent.Sms.EvidenceMapped || len(consent.Sms.Missing) != 0 {
		t.Fatalf("the Consent step's SMS line: %+v", consent.Sms)
	}
	linh.Do(http.MethodPut, ws+"/imports/"+d.ID+"/consent", map[string]any{
		"map": map[string]string{"subscribed": "confirmed", "pending": "pending", "transactional": "skip"}, "evidence": "trust", "pullSuppressions": false, "recordHistory": true,
	}).MustStatus(t, 200)
	linh.Do(http.MethodPost, ws+"/imports/"+d.ID+"/preview", nil).MustStatus(t, 200)
	linh.Do(http.MethodPost, ws+"/imports/"+d.ID+"/start", nil).MustStatus(t, 200)
	if err := svc.Run(ctx, id); err != nil {
		t.Fatalf("run: %v", err)
	}
	linh.Do(http.MethodGet, ws+"/imports/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "done" || d.Stats == nil || d.Stats.Texts["consent"] != 1 || d.Stats.Texts["no_evidence"] != 1 || d.Stats.Texts["invalid_phone"] != 1 {
		t.Fatalf("after the run: %s %+v", d.Status, d.Stats)
	}
	contact := func(email string) uuid.UUID {
		return one[uuid.UUID](t, env, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = $2`, w.ID, email)
	}
	pat, quinn := contact("pat.import@gmail.com"), contact("quinn.import@gmail.com")
	if ph := one[string](t, env, `SELECT phone FROM contacts WHERE id = $1`, pat); ph != "+13125550181" {
		t.Errorf("Pat's number: %q", ph)
	}
	var action, source, page, text string
	var at time.Time
	var importID *uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT e.action, e.source, e.page, e.consent_text, e.occurred_at, e.import_id FROM channel_consent_events e
		JOIN channel_consents cc ON cc.contact_id = e.contact_id AND cc.list_id = e.list_id AND cc.channel = 'sms' AND cc.status = 'subscribed'
		WHERE e.contact_id = $1 AND e.channel = 'sms'`, pat).Scan(&action, &source, &page, &text, &at, &importID); err != nil {
		t.Fatal(err)
	}
	if action != "imported" || source != "import:summit-booth.csv" || page != "Summit booth sign-up sheet" || !strings.HasPrefix(text, "Text me Acme Summit reminders.") ||
		at.UTC().Format("2006-01-02") != "2026-09-30" || importID == nil || *importID != id {
		t.Errorf("Pat's evidence: %s %s %q %q %v %v", action, source, page, text, at, importID)
	}
	if ph := one[string](t, env, `SELECT phone FROM contacts WHERE id = $1`, quinn); ph != "+13125550182" {
		t.Errorf("Quinn's number: %q", ph)
	}
	if n := one[int](t, env, `SELECT count(*) FROM channel_consents WHERE contact_id = $1 AND channel = 'sms' AND status = 'subscribed'`, quinn); n != 0 {
		t.Error("SMS consent without its evidence came in")
	}
	if n := one[int](t, env, `SELECT count(*) FROM channel_messages WHERE contact_id = ANY($1)`, []uuid.UUID{pat, quinn}); n != 0 {
		t.Errorf("an import texted %d times", n)
	}
}
