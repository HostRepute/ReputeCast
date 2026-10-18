package products

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Product files live on the instance's data volume (ROADMAP default: no
// object storage), content-addressed under the workspace: one copy of
// bytes however many products use them. Uploads arrive in chunks, so a
// 500 MB file never meets the server's request timeouts, and a chunk that
// fails can be sent again; nothing is kept of an upload that doesn't
// finish.

// Limits (brief §3.5).
const (
	MaxFileBytes    int64 = 500 << 20
	MaxProductBytes int64 = 2 << 30
	// ChunkBytes is the size of an upload's chunks (the last one may be
	// smaller).
	ChunkBytes int64 = 4 << 20
)

// Store keeps files on disk under a folder.
type Store struct {
	Dir string
}

// path is where a stored key lives.
func (st Store) path(key string) (string, error) {
	if !keyRe.MatchString(key) {
		return "", errors.New("products: bad storage key")
	}
	return filepath.Join(st.Dir, filepath.FromSlash(key)), nil
}

var (
	keyRe    = regexp.MustCompile(`^[0-9a-f-]{36}/[0-9a-f]{2}/[0-9a-f]{64}$`)
	uploadRe = regexp.MustCompile(`^[0-9a-f-]{36}$`)
)

// Open opens a stored file.
func (st Store) Open(key string) (*os.File, error) {
	p, err := st.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p) //nolint:gosec // the key is checked against keyRe
}

// Remove deletes a stored file's bytes.
func (st Store) Remove(key string) error {
	p, err := st.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Put stores the bytes of a finished upload under the workspace and
// returns their key; the upload's part file is moved, not copied.
func (st Store) put(wsID uuid.UUID, part string, sum []byte) (string, error) {
	hexsum := hex.EncodeToString(sum)
	key := wsID.String() + "/" + hexsum[:2] + "/" + hexsum
	p, err := st.path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil {
		// The same bytes are stored already.
		_ = os.Remove(part)
		return key, nil
	}
	if err := os.Rename(part, p); err != nil {
		return "", err
	}
	return key, nil
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

func (st Store) uploadDir() string { return filepath.Join(st.Dir, "uploads") }

func (st Store) uploadPaths(id string) (meta, part string, err error) {
	if !uploadRe.MatchString(id) {
		return "", "", errors.New("products: bad upload id")
	}
	d := st.uploadDir()
	return filepath.Join(d, id+".json"), filepath.Join(d, id+".part"), nil
}

// begin starts an upload.
func (st Store) begin(u Upload) (Upload, error) {
	u.ID = uuid.Must(uuid.NewV7()).String()
	if err := os.MkdirAll(st.uploadDir(), 0o750); err != nil {
		return u, err
	}
	meta, part, _ := st.uploadPaths(u.ID)
	b, _ := json.Marshal(u)
	if err := os.WriteFile(meta, b, 0o600); err != nil {
		return u, err
	}
	f, err := os.OpenFile(part, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // the path is built from a fresh UUID
	if err != nil {
		return u, err
	}
	return u, f.Close()
}

// upload reads an upload's state: what it is and how much arrived.
func (st Store) upload(id string) (Upload, int64, error) {
	meta, part, err := st.uploadPaths(id)
	if err != nil {
		return Upload{}, 0, err
	}
	b, err := os.ReadFile(meta) //nolint:gosec // the path is built from a checked UUID
	if err != nil {
		return Upload{}, 0, err
	}
	var u Upload
	if err := json.Unmarshal(b, &u); err != nil {
		return Upload{}, 0, err
	}
	fi, err := os.Stat(part)
	if err != nil {
		return Upload{}, 0, err
	}
	return u, fi.Size(), nil
}

// errOffset is a chunk that doesn't continue where the upload stands.
var errOffset = errors.New("products: chunk offset")

// write appends a chunk at offset (the bytes received so far).
func (st Store) write(id string, offset int64, data []byte) (int64, error) {
	u, have, err := st.upload(id)
	if err != nil {
		return 0, err
	}
	if offset != have {
		return have, errOffset
	}
	if have+int64(len(data)) > u.Size {
		return have, errTooBig
	}
	_, part, _ := st.uploadPaths(id)
	f, err := os.OpenFile(part, os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // checked UUID
	if err != nil {
		return have, err
	}
	n, err := f.Write(data)
	cerr := f.Close()
	if err == nil {
		err = cerr
	}
	return have + int64(n), err
}

var errTooBig = errors.New("products: more bytes than announced")

// abort drops an upload and what arrived of it.
func (st Store) abort(id string) {
	meta, part, err := st.uploadPaths(id)
	if err != nil {
		return
	}
	_ = os.Remove(part)
	_ = os.Remove(meta)
}

// SweepUploads drops uploads older than a day that never finished.
func (st Store) SweepUploads(before time.Time) {
	entries, err := os.ReadDir(st.uploadDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(before) {
			continue
		}
		_ = os.Remove(filepath.Join(st.uploadDir(), e.Name()))
	}
}

// finish checks a complete upload: its sum and what its content is.
func (st Store) finish(id string) (Upload, Detected, []byte, string, error) {
	u, have, err := st.upload(id)
	if err != nil {
		return u, Detected{}, nil, "", err
	}
	if have != u.Size {
		return u, Detected{}, nil, "", fmt.Errorf("products: upload incomplete: %d of %d bytes", have, u.Size)
	}
	_, part, _ := st.uploadPaths(id)
	f, err := os.Open(part) //nolint:gosec // checked UUID
	if err != nil {
		return u, Detected{}, nil, "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return u, Detected{}, nil, "", err
	}
	d, err := Detect(part)
	return u, d, h.Sum(nil), part, err
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

// Detect reads what a file contains from its bytes, never its name.
func Detect(path string) (Detected, error) {
	f, err := os.Open(path) //nolint:gosec // callers pass a stored or uploading file's path
	if err != nil {
		return Detected{}, err
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 4096)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	kind := ""
	switch {
	case bytes.HasPrefix(head, []byte("%PDF-")):
		kind = "pdf"
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		kind = zipKind(path)
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
	case n > 0 && utf8.Valid(head) && !bytes.ContainsRune(head, 0):
		kind = "text"
	}
	d := Detected{Kind: kind, ContentType: "application/octet-stream"}
	if k, ok := kinds[kind]; ok {
		d.ContentType = k.ctype
	}
	if kind == "pdf" {
		d.Pages = pdfPages(path)
	}
	return d, nil
}

// zipKind tells an Office document or an EPUB from a plain ZIP archive.
func zipKind(path string) string {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "zip"
	}
	defer func() { _ = r.Close() }()
	has := map[string]bool{}
	for _, f := range r.File {
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

// pdfPages counts a PDF's page objects (its pages, for a PDF that isn't
// compressed in object streams); 0 when unsure.
func pdfPages(path string) int {
	f, err := os.Open(path) //nolint:gosec // a stored file's path
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	// Large PDFs: the first 64 MB tell enough.
	b, err := io.ReadAll(io.LimitReader(f, 64<<20))
	if err != nil {
		return 0
	}
	return len(pageRe.FindAllIndex(b, -1))
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
	if claimed == "" || d.Kind == "" || claimed == d.Kind {
		return claimed, false
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

// randomHex is n random bytes, hex encoded.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
