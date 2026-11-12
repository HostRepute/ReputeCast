package media_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/media"
)

// A WebP from an imported post is stored as email shows it: a photo as a
// JPEG at most 1200 px wide, a logo with transparency as a PNG. Inspect
// still refuses WebP: its callers store the bytes as they come.
func TestPrepareWebP(t *testing.T) {
	ctx := context.Background()
	photo, err := os.ReadFile("testdata/photo.webp")
	if err != nil {
		t.Fatal(err)
	}
	p, err := media.Prepare(ctx, photo)
	if err != nil {
		t.Fatalf("photo: %v", err)
	}
	if p.ContentType != "image/jpeg" || p.Width != 1200 || p.Height != 600 || p.HasAlpha {
		t.Fatalf("photo stored as %s %dx%d alpha %v", p.ContentType, p.Width, p.Height, p.HasAlpha)
	}
	if o := p.Original; o == nil || o.ContentType != "image/webp" || o.Width != 1600 || o.Height != 800 || o.SizeBytes != len(photo) {
		t.Fatalf("original: %+v", p.Original)
	}
	if p.Sha256 != sha256.Sum256(photo) {
		t.Fatal("the hash isn't the file's: the same file wouldn't be found again")
	}
	m, err := jpeg.Decode(bytes.NewReader(p.Data))
	if err != nil || m.Bounds().Dx() != 1200 {
		t.Fatalf("stored photo: %v %v", err, m.Bounds())
	}

	logo, err := os.ReadFile("testdata/logo-alpha.webp")
	if err != nil {
		t.Fatal(err)
	}
	p, err = media.Prepare(ctx, logo)
	if err != nil {
		t.Fatalf("logo: %v", err)
	}
	if p.ContentType != "image/png" || p.Width != 64 || p.Height != 64 || !p.HasAlpha {
		t.Fatalf("logo stored as %s %dx%d alpha %v", p.ContentType, p.Width, p.Height, p.HasAlpha)
	}
	l, err := png.Decode(bytes.NewReader(p.Data))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := l.At(0, 0).RGBA(); a != 0 {
		t.Fatalf("the logo's corner should stay transparent, alpha %d", a)
	}
	if params := p.Params(uuid.New(), "Logo (final).webp", nil, time.Now()); params.FileName != "logo-final.png" || params.OriginalContentType == nil || *params.OriginalContentType != "image/webp" {
		t.Fatalf("params: %s %v", params.FileName, params.OriginalContentType)
	}

	if cost, w, h, ok := media.DecodeCost(photo); !ok || w != 1600 || h != 800 || cost != 8*1600*800 {
		t.Fatalf("decode cost: %d %dx%d %v", cost, w, h, ok)
	}
	if _, _, _, _, err := media.Inspect(photo); err == nil || !strings.Contains(err.Error(), "WebP") {
		t.Fatalf("Inspect must refuse WebP: %v", err)
	}
	if _, err := media.Prepare(ctx, append([]byte("RIFF\x10\x00\x00\x00WEBPVP8 "), make([]byte, 12)...)); err == nil {
		t.Fatal("a damaged WebP must be refused")
	}
}
