package sponsors

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

func TestIssueSchedule(t *testing.T) {
	sl := store.SponsorSlot{FirstNumber: 142, FirstOn: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), EveryDays: 7}
	for _, c := range []struct {
		n    int
		want string
	}{{140, "2026-09-30"}, {141, "2026-10-07"}, {142, "2026-10-14"}, {143, "2026-10-21"}, {147, "2026-11-18"}} {
		if got := IssueOn(sl, c.n).Format("2006-01-02"); got != c.want {
			t.Errorf("issue %d on %s, want %s", c.n, got, c.want)
		}
	}
	for _, c := range []struct {
		day  string
		want int
	}{
		{"2026-10-15", 143}, {"2026-10-14", 142}, {"2026-10-13", 142}, {"2026-10-07", 141}, {"2026-10-06", 141},
		{"2026-10-01", 141}, {"2026-09-30", 140}, {"2026-10-21", 143}, {"2026-10-22", 144},
	} {
		d, _ := time.Parse("2006-01-02", c.day)
		if got := NextNumber(sl, d); got != c.want {
			t.Errorf("next issue on or after %s: #%d, want #%d", c.day, got, c.want)
		}
	}
	if n, ok := NumberOn(sl, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)); !ok || n != 143 {
		t.Errorf("issue on 21 Oct: #%d %v", n, ok)
	}
	if _, ok := NumberOn(sl, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)); ok {
		t.Error("no issue goes out on 22 Oct")
	}
}

func TestInitialsAndUTM(t *testing.T) {
	for name, want := range map[string]string{"Northwind Analytics": "NA", "Contoso Cloud": "CC", "Acme": "AC", "the best co.": "TB"} {
		if got := initials(name); got != want {
			t.Errorf("initials(%q) = %q, want %q", name, got, want)
		}
	}
	got := WithUTM("northwind-analytics.example/trial", SponsorUTMs{Source: "acme-weekly-digest", Medium: "email", Campaign: "141"})
	if got != "https://northwind-analytics.example/trial?utm_campaign=141&utm_medium=email&utm_source=acme-weekly-digest" {
		t.Errorf("link with UTM: %s", got)
	}
	ws := &store.Workspace{Name: "Acme Newsletter"}
	if u := defaultUTM(ws, "Weekly Digest", 143); u.Source != "acme-weekly-digest" || u.Medium != "email" || u.Campaign != "143" {
		t.Errorf("default UTM: %+v", u)
	}
	if k := sponsorLinkKey("https://Northwind-Analytics.example/trial/?utm_campaign=141"); k != "northwind-analytics.example/trial" {
		t.Errorf("link key: %s", k)
	}
}

func TestBlockRule(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column><mj-text>Story</mj-text></mj-column></mj-section>` +
		`<mj-section rc-sponsor="1" rc-visible='{"id":"sp1","show":"only","match":"all","conditions":[]}'><mj-column><mj-text>Ad</mj-text></mj-column></mj-section></mj-body></mjml>`
	if got := blockRule(src); got != "sp1" {
		t.Errorf("block rule: %q", got)
	}
	if got := blockRule(`<mjml><mj-body><mj-section rc-sponsor="1"><mj-column></mj-column></mj-section></mj-body></mjml>`); got != "" {
		t.Errorf("a block shown to everyone has no rule: %q", got)
	}
}
