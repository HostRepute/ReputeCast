package providers

import (
	"slices"
	"testing"
)

// providers.ips keeps each sending IP once, written the canonical way and in
// address order, whatever order and spelling a provider or a person used.
func TestCleanIPs(t *testing.T) {
	clean, bad := CleanIPs([]string{" 203.0.113.25 ", "2001:DB8::25", "198.51.100.10", "198.51.100.9", "::ffff:203.0.113.25",
		"203.0.113.25", "mta.acme.io", "203.0.113.0/24", "fe80::1%eth0", ""})
	if want := []string{"198.51.100.9", "198.51.100.10", "203.0.113.25", "2001:db8::25"}; !slices.Equal(clean, want) {
		t.Fatalf("clean %q, want %q", clean, want)
	}
	if want := []string{"mta.acme.io", "203.0.113.0/24", "fe80::1%eth0", ""}; !slices.Equal(bad, want) {
		t.Fatalf("bad %q, want %q", bad, want)
	}
	if clean, bad := CleanIPs(nil); clean == nil || len(clean) != 0 || len(bad) != 0 {
		t.Fatalf("no IPs: %q, %q; want an empty list, never nil", clean, bad)
	}
}
