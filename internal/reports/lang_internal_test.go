package reports

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Bounce types, unsubscribe reasons, the A/B test's label and the
// suppression note read in the reader's language; English stays as it was.
func TestReportWordsInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	rows := []store.ReportBounceCodesRow{{BounceClass: "mailbox_unknown", N: 3, Recipients: 3, ProviderName: "Amazon SES", Adapter: "ses", OriginalCode: "550 5.1.1"}}
	if tbl := bounceTable(vi, rows); len(tbl.Rows) != 1 || tbl.Rows[0].Label != "Vĩnh viễn · người nhận không tồn tại" {
		t.Errorf("bounce type: %+v", tbl.Rows)
	}
	if tbl := bounceTable(i18n.Default, rows); len(tbl.Rows) != 1 || tbl.Rows[0].Label != "Hard · unknown user" {
		t.Errorf("bounce type in English: %+v", tbl.Rows)
	}
	u := unsubscribes(vi, 2, []store.ReportUnsubscribeReasonsRow{{Reason: "too_frequent", N: 2}})
	if len(u.Reasons) == 0 || u.Reasons[0].Label != "Quá nhiều email" {
		t.Errorf("reasons: %+v", u.Reasons)
	}
	if got := abLabel(vi, "subject"); got != "tiêu đề" {
		t.Errorf("A/B: %q", got)
	}
	args := i18n.Args{"type": "unknown_user", "campaign": "Weekly Digest #142"}
	if got := mBounceNote.Text(i18n.Default, args); got != "Hard · unknown user bounces on Weekly Digest #142" {
		t.Errorf("note in English: %q", got)
	}
	if got := mBounceNote.Keep(args).In(vi); got != "Trả thư vĩnh viễn · người nhận không tồn tại, trong Weekly Digest #142" {
		t.Errorf("note in Vietnamese: %q", got)
	}
	for _, b := range bounceTypes {
		if got, want := mBounceNote.Text(i18n.Default, i18n.Args{"type": b.Key, "campaign": "X"}), bounceLabel(i18n.Default, b.Key)+" bounces on X"; got != want {
			t.Errorf("note for %s: %q, want %q", b.Key, got, want)
		}
	}
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("no Vietnamese for %v", missing)
	}
}
