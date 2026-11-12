package products

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/blobstore"
)

// Product files live in the instance's file storage (the files folder or
// an S3-compatible bucket, storage.backend), content-addressed under the
// workspace: one copy of bytes however many products use them. Uploads
// arrive in chunks, each its own object until the upload finishes, so a
// 500 MB file never meets the server's request timeouts, a chunk that
// fails can be sent again, and several processes can take one upload's
// chunks; nothing is kept of an upload that doesn't finish.

// Limits (brief §3.5).
const (
	MaxFileBytes    int64 = 500 << 20
	MaxProductBytes int64 = 2 << 30
	// ChunkBytes is the size of an upload's chunks (the last one may be
	// smaller).
	ChunkBytes int64 = 4 << 20
	// maxChunks is the most chunks an upload arrives in: eight times what
	// the app sends for a 500 MB file.
	maxChunks = 1000
)

// Store keeps files in the instance's file storage.
type Store struct {
	Blobs blobstore.Store
}

var (
	keyRe    = regexp.MustCompile(`^[0-9a-f-]{36}/[0-9a-f]{2}/[0-9a-f]{64}$`)
	uploadRe = regexp.MustCompile(`^[0-9a-f-]{36}$`)

	errBadKey = errors.New("products: bad storage key")
)

// contentKey is where bytes with that sum live in a workspace.
func contentKey(wsID uuid.UUID, sum []byte) string {
	hexsum := hex.EncodeToString(sum)
	return wsID.String() + "/" + hexsum[:2] + "/" + hexsum
}

// Open opens a stored file to send: in order, or from a byte range.
func (st Store) Open(ctx context.Context, key string) (*blobstore.File, error) {
	if !keyRe.MatchString(key) {
		return nil, errBadKey
	}
	return blobstore.OpenFile(ctx, st.Blobs, key)
}

// Remove deletes a stored file's bytes.
func (st Store) Remove(ctx context.Context, key string) error {
	if !keyRe.MatchString(key) {
		return errBadKey
	}
	return st.Blobs.Delete(ctx, key)
}

// errChanged is an upload whose chunks read differently the second time.
var errChanged = errors.New("products: the upload changed while it was kept")

// keep stores a finished upload's bytes (its chunks, joined) under the
// workspace and returns their key. Bytes stored already (the same file
// uploaded twice) aren't copied again; a copy that doesn't match sum as
// it streams stores nothing.
func (st Store) keep(ctx context.Context, wsID uuid.UUID, chunks []blobstore.Part, size int64, sum []byte, contentType string) (string, error) {
	key := contentKey(wsID, sum)
	info, err := st.Blobs.Stat(ctx, key)
	if err == nil && info.Size == size {
		return key, nil
	}
	if err != nil && !errors.Is(err, blobstore.ErrNotExist) {
		return "", err
	}
	f := blobstore.Join(ctx, st.Blobs, chunks)
	defer func() { _ = f.Close() }()
	return key, st.Blobs.Put(ctx, key, &sumReader{r: f, h: sha256.New(), want: sum}, size, contentType)
}

// sumReader fails at its end when what it read doesn't have the sum.
type sumReader struct {
	r    io.Reader
	h    hash.Hash
	want []byte
}

func (s *sumReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	s.h.Write(p[:n])
	if errors.Is(err, io.EOF) && !bytes.Equal(s.h.Sum(nil), s.want) {
		return n, errChanged
	}
	return n, err
}

// PutBytes stores a file's bytes under the workspace (the demo's sample
// files) and returns their key and checksum.
func (st Store) PutBytes(ctx context.Context, wsID uuid.UUID, data []byte) (string, []byte, error) {
	sum := sha256.Sum256(data)
	key := contentKey(wsID, sum[:])
	if info, err := st.Blobs.Stat(ctx, key); err == nil && info.Size == int64(len(data)) {
		return key, sum[:], nil
	}
	return key, sum[:], st.Blobs.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "")
}

// Upload is a file arriving in chunks.
type Upload struct {
	ID          string    `json:"id"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
	OwnerKind   string    `json:"ownerKind"`
	OwnerID     uuid.UUID `json:"ownerId"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	Replaces    string    `json:"replaces,omitempty"`
	UserID      uuid.UUID `json:"userId"`
	Created     time.Time `json:"created"`
}

// An upload's objects: what it is (uploads/<id>/upload.json) and its
// chunks, named by where they start (uploads/<id>/000004194304).
const (
	uploadsPrefix = "uploads/"
	uploadMeta    = "upload.json"
)

func uploadPrefix(id string) string { return uploadsPrefix + id + "/" }

func chunkKey(id string, offset int64) string {
	return fmt.Sprintf("%s%012d", uploadPrefix(id), offset)
}

// begin starts an upload.
func (st Store) begin(ctx context.Context, u Upload) (Upload, error) {
	u.ID = uuid.Must(uuid.NewV7()).String()
	b, _ := json.Marshal(u)
	return u, st.Blobs.Put(ctx, uploadPrefix(u.ID)+uploadMeta, bytes.NewReader(b), int64(len(b)), "application/json")
}

// upload reads an upload's state: what it is, how much arrived, and the
// chunks that hold it, in order.
func (st Store) upload(ctx context.Context, id string) (Upload, int64, []blobstore.Part, error) {
	if !uploadRe.MatchString(id) {
		return Upload{}, 0, nil, errors.New("products: bad upload id")
	}
	rc, err := st.Blobs.Open(ctx, uploadPrefix(id)+uploadMeta)
	if err != nil {
		return Upload{}, 0, nil, err
	}
	b, err := io.ReadAll(io.LimitReader(rc, 64<<10))
	_ = rc.Close()
	if err != nil {
		return Upload{}, 0, nil, err
	}
	var u Upload
	if err := json.Unmarshal(b, &u); err != nil {
		return Upload{}, 0, nil, err
	}
	type chunk struct {
		off  int64
		part blobstore.Part
	}
	var chunks []chunk
	prefix := uploadPrefix(id)
	if err := st.Blobs.List(ctx, prefix, func(key string, info blobstore.Info) error {
		if off, err := strconv.ParseInt(strings.TrimPrefix(key, prefix), 10, 64); err == nil {
			chunks = append(chunks, chunk{off, blobstore.Part{Key: key, Size: info.Size}})
		}
		return nil
	}); err != nil {
		return Upload{}, 0, nil, err
	}
	slices.SortFunc(chunks, func(a, b chunk) int { return cmp.Compare(a.off, b.off) })
	var have int64
	parts := make([]blobstore.Part, 0, len(chunks))
	for _, c := range chunks {
		if c.off != have {
			break
		}
		parts = append(parts, c.part)
		have += c.part.Size
	}
	return u, have, parts, nil
}

// errOffset is a chunk that doesn't continue where the upload stands.
var errOffset = errors.New("products: chunk offset")

// errChunks is an upload sent in more chunks than maxChunks.
var errChunks = errors.New("products: too many chunks")

// write stores a chunk at offset (the bytes received so far). The same
// chunk sent twice at once lands on one key, so it can't count twice.
func (st Store) write(ctx context.Context, id string, offset int64, data []byte) (int64, error) {
	u, have, parts, err := st.upload(ctx, id)
	if err != nil {
		return 0, err
	}
	if offset != have {
		return have, errOffset
	}
	if have+int64(len(data)) > u.Size {
		return have, errTooBig
	}
	if len(data) == 0 {
		return have, nil
	}
	if len(parts) >= maxChunks {
		return have, errChunks
	}
	if err := st.Blobs.Put(ctx, chunkKey(id, offset), bytes.NewReader(data), int64(len(data)), "application/octet-stream"); err != nil {
		return have, err
	}
	return have + int64(len(data)), nil
}

var errTooBig = errors.New("products: more bytes than announced")

// abort drops an upload and what arrived of it, even when the request
// that asks has ended.
func (st Store) abort(ctx context.Context, id string) {
	if !uploadRe.MatchString(id) {
		return
	}
	ctx = context.WithoutCancel(ctx)
	var keys []string
	_ = st.Blobs.List(ctx, uploadPrefix(id), func(key string, _ blobstore.Info) error {
		keys = append(keys, key)
		return nil
	})
	for _, k := range keys {
		_ = st.Blobs.Delete(ctx, k)
	}
}

// SweepUploads drops uploads that never finished and got nothing since
// before (an upload still arriving is kept, however long it takes).
func (st Store) SweepUploads(ctx context.Context, before time.Time) error {
	newest := map[string]time.Time{}
	keys := map[string][]string{}
	if err := st.Blobs.List(ctx, uploadsPrefix, func(key string, info blobstore.Info) error {
		// An upload's id; files kept by earlier releases (uploads/<id>.part)
		// are each their own.
		id, _, _ := strings.Cut(strings.TrimPrefix(key, uploadsPrefix), "/")
		if info.ModTime.After(newest[id]) {
			newest[id] = info.ModTime
		}
		keys[id] = append(keys[id], key)
		return nil
	}); err != nil {
		return err
	}
	var errs []error
	for id, at := range newest {
		if at.After(before) {
			continue
		}
		for _, k := range keys[id] {
			if err := st.Blobs.Delete(ctx, k); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// finish reads a complete upload once: its sum and what its content is.
// It returns the chunks to keep.
func (st Store) finish(ctx context.Context, id string) (Upload, Detected, []byte, []blobstore.Part, error) {
	u, have, parts, err := st.upload(ctx, id)
	if err != nil {
		return u, Detected{}, nil, nil, err
	}
	if have != u.Size {
		return u, Detected{}, nil, nil, fmt.Errorf("products: upload incomplete: %d of %d bytes", have, u.Size)
	}
	f := blobstore.Join(ctx, st.Blobs, parts)
	defer func() { _ = f.Close() }()
	h := sha256.New()
	var sn sniffer
	if _, err := io.Copy(io.MultiWriter(h, &sn), f); err != nil {
		return u, Detected{}, nil, nil, err
	}
	return u, sn.detected(f, f.Size()), h.Sum(nil), parts, nil
}

// Detected is what a file's content is.
type Detected struct {
	// Kind is the content's ("pdf", "zip", "xlsx"…); "" when it couldn't
	// be told.
	Kind        string
	ContentType string
	// Pages is a PDF's page count (0 when it couldn't be read).
	Pages int
}

// kinds are the contents a file is recognised as, with the type it's
// served with and the extensions that name it.
var kinds = map[string]struct {
	ctype string
	exts  []string
}{
	"pdf":  {"application/pdf", []string{"pdf"}},
	"zip":  {"application/zip", []string{"zip"}},
	"xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", []string{"xlsx"}},
	"docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", []string{"docx"}},
	"pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", []string{"pptx"}},
	"epub": {"application/epub+zip", []string{"epub"}},
	"png":  {"image/png", []string{"png"}},
	"jpeg": {"image/jpeg", []string{"jpg", "jpeg"}},
	"gif":  {"image/gif", []string{"gif"}},
	"webp": {"image/webp", []string{"webp"}},
	"mp3":  {"audio/mpeg", []string{"mp3"}},
	"mp4":  {"video/mp4", []string{"mp4", "m4v"}},
	"m4a":  {"audio/mp4", []string{"m4a"}},
	"wav":  {"audio/wav", []string{"wav"}},
	"text": {"text/plain; charset=utf-8", []string{"txt", "md", "csv", "json", "mjml", "html", "css"}},
}

// KindProgram is a program (Windows, Linux or macOS): never sold.
const KindProgram = "program"

const (
	headBytes = 4096
	// Large PDFs: the first 64 MB tell enough.
	pdfScanBytes = 64 << 20
)

// sniffer tells what a file contains from its bytes, never its name. It
// is written the file's bytes from its start (as an upload's chunks are
// read for its sum): it keeps the first 4 KB and, for a PDF, counts the
// page objects of the first 64 MB as they pass, ignoring the rest; a ZIP
// archive's directory is then read at its end (detected).
type sniffer struct {
	head  []byte
	n     int64
	pdf   bool
	pages pageCounter
}

func (s *sniffer) Write(p []byte) (int, error) {
	at := s.n
	s.n += int64(len(p))
	if len(s.head) < headBytes {
		s.head = append(s.head, p[:min(len(p), headBytes-len(s.head))]...)
		s.pdf = bytes.HasPrefix(s.head, []byte("%PDF-"))
	}
	if s.pdf && at < pdfScanBytes {
		s.pages.write(p[:min(int64(len(p)), pdfScanBytes-at)])
	}
	return len(p), nil
}

// detected is what the bytes written are; r reads the whole file (a ZIP
// archive's directory is at its end).
func (s *sniffer) detected(r io.ReaderAt, size int64) Detected {
	head := s.head
	kind := ""
	switch {
	case bytes.HasPrefix(head, []byte("%PDF-")):
		kind = "pdf"
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		kind = zipKind(r, size)
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		kind = "png"
	case bytes.HasPrefix(head, []byte("\xff\xd8\xff")):
		kind = "jpeg"
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		kind = "gif"
	case len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		kind = "webp"
	case len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WAVE")):
		kind = "wav"
	case bytes.HasPrefix(head, []byte("ID3")), len(head) >= 2 && head[0] == 0xff && head[1]&0xe0 == 0xe0:
		kind = "mp3"
	case len(head) >= 12 && bytes.Equal(head[4:8], []byte("ftyp")):
		kind = "mp4"
		if b := string(head[8:12]); b == "M4A " || b == "M4B " {
			kind = "m4a"
		}
	case bytes.HasPrefix(head, []byte("MZ")), bytes.HasPrefix(head, []byte("\x7fELF")),
		bytes.HasPrefix(head, []byte("\xfe\xed\xfa\xce")), bytes.HasPrefix(head, []byte("\xfe\xed\xfa\xcf")),
		bytes.HasPrefix(head, []byte("\xce\xfa\xed\xfe")), bytes.HasPrefix(head, []byte("\xcf\xfa\xed\xfe")),
		bytes.HasPrefix(head, []byte("\xca\xfe\xba\xbe")):
		// A Windows, Linux or macOS program.
		kind = KindProgram
	case len(head) > 0 && utf8.Valid(head) && !bytes.ContainsRune(head, 0):
		kind = "text"
	}
	d := Detected{Kind: kind, ContentType: "application/octet-stream"}
	if k, ok := kinds[kind]; ok {
		d.ContentType = k.ctype
	}
	if kind == "pdf" {
		d.Pages = s.pages.count()
	}
	return d
}

// zipKind tells an Office document or an EPUB from a plain ZIP archive,
// from its directory.
func zipKind(r io.ReaderAt, size int64) string {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return "zip"
	}
	has := map[string]bool{}
	for _, f := range zr.File {
		name := f.Name
		if i := strings.IndexByte(name, '/'); i > 0 {
			has[name[:i+1]] = true
		}
		has[name] = true
		if name == "mimetype" && f.UncompressedSize64 < 100 {
			if rc, err := f.Open(); err == nil {
				b, _ := io.ReadAll(io.LimitReader(rc, 100))
				_ = rc.Close()
				if strings.TrimSpace(string(b)) == "application/epub+zip" {
					return "epub"
				}
			}
		}
	}
	if has["[Content_Types].xml"] {
		switch {
		case has["xl/"]:
			return "xlsx"
		case has["word/"]:
			return "docx"
		case has["ppt/"]:
			return "pptx"
		}
	}
	return "zip"
}

var pageRe = regexp.MustCompile(`/Type\s*/Page[^s]`)

// pageTail is how far back a page object may begin and still be cut by
// the end of what has arrived: it is matched once more bytes come.
const pageTail = 1024

// pageCounter counts a PDF's page objects (its pages, for a PDF that
// isn't compressed in object streams) as its bytes stream past, keeping
// only a short tail between writes.
type pageCounter struct {
	buf []byte
	n   int
}

func (c *pageCounter) write(p []byte) {
	c.buf = append(c.buf, p...)
	keep := len(c.buf) - pageTail
	if keep <= 0 {
		return
	}
	from := keep
	for _, m := range pageRe.FindAllIndex(c.buf, -1) {
		if m[0] >= keep {
			break
		}
		c.n++
		from = max(from, m[1])
	}
	c.buf = c.buf[:copy(c.buf, c.buf[from:])]
}

// count is the page objects found, the tail's too.
func (c *pageCounter) count() int {
	return c.n + len(pageRe.FindAllIndex(c.buf, -1))
}

// extOf is a file name's extension, lower case without the dot.
func extOf(name string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
}

// kindOfExt is the content kind a file name claims ("" for an extension
// that isn't one of the recognised kinds).
func kindOfExt(name string) string {
	ext := extOf(name)
	for k, v := range kinds {
		for _, e := range v.exts {
			if e == ext {
				return k
			}
		}
	}
	return ""
}

// Mismatch says whether a file's name and its content disagree: a name
// that claims one recognised kind over content of another (a ".pdf" that
// is a ZIP archive). Content that couldn't be told, or an extension that
// names no recognised kind, can't be checked.
func Mismatch(name string, d Detected) (claimed string, bad bool) {
	claimed = kindOfExt(name)
	if claimed == "" || claimed == d.Kind {
		return claimed, false
	}
	if d.Kind == "" {
		// Content that can't be told doesn't pass for a recognised kind.
		return claimed, true
	}
	// A plain-text CSV or Markdown file is text; that's what it claims.
	return claimed, true
}

// SafeName keeps a readable file name buyers download.
func SafeName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, name)
	name = strings.TrimSpace(strings.Trim(name, "."))
	if len(name) > 160 {
		ext := filepath.Ext(name)
		if len(ext) > 12 {
			ext = ""
		}
		name = name[:160-len(ext)] + ext
	}
	if name == "" {
		name = "download"
	}
	return name
}
