package liquid_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/liquid"
)

var fields = []string{"first_name", "last_name", "email", "company"}

// C8: a tag that doesn't parse is a syntax issue with its line.
func TestCheckSyntax(t *testing.T) {
	got := liquid.Check("<p>Hi</p>\n<p>{% if first_name %}Hi {{ first_name }}</p>", fields)
	if len(got) != 1 || got[0].Kind != liquid.IssueSyntax || got[0].Message == "" {
		t.Fatalf("unclosed if: %+v", got)
	}
	if got := liquid.Check("Hi {% endif %}", fields); len(got) == 0 || got[0].Kind != liquid.IssueSyntax {
		t.Fatalf("stray endif: %+v", got)
	}
}

// C8: a field that doesn't exist is flagged even with a fallback, which
// every recipient would get.
func TestCheckUnknownFields(t *testing.T) {
	got := liquid.Check(`Hi {{ frist_name | default: "there" }}, {{ compayn }}`, fields)
	if len(got) != 2 {
		t.Fatalf("issues: %+v", got)
	}
	if got[0].Kind != liquid.IssueUnknownField || got[0].Field != "frist_name" || !got[0].HasFallback || got[0].Fallback != "there" {
		t.Errorf("typo with a fallback: %+v", got[0])
	}
	if got[1].Field != "compayn" || got[1].HasFallback {
		t.Errorf("typo without a fallback: %+v", got[1])
	}
}

// Fields that exist, contact.*, the system links, loop and assigned
// variables and the per-recipient poll links aren't flagged.
func TestCheckKnown(t *testing.T) {
	src := `{{ first_name | default: "there" }} {{ contact.company }} <a href="{{ unsubscribe_url }}">Unsubscribe</a>
{% assign greeting = "Hello" %}{{ greeting }}
{% for item in (1..3) %}{{ item }} {{ forloop.index }}{% endfor %}
{% capture note %}x{% endcapture %}{{ note }}
<a href="{{ rc_poll_ab12_1 }}">Yes</a>`
	if got := liquid.Check(src, fields); len(got) != 0 {
		t.Fatalf("flagged: %+v", got)
	}
}

// Without fields only the syntax is checked; plain text has no issues.
func TestCheckWithoutFields(t *testing.T) {
	if got := liquid.Check("Hi {{ anything }}", nil); len(got) != 0 {
		t.Fatalf("no fields: %+v", got)
	}
	if got := liquid.Check("No tags at all", fields); got == nil || len(got) != 0 {
		t.Fatalf("plain text: %#v", got)
	}
}
