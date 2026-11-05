package access

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member, API key or agent limited to some lists sees only the contacts
// on one of them, in any status, as Audience's list and search do
// (internal/audience/filter.go). Every other read that names contacts
// applies the same rule through these.

// ScopeLists is the caller's list scope as a query argument: their lists,
// or an empty array when they see every contact. Never nil: Postgres reads
// a nil slice as NULL, and "cardinality(@scope_lists) = 0" would then
// match no one.
func ScopeLists(p *authz.Principal) []uuid.UUID {
	if p == nil || len(p.Scope.Lists) == 0 {
		return []uuid.UUID{}
	}
	return p.Scope.Lists
}

// ContactInScope reports whether the caller may see a contact: they aren't
// limited to some lists, or one of their lists has the contact.
func ContactInScope(ctx context.Context, q *store.Queries, p *authz.Principal, contactID uuid.UUID) (bool, error) {
	if p == nil || len(p.Scope.Lists) == 0 {
		return true, nil
	}
	return q.AccessScopeContact(ctx, store.AccessScopeContactParams{ContactID: contactID, ScopeLists: p.Scope.Lists})
}
