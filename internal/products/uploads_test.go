package products_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/products"
)

// uploader sends a product's files through the API, chunk by chunk.
type uploader struct {
	t       *testing.T
	env     *apptest.Env
	c       *apptest.Client
	product string
}

func newUploader(t *testing.T, name string) *uploader {
	t.Helper()
	env, c := demo(t)
	var p products.ProductDetail
	c.Do("POST", ws+"/products", map[string]any{"name": name}).MustStatus(t, 201).JSON(t, &p)
	return &uploader{t: t, env: env, c: c, product: p.ID}
}

func (u *uploader) start(name string, size int64, replaces string) *apptest.Response {
	u.t.Helper()
	body := map[string]any{"name": name, "size": size}
	if replaces != "" {
		body["replaces"] = replaces
	}
	return u.c.Do("POST", ws+"/products/"+u.product+"/uploads", body)
}

func (u *uploader) begin(name string, size int64) products.UploadState {
	u.t.Helper()
	var st products.UploadState
	u.start(name, size, "").MustStatus(u.t, 201).JSON(u.t, &st)
	return st
}

func (u *uploader) put(id string, offset int, chunk []byte) *apptest.Response {
	u.t.Helper()
	return u.c.DoRaw("PUT", fmt.Sprintf("%s/products/%s/uploads/%s?offset=%d", ws, u.product, id, offset), "application/octet-stream", bytes.NewReader(chunk))
}

func (u *uploader) complete(id string) *apptest.Response {
	u.t.Helper()
	return u.c.Do("POST", ws+"/products/"+u.product+"/uploads/"+id+"/complete", nil)
}

// upload sends a whole file in one chunk and finishes it.
func (u *uploader) upload(name string, data []byte) *apptest.Response {
	u.t.Helper()
	st := u.begin(name, int64(len(data)))
	u.put(st.ID, 0, data).MustStatus(u.t, 200)
	return u.complete(st.ID)
}

// files are the product's files now.
func (u *uploader) files() []products.ProductFile {
	u.t.Helper()
	var p products.ProductDetail
	u.c.Do("GET", ws+"/products/"+u.product, nil).MustStatus(u.t, 200).JSON(u.t, &p)
	return p.Files
}

// record puts files on the product as already uploaded (only their sizes
// matter: the product's 2 GB counts them).
func (u *uploader) record(sizes ...int64) []uuid.UUID {
	u.t.Helper()
	var wsID uuid.UUID
	if err := u.env.App.Pool.QueryRow(context.Background(), `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		u.t.Fatal(err)
	}
	var ids []uuid.UUID
	for i, size := range sizes {
		id := uuid.Must(uuid.NewV7())
		sum := sha256.Sum256([]byte(id.String()))
		key := wsID.String() + "/" + hex.EncodeToString(sum[:])[:2] + "/" + hex.EncodeToString(sum[:])
		if _, err := u.env.App.Pool.Exec(context.Background(), `INSERT INTO download_files (id, workspace_id, owner_kind, owner_id, name, size_bytes, kind, content_type, sha256, storage_key, position)
			VALUES ($1, $2, 'product', $3, $4, $5, 'zip', 'application/zip', $6, $7, $8)`, id, wsID, u.product, fmt.Sprintf("part-%d.zip", i+1), size, sum[:], key, i); err != nil {
			u.t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// pdf is a small PDF of n bytes or so, different for each seed.
func pdf(seed string) []byte {
	return []byte("%PDF-1.4\n% " + seed + "\n1 0 obj << /Type /Page >> endobj\ntrailer\n%%EOF\n")
}

// A file arrives in chunks, each continuing where the upload stands: a
// chunk sent again or from elsewhere answers 409 with the bytes received, so
// the browser can continue; finishing early answers 409; more bytes than
// announced end the upload; a chunk over the chunk size is refused whole;
// another product can't touch the upload. Cancelling keeps nothing, on disk
// or on the product.
func TestUploadChunks(t *testing.T) {
	u := newUploader(t, "Chunk test")
	data := bytes.Repeat(pdf("chunks"), 40)
	half := len(data) / 2
	st := u.begin("guide.pdf", int64(len(data)))
	if st.Received != 0 || st.Size != int64(len(data)) || st.ChunkBytes != products.ChunkBytes {
		t.Fatalf("started: %+v", st)
	}
	var got products.UploadState
	u.put(st.ID, 0, data[:half]).MustStatus(t, 200).JSON(t, &got)
	if got.Received != int64(half) {
		t.Fatalf("after the first chunk: %+v", got)
	}
	// The same chunk again, and one from further on: where it stands.
	for _, off := range []int{0, half + 10} {
		p := u.put(st.ID, off, data[off:]).MustStatus(t, 409).Problem(t)
		meta, _ := p["meta"].(map[string]any)
		if p["code"] != "upload_offset" || fmt.Sprint(meta["received"]) != fmt.Sprint(half) {
			t.Errorf("chunk at %d: %v", off, p)
		}
	}
	// Finishing early.
	p := u.complete(st.ID).MustStatus(t, 409).Problem(t)
	if meta, _ := p["meta"].(map[string]any); p["code"] != "upload_incomplete" || fmt.Sprint(meta["received"]) != fmt.Sprint(half) {
		t.Errorf("finishing early: %v", p)
	}
	// Another product can't write to it, finish it or cancel it.
	other := &uploader{t: t, env: u.env, c: u.c}
	var op products.ProductDetail
	u.c.Do("POST", ws+"/products", map[string]any{"name": "Other product"}).MustStatus(t, 201).JSON(t, &op)
	other.product = op.ID
	other.put(st.ID, half, data[half:]).MustStatus(t, 404)
	other.complete(st.ID).MustStatus(t, 404)
	u.c.Do("DELETE", ws+"/products/"+op.ID+"/uploads/"+st.ID, nil).MustStatus(t, 404)
	// The rest, and it's a file.
	u.put(st.ID, half, data[half:]).MustStatus(t, 200).JSON(t, &got)
	if got.Received != int64(len(data)) {
		t.Fatalf("after the last chunk: %+v", got)
	}
	var f products.ProductFile
	u.complete(st.ID).MustStatus(t, 200).JSON(t, &f)
	if f.Kind != "pdf" || f.SizeBytes != int64(len(data)) || f.Name != "guide.pdf" {
		t.Fatalf("file: %+v", f)
	}
	// Finished: the upload is gone.
	u.complete(st.ID).MustStatus(t, 404)
	u.put(st.ID, len(data), []byte("x")).MustStatus(t, 404)

	// More bytes than announced: the upload ends.
	small := u.begin("small.pdf", 10)
	u.put(small.ID, 0, data[:20]).MustStatus(t, 422)
	u.put(small.ID, 0, data[:10]).MustStatus(t, 404)

	// A chunk over the chunk size is refused whole.
	big := u.begin("big.zip", 6<<20)
	u.put(big.ID, 0, make([]byte, products.ChunkBytes+64<<10+1)).MustStatus(t, 413)
	u.put(big.ID, 0, make([]byte, 1024)).MustStatus(t, 200)

	// Cancelled: nothing kept.
	dir := filepath.Join(u.env.App.Products().Files.Dir, "uploads")
	drop := u.begin("draft.pdf", int64(len(data)))
	u.put(drop.ID, 0, data[:half]).MustStatus(t, 200)
	if _, err := os.Stat(filepath.Join(dir, drop.ID+".part")); err != nil {
		t.Fatalf("no part file while uploading: %v", err)
	}
	u.c.Do("DELETE", ws+"/products/"+u.product+"/uploads/"+drop.ID, nil).MustStatus(t, 204)
	for _, ext := range []string{".part", ".json"} {
		if _, err := os.Stat(filepath.Join(dir, drop.ID+ext)); !os.IsNotExist(err) {
			t.Errorf("a cancelled upload's %s is still there: %v", ext, err)
		}
	}
	u.put(drop.ID, half, data[half:]).MustStatus(t, 404)
	u.complete(drop.ID).MustStatus(t, 404)
	u.c.Do("DELETE", ws+"/products/"+u.product+"/uploads/"+drop.ID, nil).MustStatus(t, 404)

	// Only the finished file is the product's.
	if fs := u.files(); len(fs) != 1 || fs[0].Name != "guide.pdf" {
		t.Errorf("product files: %+v", fs)
	}
}

// A product holds 2 GB of files at most: an upload that would take it over
// is refused when it starts, one that fits is let in; a file being replaced
// doesn't count, and neither does a removed one.
func TestUploadProductTotal(t *testing.T) {
	u := newUploader(t, "Big bundle")
	const mb = int64(1) << 20
	ids := u.record(500*mb, 500*mb, 500*mb, 500*mb, products.MaxProductBytes-2000*mb-100)
	// 100 bytes left.
	p := u.start("one-more.pdf", 101, "").MustStatus(t, 422).Problem(t)
	if !strings.Contains(fmt.Sprint(p), "2 GB") {
		t.Errorf("over the product's total: %v", p)
	}
	u.start("one-more.pdf", 100, "").MustStatus(t, 201)
	// Replacing a file: what's replaced doesn't count.
	last := products.MaxProductBytes - 2000*mb - 100
	u.start("part-5-v2.zip", last+100, ids[4].String()).MustStatus(t, 201)
	p = u.start("part-5-v2.zip", last+101, ids[4].String()).MustStatus(t, 422).Problem(t)
	if !strings.Contains(fmt.Sprint(p), "2 GB") {
		t.Errorf("a replacement over the product's total: %v", p)
	}
	u.start("part-1-v2.zip", 500*mb, ids[0].String()).MustStatus(t, 201)
	u.start("part-6.zip", 500*mb, "").MustStatus(t, 422)
	// Removed files leave room; one file over 500 MB is refused whatever
	// room is left.
	u.c.Do("DELETE", ws+"/products/"+u.product+"/files/"+ids[1].String(), nil).MustStatus(t, 204)
	u.c.Do("DELETE", ws+"/products/"+u.product+"/files/"+ids[2].String(), nil).MustStatus(t, 204)
	u.start("part-6.zip", 500*mb, "").MustStatus(t, 201)
	u.start("huge.zip", products.MaxFileBytes+1, "").MustStatus(t, 422)
	if fs := u.files(); len(fs) != 3 {
		t.Errorf("product files: %d", len(fs))
	}
}

// Uploads started side by side each fit when they start; together they must
// not take the product over 2 GB when they finish.
func TestUploadsSideBySideKeepTheProductTotal(t *testing.T) {
	u := newUploader(t, "Side by side")
	const mb = int64(1) << 20
	u.record(500*mb, 500*mb, 500*mb, 500*mb, products.MaxProductBytes-2000*mb-100)
	a, b := pdf("first upload, sixty bytes or so"), pdf("second upload, sixty bytes too")
	if len(a)+len(b) <= 100 || len(a) > 100 || len(b) > 100 {
		t.Fatalf("sizes %d and %d don't make the case", len(a), len(b))
	}
	ua, ub := u.begin("a.pdf", int64(len(a))), u.begin("b.pdf", int64(len(b)))
	u.put(ua.ID, 0, a).MustStatus(t, 200)
	u.put(ub.ID, 0, b).MustStatus(t, 200)
	u.complete(ua.ID).MustStatus(t, 200)
	u.complete(ub.ID).MustStatus(t, 422)
}

// The file type is checked from the content: an executable isn't a file a
// product can sell, whatever its name says.
func TestUploadRefusesExecutables(t *testing.T) {
	u := newUploader(t, "Executables")
	windows := append([]byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff\x00\x00\xb8\x00\x00\x00"), []byte("This program cannot be run in DOS mode.\r\r\n$\x00\x00")...)
	linux := append([]byte("\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00"), make([]byte, 64)...)
	for _, tc := range []struct {
		name string
		data []byte
	}{{"setup.exe", windows}, {"deliverability-field-guide.pdf", windows}, {"install", linux}, {"planner.xlsx", linux}} {
		if r := u.upload(tc.name, tc.data); r.Status != 422 {
			t.Errorf("%s (%d bytes) uploaded: %d %s", tc.name, len(tc.data), r.Status, r.Body)
		}
	}
	if fs := u.files(); len(fs) != 0 {
		t.Errorf("the product kept %d executables", len(fs))
	}
}
