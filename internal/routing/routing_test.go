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
