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

// accessibilityChecks adds the Accessibility group. It reports whether the
// checks ran: they then cover the images' alt text.
func (s *Service) accessibilityChecks(ctx context.Context, cs checkSubject, html string, add func(TransactionalCheck)) bool {
	if s.Accessibility == nil || (strings.TrimSpace(cs.MJML) == "" && strings.TrimSpace(html) == "") {
		return false
	}
	mj := cs.MJML
	if cs.EditorMode == "html" {
		mj = ""
	}
	checks, err := s.Accessibility.TemplateChecks(ctx, cs.WorkspaceID, cs.TemplateID, mj, html)
	if err != nil {
		s.log().WarnContext(ctx, "transactional accessibility checks", "err", err)
		return false
	}
	for _, c := range checks {
		add(c)
	}
	return true
}
