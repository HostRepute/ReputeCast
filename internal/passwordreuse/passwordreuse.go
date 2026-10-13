// Package passwordreuse enforces the re-use rule of the instance password
// policy (Instance admin › Authentication › Password policy): a new
// password can't be any of the person's last N passwords, the current one
// included. Previous hashes are recorded by a database trigger whenever
// users.password_hash changes, so every way a password changes is covered.
package passwordreuse

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// Choices are the policy values the admin can pick (0 allows re-use).
var Choices = []int32{0, 5, 10}

// Used reports whether password is the current one (current is its hash)
// or one of the n-1 passwords before it. n is the policy; 0 allows re-use.
func Used(ctx context.Context, q store.Querier, userID uuid.UUID, current *string, password string, n int32) (bool, error) {
	if n <= 0 {
		return false, nil
	}
	if current != nil {
		if ok, _ := authn.VerifyPassword(password, *current); ok {
			return true, nil
		}
	}
	if n <= 1 {
		return false, nil
	}
	hashes, err := q.InstanceAdminPasswordHistory(ctx, store.InstanceAdminPasswordHistoryParams{UserID: userID, Lim: n - 1})
	if err != nil {
		return false, err
	}
	for _, h := range hashes {
		if ok, _ := authn.VerifyPassword(password, h); ok {
			return true, nil
		}
	}
	return false, nil
}

// Message explains a rejected password.
func Message(n int32) string {
	return fmt.Sprintf("You used this password recently. Choose one that isn't among your last %d passwords.", n)
}
