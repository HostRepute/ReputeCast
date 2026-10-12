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
