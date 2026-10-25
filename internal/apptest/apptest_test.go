package apptest

import (
	"testing"
	"time"
)

// A sample seeded before midnight in one of its workspaces' time zones is
// seeded again after it; within a day it is reused.
func TestSampleDay(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// Midnight in New York (EDT) is 04:00 UTC.
	if sampleDay(at("2026-10-25T03:58:00Z"), acmeZones) == sampleDay(at("2026-10-25T04:02:00Z"), acmeZones) {
		t.Error("the Acme sample spans midnight in New York")
	}
	if sampleDay(at("2026-10-25T04:02:00Z"), acmeZones) != sampleDay(at("2026-10-26T03:58:00Z"), acmeZones) {
		t.Error("the Acme sample changes within a New York day")
	}
	// Northwind's days begin in New York, then Chicago (05:00 UTC), then
	// Denver and Los Angeles.
	if sampleDay(at("2026-10-25T04:30:00Z"), northwindZones) == sampleDay(at("2026-10-25T05:30:00Z"), northwindZones) {
		t.Error("the Northwind sample spans midnight in Chicago")
	}
	if sampleDay(at("2026-10-25T07:30:00Z"), northwindZones) != sampleDay(at("2026-10-26T03:30:00Z"), northwindZones) {
		t.Error("the Northwind sample changes when no workspace's day begins")
	}
}
