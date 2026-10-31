package ingest

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The event log's Source column names what sent the message in the
// reader's language, transactional mail by the way it came in.
func TestSourceLabel(t *testing.T) {
	step := "2"
	vi := i18n.Lang{Locale: "vi"}
	for _, c := range []struct {
		campaign, automation string
		node                 *string
		kind, en, vi         string
	}{
		{"Weekly Digest #142", "", nil, "", "Weekly Digest #142", "Weekly Digest #142"},
		{"", "Welcome series", &step, "", "Welcome series · step 2", "Welcome series · bước 2"},
		{"", "", nil, "api", "API · transactional", "API · giao dịch"},
		{"", "", nil, "smtp", "SMTP relay · transactional", "SMTP relay · giao dịch"},
		{"", "", nil, "double_opt_in", "Double opt-in confirmation", "Thư xác nhận hai bước"},
		{"", "", nil, "", "", ""},
	} {
		en := SourceLabel(c.campaign, c.automation, c.node, c.kind)
		if en != c.en {
			t.Errorf("%+v: %q", c, en)
		}
		node := ""
		if c.node != nil {
			node = *c.node
		}
		row := store.IngestEventLogRow{Source: en, MessageKind: c.kind, MessageCampaign: c.campaign != "", MessageAutomation: c.automation != "", MessageNode: node}
		if got := sourceIn(vi, row); got != c.vi {
			t.Errorf("%+v (vi): %q", c, got)
		}
	}
}
