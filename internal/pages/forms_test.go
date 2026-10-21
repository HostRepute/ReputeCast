package pages_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/pages"
)

// The form builder's "Used on pages": the pages whose signup block uses a
// form, live or only in a draft; a form taken out of a published page's
// draft stays listed until the change is published.
func TestFormUsedOnPages(t *testing.T) {
	f := newFixture(t)
	aisha := f.as("aisha@acme.io") // subscribers:view is enough
	used := func(form string) []pages.FormPage {
		t.Helper()
		var out pages.FormPages
		aisha.Do(http.MethodGet, "/workspaces/acme-newsletter/forms/"+form+"/pages", nil).MustStatus(t, http.StatusOK).JSON(t, &out)
		return out.Items
	}

	dmarc := used(f.formID("frm_e7b302"))
	if len(dmarc) != 1 || dmarc[0].Name != "Live: Fix your DMARC setup in 30 minutes" || dmarc[0].Path != "/dmarc-live" || !strings.HasSuffix(dmarc[0].URL, "/dmarc-live") ||
		dmarc[0].Status != pages.StatusPublished || !dmarc[0].Live {
		t.Fatalf("the DMARC live form: %+v", dmarc)
	}
	// The waitlist block is still on the Launch Kit's page (the countdown
	// hides it on the site).
	if w := used(f.formID("frm_a91f55")); len(w) != 1 || w[0].Path != "/launch-kit" || !w[0].Live {
		t.Errorf("the waitlist form: %+v", w)
	}

	// Only in a draft.
	beta := f.formID("frm_b7a412")
	if b := used(beta); len(b) != 0 {
		t.Fatalf("Beta apply on a demo page: %+v", b)
	}
	d := f.page("Beta invite", plain("Beta invite", "Apply below.")+section("b-signup", `<rc-form data-pb-id="signup" form="`+beta+`" heading="Apply"></rc-form>`))
	if b := used(beta); len(b) != 1 || b[0].ID != d.ID || b[0].Status != pages.StatusDraft || b[0].Live || b[0].Path != "/beta-invite" {
		t.Errorf("a draft using Beta apply: %+v", b)
	}

	// About's subscribe band, taken out of its draft: listed until published.
	footer := scan[uuid.UUID](f, `SELECT id FROM forms WHERE workspace_id = $1 AND slug = 'footer-mini-form'`, f.ws.ID).String()
	about := f.pageID("about")
	if a := used(footer); len(a) != 1 || a[0].ID != about || !a[0].Live {
		t.Fatalf("the footer form: %+v", a)
	}
	f.save(f.owner, about, f.detail(f.owner, about).Draft.Version, plain("About Acme", "No signup band any more."))
	if a := used(footer); len(a) != 1 || a[0].ID != about || !a[0].Live || a[0].Path != "/about" {
		t.Errorf("the live About still has the form: %+v", a)
	}
	f.publishNow(f.owner, about)
	if a := used(footer); len(a) != 0 {
		t.Errorf("after publishing About without it: %+v", a)
	}
	if r := aisha.Do(http.MethodGet, "/workspaces/acme-newsletter/forms/not-a-uuid/pages", nil); r.Status < 400 || r.Status >= 500 {
		t.Errorf("a form id that isn't one: %d %s", r.Status, r.Body)
	}
}
