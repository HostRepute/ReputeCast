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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/sending/rules"
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

// ErrSlugTaken is returned by Create when the slug asked for is in use or
// reserved.
var ErrSlugTaken = errors.New("workspace slug taken")

// LockSlugs holds the instance's workspace slug lock until the transaction
// behind q ends. Whatever checks that a slug is free and then writes it
// takes the lock first, so two requests never both find the same slug free
// and the second write never breaks workspaces_slug_key.
func LockSlugs(ctx context.Context, q store.Querier) error {
	return q.WorkspaceSlugLock(ctx)
}

// UniqueSlug finds a free slug for name. q must be the transaction that
// then writes the slug: it takes the slug lock (LockSlugs).
func UniqueSlug(ctx context.Context, q store.Querier, name string) (string, error) {
	if err := LockSlugs(ctx, q); err != nil {
		return "", err
	}
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
	Name string
	// Slug is the address asked for; empty picks a free one from Name.
	Slug     string
	Timezone string
	Owner    uuid.UUID
	Source   string // install, admin, self, agency
}

// Create makes a workspace with the four built-in roles and the owner
// membership, inside the caller's transaction. A slug asked for that is in
// use or reserved returns ErrSlugTaken.
func Create(ctx context.Context, tx pgx.Tx, p CreateParams) (store.Workspace, error) {
	q := store.New(tx)
	slug, err := createSlug(ctx, q, p)
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
	// The response rules for the replies mailbox providers are known for
	// (Traffic rules, review GS-12): every workspace starts with them.
	if err := rules.AddDefaultReplyRules(ctx, q, ws.ID, time.Now()); err != nil {
		return store.Workspace{}, err
	}
	return q.GetWorkspace(ctx, ws.ID)
}

// createSlug is the slug a new workspace gets, checked under the slug lock.
func createSlug(ctx context.Context, q store.Querier, p CreateParams) (string, error) {
	if p.Slug == "" {
		return UniqueSlug(ctx, q, p.Name)
	}
	if err := LockSlugs(ctx, q); err != nil {
		return "", err
	}
	if reserved[p.Slug] {
		return "", ErrSlugTaken
	}
	taken, err := q.SlugTaken(ctx, p.Slug)
	if err != nil {
		return "", err
	}
	if taken {
		return "", ErrSlugTaken
	}
	return p.Slug, nil
}

// EnsureTwoFAGrace starts the two-factor grace period for a user who just
// gained a sensitive permission and has no second factor yet (A5 §3.3).
func EnsureTwoFAGrace(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE users SET twofa_grace_until = now() + make_interval(days => (SELECT twofa_grace_days FROM instance_settings))
		WHERE id = $1 AND totp_enabled_at IS NULL AND twofa_grace_until IS NULL
		  AND NOT EXISTS (SELECT 1 FROM user_passkeys p WHERE p.user_id = users.id)`, userID)
	return err
}
