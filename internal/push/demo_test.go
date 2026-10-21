package push_test

// Push in the demo (brief §4 "Web push"): Setup's numbers, the prompts'
// 30 days, the campaign's report and Sam Rivera's subscription.

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/agency"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/push"
)

const demoWS = "/workspaces/acme-newsletter"

func TestPushDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")

	var setup push.PushSetup
	linh.Do(http.MethodGet, demoWS+"/push/setup", nil).MustStatus(t, 200).JSON(t, &setup)
	sub := setup.Subscribers
	if !setup.On || setup.Keys == nil || setup.Keys.Contact != "mailto:support@acme.io" || setup.Keys.CreatedBy != "Marcus Chen" || setup.Cap != 2 {
		t.Fatalf("keys: %+v", setup.Keys)
	}
	if sub.Total != 4264 || sub.Linked != 2118 || sub.Anonymous != 4264-2118 || sub.Highlight == nil || sub.Highlight.Count != 612 ||
		sub.Highlight.Name != "Acme Summit 2026" {
		t.Fatalf("subscribers: %+v highlight %+v", sub, sub.Highlight)
	}
	browsers := map[string]int64{}
	for _, b := range sub.ByBrowser {
		browsers[b.Key] = b.Count
	}
	want := map[string]int64{"chrome": 1866, "chrome_android": 544, "edge": 512, "firefox": 401, "safari": 845, "safari_ios": 96}
	for k, v := range want {
		if browsers[k] != v {
			t.Fatalf("by browser: %v", browsers)
		}
	}
	if len(setup.Sites) != 2 || setup.Sites[0].Host != "news.acme.io" || setup.Sites[0].Subscribers != 2946 || setup.Sites[0].Worker.Status != push.WorkerBuiltIn ||
		setup.Sites[1].Host != "acme.io" || setup.Sites[1].Subscribers != 1318 || setup.Sites[1].Worker.Status != push.WorkerFound || setup.Sites[1].Worker.Version == nil ||
		*setup.Sites[1].Worker.Version != 1 || setup.Sites[1].SnippetSeenAt == nil {
		t.Fatalf("sites: %+v", setup.Sites)
	}
	if len(setup.Prompts) != 2 {
		t.Fatalf("prompts: %+v", setup.Prompts)
	}
	var bar, bell push.PushPromptCard
	for _, p := range setup.Prompts {
		if p.Config.Style == push.StyleBar {
			bar = p
		} else {
			bell = p
		}
	}
	if bar.Version != 3 || bar.Stats.Shown != 18240 || bar.Stats.Allowed != 4980 || bar.Stats.Granted != 4412 || bar.Stats.NotNow != 9104 || bar.Stats.Closed != 4156 ||
		bar.Stats.AllowedPct != 27.3 || bar.Stats.GrantedPct != 88.6 || bar.Stats.IOSSubscribers != 96 || bar.Stats.IOSSteps != 1312 || bar.Stats.IOSAdded != 141 {
		t.Fatalf("slide-in bar: %+v", bar)
	}
	if bell.Config.Style != push.StyleBell || bell.Stats.BellClicked != 1980 || bell.Stats.Allowed != 1540 || bell.Stats.Granted != 1352 || bell.Stats.AllowedPct != 77.8 ||
		bell.Stats.GrantedPct != 87.8 || !bell.Stats.SinceSnippet {
		t.Fatalf("bell: %+v", bell)
	}
	// The designer's consent record: Sam Rivera's subscription.
	var pr push.PushPrompt
	linh.Do(http.MethodGet, demoWS+"/push/prompts/"+bar.FormID, nil).MustStatus(t, 200).JSON(t, &pr)
	if pr.Consent == nil || pr.LiveVersion != 3 || pr.Live == nil {
		t.Fatalf("designer: %+v", pr)
	}

	// The campaign's report.
	var campaigns struct {
		Items []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Channel string `json:"channel"`
		} `json:"items"`
	}
	linh.Do(http.MethodGet, demoWS+"/campaigns?channel=push&pageSize=50", nil).MustStatus(t, 200).JSON(t, &campaigns)
	id := ""
	for _, c := range campaigns.Items {
		if c.Name == "New post: Feature flags at scale" && c.Channel == "push" {
			id = c.ID
		}
	}
	if id == "" {
		t.Fatalf("no push campaign in %+v", campaigns.Items)
	}
	var rep push.PushReport
	linh.Do(http.MethodGet, demoWS+"/push-campaigns/"+id+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Planned != 2931 || rep.Sent != 2912 || rep.Expired != 19 || rep.Shown != 2804 || rep.ShownPct != 96.3 || rep.Clicks != 421 || rep.ClicksPct != 14.5 ||
		rep.Unsubscribed != 7 || rep.SentBy != "Sofia Rossi" {
		t.Fatalf("report: %+v", rep)
	}
	got := map[string][3]int64{}
	for _, b := range rep.ByBrowser {
		got[b.Browser] = [3]int64{b.Sent, b.Shown, b.Clicks}
	}
	for k, v := range map[string][3]int64{"chrome": {1276, 1248, 168}, "safari": {577, 569, 92}, "chrome_android": {372, 352, 71}, "edge": {350, 341, 44},
		"firefox": {274, 232, 31}, "safari_ios": {63, 62, 15}} {
		if got[k] != v {
			t.Fatalf("by browser: %v", got)
		}
	}
	// The composer reads it back.
	var c push.PushCampaign
	linh.Do(http.MethodGet, demoWS+"/push-campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	if c.Title != "Feature flags at scale" || len(c.Buttons) != 2 || c.Image == nil || c.Image.Width != 1200 || c.Image.Height != 600 || c.TTLMinutes != 240 ||
		c.Tag != "new-post" {
		t.Fatalf("composer: %+v", c)
	}

	// Agencies count each client's push and WhatsApp messages from the
	// channel message log (X23): the notifications and the WhatsApp
	// messages are rows there, on the day they went.
	ctx := context.Background()
	for _, ch := range []struct {
		name, kind string
		min        int64
	}{{"push", "campaign", rep.Sent}, {"whatsapp", "", 1}} {
		var wsID uuid.UUID
		var day time.Time
		if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, max(m.sent_at) FROM workspaces w
			JOIN channel_messages m ON m.workspace_id = w.id AND m.channel = $1 AND m.sent_at IS NOT NULL AND ($2 = '' OR m.kind = $2)
			WHERE w.slug = 'acme-newsletter' GROUP BY w.id`, ch.name, ch.kind).Scan(&wsID, &day); err != nil {
			t.Fatalf("%s: %v", ch.name, err)
		}
		rows, err := agency.MessageChannel{Name: ch.name}.Day(ctx, env.App.Pool, wsID, day)
		if err != nil {
			t.Fatal(err)
		}
		var units int64
		for _, r := range rows {
			if r.Dimension == "provider" {
				units += r.Units
			}
		}
		if units < ch.min {
			t.Fatalf("agency usage of %s on %s: %d, want at least %d (%+v)", ch.name, day.UTC().Format(time.DateOnly), units, ch.min, rows)
		}
	}
}
