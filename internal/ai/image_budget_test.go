package ai

import (
	"bytes"
	"errors"
	"image/jpeg"
	"testing"
)

// progressiveJPEG is a progressive JPEG of w×h grey pixels in three
// components: one DC scan, every block 0. The file is small, but decoding
// it takes about 15 bytes a pixel.
func progressiveJPEG(w, h int) []byte {
	seg := func(m byte, p ...byte) []byte {
		return append([]byte{0xFF, m, byte((len(p) + 2) >> 8), byte(len(p) + 2)}, p...)
	}
	out := append([]byte{0xFF, 0xD8}, seg(0xDB, append([]byte{0}, bytes.Repeat([]byte{1}, 64)...)...)...)
	out = append(out, seg(0xC2, 8, byte(h>>8), byte(h), byte(w>>8), byte(w), 3, 1, 0x11, 0, 2, 0x11, 0, 3, 0x11, 0)...)
	out = append(out, seg(0xC4, append([]byte{0, 1}, make([]byte, 16)...)...)...) // one code: every DC difference is 0
	out = append(out, seg(0xDA, 3, 1, 0, 2, 0, 3, 0, 0, 0, 0)...)
	blocks := (w + 7) / 8 * ((h + 7) / 8) * 3
	scan := make([]byte, (blocks+7)/8)
	if r := blocks % 8; r != 0 {
		scan[len(scan)-1] = 0xFF >> r
	}
	return append(append(out, scan...), 0xFF, 0xD9)
}

// An image the model is shown is decoded only when decoding it and its
// scaled copy fit the memory an upload may take (review 2026-10): the
// file's size says nothing about it.
func TestResizeWithinDecodeBudget(t *testing.T) {
	// 146 KB that decode to 375 MB.
	_, err := resize(progressiveJPEG(5000, 5000), "image/jpeg", MaxImageWidth)
	var ie *imageError
	if !errors.As(err, &ie) || ie.m != mImageDecode {
		t.Fatalf("over the budget: %v", err)
	}
	// Within it, a progressive photo is read and scaled down.
	out, err := resize(progressiveJPEG(2000, 1000), "image/jpeg", MaxImageWidth)
	if err != nil || out.MediaType != "image/jpeg" {
		t.Fatalf("%v %+v", err, out)
	}
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(out.Data)); err != nil || cfg.Width != MaxImageWidth || cfg.Height != 512 {
		t.Fatalf("scaled to %d×%d (%v)", cfg.Width, cfg.Height, err)
	}
}
