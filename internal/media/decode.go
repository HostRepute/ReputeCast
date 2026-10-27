package media

import (
	"bytes"
	"context"
	"image/gif"
	"image/png"
)

// Decoding an image takes memory its file doesn't show: a few hundred
// bytes of header can claim gigabytes of pixels and of the decoder's
// working memory (review 2026-10). Whatever decodes an image checks
// DecodeCost, with the copies it makes, against DecodeBudget first, and
// decodes in the shared slot (DecodeSlot).

// DecodeCost is about the most memory decoding an image takes, in bytes,
// and its size in pixels, read from its header: for a PNG its pixels (8
// bytes each at 16 bits, 4 otherwise, and twice that for an interlaced
// one, decoded pass by pass next to the whole image), for a GIF its first
// frame (a byte a pixel, within the screen), for a JPEG what image/jpeg
// allocates for its frame (jpegFrame.memory). ok is false when the header
// isn't one a decoder would read, or the image is empty.
func DecodeCost(data []byte) (cost int64, width, height int, ok bool) {
	switch {
	case bytes.HasPrefix(data, pngSignature):
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			return 0, 0, 0, false
		}
		// The header is the first chunk: bit depth, then the interlace method.
		cost = int64(cfg.Width) * int64(cfg.Height) * int64(4+4*btoi(data[24] == 16))
		if data[28] == 1 {
			cost *= 2
		}
		return cost, cfg.Width, cfg.Height, true
	case bytes.HasPrefix(data, []byte("GIF8")):
		cfg, err := gif.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			return 0, 0, 0, false
		}
		return int64(cfg.Width) * int64(cfg.Height), cfg.Width, cfg.Height, true
	}
	f, found := readFrame(data)
	if !found || f.width <= 0 || f.height <= 0 {
		return 0, 0, 0, false
	}
	return int64(f.memory()), f.width, f.height, true
}

// DecodeSlot waits for the slot images are decoded in, one at a time
// across ReputeCast, and returns the function that frees it. err is ctx's
// when it ends first.
func DecodeSlot(ctx context.Context) (release func(), err error) {
	select {
	case optimizeSlots <- struct{}{}:
		return func() { <-optimizeSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
