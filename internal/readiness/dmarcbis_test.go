package readiness_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// A DMARC record in test mode (t=y, RFC 9989) passes, and its technical
// details say receivers apply the policy one level lower.
func TestDMARCTestMode(t *testing.T) {
	f := digest()
	if _, ok := techRows(mustCheck(t, readiness.Assemble(f), "auth.dmarc"))["t=y"]; ok {
		t.Fatal("a test-mode row without t=y")
	}
	for policy, applied := range map[string]string{"quarantine": "none", "reject": "quarantine", "none": "none"} {
		f.Auth.DMARCPolicy, f.Auth.DMARCTestMode = policy, true
		c := mustCheck(t, readiness.Assemble(f), "auth.dmarc")
		row := techRows(c)["t=y"]
		if c.Status != readiness.Pass || !strings.Contains(row, policy) || !strings.Contains(row, applied) {
			t.Errorf("p=%s: %s %q", policy, c.Status, row)
		}
	}
}
