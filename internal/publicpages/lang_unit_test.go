package publicpages

// The words of the poll, survey and Grow pages in the page's language
// (unit tests: no database). English stays as it was.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/store"
)

// testSite is a site whose page is in a language ("en", "vi").
func testSite(code string) *site {
	set := &forms.Settings{Branding: forms.Branding{PublisherName: "Acme"}}
	s := &site{ws: &store.Workspace{Slug: "acme", Timezone: "UTC"}, set: set, links: forms.LinksOn("https://rc.example", "acme", false)}
	s.use(pageLang{Code: code, Labels: pagelabels.For(code, nil)})
	return s
}

func pollPage(answer *int) *polls.AnswerPage {
	cfg := polls.PollConfig{Kind: polls.KindEmoji, Faces: 5, Question: "How was this issue?"}
	closes := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	return &polls.AnswerPage{Config: cfg, Answers: cfg.AnswerList(), CampaignName: "Weekly Digest #141", Answerer: polls.AnswererContact,
		Email: "pat@example.com", Picked: 5, Answer: answer, ClosesAt: &closes,
		Results: &polls.PollShares{Total: 1234, Rows: []polls.PollShare{{Key: 5, Label: "5 of 5, very happy", Percent: 60}}}}
}

func TestPollPageWords(t *testing.T) {
	h := &Handler{}
	two := 2
	en := h.pollView(testSite("en"), "tok", pollPage(&two), false)
	for got, want := range map[string]string{
		en.Kicker:                "Weekly Digest #141 · poll",
		en.Status:                "You picked very happy in the email. Your answer stays unhappy until you confirm the change.",
		en.Confirm:               "Change my answer",
		en.ConfirmNote:           "Pick another face, then change your answer.",
		en.Yours:                 "Your answer: unhappy. To change it, pick another face.",
		en.Results.Line:          "1,234 answers · closes Mon 26 Oct",
		en.Results.Rows[0].Label: "5 of 5, very happy",
		en.Answers[0].Aria:       "1 of 5, very unhappy",
		en.Note:                  "This link is yours: it answers as pat@example.com, even from a forwarded email. The latest answer counts, until the poll closes on Mon 26 Oct.",
	} {
		if got != want {
			t.Errorf("English: got %q, want %q", got, want)
		}
	}

	vi := h.pollView(testSite("vi"), "tok", pollPage(&two), false)
	for got, want := range map[string]string{
		vi.Kicker:                "Weekly Digest #141 · bình chọn",
		vi.Status:                "Bạn đã chọn rất hài lòng trong email. Câu trả lời của bạn vẫn là không hài lòng cho đến khi bạn xác nhận thay đổi.",
		vi.Confirm:               "Đổi câu trả lời",
		vi.ConfirmNote:           "Hãy chọn khuôn mặt khác, rồi đổi câu trả lời.",
		vi.Results.Line:          "1.234 câu trả lời · đóng vào Th 2, 26 thg 10",
		vi.Results.Rows[0].Label: "5 trên 5, rất hài lòng",
		vi.Answers[0].Aria:       "1 trên 5, rất không hài lòng",
	} {
		if got != want {
			t.Errorf("Vietnamese: got %q, want %q", got, want)
		}
	}

	// A first answer, saved, on a 0–10 scale.
	scale := polls.PollConfig{Kind: polls.KindScale, Question: "How likely are you to recommend us?"}
	nine := 9
	p := &polls.AnswerPage{Config: scale, Answers: scale.AnswerList(), CampaignName: "Weekly Digest", Answerer: polls.AnswererContact,
		Name: "Pat", Picked: 9, Answer: &nine, Saved: true}
	if b := h.pollView(testSite("en"), "tok", p, true); b.Status != "Thanks, Pat. Your answer is saved." || b.Answers[9].Aria != "9 out of 10" {
		t.Errorf("English scale: %q, %q", b.Status, b.Answers[9].Aria)
	}
	if b := h.pollView(testSite("vi"), "tok", p, true); b.Status != "Cảm ơn Pat. Câu trả lời của bạn đã được lưu." || b.Answers[9].Aria != "9 trên 10" {
		t.Errorf("Vietnamese scale: %q, %q", b.Status, b.Answers[9].Aria)
	}
}

func TestSurveyPageWords(t *testing.T) {
	h := &Handler{}
	closes := time.Date(2026, 10, 30, 17, 0, 0, 0, time.UTC)
	page := func() *polls.SurveyPage {
		return &polls.SurveyPage{Name: "Reader survey", Slug: "readers", Answerer: polls.AnswererContact, Email: "pat@example.com", ClosesAt: &closes,
			Settings: polls.SurveySettings{OneResponse: true, Timezone: "America/New_York"}, Answers: map[string]any{},
			Questions: []polls.SurveyQuestion{
				{Key: "q1", Type: polls.QuestionScale, Text: "How likely are you to recommend us?", Required: true},
				{Key: "q2", Type: polls.QuestionMulti, Text: "What do you read?", Required: true, Answers: []polls.SurveyOption{{Key: 1, Label: "News"}, {Key: 2, Label: "Essays"}}},
				{Key: "q3", Type: polls.QuestionLong, Text: "Anything else?"},
			},
			Errors: []polls.SurveyError{{Key: "q2", Number: 2, Text: "What do you read?", Message: "Pick at least one answer."}}}
	}
	en := h.surveyView(testSite("en"), page(), "")
	for got, want := range map[string]string{
		en.Kicker:                "Survey · Acme",
		en.Answering:             "Answering as pat@example.com · open until Fri 30 Oct, 13:00 New York time",
		en.Until:                 "You can change them until Fri 30 Oct.",
		en.ErrorTitle:            "1 question needs an answer",
		en.Errors[0].Text:        "Question 2: What do you read?",
		en.Questions[0].Low:      "Not likely",
		en.Questions[1].Label:    "Question 2 · pick any",
		en.Questions[1].Error:    "Pick at least one answer.",
		en.Questions[2].Label:    "Question 3 · optional",
		en.Questions[2].Count:    "0 of 2,000 characters",
		en.Questions[2].CountTpl: "{n} of 2,000 characters",
	} {
		if got != want {
			t.Errorf("English: got %q, want %q", got, want)
		}
	}
	vi := h.surveyView(testSite("vi"), page(), "")
	for got, want := range map[string]string{
		vi.Kicker:             "Khảo sát · Acme",
		vi.Answering:          "Trả lời dưới tên pat@example.com · mở đến Th 6, 30 thg 10, 13:00 giờ New York",
		vi.ErrorTitle:         "1 câu hỏi cần câu trả lời",
		vi.Errors[0].Text:     "Câu hỏi 2: What do you read?",
		vi.Questions[0].High:  "Rất có khả năng",
		vi.Questions[1].Label: "Câu hỏi 2 · chọn tùy ý",
		vi.Questions[1].Error: "Hãy chọn ít nhất một câu trả lời.",
		vi.Questions[2].Count: "0/2.000 ký tự",
	} {
		if got != want {
			t.Errorf("Vietnamese: got %q, want %q", got, want)
		}
	}
	// The publisher's own words for an empty question stay theirs.
	p := page()
	p.Questions[1].RequiredText = "Tell us what you read."
	if got := h.surveyView(testSite("vi"), p, "").Questions[1].Error; got != "Tell us what you read." {
		t.Errorf("the publisher's words: %q", got)
	}
}

func TestGrowWords(t *testing.T) {
	en, vi := testSite("en"), testSite("vi")
	if got := en.ordinal(15, "hub.rank"); got != "15th" {
		t.Errorf("English rank: %q", got)
	}
	if got := vi.gt("hub.rank", "rank", vi.ordinal(15, "hub.rank")); got != "Hạng 15 trên bảng xếp hạng" {
		t.Errorf("Vietnamese rank: %q", got)
	}
	if got := en.articled("The Acme Field Guide", "hub.toNext"); got != "the Acme Field Guide" {
		t.Errorf("English reward: %q", got)
	}
	if got := vi.articled("The Acme Field Guide", "hub.toNext"); got != "The Acme Field Guide" {
		t.Errorf("Vietnamese reward: %q", got)
	}
	if got := en.orList([]string{"self-referrals", "bursts", "shared IPs"}, "hub.rulesExcluded"); got != "self-referrals, bursts, or shared IPs" {
		t.Errorf("English list: %q", got)
	}
	if got := vi.orList([]string{"a", "b", "c"}, "hub.rulesExcluded"); got != "a, b hoặc c" {
		t.Errorf("Vietnamese list: %q", got)
	}
	if got := en.fileDesc("application/pdf", 2_516_582); got != "PDF, 2.4 MB" {
		t.Errorf("English file: %q", got)
	}
	if got := vi.fileDesc("application/pdf", 2_516_582); got != "PDF, 2,4 MB" {
		t.Errorf("Vietnamese file: %q", got)
	}
	now := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	day := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	if got := en.shortDate(day, now); got != "19 Oct" {
		t.Errorf("English day: %q", got)
	}
	if got := vi.shortDate(day, now); got != "19 thg 10" {
		t.Errorf("Vietnamese day: %q", got)
	}
	if got := vi.gt("hub.trackMilestone", "label", "3 lượt giới thiệu", "reward", "Field Guide", "reached", "yes"); got != "3 lượt giới thiệu Field Guide (đã đạt)" {
		t.Errorf("Vietnamese milestone: %q", got)
	}
}

func TestAnswersWithoutWorkspace(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/p/nobody/x", nil)
	r.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en;q=0.8")
	if got := growMsgs.M("page.missing").Text(browserLabels(r).Lang(), nil); got != "Trang này không tồn tại." {
		t.Errorf("Vietnamese: %q", got)
	}
	r.Header.Set("Accept-Language", "en-US")
	if got := growMsgs.M("page.missing").Text(browserLabels(r).Lang(), nil); got != "This page doesn't exist." {
		t.Errorf("English: %q", got)
	}
	if got := browserLabels(r).Text("page.error.text", nil); !strings.HasPrefix(got, "Something went wrong") {
		t.Errorf("English error: %q", got)
	}
}

func TestPreferenceFieldProblems(t *testing.T) {
	vi := pagelabels.For("vi", nil)
	for _, c := range []struct {
		fe   audience.FieldError
		want string
	}{
		{audience.FieldError{Code: "number", Message: "Enter a number."}, "Hãy nhập một số."},
		{audience.FieldError{Code: "select", Options: []string{"Daily", "Weekly"}, Message: "Pick one of: Daily, Weekly."}, "Hãy chọn một trong: Daily, Weekly."},
		{audience.FieldError{Code: "too_long", Message: "Keep it under 500 characters."}, "Hãy viết dưới 500 ký tự."},
		{audience.FieldError{Code: "something_new", Message: "As audience wrote it."}, "As audience wrote it."},
	} {
		if got := fieldProblem(&c.fe, vi); got != c.want {
			t.Errorf("%s: %q, want %q", c.fe.Code, got, c.want)
		}
	}
	if got := fieldProblem(&audience.FieldError{Code: "number"}, pagelabels.English()); got != "Enter a number." {
		t.Errorf("English: %q", got)
	}
}
