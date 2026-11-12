package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/blobstore/blobstoretest"
	"github.com/hostrepute/reputecast/internal/config"
)

// localFiles is a files folder as releases before object storage left it:
// a product file, a lead magnet, an export, an upload in progress, and a
// file half written (not an object).
func localFiles(t *testing.T) (blobstore.Local, map[string][]byte) {
	t.Helper()
	dir := t.TempDir()
	files := map[string][]byte{
		"0190a5b2-0000-7000-8000-000000000001/ab/ab01":                                          []byte("%PDF-1.4 the field guide"),
		"0190a5b2-0000-7000-8000-000000000001/cd/cd02":                                          blobstoretest.Data(9, 300_000),
		"exports/0190a5b2-0000-7000-8000-000000000001/0190a5b2-0000-7000-8000-0000000000e1.csv": []byte("email\nreader@example.com\n"),
		"uploads/0190a5b2-0000-7000-8000-0000000000f1.json":                                     []byte(`{"id":"…"}`),
	}
	for k, b := range files {
		p := filepath.Join(dir, filepath.FromSlash(k))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "uploads", ".tmp-half"), []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	return blobstore.Local{Dir: dir}, files
}

// `storage migrate --to s3`: a dry run counts every local file and copies
// nothing; the copy then takes them all, checked; run again, it skips
// them; the local files stay.
func TestStorageMigrate(t *testing.T) {
	ctx := context.Background()
	run := func(t *testing.T, dst blobstore.Store) {
		src, files := localFiles(t)
		var total int64
		for _, b := range files {
			total += int64(len(b))
		}
		var out, errw bytes.Buffer
		st, err := migrateFiles(ctx, &out, &errw, src, dst, "the bucket test", true)
		if err != nil || st.Objects != int64(len(files)) || st.Bytes != total || st.Copied != int64(len(files)) || st.Skipped != 0 || st.Failed != 0 {
			t.Fatalf("dry run: %+v %v\n%s%s", st, err, out.String(), errw.String())
		}
		if !strings.Contains(out.String(), "Dry run: 4 files") {
			t.Errorf("dry run says:\n%s", out.String())
		}
		n := 0
		_ = dst.List(ctx, "", func(string, blobstore.Info) error {
			n++
			return nil
		})
		if n != 0 {
			t.Fatalf("a dry run copied %d files", n)
		}
		out.Reset()
		st, err = migrateFiles(ctx, &out, &errw, src, dst, "the bucket test", false)
		if err != nil || st.Copied != int64(len(files)) || st.CopiedBytes != total || st.Failed != 0 {
			t.Fatalf("copy: %+v %v\n%s%s", st, err, out.String(), errw.String())
		}
		for k, b := range files {
			rc, err := dst.Open(ctx, k)
			if err != nil {
				t.Fatalf("%s: %v", k, err)
			}
			got, _ := io.ReadAll(rc)
			_ = rc.Close()
			if !bytes.Equal(got, b) {
				t.Errorf("%s: %d bytes, want %d", k, len(got), len(b))
			}
			if _, err := src.Stat(ctx, k); err != nil {
				t.Errorf("the local %s went: %v", k, err)
			}
		}
		out.Reset()
		st, err = migrateFiles(ctx, &out, &errw, src, dst, "the bucket test", false)
		if err != nil || st.Copied != 0 || st.Skipped != int64(len(files)) {
			t.Fatalf("again: %+v %v", st, err)
		}
	}
	t.Run("Local", func(t *testing.T) { run(t, blobstore.Local{Dir: t.TempDir()}) })
	t.Run("S3", func(t *testing.T) { run(t, blobstoretest.S3(t)) })
}

// Without a bucket set, the command says what to set.
func TestStorageMigrateNeedsABucket(t *testing.T) {
	cfg := config.Defaults()
	_, err := migrationBucket(context.Background(), &cfg)
	if err == nil || !strings.Contains(err.Error(), "storage.s3.bucket") || !strings.Contains(err.Error(), "REPUTECAST_STORAGE__S3__BUCKET") {
		t.Fatalf("no bucket: %v", err)
	}
	s3, ok := blobstoretest.S3Config(t)
	if !ok {
		return
	}
	cfg.Storage.S3 = s3
	if _, err := migrationBucket(context.Background(), &cfg); err != nil {
		t.Fatalf("the test bucket: %v", err)
	}
}
