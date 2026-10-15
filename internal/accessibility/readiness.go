package accessibility

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReadinessFacts is what Send Readiness reads from the accessibility checks.
type ReadinessFacts struct {
	// Ran is false when Send Readiness has no accessibility checker: the
	// group is left out and alt text stays in Content.
	Ran bool
	// Checked is false when the checks couldn't run; Problem says why.
	Checked bool
	Problem string
	Report  AccessibilityReport
	// PolicyBy and PolicyAt say who set the policy last, and when.
	PolicyBy string
	PolicyAt *time.Time
	// Fixed is the latest fix applied from Review & send, while the content
	// is still what that fix saved.
	Fixed *FixRecord
}

// FixRecord is a fix applied from Review & send.
type FixRecord struct {
	By        string
	At        time.Time
	Summary   string
	Kinds     []string
	Languages []string
}

// ReadinessFacts checks a campaign for Send Readiness.
func (s *Service) ReadinessFacts(ctx context.Context, ws *store.Workspace, c CampaignContent) ReadinessFacts {
	f := ReadinessFacts{Ran: true}
	if row, saved, err := s.policy(ctx, s.Q, ws.ID); err == nil && saved {
		at := row.UpdatedAt
		f.PolicyBy, f.PolicyAt = row.UpdatedByName, &at
	}
	if strings.TrimSpace(c.MJML) == "" && strings.TrimSpace(c.HTML) == "" {
		f.Problem = "The campaign has no content yet."
		return f
	}
	r, err := s.Report(ctx, ws, c)
	if err != nil {
		f.Problem = "The other checks finished; the verdict waits for this one."
		if errors.Is(err, ErrCompile) {
			f.Problem = "The MJML doesn't compile. The other checks finished; the verdict waits for this one."
		}
		if s.Log != nil {
			s.Log.WarnContext(ctx, "accessibility check failed", "campaign", c.ID, "err", err)
		}
		return f
	}
	f.Checked, f.Report = true, r
	if c.Hash != "" {
		if fx, err := s.Q.AccessibilityLatestFix(ctx, c.ID); err == nil && fx.ContentHash == c.Hash {
			f.Fixed = &FixRecord{By: fx.FixedByName, At: fx.FixedAt, Summary: fx.Summary, Kinds: fx.Kinds, Languages: fx.Languages}
		}
	}
	return f
}

// Fix actions Send Readiness offers.
const (
	ActionDecorative = "accessibility_decorative"
	ActionColor      = "accessibility_color"
	ActionOneClick   = "accessibility_one_click"
)

// FixChange describes what a fix from Review & send changed.
type FixChange struct {
	// Label is the version label part ("decorative image"); Applied the
	// sentence the page shows ("Marked the hero image as decorative").
	Label   string
	Applied string
	// Summary is what now passes ("Card titles are Heading 2, and footer
	// text is 12 px"); Kinds the fixed issue kinds.
	Summary string
	Kinds   []string
	// Languages are the versions the fix changed.
	Languages []string
}

// FixMJML applies a Review & send fix to a campaign's MJML. Fixes need no
// words, so they apply to every language version alike.
func (s *Service) FixMJML(ctx context.Context, ws *store.Workspace, src, action string, params map[string]string) (string, FixChange, error) {
	next, change, err := s.fixMJML(ctx, ws, src, action, params)
	if err == nil {
		change.Languages = []string{s.language(ctx, ws)}
	}
	return next, change, err
}

func (s *Service) fixMJML(ctx context.Context, ws *store.Workspace, src, action string, params map[string]string) (string, FixChange, error) {
	if strings.TrimSpace(src) == "" {
		return src, FixChange{}, errors.New("this campaign is written in HTML: fix it in the composer's code view")
	}
	path, _ := parsePathKey(params["path"])
	index, _ := strconv.Atoi(params["index"])
	switch action {
	case ActionDecorative:
		next, err := ApplyFix(src, FixRequest{Kind: FixDecorative, Path: path, Index: index})
		if err != nil {
			return src, FixChange{}, fixError(err)
		}
		what := strings.TrimSpace(params["name"])
		if what == "" {
			what = "the image"
		}
		return next, FixChange{Label: "decorative image", Applied: "Marked " + lowerFirst(what) + " as decorative", Summary: capitalize(lowerFirst(what)) + " is decorative", Kinds: []string{KindAltText}}, nil
	case ActionColor:
		color := strings.TrimSpace(params["color"])
		next, err := ApplyFix(src, FixRequest{Kind: FixColor, Path: path, Target: params["target"], From: params["from"], Value: color})
		if err != nil {
			return src, FixChange{}, fixError(err)
		}
		label := strings.TrimSpace(params["label"])
		applied := "Changed the color to " + strings.ToUpper(color)
		if label != "" {
			applied = "Changed " + quote(label) + " to " + strings.ToUpper(color)
		}
		return next, FixChange{Label: "contrast", Applied: applied, Summary: applied, Kinds: []string{KindContrast}}, nil
	case ActionOneClick:
		opts, err := s.options(ctx, ws.ID)
		if err != nil {
			return src, FixChange{}, err
		}
		r, err := Check([]Version{{Lang: s.language(ctx, ws), MJML: src, Source: true}}, opts)
		if err != nil {
			return src, FixChange{}, fixError(err)
		}
		return applyOneClick(src, r)
	}
	return src, FixChange{}, fmt.Errorf("unknown fix %q", action)
}

func fixError(err error) error {
	if errors.Is(err, ErrFixTarget) {
		return errors.New("that part of the email changed since the checks ran. Run the checks again")
	}
	return err
}

// applyOneClick applies every fix that needs no words: heading levels and
// text sizes.
func applyOneClick(src string, r AccessibilityReport) (string, FixChange, error) {
	change := FixChange{Label: "headings and text size"}
	var applied, summary []string
	out := src
	for _, is := range r.Issues {
		if !is.Fix.OneClick || is.Severity != Moderate {
			continue
		}
		switch is.Kind {
		case KindHeading:
			next, err := ApplyFix(out, FixRequest{Kind: FixHeading, Headings: is.Fix.Headings})
			if err != nil {
				return src, change, fixError(err)
			}
			out = next
			n, to := len(is.Fix.Headings), is.Fix.Headings[0].To
			applied = append(applied, fmt.Sprintf("changed %s to Heading %d", plural(n, "heading", "headings"), to))
			what := "Headings"
			if strings.HasPrefix(is.Title, "Both card titles") || strings.Contains(is.Title, " card titles ") {
				what = "Card titles"
			}
			if n == 1 {
				summary = append(summary, fmt.Sprintf("%s is Heading %d", quote(is.Fix.Headings[0].Text), to))
			} else {
				summary = append(summary, fmt.Sprintf("%s are Heading %d", what, to))
			}
			change.Kinds = append(change.Kinds, KindHeading)
		case KindTextSize:
			next, err := ApplyFix(out, FixRequest{Kind: FixTextSize, Path: is.Path, ToPx: is.Fix.ToPx})
			if err != nil {
				return src, change, fixError(err)
			}
			out = next
			what := "text"
			if is.Element.Section != "" {
				what = is.Element.Section + " text"
			}
			applied = append(applied, fmt.Sprintf("set %s to %d px", what, is.Fix.ToPx))
			summary = append(summary, fmt.Sprintf("%s is %d px", what, is.Fix.ToPx))
			change.Kinds = append(change.Kinds, KindTextSize)
		}
	}
	if len(applied) == 0 {
		return src, change, errors.New("there's nothing to fix in one click any more. Run the checks again")
	}
	change.Applied = capitalize(joinAnd(applied))
	if len(summary) > 1 {
		change.Summary = capitalize(strings.Join(summary[:len(summary)-1], ", ") + ", and " + lowerFirst(summary[len(summary)-1]))
	} else {
		change.Summary = capitalize(summary[0])
	}
	return out, change, nil
}

// RecordFix keeps the fixes applied from Review & send with the content
// they saved, and audits them.
func (s *Service) RecordFix(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, campaign, contentHash string, languages []string, changes []FixChange) error {
	if len(changes) == 0 {
		return nil
	}
	var kinds, applied, summary []string
	for _, c := range changes {
		kinds = append(kinds, c.Kinds...)
		applied = append(applied, c.Applied)
		summary = append(summary, c.Summary)
	}
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		by = &id
	}
	if languages == nil {
		languages = []string{}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AccessibilityInsertFix(ctx, store.AccessibilityInsertFixParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, CampaignID: campaignID,
			ContentHash: contentHash, Kinds: kinds, Summary: strings.Join(summary, "; "), Languages: languages, FixedBy: by, FixedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.accessibility_fix", TargetType: "campaign", TargetID: campaignID.String(), TargetName: campaign,
			After:    map[string]any{"fixed": applied, "languages": languages},
			Metadata: map[string]any{"source": "send_readiness", "kinds": kinds, "contentHash": contentHash}})
	})
}
