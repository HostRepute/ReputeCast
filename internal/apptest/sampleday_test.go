package apptest

import (
	"testing"
	"time"
)

// The Acme sample is seeded again after midnight UTC, not only after
// midnight in New York: its DMARC reports cover whole UTC days, and a
// template seeded at 23:50 UTC read one day of seven short at 00:10.
func TestSampleDayTurnsAtUTCMidnight(t *testing.T) {
	before := time.Date(2026, 11, 1, 23, 50, 0, 0, time.UTC) // 19:50 in New York
	after := time.Date(2026, 11, 2, 0, 10, 0, 0, time.UTC)   // 20:10 in New York, the same day there
	if sampleDay(before, acmeZones) == sampleDay(after, acmeZones) {
		t.Fatal("the Acme sample keeps its template across midnight UTC")
	}
	if sampleDay(before, northwindZones) == sampleDay(after, northwindZones) {
		t.Fatal("the Northwind sample keeps its template across midnight UTC")
	}
	same := time.Date(2026, 11, 2, 0, 40, 0, 0, time.UTC)
	if sampleDay(after, acmeZones) != sampleDay(same, acmeZones) {
		t.Fatal("two moments with no midnight between them seed twice")
	}
}
