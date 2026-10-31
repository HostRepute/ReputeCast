package retryaudience

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestAudienceKey: a retry's audience is its issue's, naming the issue; a
// copy of it names nothing, and everything else about the audience stays.
func TestAudienceKey(t *testing.T) {
	issue := uuid.Must(uuid.NewV7())
	raw := json.RawMessage(`{"lists":["0192c0de-0000-7000-8000-000000000001"],"exclude_tiers":["sunset"],"paid":{"who":"paid"}}`)
	retry, err := With(raw, issue)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := Of(retry); !ok || got != issue {
		t.Fatalf("Of(With) = %v %v, want %v", got, ok, issue)
	}
	var a map[string]any
	if err := json.Unmarshal(retry, &a); err != nil {
		t.Fatal(err)
	}
	if a["paid"] == nil || a["exclude_tiers"] == nil || len(a["lists"].([]any)) != 1 {
		t.Fatalf("the retry lost its issue's audience: %s", retry)
	}
	copied := Without(retry)
	if _, ok := Of(copied); ok {
		t.Fatalf("a copy still retries: %s", copied)
	}
	if err := json.Unmarshal(copied, &a); err != nil || a["paid"] == nil {
		t.Fatalf("the copy lost the audience: %s", copied)
	}
	// Without leaves an audience that retries nothing as it is.
	if got := Without(raw); string(got) != string(raw) {
		t.Fatalf("Without changed %s to %s", raw, got)
	}
	for _, bad := range []string{``, `null`, `{}`, `{"retry_of":""}`, `{"retry_of":"nope"}`, `{"retry_of":"00000000-0000-0000-0000-000000000000"}`, `[1]`} {
		if _, ok := Of(json.RawMessage(bad)); ok {
			t.Errorf("Of(%q) is a retry", bad)
		}
	}
	if fresh, err := With(nil, issue); err != nil || string(fresh) != `{"retry_of":"`+issue.String()+`"}` {
		t.Fatalf("With(nil) = %s, %v", fresh, err)
	}
}

// TestKeep: without a retry everyone may get a campaign; a retry keeps
// only its issue's blocked contacts.
func TestKeep(t *testing.T) {
	a, b := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	var all Keep
	if !all.Keeps(a) || all.Retry() || all.Len() != 0 {
		t.Fatal("an audience without a retry keeps everyone")
	}
	k := Keep{retry: true, ids: map[uuid.UUID]bool{a: true}}
	if !k.Keeps(a) || k.Keeps(b) || !k.Retry() || k.Len() != 1 {
		t.Fatalf("a retry keeps its blocked contacts only: %+v", k)
	}
	if none := (Keep{retry: true}); none.Keeps(a) {
		t.Fatal("a retry with nobody left keeps nobody")
	}
}
