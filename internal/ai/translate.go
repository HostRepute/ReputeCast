package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// translateSegment is one piece of text sent for translation.
type translateSegment struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// translateTask is a translation's request, as data.
type translateTask struct {
	Source, SourceName, Target, TargetName string
	Subject, Preheader                     string
	Segments                               []translateSegment
	Chars                                  int
	Campaign                               string
}

// AITranslateInput asks for a translation of a campaign's source into a
// workspace language.
type AITranslateInput struct {
	CampaignID string `json:"campaignId" format:"uuid"`
	Language   string `json:"language" pattern:"^[a-z]{2,3}$"`
}

// AITranslationSegment is one piece of a block: its source and the suggestion.
type AITranslationSegment struct {
	Path []int  `json:"path"`
	Kind string `json:"kind" enum:"text,button,nav,image,social,table"`
	// Source and Suggestion are HTML for text, plain text for the rest
	// (an image's alt text).
	Source     string `json:"source"`
	Suggestion string `json:"suggestion"`
}

// AITranslationBlock is one block's suggestion.
type AITranslationBlock struct {
	Index    int                    `json:"index"`
	Name     string                 `json:"name"`
	Segments []AITranslationSegment `json:"segments"`
	// SourceText and SuggestionText read the block as text.
	SourceText     string `json:"sourceText"`
	SuggestionText string `json:"suggestionText"`
	Tokens         int    `json:"tokens"`
	TokensRestored int    `json:"tokensRestored"`
	Links          int    `json:"links"`
	LinksRestored  int    `json:"linksRestored"`
	// Buttons are links drawn as buttons (a poll's answers): their
	// addresses aren't part of the text and never change.
	Buttons int `json:"buttons"`
}

// AITranslation is a suggested translation of a language version.
type AITranslation struct {
	RequestID     string `json:"requestId,omitempty"`
	Language      string `json:"language"`
	LanguageName  string `json:"languageName"`
	Source        string `json:"source"`
	SourceName    string `json:"sourceName"`
	SourceVersion int    `json:"sourceVersion"`
	// Status is the version's status before accepting: missing, draft,
	// outdated or up_to_date.
	Status              string               `json:"status"`
	Subject             string               `json:"subject"`
	SubjectSuggestion   string               `json:"subjectSuggestion"`
	SubjectHeuristics   []string             `json:"subjectHeuristics"`
	Preheader           string               `json:"preheader"`
	PreheaderSuggestion string               `json:"preheaderSuggestion"`
	Blocks              []AITranslationBlock `json:"blocks"`
	Tokens              int                  `json:"tokens"`
	TokensRestored      int                  `json:"tokensRestored"`
	Links               int                  `json:"links"`
	LinksRestored       int                  `json:"linksRestored"`
	// Readers get the language once it is accepted.
	Readers *int64 `json:"readers,omitempty"`
	Sent    AISent `json:"sent"`
	Refusal string `json:"refusal,omitempty"`
}

func segID(block, seg int) string { return "b" + strconv.Itoa(block) + "." + strconv.Itoa(seg) }

// translatable is what a segment sends: a text's HTML, a button's label,
// an image's alt text.
func translatable(seg languages.BlockSegment) (string, bool) {
	switch seg.Kind {
	case "text", "table":
		if strings.TrimSpace(plain(seg.HTML)) == "" {
			return "", false
		}
		return strings.TrimSpace(seg.HTML), true
	case "button", "nav", "social":
		t := strings.TrimSpace(seg.HTML)
		if t == "" {
			t = strings.TrimSpace(seg.Text)
		}
		return t, t != ""
	case "image":
		return strings.TrimSpace(seg.Alt), strings.TrimSpace(seg.Alt) != ""
	}
	return "", false
}

// sourceOf reads a campaign's source content.
func (s *Service) sourceOf(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.AICampaignBriefRow, error) {
	row, err := s.Q.AICampaignBrief(ctx, store.AICampaignBriefParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return row, api.NotFound("Campaign")
	}
	return row, err
}

// versionOf finds a language among a campaign's versions.
func versionOf(langs languages.EmailLanguages, lang string) (languages.EmailLanguage, bool) {
	for _, v := range langs.Versions {
		if v.Language == lang {
			return v, true
		}
	}
	return languages.EmailLanguage{}, false
}

// Translate asks for a translation of the source, block by block, with its
// subject and preheader. Every token and link of the source must still be
// there: what the model changed is restored, and the blocks say so.
func (s *Service) Translate(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AITranslateInput) (AITranslation, error) {
	c, err := s.campaignFor(ctx, ws, p, in.CampaignID)
	if err != nil {
		return AITranslation{}, err
	}
	target := languages.Base(in.Language)
	if target == c.language {
		return AITranslation{}, invalid(ctx, "body.language", mSameLanguage, nil)
	}
	settings, err := s.Languages.Settings(ctx, s.Q, ws.ID)
	if err != nil {
		return AITranslation{}, err
	}
	if !settings.Offers(target) {
		return AITranslation{}, invalid(ctx, "body.language", mNotWorkspaceLanguage, map[string]any{"language": languages.NameOf(target)})
	}
	src, err := s.sourceOf(ctx, ws, c.id)
	if err != nil {
		return AITranslation{}, err
	}
	langs, err := s.Languages.CampaignLanguages(ctx, ws, p, c.id)
	if err != nil {
		return AITranslation{}, err
	}
	status := "missing"
	var readers *int64
	if v, ok := versionOf(langs, target); ok {
		status, readers = v.Status, v.Readers
	}
	blocks := languages.Blocks(src.Mjml, src.Html)
	t := translateTask{Source: c.language, SourceName: languageName(c.language), Target: target, TargetName: languageName(target),
		Subject: src.Subject, Preheader: src.Preheader, Segments: []translateSegment{}, Campaign: c.name}
	sources := map[string]string{}
	for _, b := range blocks {
		for i, seg := range b.Segments {
			if text, ok := translatable(seg); ok {
				id := segID(b.Index, i)
				t.Segments = append(t.Segments, translateSegment{ID: id, Text: text})
				sources[id] = text
				t.Chars += chars(text)
			}
		}
	}
	t.Chars += chars(src.Subject) + chars(src.Preheader)
	if len(t.Segments) == 0 && strings.TrimSpace(src.Subject) == "" {
		return AITranslation{}, invalid(ctx, "body.campaignId", mNoSourceText, nil)
	}
	if t.Chars > maxEmailChars {
		return AITranslation{}, invalid(ctx, "body.campaignId", mTooLong, map[string]any{"count": t.Chars, "max": maxEmailChars})
	}
	res, err := s.run(ctx, ws, p, task{feature: FeatureTranslate, action: target, targetKind: "campaign", targetID: &c.id, targetLabel: c.name, req: translatePrompt(t)})
	if err != nil {
		return AITranslation{}, err
	}
	out := AITranslation{RequestID: res.RequestID.String(), Language: target, LanguageName: languages.NameOf(target), Source: c.language, SourceName: languages.NameOf(c.language),
		SourceVersion: langs.SourceVersion, Status: status, Subject: src.Subject, Preheader: src.Preheader, Blocks: []AITranslationBlock{}, SubjectHeuristics: []string{},
		Readers: readers, Sent: res.sent(t.Chars), Refusal: res.Refusal}
	if res.Refusal != "" {
		return out, nil
	}
	var a struct {
		Subject   string             `json:"subject"`
		Preheader string             `json:"preheader"`
		Segments  []translateSegment `json:"segments"`
	}
	_ = json.Unmarshal(res.JSON, &a)
	got := map[string]string{}
	for _, seg := range a.Segments {
		got[seg.ID] = seg.Text
	}
	if len(got) == 0 && a.Subject == "" {
		return out, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, map[string]any{"provider": res.conn.providerName()})
	}
	subj, k := restore(src.Subject, plain(a.Subject))
	out.SubjectSuggestion, out.Tokens, out.TokensRestored = subj, k.Tokens, k.TokensRestored
	pre, k2 := restore(src.Preheader, plain(a.Preheader))
	out.PreheaderSuggestion = pre
	out.Tokens += k2.Tokens
	out.TokensRestored += k2.TokensRestored
	if out.SubjectSuggestion == "" {
		out.SubjectSuggestion = src.Subject
	}
	if out.PreheaderSuggestion == "" {
		out.PreheaderSuggestion = src.Preheader
	}
	out.SubjectHeuristics = contentcheck.SubjectHeuristicsIn(out.SubjectSuggestion, target)
	for _, b := range blocks {
		blk := AITranslationBlock{Index: b.Index, Name: b.Name, Segments: []AITranslationSegment{}}
		var srcText, sugText []string
		for i, seg := range b.Segments {
			if seg.Kind == "button" && seg.URL != "" {
				blk.Buttons++
			}
			id := segID(b.Index, i)
			source, ok := sources[id]
			if !ok {
				continue
			}
			sug := got[id]
			if seg.Kind == "text" || seg.Kind == "table" {
				sug = sanitize(sug)
			} else {
				sug = plain(sug)
			}
			if strings.TrimSpace(plain(sug)) == "" {
				sug = source
			}
			fixed, kept := restore(source, sug)
			if seg.Kind == "text" || seg.Kind == "table" {
				fixed = linkAttrs(source, tagLook(source, fixed))
			}
			blk.Tokens += kept.Tokens
			blk.TokensRestored += kept.TokensRestored
			blk.Links += kept.Links
			blk.LinksRestored += kept.LinksRestored
			blk.Segments = append(blk.Segments, AITranslationSegment{Path: seg.Path, Kind: seg.Kind, Source: source, Suggestion: fixed})
			srcText = append(srcText, plain(source))
			sugText = append(sugText, plain(fixed))
		}
		if len(blk.Segments) == 0 {
			continue
		}
		blk.SourceText, blk.SuggestionText = strings.Join(srcText, " · "), strings.Join(sugText, " · ")
		out.Tokens += blk.Tokens
		out.TokensRestored += blk.TokensRestored
		out.Links += blk.Links
		out.LinksRestored += blk.LinksRestored
		out.Blocks = append(out.Blocks, blk)
	}
	return out, nil
}

// AIAcceptedSegment is a segment's accepted text (the suggestion, or what
// the person made of it).
type AIAcceptedSegment struct {
	Path []int  `json:"path"`
	Kind string `json:"kind" enum:"text,button,nav,image,social,table"`
	Text string `json:"text" maxLength:"60000"`
}

// AIAcceptedBlock is an accepted block.
type AIAcceptedBlock struct {
	Index    int                 `json:"index" minimum:"0"`
	Segments []AIAcceptedSegment `json:"segments" maxItems:"200"`
	// Edited: the person changed the suggestion before accepting it.
	Edited bool `json:"edited,omitempty"`
}

// AITranslationAcceptInput accepts a translation, or some of its blocks.
type AITranslationAcceptInput struct {
	RequestID string `json:"requestId" format:"uuid"`
	// SourceVersion is the source the translation was made from.
	SourceVersion int `json:"sourceVersion" minimum:"0"`
	// Subject and Preheader are set when accepted.
	Subject   *string           `json:"subject,omitempty" maxLength:"998"`
	Preheader *string           `json:"preheader,omitempty" maxLength:"500"`
	Blocks    []AIAcceptedBlock `json:"blocks" maxItems:"200"`
	// Items is how many the suggestion had (its blocks, the subject and the
	// preheader): accepting all of them marks the version translated.
	Items  int  `json:"items" minimum:"1"`
	Edited bool `json:"edited,omitempty"`
}

// AITranslationAccepted is the language version written.
type AITranslationAccepted struct {
	Language string `json:"language"`
	// Status: up_to_date (every block accepted: "Translated by AI · not
	// reviewed") or draft (some blocks still to translate).
	Status   string `json:"status" enum:"up_to_date,draft"`
	Version  int    `json:"version"`
	Accepted int    `json:"accepted"`
	Items    int    `json:"items"`
	Model    string `json:"model"`
}

// AcceptTranslation writes the accepted blocks into the language version
// (added when missing), on the source's layout. Accepting every block marks
// it translated, "Translated by AI · not reviewed" until someone reviews
// or edits it; otherwise it stays a Draft. The save is a content edit in
// the version's history.
func (s *Service) AcceptTranslation(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in AITranslationAcceptInput) (AITranslationAccepted, error) {
	lang = languages.Base(lang)
	c, err := s.campaignFor(ctx, ws, p, id.String())
	if err != nil {
		return AITranslationAccepted{}, err
	}
	reqID, err := uuid.Parse(in.RequestID)
	if err != nil {
		return AITranslationAccepted{}, api.Invalid("body.requestId", "Name the suggestion.")
	}
	r, err := s.Q.AIRequest(ctx, store.AIRequestParams{ID: reqID, WorkspaceID: ws.ID})
	uid := actorID(p)
	if err != nil || uid == nil || r.UserID == nil || *r.UserID != *uid || r.Feature != FeatureTranslate || r.Action != lang || s.now().Sub(r.CreatedAt).Hours() > 24 {
		return AITranslationAccepted{}, problem(ctx, http.StatusConflict, "ai_suggestion_gone", mSuggestionGone, nil)
	}
	accepted := len(in.Blocks)
	if in.Subject != nil {
		accepted++
	}
	if in.Preheader != nil {
		accepted++
	}
	if accepted == 0 {
		return AITranslationAccepted{}, invalid(ctx, "body.blocks", mNothingAccepted, nil)
	}
	langs, err := s.Languages.CampaignLanguages(ctx, ws, p, c.id)
	if err != nil {
		return AITranslationAccepted{}, err
	}
	if langs.SourceVersion != in.SourceVersion {
		return AITranslationAccepted{}, problem(ctx, http.StatusConflict, "ai_source_changed", mSourceChanged, nil)
	}
	src, err := s.sourceOf(ctx, ws, c.id)
	if err != nil {
		return AITranslationAccepted{}, err
	}
	conn, err := s.load(ctx, s.Q, ws.ID)
	if err != nil {
		return AITranslationAccepted{}, err
	}
	model := r.Model
	if conn != nil && conn.Model == r.Model {
		model = conn.modelName()
	}
	v, exists := versionOf(langs, lang)
	if !exists || v.Status == "missing" {
		if _, err := s.Languages.AddCampaignLanguage(ctx, ws, p, c.id, languages.AddLanguageInput{Language: lang, Start: "copy"}); err != nil {
			return AITranslationAccepted{}, err
		}
		if langs, err = s.Languages.CampaignLanguages(ctx, ws, p, c.id); err != nil {
			return AITranslationAccepted{}, err
		}
		v, _ = versionOf(langs, lang)
	}
	content, err := (languages.Campaigns{S: s.Languages}).Content(ctx, ws, c.id, lang)
	if err != nil {
		return AITranslationAccepted{}, err
	}
	mj, err := applyBlocks(src.Mjml, content.MJML, in.Blocks)
	if err != nil {
		return AITranslationAccepted{}, api.Invalid("body.blocks", err.Error())
	}
	subject, preheader := content.Subject, content.Preheader
	if in.Subject != nil {
		subject = strings.TrimSpace(*in.Subject)
	}
	if in.Preheader != nil {
		preheader = strings.TrimSpace(*in.Preheader)
	}
	saved, err := (languages.Campaigns{S: s.Languages}).Save(ctx, ws, p, c.id, lang, campaigns.ComposerSaveContent{BaseVersion: content.Version, Subject: subject, Preheader: preheader,
		MJML: mj, Label: fmt.Sprintf("Translated with AI (%s)", model), Source: "manual"}, false)
	if err != nil {
		return AITranslationAccepted{}, err
	}
	out := AITranslationAccepted{Language: lang, Status: "draft", Version: saved.Version, Accepted: accepted, Items: max(in.Items, accepted), Model: model}
	all := accepted >= in.Items
	if all {
		if v.Status == "draft" || v.Status == "missing" || v.Status == "" {
			_, err = s.Languages.MarkCampaignLanguage(ctx, ws, p, c.id, lang, true, languages.MarkInput{})
		} else {
			_, err = s.Languages.MarkCampaignLanguage(ctx, ws, p, c.id, lang, false, languages.MarkInput{})
		}
		if err != nil {
			return AITranslationAccepted{}, err
		}
		if _, err := s.Q.AITranslationMark(ctx, store.AITranslationMarkParams{AiModel: model, WorkspaceID: ws.ID, CampaignID: c.id, Language: lang}); err != nil {
			return AITranslationAccepted{}, err
		}
		out.Status = "up_to_date"
	}
	outcome, detail := OutcomeAccepted, ""
	if in.Edited || slices.ContainsFunc(in.Blocks, func(b AIAcceptedBlock) bool { return b.Edited }) {
		outcome = OutcomeAcceptedEdits
	}
	if !all {
		detail = fmt.Sprintf("%d of %d", accepted, in.Items)
	}
	s.finish(ctx, ws, p, reqID, outcome, detail)
	return out, nil
}

// MarkReviewed ends "Translated by AI · not reviewed".
func (s *Service) MarkReviewed(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string) error {
	if _, err := s.campaignFor(ctx, ws, p, id.String()); err != nil {
		return err
	}
	now := s.now()
	_, err := s.Q.AITranslationReviewed(ctx, store.AITranslationReviewedParams{ReviewedBy: actorID(p), ReviewedAt: &now, WorkspaceID: ws.ID, CampaignID: id, Language: languages.Base(lang)})
	return err
}

// AITranslationMark is a language version written by an AI translation.
type AITranslationMark struct {
	Language string `json:"language"`
	Model    string `json:"model"`
	// Reviewed: someone marked it reviewed, or edited it since.
	Reviewed bool `json:"reviewed"`
}

// AITranslationMarks lists a campaign's AI-translated versions.
type AITranslationMarks struct {
	Items []AITranslationMark `json:"items"`
}

// TranslationMarks says which versions are "Translated by AI · not reviewed".
func (s *Service) TranslationMarks(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AITranslationMarks, error) {
	rows, err := s.Q.AITranslationMarks(ctx, store.AITranslationMarksParams{WorkspaceID: ws.ID, CampaignID: id})
	if err != nil {
		return AITranslationMarks{}, err
	}
	out := AITranslationMarks{Items: []AITranslationMark{}}
	for _, r := range rows {
		out.Items = append(out.Items, AITranslationMark{Language: r.Language, Model: r.AiModel, Reviewed: r.AiVersion == 0 || r.AiVersion != r.Version})
	}
	return out, nil
}

// edit replaces src[start:end].
type edit struct {
	start, end int
	text       string
}

// applyBlocks writes accepted blocks into a version: the source's layout
// with the accepted segments' texts. A block that wasn't accepted keeps the
// version's own block when the version has the same blocks, else the
// source's.
func applyBlocks(source, version string, accepted []AIAcceptedBlock) (string, error) {
	root, err := mjml.Parse(source)
	if err != nil {
		return "", fmt.Errorf("the source doesn't parse: %w", err)
	}
	body := mjml.Body(root)
	if body == nil {
		return "", errors.New("the source has no mj-body")
	}
	byIndex := map[int]AIAcceptedBlock{}
	for _, b := range accepted {
		if b.Index < 0 || b.Index >= len(body.Children) {
			return "", fmt.Errorf("block %d isn't in the email", b.Index)
		}
		byIndex[b.Index] = b
	}
	// The version's own blocks, when it is laid out like the source.
	var own []*mjml.Node
	if vroot, err := mjml.Parse(version); err == nil && strings.TrimSpace(version) != "" {
		if vb := mjml.Body(vroot); vb != nil && len(vb.Children) == len(body.Children) {
			own = vb.Children
		}
	}
	edits := []edit{}
	for i, n := range body.Children {
		b, ok := byIndex[i]
		if !ok {
			if own != nil {
				edits = append(edits, edit{start: n.Start, end: n.End, text: version[own[i].Start:own[i].End]})
			}
			continue
		}
		for _, seg := range b.Segments {
			node := n
			for _, j := range seg.Path {
				if j < 0 || j >= len(node.Children) {
					return "", fmt.Errorf("block %d has no element at %v", i, seg.Path)
				}
				node = node.Children[j]
			}
			e, ok := segmentEdit(source, node, seg)
			if !ok {
				return "", fmt.Errorf("block %d's element %s can't take that text", i, node.Tag)
			}
			edits = append(edits, e)
		}
	}
	return applyEdits(source, edits), nil
}

// segmentEdit replaces an element's text (its inner HTML) or, for an
// image, its alt text.
func segmentEdit(src string, n *mjml.Node, seg AIAcceptedSegment) (edit, bool) {
	open := openTagEnd(src, n.Start, n.End)
	if open < 0 {
		return edit{}, false
	}
	if n.Tag == "mj-image" || seg.Kind == "image" {
		return attrEdit(src, n.Start, open, "alt", seg.Text)
	}
	closeAt := strings.LastIndex(src[open:n.End], "</")
	if closeAt < 0 {
		return edit{}, false
	}
	text := seg.Text
	if seg.Kind != "text" && seg.Kind != "table" {
		text = html.EscapeString(plain(text))
	} else {
		// The tags and links keep the source's look: sanitize leaves only the tags and the links' addresses.
		inner := src[open+1 : open+closeAt]
		text = linkAttrs(inner, tagLook(inner, sanitize(text)))
	}
	return edit{start: open + 1, end: open + closeAt, text: text}, true
}

// openTagEnd is the offset of the '>' that ends the opening tag at start.
func openTagEnd(src string, start, end int) int {
	quote := byte(0)
	for i := start; i < end && i < len(src); i++ {
		ch := src[i]
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '>':
			return i
		}
	}
	return -1
}

// attrEdit sets an attribute on the opening tag src[start:open+1].
func attrEdit(src string, start, open int, name, value string) (edit, bool) {
	tagSrc := src[start:open]
	val := `"` + html.EscapeString(value) + `"`
	low := strings.ToLower(tagSrc)
	for idx := 0; ; {
		i := strings.Index(low[idx:], name)
		if i < 0 {
			break
		}
		i += idx
		before := byte(' ')
		if i > 0 {
			before = low[i-1]
		}
		rest := strings.TrimLeft(low[i+len(name):], " \t\n")
		if (before == ' ' || before == '\t' || before == '\n') && strings.HasPrefix(rest, "=") {
			eq := i + len(name) + strings.Index(low[i+len(name):], "=")
			v := eq + 1
			for v < len(tagSrc) && (tagSrc[v] == ' ' || tagSrc[v] == '\t' || tagSrc[v] == '\n') {
				v++
			}
			if v < len(tagSrc) && (tagSrc[v] == '"' || tagSrc[v] == '\'') {
				q := tagSrc[v]
				endQ := strings.IndexByte(tagSrc[v+1:], q)
				if endQ < 0 {
					return edit{}, false
				}
				return edit{start: start + v, end: start + v + 1 + endQ + 1, text: val}, true
			}
			stop := v
			for stop < len(tagSrc) && tagSrc[stop] != ' ' && tagSrc[stop] != '\t' && tagSrc[stop] != '\n' && tagSrc[stop] != '/' {
				stop++
			}
			return edit{start: start + v, end: start + stop, text: val}, true
		}
		idx = i + len(name)
	}
	at := open
	if open > start && src[open-1] == '/' {
		at = open - 1
	}
	return edit{start: at, end: at, text: " " + name + "=" + val}, true
}

// applyEdits applies non-overlapping edits.
func applyEdits(src string, edits []edit) string {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := src
	for _, e := range edits {
		out = out[:e.start] + e.text + out[e.end:]
	}
	return out
}
