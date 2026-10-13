package reports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// BounceReview returns what the providers said for one bounce type, without
// any address, so a person can decide whether to suppress them.
func (s *Service) BounceReview(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, typeKey string) (ReportBounceReview, error) {
	t, ok := bounceTypeByKey(typeKey)
	if !ok {
		return ReportBounceReview{}, api.Invalid("path.type", "Unknown bounce type.")
	}
	rep, err := s.Report(ctx, ws, campaignID)
	if err != nil {
		return ReportBounceReview{}, err
	}
	out := ReportBounceReview{Type: t.Key, Label: t.Label, Diagnostics: []ReportBounceDiagnostic{}}
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
	t, ok := bounceTypeByKey(typeKey)
	if !ok {
		return ReportSuppressResult{}, api.Invalid("path.type", "Unknown bounce type.")
	}
	if t.Suppress == "retry" {
		return ReportSuppressResult{}, api.Conflict("Soft bounces aren't suppressed: the address is retried on the next send, and suppressed only if it keeps failing.")
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
		return ReportSuppressResult{}, api.Conflict("Every address that bounced this way is already on the suppression list.")
	}
	if s.Suppress == nil {
		return ReportSuppressResult{}, errors.New("reports: no suppression service configured")
	}
	added, already, err := s.Suppress.AddSuppressions(ctx, ws, emails, t.Label+" bounces on "+c.Name)
	return ReportSuppressResult{Added: added, Already: already}, err
}
