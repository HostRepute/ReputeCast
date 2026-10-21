package approvals

import (
	"slices"
	"testing"
	"time"
)

// Client sign-off covers campaigns on every channel: an email, a text or a
// push campaign under the rule waits for one approval from a Client
// approver, even when no other rule matches.
func TestClientSignoffCoversEveryChannel(t *testing.T) {
	p := ParsePolicy(nil)
	p.AudienceMin.Enabled, p.WarmingProvider, p.AuthorLacksSend = false, false, false
	p.ClientSignoff = ApprovalClientSignoff{Enabled: true, Automations: true}
	back := ParsePolicy(p.toDB(nil))
	if !back.ClientSignoff.Enabled || !back.ClientSignoff.Automations {
		t.Fatalf("round trip: %+v", back.ClientSignoff)
	}
	for _, channel := range []string{"", "email", "sms", "push", "whatsapp"} {
		rules := back.Evaluate(RuleFacts{Recipients: 12, AuthorCanSend: true, Channel: channel, Now: time.Now()})
		keys := ruleKeys(rules)
		if !slices.Equal(keys, []string{RuleClientSignoff}) {
			t.Fatalf("%q: rules %v", channel, keys)
		}
		if requiredFor(back, rules) != 1 {
			t.Fatalf("%q: client sign-off alone needs one approval", channel)
		}
	}
	// Turned off, nothing matches for the same send.
	back.ClientSignoff.Enabled = false
	if rules := back.Evaluate(RuleFacts{Recipients: 12, AuthorCanSend: true, Channel: "sms", Now: time.Now()}); len(rules) != 0 {
		t.Fatalf("off: %v", ruleKeys(rules))
	}
}
