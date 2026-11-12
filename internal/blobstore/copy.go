package blobstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"
)

// CopyOptions says how Copy copies.
type CopyOptions struct {
	// DryRun counts what would be copied and copies nothing.
	DryRun bool
	// Verify reads every copy back and compares its SHA-256 with the
	// source's; a copy that differs is deleted and counts as failed.
	Verify bool
	// Workers copy side by side (0: 8).
	Workers int
	// Failed hears of each object that couldn't be copied (it may be
	// called from several goroutines at once).
	Failed func(key string, err error)
}

// CopyStats counts a Copy: every object of the source and its bytes;
// those copied (or, in a dry run, to copy), those the destination holds
// already at the same size, and those that failed.
type CopyStats struct {
	Objects, Bytes      int64
	Copied, CopiedBytes int64
	Skipped, Failed     int64
}

// Copy copies every object of src into dst under the same key, skipping
// those dst holds already at the same size. It never deletes from src.
// The returned error is the walk's (a source that can't be listed, a
// context that ended); objects that fail are counted and reported to
// opts.Failed.
func Copy(ctx context.Context, dst, src Store, opts CopyOptions) (CopyStats, error) {
	workers := opts.Workers
	if workers <= 0 {
		workers = 8
	}
	type object struct {
		key  string
		size int64
	}
	var (
		st   CopyStats
		mu   sync.Mutex
		wg   sync.WaitGroup
		todo = make(chan object)
	)
	count := func(f func()) {
		mu.Lock()
		f()
		mu.Unlock()
	}
	for range workers {
		wg.Go(func() {
			for o := range todo {
				skipped, err := copyOne(ctx, dst, src, o.key, o.size, opts)
				count(func() {
					switch {
					case err != nil:
						st.Failed++
					case skipped:
						st.Skipped++
					default:
						st.Copied++
						st.CopiedBytes += o.size
					}
				})
				if err != nil && opts.Failed != nil {
					opts.Failed(o.key, err)
				}
			}
		})
	}
	err := src.List(ctx, "", func(key string, info Info) error {
		count(func() {
			st.Objects++
			st.Bytes += info.Size
		})
		select {
		case todo <- object{key, info.Size}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	close(todo)
	wg.Wait()
	return st, err
}

// copyOne copies an object unless dst has it at the same size.
func copyOne(ctx context.Context, dst, src Store, key string, size int64, opts CopyOptions) (skipped bool, err error) {
	info, err := dst.Stat(ctx, key)
	switch {
	case err == nil && info.Size == size:
		return true, nil
	case err != nil && !errors.Is(err, ErrNotExist):
		return false, err
	case opts.DryRun:
		return false, nil
	}
	rc, err := src.Open(ctx, key)
	if err != nil {
		return false, err
	}
	defer func() { _ = rc.Close() }()
	h := sha256.New()
	if err := dst.Put(ctx, key, io.TeeReader(rc, h), size, ""); err != nil {
		return false, err
	}
	if !opts.Verify {
		return false, nil
	}
	back, err := dst.Open(ctx, key)
	if err != nil {
		return false, err
	}
	hb := sha256.New()
	_, err = io.Copy(hb, back)
	_ = back.Close()
	if err == nil && !bytes.Equal(hb.Sum(nil), h.Sum(nil)) {
		err = fmt.Errorf("blobstore: %s reads back different from what was copied", key)
	}
	if err != nil {
		// A rerun copies it again rather than skipping it for its size.
		_ = dst.Delete(ctx, key)
		return false, err
	}
	return false, nil
}
