package liquid

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeData struct{ calls int }

func (f *fakeData) Roots() []string { return []string{"greeting"} }

func (f *fakeData) Vars(_ context.Context, _ uuid.UUID, rs []Recipient, roots []string) (map[uuid.UUID]map[string]any, error) {
	f.calls++
	out := map[uuid.UUID]map[string]any{}
	for _, r := range rs {
		out[r.ContactID] = map[string]any{"greeting": map[string]any{"text": "Hello " + r.ContactID.String()[:4]}}
	}
	return out, nil
}

func TestBlocks(t *testing.T) {
	RegisterBlock("test-hello", func(raw []byte) string { return `<p>{{ greeting.text }}` + string(raw) + `</p>` })
	src := `<mj-raw><!-- rc:test-hello {"a":{"b":1}} --></mj-raw><!-- rc:unknown {} --><!-- a comment -->`
	if !HasBlocks(src) {
		t.Fatal("no blocks found")
	}
	out := ExpandBlocks(src)
	if !strings.Contains(out, `<p>{{ greeting.text }}{"a":{"b":1}}</p>`) || !strings.Contains(out, "rc:unknown") || ExpandBlocks(out) != out {
		t.Fatalf("expanded: %s", out)
	}
	d := &fakeData{}
	if got := UsedRoots(d, "Greetings from greeting.text, outside Liquid"); len(got) != 0 {
		t.Fatalf("text outside Liquid used %v", got)
	}
	if got := UsedRoots(d, `{{ contact.greeting }}`); len(got) != 0 {
		t.Fatalf("a property of another variable used %v", got)
	}
	if got := UsedRoots(d, out); len(got) != 1 {
		t.Fatalf("used roots: %v", got)
	}
	if got := UsedRoots(d, `{% if greeting %}x{% endif %}`); len(got) != 1 {
		t.Fatalf("a tag's roots: %v", got)
	}
	a, b := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	vars := FillBatch(context.Background(), []RecipientData{d}, uuid.Nil, []Recipient{{ContactID: a}, {ContactID: b}}, out)
	if d.calls != 1 || len(vars) != 2 {
		t.Fatalf("one call for the page: %d calls, %d readers", d.calls, len(vars))
	}
	one := Fill(context.Background(), []RecipientData{d}, uuid.Nil, Recipient{ContactID: a}, "no liquid here", map[string]any{"x": 1})
	if d.calls != 1 || len(one) != 1 {
		t.Fatalf("a template that doesn't use the roots asked for them: %v", one)
	}
	html, err := Render(out, Merge(Vars("", "", "", nil, nil), vars[a]))
	if err != nil || !strings.Contains(html, "Hello "+a.String()[:4]) {
		t.Fatalf("render: %q %v", html, err)
	}
}
