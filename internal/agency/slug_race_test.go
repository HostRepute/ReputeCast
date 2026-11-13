package agency_test

// Clients asked for under the same address at the same moment: the slug
// is checked and written under the instance's workspace slug lock, so one
// client gets it and the others are told it's taken, not a 500.

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestClientsAskingForOneSlugAtTheSameMoment(t *testing.T) {
	_, p := northwind(t)
	var v staffView
	p["tomas"].Do("GET", "/agency/staff", nil).MustStatus(t, 200).JSON(t, &v)
	owner := v.user(t, "Nadia Okafor")

	const n = 6
	out := make([]*apptest.Response, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			out[i] = p["tomas"].Do("POST", "/agency/clients", map[string]any{"name": fmt.Sprintf("Lakeside Books %d", i), "slug": "lakeside",
				"timezone": "UTC", "ownerId": owner})
		})
	}
	close(start)
	wg.Wait()

	created := 0
	for _, r := range out {
		switch r.Status {
		case http.StatusCreated:
			var c struct{ Workspace ref }
			r.JSON(t, &c)
			if c.Workspace.Slug != "lakeside" {
				t.Fatalf("created under %q", c.Workspace.Slug)
			}
			created++
		case http.StatusUnprocessableEntity:
			if !strings.Contains(string(r.Body), `"body.slug"`) {
				t.Fatalf("not a slug problem: %s", r.Body)
			}
		default:
			t.Fatalf("status %d: %s", r.Status, r.Body)
		}
	}
	if created != 1 {
		t.Fatalf("%d clients got the slug", created)
	}
}
