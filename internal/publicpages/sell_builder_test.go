package publicpages_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/publicpages"
	"github.com/hostrepute/reputecast/internal/store"
)

// The page builder's canvas, live blocks and previews: drawn by the public
// site, inert (nothing posts), drafts included; website:view is enough
// (Aisha Bello is a Viewer), without it they're refused.
func TestPageBuilderPreviews(t *testing.T) {
	e := spDemo(t)
	aisha := e.Client()
	aisha.SignInDemo("aisha@acme.io")
	fg := e.pageID("field-guide")
	base := wsAPI + "/pages/" + fg

	var canvas publicpages.PageCanvas
	aisha.Do(http.MethodGet, base+"/canvas", nil).MustStatus(t, http.StatusOK).JSON(t, &canvas)
	if len(canvas.Styles) != 3 || !strings.Contains(canvas.Styles[0], "site.css") || !strings.Contains(canvas.Styles[1], "pages.css") || !strings.Contains(canvas.Styles[2], "theme.css") {
		t.Errorf("canvas styles %v", canvas.Styles)
	}
	if !strings.Contains(canvas.Header, `class="st-nav"`) || strings.Contains(canvas.FocusHeader, `class="st-nav"`) || canvas.Footer == "" || canvas.Lang != "en" {
		t.Errorf("canvas frame: header %q focus %q footer %d lang %s", canvas.Header, canvas.FocusHeader, len(canvas.Footer), canvas.Lang)
	}

	// Live blocks, in the order asked; anything else is no block.
	var out publicpages.PageRendered
	aisha.Do(http.MethodPost, base+"/render", map[string]any{"blocks": []string{
		`<rc-product product="` + e.productID("deliverability-field-guide") + `" button="Buy for {price}" price="1" offer="1" layout="box"></rc-product>`,
		`<rc-form form="` + e.formID("frm_e7b302") + `" heading="Save your seat"></rc-form>`,
		`<p>Not a live block</p>`,
		`<rc-product product="` + e.productID("warm-up-planner") + `"></rc-product>`,
		`<rc-form></rc-form>`,
	}}).MustStatus(t, http.StatusOK).JSON(t, &out)
	if len(out.Blocks) != 5 {
		t.Fatalf("%d blocks drawn", len(out.Blocks))
	}
	if b := out.Blocks[0]; !strings.Contains(b, "Buy for $29") || strings.Contains(b, "<form") || strings.Contains(b, "<button") {
		t.Errorf("the product block isn't inert: %s", b)
	}
	if b := out.Blocks[1]; !strings.Contains(b, "Save your seat") || !strings.Contains(b, `class="fcard"`) || strings.Contains(b, "<form") || strings.Contains(b, `name="_t"`) {
		t.Errorf("the form block isn't inert: %s", b)
	}
	if out.Blocks[2] != "" {
		t.Errorf("a paragraph drawn as a block: %q", out.Blocks[2])
	}
	// A draft product shows in the builder; an empty form block says what to do.
	if b := out.Blocks[3]; !strings.Contains(b, "$12") {
		t.Errorf("the draft product: %s", b)
	}
	if b := out.Blocks[4]; !strings.Contains(b, "Pick the signup form for this block.") {
		t.Errorf("the empty form block: %s", b)
	}

	// The preview of the draft: version 9's changes, nothing that posts.
	var pv publicpages.PublicPagePreview
	aisha.Do(http.MethodPost, base+"/preview", map[string]any{}).MustStatus(t, http.StatusOK).JSON(t, &pv)
	if !strings.Contains(pv.HTML, "Eight practical chapters on authentication") || !strings.Contains(pv.HTML, "Hannah Weiss") || pv.URL != "http://news.acme.io/field-guide" {
		t.Errorf("the draft's preview at %s lacks its changes", pv.URL)
	}
	if strings.Contains(pv.HTML, `/buy/`) || !strings.Contains(pv.HTML, ` inert`) {
		t.Error("the preview isn't inert")
	}
	// What's live.
	aisha.Do(http.MethodPost, base+"/preview", map[string]any{"live": true}).MustStatus(t, http.StatusOK).JSON(t, &pv)
	if !strings.Contains(pv.HTML, "Eight chapters on authentication, warm-up and list hygiene.") || strings.Contains(pv.HTML, "Hannah Weiss") {
		t.Error("the live preview shows the draft")
	}
	// The blocks being edited, not saved yet.
	aisha.Do(http.MethodPost, base+"/preview", map[string]any{"html": spPlain("An unsaved heading", "Typed a second ago."), "hideHeader": true}).
		MustStatus(t, http.StatusOK).JSON(t, &pv)
	// (The draft's SEO description, in the head, says "Eight practical chapters" too.)
	if !strings.Contains(pv.HTML, "An unsaved heading") || strings.Contains(pv.HTML, "Eight practical chapters on authentication") || strings.Contains(pv.HTML, `class="st-nav"`) {
		t.Error("the preview of unsaved blocks")
	}
	// A page never published.
	aisha.Do(http.MethodPost, wsAPI+"/pages/"+e.pageID("sponsor")+"/preview", map[string]any{}).MustStatus(t, http.StatusOK).JSON(t, &pv)
	if !strings.Contains(pv.HTML, "Sponsor the Weekly Digest") {
		t.Error("the sponsor draft's preview")
	}
	unknown := wsAPI + "/pages/" + uuid.Must(uuid.NewV7()).String()
	aisha.Do(http.MethodPost, unknown+"/preview", map[string]any{}).MustStatus(t, http.StatusNotFound)
	aisha.Do(http.MethodPost, unknown+"/render", map[string]any{"blocks": []string{}}).MustStatus(t, http.StatusNotFound)

	// Without website:view.
	reader := spMemberWithout(t, e, "campaigns:view")
	for _, r := range []*apptest.Response{
		reader.Do(http.MethodGet, base+"/canvas", nil),
		reader.Do(http.MethodPost, base+"/render", map[string]any{"blocks": []string{}}),
		reader.Do(http.MethodPost, base+"/preview", map[string]any{}),
		reader.Do(http.MethodGet, base, nil),
	} {
		if r.Status != http.StatusForbidden || !strings.Contains(string(r.Body), `"permission":"website:view"`) {
			t.Errorf("without website:view: %d %s", r.Status, r.Body)
		}
	}
}

// spMemberWithout signs in a new member of Acme Newsletter whose custom
// role holds only perms.
func spMemberWithout(t *testing.T, e *spEnv, perms ...string) *apptest.Client {
	t.Helper()
	ctx := context.Background()
	if _, err := e.App.Q.CreateRole(ctx, store.CreateRoleParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: e.ws.ID, Key: "campaign-reader", Name: "Campaign reader",
		Description: "Reads campaigns, nothing else", Permissions: perms}); err != nil {
		t.Fatal(err)
	}
	u := e.CreateUser(apptest.UserOpts{Email: "rae.reader@acme.io", Name: "Rae Reader"})
	e.AddMember(e.ws, u.ID, "campaign-reader")
	c := e.Client()
	c.SignInDemo("rae.reader@acme.io")
	return c
}
