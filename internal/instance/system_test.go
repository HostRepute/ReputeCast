package instance_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

// Instance admin › System (review 2026-11 OP-10): failed jobs can be run
// again, cancelled or deleted, each audited; queues, the database and the
// files folder show.
func TestSystemTab(t *testing.T) {
	env, admin := setup(t)
	ctx := context.Background()
	var acme string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	insert := func(kind, state string) int64 {
		t.Helper()
		var id int64
		finalized := "NULL"
		attempt := 3
		if state == "discarded" {
			finalized, attempt = "now()", 12
		}
		err := env.App.Pool.QueryRow(ctx, `
			INSERT INTO river_job (args, kind, max_attempts, queue, state, attempt, attempted_at, scheduled_at, finalized_at, errors)
			VALUES ($1, $2, 12, 'events', $3::river_job_state, $4::int, now(), now() + interval '5 minutes', `+finalized+`,
			        ARRAY[jsonb_build_object('at', now(), 'attempt', $4::int, 'error', 'dial tcp 203.0.113.40:443: i/o timeout', 'trace', '')])
			RETURNING id`, `{"workspaceId":"`+acme+`"}`, kind, state, attempt).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	retrying := insert("webhooks.deliver", "retryable")
	gaveUp := insert("digests.build_issue", "discarded")
	other := insert("imports.run", "discarded")

	var view struct {
		Jobs     struct{ Retrying, GivenUp int64 }
		Queues   []struct{ Name string }
		Database struct {
			Bytes         int64
			SchemaCurrent bool
			Tables        []struct {
				Name  string
				Bytes int64
			}
		}
		Files struct {
			Dir     string
			Missing bool
		}
	}
	admin.Do("GET", "/admin/system", nil).MustStatus(t, 200).JSON(t, &view)
	if view.Jobs.Retrying < 1 || view.Jobs.GivenUp < 2 || len(view.Queues) < 5 || view.Queues[0].Name != "sending" {
		t.Fatalf("jobs and queues: %+v", view)
	}
	if view.Database.Bytes == 0 || !view.Database.SchemaCurrent || len(view.Database.Tables) == 0 || view.Database.Tables[0].Bytes == 0 || view.Files.Dir == "" {
		t.Fatalf("database and files: %+v", view)
	}

	var list struct {
		Items []struct {
			ID        int64
			Kind      string
			State     string
			Error     string
			Workspace *struct{ Slug string }
		}
	}
	admin.Do("GET", "/admin/system/jobs", nil).MustStatus(t, 200).JSON(t, &list)
	byID := map[int64]int{}
	for i, j := range list.Items {
		byID[j.ID] = i
	}
	r, ok := byID[retrying]
	if !ok || list.Items[r].State != "retrying" || list.Items[r].Error == "" || list.Items[r].Workspace == nil || list.Items[r].Workspace.Slug != "acme-newsletter" {
		t.Fatalf("the retrying job: %+v", list.Items)
	}
	admin.Do("GET", "/admin/system/jobs?state=given_up", nil).MustStatus(t, 200).JSON(t, &list)
	for _, j := range list.Items {
		if j.State != "given_up" {
			t.Fatalf("given up only: %+v", list.Items)
		}
	}

	path := func(id int64, action string) string { return "/admin/system/jobs/" + strconv.FormatInt(id, 10) + "/" + action }
	var res struct{ State string }
	admin.Do("POST", path(gaveUp, "retry"), nil).MustStatus(t, 200).JSON(t, &res)
	if res.State != "available" {
		t.Fatalf("retry: %+v", res)
	}
	// Retried, it isn't a failed job any more.
	p := admin.Do("POST", path(gaveUp, "retry"), nil).MustStatus(t, http.StatusConflict).Problem(t)
	if p["code"] != "not_failed" {
		t.Fatalf("twice: %+v", p)
	}
	admin.Do("POST", path(retrying, "discard"), nil).MustStatus(t, 200).JSON(t, &res)
	if res.State != "cancelled" {
		t.Fatalf("cancel: %+v", res)
	}
	admin.Do("POST", path(other, "discard"), nil).MustStatus(t, 200).JSON(t, &res)
	var left int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE id = $1`, other).Scan(&left)
	if res.State != "deleted" || left != 0 {
		t.Fatalf("delete: %+v, %d left", res, left)
	}
	admin.Do("POST", path(999999999, "retry"), nil).MustStatus(t, http.StatusNotFound)

	var audited int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('instance.job_retried', 'instance.job_cancelled', 'instance.job_deleted') AND workspace_id IS NULL`).Scan(&audited); err != nil {
		t.Fatal(err)
	}
	if audited != 3 {
		t.Fatalf("audited %d of 3", audited)
	}

	// Only instance admins.
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	sofia.Do("GET", "/admin/system", nil).MustStatus(t, http.StatusForbidden)
	sofia.Do("POST", path(retrying, "retry"), nil).MustStatus(t, http.StatusForbidden)
}
