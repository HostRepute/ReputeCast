// Package notify creates in-app notifications (A5 §3.12) and fans them out to
// the members who should see them. Delivery preferences (in-app and/or email
// per event type) are applied here; see EmailWorker for email delivery.
package notify

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Kinds of notifications. Preferences are stored per event type (see
// Events); several kinds can share one event type.
const (
	KindApprovalRequested   = "approval_requested"
	KindApprovalDecided     = "approval_decided" // approved, rejected, changes requested
	KindApprovalInvalidated = "approval_invalidated"
	KindSendPaused          = "send_paused"
	KindSendBlocked         = "send_blocked" // the send-time recheck turned Blocked
	KindProviderDown        = "provider_down"
	KindProviderDegraded    = "provider_degraded"
	KindWebhooksSilent      = "webhooks_silent"
	KindInviteAccepted      = "invite_accepted"
	KindInviteRequested     = "invite_requested" // an invitee asked for a new invitation
	KindHostReputeAlert     = "hostrepute_alert"
	KindSendFinished        = "send_finished"
	KindJobFinished         = "job_finished" // imports, exports
	// Integrations hub (design expansion, phase 2).
	KindIntegrationAttention = "integration_attention" // an integration needs attention
	KindPostDraftCreated     = "post_draft_created"    // a WordPress post became a campaign draft
	// KindTransactionalAlert is a delivery alert of transactional mail: a
	// template's bounce rate, failed messages or the median delivery time
	// over the last hour passed its limit.
	KindTransactionalAlert = "transactional_alert"
	// KindSMTPRelayUnreachable tells instance admins the SMTP relay can't
	// be reached from the internet.
	KindSMTPRelayUnreachable = "smtp_relay_unreachable"
	// Contact scoring (design expansion, phase 4).
	KindScoreRecalculated = "score_recalculated" // every contact was scored again after a rule change
	// Recurring digests (design expansion, phase 3).
	KindDigestIssueHeld   = "digest_issue_held"   // Send Readiness blocked a digest issue: it waits for a person
	KindDigestPaused      = "digest_paused"       // a digest paused (by a person, its owner's permissions, the feed)
	KindDigestFeedFailing = "digest_feed_failing" // a digest's feed failed at preparation, then the issue was skipped
	KindDigestDraftReady  = "digest_draft_ready"  // draft mode: an issue is a draft to send through Review & send
	// Earn (design expansion, phase 6).
	KindPaymentsAttention = "payments_attention"      // a payment provider's key rejected, webhook missing or silent
	KindUnmappedPrice     = "payments_unmapped_price" // a Stripe price isn't mapped to a tier
	KindPaidImportDone    = "paid_import_finished"    // a paid-members import finished
	KindFailedPayments    = "failed_payments_daily"   // the daily summary of failed payments (off by default)
	// The publication website (design expansion, phase 7).
	KindWebsiteDomain = "website_domain" // the website's custom domain stopped checking out, or its certificate is failing
)

// Notification is what appears in the notification center.
type Notification struct {
	WorkspaceID *uuid.UUID
	Kind        string
	Severity    string // info, ok, warn, bad
	Title       string
	// Summary is the short line shown in the bell popover; Body is used when
	// it is empty. Body is the full sentence in the notification center and
	// in the email.
	Summary string
	Body    string
	// Link is an app path such as "/w/acme-newsletter/campaigns/<id>/live".
	Link string
	Data map[string]any
}

// ToUser stores a notification for one person, honouring their preferences:
// nothing is stored when both in-app and email are off for the event type.
func ToUser(ctx context.Context, q store.DBTX, userID uuid.UUID, n Notification) error {
	if n.Severity == "" {
		n.Severity = "info"
	}
	data := []byte(`{}`)
	if n.Data != nil {
		b, err := json.Marshal(n.Data)
		if err != nil {
			return err
		}
		data = b
	}
	ev := EventFor(n.Kind)
	_, err := store.New(q).InsertNotification(ctx, store.InsertNotificationParams{
		ID:           uuid.Must(uuid.NewV7()),
		WorkspaceID:  n.WorkspaceID,
		Kind:         n.Kind,
		Severity:     n.Severity,
		Title:        n.Title,
		Summary:      n.Summary,
		Body:         n.Body,
		Link:         n.Link,
		Data:         data,
		DefaultEmail: ev.DefaultEmail,
		EventType:    ev.Key,
		UserID:       userID,
	})
	return err
}

// ToMembersWith notifies every active member of the workspace who holds perm
// (everyone when perm is empty).
func ToMembersWith(ctx context.Context, q store.DBTX, workspaceID uuid.UUID, perm authz.Permission, n Notification) error {
	ids, err := MembersWith(ctx, q, workspaceID, perm)
	if err != nil {
		return err
	}
	if n.WorkspaceID == nil {
		n.WorkspaceID = &workspaceID
	}
	for _, id := range ids {
		if err := ToUser(ctx, q, id, n); err != nil {
			return err
		}
	}
	return nil
}

// MembersWith returns the active members of a workspace who hold perm.
func MembersWith(ctx context.Context, q store.DBTX, workspaceID uuid.UUID, perm authz.Permission) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `SELECT m.user_id, r.key, r.builtin, r.permissions
		FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id AND u.status = 'active'
		WHERE m.workspace_id = $1 AND m.status = 'active'`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var (
			id      uuid.UUID
			key     string
			builtin bool
			perms   []string
		)
		if err := rows.Scan(&id, &key, &builtin, &perms); err != nil {
			return nil, err
		}
		if perm == "" || authz.Effective(key, builtin, perms).Has(perm) {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}
