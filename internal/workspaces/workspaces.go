// Package workspaces creates and manages workspaces: built-in roles, the
// owner membership, settings, transfer of ownership and deletion.
package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// DefaultApprovalPolicy is the policy of a new workspace (A5 §3.8): approval
// for big Broadcasts, warming or new providers and authors without send.
var DefaultApprovalPolicy = json.RawMessage(`{"rules":{"audience_min":{"enabled":true,"n":10000},"warming_provider":{"enabled":true},"new_provider":{"enabled":false,"days":14},"ready_with_warnings":{"enabled":false},"author_lacks_send":{"enabled":true},"approval_lists":{"enabled":false}},"approvals_required":1,"approver_roles":["owner","admin"],"automation_publish":false,"emergency_bypass":true}`)

// DefaultSendThresholds pause a send when bounces or complaints spike.
var DefaultSendThresholds = json.RawMessage(`{"hard_bounce":2.0,"complaint":0.10,"scope":"route","min_sample":500}`)

// DefaultSunsetPolicy sends one re-engagement email, then stops broadcasts.
var DefaultSunsetPolicy = json.RawMessage(`{"enabled":true,"after_days":180,"reengage":true,"stop_after_days":30}`)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns "Acme Newsletter" into "acme-newsletter".
func Slugify(name string) string {
	s := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-"), "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if len(s) < 2 {
		s = "workspace"
	}
	return s
}

// reserved slugs collide with top-level app routes.
var reserved = map[string]bool{"admin": true, "api": true, "new": true, "settings": true, "account": true}

// UniqueSlug finds a free slug for name.
func UniqueSlug(ctx context.Context, q store.Querier, name string) (string, error) {
	base := Slugify(name)
	for i := 0; i < 50; i++ {
		s := base
		if i > 0 {
			s = fmt.Sprintf("%s-%d", base, i+1)
		}
		if reserved[s] {
			continue
		}
		taken, err := q.SlugTaken(ctx, s)
		if err != nil {
			return "", err
		}
		if !taken {
			return s, nil
		}
	}
	return "", errors.New("could not find a free workspace slug")
}

// CreateParams describes a new workspace.
type CreateParams struct {
	Name     string
	Timezone string
	Owner    uuid.UUID
	Source   string // install, admin, self
}

// Create makes a workspace with the four built-in roles and the owner
// membership, inside the caller's transaction.
func Create(ctx context.Context, tx pgx.Tx, p CreateParams) (store.Workspace, error) {
	q := store.New(tx)
	slug, err := UniqueSlug(ctx, q, p.Name)
	if err != nil {
		return store.Workspace{}, err
	}
	tz := p.Timezone
	if tz == "" {
		tz = "UTC"
	}
	owner := p.Owner
	ws, err := q.CreateWorkspace(ctx, store.CreateWorkspaceParams{
		ID:        uuid.Must(uuid.NewV7()),
		Slug:      slug,
		Name:      strings.TrimSpace(p.Name),
		Timezone:  tz,
		CreatedBy: &owner,
	})
	if err != nil {
		return store.Workspace{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE workspaces SET approval_policy = $2, send_thresholds = $3, sunset_policy = $4 WHERE id = $1`,
		ws.ID, DefaultApprovalPolicy, DefaultSendThresholds, DefaultSunsetPolicy); err != nil {
		return store.Workspace{}, err
	}
	var ownerRole uuid.UUID
	for _, r := range authz.BuiltinRoles {
		role, err := q.CreateRole(ctx, store.CreateRoleParams{
			ID:          uuid.Must(uuid.NewV7()),
			WorkspaceID: ws.ID,
			Key:         r.Key,
			Name:        r.Name,
			Description: r.Description,
			Builtin:     true,
			Permissions: []string{},
			CreatedBy:   nil,
		})
		if err != nil {
			return store.Workspace{}, err
		}
		if r.Key == authz.RoleOwner {
			ownerRole = role.ID
		}
	}
	source := p.Source
	if source == "" {
		source = "self"
	}
	if _, err := q.CreateMember(ctx, store.CreateMemberParams{
		ID:          uuid.Must(uuid.NewV7()),
		WorkspaceID: ws.ID,
		UserID:      p.Owner,
		RoleID:      ownerRole,
		Scope:       json.RawMessage(`{}`),
		Source:      source,
	}); err != nil {
		return store.Workspace{}, err
	}
	if err := EnsureTwoFAGrace(ctx, tx, p.Owner); err != nil {
		return store.Workspace{}, err
	}
	for stream, failover := range map[string]string{"broadcast": "pause_and_ask", "automation": "automatic", "system": "automatic"} {
		if _, err := tx.Exec(ctx, `INSERT INTO stream_policies (workspace_id, stream, failover) VALUES ($1, $2, $3)`, ws.ID, stream, failover); err != nil {
			return store.Workspace{}, err
		}
	}
	return q.GetWorkspace(ctx, ws.ID)
}

// EnsureTwoFAGrace starts the two-factor grace period for a user who just
// gained a sensitive permission and has no second factor yet (A5 §3.3).
func EnsureTwoFAGrace(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE users SET twofa_grace_until = now() + make_interval(days => (SELECT twofa_grace_days FROM instance_settings))
		WHERE id = $1 AND totp_enabled_at IS NULL AND twofa_grace_until IS NULL
		  AND NOT EXISTS (SELECT 1 FROM user_passkeys p WHERE p.user_id = users.id)`, userID)
	return err
}
