package media

import (
	"context"
	"io"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// MediaAssetList is the image library.
type MediaAssetList struct {
	Items []MediaAsset `json:"items"`
}

type listInput struct {
	api.WorkspaceInput
	Limit int `query:"limit" minimum:"1" maximum:"500" default:"200"`
}

type uploadInput struct {
	api.WorkspaceInput
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"image/png,image/jpeg,image/gif,application/octet-stream" required:"true" doc:"PNG, JPEG or GIF, up to 5 MB"`
	}]
}

type assetInput struct {
	api.WorkspaceInput
	ID string `path:"assetId" format:"uuid"`
}

// Register adds the image library operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Media"}
	api.Get(h, api.Op{ID: "list-media", Path: "/workspaces/{workspace}/media", Summary: "List the images in the library", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *listInput) (*api.Body[MediaAssetList], error) {
			items, err := s.List(ctx, api.WorkspaceFrom(ctx), in.Limit)
			return api.OK(MediaAssetList{Items: items}), err
		})
	api.Post(h, api.Op{ID: "upload-media", Path: "/workspaces/{workspace}/media", Summary: "Upload an image (PNG, JPEG or GIF, up to 5 MB)",
		Description: "The file is validated by its content, not its name. Uploading the same file twice returns the stored image.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit), MaxBodyBytes: MaxBytes + 64<<10, Status: 201},
		func(ctx context.Context, in *uploadInput) (*api.Body[MediaAsset], error) {
			f := in.RawBody.Data().File
			if !f.IsSet {
				return nil, api.Invalid("body.file", say(ctx, mChooseImage))
			}
			data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
			if err != nil {
				return nil, api.Invalid("body.file", say(ctx, mInterrupted))
			}
			var actor *uuid.UUID
			if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
				id := p.UserID
				actor = &id
			}
			out, err := s.Upload(ctx, api.WorkspaceFrom(ctx), actor, f.Filename, data)
			if err != nil {
				return nil, err
			}
			return api.OK(out), nil
		})
	api.Delete(h, api.Op{ID: "remove-media", Path: "/workspaces/{workspace}/media/{assetId}", Summary: "Remove an image from the library",
		Description: "The image disappears from the library but stays reachable, so emails already sent keep showing it.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *assetInput) (*api.Empty, error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, notFound(ctx)
			}
			return api.NoContent, s.Remove(ctx, api.WorkspaceFrom(ctx), id)
		})
}
