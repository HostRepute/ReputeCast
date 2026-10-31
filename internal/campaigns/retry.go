package campaigns

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/retryaudience"
	"github.com/hostrepute/reputecast/internal/store"
)

// Retries (review AN-7, R5.L5). The addresses of an issue that a mailbox
// provider blocked for the sender's reputation or policy are valid: they
// stay on the list. Once the block is fixed, a retry sends them the issue
// again: a draft copy of it that goes only to them, once each
// (internal/retryaudience), and out from Review & send like any campaign,
// with its checks, its approvals and the right to send campaigns.

// finished are the statuses of an issue whose send has ended.
var finished = map[string]bool{"sent": true, "cancelled": true, "failed": true}

// retryVersion is the content everyone else got: an A/B test's winner, or
// without one its default (the campaign itself).
type retryVersion struct{ subject, preheader, mjml, html string }

// winnerOf is the variant winner of an A/B test applied to base (the
// campaign or a language version of it); without a winner, base.
func winnerOf(base retryVersion, ab *ComposerABTest, winner string) retryVersion {
	if ab == nil || winner == "" {
		return base
	}
	for _, v := range ab.Variants {
		if v.Key != winner {
			continue
		}
		if v.Subject != "" {
			base.subject = v.Subject
		}
		if v.Preheader != "" {
			base.preheader = v.Preheader
		}
		if v.HTML != "" {
			base.mjml, base.html = v.MJML, v.HTML
		}
	}
	return base
}

// CreateRetry makes the retry of an issue: the issue as everyone else got
// it (an A/B test's winner, as a plain campaign), from the same sender with
// the same settings and language versions, to the issue's lists and
// segments kept to its addresses blocked for the sender's reputation that
// haven't had a retry yet. An issue has one retry at a time.
func (s *Service) CreateRetry(ctx context.Context, ws *store.Workspace, p *authz.Principal, issue uuid.UUID) (ComposerCampaign, error) {
	l := i18n.LangFrom(ctx)
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: issue})
	if err != nil {
		return ComposerCampaign{}, api.MapError(err, "Campaign")
	}
	args := i18n.Args{"name": c.Name}
	if _, ok := retryaudience.Of(c.Audience); ok {
		return ComposerCampaign{}, api.Conflict(mProblemRetryOfRetry.Text(l, args))
	}
	if !finished[c.Status] {
		return ComposerCampaign{}, api.Conflict(mProblemRetryNotSent.Text(l, args))
	}
	// A member whose role is scoped to some lists retries only an audience
	// they could pick.
	if err := s.checkAudience(ctx, s.Q, ws, p, audienceFromDB(c.Audience)); err != nil {
		return ComposerCampaign{}, err
	}
	audience, err := retryaudience.With(c.Audience, issue)
	if err != nil {
		return ComposerCampaign{}, err
	}
	ab := abFromDB(c.AbTest)
	winner := ""
	if ab != nil {
		winner = ab.Winner
	}
	v := winnerOf(retryVersion{subject: c.Subject, preheader: c.Preheader, mjml: c.Mjml, html: c.Html}, ab, winner)
	plain := contentcheck.PlainText(v.html)
	id := uuid.Must(uuid.NewV7())
	name := retryName(mLabelRetryOf.Text(l, args))
	now := s.now()
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by = &u
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.RetryLockIssue(ctx, store.RetryLockIssueParams{WorkspaceID: ws.ID, ID: issue}); err != nil {
			return api.MapError(err, "Campaign")
		}
		if r, err := q.RetryCampaignOf(ctx, store.RetryCampaignOfParams{WorkspaceID: ws.ID, CampaignID: issue.String()}); err == nil {
			return api.Conflict(mProblemRetryExists.Text(l, i18n.Args{"name": c.Name, "retry": r.Name})).WithMeta("campaignId", r.ID.String())
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		keep, err := retryaudience.LoadIssue(ctx, q, ws.ID, issue, uuid.Nil)
		if err != nil {
			return err
		}
		if keep.Len() == 0 {
			return api.Conflict(mProblemRetryNobody.Text(l, args))
		}
		if err := q.RetryInsertCampaign(ctx, store.RetryInsertCampaignParams{ID: id, WorkspaceID: ws.ID, Name: name, Subject: v.subject, Preheader: v.preheader,
			Audience: audience, Mjml: v.mjml, Html: v.html, PlainText: plain, ContentHash: ContentHash(v.subject, v.preheader, v.mjml, v.html, nil),
			CreatedBy: by, Now: now, Issue: issue}); err != nil {
			return err
		}
		if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: 1, Subject: v.subject, Preheader: v.preheader,
			Mjml: v.mjml, Html: v.html, PlainText: plain, SavedBy: by, SavedAt: now, Label: mVersionCopiedFrom.Text(i18n.Default, args), Source: "duplicate",
			SizeBytes: int32(len(v.html))}); err != nil {
			return err
		}
		if err := copyRetryLanguages(ctx, q, issue, id, winner); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			After:    map[string]any{"name": name, "fromCampaignId": issue.String(), "retryOf": issue.String(), "blocked": keep.Len()},
			Metadata: map[string]any{"retry_of": c.Name}})
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		// Someone made the retry at the same moment.
		return ComposerCampaign{}, api.Conflict(mProblemRetryExists.Text(l, i18n.Args{"name": c.Name, "retry": name}))
	}
	if err != nil {
		return ComposerCampaign{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// retryName keeps a retry's name within a campaign name's length.
func retryName(s string) string {
	if r := []rune(strings.TrimSpace(s)); len(r) > 200 {
		return string(r[:199]) + "…"
	}
	return strings.TrimSpace(s)
}

// copyRetryLanguages gives a retry its issue's language versions, each as
// everyone else got it (the A/B test's winner in that language, with no
// test of its own).
func copyRetryLanguages(ctx context.Context, q *store.Queries, issue, retry uuid.UUID, winner string) error {
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: "campaign", OwnerID: issue, OwnerVersion: 0})
	if err != nil || len(rows) == 0 {
		return err
	}
	if err := q.RetryCopyLanguageSource(ctx, store.RetryCopyLanguageSourceParams{ToID: retry, FromID: issue}); err != nil {
		return err
	}
	for _, r := range rows {
		ab := abFromDB(r.AbTest)
		v := winnerOf(retryVersion{subject: r.Subject, preheader: r.Preheader, mjml: r.Mjml, html: r.Html}, ab, winner)
		hash, plain := r.ContentHash, r.PlainText
		if ab != nil {
			hash, plain = ContentHash(v.subject, v.preheader, v.mjml, v.html, nil), contentcheck.PlainText(v.html)
		}
		if err := q.RetryCopyTranslation(ctx, store.RetryCopyTranslationParams{ID: uuid.Must(uuid.NewV7()), ToID: retry, Subject: v.subject, Preheader: v.preheader,
			Mjml: v.mjml, Html: v.html, PlainText: plain, ContentHash: hash, SrcID: r.ID}); err != nil {
			return err
		}
	}
	return nil
}
