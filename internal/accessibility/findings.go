package accessibility

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Finding is one line of a checks list outside Send Readiness (a
// transactional template's Checks tab): an issue, or a check that passed.
type Finding struct {
	// ID is the issue's ID, or "ok", "lang_dir", "layout_tables".
	ID string
	// Status is block, warn, info or pass.
	Status string
	Title  string
	Detail string
	Path   []int
	// AltText marks an image without alt text: the editor offers its field.
	AltText bool
}

// Finding statuses.
const (
	FindingBlock = "block"
	FindingWarn  = "warn"
	FindingInfo  = "info"
	FindingPass  = "pass"
)

// CheckVersions checks language versions of an email outside campaigns (a
// transactional template, an automation email) with the workspace's policy.
func (s *Service) CheckVersions(ctx context.Context, wsID uuid.UUID, versions []Version) (AccessibilityReport, error) {
	return s.run(ctx, wsID, versions)
}

// Findings lists a report as checks: serious issues warn, or block under
// the Block policy (the live version keeps sending); moderate issues
// inform; then what passed.
func Findings(r AccessibilityReport) []Finding {
	out := []Finding{}
	many := len(r.Languages) > 1
	for _, is := range r.Issues {
		f := Finding{ID: is.ID, Title: is.Title, Detail: is.Detail, Path: is.Path, AltText: is.Kind == KindAltText}
		if many {
			f.Title += " · " + strings.Join(languageNamesOf(is.Languages), ", ")
		}
		switch {
		case is.Severity == Serious && r.Policy == PolicyBlock:
			f.Status = FindingBlock
			f.Detail = strings.TrimSpace(f.Detail + " Your workspace blocks publishing with serious accessibility issues (Settings › General). The live version keeps sending.")
		case is.Severity == Serious:
			f.Status = FindingWarn
		default:
			f.Status = FindingInfo
		}
		out = append(out, f)
	}
	if len(r.Issues) == 0 {
		out = append(out, Finding{ID: "ok", Status: FindingPass, Title: "No accessibility issues",
			Detail: fmt.Sprintf("%s, link text that says where it goes, AA contrast, headings in order, text at %d px or more.", altText(r.Counts.Images), r.MinTextPx)})
	}
	out = append(out, Finding{ID: "lang_dir", Status: FindingPass, Title: "lang and dir set for every language",
		Detail: strings.Join(r.Counts.Languages, ", ") + " · " + r.Counts.Dir + " · added when the email is saved"})
	tables := Finding{ID: "layout_tables", Status: FindingPass, Title: fmt.Sprintf("role=\"presentation\" on %s", plural(r.Counts.LayoutTables, "layout table", "layout tables")),
		Detail: "MJML marked every layout table; nothing to add"}
	if r.Counts.RoleAdded > 0 {
		tables.Detail = "Added to the tables MJML missed, such as one in a Raw HTML block"
	}
	return append(out, tables)
}

// altText is "Alt text on all 7 images".
func altText(images int) string {
	switch images {
	case 0:
		return "No images"
	case 1:
		return "Alt text on the image"
	case 2:
		return "Alt text on both images"
	}
	return fmt.Sprintf("Alt text on all %d images", images)
}

func languageNamesOf(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		out = append(out, LanguageName(c))
	}
	return out
}
