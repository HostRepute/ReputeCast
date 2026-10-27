package readiness

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Warnings dismissed in the composer (review 2026-10, CO-14): someone who
// can edit the campaign judged a warning with a reason ("the image is
// decorative"). Review & send lists it as the warning it is, with who
// dismissed it and why, and counts it as acknowledged when the send is
// confirmed and when a scheduled send starts. The composer keys each of its
// warnings by its check, element and words (campaigns.IssueKey); a run
// keeps the keys of the warnings each of its checks is made of, so a
// dismissal (or its undo) shows without running the checks again.

// dismissable are the run's warnings the composer finds too, by the content
// issue they are made of.
var dismissable = map[string]string{
	"content.alt_text":        "content.alt_text",
	"content.size":            "content.size",
	"content.image_ratio":     "content.image_ratio",
	"content.plain_text":      "content.plain_text",
	"content.spam":            "subject.spam",
	"content.personalization": "personalization.unknown_field",
	"links.shortener":         "links.shortener",
	"links.empty":             "links.empty",
}

// issueKeys gives each warning the composer also shows the keys of the
// composer's warnings it is made of.
func (b *builder) issueKeys() {
	ct := b.f.Content
	if !ct.Checked {
		return
	}
	r := ct.Report
	for i := range b.checks {
		c := &b.checks[i]
		id, ok := dismissable[c.Key]
		if !ok || c.Status != Warn {
			continue
		}
		switch c.Key {
		case "content.spam":
			// The composer's warning is the subject's; the preheader's and
			// the body's heuristics are Review & send's own.
			if len(spamPlaces(r, false)) > 0 {
				continue
			}
		case "content.size":
			// Tracking can push an email the composer measured under the
			// clip over it, and conditional content sends versions of
			// their own sizes: either way it isn't the warning dismissed.
			if (ct.TrackedBytes > contentcheck.ClipBytes && r.SizeBytes <= contentcheck.ClipBytes) || b.conditionalVersions() {
				continue
			}
		}
		keys := []string{}
		for _, is := range r.Issues {
			if is.ID == id && is.Severity == contentcheck.Warn {
				keys = append(keys, campaigns.IssueKey(is, b.f.Campaign.Subject))
			}
		}
		if len(keys) > 0 {
			c.IssueKeys = keys
		}
	}
}

// MarkDismissed sets Dismissal on the run's warnings whose every composer
// warning was dismissed (d is keyed by issue key), and clears it on the
// others. The latest dismissal speaks for the warning.
func MarkDismissed(run *ReadinessRun, d map[string]campaigns.ComposerDismissal) {
	if run == nil {
		return
	}
	for i := range run.Checks {
		c := &run.Checks[i]
		c.Dismissal = nil
		if c.Status != Warn || len(c.IssueKeys) == 0 {
			continue
		}
		var last *campaigns.ComposerDismissal
		all := true
		for _, k := range c.IssueKeys {
			x, ok := d[k]
			if !ok {
				all = false
				break
			}
			if last == nil || x.At.After(last.At) {
				last = &x
			}
		}
		if all && last != nil {
			c.Dismissal = &ReadinessDismissal{ID: last.ID, By: last.By, Reason: last.Reason, ReasonText: last.ReasonText, Note: last.Note, At: last.At,
				Count: len(c.IssueKeys)}
		}
	}
}

// Dismissed marks the run's warnings dismissed in the composer, with the
// reasons in the request's language. Runs without such warnings read
// nothing.
func (s *Service) Dismissed(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, run *ReadinessRun) error {
	if run == nil {
		return nil
	}
	need := false
	for _, c := range run.Checks {
		if c.Status == Warn && len(c.IssueKeys) > 0 {
			need = true
			break
		}
	}
	if !need {
		return nil
	}
	d, err := campaigns.DismissalsOf(ctx, s.Q, i18n.LangFrom(ctx), ws.ID, campaignID)
	if err != nil {
		return err
	}
	MarkDismissed(run, d)
	return nil
}

// DismissedKeys are the keys of the run's warnings dismissed in the
// composer: acknowledged without a tick.
func DismissedKeys(run ReadinessRun) []string {
	out := []string{}
	for _, c := range run.Checks {
		if c.Status == Warn && c.Dismissal != nil {
			out = append(out, c.Key)
		}
	}
	return out
}
