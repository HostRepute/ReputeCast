package ai

import (
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The activity log keeps why a request failed, and an unsaved segment's
// label, in English; each reader sees them in their language. A provider's
// own error type and a named target stay as they are.
func TestActivityTextInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	en := i18n.For(i18n.English)
	id := uuid.New()
	cases := []struct{ name, got, want string }{
		{"rate limited", detailIn(vi, detailRateLimited), "bị giới hạn tần suất"},
		{"timed out", detailIn(vi, detailTimedOut), "hết thời gian chờ"},
		{"unreachable", detailIn(vi, detailUnreachable), "không kết nối được"},
		{"a rejected key", detailIn(vi, "401"+detailKeyRejected), "401 khóa bị từ chối"},
		{"a rejected key in English", detailIn(en, "401"+detailKeyRejected), "401 key rejected"},
		{"a provider's own error", detailIn(vi, "529 overloaded"), "529 overloaded"},
		{"not a status", detailIn(vi, "the"+detailKeyRejected), "the key rejected"},
		{"a new segment", onIn(vi, "segment", nil, newSegmentLabel), "Phân khúc mới"},
		{"a new segment in English", onIn(en, "segment", nil, newSegmentLabel), "New segment"},
		{"a segment named New segment", onIn(vi, "segment", &id, newSegmentLabel), "New segment"},
		{"a campaign", onIn(vi, "campaign", &id, "Weekly Digest #142"), "Weekly Digest #142"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.name, c.got, c.want)
		}
	}
}
