// Package permcheck checks a second permission inside a handler, for
// operations whose requirement depends on the request (saving a segment in
// SQL mode needs segments:sql on top of segments:edit). Denials are written
// to the audit log and name who can help, like the API middleware does.
package permcheck

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Require returns nil when the caller holds perm; otherwise it logs the
// denial and returns a 403 problem naming up to two members who can help.
func Require(ctx context.Context, q *store.Queries, log *audit.Logger, wsID uuid.UUID, perm authz.Permission, action string) error {
	p := api.PrincipalFrom(ctx)
	if p.Can(perm) {
		return nil
	}
	_ = log.Log(ctx, audit.Event{
		WorkspaceID: &wsID, Action: action, Result: audit.Denied, Reason: "missing " + string(perm),
		TargetType: "operation", TargetID: action,
	})
	return api.Forbidden(perm, Helpers(ctx, q, wsID, perm))
}

// Helpers names up to two active members who hold perm.
func Helpers(ctx context.Context, q *store.Queries, wsID uuid.UUID, perm authz.Permission) []string {
	rows, err := q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return nil
	}
	var names []string
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			names = append(names, r.Name)
			if len(names) == 2 {
				break
			}
		}
	}
	return names
}
