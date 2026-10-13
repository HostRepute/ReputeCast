package liquid

import (
	"strings"
	"testing"
)

func TestRenderFallbacks(t *testing.T) {
	cases := []struct {
		src  string
		vars map[string]any
		want string
	}{
		{`Hi {{ first_name | default: "there" }},`, map[string]any{"first_name": "Sam"}, "Hi Sam,"},
		{`Hi {{ first_name | default: "there" }},`, map[string]any{"first_name": ""}, "Hi there,"},
		{`Hi {{ first_name | default: "there" }},`, map[string]any{}, "Hi there,"},
		// The composer's short form.
		{`Hi {{ first_name | "there" }},`, map[string]any{}, "Hi there,"},
		// Entities written by the editor inside the delimiters.
		{`Hi {{ first_name | default: &quot;there&quot; }},`, map[string]any{}, "Hi there,"},
		{`{% if plan == &quot;team&quot; %}Team{% else %}Other{% endif %}`, map[string]any{"plan": "team"}, "Team"},
		{`{{ contact.first_name }}`, Vars("sam@x.io", "Sam", "", nil, nil), "Sam"},
		{`<a href="{{ unsubscribe_url }}">Unsubscribe</a>`, Vars("sam@x.io", "Sam", "", nil, map[string]string{"unsubscribe_url": "https://x/u"}), `<a href="https://x/u">Unsubscribe</a>`},
		{`No tokens at all`, nil, "No tokens at all"},
	}
	for _, c := range cases {
		got, err := Render(c.src, c.vars)
		if err != nil {
			t.Fatalf("%q: %v", c.src, err)
		}
		if got != c.want {
			t.Fatalf("%q: got %q, want %q", c.src, got, c.want)
		}
	}
}

func TestValidateAndLimits(t *testing.T) {
	if p := Validate(`Hi {{ first_name | default: "there" }}`); len(p) != 0 {
		t.Fatalf("valid template: %+v", p)
	}
	p := Validate("Line one\n{% if plan %}never closed")
	if len(p) != 1 || !strings.Contains(p[0].Message, "Line 2") {
		t.Fatalf("unclosed tag: %+v", p)
	}
	if p := Validate(`{% for i in (1..5000) %}x{% endfor %}`); len(p) == 0 {
		t.Fatal("huge range must be rejected")
	}
	if _, err := Render(`{% include "secrets" %}`, nil); err == nil {
		t.Fatal("include must be unavailable")
	}
}

func TestUses(t *testing.T) {
	u := Uses(`Hi {{ first_name | default: "there" }} {{ plan | "free" }} {{ first_name }} <a href="{{ unsubscribe_url }}">x</a>`)
	if len(u) != 3 {
		t.Fatalf("uses: %+v", u)
	}
	if u[0].Key != "first_name" || !u[0].HasFallback || u[0].Fallback != "there" || u[0].Count != 2 {
		t.Fatalf("first_name: %+v", u[0])
	}
	if u[1].Key != "plan" || u[1].Fallback != "free" {
		t.Fatalf("plan: %+v", u[1])
	}
	if u[2].Key != "unsubscribe_url" || u[2].HasFallback {
		t.Fatalf("unsubscribe_url: %+v", u[2])
	}
}

func TestCatalog(t *testing.T) {
	c := Catalog([]CustomField{{Key: "plan", Label: "Plan", Type: "select"}})
	keys := []string{}
	for _, f := range c {
		keys = append(keys, f.Key)
	}
	if strings.Join(keys, ",") != "first_name,last_name,email,plan,unsubscribe_url,preferences_url,view_in_browser_url" {
		t.Fatalf("catalog: %v", keys)
	}
}
