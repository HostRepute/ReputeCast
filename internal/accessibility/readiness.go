package accessibility

import (
	"context"
	"encoding/json"
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
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReadinessFacts is what Send Readiness reads from the accessibility checks.
// Its words are English, as a run stores them, with their messages kept
// beside them (ProblemText, the issues' Texts, the fix's SummaryTexts).
type ReadinessFacts struct {
	// Ran is false when Send Readiness has no accessibility checker: the
	// group is left out and alt text stays in Content.
	Ran bool
	// Checked is false when the checks couldn't run; Problem says why.
	Checked     bool
	Problem     string
	ProblemText i18n.Text
	Report      AccessibilityReport
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
	// SummaryTexts are Summary's messages, one list per fix (SummaryIn);
	// empty for fixes recorded before they were kept.
	SummaryTexts [][]i18n.Text
}

// SummaryIn writes the fix's summary in a reader's language; English
// readers, and fixes recorded before their messages were kept, read the
// English it was recorded with.
func (f FixRecord) SummaryIn(l i18n.Lang) string {
	if l.Locale == i18n.English || len(f.SummaryTexts) == 0 {
		return f.Summary
	}
	parts := make([]string, 0, len(f.SummaryTexts))
	for _, items := range f.SummaryTexts {
		s, ok := listIn(l, items)
		if !ok {
			return f.Summary
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "; ")
}

// listIn writes a fix's sentence from its items, joined as the language
// joins a list, with a capital.
func listIn(l i18n.Lang, items []i18n.Text) (string, bool) {
	if len(items) == 0 {
		return "", false
	}
	parts := make([]string, 0, len(items))
	for _, t := range items {
		s := TextIn(l, t)
		if s == "" {
			return "", false
		}
		parts = append(parts, s)
	}
	return capitalize(i18n.AndList(l.Locale, parts)), true
}

// ReadinessFacts checks a campaign for Send Readiness.
func (s *Service) ReadinessFacts(ctx context.Context, ws *store.Workspace, c CampaignContent) ReadinessFacts {
	f := ReadinessFacts{Ran: true}
	if row, saved, err := s.policy(ctx, s.Q, ws.ID); err == nil && saved {
		at := row.UpdatedAt
		f.PolicyBy, f.PolicyAt = row.UpdatedByName, &at
	}
	problem := func(m i18n.Msg) {
		f.ProblemText = m.Keep(nil)
		f.Problem = english(f.ProblemText)
	}
	if strings.TrimSpace(c.MJML) == "" && strings.TrimSpace(c.HTML) == "" {
		problem(mReadinessNoContent)
		return f
	}
	r, err := s.Report(ctx, ws, c)
	if err != nil {
		problem(mReadinessWaits)
		if errors.Is(err, ErrCompile) {
			problem(mReadinessNoCompile)
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
			_ = json.Unmarshal(fx.SummaryTexts, &f.Fixed.SummaryTexts)
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
	// LabelText, AppliedTexts and SummaryTexts are the words above as
	// messages (LabelIn, AppliedIn, SummaryIn): Applied and Summary are
	// their items joined as a list.
	LabelText    i18n.Text
	AppliedTexts []i18n.Text
	SummaryTexts []i18n.Text
}

// LabelIn is the version label part in a reader's language.
func (c FixChange) LabelIn(l i18n.Lang) string {
	if s := TextIn(l, c.LabelText); s != "" && l.Locale != i18n.English {
		return s
	}
	return c.Label
}

// AppliedIn is what the fix did, in a reader's language.
func (c FixChange) AppliedIn(l i18n.Lang) string {
	if l.Locale != i18n.English {
		if s, ok := listIn(l, c.AppliedTexts); ok {
			return s
		}
	}
	return c.Applied
}

// SummaryIn is what passes after the fix, in a reader's language.
func (c FixChange) SummaryIn(l i18n.Lang) string {
	if l.Locale != i18n.English {
		if s, ok := listIn(l, c.SummaryTexts); ok {
			return s
		}
	}
	return c.Summary
}

// FixMJML applies a Review & send fix to an email's MJML. These fixes need
// no words, so Send Readiness applies them to every language version alike.
// The change is in English with its messages kept; an error reads in the
// request's language.
func (s *Service) FixMJML(ctx context.Context, ws *store.Workspace, src, action string, params map[string]string) (string, FixChange, error) {
	if strings.TrimSpace(src) == "" {
		return src, FixChange{}, inLanguage(ctx, errText(mErrHTMLCampaign, nil))
	}
	path, _ := parsePathKey(params["path"])
	index, _ := strconv.Atoi(params["index"])
	switch action {
	case ActionDecorative:
		next, err := ApplyFix(src, FixRequest{Kind: FixDecorative, Path: path, Index: index})
		if err != nil {
			return src, FixChange{}, inLanguage(ctx, fixError(err))
		}
		what := strings.TrimSpace(params["name"])
		image := i18n.Args{"image": "unknown"}
		if what == "" {
			what = "the image"
		} else if d, err := parseDoc(src); err == nil {
			image = (&checker{d: d}).imageArgs(path)
		}
		return next, FixChange{Label: english(mFixDecorativeLabel.Keep(nil)), Applied: "Marked " + lowerFirst(what) + " as decorative",
			Summary: capitalize(lowerFirst(what)) + " is decorative", Kinds: []string{KindAltText},
			LabelText: mFixDecorativeLabel.Keep(nil), AppliedTexts: []i18n.Text{mFixDecorativeApplied.Keep(image)},
			SummaryTexts: []i18n.Text{mFixDecorativeSummary.Keep(image)}}, nil
	case ActionColor:
		color := strings.TrimSpace(params["color"])
		next, err := ApplyFix(src, FixRequest{Kind: FixColor, Path: path, Target: params["target"], From: params["from"], Value: color})
		if err != nil {
			return src, FixChange{}, inLanguage(ctx, fixError(err))
		}
		label := strings.TrimSpace(params["label"])
		applied := mFixColorApplied.Keep(i18n.Args{"color": strings.ToUpper(color)})
		if label != "" {
			applied = mFixColorAppliedLabel.Keep(i18n.Args{"label": label, "color": strings.ToUpper(color)})
		}
		return next, FixChange{Label: english(mFixContrastLabel.Keep(nil)), Applied: english(applied), Summary: english(applied), Kinds: []string{KindContrast},
			LabelText: mFixContrastLabel.Keep(nil), AppliedTexts: []i18n.Text{applied}, SummaryTexts: []i18n.Text{applied}}, nil
	case ActionOneClick:
		opts, err := s.options(ctx, ws.ID)
		if err != nil {
			return src, FixChange{}, err
		}
		r, err := Check([]Version{{Lang: s.language(ctx, ws), MJML: src, Source: true}}, opts)
		if err != nil {
			return src, FixChange{}, inLanguage(ctx, fixError(err))
		}
		next, change, err := applyOneClick(src, r)
		return next, change, inLanguage(ctx, err)
	}
	return src, FixChange{}, fmt.Errorf("unknown fix %q", action)
}

func fixError(err error) error {
	if errors.Is(err, ErrFixTarget) {
		return errText(mErrChanged, nil)
	}
	return err
}

// applyOneClick applies every fix that needs no words: heading levels and
// text sizes.
func applyOneClick(src string, r AccessibilityReport) (string, FixChange, error) {
	change := FixChange{Label: english(mFixOneClickLabel.Keep(nil)), LabelText: mFixOneClickLabel.Keep(nil)}
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
			change.AppliedTexts = append(change.AppliedTexts, mFixHeadingsApplied.Keep(i18n.Args{"count": n, "to": to}))
			if n == 1 {
				change.SummaryTexts = append(change.SummaryTexts, mFixHeadingOneSummary.Keep(i18n.Args{"text": is.Fix.Headings[0].Text, "to": to}))
			} else {
				cards, _ := is.Texts.Title.Args["cards"].(string)
				change.SummaryTexts = append(change.SummaryTexts, mFixHeadingsSummary.Keep(i18n.Args{"cards": cards, "to": to}))
			}
			change.Kinds = append(change.Kinds, KindHeading)
		case KindTextSize:
			next, err := ApplyFix(out, FixRequest{Kind: FixTextSize, Path: is.Path, ToPx: is.Fix.ToPx})
			if err != nil {
				return src, change, fixError(err)
			}
			out = next
			size := i18n.Args{"px": is.Fix.ToPx}
			appliedMsg, summaryMsg := mFixSizeApplied, mFixSizeSummary
			if is.Element.Section != "" {
				size["section"] = is.Element.Section
				appliedMsg, summaryMsg = mFixSizeAppliedIn, mFixSizeSummaryIn
			}
			change.AppliedTexts = append(change.AppliedTexts, appliedMsg.Keep(size))
			change.SummaryTexts = append(change.SummaryTexts, summaryMsg.Keep(size))
			change.Kinds = append(change.Kinds, KindTextSize)
		}
	}
	if len(change.AppliedTexts) == 0 {
		return src, change, errText(mErrNothingToClick, nil)
	}
	// English as Send Readiness has always written it: "Changed 2 headings
	// to Heading 2 and set footer text to 12 px"; "Card titles are Heading
	// 2, and footer text is 12 px".
	change.Applied, _ = listIn(i18n.Default, change.AppliedTexts)
	summary := make([]string, 0, len(change.SummaryTexts))
	for _, t := range change.SummaryTexts {
		summary = append(summary, english(t))
	}
	if len(summary) > 1 {
		change.Summary = capitalize(strings.Join(summary[:len(summary)-1], ", ") + ", and " + lowerFirst(summary[len(summary)-1]))
	} else {
		change.Summary = capitalize(summary[0])
	}
	return out, change, nil
}

// RecordFix keeps the fixes applied from Review & send with the content
// they saved, and audits them. Their summaries keep their messages, so
// Send Readiness writes them in each reader's language.
func (s *Service) RecordFix(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, campaign, contentHash string, languages []string, changes []FixChange) error {
	if len(changes) == 0 {
		return nil
	}
	var kinds, applied, summary []string
	texts := make([][]i18n.Text, 0, len(changes))
	for _, c := range changes {
		kinds = append(kinds, c.Kinds...)
		applied = append(applied, c.Applied)
		summary = append(summary, c.Summary)
		if len(c.SummaryTexts) > 0 && texts != nil {
			texts = append(texts, c.SummaryTexts)
		} else {
			texts = nil // a fix without its messages: the record reads in English
		}
	}
	keptTexts := []byte(`[]`)
	if len(texts) > 0 {
		if b, err := json.Marshal(texts); err == nil {
			keptTexts = b
		}
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
			ContentHash: contentHash, Kinds: kinds, Summary: strings.Join(summary, "; "), SummaryTexts: keptTexts, Languages: languages, FixedBy: by, FixedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.accessibility_fix", TargetType: "campaign", TargetID: campaignID.String(), TargetName: campaign,
			After:    map[string]any{"fixed": applied, "languages": languages},
			Metadata: map[string]any{"source": "send_readiness", "kinds": kinds, "contentHash": contentHash}})
	})
}
