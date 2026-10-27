package campaigns

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// SyncedUses counts the campaigns holding a synced block, by what a change
// to it does to them (review 2026-10, CO-11).
type SyncedUses struct {
	// Drafts and Scheduled get the change.
	Drafts    int `json:"drafts"`
	Scheduled int `json:"scheduled"`
	// Held are waiting for or holding an approval, sending or paused: a
	// change doesn't reach them, so an approved email stays as approved.
	Held int `json:"held"`
	// Sent keep the version they went out with.
	Sent int `json:"sent"`
}

// syncedPattern finds the campaigns that may hold a kind of synced block.
func syncedPattern(kind string) string { return `%` + mjml.AttrSynced + `="` + kind + `"%` }

// changeable reports whether a synced block's change reaches a campaign.
func changeable(status string, inApproval bool) bool {
	return (status == "draft" || status == "scheduled") && !inApproval
}

// CountSynced counts the campaigns holding a kind of synced block.
func (s *Service) CountSynced(ctx context.Context, ws *store.Workspace, kind string) (SyncedUses, error) {
	rows, err := s.Q.CampaignsHoldingSynced(ctx, store.CampaignsHoldingSyncedParams{WorkspaceID: ws.ID, Pattern: syncedPattern(kind)})
	if err != nil {
		return SyncedUses{}, err
	}
	var u SyncedUses
	for _, r := range rows {
		u.add(r.Status, r.InApproval)
	}
	return u, nil
}

func (u *SyncedUses) add(status string, inApproval bool) {
	switch {
	case changeable(status, inApproval) && status == "draft":
		u.Drafts++
	case changeable(status, inApproval):
		u.Scheduled++
	case status == "sent":
		u.Sent++
	case status == "cancelled" || status == "failed":
	default:
		u.Held++
	}
}

// ApplySynced puts a synced block's new content (block, one element marked
// rc-synced) into every draft and scheduled campaign holding that kind, as
// a new version saved by p (source "synced", label). Campaigns in an
// approval, sending or paused, and sent ones, keep their content; the
// counts say how many of each there were, and changed lists the campaigns
// that got the block.
func (s *Service) ApplySynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind, block, label string) (SyncedUses, []uuid.UUID, error) {
	rows, err := s.Q.CampaignsHoldingSynced(ctx, store.CampaignsHoldingSyncedParams{WorkspaceID: ws.ID, Pattern: syncedPattern(kind)})
	if err != nil {
		return SyncedUses{}, nil, err
	}
	var uses SyncedUses
	var changed []uuid.UUID
	for _, r := range rows {
		if !changeable(r.Status, r.InApproval) {
			uses.add(r.Status, r.InApproval)
			continue
		}
		done := false
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			cur, err := q.ComposerLockCampaign(ctx, store.ComposerLockCampaignParams{WorkspaceID: ws.ID, ID: r.ID})
			if err != nil {
				return err
			}
			// Checked again under the lock: the campaign may have moved on.
			a, err := s.openApproval(ctx, q, r.ID)
			if err != nil {
				return err
			}
			if !changeable(cur.Status, a != nil) {
				uses.add(cur.Status, a != nil)
				return nil
			}
			next, ok := mjml.ReplaceSynced(cur.Mjml, kind, block)
			ab := abFromDB(cur.AbTest)
			if ab != nil {
				for i, v := range ab.Variants {
					if m, vok := mjml.ReplaceSynced(v.MJML, kind, block); vok {
						ab.Variants[i].MJML, ab.Variants[i].HTML, ok = m, "", true
					}
				}
			}
			if !ok {
				uses.add(cur.Status, false) // already current
				return nil
			}
			var settings ComposerContentSettings
			_ = json.Unmarshal(cur.ContentSettings, &settings)
			// The server compiles the new MJML itself (no HTML is sent).
			c, err := s.prepareContent(ctx, q, ws, r.ID, ComposerSaveContent{Subject: cur.Subject, Preheader: cur.Preheader, MJML: next, ABTest: ab}, settings)
			if err != nil {
				return err
			}
			if _, err := s.storeVersion(ctx, q, ws, p, r.ID, cur, c, label, "synced"); err != nil {
				return err
			}
			uses.add(cur.Status, false)
			done = true
			return nil
		})
		if err != nil {
			return uses, changed, err
		}
		if done {
			changed = append(changed, r.ID)
			s.changed(ctx, ws, r.ID)
		}
	}
	return uses, changed, nil
}

// SyncedLabel is the version label of a synced block's change ("Footer
// updated in Brand kit"), in English like other stored labels.
func SyncedLabel(kind string) string {
	if kind == "" {
		return "Synced block updated in Brand kit"
	}
	return strings.ToUpper(kind[:1]) + kind[1:] + " updated in Brand kit"
}
