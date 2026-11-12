// Package blobstore keeps the files ReputeCast stores outside the database
// (digital products' files, lead magnets, chunked uploads, audience
// exports) in the local files folder or in an S3-compatible bucket
// (storage.backend). Both backends name objects with the same keys, so
// moving from one to the other is a copy (`reputecast storage migrate`).
package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/config"
)

// Store keeps objects under slash-separated keys ("<workspace>/ab/<sha256>",
// "uploads/<id>/000000000000").
type Store interface {
	// Put stores r's bytes under key, replacing what was there. size is
	// r's length, or -1 when it isn't known (a reader that ends early or
	// runs past size fails the Put, which then stores nothing). A Put is
	// atomic: readers see the old object or the new one, never a part.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Open reads an object from its start.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// OpenRange reads at most n bytes from off (n < 0: to the end). A
	// range past the end reads nothing.
	OpenRange(ctx context.Context, key string, off, n int64) (io.ReadCloser, error)
	// Stat reports an object's size and when it was written.
	Stat(ctx context.Context, key string) (Info, error)
	// Delete removes an object; a key that doesn't exist isn't an error.
	Delete(ctx context.Context, key string) error
	// List calls fn for every object whose key starts with prefix, in no
	// particular order; fn's error stops the walk and is returned.
	List(ctx context.Context, prefix string, fn func(key string, info Info) error) error
}

// Info describes a stored object.
type Info struct {
	Size    int64
	ModTime time.Time
}

// ErrNotExist is what every backend returns for a key that holds no
// object (errors.Is(err, fs.ErrNotExist) holds too).
var ErrNotExist = fs.ErrNotExist

// ErrSize is a Put whose reader holds more or fewer bytes than its size.
var ErrSize = errors.New("blobstore: the content's length isn't the size given")

// MaxKeyBytes is the longest key (S3's limit, less room for a prefix).
const MaxKeyBytes = 900

// tempPrefix starts the names of the local backend's files being written.
const tempPrefix = ".tmp-"

// ValidKey checks a key is one every backend stores the same way: slash
// separated, relative, without empty, "." or ".." segments, backslashes or
// control characters.
func ValidKey(key string) error {
	if key == "" || len(key) > MaxKeyBytes || !utf8.ValidString(key) {
		return fmt.Errorf("blobstore: bad key %q", key)
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return fmt.Errorf("blobstore: bad key %q", key)
		}
	}
	for seg := range strings.SplitSeq(key, "/") {
		if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, tempPrefix) {
			return fmt.Errorf("blobstore: bad key %q", key)
		}
	}
	return nil
}

// Open builds the store the configuration names: the files folder
// (storage.backend local) or the bucket (s3).
func Open(ctx context.Context, cfg *config.Config) (Store, error) {
	switch cfg.Storage.Backend {
	case config.StorageS3:
		return NewS3(ctx, cfg.Storage.S3)
	case config.StorageLocal, "":
		return Local{Dir: cfg.FilesDir()}, nil
	}
	return nil, fmt.Errorf("blobstore: storage.backend must be local or s3, got %q", cfg.Storage.Backend)
}

// sizedReader fails a read that finds more or fewer bytes than size.
type sizedReader struct {
	r    io.Reader
	left int64
}

func (s *sizedReader) Read(p []byte) (int, error) {
	if s.left == 0 {
		// One more byte would be past size.
		var b [1]byte
		n, err := s.r.Read(b[:])
		if n > 0 {
			return 0, ErrSize
		}
		if err == nil {
			return 0, nil
		}
		return 0, err
	}
	if int64(len(p)) > s.left {
		p = p[:s.left]
	}
	n, err := s.r.Read(p)
	s.left -= int64(n)
	if errors.Is(err, io.EOF) && s.left > 0 {
		return n, ErrSize
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

// exactly wraps r so that reading it fails unless it holds size bytes
// (size < 0: any length).
func exactly(r io.Reader, size int64) io.Reader {
	if size < 0 {
		return r
	}
	return &sizedReader{r: r, left: size}
}
