package httpgeneric

import (
	"encoding/json"
	"testing"
)

func TestRenderJSONContexts(t *testing.T) {
	data := map[string]any{
		"subject": `Say "hi" \ <now>` + "\n",
		"headers": map[string]any{"List-Unsubscribe": "<https://x.io/u>"},
		"empty":   "",
		"secrets": map[string]string{"api_key": "k3y"},
		"n":       json.Number("42"),
	}
	cases := []struct{ tpl, want string }{
		{`{"s": "{{ subject }}"}`, `{"s": "Say \"hi\" \\ <now>\n"}`},
		{`{"s": {{ subject }}}`, `{"s": "Say \"hi\" \\ <now>\n"}`},
		{`{"h": {{ headers }}}`, `{"h": {"List-Unsubscribe":"<https://x.io/u>"}}`},
		{`{"m": {{ missing }}}`, `{"m": null}`},
		{`{"n": {{ n }}, "ns": "{{ n }}"}`, `{"n": 42, "ns": "42"}`},
		{`{"d": "{{ empty | default: 'fallback' }}"}`, `{"d": "fallback"}`},
		{`{"a": "{{ "api:" | append: secrets.api_key | base64_encode }}"}`, `{"a": "YXBpOmszeQ=="}`},
		{`{"u": "{{ subject | strip | downcase | prepend: '>' }}"}`, `{"u": ">say \"hi\" \\ <now>"}`},
		{`{"j": "{{ headers | json }}"}`, `{"j": "{\"List-Unsubscribe\":\"<https://x.io/u>\"}"}`},
		{`{"lit": "\{{ not a placeholder"}`, `{"lit": "\{{ not a placeholder"}`},
	}
	for _, c := range cases {
		tpl, err := parseTemplate(c.tpl, modeJSON)
		if err != nil {
			t.Fatalf("%s: %v", c.tpl, err)
		}
		got, err := tpl.render(data)
		if err != nil {
			t.Fatalf("%s: %v", c.tpl, err)
		}
		if got != c.want {
			t.Fatalf("%s\n got  %s\n want %s", c.tpl, got, c.want)
		}
	}
}

func TestRenderURLAndText(t *testing.T) {
	data := map[string]any{"settings": map[string]string{"domain": "news/acme.io", "q": "a&b=c d"}, "v": "line\r\nInjected: 1"}
	u, _ := parseTemplate("/v3/{{ settings.domain }}/messages?tag={{ settings.q }}", modeURL)
	if got, _ := u.render(data); got != "/v3/news%2Facme.io/messages?tag=a%26b%3Dc+d" {
		t.Fatalf("url: %s", got)
	}
	h, _ := parseTemplate("Bearer {{ v }}", modeText)
	if got, _ := h.render(data); got != "Bearer line  Injected: 1" {
		t.Fatalf("header values can't break lines: %q", got)
	}
}

func TestParseTemplateErrors(t *testing.T) {
	for _, src := range []string{`{{ x `, `{{ x | nope }}`, `{{ x | default }}`, `{{ x | append: 'a', 'b' }}`, `{{ a..b }}`, `{{ 'open }}`, `{{ a$b }}`} {
		if _, err := parseTemplate(src, modeJSON); err == nil {
			t.Fatalf("%s must not parse", src)
		}
	}
}

func TestJSONPath(t *testing.T) {
	doc, err := decodeJSON([]byte(`{"data":{"id":"x-1","list":[{"a":1},{"a":2}],"odd key":true},"arr":[[1,2],[3]]}`))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path string
		want string
	}{
		{"$.data.id", `["x-1"]`},
		{"$['data']['odd key']", `[true]`},
		{`$["data"].list[1].a`, `[2]`},
		{"$.data.list[*].a", `[1,2]`},
		{"$.arr[*]", `[[1,2],[3]]`},
		{"$.data.missing", `null`},
		{"$.data.list[9]", `null`},
	}
	for _, c := range cases {
		p, err := parsePath(c.path)
		if err != nil {
			t.Fatalf("%s: %v", c.path, err)
		}
		b, _ := json.Marshal(p.all(doc))
		if string(b) != c.want {
			t.Fatalf("%s: got %s, want %s", c.path, b, c.want)
		}
	}
	for _, bad := range []string{"data.id", "$.", "$[x]", "$[1", "$..a", "$.a[?(@.x)]"} {
		if _, err := parsePath(bad); err == nil {
			t.Fatalf("%s must not parse", bad)
		}
	}
}
