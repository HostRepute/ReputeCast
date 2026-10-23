package mcpserver

import (
	"context"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// accessibilitySummary is a report's summary kept for later readers: the
// accessibility package's own messages (AccessibilityReport.SummaryIn).
func accessibilitySummary(r accessibility.AccessibilityReport) i18n.Text {
	if len(r.Issues) == 0 {
		return i18n.Text{ID: "accessibility.summary.none"}
	}
	return i18n.Text{ID: "accessibility.summary.issues", Args: i18n.Args{"count": len(r.Issues), "serious": r.Serious, "moderate": r.Moderate}}
}

// AccessibilityReader returns a campaign's accessibility issues (X06).
type AccessibilityReader interface {
	CampaignReport(ctx context.Context, ws *store.Workspace, id uuid.UUID) (accessibility.AccessibilityReport, error)
}

// addAccessibilityTools adds get_accessibility_issues: read-only, so agents
// can explain or propose fixes; a person applies them.
func (s *Service) addAccessibilityTools(srv *mcp.Server) {
	tool(s, srv, "get_accessibility_issues", authz.CampaignsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "campaign.accessibility_read", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		if s.Accessibility == nil {
			return map[string]any{"campaign": cp.Name, "message": "Accessibility checks aren't available on this server."}, e, nil
		}
		r, err := s.Accessibility.CampaignReport(ctx, &c.Workspace, cp.ID)
		if err != nil {
			return nil, e, err
		}
		type fix struct {
			Kind        string   `json:"kind"`
			OneClick    bool     `json:"one_click"`
			PerLanguage bool     `json:"per_language"`
			Suggestions []string `json:"suggested_colors,omitempty"`
		}
		type issue struct {
			Severity  string   `json:"severity"`
			Issue     string   `json:"issue"`
			Detail    string   `json:"detail,omitempty"`
			Languages []string `json:"languages"`
			Element   string   `json:"element"`
			Fix       fix      `json:"fix"`
		}
		issues := []issue{}
		for _, is := range r.Issues {
			f := fix{Kind: is.Fix.Kind, OneClick: is.Fix.OneClick, PerLanguage: is.Fix.PerLanguage}
			for _, sg := range is.Fix.Suggestions {
				f.Suggestions = append(f.Suggestions, sg.Color+" ("+accessibility.FormatRatio(sg.Ratio)+")")
			}
			loc := ""
			for i, p := range is.Location {
				if i > 0 {
					loc += " › "
				}
				loc += p
			}
			issues = append(issues, issue{Severity: is.Severity, Issue: is.Title, Detail: is.Detail, Languages: is.Languages, Element: loc, Fix: f})
		}
		e.detail, e.detailText = r.Summary(), accessibilitySummary(r)
		return map[string]any{
			"campaign": cp.Name, "summary": r.Summary(), "serious": r.Serious, "moderate": r.Moderate,
			"policy": r.Policy, "min_text_px": r.MinTextPx, "languages": r.Counts.Languages, "issues": issues,
			"added_at_compile": map[string]any{"lang_dir": r.Counts.Languages, "dir": r.Counts.Dir, "layout_tables_with_role_presentation": r.Counts.LayoutTables},
			"images":           map[string]int{"total": r.Counts.Images, "with_alt_text_or_decorative": r.Counts.ImagesOK},
			"links":            map[string]int{"total": r.Counts.Links, "say_where_they_go": r.Counts.LinksOK},
			"note":             "Serious issues block sending only when the workspace's policy is Block. Fixes are applied by a person in the composer or in Review & send.",
		}, e, nil
	}, s.campaignTarget("campaign.accessibility_read"))
}
