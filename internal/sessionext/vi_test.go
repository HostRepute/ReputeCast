package sessionext_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// A session that ended reads so in the member's language.
func TestEndedSessionInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c, userID := signedIn(t, env, "aisha@acme.io")
	c.Header = http.Header{"Accept-Language": {"vi"}}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE sessions SET last_seen_at = now() - interval '13 hours' WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	p := c.Do("POST", "/me/session/extend", nil).MustStatus(t, 401).Problem(t)
	if d, _ := p["detail"].(string); p["code"] != "session_expired" || !strings.HasPrefix(d, "Phiên đăng nhập") || !strings.HasSuffix(d, "nội dung chưa lưu trong thẻ này vẫn được giữ.") {
		t.Errorf("ended: %v", p)
	}
}
