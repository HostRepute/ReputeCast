package smtprelay_test

import (
	"os"
	"testing"

	"github.com/hostrepute/reputecast/internal/demo"
)

// These tests set up the relay and its sender themselves: the demo's Acme
// Cloud mail, relay included, stays out of their seed.
func TestMain(m *testing.M) {
	demo.SkipTransactional = true
	os.Exit(m.Run())
}
