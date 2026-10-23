package pages_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// The pages' server text reads in the member's language.
func TestPagesInVietnamese(t *testing.T) {
	f := newFixture(t)
	vi := f.as("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	if p := vi.Do("GET", pagesAPI+"/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Không tìm thấy trang." {
		t.Fatalf("not found (vi): %v", p)
	}
	if p := f.owner.Do("GET", pagesAPI+"/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Page not found." {
		t.Fatalf("not found (en): %v", p)
	}
}
