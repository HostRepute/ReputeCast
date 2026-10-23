package ai

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxImageWidth is how wide an image goes to the model (1,024 px).
const MaxImageWidth = 1024

// imageData is an image ready for the model.
type imageData struct {
	MediaType string
	Data      []byte
}

func (i *imageData) part() *llm.Image {
	if i == nil {
		return nil
	}
	return &llm.Image{MediaType: i.MediaType, Data: i.Data}
}

// loadImage reads an image of the email: from the media library when it is
// one of the workspace's, else from its address through the SSRF guard.
// It is resized to MaxImageWidth.
func (s *Service) loadImage(ctx context.Context, ws *store.Workspace, src string) (*imageData, error) {
	raw, ctype, err := s.fetchImage(ctx, ws, src)
	if err != nil {
		return nil, invalid(ctx, "body.src", mImageUnreadable, map[string]any{"detail": imageReason(ctx, err)})
	}
	out, err := resize(raw, ctype, MaxImageWidth)
	if err != nil {
		return nil, invalid(ctx, "body.src", mImageUnreadable, map[string]any{"detail": imageReason(ctx, err)})
	}
	return out, nil
}

// imageError is why an image can't be read, as a message of the catalog
// ("it is over 5 MB"); Error is its English.
type imageError struct {
	m    i18n.Msg
	args i18n.Args
}

func (e *imageError) Error() string { return e.m.Text(i18n.Default, e.args) }

func imageErr(m i18n.Msg, args i18n.Args) error { return &imageError{m: m, args: args} }

// imageReason is why an image can't be read, in the request's language.
func imageReason(ctx context.Context, err error) string {
	var ie *imageError
	if errors.As(err, &ie) {
		return text(ctx, ie.m, ie.args)
	}
	return err.Error()
}

func (s *Service) fetchImage(ctx context.Context, ws *store.Workspace, src string) ([]byte, string, error) {
	u, err := url.Parse(strings.TrimSpace(src))
	if err != nil {
		return nil, "", imageErr(mImageAddress, nil)
	}
	if strings.HasPrefix(u.Path, media.PathPrefix) {
		rest := strings.TrimPrefix(u.Path, media.PathPrefix)
		if i := strings.Index(rest, "/"); i > 0 {
			rest = rest[:i]
		}
		if id, err := uuid.Parse(rest); err == nil {
			row, err := s.Q.AIMediaImage(ctx, store.AIMediaImageParams{ID: id, WorkspaceID: ws.ID})
			if err == nil {
				return row.Data, row.ContentType, nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, "", err
			}
			if u.Host == "" {
				return nil, "", imageErr(mImageNotInLibrary, nil)
			}
		}
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, "", imageErr(mImageWhere, nil)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	client := s.HTTP
	if client == nil {
		client = s.policy().Client(10 * time.Second)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, "", imageErr(mImageNoAnswer, i18n.Args{"host": u.Hostname()})
	}
	defer res.Body.Close() //nolint:errcheck
	if res.StatusCode != http.StatusOK {
		return nil, "", imageErr(mImageStatus, i18n.Args{"host": u.Hostname(), "status": res.StatusCode})
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, media.MaxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > media.MaxBytes {
		return nil, "", imageErr(mImageTooLarge, i18n.Args{"mb": media.MaxBytes >> 20})
	}
	return data, http.DetectContentType(data), nil
}

// resize decodes a PNG, JPEG or GIF and scales it down to width with a box
// filter; a JPEG stays a JPEG, the rest become PNG.
func resize(data []byte, ctype string, width int) (*imageData, error) {
	var img image.Image
	var err error
	switch {
	case strings.HasPrefix(ctype, "image/png"):
		img, err = png.Decode(bytes.NewReader(data))
	case strings.HasPrefix(ctype, "image/jpeg"):
		img, err = jpeg.Decode(bytes.NewReader(data))
	case strings.HasPrefix(ctype, "image/gif"):
		img, err = gif.Decode(bytes.NewReader(data))
	default:
		return nil, imageErr(mImageFormat, nil)
	}
	if err != nil {
		return nil, imageErr(mImageDecode, nil)
	}
	b := img.Bounds()
	if b.Dx() > width {
		img = boxScale(img, width, max(1, b.Dy()*width/b.Dx()))
	}
	var buf bytes.Buffer
	if strings.HasPrefix(ctype, "image/jpeg") {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, err
		}
		return &imageData{MediaType: "image/jpeg", Data: buf.Bytes()}, nil
	}
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &imageData{MediaType: "image/png", Data: buf.Bytes()}, nil
}

// boxScale scales an image down by averaging the source pixels each
// destination pixel covers.
func boxScale(src image.Image, w, h int) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		y0 := b.Min.Y + y*sh/h
		y1 := max(y0+1, b.Min.Y+(y+1)*sh/h)
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*sw/w
			x1 := max(x0+1, b.Min.X+(x+1)*sw/w)
			var r, g, bl, a, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					c := color.NRGBAModel.Convert(src.At(xx, yy)).(color.NRGBA)
					r += uint64(c.R)
					g += uint64(c.G)
					bl += uint64(c.B)
					a += uint64(c.A)
					n++
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(bl / n), A: uint8(a / n)})
		}
	}
	return dst
}
