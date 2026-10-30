package rules_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/migrations"
)

// sameRules reports whether a workspace's rules are the defaults, in order.
func sameRules(got []rules.ReplyRule) bool {
	if len(got) != len(rules.DefaultReplyRules) {
		return false
	}
	for i, r := range got {
		d := rules.DefaultReplyRules[i]
		if r.Mailbox != d.Mailbox || r.Contains != d.Contains || r.Count != d.Count || r.WindowMinutes != d.WindowMinutes || r.Action != d.Action ||
			r.DurationMinutes != d.DurationMinutes || !r.Enabled {
			return false
		}
	}
	return true
}

// Every workspace starts with the response rules for the replies mailbox
// providers are known for (review GS-12), as the Traffic rules board draws
// them: Outlook's S3150 pauses Outlook on that route until someone
// resumes, Yahoo's TSS04 and TS04 pause Yahoo for 2 hours after 10 in 10
// minutes, Gmail's 421-4.7.28 slows Gmail for 30 minutes after 5.
func TestNewWorkspaceStartsWithDefaultReplyRules(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Owner"})
	ws := env.CreateWorkspace("Rules", owner.ID)
	got, err := rules.ReplyRules(context.Background(), env.App.Q, ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !sameRules(got) {
		t.Fatalf("rules %+v, want the defaults", got)
	}
	byText := map[string]rules.ReplyRule{}
	for _, r := range got {
		byText[r.Contains] = r
	}
	if r := byText["S3150"]; r.Mailbox != "outlook" || r.Count != 1 || r.Action != "pause_route" || r.DurationMinutes != 0 {
		t.Errorf("Outlook's S3150 %+v, want its first reply to pause Outlook on that route until someone resumes", r)
	}
	if r := byText["TS04"]; r.Mailbox != "yahoo" || r.Count != 10 || r.WindowMinutes != 10 || r.DurationMinutes != 120 || rules.KnownReplyKey(r.Contains) != "tss04" {
		t.Errorf("Yahoo's TS04 %+v, want TSS04's rule and meaning", r)
	}
	if r := byText["421-4.7.28"]; r.Mailbox != "gmail" || r.Count != 5 || r.Action != "slow" || r.DurationMinutes != 30 {
		t.Errorf("Gmail's 421-4.7.28 %+v, want 5 in 10 minutes to slow Gmail for 30 minutes", r)
	}
}

// Workspaces made before the defaults get them from the migration, unless
// they have a rule of their own: those keep exactly theirs. Going down
// takes away only defaults nobody touched.
func TestDefaultReplyRulesBackfill(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Owner"})
	bare := env.CreateWorkspace("Bare", owner.ID)
	custom := env.CreateWorkspace("Custom", owner.ID)
	p, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(env.App.Pool), migrations.FS, goose.WithAllowOutofOrder(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	const version = 3312
	if _, err := p.ApplyVersion(ctx, version, false); err != nil {
		t.Fatal(err)
	}
	count := func(ws uuid.UUID) int {
		got, err := rules.ReplyRules(ctx, env.App.Q, ws)
		if err != nil {
			t.Fatal(err)
		}
		return len(got)
	}
	if count(bare.ID) != 0 || count(custom.ID) != 0 {
		t.Fatal("going down left untouched defaults")
	}
	// Before the defaults: Custom set up its own rule.
	own := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO traffic_reply_rules (id, workspace_id, mailbox_provider, contains, min_count, window_seconds, action, created_by)
		VALUES ($1, $2, 'yahoo', 'TSS04', 3, 300, 'alert', $3)`, own, custom.ID, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ApplyVersion(ctx, version, true); err != nil {
		t.Fatal(err)
	}
	got, err := rules.ReplyRules(ctx, env.App.Q, bare.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !sameRules(got) {
		t.Fatalf("Bare's rules %+v, want the defaults", got)
	}
	mine, err := rules.ReplyRules(ctx, env.App.Q, custom.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || mine[0].ID != own {
		t.Fatalf("Custom's rules %+v, want only its own", mine)
	}
	// A default that matched in a send is the workspace's history now:
	// going down keeps it, and the person's own rule.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE traffic_reply_rules SET last_matched_at = now() WHERE workspace_id = $1 AND contains = 'S3150'`, bare.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ApplyVersion(ctx, version, false); err != nil {
		t.Fatal(err)
	}
	if n := count(bare.ID); n != 1 {
		t.Fatalf("%d rules left in Bare, want the one that matched", n)
	}
	if n := count(custom.ID); n != 1 {
		t.Fatalf("%d rules left in Custom, want its own", n)
	}
	if _, err := p.ApplyVersion(ctx, version, true); err != nil {
		t.Fatal(err)
	}
}
