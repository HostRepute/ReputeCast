// Package blobstoretest is the contract every blobstore backend passes,
// and the test bucket integration tests use.
//
// The S3 runs need an S3-compatible server (MinIO in CI):
//
//	REPUTECAST_S3_TEST_ENDPOINT    its URL, e.g. http://localhost:19000 (unset: the S3 runs skip)
//	REPUTECAST_S3_TEST_BUCKET      the bucket, made when missing (default reputecast-test)
//	REPUTECAST_S3_TEST_REGION      default us-east-1
//	REPUTECAST_S3_TEST_ACCESS_KEY_ID, REPUTECAST_S3_TEST_SECRET_ACCESS_KEY
//	                               default minioadmin / minioadmin (MinIO's own)
//
// Every test keeps its objects under a prefix of its own and deletes them
// when it ends.
package blobstoretest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/blobstore"
	"github.com/hostrepute/reputecast/internal/config"
)

// The environment variables the S3 runs read.
const (
	EnvEndpoint  = "REPUTECAST_S3_TEST_ENDPOINT"
	EnvBucket    = "REPUTECAST_S3_TEST_BUCKET"
	EnvRegion    = "REPUTECAST_S3_TEST_REGION"
	EnvAccessKey = "REPUTECAST_S3_TEST_ACCESS_KEY_ID"
	EnvSecretKey = "REPUTECAST_S3_TEST_SECRET_ACCESS_KEY" //nolint:gosec // G101: an environment variable's name, not a credential
)

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// made holds the buckets made (or found) in this test binary.
var made sync.Map

// S3Config is the test bucket's settings with a prefix of the test's own,
// emptied when the test ends; false when REPUTECAST_S3_TEST_ENDPOINT isn't
// set.
func S3Config(t testing.TB) (config.S3Config, bool) {
	t.Helper()
	endpoint := os.Getenv(EnvEndpoint)
	if endpoint == "" {
		return config.S3Config{}, false
	}
	c := config.S3Config{
		Bucket: env(EnvBucket, "reputecast-test"), Region: env(EnvRegion, "us-east-1"), Endpoint: endpoint, PathStyle: true,
		AccessKeyID: env(EnvAccessKey, "minioadmin"), SecretAccessKey: env(EnvSecretKey, "minioadmin"),
		Prefix: "test/" + uuid.NewString() + "/",
	}
	ctx := context.Background()
	client, err := blobstore.NewS3Client(ctx, c)
	if err != nil {
		t.Fatalf("test bucket: %v", err)
	}
	if _, ok := made.Load(c.Bucket); !ok {
		_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(c.Bucket)})
		var owned *types.BucketAlreadyOwnedByYou
		var exists *types.BucketAlreadyExists
		if err != nil && !errors.As(err, &owned) && !errors.As(err, &exists) {
			t.Fatalf("make the test bucket %s at %s: %v", c.Bucket, endpoint, err)
		}
		made.Store(c.Bucket, true)
	}
	t.Cleanup(func() {
		st, err := blobstore.NewS3(context.Background(), c)
		if err != nil {
			return
		}
		var keys []string
		_ = st.List(context.Background(), "", func(key string, _ blobstore.Info) error {
			keys = append(keys, key)
			return nil
		})
		for _, k := range keys {
			_ = st.Delete(context.Background(), k)
		}
	})
	return c, true
}

// S3 is a store on the test bucket under a prefix of the test's own; the
// test skips when REPUTECAST_S3_TEST_ENDPOINT isn't set.
func S3(t testing.TB) *blobstore.S3 {
	t.Helper()
	c, ok := S3Config(t)
	if !ok {
		t.Skip(EnvEndpoint + " isn't set: no S3-compatible server to test against")
	}
	st, err := blobstore.NewS3(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// Data is n bytes that are the same for a seed, and hard to compress.
func Data(seed uint64, n int) []byte {
	b := make([]byte, n)
	r := rand.NewChaCha8([32]byte{byte(seed), byte(seed >> 8), byte(seed >> 16), byte(seed >> 24)})
	_, _ = r.Read(b)
	return b
}

// onlyReader hides every method but Read (a stream of unknown length).
type onlyReader struct{ r io.Reader }

func (o onlyReader) Read(p []byte) (int, error) { return o.r.Read(p) }

// Run checks a backend against the contract; open returns a new, empty
// store each time it is called.
func Run(t *testing.T, open func(t *testing.T) blobstore.Store) {
	ctx := context.Background()
	read := func(t *testing.T, rc io.ReadCloser, err error) []byte {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rc.Close() }()
		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	put := func(t *testing.T, s blobstore.Store, key string, b []byte) {
		t.Helper()
		if err := s.Put(ctx, key, bytes.NewReader(b), int64(len(b)), "application/octet-stream"); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}
	keys := func(t *testing.T, s blobstore.Store, prefix string) []string {
		t.Helper()
		out := []string{}
		if err := s.List(ctx, prefix, func(key string, _ blobstore.Info) error {
			out = append(out, key)
			return nil
		}); err != nil {
			t.Fatalf("list %q: %v", prefix, err)
		}
		slices.Sort(out)
		return out
	}

	t.Run("PutOpenStat", func(t *testing.T) {
		s := open(t)
		data := Data(1, 70_000)
		before := time.Now().Add(-time.Minute)
		put(t, s, "ws/ab/object", data)
		rc, err := s.Open(ctx, "ws/ab/object")
		if got := read(t, rc, err); !bytes.Equal(got, data) {
			t.Fatalf("read %d bytes, want the %d put", len(got), len(data))
		}
		info, err := s.Stat(ctx, "ws/ab/object")
		if err != nil || info.Size != int64(len(data)) || info.ModTime.Before(before) || info.ModTime.After(time.Now().Add(time.Minute)) {
			t.Fatalf("stat: %+v, %v", info, err)
		}
	})

	t.Run("Range", func(t *testing.T) {
		s := open(t)
		data := Data(2, 10_000)
		put(t, s, "r", data)
		for _, c := range []struct {
			off, n int64
			want   []byte
		}{
			{0, 10, data[:10]},
			{1234, 1000, data[1234:2234]},
			{9_990, 100, data[9_990:]},
			{500, -1, data[500:]},
			{0, -1, data},
			{10_000, 10, nil},
			{20_000, -1, nil},
			{100, 0, nil},
		} {
			rc, err := s.OpenRange(ctx, "r", c.off, c.n)
			if got := read(t, rc, err); !bytes.Equal(got, c.want) {
				t.Errorf("range %d+%d: %d bytes, want %d", c.off, c.n, len(got), len(c.want))
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		s := open(t)
		if _, err := s.Open(ctx, "missing/key"); !errors.Is(err, fs.ErrNotExist) || !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("open: %v", err)
		}
		if _, err := s.OpenRange(ctx, "missing/key", 10, 10); !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("open range: %v", err)
		}
		if _, err := s.OpenRange(ctx, "missing/key", 0, 0); !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("open empty range: %v", err)
		}
		if _, err := s.Stat(ctx, "missing/key"); !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("stat: %v", err)
		}
		if _, err := blobstore.OpenFile(ctx, s, "missing/key"); !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("open file: %v", err)
		}
		if err := s.Delete(ctx, "missing/key"); err != nil {
			t.Errorf("deleting a missing key: %v", err)
		}
		// A key's folder is no object.
		put(t, s, "folder/inside", []byte("x"))
		if _, err := s.Stat(ctx, "folder"); !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("stat of a folder: %v", err)
		}
		if err := s.Delete(ctx, "folder"); err != nil {
			t.Errorf("deleting a folder: %v", err)
		}
		if got := keys(t, s, ""); !slices.Equal(got, []string{"folder/inside"}) {
			t.Errorf("after deleting a folder: %v", got)
		}
	})

	t.Run("OverwriteDelete", func(t *testing.T) {
		s := open(t)
		put(t, s, "k", []byte("first version"))
		put(t, s, "k", []byte("second"))
		rc, err := s.Open(ctx, "k")
		if got := read(t, rc, err); string(got) != "second" {
			t.Fatalf("after overwriting: %q", got)
		}
		if info, err := s.Stat(ctx, "k"); err != nil || info.Size != 6 {
			t.Fatalf("stat after overwriting: %+v %v", info, err)
		}
		if err := s.Delete(ctx, "k"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Stat(ctx, "k"); !errors.Is(err, blobstore.ErrNotExist) {
			t.Fatalf("stat after delete: %v", err)
		}
		if got := keys(t, s, ""); len(got) != 0 {
			t.Fatalf("listed after delete: %v", got)
		}
	})

	t.Run("List", func(t *testing.T) {
		s := open(t)
		for _, k := range []string{"x/1", "x/2", "x/sub/3", "xy/4", "y/5"} {
			put(t, s, k, []byte(k))
		}
		for prefix, want := range map[string][]string{
			"x/":     {"x/1", "x/2", "x/sub/3"},
			"x":      {"x/1", "x/2", "x/sub/3", "xy/4"},
			"x/sub/": {"x/sub/3"},
			"":       {"x/1", "x/2", "x/sub/3", "xy/4", "y/5"},
			"none/":  {},
			"x/s":    {"x/sub/3"},
		} {
			if got := keys(t, s, prefix); !slices.Equal(got, want) {
				t.Errorf("list %q: %v, want %v", prefix, got, want)
			}
		}
		var size int64
		_ = s.List(ctx, "x/sub/", func(_ string, info blobstore.Info) error {
			size = info.Size
			return nil
		})
		if size != int64(len("x/sub/3")) {
			t.Errorf("listed size %d", size)
		}
		stop := errors.New("stop")
		n := 0
		if err := s.List(ctx, "", func(string, blobstore.Info) error {
			n++
			return stop
		}); !errors.Is(err, stop) || n != 1 {
			t.Errorf("a stopped walk: %v after %d", err, n)
		}
	})

	t.Run("PrefixIsolation", func(t *testing.T) {
		a, b := open(t), open(t)
		put(t, a, "same/key", []byte("a"))
		if _, err := b.Stat(ctx, "same/key"); !errors.Is(err, blobstore.ErrNotExist) {
			t.Errorf("another store sees the key: %v", err)
		}
		if got := keys(t, b, ""); len(got) != 0 {
			t.Errorf("another store lists %v", got)
		}
	})

	t.Run("Size", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, "short", bytes.NewReader([]byte("12345")), 10, ""); !errors.Is(err, blobstore.ErrSize) {
			t.Errorf("fewer bytes than the size: %v", err)
		}
		if err := s.Put(ctx, "long", bytes.NewReader([]byte("12345")), 3, ""); !errors.Is(err, blobstore.ErrSize) {
			t.Errorf("more bytes than the size: %v", err)
		}
		if err := s.Put(ctx, "nonempty", bytes.NewReader([]byte("1")), 0, ""); !errors.Is(err, blobstore.ErrSize) {
			t.Errorf("bytes for an empty object: %v", err)
		}
		if got := keys(t, s, ""); len(got) != 0 {
			t.Errorf("a failed put left %v", got)
		}
		if err := s.Put(ctx, "empty", bytes.NewReader(nil), 0, ""); err != nil {
			t.Fatal(err)
		}
		rc, err := s.Open(ctx, "empty")
		if got := read(t, rc, err); len(got) != 0 {
			t.Errorf("empty object: %q", got)
		}
		rc, err = s.OpenRange(ctx, "empty", 0, 10)
		if got := read(t, rc, err); len(got) != 0 {
			t.Errorf("range of an empty object: %q", got)
		}
		if err := s.Put(ctx, "unknown", onlyReader{bytes.NewReader([]byte("some bytes"))}, -1, "text/plain"); err != nil {
			t.Fatal(err)
		}
		if info, err := s.Stat(ctx, "unknown"); err != nil || info.Size != 10 {
			t.Errorf("an unknown size: %+v %v", info, err)
		}
	})

	t.Run("Keys", func(t *testing.T) {
		s := open(t)
		for _, k := range []string{"", "/abs", "../up", "a/../b", "a//b", "a/./b", `a\b`, "a/", "a\x00b", "a/.tmp-x"} {
			if err := s.Put(ctx, k, bytes.NewReader([]byte("x")), 1, ""); err == nil {
				t.Errorf("put %q worked", k)
			}
			if _, err := s.Open(ctx, k); err == nil {
				t.Errorf("open %q worked", k)
			}
		}
	})

	t.Run("Large", func(t *testing.T) {
		// Past the 16 MB the S3 backend sends in one request, of a size it
		// isn't told: it goes up in parts.
		s := open(t)
		data := Data(3, 21<<20+12345)
		if err := s.Put(ctx, "large/object", onlyReader{bytes.NewReader(data)}, -1, "application/zip"); err != nil {
			t.Fatal(err)
		}
		info, err := s.Stat(ctx, "large/object")
		if err != nil || info.Size != int64(len(data)) {
			t.Fatalf("stat: %+v %v", info, err)
		}
		rc, err := s.Open(ctx, "large/object")
		if got := read(t, rc, err); sha256.Sum256(got) != sha256.Sum256(data) {
			t.Fatalf("read %d bytes that differ from the %d put", len(got), len(data))
		}
		// A range across the first part's end, and the tail.
		off := int64(8<<20 - 100)
		rc, err = s.OpenRange(ctx, "large/object", off, 300)
		if got := read(t, rc, err); !bytes.Equal(got, data[off:off+300]) {
			t.Errorf("range across a part: %d bytes", len(got))
		}
		known := Data(4, 17<<20)
		if err := s.Put(ctx, "large/known", bytes.NewReader(known), int64(len(known)), ""); err != nil {
			t.Fatal(err)
		}
		if info, err := s.Stat(ctx, "large/known"); err != nil || info.Size != int64(len(known)) {
			t.Fatalf("stat of a known size: %+v %v", info, err)
		}
	})

	t.Run("File", func(t *testing.T) {
		s := open(t)
		a, b, c := Data(5, 1000), Data(6, 300_000), Data(7, 77)
		put(t, s, "f/a", a)
		put(t, s, "f/b", b)
		put(t, s, "f/c", c)
		whole := slices.Concat(a, b, c)
		f := blobstore.Join(ctx, s, []blobstore.Part{{Key: "f/a", Size: 1000}, {Key: "f/empty", Size: 0}, {Key: "f/b", Size: 300_000}, {Key: "f/c", Size: 77}})
		defer func() { _ = f.Close() }()
		if got, err := io.ReadAll(f); err != nil || !bytes.Equal(got, whole) {
			t.Fatalf("joined: %d bytes, %v", len(got), err)
		}
		// Seek as http.ServeContent does, then read a range.
		if end, err := f.Seek(0, io.SeekEnd); err != nil || end != int64(len(whole)) {
			t.Fatalf("seek to the end: %d %v", end, err)
		}
		if _, err := f.Seek(999, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		got := make([]byte, 2000)
		if _, err := io.ReadFull(f, got); err != nil || !bytes.Equal(got, whole[999:2999]) {
			t.Fatalf("read from 999: %v", err)
		}
		for _, r := range []struct{ off, n int }{{0, 10}, {995, 10}, {1000, 4096}, {262_000, 5000}, {301_000, 77}, {100, 300_000}} {
			p := make([]byte, r.n)
			n, err := f.ReadAt(p, int64(r.off))
			if err != nil || n != r.n || !bytes.Equal(p, whole[r.off:r.off+r.n]) {
				t.Errorf("read at %d+%d: %d %v", r.off, r.n, n, err)
			}
		}
		p := make([]byte, 100)
		if n, err := f.ReadAt(p, int64(len(whole)-50)); n != 50 || !errors.Is(err, io.EOF) || !bytes.Equal(p[:50], whole[len(whole)-50:]) {
			t.Errorf("read at past the end: %d %v", n, err)
		}
		one, err := blobstore.OpenFile(ctx, s, "f/b")
		if err != nil || one.Size() != 300_000 || one.ModTime().IsZero() {
			t.Fatalf("open file: %v", err)
		}
		defer func() { _ = one.Close() }()
		if _, err := one.Seek(-77, io.SeekEnd); err != nil {
			t.Fatal(err)
		}
		if got, err := io.ReadAll(one); err != nil || !bytes.Equal(got, b[len(b)-77:]) {
			t.Errorf("the last 77 bytes: %d %v", len(got), err)
		}
		// A part shorter than it says fails the read.
		short := blobstore.Join(ctx, s, []blobstore.Part{{Key: "f/c", Size: 100}})
		if _, err := io.ReadAll(short); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("a short part: %v", err)
		}
	})

	t.Run("Concurrent", func(t *testing.T) {
		s := open(t)
		var wg sync.WaitGroup
		errs := make(chan error, 8)
		for i := range 8 {
			wg.Go(func() {
				data := Data(uint64(100+i), 5000)
				key := fmt.Sprintf("c/%d", i)
				if err := s.Put(ctx, key, bytes.NewReader(data), int64(len(data)), ""); err != nil {
					errs <- err
					return
				}
				if err := s.Delete(ctx, key); err != nil {
					errs <- err
				}
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Error(err)
		}
	})
}
