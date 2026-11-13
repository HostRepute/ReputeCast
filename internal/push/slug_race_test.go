package push_test

// Push prompts named alike at the same moment: a prompt is a form, and its
// slug is picked under the workspace's form slug lock, so two prompts never
// pick the same slug and the second insert never breaks forms_slug_key.

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestPushPromptsNamedAlikeAtTheSameMoment(t *testing.T) {
	f := newFixture(t)
	site := f.pushOn(t, 2)

	const n = 6
	out := make([]*apptest.Response, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			out[i] = f.owner.Do(http.MethodPost, f.path+"/push/prompts", map[string]any{"name": "Help center bell", "siteId": site.ID})
		})
	}
	close(start)
	wg.Wait()
	for _, r := range out {
		r.MustStatus(t, 201)
	}

	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT slug FROM forms WHERE workspace_id = $1 AND kind = 'push'`, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := map[string]bool{"help-center-bell": true}
	for i := 2; i <= n; i++ {
		want[fmt.Sprintf("help-center-bell-%d", i)] = true
	}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			t.Fatal(err)
		}
		if !want[slug] {
			t.Fatalf("slug %q is unexpected or given twice", slug)
		}
		delete(want, slug)
	}
	if len(want) != 0 {
		t.Fatalf("missing slugs: %v", want)
	}
}
