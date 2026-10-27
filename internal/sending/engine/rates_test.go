package engine_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testRatesDuringSend: a person who can send changes how fast a mailbox
// provider gets mail for the rest of the send (review GS-13). The rate is
// checked and audited, people who can't send can't change it, the send
// loop keeps to it within seconds, and 0 gives the mailbox provider the
// sending providers' full rate again.
func testRatesDuringSend(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 400, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true},
		sendtest.ContactSpec{N: 100, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	setClock(t, f, f.Start)
	path := "/workspaces/acme-newsletter/campaigns/" + f.Campaign.String() + "/live/rates"
	viewer := f.Env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("PUT", path, map[string]any{"rates": map[string]any{"outlook": 2}}).MustStatus(t, 403)
	linh := f.Env.Client()
	linh.SignInDemo("linh@acme.io")
	linh.Do("PUT", path, map[string]any{"rates": map[string]any{"hotmail": 2}}).MustStatus(t, 422)
	linh.Do("PUT", path, map[string]any{"rates": map[string]any{"outlook": 0.05}}).MustStatus(t, 422)
	linh.Do("PUT", path, map[string]any{"rates": map[string]any{"outlook": 5000}}).MustStatus(t, 422)
	linh.Do("PUT", path, map[string]any{"rates": map[string]any{}}).MustStatus(t, 422)
	linh.Do("PUT", path, map[string]any{"rates": map[string]any{"outlook": 2}}).MustStatus(t, 200)

	rates := func() map[string]float64 {
		t.Helper()
		var raw []byte
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT throttles FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		out := map[string]float64{}
		_ = json.Unmarshal(raw, &out)
		return out
	}
	if r := rates(); r["outlook"] != 2 {
		t.Fatalf("rates %v, want Outlook at 2 /s", r)
	}
	var summary string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT metadata->>'summary' FROM audit_events WHERE target_id = $1 AND action = 'campaign.rates_changed'`,
		f.Campaign.String()).Scan(&summary); err != nil {
		t.Fatalf("not audited: %v", err)
	}
	if !strings.Contains(summary, "Outlook") || !strings.HasSuffix(summary, "→ 2 /s") {
		t.Fatalf("audited as %q, want what Outlook's rate was and is", summary)
	}

	sent := func(mbp string) int64 {
		t.Helper()
		return f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = $2 AND sent_at IS NOT NULL`, f.Campaign, mbp)
	}
	at, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if n := sent("outlook"); n == 0 || n > 2*10+2 {
		f.Dump()
		t.Fatalf("%d sent to Outlook in 10 s, want up to 20: 2 /s", n)
	}
	if n := sent("gmail"); n < 50 {
		t.Fatalf("%d sent to Gmail in 10 s: its rate didn't change", n)
	}

	// Back to the full rate.
	setClock(t, f, at)
	linh.Do("PUT", path, map[string]any{"rates": map[string]any{"outlook": 0}}).MustStatus(t, 200)
	if r := rates(); r["outlook"] != 0 {
		t.Fatalf("rates %v, want Outlook at the full rate (0)", r)
	}
	before := sent("outlook")
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 20*time.Second); err != nil {
		t.Fatal(err)
	}
	if n := sent("outlook") - before; n <= 2*20+5 {
		f.Dump()
		t.Fatalf("%d sent to Outlook in 20 s at the full rate, want more than at 2 /s", n)
	}
}
