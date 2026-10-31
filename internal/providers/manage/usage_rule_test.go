package manage

import "testing"

// R1.L5c: one rule for judging a provider's 7-day rates, for Send
// Readiness and Deliverability's provider card: ReputeCast's own rates need
// MinRateVolume messages; a provider's own rates are judged as they come.
func TestTooFewToJudge(t *testing.T) {
	for _, c := range []struct {
		reported bool
		sent     int64
		want     bool
	}{
		{false, 3, true}, {false, MinRateVolume - 1, true}, {false, MinRateVolume, false}, {false, 0, false}, {true, 3, false},
	} {
		if got := (Usage{ProviderReported: c.reported, Sent7d: c.sent}).TooFewToJudge(); got != c.want {
			t.Errorf("reported %v, %d sent: %v", c.reported, c.sent, got)
		}
	}
}
