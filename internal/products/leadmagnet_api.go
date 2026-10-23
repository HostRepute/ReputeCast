package products

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

// The form builder's "After confirmation: offer a download" (design
// expansion phase 7, X19): one file per form, uploaded in chunks like a
// product's, and how many days its links work.

// FormDownloadInput sets how long the confirmation page's links work.
type FormDownloadInput struct {
	Days int32 `json:"days" minimum:"1" maximum:"30" doc:"How many days a link works"`
}

type formDownloadInput struct {
	forms.FormPath
	Body FormDownloadInput
}

type formUploadStartInput struct {
	forms.FormPath
	Body UploadStart
}

// FormUploadPath is an upload of a form's download (exported: Huma reads
// embedded path structs only when their type is exported).
type FormUploadPath struct {
	forms.FormPath
	Upload string `path:"upload" format:"uuid"`
}

type formChunkInput struct {
	FormUploadPath
	Offset  int64  `query:"offset" minimum:"0" doc:"Where this chunk starts: the bytes received so far"`
	RawBody []byte `contentType:"application/octet-stream" doc:"The next chunk, up to 4 MB"`
}

func formID(in forms.FormPath) uuid.UUID { return uuid.MustParse(in.Form) }

// formFile is the form's current file, if any.
func (s *Service) formFile(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.FormDownloadGetRow, error) {
	d, err := s.Q.FormDownloadGet(ctx, store.FormDownloadGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return d, mapErr(ctx, err, "form")
	}
	return d, nil
}

// StartFormUpload starts the upload of a form's file; it replaces the one
// the form offers once it finished.
func (s *Service) StartFormUpload(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in UploadStart) (UploadState, error) {
	d, err := s.formFile(ctx, ws, id)
	if err != nil {
		return UploadState{}, err
	}
	if in.Replaces == "" && d.DownloadFileID != nil {
		in.Replaces = d.DownloadFileID.String()
	}
	return s.StartUpload(ctx, ws, p, OwnerForm, id, in)
}

// FinishFormUpload checks the file and makes it the form's download.
func (s *Service) FinishFormUpload(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, upload string) (FormDownload, error) {
	f, err := s.FinishUpload(ctx, ws, p, OwnerForm, id, upload)
	if err != nil {
		return FormDownload{}, err
	}
	fid, err := uuid.Parse(f.ID)
	if err != nil {
		return FormDownload{}, err
	}
	d, err := s.formFile(ctx, ws, id)
	if err != nil {
		return FormDownload{}, err
	}
	if err := s.Q.FormDownloadSet(ctx, store.FormDownloadSetParams{ID: id, WorkspaceID: ws.ID, FileID: &fid, Days: d.DownloadDays}); err != nil {
		return FormDownload{}, err
	}
	return s.FormDownloadOf(ctx, ws, id)
}

// RemoveFormDownload stops offering a download after confirmation. Links
// already sent stop working.
func (s *Service) RemoveFormDownload(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (FormDownload, error) {
	d, err := s.formFile(ctx, ws, id)
	if err != nil {
		return FormDownload{}, err
	}
	if d.DownloadFileID == nil {
		return s.FormDownloadOf(ctx, ws, id)
	}
	if err := s.RemoveFile(ctx, ws, p, OwnerForm, id, *d.DownloadFileID); err != nil {
		return FormDownload{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return store.New(tx).FormDownloadSet(ctx, store.FormDownloadSetParams{ID: id, WorkspaceID: ws.ID, FileID: nil, Days: d.DownloadDays})
	})
	if err != nil {
		return FormDownload{}, err
	}
	return s.FormDownloadOf(ctx, ws, id)
}

func registerFormDownloads(h huma.API, s *Service) {
	tags := []string{"Signup forms"}
	base := "/workspaces/{workspace}/forms/{form}/download"
	edit := api.Workspace(authz.FormsEdit)
	api.Get(h, api.Op{ID: "get-form-download", Path: base, Summary: "The file a form offers after confirmation, and its numbers",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *forms.FormPath) (*api.Body[FormDownload], error) {
			out, err := s.FormDownloadOf(ctx, api.WorkspaceFrom(ctx), formID(*in))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-form-download", Path: base, Summary: "How many days the confirmation page's download links work", Tags: tags, Access: edit},
		func(ctx context.Context, in *formDownloadInput) (*api.Body[FormDownload], error) {
			out, err := s.FormDownloadDays(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), formID(in.FormPath), in.Body.Days)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-form-download", Path: base, Summary: "Stop offering a download after confirmation; links already sent stop working",
		Tags: tags, Access: edit, Status: http.StatusOK},
		func(ctx context.Context, in *forms.FormPath) (*api.Body[FormDownload], error) {
			out, err := s.RemoveFormDownload(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), formID(*in))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-form-download-upload", Path: base + "/uploads", Summary: "Announce the file to offer: it replaces the current one once it finished",
		Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *formUploadStartInput) (*api.Body[UploadState], error) {
			out, err := s.StartFormUpload(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), formID(in.FormPath), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "upload-form-download-chunk", Path: base + "/uploads/{upload}", Summary: "Send the next chunk of the file",
		Tags: tags, Access: edit, MaxBodyBytes: ChunkBytes + 64<<10},
		func(ctx context.Context, in *formChunkInput) (*api.Body[UploadState], error) {
			out, err := s.WriteChunk(ctx, api.WorkspaceFrom(ctx), OwnerForm, formID(in.FormPath), in.Upload, in.Offset, in.RawBody)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "finish-form-download-upload", Path: base + "/uploads/{upload}/complete",
		Summary: "Finish the upload: the file's type is checked from its content, then the form offers it", Tags: tags, Access: edit},
		func(ctx context.Context, in *FormUploadPath) (*api.Body[FormDownload], error) {
			out, err := s.FinishFormUpload(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), formID(in.FormPath), in.Upload)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "cancel-form-download-upload", Path: base + "/uploads/{upload}", Summary: "Cancel an upload; nothing is kept", Tags: tags, Access: edit},
		func(ctx context.Context, in *FormUploadPath) (*api.Empty, error) {
			return api.NoContent, s.CancelUpload(ctx, api.WorkspaceFrom(ctx), OwnerForm, formID(in.FormPath), in.Upload)
		})
}
