package visibility

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// VisibilityHiddenBlock is a block a contact doesn't get, and why.
type VisibilityHiddenBlock struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Why is "Sam Rivera isn't on Beta Program".
	Why string `json:"why"`
}

// VisibilityPreview is the version one contact gets.
type VisibilityPreview struct {
	ContactID string `json:"contactId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	// Hidden are the blocks the contact doesn't get; Shown the ids of those
	// they do.
	Hidden []VisibilityHiddenBlock `json:"hidden"`
	Shown  []string                `json:"shown"`
	// Version is the version's number in the versions table (0 when the
	// audience isn't counted), Versions how many there are, Recipients
	// how many get this one.
	Version    int   `json:"version"`
	Versions   int   `json:"versions"`
	Recipients int64 `json:"recipients"`
	// Shows names the conditional blocks it shows ("" for none of them).
	Shows string `json:"shows"`
	// Empty is true when nothing is left between the header and the footer.
	Empty bool `json:"empty"`
	// Traits say what the rules read about the contact ("Active", "plan
	// team", "not on Beta Program").
	Traits []string `json:"traits"`
	// InAudience is false for a contact the campaign wouldn't reach.
	InAudience bool `json:"inAudience"`
}

// PreviewInput asks for a contact's version; MJML carries the composer's
// unsaved rules.
type PreviewInput struct {
	ContactID string `json:"contactId" format:"uuid"`
	MJML      string `json:"mjml,omitempty" maxLength:"2000000"`
	HTML      string `json:"html,omitempty" maxLength:"2500000"`
}

// PreviewAs decides a campaign's conditional blocks for one contact.
func (s *Service) PreviewAs(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, in PreviewInput) (VisibilityPreview, error) {
	cid, err := uuid.Parse(in.ContactID)
	if err != nil {
		return VisibilityPreview{}, api.Invalid("body.contactId", mProblemNoContact.Text(lang(ctx), nil))
	}
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return VisibilityPreview{}, api.MapError(err, "Campaign")
	}
	src, html := c.Mjml, c.Html
	if in.MJML != "" {
		src, html = in.MJML, in.HTML
	}
	p, _, err := s.preview(ctx, ws, campaignID, c.Audience, src, html, cid)
	return p, err
}

// preview decides the blocks of src for a contact, and finds the version
// in the counted versions.
func (s *Service) preview(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, audience []byte, src, html string, contactID uuid.UUID) (VisibilityPreview, Decision, error) {
	out := VisibilityPreview{ContactID: contactID.String(), Hidden: []VisibilityHiddenBlock{}, Shown: []string{}, Traits: []string{}}
	e := Read(src)
	n, err := LoadNames(ctx, s.Q, ws.ID)
	if err != nil {
		return out, Decision{}, err
	}
	segs, err := LoadSegments(ctx, s.Segments, ws.ID, e.Segments(), n)
	if err != nil {
		return out, Decision{}, err
	}
	facts, err := LoadFacts(ctx, s.Q, ws.ID, []uuid.UUID{contactID}, segs, e)
	if err != nil {
		return out, Decision{}, err
	}
	f, ok := facts[contactID]
	if !ok {
		return out, Decision{}, api.NotFound("Contact")
	}
	out.Name, out.Email = f.Name(), f.Email
	d := Decide(e, f, n)
	names := []string{}
	for _, b := range e.Blocks {
		if d.Hidden[b.ID] {
			// A block inside a hidden one isn't named again.
			if b.Parent != "" && d.Hidden[b.Parent] {
				continue
			}
			out.Hidden = append(out.Hidden, VisibilityHiddenBlock{ID: b.ID, Name: b.Name, Why: Why(b.Rule, d.Matched[b.ID], f, n)})
			continue
		}
		out.Shown = append(out.Shown, b.ID)
		if b.Parent == "" || d.Hidden[b.Parent] {
			names = append(names, b.Name)
		}
	}
	out.Shows = strings.Join(names, ", ")
	out.Empty = e.Empty(d.Hidden)
	out.Traits = traits(e, f, n)
	for _, id := range ParseAudience(audience).Lists {
		if f.Lists[id] {
			out.InAudience = true
		}
	}
	if len(e.Blocks) > 0 && !ParseAudience(audience).Empty() {
		if res, err := s.tally(ctx, ws, campaignID, audience, e, n, false); err == nil {
			vs := Versions(e, res.Tally, "")
			out.Versions = len(vs)
			key := d.Hidden.String()
			for _, v := range vs {
				if v.key() == key {
					out.Version, out.Recipients = v.Number, v.Recipients
				}
			}
		} else {
			s.log().WarnContext(ctx, "count versions for a preview", "campaign", campaignID, "err", err)
		}
	}
	return out, d, nil
}

// traits says what the rules read about a contact, once per thing read.
func traits(e Email, f *Facts, n *Names) []string {
	out := []string{}
	seen := map[string]bool{}
	add := func(k, s string) {
		if s != "" && !seen[k] {
			seen[k] = true
			out = append(out, s)
		}
	}
	for _, b := range e.Blocks {
		for _, c := range b.Rule.Conditions {
			switch c.Kind {
			case KindTier:
				add("tier", tierName(n.Lang, f.Tier))
			case KindField:
				v := fieldText(f.Attributes[c.Field])
				if v == "" {
					add("field:"+c.Field, mTraitNoField.Text(n.Lang, i18n.Args{"field": c.Field}))
				} else {
					add("field:"+c.Field, mTraitField.Text(n.Lang, i18n.Args{"field": c.Field, "value": v}))
				}
			case KindList:
				name := n.list(c.Value, c.Label)
				if f.Lists[c.Value] {
					add("list:"+c.Value, mTraitOnList.Text(n.Lang, i18n.Args{"list": name}))
				} else {
					add("list:"+c.Value, mTraitNotOnList.Text(n.Lang, i18n.Args{"list": name}))
				}
			case KindSegment:
				name := n.segment(c.Value, c.Label)
				if f.Segments[c.Value] {
					add("segment:"+c.Value, mTraitInSegment.Text(n.Lang, i18n.Args{"segment": name}))
				} else {
					add("segment:"+c.Value, mTraitNotInSegment.Text(n.Lang, i18n.Args{"segment": name}))
				}
			case KindTag:
				if f.Tags[c.Value] {
					add("tag:"+c.Value, mTraitTag.Text(n.Lang, i18n.Args{"tag": c.Value}))
				} else {
					add("tag:"+c.Value, mTraitNoTag.Text(n.Lang, i18n.Args{"tag": c.Value}))
				}
			case KindLanguage:
				add("language", languageName(f.Language))
			case KindScore, KindScoreBand:
				add("score", scoreTrait(f, n))
			}
		}
	}
	return out
}

// VisibilityPreviewContact is a contact to preview as: one per version
// (the most common first), then one per other language.
type VisibilityPreviewContact struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Language string `json:"language"`
	// Trait is what sets them apart ("English", "Beta Program").
	Trait string `json:"trait"`
	// Version is the version they get.
	Version int `json:"version"`
}

// PreviewContacts picks contacts who get each version, so a person can
// preview them all from the canvas.
func (s *Service) PreviewContacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) ([]VisibilityPreviewContact, error) {
	out := []VisibilityPreviewContact{}
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return out, api.MapError(err, "Campaign")
	}
	e := Read(c.Mjml)
	a := ParseAudience(c.Audience)
	if len(e.Blocks) == 0 || a.Empty() {
		return out, nil
	}
	n, err := LoadNames(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	segs, err := LoadSegments(ctx, s.Segments, ws.ID, e.Segments(), n)
	if err != nil {
		return out, err
	}
	recipients, err := FinalRecipients(ctx, s.Q, s.Segments, ws.ID, a, segs)
	if err != nil {
		return out, err
	}
	byID := make(map[uuid.UUID]*Facts, len(recipients))
	for _, f := range recipients {
		byID[f.ID] = f
	}
	if err := loadExtra(ctx, s.Q, ws.ID, byID, e); err != nil {
		return out, err
	}
	res, err := s.tally(ctx, ws, campaignID, c.Audience, e, n, false)
	if err != nil {
		return out, err
	}
	vs := Versions(e, res.Tally, "")
	numbers := map[string]int{}
	for _, v := range vs {
		numbers[v.key()] = v.Number
	}
	// Prefer contacts with a name, the way the composer's preview does.
	sort.SliceStable(recipients, func(i, j int) bool {
		return (recipients[i].FirstName != "") && (recipients[j].FirstName == "")
	})
	picked := map[int]bool{}
	langs := map[string]bool{}
	for _, f := range recipients {
		d := Decide(e, f, n)
		num := numbers[d.Hidden.String()]
		if num == 0 || picked[num] || len(out) >= 12 {
			continue
		}
		picked[num] = true
		trait := languageName(f.Language)
		for _, b := range e.Blocks {
			if !d.Hidden[b.ID] {
				trait = blockTrait(b, f, n)
				break
			}
		}
		if trait == "" {
			trait = languageName("en")
		}
		langs[baseLanguage(f.Language)] = true
		out = append(out, VisibilityPreviewContact{ID: f.ID.String(), Name: f.Name(), Email: f.Email, Language: f.Language, Trait: trait, Version: num})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	// One contact per other language, for emails in several languages.
	for _, f := range recipients {
		l := baseLanguage(f.Language)
		if l == "" || langs[l] || len(out) >= 16 {
			continue
		}
		langs[l] = true
		d := Decide(e, f, n)
		out = append(out, VisibilityPreviewContact{ID: f.ID.String(), Name: f.Name(), Email: f.Email, Language: f.Language, Trait: languageName(f.Language), Version: numbers[d.Hidden.String()]})
	}
	return out, nil
}

// blockTrait is the short name of what a shown block's rule reads: the
// list, segment, tier or field it checks ("Beta Program"), or the contact's
// own score ("score 52 · Hot").
func blockTrait(b Block, f *Facts, n *Names) string {
	if len(b.Rule.Conditions) == 0 {
		return b.Name
	}
	c := b.Rule.Conditions[0]
	switch c.Kind {
	case KindList:
		return n.list(c.Value, c.Label)
	case KindSegment:
		return n.segment(c.Value, c.Label)
	case KindTier:
		return tierName(n.Lang, c.Value)
	case KindLanguage:
		return languageName(c.Value)
	case KindTag:
		return c.Value
	case KindScore, KindScoreBand:
		return scoreTrait(f, n)
	}
	return b.Name
}

// PreviewHTML is the email a contact gets, for the composer's preview: the
// hidden blocks cut out of html, or, with showWhy, each replaced by a line
// saying why. src is the source's MJML (its rules decide every language
// version's blocks).
func (s *Service) PreviewHTML(ctx context.Context, ws *store.Workspace, campaignID, contactID uuid.UUID, src, html string, showWhy bool) (string, *VisibilityPreview, error) {
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return html, nil, api.MapError(err, "Campaign")
	}
	if src == "" {
		src = c.Mjml
	}
	if !strings.Contains(src, "rc-visible") {
		return html, nil, nil
	}
	if contactID == uuid.Nil {
		// A sample contact: no facts, so every rule reads an empty contact.
		e := Read(src)
		n, err := LoadNames(ctx, s.Q, ws.ID)
		if err != nil {
			return html, nil, err
		}
		d := Decide(e, &Facts{Lists: map[string]bool{}, Segments: map[string]bool{}, Tags: map[string]bool{}, Attributes: map[string]any{}, Extra: map[string]any{}}, n)
		return Apply(html, d.Hidden), nil, nil
	}
	p, d, err := s.preview(ctx, ws, campaignID, c.Audience, src, html, contactID)
	if err != nil {
		return html, nil, err
	}
	if !showWhy {
		return Apply(html, d.Hidden), &p, nil
	}
	whys := map[string]string{}
	for _, h := range p.Hidden {
		whys[h.ID] = mWhyHidden.Text(lang(ctx), i18n.Args{"block": h.Name, "why": h.Why})
	}
	out := ApplyWhy(html, d.Hidden, func(id string) string { return whys[id] })
	if p.Empty {
		out = insertEmpty(out, mWhyEmpty.Text(lang(ctx), nil))
	}
	return out, &p, nil
}

// insertEmpty puts the "Nothing between the header and the footer" box
// after the last "Hidden" line of a preview.
func insertEmpty(html, text string) string {
	i := strings.LastIndex(html, `<div class="rc-why"`)
	if i < 0 {
		return html
	}
	end := strings.Index(html[i:], "</div>")
	if end < 0 {
		return html
	}
	at := i + end + len("</div>")
	return html[:at] + EmptyLine(text) + html[at:]
}
