package accessibility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// AccessibilityImpactCampaign is a campaign the Block policy would stop.
type AccessibilityImpactCampaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status" enum:"draft,approval,approved,scheduled"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	Serious     int        `json:"serious"`
	// Summary names the serious issues: "the hero image's alt text and the
	// button contrast".
	Summary string `json:"summary"`
	// Held is true for a scheduled send its send-time recheck would hold.
	Held bool `json:"held"`
}

// AccessibilityImpactFlow is an automation draft whose publishing the
// Block policy would stop. The live version keeps sending.
type AccessibilityImpactFlow struct {
	AutomationID string `json:"automationId"`
	Name         string `json:"name"`
	Version      int    `json:"version"`
	LiveVersion  *int   `json:"liveVersion,omitempty"`
	// Step is how the flow names the step ("step 3 B").
	Step    string `json:"step"`
	Serious int    `json:"serious"`
	Summary string `json:"summary"`
}

// AccessibilityImpact is what a policy would affect now.
type AccessibilityImpact struct {
	OnSerious string                        `json:"onSerious" enum:"warn,block"`
	Campaigns []AccessibilityImpactCampaign `json:"campaigns"`
	Flows     []AccessibilityImpactFlow     `json:"flows"`
	// Held counts the scheduled campaigns that would be held at their
	// send-time recheck.
	Held int `json:"held"`
}

// Impact lists the campaigns and automation drafts with serious issues: a
// Block policy would hold the scheduled ones at their send-time recheck and
// stop the others until they're fixed.
func (s *Service) Impact(ctx context.Context, ws *store.Workspace, in AccessibilityPolicyInput) (AccessibilityImpact, error) {
	l := i18n.LangFrom(ctx)
	out := AccessibilityImpact{OnSerious: in.OnSerious, Campaigns: []AccessibilityImpactCampaign{}, Flows: []AccessibilityImpactFlow{}}
	if out.OnSerious != PolicyBlock {
		out.OnSerious = PolicyWarn
	}
	rows, err := s.Q.AccessibilityPendingCampaigns(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, c := range rows {
		r, err := s.Report(ctx, ws, CampaignContent{ID: c.ID, Name: c.Name, Status: c.Status, MJML: c.Mjml, HTML: c.Html})
		if err != nil || r.Serious == 0 {
			continue // an email that doesn't compile is reported by Send Readiness itself
		}
		ic := AccessibilityImpactCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, ScheduledAt: c.ScheduledAt, Serious: r.Serious, Summary: SeriousSummaryIn(l, r.Issues, true)}
		if c.Status == "scheduled" && out.OnSerious == PolicyBlock {
			ic.Held = true
			out.Held++
		}
		out.Campaigns = append(out.Campaigns, ic)
	}
	if s.Drafts != nil {
		drafts, err := s.Drafts(ctx, ws)
		if err != nil {
			return out, err
		}
		opts, err := s.options(ctx, ws.ID)
		if err != nil {
			return out, err
		}
		for _, d := range drafts {
			r, err := Check(d.Versions, opts)
			if err != nil || r.Serious == 0 {
				continue
			}
			out.Flows = append(out.Flows, AccessibilityImpactFlow{AutomationID: d.AutomationID.String(), Name: d.Automation, Version: d.Version, LiveVersion: d.LiveVersion,
				Step: d.Step, Serious: r.Serious, Summary: SeriousSummaryIn(l, r.Issues, false)})
		}
	}
	return out, nil
}

// SeriousSummary names an email's serious issues in a few words: "the hero
// image's alt text and the button contrast" (or, without articles, "button
// contrast").
func SeriousSummary(issues []AccessibilityIssue, articles bool) string {
	return SeriousSummaryIn(i18n.Default, issues, articles)
}

// SeriousSummaryIn is SeriousSummary in a reader's language.
func SeriousSummaryIn(l i18n.Lang, issues []AccessibilityIssue, articles bool) string {
	m := mPhraseSeriousBare
	if articles {
		m = mPhraseSerious
	}
	var parts []string
	seen := map[string]bool{}
	for _, is := range issues {
		if is.Severity != Serious {
			continue
		}
		p := strings.ToLower(is.Tag)
		if key := seriousKey(is); key != "" {
			p = m.Text(l, i18n.Args{"what": key})
		}
		if !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	return i18n.AndList(l.Locale, parts)
}

// seriousKey names a serious issue for phrase.serious: the hero image's
// alt text, the button contrast…; "" for other kinds.
func seriousKey(is AccessibilityIssue) string {
	switch is.Kind {
	case KindAltText:
		switch is.Element.Section {
		case "hero":
			return "heroAlt"
		case "header":
			return "logoAlt"
		}
		return "imageAlt"
	case KindLinkNoText:
		switch is.Element.Role {
		case "linked_image":
			if is.Element.Section == "header" {
				return "linkedLogo"
			}
			return "linkedImage"
		case "button":
			return "buttonNoText"
		}
		return "linkNoText"
	case KindContrast:
		if is.Element.Role == "button" {
			return "buttonContrast"
		}
		return "textContrast"
	}
	return ""
}

// ShortName names an issue inside a sentence: "the hero image's alt text",
// "the button contrast", "“Click here”", "2 “Read more” links", "a bare
// URL", "a skipped heading level", "11 px footer text".
func ShortName(is AccessibilityIssue) string {
	if t := shortText(is); !t.IsZero() {
		return english(t)
	}
	return strings.ToLower(is.Tag)
}

// shortText is ShortName's message; zero for kinds it doesn't name.
func shortText(is AccessibilityIssue) i18n.Text {
	if is.Severity == Serious {
		if key := seriousKey(is); key != "" {
			return mPhraseSerious.Keep(i18n.Args{"what": key})
		}
		return i18n.Text{}
	}
	switch is.Kind {
	case KindLinkText:
		n := max(is.Element.Count, 1)
		if strings.HasPrefix(is.Title, "A bare URL") || strings.Contains(is.Title, "show a bare URL") {
			return mPhraseBareURLs.Keep(i18n.Args{"count": n})
		}
		return mPhraseVagueLinks.Keep(i18n.Args{"count": n, "text": is.Element.Label})
	case KindHeading:
		return mPhraseSkipped.Keep(nil)
	case KindTextSize:
		if is.Element.Section != "" {
			return mPhraseTextSizeIn.Keep(i18n.Args{"px": is.Fix.FromPx, "section": is.Element.Section})
		}
		return mPhraseTextSize.Keep(i18n.Args{"px": is.Fix.FromPx})
	}
	return i18n.Text{}
}

// IssueCount is "1 serious issue" or "2 serious issues".
func IssueCount(n int, severity string) string {
	return fmt.Sprintf("%s %s", count(n), map[bool]string{true: severity + " issue", false: severity + " issues"}[n == 1])
}
