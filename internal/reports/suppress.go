package reports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// BounceReview returns what the providers said for one bounce type, without
// any address, so a person can decide whether to suppress them.
func (s *Service) BounceReview(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, typeKey string) (ReportBounceReview, error) {
	l := i18n.LangFrom(ctx)
	t, ok := bounceTypeByKey(typeKey)
	if !ok {
		return ReportBounceReview{}, api.Invalid("path.type", mUnknownBounce.Text(l, nil))
	}
	rep, err := s.Report(ctx, ws, campaignID)
	if err != nil {
		return ReportBounceReview{}, err
	}
	out := ReportBounceReview{Type: t.Key, Label: bounceLabel(l, t.Key), Diagnostics: []ReportBounceDiagnostic{}}
	for _, r := range rep.Bounces.Rows {
		if r.Type == t.Key {
			out.Count, out.Unsuppressed = r.Count, r.Unsuppressed
		}
	}
	cid := campaignID
	rows, err := s.Q.ReportBounceDiagnostics(ctx, store.ReportBounceDiagnosticsParams{CampaignID: &cid, WorkspaceID: ws.ID, Classes: t.Classes})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Diagnostics = append(out.Diagnostics, ReportBounceDiagnostic{Provider: providerShort(r.ProviderName, r.Adapter, ""), Type: r.OriginalType,
			Code: r.OriginalCode, Diagnostic: r.Diagnostic, Count: r.N})
	}
	return out, nil
}

// SuppressBounces adds the bounced addresses of one type that aren't on the
// shared suppression list yet, through the suppression service (which
// writes the audit entry and tells every provider).
func (s *Service) SuppressBounces(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, typeKey string) (ReportSuppressResult, error) {
	l := i18n.LangFrom(ctx)
	t, ok := bounceTypeByKey(typeKey)
	if !ok {
		return ReportSuppressResult{}, api.Invalid("path.type", mUnknownBounce.Text(l, nil))
	}
	switch t.Suppress {
	case "retry":
		return ReportSuppressResult{}, api.Conflict(mSoftBounces.Text(l, nil))
	case "never":
		// A block for the sender's reputation or policy (review AN-7).
		return ReportSuppressResult{}, api.Conflict(mSenderBlocks.Text(l, nil))
	}
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return ReportSuppressResult{}, api.MapError(err, "Campaign")
	}
	cid := c.ID
	emails, err := s.Q.ReportBounceRecipients(ctx, store.ReportBounceRecipientsParams{CampaignID: &cid, WorkspaceID: ws.ID, Classes: t.Classes})
	if err != nil {
		return ReportSuppressResult{}, err
	}
	if len(emails) == 0 {
		return ReportSuppressResult{}, api.Conflict(mAllSuppressed.Text(l, nil))
	}
	if s.Suppress == nil {
		return ReportSuppressResult{}, errors.New("reports: no suppression service configured")
	}
	args := i18n.Args{"type": t.Key, "campaign": c.Name}
	added, already, err := s.Suppress.AddSuppressions(ctx, ws, emails, mBounceNote.Text(i18n.Default, args), mBounceNote.Keep(args))
	return ReportSuppressResult{Added: added, Already: already}, err
}
