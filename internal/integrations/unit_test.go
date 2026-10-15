package integrations

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/authz"
)

func TestParseWorkflow(t *testing.T) {
	for _, tc := range []struct{ in, id, name string }{
		{"Stripe payment failed → payment-failed email; id=284113907", "284113907", "Stripe payment failed → payment-failed email"},
		{"Stripe%20payment%20failed%20%E2%86%92%20email; id=7", "7", "Stripe payment failed → email"},
		{`name="Beta (Typeform)"; id="42"`, "42", "Beta (Typeform)"},
		{"name*=UTF-8''Acme%20Cloud", "", "Acme Cloud"},
		{"id=99", "99", ""},
		{"", "", ""},
		{"Just a name", "", "Just a name"},
	} {
		w := ParseWorkflow(tc.in)
		if w.ID != tc.id || w.Name != tc.name {
			t.Errorf("ParseWorkflow(%q) = %+v, want id %q name %q", tc.in, w, tc.id, tc.name)
		}
	}
	if (Workflow{ID: "1", Name: "x"}).Key() != "1" || (Workflow{Name: "x"}).Key() != "x" {
		t.Error("a workflow is keyed by its id, else its name")
	}
}

func TestCatalog(t *testing.T) {
	all := []string{}
	for _, it := range Items(KindZapier) {
		all = append(all, it.Key)
	}
	perms := Permissions(KindZapier, all)
	got := []string{}
	for _, p := range perms {
		got = append(got, string(p))
	}
	// subscribers:view is left out: subscribers:edit includes it.
	want := "subscribers:create,subscribers:edit,campaigns:view,transactional:view,transactional:send,webhooks:manage"
	if strings.Join(got, ",") != want {
		t.Errorf("Zapier with everything on holds %s, want %s", strings.Join(got, ","), want)
	}
	if slices.Contains(DefaultItems(KindZapier), "send_transactional") {
		t.Error("actions that need a ⚑ permission start off")
	}
	if len(DefaultItems(KindZapier)) != 10 {
		t.Errorf("10 of 11 start on, got %d", len(DefaultItems(KindZapier)))
	}
	ops := Operations(KindMake, []string{"upsert_contact"})
	if !ops[OpMe] || !ops[OpUpsert] || ops[OpTag] || ops[OpSubscribe] {
		t.Errorf("operations of Make with one action: %v", ops)
	}
	ops = Operations(KindN8n, []string{"contact.updated"})
	if !ops[OpSubscribe] || !ops[OpUnsubscribe] || !ops[OpSample] || ops[OpUpsert] {
		t.Errorf("operations of a trigger: %v", ops)
	}
	wp := Operations(KindWordPress, []string{"list_forms"})
	if !wp[OpWPReport] || !wp[OpWPForms] || wp[OpWPDraft] || wp[OpUpsert] {
		t.Errorf("a site that only lists forms: %v", wp)
	}
	if p := Permissions(KindN8n, []string{"record_event"}); len(p) != 1 || p[0] != authz.SubscribersEdit {
		t.Errorf("record an event needs subscribers:edit only, got %v", p)
	}
	// Triggers also need webhooks:manage from the person who turns them on.
	it, _ := ItemByKey(KindZapier, "contact.subscribed")
	if !slices.Contains(Needs(it), authz.WebhooksManage) {
		t.Error("a trigger needs webhooks:manage")
	}
	nonEmpty := 0
	for _, g := range Groups {
		if len(g.Entries) > 0 {
			nonEmpty++
		}
	}
	if nonEmpty != 4 || Groups[0].Key != "automation" || Groups[3].Key != "developer" {
		t.Errorf("phase 2 fills four groups, in order: %+v", Groups)
	}
}

func TestStoredPermissions(t *testing.T) {
	got := storedPermissions(KindN8n, []string{"contact.unsubscribed", "upsert_contact", "record_event"})
	if !slices.Contains(got, "subscribers:view") || got[0] != "subscribers:create" {
		t.Errorf("stored: %v", got)
	}
	if s := shortPermissions(got); strings.Join(s, ",") != "subscribers:create,subscribers:edit,webhooks:manage" {
		t.Errorf("short: %v", s)
	}
}

func TestKeyName(t *testing.T) {
	if n, l := keyName(KindMake, "Make · Acme ops", "Acme"); n != "Make · Acme ops" || l != "Acme ops" {
		t.Errorf("%q %q", n, l)
	}
	if n, l := keyName(KindMake, "Ops team", "Acme"); n != "Make · Ops team" || l != "Ops team" {
		t.Errorf("%q %q", n, l)
	}
	if n, _ := keyName(KindZapier, "", "Acme ops"); n != "Zapier · Acme ops" {
		t.Errorf("default: %q", n)
	}
}

func TestSiteHost(t *testing.T) {
	for in, want := range map[string]string{"acme.io": "acme.io", "https://Blog.Acme.io/": "blog.acme.io", "http://acme.io/news/": "http://acme.io/news", "blog.acme.io:8443": "blog.acme.io:8443"} {
		if got, err := siteHost(in); err != nil || got != want {
			t.Errorf("siteHost(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := siteHost(" "); err == nil {
		t.Error("an empty address is refused")
	}
}

func TestNormalizeLanguage(t *testing.T) {
	p := func(s string) *string { return &s }
	for in, want := range map[string]string{"es_MX": "es-MX", "ES": "es", "Spanish": "es", "zh-hant-tw": "zh-Hant-TW", "pt-br": "pt-BR"} {
		if got, err := normalizeLanguage(p(in)); err != nil || got != want {
			t.Errorf("normalizeLanguage(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := normalizeLanguage(p("Spanglish!")); err == nil || !strings.Contains(err.Error(), "es") {
		t.Errorf("a near miss names the closest match: %v", err)
	}
	if got, err := normalizeLanguage(nil); err != nil || got != "" {
		t.Error("no language is fine")
	}
}

func TestMaskBody(t *testing.T) {
	var v any
	_ = json.Unmarshal([]byte(`{"template":"payment-failed","to":{"email":"a@b.co"},"data":{"first_name":"Priya","update_url":"https://x","api_token":"t"},"password":"p"}`), &v)
	out, _ := json.Marshal(maskValue(v, "", map[string]bool{"update_url": true}))
	s := string(out)
	if strings.Contains(s, "https://x") || strings.Contains(s, `"t"`) || strings.Contains(s, `"p"`) || !strings.Contains(s, "Priya") {
		t.Errorf("masked: %s", s)
	}
	hidden := hideAddresses(json.RawMessage(`{"to":"finance@initech.example"}`), "a contact")
	if strings.Contains(string(hidden), "@") {
		t.Errorf("addresses hidden: %s", hidden)
	}
}

func TestVersions(t *testing.T) {
	if !newerVersion("1.3.0", "1.2.0") || newerVersion("1.2.0", "1.2.0") || newerVersion("1.10.0", "1.9.9") == false || newerVersion("", "1.0.0") {
		t.Error("versions compare by number")
	}
}

func TestClientLabel(t *testing.T) {
	for ua, want := range map[string]string{
		"ReputeCast-Zapier/1.2.0 (zapier.com)":  "Zapier, ReputeCast app 1.2.0",
		"ReputeCast-Make/1.0.4":                 "Make, ReputeCast app 1.0.4",
		"ReputeCast-n8n/1.1.0":                  "n8n, ReputeCast node 1.1.0",
		"WordPress/6.9.2; https://blog.acme.io": "WordPress",
		"curl/8.4.0":                            "curl",
		"":                                      "",
	} {
		if got := clientLabel(ua); got != want {
			t.Errorf("clientLabel(%q) = %q, want %q", ua, got, want)
		}
	}
}

func TestMaskEndpoint(t *testing.T) {
	if got := maskEndpoint("https://n8n.acme.io/webhook/7c1e9d42-3b8a-4f6e-a1c5-2d9e8f7b6a30/reputecast"); got != "https://n8n.acme.io/webhook/7c1e…/reputecast" {
		t.Errorf("masked: %q", got)
	}
}
