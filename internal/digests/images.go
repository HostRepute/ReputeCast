package digests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/url"
	"path"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
)

// ImageWidth is the width items' images are copied at: the Cards layout's
// 552 px (600 minus the 24 px margins). The Grid draws the same copy at 264.
const ImageWidth = 552

// copyImages copies the new items' images into the media library, resized
// to ImageWidth, so issues load them from the workspace's domain and not
// from the blog. An image that can't be copied is recorded; its post uses
// the block's fallback image.
func (s *Service) copyImages(ctx context.Context, ws *store.Workspace, digestID uuid.UUID) {
	rows, err := s.Q.DigestItemsNeedingImage(ctx, store.DigestItemsNeedingImageParams{DigestID: digestID, Lim: imagesPerRead})
	if err != nil {
		return
	}
	for _, r := range rows {
		up := store.DigestItemSetMediaParams{ID: r.ID}
		id, name, w, h, err := s.copyImage(ctx, ws, r.ImageUrl)
		if err != nil {
			up.ImageError = clip(err.Error(), 300)
		} else {
			up.MediaID, up.MediaUrl, up.MediaWidth, up.MediaHeight = &id, media.URL(id, name), int32(w), int32(h)
		}
		if err := s.Q.DigestItemSetMedia(ctx, up); err != nil {
			s.log().WarnContext(ctx, "record a feed image", "digest", digestID, "err", err)
		}
	}
}

type imageError string

func (e imageError) Error() string { return string(e) }

func (s *Service) copyImage(ctx context.Context, ws *store.Workspace, raw string) (uuid.UUID, string, int, int, error) {
	res, err := s.fetcher().Get(ctx, feed.Request{URL: raw, Accept: "image/png, image/jpeg, image/gif, image/*;q=0.8"})
	if err != nil {
		return uuid.Nil, "", 0, 0, err
	}
	data := res.Body
	ctype, w, h, alpha, err := media.Inspect(data)
	if err != nil {
		return uuid.Nil, "", 0, 0, err
	}
	if w > ImageWidth {
		data, ctype, w, h, err = resize(data, ctype, alpha, ImageWidth)
		if err != nil {
			return uuid.Nil, "", 0, 0, err
		}
	}
	name := media.SafeFileName(fileNameOf(raw), ctype)
	sum := sha256.Sum256(data)
	row, err := s.Q.InsertMediaAsset(ctx, store.InsertMediaAssetParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, FileName: name, ContentType: ctype,
		SizeBytes: int32(len(data)), Width: int32(w), Height: int32(h), HasAlpha: alpha, Sha256: sum[:], Data: data, CreatedAt: s.now()})
	if err != nil {
		return uuid.Nil, "", 0, 0, err
	}
	return row.ID, row.FileName, int(row.Width), int(row.Height), nil
}

func fileNameOf(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		if b := path.Base(u.Path); b != "" && b != "/" && b != "." {
			return b
		}
	}
	return "feed-image"
}

// resize scales an image down to width with an area average (each pixel
// of the copy is the mean of the pixels it covers), which keeps photos and
// screenshots sharp enough at email sizes without an imaging library.
// Photos are written as JPEG, images with transparency as PNG; a GIF's
// first frame is kept.
func resize(data []byte, ctype string, alpha bool, width int) ([]byte, string, int, int, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", 0, 0, imageError("the image couldn't be read")
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= width {
		return data, ctype, sw, sh, nil
	}
	height := max(1, (sh*width+sw/2)/sw)
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		y0 := y * sh / height
		y1 := max(y0+1, (y+1)*sh/height)
		for x := range width {
			x0 := x * sw / width
			x1 := max(x0+1, (x+1)*sw/width)
			var r, g, bl, a, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					c := color.NRGBA64Model.Convert(src.At(b.Min.X+xx, b.Min.Y+yy)).(color.NRGBA64)
					r += uint64(c.R)
					g += uint64(c.G)
					bl += uint64(c.B)
					a += uint64(c.A)
					n++
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n >> 8), G: uint8(g / n >> 8), B: uint8(bl / n >> 8), A: uint8(a / n >> 8)})
		}
	}
	var out bytes.Buffer
	switch {
	case alpha || ctype == "image/png" && !isPhoto(src):
		if err := png.Encode(&out, dst); err != nil {
			return nil, "", 0, 0, err
		}
		ctype = "image/png"
	case ctype == "image/gif":
		if err := gif.Encode(&out, dst, nil); err != nil {
			return nil, "", 0, 0, err
		}
	default:
		if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", 0, 0, err
		}
		ctype = "image/jpeg"
	}
	return out.Bytes(), ctype, width, height, nil
}

// isPhoto guesses whether a PNG is a photograph (many colors, better as
// JPEG) or a diagram or screenshot (few colors, better as PNG).
func isPhoto(img image.Image) bool {
	b := img.Bounds()
	seen := map[uint32]bool{}
	step := max(1, b.Dx()*b.Dy()/4000)
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i++
			if i%step != 0 {
				continue
			}
			r, g, bl, _ := img.At(x, y).RGBA()
			seen[(r>>11)<<10|(g>>11)<<5|(bl>>11)] = true
			if len(seen) > 600 {
				return true
			}
		}
	}
	return false
}
