package products

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Owner kinds of download files.
const (
	OwnerProduct = "product"
	OwnerForm    = "form"
)

// UploadStart announces a file.
type UploadStart struct {
	Name string `json:"name" minLength:"1" maxLength:"255"`
	Size int64  `json:"size" minimum:"1"`
	// Replaces is a file the new one replaces once it finished uploading.
	Replaces string `json:"replaces,omitempty" format:"uuid"`
}

// UploadState is where an upload stands.
type UploadState struct {
	ID         string `json:"id"`
	Received   int64  `json:"received"`
	Size       int64  `json:"size"`
	ChunkBytes int64  `json:"chunkBytes"`
}

// humanSize is a size as the messages say it ("4.2 MB").
func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/(1<<30)), ".0") + " GB"
	case n >= 1<<20:
		// "4.2 MB", "38 MB".
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/(1<<20)), ".0") + " MB"
	case n >= 1<<10:
		return fmt.Sprintf("%d KB", n/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// ownerCheck is the owner's file budget: the bytes its files hold now
// (minus a file being replaced).
func (s *Service) ownerBytes(ctx context.Context, wsID uuid.UUID, kind string, owner uuid.UUID, replaces string) (int64, error) {
	total, err := s.Q.DownloadFilesSize(ctx, store.DownloadFilesSizeParams{OwnerKind: kind, OwnerID: owner})
	if err != nil {
		return 0, err
	}
	if id, err := uuid.Parse(replaces); err == nil {
		if f, err := s.Q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: wsID, ID: id}); err == nil && f.OwnerID == owner && f.RemovedAt == nil {
			total -= f.SizeBytes
		}
	}
	return total, nil
}

// StartUpload starts an upload for an owner (a product, a form).
func (s *Service) StartUpload(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, owner uuid.UUID, in UploadStart) (UploadState, error) {
	l := lang(ctx)
	if in.Size > MaxFileBytes {
		return UploadState{}, api.Invalid("body.size", word(l, "file.tooBig", map[string]any{"size": humanSize(in.Size)}))
	}
	have, err := s.ownerBytes(ctx, ws.ID, kind, owner, in.Replaces)
	if err != nil {
		return UploadState{}, err
	}
	if have+in.Size > MaxProductBytes {
		return UploadState{}, api.Invalid("body.size", word(l, "file.productFull", map[string]any{"size": humanSize(have + in.Size)}))
	}
	var uid uuid.UUID
	if p != nil {
		uid = p.UserID
	}
	u, err := s.Files.begin(Upload{WorkspaceID: ws.ID, OwnerKind: kind, OwnerID: owner, Name: SafeName(in.Name), Size: in.Size, Replaces: in.Replaces,
		UserID: uid, Created: s.now()})
	if err != nil {
		return UploadState{}, err
	}
	return UploadState{ID: u.ID, Size: u.Size, ChunkBytes: ChunkBytes}, nil
}

// upload reads an upload of the owner.
func (s *Service) upload(ctx context.Context, ws *store.Workspace, kind string, owner uuid.UUID, id string) (Upload, int64, error) {
	u, have, err := s.Files.upload(id)
	if err != nil || u.WorkspaceID != ws.ID || u.OwnerKind != kind || u.OwnerID != owner {
		return Upload{}, 0, api.Err(404, "upload_gone", word(lang(ctx), "file.uploadGone", nil))
	}
	return u, have, nil
}

// WriteChunk appends a chunk at offset.
func (s *Service) WriteChunk(ctx context.Context, ws *store.Workspace, kind string, owner uuid.UUID, id string, offset int64, data []byte) (UploadState, error) {
	u, _, err := s.upload(ctx, ws, kind, owner, id)
	if err != nil {
		return UploadState{}, err
	}
	got, err := s.Files.write(id, offset, data)
	switch {
	case errors.Is(err, errOffset):
		return UploadState{}, api.Err(409, "upload_offset", word(lang(ctx), "file.offset", map[string]any{"offset": got})).WithMeta("received", got)
	case errors.Is(err, errTooBig):
		s.Files.abort(id)
		return UploadState{}, api.Invalid("body", word(lang(ctx), "file.tooBig", map[string]any{"size": humanSize(offset + int64(len(data)))}))
	case err != nil:
		return UploadState{}, err
	}
	return UploadState{ID: id, Received: got, Size: u.Size, ChunkBytes: ChunkBytes}, nil
}

// CancelUpload drops an upload and what arrived of it.
func (s *Service) CancelUpload(ctx context.Context, ws *store.Workspace, kind string, owner uuid.UUID, id string) error {
	if _, _, err := s.upload(ctx, ws, kind, owner, id); err != nil {
		return err
	}
	s.Files.abort(id)
	return nil
}

// FinishUpload checks a complete upload's content and keeps the file: it
// joins the owner's files (replacing one when asked).
func (s *Service) FinishUpload(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, owner uuid.UUID, id string) (ProductFile, error) {
	l := lang(ctx)
	u, have, err := s.upload(ctx, ws, kind, owner, id)
	if err != nil {
		return ProductFile{}, err
	}
	if have != u.Size {
		return ProductFile{}, api.Err(409, "upload_incomplete", word(l, "file.incomplete", map[string]any{"have": have, "size": u.Size})).WithMeta("received", have)
	}
	_, det, sum, part, err := s.Files.finish(id)
	if err != nil {
		s.Files.abort(id)
		return ProductFile{}, err
	}
	if det.Kind == KindProgram {
		s.Files.abort(id)
		return ProductFile{}, api.Err(422, "file_program", word(l, "file.program", nil))
	}
	if claimed, bad := Mismatch(u.Name, det); bad {
		s.Files.abort(id)
		actualExt := "file"
		if k, ok := kinds[det.Kind]; ok && len(k.exts) > 0 {
			actualExt = k.exts[0]
		}
		return ProductFile{}, api.Err(422, "file_mismatch", word(l, "file.mismatch", map[string]any{
			"ext": extOf(u.Name), "content": word(l, "content."+det.Kind, nil), "claimed": strings.ToUpper(claimed),
			"actual": strings.ToUpper(det.Kind), "actualExt": actualExt})).WithMeta("claimed", claimed, "actual", det.Kind)
	}
	key, err := s.Files.put(ws.ID, part, sum)
	if err != nil {
		s.Files.abort(id)
		return ProductFile{}, err
	}
	meta, _, _ := s.Files.uploadPaths(id)
	_ = os.Remove(meta)
	fid := newID()
	var pages *int32
	if det.Pages > 0 {
		n := int32(det.Pages)
		pages = &n
	}
	now := s.now()
	var replaced *store.DownloadFile
	var name string
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := ownerName(ctx, q, ws, kind, owner)
		if err != nil {
			return err
		}
		name = n
		// Under the owner's lock: uploads finishing side by side can't take
		// it past its total.
		total, err := q.DownloadFilesSize(ctx, store.DownloadFilesSizeParams{OwnerKind: kind, OwnerID: owner})
		if err != nil {
			return err
		}
		if rid, err := uuid.Parse(u.Replaces); err == nil {
			if f, err := q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: rid}); err == nil && f.OwnerID == owner && f.RemovedAt == nil {
				total -= f.SizeBytes
			}
		}
		if total+u.Size > MaxProductBytes {
			return api.Invalid("body.size", word(l, "file.productFull", map[string]any{"size": humanSize(total + u.Size)}))
		}
		pos, err := q.DownloadFilesNextPosition(ctx, store.DownloadFilesNextPositionParams{OwnerKind: kind, OwnerID: owner})
		if err != nil {
			return err
		}
		if rid, err := uuid.Parse(u.Replaces); err == nil {
			if f, err := q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: rid}); err == nil && f.OwnerID == owner && f.RemovedAt == nil {
				replaced = &f
				pos = f.Position
				if err := q.DownloadFileRemove(ctx, store.DownloadFileRemoveParams{ID: rid, At: now}); err != nil {
					return err
				}
			}
		}
		if err := q.DownloadFileInsert(ctx, store.DownloadFileInsertParams{ID: fid, WorkspaceID: ws.ID, OwnerKind: kind, OwnerID: owner, Name: u.Name,
			SizeBytes: u.Size, Kind: det.Kind, ContentType: det.ContentType, Pages: pages, Sha256: sum, StorageKey: key, Position: pos,
			UploadedBy: actor(p), At: now}); err != nil {
			return err
		}
		after := map[string]any{"file": u.Name, "size": u.Size, "kind": det.Kind}
		if det.Pages > 0 {
			after["pages"] = det.Pages
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "product.file_added", TargetType: "product", TargetID: owner.String(), TargetName: name, After: after}
		if kind == OwnerForm {
			ev.Action, ev.TargetType = "form.download_added", "form"
		}
		if replaced != nil {
			ev.Action = strings.Replace(ev.Action, "added", "replaced", 1)
			ev.Before = map[string]any{"file": replaced.Name, "size": replaced.SizeBytes, "kind": replaced.Kind}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return ProductFile{}, err
	}
	if replaced != nil {
		s.dropBytes(ctx, *replaced)
	}
	f, err := s.Q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: fid})
	if err != nil {
		return ProductFile{}, err
	}
	by := ""
	if p != nil {
		by = p.Name
	}
	return fileOf(f, by), nil
}

// dropBytes deletes a removed file's bytes once nothing else keeps them.
func (s *Service) dropBytes(ctx context.Context, f store.DownloadFile) {
	inUse, err := s.Q.DownloadFileInUse(ctx, store.DownloadFileInUseParams{StorageKey: f.StorageKey, ID: f.ID})
	if err != nil || inUse {
		return
	}
	if err := s.Files.Remove(f.StorageKey); err != nil {
		s.log().WarnContext(ctx, "products: removing a file's bytes", "err", err)
	}
}

// ownerName locks a product (or reads a form) of the workspace: the name
// its audit entries carry.
func ownerName(ctx context.Context, q *store.Queries, ws *store.Workspace, kind string, owner uuid.UUID) (string, error) {
	if kind == OwnerForm {
		d, err := q.FormDownloadGet(ctx, store.FormDownloadGetParams{ID: owner, WorkspaceID: ws.ID})
		if err != nil {
			return "", api.MapError(err, "Form")
		}
		return d.Name, nil
	}
	pr, err := q.ProductLock(ctx, store.ProductLockParams{WorkspaceID: ws.ID, ID: owner})
	if err != nil {
		return "", api.MapError(err, "Product")
	}
	return pr.Name, nil
}

// RemoveFile takes a file off its owner. Buyers' download pages show the
// files the product has now.
func (s *Service) RemoveFile(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, owner, fileID uuid.UUID) error {
	var removed store.DownloadFile
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: fileID})
		if err != nil || f.OwnerID != owner || f.OwnerKind != kind || f.RemovedAt != nil {
			return api.Err(404, "not_found", word(lang(ctx), "file.notFound", nil))
		}
		name, err := ownerName(ctx, q, ws, kind, owner)
		if err != nil {
			return err
		}
		if err := q.DownloadFileRemove(ctx, store.DownloadFileRemoveParams{ID: fileID, At: s.now()}); err != nil {
			return err
		}
		removed = f
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "product.file_removed", TargetType: "product", TargetID: owner.String(), TargetName: name,
			Before: map[string]any{"file": f.Name, "size": f.SizeBytes, "kind": f.Kind}}
		if kind == OwnerForm {
			ev.Action, ev.TargetType = "form.download_removed", "form"
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return err
	}
	s.dropBytes(ctx, removed)
	return nil
}
