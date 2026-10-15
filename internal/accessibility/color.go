package accessibility

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Color is an sRGB color with alpha (0 to 1).
type Color struct {
	R, G, B uint8
	A       float64
}

// Opaque reports whether nothing shows through the color.
func (c Color) Opaque() bool { return c.A >= 0.999 }

// Hex is "#RRGGBB" in capitals, as the boards write colors.
func (c Color) Hex() string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// Over composites c onto an opaque background.
func (c Color) Over(bg Color) Color {
	if c.Opaque() {
		return Color{c.R, c.G, c.B, 1}
	}
	mix := func(f, b uint8) uint8 {
		return uint8(math.Round(float64(f)*c.A + float64(b)*(1-c.A)))
	}
	return Color{mix(c.R, bg.R), mix(c.G, bg.G), mix(c.B, bg.B), 1}
}

var (
	white = Color{255, 255, 255, 1}
	black = Color{0, 0, 0, 1}
	// defaultLink is the link color mail clients and browsers use when a
	// link sets none.
	defaultLink = Color{0, 0, 238, 1}
)

// ParseColor reads a CSS color: #RGB, #RGBA, #RRGGBB, #RRGGBBAA, rgb(),
// rgba(), hsl(), hsla() and the named colors. ok is false for values that
// aren't a color (inherit, currentColor, a gradient, an image).
func ParseColor(v string) (Color, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.TrimSuffix(strings.TrimSpace(strings.TrimSuffix(s, "!important")), ";")
	s = strings.TrimSpace(s)
	if s == "" {
		return Color{}, false
	}
	if s == "transparent" {
		return Color{0, 0, 0, 0}, true
	}
	if strings.HasPrefix(s, "#") {
		return parseHex(s[1:])
	}
	if strings.HasPrefix(s, "rgb") {
		return parseRGB(s)
	}
	if strings.HasPrefix(s, "hsl") {
		return parseHSLFunc(s)
	}
	if c, ok := namedColors[s]; ok {
		return c, true
	}
	return Color{}, false
}

func parseHex(h string) (Color, bool) {
	expand := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		return b.String()
	}
	switch len(h) {
	case 3, 4:
		h = expand(h)
	case 6, 8:
	default:
		return Color{}, false
	}
	n, err := strconv.ParseUint(h, 16, 64)
	if err != nil {
		return Color{}, false
	}
	if len(h) == 8 {
		return Color{uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), float64(uint8(n)) / 255}, true
	}
	return Color{uint8(n >> 16), uint8(n >> 8), uint8(n), 1}, true
}

// funcArgs splits "rgb(1, 2, 3)" or "rgb(1 2 3 / 50%)" into its arguments.
func funcArgs(s string) []string {
	open, close := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if open < 0 || close < open {
		return nil
	}
	inner := strings.ReplaceAll(s[open+1:close], "/", " ")
	inner = strings.ReplaceAll(inner, ",", " ")
	return strings.Fields(inner)
}

func parseRGB(s string) (Color, bool) {
	args := funcArgs(s)
	if len(args) < 3 {
		return Color{}, false
	}
	var ch [3]uint8
	for i := 0; i < 3; i++ {
		v, ok := number(args[i], 255)
		if !ok {
			return Color{}, false
		}
		ch[i] = uint8(math.Round(math.Max(0, math.Min(255, v))))
	}
	a := 1.0
	if len(args) >= 4 {
		v, ok := number(args[3], 1)
		if !ok {
			return Color{}, false
		}
		a = math.Max(0, math.Min(1, v))
	}
	return Color{ch[0], ch[1], ch[2], a}, true
}

func parseHSLFunc(s string) (Color, bool) {
	args := funcArgs(s)
	if len(args) < 3 {
		return Color{}, false
	}
	h, err := strconv.ParseFloat(strings.TrimSuffix(args[0], "deg"), 64)
	if err != nil {
		return Color{}, false
	}
	sat, ok1 := number(args[1], 1)
	light, ok2 := number(args[2], 1)
	if !ok1 || !ok2 {
		return Color{}, false
	}
	c := hslToColor(hsl{h, sat, light})
	if len(args) >= 4 {
		if a, ok := number(args[3], 1); ok {
			c.A = math.Max(0, math.Min(1, a))
		}
	}
	return c, true
}

// number reads "128", "50%" (of scale) or "0.5".
func number(s string, scale float64) (float64, bool) {
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil {
			return 0, false
		}
		return v / 100 * scale, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// Luminance is the WCAG 2.2 relative luminance.
func Luminance(c Color) float64 {
	ch := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// Ratio is the WCAG contrast ratio of two opaque colors (1 to 21).
func Ratio(a, b Color) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// RoundDown rounds a ratio down to one decimal, so 4.49 never shows as 4.5.
func RoundDown(r float64) float64 {
	return math.Floor(r*10+1e-9) / 10
}

// FormatRatio is "2.1:1".
func FormatRatio(r float64) string {
	return strconv.FormatFloat(RoundDown(r), 'f', 1, 64) + ":1"
}

// Contrast requirements of WCAG 2.2.
const (
	AANormal  = 4.5
	AALarge   = 3.0
	AAANormal = 7.0
	AAALarge  = 4.5
)

// Large reports whether text counts as large: 24 px, or 18.66 px bold.
func Large(sizePx float64, bold bool) bool {
	return sizePx >= 24 || (bold && sizePx >= 18.66)
}

// Needs is the AA ratio text of this size needs.
func Needs(large bool) float64 {
	if large {
		return AALarge
	}
	return AANormal
}

// NeedsAAA is the AAA ratio text of this size needs.
func NeedsAAA(large bool) float64 {
	if large {
		return AAALarge
	}
	return AAANormal
}

// passes compares with a tolerance for floating-point error; the ratio
// itself is never rounded up.
func passes(ratio, need float64) bool { return ratio+1e-9 >= need }

type hsl struct{ H, S, L float64 }

func colorToHSL(c Color) hsl {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l := (mx + mn) / 2
	if mx == mn {
		return hsl{0, 0, l}
	}
	d := mx - mn
	s := d / (1 - math.Abs(2*l-1))
	var h float64
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return hsl{h, s, l}
}

func hslToColor(x hsl) Color {
	c := (1 - math.Abs(2*x.L-1)) * x.S
	hp := math.Mod(x.H, 360) / 60
	xx := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch {
	case hp < 1:
		r, g, b = c, xx, 0
	case hp < 2:
		r, g, b = xx, c, 0
	case hp < 3:
		r, g, b = 0, c, xx
	case hp < 4:
		r, g, b = 0, xx, c
	case hp < 5:
		r, g, b = xx, 0, c
	default:
		r, g, b = c, 0, xx
	}
	m := x.L - c/2
	to := func(v float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, v+m)) * 255)) }
	return Color{to(r), to(g), to(b), 1}
}

// Nearest returns the color closest to from (same hue and saturation, the
// lightness moved one percent at a time) that reaches need against fixed,
// and its ratio. ok is false when no shade of that hue gets there.
func Nearest(from, fixed Color, need float64) (Color, float64, bool) {
	if r := Ratio(from, fixed); passes(r, need) {
		return from, r, true
	}
	base := colorToHSL(from)
	try := func(dir float64) (Color, float64, int, bool) {
		for step := 1; step <= 100; step++ {
			l := base.L + dir*float64(step)/100
			if l < 0 || l > 1 {
				break
			}
			c := hslToColor(hsl{base.H, base.S, l})
			if r := Ratio(c, fixed); passes(r, need) {
				return c, r, step, true
			}
		}
		return Color{}, 0, 0, false
	}
	dc, dr, ds, dok := try(-1)
	lc, lr, ls, lok := try(+1)
	switch {
	case dok && lok:
		if ls < ds {
			return lc, lr, true
		}
		return dc, dr, true
	case dok:
		return dc, dr, true
	case lok:
		return lc, lr, true
	}
	return Color{}, 0, false
}

// ColorName is "white" or "black" for those two, the hex value otherwise.
func ColorName(c Color) string {
	switch c.Hex() {
	case "#FFFFFF":
		return "white"
	case "#000000":
		return "black"
	}
	return c.Hex()
}

var namedColors = map[string]Color{
	"aliceblue": {240, 248, 255, 1}, "antiquewhite": {250, 235, 215, 1}, "aqua": {0, 255, 255, 1}, "aquamarine": {127, 255, 212, 1},
	"azure": {240, 255, 255, 1}, "beige": {245, 245, 220, 1}, "bisque": {255, 228, 196, 1}, "black": {0, 0, 0, 1},
	"blanchedalmond": {255, 235, 205, 1}, "blue": {0, 0, 255, 1}, "blueviolet": {138, 43, 226, 1}, "brown": {165, 42, 42, 1},
	"burlywood": {222, 184, 135, 1}, "cadetblue": {95, 158, 160, 1}, "chartreuse": {127, 255, 0, 1}, "chocolate": {210, 105, 30, 1},
	"coral": {255, 127, 80, 1}, "cornflowerblue": {100, 149, 237, 1}, "cornsilk": {255, 248, 220, 1}, "crimson": {220, 20, 60, 1},
	"cyan": {0, 255, 255, 1}, "darkblue": {0, 0, 139, 1}, "darkcyan": {0, 139, 139, 1}, "darkgoldenrod": {184, 134, 11, 1},
	"darkgray": {169, 169, 169, 1}, "darkgreen": {0, 100, 0, 1}, "darkgrey": {169, 169, 169, 1}, "darkkhaki": {189, 183, 107, 1},
	"darkmagenta": {139, 0, 139, 1}, "darkolivegreen": {85, 107, 47, 1}, "darkorange": {255, 140, 0, 1}, "darkorchid": {153, 50, 204, 1},
	"darkred": {139, 0, 0, 1}, "darksalmon": {233, 150, 122, 1}, "darkseagreen": {143, 188, 143, 1}, "darkslateblue": {72, 61, 139, 1},
	"darkslategray": {47, 79, 79, 1}, "darkslategrey": {47, 79, 79, 1}, "darkturquoise": {0, 206, 209, 1}, "darkviolet": {148, 0, 211, 1},
	"deeppink": {255, 20, 147, 1}, "deepskyblue": {0, 191, 255, 1}, "dimgray": {105, 105, 105, 1}, "dimgrey": {105, 105, 105, 1},
	"dodgerblue": {30, 144, 255, 1}, "firebrick": {178, 34, 34, 1}, "floralwhite": {255, 250, 240, 1}, "forestgreen": {34, 139, 34, 1},
	"fuchsia": {255, 0, 255, 1}, "gainsboro": {220, 220, 220, 1}, "ghostwhite": {248, 248, 255, 1}, "gold": {255, 215, 0, 1},
	"goldenrod": {218, 165, 32, 1}, "gray": {128, 128, 128, 1}, "green": {0, 128, 0, 1}, "greenyellow": {173, 255, 47, 1},
	"grey": {128, 128, 128, 1}, "honeydew": {240, 255, 240, 1}, "hotpink": {255, 105, 180, 1}, "indianred": {205, 92, 92, 1},
	"indigo": {75, 0, 130, 1}, "ivory": {255, 255, 240, 1}, "khaki": {240, 230, 140, 1}, "lavender": {230, 230, 250, 1},
	"lavenderblush": {255, 240, 245, 1}, "lawngreen": {124, 252, 0, 1}, "lemonchiffon": {255, 250, 205, 1}, "lightblue": {173, 216, 230, 1},
	"lightcoral": {240, 128, 128, 1}, "lightcyan": {224, 255, 255, 1}, "lightgoldenrodyellow": {250, 250, 210, 1}, "lightgray": {211, 211, 211, 1},
	"lightgreen": {144, 238, 144, 1}, "lightgrey": {211, 211, 211, 1}, "lightpink": {255, 182, 193, 1}, "lightsalmon": {255, 160, 122, 1},
	"lightseagreen": {32, 178, 170, 1}, "lightskyblue": {135, 206, 250, 1}, "lightslategray": {119, 136, 153, 1}, "lightslategrey": {119, 136, 153, 1},
	"lightsteelblue": {176, 196, 222, 1}, "lightyellow": {255, 255, 224, 1}, "lime": {0, 255, 0, 1}, "limegreen": {50, 205, 50, 1},
	"linen": {250, 240, 230, 1}, "magenta": {255, 0, 255, 1}, "maroon": {128, 0, 0, 1}, "mediumaquamarine": {102, 205, 170, 1},
	"mediumblue": {0, 0, 205, 1}, "mediumorchid": {186, 85, 211, 1}, "mediumpurple": {147, 112, 219, 1}, "mediumseagreen": {60, 179, 113, 1},
	"mediumslateblue": {123, 104, 238, 1}, "mediumspringgreen": {0, 250, 154, 1}, "mediumturquoise": {72, 209, 204, 1}, "mediumvioletred": {199, 21, 133, 1},
	"midnightblue": {25, 25, 112, 1}, "mintcream": {245, 255, 250, 1}, "mistyrose": {255, 228, 225, 1}, "moccasin": {255, 228, 181, 1},
	"navajowhite": {255, 222, 173, 1}, "navy": {0, 0, 128, 1}, "oldlace": {253, 245, 230, 1}, "olive": {128, 128, 0, 1},
	"olivedrab": {107, 142, 35, 1}, "orange": {255, 165, 0, 1}, "orangered": {255, 69, 0, 1}, "orchid": {218, 112, 214, 1},
	"palegoldenrod": {238, 232, 170, 1}, "palegreen": {152, 251, 152, 1}, "paleturquoise": {175, 238, 238, 1}, "palevioletred": {219, 112, 147, 1},
	"papayawhip": {255, 239, 213, 1}, "peachpuff": {255, 218, 185, 1}, "peru": {205, 133, 63, 1}, "pink": {255, 192, 203, 1},
	"plum": {221, 160, 221, 1}, "powderblue": {176, 224, 230, 1}, "purple": {128, 0, 128, 1}, "rebeccapurple": {102, 51, 153, 1},
	"red": {255, 0, 0, 1}, "rosybrown": {188, 143, 143, 1}, "royalblue": {65, 105, 225, 1}, "saddlebrown": {139, 69, 19, 1},
	"salmon": {250, 128, 114, 1}, "sandybrown": {244, 164, 96, 1}, "seagreen": {46, 139, 87, 1}, "seashell": {255, 245, 238, 1},
	"sienna": {160, 82, 45, 1}, "silver": {192, 192, 192, 1}, "skyblue": {135, 206, 235, 1}, "slateblue": {106, 90, 205, 1},
	"slategray": {112, 128, 144, 1}, "slategrey": {112, 128, 144, 1}, "snow": {255, 250, 250, 1}, "springgreen": {0, 255, 127, 1},
	"steelblue": {70, 130, 180, 1}, "tan": {210, 180, 140, 1}, "teal": {0, 128, 128, 1}, "thistle": {216, 191, 216, 1},
	"tomato": {255, 99, 71, 1}, "turquoise": {64, 224, 208, 1}, "violet": {238, 130, 238, 1}, "wheat": {245, 222, 179, 1},
	"white": {255, 255, 255, 1}, "whitesmoke": {245, 245, 245, 1}, "yellow": {255, 255, 0, 1}, "yellowgreen": {154, 205, 50, 1},
}
