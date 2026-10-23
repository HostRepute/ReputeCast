package earn

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Every catalog key the package names is in messages/en.json (msgs.M
// panics on a missing key, which would otherwise surface only when that
// problem or page is first shown).
func TestMessageKeysExist(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`"((?:cancel|connect|grant|import|notify|page|pages|portal|prefs|tier|tokens|unsub|visibility|welcome)\.[A-Za-z_.]+)"`)
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

// Prices, dates and Stripe's reasons in each language: ReputeCast's own
// formats stay as they were in English, the region's in Vietnamese.
func TestFormatsInEachLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	day := time.Date(2027, 10, 7, 9, 0, 0, 0, time.UTC)
	for _, c := range []struct{ got, want string }{
		{publicPrice(800, "usd", "month"), "$8 a month"},
		{publicPriceIn(vi, 800, "usd", "month"), "$8 mỗi tháng"},
		{priceText(8000, "usd", "year"), "$80/year"},
		{priceTextIn(vi, 8000, "usd", "year"), "$80/năm"},
		{priceText(450, "eur", ""), "€4.50"},
		{dateIn(defaultLang, day), "7 Oct 2027"},
		{dateIn(vi, day), "7 thg 10, 2027"},
		{dayMonthIn(defaultLang, day), "7 Oct"},
		{shortDateIn(defaultLang, day), "Thu 7 Oct"},
		{currencyNameIn(defaultLang, "usd"), "US dollars"},
		{currencyNameIn(vi, "usd"), "đô la Mỹ"},
		{currencyNameIn(vi, "chf"), "CHF"},
		{platformNameIn(defaultLang, "csv"), "a CSV file"},
		{platformNameIn(vi, "csv"), "tệp CSV"},
		{platformNameIn(vi, "substack"), "Substack"},
		{failureShort(defaultLang, "Your card has insufficient funds."), "insufficient funds"},
		{failureShort(vi, "Your card has insufficient funds."), "không đủ số dư"},
		{failureShort(vi, "Your card was declined."), "thẻ bị từ chối"},
		{failureShort(defaultLang, "Your card has a problem."), "a problem"},
		{failureSentence(vi, "Your card has insufficient funds."), "Thẻ của bạn không đủ số dư."},
		{failureSentence(vi, "Something else."), "Something else."},
		{syncErrorIn(vi, "no answer"), "không có phản hồi"},
		{syncErrorIn(vi, "402 Payment Required"), "402 Payment Required"},
	} {
		if c.got != c.want {
			t.Errorf("%q, want %q", c.got, c.want)
		}
	}
	if !englishText(defaultLang, "page.become") || englishText(vi, "page.become") {
		t.Error("page.become is English in English only")
	}
	for _, k := range stripeReasons {
		for _, form := range []string{"full", "short"} {
			if !slices.Contains(msgs.Keys(), "reason."+form+"."+k) {
				t.Errorf("reason.%s.%s is missing", form, k)
			}
		}
	}
	for _, k := range []string{"paidMember", "complimentary", "price", "connection", "import", "tier", "contact"} {
		if !slices.Contains(msgs.Keys(), "notFound."+k) || slices.Contains(msgs.Missing(i18n.Vietnamese), "notFound."+k) {
			t.Errorf("notFound.%s is missing or untranslated", k)
		}
	}
}

// A tier's address from a name in another language keeps its letters,
// without their marks (đ included).
func TestSlugFromFoldsMarks(t *testing.T) {
	for in, want := range map[string]string{"Hội viên Đặc biệt": "hoi-vien-dac-biet", "Weekly Digest Insider": "weekly-digest-insider"} {
		if got := slugFrom(in); got != want {
			t.Errorf("slugFrom(%q) = %q, want %q", in, got, want)
		}
	}
}

// A source ReputeCast wrote before sources kept their message reads in the
// reader's language; one a person wrote stays as stored.
func TestSourceKept(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	if got := sourceKept("gift", "Gift from Priya Natarajan", "Priya Natarajan").In(vi); got != "Quà tặng từ Priya Natarajan" {
		t.Errorf("gift: %q", got)
	}
	if got := sourceKept("import", "Imported from Substack", "").In(vi); got != "Nhập từ Substack" {
		t.Errorf("import: %q", got)
	}
	if got := sourceKept("import", "Imported from a CSV file", "").In(vi); got != "Nhập từ tệp CSV" {
		t.Errorf("CSV import: %q", got)
	}
	for _, c := range [][3]string{{"gift", "A present from Priya", "Priya Natarajan"}, {"gift", "Gift from Priya", ""}, {"page", "Upgrade page", ""}, {"import", "From the old shop", ""}} {
		if !sourceKept(c[0], c[1], c[2]).IsZero() {
			t.Errorf("%v: kept a label ReputeCast didn't write", c)
		}
	}
}
