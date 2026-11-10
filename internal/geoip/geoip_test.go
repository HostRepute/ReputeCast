package geoip

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRefusesWhatIsntADatabase(t *testing.T) {
	dir := t.TempDir()
	if _, err := Open(filepath.Join(dir, "missing.mmdb")); err == nil {
		t.Error("Open of a missing file: no error")
	}
	bad := filepath.Join(dir, "bad.mmdb")
	if err := os.WriteFile(bad, []byte("not a MaxMind DB file, just text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(bad); err == nil {
		t.Error("Open of a non-mmdb file: no error")
	}
	if _, err := Open(""); err == nil {
		t.Error("Open without a path: no error")
	}
}

func TestOffFindsNothing(t *testing.T) {
	if d := Load("", nil); d != nil {
		t.Fatalf("Load(\"\") = %v, want nil (off)", d)
	}
	var off *DB
	if off.Loaded() {
		t.Error("nil DB is loaded")
	}
	if c, city := off.Lookup(netip.MustParseAddr("81.2.69.142")); c != "" || city != "" {
		t.Errorf("nil DB Lookup = %q, %q", c, city)
	}
	if off.Describe() != "" {
		t.Error("nil DB describes itself")
	}
	loading := &DB{}
	if loading.Loaded() || loading.Describe() != "" {
		t.Error("a DB still loading is on")
	}
	if c, _ := loading.Lookup(netip.MustParseAddr("81.2.69.142")); c != "" {
		t.Errorf("a DB still loading found %q", c)
	}
}
