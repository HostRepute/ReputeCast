package products_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/products"
)

const ws = "/workspaces/acme-newsletter"

func demo(t *testing.T) (*apptest.Env, *apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	return env, c
}

// A product saved through the API: its draft changes, its address stays
// unique, its price is checked, and the delivery template comes with it.
func TestSaveProductThroughTheAPI(t *testing.T) {
	env, c := demo(t)
	var p products.ProductDetail
	c.Do("POST", ws+"/products", map[string]any{"name": "Warm-up planner test"}).MustStatus(t, 201).JSON(t, &p)
	if p.Status != products.StatusDraft || p.Draft.Slug != "warm-up-planner-test" || p.Draft.Delivery.TemplateID == "" {
		t.Fatalf("created: %+v", p.Draft)
	}
	f := p.Draft
	f.Name = "Warm-up planner, test edition"
	f.ShortDescription = "A four-week, day-by-day sending plan."
	f.Price = products.ProductPrice{Kind: products.PriceOneTime, AmountCents: 1200}
	var saved products.ProductDetail
	c.Do("PUT", ws+"/products/"+p.ID, f).MustStatus(t, 200).JSON(t, &saved)
	if saved.Draft.Name != f.Name || saved.Draft.Price.AmountCents != 1200 || saved.ID != p.ID {
		t.Fatalf("saved: %+v", saved.Draft)
	}
	var got products.ProductDetail
	c.Do("GET", ws+"/products/"+p.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if got.Draft.ShortDescription != f.ShortDescription {
		t.Fatalf("read back: %+v", got.Draft)
	}
	// A price under the minimum is refused.
	bad := got.Draft
	bad.Price.AmountCents = 50
	c.Do("PUT", ws+"/products/"+p.ID, bad).MustStatus(t, 422)
	// Another product can't take the address.
	var other products.ProductDetail
	c.Do("POST", ws+"/products", map[string]any{"name": "Other"}).MustStatus(t, 201).JSON(t, &other)
	clash := other.Draft
	clash.Slug = got.Draft.Slug
	c.Do("PUT", ws+"/products/"+other.ID, clash).MustStatus(t, 409)
	// The template is a draft in the Essential Purchases category.
	var kind string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT c.kind FROM transactional_templates t JOIN transactional_categories c ON c.id = t.category_id
		WHERE t.key = 'product-delivery'`).Scan(&kind); err != nil || kind != "essential" {
		t.Fatalf("template category: %q %v", kind, err)
	}
	// Every change is in the audit log.
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('product.created', 'product.updated') AND target_id = $1`, p.ID).Scan(&n); err != nil || n != 2 {
		t.Fatalf("audit: %d %v", n, err)
	}
	// A draft that was never published can be deleted.
	c.Do("DELETE", ws+"/products/"+other.ID, nil).MustStatus(t, 204)
	c.Do("GET", ws+"/products/"+other.ID, nil).MustStatus(t, 404)
}

// A file goes up in chunks and its type is read from its content; a file
// whose name says something else is refused.
func TestUploadProductFile(t *testing.T) {
	_, c := demo(t)
	var p products.ProductDetail
	c.Do("POST", ws+"/products", map[string]any{"name": "Upload test"}).MustStatus(t, 201).JSON(t, &p)
	pdf := []byte("%PDF-1.4\n1 0 obj << /Type /Page >> endobj\n2 0 obj << /Type /Page >> endobj\ntrailer\n%%EOF\n")
	var up products.UploadState
	c.Do("POST", ws+"/products/"+p.ID+"/uploads", map[string]any{"name": "guide.pdf", "size": len(pdf)}).MustStatus(t, 201).JSON(t, &up)
	c.DoRaw("PUT", fmt.Sprintf("%s/products/%s/uploads/%s?offset=0", "/api/v1"+ws, p.ID, up.ID), "application/octet-stream", bytes.NewReader(pdf)).MustStatus(t, 200)
	var file products.ProductFile
	c.Do("POST", ws+"/products/"+p.ID+"/uploads/"+up.ID+"/complete", nil).MustStatus(t, 200).JSON(t, &file)
	if file.Kind != "pdf" || file.SizeBytes != int64(len(pdf)) || file.Pages != 2 {
		t.Fatalf("file: %+v", file)
	}
	// A PNG named .pdf.
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00")
	c.Do("POST", ws+"/products/"+p.ID+"/uploads", map[string]any{"name": "cover.pdf", "size": len(png)}).MustStatus(t, 201).JSON(t, &up)
	c.DoRaw("PUT", fmt.Sprintf("%s/products/%s/uploads/%s?offset=0", "/api/v1"+ws, p.ID, up.ID), "application/octet-stream", bytes.NewReader(png)).MustStatus(t, 200)
	prob := c.Do("POST", ws+"/products/"+p.ID+"/uploads/"+up.ID+"/complete", nil).MustStatus(t, 422).Problem(t)
	if prob["code"] != "file_mismatch" || !strings.Contains(prob["detail"].(string), ".pdf") {
		t.Fatalf("mismatch: %v", prob)
	}
	// Too big for one file.
	c.Do("POST", ws+"/products/"+p.ID+"/uploads", map[string]any{"name": "huge.zip", "size": products.MaxFileBytes + 1}).MustStatus(t, 422)
}
