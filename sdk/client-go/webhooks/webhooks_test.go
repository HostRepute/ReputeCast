package webhooks_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/sdk/client-go/webhooks"
)

type vector struct {
	Name    string            `json:"name"`
	Secrets []string          `json:"secrets"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Now     int64             `json:"now"`
	Valid   bool              `json:"valid"`
	Error   string            `json:"error"`
}

// The deliveries the server signs (internal/webhooks/vectors_test.go).
func TestVectors(t *testing.T) {
	b, err := os.ReadFile("../../testdata/webhook-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		ToleranceSeconds int      `json:"toleranceSeconds"`
		Vectors          []vector `json:"vectors"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Vectors) < 10 || time.Duration(file.ToleranceSeconds)*time.Second != webhooks.Tolerance {
		t.Fatalf("%d vectors, tolerance %d s", len(file.Vectors), file.ToleranceSeconds)
	}
	reasons := map[string]error{"missing_header": webhooks.ErrMissingHeader, "timestamp": webhooks.ErrTimestamp, "signature": webhooks.ErrSignature}
	for _, v := range file.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			h := http.Header{}
			for k, val := range v.Headers {
				h.Set(k, val)
			}
			err := webhooks.Verify(v.Secrets, []byte(v.Body), h, time.Unix(v.Now, 0))
			switch {
			case v.Valid && err != nil:
				t.Errorf("refused: %v", err)
			case !v.Valid && !errors.Is(err, reasons[v.Error]):
				t.Errorf("got %v, want %v", err, reasons[v.Error])
			}
		})
	}
}
