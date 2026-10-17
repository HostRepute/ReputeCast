package publicpages

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

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
	Error   string
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

// zoneCity is "New York time" for America/New_York.
func zoneCity(tz string) string {
	if i := strings.LastIndexByte(tz, '/'); i >= 0 {
		return strings.ReplaceAll(tz[i+1:], "_", " ") + " time"
	}
	if tz == "" || tz == "UTC" {
		return "UTC"
	}
	return tz + " time"
}

func (h *Handler) surveyView(s *site, p *polls.SurveyPage, token string) surveyBody {
	tz := p.Settings.Timezone
	if tz == "" {
		tz = s.ws.Timezone
	}
	loc := state.Location(tz)
	b := surveyBody{Kicker: "Survey · " + s.set.Branding.PublisherName, Name: p.Name, Description: p.Description, Back: s.links.Archive(),
		Footer: pollFooter{Privacy: s.set.Branding.PrivacyURL, Preferences: s.links.PreferencesRequest()}, Test: p.Answerer == polls.AnswererTest}
	action := s.links.Survey(p.Slug)
	if token != "" {
		action += "?t=" + url.QueryEscape(token)
	}
	b.Action, b.ChangeURL = action, action
	until, untilClock := "", ""
	if p.ClosesAt != nil {
		until = p.ClosesAt.In(loc).Format("Mon 2 Jan")
		untilClock = p.ClosesAt.In(loc).Format("15:04")
	}
	switch {
	case p.Closed:
		b.State = "closed"
		b.Heading = p.Name
		when := ""
		if p.ClosedAt != nil {
			when = " on " + p.ClosedAt.In(loc).Format("Mon 2 Jan") + " at " + p.ClosedAt.In(loc).Format("15:04")
		}
		b.Text = "This survey closed" + when + ". It doesn't take answers anymore. Answers you sent before it closed are kept."
		return b
	case p.Refused:
		b.State = "refused"
		b.Heading = "This survey is for " + s.set.Branding.PublisherName + "'s subscribers"
		b.Text = "Open it from the email we sent you, so your answers are linked to you. A copied or shared link can't take answers."
		b.Join = s.links.Subscribe("")
		return b
	}
	b.State = "open"
	switch p.Answerer {
	case polls.AnswererContact:
		b.Answering = "Answering as " + p.Email
	case polls.AnswererTest:
		b.Answering = "Test answers of " + p.Email + " · never counted"
	}
	if until != "" {
		if b.Answering != "" {
			b.Answering += " · "
		}
		b.Answering += fmt.Sprintf("open until %s, %s %s", until, untilClock, zoneCity(tz))
	}
	b.Submit = "Send my answers"
	if p.Settings.OneResponse && until != "" {
		b.Until = "You can change them until " + until + "."
	}
	switch {
	case p.Saved:
		b.State = "done"
		b.DoneTitle = "Your answers are saved"
		thanks := "Thanks."
		if p.FirstName != "" {
			thanks = "Thanks, " + p.FirstName + "."
		}
		switch {
		case p.Answerer == polls.AnswererTest:
			b.DoneText = "They're saved as a test response, which is never counted."
		case p.Settings.OneResponse && until != "":
			b.DoneText = thanks + " You can change them until " + until + " at " + untilClock + "; your newest answers count."
		default:
			b.DoneText = thanks + " " + strings.TrimSpace(p.Settings.Thanks)
		}
		if !p.Settings.OneResponse || p.Answerer != polls.AnswererContact {
			b.ChangeURL = ""
		}
		return b
	case p.CompletedAt != nil && p.Answerer == polls.AnswererContact:
		at := p.CompletedAt.In(loc)
		b.Intro = fmt.Sprintf("You answered on %s at %s. Change anything and save again. Your newest answers replace the old ones.", at.Format("Mon 2 Jan"), at.Format("15:04"))
		b.IntroTone = "neutral"
		b.Submit = "Save changes"
		b.Keep = action
		if strings.Contains(b.Keep, "?") {
			b.Keep += "&view=done"
		} else {
			b.Keep += "?view=done"
		}
	case p.FromPoll:
		b.Intro = "We kept your answer to question 1 from the email. Change it if you like."
		b.IntroTone = "ok"
	}
	for i, q := range p.Questions {
		qv := surveyQuestionView{Key: q.Key, Number: i + 1, Type: q.Type, Text: q.Text, Description: q.Description, Required: q.Required, Error: p.Error(q.Key),
			FromPoll: i == 0 && p.FromPoll}
		qv.Label = "Question " + strconv.Itoa(i+1)
		switch {
		case !q.Required:
			qv.Label += " · optional"
		case q.Type == polls.QuestionMulti:
			qv.Label += " · pick any"
		}
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
			qv.Count = fmt.Sprintf("%d of %s characters", len([]rune(qv.Text1)), countWord(int64(qv.Max)))
		default:
			for _, o := range polls.QuestionOptions(q) {
				ov := surveyOptionView{Value: strconv.Itoa(o.Key), Label: o.Label, Aria: o.Label, Checked: picked[o.Key]}
				switch q.Type {
				case polls.QuestionScale:
					ov.Aria = fmt.Sprintf("%d out of 10", o.Key)
				case polls.QuestionEmoji:
					faces := q.Faces
					if faces != 3 {
						faces = 5
					}
					ov.Face = s.links.Static(FacePath(faces, o.Key, accentHex(s)))
				}
				qv.Options = append(qv.Options, ov)
			}
			if q.Type == polls.QuestionScale {
				qv.Low, qv.High = firstOf(q.Low, "Not likely"), firstOf(q.High, "Very likely")
			}
		}
		b.Questions = append(b.Questions, qv)
	}
	if len(p.Errors) > 0 {
		b.ErrorTitle = fmt.Sprintf("%d %s an answer", len(p.Errors), plural(int64(len(p.Errors)), "question needs", "questions need"))
		for _, e := range p.Errors {
			b.Errors = append(b.Errors, surveyErrorView{Href: "#" + e.Key, Text: fmt.Sprintf("Question %d: %s", e.Number, e.Text)})
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
	v.Scripts = []string{s.links.Static("poll.js?v=" + h.version)}
	h.render(w, r, s, "survey", status, v, csp{}, true)
}

func (h *Handler) badSurveyLink(w http.ResponseWriter, r *http.Request, s *site) {
	h.pollMessage(w, r, s, http.StatusNotFound, "Link not valid · "+s.set.Branding.PublisherName, messageBody{
		Kicker:  "Survey",
		Heading: "This link doesn't work anymore",
		Text:    "It may have been cut short, or changed after it was sent, or the survey was removed. Open it again from your email. Nothing was recorded.",
		Actions: []action{{Label: "Back to the newsletter", Href: s.links.Archive()}},
	}, true)
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
		http.Error(w, "Those answers couldn't be read. Go back and send them again.", http.StatusBadRequest)
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
	h.renderSurvey(w, r, s, status, h.surveyView(s, p, token))
}
