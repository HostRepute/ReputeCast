package workspaces_test

// Workspaces created at the same moment: each create takes the instance's
// workspace slug lock before it picks or checks a slug, so two of them never
// take the same one and the second insert never breaks workspaces_slug_key.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// createAtOnce runs n workspace creates at the same moment, each in a
// transaction of its own, and returns the slugs and errors in order.
func createAtOnce(env *apptest.Env, n int, params func(i int) workspaces.CreateParams) ([]string, []error) {
	ctx := context.Background()
	slugs, errs := make([]string, n), make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			errs[i] = db.Tx(ctx, env.App.Pool, func(tx pgx.Tx) error {
				ws, err := workspaces.Create(ctx, tx, params(i))
				slugs[i] = ws.Slug
				return err
			})
		})
	}
	close(start)
	wg.Wait()
	return slugs, errs
}

func TestWorkspaceSlugsPickedAtTheSameMoment(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@slugs.test", Name: "Owner"})

	const n = 8
	slugs, errs := createAtOnce(env, n, func(int) workspaces.CreateParams {
		return workspaces.CreateParams{Name: "Slug Race", Owner: owner.ID}
	})
	want := map[string]bool{"slug-race": true}
	for i := 2; i <= n; i++ {
		want[fmt.Sprintf("slug-race-%d", i)] = true
	}
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("create %d: %v", i, errs[i])
		}
		if !want[slugs[i]] {
			t.Fatalf("slug %q is unexpected or given twice", slugs[i])
		}
		delete(want, slugs[i])
	}
}

func TestWorkspaceSlugAskedForAtTheSameMoment(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@slugs.test", Name: "Owner"})

	// One create gets the address; the others are told it's taken.
	const n = 6
	slugs, errs := createAtOnce(env, n, func(i int) workspaces.CreateParams {
		return workspaces.CreateParams{Name: fmt.Sprintf("Launch Desk %d", i), Slug: "launch-desk", Owner: owner.ID}
	})
	got := 0
	for i := range n {
		switch {
		case errs[i] == nil && slugs[i] == "launch-desk":
			got++
		case errors.Is(errs[i], workspaces.ErrSlugTaken):
		default:
			t.Fatalf("create %d: %q, %v", i, slugs[i], errs[i])
		}
	}
	if got != 1 {
		t.Fatalf("%d workspaces got the slug", got)
	}

	// A reserved address is taken by the app's own routes.
	_, errs = createAtOnce(env, 1, func(int) workspaces.CreateParams {
		return workspaces.CreateParams{Name: "Admin", Slug: "admin", Owner: owner.ID}
	})
	if !errors.Is(errs[0], workspaces.ErrSlugTaken) {
		t.Fatalf("reserved slug: %v", errs[0])
	}
}
