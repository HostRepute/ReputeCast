package digests

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the starter writes in English stays exactly as it always was, and
// another language writes its own words around the same Liquid.
func TestStarterTextByLanguage(t *testing.T) {
	en, vi := i18n.For(i18n.English), i18n.For(i18n.Vietnamese)
	at := time.Date(2026, time.October, 22, 9, 30, 0, 0, time.UTC)
	cases := []struct{ name, got, want string }{
		{"subject", starterSubjectIn(en), `{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %} from {{ feed.title }}`},
		{"subject in Vietnamese", starterSubjectIn(vi), `{{ issue.items.first.title }}{% if issue.items.size > 1 %} và {{ issue.items.size | minus: 1 }} bài khác{% endif %} từ {{ feed.title }}`},
		{"preheader", starterPreheaderIn(en, "blog.acme.io"), "New on blog.acme.io · issue {{ issue.number }}"},
		{"preheader without a host", starterPreheaderIn(en, ""), "New on the blog · issue {{ issue.number }}"},
		{"preheader in Vietnamese", starterPreheaderIn(vi, "blog.acme.io"), "Bài mới trên blog.acme.io · số {{ issue.number }}"},
		{"name", starterNameIn(en, "blog.acme.io"), "blog.acme.io digest"},
		{"name without a host", starterNameIn(en, ""), "Blog digest"},
		{"name in Vietnamese", starterNameIn(vi, "blog.acme.io"), "Bản tin định kỳ blog.acme.io"},
		{"a post's date", emailDate(en, at, datePost), "Thursday 22 October 2026"},
		{"a card's date", emailDate(en, at, dateCard), "Thu 22 Oct"},
		{"the issue's date", emailDate(en, at, dateIssue), "Thursday 22 October"},
		{"no language is English", emailDate(i18n.Lang{}, at, datePost), "Thursday 22 October 2026"},
		{"a post's date in Vietnamese", emailDate(vi, at, datePost), "Thứ Năm, 22 tháng 10, 2026"},
		{"a card's date in Vietnamese", emailDate(vi, at, dateCard), "Th 5, 22 thg 10"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.name, c.got, c.want)
		}
	}
}
