package products_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/blobstore/blobstoretest"
	"github.com/hostrepute/reputecast/internal/products"
)

// guide is a PDF of n bytes, sent in several chunks, with page objects at
// its start, across the first chunk's end and in its third chunk (those
// that fit): the page count is read as the bytes stream past, never the
// whole file at once.
func guide(n int) []byte {
	data := blobstoretest.Data(42, n)
	copy(data, "%PDF-1.7\n")
	const page = "<< /Type /Page >>"
	for _, at := range []int{200, int(products.ChunkBytes) - 6, 2*int(products.ChunkBytes) + 999} {
		if at+len(page) < n {
			copy(data[at:], page)
		}
	}
	return data
}

// A file uploaded in chunks through the API is kept once, in the file
// storage (the files folder, or the test bucket when
// REPUTECAST_S3_TEST_ENDPOINT is set), and its buyer downloads it byte for
// byte: a range gets its slice (206), a resume with the file's If-Range is
// the rest and isn't counted again, an If-Range of another time gets the
// whole file.
func TestStoredFileRoundTrip(t *testing.T) {
	s := newShop(t)
	ctx := context.Background()
	u := &uploader{t: t, env: s.env, c: s.owner, product: s.fieldGuide.String()}
	data := guide(2*int(products.ChunkBytes) + 123_457)
	st := u.begin("field-guide-2.pdf", int64(len(data)))
	for off := 0; off < len(data); off += int(st.ChunkBytes) {
		u.put(st.ID, off, data[off:min(off+int(st.ChunkBytes), len(data))]).MustStatus(t, 200)
	}
	var f products.ProductFile
	u.complete(st.ID).MustStatus(t, 200).JSON(t, &f)
	if f.Kind != "pdf" || f.Pages != 3 || f.SizeBytes != int64(len(data)) {
		t.Fatalf("file: %+v", f)
	}
	var key string
	var sum []byte
	s.scan(`SELECT storage_key, sha256 FROM download_files WHERE id = $1`, []any{f.ID}, &key, &sum)
	want := sha256.Sum256(data)
	if !bytes.Equal(sum, want[:]) {
		t.Fatalf("recorded sum %x", sum)
	}
	if info, err := s.env.App.Blobs.Stat(ctx, key); err != nil || info.Size != int64(len(data)) {
		t.Fatalf("stored: %+v %v", info, err)
	}
	// The same bytes again are the same object; nothing of either upload
	// stays.
	again := u.begin("copy.pdf", int64(len(data)))
	u.put(again.ID, 0, data[:products.ChunkBytes]).MustStatus(t, 200)
	u.put(again.ID, int(products.ChunkBytes), data[products.ChunkBytes:2*products.ChunkBytes]).MustStatus(t, 200)
	u.put(again.ID, 2*int(products.ChunkBytes), data[2*products.ChunkBytes:]).MustStatus(t, 200)
	var dup products.ProductFile
	u.complete(again.ID).MustStatus(t, 200).JSON(t, &dup)
	var dupKey string
	s.scan(`SELECT storage_key FROM download_files WHERE id = $1`, []any{dup.ID}, &dupKey)
	if dupKey != key {
		t.Errorf("the same bytes stored twice: %s and %s", key, dupKey)
	}
	if err := s.env.App.Blobs.List(ctx, "uploads/", func(k string, _ blobstore.Info) error {
		t.Errorf("left of the uploads: %s", k)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	o, _ := s.buy(s.fieldGuide, buyer{Email: "round.trip@example.com"})
	link := s.link(o.ID)
	url := s.site() + "/d/" + s.token(o.ID, link.ID) + "/" + f.ID
	type answer struct {
		StatusCode int
		Header     http.Header
	}
	get := func(hdr map[string]string) (answer, []byte) {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		return answer{resp.StatusCode, resp.Header}, b
	}
	used := func() int32 { return s.link(o.ID).DownloadsUsed }

	resp, body := get(nil)
	if resp.StatusCode != http.StatusOK || !bytes.Equal(body, data) || resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("download: %d, %d bytes of %d, %q", resp.StatusCode, len(body), len(data), resp.Header.Get("Content-Type"))
	}
	modified := resp.Header.Get("Last-Modified")
	if used() != 1 {
		t.Fatalf("downloads used %d", used())
	}
	from, to := int(products.ChunkBytes)-50, int(products.ChunkBytes)+49
	resp, body = get(map[string]string{"Range": fmt.Sprintf("bytes=%d-%d", from, to)})
	if resp.StatusCode != http.StatusPartialContent || !bytes.Equal(body, data[from:to+1]) ||
		resp.Header.Get("Content-Range") != fmt.Sprintf("bytes %d-%d/%d", from, to, len(data)) {
		t.Fatalf("range: %d %q, %d bytes", resp.StatusCode, resp.Header.Get("Content-Range"), len(body))
	}
	resp, body = get(map[string]string{"Range": "bytes=5000000-", "If-Range": modified})
	if resp.StatusCode != http.StatusPartialContent || !bytes.Equal(body, data[5000000:]) {
		t.Fatalf("resume: %d, %d bytes", resp.StatusCode, len(body))
	}
	if used() != 1 {
		t.Errorf("a resume counted: %d used", used())
	}
	resp, body = get(map[string]string{"Range": "bytes=5000000-", "If-Range": "Mon, 02 Jan 2006 15:04:05 GMT"})
	if resp.StatusCode != http.StatusOK || !bytes.Equal(body, data) {
		t.Fatalf("another time's If-Range: %d, %d bytes", resp.StatusCode, len(body))
	}
	if used() != 2 {
		t.Errorf("a whole download after an If-Range of another time: %d used", used())
	}
}

// An upload that sat for a day without a chunk is swept; one still
// arriving isn't.
func TestSweepUnfinishedUploads(t *testing.T) {
	s := newShop(t)
	ctx := context.Background()
	u := &uploader{t: t, env: s.env, c: s.owner, product: s.fieldGuide.String()}
	data := guide(int(products.ChunkBytes) + 10)
	st := u.begin("stale.pdf", int64(len(data)))
	u.put(st.ID, 0, data[:products.ChunkBytes]).MustStatus(t, 200)
	count := func() int {
		n := 0
		_ = s.env.App.Blobs.List(ctx, "uploads/"+st.ID+"/", func(string, blobstore.Info) error {
			n++
			return nil
		})
		return n
	}
	if err := s.svc.Sweep(ctx); err != nil || count() != 2 {
		t.Fatalf("a fresh upload swept: %d left, %v", count(), err)
	}
	s.clock.add(25 * time.Hour)
	if err := s.svc.Sweep(ctx); err != nil || count() != 0 {
		t.Fatalf("a day-old upload: %d left, %v", count(), err)
	}
	u.put(st.ID, int(products.ChunkBytes), data[products.ChunkBytes:]).MustStatus(t, 404)
}
