package brandkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
)

// A role change that commits while VoiceActor waits for the membership's
// lock leaves the person with the new role's permissions, not without any.
// Under READ COMMITTED the locked read re-checks the updated membership
// against the role row it read first; the new role_id doesn't match it, so
// that read finds no row. The assistant's stream then ended with "Using the
// AI assistant needs a permission you don't have" for a member who still
// had ai:use (TestAssistantLiveAnswerStopsWhenASourceIsLost on CI).
func TestVoiceActorDuringRoleChange(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "owner@voice.test", Name: "Voice Owner"})
	w := env.CreateWorkspace("Voice", u.ID)
	role := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO roles (id, workspace_id, key, name, builtin, permissions) VALUES ($1, $2, 'chat-only', 'Chat only', false, ARRAY['ai:use'])`,
		role, w.ID); err != nil {
		t.Fatal(err)
	}
	p := &authz.Principal{Type: authz.ActorUser, ID: u.ID, UserID: u.ID, WorkspaceID: w.ID}

	// The role change holds the membership's row until it commits.
	tx, err := env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE workspace_members SET role_id = $1 WHERE workspace_id = $2 AND user_id = $3`, role, w.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	type result struct {
		p   *authz.Principal
		err error
	}
	done := make(chan result, 1)
	go func() {
		fresh, err := brandkit.VoiceActor(ctx, env.App.Q, &w, p)
		done <- result{fresh, err}
	}()
	// VoiceActor waits for the row lock: commit only then.
	deadline := time.Now().Add(20 * time.Second)
	for {
		var waiting int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%FROM workspace_members m JOIN roles r%'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("VoiceActor never waited for the role change's lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	r := <-done
	if r.err != nil {
		t.Fatalf("VoiceActor during a role change: %v", r.err)
	}
	if !r.p.Can(authz.AIUse) || r.p.Can(authz.CampaignsView) {
		t.Fatalf("permissions after the role change: %v", r.p.Permissions)
	}
}
