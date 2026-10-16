package accessibility

import "testing"

func mustColor(t *testing.T, s string) Color {
	t.Helper()
	c, ok := ParseColor(s)
	if !ok {
		t.Fatalf("ParseColor(%q) failed", s)
	}
	return c
}

func TestParseColor(t *testing.T) {
	cases := map[string]string{
		"#fff":                     "#FFFFFF",
		"#5ebfb0":                  "#5EBFB0",
		"#5EBFB0 !important":       "#5EBFB0",
		"rgb(94, 191, 176)":        "#5EBFB0",
		"rgb(94 191 176 / 50%)":    "#5EBFB0",
		"white":                    "#FFFFFF",
		"hsl(170.7, 43.1%, 55.9%)": "#5EBFB0",
		"ReBeccaPurple":            "#663399",
	}
	for in, want := range cases {
		if got := mustColor(t, in).Hex(); got != want {
			t.Errorf("ParseColor(%q) = %s, want %s", in, got, want)
		}
	}
	for _, bad := range []string{"", "inherit", "currentColor", "url(x.png)", "#12", "rgb(1,2)"} {
		if _, ok := ParseColor(bad); ok {
			t.Errorf("ParseColor(%q) should fail", bad)
		}
	}
	if c := mustColor(t, "rgba(0,0,0,0.5)"); c.Opaque() || c.A != 0.5 {
		t.Errorf("alpha: %+v", c)
	}
	if c := mustColor(t, "#00000080"); c.Opaque() {
		t.Errorf("8-digit hex alpha: %+v", c)
	}
}

// The sample of the brief: white on #5EBFB0 is 2.197:1, shown rounded down.
func TestRatioRoundsDown(t *testing.T) {
	w := mustColor(t, "#FFFFFF")
	cases := []struct {
		bg   string
		want string
	}{
		{"#5EBFB0", "2.1:1"},
		{"#0F766E", "5.4:1"},
		{"#348377", "4.5:1"},
		{"#276259", "7.0:1"},
		{"#2B9A8C", "3.4:1"},
		{"#000000", "21.0:1"},
		{"#FFFFFF", "1.0:1"},
	}
	for _, c := range cases {
		if got := FormatRatio(Ratio(w, mustColor(t, c.bg))); got != c.want {
			t.Errorf("white on %s = %s, want %s", c.bg, got, c.want)
		}
	}
	// 4.49 never shows as 4.5 and never passes.
	if RoundDown(4.4999) != 4.4 || passes(4.4999, AANormal) {
		t.Errorf("4.4999 must show as 4.4 and fail")
	}
}

func TestNearest(t *testing.T) {
	w, bg := mustColor(t, "#FFFFFF"), mustColor(t, "#5EBFB0")
	aa, r, ok := Nearest(bg, w, AANormal)
	if !ok || aa.Hex() != "#348377" || FormatRatio(r) != "4.5:1" {
		t.Errorf("nearest AA = %s %s %v, want #348377 4.5:1", aa.Hex(), FormatRatio(r), ok)
	}
	aaa, r, ok := Nearest(bg, w, AAANormal)
	if !ok || aaa.Hex() != "#276259" || FormatRatio(r) != "7.0:1" {
		t.Errorf("nearest AAA = %s %s %v, want #276259 7.0:1", aaa.Hex(), FormatRatio(r), ok)
	}
	// A passing color is its own nearest.
	same, _, ok := Nearest(mustColor(t, "#0F766E"), w, AANormal)
	if !ok || same.Hex() != "#0F766E" {
		t.Errorf("passing color moved to %s", same.Hex())
	}
	// Light text on a light background gets darker text, not lighter.
	dark, _, ok := Nearest(mustColor(t, "#9CA3AF"), w, AANormal)
	if !ok || Luminance(dark) >= Luminance(mustColor(t, "#9CA3AF")) {
		t.Errorf("expected a darker gray, got %s", dark.Hex())
	}
	// AAA against mid gray can't be reached by any shade of white.
	if _, _, ok := Nearest(mustColor(t, "#FFFFFF"), mustColor(t, "#777777"), 21); ok {
		t.Errorf("21:1 against gray should be impossible")
	}
}

func TestLarge(t *testing.T) {
	if !Large(24, false) || !Large(18.67, true) || Large(18.67, false) || Large(18, true) {
		t.Error("large text thresholds")
	}
	if Needs(true) != 3 || Needs(false) != 4.5 || NeedsAAA(true) != 4.5 || NeedsAAA(false) != 7 {
		t.Error("needs")
	}
}

func TestOver(t *testing.T) {
	half := Color{0, 0, 0, 0.5}
	got := half.Over(white)
	if got.Hex() != "#808080" {
		t.Errorf("50%% black over white = %s", got.Hex())
	}
}
