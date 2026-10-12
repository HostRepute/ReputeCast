// Package notify creates in-app notifications (A5 §3.12) and fans them out to
// the members who should see them. Delivery preferences (in-app and/or email
// per event type) are applied here; email delivery goes through the System
// stream when one is connected.
package notify

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Kinds of notifications. Preferences are stored per kind.
const (
	KindApprovalRequested   = "approval_requested"
	KindApprovalDecided     = "approval_decided" // approved, rejected, changes requested
	KindApprovalInvalidated = "approval_invalidated"
	KindSendPaused          = "send_paused"
	KindProviderDown        = "provider_down"
	KindWebhooksSilent      = "webhooks_silent"
	KindInviteAccepted      = "invite_accepted"
	KindHostReputeAlert     = "hostrepute_alert"
	KindSendFinished        = "send_finished"
	KindJobFinished         = "job_finished" // imports, exports
)

// Notification is what appears in the notification center.
type Notification struct {
	WorkspaceID *uuid.UUID
	Kind        string
	Severity    string // info, ok, warn, bad
	Title       string
	Body        string
	// Link is an app path such as "/w/acme-newsletter/campaigns/<id>/live".
	Link string
	Data map[string]any
}

// ToUser stores a notification for one person.
func ToUser(ctx context.Context, q store.DBTX, userID uuid.UUID, n Notification) error {
	if n.Severity == "" {
		n.Severity = "info"
	}
	data, _ := json.Marshal(n.Data)
	if n.Data == nil {
		data = []byte(`{}`)
	}
	_, err := q.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, body, link, data)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9
		WHERE coalesce((SELECT in_app FROM notification_preferences WHERE user_id = $2 AND event_type = $4), true)`,
		uuid.Must(uuid.NewV7()), userID, n.WorkspaceID, n.Kind, n.Severity, n.Title, n.Body, n.Link, data)
	return err
}

// ToMembersWith notifies every active member of the workspace who holds perm.
func ToMembersWith(ctx context.Context, q store.DBTX, workspaceID uuid.UUID, perm authz.Permission, n Notification) error {
	rows, err := q.Query(ctx, `SELECT m.user_id, r.key, r.builtin, r.permissions
		FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id AND u.status = 'active'
		WHERE m.workspace_id = $1 AND m.status = 'active'`, workspaceID)
	if err != nil {
		return err
	}
	type member struct {
		id      uuid.UUID
		key     string
		builtin bool
		perms   []string
	}
	var ms []member
	for rows.Next() {
		var m member
		if err := rows.Scan(&m.id, &m.key, &m.builtin, &m.perms); err != nil {
			rows.Close()
			return err
		}
		ms = append(ms, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if n.WorkspaceID == nil {
		n.WorkspaceID = &workspaceID
	}
	for _, m := range ms {
		if perm == "" || authz.Effective(m.key, m.builtin, m.perms).Has(perm) {
			if err := ToUser(ctx, q, m.id, n); err != nil {
				return err
			}
		}
	}
	return nil
}
