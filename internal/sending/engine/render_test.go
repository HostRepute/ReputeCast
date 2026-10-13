package engine

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const sampleHTML = `<html><body>
<a href="https://acme.io/blog" style="color:#5B6472">Blog</a> ·
<a href='https://docs.acme.io/flags?utm=x&amp;y=1'>Read <b>the</b> docs</a>
<a href="https://acme.io/u/{{ contact.id }}">Your account</a>
<a href="{{ unsubscribe_url }}">Unsubscribe</a> · <a href="{{ preferences_url }}">Email preferences</a>
<a href="mailto:hello@acme.io">Write to us</a> <a href="#top">Top</a>
<p>Hi {{ first_name | "there" }},</p>
</body></html>`

func TestPrepareRewritesLinksAndAddsPixel(t *testing.T) {
	tpl, err := Prepare(Content{Subject: "Hello {{ first_name | default: \"there\" }}", HTML: sampleHTML}, Tracking{Opens: true, Clicks: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(tpl.Links) != 3 {
		t.Fatalf("%d tracked links, want 3 (unsubscribe, preferences, mailto and anchors stay): %+v", len(tpl.Links), tpl.Links)
	}
	if tpl.Links[1].URL != "https://docs.acme.io/flags?utm=x&y=1" || tpl.Links[1].Label != "Read the docs" {
		t.Fatalf("link 1 = %+v", tpl.Links[1])
	}
	vars := map[string]any{"first_name": "Ana", "contact": map[string]any{"id": "c1"}, "unsubscribe_url": "https://x/p/acme/unsubscribe/T",
		"preferences_url": "https://x/p/acme/preferences/P", OpenVar: "https://x/t/o/OPEN", LinkVar(0): "https://x/t/c/L0", LinkVar(1): "https://x/t/c/L1", LinkVar(2): "https://x/t/c/L2"}
	out, err := tpl.Render(vars, true)
	if err != nil {
		t.Fatal(err)
	}
	if out.Subject != "Hello Ana" {
		t.Fatalf("subject %q", out.Subject)
	}
	for _, want := range []string{`href="https://x/t/c/L0"`, `href="https://x/t/c/L1"`, `href="https://x/t/c/L2"`, `href="https://x/p/acme/unsubscribe/T"`,
		`href="mailto:hello@acme.io"`, `href="#top"`, `src="https://x/t/o/OPEN"`, "Hi Ana,", `style="color:#5B6472"`} {
		if !strings.Contains(out.HTML, want) {
			t.Errorf("rendered HTML lacks %s:\n%s", want, out.HTML)
		}
	}
	if strings.Index(out.HTML, "t/o/OPEN") > strings.Index(out.HTML, "</body>") {
		t.Error("the pixel must come before </body>")
	}
	plain, err := tpl.Render(vars, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.HTML, "/t/c/") || strings.Contains(plain.HTML, "/t/o/") || !strings.Contains(plain.HTML, `href="https://acme.io/blog"`) {
		t.Fatalf("the untracked variant (provider-side tracking kept) must keep the original links:\n%s", plain.HTML)
	}
	if !strings.Contains(out.Text, "Hi Ana") {
		t.Fatalf("plain text %q", out.Text)
	}
}

func TestPrepareWithoutClickTracking(t *testing.T) {
	tpl, err := Prepare(Content{Subject: "s", HTML: sampleHTML}, Tracking{Opens: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(tpl.Links) != 0 {
		t.Fatalf("links tracked with click tracking off: %+v", tpl.Links)
	}
}

func TestPrepareRejectsBrokenLiquid(t *testing.T) {
	if _, err := Prepare(Content{Subject: "Hi", HTML: "<p>{% if vip %}VIP</p>"}, Tracking{}); err == nil {
		t.Fatal("broken Liquid must fail before anything is sent")
	}
}

func TestBulkHeaders(t *testing.T) {
	h := bulkHeaders("https://reputecast.acme.io/p/acme-newsletter/unsubscribe/TOKEN", "TOKEN", "digest@news.acme.io", feedbackID("0199ab12-3456", "acme-newsletter", "broadcast"))
	if h["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
		t.Fatalf("List-Unsubscribe-Post %q", h["List-Unsubscribe-Post"])
	}
	want := "<https://reputecast.acme.io/p/acme-newsletter/unsubscribe/TOKEN>, <mailto:digest@news.acme.io?subject=unsubscribe%20TOKEN>"
	if h["List-Unsubscribe"] != want {
		t.Fatalf("List-Unsubscribe %q\nwant %q", h["List-Unsubscribe"], want)
	}
	if h["Feedback-ID"] != "0199ab12:acme-newsletter:broadcast:reputecast" {
		t.Fatalf("Feedback-ID %q", h["Feedback-ID"])
	}
}

func TestMaskAddress(t *testing.T) {
	cases := map[string]string{
		"ana.kowalski@gmail.com": "a.k···@gmail.com",
		"rwang@outlook.com":      "r···@outlook.com",
		"m_t@acme-partner.co":    "m_t···@acme-partner.co",
		"x@icloud.com":           "x···@icloud.com",
		"not-an-address":         "···",
	}
	for in, want := range cases {
		if got := MaskAddress(in); got != want {
			t.Errorf("MaskAddress(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseThresholds(t *testing.T) {
	th := ParseThresholds(json.RawMessage(`{"bounce":1.5,"complaint":0.08,"scope":"campaign"}`), nil)
	if th.Bounce != 1.5 || th.Complaint != 0.08 || th.MailboxComplaint != 0.08 || th.Deferral != 5 || th.APIErrors != 5 || th.Scope != "campaign" {
		t.Fatalf("%+v", th)
	}
	if d := ParseThresholds(nil, nil); d != DefaultThresholds {
		t.Fatalf("defaults %+v", d)
	}
	if th := ParseThresholds(json.RawMessage(`{"bounce":-1,"scope":"everything"}`), nil); th.Bounce != 2 || th.Scope != "route" {
		t.Fatalf("invalid values must keep defaults: %+v", th)
	}
	// The workspace's defaults (Settings › Providers spells them its own
	// way), then the campaign's (Review & send).
	ws := json.RawMessage(`{"hard_bounce":3.0,"complaint":0.2,"mailbox_complaint":0.3,"provider_error_rate":10,"scope":"campaign","min_sample":500}`)
	if th := ParseThresholds(nil, ws); th.Bounce != 3 || th.Complaint != 0.2 || th.MailboxComplaint != 0.3 || th.APIErrors != 10 || th.Scope != "campaign" {
		t.Fatalf("workspace defaults: %+v", th)
	}
	campaign := json.RawMessage(`{"bounce":1.5,"complaint":0.1,"provider_errors":4,"scope":"route"}`)
	if th := ParseThresholds(campaign, ws); th.Bounce != 1.5 || th.Complaint != 0.1 || th.MailboxComplaint != 0.3 || th.APIErrors != 4 || th.Scope != "route" {
		t.Fatalf("campaign over workspace: %+v", th)
	}
}

func TestSendThrottles(t *testing.T) {
	got := sendThrottles(json.RawMessage(`{"throttles":{"gmail":0,"outlook":8,"yahoo":0},"suppression_sync":"15m"}`), map[string]float64{"icloud": 2})
	want := map[string]float64{"outlook": 8, "icloud": 2}
	if len(got) != len(want) || got["outlook"] != 8 || got["icloud"] != 2 {
		t.Fatalf("throttles %v, want %v (0 = no limit)", got, want)
	}
	if d := sendThrottles(nil, nil); d["outlook"] != 5 || d["yahoo"] != 4 || d["icloud"] != 3 || len(d) != 3 {
		t.Fatalf("defaults %v", d)
	}
}

func TestNextAtHour(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC)
	for hour, want := range map[int]time.Time{
		9:  now,                                            // within the hour: at once
		14: time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC), // later today
		7:  time.Date(2026, 10, 15, 7, 0, 0, 0, time.UTC),  // tomorrow
	} {
		if got := nextAtHour(now, hour); !got.Equal(want) {
			t.Errorf("nextAtHour(%d) = %v, want %v", hour, got, want)
		}
	}
}

func TestBackoff(t *testing.T) {
	want := []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, 60 * time.Second, 60 * time.Second}
	for i, w := range want {
		if got := backoff(i + 1); got != w {
			t.Errorf("backoff(%d) = %v, want %v", i+1, got, w)
		}
	}
}

func TestAllocateRespectsThrottlesAndHolds(t *testing.T) {
	pid := uuid.New()
	rs := &runState{
		routes:    map[uuid.UUID]*runRoute{},
		throttles: map[string]float64{"outlook": 5},
		tokens:    map[string]float64{"outlook": 5},
		holds:     map[string]hold{"yahoo": {Since: time.Now()}},
		queue: map[queueKey]queueCell{
			{pid, "gmail"}:   {queued: 900},
			{pid, "outlook"}: {queued: 900},
			{pid, "yahoo"}:   {queued: 300},
			{pid, "icloud"}:  {queued: 100, waiting: 100},
		},
	}
	r := &runRoute{}
	r.provider.ID = pid
	got := (&Engine{}).allocate(rs, r, 14)
	if got["yahoo"] != 0 {
		t.Fatalf("held Yahoo got %d", got["yahoo"])
	}
	if got["icloud"] != 0 {
		t.Fatalf("iCloud, all waiting for retries, got %d", got["icloud"])
	}
	if got["outlook"] > 5 {
		t.Fatalf("Outlook got %d over its 5 /s throttle", got["outlook"])
	}
	if got["gmail"]+got["outlook"] != 14 {
		t.Fatalf("allocated %v, want all 14", got)
	}
}

func TestAssignVariant(t *testing.T) {
	ab := &abTest{Kind: "subject", TestPercent: 20, Variants: []abVariant{{Key: "A"}, {Key: "B"}}}
	held, sample := 0, map[string]int{}
	for range 5000 {
		v, status, _ := assignVariant(ab, uuid.New(), nil)
		if status == "held" {
			held++
			continue
		}
		sample[v]++
	}
	if held < 3800 || held > 4200 {
		t.Fatalf("%d of 5,000 wait for the winner, want about 4,000", held)
	}
	if sample["A"] < 400 || sample["B"] < 400 {
		t.Fatalf("sample split %v", sample)
	}
	id := uuid.New()
	v1, s1, _ := assignVariant(ab, id, nil)
	v2, s2, _ := assignVariant(ab, id, nil)
	if v1 != v2 || s1 != s2 {
		t.Fatal("assignment must be stable for a contact")
	}
}

func TestOverNeedsEvidence(t *testing.T) {
	for _, tc := range []struct {
		events, base int64
		limit        float64
		want         bool
	}{
		{3, 1923, 0.1, false},   // 0.16% on a small sample: chance
		{7, 3980, 0.1, true},    // LiveMonitorPaused: 0.18%
		{3, 1284, 0.1, false},   // 0.23%, but three events
		{5, 2000, 0.1, true},    // 0.25%
		{4, 2000, 0.1, false},   // under the least number of events
		{55, 50000, 0.1, false}, // 0.11% on a large sample: not sure yet
		{60, 50000, 0.1, true},  // 0.12%
		{25, 1000, 2.0, false},  // bounces: 2.5%
		{27, 1000, 2.0, true},   // 2.7%
		{9, 10000, 0.1, false},  // under the limit
	} {
		if got := over(tc.events, tc.base, tc.limit, minEvents); got != tc.want {
			t.Errorf("over(%d, %d, %.1f%%) = %v, want %v", tc.events, tc.base, tc.limit, got, tc.want)
		}
	}
}
