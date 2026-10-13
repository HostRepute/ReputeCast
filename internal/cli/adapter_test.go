package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(`{"protocol":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, source, err := readManifest(context.Background(), path)
	if err != nil || string(raw) != `{"protocol":1}` || source != "" {
		t.Fatalf("file: %q %q %v", raw, source, err)
	}
	big := filepath.Join(dir, "big.json")
	if err := os.WriteFile(big, []byte(strings.Repeat(" ", maxManifest+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readManifest(context.Background(), big); err == nil || !strings.Contains(err.Error(), "256 KiB") {
		t.Fatalf("big: %v", err)
	}
	for _, u := range []string{"http://example.com/manifest.json", "ftp://example.com/m.json"} {
		if _, _, err := readManifest(context.Background(), u); err == nil || !strings.Contains(err.Error(), "https://") {
			t.Fatalf("%s: %v", u, err)
		}
	}
	// Manifests are only downloaded from public addresses.
	if _, _, err := readManifest(context.Background(), "https://127.0.0.1:1/manifest.json"); err == nil || !strings.Contains(err.Error(), "Blocked") {
		t.Fatalf("loopback: %v", err)
	}
}
