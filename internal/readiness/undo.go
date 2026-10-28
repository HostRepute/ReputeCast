package readiness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Undo for one-click fixes (review R1.U3). "Apply fixes" can change a
// campaign in four ways at once: its content (one new version, saved
// through the composer), its content settings (Minify), its route override
// (Send via SES only) and its audience (Exclude 412 unengaged, Include
// them). Each batch keeps those parts as they were before and after it, so
// Undo puts them back while the campaign is still as the batch left it:
// the content through the composer's version restore, which saves a version
// and an audit entry of its own, the rest directly, each change audited.

// Why a batch can't be undone (readiness_fix_batches.blocked).
const (
	// undoLanguages: it changed a language version or a language's state,
	// which a version restore of the campaign doesn't put back.
	undoLanguages = "languages"
	// undoMerged: its content was saved merged with someone's edit, which
	// an undo would lose too.
	undoMerged = "merged"
)

// ReadinessFixBatch is one "Apply fixes": the fixes applied together, by
// whom and when, and whether Undo can put the campaign back as it was.
type ReadinessFixBatch struct {
	ID        string    `json:"id"`
	AppliedAt time.Time `json:"appliedAt"`
	AppliedBy string    `json:"appliedBy"`
	// Applied are the fixes as the person who applied them read them; empty
	// for readers of another language, who read Count.
	Applied []string `json:"applied"`
	Count   int      `json:"count"`
	// Version is the campaign version the batch's content fixes saved; 0
	// when they changed no content.
	Version int `json:"version,omitempty"`
	// Undoable is false when the batch changed what Undo can't put back (a
	// language version or a language's state; Version history can).
	Undoable bool `json:"undoable"`
	// Route is true when the batch changed the route: undoing it needs the
	// right to send, as applying it did.
	Route bool `json:"route,omitempty"`
}

// ReadinessUndoResult is what Undo put back.
type ReadinessUndoResult struct {
	// Count is how many fixes were undone.
	Count int `json:"count"`
	// Restored is the version whose content came back (0 when the batch
	// changed no content), saved as Version.
	Restored int `json:"restored,omitempty"`
	Version  int `json:"version"`
}

// fixSnapshot is what a batch changes, as it was before and after.
type fixSnapshot struct {
	version                   int32
	route, audience, settings json.RawMessage
}

func snapshotOf(c store.ReadinessCampaignRow) fixSnapshot {
	return fixSnapshot{version: c.Version, route: c.RouteOverride, audience: c.Audience, settings: c.ContentSettings}
}

// sameJSON reports whether two JSON documents say the same, key order and
// spacing aside; empty and null are the same.
func sameJSON(a, b json.RawMessage) bool {
	var x, y any
	if len(bytes.TrimSpace(a)) > 0 && json.Unmarshal(a, &x) != nil {
		return false
	}
	if len(bytes.TrimSpace(b)) > 0 && json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// orNull is a JSON document to store, nil for empty or null.
func orNull(raw json.RawMessage) []byte {
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || bytes.Equal(t, []byte("null")) {
		return nil
	}
	return raw
}

// recordFixes keeps a batch of applied fixes for Undo: the parts of the
// campaign they changed, before and after. A batch that changed nothing
// isn't kept (nil).
func (s *Service) recordFixes(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, before, after fixSnapshot,
	applied []string, saved *campaigns.ComposerSavedContent, blocked string) (*ReadinessFixBatch, error) {
	content := saved != nil && !saved.Unchanged && int32(saved.Version) != before.version
	route, audience, settings := !sameJSON(before.route, after.route), !sameJSON(before.audience, after.audience), !sameJSON(before.settings, after.settings)
	if !content && !route && !audience && !settings && blocked == "" {
		return nil, nil
	}
	if content && saved.Merged != nil && blocked == "" {
		blocked = undoMerged
	}
	arg := store.ReadinessInsertFixBatchParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, CampaignID: campaignID, AppliedBy: &p.UserID, AppliedAt: s.now(),
		AppliedLang: string(i18n.LangFrom(ctx).Locale), Blocked: blocked,
		RouteChanged: route, AudienceChanged: audience, SettingsChanged: settings}
	arg.Applied, _ = json.Marshal(applied)
	if content {
		vb, va := before.version, int32(saved.Version)
		arg.VersionBefore, arg.VersionAfter = &vb, &va
	}
	if route {
		arg.RouteBefore, arg.RouteAfter = orNull(before.route), orNull(after.route)
	}
	if audience {
		arg.AudienceBefore, arg.AudienceAfter = orNull(before.audience), orNull(after.audience)
	}
	if settings {
		arg.SettingsBefore, arg.SettingsAfter = orNull(before.settings), orNull(after.settings)
	}
	if err := s.Q.ReadinessInsertFixBatch(ctx, arg); err != nil {
		return nil, err
	}
	row, err := s.Q.ReadinessFixBatch(ctx, store.ReadinessFixBatchParams{WorkspaceID: ws.ID, CampaignID: campaignID, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	v := batchView(ctx, row)
	return &v, nil
}

// batchView is a kept batch as a reader sees it.
func batchView(ctx context.Context, b store.ReadinessFixBatchRow) ReadinessFixBatch {
	out := ReadinessFixBatch{ID: b.ID.String(), AppliedAt: b.AppliedAt, AppliedBy: b.AppliedByName, Applied: []string{}, Undoable: b.Blocked == "", Route: b.RouteChanged}
	var applied []string
	_ = json.Unmarshal(b.Applied, &applied)
	out.Count = len(applied)
	if string(i18n.LangFrom(ctx).Locale) == b.AppliedLang {
		out.Applied = applied
	}
	if b.VersionAfter != nil {
		out.Version = int(*b.VersionAfter)
	}
	return out
}

// leftAsIs reports whether the campaign is still as the batch left it:
// nothing saved, routed, targeted or set differently since.
func leftAsIs(b store.ReadinessFixBatchRow, version int32, route, audience, settings json.RawMessage) bool {
	switch {
	case b.VersionAfter != nil && version != *b.VersionAfter:
		return false
	case b.RouteChanged && !sameJSON(route, b.RouteAfter):
		return false
	case b.AudienceChanged && !sameJSON(audience, b.AudienceAfter):
		return false
	case b.SettingsChanged && !sameJSON(settings, b.SettingsAfter):
		return false
	}
	return true
}

// UndoableFixes is the campaign's latest batch of fixes while Undo can put
// it back: not undone, nothing in it Undo can't restore, and the campaign
// still as it left it. Nil otherwise.
func (s *Service) UndoableFixes(ctx context.Context, c store.ReadinessCampaignRow) (*ReadinessFixBatch, error) {
	b, err := s.Q.ReadinessLatestFixBatch(ctx, c.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if b.UndoneAt != nil || b.Blocked != "" || !editable(c.Status) || !leftAsIs(store.ReadinessFixBatchRow(b), c.Version, c.RouteOverride, c.Audience, c.ContentSettings) {
		return nil, nil
	}
	v := batchView(ctx, store.ReadinessFixBatchRow(b))
	return &v, nil
}

// UndoFixes puts a campaign back as it was before a batch of fixes. It
// fails with 409 when the batch was undone already, when it changed what
// Undo can't put back, or when the campaign changed since (a save, another
// batch, an audience or route edit): undoing would lose that change. A
// batch that changed the route needs the right to send or to manage
// providers, as applying it did.
func (s *Service) UndoFixes(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID, batchID uuid.UUID) (ReadinessUndoResult, error) {
	c, err := s.Campaign(ctx, ws, campaignID)
	if err != nil {
		return ReadinessUndoResult{}, err
	}
	b, err := s.Q.ReadinessFixBatch(ctx, store.ReadinessFixBatchParams{WorkspaceID: ws.ID, CampaignID: campaignID, ID: batchID})
	if err != nil {
		return ReadinessUndoResult{}, api.MapError(err, "Fixes")
	}
	var applied []string
	_ = json.Unmarshal(b.Applied, &applied)
	out := ReadinessUndoResult{Count: len(applied), Version: int(c.Version)}
	switch {
	case !editable(c.Status):
		return out, api.Conflict(say(ctx, "problem.alreadySending", nil))
	case b.UndoneAt != nil:
		return out, api.Err(http.StatusConflict, "fixes_undone", say(ctx, "problem.fixesUndone", nil))
	case b.Blocked != "":
		return out, api.Err(http.StatusConflict, "fixes_not_undoable", say(ctx, "problem.fixesNotUndoable", A{"why": b.Blocked}))
	case !leftAsIs(b, c.Version, c.RouteOverride, c.Audience, c.ContentSettings):
		return out, changedSince(ctx)
	}
	if latest, err := s.Q.ReadinessLatestFixBatch(ctx, campaignID); err == nil && latest.ID != b.ID && latest.UndoneAt == nil {
		// A later batch of fixes is the campaign as it stands: it goes first.
		return out, changedSince(ctx)
	}
	if b.RouteChanged && !p.Can(authz.CampaignsSend) && !p.Can(authz.ProvidersManage) {
		helpers := access.Helpers(ctx, s.Q, ws.ID, authz.CampaignsSend)
		names := "none"
		if len(helpers) > 0 {
			names = i18n.OrList(i18n.LangFrom(ctx).Locale, helpers)
		}
		pr := api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, "problem.undoRouteNeedsSend", A{"names": names}))
		pr.Permission, pr.Helpers = string(authz.CampaignsSend), helpers
		return out, pr
	}
	// One undo at a time: the batch is claimed before anything changes.
	now := s.now()
	n, err := s.Q.ReadinessClaimFixUndo(ctx, store.ReadinessClaimFixUndoParams{UndoneAt: &now, UndoneBy: &p.UserID, ID: b.ID})
	if err != nil {
		return out, err
	}
	if n == 0 {
		return out, api.Err(http.StatusConflict, "fixes_undone", say(ctx, "problem.fixesUndone", nil))
	}
	release := func(err error) (ReadinessUndoResult, error) {
		if _, rerr := s.Q.ReadinessReleaseFixUndo(context.WithoutCancel(ctx), b.ID); rerr != nil {
			err = errors.Join(err, rerr)
		}
		return out, err
	}
	// The content first, through the composer's restore: a version of its
	// own and an audit entry, and a conflict when anyone saved since.
	if b.VersionBefore != nil && b.VersionAfter != nil {
		if s.Campaigns == nil {
			return release(api.Conflict(say(ctx, "problem.noContentFixes", nil)))
		}
		// The version comes back as it was saved: with the content settings
		// it had, when the batch changed them (Minify).
		var settings *campaigns.ComposerContentSettings
		if b.SettingsChanged {
			settings = &campaigns.ComposerContentSettings{}
			if raw := orNull(b.SettingsBefore); raw != nil {
				_ = json.Unmarshal(raw, settings)
			}
		}
		saved, err := s.Campaigns.RestoreExactly(ctx, ws, p, campaignID, int(*b.VersionBefore), int(*b.VersionAfter), settings)
		if err != nil {
			var pr *api.Problem
			if errors.As(err, &pr) && pr.Code == campaigns.CodeVersionConflict {
				err = changedSince(ctx)
			}
			return release(err)
		}
		out.Restored, out.Version = int(*b.VersionBefore), saved.Version
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ReadinessLockCampaign(ctx, store.ReadinessLockCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
		if err != nil {
			return api.MapError(err, "Campaign")
		}
		// Checked again under the lock: nobody changed them meanwhile.
		if (b.RouteChanged && !sameJSON(cur.RouteOverride, b.RouteAfter)) || (b.AudienceChanged && !sameJSON(cur.Audience, b.AudienceAfter)) ||
			(b.SettingsChanged && !sameJSON(cur.ContentSettings, b.SettingsAfter)) {
			return changedSince(ctx)
		}
		meta := map[string]any{"source": "send_readiness", "undo": b.ID.String()}
		if b.SettingsChanged {
			settings := orNull(b.SettingsBefore)
			if settings == nil {
				settings = []byte("{}")
			}
			if err := q.ReadinessSetContentSettings(ctx, store.ReadinessSetContentSettingsParams{ContentSettings: settings, WorkspaceID: ws.ID, ID: campaignID}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"settings": json.RawMessage(cur.ContentSettings)}, After: map[string]any{"settings": json.RawMessage(settings)}, Metadata: meta}); err != nil {
				return err
			}
		}
		if b.RouteChanged {
			route := orNull(b.RouteBefore)
			if err := q.ReadinessSetRouteOverride(ctx, store.ReadinessSetRouteOverrideParams{RouteOverride: route, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: campaignID}); err != nil {
				return err
			}
			var was, back any
			if r := orNull(cur.RouteOverride); r != nil {
				was = json.RawMessage(r)
			}
			if route != nil {
				back = json.RawMessage(route)
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.route_override", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"route_override": was}, After: map[string]any{"route_override": back}, Metadata: meta}); err != nil {
				return err
			}
		}
		if b.AudienceChanged {
			audience := orNull(b.AudienceBefore)
			if audience == nil {
				audience = []byte("{}")
			}
			if err := q.ReadinessSetAudience(ctx, store.ReadinessSetAudienceParams{Audience: audience, AudienceCount: c.AudienceCount, UpdatedBy: &p.UserID, Now: now,
				WorkspaceID: ws.ID, ID: campaignID}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"audience": json.RawMessage(cur.Audience)}, After: map[string]any{"audience": json.RawMessage(audience)}, Metadata: meta}); err != nil {
				return err
			}
		}
		after := map[string]any{"fixes": len(applied), "route": b.RouteChanged, "audience": b.AudienceChanged, "settings": b.SettingsChanged}
		if out.Restored > 0 {
			after["restored"], after["version"] = out.Restored, out.Version
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "readiness.fixes_undone", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"applied": applied, "appliedAt": b.AppliedAt}, After: after, Metadata: map[string]any{"batch": b.ID.String()}})
	})
	if err != nil {
		return release(err)
	}
	s.changed(ctx, ws, campaignID)
	return out, nil
}

// changedSince is the 409 of an undo the campaign moved past.
func changedSince(ctx context.Context) error {
	return api.Err(http.StatusConflict, "fixes_changed", say(ctx, "problem.fixesChanged", nil))
}
