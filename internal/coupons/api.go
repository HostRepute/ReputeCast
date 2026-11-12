package coupons

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// CouponPoolPath names a pool in a workspace.
type CouponPoolPath struct {
	api.WorkspaceInput
	Pool string `path:"pool" format:"uuid" doc:"Pool id"`
}

type poolCreateInput struct {
	api.WorkspaceInput
	Body CouponPoolInput
}

type poolChangeInput struct {
	CouponPoolPath
	Body CouponPoolChange
}

// CouponArchiveInput confirms archiving a pool live automations use.
type CouponArchiveInput struct {
	Confirm bool `json:"confirm,omitempty" doc:"Archive even though live automations send its codes"`
}

type poolArchiveInput struct {
	CouponPoolPath
	Body CouponArchiveInput
}

type poolUploadInput struct {
	CouponPoolPath
	RawBody []byte `contentType:"text/plain" doc:"Codes, one per line, or a CSV whose first column is the code (up to 6 MB, 100,000 codes)"`
}

type poolGenerateInput struct {
	CouponPoolPath
	Body CouponGenerateInput
}

// CouponCodesFile is a pool's codes as CSV.
type CouponCodesFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

func (in CouponPoolPath) id() uuid.UUID { return uuid.MustParse(in.Pool) }

// Register adds the discount code operations: pools are seen with
// campaigns:view; every change and the export need coupons:manage and a
// person.
func Register(h huma.API, s *Service) {
	tags := []string{"Discount codes"}
	base := "/workspaces/{workspace}/coupons"
	view := api.Workspace(authz.CampaignsView)
	manage := api.WorkspaceHuman(authz.CouponsManage)

	api.Get(h, api.Op{ID: "list-coupon-pools", Path: base + "/pools", Summary: "List discount code pools with their numbers (never their codes)", Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CouponPoolList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-coupon-sources", Path: base + "/sources", Summary: "The connected stores a pool can create codes in, and why a store can't",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CouponSourceList], error) {
			out, err := s.SourceList(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-coupon-pool", Path: base + "/pools", Summary: "Create a discount code pool (uploaded codes, or codes created in a store)",
		Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *poolCreateInput) (*api.Body[CouponPool], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-coupon-pool", Path: base + "/pools/{pool}", Summary: "A discount code pool: numbers, settings, the latest batch and what uses it",
		Tags: tags, Access: view},
		func(ctx context.Context, in *CouponPoolPath) (*api.Body[CouponPool], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-coupon-pool", Path: base + "/pools/{pool}", Summary: "Change a pool's name, prefix, low mark, one code per contact or settings",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *poolChangeInput) (*api.Body[CouponPool], error) {
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "archive-coupon-pool", Path: base + "/pools/{pool}/archive", Summary: "Archive a pool: it gives no more codes (409 when live automations use it, unless confirmed)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *poolArchiveInput) (*api.Body[CouponPool], error) {
			out, err := s.Archive(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body.Confirm)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "upload-coupon-codes", Path: base + "/pools/{pool}/codes", Summary: "Add codes to an upload pool from text or a CSV (counts only are logged)",
		Tags: tags, Access: manage, MaxBodyBytes: MaxUploadBytes},
		func(ctx context.Context, in *poolUploadInput) (*api.Body[CouponCodesResult], error) {
			out, err := s.Upload(ctx, api.WorkspaceFrom(ctx), in.id(), in.RawBody)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "remove-unused-coupon-codes", Path: base + "/pools/{pool}/codes/remove-unused", Summary: "Remove the codes nobody got yet (in the store too)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *CouponPoolPath) (*api.Body[CouponRemoveResult], error) {
			out, err := s.RemoveUnused(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "generate-coupon-codes", Path: base + "/pools/{pool}/generate", Summary: "Create more codes in the pool's store, in the background",
		Tags: tags, Access: manage, Status: http.StatusAccepted},
		func(ctx context.Context, in *poolGenerateInput) (*api.Body[CouponBatch], error) {
			out, err := s.RequestCodes(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body.Count)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-coupon-codes", Path: base + "/pools/{pool}/export.csv", Summary: "The pool's codes as CSV: code, status, given and redeemed times, order (logged)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *CouponPoolPath) (*CouponCodesFile, error) {
			body, name, err := s.Export(ctx, api.WorkspaceFrom(ctx), in.id())
			if err != nil {
				return nil, err
			}
			return &CouponCodesFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, CacheControl: "no-store", Body: body}, nil
		})
}
