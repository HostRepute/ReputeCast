// Package readinesstest runs Send Readiness in integration tests. Runs are
// background jobs (SR-8) and a test's app runs no workers, so a run asked
// for through the API waits in the job queue until the test works it.
package readinesstest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// Run asks for a run of a campaign's checks through the API (as c), works
// it and returns the stored run as c reads it (in c's language).
func Run(t testing.TB, env *apptest.Env, c *apptest.Client, workspace, campaignID string) readiness.ReadinessRun {
	t.Helper()
	var run readiness.ReadinessRun
	Post(t, env, c, "/workspaces/"+workspace+"/campaigns/"+campaignID+"/readiness").MustStatus(t, http.StatusOK).JSON(t, &run)
	return run
}

// Post asks for a run at path (…/campaigns/{id}/readiness) as c, works it
// and answers with the stored run as c reads it, as the endpoint answered
// before runs moved to the background. A refused request (403, 409) is
// returned as it is.
func Post(t testing.TB, env *apptest.Env, c *apptest.Client, path string) *apptest.Response {
	t.Helper()
	res := c.Do(http.MethodPost, path, nil)
	switch {
	case res.Status == http.StatusConflict && Work(t, env) > 0:
		// A run a fix queued was waiting: it reads the campaign as it is.
	case res.Status != http.StatusAccepted:
		return res
	default:
		Work(t, env)
	}
	return latest(t, c, strings.TrimSuffix(path, "/readiness")+"/review")
}

// Fix applies fixes at path (…/campaigns/{id}/readiness/fixes) as c and
// works the run they start. The answer carries the run after the fixes in
// "run", as the endpoint answered before runs moved to the background.
func Fix(t testing.TB, env *apptest.Env, c *apptest.Client, path string, body any) *apptest.Response {
	t.Helper()
	res := c.Do(http.MethodPost, path, body)
	if res.Status != http.StatusOK {
		return res
	}
	Work(t, env)
	run := latest(t, c, strings.TrimSuffix(path, "/readiness/fixes")+"/review")
	var out map[string]json.RawMessage
	if err := json.Unmarshal(res.Body, &out); err != nil {
		t.Fatalf("fix outcome: %v", err)
	}
	out["run"] = run.Body
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	res.Body = b
	return res
}

// Work runs the Send Readiness runs waiting in the job queue (those a fix,
// a request for approval or a run asked for left there).
func Work(t testing.TB, env *apptest.Env) int {
	t.Helper()
	n, err := app.ReadinessQueue(context.Background(), env.App)
	if err != nil {
		t.Fatalf("send readiness runs: %v", err)
	}
	return n
}

// Latest is the campaign's latest run as c reads it.
func Latest(t testing.TB, c *apptest.Client, workspace, campaignID string) readiness.ReadinessRun {
	t.Helper()
	var run readiness.ReadinessRun
	latest(t, c, "/workspaces/"+workspace+"/campaigns/"+campaignID+"/review").JSON(t, &run)
	return run
}

// latest reads the run of the review at path into a response of its own.
func latest(t testing.TB, c *apptest.Client, path string) *apptest.Response {
	t.Helper()
	var review struct {
		Run    json.RawMessage
		Failed *readiness.ReadinessRunFailure
	}
	c.Do(http.MethodGet, path, nil).MustStatus(t, http.StatusOK).JSON(t, &review)
	if review.Failed != nil {
		t.Fatalf("the run failed: %s", review.Failed.Error)
	}
	if len(review.Run) == 0 || string(review.Run) == "null" {
		t.Fatal("no run")
	}
	return &apptest.Response{Status: http.StatusOK, Header: http.Header{}, Body: review.Run}
}
