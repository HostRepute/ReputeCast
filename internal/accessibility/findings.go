package accessibility

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
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
// inform; then what passed. They are in English; FindingsIn writes them in
// a reader's language.
func Findings(r AccessibilityReport) []Finding { return FindingsIn(i18n.Default, r) }

// FindingsIn is Findings in a reader's language.
func FindingsIn(l i18n.Lang, r AccessibilityReport) []Finding {
	out := []Finding{}
	many := len(r.Languages) > 1
	for _, is := range r.Issues {
		is = is.In(l)
		f := Finding{ID: is.ID, Title: is.Title, Detail: is.Detail, Path: is.Path, AltText: is.Kind == KindAltText}
		if many {
			names := make([]string, 0, len(is.Languages))
			for _, c := range is.Languages {
				names = append(names, LanguageNameIn(l, c))
			}
			f.Title = mFindingsLanguages.Text(l, i18n.Args{"title": f.Title, "languages": strings.Join(names, ", ")})
		}
		switch {
		case is.Severity == Serious && r.Policy == PolicyBlock:
			f.Status = FindingBlock
			f.Detail = strings.TrimSpace(f.Detail + " " + mFindingsBlocks.Text(l, nil))
		case is.Severity == Serious:
			f.Status = FindingWarn
		default:
			f.Status = FindingInfo
		}
		out = append(out, f)
	}
	if len(r.Issues) == 0 {
		out = append(out, Finding{ID: "ok", Status: FindingPass, Title: mFindingsOkTitle.Text(l, nil),
			Detail: mFindingsOkDetail.Text(l, i18n.Args{"images": r.Counts.Images, "min": r.MinTextPx})})
	}
	out = append(out, Finding{ID: "lang_dir", Status: FindingPass, Title: mFindingsLangDirTitle.Text(l, nil),
		Detail: mFindingsLangDirDetail.Text(l, i18n.Args{"languages": strings.Join(r.Counts.Languages, ", "), "dir": r.Counts.Dir})})
	tables := Finding{ID: "layout_tables", Status: FindingPass, Title: mFindingsTablesTitle.Text(l, i18n.Args{"count": r.Counts.LayoutTables}),
		Detail: mFindingsTablesNone.Text(l, nil)}
	if r.Counts.RoleAdded > 0 {
		tables.Detail = mFindingsTablesAdded.Text(l, nil)
	}
	return append(out, tables)
}
