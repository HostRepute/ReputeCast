package products

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// TemplateKey is the delivery template every new product starts with.
const TemplateKey = "product-delivery"

// CategoryName is the Essential transactional category of buyers' emails,
// in English; a workspace that writes in another language gets it in that
// language (the workspace's content from then on).
const CategoryName = "Purchases"

// ensureTemplate makes the delivery template (product-delivery, a draft
// in the Essential "Purchases" category) when the workspace has none, and
// beside it the notice of a bank payment that failed after checkout
// (product-payment-failed). ReputeCast makes them, so a person who can't
// change Transactional settings can still start a product; publishing them
// stays theirs. Their names, their category's and their content are in the
// workspace's default language.
func (s *Service) ensureTemplate(ctx context.Context, ws *store.Workspace) (*uuid.UUID, error) {
	id, err := s.ensureStarter(ctx, ws, TemplateKey, "template.deliveryName")
	if err != nil || id == nil {
		return id, err
	}
	if _, err := s.ensureStarter(ctx, ws, FailedTemplateKey, "template.failedName"); err != nil {
		return nil, err
	}
	return id, nil
}

// ensureStarter makes the template key from its starter, named by the
// label name, unless the workspace has it.
func (s *Service) ensureStarter(ctx context.Context, ws *store.Workspace, key, name string) (*uuid.UUID, error) {
	if t, err := s.Q.ProductTemplateByKey(ctx, store.ProductTemplateByKeyParams{WorkspaceID: ws.ID, Key: key}); err == nil {
		return &t.ID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if s.Transactional == nil || s.Transactional() == nil {
		return nil, nil
	}
	sys := &authz.Principal{Type: authz.ActorSystem, Name: "ReputeCast", WorkspaceID: ws.ID,
		Permissions: authz.Set{authz.TransactionalPublish: {}, authz.TransactionalEdit: {}}}
	cat, err := s.purchasesCategory(ctx, ws, sys)
	if err != nil {
		return nil, err
	}
	l := pagelabels.Lang(s.defaultLanguage(ctx, ws.ID))
	t, err := s.Transactional().CreateTemplate(ctx, ws, sys, transactional.TransactionalTemplateCreate{Name: word(l, name, nil), Key: key, Starter: key,
		CategoryID: cat.String(), Origin: map[string]any{"source": "products"}})
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// purchasesCategory is the "Purchases" category, made when missing.
func (s *Service) purchasesCategory(ctx context.Context, ws *store.Workspace, sys *authz.Principal) (uuid.UUID, error) {
	cats, err := s.Q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	l := pagelabels.Lang(s.defaultLanguage(ctx, ws.ID))
	name := word(l, "category.purchases", nil)
	for _, c := range cats {
		if c.Name == CategoryName || c.Name == name {
			return c.ID, nil
		}
	}
	c, err := s.Transactional().CreateCategory(ctx, ws, sys, transactional.TransactionalCategoryInput{Name: name, Kind: transactional.KindEssential,
		Description: word(l, "category.purchasesText", nil)})
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(c.ID)
}
