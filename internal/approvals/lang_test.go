package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

var vi = i18n.For(i18n.Vietnamese)

// The rules a send matches keep their messages in the request's snapshot:
// the English stays as API clients read it, and a Vietnamese reader reads
// them in Vietnamese, also after the snapshot was stored. Rules stored
// without them read as stored.
func TestRulesReadInTheReadersLanguage(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	f := RuleFacts{Recipients: 47912, Verdict: readiness.VerdictWarnings, Warnings: 2, AuthorName: "Daniel Park", AuthorRole: "Content Writer", Now: now,
		Shares: []RuleShare{{Short: "Postal", Recipients: 4791, WarmingUp: true, WarmupDay: 9, WarmupDays: 21}}}
	p := ParsePolicy(nil)
	p.ApprovalLists = true
	f.ApprovalLists = []string{"Enterprise customers", "Beta Program"}
	rules := p.Evaluate(f)
	if got := rulesLine(rules); got != "rules: audience ≥ 10,000 · warming-up provider · author lacks send · list requires approval" {
		t.Fatalf("English: %s", got)
	}
	if rules[3].Label != "The audience includes Enterprise customers and Beta Program, which require approval" {
		t.Fatalf("English lists rule: %s", rules[3].Label)
	}

	raw, err := json.Marshal(Snapshot{Rules: rules})
	if err != nil {
		t.Fatal(err)
	}
	stored := ParseSnapshot(raw)
	local := localRules(vi, stored.Rules)
	want := []string{
		"Số người nhận 47.912, từ 10.000 trở lên",
		"Tuyến có Postal, đang khởi động (ngày 9/21)",
		"Daniel Park không có quyền campaigns:send · Content Writer",
		"Người nhận gồm Enterprise customers và Beta Program, được đánh dấu “Cần phê duyệt”",
	}
	for i, w := range want {
		if local[i].Label != w {
			t.Errorf("rule %d: %q, want %q", i, local[i].Label, w)
		}
	}
	if local[0].Badge != "người nhận ≥ 10.000" || local[1].Badge != "Nhà cung cấp đang khởi động" {
		t.Errorf("badges: %q, %q", local[0].Badge, local[1].Badge)
	}
	if got := rulesLineIn(vi, stored.Rules); got != "quy tắc: người nhận ≥ 10.000 · nhà cung cấp đang khởi động · tác giả không có quyền gửi · danh sách cần phê duyệt" {
		t.Errorf("inbox line: %s", got)
	}
	// English readers read what was stored.
	if en := localRules(i18n.Default, stored.Rules); en[0].Label != "Audience of 47,912 is at least 10,000" {
		t.Errorf("English after storing: %q", en[0].Label)
	}
	// A snapshot stored before the rules kept their messages reads in English.
	old := ParseSnapshot(json.RawMessage(`{"rules":[{"key":"audience_min","label":"Audience of 15,002 is at least 10,000","short":"audience ≥ 10,000","badge":"audience ≥ 10,000"}]}`))
	if got := localRules(vi, old.Rules); got[0].Label != "Audience of 15,002 is at least 10,000" || got[0].Badge != "audience ≥ 10,000" {
		t.Errorf("old snapshot: %+v", got[0])
	}
	if got := strings.Join(p.RuleLabelsIn(vi), " · "); got != "người nhận ≥ 10.000 · nhà cung cấp đang khởi động · tác giả không có quyền gửi · danh sách cần phê duyệt" {
		t.Errorf("policy in force: %s", got)
	}
}

// An invalidation's changes and reason keep their messages: the stored
// English is what it always was (without "after approval", which the screens
// add once), and a Vietnamese reader reads them in Vietnamese.
func TestInvalidationReadsInTheReadersLanguage(t *testing.T) {
	aud := json.RawMessage(`{"lists":["a"]}`)
	sched := json.RawMessage(`{"mode":"at","at":"2026-11-04T14:00:00Z","timezone":"America/New_York"}`)
	snap := Snapshot{Version: 9, ContentHash: "h1", Subject: "November Product Update", Audience: aud, AudienceCount: 47912, Schedule: sched, Links: []string{}, Sections: []string{}}
	cur := Current{Version: 9, ContentHash: "h1", Subject: snap.Subject, Audience: aud, AudienceCount: 54620, Schedule: sched}
	changes := diff(snap, cur, time.UTC)
	if len(changes) != 1 || changes[0].After != "54,620 (+14%)" {
		t.Fatalf("audience: %+v", changes)
	}
	reason := reasonOf(changes)
	if en := render(i18n.Default, reason); en != "audience moved +14%" {
		t.Fatalf("English reason: %s", en)
	}
	if got := render(vi, reason); got != "số người nhận đã thay đổi +14%" {
		t.Errorf("Vietnamese reason: %s", got)
	}
	raw, _ := json.Marshal(changes)
	local := changesFor(vi, raw)
	if local[0].Label != "Người nhận" || local[0].Before != "47.912" || local[0].After != "54.620 (+14%)" {
		t.Errorf("Vietnamese change: %+v", local[0])
	}
	if en := changesFor(i18n.Default, raw); en[0].Label != "Audience" || en[0].After != "54,620 (+14%)" {
		t.Errorf("English change: %+v", en[0])
	}

	// The subject and the schedule.
	cur.AudienceCount = 47912
	cur.Subject = "November Product Update: faster exports"
	cur.Schedule = json.RawMessage(`{"mode":"at","at":"2026-11-05T14:00:00Z","timezone":"America/New_York"}`)
	changes = diff(snap, cur, time.UTC)
	raw, _ = json.Marshal(changes)
	local = changesFor(vi, raw)
	if len(local) != 2 || local[0].Label != "Tiêu đề" || local[0].After != cur.Subject || local[1].Label != "Lịch gửi" || local[1].Before != "4 thg 11, 09:00" {
		t.Errorf("subject and schedule: %+v", local)
	}
	text, _ := json.Marshal(reasonOf(changes))
	if got := reasonFor(vi, "subject and schedule changed", text); got != "đã thay đổi tiêu đề và lịch gửi" {
		t.Errorf("stored reason, Vietnamese: %s", got)
	}
	if got := fieldsFor(vi, "subject and schedule changed", text); got != "tiêu đề và lịch gửi" {
		t.Errorf("fields, Vietnamese: %s", got)
	}
	if got := fieldsFor(i18n.Default, "subject and schedule changed", text); got != "subject and schedule" {
		t.Errorf("fields, English: %s", got)
	}

	// A reason stored before kept its message: English, said once.
	if got := reasonFor(vi, "audience moved +14% after approval", nil); got != "audience moved +14%" {
		t.Errorf("old reason: %s", got)
	}
	if got := changesFor(vi, json.RawMessage(`[{"field":"audience_count","label":"Audience","before":"15,002","after":"17,110 (+14%)"}]`)); got[0].Label != "Audience" || got[0].After != "17,110 (+14%)" {
		t.Errorf("old change: %+v", got[0])
	}
}

// An automation publish request keeps its change list's messages: a
// Vietnamese reader reads them in Vietnamese, older requests as stored.
func TestAutomationChangesInVietnamese(t *testing.T) {
	snap := Snapshot{Automation: &ApprovalAutomationSnapshot{Version: 4, Changes: []string{"added: A/B split"}},
		AutomationItems: []automationItem{{Kind: "added", Title: i18n.Text{ID: "automations.diff.headSplit"}}}}
	raw, _ := json.Marshal(snap)
	back := ParseSnapshot(raw)
	if got := localAutomation(vi, back).Changes[0]; got != "đã thêm: Chia A/B" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := localAutomation(i18n.Default, back).Changes[0]; got != "added: A/B split" {
		t.Errorf("English: %q", got)
	}
	back.AutomationItems = nil
	if got := localAutomation(vi, back).Changes[0]; got != "added: A/B split" {
		t.Errorf("stored before: %q", got)
	}
}

func TestHeldReasonsAndScheduleInVietnamese(t *testing.T) {
	if got := gapIn(vi, "Approval is still pending"); got != "Phê duyệt vẫn đang chờ" {
		t.Errorf("engine's reason: %s", got)
	}
	if got := gapIn(vi, "the approval no longer covers the campaign"); got != "phê duyệt không còn áp dụng cho chiến dịch" {
		t.Errorf("stored reason: %s", got)
	}
	if got := gapIn(vi, "Links: go.acme-promo.link is listed on SURBL"); got != "Links: go.acme-promo.link is listed on SURBL" {
		t.Errorf("a check's title stays as written: %s", got)
	}
	best := json.RawMessage(`{"mode":"best_time","at":"2026-11-04T14:00:00Z","timezone":"America/New_York"}`)
	if got := scheduleTextIn(vi, best, time.UTC); got != "Giờ tốt nhất trong vòng 24 giờ từ 4 thg 11, 09:00" {
		t.Errorf("schedule: %s", got)
	}
}

// explained is an engine error that says why in a reader's language.
type explained struct{}

func (explained) Error() string { return "Weekly Digest #142 is already sending." }
func (explained) In(l i18n.Lang) string {
	if l.Locale == i18n.Vietnamese {
		return "Weekly Digest #142 đang được gửi."
	}
	return "Weekly Digest #142 is already sending."
}

// A send the engine can't start answers why, in the reader's language.
func TestStartProblemSaysWhyInTheReadersLanguage(t *testing.T) {
	err := startProblem(i18n.WithLang(context.Background(), vi), fmt.Errorf("start: %w", explained{}))
	var p *api.Problem
	if !errors.As(err, &p) || p.Status != http.StatusConflict || p.Detail != "Weekly Digest #142 đang được gửi." {
		t.Fatalf("got %v", err)
	}
	if err := startProblem(context.Background(), errors.New("boom")); err == nil || err.Error() != "boom" {
		t.Fatalf("other errors pass through: %v", err)
	}
}
