package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color/palette"
	"image/gif"
	"testing"
)

// pngHeader is a PNG's signature and header (IHDR): enough for DecodeCost.
func pngHeader(w, h int, depth, colorType, interlace byte) []byte {
	ihdr := binary.BigEndian.AppendUint32(nil, uint32(w))
	ihdr = binary.BigEndian.AppendUint32(ihdr, uint32(h))
	ihdr = append(ihdr, depth, colorType, 0, 0, interlace)
	return append(append([]byte{}, pngSignature...), pngChunk("IHDR", ihdr)...)
}

func TestDecodeCost(t *testing.T) {
	var g bytes.Buffer
	if err := gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 300, 200), palette.Plan9), nil); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		data       []byte
		cost       int64
		w, h       int
		ok         bool
		overBudget bool
	}{
		"png":            {encPNG(t, halves(64, 32)), 64 * 32 * 4, 64, 32, true, false},
		"16-bit png":     {pngHeader(1000, 500, 16, 6, 0), 1000 * 500 * 8, 1000, 500, true, false},
		"interlaced png": {pngHeader(1000, 500, 8, 2, 1), 1000 * 500 * 4 * 2, 1000, 500, true, false},
		// A header of 33 bytes claiming 4 gigapixels.
		"huge png": {pngHeader(65535, 65535, 8, 6, 0), 65535 * 65535 * 4, 65535, 65535, true, true},
		"gif":      {g.Bytes(), 300 * 200, 300, 200, true, false},
		"jpeg":     {encJPEG(t, halves(64, 32), 90), 3 * 64 * 32, 64, 32, true, false},
		// 135 bytes claiming 7071×7071 pixels, progressive: about 1.2 GB.
		"progressive jpeg": {progressiveJPEG(7071, 7071, 4, false), -1, 7071, 7071, true, true},
		"empty png":        {pngHeader(0, 10, 8, 6, 0), 0, 0, 0, false, false},
		"webp":             {[]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), 0, 0, 0, false, false},
		"cut short":        {encJPEG(t, halves(64, 32), 90)[:40], 0, 0, 0, false, false},
	} {
		cost, w, h, ok := DecodeCost(c.data)
		if ok != c.ok || w != c.w || h != c.h || (c.cost >= 0 && cost != c.cost) || (cost > DecodeBudget) != c.overBudget {
			t.Errorf("%s: cost %d, %d×%d, ok %v", name, cost, w, h, ok)
		}
	}
}

func TestDecodeSlot(t *testing.T) {
	release, err := DecodeSlot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// One image at a time: another decode waits, until its request ends.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DecodeSlot(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("while the slot is taken: %v", err)
	}
	release()
	again, err := DecodeSlot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	again()
}
