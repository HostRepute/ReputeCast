package referrals

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxFileBytes: download rewards are up to 100 MB.
const MaxFileBytes = 100 << 20

const chunkSize = 1 << 20

// allowedTypes are the files a download reward can be: PDF, ZIP or an image.
var allowedTypes = map[string]string{
	"application/pdf": ".pdf", "application/zip": ".zip", "image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp",
}

// sniff reads a file's type from its first bytes.
func sniff(head []byte, name string) string {
	switch {
	case bytes.HasPrefix(head, []byte("%PDF-")):
		return "application/pdf"
	case bytes.HasPrefix(head, []byte("PK\x03\x04")), bytes.HasPrefix(head, []byte("PK\x05\x06")):
		return "application/zip"
	case bytes.HasPrefix(head, []byte("RIFF")) && len(head) >= 12 && string(head[8:12]) == "WEBP":
		return "image/webp"
	}
	ct := http.DetectContentType(head)
	if _, ok := allowedTypes[ct]; ok {
		return ct
	}
	_ = name
	return ""
}

// safeName keeps a readable file name.
func safeName(name, contentType string) string {
	name = path.Base(strings.ReplaceAll(strings.TrimSpace(name), `\`, "/"))
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" {
		out = "reward"
	}
	if ext := allowedTypes[contentType]; ext != "" && !strings.HasSuffix(out, ext) && (ext != ".jpg" || !strings.HasSuffix(out, ".jpeg")) {
		out += ext
	}
	if len(out) > 120 {
		out = out[len(out)-120:]
	}
	return out
}

// UploadFile keeps a file for download rewards, in 1 MiB chunks.
func (s *Service) UploadFile(ctx context.Context, ws *store.Workspace, programID uuid.UUID, name string, r io.Reader) (ReferralFile, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralFile{}, err
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return ReferralFile{}, api.Invalid("body.file", s.text(ctx, msgs.M("file.empty"), nil))
	}
	head = head[:n]
	if n == 0 {
		return ReferralFile{}, api.Invalid("body.file", s.text(ctx, msgs.M("file.empty"), nil))
	}
	ct := sniff(head, name)
	if ct == "" {
		return ReferralFile{}, api.Invalid("body.file", s.text(ctx, msgs.M("file.type"), nil))
	}
	id := uuid.Must(uuid.NewV7())
	fileName := safeName(name, ct)
	now := s.now()
	h := sha256.New()
	var size int64
	src := io.MultiReader(bytes.NewReader(head), r)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// The row first (chunks reference it), with its size and hash fixed
		// once every chunk is in.
		if err := q.ReferralFileInsert(ctx, store.ReferralFileInsertParams{ID: id, WorkspaceID: ws.ID, ProgramID: programID, FileName: fileName,
			ContentType: ct, SizeBytes: 1, Sha256: []byte{0}, UploadedBy: actorUser(ctx), Now: now}); err != nil {
			return err
		}
		buf := make([]byte, chunkSize)
		seq := int32(0)
		for {
			k, rerr := io.ReadFull(src, buf)
			if k > 0 {
				size += int64(k)
				if size > MaxFileBytes {
					return api.Invalid("body.file", s.text(ctx, msgs.M("file.tooLarge"), nil))
				}
				h.Write(buf[:k])
				if err := q.ReferralFileChunkInsert(ctx, store.ReferralFileChunkInsertParams{FileID: id, Seq: seq, Data: append([]byte(nil), buf[:k]...)}); err != nil {
					return err
				}
				seq++
			}
			if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
				break
			}
			if rerr != nil {
				return rerr
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE referral_files SET size_bytes = $2, sha256 = $3 WHERE id = $1`, id, size, h.Sum(nil)); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.file_uploaded", TargetType: "referral_file", TargetID: id.String(),
			TargetName: fileName, After: map[string]any{"name": fileName, "type": ct, "size": size}, Metadata: map[string]any{"related": cur.ReferralProgram.ID.String()}})
	})
	if err != nil {
		return ReferralFile{}, err
	}
	f, err := s.Q.ReferralFileGet(ctx, id)
	if err != nil {
		return ReferralFile{}, err
	}
	return ReferralFile{ID: f.ID.String(), Name: f.FileName, ContentType: f.ContentType, Size: f.SizeBytes, UploadedBy: f.UploadedByName, UploadedAt: f.UploadedAt}, nil
}

// RemoveFile takes a file out of the program's files. Rewards already sent
// keep working until their links expire.
func (s *Service) RemoveFile(ctx context.Context, ws *store.Workspace, programID, fileID uuid.UUID) error {
	f, err := s.Q.ReferralFileGet(ctx, fileID)
	if err != nil || f.WorkspaceID != ws.ID || f.ProgramID != programID {
		return api.NotFound("File")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralFileRemove(ctx, store.ReferralFileRemoveParams{ID: fileID, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.file_removed", TargetType: "referral_file", TargetID: fileID.String(),
			TargetName: f.FileName, Before: map[string]any{"name": f.FileName, "size": f.SizeBytes}})
	})
}
