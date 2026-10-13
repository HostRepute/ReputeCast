package team_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

func hashToken(t string) []byte { return authn.HashToken(t) }

func addMember(t *testing.T, env *apptest.Env, wsID, userID, roleID uuid.UUID) {
	t.Helper()
	if _, err := env.App.Q.CreateMember(context.Background(), store.CreateMemberParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: userID, RoleID: roleID, Scope: json.RawMessage(`{}`), Source: "invite",
	}); err != nil {
		t.Fatal(err)
	}
}
