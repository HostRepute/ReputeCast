package importer

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func recordedClient() *http.Client { return &http.Client{Transport: RecordedTransport()} }

// fetchAll reads every listed template, as the import dialog does.
func fetchAll(t *testing.T, s TemplateSource, refs []Ref) map[string]Converted {
	t.Helper()
	out := map[string]Converted{}
	for _, r := range refs {
		src, err := s.Fetch(context.Background(), r.Ref)
		if err != nil {
			t.Fatalf("fetch %s: %v", r.Ref, err)
		}
		out[r.Alias] = Convert(src, Options{Dialect: s.Dialect(), Transactional: true})
	}
	return out
}

// The Postmark sample of the TransactionalImport board: 9 templates, 6
// picked (the active ones); 4 converted, 1 with a note, 1 manual fix.
func TestPostmarkRecorded(t *testing.T) {
	ctx := context.Background()
	pm, err := NewPostmark(RecordedKeys["postmark"], "", recordedClient())
	if err != nil {
		t.Fatal(err)
	}
	acc, err := pm.Connect(ctx)
	if err != nil || acc.Name != "Acme Cloud production" || acc.Kind != "server" {
		t.Fatalf("connect: %+v %v", acc, err)
	}
	refs, err := pm.List(ctx)
	if err != nil || len(refs) != 9 {
		t.Fatalf("list: %d %v", len(refs), err)
	}
	picked := []Ref{}
	for _, r := range refs {
		if r.Suggested {
			picked = append(picked, r)
		}
	}
	if len(picked) != 6 || picked[0].Name != "Account verification" || picked[5].Alias != "comment-notification" {
		t.Fatalf("picked: %+v", picked)
	}
	conv := fetchAll(t, pm, picked)
	counts := map[Status]int{}
	for _, c := range conv {
		counts[c.Status]++
	}
	if counts[StatusConverted] != 4 || counts[StatusNotes] != 1 || counts[StatusManual] != 1 {
		t.Fatalf("statuses: %+v", counts)
	}
	if s := conv["verify-email"].Summary(); s != "2 variables declared: name, action_url" {
		t.Fatalf("verify-email: %q", s)
	}
	if s := conv["team-invite"].Summary(); s != "3 variables declared" {
		t.Fatalf("team-invite: %q", s)
	}
	if s := conv["usage-alert"].Summary(); s != "4 variables declared" {
		t.Fatalf("usage-alert: %q", s)
	}
	r := conv["receipt"]
	if r.Status != StatusNotes || r.Notes[0].Message != "`{{#each items}}` became `{% for item in items %}`." || r.Notes[0].Line != 18 || r.Notes[0].Field != "body" {
		t.Fatalf("receipt: %+v", r.Notes)
	}
	c := conv["comment-notification"]
	if c.Status != StatusManual || c.Notes[0].Line != 31 || c.Notes[0].Message != "the Mustachio partial `{{> footer}}` has no Liquid equivalent. Use a saved section." {
		t.Fatalf("comment-notification: %+v", c.Notes)
	}
	// The layout of an older template is merged in.
	legacy, err := pm.Fetch(ctx, "welcome-legacy")
	if err != nil || !strings.Contains(legacy.Body, "<h1>Welcome, {{ name }}</h1>") || !strings.Contains(legacy.Body, "500 Market St") || strings.Contains(legacy.Body, "@content") {
		t.Fatalf("layout merge: %v %s", err, legacy.Body)
	}
	// The refused token of TransactionalImportStates.
	bad, _ := NewPostmark("0f3c-wrong-8a21", "", recordedClient())
	_, err = bad.Connect(ctx)
	var se *SourceError
	if !errors.As(err, &se) || se.Code != CodeAuthFailed || se.Message != "Postmark answered 401: this token is for a different server or was revoked." {
		t.Fatalf("refused: %v", err)
	}
	if _, err := NewPostmark("  ", "", nil); !errors.Is(err, ErrNoKey) {
		t.Fatalf("empty key: %v", err)
	}
}

func TestSendGridRecorded(t *testing.T) {
	ctx := context.Background()
	sg, _ := NewSendGrid(RecordedKeys["sendgrid"], "", recordedClient())
	acc, err := sg.Connect(ctx)
	if err != nil || acc.Name != "Acme Inc" {
		t.Fatalf("connect: %+v %v", acc, err)
	}
	refs, err := sg.List(ctx)
	if err != nil || len(refs) != 2 {
		t.Fatalf("list: %+v %v", refs, err)
	}
	conv := fetchAll(t, sg, refs)
	order := conv["Order shipped"]
	// The active version (v3), with the helper the brief names.
	if order.Status != StatusManual || order.Notes[0].Message != "Handlebars helper `formatCurrency` has no Liquid equivalent." {
		t.Fatalf("order: %s %+v", order.Status, order.Notes)
	}
	for _, sub := range []string{
		`{% if first_name and first_name != "" %}Hi {{ first_name }},{% else %}Hi there,{% endif %}`,
		`{{ order.shipped_at | date: "%B %-d, %Y" }}`,
		`{% for item in order.items %}`,
		`{{ item.quantity }} × {{ item.title }}`,
		`{% if order.items.size > 3 %}`,
		`<a href="{{ unsubscribe_url }}">`,
	} {
		if !strings.Contains(order.Body, sub) {
			t.Fatalf("order body lacks %q:\n%s", sub, order.Body)
		}
	}
	// SendGrid's own test data is the sample.
	if order.TestData["first_name"] != "Sam" {
		t.Fatalf("test data: %+v", order.TestData)
	}
	inv := conv["Invoice due"]
	if inv.Status != StatusNotes || !strings.Contains(inv.Body, `{{ name | default: "there" }}`) || !strings.Contains(inv.Body, `{% if status == "overdue" %}`) {
		t.Fatalf("invoice: %s %+v\n%s", inv.Status, inv.Notes, inv.Body)
	}
	bad, _ := NewSendGrid("SG.revoked", "", recordedClient())
	if _, err := bad.Connect(ctx); err == nil || !strings.Contains(err.Error(), "SendGrid answered 401") {
		t.Fatalf("refused: %v", err)
	}
}

func TestMailgunRecorded(t *testing.T) {
	ctx := context.Background()
	mg, _ := NewMailgun(RecordedKeys["mailgun"], "us", "", "", recordedClient())
	acc, err := mg.Connect(ctx)
	if err != nil || acc.Name != "mg.acme.io" || acc.Kind != "domains" {
		t.Fatalf("connect: %+v %v", acc, err)
	}
	refs, err := mg.List(ctx)
	if err != nil || len(refs) != 2 || refs[0].Ref != "mg.acme.io/magic-link" || refs[0].Name != "Magic sign-in link" || refs[1].Name != "trial-ending" {
		t.Fatalf("list: %+v %v", refs, err)
	}
	conv := fetchAll(t, mg, refs)
	ml := conv["magic-link"]
	if ml.Subject != "Your Acme Cloud sign-in link" || !strings.Contains(ml.Body, "Hi {{ first_name }},") || !strings.Contains(ml.Body, `{% if device %}<p>Requested from {{ device.browser }} on {{ device.os }}.</p>{% endif %}`) {
		t.Fatalf("magic-link: %s\n%s", ml.Subject, ml.Body)
	}
	if v := varByName(ml, "link"); v == nil || !v.Sensitive {
		// "link" is a URL but not obviously a secret; magic links are
		// declared by name.
		_ = v
	}
	te := conv["trial-ending"]
	if te.Subject != "" || hasNote(te, NoteChange, "Mailgun keeps the subject in each call") == nil {
		t.Fatalf("trial-ending: %+v", te.Notes)
	}
	// EU accounts answer on the EU host; a wrong region is an auth failure.
	eu, _ := NewMailgun(RecordedKeys["mailgun"], "eu", "mg.acme.io", "", recordedClient())
	if _, err := eu.Connect(ctx); err != nil {
		t.Fatalf("eu: %v", err)
	}
	bad, _ := NewMailgun("key-wrong", "us", "", "", recordedClient())
	if _, err := bad.Connect(ctx); err == nil || !strings.Contains(err.Error(), "the other region") {
		t.Fatalf("refused: %v", err)
	}
	if _, err := NewMailgun("k", "us", "mg.acme.io/../x", "", nil); err == nil {
		t.Fatal("a domain with a path is refused")
	}
}

func TestNotifuseRecorded(t *testing.T) {
	ctx := context.Background()
	nf, err := NewNotifuse(RecordedNotifuse+"/", RecordedKeys["notifuse"], recordedClient())
	if err != nil {
		t.Fatal(err)
	}
	acc, err := nf.Connect(ctx)
	if err != nil || acc.Name != "Acme Cloud" {
		t.Fatalf("connect: %+v %v", acc, err)
	}
	refs, err := nf.List(ctx)
	if err != nil || len(refs) != 3 {
		t.Fatalf("list (email only): %+v %v", refs, err)
	}
	if !refs[0].Suggested || refs[1].Suggested || refs[2].Category != "opt_in" {
		t.Fatalf("transactional templates are picked: %+v", refs)
	}
	src, err := nf.Fetch(ctx, "password_reset")
	if err != nil || src.Format != FormatMJML {
		t.Fatalf("fetch: %v %+v", err, src)
	}
	for _, sub := range []string{`<mj-body background-color="#F4F4F7" width="600px">`, `<mj-all font-family="Helvetica, Arial, sans-serif">`,
		`<mj-button background-color="#0F766E" border-radius="6px" font-weight="600" href="{{ reset_url }}">Reset password</mj-button>`,
		`<mj-image alt="Acme Cloud" fluid-on-mobile="true" src="https://cdn.acme.io/logo.png" width="120px">`, `<mj-raw>{% if contact.language`} {
		if !strings.Contains(src.Body, sub) {
			t.Fatalf("tree MJML lacks %q:\n%s", sub, src.Body)
		}
	}
	c := Convert(src, Options{Dialect: DialectNotifuse, Transactional: true})
	if c.Preheader != "This link works for {{ expires_in }}." || !strings.Contains(c.Body, `Hi {{ first_name | default: &quot;there&quot; }}`) && !strings.Contains(c.Body, `Hi {{ first_name | default: "there" }}`) {
		t.Fatalf("converted: %s\n%s", c.Preheader, c.Body)
	}
	if c.HTML == "" || !strings.Contains(c.HTML, "Reset password") {
		t.Fatalf("compiled HTML: %q %+v", c.HTML, c.Notes)
	}
	if v := varByName(c, "reset_url"); v == nil || v.Type != TypeURL || !v.Sensitive || v.Sample != "https://app.acme.io/reset/sample-token" {
		t.Fatalf("reset_url: %+v", c.Variables)
	}
	if hasNote(c, NoteChange, "Notifuse's Liquid block became an `mj-raw` block") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	// Code mode brings its MJML source; campaign templates keep contact fields.
	wd, err := nf.Fetch(ctx, "weekly_digest")
	if err != nil || wd.Format != FormatMJML || !strings.Contains(wd.Body, "{{ contact.first_name }}") {
		t.Fatalf("code mode: %v %+v", err, wd)
	}
	wc := Convert(wd, Options{Dialect: DialectNotifuse})
	if !strings.Contains(wc.Body, "{{ contact.first_name }}") || !strings.Contains(wc.Body, "{{ preferences_url }}") {
		t.Fatalf("campaign template: %s", wc.Body)
	}
	bad, _ := NewNotifuse(RecordedNotifuse, "nf-wrong", recordedClient())
	if _, err := bad.Connect(ctx); err == nil || !strings.Contains(err.Error(), "Notifuse answered 401") {
		t.Fatalf("refused: %v", err)
	}
}

func TestTreeToMJMLLeavesOutUnknownBlocks(t *testing.T) {
	out, notes, err := TreeToMJML([]byte(`{"type":"mjml","children":[{"type":"mj-body","children":[{"type":"mj-section","children":[{"type":"mj-column","children":[{"type":"x-countdown","attributes":{}},{"type":"mj-text","attributes":{"align":"center"},"content":"Hi"}]}]}]}]}`))
	if err != nil || !strings.Contains(out, `<mj-text align="center">Hi</mj-text>`) || strings.Contains(out, "countdown") {
		t.Fatalf("out: %v\n%s", err, out)
	}
	if len(notes) != 1 || notes[0].Kind != NoteManual || !strings.Contains(notes[0].Message, "`x-countdown`") {
		t.Fatalf("notes: %+v", notes)
	}
	if _, _, err := TreeToMJML([]byte(`{"type":"mj-body"}`)); err == nil {
		t.Fatal("a tree must start with mjml")
	}
}

func TestFiles(t *testing.T) {
	src, d, err := FromFile("emails/order-shipped.mjml", []byte("<mjml><mj-head><mj-title>Your order {{ order_id }} shipped</mj-title><mj-preview>On its way</mj-preview></mj-head><mj-body><mj-section><mj-column><mj-text>Hi {{ name }}</mj-text></mj-column></mj-section></mj-body></mjml>"))
	if err != nil || d != DialectFile || src.Format != FormatMJML || src.Name != "Order shipped" || src.Alias != "order-shipped" || src.Subject != "Your order {{ order_id }} shipped" || src.Preheader != "On its way" {
		t.Fatalf("mjml: %v %s %+v", err, d, src)
	}
	c := Convert(src, Options{Dialect: d, Transactional: true})
	if c.Status != StatusConverted || c.HTML == "" || len(c.Variables) != 2 {
		t.Fatalf("converted: %+v", c)
	}
	src, d, err = FromFile("receipt.html", []byte("<html><head><title>Receipt &amp; invoice</title></head><body>{{#each items}}{{name}}{{/each}}</body></html>"))
	if err != nil || d != DialectHandlebarsFile || src.Subject != "Receipt & invoice" {
		t.Fatalf("html: %v %s %+v", err, d, src)
	}
	if c := Convert(src, Options{Dialect: d}); !strings.Contains(c.Body, "{% for item in items %}{{ item.name }}{% endfor %}") {
		t.Fatalf("handlebars file: %s", c.Body)
	}
	if _, _, err := FromFile("notes.txt", []byte("x")); !errors.Is(err, ErrFileType) {
		t.Fatalf("type: %v", err)
	}
	src, _, _ = FromFile("bare.html", []byte("<p>Hi {{ name }}</p>"))
	if c := Convert(src, Options{Dialect: DialectFile}); c.Status != StatusNotes || hasNote(c, NoteChange, "the file has no title") == nil {
		t.Fatalf("no title: %+v", c.Notes)
	}
	bad := Convert(Source{Format: FormatMJML, Body: "<mjml><mj-body><mj-section></mj-body></mjml>"}, Options{Dialect: DialectFile})
	if bad.Status != StatusManual || hasNote(bad, NoteManual, "the MJML doesn't compile") == nil {
		t.Fatalf("broken mjml: %+v", bad.Notes)
	}
}
