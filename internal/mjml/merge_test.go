package mjml

import (
	"strings"
	"testing"
)

// CO-10: two edits of the same version merge when they changed different
// sections.

func mergeFixture(head, s1, s2, s3 string) string {
	h := ""
	if head != "" {
		h = `<mj-head>` + head + `</mj-head>`
	}
	return `<mjml>` + h + `<mj-body>
<mj-section><mj-column><mj-text>` + s1 + `</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-text>` + s2 + `</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-button href="https://acme.io/docs">` + s3 + `</mj-button></mj-column></mj-section>
</mj-body></mjml>`
}

const (
	arial   = `<mj-title>Hello</mj-title><mj-attributes><mj-all font-family="Arial"></mj-all></mj-attributes>`
	georgia = `<mj-title>Hello</mj-title><mj-attributes><mj-all font-family="Georgia"></mj-all></mj-attributes>`
)

func TestMergeSections(t *testing.T) {
	base := mergeFixture(arial, "Hero", "Feature flags", "Read the docs")

	// They changed the button, I changed the hero: both kept.
	theirs := mergeFixture(arial, "Hero", "Feature flags", "See the changelog")
	mine := mergeFixture(arial, "My hero", "Feature flags", "Read the docs")
	m, ok := MergeSections(base, theirs, mine)
	if !ok {
		t.Fatal("different sections should merge")
	}
	if !strings.Contains(m.MJML, "My hero") || !strings.Contains(m.MJML, "See the changelog") || strings.Contains(m.MJML, "Read the docs") {
		t.Fatalf("merged:\n%s", m.MJML)
	}
	if len(m.Theirs) != 1 || m.Theirs[0].Index != 2 || !strings.Contains(m.Theirs[0].MJML, "See the changelog") || m.Head != "" {
		t.Fatalf("theirs: %+v head %q", m.Theirs, m.Head)
	}
	if _, err := Parse(m.MJML); err != nil {
		t.Fatalf("merged document doesn't parse: %v", err)
	}
	// The title and preview follow the subject and preheader: left out,
	// SetHeadMeta puts them back.
	if strings.Contains(m.MJML, "mj-title") || !strings.Contains(m.MJML, `font-family="Arial"`) {
		t.Fatalf("head:\n%s", m.MJML)
	}

	// Both changed the same section differently: a person chooses.
	theirs2 := mergeFixture(arial, "Their hero", "Feature flags", "Read the docs")
	if _, ok := MergeSections(base, theirs2, mine); ok {
		t.Fatal("the same section changed on both sides must not merge")
	}
	// The same change on both sides isn't a conflict.
	if m, ok := MergeSections(base, mine, mine); !ok || !strings.Contains(m.MJML, "My hero") || len(m.Theirs) != 0 {
		t.Fatalf("same edit: %v %+v", ok, m)
	}
	// Titles that differ (other subjects) don't conflict.
	retitled := strings.Replace(theirs, "<mj-title>Hello</mj-title>", "<mj-title>Their subject</mj-title>", 1)
	if _, ok := MergeSections(base, retitled, mine); !ok {
		t.Fatal("a different title alone must not conflict")
	}

	// Their head (styles, fonts) comes along when mine left it.
	theirs3 := mergeFixture(georgia, "Hero", "Feature flags", "Read the docs")
	m, ok = MergeSections(base, theirs3, mine)
	if !ok || !strings.Contains(m.MJML, `font-family="Georgia"`) || strings.Contains(m.MJML, `font-family="Arial"`) || !strings.Contains(m.MJML, "My hero") ||
		m.Head != `<mj-attributes><mj-all font-family="Georgia"></mj-all></mj-attributes>` {
		t.Fatalf("head merge: %v %+v", ok, m)
	}
	// Even when mine has no head at all.
	bare := mergeFixture("", "Hero", "Feature flags", "Read the docs")
	m, ok = MergeSections(bare, mergeFixture(georgia, "Hero", "Feature flags", "Read the docs"), mergeFixture("", "My hero", "Feature flags", "Read the docs"))
	if !ok || !strings.HasPrefix(m.MJML, "<mjml>\n  <mj-head>") || !strings.Contains(m.MJML, "Georgia") {
		t.Fatalf("head into a document without one: %v\n%s", ok, m.MJML)
	}
	if _, err := Parse(m.MJML); err != nil {
		t.Fatalf("doesn't parse: %v", err)
	}
	// Both changed the head: a person chooses.
	mine2 := mergeFixture(strings.Replace(arial, "Arial", "Inter", 1), "My hero", "Feature flags", "Read the docs")
	if _, ok := MergeSections(base, theirs3, mine2); ok {
		t.Fatal("both heads changed must not merge")
	}

	// A section added, removed or moved: a person chooses.
	added := strings.Replace(theirs, "</mj-body>", `<mj-section><mj-column><mj-text>New</mj-text></mj-column></mj-section></mj-body>`, 1)
	if _, ok := MergeSections(base, added, mine); ok {
		t.Fatal("an added section must not merge")
	}
	// Anything that isn't MJML: a person chooses.
	if _, ok := MergeSections("", theirs, mine); ok {
		t.Fatal("an empty base must not merge")
	}
}
