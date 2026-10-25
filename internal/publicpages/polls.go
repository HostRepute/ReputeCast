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
//
// The pages read in the language of the email the link came from (the
// contact's, for a link that names no email), else the visitor's; their
// words are in the package's catalog (poll.*, survey.*, face.*).

import (
	"errors"
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

// dayIn is a day in the workspace's time zone, in the page's region
// format: "Mon 2 Jan", "Th 2, 2 thg 1".
func (h *Handler) dayIn(s *site, t time.Time) string {
	return s.lang().Region.ShortDate(t.In(state.Location(s.ws.Timezone)))
}

// faceWordKeys are the emoji scale's words, from unhappy to happy
// (polls.FaceWord's), as messages.
var faceWordKeys = map[int][]string{
	3: {"face.unhappy", "face.neutral", "face.happy"},
	5: {"face.veryUnhappy", "face.unhappy", "face.neutral", "face.happy", "face.veryHappy"},
}

// faceWord is the words of an emoji answer in the page's language ("very
// unhappy").
func (s *site) faceWord(faces, key int) string {
	w := faceWordKeys[faces]
	if key < 1 || key > len(w) {
		return ""
	}
	return s.gt(w[key-1])
}

// faceLabel is an emoji answer as screen readers and plain text say it:
// "1 of 5, very unhappy".
func (s *site) faceLabel(faces, key int) string {
	return s.gt("face.label", "key", key, "faces", faces, "word", s.faceWord(faces, key))
}

// answerKind is how a sentence names a poll's answers: a number on a 0–10
// scale, a face, or an answer.
func answerKind(kind string) string {
	switch kind {
	case polls.KindScale:
		return "scale"
	case polls.KindEmoji:
		return "emoji"
	}
	return "other"
}

// pollLang picks an answer page's language: the email's the link came
// from, else the visitor's.
func (h *Handler) pollLang(w http.ResponseWriter, r *http.Request, s *site, p *polls.AnswerPage) pageLang {
	return h.linkLang(w, r, s, linktoken.Claims{ContactID: p.ContactID, MessageID: p.MessageID})
}

// accentHex is the publisher's accent without "#", for the faces.
func accentHex(s *site) string {
	return strings.TrimPrefix(strings.ToLower(s.set.Branding.Palette.Accent), "#")
}

func answerLabel(s *site, p *polls.AnswerPage, key int) string {
	for _, a := range p.Answers {
		if a.Key == key {
			if p.Config.Kind == polls.KindEmoji {
				return s.faceWord(p.Config.Faces, key)
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
	b.Kicker = s.gt("poll.kicker", "campaign", p.CampaignName)
	switch p.Answerer {
	case polls.AnswererNonContact:
		b.Kicker = s.gt("poll.kickerArchive", "campaign", p.CampaignName)
	case polls.AnswererTest:
		b.Test = true
		b.Kicker = s.gt("poll.kickerTest", "campaign", p.CampaignName)
	}
	selected := p.Picked
	if p.Answer != nil && (posted || p.Closed || p.Refused) {
		selected = *p.Answer
	}
	kind := answerKind(c.Kind)
	for _, a := range p.Answers {
		v := pollAnswerView{Key: a.Key, Value: strconv.Itoa(a.Key), Label: a.Label, Aria: a.Label, Emoji: a.Emoji, Checked: a.Key == selected,
			Mine: p.Answer != nil && *p.Answer == a.Key}
		switch c.Kind {
		case polls.KindScale:
			v.Aria = s.gt("poll.outOf10", "n", strconv.Itoa(a.Key))
		case polls.KindEmoji:
			v.Label = s.faceLabel(c.Faces, a.Key)
			v.Aria = v.Label
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
		b.Status = s.gt("poll.closed", "date", when)
	case p.Refused:
		b.State, b.Tone = "refused", "neutral"
		b.Status = s.gt("poll.refused")
		b.Subscribe = s.links.Subscribe("")
	case posted && p.Saved && p.Previous != nil:
		b.State = "changed"
		b.Status = s.gt("poll.changed", "from", answerLabel(s, p, *p.Previous), "to", answerLabel(s, p, *p.Answer))
	case posted && p.Saved:
		b.State = "saved"
		switch p.Answerer {
		case polls.AnswererNonContact:
			b.Status = s.gt("poll.savedArchive")
		case polls.AnswererTest:
			b.Status = s.gt("poll.savedTest")
		default:
			b.Status = s.gt("poll.saved")
			if name := strings.TrimSpace(p.Name); name != "" {
				b.Status = s.gt("poll.savedNamed", "name", name)
			}
		}
		b.Thanks = strings.TrimSpace(c.Thanks)
	case p.Answer != nil && *p.Answer == p.Picked:
		b.State = "same"
	default:
		b.State, b.Tone = "pending", "neutral"
		b.Status = s.gt("poll.pending", "picked", answerLabel(s, p, p.Picked))
		if p.Answer != nil {
			b.Status = s.gt("poll.pendingChange", "picked", answerLabel(s, p, p.Picked), "answer", answerLabel(s, p, *p.Answer))
		}
	}
	b.CanAnswer = !p.Closed && !p.Refused
	if b.CanAnswer {
		b.Confirm = s.gt("poll.confirm")
		b.ConfirmNote = s.gt("poll.confirmNote", "kind", kind)
		if p.Answer != nil {
			b.Confirm = s.gt("poll.change")
			b.ConfirmNote = s.gt("poll.changeNote", "kind", kind)
		}
	}
	if p.Answer != nil && !p.Refused {
		b.Yours = s.gt("poll.yours", "answer", answerLabel(s, p, *p.Answer))
		if b.CanAnswer {
			b.Yours = s.gt("poll.yoursChangeable", "answer", answerLabel(s, p, *p.Answer), "kind", kind)
		}
	}
	b.ShowComment = p.Answer != nil && !p.Closed && (p.Answerer == polls.AnswererContact || p.Answerer == polls.AnswererTest)
	b.CommentSaved = p.Comment != ""
	if p.Results != nil {
		state := "open"
		switch {
		case p.Closed:
			state = "closed"
		case closes != "":
			state = "closes"
		}
		rv := &pollResultsView{Line: s.gt("poll.resultsLine", "count", p.Results.Total, "state", state, "date", closes)}
		for _, r := range p.Results.Rows {
			label := r.Label
			if c.Kind == polls.KindEmoji {
				label = s.faceLabel(c.Faces, r.Key)
			}
			rv.Rows = append(rv.Rows, pollResultRow{Label: label, Percent: r.Percent, Mine: p.Answer != nil && *p.Answer == r.Key})
		}
		b.Results = rv
	}
	switch p.Answerer {
	case polls.AnswererContact:
		if p.Email != "" && b.CanAnswer {
			b.Note = s.gt("poll.noteYours", "email", p.Email)
			if closes != "" {
				b.Note = s.gt("poll.noteYoursUntil", "email", p.Email, "date", closes)
			}
		}
	case polls.AnswererTest:
		b.Note = s.gt("poll.noteTest", "email", p.Email)
	case polls.AnswererNonContact:
		if !p.Refused {
			b.Subscribe = s.links.Subscribe("")
			b.SubscribeText = s.gt("poll.subscribeText", "publisher", s.set.Branding.PublisherName)
		}
	}
	if p.SurveyURL != "" {
		b.Next = p.SurveyURL
	}
	return b
}

func (h *Handler) visit(r *http.Request) polls.Visit {
	v := polls.Visit{UserAgent: r.UserAgent(), Method: r.Method, Script: r.Header.Get("X-RC-Script") == "1"}
	if ip := clientIP(r); ip != nil {
		v.IP = *ip
	}
	return v
}

// badLink is a broken or altered poll or survey link, in the visitor's
// language: nothing is recorded.
func (h *Handler) badLink(w http.ResponseWriter, r *http.Request, s *site, kickerKey, textKey string, button func(pageLang) string) {
	pl := h.visitorLang(w, r, s, nil)
	h.messageIn(w, r, s, pl, http.StatusNotFound, pageTitle(pl, s, "page.notFound.title"), messageBody{
		Kicker:  s.gt(kickerKey),
		Heading: s.gt("badLink.heading"),
		Text:    s.gt(textKey),
		Actions: []action{{Label: button(pl), Href: s.links.Archive()}},
	}, true)
}

// badPollLink is a broken or altered poll link: nothing is recorded.
func (h *Handler) badPollLink(w http.ResponseWriter, r *http.Request, s *site) {
	h.badLink(w, r, s, "poll.kickerShort", "poll.badLink", func(pl pageLang) string { return pl.Labels.Text("confirm.latest", nil) })
}

func (h *Handler) renderPoll(w http.ResponseWriter, r *http.Request, s *site, status int, b pollBody) {
	v := h.view(s, b.Question+" · "+s.set.Branding.PublisherName, b)
	h.setLang(v, r, s, s.page(), nil)
	v.Scripts = []string{s.links.Static("poll.js?v=" + h.version)}
	h.render(w, r, s, "poll", status, v, csp{}, true)
}

func pollAnswer(r *http.Request) (int, bool) {
	n, err := strconv.ParseInt(chi.URLParam(r, "answer"), 10, 32)
	return int(n), err == nil && n >= 0 && n <= 10
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
	h.pollLang(w, r, s, p)
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
		http.Error(w, growText(h.visitorLang(nil, r, s, nil).Labels.Lang(), "poll.answerUnreadable"), http.StatusBadRequest)
		return
	}
	answer := picked
	if v := strings.TrimSpace(r.PostForm.Get("answer")); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			h.badPollLink(w, r, s)
			return
		}
		answer = int(n)
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
		h.pollLang(w, r, s, pg)
		b := h.pollView(s, token, pg, false)
		b.State, b.Tone = "error", "error"
		b.Status = s.gt("poll.saveFailed")
		b.Confirm = s.gt("poll.tryAgain")
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
	h.pollLang(w, r, s, p)
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
		http.Error(w, growText(h.visitorLang(nil, r, s, nil).Labels.Lang(), "poll.commentUnreadable"), http.StatusBadRequest)
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
	h.pollLang(w, r, s, p)
	b := h.pollView(s, token, p, false)
	if p.Saved {
		b.State, b.Tone = "commented", "ok"
		b.Status = ""
		if p.Comment == "" {
			b.Status = s.gt("poll.commentRemoved")
		}
	}
	h.renderPoll(w, r, s, http.StatusOK, b)
}
