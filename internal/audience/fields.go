package audience

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	deps "github.com/hostrepute/reputecast/internal/dependents"
	"github.com/hostrepute/reputecast/internal/store"
)

// Custom fields on Audience › Lists & fields (review 2026-11 AU-11): what
// uses each, how many contacts have a value, and a delete that waits
// fieldRestoreDays before the values go.

const fieldRestoreDays = 30

// FieldUsage is a field's row on Lists & fields.
type FieldUsage struct {
	ID string `json:"id"`
	// Filled is how many contacts have a value (absent when counting took
	// too long on a large audience).
	Filled *int64     `json:"filled,omitempty"`
	Uses   []deps.Use `json:"uses"`
}

// DeletedField is a field waiting for its values to go.
type DeletedField struct {
	CustomFieldDef
	DeletedAt time.Time `json:"deletedAt"`
	// RestoreUntil is when its values are removed for good.
	RestoreUntil time.Time `json:"restoreUntil"`
}

// FieldsOverview is the fields card: each field's usage and the deleted
// fields that can still be restored.
type FieldsOverview struct {
	Usage   []FieldUsage   `json:"usage"`
	Deleted []DeletedField `json:"deleted"`
}

// FieldsOverview reads what uses each field, how many contacts have a value
// (in one pass, given 3 seconds), and the deleted fields.
func (s *Service) FieldsOverview(ctx context.Context, ws *store.Workspace) (FieldsOverview, error) {
	out := FieldsOverview{Usage: []FieldUsage{}, Deleted: []DeletedField{}}
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	idx, err := deps.Load(ctx, s.Pool, ws.ID)
	if err != nil {
		return out, err
	}
	filled := map[string]int64{}
	counted := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = 3000"); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT k, count(*) FROM contacts c, jsonb_object_keys(c.attributes) k
			WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.attributes <> '{}' GROUP BY k`, ws.ID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				return err
			}
			filled[k] = n
		}
		return rows.Err()
	}) == nil
	for _, f := range fields {
		u := FieldUsage{ID: f.ID.String(), Uses: idx.Field(f.Key)}
		if counted {
			n := filled[f.Key]
			u.Filled = &n
		}
		out.Usage = append(out.Usage, u)
	}
	deleted, err := s.Q.DeletedCustomFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, d := range deleted {
		def := fieldDef(store.CustomField{ID: d.ID, WorkspaceID: d.WorkspaceID, Key: d.Key, Label: d.Label, Type: d.Type, Options: d.Options, Display: d.Display, CreatedAt: d.CreatedAt})
		out.Deleted = append(out.Deleted, DeletedField{CustomFieldDef: def, DeletedAt: d.DeletedAt, RestoreUntil: d.DeletedAt.AddDate(0, 0, fieldRestoreDays)})
	}
	return out, nil
}

// FieldImpact is what deleting a field would affect.
type FieldImpact struct {
	ID    string     `json:"id"`
	Key   string     `json:"key"`
	Label string     `json:"label"`
	Uses  []deps.Use `json:"uses"`
}

// FieldImpact lists what uses a field.
func (s *Service) FieldImpact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (FieldImpact, error) {
	f, err := s.Q.GetCustomField(ctx, store.GetCustomFieldParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return FieldImpact{}, api.MapError(err, "Field")
	}
	idx, err := deps.Load(ctx, s.Pool, ws.ID)
	if err != nil {
		return FieldImpact{}, err
	}
	return FieldImpact{ID: f.ID.String(), Key: f.Key, Label: f.Label, Uses: idx.Field(f.Key)}, nil
}

// DeleteField takes a field out of every form, segment, view and export;
// its values stay on contacts for fieldRestoreDays, so it can be restored
// as it was, and then go.
func (s *Service) DeleteField(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := q.GetCustomField(ctx, store.GetCustomFieldParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Field")
		}
		if err := q.CustomFieldSetAside(ctx, store.CustomFieldSetAsideParams{WorkspaceID: ws.ID, ID: id, At: s.now(), DeletedBy: actorID(ctx)}); err != nil {
			return err
		}
		if _, err := q.DeleteCustomField(ctx, store.DeleteCustomFieldParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "field.delete", TargetType: "custom_field", TargetID: id.String(), TargetName: f.Key,
			Before: fieldDef(f), Metadata: map[string]any{"restorableDays": fieldRestoreDays}})
	})
}

// RestoreField brings a deleted field back with its values.
func (s *Service) RestoreField(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CustomFieldDef, error) {
	var out CustomFieldDef
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := q.CustomFieldRestore(ctx, store.CustomFieldRestoreParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Field")
		}
		if err != nil {
			return err
		}
		out = fieldDef(f)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "field.restore", TargetType: "custom_field", TargetID: id.String(), TargetName: f.Key, After: out})
	})
	return out, err
}

// PurgeDeletedFields strips the values of fields deleted fieldRestoreDays
// ago, a batch of contacts at a time, then forgets the field.
func (s *Service) PurgeDeletedFields(ctx context.Context) error {
	due, err := s.Q.DeletedCustomFieldsDue(ctx, s.now().AddDate(0, 0, -fieldRestoreDays))
	if err != nil {
		return err
	}
	for _, f := range due {
		for {
			res, err := s.Pool.Exec(ctx, `UPDATE contacts SET attributes = attributes - $2::text
				WHERE id IN (SELECT id FROM contacts WHERE workspace_id = $1 AND attributes ? $2::text LIMIT 5000)`, f.WorkspaceID, f.Key)
			if err != nil {
				return fmt.Errorf("audience: strip field %s: %w", f.Key, err)
			}
			if res.RowsAffected() == 0 {
				break
			}
		}
		if err := s.Q.DeletedCustomFieldGone(ctx, f.ID); err != nil {
			return err
		}
		wsID := f.WorkspaceID
		if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "field.values_removed", TargetType: "custom_field", TargetID: f.ID.String(), TargetName: f.Key,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}}); err != nil {
			return err
		}
	}
	return nil
}
