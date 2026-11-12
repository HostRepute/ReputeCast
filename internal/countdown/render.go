package countdown

import (
	_ "embed"
	"image"
	"image/color"
	"math"
	"strconv"
	"sync"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// The image is drawn with two embedded IBM Plex faces (SIL Open Font
// License, fonts/OFL.txt): Mono SemiBold for the digits, Sans Medium for
// the labels and the message shown at the end (full Latin, Vietnamese
// included).

//go:embed fonts/IBMPlexMono-SemiBold.ttf
var monoTTF []byte

//go:embed fonts/IBMPlexSans-Medium.ttf
var sansTTF []byte

// faces are the parsed fonts. They are read-only and shared; each atlas
// makes its own faces from them (a face isn't safe for concurrent use).
var faces = sync.OnceValues(func() ([2]*opentype.Font, error) {
	mono, err := opentype.Parse(monoTTF)
	if err != nil {
		return [2]*opentype.Font{}, err
	}
	sans, err := opentype.Parse(sansTTF)
	return [2]*opentype.Font{mono, sans}, err
})

// Sizes of the image, in pixels.
var sizes = map[string]image.Point{"s": {X: 360, Y: 96}, "m": {X: 480, Y: 120}, "l": {X: 600, Y: 150}}

// Units the timer shows: days, hours, minutes, seconds.
const (
	unitDays = iota
	unitHours
	unitMinutes
	unitSeconds
)

var unitSets = map[string][]int{
	"dhms": {unitDays, unitHours, unitMinutes, unitSeconds},
	"hms":  {unitHours, unitMinutes, unitSeconds},
	"dhm":  {unitDays, unitHours, unitMinutes},
}

// Frames is how many seconds one image plays: it stops on its last frame.
const Frames = 60

// frameDelay is a frame's delay in hundredths of a second.
const frameDelay = 100

// maxFirst is the most the first unit shows (999 days).
const maxFirst = 999

// Palette layout: the background, eight anti-alias steps from it to the
// digits' colour and the digits, the same for the labels (19 colours).
const (
	idxBG     = 0
	idxDigits = 9  // 1–8 are the steps to it
	idxLabels = 18 // 10–17 are the steps to it
	aaSteps   = 9
)

// tile is a drawn piece of the image, as palette indices.
type tile struct {
	x, y, w, h int
	pix        []uint8
}

// atlas is everything drawn ahead for one set of settings: the palette,
// the ten digits, the labels in place and the end message. Images of the
// countdown are put together from it without drawing a glyph.
type atlas struct {
	w, h  int
	pal   color.Palette
	units []int
	// pitch is the distance between digits; digits are tiles of pitch ×
	// digitH at digitY.
	pitch, digitH, digitY int
	digits                [10][]uint8
	// base is the background with the labels; ended the end message on the
	// background (nil when the end shows zeros).
	base, ended *image.Paletted
}

// newAtlas draws an atlas for normalized settings.
func newAtlas(s CountdownSettings) (*atlas, error) {
	ff, err := faces()
	if err != nil {
		return nil, err
	}
	sz := sizes[s.Size]
	a := &atlas{w: sz.X, h: sz.Y, units: unitSets[s.Units]}
	bg, dg, lb := hexColor(s.Background), hexColor(s.Digits), hexColor(s.Labels)
	a.pal = make(color.Palette, 0, idxLabels+1)
	a.pal = append(a.pal, bg)
	for k := 1; k <= aaSteps; k++ {
		a.pal = append(a.pal, mix(bg, dg, float64(k)/aaSteps))
	}
	for k := 1; k <= aaSteps; k++ {
		a.pal = append(a.pal, mix(bg, lb, float64(k)/aaSteps))
	}
	scale := float64(a.w) / 600
	digitFace, err := opentype.NewFace(ff[0], &opentype.FaceOptions{Size: 62 * scale, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	defer func() { _ = digitFace.Close() }()
	labelFace, err := opentype.NewFace(ff[1], &opentype.FaceOptions{Size: math.Max(15*scale, 10), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	defer func() { _ = labelFace.Close() }()

	// The digits' common box: every digit fits it, so a changed digit is
	// one rectangle of the same size.
	var ink fixed.Rectangle26_6
	for d := '0'; d <= '9'; d++ {
		b, _ := font.BoundString(digitFace, string(d))
		ink = ink.Union(b)
	}
	adv := font.MeasureString(digitFace, "0")
	x0, x1 := min(0, ink.Min.X.Floor()), max(adv.Ceil(), ink.Max.X.Ceil())
	top, bottom := ink.Min.Y.Floor()-1, ink.Max.Y.Ceil()+1
	a.pitch, a.digitH = x1-x0, bottom-top

	// The labels' box, for the vertical layout.
	var lInk fixed.Rectangle26_6
	labels := make([]string, len(a.units))
	for i, u := range a.units {
		labels[i] = msgs.M(labelKeys[u]).Text(langOf(s.Language), nil)
		b, _ := font.BoundString(labelFace, labels[i])
		lInk = lInk.Union(b)
	}
	gap := int(math.Round(12 * scale))
	block := a.digitH + gap + (lInk.Max.Y.Ceil() - lInk.Min.Y.Floor())
	a.digitY = (a.h - block) / 2
	labelBaseline := a.digitY + a.digitH + gap - lInk.Min.Y.Floor()

	for d := range 10 {
		m := image.NewAlpha(image.Rect(0, 0, a.pitch, a.digitH))
		(&font.Drawer{Dst: m, Src: image.Opaque, Face: digitFace, Dot: fixed.P(-x0, -top)}).DrawString(strconv.Itoa(d))
		a.digits[d] = indices(m.Pix, 0)
	}

	a.base = image.NewPaletted(image.Rect(0, 0, a.w, a.h), a.pal)
	cell := a.w / len(a.units)
	for i, text := range labels {
		b, _ := font.BoundString(labelFace, text)
		wd := (b.Max.X - b.Min.X).Ceil() + 2
		cx := i*cell + cell/2
		t := drawText(labelFace, text, b, wd, cx-wd/2, labelBaseline, a.w, a.h, idxDigits)
		a.blit(a.base, t)
	}
	if s.Ended.Mode == "text" {
		a.ended = image.NewPaletted(image.Rect(0, 0, a.w, a.h), a.pal)
		size := 34 * scale
		margin := fixed.I(int(math.Round(20 * scale)))
		for {
			f, err := opentype.NewFace(ff[1], &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
			if err != nil {
				return nil, err
			}
			b, _ := font.BoundString(f, s.Ended.Text)
			if b.Max.X-b.Min.X <= fixed.I(a.w)-2*margin || size <= 10 {
				wd := (b.Max.X - b.Min.X).Ceil() + 2
				hd := (b.Max.Y - b.Min.Y).Ceil()
				base := (a.h-hd)/2 - b.Min.Y.Floor()
				a.blit(a.ended, drawText(f, s.Ended.Text, b, wd, (a.w-wd)/2, base, a.w, a.h, 0))
				_ = f.Close()
				break
			}
			_ = f.Close()
			size--
		}
	}
	return a, nil
}

// drawText draws text in a tile wd wide whose left edge is at x, on the
// baseline y, as palette indices from base (0 the digits' ramp, idxDigits
// the labels'). The tile is clipped to the image.
func drawText(f font.Face, text string, ink fixed.Rectangle26_6, wd, x, y, w, h int, base uint8) tile {
	top, bottom := y+ink.Min.Y.Floor()-1, y+ink.Max.Y.Ceil()+1
	r := image.Rect(x, top, x+wd, bottom).Intersect(image.Rect(0, 0, w, h))
	m := image.NewAlpha(r)
	(&font.Drawer{Dst: m, Src: image.Opaque, Face: f, Dot: fixed.Point26_6{X: fixed.I(x+1) - ink.Min.X, Y: fixed.I(y)}}).DrawString(text)
	return tile{x: r.Min.X, y: r.Min.Y, w: r.Dx(), h: r.Dy(), pix: indices(m.Pix, base)}
}

// indices turns coverage into palette indices: nothing is the background,
// full coverage the ink, and eight steps between.
func indices(alpha []uint8, base uint8) []uint8 {
	out := make([]uint8, len(alpha))
	for i, a := range alpha {
		if l := (int(a)*aaSteps + 127) / 255; l > 0 {
			out[i] = base + uint8(l)
		}
	}
	return out
}

// blit copies a tile onto an image.
func (a *atlas) blit(dst *image.Paletted, t tile) {
	for row := range t.h {
		o := (t.y+row-dst.Rect.Min.Y)*dst.Stride + t.x - dst.Rect.Min.X
		copy(dst.Pix[o:o+t.w], t.pix[row*t.w:(row+1)*t.w])
	}
}

// digit puts a digit at x on the digits' row.
func (a *atlas) digit(dst *image.Paletted, x, d int) {
	a.blit(dst, tile{x: x, y: a.digitY, w: a.pitch, h: a.digitH, pix: a.digits[d]})
}

// counts are the values of the units shown, first unit first.
type counts [4]int

// values splits the seconds left into the units shown: the first unit
// takes what the units it leaves out would show (hours count whole days
// without a days unit), capped at 999.
func (a *atlas) values(left int64) counts {
	var out counts
	for i, u := range a.units {
		var v int64
		switch u {
		case unitDays:
			v = left / 86400
		case unitHours:
			v = left / 3600
			if i > 0 {
				v %= 24
			}
		case unitMinutes:
			v = left / 60 % 60
		case unitSeconds:
			v = left % 60
		}
		if i == 0 && v > maxFirst {
			v = maxFirst
		}
		out[i] = int(v)
	}
	return out
}

// layout is where the digits go: each unit's digits' x positions.
type layout struct {
	n [4]int
	x [4][3]int
}

// slots lays the digits out, first unit first: the first unit has three
// digits when it starts at 100 or more.
func (a *atlas) slots(first int) layout {
	var l layout
	cell := a.w / len(a.units)
	for i := range a.units {
		n := 2
		if i == 0 && first >= 100 {
			n = 3
		}
		l.n[i] = n
		x := i*cell + (cell-n*a.pitch)/2
		for k := range n {
			l.x[i][k] = x + k*a.pitch
		}
	}
	return l
}

// draw writes the values' digits; with prev, only those that changed. It
// returns the rectangle it changed.
func (a *atlas) draw(dst *image.Paletted, l layout, vals counts, prev *counts) image.Rectangle {
	var changed image.Rectangle
	for i := range a.units {
		v, p := vals[i], -1
		if prev != nil {
			p = prev[i]
		}
		for k := l.n[i] - 1; k >= 0; k-- {
			d, pd := v%10, -1
			if p >= 0 {
				pd = p % 10
				p /= 10
			}
			v /= 10
			if d == pd {
				continue
			}
			x := l.x[i][k]
			a.digit(dst, x, d)
			changed = changed.Union(image.Rect(x, a.digitY, x+a.pitch, a.digitY+a.digitH))
		}
	}
	return changed
}

// step is one frame: what it shows and for how long; end is the end
// message.
type step struct {
	vals  counts
	delay int
	end   bool
}

// render draws the image for a countdown with left whole seconds to go: up
// to 60 frames a second apart, the first whole and each next one only the
// digits that changed; frames where nothing changes (no seconds shown)
// lengthen the one before. A countdown that has ended is one frame.
func (a *atlas) render(left int64) ([]byte, error) {
	g := writers.Get().(*gifWriter)
	defer writers.Put(g)
	g.start(a.w, a.h, a.pal)
	full := a.base.Rect
	if left <= 0 && a.ended != nil {
		g.frame(a.ended, full, 0)
		return g.finish(), nil
	}
	canvas := g.canvasOf(a.base)
	if left <= 0 {
		a.draw(canvas, a.slots(0), a.values(0), nil)
		g.frame(canvas, full, 0)
		return g.finish(), nil
	}
	// The frames first: a frame's delay is written before its pixels.
	n := min(int64(Frames), left+1)
	steps := append(g.steps[:0], step{vals: a.values(left), delay: frameDelay})
	for i := int64(1); i < n; i++ {
		if left-i == 0 && a.ended != nil {
			steps = append(steps, step{delay: frameDelay, end: true})
			break
		}
		v := a.values(left - i)
		if last := &steps[len(steps)-1]; v == last.vals {
			last.delay += frameDelay
			continue
		}
		steps = append(steps, step{vals: v, delay: frameDelay})
	}
	g.steps = steps
	l := a.slots(steps[0].vals[0])
	a.draw(canvas, l, steps[0].vals, nil)
	g.frame(canvas, full, steps[0].delay)
	for i := 1; i < len(steps); i++ {
		st := steps[i]
		if st.end {
			g.frame(a.ended, full, st.delay)
			break
		}
		r := a.draw(canvas, l, st.vals, &steps[i-1].vals)
		g.frame(canvas, r, st.delay)
	}
	return g.finish(), nil
}

// hexColor reads a normalized #RRGGBB.
func hexColor(s string) color.RGBA {
	n, _ := strconv.ParseUint(s[1:], 16, 32)
	return color.RGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 0xff}
}

// mix is the colour t of the way from a to b.
func mix(a, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B), A: 0xff}
}

// missingGlyphs are the characters of text the labels' font can't draw
// (spaces aside).
func missingGlyphs(text string) []rune {
	ff, err := faces()
	if err != nil {
		return nil
	}
	var buf sfnt.Buffer
	var out []rune
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if i, err := ff[1].GlyphIndex(&buf, r); err != nil || i == 0 {
			out = append(out, r)
		}
	}
	return out
}
