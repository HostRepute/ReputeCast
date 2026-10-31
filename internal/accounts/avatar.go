package accounts

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // decode uploaded JPEGs
	"image/png"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
)

const (
	avatarMaxBytes = 2 << 20 // 2 MB, as the upload hint says
	avatarSize     = 256
	avatarMaxSide  = 6000 // refuse decompression bombs before decoding
)

// AccountAvatarInput is a PNG or JPEG photo, base64-encoded in JSON.
type AccountAvatarInput struct {
	Data []byte `json:"data" doc:"PNG or JPEG, up to 2 MB, base64"`
}

// SetAvatar stores a profile photo. It is cropped to a square, scaled to
// 256 px and re-encoded as PNG, which also drops metadata such as location.
func (s *Service) SetAvatar(ctx context.Context, data []byte) (AccountProfile, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountProfile{}, err
	}
	if len(data) == 0 {
		return AccountProfile{}, api.Invalid("body.data", say(ctx, msgs.M("problem.photoMissing")))
	}
	if len(data) > avatarMaxBytes {
		return AccountProfile{}, api.Invalid("body.data", say(ctx, msgs.M("problem.photoLarge")))
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return AccountProfile{}, api.Invalid("body.data", say(ctx, msgs.M("problem.photoFormat")))
	}
	if cfg.Width > avatarMaxSide || cfg.Height > avatarMaxSide || cfg.Width < 16 || cfg.Height < 16 {
		return AccountProfile{}, api.Invalid("body.data", say(ctx, msgs.M("problem.photoSize"), "min", 16, "max", avatarMaxSide))
	}
	// Decoding, and the square squareThumbnail crops, must fit the memory an
	// upload may take: a progressive JPEG can claim a gigabyte within
	// 6,000 px a side.
	side := int64(min(cfg.Width, cfg.Height))
	if cost, _, _, ok := media.DecodeCost(data); !ok || cost+4*side*side > media.DecodeBudget {
		return AccountProfile{}, api.Invalid("body.data", say(ctx, msgs.M("problem.photoUnreadable")))
	}
	// One image is decoded at a time across ReputeCast (media.DecodeSlot).
	release, err := media.DecodeSlot(ctx)
	if err != nil {
		return AccountProfile{}, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		release()
		return AccountProfile{}, api.Invalid("body.data", say(ctx, msgs.M("problem.photoUnreadable")))
	}
	var buf bytes.Buffer
	err = png.Encode(&buf, squareThumbnail(img, avatarSize))
	release()
	if err != nil {
		return AccountProfile{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.UpsertAvatar(ctx, store.UpsertAvatarParams{UserID: u.ID, ContentType: "image/png", Data: buf.Bytes()}); err != nil {
			return err
		}
		if err := q.SetAvatarUpdatedAt(ctx, store.SetAvatarUpdatedAtParams{ID: u.ID, AvatarUpdatedAt: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.avatar_update", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email})
	})
	if err != nil {
		return AccountProfile{}, err
	}
	nu := *u
	nu.AvatarUpdatedAt = &now
	return AccountProfile{Name: nu.Name, Email: nu.Email, Timezone: nu.Timezone, AvatarColor: nu.AvatarColor, AvatarURL: avatarURL(nu), IsInstanceAdmin: nu.IsInstanceAdmin}, nil
}

// RemoveAvatar deletes the profile photo; initials show instead.
func (s *Service) RemoveAvatar(ctx context.Context) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.DeleteAvatar(ctx, u.ID)
		if err != nil || n == 0 {
			return err
		}
		if err := q.SetAvatarUpdatedAt(ctx, store.SetAvatarUpdatedAtParams{ID: u.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.avatar_remove", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email})
	})
}

// AvatarImage is a profile photo response.
type AvatarImage struct {
	ContentType  string    `header:"Content-Type"`
	CacheControl string    `header:"Cache-Control"`
	LastModified time.Time `header:"Last-Modified"`
	Body         []byte
}

// Avatar returns a person's photo to themselves and to people who share a
// workspace with them.
func (s *Service) Avatar(ctx context.Context, userID uuid.UUID) (*AvatarImage, error) {
	viewer, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if viewer.ID != userID && !viewer.IsInstanceAdmin {
		ok, err := s.Q.UsersShareWorkspace(ctx, store.UsersShareWorkspaceParams{Viewer: viewer.ID, Subject: userID})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, api.NotFound("Photo")
		}
	}
	a, err := s.Q.GetAvatar(ctx, userID)
	if err != nil {
		return nil, api.MapError(err, "Photo")
	}
	return &AvatarImage{ContentType: a.ContentType, CacheControl: "private, max-age=86400", LastModified: a.UpdatedAt, Body: a.Data}, nil
}

// squareThumbnail center-crops img to a square and scales it to size×size
// with an area-averaging filter (no dependency needed for one small image).
func squareThumbnail(img image.Image, size int) image.Image {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	x0 := b.Min.X + (b.Dx()-side)/2
	y0 := b.Min.Y + (b.Dy()-side)/2
	src := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(src, src.Bounds(), img, image.Pt(x0, y0), draw.Src)
	if side <= size {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	scale := float64(side) / float64(size)
	for y := 0; y < size; y++ {
		sy0, sy1 := int(float64(y)*scale), int(float64(y+1)*scale)
		for x := 0; x < size; x++ {
			sx0, sx1 := int(float64(x)*scale), int(float64(x+1)*scale)
			var r, g, bl, a, n uint32
			for sy := sy0; sy < max(sy1, sy0+1); sy++ {
				for sx := sx0; sx < max(sx1, sx0+1); sx++ {
					c := src.RGBAAt(sx, sy)
					r, g, bl, a = r+uint32(c.R), g+uint32(c.G), bl+uint32(c.B), a+uint32(c.A)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(bl / n), A: uint8(a / n)})
		}
	}
	return dst
}
