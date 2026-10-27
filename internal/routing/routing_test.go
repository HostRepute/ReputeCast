package routing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

var (
	ses      = uuid.MustParse("01990000-0000-7000-8000-000000000001")
	postal   = uuid.MustParse("01990000-0000-7000-8000-000000000002")
	postmark = uuid.MustParse("01990000-0000-7000-8000-000000000003")
	beta     = "01990000-0000-7000-8000-0000000000b1"
	digest   = "01990000-0000-7000-8000-0000000000d1"
)

// The sample routing of the design: Broadcasts SES 90 / Postal 10 with two
// conditions, Automations SES → Postmark, System Postmark → SES.
func sample() Set {
	return Set{Rules: []Rule{
		{Stream: providers.StreamBroadcast, Priority: 10, Name: "Outlook and iCloud are strict with new IPs", Enabled: true,
			Conditions: Conditions{MailboxProviders: []string{Outlook, ICloud}}, Splits: []Split{{ses, 100}}},
		{Stream: providers.StreamBroadcast, Priority: 20, Name: "small, high-value list", Enabled: true,
			Conditions: Conditions{Lists: []string{beta}}, Splits: []Split{{ses, 100}}},
		{Stream: providers.StreamBroadcast, Priority: 100, Enabled: true, Splits: []Split{{ses, 90}, {postal, 10}}},
		{Stream: providers.StreamAutomation, Priority: 100, Enabled: true, Splits: []Split{{ses, 100}}, Fallback: &postmark},
		{Stream: providers.StreamSystem, Priority: 100, Enabled: true, Splits: []Split{{postmark, 100}}, Fallback: &ses},
	}}
}

func TestResolveConditions(t *testing.T) {
	s := sample()
	d, ok := s.Resolve(Recipient{Stream: providers.StreamBroadcast, Email: "sam.rivera@outlook.com", Lists: []string{digest}})
	if !ok || d.ProviderID != ses || d.Matched != "mailbox_provider" || d.MatchedValue != Outlook || d.Split {
		t.Fatalf("outlook recipient: %+v", d)
	}
	d, _ = s.Resolve(Recipient{Stream: providers.StreamBroadcast, Email: "x@gmail.com", Lists: []string{beta}})
	if d.ProviderID != ses || d.Matched != "list" || d.MatchedValue != beta {
		t.Fatalf("beta list: %+v", d)
	}
	if d.Policy != PauseAndAsk {
		t.Fatalf("broadcast policy: %s", d.Policy)
	}
	d, _ = s.Resolve(Recipient{Stream: providers.StreamSystem, Email: "x@gmail.com"})
	if d.ProviderID != postmark || d.Fallback == nil || *d.Fallback != ses || d.Policy != Automatic {
		t.Fatalf("system: %+v", d)
	}
	if _, ok := (Set{}).Resolve(Recipient{Stream: providers.StreamSystem, Email: "x@y.z"}); ok {
		t.Fatal("no rules: no decision")
	}
}

func TestWeightedSplitIsDeterministicAndExact(t *testing.T) {
	s := sample()
	counts := map[uuid.UUID]int{}
	for i := 0; i < 20000; i++ {
		email := fmt.Sprintf("reader%d@gmail.com", i)
		d, _ := s.Resolve(Recipient{Stream: providers.StreamBroadcast, Email: email})
		again, _ := s.Resolve(Recipient{Stream: providers.StreamBroadcast, Email: email})
		if d.ProviderID != again.ProviderID || !d.Split {
			t.Fatal("the split must be deterministic")
		}
		if (d.Bucket >= 90) != (d.ProviderID == postal) {
			t.Fatalf("bucket %d went to %s", d.Bucket, d.ProviderID)
		}
		counts[d.ProviderID]++
	}
	share := float64(counts[postal]) / 20000
	if share < 0.09 || share > 0.11 {
		t.Fatalf("Postal share %.3f, want about 0.10", share)
	}
}

// PR-2: the split is order-independent. Reordering a 40/60 split (what a
// save that sorted splits by weight did) moves nobody; before rendezvous
// hashing it moved about 80% of addresses.
func TestReorderingASplitMovesNobody(t *testing.T) {
	a := []Split{{ses, 40}, {postal, 60}}
	b := []Split{{postal, 60}, {ses, 40}}
	three := []Split{{ses, 50}, {postal, 30}, {postmark, 20}}
	threeAgain := []Split{{postmark, 20}, {ses, 50}, {postal, 30}}
	for i := 0; i < 20000; i++ {
		email := fmt.Sprintf("reader%d@example.com", i)
		x, _, _ := pick(a, email)
		y, _, _ := pick(b, email)
		if x != y {
			t.Fatalf("%s moved from %s to %s when the split was reordered", email, x, y)
		}
		x, _, _ = pick(three, email)
		y, _, _ = pick(threeAgain, email)
		if x != y {
			t.Fatalf("%s moved in a reordered three-way split", email)
		}
	}
}

// Raising a share moves about the difference, only towards the provider
// that grows: 90/10 → 85/15 moves about 5% of addresses from SES to Postal
// and nobody from Postal to SES.
func TestRaisingAShareMovesOnlyTheDifference(t *testing.T) {
	before := []Split{{ses, 90}, {postal, 10}}
	after := []Split{{ses, 85}, {postal, 15}}
	moved := 0
	const n = 40000
	for i := 0; i < n; i++ {
		email := fmt.Sprintf("member%d@acme.io", i)
		x, _, _ := pick(before, email)
		y, _, _ := pick(after, email)
		if x == y {
			continue
		}
		if x != ses || y != postal {
			t.Fatalf("%s moved from %s to %s", email, x, y)
		}
		moved++
	}
	if share := float64(moved) / n; share < 0.04 || share > 0.06 {
		t.Fatalf("%.3f of addresses moved, want about 0.05", share)
	}
}

// Every provider gets its weight's share, whatever the split and the ids.
func TestRendezvousSharesFollowWeights(t *testing.T) {
	ids := []uuid.UUID{ses, postal, postmark, uuid.MustParse("01990000-0000-7000-8000-0000000000aa"), uuid.MustParse("0199ffff-0000-7000-8000-000000000001")}
	for _, weights := range [][]int{{50, 50}, {70, 30}, {97, 3}, {34, 33, 33}, {60, 30, 10}, {40, 20, 20, 10, 10}} {
		split := make([]Split, len(weights))
		for i, w := range weights {
			split[i] = Split{ProviderID: ids[i], Weight: w}
		}
		counts := map[uuid.UUID]int{}
		const n = 30000
		for i := 0; i < n; i++ {
			id, _, _ := pick(split, fmt.Sprintf("c%d@example.org", i))
			counts[id]++
		}
		for _, sp := range split {
			got := float64(counts[sp.ProviderID]) / n
			if want := float64(sp.Weight) / 100; got < want-0.015 || got > want+0.015 {
				t.Errorf("split %v: provider %d got %.3f, want %.2f", weights, sp.Weight, got, want)
			}
		}
	}
}

// The bucket of a two-provider split is exact: the first provider owns the
// buckets below its weight, in whichever order the split is shown.
func TestPairBucketMatchesTheWinner(t *testing.T) {
	for _, split := range [][]Split{{{ses, 85}, {postal, 15}}, {{postal, 15}, {ses, 85}}, {{ses, 50}, {postal, 50}}} {
		for i := 0; i < 5000; i++ {
			email := fmt.Sprintf("p%d@gmail.com", i)
			id, bucket, ok := pick(split, email)
			if !ok || bucket < 0 || bucket > 99 {
				t.Fatalf("bucket %d", bucket)
			}
			if (bucket < split[0].Weight) != (id == split[0].ProviderID) {
				t.Fatalf("%s: bucket %d of %v went to %s", email, bucket, split, id)
			}
		}
	}
	if _, bucket, _ := pick([]Split{{ses, 50}, {postal, 30}, {postmark, 20}}, "a@b.co"); bucket != -1 {
		t.Fatalf("three-way split bucket %d, want -1", bucket)
	}
}

func TestMailboxProvider(t *testing.T) {
	for in, want := range map[string]string{
		"a@gmail.com": Gmail, "a@googlemail.com": Gmail, "a@hotmail.co.uk": Outlook, "a@live.com": Outlook,
		"a@yahoo.co.jp": Yahoo, "a@aol.com": Yahoo, "a@me.com": ICloud, "a@acme.io": Other,
	} {
		if got := MailboxProvider(in); got != want {
			t.Errorf("%s: %s, want %s", in, got, want)
		}
	}
}

func TestFailover(t *testing.T) {
	healthy := func(uuid.UUID) ProviderState { return ProviderState{Available: true} }
	warming := func(uuid.UUID) ProviderState { return ProviderState{Available: true, WarmingUp: true} }
	down := func(uuid.UUID) ProviderState { return ProviderState{} }
	if to, hold := Failover(Automatic, &postmark, healthy); hold || *to != postmark {
		t.Fatal("automatic moves to the fallback")
	}
	if _, hold := Failover(PauseAndAsk, &postmark, healthy); !hold {
		t.Fatal("pause and ask holds")
	}
	if _, hold := Failover(WarmOnly, &postal, warming); !hold {
		t.Fatal("warm-only holds while the fallback warms up")
	}
	if to, hold := Failover(WarmOnly, &postal, healthy); hold || *to != postal {
		t.Fatal("warm-only moves to a warmed-up fallback")
	}
	if _, hold := Failover(Automatic, &postmark, down); !hold {
		t.Fatal("an unavailable fallback holds")
	}
	if _, hold := Failover(Automatic, nil, healthy); !hold {
		t.Fatal("no fallback holds")
	}
}

// PR-10: a provider a stream routes to that can't take mail raises an
// issue (a warning, not a block), once per stream and role.
func TestUnavailableProviderRaisesAnIssue(t *testing.T) {
	facts := map[uuid.UUID]Facts{
		ses:      {ID: ses, Name: "Amazon SES", Status: "down", BulkAllowed: true},
		postal:   {ID: postal, Name: "Postal", Status: "healthy", BulkAllowed: true, Available: true},
		postmark: {ID: postmark, Name: "Postmark", Status: "degraded", BulkAllowed: true},
	}
	issues := Validate(i18n.Default, sample(), facts)
	var got []string
	for _, is := range issues {
		if is.Code != IssueUnavailable {
			continue
		}
		if is.Blocking {
			t.Fatalf("an unavailable provider blocks a save: %+v", is)
		}
		got = append(got, is.Stream+"/"+is.Field+"/"+is.ProviderID[len(is.ProviderID)-1:]+": "+is.Message)
	}
	want := []string{
		"broadcast/splits/1: Amazon SES can't take mail right now: it's down. Broadcasts routed to it fail over or wait, as the failover policy says.",
		"automation/splits/1: Amazon SES can't take mail right now: it's down. Automations routed to it fail over or wait, as the failover policy says.",
		"automation/fallback/3: Postmark can't take mail right now: its webhooks are silent. Automations have no working fallback until it's back.",
		"system/splits/3: Postmark can't take mail right now: its webhooks are silent. System mail routed to it fails over or waits, as the failover policy says.",
		"system/fallback/1: Amazon SES can't take mail right now: it's down. System has no working fallback until it's back.",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("issues:\n%s", strings.Join(got, "\n"))
	}
}

// PR-12: a condition can split by weight. Its recipients split like the
// stream's (sticky, order-independent), and the split is checked the same
// way: weights add up to 100 and a provider appears once.
func TestConditionSplitsByWeight(t *testing.T) {
	s := sample()
	s.Rules = append([]Rule{{Stream: providers.StreamBroadcast, Priority: 5, Name: "product announcements", Enabled: true,
		Conditions: Conditions{SenderDomains: []string{"updates.acme.io"}}, Splits: []Split{{ses, 70}, {postal, 30}}}}, s.Rules...)
	counts := map[uuid.UUID]int{}
	for i := 0; i < 10000; i++ {
		email := fmt.Sprintf("reader%d@gmail.com", i)
		d, _ := s.Resolve(Recipient{Stream: providers.StreamBroadcast, Email: email, SenderDomain: "Updates.Acme.io"})
		if d.Matched != "sender_domain" || !d.Split {
			t.Fatalf("not the condition's split: %+v", d)
		}
		swapped := s
		swapped.Rules = append([]Rule(nil), s.Rules...)
		swapped.Rules[0].Splits = []Split{{postal, 30}, {ses, 70}}
		again, _ := swapped.Resolve(Recipient{Stream: providers.StreamBroadcast, Email: email, SenderDomain: "updates.acme.io"})
		if again.ProviderID != d.ProviderID {
			t.Fatal("reordering a condition's split moved an address")
		}
		counts[d.ProviderID]++
	}
	if share := float64(counts[postal]) / 10000; share < 0.28 || share > 0.32 {
		t.Fatalf("Postal's share of the condition %.3f, want about 0.30", share)
	}
	facts := map[uuid.UUID]Facts{ses: {ID: ses, Name: "Amazon SES", BulkAllowed: true, Available: true}, postal: {ID: postal, Name: "Postal", BulkAllowed: true, Available: true},
		postmark: {ID: postmark, Name: "Postmark", BulkAllowed: false, Available: true}}
	for _, i := range Validate(i18n.Default, s, facts) {
		if i.Blocking {
			t.Fatalf("a valid condition split is blocked: %+v", i)
		}
	}
	bad := s
	bad.Rules = append([]Rule(nil), s.Rules...)
	bad.Rules[0].Splits = []Split{{ses, 70}, {postal, 20}}
	blocked := false
	for _, i := range Validate(i18n.Default, bad, facts) {
		if i.Blocking && i.Code == IssueWeights && i.Field == "conditions[0]" && strings.Contains(i.Message, "90%") {
			blocked = true
		}
	}
	if !blocked {
		t.Fatal("a condition split adding up to 90% isn't blocked")
	}
	bad.Rules[0].Splits = []Split{{ses, 50}, {ses, 50}}
	dup := false
	for _, i := range Validate(i18n.Default, bad, facts) {
		dup = dup || (i.Blocking && i.Code == IssueDuplicate && i.Field == "conditions[0]")
	}
	if !dup {
		t.Fatal("a provider twice in a condition split isn't blocked")
	}
}
