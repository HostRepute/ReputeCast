package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Local keeps objects as files under a folder (files.dir), a key's
// slashes its subfolders: the layout ReputeCast has always used, so a
// folder written before object storage existed reads as it is.
type Local struct {
	Dir string
}

// path is where a key's file lives.
func (l Local) path(key string) (string, error) {
	if err := ValidKey(key); err != nil {
		return "", err
	}
	return filepath.Join(l.Dir, filepath.FromSlash(key)), nil
}

// Put writes to a temporary file next to the key's and renames it into
// place once it is complete and on disk.
func (l Local) Put(ctx context.Context, key string, r io.Reader, size int64, _ string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	var f *os.File
	for attempt := 0; ; attempt++ {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		f, err = os.CreateTemp(dir, tempPrefix+"*")
		// A Delete may have removed the folder, empty, in between.
		if errors.Is(err, fs.ErrNotExist) && attempt < 3 {
			continue
		}
		if err != nil {
			return err
		}
		break
	}
	tmp := f.Name()
	_, err = io.Copy(f, exactly(ctxReader{ctx, r}, size))
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, p)
	}
	if err != nil {
		_ = os.Remove(tmp)
	}
	return err
}

// open opens a key's file, refusing folders.
func (l Local) open(key string) (*os.File, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p) //nolint:gosec // G304: the key is checked by ValidKey, under the files folder
	if err != nil {
		return nil, err
	}
	if fi, err := f.Stat(); err != nil || fi.IsDir() {
		_ = f.Close()
		if err != nil {
			return nil, err
		}
		return nil, &fs.PathError{Op: "open", Path: p, Err: ErrNotExist}
	}
	return f, nil
}

func (l Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return l.open(key)
}

func (l Local) OpenRange(_ context.Context, key string, off, n int64) (io.ReadCloser, error) {
	if off < 0 {
		return nil, fmt.Errorf("blobstore: negative offset %d", off)
	}
	f, err := l.open(key)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, err
	}
	if n < 0 {
		return f, nil
	}
	return readCloser{io.LimitReader(f, n), f}, nil
}

func (l Local) Stat(_ context.Context, key string) (Info, error) {
	p, err := l.path(key)
	if err != nil {
		return Info{}, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return Info{}, err
	}
	if fi.IsDir() {
		return Info{}, &fs.PathError{Op: "stat", Path: p, Err: ErrNotExist}
	}
	return Info{Size: fi.Size(), ModTime: fi.ModTime()}, nil
}

// Delete removes a key's file and the folders it leaves empty (an
// upload's, a workspace's), never the files folder itself.
func (l Local) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if fi, err := os.Lstat(p); err != nil || fi.IsDir() {
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			// No object there (a folder holds others).
			return nil
		}
		return err
	}
	if err := os.Remove(p); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	root := filepath.Clean(l.Dir)
	for d := filepath.Dir(p); d != root && strings.HasPrefix(d, root+string(filepath.Separator)); d = filepath.Dir(d) {
		if !removedEmpty(d) {
			break
		}
	}
	return nil
}

// removedEmpty removes a folder if it is empty; false when it still holds
// something (or can't be removed), where clearing up stops.
func removedEmpty(dir string) bool {
	return os.Remove(dir) == nil
}

// List walks the folders the prefix reaches into; files being written
// aren't listed.
func (l Local) List(ctx context.Context, prefix string, fn func(key string, info Info) error) error {
	root := filepath.Clean(l.Dir)
	start := root
	if i := strings.LastIndexByte(prefix, '/'); i > 0 {
		if err := ValidKey(prefix[:i]); err != nil {
			return err
		}
		start = filepath.Join(root, filepath.FromSlash(prefix[:i]))
	}
	return filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == start && errors.Is(err, fs.ErrNotExist) {
				// Nothing was ever stored there.
				return fs.SkipAll
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		if d.IsDir() {
			// Only folders on the prefix's way, or inside it.
			if !strings.HasPrefix(key+"/", prefix) && !strings.HasPrefix(prefix, key+"/") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), tempPrefix) || !strings.HasPrefix(key, prefix) {
			return nil
		}
		info, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			// Deleted since its folder was read.
			return nil
		}
		if err != nil {
			return err
		}
		return fn(key, Info{Size: info.Size(), ModTime: info.ModTime()})
	})
}

// readCloser closes c once r is read.
type readCloser struct {
	io.Reader
	io.Closer
}

// ctxReader stops a copy once its context ends.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
