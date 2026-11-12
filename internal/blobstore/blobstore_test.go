package blobstore_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/blobstore/blobstoretest"
	"github.com/hostrepute/reputecast/internal/config"
)

func TestLocalContract(t *testing.T) {
	blobstoretest.Run(t, func(t *testing.T) blobstore.Store { return blobstore.Local{Dir: t.TempDir()} })
}

// Against MinIO (or any S3-compatible server) when
// REPUTECAST_S3_TEST_ENDPOINT is set; see package blobstoretest.
func TestS3Contract(t *testing.T) {
	blobstoretest.S3(t) // skips without a server
	blobstoretest.Run(t, func(t *testing.T) blobstore.Store { return blobstoretest.S3(t) })
}

// The bucket answers Check; one that doesn't exist says so.
func TestS3Check(t *testing.T) {
	s := blobstoretest.S3(t)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	c, _ := blobstoretest.S3Config(t)
	c.Bucket = "reputecast-no-such-bucket"
	missing, err := blobstore.NewS3(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if err := missing.Check(context.Background()); err == nil {
		t.Fatal("a missing bucket checks")
	}
	// A GET says which is missing (a HEAD can't): a bucket is a setting to
	// fix, not a missing file.
	if _, err := missing.Open(context.Background(), "k"); err == nil || errors.Is(err, blobstore.ErrNotExist) {
		t.Errorf("a key of a missing bucket: %v", err)
	}
}

// Files written before object storage existed read under their keys, as
// they are; files being written aren't listed; a deleted file's empty
// folders go, the files folder stays.
func TestLocalLayout(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := blobstore.Local{Dir: dir}
	old := filepath.Join(dir, "0190a5b2-0000-7000-8000-000000000001", "ab", "abcd")
	if err := os.MkdirAll(filepath.Dir(old), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("bytes from before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0190a5b2-0000-7000-8000-000000000001", "ab", ".tmp-123"), []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Open(ctx, "0190a5b2-0000-7000-8000-000000000001/ab/abcd")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(b) != "bytes from before" {
		t.Fatalf("read %q", b)
	}
	var keys []string
	if err := s.List(ctx, "", func(k string, _ blobstore.Info) error {
		keys = append(keys, k)
		return nil
	}); err != nil || len(keys) != 1 || keys[0] != "0190a5b2-0000-7000-8000-000000000001/ab/abcd" {
		t.Fatalf("listed %v %v", keys, err)
	}
	if err := s.Put(ctx, "uploads/u1/000000000000", bytes.NewReader([]byte("chunk")), 5, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "uploads/u1/000000000000"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "uploads")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("an empty upload folder stays: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the files folder went: %v", err)
	}
	// Listing a folder that doesn't exist yet lists nothing.
	if err := (blobstore.Local{Dir: filepath.Join(dir, "never")}).List(ctx, "x/", func(string, blobstore.Info) error {
		t.Error("listed something")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Open builds the backend storage.backend names.
func TestOpen(t *testing.T) {
	cfg := config.Defaults()
	cfg.Files.Dir = t.TempDir()
	st, err := blobstore.Open(context.Background(), &cfg)
	if l, ok := st.(blobstore.Local); err != nil || !ok || l.Dir != cfg.Files.Dir {
		t.Fatalf("local: %#v %v", st, err)
	}
	cfg.Storage.Backend = config.StorageS3
	cfg.Storage.S3 = config.S3Config{Bucket: "files", Region: "eu-west-1", Prefix: "/reputecast/", AccessKeyID: "AKIA", SecretAccessKey: "secret"}
	st, err = blobstore.Open(context.Background(), &cfg)
	s3, ok := st.(*blobstore.S3)
	if err != nil || !ok || s3.Bucket() != "files" || s3.Prefix() != "reputecast/" {
		t.Fatalf("s3: %#v %v", st, err)
	}
	cfg.Storage.Backend = "ftp"
	if _, err := blobstore.Open(context.Background(), &cfg); err == nil {
		t.Fatal("an unknown backend opened")
	}
}
