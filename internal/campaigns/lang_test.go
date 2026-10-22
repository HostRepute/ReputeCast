package campaigns

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestListNotesInTheReadersLanguage(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	cases := []struct{ got, want string }{
		{abNote(en, "subject", "B"), "A/B subject, winner B"},
		{abNote(en, "send_time", ""), "A/B send time"},
		{abNote(vi, "subject", "B"), "A/B tiêu đề, B thắng"},
		{autoPausedNote(en, 1, "yahoo", "deferral"), "auto-paused once, Yahoo deferrals"},
		{autoPausedNote(en, 2, "", ""), "auto-paused twice"},
		{autoPausedNote(en, 3, "gmail", "hard_bounce"), "auto-paused 3 times, Gmail bounces"},
		{autoPausedNote(en, 1, "comcast", "complaint"), "auto-paused once, Other complaints"},
		{autoPausedNote(vi, 1, "yahoo", "deferral"), "tự tạm dừng một lần vì Yahoo hoãn nhận"},
		{autoPausedNote(vi, 2, "", ""), "tự tạm dừng 2 lần"},
		{mAudienceAll.Text(vi, nil), "Tất cả người đăng ký"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	// Version labels campaigns wrote in English, rows from before included.
	for _, c := range []struct{ stored, want string }{
		{"created from template “Product update”", "tạo từ mẫu “Product update”"},
		{"made from the web post", "tạo từ bài viết trên trang web"},
		{"restored v12", "khôi phục v12"},
		{"copied from “Weekly Digest #141”", "sao chép từ “Weekly Digest #141”"},
		{"link replaced from Review & send", "link replaced from Review & send"},
		{"Before the redesign", "Before the redesign"},
	} {
		if got := versionLabel(vi, c.stored); got != c.want {
			t.Errorf("version label %q: %q, want %q", c.stored, got, c.want)
		}
		if got := versionLabel(en, c.stored); got != c.stored {
			t.Errorf("English version label %q: %q", c.stored, got)
		}
	}
	if got := mVersionFromTemplate.Text(en, i18n.Args{"name": "Product update"}); got != "created from template “Product update”" {
		t.Errorf("stored English: %q", got)
	}
	// One message, so one pair of brackets: a pseudo word inside another would nest them.
	if p := abNote(i18n.For(i18n.Pseudo), "subject", "B"); !strings.HasPrefix(p, "⟦") || strings.Count(p, "⟦") != 1 || !strings.Contains(p, "A/B") {
		t.Errorf("pseudo-locale: %q", p)
	}
}
