package media_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/media"
)

func pngBytes(t *testing.T, w, h int, alpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			a := uint8(255)
			if alpha && x == 0 {
				a = 0
			}
			img.Set(x, y, color.NRGBA{R: 15, G: 118, B: 110, A: a})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func upload(t *testing.T, c *apptest.Client, name string, data []byte) *apptest.Response {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(data)
	_ = mw.Close()
	return c.DoRaw("POST", "/workspaces/acme-newsletter/media", mw.FormDataContentType(), &body)
}

func TestInspect(t *testing.T) {
	ct, w, h, alpha, err := media.Inspect(pngBytes(t, 40, 20, true))
	if err != nil || ct != "image/png" || w != 40 || h != 20 || !alpha {
		t.Fatalf("png: %s %d %d %v %v", ct, w, h, alpha, err)
	}
	if _, _, _, _, err := media.Inspect([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)); err == nil || !strings.Contains(err.Error(), "SVG") {
		t.Fatalf("svg must be refused: %v", err)
	}
	if _, _, _, _, err := media.Inspect([]byte("\x89PNG\r\n\x1a\nnot really")); err == nil {
		t.Fatal("broken png must be refused")
	}
	if got := media.SafeFileName("../../Hero Image (final).PNG", "image/png"); got != "hero-image-final.png" {
		t.Fatalf("file name: %s", got)
	}
	if got := media.Absolutize(`<img src="/media/1/a.png"><td background="/media/2/b.png" style="background:url('/media/2/b.png')">`, "https://news.acme.io/"); strings.Count(got, "https://news.acme.io/media/") != 3 {
		t.Fatalf("absolutize: %s", got)
	}
}

func TestUploadServeRemove(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	data := pngBytes(t, 120, 60, false)
	var a struct {
		ID, URL, FileName, ContentType string
		Width, Height, SizeBytes       int
	}
	upload(t, c, "Logo.png", data).MustStatus(t, 201).JSON(t, &a)
	if a.ContentType != "image/png" || a.Width != 120 || a.Height != 60 || a.FileName != "logo.png" || !strings.HasPrefix(a.URL, "/media/"+a.ID+"/") {
		t.Fatalf("upload: %+v", a)
	}
	// The same bytes again give the same image.
	var again struct{ ID string }
	upload(t, c, "logo-copy.png", data).MustStatus(t, 201).JSON(t, &again)
	if again.ID != a.ID {
		t.Fatalf("dedupe: %s vs %s", again.ID, a.ID)
	}
	r := upload(t, c, "evil.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	r.MustStatus(t, 422)
	if !strings.Contains(string(r.Body), "SVG") {
		t.Fatalf("svg problem: %s", r.Body)
	}

	// Served publicly, cached, typed by content.
	anon := env.Client()
	g := anon.DoRaw("GET", a.URL, "", nil)
	g.MustStatus(t, 200)
	if !bytes.Equal(g.Body, data) || g.Header.Get("Content-Type") != "image/png" || !strings.Contains(g.Header.Get("Cache-Control"), "immutable") || g.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("serve headers: %v", g.Header)
	}
	anon.DoRaw("GET", "/media/not-a-uuid/x.png", "", nil).MustStatus(t, 404)
	anon.DoRaw("GET", "/media/starters/hero.png", "", nil).MustStatus(t, 200)
	anon.DoRaw("GET", "/media/starters/..%2fblank.mjml", "", nil).MustStatus(t, 404)

	var list struct{ Items []struct{ ID string } }
	c.Do("GET", "/workspaces/acme-newsletter/media", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) < 7 {
		t.Fatalf("library: %d", len(list.Items))
	}
	c.Do("DELETE", "/workspaces/acme-newsletter/media/"+a.ID, nil).MustStatus(t, 204)
	c.Do("GET", "/workspaces/acme-newsletter/media", nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.ID == a.ID {
			t.Fatal("removed image still listed")
		}
	}
	anon.DoRaw("GET", a.URL, "", nil).MustStatus(t, 200) // sent emails keep it

	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('media.upload','media.remove')`, a.ID).Scan(&n); err != nil || n != 3 {
		t.Fatalf("media audit: %d %v", n, err)
	}

	// Viewers can't upload.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	upload(t, v, "x.png", data).MustStatus(t, 403)
}

// photoWithLocation is a 3000×2000 JPEG stored sideways with EXIF saying
// "turn right" (orientation 6) and a stand-in for its GPS position.
func photoWithLocation(t *testing.T) []byte {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, 3000, 2000))
	for y := range 2000 {
		for x := range 3000 {
			m.SetRGBA(x, y, color.RGBA{uint8(x / 12), uint8(y / 8), 90, 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, m, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x02")
	tiff = append(tiff, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0)
	text := []byte("GPS 10.7769 N 106.7009 E\x00")
	tiff = append(tiff, 0x01, 0x0E, 0, 2)
	tiff = binary.BigEndian.AppendUint32(tiff, uint32(len(text)))
	tiff = binary.BigEndian.AppendUint32(tiff, 8+2+24+4)
	tiff = append(append(tiff, 0, 0, 0, 0), text...)
	app1 := append([]byte("Exif\x00\x00"), tiff...)
	seg := append([]byte{0xFF, 0xE1, byte((len(app1) + 2) >> 8), byte(len(app1) + 2)}, app1...)
	jpg := b.Bytes()
	return append(append(append([]byte{}, jpg[:2]...), seg...), jpg[2:]...)
}

// CO-12: what's stored is made for email: upright, at most 1,200 px wide,
// compressed, without the photo's location. The library says what changed.
func TestUploadIsResizedAndStripped(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	photo := photoWithLocation(t)
	type original struct {
		SizeBytes, Width, Height int
		ContentType              string
	}
	var a struct {
		ID, URL, FileName, ContentType string
		Width, Height, SizeBytes       int
		Original                       *original
	}
	upload(t, c, "IMG_2041.JPG", photo).MustStatus(t, 201).JSON(t, &a)
	if a.ContentType != "image/jpeg" || a.Width != 1200 || a.Height != 1800 || a.FileName != "img_2041.jpg" {
		t.Fatalf("stored: %+v", a)
	}
	if a.Original == nil || *a.Original != (original{SizeBytes: len(photo), Width: 3000, Height: 2000, ContentType: "image/jpeg"}) {
		t.Fatalf("original: %+v", a.Original)
	}
	if a.SizeBytes >= len(photo) {
		t.Fatalf("not compressed: %d of %d", a.SizeBytes, len(photo))
	}
	g := env.Client().DoRaw("GET", a.URL, "", nil)
	g.MustStatus(t, 200)
	if bytes.Contains(g.Body, []byte("GPS")) || bytes.Contains(g.Body, []byte("Exif")) || len(g.Body) != a.SizeBytes {
		t.Fatal("served image must be the stripped copy")
	}
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(g.Body)); err != nil || cfg.Width != 1200 || cfg.Height != 1800 {
		t.Fatalf("served: %+v %v", cfg, err)
	}

	// The same photo again is found by the uploaded file, not the copy.
	var again struct{ ID string }
	upload(t, c, "IMG_2041 copy.JPG", photo).MustStatus(t, 201).JSON(t, &again)
	if again.ID != a.ID {
		t.Fatalf("dedupe on the uploaded file: %s vs %s", again.ID, a.ID)
	}
	var list struct {
		Items []struct {
			ID       string
			Original *original
		}
	}
	c.Do("GET", "/workspaces/acme-newsletter/media", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Items[0].ID != a.ID || list.Items[0].Original == nil || list.Items[0].Original.Width != 3000 {
		t.Fatalf("library: %+v", list.Items[0])
	}
	// An image stored exactly as uploaded says nothing about an original.
	var small struct{ Original *original }
	upload(t, c, "dot.png", pngBytes(t, 8, 8, false)).MustStatus(t, 201).JSON(t, &small)
	if small.Original != nil {
		t.Fatalf("unchanged image: %+v", small.Original)
	}

	var after string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT after::text FROM audit_events WHERE target_id = $1 AND action = 'media.upload' ORDER BY occurred_at LIMIT 1`, a.ID).Scan(&after); err != nil ||
		!strings.Contains(after, `"original": {`) || !strings.Contains(after, `"width": 3000`) {
		t.Fatalf("audit: %s %v", after, err)
	}
}
