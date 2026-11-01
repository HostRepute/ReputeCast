// Package cadence decides which readers a campaign reaches by the rhythm
// they chose on the preference center (review 2026-11 CA-6, decision D1).
//
// Readers pick "Every issue" (all, also everyone who never chose),
// "Weekly roundup" (weekly) or "Monthly best-of" (monthly). A regular
// campaign reaches "Every issue" readers only; a weekly or monthly digest
// issue reaches its rhythm's readers as well; an announcement reaches
// everyone on its lists whatever they chose.
package cadence

// Announcement is the campaign cadence that reaches everyone.
const Announcement = "announcement"

// Frequencies are the preference-center choices a campaign reaches: its
// audience cadence ("" or "regular", "announcement") and, for a digest
// issue, the digest's frequency ("weekly", "monthly", "daily", "items").
// nil reaches everyone.
func Frequencies(audienceCadence, digestFrequency string) []string {
	if audienceCadence == Announcement {
		return nil
	}
	switch digestFrequency {
	case "weekly", "monthly":
		return []string{"all", digestFrequency}
	}
	return []string{"all"}
}
