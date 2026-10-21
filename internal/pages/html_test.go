package pages

import (
	"strings"
	"testing"
	"time"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"keeps blocks", `<section class="sp-sec lav" data-pb="text"><div class="sp-in"><h2 class="sp-h2">Hi</h2><p>Text <a href="/about">about</a></p></div></section>`,
			`<section class="sp-sec lav" data-pb="text"><div class="sp-in"><h2 class="sp-h2">Hi</h2><p>Text <a href="/about">about</a></p></div></section>`},
		{"drops scripts and styles", `<p style="color:red" onclick="x()">A<script>alert(1)</script></p><style>p{}</style>`, `<p>A</p>`},
		{"refuses javascript links", `<a href="javascript:alert(1)">x</a><a href="data:text/html,1">y</a>`, `<a>x</a><a>y</a>`},
		{"keeps https and mailto", `<a href="https://acme.io/x?y=1" target="_blank">x</a><a href="mailto:hi@acme.io">m</a>`,
			`<a href="https://acme.io/x?y=1" target="_blank" rel="noopener noreferrer">x</a><a href="mailto:hi@acme.io">m</a>`},
		{"protocol-relative is refused", `<img src="//evil.example/x.png" alt="x">`, `<img alt="x">`},
		{"unwraps unknown elements", `<marquee>Hello <b>you</b></marquee>`, `Hello <b>you</b>`},
		{"drops forms and frames", `<form action="/x"><input name="a"></form><iframe src="https://x"></iframe>ok`, `ok`},
		{"bad classes", `<div class="ok gjs-selected Bad;x"></div>`, `<div class="ok"></div>`},
		{"custom elements keep their attributes", `<rc-product product="0190e9b6-1a2b-7c3d-8e4f-5a6b7c8d9e0f" button="Buy for $29" price="1" offer="1" onclick="x" data-pb-id="buy"><b>inside</b></rc-product>`,
			`<rc-product product="0190e9b6-1a2b-7c3d-8e4f-5a6b7c8d9e0f" button="Buy for $29" price="1" offer="1" data-pb-id="buy"></rc-product>`},
		{"custom element bad attribute", `<rc-countdown at="soon" tz="Mars/Base" units="days" end="reveal"></rc-countdown>`, `<rc-countdown units="days" end="reveal"></rc-countdown>`},
		{"event link must be http", `<rc-event date="2026-10-29" start="17:00" tz="UTC" link="javascript:x"></rc-event>`, `<rc-event date="2026-10-29" start="17:00" tz="UTC"></rc-event>`},
		{"details open", `<details open><summary>Q</summary><p>A</p></details>`, `<details open=""><summary>Q</summary><p>A</p></details>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Sanitize(c.in)
			if got != c.want {
				t.Fatalf("Sanitize(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
			if again := Sanitize(got); again != got {
				t.Fatalf("not idempotent: %q → %q", got, again)
			}
		})
	}
}

func TestVideoOf(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ": "YouTube dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                "YouTube dQw4w9WgXcQ",
		"https://www.youtube.com/embed/dQw4w9WgXcQ":   "YouTube dQw4w9WgXcQ",
		"https://vimeo.com/76979871":                  "Vimeo 76979871",
		"https://player.vimeo.com/video/76979871":     "Vimeo 76979871",
	} {
		site, id, ok := VideoOf(raw)
		if !ok || site+" "+id != want {
			t.Fatalf("VideoOf(%s) = %s %s %v", raw, site, id, ok)
		}
	}
	if _, _, ok := VideoOf("https://evil.example/watch?v=x"); ok {
		t.Fatal("other sites are refused")
	}
}

func TestAnalyze(t *testing.T) {
	src := `<section data-pb="hero"><div><h1>Title</h1><img src="/media/x.png" alt="Cover"><img src="/media/y.png" alt="">
<rc-product product="0190e9b6-1a2b-7c3d-8e4f-5a6b7c8d9e0f"></rc-product></div></section>
<section data-pb="text"><p><a href="https://acme.io/beta-terms">Beta terms</a></p><rc-form form="0190e9b6-1a2b-7c3d-8e4f-5a6b7c8d9e01"></rc-form>
<rc-countdown at="2026-10-06T09:00" tz="America/New_York"></rc-countdown><rc-event date="2026-10-29" start="17:00" tz="UTC"></rc-event></section>`
	u := Analyze(src)
	if u.Blocks != 2 || u.FirstHeading != 1 || u.Images != 2 || len(u.ImagesNoAlt) != 1 || u.BuyButtons != 1 || len(u.Products) != 1 || len(u.Forms) != 1 {
		t.Fatalf("usage: %+v", u)
	}
	if len(u.Links) != 1 || u.Links[0].Href != "https://acme.io/beta-terms" || u.Links[0].Text != "Beta terms" {
		t.Fatalf("links: %+v", u.Links)
	}
	want := time.Date(2026, 10, 6, 13, 0, 0, 0, time.UTC)
	if len(u.Countdowns) != 1 || !u.Countdowns[0].At.Equal(want) {
		t.Fatalf("countdown: %+v", u.Countdowns)
	}
	if len(u.Events) != 1 || !u.Events[0].At.Equal(time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC)) {
		t.Fatalf("event: %+v", u.Events)
	}
	if !strings.Contains(Sanitize(src), "rc-countdown") {
		t.Fatal("custom elements survive")
	}
}
