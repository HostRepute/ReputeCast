package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"image"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Hidden keeps an image someone outside the team sent (a sponsor's logo or
// image with a self-serve request, M65): checked and stored the way
// uploads are (resized for email, its metadata removed), served at its
// /media/ address, and never shown in the library. It's no member's
// upload, so nothing is audited; the caller records what it's for. A
// refused image's error is an ImageError in the reader's language.
func (s *Service) Hidden(ctx context.Context, q *store.Queries, ws *store.Workspace, fileName string, data []byte) (MediaAsset, error) {
	l := i18n.LangFrom(ctx)
	ctype, w, h, _, err := inspect(l, data, MaxUploadBytes, false)
	if err != nil {
		return MediaAsset{}, err
	}
	release, err := DecodeSlot(ctx)
	if err != nil {
		return MediaAsset{}, err
	}
	o, err := optimize(ctype, data, w, h)
	release()
	if err != nil {
		return MediaAsset{}, ImageError(mDamaged.Text(l, nil))
	}
	if len(o.data) > MaxBytes {
		return MediaAsset{}, ImageError(mStillTooLarge.Text(l, i18n.Args{"size": mb(l, len(o.data))}))
	}
	// The sent file's hash, as uploads keep it: the same file sent twice
	// is found again.
	sum := sha256.Sum256(data)
	p := store.InsertHiddenMediaAssetParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, FileName: SafeFileName(fileName, o.contentType), ContentType: o.contentType,
		SizeBytes: int32(len(o.data)), Width: int32(o.width), Height: int32(o.height), HasAlpha: hasAlpha(o.contentType, o.data, image.Config{}), Sha256: sum[:],
		Data: o.data, CreatedAt: s.now()}
	if !bytes.Equal(o.data, data) {
		size, ow, oh, oct := int32(len(data)), int32(w), int32(h), ctype
		p.OriginalSizeBytes, p.OriginalWidth, p.OriginalHeight, p.OriginalContentType = &size, &ow, &oh, &oct
	}
	row, err := q.InsertHiddenMediaAsset(ctx, p)
	if err != nil {
		return MediaAsset{}, err
	}
	return MediaAsset{ID: row.ID.String(), URL: URL(row.ID, row.FileName), FileName: row.FileName, ContentType: o.contentType, SizeBytes: int(row.SizeBytes),
		Width: o.width, Height: o.height}, nil
}
