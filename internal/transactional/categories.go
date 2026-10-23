package transactional

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Category kinds.
const (
	KindEssential = "essential"
	KindOptional  = "optional"
)

// TransactionalCategory groups templates. Essential categories are always
// sent; recipients can turn Optional ones off.
type TransactionalCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind" enum:"essential,optional"`
	Description string `json:"description" doc:"What recipients see in the preference center"`
	Templates   int64  `json:"templates"`
	OptedOut    int64  `json:"optedOut"`
}

// TransactionalCategoryRef names a template's category.
type TransactionalCategoryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind" enum:"essential,optional"`
}

// TransactionalCategoryList is the Categories card.
type TransactionalCategoryList struct {
	Items []TransactionalCategory `json:"items"`
	// CanEdit is transactional:publish; Reason explains a read-only card.
	CanEdit bool   `json:"canEdit"`
	Reason  string `json:"reason,omitempty"`
}

// TransactionalCategoryInput creates or changes a category.
type TransactionalCategoryInput struct {
	Name        string `json:"name" minLength:"1" maxLength:"60"`
	Kind        string `json:"kind" enum:"essential,optional"`
	Description string `json:"description,omitempty" maxLength:"200"`
}

func categoryRef(id *uuid.UUID, name, kind string) *TransactionalCategoryRef {
	if id == nil {
		return nil
	}
	return &TransactionalCategoryRef{ID: id.String(), Name: name, Kind: kind}
}

// Categories lists the workspace's categories.
func (s *Service) Categories(ctx context.Context, ws *store.Workspace, p *authz.Principal) (TransactionalCategoryList, error) {
	rows, err := s.Q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return TransactionalCategoryList{}, err
	}
	out := TransactionalCategoryList{Items: make([]TransactionalCategory, 0, len(rows)), CanEdit: p.Can(authz.TransactionalPublish)}
	for _, r := range rows {
		out.Items = append(out.Items, TransactionalCategory{ID: r.ID.String(), Name: r.Name, Kind: r.Kind, Description: r.Description, Templates: r.Templates, OptedOut: r.OptedOut})
	}
	if !out.CanEdit {
		out.Reason = deniedMessage(ctx, "categories", authz.TransactionalPublish, s.helpers(ctx, ws.ID, authz.TransactionalPublish, 3))
	}
	return out, nil
}

func (s *Service) requireSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal) error {
	if p.Can(authz.TransactionalPublish) {
		return nil
	}
	pr := api.Forbidden(authz.TransactionalPublish, s.helpers(ctx, ws.ID, authz.TransactionalPublish, 3))
	pr.Detail = deniedMessage(ctx, "settings", authz.TransactionalPublish, pr.Helpers)
	return pr
}

func cleanCategory(ctx context.Context, in TransactionalCategoryInput) (TransactionalCategoryInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" {
		return in, api.Invalid("body.name", say(ctx, "category.nameRequired", nil))
	}
	if in.Kind != KindEssential && in.Kind != KindOptional {
		return in, api.Invalid("body.kind", say(ctx, "category.kindPick", nil))
	}
	if in.Kind == KindOptional && in.Description == "" {
		return in, api.Invalid("body.description", say(ctx, "category.descriptionRequired", nil))
	}
	return in, nil
}

func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

// CreateCategory adds a category.
func (s *Service) CreateCategory(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TransactionalCategoryInput) (TransactionalCategory, error) {
	if err := s.requireSettings(ctx, ws, p); err != nil {
		return TransactionalCategory{}, err
	}
	in, err := cleanCategory(ctx, in)
	if err != nil {
		return TransactionalCategory{}, err
	}
	id := newID()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.TxnCategoryInsert(ctx, store.TxnCategoryInsertParams{ID: id, WorkspaceID: ws.ID, Name: in.Name, Kind: in.Kind, Description: in.Description,
			CreatedBy: userRef(p), CreatedAt: s.now()}); err != nil {
			if isUnique(err) {
				return api.Invalid("body.name", say(ctx, "category.nameTaken", i18n.Args{"name": in.Name}))
			}
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.category_created", TargetType: "transactional_category", TargetID: id.String(), TargetName: in.Name,
			After: map[string]any{"name": in.Name, "kind": in.Kind, "description": in.Description}})
	})
	if err != nil {
		return TransactionalCategory{}, err
	}
	return TransactionalCategory{ID: id.String(), Name: in.Name, Kind: in.Kind, Description: in.Description}, nil
}

// UpdateCategory changes a category; its kind can't change while it has templates.
func (s *Service) UpdateCategory(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in TransactionalCategoryInput) (TransactionalCategory, error) {
	if err := s.requireSettings(ctx, ws, p); err != nil {
		return TransactionalCategory{}, err
	}
	in, err := cleanCategory(ctx, in)
	if err != nil {
		return TransactionalCategory{}, err
	}
	var out TransactionalCategory
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnCategory(ctx, store.TxnCategoryParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapErr(ctx, err, "category")
		}
		n, err := q.TxnCategoryTemplateCount(ctx, &id)
		if err != nil {
			return err
		}
		if cur.Kind != in.Kind && n > 0 {
			return api.Err(http.StatusConflict, api.CodeConflict, say(ctx, "category.kindFixed", i18n.Args{"name": cur.Name, "count": n}))
		}
		if err := q.TxnCategoryUpdate(ctx, store.TxnCategoryUpdateParams{WorkspaceID: ws.ID, ID: id, Name: in.Name, Kind: in.Kind, Description: in.Description, UpdatedAt: s.now()}); err != nil {
			if isUnique(err) {
				return api.Invalid("body.name", say(ctx, "category.nameTaken", i18n.Args{"name": in.Name}))
			}
			return err
		}
		out = TransactionalCategory{ID: id.String(), Name: in.Name, Kind: in.Kind, Description: in.Description, Templates: n}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.category_updated", TargetType: "transactional_category", TargetID: id.String(), TargetName: in.Name,
			Before: map[string]any{"name": cur.Name, "kind": cur.Kind, "description": cur.Description},
			After:  map[string]any{"name": in.Name, "kind": in.Kind, "description": in.Description}})
	})
	return out, err
}

// DeleteCategory removes a category without templates.
func (s *Service) DeleteCategory(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	if err := s.requireSettings(ctx, ws, p); err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnCategory(ctx, store.TxnCategoryParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapErr(ctx, err, "category")
		}
		n, err := q.TxnCategoryTemplateCount(ctx, &id)
		if err != nil {
			return err
		}
		if n > 0 {
			return api.Err(http.StatusConflict, api.CodeConflict, say(ctx, "category.hasTemplates", i18n.Args{"name": cur.Name, "count": n}))
		}
		if _, err := q.TxnCategoryDelete(ctx, store.TxnCategoryDeleteParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.category_deleted", TargetType: "transactional_category", TargetID: id.String(), TargetName: cur.Name,
			Before: map[string]any{"name": cur.Name, "kind": cur.Kind, "description": cur.Description}})
	})
}

func userRef(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
