// Package retryaudience is the audience of a retry (review AN-7, R5.L5):
// the addresses of an issue that a mailbox provider blocked for the
// sender's reputation or policy. They are valid addresses, so once the
// block is fixed the issue goes to them again, once, as a campaign of its
// own: the issue's lists and segments, with "retry_of" naming the issue.
// The sending engine and Send Readiness both keep only who Load says, so
// the count Review & send confirms is who gets it, and the audience's
// usual rules (unsubscribed, suppressed, paused) still leave people out.
package retryaudience

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Key is the audience key that names the issue a campaign retries.
const Key = "retry_of"

// Of reads the issue a campaign's audience retries.
func Of(raw json.RawMessage) (uuid.UUID, bool) {
	var a map[string]any
	if json.Unmarshal(raw, &a) != nil {
		return uuid.Nil, false
	}
	s, _ := a[Key].(string)
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// With is an audience that retries an issue: the issue's own audience with
// Key naming it.
func With(raw json.RawMessage, issue uuid.UUID) (json.RawMessage, error) {
	a := map[string]any{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
	}
	a[Key] = issue.String()
	return json.Marshal(a)
}

// Without is an audience that retries nothing (a copy of a retry).
func Without(raw json.RawMessage) json.RawMessage {
	var a map[string]any
	if json.Unmarshal(raw, &a) != nil || a == nil {
		return raw
	}
	if _, ok := a[Key]; !ok {
		return raw
	}
	delete(a, Key)
	b, err := json.Marshal(a)
	if err != nil {
		return raw
	}
	return b
}

// Keep says who of an audience may get it: everyone, or for a retry the
// issue's blocked addresses.
type Keep struct {
	retry bool
	ids   map[uuid.UUID]bool
}

// Load reads who of a campaign's audience may get it: for a retry
// (audience naming an issue), the issue's contacts whose email was blocked
// for the sender's reputation or policy and never delivered, who haven't
// been sent a retry of it yet; except is the retry itself, whose own
// messages don't count. Without a retry, everyone.
func Load(ctx context.Context, q *store.Queries, wsID uuid.UUID, audience json.RawMessage, except uuid.UUID) (Keep, error) {
	issue, ok := Of(audience)
	if !ok {
		return Keep{}, nil
	}
	return LoadIssue(ctx, q, wsID, issue, except)
}

// LoadIssue reads who a retry of an issue may go to (Load).
func LoadIssue(ctx context.Context, q *store.Queries, wsID, issue, except uuid.UUID) (Keep, error) {
	ids, err := q.RetryBlockedContacts(ctx, store.RetryBlockedContactsParams{CampaignID: &issue, WorkspaceID: wsID, RetryID: except})
	if err != nil {
		return Keep{}, err
	}
	k := Keep{retry: true, ids: make(map[uuid.UUID]bool, len(ids))}
	for _, id := range ids {
		k.ids[id] = true
	}
	return k, nil
}

// Keeps reports whether a contact may get the campaign.
func (k Keep) Keeps(id uuid.UUID) bool { return !k.retry || k.ids[id] }

// Retry reports whether the audience is a retry's.
func (k Keep) Retry() bool { return k.retry }

// Len is how many contacts a retry may go to.
func (k Keep) Len() int { return len(k.ids) }
