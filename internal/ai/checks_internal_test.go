package ai

import "testing"

func TestLinkAttrsKeepsTheSourcesLook(t *testing.T) {
	src := `Hi {{ first_name }}, read <a style="color:#0F766E;" href="https://docs.acme.io/a" target="_blank">the docs</a> or <a class="x" href="https://acme.io/b">this</a>.`
	cases := []struct{ name, got, want string }{
		{"same address", `Lies <a href="https://docs.acme.io/a">die Doku</a>.`, `Lies <a href="https://docs.acme.io/a" style="color:#0F766E;" target="_blank">die Doku</a>.`},
		{"two links", `<a href="https://acme.io/b">das</a> · <a href="https://docs.acme.io/a">Doku</a>`, `<a href="https://acme.io/b" class="x">das</a> · <a href="https://docs.acme.io/a" style="color:#0F766E;" target="_blank">Doku</a>`},
		{"an address the source doesn't have", `<a href="https://other.example">x</a>`, `<a href="https://other.example">x</a>`},
		{"no links", `Hallo {{ first_name }}`, `Hallo {{ first_name }}`},
	}
	for _, c := range cases {
		if got := linkAttrs(src, sanitize(c.got)); got != c.want {
			t.Errorf("%s: linkAttrs = %q, want %q", c.name, got, c.want)
		}
	}
	// A suggestion's own attributes never survive sanitize, so the source's are the only ones.
	if got := linkAttrs(src, sanitize(`<a href="https://docs.acme.io/a" onclick="x()" style="color:red">y</a>`)); got != `<a href="https://docs.acme.io/a" style="color:#0F766E;" target="_blank">y</a>` {
		t.Errorf("a suggestion's attributes: %q", got)
	}
}
