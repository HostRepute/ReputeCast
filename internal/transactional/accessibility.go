package transactional

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Accessibility checks (design expansion X06): the Accessibility group of
// a template's checks. Serious issues warn, or block publishing under the
// workspace's Block policy; the live version keeps sending meanwhile.

// GroupAccessibility is the checks' Accessibility group.
const GroupAccessibility = "Accessibility"

// AccessibilityChecker runs the accessibility checks on a template's
// content (internal/accessibility), as findings of the Accessibility group.
type AccessibilityChecker interface {
	TemplateChecks(ctx context.Context, wsID, templateID uuid.UUID, mjml, html string) ([]TransactionalCheck, error)
}

// accessibilityChecks runs the Accessibility group. It reports whether the
// checks ran: they then cover the images' alt text.
func (s *Service) accessibilityChecks(ctx context.Context, cs checkSubject, html string) ([]TransactionalCheck, bool) {
	if s.Accessibility == nil || (strings.TrimSpace(cs.MJML) == "" && strings.TrimSpace(html) == "") {
		return nil, false
	}
	mj := cs.MJML
	if cs.EditorMode == "html" {
		mj = ""
	}
	checks, err := s.Accessibility.TemplateChecks(ctx, cs.WorkspaceID, cs.TemplateID, mj, html)
	if err != nil {
		if s.Log != nil {
			s.Log.WarnContext(ctx, "transactional accessibility checks", "err", err)
		}
		return nil, false
	}
	return checks, true
}

// publishChecks is the checks as the publish dialog lists them: of the
// Accessibility group, its issues and "No accessibility issues". The
// additions made on save (lang and dir, the layout tables' role) stay in the
// Checks tab.
func publishChecks(r TransactionalChecks) TransactionalChecks {
	kept := make([]TransactionalCheck, 0, len(r.Checks))
	for _, c := range r.Checks {
		if c.Group == GroupAccessibility && c.Severity == SevPass && c.ID != "accessibility.ok" {
			r.Passed--
			continue
		}
		kept = append(kept, c)
	}
	r.Checks = kept
	return r
}
