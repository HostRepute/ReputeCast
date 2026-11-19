package mcpclient_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hostrepute/reputecast/internal/mcpclient/mcptest"
)

// The tests' https servers use httptest's certificate: it is the only root
// the package's TLS clients trust (no test reaches a real host).
func TestMain(m *testing.M) {
	done, err := mcptest.TrustTLS()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	done()
	os.Exit(code)
}
