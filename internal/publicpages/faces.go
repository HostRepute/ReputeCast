package publicpages

import (
	"bytes"
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
//	/p/_/faces/{faces}-{key}.png    faces 3 or 5, key 1…faces

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
	faceFile  = regexp.MustCompile(`^([35])-([1-5])\.png$`)
	hexColor  = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
	faceCache sync.Map // "5-1-0f766e" → []byte
)

// FacePath is a face image's path under /p/_/ with its colour ("0f766e").
func FacePath(faces, key int, hex string) string {
	return "faces/" + strconv.Itoa(faces) + "-" + strconv.Itoa(key) + ".png?c=" + strings.ToLower(strings.TrimPrefix(hex, "#"))
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
	hex := strings.ToLower(r.URL.Query().Get("c"))
	if !hexColor.MatchString(hex) {
		hex = "0f766e"
	}
	ck := m[1] + "-" + m[2] + "-" + hex
	data, ok := faceCache.Load(ck)
	if !ok {
		v, _ := strconv.ParseUint(hex, 16, 32)
		ink := color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
		var buf bytes.Buffer
		if err := png.Encode(&buf, drawFace(faces, key, 56, ink)); err != nil {
			http.Error(w, "The image couldn't be drawn.", http.StatusInternalServerError)
			return
		}
		data, _ = faceCache.LoadOrStore(ck, buf.Bytes())
	}
	hd := w.Header()
	hd.Set("Content-Type", "image/png")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cross-Origin-Resource-Policy", "cross-origin")
	hd.Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, "face.png", staticTime, bytes.NewReader(data.([]byte)))
}
