package engine

import (
	"encoding/json"
	"testing"

	"github.com/hostrepute/reputecast/internal/store"
)

// SR-11 (C2): the unengaged (the sunset tier) are left out of a send
// unless the campaign includes them, as Send Readiness counts them; an
// override of the sunset policy includes them, an excluded tier or an
// explicit "leave them out" doesn't.
func TestUnengagedLeftOutByDefault(t *testing.T) {
	row := store.SendingAudiencePageRow{Email: "g.okafor@example.org", Status: "active", Tier: "sunset"}
	for raw, want := range map[string]string{
		`{"lists":[]}`:                                          "tiers",
		`{"include_unengaged":true}`:                            "",
		`{"includeUnengaged":true}`:                             "",
		`{"include_sunset":true}`:                               "",
		`{"include_sunset":true,"include_unengaged":false}`:     "tiers",
		`{"include_unengaged":true,"exclude_tiers":["sunset"]}`: "tiers",
	} {
		def := parseAudience(json.RawMessage(raw))
		def.leaveOutUnengaged(true)
		if got := eligibility(row, def, false); got != want {
			t.Errorf("%s: %q, want %q", raw, got, want)
		}
	}
	active := row
	active.Tier = "active"
	def := parseAudience(json.RawMessage(`{}`))
	def.leaveOutUnengaged(true)
	if got := eligibility(active, def, false); got != "" {
		t.Errorf("an engaged contact: %q", got)
	}
}
