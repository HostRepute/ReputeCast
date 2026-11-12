package engine_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// M63 G-33: with the holdout group at 5% of campaigns, a send to 10,000
// contacts leaves out the very contacts Send Readiness counts in its
// "Holdout group" row (SR-11: one definition, rc_holdout_bucket), and a
// campaign that includes the group leaves nobody out for it.
func TestHoldoutLeavesOutWhatReadinessCounts(t *testing.T) {
	s := sendtest.NewSuite(t)
	const seed = 20261014
	open := func(f *sendtest.Fixture) {
		f.Exec(`UPDATE holdout_periods SET ended_at = now() WHERE workspace_id = $1 AND ended_at IS NULL`, f.WS)
		f.Exec(`INSERT INTO holdout_periods (id, workspace_id, seed, basis_points, campaigns, automations, started_at) VALUES ($1, $2, $3, 500, true, true, $4)`,
			uuid.Must(uuid.NewV7()), f.WS, int64(seed), f.Start.Add(-60*24*time.Hour))
	}
	facts := func(f *sendtest.Fixture) readiness.AudienceFacts {
		t.Helper()
		w, err := f.Env.App.Q.GetWorkspaceBySlug(f.Ctx, "acme-newsletter")
		if err != nil {
			t.Fatal(err)
		}
		var audience json.RawMessage
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT audience FROM campaigns WHERE id = $1`, f.Campaign).Scan(&audience); err != nil {
			t.Fatal(err)
		}
		rs := &readiness.Service{Q: f.Env.App.Q, Now: func() time.Time { return f.Start }}
		a, err := rs.Audience(f.Ctx, &w, audience)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	excluded := func(f *sendtest.Fixture) engine.Excluded {
		t.Helper()
		var raw []byte
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT excluded FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var x engine.Excluded
		if err := json.Unmarshal(raw, &x); err != nil {
			t.Fatal(err)
		}
		return x
	}

	t.Run("LeftOut", func(t *testing.T) {
		f := s.Fixture(t, "Holdout 5%")
		f.AddContacts(sendtest.ContactSpec{N: 10000, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
		open(f)
		held := f.Count(`SELECT count(*) FROM list_memberships WHERE list_id = $1 AND rc_holdout_bucket(contact_id, $2) < 500`, f.List, int64(seed))
		if held < 440 || held > 560 {
			t.Fatalf("%d of 10,000 in the 5%% holdout group", held)
		}
		a := facts(f)
		if a.HoldoutPercent != 5 || a.HoldoutIncluded || a.Holdout != held || a.Final != 10000-held {
			t.Fatalf("readiness: holdout %v%% included %v left out %d final %d, want %d left out", a.HoldoutPercent, a.HoldoutIncluded, a.Holdout, a.Final, held)
		}
		f.StartSend(engine.StartOptions{})
		if x := excluded(f); x.Holdout != a.Holdout {
			t.Fatalf("the send left out %d for the holdout group, Send Readiness %d", x.Holdout, a.Holdout)
		}
		if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 10000-held {
			t.Fatalf("%d messages queued, want %d", n, 10000-held)
		}
		if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND rc_holdout_bucket(contact_id, $2) < 500`, f.Campaign, int64(seed)); n != 0 {
			t.Fatalf("%d messages for the holdout group", n)
		}
		// A change of the settings mid-build doesn't split the send: the
		// share was captured when the build began.
		var content struct {
			Holdout *struct {
				Seed        int64 `json:"seed"`
				BasisPoints int32 `json:"basis_points"`
			} `json:"holdout"`
		}
		var raw []byte
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT content FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &content); err != nil || content.Holdout == nil || content.Holdout.Seed != seed || content.Holdout.BasisPoints != 500 {
			t.Fatalf("the send's snapshot: %s", raw)
		}
	})

	t.Run("Included", func(t *testing.T) {
		f := s.Fixture(t, "Holdout included")
		f.AddContacts(sendtest.ContactSpec{N: 2000, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
		open(f)
		f.Exec(`UPDATE campaigns SET audience = audience || '{"include_holdout": true}' WHERE id = $1`, f.Campaign)
		held := f.Count(`SELECT count(*) FROM list_memberships WHERE list_id = $1 AND rc_holdout_bucket(contact_id, $2) < 500`, f.List, int64(seed))
		a := facts(f)
		if !a.HoldoutIncluded || a.Holdout != 0 || a.HoldoutReached != held || a.Final != 2000 {
			t.Fatalf("readiness: included %v left out %d reached %d final %d, want %d reached", a.HoldoutIncluded, a.Holdout, a.HoldoutReached, a.Final, held)
		}
		f.StartSend(engine.StartOptions{})
		if x := excluded(f); x.Holdout != 0 {
			t.Fatalf("an included holdout group left %d out", x.Holdout)
		}
		if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 2000 {
			t.Fatalf("%d messages queued, want all 2,000", n)
		}
	})
}
