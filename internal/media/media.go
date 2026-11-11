// Package media is the workspace image library used by the composer's
// asset manager. Uploads are validated by their content (PNG, JPEG or GIF,
// decoded to read the dimensions), resized and compressed for email with
// their metadata removed (optimize.go), stored in Postgres and served
// publicly at /media/{id}/{file} with immutable caching, because email
// clients fetch them without a session. Content stores root-relative URLs;
// Absolutize turns them into public URLs when an email leaves ReputeCast.
// Images that come from elsewhere (posts imported from another platform)
// go through Prepare, which also takes WebP and stores it as JPEG, or PNG
// when it has transparency: WebP doesn't display in Outlook.
package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math"
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
	xwebp "golang.org/x/image/webp"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxBytes is the largest image stored (5 MB), and the largest GIF
// accepted, since GIFs are kept as uploaded. Email images should be far
// smaller; Gmail and Outlook slow down above a few hundred KB.
const MaxBytes = 5 << 20

// MaxPixels guards against decompression bombs (50 megapixels, a phone's
// full-resolution photo).
const MaxPixels = 50_000_000

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
	// Original is the file as uploaded, when what's stored differs from it.
	Original *MediaOriginal `json:"original,omitempty" doc:"The file as uploaded, when ReputeCast resized or compressed it or removed its metadata"`
}

// MediaOriginal is what was uploaded, kept as numbers only (the file itself
// isn't kept: its metadata can hold where a photo was taken).
type MediaOriginal struct {
	SizeBytes   int    `json:"sizeBytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ContentType string `json:"contentType" enum:"image/png,image/jpeg,image/gif,image/webp"`
}

func originalOf(size, width, height *int32, contentType *string) *MediaOriginal {
	if size == nil || width == nil || height == nil || contentType == nil {
		return nil
	}
	return &MediaOriginal{SizeBytes: int(*size), Width: int(*width), Height: int(*height), ContentType: *contentType}
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
// dimensions and whether the image has transparency. Its ImageError reads
// in English. WebP is refused: its callers store the bytes as they are.
func Inspect(data []byte) (contentType string, width, height int, alpha bool, err error) {
	return inspect(i18n.Default, data, MaxBytes, false)
}

// inspect is Inspect with the reason in a reader's language. limit is the
// largest file accepted: MaxBytes, or MaxUploadBytes for an upload that is
// compressed before it's stored. webp accepts WebP, for a caller that
// stores what optimize makes of it.
func inspect(l i18n.Lang, data []byte, limit int, webp bool) (contentType string, width, height int, alpha bool, err error) {
	refuse := func(m i18n.Msg, args i18n.Args) (string, int, int, bool, error) {
		return "", 0, 0, false, ImageError(m.Text(l, args))
	}
	if len(data) == 0 {
		return refuse(mEmpty, nil)
	}
	if len(data) > limit {
		m := mTooLarge
		if limit == MaxUploadBytes {
			m = mUploadTooLarge
		}
		return refuse(m, i18n.Args{"size": mb(l, len(data))})
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
		if !webp {
			return refuse(mWebP, nil)
		}
		cfg, err = xwebp.DecodeConfig(bytes.NewReader(data))
	default:
		if strings.Contains(string(data[:min(len(data), 512)]), "<svg") {
			return refuse(mSVG, nil)
		}
		return refuse(mNotImage, nil)
	}
	if err != nil {
		return refuse(mDamaged, nil)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return refuse(mDimensions, nil)
	}
	if sniffed == "image/gif" && len(data) > MaxBytes {
		return refuse(mGIFTooLarge, i18n.Args{"size": mb(l, len(data))})
	}
	return sniffed, cfg.Width, cfg.Height, hasAlpha(sniffed, data, cfg), nil
}

// mb is a size in megabytes with one decimal, as the reader writes it,
// rounded up so a file just over a limit doesn't read as the limit.
func mb(l i18n.Lang, n int) string {
	return l.Region.Decimal(math.Ceil(float64(n)/(1<<20)*10)/10, 1)
}

// hasAlpha reports transparency: PNG with an alpha channel or a tRNS chunk,
// GIF with a transparent color index, WebP with an alpha channel (lossless
// WebP always has one; optimize looks at its pixels).
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
	case "image/webp":
		return cfg.ColorModel == color.NRGBAModel || cfg.ColorModel == color.NYCbCrAModel
	}
	return false
}

// Prepared is an image ready for the library: checked by its content,
// resized and compressed for email, its metadata removed.
type Prepared struct {
	Data          []byte
	ContentType   string
	Width, Height int
	HasAlpha      bool
	// Sha256 is the file's own: the same file found again is the same asset.
	Sha256 [32]byte
	// Original is what came in, when what's stored differs from it.
	Original *MediaOriginal
}

// Prepare readies an image that comes from elsewhere (an imported post's)
// for the library, as Upload readies an upload, without its audit entry:
// up to MaxUploadBytes in, MaxBytes stored. WebP is accepted and stored as
// JPEG, or PNG when it has transparency. Its ImageError reads in English.
func Prepare(ctx context.Context, data []byte) (Prepared, error) {
	return prepare(ctx, i18n.Default, data, true)
}

// prepare is Prepare in a reader's language; webp says whether WebP is
// taken.
func prepare(ctx context.Context, l i18n.Lang, data []byte, webp bool) (Prepared, error) {
	ctype, w, h, _, err := inspect(l, data, MaxUploadBytes, webp)
	if err != nil {
		return Prepared{}, err
	}
	release, err := DecodeSlot(ctx)
	if err != nil {
		return Prepared{}, err
	}
	o, err := optimize(ctype, data, w, h)
	release()
	if err != nil {
		return Prepared{}, ImageError(mDamaged.Text(l, nil))
	}
	if len(o.data) > MaxBytes {
		return Prepared{}, ImageError(mStillTooLarge.Text(l, i18n.Args{"size": mb(l, len(o.data))}))
	}
	p := Prepared{Data: o.data, ContentType: o.contentType, Width: o.width, Height: o.height,
		HasAlpha: hasAlpha(o.contentType, o.data, image.Config{}), Sha256: sha256.Sum256(data)}
	if !bytes.Equal(o.data, data) {
		p.Original = &MediaOriginal{SizeBytes: len(data), Width: w, Height: h, ContentType: ctype}
	}
	return p, nil
}

// Params are the library row of a prepared image: its file name made safe,
// who added it and when.
func (p Prepared) Params(ws uuid.UUID, fileName string, by *uuid.UUID, at time.Time) store.InsertMediaAssetParams {
	params := store.InsertMediaAssetParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, FileName: SafeFileName(fileName, p.ContentType), ContentType: p.ContentType,
		SizeBytes: int32(len(p.Data)), Width: int32(p.Width), Height: int32(p.Height), HasAlpha: p.HasAlpha, Sha256: p.Sha256[:], Data: p.Data,
		CreatedBy: by, CreatedAt: at,
	}
	if o := p.Original; o != nil {
		size, ow, oh, oct := int32(o.SizeBytes), int32(o.Width), int32(o.Height), o.ContentType
		params.OriginalSizeBytes, params.OriginalWidth, params.OriginalHeight, params.OriginalContentType = &size, &ow, &oh, &oct
	}
	return params
}

// Upload stores an image (or returns the identical one already stored),
// resized and compressed for email with its metadata removed.
func (s *Service) Upload(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, fileName string, data []byte) (MediaAsset, error) {
	prep, err := prepare(ctx, i18n.LangFrom(ctx), data, false)
	var refused ImageError
	if errors.As(err, &refused) {
		return MediaAsset{}, api.Invalid("body.file", refused.Error())
	}
	if err != nil {
		return MediaAsset{}, err
	}
	p := prep.Params(ws.ID, fileName, actor, s.now())
	var out MediaAsset
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.InsertMediaAsset(ctx, p)
		if err != nil {
			return err
		}
		out = MediaAsset{ID: row.ID.String(), URL: URL(row.ID, row.FileName), FileName: row.FileName, ContentType: row.ContentType, SizeBytes: int(row.SizeBytes),
			Width: int(row.Width), Height: int(row.Height), HasAlpha: row.HasAlpha, CreatedAt: row.CreatedAt,
			Original: originalOf(row.OriginalSizeBytes, row.OriginalWidth, row.OriginalHeight, row.OriginalContentType)}
		after := map[string]any{"fileName": row.FileName, "contentType": row.ContentType, "sizeBytes": row.SizeBytes, "width": row.Width, "height": row.Height, "new": row.Inserted}
		if out.Original != nil {
			after["original"] = map[string]any{"sizeBytes": out.Original.SizeBytes, "width": out.Original.Width, "height": out.Original.Height, "contentType": out.Original.ContentType}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "media.upload", TargetType: "media", TargetID: row.ID.String(), TargetName: row.FileName,
			After: after})
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
			Width: int(r.Width), Height: int(r.Height), HasAlpha: r.HasAlpha, CreatedAt: r.CreatedAt, CreatedBy: r.CreatedByName,
			Original: originalOf(r.OriginalSizeBytes, r.OriginalWidth, r.OriginalHeight, r.OriginalContentType)})
	}
	return out, nil
}

// Remove hides an image from the library. It stays reachable so emails
// that were already sent keep showing it.
func (s *Service) Remove(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx)
		}
		if err != nil {
			return api.MapError(err, "Image")
		}
		now := s.now()
		n, err := q.HideMediaAsset(ctx, store.HideMediaAssetParams{WorkspaceID: ws.ID, ID: id, Now: &now})
		if err != nil {
			return err
		}
		if n == 0 {
			return notFound(ctx)
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
