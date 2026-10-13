package media_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
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
