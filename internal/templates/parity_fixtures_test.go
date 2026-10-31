package templates_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hostrepute/reputecast/internal/templates"
)

// TestWriteParityFixtures writes the starters, as a workspace gets them, to
// the gomjml ↔ mjml-browser parity fixtures (internal/mjml/testdata/parity,
// CO-3). It only runs with REPUTECAST_WRITE_PARITY=1, after a starter
// changes; then the composer's parity test (UPDATE_PARITY=1) compiles them
// with mjml-browser.
func TestWriteParityFixtures(t *testing.T) {
	if os.Getenv("REPUTECAST_WRITE_PARITY") != "1" {
		t.Skip("set REPUTECAST_WRITE_PARITY=1 to write the parity fixtures")
	}
	for _, key := range []string{"blank", "weekly-digest", "product-update", "plain-text", "announcement", "product-grid", "blog-digest"} {
		src, ok := templates.StarterMJML(key, "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105")
		if !ok {
			t.Fatalf("starter %s missing", key)
		}
		if err := os.WriteFile(filepath.Join("..", "mjml", "testdata", "parity", "starter-"+key+".mjml"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
