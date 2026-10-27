package readiness_test

import (
	"encoding/json"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// SR-11 (C2): the sunset policy leaves the unengaged out of every send by
// default. The audience shows them as a line with "Include them", and the
// count and the estimates are the ones after it: 31,604 − 412 = 31,192,
// SES 28,073 · Postal 3,119.
func boardAudience() readiness.Facts {
	f := digest()
	f.Audience.Final, f.Audience.Unengaged, f.Audience.SunsetExcluded, f.Audience.UnengagedExcluded = 31192, 0, 412, true
	f.Audience.Mailboxes = map[string]int64{"gmail": 14348, "outlook": 6551, "yahoo": 2807, "icloud": 2495, "other": 4991}
	f.Route.Shares = []readiness.ShareFacts{ses(28073, 90), postal(3119, 10)}
	return f
}

func TestUnengagedExcludedByDefault(t *testing.T) {
	res := stored(t, readiness.Assemble(boardAudience()))
	un := byKey(res)["audience.unengaged"]
	if un.Status != readiness.Pass || un.Title != "412 unengaged excluded by your sunset policy" || un.Detail != "No human open or click in 180 days or more" {
		t.Fatalf("the check: %+v", un)
	}
	if len(un.Fixes) != 1 || un.Fixes[0].Label != "Sunset policy" || un.Fixes[0].Href != "/w/acme-newsletter/audience/health" {
		t.Errorf("the policy's link: %+v", un.Fixes)
	}
	a := res.Estimates.Audience
	last := a.Rows[len(a.Rows)-1]
	if a.Final != 31192 || a.ExcludedUnengaged != 412 || !a.UnengagedExcluded || last.Label != "Unengaged 180+ days (sunset policy)" || last.Count != 412 ||
		last.Kind != "minus" || last.Action != readiness.FixIncludeUnengaged {
		t.Fatalf("the audience: %+v", a)
	}
	var sum int64
	for _, r := range a.Rows[1:] {
		sum += r.Count
	}
	if a.Rows[0].Count-sum != a.Final {
		t.Errorf("the lines don't add up: %+v", a.Rows)
	}
	// The estimates are the ones after it.
	if sh := res.Estimates.Route.Shares; len(sh) != 2 || sh[0].Recipients != 28073 || sh[1].Recipients != 3119 {
		t.Errorf("the route: %+v", sh)
	}
	if c := res.Estimates.Route.TotalCost; c < 2.805 || c > 2.815 {
		t.Errorf("the cost: %v, want $2.81", c)
	}
	// In Vietnamese too.
	vi := res.In(i18n.For(i18n.Vietnamese))
	if r := vi.Estimates.Audience.Rows[len(vi.Estimates.Audience.Rows)-1]; r.Label != "Không tương tác từ 180 ngày (chính sách ngừng gửi dần)" {
		t.Errorf("the line in Vietnamese: %q", r.Label)
	}
}

// Included ("Include them"): the check warns and offers to exclude them
// again; the audience has no line for them.
func TestUnengagedIncluded(t *testing.T) {
	res := stored(t, readiness.Assemble(digest()))
	un := byKey(res)["audience.unengaged"]
	if un.Status != readiness.Warn || len(un.Fixes) == 0 || un.Fixes[0].Action != readiness.FixExcludeSunset {
		t.Fatalf("included: %+v", un)
	}
	for _, r := range res.Estimates.Audience.Rows {
		if r.Action != "" {
			t.Errorf("a line offers %s", r.Action)
		}
	}
	if res.Estimates.Audience.ExcludedUnengaged != 0 || res.Estimates.Audience.Final != 31604 {
		t.Errorf("included: %+v", res.Estimates.Audience)
	}
}

// Nobody unengaged: nothing to include.
func TestNoUnengaged(t *testing.T) {
	f := boardAudience()
	f.Audience.SunsetExcluded, f.Audience.Final = 0, 31604
	res := stored(t, readiness.Assemble(f))
	if un := byKey(res)["audience.unengaged"]; un.Status != readiness.Pass || un.Title != "No unengaged contacts in this audience" {
		t.Errorf("nobody unengaged: %+v", un)
	}
	for _, r := range res.Estimates.Audience.Rows {
		if r.Action != "" {
			t.Errorf("a line offers %s", r.Action)
		}
	}
}

// The default reads from campaigns.audience: excluded unless the campaign
// includes them or overrides the sunset policy; an excluded tier wins.
func TestAudienceLeavesOutUnengagedByDefault(t *testing.T) {
	for raw, want := range map[string]bool{
		`{"lists":["l1"]}`:                                                     true,
		`{"lists":["l1"],"include_unengaged":true}`:                            false,
		`{"lists":["l1"],"includeUnengaged":true}`:                             false,
		`{"lists":["l1"],"include_sunset":true}`:                               false,
		`{"lists":["l1"],"include_sunset":true,"include_unengaged":false}`:     true,
		`{"lists":["l1"],"include_unengaged":true,"exclude_tiers":["sunset"]}`: true,
	} {
		if got := readiness.ParseAudience(json.RawMessage(raw)).ExcludesSunset(); got != want {
			t.Errorf("%s: excluded %v, want %v", raw, got, want)
		}
	}
}
