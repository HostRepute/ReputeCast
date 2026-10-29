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

// Only an address mail can leave from on the internet is a sending IP.
func TestPublicIP(t *testing.T) {
	for s, want := range map[string]bool{
		"203.0.113.25": true, "198.51.100.7": true, "2001:db8::25": true, "54.240.8.1": true,
		"10.0.0.5": false, "172.16.4.1": false, "192.168.1.20": false, "100.64.0.1": false, "127.0.0.1": false,
		"0.0.0.0": false, "169.254.0.1": false, "224.0.0.1": false, "::1": false, "fd00::25": false, "fe80::1": false,
	} {
		a, ok := ParseIP(s)
		if !ok {
			t.Fatalf("%s doesn't parse", s)
		}
		if got := PublicIP(a); got != want {
			t.Errorf("PublicIP(%s) = %v, want %v", s, got, want)
		}
	}
}
