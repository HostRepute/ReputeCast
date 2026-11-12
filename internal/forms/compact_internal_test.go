package forms

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The served script keeps every statement of the source: comments and
// indentation are all it drops, and it still parses.
func TestCompactScript(t *testing.T) {
	if strings.Contains(embedScript, "/*") || strings.Contains(embedScript, "\n//") {
		t.Fatal("comments left in the served script")
	}
	if !strings.Contains(embedScript, "__RC_FORM__") || len(embedScript) > len(embedScriptSource)*3/4 {
		t.Fatalf("served script: %d bytes of %d", len(embedScript), len(embedScriptSource))
	}
	if got := compactScript("a()\n  /* x\n y */ b()\n  // c\n\td()\n"); got != "a()\nb()\nd()\n" {
		t.Fatalf("compactScript = %q", got)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node isn't installed")
	}
	f := filepath.Join(t.TempDir(), "form.js")
	if err := os.WriteFile(f, []byte(strings.Replace(embedScript, "__RC_FORM__", "null", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(node, "--check", f).CombinedOutput(); err != nil { //nolint:gosec // G204: the test's own file
		t.Fatalf("node --check: %v\n%s", err, out)
	}
}
