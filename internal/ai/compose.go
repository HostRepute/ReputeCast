package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Size limits of what a person can send at once.
const (
	maxRewriteChars = 12000
	maxEmailChars   = 60000
	maxContextChars = 2000
)

// campaign is what the composer's features know about the email.
type campaign struct {
	id       uuid.UUID
	name     string
	language string
}

// campaignFor checks the campaign and that the person may edit it.
func (s *Service) campaignFor(ctx context.Context, ws *store.Workspace, p *authz.Principal, raw string) (campaign, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return campaign{}, api.Invalid("body.campaignId", "Name the campaign.")
	}
	if !p.Can(authz.CampaignsEdit) {
		return campaign{}, api.Forbidden(authz.CampaignsEdit, nil)
	}
	c, err := s.Q.AICampaignBrief(ctx, store.AICampaignBriefParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return campaign{}, api.NotFound("Campaign")
	}
	if err != nil {
		return campaign{}, err
	}
	out := campaign{id: c.ID, name: c.Name}
	if s.Languages != nil {
		if lang, err := (languages.Campaigns{S: s.Languages}).SourceLanguage(ctx, ws, id); err == nil {
			out.language = lang
		}
	}
	if out.language == "" {
		out.language = "en"
	}
	return out, nil
}

// languageName names a language for a prompt ("English").
func languageName(code string) string {
	if code == "" {
		return "English"
	}
	return languages.NameOf(languages.Base(code))
}

// emailText is an email's text without images, tokens as written: what
// subject ideas send.
func emailText(mjml string) string {
	var b strings.Builder
	for _, blk := range languages.Blocks(mjml, "") {
		for _, seg := range blk.Segments {
			switch seg.Kind {
			case "text", "button", "nav", "table":
				if t := strings.TrimSpace(seg.Text); t != "" {
					b.WriteString(t)
					b.WriteString("\n")
				}
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// subjectTask is subject ideas' request, as data.
type subjectTask struct {
	Subject, Preheader, Text, Language, LanguageName string
	Campaign                                         string
}

// AISubjectIdeasInput asks for subject ideas.
type AISubjectIdeasInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
	Subject    string `json:"subject" maxLength:"500"`
	Preheader  string `json:"preheader,omitempty" maxLength:"500"`
	// MJML is the email as it is on the canvas; its text goes, without images.
	MJML string `json:"mjml" maxLength:"2000000"`
	// Language is the version open ("" is the source).
	Language string `json:"language,omitempty" maxLength:"12"`
}

// AISubjectIdea is one idea with the subject checks.
type AISubjectIdea struct {
	Subject    string   `json:"subject"`
	Characters int      `json:"characters"`
	Verdict    string   `json:"verdict" enum:"pass,warn"`
	Heuristics []string `json:"heuristics"`
	// Phrase is the words that trip a heuristic ("you won't believe").
	Phrase string `json:"phrase,omitempty"`
}

// AISubjectIdeas are five ideas.
type AISubjectIdeas struct {
	RequestID string          `json:"requestId,omitempty"`
	Ideas     []AISubjectIdea `json:"ideas"`
	Sent      AISent          `json:"sent"`
	Language  string          `json:"language"`
	Refusal   string          `json:"refusal,omitempty"`
}

// checkSubject runs the subject heuristics on an idea.
func checkSubject(subject, lang string) AISubjectIdea {
	h := contentcheck.SubjectHeuristicsIn(subject, lang)
	idea := AISubjectIdea{Subject: subject, Characters: chars(subject), Verdict: "pass", Heuristics: h}
	if len(h) > 0 {
		idea.Verdict = "warn"
		idea.Phrase = contentcheck.SubjectHeuristicPhrase(subject, lang)
	}
	return idea
}

// SubjectIdeas asks for five subject lines and checks each one.
func (s *Service) SubjectIdeas(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AISubjectIdeasInput) (AISubjectIdeas, error) {
	c, err := s.campaignFor(ctx, ws, p, in.CampaignID)
	if err != nil {
		return AISubjectIdeas{}, err
	}
	lang := c.language
	if in.Language != "" {
		lang = languages.Base(in.Language)
	}
	body := emailText(in.MJML)
	if chars(body) > maxEmailChars {
		body = string([]rune(body)[:maxEmailChars])
	}
	t := subjectTask{Subject: in.Subject, Preheader: in.Preheader, Text: body, Language: lang, LanguageName: languageName(lang), Campaign: c.name}
	res, err := s.run(ctx, ws, p, task{feature: FeatureSubjectIdeas, targetKind: "campaign", targetID: &c.id, targetLabel: c.name, req: subjectPrompt(t)})
	if err != nil {
		return AISubjectIdeas{}, err
	}
	out := AISubjectIdeas{RequestID: res.RequestID.String(), Ideas: []AISubjectIdea{}, Sent: res.sent(chars(in.Subject) + chars(in.Preheader) + chars(body)), Language: lang, Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Subjects []string `json:"subjects"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	seen := map[string]bool{}
	for _, subj := range a.Subjects {
		subj = strings.TrimSpace(strings.Trim(strings.TrimSpace(subj), `"`))
		if subj == "" || seen[subj] || len(out.Ideas) == 5 {
			continue
		}
		seen[subj] = true
		out.Ideas = append(out.Ideas, checkSubject(subj, lang))
	}
	if len(out.Ideas) == 0 {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	return out, nil
}

// rewriteTask is a rewrite's request, as data.
type rewriteTask struct {
	Action, Instruction, HTML, Field, Language, LanguageName string
}

// AIRewriteInput asks for a rewrite of a text.
type AIRewriteInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
	Action     string `json:"action" enum:"shorten,friendlier,formal,direct,instruction"`
	// Instruction is what to change, for "With an instruction…".
	Instruction string `json:"instruction,omitempty" maxLength:"500"`
	// HTML is the selected text, or the whole paragraph, as the editor holds it.
	HTML string `json:"html" maxLength:"60000"`
	// Field is "subject" or "preheader" when those are rewritten.
	Field    string `json:"field,omitempty" enum:"subject,preheader"`
	Language string `json:"language,omitempty" maxLength:"12"`
}

// AIReputation is the answer to a question about sender reputation: the
// assistant doesn't look at those, and nothing was sent.
type AIReputation struct {
	// HostRepute: HostRepute is connected, so its assistant can answer.
	HostRepute bool `json:"hostRepute"`
}

// AIRewrite is a rewrite suggestion.
type AIRewrite struct {
	RequestID string `json:"requestId,omitempty"`
	// HTML is the suggestion, tokens and link addresses restored.
	HTML           string        `json:"html"`
	WordsBefore    int           `json:"wordsBefore"`
	WordsAfter     int           `json:"wordsAfter"`
	Tokens         []string      `json:"tokens" doc:"The variables of the tokens kept: first_name"`
	TokensRestored int           `json:"tokensRestored"`
	Links          int           `json:"links"`
	LinksRestored  int           `json:"linksRestored"`
	Sent           AISent        `json:"sent"`
	Refusal        string        `json:"refusal,omitempty"`
	Reputation     *AIReputation `json:"reputation,omitempty"`
}

// Rewrite asks for a shorter, friendlier, more formal or more direct text,
// or a rewrite as instructed. An instruction about sender reputation isn't
// sent: the answer points at HostRepute's assistant.
func (s *Service) Rewrite(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIRewriteInput) (AIRewrite, error) {
	c, err := s.campaignFor(ctx, ws, p, in.CampaignID)
	if err != nil {
		return AIRewrite{}, err
	}
	src := strings.TrimSpace(in.HTML)
	if plain(src) == "" {
		return AIRewrite{}, invalid(ctx, "body.html", mNothingToRewrite, nil)
	}
	if n := chars(src); n > maxRewriteChars {
		return AIRewrite{}, invalid(ctx, "body.html", mTooLong, map[string]any{"count": n, "max": maxRewriteChars})
	}
	if in.Action == "instruction" {
		if strings.TrimSpace(in.Instruction) == "" {
			return AIRewrite{}, invalid(ctx, "body.instruction", mInstructionNeeded, nil)
		}
		if asksAboutReputation(in.Instruction) {
			// Checked before anything is sent: whether HostRepute is
			// connected decides where the answer points.
			if _, err := s.usable(ctx, ws, FeatureRewrite); err != nil {
				return AIRewrite{}, err
			}
			return AIRewrite{HTML: "", Tokens: []string{}, Reputation: &AIReputation{HostRepute: s.hostReputeConnected(ctx, ws.ID)}}, nil
		}
	}
	lang := c.language
	if in.Language != "" {
		lang = languages.Base(in.Language)
	}
	t := rewriteTask{Action: in.Action, Instruction: strings.TrimSpace(in.Instruction), HTML: src, Field: in.Field, Language: lang, LanguageName: languageName(lang)}
	label := c.name
	if in.Field != "" {
		label = c.name + " " + in.Field
	}
	res, err := s.run(ctx, ws, p, task{feature: FeatureRewrite, action: in.Action, targetKind: "campaign", targetID: &c.id, targetLabel: label, req: rewritePrompt(t)})
	if err != nil {
		return AIRewrite{}, err
	}
	out := AIRewrite{RequestID: res.RequestID.String(), Tokens: tokenNames(tokens(src)), WordsBefore: words(src), Sent: res.sent(chars(src) + chars(t.Instruction)), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	got := sanitize(strings.TrimSpace(a.Text))
	if in.Field != "" {
		got = plain(got)
	}
	if plain(got) == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	got, kept := restore(src, got)
	out.HTML, out.WordsAfter, out.TokensRestored, out.Links, out.LinksRestored = got, words(got), kept.TokensRestored, kept.Links, kept.LinksRestored
	return out, nil
}

// hostReputeConnected reports whether the workspace has HostRepute.
func (s *Service) hostReputeConnected(ctx context.Context, ws uuid.UUID) bool {
	c, err := s.Q.GetHostReputeConnection(ctx, ws)
	return err == nil && c.Status != "revoked"
}

// altTask is an alt text request, as data.
type altTask struct {
	FileName, Context, Language, LanguageName string
}

// AIAltTextInput asks for an image's alt text.
type AIAltTextInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
	// Src is the image's address (the media library's, or any https URL).
	Src string `json:"src" maxLength:"2000"`
	// Context is the text around the image.
	Context  string `json:"context,omitempty" maxLength:"4000"`
	Language string `json:"language,omitempty" maxLength:"12"`
}

// AIAltText is an alt text suggestion with its checks.
type AIAltText struct {
	RequestID  string `json:"requestId,omitempty"`
	Text       string `json:"text"`
	Characters int    `json:"characters"`
	// FileName is the image's file name ("oncall-heatmap.png").
	FileName string `json:"fileName"`
	// Checks: under 150 characters, doesn't start with "Image of", more
	// than the file name.
	Short     bool   `json:"short"`
	NoImageOf bool   `json:"noImageOf"`
	SaysWhat  bool   `json:"saysWhat"`
	Sent      AISent `json:"sent"`
	Refusal   string `json:"refusal,omitempty"`
}

// AltText looks at an image, resized to 1,024 px, and the text around it.
func (s *Service) AltText(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAltTextInput) (AIAltText, error) {
	c, err := s.campaignFor(ctx, ws, p, in.CampaignID)
	if err != nil {
		return AIAltText{}, err
	}
	conn, err := s.usable(ctx, ws, FeatureAltText)
	if err != nil {
		return AIAltText{}, err
	}
	if strings.TrimSpace(in.Src) == "" {
		return AIAltText{}, invalid(ctx, "body.src", mImageMissing, nil)
	}
	name := imageFileName(in.Src)
	lang := c.language
	if in.Language != "" {
		lang = languages.Base(in.Language)
	}
	t := altTask{FileName: name, Context: clip(plain(in.Context), maxContextChars), Language: lang, LanguageName: languageName(lang)}
	var img *imageData
	if !conn.Demo {
		img, err = s.loadImage(ctx, ws, in.Src)
		if err != nil {
			return AIAltText{}, err
		}
	}
	res, err := s.send(ctx, ws, p, conn, task{feature: FeatureAltText, targetKind: "campaign", targetID: &c.id, targetLabel: c.name, req: altPrompt(t, img.part())})
	if err != nil {
		return AIAltText{}, err
	}
	out := AIAltText{RequestID: res.RequestID.String(), FileName: name, Sent: res.sent(chars(t.Context)), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Alt string `json:"alt"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	out.Text = strings.TrimSpace(plain(a.Alt))
	if out.Text == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	out.Characters = chars(out.Text)
	out.Short = out.Characters <= 150
	out.NoImageOf = !imageOfRe.MatchString(out.Text)
	base := strings.TrimSuffix(name, path.Ext(name))
	out.SaysWhat = !strings.EqualFold(strings.NewReplacer("-", " ", "_", " ").Replace(base), out.Text) && words(out.Text) >= 3
	return out, nil
}

// imageFileName is an image's file name from its address.
func imageFileName(src string) string {
	s := src
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	n := path.Base(s)
	if n == "." || n == "/" {
		return "image"
	}
	return n
}

// linkTask is an accessibility fix's request, as data.
type linkTask struct {
	Text, Href, Context, Language, LanguageName string
}

// AILinkTextInput asks for link text that says where a link goes.
type AILinkTextInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
	Text       string `json:"text" maxLength:"500"`
	Href       string `json:"href" maxLength:"2000"`
	// Context is the sentence around the link.
	Context  string `json:"context,omitempty" maxLength:"4000"`
	Language string `json:"language,omitempty" maxLength:"12"`
}

// AILinkText is a link text suggestion with its checks.
type AILinkText struct {
	RequestID string `json:"requestId,omitempty"`
	Text      string `json:"text"`
	// SaysWhere: the new text isn't one that says nothing; Href is the same
	// link, unchanged.
	SaysWhere bool   `json:"saysWhere"`
	Href      string `json:"href"`
	Sent      AISent `json:"sent"`
	Refusal   string `json:"refusal,omitempty"`
}

// LinkText suggests link text for an accessibility issue: the link text,
// its address and the sentence around it.
func (s *Service) LinkText(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AILinkTextInput) (AILinkText, error) {
	c, err := s.campaignFor(ctx, ws, p, in.CampaignID)
	if err != nil {
		return AILinkText{}, err
	}
	lang := c.language
	if in.Language != "" {
		lang = languages.Base(in.Language)
	}
	t := linkTask{Text: strings.TrimSpace(in.Text), Href: strings.TrimSpace(in.Href), Context: clip(plain(in.Context), maxContextChars), Language: lang, LanguageName: languageName(lang)}
	res, err := s.run(ctx, ws, p, task{feature: FeatureAccessibility, targetKind: "campaign", targetID: &c.id, targetLabel: c.name, req: linkPrompt(t)})
	if err != nil {
		return AILinkText{}, err
	}
	out := AILinkText{RequestID: res.RequestID.String(), Href: t.Href, Sent: res.sent(chars(t.Text) + chars(t.Href) + chars(t.Context)), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	out.Text = strings.TrimSpace(plain(a.Text))
	if out.Text == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	out.SaysWhere = !accessibility.SaysNothing(out.Text, lang) && !accessibility.BareURL(out.Text, t.Href)
	return out, nil
}

var (
	dropBlockRe = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed)\b.*?</(script|style|iframe|object|embed)\s*>`)
	anyTagRe    = regexp.MustCompile(`(?s)<(/?)([a-zA-Z][a-zA-Z0-9]*)\b([^>]*)>`)
	hrefAttrRe  = regexp.MustCompile(`(?is)\bhref\s*=\s*("[^"]*"|'[^']*')`)
)

// allowedTags are what a rewrite may keep: inline formatting and links.
var allowedTags = map[string]bool{"a": true, "b": true, "strong": true, "i": true, "em": true, "u": true, "br": true, "span": true, "p": true}

// sanitize keeps a suggestion's inline formatting and links, and drops
// every other tag and attribute.
func sanitize(s string) string {
	s = dropBlockRe.ReplaceAllString(s, "")
	return anyTagRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := anyTagRe.FindStringSubmatch(m)
		name := strings.ToLower(sub[2])
		if !allowedTags[name] {
			return ""
		}
		if sub[1] == "/" {
			return "</" + name + ">"
		}
		if name == "a" {
			if h := hrefAttrRe.FindStringSubmatch(sub[3]); h != nil {
				v := strings.Trim(h[1], `"'`)
				low := strings.ToLower(strings.TrimSpace(v))
				if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "mailto:") || strings.HasPrefix(low, "{{") || strings.HasPrefix(low, "#") {
					return `<a href="` + strings.ReplaceAll(v, `"`, "&quot;") + `">`
				}
			}
			return "<a>"
		}
		if name == "br" {
			return "<br>"
		}
		return "<" + name + ">"
	})
}
