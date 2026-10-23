package publicpages

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// surveyOptionView is one answer of a choice, scale or face question.
type surveyOptionView struct {
	Value   string
	Label   string
	Aria    string
	Face    string
	Checked bool
}

// surveyQuestionView is one question on the page.
type surveyQuestionView struct {
	Key         string
	Number      int
	Type        string
	Text        string
	Description string
	Required    bool
	// Label is "Question 1", "Question 2 · pick any", "Question 4 · optional".
	Label   string
	Options []surveyOptionView
	Low     string
	High    string
	Text1   string
	Max     int
	Count   string
	// CountTpl is Count for the script, "{n}" standing for the count.
	CountTpl string
	Error    string
	// FromPoll marks question 1 when the email's poll answered it.
	FromPoll bool
}

type surveyErrorView struct {
	Href string
	Text string
}

// surveyBody is what survey.html renders.
type surveyBody struct {
	State       string
	Kicker      string
	Name        string
	Description string
	Answering   string
	Intro       string
	IntroTone   string
	Questions   []surveyQuestionView
	Errors      []surveyErrorView
	ErrorTitle  string
	Action      string
	Submit      string
	Keep        string
	Until       string
	// Done state.
	DoneTitle string
	DoneText  string
	ChangeURL string
	Back      string
	// Closed and refused states.
	Heading string
	Text    string
	Join    string
	Test    bool
	Footer  pollFooter
}

// zoneCity is "New York time" for America/New_York, in the page's
// language ("giờ New York").
func (s *site) zoneCity(tz string) string {
	if i := strings.LastIndexByte(tz, '/'); i >= 0 {
		return s.gt("survey.zone", "city", strings.ReplaceAll(tz[i+1:], "_", " "))
	}
	if tz == "" || tz == "UTC" {
		return "UTC"
	}
	return s.gt("survey.zone", "city", tz)
}

// requiredText is what a question left empty says: the publisher's own
// words, else polls' sentence for its type in the page's language.
func (s *site) requiredText(q polls.SurveyQuestion) string {
	if strings.TrimSpace(q.RequiredText) != "" {
		return q.RequiredText
	}
	switch q.Type {
	case polls.QuestionSingle, polls.QuestionMulti, polls.QuestionScale, polls.QuestionEmoji:
		return s.gt("survey.required." + q.Type)
	}
	return s.gt("survey.required.text")
}

// surveyLang picks a survey page's language: the email's the link came
// from (the contact's, for a link without one), else the visitor's.
func (h *Handler) surveyLang(w http.ResponseWriter, r *http.Request, s *site, token string) pageLang {
	for _, kind := range []linktoken.Kind{linktoken.Survey, linktoken.SurveyTest} {
		if c, ok := h.claims(s, token, kind); ok {
			return h.linkLang(w, r, s, c)
		}
	}
	return h.visitorLang(w, r, s, nil)
}

func (h *Handler) surveyView(s *site, p *polls.SurveyPage, token string) surveyBody {
	tz := p.Settings.Timezone
	if tz == "" {
		tz = s.ws.Timezone
	}
	loc := state.Location(tz)
	reg := s.lang().Region
	b := surveyBody{Kicker: s.gt("survey.kicker", "publisher", s.set.Branding.PublisherName), Name: p.Name, Description: p.Description, Back: s.links.Archive(),
		Footer: pollFooter{Privacy: s.set.Branding.PrivacyURL, Preferences: s.links.PreferencesRequest()}, Test: p.Answerer == polls.AnswererTest}
	action := s.links.Survey(p.Slug)
	if token != "" {
		action += "?t=" + url.QueryEscape(token)
	}
	b.Action, b.ChangeURL = action, action
	until, untilClock := "", ""
	if p.ClosesAt != nil {
		until = reg.ShortDate(p.ClosesAt.In(loc))
		untilClock = reg.Time(p.ClosesAt.In(loc), "")
	}
	switch {
	case p.Closed:
		b.State = "closed"
		b.Heading = p.Name
		b.Text = s.gt("survey.closed")
		if p.ClosedAt != nil {
			b.Text = s.gt("survey.closedOn", "date", reg.ShortDate(p.ClosedAt.In(loc)), "time", reg.Time(p.ClosedAt.In(loc), ""))
		}
		return b
	case p.Refused:
		b.State = "refused"
		b.Heading = s.gt("survey.refusedHeading", "publisher", s.set.Branding.PublisherName)
		b.Text = s.gt("survey.refusedText")
		b.Join = s.links.Subscribe("")
		return b
	}
	b.State = "open"
	var answering []string
	switch p.Answerer {
	case polls.AnswererContact:
		answering = append(answering, s.gt("survey.answeringAs", "email", p.Email))
	case polls.AnswererTest:
		answering = append(answering, s.gt("survey.testAnswers", "email", p.Email))
	}
	if until != "" {
		answering = append(answering, s.gt("survey.openUntil", "date", until, "time", untilClock, "zone", s.zoneCity(tz)))
	}
	b.Answering = strings.Join(answering, " · ")
	b.Submit = s.gt("survey.submit")
	if p.Settings.OneResponse && until != "" {
		b.Until = s.gt("survey.until", "date", until)
	}
	switch {
	case p.Saved:
		b.State = "done"
		b.DoneTitle = s.gt("survey.doneTitle")
		thanks := s.gt("survey.thanks")
		if p.FirstName != "" {
			thanks = s.gt("survey.thanksNamed", "name", p.FirstName)
		}
		switch {
		case p.Answerer == polls.AnswererTest:
			b.DoneText = s.gt("survey.doneTest")
		case p.Settings.OneResponse && until != "":
			b.DoneText = thanks + " " + s.gt("survey.doneChange", "date", until, "time", untilClock)
		default:
			b.DoneText = thanks + " " + strings.TrimSpace(p.Settings.Thanks)
		}
		if !p.Settings.OneResponse || p.Answerer != polls.AnswererContact {
			b.ChangeURL = ""
		}
		return b
	case p.CompletedAt != nil && p.Answerer == polls.AnswererContact:
		at := p.CompletedAt.In(loc)
		b.Intro = s.gt("survey.answeredOn", "date", reg.ShortDate(at), "time", reg.Time(at, ""))
		b.IntroTone = "neutral"
		b.Submit = s.gt("survey.saveChanges")
		b.Keep = action
		if strings.Contains(b.Keep, "?") {
			b.Keep += "&view=done"
		} else {
			b.Keep += "?view=done"
		}
	case p.FromPoll:
		b.Intro = s.gt("survey.fromPoll")
		b.IntroTone = "ok"
	}
	for i, q := range p.Questions {
		qv := surveyQuestionView{Key: q.Key, Number: i + 1, Type: q.Type, Text: q.Text, Description: q.Description, Required: q.Required,
			FromPoll: i == 0 && p.FromPoll}
		if p.Error(q.Key) != "" {
			qv.Error = s.requiredText(q)
		}
		mark := "none"
		switch {
		case !q.Required:
			mark = "optional"
		case q.Type == polls.QuestionMulti:
			mark = "multi"
		}
		qv.Label = s.gt("survey.question", "n", strconv.Itoa(i+1), "mark", mark)
		given := p.Answers[q.Key]
		picked := map[int]bool{}
		for _, k := range pickedOf(given) {
			picked[k] = true
		}
		switch q.Type {
		case polls.QuestionShort, polls.QuestionLong:
			qv.Text1, _ = given.(string)
			qv.Max = polls.MaxShortAnswer
			if q.Type == polls.QuestionLong {
				qv.Max = polls.MaxLongAnswer
			}
			qv.Count = s.gt("survey.count", "count", len([]rune(qv.Text1)), "max", qv.Max)
			qv.CountTpl = s.gt("survey.count", "count", "{n}", "max", qv.Max)
		default:
			for _, o := range polls.QuestionOptions(q) {
				ov := surveyOptionView{Value: strconv.Itoa(o.Key), Label: o.Label, Aria: o.Label, Checked: picked[o.Key]}
				switch q.Type {
				case polls.QuestionScale:
					ov.Aria = s.gt("poll.outOf10", "n", strconv.Itoa(o.Key))
				case polls.QuestionEmoji:
					faces := q.Faces
					if faces != 3 {
						faces = 5
					}
					ov.Label = s.faceLabel(faces, o.Key)
					ov.Aria = ov.Label
					ov.Face = s.links.Static(FacePath(faces, o.Key, accentHex(s)))
				}
				qv.Options = append(qv.Options, ov)
			}
			if q.Type == polls.QuestionScale {
				qv.Low, qv.High = firstOf(q.Low, s.gt("survey.notLikely")), firstOf(q.High, s.gt("survey.veryLikely"))
			}
		}
		b.Questions = append(b.Questions, qv)
	}
	if len(p.Errors) > 0 {
		b.ErrorTitle = s.gt("survey.errorsTitle", "count", len(p.Errors))
		for _, e := range p.Errors {
			b.Errors = append(b.Errors, surveyErrorView{Href: "#" + e.Key, Text: s.gt("survey.errorItem", "n", strconv.Itoa(e.Number), "text", e.Text)})
		}
	}
	return b
}

func firstOf(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// pickedOf reads a stored choice answer: a number or a list of numbers.
func pickedOf(v any) []int {
	switch x := v.(type) {
	case float64:
		return []int{int(x)}
	case int:
		return []int{x}
	case []int:
		return x
	case []any:
		out := []int{}
		for _, e := range x {
			if f, ok := e.(float64); ok {
				out = append(out, int(f))
			}
		}
		return out
	}
	return nil
}

func (h *Handler) renderSurvey(w http.ResponseWriter, r *http.Request, s *site, status int, b surveyBody) {
	title := b.Name
	if title == "" {
		title = b.Heading
	}
	v := h.view(s, title+" · "+s.set.Branding.PublisherName, b)
	h.setLang(v, r, s, s.page(), nil)
	v.Scripts = []string{s.links.Static("poll.js?v=" + h.version)}
	h.render(w, r, s, "survey", status, v, csp{}, true)
}

func (h *Handler) badSurveyLink(w http.ResponseWriter, r *http.Request, s *site) {
	h.badLink(w, r, s, "survey.kickerShort", "survey.badLink", func(pageLang) string { return s.gt("survey.back") })
}

func (h *Handler) surveyPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Polls == nil {
		h.badSurveyLink(w, r, s)
		return
	}
	token := r.URL.Query().Get("t")
	p, err := h.Polls.SurveyPageFor(r.Context(), s.ws.ID, chi.URLParam(r, "slug"), token)
	if errors.Is(err, polls.ErrBadLink) {
		h.badSurveyLink(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if r.URL.Query().Get("view") == "done" && p.CompletedAt != nil && !p.Closed {
		p.Saved = true
	}
	h.surveyLang(w, r, s, token)
	h.renderSurvey(w, r, s, http.StatusOK, h.surveyView(s, p, token))
}

func (h *Handler) surveyPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Polls == nil {
		h.badSurveyLink(w, r, s)
		return
	}
	if !h.parseForm(w, r, 128<<10) {
		http.Error(w, growText(h.visitorLang(nil, r, s, nil).Labels.Lang(), "survey.unreadable"), http.StatusBadRequest)
		return
	}
	token := r.URL.Query().Get("t")
	p, err := h.Polls.SubmitSurvey(r.Context(), s.ws.ID, chi.URLParam(r, "slug"), token, r.PostForm)
	if errors.Is(err, polls.ErrBadLink) {
		h.badSurveyLink(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	status := http.StatusOK
	if len(p.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	h.surveyLang(w, r, s, token)
	h.renderSurvey(w, r, s, status, h.surveyView(s, p, token))
}
