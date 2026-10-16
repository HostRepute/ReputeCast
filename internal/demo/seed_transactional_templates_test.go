package demo_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The transactional templates the editor and publish boards show come out
// of the demo seed in their states: password-reset v3 matching its draft,
// welcome's draft with an undeclared and an unused variable, trial-ending's
// draft v2 breaking the 71 calls of the last 7 days, and usage-summary
// ready to publish v7.
func TestTransactionalTemplateSeed(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	const ws = "/workspaces/acme-newsletter/transactional/templates/"
	var tpl struct {
		LiveVersion  *int
		DraftChanges bool
		Category     struct{ Name, Kind string }
		Variables    []struct {
			Name      string
			Sensitive bool
		}
	}
	linh.Do("GET", ws+"password-reset", nil).MustStatus(t, 200).JSON(t, &tpl)
	if tpl.LiveVersion == nil || *tpl.LiveVersion != 3 || tpl.DraftChanges || tpl.Category.Name != "Account" || len(tpl.Variables) != 3 || !tpl.Variables[1].Sensitive {
		t.Fatalf("password-reset: %+v", tpl)
	}

	var checks struct {
		Variables struct {
			Undeclared []struct{ Name string }
			Unused     []string
		}
	}
	linh.Do("POST", ws+"welcome/checks", map[string]any{}).MustStatus(t, 200).JSON(t, &checks)
	if len(checks.Variables.Undeclared) != 1 || checks.Variables.Undeclared[0].Name != "company_name" || len(checks.Variables.Unused) != 1 || checks.Variables.Unused[0] != "plan_name" {
		t.Fatalf("welcome checks: %+v", checks)
	}

	var pv struct {
		NextVersion  int
		DraftSavedBy string
		Breaking     struct {
			Calls    int
			Breaking []struct{ Name string }
			Summary  string
		}
		TestRenders []struct {
			Source string
			OK     bool
		}
	}
	linh.Do("POST", ws+"trial-ending/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.NextVersion != 2 || pv.DraftSavedBy != "Daniel Park" || pv.Breaking.Calls != 71 || len(pv.Breaking.Breaking) != 1 || pv.Breaking.Breaking[0].Name != "trial_end_date" ||
		!strings.Contains(pv.Breaking.Summary, "none of the 71 calls") {
		t.Fatalf("trial-ending: %+v", pv)
	}
	last := pv.TestRenders[len(pv.TestRenders)-1]
	if last.Source != "last_call" || last.OK {
		t.Fatalf("the last real call fails as it is: %+v", pv.TestRenders)
	}

	linh.Do("POST", ws+"usage-summary/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.NextVersion != 7 || len(pv.Breaking.Breaking) != 0 {
		t.Fatalf("usage-summary: %+v", pv)
	}
}
