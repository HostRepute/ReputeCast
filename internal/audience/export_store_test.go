package audience_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/blobstore"
)

// partsStore stops an export after n of its parts are stored: cancelling
// the job's context once the part is up (as a worker that shuts down,
// before the batch's progress is recorded), or failing the part's Put.
type partsStore struct {
	blobstore.Store
	n      int
	cancel context.CancelFunc
	fail   error
}

func (p *partsStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if !strings.Contains(key, ".parts/") {
		return p.Store.Put(ctx, key, r, size, contentType)
	}
	p.n--
	if p.n == 0 && p.fail != nil {
		return p.fail
	}
	err := p.Store.Put(ctx, key, r, size, contentType)
	if p.n == 0 && p.cancel != nil {
		p.cancel()
	}
	return err
}

// bulkExport starts an export of 4,500 contacts (three batches); run runs
// it, through p when it isn't nil.
func bulkExport(t *testing.T) (run func(context.Context, *partsStore) error, parts func() []string, download func() [][]string, status func() (string, int64)) {
	t.Helper()
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	ws := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name)
		SELECT gen_random_uuid(), $1, 'bulk' || g || '@paging.example', 'Bulk' FROM generate_series(1, 4500) g`, ws); err != nil {
		t.Fatal(err)
	}
	var job jobView
	c.Do(http.MethodPost, base+"/contacts/jobs/export", map[string]any{"allMatching": true, "filter": map[string]any{"q": "bulk"}}).MustStatus(t, 202).JSON(t, &job)
	id := uuid.MustParse(job.ID)
	run = func(ctx context.Context, p *partsStore) error {
		svc := env.App.Audience()
		if p != nil {
			p.Store = env.App.Blobs
			svc.Blobs = p
		}
		return svc.RunJob(ctx, id)
	}
	parts = func() []string {
		keys := []string{}
		_ = env.App.Blobs.List(ctx, "exports/"+ws.String()+"/"+job.ID+".csv.parts/", func(k string, _ blobstore.Info) error {
			keys = append(keys, k)
			return nil
		})
		return keys
	}
	download = func() [][]string {
		c.Do(http.MethodGet, base+"/contacts/jobs/"+job.ID, nil).MustStatus(t, 200).JSON(t, &job)
		if job.DownloadURL == "" {
			t.Fatalf("no download: %+v", job)
		}
		recs, err := csv.NewReader(bytes.NewReader(c.Do(http.MethodGet, job.DownloadURL, nil).MustStatus(t, 200).Body)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return recs
	}
	status = func() (string, int64) {
		var s string
		var done int64
		if err := env.App.Pool.QueryRow(ctx, `SELECT status, done FROM audience_jobs WHERE id = $1`, id).Scan(&s, &done); err != nil {
			t.Fatal(err)
		}
		return s, done
	}
	return run, parts, download, status
}

// Object storage can't append: each batch of an export is a part, and the
// file is the parts joined. A worker that stops after a batch's part went
// up but before its progress was recorded leaves that part; the job run
// again writes the batch over it, so no row comes twice and none is
// missing, and once joined the parts go.
func TestExportResumesFromItsParts(t *testing.T) {
	run, parts, download, status := bulkExport(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := &partsStore{n: 2, cancel: cancel}
	if err := run(ctx, stop); err == nil {
		t.Fatal("the stopped run reported no error, so River wouldn't run it again")
	}
	if s, done := status(); s != "running" || done != 2000 {
		t.Fatalf("stopped: %s, %d done", s, done)
	}
	if got := parts(); len(got) != 2 {
		t.Fatalf("parts after the stop: %v", got)
	}
	if err := run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if s, done := status(); s != "done" || done != 4500 {
		t.Fatalf("resumed: %s, %d done", s, done)
	}
	recs := download()
	seen := map[string]bool{}
	for _, r := range recs[1:] {
		if seen[r[0]] {
			t.Fatalf("%s exported twice", r[0])
		}
		seen[r[0]] = true
	}
	if len(recs) != 4501 || recs[0][0] != "email" || len(seen) != 4500 {
		t.Fatalf("file: %d rows, %d addresses", len(recs), len(seen))
	}
	if got := parts(); len(got) != 0 {
		t.Errorf("parts left once joined: %v", got)
	}
}

// A failed export keeps none of its rows.
func TestFailedExportKeepsNothing(t *testing.T) {
	run, parts, _, status := bulkExport(t)
	if err := run(context.Background(), &partsStore{n: 2, fail: errors.New("the bucket is full")}); err != nil {
		t.Fatal(err)
	}
	if s, _ := status(); s != "failed" {
		t.Fatalf("status %s", s)
	}
	if got := parts(); len(got) != 0 {
		t.Errorf("a failed export left %v", got)
	}
}
