package products

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type overviewInput struct {
	api.WorkspaceInput
	Period string `query:"period" enum:"30d,90d,12m" default:"30d"`
}

type productInput struct {
	api.WorkspaceInput
	Product string `path:"product" format:"uuid"`
}

type createInput struct {
	api.WorkspaceInput
	Body ProductCreate
}

type saveInput struct {
	productInput
	Body ProductFields
}

type publishInput struct {
	productInput
	Body ProductPublishInput
}

type uploadStartInput struct {
	productInput
	Body UploadStart
}

type uploadInput struct {
	productInput
	Upload string `path:"upload" format:"uuid"`
}

type chunkInput struct {
	uploadInput
	Offset  int64  `query:"offset" minimum:"0" doc:"Where this chunk starts: the bytes received so far"`
	RawBody []byte `contentType:"application/octet-stream" doc:"The next chunk, up to 4 MB"`
}

type fileInput struct {
	productInput
	File string `path:"file" format:"uuid"`
}

type coverInput struct {
	productInput
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"image/png,image/jpeg,image/gif,application/octet-stream" required:"true" doc:"PNG, JPEG or GIF, up to 5 MB"`
	}]
}

type ordersInput struct {
	api.WorkspaceInput
	Q        string `query:"q" maxLength:"200" doc:"An order number or part of the buyer's email"`
	Product  string `query:"product" doc:"A product's id"`
	Status   string `query:"status" enum:",paid,refunded,partially_refunded,disputed,test"`
	Period   string `query:"period" enum:"7d,30d,90d,all" default:"30d"`
	Page     int    `query:"page" minimum:"1" default:"1"`
	PageSize int    `query:"pageSize" minimum:"1" maximum:"100" default:"10"`
}

func (in ordersInput) query() OrdersQuery {
	return OrdersQuery{Q: in.Q, Product: in.Product, Status: in.Status, Period: in.Period, Page: in.Page, PageSize: in.PageSize}
}

type orderInput struct {
	api.WorkspaceInput
	Order string `path:"order" format:"uuid"`
}

type newLinkInput struct {
	orderInput
	Body NewLinkInput
}

// ProductsFile is a CSV download.
type ProductsFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// ProductCover is an uploaded cover image.
type ProductCover struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ProductsSummary is the Earn overview's "Product sales" card.
type ProductsSummary struct {
	Orders         int64     `json:"orders"`
	GrossCents     int64     `json:"grossCents"`
	RefundedCents  int64     `json:"refundedCents"`
	RefundedOrders int64     `json:"refundedOrders"`
	Currency       string    `json:"currency"`
	Products       int64     `json:"products"`
	From           time.Time `json:"from"`
	Until          time.Time `json:"until"`
}

// Register adds the products and orders operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Products"}
	base := "/workspaces/{workspace}/products"
	view, edit, publish := api.Workspace(authz.ProductsView), api.Workspace(authz.ProductsEdit), api.Workspace(authz.ProductsPublish)
	pid := func(in productInput) uuid.UUID { return uuid.MustParse(in.Product) }

	api.Get(h, api.Op{ID: "list-products", Path: base, Summary: "Products with their numbers, where sales came from and the payment connection", Tags: tags, Access: view},
		func(ctx context.Context, in *overviewInput) (*api.Body[ProductsOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), in.Period)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-products-summary", Path: base + "/summary", Summary: "Product sales over 30 days (the Earn overview's card)", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ProductsSummary], error) {
			out, err := s.Summary(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-product", Path: base, Summary: "Start a draft product", Tags: tags, Access: edit, Status: 201},
		func(ctx context.Context, in *createInput) (*api.Body[ProductDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-product", Path: base + "/{product}", Summary: "A product: its draft and what's live, files, publish checks and numbers", Tags: tags, Access: view},
		func(ctx context.Context, in *productInput) (*api.Body[ProductDetail], error) {
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), pid(*in))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-product", Path: base + "/{product}", Summary: "Save a product's draft (a published product changes with Publish changes)", Tags: tags, Access: edit},
		func(ctx context.Context, in *saveInput) (*api.Body[ProductDetail], error) {
			out, err := s.Save(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), pid(in.productInput), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-product", Path: base + "/{product}/publish", Summary: "Publish the draft: its price in Stripe, the shop and its pages",
		Description: "Blocked until every publish check passes. A new price on a published product needs confirmPrice: existing buyers keep their access.",
		Tags:        tags, Access: publish},
		func(ctx context.Context, in *publishInput) (*api.Body[ProductDetail], error) {
			out, err := s.Publish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), pid(in.productInput), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "archive-product", Path: base + "/{product}/archive", Summary: "Stop selling a product; its buyers keep their downloads", Tags: tags, Access: publish},
		func(ctx context.Context, in *productInput) (*api.Body[ProductDetail], error) {
			out, err := s.Archive(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), pid(*in), true)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unarchive-product", Path: base + "/{product}/unarchive", Summary: "Sell an archived product again", Tags: tags, Access: publish},
		func(ctx context.Context, in *productInput) (*api.Body[ProductDetail], error) {
			out, err := s.Archive(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), pid(*in), false)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-product", Path: base + "/{product}", Summary: "Delete a product that was never published", Tags: tags, Access: edit},
		func(ctx context.Context, in *productInput) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), pid(*in))
		})

	// Files, uploaded in chunks.
	api.Post(h, api.Op{ID: "start-product-upload", Path: base + "/{product}/uploads", Summary: "Announce a file: up to 500 MB, 2 GB per product",
		Tags: tags, Access: edit, Status: 201},
		func(ctx context.Context, in *uploadStartInput) (*api.Body[UploadState], error) {
			out, err := s.StartUpload(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), OwnerProduct, pid(in.productInput), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "upload-product-chunk", Path: base + "/{product}/uploads/{upload}", Summary: "Send the next chunk of a file",
		Tags: tags, Access: edit, MaxBodyBytes: ChunkBytes + 64<<10},
		func(ctx context.Context, in *chunkInput) (*api.Body[UploadState], error) {
			out, err := s.WriteChunk(ctx, api.WorkspaceFrom(ctx), OwnerProduct, pid(in.productInput), in.Upload, in.Offset, in.RawBody)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "finish-product-upload", Path: base + "/{product}/uploads/{upload}/complete",
		Summary: "Finish an upload: the file's type is checked from its content", Tags: tags, Access: edit},
		func(ctx context.Context, in *uploadInput) (*api.Body[ProductFile], error) {
			out, err := s.FinishUpload(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), OwnerProduct, pid(in.productInput), in.Upload)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "cancel-product-upload", Path: base + "/{product}/uploads/{upload}", Summary: "Cancel an upload; nothing is kept", Tags: tags, Access: edit},
		func(ctx context.Context, in *uploadInput) (*api.Empty, error) {
			return api.NoContent, s.CancelUpload(ctx, api.WorkspaceFrom(ctx), OwnerProduct, pid(in.productInput), in.Upload)
		})
	api.Delete(h, api.Op{ID: "remove-product-file", Path: base + "/{product}/files/{file}", Summary: "Remove a file from a product", Tags: tags, Access: edit},
		func(ctx context.Context, in *fileInput) (*api.Empty, error) {
			return api.NoContent, s.RemoveFile(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), OwnerProduct, pid(in.productInput), uuid.MustParse(in.File))
		})
	api.Post(h, api.Op{ID: "upload-product-cover", Path: base + "/{product}/cover", Summary: "Upload a cover image (PNG, JPEG or GIF, up to 5 MB)",
		Tags: tags, Access: edit, MaxBodyBytes: 5<<20 + 64<<10, Status: 201},
		func(ctx context.Context, in *coverInput) (*api.Body[ProductCover], error) {
			f := in.RawBody.Data().File
			if !f.IsSet {
				return nil, api.Invalid("body.file", "Choose an image to upload.")
			}
			data, err := io.ReadAll(io.LimitReader(f, 5<<20+1))
			if err != nil {
				return nil, api.Invalid("body.file", "The upload was interrupted. Try again.")
			}
			ws := api.WorkspaceFrom(ctx)
			if _, err := s.Q.ProductGet(ctx, store.ProductGetParams{WorkspaceID: ws.ID, ID: pid(in.productInput)}); err != nil {
				return nil, api.MapError(err, "Product")
			}
			a, err := s.Media.Upload(ctx, ws, actor(api.PrincipalFrom(ctx)), f.Filename, data)
			if err != nil {
				return nil, err
			}
			return api.OK(ProductCover{URL: a.URL, Width: a.Width, Height: a.Height}), nil
		})

	// Orders.
	orders := base + "/orders"
	api.Get(h, api.Op{ID: "list-product-orders", Path: orders, Summary: "Orders, newest first, with their download links", Tags: tags, Access: view},
		func(ctx context.Context, in *ordersInput) (*api.Body[ProductOrderList], error) {
			out, err := s.Orders(ctx, api.WorkspaceFrom(ctx), in.query())
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-product-orders", Path: orders + "/export", Summary: "The orders a filter shows, as CSV (they hold buyers' addresses; logged)",
		Tags: tags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *ordersInput) (*ProductsFile, error) {
			ws := api.WorkspaceFrom(ctx)
			body, _, err := s.Export(ctx, ws, api.PrincipalFrom(ctx), in.query())
			if err != nil {
				return nil, err
			}
			name := fmt.Sprintf("%s-orders-%s.csv", ws.Slug, s.now().Format("2006-01-02"))
			return &ProductsFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + strings.ReplaceAll(name, `"`, "") + `"`,
				CacheControl: "no-store", Body: body}, nil
		})
	api.Get(h, api.Op{ID: "get-product-order", Path: orders + "/{order}", Summary: "An order: its buyer and consent, payment, delivery, downloads and activity",
		Tags: tags, Access: view},
		func(ctx context.Context, in *orderInput) (*api.Body[ProductOrderDetail], error) {
			out, err := s.Order(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Order))
			return api.OK(out), err
		})
	manage := api.Workspace(authz.OrdersManage)
	api.Post(h, api.Op{ID: "send-order-link", Path: orders + "/{order}/new-link", Summary: "Send a new download link, to the address on the order only",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *newLinkInput) (*api.Body[ProductOrderDetail], error) {
			out, err := s.SendNewLink(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), uuid.MustParse(in.Order), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "revoke-order-access", Path: orders + "/{order}/revoke", Summary: "Revoke an order's downloads", Tags: tags, Access: manage},
		func(ctx context.Context, in *orderInput) (*api.Body[ProductOrderDetail], error) {
			out, err := s.Revoke(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), uuid.MustParse(in.Order), true)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "restore-order-access", Path: orders + "/{order}/restore", Summary: "Let an order's downloads work again", Tags: tags, Access: manage},
		func(ctx context.Context, in *orderInput) (*api.Body[ProductOrderDetail], error) {
			out, err := s.Revoke(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), uuid.MustParse(in.Order), false)
			return api.OK(out), err
		})
}
