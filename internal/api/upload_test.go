package api

import (
	"context"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
)

type rawUpload struct {
	RawBody []byte `contentType:"text/csv"`
}

type jsonUpload struct {
	Body struct {
		Name string `json:"name"`
	}
}

// Huma reads a body with a 5 s deadline unless the operation sets its own;
// a raw upload gets the server's read timeout instead.
func TestRawUploadsGetTheReadTimeout(t *testing.T) {
	_, h := humatest.New(t)
	raw := func(context.Context, *rawUpload) (*struct{}, error) { return nil, nil }
	Post(h, Op{ID: "raw", Path: "/raw", Access: Public}, raw)
	Post(WithReadTimeout(h, 2*time.Minute), Op{ID: "raw-configured", Path: "/raw-configured", Access: Public}, raw)
	Post(h, Op{ID: "json", Path: "/json", Access: Public}, func(context.Context, *jsonUpload) (*struct{}, error) { return nil, nil })
	for path, want := range map[string]time.Duration{
		Version + "/raw":            UploadReadTimeout,
		Version + "/raw-configured": 2 * time.Minute,
		Version + "/json":           5 * time.Second,
	} {
		if got := h.OpenAPI().Paths[path].Post.BodyReadTimeout; got != want {
			t.Errorf("%s reads its body for %s, want %s", path, got, want)
		}
	}
}
