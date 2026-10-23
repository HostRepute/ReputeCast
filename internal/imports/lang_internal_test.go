package imports

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The English an import writes reads exactly as it did before its words
// moved to messages/en.json.
func TestEnglishUnchanged(t *testing.T) {
	en := i18n.Default
	quoted := strconv.Quote("Acme Customers")
	xmlErr := errors.New("XML syntax error on line 1: unexpected EOF")
	sheetErrors := []*xlsxError{
		{reason: "notXlsx", err: errors.New("this isn't an .xlsx file (it can't be opened as a spreadsheet)")},
		{reason: "noWorkbook", err: errors.New("this .xlsx file has no workbook")},
		{reason: "workbook", err: fmt.Errorf("the workbook can't be read: %w", xmlErr)},
		{reason: "noSheets", err: errors.New("the workbook has no sheets")},
		{reason: "missingSheet", sheet: "xl/worksheets/sheet1.xml", err: fmt.Errorf("the workbook points to %s, which is missing", "xl/worksheets/sheet1.xml")},
		{reason: "sharedStrings", err: fmt.Errorf("the shared strings can't be read: %w", xmlErr)},
		{reason: "sheet", err: fmt.Errorf("the sheet can't be read: %w", xmlErr)},
		{reason: "tooManyRows", max: MaxFileRows + 1, err: fmt.Errorf("the sheet has more than %d rows", MaxFileRows+1)},
		{reason: "cell", err: fmt.Errorf("a cell can't be read: %w", xmlErr)},
	}
	cases := []struct{ got, want string }{
		{english(mLogStart.Keep(i18n.Args{"rows": int64(13739)})), "Import started · 13,739 rows"},
		{english(mLogMapping.Keep(i18n.Args{"kind": "audience", "source": quoted, "target": "Product Updates"})), `audience "Acme Customers" → list Product Updates`},
		{english(mLogMapping.Keep(i18n.Args{"kind": "group", "source": quoted, "target": "Product Updates"})), `group "Acme Customers" → list Product Updates`},
		{english(mLogMapping.Keep(i18n.Args{"kind": "list", "source": quoted, "target": "Product Updates"})), `list "Acme Customers" → list Product Updates`},
		{english(mLogMapping.Keep(i18n.Args{"kind": "file", "source": "people.csv", "target": "Newsletter"})), "people.csv → list Newsletter"},
		{english(mLogConsent.Keep(i18n.Args{"consent": "trust"})), "consent: trust double opt-in"},
		{english(mLogConsent.Keep(i18n.Args{"consent": "repermission"})), "consent: re-permission everyone"},
		{english(mLogConsent.Keep(i18n.Args{"consent": "source"})), "consent: keep source opt-in"},
		{english(mLogPulled.Keep(i18n.Args{"providers": []string{"SES", "Postmark", "Postal"}, "added": "1200"})), "Pulled suppressions from SES, Postmark and Postal: 1200 new"},
		{english(mLogBatch.Keep(i18n.Args{"batch": "1000", "rows": 500, "imported": int64(12345), "merged": int64(3), "suppressed": int64(0), "skipped": int64(1)})),
			"Batch 1000 · 500 rows: 12,345 imported, 3 merged, 0 suppressed, 1 skipped"},
		{english(mLogPushed.Keep(i18n.Args{"count": 1033, "providers": []string{"Amazon SES", "Postmark"}})), "Pushed 1,033 new suppressions to Amazon SES and Postmark"},
		{english(mLogNextSync.Keep(i18n.Args{"count": 4, "providers": []string{"Postal"}})), "4 new suppressions reach Postal with the next provider sync"},
		{english(mLogRepermWaiting.Keep(i18n.Args{"count": 6716})), "6,716 contacts wait for re-permission; connect a System provider to email them"},
		{english(mLogFinished.Keep(i18n.Args{"contacts": int64(46118), "suppressions": int64(1033)})), "Import finished · 46,118 contacts · 1,033 suppressions"},
		{english(mLogRetry.Keep(i18n.Args{"source": "Brevo", "status": "503", "attempt": "2"})), "Brevo answered 503 while reading contacts; retrying (attempt 2)"},
		{english(mLogRetry429.Keep(i18n.Args{"source": "Brevo", "wait": (2 * time.Second).String()})), "Brevo returned 429 Too Many Requests; waiting 2s before retrying"},
		{english(mLogRead.Keep(i18n.Args{"rows": int32(47365), "fields": "11", "source": "Brevo"})), "Read 47,365 rows and 11 fields from Brevo"},
		{english(mLogPaused.Keep(i18n.Args{"source": "Brevo", "at": "14:05:00 UTC"})), "Brevo is rate-limiting us (429); paused, resuming at 14:05:00 UTC"},
		{english(mLogSlowed.Keep(i18n.Args{"pace": "3", "source": "Brevo"})), "Slowed down to 3 requests a second to Brevo"},
		{english(mLogTemplatesFix.Keep(i18n.Args{"transactional": "4", "saved": "2", "manual": "1"})), "Imported 4 transactional templates as drafts and 2 saved templates; 1 need a manual fix"},
		{english(mMailchimpRejected.Keep(i18n.Args{"said": `: "API Key Invalid"`, "dc": "us21"})),
			`Mailchimp rejected the API key: "API Key Invalid" from the us21 data center. Keys look like xxxxxxxx-us21; the suffix must match the data center the account lives in.`},
		{fileError(mBadColumns, i18n.Args{"file": "people.csv", "row": 1204, "columns": "9", "header": "8"}).Message, "people.csv: row 1,204 has 9 columns, the header has 8"},
		{fileError(mTooManyRows, i18n.Args{"file": "people.csv", "max": strconv.Itoa(MaxFileRows)}).Message, "people.csv has more than 500000 rows. Split it into smaller files."},
		{rowUnreadable("people.csv", &csv.ParseError{Line: 12, Err: csv.ErrBareQuote}).Message, fmt.Sprintf("%s: row %d can't be read (%v). Fix the file and upload it again.", "people.csv", 12, csv.ErrBareQuote)},
		{rowUnreadable("people.csv", &csv.ParseError{Line: 12, Err: csv.ErrQuote}).Message, fmt.Sprintf("%s: row %d can't be read (%v). Fix the file and upload it again.", "people.csv", 12, csv.ErrQuote)},
		{unreadable("people.csv", errors.New("unexpected EOF")).Message, "people.csv can't be read: unexpected EOF."},
		{say(context.Background(), mEffectBlockedPushed, i18n.Args{"providers": []string{"SES", "Postmark"}}), "All streams blocked, pushed to SES and Postmark"},
		{mergeChange{kind: "tags", addTags: []string{"a"}}.result(en), "Merge · adds 1 tag, keeps consent"},
		{mergeChange{kind: "tags", addTags: []string{"a", "b", "c"}}.result(en), "Merge · adds 3 tags, keeps consent"},
		{mergeChange{kind: "fills", fills: []string{"first name", "company", "plan"}}.result(en), "Merge · fills first name, company and plan"},
		{mDupOnList.Text(en, i18n.Args{"reason": "hard_bounce", "day": "2 Jan"}), "On suppression list · hard bounced 2 Jan"},
		{mDupOnList.Text(en, i18n.Args{"reason": "manual", "day": "2 Jan"}), "On suppression list · suppressed 2 Jan"},
		{listSummary(en, nil), "no lists"},
		{listSummary(en, []string{"A", "B"}), "A, B"},
		{listSummary(en, []string{"A", "B", "C"}), "3 lists"},
		{sourceShortIn(en, "import:mailchimp"), "Mailchimp"},
		{sourceShortIn(en, "import:"), "File"},
		{sourceShortIn(en, "form:docs"), "Signup form · docs"},
		{eta(en, 30*time.Second, 50, 100), "less than a minute left"},
		{eta(en, time.Minute, 50, 100), "about 1 minute left"},
		{eta(en, 3*time.Minute, 50, 100), "about 3 minutes left"},
		{invalidProblemIn(en, ProblemNoMX), ProblemNoMX},
		{invalidProblemIn(en, ProblemService), ProblemService},
		{invalidProblemIn(en, "no domain"), "no domain"},
	}
	for _, xe := range sheetErrors {
		cases = append(cases, struct{ got, want string }{unreadable("people.xlsx", xe).Message, fmt.Sprintf("%s can't be read: %v.", "people.xlsx", xe)})
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got  %q\nwant %q", c.got, c.want)
		}
	}
}

// Every risky group has its words, and Vietnamese has every message.
func TestCatalog(t *testing.T) {
	for _, m := range riskyMeta {
		for _, part := range []string{"label", "why"} {
			if _, err := msgs.Lookup("risky." + m.key + "." + part); err != nil {
				t.Errorf("risky.%s.%s: %v", m.key, part, err)
			}
		}
	}
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("Vietnamese is missing %v", missing)
	}
}

// A kept line reads in the reader's language after it went through the
// database, lists joined as the language writes them; English readers and
// lines whose message is gone read the English stored.
func TestLineIn(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	parts := []i18n.Text{mLogPushed.Keep(i18n.Args{"count": 1033, "providers": []string{"Amazon SES", "Postmark"}})}
	raw, err := json.Marshal(parts)
	if err != nil {
		t.Fatal(err)
	}
	stored := english(parts[0])
	if got := lineIn(vi, stored, raw); got != "Đã đẩy 1.033 địa chỉ chặn gửi mới đến Amazon SES và Postmark" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := lineIn(i18n.Default, stored, raw); got != stored {
		t.Errorf("English: %q", got)
	}
	if got := lineIn(vi, "Old line", []byte(`[{"id":"imports.log.gone"}]`)); got != "Old line" {
		t.Errorf("a message that's gone: %q", got)
	}
	if got := lineIn(vi, "Old line", nil); got != "Old line" {
		t.Errorf("a line from before: %q", got)
	}
	if got := errorIn(vi, ErrNoKey); got != "Hãy dán một khóa API." {
		t.Errorf("no key: %q", got)
	}
	if got := errorIn(vi, errors.New("connection reset")); got != "connection reset" {
		t.Errorf("an error in its own words: %q", got)
	}
}
