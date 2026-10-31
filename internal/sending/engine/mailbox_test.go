package engine_test

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testMailboxFromMX sends to company addresses whose mail is hosted by
// Google Workspace and Microsoft 365 (review F-1): the send's start reads
// their domains' MX records, and they are routed, counted and reported as
// Gmail and Outlook recipients, keeping which hosting it was.
func testMailboxFromMX(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	// The MX answers, as the classifier's lookup cached them.
	f.Exec(`INSERT INTO mx_cache (domain, has_mx, provider) VALUES ('northwind-hq.example', true, 'Google Workspace'), ('contoso-mail.example', true, 'Microsoft 365')
		ON CONFLICT (domain) DO UPDATE SET has_mx = excluded.has_mx, provider = excluded.provider, checked_at = now()`)
	f.AddContacts(
		sendtest.ContactSpec{N: 12, Domain: "northwind-hq.example", Engaged: true, Unclassified: true},
		sendtest.ContactSpec{N: 8, Domain: "contoso-mail.example", Engaged: true, Unclassified: true},
	)
	f.StartSend(engine.StartOptions{})

	for domain, want := range map[string]struct {
		provider, detail string
		n                int64
	}{
		"northwind-hq.example": {"gmail", "Google Workspace", 12},
		"contoso-mail.example": {"outlook", "Microsoft 365", 8},
	} {
		if n := f.Count(`SELECT count(*) FROM contacts WHERE workspace_id = $1 AND email LIKE '%@' || $2 AND mailbox_provider = $3 AND mailbox_detail = $4 AND mailbox_checked_at IS NOT NULL`,
			f.WS, domain, want.provider, want.detail); n != want.n {
			t.Fatalf("%s: %d contacts classified %s (%s), want %d", domain, n, want.provider, want.detail, want.n)
		}
		if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND email LIKE '%@' || $2 AND mailbox_provider = $3`, f.Campaign, domain, want.provider); n != want.n {
			t.Fatalf("%s: %d messages for %s, want %d", domain, n, want.provider, want.n)
		}
	}
	// Microsoft 365 is Outlook for routing and warm-up caps: its messages
	// carry the mailbox provider the routing and Postal's caps read.
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND email LIKE '%@contoso-mail.example' AND mailbox_provider <> 'outlook'`, f.Campaign); n != 0 {
		t.Fatalf("%d Microsoft 365 recipients not routed as Outlook", n)
	}
	if _, done := f.Drive(3 * time.Hour); !done {
		f.Dump()
		t.Fatal("the send didn't finish")
	}
	// The per-mailbox counters (live monitor, guard, report) count them
	// under their mailbox provider.
	if n := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'gmail'`, f.Campaign); n != 12 {
		t.Fatalf("gmail counters: %d sent, want 12", n)
	}
	if n := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'outlook'`, f.Campaign); n != 8 {
		t.Fatalf("outlook counters: %d sent, want 8", n)
	}
}
