package visibility_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// The conditions' words read in the request's language; API clients that
// don't ask get English.
func TestCatalogInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	var cat visibility.VisibilityCatalog
	vi.Do(http.MethodGet, "/workspaces/acme-newsletter/visibility/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	tiers := map[string]string{}
	for _, o := range cat.Tiers {
		tiers[o.Value] = o.Label
	}
	if tiers["active"] != "Tích cực" || tiers["at_risk"] != "Có nguy cơ" {
		t.Fatalf("Vietnamese tiers: %v", cat.Tiers)
	}
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do(http.MethodGet, "/workspaces/acme-newsletter/visibility/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	if cat.Tiers[0].Label != "Active" {
		t.Fatalf("English tiers: %v", cat.Tiers)
	}
}

// A fix's line in Review & send reads in the reader's language; the
// version's label stays English, as the other fixes' labels do.
func TestFixInVietnamese(t *testing.T) {
	rule := visibility.VisibilityRule{ID: "hero01", Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: "tag", Op: "has", Value: "vip"}}}
	src := `<mjml><mj-body><mj-section rc-name="Hero" rc-visible='` + rule.Encode() + `'><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>`
	params := map[string]string{"block": "hero01", "condition": "0"}
	for _, c := range []struct {
		action, en, vi string
	}{
		{visibility.FixHideBlock, "Hid “Hero” for everyone", "Đã ẩn “Hero” với mọi người"},
		{visibility.FixShowEveryone, "Showed “Hero” to everyone", "Đã hiện “Hero” cho mọi người"},
		{visibility.FixRemoveCondition, "Removed the condition of “Hero”", "Đã xóa điều kiện của “Hero”"},
	} {
		_, label, applied, err := visibility.Fix(src, c.action, params)
		if err != nil || applied != c.en {
			t.Fatalf("%s English: %q %v", c.action, applied, err)
		}
		_, viLabel, viApplied, err := visibility.FixIn(i18n.For(i18n.Vietnamese), src, c.action, params)
		if err != nil || viApplied != c.vi || viLabel != label {
			t.Fatalf("%s Vietnamese: %q, label %q (English %q) %v", c.action, viApplied, viLabel, label, err)
		}
	}
}
