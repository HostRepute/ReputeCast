package accessibility

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/templates"
)

// The built-in templates are all clear, as the gallery says.
func TestStartersAllClear(t *testing.T) {
	for _, key := range []string{"blank", "weekly-digest", "product-update", "plain-text", "announcement", "product-grid"} {
		src, ok := templates.StarterMJML(key, "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105")
		if !ok {
			t.Fatalf("starter %s missing", key)
		}
		r, err := Check([]Version{{Lang: "en", MJML: src, Source: true}}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		for _, is := range r.Issues {
			t.Errorf("%s: %s (%s)", key, is.Title, is.ID)
		}
	}
}
