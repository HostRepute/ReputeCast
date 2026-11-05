package publicpages

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

func faceCount() int {
	faceCache.Lock()
	defer faceCache.Unlock()
	return len(faceCache.m)
}

// inkOf is the colour of a face image's solid pixels ("0f766e").
func inkOf(t *testing.T, body []byte) string {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, g, bl, a := img.At(x, y).RGBA(); a == 0xffff {
				return fmt.Sprintf("%02x%02x%02x", r>>8, g>>8, bl>>8)
			}
		}
	}
	t.Fatal("no solid pixel")
	return ""
}

// The faces are drawn in the default colour or in an accent the instance
// signed for a page; any other colour is drawn in the default one and
// keeps nothing new in memory.
func TestFaceColours(t *testing.T) {
	signer, err := linktoken.NewSigner(bytes.Repeat([]byte{5}, 32))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{Tokens: signer}
	r := chi.NewRouter()
	r.Get("/p/_/faces/{file}", h.face)
	get := func(path string) []byte {
		t.Helper()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p/_/"+path, nil))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s: %d %s", path, w.Code, w.Header().Get("Content-Type"))
		}
		return w.Body.Bytes()
	}

	// The default colour, as emails ask for it.
	if ink := inkOf(t, get("faces/5-1.png?c=0f766e")); ink != faceDefault {
		t.Fatalf("the default: %s", ink)
	}
	before := faceCount()
	for i := range 300 {
		path := fmt.Sprintf("faces/5-%d.png?c=%06x", i%5+1, 0x100000+i)
		if i%3 == 0 {
			path += "&k=0123456789abcdef"
		}
		if ink := inkOf(t, get(path)); ink != faceDefault {
			t.Fatalf("%s drawn in %s", path, ink)
		}
	}
	if grown := faceCount() - before; grown > 4 {
		t.Fatalf("300 colours nobody signed kept %d more images", grown)
	}

	// The publisher's accent, signed on the page.
	path := h.facePath(5, 2, "#4338CA")
	if !strings.Contains(path, "c=4338ca&k=") {
		t.Fatalf("the page's address: %s", path)
	}
	if ink := inkOf(t, get(path)); ink != "4338ca" {
		t.Fatalf("the accent drawn in %s", ink)
	}
	// Signed for another colour, or by another instance: the default.
	other, _ := linktoken.NewSigner(bytes.Repeat([]byte{6}, 32))
	forged := (&Handler{Tokens: other}).facePath(5, 2, "#4338CA")
	sig := path[strings.Index(path, "&k=")+3:]
	for _, p := range []string{forged, "faces/5-2.png?c=4338cb&k=" + sig, "faces/5-2.png?c=4338ca"} {
		if ink := inkOf(t, get(p)); ink != faceDefault {
			t.Errorf("%s drawn in %s", p, ink)
		}
	}
	// The default needs no signature; without a signer, every page gets it.
	if p := h.facePath(3, 1, "#0F766E"); p != "faces/3-1.png?c=0f766e" {
		t.Errorf("the default's address: %s", p)
	}
	if p := (&Handler{}).facePath(3, 1, "#4338CA"); p != "faces/3-1.png?c=0f766e" {
		t.Errorf("without a signer: %s", p)
	}
}
