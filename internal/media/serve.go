package media

import (
	"bytes"
	"container/list"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type cached struct {
	id          uuid.UUID
	fileName    string
	contentType string
	etag        string
	data        []byte
	created     time.Time
}

// lru keeps recently served images in memory (bounded by bytes) so a
// campaign's first opens don't each read the database.
type lru struct {
	mu    sync.Mutex
	max   int
	size  int
	order *list.List
	items map[uuid.UUID]*list.Element
}

func newLRU(maxBytes int) *lru {
	return &lru{max: maxBytes, order: list.New(), items: map[uuid.UUID]*list.Element{}}
}

func (c *lru) get(id uuid.UUID) (*cached, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[id]; ok {
		c.order.MoveToFront(e)
		return e.Value.(*cached), true
	}
	return nil, false
}

func (c *lru) put(v *cached) {
	if len(v.data) > c.max/4 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[v.id]; ok {
		return
	}
	c.items[v.id] = c.order.PushFront(v)
	c.size += len(v.data)
	for c.size > c.max {
		last := c.order.Back()
		if last == nil {
			break
		}
		old := last.Value.(*cached)
		c.order.Remove(last)
		delete(c.items, old.id)
		c.size -= len(old.data)
	}
}

// ServeHTTP serves GET /media/{id}/{file}. Images are immutable (a new
// upload gets a new id), so they are cached for a year. The file name in
// the path is cosmetic; the id decides.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, PathPrefix)
	idPart, file, _ := strings.Cut(rest, "/")
	if idPart == "starters" && s.Static != nil {
		s.serveStatic(w, r, file)
		return
	}
	id, err := uuid.Parse(idPart)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.once.Do(func() { s.cache = newLRU(32 << 20) })
	item, ok := s.cache.get(id)
	if !ok {
		row, err := s.Q.GetMediaData(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "image unavailable", http.StatusServiceUnavailable)
			return
		}
		item = &cached{id: row.ID, fileName: row.FileName, contentType: row.ContentType, etag: `"` + hex.EncodeToString(row.Sha256[:16]) + `"`, data: row.Data, created: row.CreatedAt}
		s.cache.put(item)
	}
	h := w.Header()
	h.Set("Content-Type", item.contentType)
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("ETag", item.etag)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Disposition", `inline; filename="`+item.fileName+`"`)
	// Images are embedded by mail clients and their proxies on other origins.
	h.Set("Cross-Origin-Resource-Policy", "cross-origin")
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(w, r, item.fileName, item.created, bytes.NewReader(item.data))
}

var staticStart = time.Now()

// serveStatic serves the starter templates' placeholder images.
func (s *Service) serveStatic(w http.ResponseWriter, r *http.Request, file string) {
	if file == "" || strings.ContainsAny(file, "/\\") || !strings.HasSuffix(file, ".png") {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(s.Static, file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "image/png")
	h.Set("Cache-Control", "public, max-age=86400")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cross-Origin-Resource-Policy", "cross-origin")
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(w, r, file, staticStart, bytes.NewReader(data))
}
