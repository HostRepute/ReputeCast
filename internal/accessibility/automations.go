package accessibility

import (
	"context"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// DraftStep is an email step of an automation version.
type DraftStep struct {
	// Step is how the flow names it ("Step 3 B").
	Step string
	MJML string
	HTML string
}

// DraftBadge is a badge of the publish dialog.
type DraftBadge struct {
	Status string // ok, warn or bad
	Label  string
}

// DraftCheck is what the publish dialog shows about accessibility.
type DraftCheck struct {
	Badges []DraftBadge
	// Blocked is set when the workspace blocks publishing with serious
	// issues; Blocker says what to fix.
	Blocked bool
	Blocker string
}

// CheckDraft checks the emails of an automation version before it is
// published: a badge per step with serious issues ("Step 3 B: 1 serious
// issue, button contrast"), which stop publishing when the workspace blocks
// on them. The live version keeps sending. Its words are in the request's
// language.
func (s *Service) CheckDraft(ctx context.Context, ws *store.Workspace, version int, live *int, steps []DraftStep) (DraftCheck, error) {
	out := DraftCheck{Badges: []DraftBadge{}}
	if len(steps) == 0 {
		return out, nil
	}
	opts, err := s.options(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	l := i18n.LangFrom(ctx)
	lang := s.language(ctx, ws)
	status := "warn"
	if opts.Policy == PolicyBlock {
		status = "bad"
	}
	var fixes []string
	for _, st := range steps {
		r, err := Check([]Version{{Lang: lang, MJML: st.MJML, HTML: st.HTML, Source: true}}, opts)
		if err != nil || r.Serious == 0 {
			continue
		}
		out.Badges = append(out.Badges, DraftBadge{Status: status,
			Label: mDraftBadge.Text(l, i18n.Args{"step": st.Step, "count": r.Serious, "summary": SeriousSummaryIn(l, r.Issues, false)})})
		fixes = append(fixes, mDraftFixIn.Text(l, i18n.Args{"summary": SeriousSummaryIn(l, r.Issues, true), "step": LowerStep(st.Step)}))
	}
	if len(out.Badges) == 0 {
		out.Badges = append(out.Badges, DraftBadge{Status: "ok", Label: mDraftNone.Text(l, i18n.Args{"count": len(steps)})})
		return out, nil
	}
	if opts.Policy == PolicyBlock {
		out.Blocked = true
		args := i18n.Args{"fixes": i18n.AndList(l.Locale, fixes), "version": version}
		out.Blocker = mDraftBlocker.Text(l, args)
		if live != nil {
			args["live"] = *live
			out.Blocker = mDraftBlockerLive.Text(l, args)
		}
	}
	return out, nil
}

// LowerStep is "step 3 B" for "Step 3 B", for use inside a sentence.
func LowerStep(s string) string {
	if strings.HasPrefix(s, "Step ") {
		return "step " + s[len("Step "):]
	}
	return s
}
