package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

var vietnamese = i18n.For(i18n.Vietnamese)

// The kept sentences write the English exactly as the engine always has:
// counts grouped, rates per second with at most one decimal, percentages
// with two decimals below 1%, a day as "2 Oct".
func TestSentenceEnglish(t *testing.T) {
	day := time.Date(2026, 10, 2, 9, 0, 0, 0, time.FixedZone("ICT", 7*3600)).Format(time.RFC3339)
	cases := []struct {
		t    i18n.Text
		want string
	}{
		{keep("cause.imported", i18n.Args{"scope": "mailbox_provider", "mailbox": "outlook", "who": "", "total": 1420, "events": 6, "complaints": 7,
			"source": "mailchimp", "sourceName": "Mailchimp", "importDay": day}),
			"1,420 Outlook recipients came from the Mailchimp import on 2 Oct and have never opened. 6 of the 7 complaints are from this group."},
		{keep("cause.imported", i18n.Args{"scope": "route", "who": "", "total": 3, "events": 2, "complaints": 2, "source": "none", "sourceName": "an", "importDay": "none"}),
			"3 recipients came from the an import and have never opened. 2 of the 2 complaints are from this group."},
		{keep("rec.excludeImported", i18n.Args{"scope": "route", "who": "", "exclude": 1200, "held": 2522, "expectedPct": 0.05}),
			"Exclude the 1,200 never-engaged  imports from the remaining 2,522. Expected complaint rate after the change: about 0.05%."},
		{keep("rec.excludeNever", i18n.Args{"scope": "mailbox_provider", "mailbox": "other", "who": "", "exclude": 12, "held": 40, "expectedPct": 1.25}),
			"Exclude the 12 never-engaged Other recipients from the remaining 40. Expected complaint rate after the change: about 1.2%."},
		{keep("cause.throttled", i18n.Args{"scope": "mailbox_provider", "mailbox": "yahoo", "who": "", "throttleRate": 4.0}), "Yahoo was throttled to 4 /s with 0 deferrals."},
		{keep("rec.lower", i18n.Args{"scope": "route", "who": "SES", "lowerRate": 2.5}), "Lower the SES rate to 2.5 /s for the rest of this send."},
		{keep("cause.slowingLead", i18n.Args{"scope": "campaign", "who": ""}), "the campaign is slowing the send down."},
		{keep("opt.changes", i18n.Args{"combo": "all", "exclude": 1420, "group": "imported", "rest": 1102, "changeRate": 2, "automation": "Re-permission"}),
			"Exclude 1,420 never-engaged imports, send to 1,102 at 2 /s, add the excluded contacts to Re-permission."},
		{keep("opt.changes", i18n.Args{"combo": "rate", "exclude": 0, "group": "never", "rest": 2810, "changeRate": 2, "automation": ""}), "Send to 2,810 at 2 /s."},
		{keep("opt.resumeAll", i18n.Args{"held": 120, "metric": "none", "ratePct": 0.0, "expect": int64(0), "more": "no"}),
			"Send to the 120 remaining recipients. The thresholds re-arm from the current counts, so only new events can pause the campaign again."},
		{keep("opt.resumeScoped", i18n.Args{"scope": "mailbox_provider", "mailbox": "outlook", "who": "", "held": 2436, "metric": "complaint", "ratePct": 0.18, "expect": int64(4), "more": "yes"}),
			"Send to all 2,436 remaining Outlook recipients. The threshold re-arms from the current counts, so only new complaints can pause it again. At today's 0.18%, expect about 4 more."},
		{keep("cause.warmupCap", i18n.Args{"sentToday": 1960, "cap": 4000, "perSecondRate": 14.25}),
			"Not a quota or rate issue: 1,960 of today's 4,000 warm-up cap used, 14.2 /s."},
	}
	for _, c := range cases {
		if got := sentence(i18n.Default, c.t); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.t.ID, got, c.want)
		}
	}
	// Kept as JSON, the values come back as float64 and RFC 3339 strings.
	b, _ := json.Marshal(cases[0].t)
	var back i18n.Text
	_ = json.Unmarshal(b, &back)
	if got := sentence(vietnamese, back); got != "1.420 người nhận ở Outlook đến từ lần nhập từ Mailchimp ngày 2 thg 10 và chưa từng mở thư. 6 trên 7 khiếu nại đến từ nhóm này." {
		t.Errorf("Vietnamese: %q", got)
	}
}

// A detail keeps its messages in campaign_incidents.detail ("i18n") and
// reads in the reader's language; one stored before reads its note.
func TestDetailIn(t *testing.T) {
	var d IncidentDetail
	d.setNote(keep("note.bounce", nil))
	d.addCause(keep("cause.staleLead", nil), keep("cause.stale", i18n.Args{"scope": "mailbox_provider", "mailbox": "other", "who": "", "events": 40, "sent": 1000}))
	d.addCause(i18n.Text{}, keep("cause.quota", i18n.Args{"used": 100, "max": 50000}))
	d.addRecommendation(keep("rec.lower", i18n.Args{"scope": "mailbox_provider", "mailbox": "other", "who": "", "lowerRate": 2}))
	d.setOptions(thresholdOptions(trip{scope: "mailbox_provider", mailbox: "other", metric: "bounce"},
		i18n.Args{"scope": "mailbox_provider", "mailbox": "other", "who": ""}, 300, &ResumeChanges{Rate: 2}))
	d.ErrorKind, d.Continuing = "connection errors", []string{"Gmail", "Other"}
	d.Log = []IncidentLogLine{{Text: "ECONNREFUSED mta.acme.io:443", Bad: true}, {Text: english("probe.failing", nil)}}
	if d.Causes[0].Body != "40 of 1,000 Other recipients bounced; most hard bounces are unknown mailboxes, which points at an old or unverified list." ||
		d.Options[2].Title != "Cancel the Other share" || d.Causes[1].Lead != "" || d.Recommendations[0] != "Lower the Other rate to 2 /s for the rest of this send." {
		t.Fatalf("English: %+v", d)
	}
	back := parseDetail(d.stored())
	var raw map[string]any
	_ = json.Unmarshal(d.stored(), &raw)
	if raw["i18n"] == nil || raw["note"] != d.Note {
		t.Fatalf("stored: %s", d.stored())
	}
	v := back.In(vietnamese)
	if v.Note != text(vietnamese, "note.bounce", nil) || v.Causes[0].Lead != "Địa chỉ cũ hoặc nhập sai." ||
		v.Causes[0].Body != "40 trên 1.000 người nhận ở Khác bị trả thư; phần lớn thư trả lại vĩnh viễn là do hộp thư không tồn tại, cho thấy danh sách đã cũ hoặc chưa được xác minh." ||
		v.Causes[1].Body != "Không phải vấn đề hạn mức: đã dùng 100 trên 50.000." || v.Options[2].Title != "Hủy phần của Khác" ||
		v.ErrorKind != "lỗi kết nối" || v.Continuing[1] != "Khác" || v.Log[0].Text != "ECONNREFUSED mta.acme.io:443" || v.Log[1].Text != "kiểm tra tình trạng · vẫn lỗi" {
		t.Fatalf("Vietnamese: %+v", v)
	}
	if back.In(i18n.Default).Causes[0].Body != d.Causes[0].Body {
		t.Fatal("English readers read what is stored")
	}
	// Stored before its sentences were kept: the note reads in Vietnamese,
	// sentences with values stay as stored.
	old := parseDetail([]byte(`{"note":"Bounce rates above 2% tell mailbox providers the list is old; providers such as SES review accounts above 5%.",
		"causes":[{"lead":"Stale or mistyped addresses.","body":"40 of 1,000 Gmail recipients bounced."}]}`))
	if o := old.In(vietnamese); o.Note != text(vietnamese, "note.bounce", nil) || o.Causes[0].Body != "40 of 1,000 Gmail recipients bounced." {
		t.Fatalf("older detail: %+v", o)
	}
}

// The snapshot's own words read in the reader's language; names and the
// providers' replies stay.
func TestSnapshotIn(t *testing.T) {
	s := LiveSnapshot{PausedBy: "System", StartedBy: "Linh Nguyen", Error: english("fail.render", i18n.Args{"error": "liquid: unknown tag"}),
		Mailboxes:    []LiveMailbox{{Key: "gmail", Name: "Gmail"}, {Key: "other", Name: "Other"}},
		DeliveredMix: []LiveShare{{Key: "other", Name: "Other", Percent: 12.5}},
		Routes:       []LiveRouteShare{{Short: "SES", Percent: 60}, {Short: "Postal", Percent: 40}},
		Details:      LiveDetails{Route: "SES 60% · Postal 40%", Audience: "All subscribers + VIP"},
		Failovers:    []LiveFailover{{Stream: "Automations", To: "Postmark"}}, SharedStreams: []string{"System"},
		MailboxesHeld: []string{"Other"}, Continuing: []string{"Gmail"}, EventsVia: "SNS, the bounce mailbox and Postal webhooks",
		text: &liveText{via: []viaPart{{kind: "sns"}, {kind: "mailbox"}, {kind: "webhooks", name: "Postal"}}, audience: audienceParts{all: true, names: []string{"VIP"}},
			shared: []string{"system"}, failovers: []string{"automation"}, held: []string{"other"}, continuing: []string{"gmail"}}}
	hard := &eventParts{kind: "hardBounce", short: "SES", code: "550", diag: "5.1.1 user unknown", suppresses: true}
	open := &eventParts{kind: "openedHuman", diag: "Chrome on macOS", vary: "B"}
	s.Events = []LiveEvent{{Label: "hard bounce", Short: "bounce", Detail: hard.detail(i18n.Default), parts: hard}, {Label: "opened · human", Short: "open", Detail: open.detail(i18n.Default), parts: open}}
	if s.Events[0].Detail != "SES · 550 5.1.1 user unknown · added to suppression list" || s.Events[1].Detail != "Chrome on macOS · variant B" {
		t.Fatalf("English feed: %+v", s.Events)
	}
	v := s.In(vietnamese)
	if v.PausedBy != "Hệ thống" || v.StartedBy != "Linh Nguyen" || v.Error != "Không thể dựng email: liquid: unknown tag" ||
		v.Mailboxes[1].Name != "Khác" || v.DeliveredMix[0].Name != "Khác" || v.Details.Route != "SES 60% · Postal 40%" ||
		v.Details.Audience != "Tất cả người đăng ký + VIP" || v.Failovers[0].Stream != "Tự động hóa" || v.SharedStreams[0] != "Hệ thống" ||
		v.MailboxesHeld[0] != "Khác" || v.Continuing[0] != "Gmail" || v.EventsVia != "SNS, hộp thư nhận thư trả lại và webhook của Postal" {
		t.Fatalf("Vietnamese snapshot: %+v", v)
	}
	if v.Events[0].Label != "trả thư vĩnh viễn" || v.Events[0].Detail != "SES · 550 5.1.1 user unknown · đã thêm vào danh sách chặn gửi" ||
		!strings.HasSuffix(v.Events[1].Detail, " · biến thể B") || v.Events[1].Short != "mở" {
		t.Fatalf("Vietnamese feed: %+v", v.Events)
	}
	if s.Mailboxes[1].Name != "Other" || s.Events[0].Label != "hard bounce" {
		t.Fatal("In changed the snapshot it was given")
	}
	if s.In(i18n.Default).Details.Audience != "All subscribers + VIP" {
		t.Fatal("English readers read the snapshot as built")
	}
}

// Problems keep their English and read in the reader's language.
func TestStateError(t *testing.T) {
	var se *StateError
	if !errors.As(stateErr("state.notSending", campaignArgs(`Q4 "Launch"`, "sent")), &se) {
		t.Fatal("not a StateError")
	}
	if se.Msg != `"Q4 \"Launch\"" isn't sending (it is sent).` || se.In(vietnamese) != `“Q4 \"Launch\"” không đang gửi (trạng thái hiện tại: đã gửi).` {
		t.Fatalf("%q / %q", se.Msg, se.In(vietnamese))
	}
	if got := sendErrorIn(vietnamese, english("fail.noRoute", nil)); !strings.HasPrefix(got, "Chưa có nhà cung cấp gửi thư nào") {
		t.Fatalf("send error: %q", got)
	}
	if got := sendErrorIn(vietnamese, "something else"); got != "something else" {
		t.Fatalf("other error: %q", got)
	}
}
