// Package media is the workspace image library used by the composer's
// asset manager. Uploads are validated by their content (PNG, JPEG or GIF,
// decoded to read the dimensions), stored in Postgres and served publicly at
// /media/{id}/{file} with immutable caching, because email clients fetch
// them without a session. Content stores root-relative URLs; Absolutize
// turns them into public URLs when an email leaves ReputeCast.
package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxBytes is the largest image accepted (5 MB). Email images should be far
// smaller; Gmail and Outlook slow down above a few hundred KB.
const MaxBytes = 5 << 20

// MaxPixels guards against decompression bombs (100 megapixels).
const MaxPixels = 100_000_000

// PathPrefix is where images are served.
const PathPrefix = "/media/"

// Service holds the media domain dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
	// Static serves /media/starters/{file}: the placeholder images of the
	// built-in starter templates, shipped in the binary.
	Static fs.FS
	once   sync.Once
	cache  *lru
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// MediaAsset is one image in the library.
type MediaAsset struct {
	ID          string    `json:"id"`
	URL         string    `json:"url" doc:"Root-relative URL; emails get the absolute public URL"`
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType" enum:"image/png,image/jpeg,image/gif"`
	SizeBytes   int       `json:"sizeBytes"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	HasAlpha    bool      `json:"hasAlpha" doc:"Transparent pixels (logos survive dark mode)"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   string    `json:"createdBy"`
}

// URL is the root-relative URL of an asset.
func URL(id uuid.UUID, fileName string) string {
	return PathPrefix + id.String() + "/" + fileName
}

var (
	unsafeName = regexp.MustCompile(`[^a-z0-9._-]+`)
	dashes     = regexp.MustCompile(`-{2,}`)
)

// SafeFileName keeps a readable, URL-safe file name with the extension the
// content type implies.
func SafeFileName(name, contentType string) string {
	base := strings.ToLower(path.Base(strings.ReplaceAll(name, "\\", "/")))
	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '-'
		}
		return r
	}, base)
	base = strings.Trim(dashes.ReplaceAllString(unsafeName.ReplaceAllString(base, "-"), "-"), "-.")
	if len(base) > 80 {
		base = base[:80]
	}
	if base == "" {
		base = "image"
	}
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif"}[contentType]
	return base + ext
}

// ImageError is why an upload was refused, worded for the person uploading.
type ImageError string

func (e ImageError) Error() string { return string(e) }

// Inspect validates image bytes by their content and returns the type,
// dimensions and whether the image has transparency.
func Inspect(data []byte) (contentType string, width, height int, alpha bool, err error) {
	if len(data) == 0 {
		return "", 0, 0, false, ImageError("The file is empty.")
	}
	if len(data) > MaxBytes {
		return "", 0, 0, false, ImageError(fmt.Sprintf("The image is %.1f MB; the limit is 5 MB. Email images work best under 200 KB.", float64(len(data))/(1<<20)))
	}
	sniffed := http.DetectContentType(data)
	var cfg image.Config
	switch sniffed {
	case "image/png":
		cfg, err = png.DecodeConfig(bytes.NewReader(data))
	case "image/jpeg":
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case "image/gif":
		cfg, err = gif.DecodeConfig(bytes.NewReader(data))
	case "image/webp":
		return "", 0, 0, false, ImageError("WebP images don't display in Outlook and several other email clients. Upload a PNG, JPEG or GIF.")
	default:
		if strings.Contains(string(data[:min(len(data), 512)]), "<svg") {
			return "", 0, 0, false, ImageError("SVG images don't display in Gmail or Outlook. Upload a PNG, JPEG or GIF.")
		}
		return "", 0, 0, false, ImageError("That file isn't an image ReputeCast can use. Upload a PNG, JPEG or GIF.")
	}
	if err != nil {
		return "", 0, 0, false, ImageError("The image is damaged or incomplete and couldn't be read. Export it again and retry.")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return "", 0, 0, false, ImageError("The image dimensions are invalid or too large (100 megapixels at most).")
	}
	return sniffed, cfg.Width, cfg.Height, hasAlpha(sniffed, data, cfg), nil
}

// hasAlpha reports transparency: PNG with an alpha channel or a tRNS chunk,
// GIF with a transparent color index.
func hasAlpha(contentType string, data []byte, cfg image.Config) bool {
	switch contentType {
	case "image/png":
		// Color type byte of the IHDR chunk: 4 (gray+alpha) or 6 (RGBA).
		if len(data) > 25 && (data[25] == 4 || data[25] == 6) {
			return true
		}
		return bytes.Contains(data[:min(len(data), 1<<16)], []byte("tRNS"))
	case "image/gif":
		// Graphic Control Extension with the transparency flag.
		i := bytes.Index(data, []byte{0x21, 0xF9, 0x04})
		return i >= 0 && len(data) > i+3 && data[i+3]&0x01 == 1
	}
	_ = cfg
	return false
}

// Upload stores an image (or returns the identical one already stored).
func (s *Service) Upload(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, fileName string, data []byte) (MediaAsset, error) {
	ctype, w, h, alpha, err := Inspect(data)
	if err != nil {
		return MediaAsset{}, api.Invalid("body.file", err.Error())
	}
	sum := sha256.Sum256(data)
	name := SafeFileName(fileName, ctype)
	var out MediaAsset
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.InsertMediaAsset(ctx, store.InsertMediaAssetParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, FileName: name, ContentType: ctype, SizeBytes: int32(len(data)),
			Width: int32(w), Height: int32(h), HasAlpha: alpha, Sha256: sum[:], Data: data, CreatedBy: actor, CreatedAt: s.now(),
		})
		if err != nil {
			return err
		}
		out = MediaAsset{ID: row.ID.String(), URL: URL(row.ID, row.FileName), FileName: row.FileName, ContentType: row.ContentType, SizeBytes: int(row.SizeBytes),
			Width: int(row.Width), Height: int(row.Height), HasAlpha: row.HasAlpha, CreatedAt: row.CreatedAt}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "media.upload", TargetType: "media", TargetID: row.ID.String(), TargetName: row.FileName,
			After: map[string]any{"fileName": row.FileName, "contentType": row.ContentType, "sizeBytes": row.SizeBytes, "width": row.Width, "height": row.Height, "new": row.Inserted}})
	})
	return out, err
}

// List returns the library, newest first.
func (s *Service) List(ctx context.Context, ws *store.Workspace, limit int) ([]MediaAsset, error) {
	rows, err := s.Q.ListMediaAssets(ctx, store.ListMediaAssetsParams{WorkspaceID: ws.ID, Lim: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]MediaAsset, 0, len(rows))
	for _, r := range rows {
		out = append(out, MediaAsset{ID: r.ID.String(), URL: URL(r.ID, r.FileName), FileName: r.FileName, ContentType: r.ContentType, SizeBytes: int(r.SizeBytes),
			Width: int(r.Width), Height: int(r.Height), HasAlpha: r.HasAlpha, CreatedAt: r.CreatedAt, CreatedBy: r.CreatedByName})
	}
	return out, nil
}

// Remove hides an image from the library. It stays reachable so emails
// that were already sent keep showing it.
func (s *Service) Remove(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Image")
		}
		now := s.now()
		n, err := q.HideMediaAsset(ctx, store.HideMediaAssetParams{WorkspaceID: ws.ID, ID: id, Now: &now})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound("Image")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "media.remove", TargetType: "media", TargetID: id.String(), TargetName: a.FileName,
			Before: map[string]any{"fileName": a.FileName}})
	})
}

var relMedia = regexp.MustCompile(`((?:src|href|background)=["']|url\(\s*["']?)(/media/)`)

// Absolutize rewrites root-relative /media/ URLs in HTML to the public URL,
// for anything that leaves ReputeCast (sends, tests, exported HTML).
func Absolutize(html, publicURL string) string {
	base := strings.TrimRight(publicURL, "/")
	if base == "" || !strings.Contains(html, "/media/") {
		return html
	}
	return relMedia.ReplaceAllString(html, "${1}"+base+"/media/")
}
