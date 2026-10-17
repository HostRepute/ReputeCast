package earn

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

// Every catalog key the package names is in messages/en.json (msgs.M
// panics on a missing key, which would otherwise surface only when that
// problem or page is first shown).
func TestMessageKeysExist(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`"((?:cancel|connect|grant|import|notify|page|pages|tier|welcome)\.[A-Za-z_.]+)"`)
	keys := msgs.Keys()
	seen := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" || f == "messages_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			key := m[1]
			if key == "notify.attention." || key[len(key)-1] == '.' {
				continue
			}
			seen++
			if !slices.Contains(keys, key) {
				t.Errorf("%s: %q isn't in messages/en.json", f, key)
			}
		}
	}
	for _, reason := range []string{"key_rejected", "webhook_missing", "webhook_silent"} {
		for _, part := range []string{"title", "body"} {
			if !slices.Contains(keys, "notify.attention."+reason+"."+part) {
				t.Errorf("notify.attention.%s.%s is missing", reason, part)
			}
		}
	}
	if seen < 50 {
		t.Fatalf("found only %d keys; the pattern is out of date", seen)
	}
}

func TestMessagesRender(t *testing.T) {
	cases := []struct {
		key  string
		args map[string]any
		want string
	}{
		{"page.saving", map[string]any{"count": 2}, "2 months free"},
		{"page.trialDays", map[string]any{"count": 14}, "14 days free"},
		{"page.youreMember", map[string]any{"tier": "Insider", "article": article("Insider")}, "You're an Insider"},
		{"page.become", map[string]any{"tier": "founding member", "article": article("Founding member")}, "Become a founding member"},
		{"welcome.note", map[string]any{"tier": "Insider"}, "Stripe emails your receipt. “Welcome to Insider” is on its way too."},
		{"page.youreMember", map[string]any{"tier": "none"}, "You're a paid member"},
		{"page.trialNote", map[string]any{"date": "29 Oct", "price": "$80 a year"}, "Card needed. Free until 29 Oct, then $80 a year from 29 Oct. We email you 3 days before the trial ends."},
		{"page.defaultWelcome", nil, "Welcome to {{ tier }}, {{ first_name }}. Your first members-only issue arrives {{ next_issue }}."},
		{"connect.modeConnected", map[string]any{"mode": "live"}, "A live key is already connected. Replace it from Earn › Settings › Payments."},
		{"notify.importBody", map[string]any{"count": 1150, "paid": 1121, "free": 12, "comp": 17}, "1,150 rows: 1,121 paid, 12 free, 17 complimentary."},
	}
	for _, c := range cases {
		if got := msgs.M(c.key).Text(defaultLang, c.args); got != c.want {
			t.Errorf("%s = %q, want %q", c.key, got, c.want)
		}
	}
}
