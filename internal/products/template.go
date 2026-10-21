package products

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// TemplateKey is the delivery template every new product starts with.
const TemplateKey = "product-delivery"

// CategoryName is the Essential transactional category of buyers' emails.
const CategoryName = "Purchases"

// ensureTemplate makes the delivery template (product-delivery, a draft
// in the Essential "Purchases" category) when the workspace has none.
// ReputeCast makes it, so a person who can't change Transactional
// settings can still start a product; publishing it stays theirs.
func (s *Service) ensureTemplate(ctx context.Context, ws *store.Workspace) (*uuid.UUID, error) {
	if t, err := s.Q.ProductTemplateByKey(ctx, store.ProductTemplateByKeyParams{WorkspaceID: ws.ID, Key: TemplateKey}); err == nil {
		return &t.ID, nil
	}
	if s.Transactional == nil || s.Transactional() == nil {
		return nil, nil
	}
	ts := s.Transactional()
	sys := &authz.Principal{Type: authz.ActorSystem, Name: "ReputeCast", WorkspaceID: ws.ID,
		Permissions: authz.Set{authz.TransactionalPublish: {}, authz.TransactionalEdit: {}}}
	cat, err := s.purchasesCategory(ctx, ws, sys)
	if err != nil {
		return nil, err
	}
	t, err := ts.CreateTemplate(ctx, ws, sys, transactional.TransactionalTemplateCreate{Name: "Product delivery", Key: TemplateKey, Starter: TemplateKey,
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
	for _, c := range cats {
		if c.Name == CategoryName {
			return c.ID, nil
		}
	}
	c, err := s.Transactional().CreateCategory(ctx, ws, sys, transactional.TransactionalCategoryInput{Name: CategoryName, Kind: transactional.KindEssential,
		Description: "Buyers' emails: the download links of digital products"})
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(c.ID)
}
