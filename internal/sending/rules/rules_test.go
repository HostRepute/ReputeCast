package rules

import "testing"

func TestReplyRuleMatches(t *testing.T) {
	r := ReplyRule{Contains: "TSS04"}
	for text, want := range map[string]bool{
		"421 4.7.0 [TSS04] Messages from 203.0.113.7 temporarily deferred": true,
		"421 4.7.0 tss04 retry in 5 min":                                   true,
		"421 4.7.0 TS04":                                                   false,
		"":                                                                 false,
	} {
		if got := r.Matches(text); got != want {
			t.Errorf("Matches(%q) = %v, want %v", text, got, want)
		}
	}
	if (ReplyRule{}).Matches("anything") {
		t.Error("a rule without a text matched")
	}
}

func TestKnownReplyKey(t *testing.T) {
	for contains, want := range map[string]string{"TSS04": "tss04", "TS04": "tss04", " s3150 ": "s3150", "S3140": "s3150", "4.7.650": "outlook_throttle", "blocked": ""} {
		if got := KnownReplyKey(contains); got != want {
			t.Errorf("KnownReplyKey(%q) = %q, want %q", contains, got, want)
		}
	}
}

func TestParseSettingsKeepsValuesInRange(t *testing.T) {
	c := ParseCanary([]byte(`{"enabled":true,"percent":50,"watchMinutes":45}`))
	if c.Percent != DefaultCanary.Percent || c.WatchMinutes != DefaultCanary.WatchMinutes {
		t.Fatalf("canary %+v: out-of-range values should fall back to the defaults", c)
	}
	if c := ParseCanary(nil); c != DefaultCanary {
		t.Fatalf("canary %+v, want the defaults", c)
	}
	w := ParseWatch([]byte(`{"enabled":false,"hours":12}`))
	if w.Enabled || w.Hours != DefaultWatch.Hours {
		t.Fatalf("watch %+v: off, with the default hours", w)
	}
	if w := (Watch{Hours: 168, Enabled: true}).Normalized(); w.Hours != 168 {
		t.Fatalf("watch %+v, want 7 days kept", w)
	}
}
