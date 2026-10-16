package transactional_test

import (
	"os"
	"testing"

	"github.com/hostrepute/reputecast/internal/demo"
)

// These tests start from an empty Transactional tab: the demo's Acme Cloud
// mail stays out of their seed (internal/demo checks it against the
// boards).
func TestMain(m *testing.M) {
	demo.SkipTransactional = true
	os.Exit(m.Run())
}
