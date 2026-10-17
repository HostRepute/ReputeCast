package growth_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/growth"
)

// Audience › Growth on the demo (GrowthSources, GrowthStates): everyone who
// sees contacts reads it; exports hold totals only and are logged.
func TestGrowthAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("daniel@acme.io")
	var p growth.GrowthPage
	c.Do("GET", "/workspaces/acme-newsletter/growth", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Tiles.Joined != 1756 || p.Tiles.Left != 472 || p.Tiles.Imports != 9509 || len(p.Tiles.ImportItems) != 2 || len(p.Sources) != 10 {
		t.Fatalf("tiles %+v, %d sources", p.Tiles, len(p.Sources))
	}
	var api, mc *growth.GrowthSourceRow
	for i := range p.Sources {
		switch p.Sources[i].Source.Name {
		case "Website signup form":
			api = &p.Sources[i]
		case "Mailchimp":
			mc = &p.Sources[i]
		}
	}
	if api == nil || api.BounceFlag == nil || api.Source.KeyPrefix != "rc_live_8f2c…a91d" || mc == nil || mc.ComplaintFlag == nil || mc.Joined != nil {
		t.Fatalf("flagged sources: %+v %+v", api, mc)
	}
	// The chart's four series and Other.
	if len(p.Chart.Series) != 5 || p.Chart.Series[4].Joined != 95 || p.Chart.Series[4].Left != 368 || len(p.Chart.Days) != 30 {
		t.Fatalf("chart: %+v", p.Chart.Series)
	}
	var imports growth.GrowthPage
	c.Do("GET", "/workspaces/acme-newsletter/growth?imports=true", nil).MustStatus(t, 200).JSON(t, &imports)
	if imports.Tiles.Joined != 11265 || imports.Tiles.Net != 10793 {
		t.Fatalf("with imports: %+v", imports.Tiles)
	}
	// The contacts filter's sources, and the Zapier contacts.
	var src growth.GrowthSourceList
	c.Do("GET", "/workspaces/acme-newsletter/growth/sources", nil).MustStatus(t, 200).JSON(t, &src)
	zapier := ""
	for _, s := range src.Items {
		if s.Source.Kind == growth.KindIntegration {
			zapier = s.Source.Key
			if s.Contacts != 38 {
				t.Fatalf("Zapier: %d contacts", s.Contacts)
			}
		}
	}
	var list struct{ Total int64 }
	c.Do("GET", "/workspaces/acme-newsletter/contacts?growthSource="+url.QueryEscape(zapier), nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 38 {
		t.Fatalf("contacts from Zapier: %d", list.Total)
	}
	c.Do("GET", "/workspaces/acme-newsletter/contacts?utmCampaign=flags-ga", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 272 {
		t.Fatalf("utm_campaign flags-ga: %d", list.Total)
	}
	// The export: totals, never an address, and an audit entry.
	r := c.Do("GET", "/workspaces/acme-newsletter/growth/export", nil).MustStatus(t, 200)
	body := string(r.Body)
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") || !strings.Contains(body, "Newsletter signup,form,731,39,692") || strings.Contains(body, "@") {
		t.Fatalf("export:\n%s", body)
	}
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'growth.exported' AND actor_name = 'Daniel Park'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("export audit: %d %v", n, err)
	}
	// An unknown list is a 404; a period that ends before it starts, a 422.
	c.Do("GET", "/workspaces/acme-newsletter/growth?list=0199c0de-0000-7000-8000-000000000000", nil).MustStatus(t, 404)
	c.Do("GET", "/workspaces/acme-newsletter/growth?period=custom&from=2026-10-10&to=2026-10-01", nil).MustStatus(t, 422)
}
