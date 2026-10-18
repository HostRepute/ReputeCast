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

func TestTagLookKeepsTheBlocksStructure(t *testing.T) {
	// The Beta corner (X11's #143): styled divs and a button-like link.
	src := `<div style="border-left:3px solid #0F766E"><div style="font-size:12px;color:#0F766E">Beta corner</div><div style="font-size:19px">Flag rules by region are in beta</div><a href="https://acme.io/beta" style="background-color:#0F766E">Try region rules</a></div>`
	answer := `<div style="color:red" onclick="x()"><div>Beta-Ecke</div><div class="x">Flag-Regeln nach Region sind in der Beta</div><a href="https://acme.io/beta">Regionsregeln testen</a></div>`
	want := `<div style="border-left:3px solid #0F766E"><div style="font-size:12px;color:#0F766E">Beta-Ecke</div><div style="font-size:19px">Flag-Regeln nach Region sind in der Beta</div><a href="https://acme.io/beta" style="background-color:#0F766E">Regionsregeln testen</a></div>`
	if got := linkAttrs(src, tagLook(src, sanitize(answer))); got != want {
		t.Errorf("the Beta corner:\n got %q\nwant %q", got, want)
	}
	if got := plain(sanitize(answer)); got != "Beta-Ecke Flag-Regeln nach Region sind in der Beta Regionsregeln testen" {
		t.Errorf("as text: %q", got)
	}
	// Tags of a name keep their order; a tag the source doesn't have stays bare.
	src = `<b>Hi</b>, read <b style="color:#0B1220">this</b>.`
	cases := []struct{ name, answer, want string }{
		{"same tags", `<b>Hallo</b>, lies <b>das</b>.`, `<b>Hallo</b>, lies <b style="color:#0B1220">das</b>.`},
		{"a tag added", `<b>Hallo</b>, <i>lies</i> <b>das</b>.`, `<b>Hallo</b>, <i>lies</i> <b style="color:#0B1220">das</b>.`},
		{"more tags than the source", `<b>a</b> <b>b</b> <b>c</b>`, `<b>a</b> <b style="color:#0B1220">b</b> <b>c</b>`},
		{"the model's attributes", `<b class="x">Hallo</b> <b style="color:red">das</b>`, `<b>Hallo</b> <b style="color:#0B1220">das</b>`},
	}
	for _, c := range cases {
		if got := tagLook(src, sanitize(c.answer)); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
