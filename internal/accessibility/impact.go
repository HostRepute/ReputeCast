package accessibility

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		ic := AccessibilityImpactCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, ScheduledAt: c.ScheduledAt, Serious: r.Serious, Summary: SeriousSummary(r.Issues, true)}
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
				Step: d.Step, Serious: r.Serious, Summary: SeriousSummary(r.Issues, false)})
		}
	}
	return out, nil
}

// SeriousSummary names an email's serious issues in a few words: "the hero
// image's alt text and the button contrast" (or, without articles, "button
// contrast").
func SeriousSummary(issues []AccessibilityIssue, articles bool) string {
	var parts []string
	seen := map[string]bool{}
	for _, is := range issues {
		if is.Severity != Serious {
			continue
		}
		p := seriousPhrase(is, articles)
		if !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	return joinAnd(parts)
}

func seriousPhrase(is AccessibilityIssue, articles bool) string {
	the := func(s string) string {
		if articles {
			return "the " + s
		}
		return s
	}
	switch is.Kind {
	case KindAltText:
		switch is.Element.Section {
		case "hero":
			return the("hero image's alt text")
		case "header":
			return the("logo's alt text")
		}
		if articles {
			return "an image's alt text"
		}
		return "image alt text"
	case KindLinkNoText:
		switch is.Element.Role {
		case "linked_image":
			if is.Element.Section == "header" {
				return the("linked logo's alt text")
			}
			return the("linked image's alt text")
		case "button":
			return the("button with no text")
		}
		return the("link with no text")
	case KindContrast:
		if is.Element.Role == "button" {
			return the("button contrast")
		}
		return the("text contrast")
	}
	return strings.ToLower(is.Tag)
}

// IssueCount is "1 serious issue" or "2 serious issues".
func IssueCount(n int, severity string) string {
	return fmt.Sprintf("%s %s", count(n), map[bool]string{true: severity + " issue", false: severity + " issues"}[n == 1])
}
