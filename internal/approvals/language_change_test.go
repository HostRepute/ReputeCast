package approvals

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

// A language version that changed after approval is stored in English
// and reads in the reader's language later: its name, the versions, what
// changed, and the invalidation reason that names it.
func TestLanguageChangesReadInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	snap := Snapshot{Languages: []languages.VersionSnapshot{{Language: "de", Status: "translated", Version: 3, Hash: "a"}}}
	cur := Current{Languages: []languages.VersionSnapshot{{Language: "de", Status: "draft", Version: 3, Hash: "a"}}}
	changes := []storedChange{}
	for _, c := range diff(snap, cur, time.UTC) {
		if c.Field == "language" {
			changes = append(changes, c)
		}
	}
	if len(changes) != 1 || changes[0].Label != "German" || changes[0].What != "back to draft" || changes[0].Before != "translated" {
		t.Fatalf("English: %+v", changes)
	}
	raw, _ := json.Marshal(changes)
	local := changesFor(vi, raw)
	if local[0].Label != "Tiếng Đức" || local[0].Before != "đã dịch" || local[0].After != "bản nháp" || local[0].What != "trở lại bản nháp" {
		t.Errorf("Vietnamese change: %+v", local[0])
	}
	if en := changesFor(i18n.Default, raw); en[0].Label != "German" || en[0].What != "back to draft" {
		t.Errorf("English change: %+v", en[0])
	}

	// One change: the reason names the language and what changed.
	reason := reasonOf(changes)
	text, _ := json.Marshal(reason)
	english := render(i18n.Default, reason)
	if english != "German: back to draft" {
		t.Fatalf("English reason: %q", english)
	}
	if got := reasonFor(vi, english, text); got != "Tiếng Đức: trở lại bản nháp" {
		t.Errorf("Vietnamese reason: %q", got)
	}

	// With other changes, the language is one of the fields.
	both := append([]storedChange{{ApprovalChange: ApprovalChange{Field: "subject", Label: "Subject"}}}, changes...)
	text, _ = json.Marshal(reasonOf(both))
	if got := fieldsFor(vi, "", text); got != "tiêu đề và phiên bản tiếng Đức" {
		t.Errorf("Vietnamese fields: %q", got)
	}
	if got := fieldsFor(i18n.Default, "", text); got != "subject and German version" {
		t.Errorf("English fields: %q", got)
	}
}
