package publicpages

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Only a request http.ServeContent answers with the rest of the file is a
// resumed download; everything that gets the file from its start counts
// ("bytes=1-,0-" used to get the whole file, uncounted).
func TestResumesDownload(t *testing.T) {
	modtime := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	file := bytes.Repeat([]byte("0123456789"), 100)
	size := int64(len(file))
	for _, c := range []struct {
		method, rng, ifRange string
		want                 bool
	}{
		{http.MethodGet, "", "", false},
		{http.MethodGet, "bytes=0-", "", false},
		{http.MethodGet, "bytes=0-499", "", false},
		{http.MethodGet, "bytes=1-", "", true},
		{http.MethodGet, "bytes=500-", "", true},
		{http.MethodGet, "bytes=500-999", "", true},
		{http.MethodGet, "bytes=500-5000", "", true},
		{http.MethodGet, "bytes= 500 - ", "", true},
		{http.MethodGet, "bytes=-500", "", true},
		// Several ranges: together longer than the file, ServeContent sends
		// it whole; otherwise they still include the start.
		{http.MethodGet, "bytes=1-,0-", "", false},
		{http.MethodGet, "bytes=1-,2-", "", false},
		{http.MethodGet, "bytes=0-0,500-", "", false},
		{http.MethodGet, "bytes=500-,", "", false},
		{http.MethodGet, "bytes=500-" + strings.Repeat(",", 200), "", false},
		// Ranges it can't satisfy, and what isn't a range.
		{http.MethodGet, "bytes=1000-", "", false},
		{http.MethodGet, "bytes=-1000", "", false},
		{http.MethodGet, "bytes=-0", "", false},
		{http.MethodGet, "bytes=--5", "", false},
		{http.MethodGet, "bytes=600-500", "", false},
		{http.MethodGet, "bytes=abc-", "", false},
		{http.MethodGet, "bytes=500", "", false},
		{http.MethodGet, "items=500-", "", false},
		// If-Range: the file's own time keeps the range; anything else
		// makes ServeContent send the whole file.
		{http.MethodGet, "bytes=500-", modtime.Format(http.TimeFormat), true},
		{http.MethodGet, "bytes=500-", modtime.Add(time.Second).Format(http.TimeFormat), false},
		{http.MethodGet, "bytes=500-", `"an-etag"`, false},
		{http.MethodGet, "bytes=500-", "yesterday", false},
		{http.MethodPost, "bytes=500-", "", false},
	} {
		r := httptest.NewRequest(c.method, "/d/token/file", nil)
		if c.rng != "" {
			r.Header.Set("Range", c.rng)
		}
		if c.ifRange != "" {
			r.Header.Set("If-Range", c.ifRange)
		}
		got := resumesDownload(r, size, modtime)
		if got != c.want {
			t.Errorf("%s Range %q If-Range %q: resumes %v, want %v", c.method, c.rng, c.ifRange, got, c.want)
		}
		if c.method != http.MethodGet {
			continue
		}
		// What ServeContent sends for it: a resume never includes the
		// first byte.
		w := httptest.NewRecorder()
		http.ServeContent(w, r, "", modtime, bytes.NewReader(file))
		whole := w.Code == http.StatusOK && int64(w.Body.Len()) == size
		if got && (w.Code != http.StatusPartialContent || strings.HasPrefix(w.Header().Get("Content-Range"), "bytes 0-") || whole) {
			t.Errorf("Range %q If-Range %q: a resume, but ServeContent answered %d %q", c.rng, c.ifRange, w.Code, w.Header().Get("Content-Range"))
		}
	}

	// The request of the review: the whole file, so it must count.
	r := httptest.NewRequest(http.MethodGet, "/d/token/file", nil)
	r.Header.Set("Range", "bytes=1-,0-")
	w := httptest.NewRecorder()
	http.ServeContent(w, r, "", modtime, bytes.NewReader(file))
	if w.Code != http.StatusOK || int64(w.Body.Len()) != size || resumesDownload(r, size, modtime) {
		t.Fatalf("bytes=1-,0-: %d, %d bytes", w.Code, w.Body.Len())
	}
}
