package readiness_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// TestAssembleRetry: Review & send says what a retry is (review AN-7):
// whose issue it sends again, to how many of its blocked addresses, and to
// send it once the block is fixed. Other campaigns have no such line.
func TestAssembleRetry(t *testing.T) {
	if _, ok := byKey(readiness.Assemble(digest()))["audience.retry"]; ok {
		t.Fatal("a campaign that isn't a retry has a retry line")
	}
	f := digest()
	f.Audience.Final = 9
	f.Audience.Retry = &readiness.RetryAudienceFacts{Issue: "Weekly Digest #141", Blocked: 11}
	res := readiness.Assemble(f)
	c, ok := byKey(res)["audience.retry"]
	if !ok {
		t.Fatalf("no retry line: %v", statuses(res, readiness.Pass))
	}
	if c.Status != readiness.Pass || c.Group != readiness.GroupAudience {
		t.Fatalf("retry line %s in %s", c.Status, c.Group)
	}
	if c.Title != "A retry of Weekly Digest #141: 9 addresses blocked for the sender's reputation" {
		t.Fatalf("title %q", c.Title)
	}
	if !strings.Contains(c.Detail, "11 addresses of Weekly Digest #141") || !strings.Contains(c.Detail, "once the block is fixed") {
		t.Fatalf("detail %q", c.Detail)
	}
	if len(c.Fixes) != 1 || c.Fixes[0].Kind != "link" || !strings.HasSuffix(c.Fixes[0].Href, "/deliverability") {
		t.Fatalf("no link to Deliverability: %+v", c.Fixes)
	}
	// An empty audience says so instead.
	f.Audience.Empty = true
	if _, ok := byKey(readiness.Assemble(f))["audience.retry"]; ok {
		t.Fatal("an empty retry still has its line")
	}
}
