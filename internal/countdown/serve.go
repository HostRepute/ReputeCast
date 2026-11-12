package countdown

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Bounds of the image path.
const (
	// atlasCache is how many sets of settings keep their drawn digits.
	atlasCache = 64
	// gifCache is how many images are kept, one per countdown and second.
	gifCache = 512
	// rowCache is how many countdown rows are kept, rowFor how long.
	rowCache = 1024
	rowFor   = time.Minute
	// renders is how many images are drawn at once; a request waits up to
	// slotWait for its turn, then gets the countdown's newest image if it
	// is at most staleFor old.
	renders   = 4
	slotWait  = 2 * time.Second
	staleFor  = 10 * time.Second
	servedFor = time.Minute
	// endedFor is how long an image that no longer changes may be cached.
	endedFor = time.Hour
)

// endedSecond keys the one image of a countdown that has ended.
const endedSecond = math.MinInt64

var errBusy = errors.New("countdown: every renderer is busy")

type gifKey struct {
	id  uuid.UUID
	sec int64
}

type stamped struct {
	sec int64
	gif []byte
}

// row is a countdown as the image path keeps it.
type row struct {
	id, workspace uuid.UUID
	hash          [32]byte
	settings      CountdownSettings
	endsAt        time.Time
	loadedAt      time.Time
	// servedAt is when last_served_at was last written (unix seconds).
	servedAt atomic.Int64
}

// caches are a Service's in-process state of the image path.
type caches struct {
	atlases *lru[[32]byte, *atlas]
	gifs    *lru[gifKey, []byte]
	newest  *lru[uuid.UUID, stamped]
	rows    *lru[uuid.UUID, *row]
	slots   chan struct{}

	drawing  flight[gifKey, []byte]
	building flight[[32]byte, *atlas]
}

func newCaches() *caches {
	return &caches{atlases: newLRU[[32]byte, *atlas](atlasCache), gifs: newLRU[gifKey, []byte](gifCache), newest: newLRU[uuid.UUID, stamped](gifCache),
		rows: newLRU[uuid.UUID, *row](rowCache), slots: make(chan struct{}, renders)}
}

func (s *Service) caches() *caches {
	s.once.Do(func() { s.cache = newCaches() })
	return s.cache
}

// atlas returns the drawn digits of a set of settings.
func (c *caches) atlas(ctx context.Context, hash [32]byte, set CountdownSettings) (*atlas, error) {
	if a, ok := c.atlases.get(hash); ok {
		return a, nil
	}
	return c.building.do(ctx, hash, func() (*atlas, error) {
		if a, ok := c.atlases.get(hash); ok {
			return a, nil
		}
		a, err := newAtlas(set)
		if err == nil {
			c.atlases.put(hash, a)
		}
		return a, err
	})
}

// secondsLeft is how many whole seconds a countdown has left at now, by
// the second now falls in (every request of one second sees one image).
func secondsLeft(endsAt, now time.Time) int64 { return endsAt.Unix() - now.Unix() }

// picture is a countdown's image at now, and whether it has ended. When
// every renderer stays busy it is the countdown's newest image if that is
// recent enough, else errBusy.
func (c *caches) picture(ctx context.Context, r *row, now time.Time) ([]byte, bool, error) {
	left := secondsLeft(r.endsAt, now)
	ended := left <= 0
	key := gifKey{id: r.id, sec: now.Unix()}
	if ended {
		key.sec = endedSecond
	}
	if b, ok := c.gifs.get(key); ok {
		return b, ended, nil
	}
	b, err := c.drawing.do(ctx, key, func() ([]byte, error) {
		if b, ok := c.gifs.get(key); ok {
			return b, nil
		}
		wait := time.NewTimer(slotWait)
		select {
		case c.slots <- struct{}{}:
			wait.Stop()
		case <-wait.C:
			return nil, errBusy
		case <-ctx.Done():
			wait.Stop()
			return nil, ctx.Err()
		}
		defer func() { <-c.slots }()
		a, err := c.atlas(ctx, r.hash, r.settings)
		if err != nil {
			return nil, err
		}
		b, err := a.render(left)
		if err != nil {
			return nil, err
		}
		c.gifs.put(key, b)
		if !ended {
			c.newest.put(r.id, stamped{sec: key.sec, gif: b})
		}
		return b, nil
	})
	if errors.Is(err, errBusy) && !ended {
		if n, ok := c.newest.get(r.id); ok && key.sec-n.sec <= int64(staleFor/time.Second) {
			return n.gif, false, nil
		}
	}
	return b, ended, err
}

// Routes mounts the image, next to the tracking endpoints:
//
//	GET|HEAD /t/cd/{token}.gif
func (s *Service) Routes(r chi.Router) {
	r.Get("/t/cd/{token}", s.serve)
	r.Head("/t/cd/{token}", s.serve)
}

// serve answers the image: drawn for this second while the countdown
// runs (never cached), one frame kept an hour once it has ended.
func (s *Service) serve(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	tracking.NoStore(h)
	h.Set("Cross-Origin-Resource-Policy", "cross-origin")
	token := strings.TrimSuffix(chi.URLParam(r, "token"), ".gif")
	c, err := s.Links.Verify(token, Kind)
	if err != nil || c.MessageID == uuid.Nil || !s.carries(r, c.WorkspaceID) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	ctx := r.Context()
	cd, err := s.row(ctx, c.WorkspaceID, c.MessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		s.log().WarnContext(ctx, "countdown row", "countdown", c.MessageID, "err", err)
		busy(w)
		return
	}
	now := s.now()
	img, ended, err := s.caches().picture(ctx, cd, now)
	if err != nil {
		if !errors.Is(err, errBusy) && !errors.Is(err, context.Canceled) {
			s.log().WarnContext(ctx, "countdown image", "countdown", cd.id, "err", err)
		}
		busy(w)
		return
	}
	s.served(cd, now)
	h.Set("Content-Type", "image/gif")
	h.Set("Content-Length", strconv.Itoa(len(img)))
	if ended {
		h.Del("Pragma")
		h.Del("Expires")
		h.Set("Cache-Control", "public, max-age="+strconv.Itoa(int(endedFor/time.Second)))
	}
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(img)
	}
}

// busy asks the mail client to try again in a second.
func busy(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
}

// carries reports whether the host a request came to may carry the
// workspace's images: one of its own hosts, the instance's, or nobody's.
func (s *Service) carries(r *http.Request, workspace uuid.UUID) bool {
	ok, err := s.Hosts.Carries(r.Context(), r.Host, workspace)
	if err != nil {
		s.log().WarnContext(r.Context(), "countdown host", "err", err)
	}
	return ok
}

// row is a countdown of a workspace, read again after a minute.
func (s *Service) row(ctx context.Context, workspace, id uuid.UUID) (*row, error) {
	c := s.caches()
	now := s.now()
	if r, ok := c.rows.get(id); ok && r.workspace == workspace && now.Sub(r.loadedAt) < rowFor {
		return r, nil
	}
	got, err := s.Q.CountdownGet(ctx, store.CountdownGetParams{ID: id, WorkspaceID: workspace})
	if err != nil {
		return nil, err
	}
	r := &row{id: got.ID, workspace: got.WorkspaceID, endsAt: got.EndsAt, loadedAt: now}
	copy(r.hash[:], got.Hash)
	if err := json.Unmarshal(got.Settings, &r.settings); err != nil {
		return nil, err
	}
	if old, ok := c.rows.get(id); ok {
		r.servedAt.Store(old.servedAt.Load())
	}
	c.rows.put(id, r)
	return r, nil
}

// served notes that the image was opened, at most once a minute.
func (s *Service) served(r *row, now time.Time) {
	last := r.servedAt.Load()
	if now.Unix()-last < int64(servedFor/time.Second) || !r.servedAt.CompareAndSwap(last, now.Unix()) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		before := now.Add(-servedFor)
		if err := s.Q.CountdownServed(ctx, store.CountdownServedParams{ID: r.id, At: &now, Before: &before}); err != nil {
			s.log().WarnContext(ctx, "countdown served", "countdown", r.id, "err", err)
		}
	}()
}
