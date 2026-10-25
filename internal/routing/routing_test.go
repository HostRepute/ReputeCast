package routing

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

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
