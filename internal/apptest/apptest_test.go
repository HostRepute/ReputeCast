package apptest

import (
	"testing"
	"time"
)

// A sample seeded before midnight in one of its workspaces' time zones, or
// in UTC, is seeded again after it; within a day it is reused. UTC is in
// both lists because DMARC reports cover whole UTC days: a template seeded
// at 23:50 UTC read one day of seven short at 00:10.
func TestSampleDay(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	same := func(name string, zones []*time.Location, a, b string) {
		t.Helper()
		if sampleDay(at(a), zones) != sampleDay(at(b), zones) {
			t.Errorf("the %s sample changes between %s and %s, with no midnight between them", name, a, b)
		}
	}
	differ := func(name string, zones []*time.Location, a, b, midnight string) {
		t.Helper()
		if sampleDay(at(a), zones) == sampleDay(at(b), zones) {
			t.Errorf("the %s sample spans midnight in %s", name, midnight)
		}
	}

	// Midnight in New York (EDT) is 04:00 UTC.
	differ("Acme", acmeZones, "2026-10-25T03:58:00Z", "2026-10-25T04:02:00Z", "New York")
	differ("Acme", acmeZones, "2026-10-25T23:50:00Z", "2026-10-26T00:10:00Z", "UTC")
	same("Acme", acmeZones, "2026-10-25T04:02:00Z", "2026-10-25T23:58:00Z")
	same("Acme", acmeZones, "2026-10-26T00:02:00Z", "2026-10-26T03:58:00Z")

	// Northwind's days begin in UTC, New York, then Chicago (05:00 UTC),
	// then Denver and Los Angeles.
	differ("Northwind", northwindZones, "2026-10-25T04:30:00Z", "2026-10-25T05:30:00Z", "Chicago")
	differ("Northwind", northwindZones, "2026-10-25T23:50:00Z", "2026-10-26T00:10:00Z", "UTC")
	same("Northwind", northwindZones, "2026-10-25T07:30:00Z", "2026-10-25T23:30:00Z")
	same("Northwind", northwindZones, "2026-10-26T00:30:00Z", "2026-10-26T03:30:00Z")
}
