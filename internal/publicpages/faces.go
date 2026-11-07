package publicpages

import (
	"bytes"
	"crypto/subtle"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

// The emoji scale's faces (design expansion, X11): PNG images, because
// mail clients don't show SVG, drawn from the composer's 16-unit icons at
// twice their 28 px size, in the email's accent colour (?c=0f766e).
//
//	/p/_/faces/{faces}-{key}.png?c={colour}[&k={signature}]    faces 3 or 5, key 1…faces
//
// Emails use the default colour; the poll and survey pages use the
// publisher's accent, which the instance signs (facePath). Any other
// colour is drawn in the default one: an image is drawn once and kept, so
// a colour anyone could pick would let anyone fill the memory, one colour
// at a time.

type point struct{ x, y float64 }

// mouths are the five mouths, from very unhappy to very happy, as cubic
// curves in the icon's 16-unit box: start, two control points, end.
var mouths = [5][4]point{
	{{5.3, 11.6}, {6.7, 9.7}, {9.3, 9.7}, {10.7, 11.6}},
	{{5.6, 11}, {6.9, 9.9}, {9.1, 9.9}, {10.4, 11}},
	{{5.6, 10.4}, {7.2, 10.4}, {8.8, 10.4}, {10.4, 10.4}},
	{{5.6, 9.8}, {6.9, 10.9}, {9.1, 10.9}, {10.4, 9.8}},
	{{5.1, 9.3}, {6.6, 11.6}, {9.4, 11.6}, {10.9, 9.3}},
}

// mouthOf picks a face's mouth: five faces use all five, three use the
// two ends and the middle.
func mouthOf(faces, key int) [4]point {
	if faces == 3 {
		return mouths[(key-1)*2]
	}
	return mouths[key-1]
}

func bezier(c [4]point, t float64) point {
	u := 1 - t
	return point{
		u*u*u*c[0].x + 3*u*u*t*c[1].x + 3*u*t*t*c[2].x + t*t*t*c[3].x,
		u*u*u*c[0].y + 3*u*u*t*c[1].y + 3*u*t*t*c[2].y + t*t*t*c[3].y,
	}
}

func segDist(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l := dx*dx + dy*dy
	t := 0.0
	if l > 0 {
		t = math.Max(0, math.Min(1, ((p.x-a.x)*dx+(p.y-a.y)*dy)/l))
	}
	ex, ey := p.x-(a.x+t*dx), p.y-(a.y+t*dy)
	return math.Sqrt(ex*ex + ey*ey)
}

// drawFace renders a face at size px, antialiased by supersampling.
func drawFace(faces, key, size int, ink color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	mouth := mouthOf(faces, key)
	var poly []point
	for i := 0; i <= 24; i++ {
		poly = append(poly, bezier(mouth, float64(i)/24))
	}
	scale := 16 / float64(size)
	const sub = 4
	inside := func(p point) bool {
		// The outline: a ring of width 1.5 around r = 6.6.
		if d := math.Hypot(p.x-8, p.y-8); math.Abs(d-6.6) <= 0.75 {
			return true
		}
		// The eyes: round-capped strokes of width 1.9 from y 6.3 to 6.5.
		for _, ex := range []float64{5.9, 10.1} {
			if segDist(p, point{ex, 6.3}, point{ex, 6.5}) <= 0.95 {
				return true
			}
		}
		for i := 1; i < len(poly); i++ {
			if segDist(p, poly[i-1], poly[i]) <= 0.75 {
				return true
			}
		}
		return false
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			hits := 0
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					p := point{(float64(x) + (float64(sx)+0.5)/sub) * scale, (float64(y) + (float64(sy)+0.5)/sub) * scale}
					if inside(p) {
						hits++
					}
				}
			}
			if hits > 0 {
				c := ink
				c.A = uint8(int(ink.A) * hits / (sub * sub))
				img.SetNRGBA(x, y, c)
			}
		}
	}
	return img
}

var (
	faceFile = regexp.MustCompile(`^([35])-([1-5])\.png$`)
	hexColor = regexp.MustCompile(`^[0-9a-f]{6}$`)
)

// faceDefault is the faces' colour in emails, and of any colour the
// instance didn't sign.
const faceDefault = "0f766e"

// faceCacheMax bounds the drawn images kept: eight faces for each accent
// in use. Past it, an image is drawn for each request instead.
const faceCacheMax = 512

// faceCache keeps the drawn images ("5-1-0f766e" → PNG).
var faceCache = struct {
	sync.Mutex
	m map[string][]byte
}{m: map[string][]byte{}}

func faceCached(key string) ([]byte, bool) {
	faceCache.Lock()
	defer faceCache.Unlock()
	b, ok := faceCache.m[key]
	return b, ok
}

func faceKeep(key string, b []byte) {
	faceCache.Lock()
	defer faceCache.Unlock()
	if len(faceCache.m) < faceCacheMax {
		faceCache.m[key] = b
	}
}

// faceSignature signs a colour for the faces' addresses.
func (h *Handler) faceSignature(colour string) string {
	if h.Tokens == nil {
		return ""
	}
	return hex.EncodeToString(h.Tokens.Digest([]byte("publicpages/face"), []byte(colour))[:8])
}

// facePath is a face image's path under /p/_/ in a colour ("#4338CA"),
// signed unless it's the default.
func (h *Handler) facePath(faces, key int, colour string) string {
	p := "faces/" + strconv.Itoa(faces) + "-" + strconv.Itoa(key) + ".png?c="
	c := strings.ToLower(strings.TrimPrefix(colour, "#"))
	sig := ""
	if c != faceDefault && hexColor.MatchString(c) {
		sig = h.faceSignature(c)
	}
	if sig == "" {
		return p + faceDefault
	}
	return p + c + "&k=" + sig
}

// faceColour is the colour a face is drawn in: the one asked for when the
// instance signed it, else the default.
func (h *Handler) faceColour(c, sig string) string {
	c = strings.ToLower(c)
	if c == faceDefault || !hexColor.MatchString(c) {
		return faceDefault
	}
	want := h.faceSignature(c)
	if want == "" || subtle.ConstantTimeCompare([]byte(want), []byte(sig)) != 1 {
		return faceDefault
	}
	return c
}

func (h *Handler) face(w http.ResponseWriter, r *http.Request) {
	m := faceFile.FindStringSubmatch(chi.URLParam(r, "file"))
	if m == nil {
		http.NotFound(w, r)
		return
	}
	faces, _ := strconv.Atoi(m[1])
	key, _ := strconv.Atoi(m[2])
	if key > faces {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	colour := h.faceColour(q.Get("c"), q.Get("k"))
	ck := m[1] + "-" + m[2] + "-" + colour
	data, ok := faceCached(ck)
	if !ok {
		v, _ := strconv.ParseUint(colour, 16, 32)
		ink := color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
		var buf bytes.Buffer
		if err := png.Encode(&buf, drawFace(faces, key, 56, ink)); err != nil {
			http.Error(w, "The image couldn't be drawn.", http.StatusInternalServerError)
			return
		}
		data = buf.Bytes()
		faceKeep(ck, data)
	}
	hd := w.Header()
	hd.Set("Content-Type", "image/png")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cross-Origin-Resource-Policy", "cross-origin")
	hd.Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, "face.png", staticTime, bytes.NewReader(data))
}
