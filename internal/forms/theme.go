package forms

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Colours of the public pages. The pages take the publisher's accent and
// background; the rest are Clearwater tokens (design/design-system/tokens.json).
const (
	inkColor     = "#0B1220"
	ink3Color    = "#5B6472"
	whiteColor   = "#FFFFFF"
	defaultTint  = "#F5F3FF" // publisher-ground
	defaultBrand = "#4338CA" // publisher-accent
)

// rgb is a colour with 8-bit channels.
type rgb struct{ r, g, b float64 }

// ParseColor reads "#4338CA" or "#43C" (any case).
func ParseColor(s string) (rgb, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return rgb{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rgb{}, false
	}
	return rgb{float64(v >> 16 & 0xff), float64(v >> 8 & 0xff), float64(v & 0xff)}, true
}

// Hex writes the colour as "#4338CA".
func (c rgb) Hex() string {
	cl := func(v float64) int { return int(math.Max(0, math.Min(255, math.Round(v)))) }
	return fmt.Sprintf("#%02X%02X%02X", cl(c.r), cl(c.g), cl(c.b))
}

// luminance is the WCAG 2.2 relative luminance.
func (c rgb) luminance() float64 {
	ch := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.r) + 0.7152*ch(c.g) + 0.0722*ch(c.b)
}

// Contrast is the WCAG contrast ratio of two colours (1 to 21).
func Contrast(a, b rgb) float64 {
	la, lb := a.luminance(), b.luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// mix returns t of a and 1-t of b.
func mix(a, b rgb, t float64) rgb {
	return rgb{a.r*t + b.r*(1-t), a.g*t + b.g*(1-t), a.b*t + b.b*(1-t)}
}

var (
	black = rgb{0, 0, 0}
	white = rgb{255, 255, 255}
)

// rounded is c as it will be written ("#4338CA"): contrast is checked on
// the colour that ships, not on its unrounded value.
func (c rgb) rounded() rgb {
	r, _ := ParseColor(c.Hex())
	return r
}

// darkenUntil darkens c until its written value reaches ratio against
// every background.
func darkenUntil(c rgb, ratio float64, bgs ...rgb) rgb {
	c = c.rounded()
	for range 60 {
		ok := true
		for _, bg := range bgs {
			if Contrast(c, bg) < ratio {
				ok = false
				break
			}
		}
		if ok {
			return c
		}
		c = mix(c, black, 0.94).rounded()
	}
	return c
}

// Palette is what the public pages derive from the publisher's colours.
type Palette struct {
	Accent     string `json:"accent"`
	Background string `json:"background"`
	// Hover darkens buttons and links; it is also the text on tinted panels.
	Hover string `json:"accentHover"`
	// Soft tints borders and chips (the confirmation tick, the active archive filter).
	Soft string `json:"accentSoft"`
	// Text is the accent as link text: at least 4.5:1 on white and on the background.
	Text string `json:"accentText"`
	// OnAccent is the text on accent buttons: white, or ink when white fails 4.5:1.
	OnAccent string `json:"onAccent"`
	// Notes say what was adjusted for contrast, in English; notesIn writes
	// them in a reader's language.
	Notes []string `json:"notes"`
	notes []paletteNote
}

// paletteNote is an adjustment for contrast: buttons use dark text because
// white on the accent (colour) is only ratio:1, or links use a darker shade
// (colour).
type paletteNote struct {
	white  bool
	colour string
	ratio  float64
}

func (n paletteNote) in(l i18n.Lang) string {
	if n.white {
		return mWhiteText.Text(l, i18n.Args{"accent": n.colour, "ratio": l.Region.Decimal(n.ratio, 1)})
	}
	return mDarkerLinks.Text(l, i18n.Args{"colour": n.colour})
}

// notesIn are the palette's notes in a reader's language.
func (p Palette) notesIn(l i18n.Lang) []string {
	out := make([]string, 0, len(p.notes))
	for _, n := range p.notes {
		out = append(out, n.in(l))
	}
	return out
}

// paletteError is why colours can't be used, explained for people: Error
// is its English, the message keeps it for readers of other languages.
type paletteError struct {
	background bool // the background is too dark (the others are about a hex value)
	text       i18n.Text
}

func (e *paletteError) Error() string { return english(e.text) }

// DerivePalette computes the colours of the public pages from an accent and
// a page background, keeping every text at WCAG 2.2 AA contrast.
func DerivePalette(accent, background string) (Palette, error) {
	a, ok := ParseColor(accent)
	if !ok {
		return Palette{}, &paletteError{text: mNotColour.Keep(i18n.Args{"value": strconv.Quote(accent), "example": "#4338CA"})}
	}
	bg, ok := ParseColor(background)
	if !ok {
		return Palette{}, &paletteError{text: mNotColour.Keep(i18n.Args{"value": strconv.Quote(background), "example": "#F5F3FF"})}
	}
	if muted, _ := ParseColor(ink3Color); Contrast(muted, bg) < 4.5 {
		return Palette{}, &paletteError{background: true, text: mTooDark.Keep(i18n.Args{"colour": bg.Hex()})}
	}
	a, bg = a.rounded(), bg.rounded()
	p := Palette{Accent: a.Hex(), Background: bg.Hex(), Notes: []string{}}
	p.OnAccent = whiteColor
	note := func(n paletteNote) {
		p.notes = append(p.notes, n)
		p.Notes = append(p.Notes, n.in(i18n.Default))
	}
	if Contrast(white, a) < 4.5 {
		// Ink when it reads well enough, else black: when white is under
		// 4.5:1, black is always above it.
		p.OnAccent = inkColor
		if ink, _ := ParseColor(inkColor); Contrast(ink, a) < 4.5 {
			p.OnAccent = "#000000"
		}
		note(paletteNote{white: true, colour: a.Hex(), ratio: Contrast(white, a)})
	}
	text := darkenUntil(a, 4.5, white, bg)
	if text != a {
		note(paletteNote{colour: text.Hex()})
	}
	p.Text = text.Hex()
	p.Hover = darkenUntil(mix(text, black, 0.82), 4.5, white, bg).Hex()
	soft := mix(a, white, 0.2).rounded()
	hover, _ := ParseColor(p.Hover)
	// Text on the soft tint is the hover colour: keep it readable.
	for range 20 {
		if Contrast(hover, soft) >= 4.5 {
			break
		}
		soft = mix(soft, white, 0.6).rounded()
	}
	p.Soft = soft.Hex()
	return p, nil
}
