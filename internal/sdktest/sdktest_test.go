package sdktest_test

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

var sdkDir = filepath.Join("..", "..", "sdk")

// need skips the test when a toolchain or an install is missing, unless
// REPUTECAST_SDK_E2E=required (make sdks, CI), where that is a failure.
func need(t *testing.T, format string, a ...any) {
	t.Helper()
	if os.Getenv("REPUTECAST_SDK_E2E") == "required" {
		t.Fatalf(format, a...)
	}
	t.Skipf(format, a...)
}

// run runs a command of an SDK's end-to-end check and fails the test with
// its output when it fails.
func run(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // the test's own toolchains and scripts
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	t.Logf("%s", strings.TrimSpace(string(out)))
}

// The three client SDKs against the real API: list the lists, upsert a
// contact, page through contacts with the cursor, send a transactional
// message with an Idempotency-Key, and read a 403 as the SDK's error.
func TestClientSDKs(t *testing.T) {
	if os.Getenv("REPUTECAST_SDK_E2E") == "" {
		t.Skip("set REPUTECAST_SDK_E2E=1 (or required) to run the client SDKs against the app")
	}
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	// A key that may read, add and update contacts and send transactional
	// mail, but not see the team: listing members answers 403.
	var key struct{ Secret string }
	c.Do(http.MethodPost, "/workspaces/acme-newsletter/api-keys", map[string]any{
		"name":        "Client SDKs end to end",
		"permissions": []string{"subscribers:view", "subscribers:create", "subscribers:edit", "transactional:send"},
	}).MustStatus(t, http.StatusCreated).JSON(t, &key)
	environ := append(os.Environ(), "REPUTECAST_URL="+env.Server.URL, "REPUTECAST_API_KEY="+key.Secret, "REPUTECAST_WORKSPACE=acme-newsletter")

	t.Run("typescript", func(t *testing.T) {
		dir := filepath.Join(sdkDir, "client-ts")
		bin := os.Getenv("REPUTECAST_TEST_NODE")
		if bin == "" {
			var err error
			if bin, err = exec.LookPath("node"); err != nil {
				need(t, "Node isn't installed: the TypeScript SDK needs Node 20 or later")
			}
		}
		out, err := exec.CommandContext(t.Context(), bin, "--version").Output() //nolint:gosec // the test's own Node binary
		m := regexp.MustCompile(`^v(\d+)\.`).FindStringSubmatch(strings.TrimSpace(string(out)))
		if err != nil || m == nil {
			need(t, "node --version: %v %q", err, out)
		}
		if major, _ := strconv.Atoi(m[1]); major < 20 {
			need(t, "Node %s is too old for the TypeScript SDK; it needs 20 (set REPUTECAST_TEST_NODE)", strings.TrimSpace(string(out)))
		}
		if _, err := os.Stat(filepath.Join(dir, "dist", "index.js")); err != nil {
			need(t, "the TypeScript SDK isn't built: run `pnpm install && pnpm --filter @reputecast/client build` in sdk/")
		}
		run(t, dir, environ, bin, filepath.Join("test", "e2e.mjs"))
	})

	t.Run("python", func(t *testing.T) {
		dir := filepath.Join(sdkDir, "client-python")
		// The virtual environment `make sdks` makes, else
		// REPUTECAST_TEST_PYTHON or python3 with the package installed.
		venv := filepath.Join(dir, ".venv", "bin", "python")
		if runtime.GOOS == "windows" {
			venv = filepath.Join(dir, ".venv", "Scripts", "python.exe")
		}
		bin, err := filepath.Abs(venv)
		if _, statErr := os.Stat(venv); err != nil || statErr != nil {
			name := os.Getenv("REPUTECAST_TEST_PYTHON")
			if name == "" {
				name = "python3"
			}
			if bin, err = exec.LookPath(name); err != nil {
				need(t, "Python isn't installed: the Python SDK needs 3.11 or later")
			}
		}
		if out, err := exec.CommandContext(t.Context(), bin, "-c", "import reputecast").CombinedOutput(); err != nil { //nolint:gosec // the test's own Python
			need(t, "the Python SDK isn't installed in %s: run `make sdks` (or `pip install -e sdk/client-python`): %s", bin, out)
		}
		run(t, dir, environ, bin, filepath.Join("tests", "e2e.py"))
	})

	t.Run("go", func(t *testing.T) {
		bin, err := exec.LookPath("go")
		if err != nil {
			need(t, "Go isn't on PATH")
		}
		run(t, filepath.Join(sdkDir, "client-go"), environ, bin, "test", "-count=1", "-v", "./e2e")
	})
}
