package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// Part is one object of a File made of several, in order.
type Part struct {
	Key  string
	Size int64
}

// File reads a stored object, or several one after the other (an upload's
// chunks), as one file: in order (io.Reader), from anywhere (io.Seeker,
// for http.ServeContent's ranges) and at any offset (io.ReaderAt, for an
// archive's directory). Reading in order holds one ranged request open
// from where it began; Seek costs nothing until the next Read.
type File struct {
	ctx   context.Context
	store Store
	parts []Part
	ends  []int64 // ends[i]: where parts[i] ends in the file
	size  int64
	mod   time.Time

	pos    int64
	rc     io.ReadCloser // reads parts[rcPart] from rcPos
	rcPart int
	rcPos  int64
	closed bool

	mu    sync.Mutex
	cache [cacheBlocks]block
	clock int
}

// ReadAt reads small ranges through a few cached 256 KB blocks, so a ZIP
// directory read 4 KB at a time costs a request per block, not per read.
const (
	blockBytes  = 256 << 10
	cacheBlocks = 4
)

type block struct {
	off  int64
	data []byte
	used int
}

// OpenFile opens an object to read as a File (its size and time from
// Stat, so a missing key fails here, before anything is sent).
func OpenFile(ctx context.Context, s Store, key string) (*File, error) {
	info, err := s.Stat(ctx, key)
	if err != nil {
		return nil, err
	}
	f := Join(ctx, s, []Part{{Key: key, Size: info.Size}})
	f.mod = info.ModTime
	return f, nil
}

// Join reads parts one after the other as one File of their total size.
func Join(ctx context.Context, s Store, parts []Part) *File {
	f := &File{ctx: ctx, store: s, parts: parts, ends: make([]int64, len(parts))}
	for i, p := range parts {
		f.size += p.Size
		f.ends[i] = f.size
	}
	for i := range f.cache {
		f.cache[i].off = -1
	}
	return f
}

// Size is the file's length in bytes.
func (f *File) Size() int64 { return f.size }

// ModTime is when the object was written (OpenFile only).
func (f *File) ModTime() time.Time { return f.mod }

// part is the index of the part holding byte off (off < size).
func (f *File) part(off int64) int {
	return sort.Search(len(f.ends), func(i int) bool { return f.ends[i] > off })
}

func (f *File) start(i int) int64 {
	if i == 0 {
		return 0
	}
	return f.ends[i-1]
}

func (f *File) Read(p []byte) (int, error) {
	if f.closed {
		return 0, errors.New("blobstore: read of a closed file")
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if f.pos >= f.size {
			f.drop()
			return 0, io.EOF
		}
		if f.rc == nil || f.rcPos != f.pos {
			f.drop()
			i := f.part(f.pos)
			from := f.pos - f.start(i)
			rc, err := f.store.OpenRange(f.ctx, f.parts[i].Key, from, f.parts[i].Size-from)
			if err != nil {
				return 0, err
			}
			f.rc, f.rcPart, f.rcPos = rc, i, f.pos
		}
		end := f.ends[f.rcPart]
		if int64(len(p)) > end-f.pos {
			p = p[:end-f.pos]
		}
		n, err := f.rc.Read(p)
		f.pos += int64(n)
		f.rcPos = f.pos
		switch {
		case f.pos == end:
			// The part is read: the next Read opens the next one.
			f.drop()
			return n, nil
		case errors.Is(err, io.EOF):
			f.drop()
			if n > 0 {
				return n, nil
			}
			return 0, fmt.Errorf("blobstore: %s ends before its %d bytes: %w", f.parts[f.rcPart].Key, f.parts[f.rcPart].Size, io.ErrUnexpectedEOF)
		case err != nil:
			f.drop()
			return n, err
		case n > 0:
			return n, nil
		}
	}
}

// drop closes the open request.
func (f *File) drop() {
	if f.rc != nil {
		_ = f.rc.Close()
		f.rc = nil
	}
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	var to int64
	switch whence {
	case io.SeekStart:
		to = offset
	case io.SeekCurrent:
		to = f.pos + offset
	case io.SeekEnd:
		to = f.size + offset
	default:
		return f.pos, errors.New("blobstore: bad whence")
	}
	if to < 0 {
		return f.pos, errors.New("blobstore: seek before the start")
	}
	f.pos = to
	return to, nil
}

// ReadAt reads len(p) bytes at off. It doesn't move Read's position, and
// several ReadAt calls may run at once.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("blobstore: negative offset")
	}
	if off >= f.size {
		return 0, io.EOF
	}
	want := len(p)
	if rest := f.size - off; int64(want) > rest {
		p = p[:rest]
	}
	var n int
	var err error
	if len(p) >= blockBytes {
		n, err = f.fill(p, off)
	} else {
		n, err = f.cached(p, off)
	}
	if err == nil && n < want {
		err = io.EOF
	}
	return n, err
}

// cached copies from the blocks holding [off, off+len(p)).
func (f *File) cached(p []byte, off int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for n < len(p) {
		at := off + int64(n)
		b, err := f.block(at - at%blockBytes)
		if err != nil {
			return n, err
		}
		n += copy(p[n:], b.data[at-b.off:])
	}
	return n, nil
}

// block is the cached block at off, read now when it isn't cached (the
// least recently used one makes room).
func (f *File) block(off int64) (*block, error) {
	f.clock++
	oldest := 0
	for i := range f.cache {
		if f.cache[i].off == off {
			f.cache[i].used = f.clock
			return &f.cache[i], nil
		}
		if f.cache[i].used < f.cache[oldest].used {
			oldest = i
		}
	}
	b := &f.cache[oldest]
	size := min(int64(blockBytes), f.size-off)
	if int64(cap(b.data)) < size {
		b.data = make([]byte, blockBytes)
	}
	b.off = -1
	n, err := f.fill(b.data[:size], off)
	if err != nil {
		return nil, err
	}
	b.data, b.off, b.used = b.data[:n], off, f.clock
	return b, nil
}

// fill reads [off, off+len(p)) (within the file) from the parts it spans.
func (f *File) fill(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		at := off + int64(n)
		i := f.part(at)
		from := at - f.start(i)
		want := min(int64(len(p)-n), f.parts[i].Size-from)
		rc, err := f.store.OpenRange(f.ctx, f.parts[i].Key, from, want)
		if err != nil {
			return n, err
		}
		got, err := io.ReadFull(rc, p[n:n+int(want)])
		_ = rc.Close()
		n += got
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return n, fmt.Errorf("blobstore: read %s: %w", f.parts[i].Key, err)
		}
	}
	return n, nil
}

// Close ends the open request; the File reads nothing more.
func (f *File) Close() error {
	f.drop()
	f.closed = true
	return nil
}
