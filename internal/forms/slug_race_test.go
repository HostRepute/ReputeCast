package forms_test

// Requests that pick a form's slug at the same moment in one workspace: a
// create or a rename takes the workspace's slug lock first, so two of them
// never pick the same slug and the second insert never breaks
// forms_slug_key with a 500.

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestFormSlugsPickedAtTheSameMoment(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@slugs.test", Name: "Owner"})
	ws := env.CreateWorkspace("Slug Race", owner.ID)
	c := env.Client()
	c.SignIn(owner.Email, "")
	forms := "/workspaces/" + ws.Slug + "/forms"

	const n = 8
	// race sends n requests at once and returns their responses.
	race := func(do func(i int) *apptest.Response) []*apptest.Response {
		out := make([]*apptest.Response, n)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range n {
			wg.Go(func() {
				<-start
				out[i] = do(i)
			})
		}
		close(start)
		wg.Wait()
		return out
	}
	// slugs checks every response and that the slugs are base, base-2 … base-n.
	slugs := func(rs []*apptest.Response, status int, base string) {
		t.Helper()
		want := map[string]bool{base: true}
		for i := 2; i <= n; i++ {
			want[fmt.Sprintf("%s-%d", base, i)] = true
		}
		for _, r := range rs {
			var f formJSON
			r.MustStatus(t, status).JSON(t, &f)
			if !want[f.Slug] {
				t.Fatalf("slug %q is unexpected or given twice", f.Slug)
			}
			delete(want, f.Slug)
		}
	}

	created := race(func(int) *apptest.Response {
		return c.Do("POST", forms, map[string]any{"name": "Launch popup", "kind": "popup"})
	})
	slugs(created, 201, "launch-popup")

	// Renamed to the same new name at once, from names of their own.
	ids := make([]string, n)
	for i := range n {
		var f formJSON
		c.Do("POST", forms, map[string]any{"name": fmt.Sprintf("Form %d", i), "kind": "embedded"}).MustStatus(t, 201).JSON(t, &f)
		ids[i] = f.ID
	}
	renamed := race(func(i int) *apptest.Response {
		return c.Do("POST", forms+"/"+ids[i]+"/rename", map[string]any{"name": "Spring popup"})
	})
	slugs(renamed, 200, "spring-popup")
}
