package i18n

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func mustFormat(t *testing.T, l Lang, msg string, args Args) string {
	t.Helper()
	out, err := Format(l, msg, args)
	if err != nil {
		t.Fatalf("format %q: %v", msg, err)
	}
	return out
}

func TestFormatArgumentsPluralsAndSelect(t *testing.T) {
	en, vi := For(English), For(Vietnamese)
	cases := []struct {
		lang Lang
		msg  string
		args Args
		want string
	}{
		{en, "Hi {name},", Args{"name": "Aisha"}, "Hi Aisha,"},
		{en, "{count, plural, one {# message} other {# messages}} sent", Args{"count": 1}, "1 message sent"},
		{en, "{count, plural, one {# message} other {# messages}} sent", Args{"count": 48213}, "48,213 messages sent"},
		{vi, "Đã gửi {count, plural, other {# thư}}", Args{"count": 48213}, "Đã gửi 48.213 thư"},
		{en, "{n, plural, =0 {No Zaps} one {One Zap} other {# Zaps}}", Args{"n": 0}, "No Zaps"},
		{en, "{n, plural, offset:1 =0 {nobody} one {{name}} other {{name} and # others}}", Args{"n": 3, "name": "Linh"}, "Linh and 2 others"},
		{en, "{role, select, owner {Owner} other {Member}}", Args{"role": "owner"}, "Owner"},
		{en, "{role, select, owner {Owner} other {Member}}", Args{"role": "x"}, "Member"},
		{en, "{kind, select, a {{n, plural, one {# A} other {# As}}} other {none}}", Args{"kind": "a", "n": 2}, "2 As"},
		{en, "It''s {n, number} or '{literal}'", Args{"n": 1234.5}, "It's 1,234.5 or {literal}"},
		{en, "Don't change {x}", Args{"x": "it"}, "Don't change it"},
		{en, "{rate, number, ::percent .0}", Args{"rate": 0.991}, "99.1%"},
		{For(Vietnamese), "{rate, number, ::percent .0}", Args{"rate": 0.991}, "99,1%"},
		{en, "{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}", Args{"n": 22}, "22nd"},
		{en, "Sent to <b>{name}</b>", Args{"name": "Sam"}, "Sent to Sam"},
		{en, "a # b", Args{}, "a # b"},
		{en, "{missing}", Args{}, "{missing}"},
	}
	for _, c := range cases {
		if got := mustFormat(t, c.lang, c.msg, c.args); got != c.want {
			t.Errorf("%q → %q, want %q", c.msg, got, c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, bad := range []string{
		"{", "}", "{name", "{n, plural, one {x}}", "{n, select, a {x}}", "<b>x", "x</b>", "{n, wat}", "{n, plural, one {x} one {y} other {z}}",
	} {
		if err := Validate(bad); err == nil {
			t.Errorf("%q parsed, want an error", bad)
		}
	}
}

func TestPseudoLocale(t *testing.T) {
	SetPseudo(true)
	defer SetPseudo(false)
	fsys := fstest.MapFS{
		"messages/en.json": {Data: []byte(`{"schedule":"Schedule send to {count} recipients","home":"Home of ReputeCast","sent":"Sent to <b>{name}</b>"}`)},
	}
	c, err := Load(fsys, "pseudotest")
	if err != nil {
		t.Fatal(err)
	}
	p := Lang{Locale: Pseudo}
	// The brief's example: names and values keep their spelling.
	if got := c.M("schedule").Text(p, Args{"count": "31,604"}); got != "⟦Šçĥéðûļé šéñð ţö 31,604 ŕéçîþîéñţš ·············⟧" {
		t.Errorf("schedule: %q", got)
	}
	if got := c.M("home").Text(p, nil); got != "⟦Ĥömé öƒ ReputeCast ······⟧" {
		t.Errorf("home: %q", got)
	}
	if got := c.M("sent").HTML(p, Args{"name": "<Sam>"}, nil); got != "⟦Šéñţ ţö <b>&lt;Sam&gt;</b> ····⟧" {
		t.Errorf("html: %q", got)
	}
	if got := accent("Edit A/B SES"); got != "Éðîţ A/B SES" {
		t.Errorf("accent: %q", got)
	}
}

func TestCatalogFallbackAndChecks(t *testing.T) {
	fsys := fstest.MapFS{
		"messages/en.json": {Data: []byte(`{"title":"Transactional","sent":{"message":"{count, plural, one {# message} other {# messages}} sent","note":"a count"},"only":"Only in English"}`)},
		"messages/vi.json": {Data: []byte(`{"title":"Giao dịch","sent":"Đã gửi {count, plural, other {# thư}}"}`)},
	}
	c, err := Load(fsys, "cattest")
	if err != nil {
		t.Fatal(err)
	}
	vi := For(Vietnamese)
	if got := c.M("title").Text(vi, nil); got != "Giao dịch" {
		t.Errorf("title: %q", got)
	}
	if got := c.M("sent").Text(vi, Args{"count": 1}); got != "Đã gửi 1 thư" {
		t.Errorf("sent: %q", got)
	}
	// A missing translation falls back to English, with English plurals.
	if got := c.M("only").Text(vi, nil); got != "Only in English" {
		t.Errorf("only: %q", got)
	}
	if m := c.Missing(Vietnamese); len(m) != 1 || m[0] != "only" {
		t.Errorf("missing: %v", m)
	}
	if c.M("title").ID() != "cattest.title" {
		t.Errorf("id %s", c.M("title").ID())
	}
	defer func() {
		if recover() == nil {
			t.Error("M with an unknown key didn't panic")
		}
	}()
	c.M("nope")
}

func TestCatalogRejectsBadTranslations(t *testing.T) {
	for name, vi := range map[string]string{
		"unknown key":      `{"nope":"x"}`,
		"unknown argument": `{"a":"Xin chào {nam}"}`,
		"broken syntax":    `{"a":"{name"}`,
	} {
		fsys := fstest.MapFS{
			"messages/en.json": {Data: []byte(`{"a":"Hello {name}"}`)},
			"messages/vi.json": {Data: []byte(vi)},
		}
		if _, err := Load(fsys, "bad"); err == nil {
			t.Errorf("%s: loaded, want an error", name)
		}
	}
}

func TestHTMLEscapesTextAndValues(t *testing.T) {
	fsys := fstest.MapFS{"messages/en.json": {Data: []byte(`{"a":"<b>{inviter}</b> invited you to {ws} & more. <link>Accept</link>"}`)}}
	c, err := Load(fsys, "htmltest")
	if err != nil {
		t.Fatal(err)
	}
	got := c.M("a").HTML(Default, Args{"inviter": "Linh <script>", "ws": HTML("<i>Acme</i>")}, Tags{"link": func(in string) string { return `<a href="https://x">` + in + `</a>` }})
	want := `<b>Linh &lt;script&gt;</b> invited you to <i>Acme</i> &amp; more. <a href="https://x">Accept</a>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if txt := c.M("a").Text(Default, Args{"inviter": "Linh", "ws": HTML("<i>Acme</i>")}); txt != "Linh invited you to Acme & more. Accept" {
		t.Errorf("text: %q", txt)
	}
}

func TestNegotiate(t *testing.T) {
	cases := []struct {
		header string
		want   Locale
		ok     bool
	}{
		{"vi-VN,vi;q=0.9,en-US;q=0.8", Vietnamese, true},
		{"fr-FR,fr;q=0.9,vi;q=0.5", Vietnamese, true},
		{"en-GB", English, true},
		{"de-DE", English, false},
		{"", English, false},
		{"en-XA", English, true},
	}
	for _, c := range cases {
		if got, ok := Negotiate(c.header); got != c.want || ok != c.ok {
			t.Errorf("Negotiate(%q) = %s, %v; want %s, %v", c.header, got, ok, c.want, c.ok)
		}
	}
	SetPseudo(true)
	defer SetPseudo(false)
	if got, ok := Negotiate("en-XA"); got != Pseudo || !ok {
		t.Errorf("pseudo on: %s %v", got, ok)
	}
	if got, _ := Negotiate("en-US,en;q=0.9"); got != English {
		t.Errorf("en-US with the pseudo-locale on: %s", got)
	}
}

func TestParseLocaleAndResolve(t *testing.T) {
	if l, ok := ParseLocale("vi-VN"); !ok || l != Vietnamese {
		t.Errorf("vi-VN: %s %v", l, ok)
	}
	if _, ok := ParseLocale("en-XA"); ok {
		t.Error("the pseudo-locale parsed while it's off")
	}
	cases := []struct {
		ml, mr, dl, dr string
		want           Lang
	}{
		{"", "", "en", "standard", Lang{English, Standard}},
		{"", "", "vi", "standard", Lang{Vietnamese, Standard}},
		{"vi", "", "en", "standard", Lang{Vietnamese, "vi-VN"}},
		{"en", "en-US", "en", "standard", Lang{English, "en-US"}},
		{"", "de-DE", "en", "standard", Lang{English, "de-DE"}},
		{"xx", "", "", "", Lang{English, Standard}},
	}
	for _, c := range cases {
		if got := Resolve(c.ml, c.mr, c.dl, c.dr); got != c.want {
			t.Errorf("Resolve(%q,%q,%q,%q) = %+v, want %+v", c.ml, c.mr, c.dl, c.dr, got, c.want)
		}
	}
	ctx := WithLang(context.Background(), Lang{Locale: Vietnamese})
	if got := LangFrom(ctx); got.Region != "vi-VN" {
		t.Errorf("WithLang fills the region: %+v", got)
	}
	if LangFrom(context.Background()) != Default {
		t.Error("LangFrom without a language isn't Default")
	}
}

// TestRegionVectors checks the vectors web/src/i18n/regions.test.ts checks too.
func TestRegionVectors(t *testing.T) {
	b, err := os.ReadFile("testdata/regions.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		At       time.Time                    `json:"at"`
		PM       time.Time                    `json:"pm"`
		TimeZone string                       `json:"timeZone"`
		Regions  map[string]map[string]string `json:"regions"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	loc, err := time.LoadLocation(v.TimeZone)
	if err != nil {
		t.Fatal(err)
	}
	at, pm := v.At.In(loc), v.PM.In(loc)
	if len(v.Regions) != len(Regions()) {
		t.Errorf("vectors for %d regions, the package has %d", len(v.Regions), len(Regions()))
	}
	for code, want := range v.Regions {
		r := Region(code)
		if !r.Valid() {
			t.Errorf("%s isn't a region", code)
			continue
		}
		weekStart := "monday"
		if r.WeekStartsSunday() {
			weekStart = "sunday"
		}
		got := map[string]string{
			"name": r.Name(), "date": r.Date(at), "shortDate": r.ShortDate(at), "weekdayDate": r.WeekdayDate(at),
			"longDate": r.LongDate(at), "dayMonth": r.DayMonth(at), "time": r.Time(at, ""), "time12": r.Time(at, H12),
			"timePM": r.Time(pm, ""), "timePM12": r.Time(pm, H12),
			"int": r.Int(48213), "int4": r.Int(4821), "int7": r.Int(1234567), "percent": r.Percent(99.1, 1),
			"money": r.Money(2.84, 2), "money2": r.Money(18.21, 2), "hourCycle": string(r.HourCycle()), "weekStart": weekStart,
		}
		for k, w := range want {
			if got[k] != w {
				t.Errorf("%s %s = %q, want %q", code, k, got[k], w)
			}
		}
	}
}

func TestNumbersInMessagesFollowTheRegion(t *testing.T) {
	de := Lang{Locale: English, Region: "de-DE"}
	if got := mustFormat(t, de, "{n} contacts", Args{"n": 48213}); got != "48.213 contacts" {
		t.Errorf("de: %q", got)
	}
	if got := mustFormat(t, Default, "Expires {d, date}", Args{"d": time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)}); got != "Expires Oct 22, 2026" {
		t.Errorf("date: %q", got)
	}
	if strings.Contains(mustFormat(t, Default, "{n}", Args{"n": -5}), "−") {
		t.Error("negative numbers use a hyphen-minus")
	}
}
