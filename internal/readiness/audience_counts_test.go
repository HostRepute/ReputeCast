package readiness_test

import (
	"context"
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// Counting an audience without each contact's lists (Review & send, the
// language switcher, approvals) gives the numbers a run's recipients add up
// to, and the recipients a run routes still carry their lists.
func TestAudienceCountsWithoutLists(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s := &readiness.Service{Q: env.App.Q, Now: func() time.Time { return now }}
	for _, name := range []string{"Weekly Digest #142", "November Product Update"} {
		var audience json.RawMessage
		if err := env.App.Pool.QueryRow(ctx, `SELECT audience FROM campaigns WHERE workspace_id = $1 AND name = $2`, w.ID, name).Scan(&audience); err != nil {
			t.Fatal(name, err)
		}
		facts, err := s.Audience(ctx, &w, audience)
		if err != nil {
			t.Fatal(err)
		}
		recipients, err := readiness.FinalRecipients(ctx, env.App.Q, w.ID, audience, w.SunsetPolicy, now)
		if err != nil {
			t.Fatal(err)
		}
		languages, mailboxes := map[string]int64{}, map[string]int64{}
		for _, r := range recipients {
			languages[strings.ToLower(strings.TrimSpace(r.Language))]++
			mailboxes[r.MailboxProvider]++
			if len(r.Lists) == 0 {
				t.Fatalf("%s: a recipient without lists: %+v", name, r)
			}
		}
		if facts.Final == 0 || int64(len(recipients)) != facts.Final || !maps.Equal(languages, facts.Languages) || !maps.Equal(mailboxes, facts.Mailboxes) {
			t.Fatalf("%s: counted %d %v %v, recipients %d %v %v", name, facts.Final, facts.Languages, facts.Mailboxes, len(recipients), languages, mailboxes)
		}
	}
}
