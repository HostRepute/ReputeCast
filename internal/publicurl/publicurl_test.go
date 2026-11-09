package publicurl

import "testing"

func TestBuilder(t *testing.T) {
	b := New("https://reputecast.acme.io/")
	cases := map[string]string{
		b.Open("tok"):                           "https://reputecast.acme.io/t/o/tok",
		b.Click("tok"):                          "https://reputecast.acme.io/t/c/tok",
		b.Unsubscribe("acme-newsletter", "tok"): "https://reputecast.acme.io/p/acme-newsletter/unsubscribe/tok",
		b.Preferences("acme-newsletter", "tok"): "https://reputecast.acme.io/p/acme-newsletter/preferences/tok",
		b.Confirm("acme-newsletter", "tok"):     "https://reputecast.acme.io/p/acme-newsletter/confirm/tok",
		b.ViewOnline("acme-newsletter", "tok"):  "https://reputecast.acme.io/p/acme-newsletter/view/tok",
		b.Subscribe("acme-newsletter", ""):      "https://reputecast.acme.io/p/acme-newsletter/subscribe",
		b.Subscribe("acme-newsletter", "beta"):  "https://reputecast.acme.io/p/acme-newsletter/subscribe/beta",
		b.Archive("acme-newsletter"):            "https://reputecast.acme.io/p/acme-newsletter/archive",
		b.ArchiveIssue("acme-newsletter", "x"):  "https://reputecast.acme.io/p/acme-newsletter/archive/x",
		b.ArchiveFeed("acme-newsletter"):        "https://reputecast.acme.io/p/acme-newsletter/archive.rss",
		b.Report("tok"):                         "https://reputecast.acme.io/r/tok",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	if got := b.Subscribe("a b", "c/d"); got != "https://reputecast.acme.io/p/a%20b/subscribe/c%2Fd" {
		t.Errorf("path segments are not escaped: %q", got)
	}
}

func TestSourceFollowsChanges(t *testing.T) {
	src := NewSource("https://reputecast.acme.io/")
	b := From(src)
	if b.Report("tok") != "https://reputecast.acme.io/r/tok" {
		t.Fatalf("report: %q", b.Report("tok"))
	}
	if src.Set("https://reputecast.acme.io") {
		t.Fatal("the same URL without its slash is no change")
	}
	if !src.Set(" https://news.acme.io/ ") || b.Base() != "https://news.acme.io" || b.Open("x") != "https://news.acme.io/t/o/x" {
		t.Fatalf("after a change: %q", b.Base())
	}
	if (Builder{}).Base() != "" {
		t.Fatal("a zero Builder has no base")
	}
}

func TestEffective(t *testing.T) {
	for _, c := range []struct {
		saved, configured string
		locked            bool
		want              string
	}{
		{"https://news.acme.io/", "http://localhost:8080", false, "https://news.acme.io"},
		{"  ", "http://localhost:8080/", false, "http://localhost:8080"},
		{"", "", false, ""},
		// The server's operator pins the address (http.public_url_locked):
		// what an admin saved is ignored.
		{"https://news.acme.io", "https://reputecast.acme.io/", true, "https://reputecast.acme.io"},
		{"", "https://reputecast.acme.io", true, "https://reputecast.acme.io"},
	} {
		if got := Effective(c.saved, c.configured, c.locked); got != c.want {
			t.Errorf("Effective(%q, %q, %v) = %q, want %q", c.saved, c.configured, c.locked, got, c.want)
		}
	}
}

func TestSetting(t *testing.T) {
	for _, c := range []struct {
		saved, configured string
		locked            bool
		want              string
	}{
		{"https://news.acme.io", "http://localhost:8080", false, "https://news.acme.io"},
		// Nothing saved reads as nothing saved, as before.
		{"", "http://localhost:8080", false, ""},
		{"https://news.acme.io", "https://reputecast.acme.io/", true, "https://reputecast.acme.io"},
	} {
		if got := Setting(c.saved, c.configured, c.locked); got != c.want {
			t.Errorf("Setting(%q, %q, %v) = %q, want %q", c.saved, c.configured, c.locked, got, c.want)
		}
	}
}

func TestSame(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"https://news.acme.io", "https://NEWS.acme.io/", true},
		{" https://news.acme.io/rc/ ", "https://news.acme.io/rc", true},
		{"https://news.acme.io?x=1", "https://news.acme.io", true},
		{"https://news.acme.io", "http://news.acme.io", false},
		{"https://news.acme.io:8443", "https://news.acme.io", false},
		{"https://news.acme.io/rc", "https://news.acme.io", false},
		{"not a url", "https://news.acme.io", false},
		{"", "", true},
	} {
		if got := Same(c.a, c.b); got != c.want {
			t.Errorf("Same(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
