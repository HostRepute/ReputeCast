package readiness

import (
	"encoding/json"
	"testing"
)

// The thresholds are stored the way the sending engine reads them: the
// provider errors threshold as api_errors, keys this screen doesn't edit
// (deferral) kept, older spellings still read.
func TestThresholdsStoredForTheEngine(t *testing.T) {
	stored := json.RawMessage(`{"bounce":2,"complaint":0.1,"mailbox_complaint":0.1,"provider_errors":5,"deferral":4,"scope":"route"}`)
	if th := ParseThresholds(stored, nil); th.ProviderErrors != 5 || th.Bounce != 2 {
		t.Fatalf("older spelling: %+v", th)
	}
	next := ReadinessThresholds{Bounce: 1.5, Complaint: 0.08, MailboxComplaint: 0.1, ProviderErrors: 7, Scope: "campaign"}.toDB(stored)
	var m map[string]any
	if err := json.Unmarshal(next, &m); err != nil {
		t.Fatal(err)
	}
	if m["api_errors"] != 7.0 || m["deferral"] != 4.0 || m["bounce"] != 1.5 || m["scope"] != "campaign" {
		t.Fatalf("stored: %s", next)
	}
	if _, ok := m["provider_errors"]; ok {
		t.Fatalf("the older key goes: %s", next)
	}
	if th := ParseThresholds(next, nil); th.ProviderErrors != 7 || th.Complaint != 0.08 {
		t.Fatalf("read back: %+v", th)
	}
}
