package scoring

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/testdb"
)

// Review 2026-11 AT-14: an automation email's lists are the lists of every
// "Joined list" trigger of its flow, not the first trigger's only.
func TestAutomationEmailListsReadEveryTrigger(t *testing.T) {
	pool := testdb.New(t)
	digest, beta := uuid.New(), uuid.New()
	triggers := `[{"kind": "tag_added", "tag": "vip"}, {"kind": "joined_list", "listId": "` + digest.String() + `"},
		{"kind": "joined_list", "listId": "` + beta.String() + `"}, {"kind": "joined_list", "listId": "not-an-id"}]`
	var lists []uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT `+listsSQL+`
		FROM (SELECT NULL::uuid AS id, NULL::jsonb AS audience) cp, (SELECT gen_random_uuid() AS id, $1::jsonb AS triggers) a`, triggers).Scan(&lists); err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 || !slices.Contains(lists, digest) || !slices.Contains(lists, beta) {
		t.Fatalf("lists: %v", lists)
	}
}
