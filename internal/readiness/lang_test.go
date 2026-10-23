package readiness_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// stored is a result as a run stores it and reads it back.
func stored(t *testing.T, res readiness.ReadinessResult) readiness.ReadinessResult {
	t.Helper()
	checks, est, err := readiness.EncodeResult(res)
	if err != nil {
		t.Fatal(err)
	}
	var back readiness.ReadinessResult
	if err := json.Unmarshal(checks, &back); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(est, &back.Estimates); err != nil {
		t.Fatal(err)
	}
	return back
}

func check(t *testing.T, res readiness.ReadinessResult, key string) readiness.ReadinessCheck {
	t.Helper()
	for _, c := range res.Checks {
		if c.Key == key {
			return c
		}
	}
	t.Fatalf("no check %s", key)
	return readiness.ReadinessCheck{}
}

// A run stored once reads in each reader's language: Vietnamese for a
// Vietnamese reader, exactly the stored English for API clients.
func TestStoredRunReadsInTheReadersLanguage(t *testing.T) {
	res := readiness.Assemble(digest())
	back := stored(t, res)
	vi := back.In(i18n.For(i18n.Vietnamese))
	en := back.In(i18n.Default)

	un := check(t, vi, "audience.unengaged")
	if un.Title != "Người nhận có 412 liên hệ không tương tác (không mở thư trong 180 ngày)" {
		t.Errorf("title in Vietnamese: %q", un.Title)
	}
	if un.Ack != "Tôi hiểu 412 liên hệ không tương tác vẫn được gửi." || !strings.HasPrefix(un.Evidence, "1,3% người nhận") {
		t.Errorf("ack and evidence in Vietnamese: %q %q", un.Ack, un.Evidence)
	}
	if un.Fixes[0].Label != "Loại 412 liên hệ không tương tác" {
		t.Errorf("fix in Vietnamese: %q", un.Fixes[0].Label)
	}
	if c := check(t, vi, "content.size"); !strings.Contains(c.Title, "64 KB") || !strings.Contains(c.Title, "Gmail") || c.Title == check(t, en, "content.size").Title {
		t.Errorf("size in Vietnamese: %q", c.Title)
	}
	// Numbers and amounts follow the reader's region.
	if s := vi.Estimates.Route.Shares[0]; s.PriceLabel != "0,10\u00a0US$ mỗi 1.000 thư" || s.RateLabel != "14/giây" {
		t.Errorf("SES in Vietnamese: %q %q", s.PriceLabel, s.RateLabel)
	}
	if vi.Estimates.Route.SplitLabel != "Gửi hàng loạt 90 / 10" || vi.Estimates.Audience.Rows[1].Label != "Danh sách chặn gửi (dùng chung)" {
		t.Errorf("estimates in Vietnamese: %q %q", vi.Estimates.Route.SplitLabel, vi.Estimates.Audience.Rows[1].Label)
	}
	for _, g := range vi.Groups {
		if g.Key == "audience" && g.Title != "Chất lượng người nhận" {
			t.Errorf("group: %+v", g)
		}
	}

	// English readers with the standard formats read what was stored.
	for i, c := range res.Checks {
		got := en.Checks[i]
		if got.Title != c.Title || got.Detail != c.Detail || got.Evidence != c.Evidence || got.Ack != c.Ack || got.FlatTitle != c.FlatTitle || got.Short != c.Short {
			t.Errorf("%s in English: %+v, stored %+v", c.Key, got, c)
		}
	}
	if en.Estimates.Route.Shares[0].PriceLabel != "$0.10 per 1,000" {
		t.Errorf("price in English: %q", en.Estimates.Route.Shares[0].PriceLabel)
	}

	// What a reader gets carries no kept messages.
	b, _ := json.Marshal(vi)
	if strings.Contains(string(b), `"i18n"`) {
		t.Errorf("kept messages leak: %s", b)
	}
	// A warning and its key: the same in every language.
	if check(t, vi, "audience.unengaged").AckKey == "" || check(t, vi, "audience.unengaged").AckKey != check(t, en, "audience.unengaged").AckKey {
		t.Errorf("ack keys differ by language")
	}
}

// Runs stored before their texts were kept read in English for everyone.
func TestOldRunsReadInEnglish(t *testing.T) {
	old := []byte(`{"verdict":"warnings","checks":[{"key":"audience.unengaged","group":"audience","status":"warn","title":"412 unengaged contacts are in this audience","flatTitle":"Audience: 412 unengaged contacts included","ack":"I understand 412 unengaged contacts are included.","fixes":[{"kind":"apply","action":"exclude_sunset","label":"Exclude 412 unengaged"}]}],"groups":[{"key":"audience","title":"Audience quality","caption":"Weekly Digest · 31,988 contacts","status":"warn","passed":0,"total":1,"summary":"","running":""}]}`)
	var res readiness.ReadinessResult
	if err := json.Unmarshal(old, &res); err != nil {
		t.Fatal(err)
	}
	vi := res.In(i18n.For(i18n.Vietnamese))
	c := vi.Checks[0]
	if c.Title != "412 unengaged contacts are in this audience" || c.Fixes[0].Label != "Exclude 412 unengaged" || vi.Groups[0].Caption != "Weekly Digest · 31,988 contacts" {
		t.Errorf("old run: %+v %+v", c, vi.Groups[0])
	}
}

// An approval's snapshot copies a provider's status label and the route's
// split: they read in each reader's language from the message the run kept,
// or from what the snapshot copied when it says the same in English.
func TestCopiedRouteLabels(t *testing.T) {
	back := stored(t, readiness.Assemble(digest()))
	vi := i18n.For(i18n.Vietnamese)
	i := len(back.Estimates.Route.Shares) - 1
	s := back.Estimates.Route.Shares[i]
	if s.Status != "warming_up" || s.StatusLabel != "Warming up · day 9 of 21" {
		t.Fatalf("share: %+v", s)
	}
	kept := back.Estimates.ShareStatus(i)
	if got := readiness.ShareStatusIn(s.StatusLabel, kept, s.Status, 0, 0, vi); got != "Đang khởi động · ngày 9/21" {
		t.Errorf("kept, in Vietnamese: %q", got)
	}
	if got := readiness.ShareStatusIn(s.StatusLabel, kept, s.Status, 0, 0, i18n.Default); got != s.StatusLabel {
		t.Errorf("kept, in English: %q", got)
	}
	for _, c := range []struct {
		english, status string
		day, days       int
		want            string
	}{
		{"Warming up · day 9 of 21", "warming_up", 9, 21, "Đang khởi động · ngày 9/21"},
		{"Warming up", "warming_up", 0, 0, "Đang khởi động"},
		{"Healthy", "healthy", 0, 0, "Ổn định"},
		// Nothing says when it went down: the copy stays as it is.
		{"Down since 08:52", "down", 0, 0, "Down since 08:52"},
		{"Warming up · day 3 of 21", "warming_up", 9, 21, "Warming up · day 3 of 21"},
	} {
		if got := readiness.ShareStatusIn(c.english, nil, c.status, c.day, c.days, vi); got != c.want {
			t.Errorf("%q: %q, want %q", c.english, got, c.want)
		}
	}
	for english, want := range map[string]string{"Broadcasts 90 / 10": "Gửi hàng loạt 90 / 10", "Broadcasts": "Gửi hàng loạt", "SES 100": "SES 100"} {
		if got := readiness.SplitLabelIn(english, vi); got != want {
			t.Errorf("split %q: %q, want %q", english, got, want)
		}
		if got := readiness.SplitLabelIn(english, i18n.Default); got != english {
			t.Errorf("split %q in English: %q", english, got)
		}
	}
}

// MCP agents read the stored run without the kept messages, everything
// else as it was.
func TestWithoutTexts(t *testing.T) {
	checks, est, err := readiness.EncodeResult(readiness.Assemble(digest()))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []json.RawMessage{checks, est} {
		if !strings.Contains(string(raw), `"i18n"`) {
			t.Fatalf("nothing kept: %s", raw)
		}
		out := readiness.WithoutTexts(raw)
		if strings.Contains(string(out), `"i18n"`) {
			t.Errorf("still kept: %s", out)
		}
		var a, b any
		if json.Unmarshal(out, &b) != nil {
			t.Fatalf("invalid: %s", out)
		}
		_ = json.Unmarshal(raw, &a)
		strip(a)
		if x, y := marshal(a), marshal(b); x != y {
			t.Errorf("changed:\n%s\n%s", x, y)
		}
	}
	if got := readiness.WithoutTexts(json.RawMessage(`{"a":1}`)); string(got) != `{"a":1}` {
		t.Errorf("untouched: %s", got)
	}
}

func strip(v any) {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "i18n")
		for _, y := range x {
			strip(y)
		}
	case []any:
		for _, y := range x {
			strip(y)
		}
	}
}

func marshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
