// Package polls is the Poll block and surveys (design expansion, M21): a
// question in an email whose answers are plain links, the answer page that
// records the answer (never the click, so link scanners can't vote), the
// results in the campaign report, and hosted multi-question surveys built
// in Forms & pages.
//
// A poll's settings live in the email's MJML, in the poll section's rc-poll
// attribute, so they are versioned with the content. When a test or a send
// needs the poll's links, Sync copies them into polls; each recipient's
// answer link carries a signed token (linktoken.Poll) naming the poll, the
// message and the contact.
package polls

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// Poll kinds.
const (
	KindChoice = "choice"
	KindScale  = "scale"
	KindEmoji  = "emoji"
)

// Where an answer is saved on the contact.
const (
	SaveNone  = "none"
	SaveTag   = "tag"
	SaveField = "field"
)

// When a poll closes.
const (
	CloseDays   = "days"
	CloseDate   = "date"
	CloseManual = "manual"
)

// PollAnswerConfig is one answer of a Choice poll.
type PollAnswerConfig struct {
	Label string `json:"label"`
	// Emoji is shown before the label; Tag is added when Save is "tag".
	Emoji string `json:"emoji,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// PollClose says when the poll stops recording answers.
type PollClose struct {
	Mode string `json:"mode" enum:"days,date,manual"`
	Days int    `json:"days,omitempty"`
	// Date is "2026-10-28": the poll closes at the end of that day in the
	// workspace's time zone.
	Date string `json:"date,omitempty"`
}

// PollConfig is a Poll block's settings, as its rc-poll attribute stores them.
type PollConfig struct {
	ID       string             `json:"id"`
	Kind     string             `json:"kind" enum:"choice,scale,emoji"`
	Question string             `json:"question"`
	Answers  []PollAnswerConfig `json:"answers"`
	// Layout of Choice answers: stacked buttons or one row.
	Layout string    `json:"layout,omitempty" enum:"stacked,row"`
	Save   string    `json:"save" enum:"none,tag,field"`
	Field  string    `json:"field,omitempty"`
	Close  PollClose `json:"close"`
	// After answering: the thank-you message, or continue to a survey whose
	// first question the answer fills.
	After  string `json:"after" enum:"thanks,survey"`
	Thanks string `json:"thanks,omitempty"`
	Survey string `json:"survey,omitempty"`
	// NonContacts: count answers from people who aren't contacts apart, or
	// refuse them.
	NonContacts string `json:"nonContacts" enum:"count,refuse"`
	// Results shows the shares on the answer page after answering.
	Results bool `json:"results"`
	// NPS uses the Net Promoter Score wording on a 0–10 scale; Low and
	// High are its ends' words.
	NPS  bool   `json:"nps,omitempty"`
	Low  string `json:"low,omitempty"`
	High string `json:"high,omitempty"`
	// Faces of an emoji scale: three or five.
	Faces int `json:"faces,omitempty"`
}

func (c *PollConfig) normalize() {
	switch c.Kind {
	case KindScale, KindEmoji:
	default:
		c.Kind = KindChoice
	}
	if c.Layout != "row" {
		c.Layout = "stacked"
	}
	switch c.Save {
	case SaveTag, SaveField:
	default:
		c.Save = SaveNone
	}
	switch c.Close.Mode {
	case CloseDate, CloseManual:
	default:
		c.Close.Mode = CloseDays
		if c.Close.Days <= 0 {
			c.Close.Days = 7
		}
	}
	if c.After != "survey" {
		c.After = "thanks"
	}
	if c.NonContacts != "refuse" {
		c.NonContacts = "count"
	}
	if c.Kind == KindEmoji && c.Faces != 3 {
		c.Faces = 5
	}
	if c.Kind == KindScale {
		if c.Low == "" {
			c.Low = "Not likely"
		}
		if c.High == "" {
			c.High = "Very likely"
		}
	}
	if c.Answers == nil {
		c.Answers = []PollAnswerConfig{}
	}
}

// ParseConfig reads an rc-poll attribute value.
func ParseConfig(raw string) (PollConfig, error) {
	var c PollConfig
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &c); err != nil {
		return PollConfig{}, fmt.Errorf("polls: %w", err)
	}
	c.normalize()
	return c, nil
}

// Encode writes the settings as the attribute value.
func (c PollConfig) Encode() string {
	c.normalize()
	b, _ := json.Marshal(c)
	return string(b)
}

// Answer is one answer of a poll with its key: 1… for choices and faces,
// 0–10 for a scale.
type Answer struct {
	Key   int    `json:"key"`
	Label string `json:"label"`
	Emoji string `json:"emoji,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// faceWords are the emoji scale's words, from unhappy to happy.
var faceWords = map[int][]string{
	3: {"unhappy", "neutral", "happy"},
	5: {"very unhappy", "unhappy", "neutral", "happy", "very happy"},
}

// FaceWord is the words of an emoji answer ("very unhappy").
func FaceWord(faces, key int) string {
	w := faceWords[faces]
	if key < 1 || key > len(w) {
		return ""
	}
	return w[key-1]
}

// AnswerList is the answers a poll takes.
func (c PollConfig) AnswerList() []Answer {
	switch c.Kind {
	case KindScale:
		out := make([]Answer, 0, 11)
		for i := 0; i <= 10; i++ {
			out = append(out, Answer{Key: i, Label: strconv.Itoa(i)})
		}
		return out
	case KindEmoji:
		out := make([]Answer, 0, c.Faces)
		for i := 1; i <= c.Faces; i++ {
			// "1 of 5, very unhappy": what screen readers and plain text say.
			out = append(out, Answer{Key: i, Label: fmt.Sprintf("%d of %d, %s", i, c.Faces, FaceWord(c.Faces, i))})
		}
		return out
	}
	out := make([]Answer, 0, len(c.Answers))
	for i, a := range c.Answers {
		out = append(out, Answer{Key: i + 1, Label: strings.TrimSpace(a.Label), Emoji: a.Emoji, Tag: strings.TrimSpace(a.Tag)})
	}
	return out
}

// Problems are what keeps a poll from working, in English: a missing
// question, fewer than two or more than five answers, a tag or field
// missing where answers are saved.
func (c PollConfig) Problems() []string { return c.ProblemsIn(i18n.Default) }

// ProblemsIn are the poll's problems in a reader's language.
func (c PollConfig) ProblemsIn(l i18n.Lang) []string {
	out := []string{}
	for _, m := range c.problems() {
		out = append(out, m.Text(l, nil))
	}
	return out
}

// ProblemTexts are the poll's problems kept as messages, for text stored
// for later readers (a Send Readiness run).
func (c PollConfig) ProblemTexts() []i18n.Text {
	out := []i18n.Text{}
	for _, m := range c.problems() {
		out = append(out, m.Keep(nil))
	}
	return out
}

func (c PollConfig) problems() []i18n.Msg {
	out := []i18n.Msg{}
	if strings.TrimSpace(c.Question) == "" {
		out = append(out, mPollQuestion)
	}
	if c.Kind == KindChoice {
		n := 0
		for _, a := range c.Answers {
			if strings.TrimSpace(a.Label) != "" {
				n++
			}
		}
		if n < 2 {
			out = append(out, mPollTwoAnswers)
		}
		if len(c.Answers) > 5 {
			out = append(out, mPollFive)
		}
		if c.Save == SaveTag {
			for _, a := range c.Answers {
				if strings.TrimSpace(a.Tag) == "" {
					out = append(out, mPollTags)
					break
				}
			}
		}
	}
	if c.Save == SaveField && strings.TrimSpace(c.Field) == "" {
		out = append(out, mPollField)
	}
	if c.After == "survey" && c.Survey == "" {
		out = append(out, mPollSurvey)
	}
	return out
}

// VarName is the Liquid variable holding a recipient's answer link base:
// an answer's link is {{ rc_poll_<id> }}/<key>.
func VarName(id string) string { return "rc_poll_" + id }

// SurveyVar is the Liquid variable holding a recipient's link to a survey.
func SurveyVar(number int64) string { return "rc_survey_" + strconv.FormatInt(number, 10) }

var idRe = regexp.MustCompile(`^[a-z0-9]{1,24}$`)

// PollBlock is a Poll block found in an email.
type PollBlock struct {
	Config PollConfig
	// Path is the section's element path from mj-body.
	Path []int
}

// Read finds an email's Poll blocks (sections with rc-poll).
func Read(src string) []PollBlock {
	out := []PollBlock{}
	if !strings.Contains(src, mjml.AttrPoll) {
		return out
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return out
	}
	body := mjml.Body(root)
	if body == nil {
		return out
	}
	seen := map[string]bool{}
	body.Walk(func(n *mjml.Node, path []int) {
		raw, ok := n.Get(mjml.AttrPoll)
		if !ok {
			return
		}
		c, err := ParseConfig(raw)
		if err != nil || !idRe.MatchString(c.ID) || seen[c.ID] {
			return
		}
		seen[c.ID] = true
		out = append(out, PollBlock{Config: c, Path: append([]int(nil), path...)})
	})
	return out
}

// Styles of the generated Poll block: the email's accent and fonts.
const (
	pollAccent = "#0F766E"
	pollSerif  = "Georgia, 'Times New Roman', serif"
	pollSans   = "'IBM Plex Sans', Helvetica, Arial, sans-serif"
)

func attr(v string) string { return html.EscapeString(v) }

// Section writes a Poll block's MJML from its settings: the question, the
// answers as plain links (MJML buttons, or cells for a scale), the ends'
// words and the note. The composer writes the same structure; this one is
// for the API and the demo. faceURL gives an emoji face's image.
func Section(c PollConfig, faceURL func(faces, key int) string) string {
	c.normalize()
	var b strings.Builder
	b.WriteString(`<!-- rc:poll -->` + "\n")
	b.WriteString(`<mj-section rc-poll="` + attr(c.Encode()) + `" rc-name="Poll" background-color="#ffffff" padding="12px 24px 20px">` + "\n<mj-column>\n")
	b.WriteString(`<mj-text font-family="` + attr(pollSerif) + `" font-size="20px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 12px">` +
		html.EscapeString(c.Question) + "</mj-text>\n")
	link := func(key int) string { return "{{ " + VarName(c.ID) + " }}/" + strconv.Itoa(key) }
	switch c.Kind {
	case KindScale:
		var cells strings.Builder
		for i := 0; i <= 10; i++ {
			cells.WriteString(`<a href="` + link(i) + `" style="display:inline-block;width:40px;height:40px;margin:0 6px 6px 0;border:2px solid ` + pollAccent + `;border-radius:6px;` +
				`color:` + pollAccent + `;text-align:center;font-family:` + pollSans + `;font-size:14px;font-weight:600;line-height:40px;text-decoration:none">` + strconv.Itoa(i) + `</a>`)
		}
		b.WriteString(`<mj-text css-class="rc-poll-scale" padding="0px">` + `<div style="font-size:0;line-height:0">` + cells.String() + `</div>` +
			`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="font-family:` + pollSans + `;font-size:12px;color:#5B6472"><tr>` +
			`<td style="padding:4px 0 0">0 · ` + html.EscapeString(c.Low) + `</td><td align="right" style="padding:4px 0 0">10 · ` + html.EscapeString(c.High) + `</td></tr></table></mj-text>` + "\n")
	case KindEmoji:
		var cells strings.Builder
		for _, a := range c.AnswerList() {
			src := ""
			if faceURL != nil {
				src = faceURL(c.Faces, a.Key)
			}
			cells.WriteString(`<a href="` + link(a.Key) + `" style="display:inline-block;width:96px;height:48px;margin:0 8px 8px 0;border:2px solid ` + pollAccent + `;border-radius:6px;text-align:center;text-decoration:none">` +
				`<img src="` + attr(src) + `" alt="` + attr(a.Label) + `" width="28" height="28" style="display:inline-block;margin-top:10px;border:0"></a>`)
		}
		b.WriteString(`<mj-text css-class="rc-poll-emoji" padding="0px">` + `<div style="font-size:0;line-height:0">` + cells.String() + `</div>` +
			`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="font-family:` + pollSans + `;font-size:12px;color:#5B6472"><tr>` +
			`<td style="padding:2px 0 0">` + html.EscapeString(capitalize(FaceWord(c.Faces, 1))) + `</td><td align="right" style="padding:2px 0 0">` +
			html.EscapeString(capitalize(FaceWord(c.Faces, c.Faces))) + `</td></tr></table></mj-text>` + "\n")
	default:
		if c.Layout == "row" {
			var cells strings.Builder
			for _, a := range c.AnswerList() {
				label := a.Label
				if a.Emoji != "" {
					label = a.Emoji + " " + label
				}
				cells.WriteString(`<a href="` + link(a.Key) + `" style="display:inline-block;min-width:96px;margin:0 8px 8px 0;padding:11px 16px;border:2px solid ` + pollAccent + `;border-radius:6px;` +
					`color:` + pollAccent + `;text-align:center;font-family:` + pollSans + `;font-size:14px;font-weight:600;line-height:20px;text-decoration:none">` + html.EscapeString(label) + `</a>`)
			}
			b.WriteString(`<mj-text css-class="rc-poll-row" padding="0px">` + `<div style="font-size:0;line-height:0">` + cells.String() + `</div></mj-text>` + "\n")
		} else {
			for _, a := range c.AnswerList() {
				label := a.Label
				if a.Emoji != "" {
					label = a.Emoji + " " + label
				}
				b.WriteString(`<mj-button href="` + link(a.Key) + `" css-class="rc-poll-answer" width="100%" align="center" background-color="#ffffff" color="` + pollAccent +
					`" border="2px solid ` + pollAccent + `" border-radius="6px" font-family="` + attr(pollSans) + `" font-size="14px" font-weight="600" inner-padding="11px 16px" padding="0px 0px 8px">` +
					html.EscapeString(label) + "</mj-button>\n")
			}
		}
	}
	note := "One click opens a page that saves your answer."
	if c.Results {
		note += " You'll see the results there."
	}
	b.WriteString(`<mj-text font-family="` + attr(pollSans) + `" font-size="12px" line-height="1.5" color="#5B6472" padding="4px 0px 0px">` + html.EscapeString(note) + "</mj-text>\n")
	b.WriteString("</mj-column>\n</mj-section>")
	return b.String()
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
