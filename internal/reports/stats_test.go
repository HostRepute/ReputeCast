package reports

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

func TestBounceTable(t *testing.T) {
	ses, postal := uuid.New(), uuid.New()
	rows := []store.ReportBounceCodesRow{
		{BounceClass: "mailbox_unknown", Type: "hard_bounce", ProviderID: &ses, ProviderName: "Amazon SES", Adapter: "ses", OriginalType: "Permanent/General", OriginalCode: "550 5.1.1", N: 145, Recipients: 145, Suppressed: 145},
		{BounceClass: "mailbox_unknown", Type: "hard_bounce", ProviderID: &postal, ProviderName: "Postal", Adapter: "postal", OriginalType: "HardFail", OriginalCode: "550 5.1.1", N: 26, Recipients: 26, Suppressed: 26},
		{BounceClass: "temporary", Type: "soft_bounce", ProviderID: &postal, ProviderName: "Postal", Adapter: "postal", OriginalCode: "421 4.7.0 TSS04", N: 8, Recipients: 8},
		{BounceClass: "reputation", Type: "hard_bounce", ProviderID: &postal, ProviderName: "Postal", Adapter: "postal", OriginalCode: "550 5.7.1", N: 3, Recipients: 3, Suppressed: 1},
		{BounceClass: "policy_block", Type: "hard_bounce", ProviderID: &ses, ProviderName: "Amazon SES", Adapter: "ses", OriginalType: "Permanent/Suppressed", N: 4, Recipients: 4},
		{BounceClass: "", Type: "soft_bounce", ProviderID: &ses, ProviderName: "Amazon SES", Adapter: "ses", OriginalType: "Undetermined", N: 2},
	}
	tbl := bounceTable(i18n.Default, rows)
	if tbl.Total != 188 || len(tbl.Rows) != 4 {
		t.Fatalf("table: %+v", tbl)
	}
	want := []struct {
		typ, state  string
		count, left int64
		first       string
	}{
		{"unknown_user", "auto", 171, 0, "SES Permanent/General 550 5.1.1"},
		{"deferral_expired", "retry", 8, 8, "Postal 421 4.7.0 TSS04"},
		{"blocked", "add", 7, 6, "SES Permanent/Suppressed"},
		{"other", "review", 2, 0, "SES Undetermined"},
	}
	for i, w := range want {
		r := tbl.Rows[i]
		if r.Type != w.typ || r.Suppression != w.state || r.Count != w.count || r.Unsuppressed != w.left || r.Codes[0].CodeText() != w.first {
			t.Errorf("row %d: %+v (codes %v)", i, r, r.Codes)
		}
	}
}

func TestUnsubscribes(t *testing.T) {
	u := unsubscribes(i18n.Default, 112, []store.ReportUnsubscribeReasonsRow{
		{Reason: "too_frequent", N: 41}, {Reason: "not_relevant", N: 33}, {Reason: "never_signed_up", N: 9}, {Reason: "other", N: 7},
		{Reason: "too_frequent", ChoseFewer: true, N: 28}, {Reason: "one_click", N: 5},
	})
	got := []string{}
	for _, r := range u.Reasons {
		got = append(got, r.Key)
	}
	if strings.Join(got, ",") != "too_frequent,not_relevant,none,never_signed_up,other" || u.Reasons[2].Count != 22 || u.ChoseFewer != 28 || u.Reasons[0].Share != 36.6 {
		t.Fatalf("reasons: %+v", u)
	}
	// More answers than counted unsubscribes (counters still catching up).
	if u := unsubscribes(i18n.Default, 3, []store.ReportUnsubscribeReasonsRow{{Reason: "spam", N: 5}}); u.Total != 5 || len(u.Reasons) != 1 {
		t.Fatalf("catch-up: %+v", u)
	}
}

func TestClickMap(t *testing.T) {
	// Clicks count every human click; the map counts people per link.
	rows := []store.ReportLinksRow{
		{ID: uuid.New(), Url: "https://docs.acme.io/flags", Label: "Read the docs", Position: 0, Clicks: 450, UniqueClicks: 380},
		{ID: uuid.New(), Url: "https://acme.io/changelog", Label: "", Position: 1, Clicks: 260, UniqueClicks: 240},
		{ID: uuid.New(), Url: "https://acme.io/blog", Label: "Blog", Position: 2, Clicks: 330, UniqueClicks: 300},
		{ID: uuid.New(), Url: "https://acme.io/about", Label: "About", Position: 3, Clicks: 41, UniqueClicks: 40, BotClicks: 9},
		{ID: uuid.New(), Url: "https://acme.io/jobs", Label: "Jobs", Position: 4, Clicks: 44, UniqueClicks: 40, BotClicks: 9},
	}
	m := clickMap(i18n.Default, "<a href='x'>x</a>", rows, nil)
	if !m.Available || m.Total != 1000 || m.BotTotal != 18 || m.Links[1].Label != "acme.io/changelog" || !m.Links[0].Hot || !m.Links[1].Hot || m.Links[3].Hot ||
		!m.Links[3].Grouped || m.Links[0].Share != 38 || len(m.Rows) != 0 {
		t.Fatalf("map: %+v", m)
	}
	if m.Other == nil || m.Other.Links != 2 || m.Other.Share != 8 || m.Other.Clicks != 80 || strings.Join(m.Other.Labels, ",") != "About,Jobs" {
		t.Fatalf("other: %+v", m.Other)
	}
	// The two small links sit on one line at the end: one footer entry.
	m = clickMap(i18n.Default, "<a href='x'>x</a>", rows, [][]int{{3, 4}, {0, 1}})
	if len(m.Rows) != 1 || m.Rows[0].Place != "footer" || m.Rows[0].Clicks != 80 || m.Rows[0].BotClicks != 18 || m.Rows[0].Share != 8 ||
		strings.Join(m.Rows[0].Labels, ",") != "About,Jobs" || m.Other != nil || m.Links[3].Grouped {
		t.Fatalf("rows: %+v %+v", m.Rows, m.Other)
	}
	if clickMap(i18n.Default, "  ", rows, nil).Available {
		t.Fatal("missing content must not render")
	}
}

func TestEngagement(t *testing.T) {
	rows := []store.ReportHourlyRow{{Hour: 0, OpenedHuman: 50, Clicked: 5}, {Hour: 5, OpenedHuman: 12}, {Hour: 30, OpenedHuman: 38, OpenedMachine: 4}}
	e := engagement(rows, 1000, 72)
	if !e.Available || len(e.Points) != 73 || e.Points[72].HumanOpens != 100 || e.Points[6].HumanOpens != 62 || e.HumanWithin6h == nil || *e.HumanWithin6h != 62 {
		t.Fatalf("engagement: %+v", e)
	}
	// A live report only draws the hours so far.
	if e := engagement(rows[:1], 1000, 3); len(e.Points) != 4 {
		t.Fatalf("live points: %d", len(e.Points))
	}
	if e := engagement(nil, 1000, 72); e.Available || len(e.Points) != 0 {
		t.Fatalf("no data: %+v", e)
	}
}

func TestLimiter(t *testing.T) {
	l := &Limiter{Requests: 3, Misses: 2, Window: time.Minute, MissWindow: 10 * time.Minute}
	now := time.Now()
	for i := 0; i < 3; i++ {
		if !l.Allow("a", now) {
			t.Fatalf("request %d refused", i)
		}
	}
	if l.Allow("a", now) {
		t.Fatal("4th request in a minute allowed")
	}
	if !l.Allow("b", now) {
		t.Fatal("other clients are separate")
	}
	if !l.Allow("a", now.Add(61*time.Second)) {
		t.Fatal("next window refused")
	}
	l.Miss("c", now)
	l.Miss("c", now)
	if l.Allow("c", now) {
		t.Fatal("too many unknown links allowed")
	}
	if !l.Allow("c", now.Add(11*time.Minute)) {
		t.Fatal("misses expire")
	}
}

func TestFileStem(t *testing.T) {
	if got := fileStem("Weekly Digest #142"); got != "weekly-digest-142" {
		t.Fatal(got)
	}
	if got := fileStem("!!!"); got != "campaign" {
		t.Fatal(got)
	}
	if got := csvSafe("=HYPERLINK(1)", "ok")[0]; got != "'=HYPERLINK(1)" {
		t.Fatal(got)
	}
}
