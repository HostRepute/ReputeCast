package approvals

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// The outgoing webhook events of Review & send and approvals (the catalog in
// internal/webhooks), each queued in the transaction that makes the change.
// The dispatcher never calls the demo's sample endpoints.

// emitRequested queues approval.requested.
func emitRequested(ctx context.Context, q store.DBTX, ws *store.Workspace, requestID uuid.UUID, subject string, subjectID uuid.UUID, name, by string, rules []string, recipients int64) error {
	data := map[string]any{"request_id": requestID.String(), "subject_type": subject, subject + "_id": subjectID.String(), "name": name, "requested_by": by, "rules": rules}
	if subject == "campaign" {
		data["recipients"] = recipients
	}
	_, err := webhooks.Emit(ctx, q, ws.ID, ws.Slug, "approval.requested", data)
	return err
}

// emitDecided queues approval.decided once a request is approved, sent back
// or rejected.
func emitDecided(ctx context.Context, q store.DBTX, ws *store.Workspace, r store.ApprovalRequest, name, status, by, comment string) error {
	data := map[string]any{"request_id": r.ID.String(), "subject_type": r.SubjectType, "name": name, "decision": status, "decided_by": by}
	if r.CampaignID != nil {
		data["campaign_id"] = r.CampaignID.String()
	}
	if r.AutomationID != nil {
		data["automation_id"] = r.AutomationID.String()
	}
	if r.WebsitePostID != nil {
		data["website_post_id"] = r.WebsitePostID.String()
	}
	if r.WebsitePageID != nil {
		data["website_page_id"] = r.WebsitePageID.String()
	}
	if comment != "" {
		data["comment"] = comment
	}
	_, err := webhooks.Emit(ctx, q, ws.ID, ws.Slug, "approval.decided", data)
	return err
}

// emitScheduled queues campaign.scheduled.
func emitScheduled(ctx context.Context, q store.DBTX, ws *store.Workspace, campaignID uuid.UUID, name string, at time.Time, recipients int64, approvalID string) error {
	data := map[string]any{"campaign_id": campaignID.String(), "name": name, "scheduled_at": at.UTC(), "recipients": recipients}
	if approvalID != "" {
		data["approval_id"] = approvalID
	}
	_, err := webhooks.Emit(ctx, q, ws.ID, ws.Slug, "campaign.scheduled", data)
	return err
}

// emitHeld queues campaign.paused for a send the send-time checks held
// ("if anything turns to Block, the send pauses and you get an email and a
// Slack message").
func emitHeld(ctx context.Context, q store.DBTX, ws *store.Workspace, campaignID uuid.UUID, name, reason string, held int64) error {
	_, err := webhooks.Emit(ctx, q, ws.ID, ws.Slug, "campaign.paused", map[string]any{"campaign_id": campaignID.String(), "name": name,
		"paused_by": "send_readiness", "reason": reason, "held": held})
	return err
}
