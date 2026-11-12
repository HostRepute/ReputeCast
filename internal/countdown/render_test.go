package countdown

import (
	"bytes"
	"context"
	"image"
	"image/gif"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// blackFriday is the demo's countdown (Black Friday Early Access).
var blackFriday = CountdownSettings{EndsAt: "2026-11-27T23:59", TimeZone: "America/New_York", Units: "dhms", Language: "en", Size: "l",
	Background: "#0F172A", Digits: "#FFFFFF", Labels: "#CBD5E1", Ended: CountdownEnded{Mode: "zeros"}}

const twoDays14h5m9s = 2*24*time.Hour + 14*time.Hour + 5*time.Minute + 9*time.Second

func testAtlas(t testing.TB, s CountdownSettings) (*atlas, CountdownSettings, time.Time) {
	t.Helper()
	set, endsAt, err := Normalize(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newAtlas(set)
	if err != nil {
		t.Fatal(err)
	}
	return a, set, endsAt
}

// shown reads the digits an image shows, slot by slot, by matching each
// digit's place against the ten drawn digits (every one differs).
func shown(t *testing.T, a *atlas, img *image.Paletted, l layout) [][]int {
	t.Helper()
	out := make([][]int, len(a.units))
	for g := range a.units {
		for _, x := range l.x[g][:l.n[g]] {
			found := -1
			for d := range 10 {
				if same(img, a.digits[d], x, a.digitY, a.pitch, a.digitH) {
					if found >= 0 {
						t.Fatalf("slot at %d matches %d and %d", x, found, d)
					}
					found = d
				}
			}
			if found < 0 {
				t.Fatalf("slot at x=%d shows no digit", x)
			}
			out[g] = append(out[g], found)
		}
	}
	return out
}

func same(img *image.Paletted, tile []uint8, x, y, w, h int) bool {
	for row := range h {
		for col := range w {
			if img.ColorIndexAt(x+col, y+row) != tile[row*w+col] {
				return false
			}
		}
	}
	return true
}

// compose plays the frames onto one canvas (every frame keeps what was
// drawn before it).
func compose(g *gif.GIF, upTo int) *image.Paletted {
	canvas := image.NewPaletted(image.Rect(0, 0, g.Config.Width, g.Config.Height), g.Image[0].Palette)
	for _, f := range g.Image[:upTo+1] {
		b := f.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				canvas.SetColorIndex(x, y, f.ColorIndexAt(x, y))
			}
		}
	}
	return canvas
}

func TestRenderRunning(t *testing.T) {
	a, _, endsAt := testAtlas(t, blackFriday)
	now := endsAt.Add(-twoDays14h5m9s)
	b, err := a.render(secondsLeft(endsAt, now))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > 80<<10 {
		t.Fatalf("the image is %d bytes; at most 80 KB", len(b))
	}
	g, err := gif.DecodeAll(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != 60 || g.LoopCount != -1 || g.Config.Width != 600 || g.Config.Height != 150 {
		t.Fatalf("frames %d, loop %d, %dx%d", len(g.Image), g.LoopCount, g.Config.Width, g.Config.Height)
	}
	if len(g.Image[0].Palette) > 32 {
		t.Fatalf("%d colours", len(g.Image[0].Palette))
	}
	for i, d := range g.Delay {
		if d != 100 || g.Disposal[i] != gif.DisposalNone {
			t.Fatalf("frame %d: delay %d, disposal %d", i, d, g.Disposal[i])
		}
	}
	if g.Image[0].Bounds() != image.Rect(0, 0, 600, 150) {
		t.Fatalf("first frame %v", g.Image[0].Bounds())
	}
	// Later frames hold only the digits that changed: one a second, two
	// when the tens change.
	if r := g.Image[1].Bounds(); r.Dx() != a.pitch || r.Dy() != a.digitH {
		t.Fatalf("second frame %v, a digit is %dx%d", r, a.pitch, a.digitH)
	}
	slots := a.slots(2)
	want := [][]int{{0, 2}, {1, 4}, {0, 5}, {0, 9}}
	if got := shown(t, a, g.Image[0], slots); !equal(got, want) {
		t.Fatalf("first frame shows %v, want 02 14 05 09", got)
	}
	// 59 seconds later: 2 days 14 h 4 min 10 s.
	if got := shown(t, a, compose(g, 59), slots); !equal(got, [][]int{{0, 2}, {1, 4}, {0, 4}, {1, 0}}) {
		t.Fatalf("last frame shows %v, want 02 14 04 10", got)
	}
	// The labels are drawn, in the labels' colours.
	labels := 0
	for _, p := range g.Image[0].Pix {
		if p > idxDigits {
			labels++
		}
	}
	if labels < 200 {
		t.Fatalf("%d label pixels", labels)
	}
}

func equal(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for k := range a[i] {
			if a[i][k] != b[i][k] {
				return false
			}
		}
	}
	return true
}

func TestRenderEnded(t *testing.T) {
	for _, mode := range []CountdownEnded{{Mode: "zeros"}, {Mode: "text", Text: "Ưu đãi đã kết thúc"}} {
		s := blackFriday
		s.Ended = mode
		a, _, endsAt := testAtlas(t, s)
		b, err := a.render(secondsLeft(endsAt, endsAt.Add(3*time.Hour)))
		if err != nil {
			t.Fatal(err)
		}
		g, err := gif.DecodeAll(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if len(g.Image) != 1 {
			t.Fatalf("%s: %d frames", mode.Mode, len(g.Image))
		}
		if mode.Mode == "zeros" {
			if got := shown(t, a, g.Image[0], a.slots(0)); !equal(got, [][]int{{0, 0}, {0, 0}, {0, 0}, {0, 0}}) {
				t.Fatalf("ended shows %v", got)
			}
			continue
		}
		ink := 0
		for _, p := range g.Image[0].Pix {
			if p != idxBG {
				ink++
			}
		}
		if ink < 300 {
			t.Fatalf("the message drew %d pixels", ink)
		}
	}
}

// Within its last minute the image stops at the end: zeros, or the message.
func TestRenderLastSeconds(t *testing.T) {
	a, _, endsAt := testAtlas(t, blackFriday)
	b, err := a.render(secondsLeft(endsAt, endsAt.Add(-5*time.Second)))
	if err != nil {
		t.Fatal(err)
	}
	g, _ := gif.DecodeAll(bytes.NewReader(b))
	if len(g.Image) != 6 {
		t.Fatalf("%d frames", len(g.Image))
	}
	if got := shown(t, a, compose(g, 5), a.slots(0)); !equal(got, [][]int{{0, 0}, {0, 0}, {0, 0}, {0, 0}}) {
		t.Fatalf("last frame %v", got)
	}
}

// Without seconds, frames that change nothing lengthen the one before; the
// image still lasts a minute. Over 999 days shows 999.
func TestRenderMinutesAndCap(t *testing.T) {
	s := blackFriday
	s.Units, s.Size = "dhm", "s"
	a, _, endsAt := testAtlas(t, s)
	b, _ := a.render(secondsLeft(endsAt, endsAt.Add(-(3*time.Hour + 30*time.Second))))
	g, _ := gif.DecodeAll(bytes.NewReader(b))
	total := 0
	for _, d := range g.Delay {
		total += d
	}
	if len(g.Image) != 2 || total != 6000 || g.Config.Width != 360 || g.Config.Height != 96 {
		t.Fatalf("frames %d, total delay %d, %dx%d", len(g.Image), total, g.Config.Width, g.Config.Height)
	}
	if got := shown(t, a, g.Image[0], a.slots(0)); !equal(got, [][]int{{0, 0}, {0, 3}, {0, 0}}) {
		t.Fatalf("shows %v", got)
	}
	vals := a.values(int64(1200 * 24 * 3600))
	if vals[0] != 999 {
		t.Fatalf("days %v", vals)
	}
	b, _ = a.render(int64(1200 * 24 * 3600))
	g, _ = gif.DecodeAll(bytes.NewReader(b))
	if got := shown(t, a, g.Image[0], a.slots(999)); !equal(got, [][]int{{9, 9, 9}, {0, 0}, {0, 0}}) {
		t.Fatalf("shows %v", got)
	}
}

func TestNormalize(t *testing.T) {
	ctx := context.Background()
	in := blackFriday
	in.Background, in.Digits = "#0f172a", "#ffffff"
	set, endsAt, err := Normalize(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if set.Background != "#0F172A" || !endsAt.Equal(time.Date(2026, 11, 28, 4, 59, 0, 0, time.UTC)) {
		t.Fatalf("%+v %v", set, endsAt)
	}
	if h1, _ := Hash(set); string(h1) != string(must(Hash(must2(Normalize(ctx, blackFriday))))) {
		t.Fatal("the same settings hash differently")
	}
	if got := Alt(set, endsAt); got != "Offer ends Fri 27 Nov at 23:59 (EST)" {
		t.Fatalf("alt %q", got)
	}
	vi := set
	vi.Language = "vi"
	if got := Alt(vi, endsAt); got == "" || got == Alt(set, endsAt) {
		t.Fatalf("Vietnamese alt %q", got)
	}
	low := blackFriday
	low.Digits = "#334155"
	_, _, err = Normalize(ctx, low)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("1.7:1")) {
		t.Fatalf("low contrast: %v", err)
	}
	for name, mod := range map[string]func(*CountdownSettings){
		"zone":  func(s *CountdownSettings) { s.TimeZone = "Mars/Olympus" },
		"local": func(s *CountdownSettings) { s.EndsAt = "2026-02-30T10:00" },
		"color": func(s *CountdownSettings) { s.Labels = "white" },
		"text":  func(s *CountdownSettings) { s.Ended = CountdownEnded{Mode: "text"} },
		"glyph": func(s *CountdownSettings) { s.Ended = CountdownEnded{Mode: "text", Text: "セール終了"} },
	} {
		s := blackFriday
		mod(&s)
		if _, _, err := Normalize(ctx, s); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}

func must(b, _ []byte) []byte { return b }

func must2(s CountdownSettings, _ time.Time, err error) CountdownSettings {
	if err != nil {
		panic(err)
	}
	return s
}

// Every language's labels and alt text draw with the embedded fonts.
func TestLabelsDraw(t *testing.T) {
	for _, l := range []i18n.Locale{i18n.English, i18n.German, i18n.Spanish, i18n.French, i18n.Vietnamese} {
		for _, k := range append(labelKeys[:], "alt") {
			if m := missingGlyphs(msgs.M(k).Text(i18n.For(l), i18n.Args{"date": "", "time": "", "zone": ""})); len(m) > 0 {
				t.Fatalf("%s %s: missing %q", l, k, string(m))
			}
		}
	}
}

// BenchmarkRender draws an image that isn't cached (the atlas is).
func BenchmarkRender(b *testing.B) {
	a, _, endsAt := testAtlas(b, blackFriday)
	left := secondsLeft(endsAt, endsAt.Add(-twoDays14h5m9s))
	b.ReportAllocs()
	var size int
	for i := 0; b.Loop(); i++ {
		img, err := a.render(left - int64(i%3600))
		if err != nil {
			b.Fatal(err)
		}
		size = len(img)
	}
	b.ReportMetric(float64(size), "bytes/gif")
}

// BenchmarkAtlas draws a set of settings' digits and labels.
func BenchmarkAtlas(b *testing.B) {
	set, _, err := Normalize(context.Background(), blackFriday)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := newAtlas(set); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderCached answers an image already drawn this second.
func BenchmarkRenderCached(b *testing.B) {
	_, set, endsAt := testAtlas(b, blackFriday)
	c := newCaches()
	r := &row{id: uuid.Must(uuid.NewV7()), settings: set, endsAt: endsAt}
	h, _ := Hash(set)
	copy(r.hash[:], h)
	now := endsAt.Add(-twoDays14h5m9s)
	ctx := context.Background()
	if _, _, err := c.picture(ctx, r, now); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := c.picture(ctx, r, now); err != nil {
			b.Fatal(err)
		}
	}
}
