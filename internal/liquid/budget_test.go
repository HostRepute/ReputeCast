package liquid

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	osteele "github.com/osteele/liquid"
)

// Templates anyone who can preview an email could send, each of which once
// pinned a core or grew the heap until the process was killed. Each must
// now fail fast, with ErrTooMuchWork (or ErrTooLarge when the output fills
// up first), and leave the heap small.
func TestBudgetStopsRunawayTemplates(t *testing.T) {
	for name, src := range map[string]string{
		// A range read from a variable isn't a literal, so the 1,000-item
		// check never saw it.
		"variable range": `{% assign n = 50000000 %}{% for i in (1..n) %}{% endfor %}`,
		// Three loops of 1,000 pass the literal check one by one.
		"nested loops": `{% for a in (1..1000) %}{% for b in (1..1000) %}{% for c in (1..1000) %}{% endfor %}{% endfor %}{% endfor %}`,
		// The same loops with a body, inside an if (whose first branch the
		// parser copies when it compiles it).
		"nested loops in if": `{% if true %}{% for a in (1..1000) %}{% for b in (1..1000) %}{% for c in (1..1000) %}x{% endfor %}{% endfor %}{% endfor %}{% endif %}`,
		// A string that doubles on every pass is never written to the
		// output, so the 4 MB output limit never saw it.
		"append doubling":  `{% assign s = "x" %}{% for i in (1..40) %}{% assign s = s | append: s %}{% endfor %}{{ s | size }}`,
		"capture doubling": `{% assign s = "x" %}{% for i in (1..40) %}{% capture s %}{{ s }}{{ s }}{% endcapture %}{% endfor %}{{ s | size }}`,
		"replace doubling": `{% assign s = "x" %}{% for i in (1..40) %}{% assign s = s | replace: "x", "xx" %}{% endfor %}`,
		"concat doubling":  `{% assign a = "x" | split: "," %}{% for i in (1..40) %}{% assign a = a | concat: a %}{% endfor %}`,
		// A filter would turn the range into fifty million strings before
		// it ran.
		"huge range filtered": `{% assign n = 50000000 %}{{ (1..n) | join: "," }}`,
		// Many large copies, each small enough on its own.
		"many copies": `{% assign s = "xxxxxxxxxxxxxxxx" %}{% for i in (1..17) %}{% assign s = s | append: s %}{% endfor %}{% for i in (1..1000) %}{% assign t = s | append: "y" %}{% endfor %}`,
		// tablerow is a loop too.
		"tablerow": `{% assign n = 50000000 %}{% tablerow i in (1..n) %}{% endtablerow %}`,
	} {
		t.Run(name, func(t *testing.T) {
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			out, err := Render(src, map[string]any{})
			took := time.Since(start)
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			if !errors.Is(err, ErrTooMuchWork) && !errors.Is(err, ErrTooLarge) {
				t.Fatalf("rendered %d bytes, err %v; want ErrTooMuchWork", len(out), err)
			}
			if took > MaxRenderTime+time.Second {
				t.Errorf("took %v", took)
			}
			if grown := int64(after.TotalAlloc) - int64(before.TotalAlloc); grown > 1<<30 {
				t.Errorf("allocated %d MB", grown>>20)
			}
		})
	}
}

// The time limit catches what the counts don't: contains on a long string
// costs no filter work.
func TestBudgetStopsAtTheDeadline(t *testing.T) {
	defer func(d time.Duration) { MaxRenderTime = d }(MaxRenderTime)
	MaxRenderTime = 50 * time.Millisecond
	src := `{% assign s = "x" %}{% for i in (1..22) %}{% assign s = s | append: s %}{% endfor %}` +
		`{% for i in (1..1000) %}{% for j in (1..150) %}{% if s contains "y" %}{% endif %}{% endfor %}{% endfor %}`
	start := time.Now()
	if _, err := Render(src, nil); !errors.Is(err, ErrTooMuchWork) {
		t.Fatalf("err %v; want ErrTooMuchWork", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("took %v", took)
	}
}

// A template that overwrites the budget's variable only fails.
func TestBudgetCantBeTamperedWith(t *testing.T) {
	for _, src := range []string{
		`{% assign __rc_render = 1 %}{% for i in (1..3) %}{{ i }}{% endfor %}`,
		`{% capture __rc_render %}x{% endcapture %}{{ "a" | append: "b" }}`,
		`{% for __rc_render in (1..3) %}{{ __rc_render }}{% endfor %}`,
	} {
		if out, err := Render(src, nil); err == nil {
			t.Errorf("%s rendered %q; want an error", src, out)
		}
	}
}

// Templates an email really uses render exactly as osteele/liquid renders
// them on its own: the budget's ticks and the bounded capture change no
// output, whitespace control included.
func TestBudgetKeepsOutput(t *testing.T) {
	plain := osteele.NewEngine()
	vars := map[string]any{
		"first_name": "Sam",
		"plan":       "team",
		"items":      []any{"a", "b", "c"},
		"empty":      []any{},
		"n":          3,
		"price":      12.5,
		"contact":    map[string]any{"first_name": "Sam", "tags": []any{"vip", "beta"}},
	}
	for _, src := range []string{
		`Hi {{ first_name | default: "there" }},`,
		`{% if plan == "team" %}Team{% elsif plan == "pro" %}Pro{% else %}Other{% endif %}`,
		`{% unless plan == "free" %}Paid{% else %}Free{% endunless %}`,
		`{% case plan %}{% when "free" %}F{% when "team", "pro" %}T{% else %}?{% endcase %}`,
		`{% for i in items %}{{ forloop.index }}:{{ i | upcase }}{% unless forloop.last %}, {% endunless %}{% endfor %}`,
		`{% for i in empty %}x{% else %}none{% endfor %}`,
		`{% for i in (1..n) %}{% if i == 2 %}{% continue %}{% endif %}{{ i }}{% endfor %}`,
		`{% for i in (1..10) %}{% if i > 3 %}{% break %}{% endif %}{{ i }}{% endfor %}`,
		`{% for i in items limit: 2 offset: 1 %}{{ i }}{% endfor %}{% for i in items reversed %}{{ i }}{% endfor %}`,
		`{% for i in items %}{% cycle "odd", "even" %}{% endfor %}`,
		`{% tablerow i in items cols: 2 %}{{ i }}{% endtablerow %}`,
		`{% capture greeting %}Hello {{ first_name }}!{% endcapture %}[{{ greeting }}]`,
		`{% if true %}{% capture x %}{{ n | plus: 1 }}{% endcapture %}{% endif %}{{ x }}`,
		"{%- assign a = 1 -%}\n  [{{- a -}}]  \n{%- if true -%}\n  yes\n{%- endif -%}\n",
		"<ul>\n{% for i in items -%}\n  <li>{{ i }}</li>\n{% endfor -%}\n</ul>",
		`{% raw %}{{ not_rendered }}{% endraw %}{% comment %}gone{% endcomment %}`,
		`{{ price | times: 2 | round: 1 }} {{ "a,b" | split: "," | join: "+" }} {{ contact.tags | join: " " }} {{ items | size }}`,
		`{{ false | default: "x", allow_false: true }} {{ nil | default: "y" }}`,
		`{{ "now" | date: "%Y" | size }}`,
		`{% assign s = "abc" %}{{ s | slice: 1, 2 }}{{ s | replace_first: "a", "z" }}{{ s | prepend: ">" }}`,
	} {
		want, werr := plain.ParseAndRenderString(src, vars)
		got, err := Render(src, vars)
		if werr != nil {
			t.Fatalf("%q: osteele/liquid: %v", src, werr)
		}
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		if got != want {
			t.Errorf("%q:\n got %q\nwant %q", src, got, want)
		}
	}
}

// A parsed template is shared by every recipient of a send: renders run at
// once on several goroutines, each on its own budget.
func TestTemplateRendersConcurrently(t *testing.T) {
	tpl, err := Parse(`{% capture g %}Hi {{ first_name }}{% endcapture %}{% for i in (1..5) %}{{ g | append: "!" }}{% endfor %}`)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := strings.Repeat("x", i+1)
			out, err := tpl.Render(map[string]any{"first_name": name})
			if err != nil {
				t.Error(err)
				return
			}
			if want := strings.Repeat("Hi "+name+"!", 5); out != want {
				t.Errorf("got %q, want %q", out, want)
			}
		}()
	}
	wg.Wait()
}
