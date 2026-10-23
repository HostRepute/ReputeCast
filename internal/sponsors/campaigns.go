package sponsors

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// CampaignChip names the sponsor of a campaign's issue: the Campaigns
// list's "Sponsored · Northwind Analytics" chip.
type CampaignChip struct {
	CampaignID string `json:"campaignId"`
	Sponsor    string `json:"sponsor"`
}

// CampaignChips is every campaign with a booked or sent sponsorship.
type CampaignChips struct {
	Items []CampaignChip `json:"items"`
}

// CampaignSponsor is a campaign report's Sponsor section: the issue's
// booking with its report, or nothing when the issue had no sponsor.
type CampaignSponsor struct {
	Booking *SponsorBookingDetail `json:"booking,omitempty"`
}

// Chips lists the sponsor of each campaign that has one. A campaign keeps
// its latest booking's sponsor.
func (s *Service) Chips(ctx context.Context, ws *store.Workspace) (CampaignChips, error) {
	out := CampaignChips{Items: []CampaignChip{}}
	rows, err := s.Q.SponsorCampaignChips(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	seen := map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.CampaignID == nil || seen[*r.CampaignID] {
			continue
		}
		seen[*r.CampaignID] = true
		out.Items = append(out.Items, CampaignChip{CampaignID: r.CampaignID.String(), Sponsor: r.Sponsor})
	}
	return out, nil
}

// CampaignSponsor is the Sponsor section of a campaign's report.
func (s *Service) CampaignSponsor(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (CampaignSponsor, error) {
	if _, err := s.Q.SponsorCampaignBrief(ctx, store.SponsorCampaignBriefParams{ID: campaignID, WorkspaceID: ws.ID}); err != nil {
		return CampaignSponsor{}, mapErr(ctx, err, "campaign")
	}
	b, err := s.Q.SponsorBookingOfCampaign(ctx, &campaignID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && b.WorkspaceID != ws.ID) {
		return CampaignSponsor{}, nil
	}
	if err != nil {
		return CampaignSponsor{}, err
	}
	d, err := s.Booking(ctx, ws, p, b.ID)
	if err != nil {
		return CampaignSponsor{}, err
	}
	return CampaignSponsor{Booking: &d}, nil
}
