package audience

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// AudienceListInput creates or renames a list.
type AudienceListInput struct {
	Name        string `json:"name" minLength:"1" maxLength:"100"`
	Description string `json:"description,omitempty" maxLength:"500"`
	DoubleOptIn *bool  `json:"doubleOptIn,omitempty" doc:"New signups confirm by email first (default true)"`
}

// CreateList adds a list.
func (s *Service) CreateList(ctx context.Context, ws *store.Workspace, in AudienceListInput) (AudienceListSummary, error) {
	return s.createList(ctx, s.Pool, ws, in)
}

type beginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

func (s *Service) createList(ctx context.Context, conn beginner, ws *store.Workspace, in AudienceListInput) (AudienceListSummary, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return AudienceListSummary{}, api.Invalid("body.name", "Give the list a name.")
	}
	double := true
	if in.DoubleOptIn != nil {
		double = *in.DoubleOptIn
	}
	var l store.List
	tx, err := conn.Begin(ctx)
	if err != nil {
		return AudienceListSummary{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	q := store.New(tx)
	taken, err := q.ListNameTaken(ctx, store.ListNameTakenParams{WorkspaceID: ws.ID, Name: name, ExceptID: uuid.Nil})
	if err != nil {
		return AudienceListSummary{}, err
	}
	if taken {
		return AudienceListSummary{}, api.Conflict(fmt.Sprintf("A list called %q already exists.", name))
	}
	l, err = q.CreateList(ctx, store.CreateListParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Name: name, Description: strings.TrimSpace(in.Description), DoubleOptIn: double})
	if err != nil {
		return AudienceListSummary{}, err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.create", TargetType: "list", TargetID: l.ID.String(), TargetName: l.Name, After: in}); err != nil {
		return AudienceListSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AudienceListSummary{}, err
	}
	return AudienceListSummary{ID: l.ID.String(), Name: l.Name, Description: l.Description, DoubleOptIn: l.DoubleOptIn, CreatedAt: l.CreatedAt}, nil
}

// UpdateList renames a list or changes its opt-in setting.
func (s *Service) UpdateList(ctx context.Context, ws *store.Workspace, id uuid.UUID, in AudienceListInput) (AudienceListSummary, error) {
	var out AudienceListSummary
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "List")
		}
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return api.Invalid("body.name", "Give the list a name.")
		}
		taken, err := q.ListNameTaken(ctx, store.ListNameTakenParams{WorkspaceID: ws.ID, Name: name, ExceptID: id})
		if err != nil {
			return err
		}
		if taken {
			return api.Conflict(fmt.Sprintf("A list called %q already exists.", name))
		}
		double := before.DoubleOptIn
		if in.DoubleOptIn != nil {
			double = *in.DoubleOptIn
		}
		l, err := q.UpdateList(ctx, store.UpdateListParams{WorkspaceID: ws.ID, ID: id, Name: name, Description: strings.TrimSpace(in.Description), DoubleOptIn: double})
		if err != nil {
			return err
		}
		out = AudienceListSummary{ID: l.ID.String(), Name: l.Name, Description: l.Description, DoubleOptIn: l.DoubleOptIn, CreatedAt: l.CreatedAt}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.update", TargetType: "list", TargetID: id.String(), TargetName: l.Name,
			Before: map[string]any{"name": before.Name, "description": before.Description, "doubleOptIn": before.DoubleOptIn},
			After:  map[string]any{"name": l.Name, "description": l.Description, "doubleOptIn": l.DoubleOptIn}})
	})
	return out, err
}

// DeleteList removes a list. Its contacts stay; only their membership of
// this list goes.
func (s *Service) DeleteList(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		l, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "List")
		}
		var members int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM list_memberships WHERE list_id = $1 AND status = 'subscribed'`, id).Scan(&members); err != nil {
			return err
		}
		if _, err := q.SoftDeleteList(ctx, store.SoftDeleteListParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.delete", TargetType: "list", TargetID: id.String(), TargetName: l.Name,
			Before: map[string]any{"name": l.Name, "subscribed": members}})
	})
}

// CustomFieldDef is a custom field definition.
type CustomFieldDef struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	Type      string    `json:"type" enum:"text,number,date,boolean,select"`
	Options   []string  `json:"options"`
	Display   string    `json:"display" enum:",currency"`
	CreatedAt time.Time `json:"createdAt"`
}

// CustomFieldList is the custom fields of a workspace.
type CustomFieldList struct {
	Items []CustomFieldDef `json:"items"`
}

func fieldDef(f store.CustomField) CustomFieldDef {
	d := CustomFieldDef{ID: f.ID.String(), Key: f.Key, Label: f.Label, Type: f.Type, Display: f.Display, CreatedAt: f.CreatedAt, Options: []string{}}
	_ = json.Unmarshal(f.Options, &d.Options)
	if d.Options == nil {
		d.Options = []string{}
	}
	return d
}

// Fields lists the custom fields.
func (s *Service) Fields(ctx context.Context, ws *store.Workspace) (CustomFieldList, error) {
	rows, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return CustomFieldList{}, err
	}
	out := CustomFieldList{Items: make([]CustomFieldDef, 0, len(rows))}
	for _, f := range rows {
		out.Items = append(out.Items, fieldDef(f))
	}
	return out, nil
}

// CustomFieldInput creates or changes a custom field.
type CustomFieldInput struct {
	Key     string   `json:"key,omitempty" maxLength:"63" doc:"Lowercase letters, digits and _; fixed after creation"`
	Label   string   `json:"label" minLength:"1" maxLength:"80"`
	Type    string   `json:"type,omitempty" enum:"text,number,date,boolean,select"`
	Options []string `json:"options,omitempty" maxItems:"100"`
	Display string   `json:"display,omitempty" enum:",currency"`
}

var fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// reservedKeys are built-in contact columns a custom field can't shadow.
var reservedKeys = map[string]bool{"email": true, "first_name": true, "last_name": true, "company": true, "country": true,
	"timezone": true, "language": true, "tags": true, "tier": true, "consent": true, "source": true, "id": true}

// CreateField adds a custom field.
func (s *Service) CreateField(ctx context.Context, ws *store.Workspace, in CustomFieldInput) (CustomFieldDef, error) {
	key := strings.TrimSpace(in.Key)
	if !fieldKeyRe.MatchString(key) {
		return CustomFieldDef{}, api.Invalid("body.key", "Keys use lowercase letters, digits and _, and start with a letter (for example signup_source).")
	}
	if reservedKeys[key] {
		return CustomFieldDef{}, api.Invalid("body.key", fmt.Sprintf("%s is a built-in field.", key))
	}
	if in.Type == "" {
		in.Type = "text"
	}
	if in.Display == "currency" && in.Type != "number" {
		return CustomFieldDef{}, api.Invalid("body.display", "Only number fields can show as money.")
	}
	opts, _ := json.Marshal(nonNil(in.Options))
	var f store.CustomField
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		f, err = q.CreateCustomField(ctx, store.CreateCustomFieldParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Key: key, Label: strings.TrimSpace(in.Label),
			Type: in.Type, Options: opts, Display: in.Display})
		if err != nil {
			if strings.Contains(err.Error(), "custom_fields_key") {
				return api.Conflict(fmt.Sprintf("A field with the key %s already exists.", key))
			}
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "field.create", TargetType: "custom_field", TargetID: f.ID.String(), TargetName: f.Key, After: in})
	})
	return fieldDef(f), err
}

// UpdateField changes a field's label, options or display. Its key and
// type are fixed: changing them would change what stored values mean.
func (s *Service) UpdateField(ctx context.Context, ws *store.Workspace, id uuid.UUID, in CustomFieldInput) (CustomFieldDef, error) {
	var f store.CustomField
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.GetCustomField(ctx, store.GetCustomFieldParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Field")
		}
		if in.Display == "currency" && before.Type != "number" {
			return api.Invalid("body.display", "Only number fields can show as money.")
		}
		opts, _ := json.Marshal(nonNil(in.Options))
		f, err = q.UpdateCustomField(ctx, store.UpdateCustomFieldParams{WorkspaceID: ws.ID, ID: id, Label: strings.TrimSpace(in.Label), Options: opts, Display: in.Display})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "field.update", TargetType: "custom_field", TargetID: id.String(), TargetName: f.Key,
			Before: fieldDef(before), After: fieldDef(f)})
	})
	return fieldDef(f), err
}

// DeleteField removes a field and its values from every contact.
func (s *Service) DeleteField(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := q.GetCustomField(ctx, store.GetCustomFieldParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Field")
		}
		if _, err := q.DeleteCustomField(ctx, store.DeleteCustomFieldParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE contacts SET attributes = attributes - $2::text WHERE workspace_id = $1 AND attributes ? $2::text`, ws.ID, f.Key)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "field.delete", TargetType: "custom_field", TargetID: id.String(), TargetName: f.Key,
			Before: fieldDef(f), Metadata: map[string]any{"valuesRemoved": tag.RowsAffected()}})
	})
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}
