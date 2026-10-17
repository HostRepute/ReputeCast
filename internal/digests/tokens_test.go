package digests

import (
	"testing"
	"time"
)

const starterSubject = `{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %} from the Acme blog`

func issueWith(n int) IssueData {
	items := sampleItems()
	if n < len(items) {
		items = items[:n]
	}
	for len(items) < n {
		items = append(items, BlockItem{Title: "Another post"})
	}
	return IssueData{FeedTitle: "Acme engineering blog", Number: 3, Date: time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC), Items: items}
}

func TestSubjectWithItems(t *testing.T) {
	for n, want := range map[int]string{
		1: "A year of on-call in numbers from the Acme blog",
		2: "A year of on-call in numbers and 1 more from the Acme blog",
		6: "A year of on-call in numbers and 5 more from the Acme blog",
	} {
		got, err := RenderIssue(starterSubject, issueWith(n), false)
		if err != nil || got != want {
			t.Fatalf("%d items: %q %v", n, got, err)
		}
		if SaysZeroMore(got) {
			t.Fatalf("%d items flagged: %q", n, got)
		}
	}
	// Without the {% if %}, one item reads "and 0 more": the preview catches it.
	bare := `{{ issue.items.first.title }} and {{ issue.items.size | minus: 1 }} more from the Acme blog`
	got, _ := RenderIssue(bare, issueWith(1), false)
	if got != "A year of on-call in numbers and 0 more from the Acme blog" || !SaysZeroMore(got) {
		t.Fatalf("zero more: %q", got)
	}
	fixed, ok := WrapCount(bare)
	if !ok || fixed != starterSubject {
		t.Fatalf("wrap: %q %v", fixed, ok)
	}
	if _, ok := WrapCount(starterSubject); ok {
		t.Fatal("wrapped twice")
	}
}

func TestRenderIssueKeepsPersonalization(t *testing.T) {
	src := `Hi {{ first_name | default: "there" }}, issue {{ issue.number }} of {{ feed.title }} · {% if first_name %}Hello {{ first_name }}, {{ issue.items.size }} posts{% else %}Hello{% endif %}` +
		` {% for item in issue.items limit: 2 %}<a href="{{ item.url }}">{{ item.title }}</a> for {{ contact.email }}{% if forloop.last %}.{% endif %} {% endfor %}{% raw %}{{ literally }}{% endraw %} <a href="{{ unsubscribe_url }}">Unsubscribe</a>`
	got, err := RenderIssue(src, issueWith(3), true)
	if err != nil {
		t.Fatal(err)
	}
	want := `Hi {{ first_name | default: "there" }}, issue 3 of Acme engineering blog · {% if first_name %}Hello {{ first_name }}, 3 posts{% else %}Hello{% endif %}` +
		` <a href="https://blog.acme.io/on-call-in-numbers/">A year of on-call in numbers</a> for {{ contact.email }} <a href="https://blog.acme.io/testing-migrations/">How we test database migrations &amp; locks</a> for {{ contact.email }}. {% raw %}{{ literally }}{% endraw %} <a href="{{ unsubscribe_url }}">Unsubscribe</a>`
	if got != want {
		t.Fatalf("render:\n got %s\nwant %s", got, want)
	}
	// Entities around tokens, as the composer writes them in HTML.
	got, err = RenderIssue(`{{ issue.items.first.title | default: &quot;New posts&quot; }}`, issueWith(1), false)
	if err != nil || got != "A year of on-call in numbers" {
		t.Fatalf("entities: %q %v", got, err)
	}
}

func TestUsesFeedTokens(t *testing.T) {
	if !UsesFeedTokens(starterSubject) || UsesFeedTokens(`Hello {{ first_name }}`) || !UsesFeedTokens(`{% for p in issue.items %}{% endfor %}`) {
		t.Fatal("feed token detection")
	}
}
