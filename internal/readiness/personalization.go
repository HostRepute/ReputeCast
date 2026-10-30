package readiness

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/liquid"
)

// Personalization (SR-7, C8): a tag that doesn't parse stops the email from
// rendering for anyone (Block, found by liquid.Check through the content
// check); a field that doesn't exist prints nothing or its fallback for
// everyone (Warn); a first-name tag without a fallback greets the
// recipients without a first name with nothing (Warn, with their count).

var outputSpan = regexp.MustCompile(`\{\{-?[\s\S]*?-?\}\}`)

// fieldUses are the workspace fields (known) an email's texts print, each
// once: HasFallback only when every use has one, Fallback the first one.
func fieldUses(known map[string]bool, texts ...string) []FieldUse {
	out := []FieldUse{}
	idx := map[string]int{}
	for _, t := range texts {
		for _, span := range outputSpan.FindAllString(t, -1) {
			for _, u := range liquid.Uses(span) {
				key := strings.TrimPrefix(u.Key, "contact.")
				if !known[key] {
					continue
				}
				i, ok := idx[key]
				if !ok {
					idx[key] = len(out)
					out = append(out, FieldUse{Field: key, HasFallback: u.HasFallback, Fallback: u.Fallback})
					continue
				}
				if !u.HasFallback {
					out[i].HasFallback = false
				} else if out[i].Fallback == "" {
					out[i].Fallback = u.Fallback
				}
			}
		}
	}
	return out
}

// personalizationFields are the workspace's contact fields (first and last
// name, email and its custom fields) the texts print.
func (s *Service) personalizationFields(ctx context.Context, wsID uuid.UUID, texts ...string) []FieldUse {
	known := map[string]bool{}
	for _, f := range liquid.ContactFields {
		known[f.Key] = true
	}
	if rows, err := s.Q.ComposerCustomFields(ctx, wsID); err == nil {
		for _, f := range rows {
			known[f.Key] = true
		}
	}
	return fieldUses(known, texts...)
}

// issueTitle and issueDetail are a content check issue's words in each
// reader's language (English when the issue kept none).
func issueTitle(is contentcheck.ContentCheckIssue) Text {
	if !is.TitleText.IsZero() {
		return Kept(is.TitleText)
	}
	return RawText(is.Title)
}

func issueDetail(is contentcheck.ContentCheckIssue) Text {
	if len(is.DetailText) > 0 {
		out := Text{}
		for _, t := range is.DetailText {
			out = append(out, Kept(t)...)
		}
		return out
	}
	return RawText(is.Detail)
}

// personalization is the Personalization row of the Content group.
func (b *builder) personalization() {
	ct := b.f.Content
	if !ct.Checked {
		return
	}
	r := ct.Report
	label := tx("label.personalization", nil)
	editor := link(tx("fix.openInEditor", nil), b.editor(nil, "deliverability"))
	// Language versions break on their own: a broken tag in any of them
	// stops that version.
	reports := []contentcheck.ContentCheckReport{r}
	for _, v := range ct.Versions {
		reports = append(reports, v.Report)
	}
	for i, rep := range reports {
		syntax := issuesOf(rep, "personalization.syntax")
		if len(syntax) == 0 {
			continue
		}
		c := check{Key: "content.personalization", Group: GroupContent, Status: Block, Label: label,
			Title:  tx("personalization.broken", nil),
			Detail: sentences(issueTitle(syntax[0]), tx("personalization.brokenDetail", A{"more": len(syntax) - 1})),
			Fixes:  []fix{editor}}
		if i > 0 {
			v := ct.Versions[i-1]
			c.Title = tx("personalization.brokenIn", A{"language": ArgLanguage(v.Language)})
			c.Fixes = []fix{link(tx("fix.openLanguage", A{"language": ArgLanguage(v.Language)}), b.href("/campaigns/"+b.f.Campaign.ID+"/edit?language="+v.Language))}
		}
		b.add(c)
		return
	}
	if unknown := issuesOf(r, "personalization.unknown_field"); len(unknown) > 0 {
		detail := issueDetail(unknown[0])
		if len(unknown) > 1 {
			detail = sentences(detail, tx("personalization.moreUnknown", A{"count": len(unknown) - 1}))
		}
		b.add(check{Key: "content.personalization", Group: GroupContent, Status: Warn, Label: label,
			Title: issueTitle(unknown[0]), Detail: detail, Ack: tx("personalization.unknownAck", A{"count": len(unknown)}),
			Fixes: []fix{editor}})
		return
	}
	var first *FieldUse
	for i, u := range ct.Fields {
		if u.Field == "first_name" {
			first = &ct.Fields[i]
		}
	}
	missing := b.f.Audience.NoFirstName
	switch {
	case len(ct.Fields) == 0:
		b.add(check{Key: "content.personalization", Group: GroupContent, Status: Pass, Label: label,
			Title: tx("personalization.none", nil), Short: tx("personalization.noneShort", nil)})
	case first != nil && !first.HasFallback && missing > 0:
		b.add(check{Key: "content.personalization", Group: GroupContent, Status: Warn, Label: label,
			Title:  tx("personalization.noFallback", A{"count": missing}),
			Detail: tx("personalization.noFallbackDetail", nil),
			Ack:    tx("personalization.noFallbackAck", A{"count": missing}), Fixes: []fix{editor}})
	default:
		c := check{Key: "content.personalization", Group: GroupContent, Status: Pass, Label: label,
			Title: tx("personalization.fallbacks", nil), Short: tx("personalization.fallbacksShort", nil)}
		without := []string{}
		for _, u := range ct.Fields {
			if !u.HasFallback && u.Field != "first_name" {
				without = append(without, "{{ "+u.Field+" }}")
			}
		}
		switch {
		case first != nil && first.HasFallback:
			c.Detail = tx("personalization.firstNameFallback", A{"fallback": first.Fallback, "count": missing})
		case first != nil:
			c.Title = tx("personalization.everyoneNamed", nil)
			c.Detail = tx("personalization.everyoneNamedDetail", nil)
		default:
			fields := []string{}
			for _, u := range ct.Fields {
				if u.HasFallback {
					fields = append(fields, "{{ "+u.Field+" }}")
				}
			}
			if len(fields) > 0 {
				c.Detail = tx("personalization.fieldsFallback", A{"fields": ArgAnd(fields), "count": len(fields)})
			}
		}
		if len(without) > 0 {
			slices.Sort(without)
			if len(without) == len(ct.Fields) {
				c.Title = tx("personalization.checked", nil)
			}
			c.Detail = sentences(c.Detail, tx("personalization.withoutFallback", A{"fields": ArgAnd(without), "count": len(without)}))
		}
		b.add(c)
	}
}
