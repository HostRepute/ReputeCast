package digests

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/hostrepute/reputecast/internal/media"
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

// A feed's image is decoded one at a time, like uploads, and only when
// decoding it and its copy fit the memory an upload may take (review
// 2026-10): the file's size says nothing about it.
func TestResizeWithinDecodeBudget(t *testing.T) {
	ctx := context.Background()
	// 146 KB that decode to 375 MB: the item keeps the block's fallback.
	if _, _, _, _, err := resize(ctx, progressiveJPEG(5000, 5000), "image/jpeg", false, ImageWidth); err == nil || err.Error() != "the image couldn't be read" {
		t.Fatalf("over the budget: %v", err)
	}
	// Within it, a progressive photo is copied at ImageWidth.
	small := progressiveJPEG(2000, 1000)
	if _, ctype, w, h, err := resize(ctx, small, "image/jpeg", false, ImageWidth); err != nil || ctype != "image/jpeg" || w != ImageWidth || h != 276 {
		t.Fatalf("%v %s %d×%d", err, ctype, w, h)
	}
	// While an upload holds the slot, a feed's image waits for it.
	release, err := media.DecodeSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	waiting, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, _, _, err := resize(waiting, small, "image/jpeg", false, ImageWidth); !errors.Is(err, context.Canceled) {
		t.Fatalf("decoded beside an upload: %v", err)
	}
}
