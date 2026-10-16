package publicpages

// The poll answer page and the survey page (design expansion, X11):
//
//	/p/{workspace}/poll/{token}/{answer}          answer page: GET shows it, POST records
//	/p/{workspace}/poll/{token}/{answer}/comment  "Anything to add?"
//	/p/{workspace}/s/{slug}                       survey page: GET shows it, POST saves
//
// Opening an answer page records nothing: its script posts the answer once
// the page has been visible for a moment, and without a script the person
// presses "Confirm my answer". Link scanners don't run scripts or submit
// forms; the ones that do are caught by the tracking classifier and counted
// as automated visits.

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/polls"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// pollAnswerView is one answer on the page.
type pollAnswerView struct {
	Key   int
	Value string
	Label string
	Aria  string
	Emoji string
	Face  string
	// Checked is the answer selected in the form; Mine the recorded one.
	Checked bool
	Mine    bool
}

type pollResultRow struct {
	Label   string
	Percent int
	Mine    bool
}

type pollResultsView struct {
	Line string
	Rows []pollResultRow
}

// pollBody is what poll.html renders.
type pollBody struct {
	State    string
	Kicker   string
	Test     bool
	Question string
	Kind     string
	Row      bool
	Answers  []pollAnswerView
	Low      string
	High     string
	// Status is the status region's sentence; Tone ok, neutral or error.
	Status  string
	Tone    string
	Thanks  string
	Yours   string
	Action  string
	Confirm string
	// ConfirmNote is said under the Confirm button without a script.
	ConfirmNote   string
	CanAnswer     bool
	ShowComment   bool
	CommentAction string
	Comment       string
	CommentSaved  bool
	EditComment   bool
	Results       *pollResultsView
	Note          string
	Next          string
	Subscribe     string
	SubscribeText string
	Latest        string
	Footer        pollFooter
}

// pollFooter is the usual footer: Privacy · Email preferences · Sent with ReputeCast.
type pollFooter struct {
	Privacy     string
	Preferences string
}

func (h *Handler) dayIn(s *site, t time.Time) string {
	return t.In(state.Location(s.ws.Timezone)).Format("Mon 2 Jan")
}

// accentHex is the publisher's accent without "#", for the faces.
func accentHex(s *site) string {
	return strings.TrimPrefix(strings.ToLower(s.set.Branding.Palette.Accent), "#")
}

func answerLabel(p *polls.AnswerPage, key int) string {
	for _, a := range p.Answers {
		if a.Key == key {
			if p.Config.Kind == polls.KindEmoji {
				return polls.FaceWord(p.Config.Faces, key)
			}
			return a.Label
		}
	}
	return strconv.Itoa(key)
}

// pollView turns the service's page into the template's.
func (h *Handler) pollView(s *site, token string, p *polls.AnswerPage, posted bool) pollBody {
	c := p.Config
	b := pollBody{Question: c.Question, Kind: c.Kind, Row: c.Layout == "row", Low: c.Low, High: c.High, Tone: "ok",
		Action: s.links.Poll(token) + "/" + strconv.Itoa(p.Picked), CommentAction: s.links.Poll(token) + "/" + strconv.Itoa(p.Picked) + "/comment",
		Comment: p.Comment, Latest: s.links.Archive(), Footer: pollFooter{Privacy: s.set.Branding.PrivacyURL, Preferences: s.links.PreferencesRequest()}}
	if p.Answerer == polls.AnswererContact && p.ContactID != uuid.Nil {
		b.Footer.Preferences = s.links.Preferences(h.sign(s, linktoken.Preferences, p.ContactID, p.MessageID))
	}
	b.Kicker = p.CampaignName + " · poll"
	switch p.Answerer {
	case polls.AnswererNonContact:
		b.Kicker = p.CampaignName + " · from the web archive"
	case polls.AnswererTest:
		b.Test = true
		b.Kicker = p.CampaignName + " · test email"
	}
	selected := p.Picked
	if p.Answer != nil && (posted || p.Closed || p.Refused) {
		selected = *p.Answer
	}
	unit := "answer"
	switch c.Kind {
	case polls.KindScale:
		unit = "number"
	case polls.KindEmoji:
		unit = "face"
	}
	for _, a := range p.Answers {
		v := pollAnswerView{Key: a.Key, Value: strconv.Itoa(a.Key), Label: a.Label, Aria: a.Label, Emoji: a.Emoji, Checked: a.Key == selected,
			Mine: p.Answer != nil && *p.Answer == a.Key}
		switch c.Kind {
		case polls.KindScale:
			v.Aria = fmt.Sprintf("%d out of 10", a.Key)
		case polls.KindEmoji:
			v.Face = s.links.Static(FacePath(c.Faces, a.Key, accentHex(s)))
		}
		b.Answers = append(b.Answers, v)
	}
	closes := ""
	if p.ClosesAt != nil {
		closes = h.dayIn(s, *p.ClosesAt)
	}
	switch {
	case p.Closed:
		b.State, b.Tone = "closed", "neutral"
		when := closes
		if p.ClosedAt != nil {
			when = h.dayIn(s, *p.ClosedAt)
		}
		b.Status = "This poll closed on " + when + ". It doesn't record answers anymore, and this visit changed nothing."
	case p.Refused:
		b.State, b.Tone = "refused", "neutral"
		b.Status = "This poll is for subscribers. Answers count only from the link in your own email. Subscribe, and you'll get the next poll."
		b.Subscribe = s.links.Subscribe("")
	case posted && p.Saved && p.Previous != nil:
		b.State = "changed"
		b.Status = fmt.Sprintf("Changed from %s to %s. Your new answer is saved; the latest one counts.", answerLabel(p, *p.Previous), answerLabel(p, *p.Answer))
	case posted && p.Saved:
		b.State = "saved"
		switch p.Answerer {
		case polls.AnswererNonContact:
			b.Status = "Thanks. Your answer is counted, but it isn't linked to you. Links on the web archive carry no name, so this answer is counted with the ones from people who aren't subscribers."
		case polls.AnswererTest:
			b.Status = "Saved as a test answer. Test answers aren't counted."
		default:
			b.Status = "Thanks. Your answer is saved."
			if name := strings.TrimSpace(p.Name); name != "" {
				b.Status = "Thanks, " + name + ". Your answer is saved."
			}
		}
		b.Thanks = strings.TrimSpace(c.Thanks)
	case p.Answer != nil && *p.Answer == p.Picked:
		b.State = "same"
	default:
		b.State, b.Tone = "pending", "neutral"
		b.Status = fmt.Sprintf("You picked %s in the email. Nothing is saved until you confirm it.", answerLabel(p, p.Picked))
		if p.Answer != nil {
			b.Status = fmt.Sprintf("You picked %s in the email. Your answer stays %s until you confirm the change.", answerLabel(p, p.Picked), answerLabel(p, *p.Answer))
		}
	}
	b.CanAnswer = !p.Closed && !p.Refused
	if b.CanAnswer {
		b.Confirm = "Confirm my answer"
		b.ConfirmNote = "Pick another " + unit + " first if you meant a different one."
		if p.Answer != nil {
			b.Confirm = "Change my answer"
			b.ConfirmNote = "Pick another " + unit + ", then change your answer."
		}
	}
	if p.Answer != nil && !p.Refused {
		b.Yours = "Your answer: " + answerLabel(p, *p.Answer) + "."
		if b.CanAnswer {
			b.Yours += " To change it, pick another " + unit + "."
		}
	}
	b.ShowComment = p.Answer != nil && !p.Closed && (p.Answerer == polls.AnswererContact || p.Answerer == polls.AnswererTest)
	b.CommentSaved = p.Comment != ""
	if p.Results != nil {
		rv := &pollResultsView{Line: fmt.Sprintf("%s %s", countWord(p.Results.Total), plural(p.Results.Total, "answer", "answers"))}
		switch {
		case p.Closed:
			rv.Line += " · closed"
		case closes != "":
			rv.Line += " · closes " + closes
		}
		for _, r := range p.Results.Rows {
			rv.Rows = append(rv.Rows, pollResultRow{Label: r.Label, Percent: r.Percent, Mine: p.Answer != nil && *p.Answer == r.Key})
		}
		b.Results = rv
	}
	switch p.Answerer {
	case polls.AnswererContact:
		if p.Email != "" && b.CanAnswer {
			b.Note = "This link is yours: it answers as " + p.Email + ", even from a forwarded email. The latest answer counts"
			if closes != "" {
				b.Note += ", until the poll closes on " + closes + "."
			} else {
				b.Note += " until the poll closes."
			}
		}
	case polls.AnswererTest:
		b.Note = "This is a test email: answers are saved as test answers of " + p.Email + " and never counted."
	case polls.AnswererNonContact:
		if !p.Refused {
			b.Subscribe = s.links.Subscribe("")
			b.SubscribeText = "Want " + s.set.Branding.PublisherName + " in your inbox?"
		}
	}
	if p.SurveyURL != "" {
		b.Next = p.SurveyURL
	}
	return b
}

func countWord(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (h *Handler) visit(r *http.Request) polls.Visit {
	v := polls.Visit{UserAgent: r.UserAgent(), Method: r.Method, Script: r.Header.Get("X-RC-Script") == "1"}
	if ip := clientIP(r); ip != nil {
		v.IP = *ip
	}
	return v
}

// badPollLink is a broken or altered link: nothing is recorded.
func (h *Handler) badPollLink(w http.ResponseWriter, r *http.Request, s *site) {
	h.message(w, r, s, http.StatusNotFound, "Link not valid · "+s.set.Branding.PublisherName, messageBody{
		Kicker:  "Poll",
		Heading: "This link doesn't work anymore",
		Text:    "It may have been cut short, or changed after it was sent. Open the poll again from your email. Nothing was recorded.",
		Actions: []action{{Label: "Read the latest issue", Href: s.links.Archive()}},
	}, true)
}

func (h *Handler) renderPoll(w http.ResponseWriter, r *http.Request, s *site, status int, b pollBody) {
	v := h.view(s, b.Question+" · "+s.set.Branding.PublisherName, b)
	v.Scripts = []string{s.links.Static("poll.js?v=" + h.version)}
	h.render(w, r, s, "poll", status, v, csp{}, true)
}

func pollAnswer(r *http.Request) (int, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, "answer"))
	return n, err == nil && n >= 0 && n <= 10
}

func (h *Handler) pollPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	picked, ok := pollAnswer(r)
	if !ok || h.Polls == nil {
		h.badPollLink(w, r, s)
		return
	}
	p, err := h.Polls.Page(r.Context(), s.ws.ID, token, picked, h.visit(r))
	if errors.Is(err, polls.ErrBadLink) {
		h.badPollLink(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	b := h.pollView(s, token, p, false)
	b.EditComment = r.URL.Query().Get("edit") == "comment"
	h.renderPoll(w, r, s, http.StatusOK, b)
}

func (h *Handler) pollPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	picked, ok := pollAnswer(r)
	if !ok || h.Polls == nil {
		h.badPollLink(w, r, s)
		return
	}
	if !h.parseForm(w, r, 8<<10) {
		http.Error(w, "That answer couldn't be read. Go back and pick it again.", http.StatusBadRequest)
		return
	}
	answer := picked
	if v := strings.TrimSpace(r.PostForm.Get("answer")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			h.badPollLink(w, r, s)
			return
		}
		answer = n
	}
	p, err := h.Polls.Record(r.Context(), s.ws.ID, token, answer, h.visit(r))
	if errors.Is(err, polls.ErrBadLink) {
		h.badPollLink(w, r, s)
		return
	}
	if err != nil {
		h.log().ErrorContext(r.Context(), "record a poll answer", "err", err)
		// Show the page again with "couldn't save" and Try again.
		pg, perr := h.Polls.Page(r.Context(), s.ws.ID, token, picked, polls.Visit{UserAgent: r.UserAgent(), Method: http.MethodGet, Script: true})
		if perr != nil {
			h.serverError(w, r, s, err)
			return
		}
		pg.Picked = answer
		b := h.pollView(s, token, pg, false)
		b.State, b.Tone = "error", "error"
		b.Status = "We couldn't save your answer. Nothing was recorded. Check your connection and try again."
		b.Confirm = "Try again"
		h.renderPoll(w, r, s, http.StatusServiceUnavailable, b)
		return
	}
	// Continuing to a survey: without a script, straight there. The link is
	// the workspace's own survey page, never one from the request.
	if p.SurveyURL != "" && r.Header.Get("X-RC-Script") != "1" && strings.HasPrefix(p.SurveyURL, s.canon.Origin()+"/") {
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, p.SurveyURL, http.StatusSeeOther) //nolint:gosec // G710: built from the workspace's page links, checked above
		return
	}
	if p.Automated {
		// A machine's post records nothing; the page looks as before.
		h.renderPoll(w, r, s, http.StatusOK, h.pollView(s, token, p, false))
		return
	}
	h.renderPoll(w, r, s, http.StatusOK, h.pollView(s, token, p, true))
}

func (h *Handler) pollComment(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	picked, ok := pollAnswer(r)
	if !ok || h.Polls == nil {
		h.badPollLink(w, r, s)
		return
	}
	if !h.parseForm(w, r, 16<<10) {
		http.Error(w, "That comment couldn't be read. Go back and send it again.", http.StatusBadRequest)
		return
	}
	text := r.PostForm.Get("comment")
	if utf8.RuneCountInString(text) > polls.MaxComment {
		text = string([]rune(text)[:polls.MaxComment])
	}
	p, err := h.Polls.Comment(r.Context(), s.ws.ID, token, text)
	if errors.Is(err, polls.ErrBadLink) {
		h.badPollLink(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	p.Picked = picked
	if p.Answer != nil {
		p.Picked = *p.Answer
	}
	b := h.pollView(s, token, p, false)
	if p.Saved {
		b.State, b.Tone = "commented", "ok"
		b.Status = ""
		if p.Comment == "" {
			b.Status = "Your comment is removed."
		}
	}
	h.renderPoll(w, r, s, http.StatusOK, b)
}
