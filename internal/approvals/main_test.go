package approvals_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/testsplit"
)

// CI runs this package's tests in several shards (scripts/testshard).
func TestMain(m *testing.M) {
	testsplit.Main(m, "github.com/hostrepute/reputecast/internal/approvals")
}
