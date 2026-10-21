package pages

import (
	"context"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// The form builder's "Used on pages" (design expansion phase 7, X19): the
// pages whose signup form block uses a form, in their draft or live.

// FormPage is a page that uses a form.
type FormPage struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	URL    string `json:"url"`
	Status string `json:"status" enum:"draft,scheduled,published,unpublished"`
	// Live: the live version uses the form (not only the draft).
	Live bool `json:"live"`
}

// FormPages are the pages that use a form.
type FormPages struct {
	Items []FormPage `json:"items"`
}

// FormPathInput names a form in the path.
type FormPathInput struct {
	api.WorkspaceInput
	Form string `path:"form" format:"uuid" doc:"Form id"`
}

// PagesUsingForm lists the pages whose blocks use a form.
func (s *Service) PagesUsingForm(ctx context.Context, ws *store.Workspace, formID uuid.UUID) (FormPages, error) {
	out := FormPages{Items: []FormPage{}}
	rows, err := s.Q.PagesList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	links := s.links(ctx, ws)
	for _, r := range rows {
		pg := r.Page
		draft := slices.Contains(Analyze(pg.Html).Forms, formID)
		live := false
		if pg.LiveVersion != nil && *pg.LiveVersion != pg.Version {
			if v, err := s.Q.PageVersionGet(ctx, store.PageVersionGetParams{PageID: pg.ID, Version: *pg.LiveVersion}); err == nil {
				live = slices.Contains(Analyze(v.PageVersion.Html).Forms, formID)
			}
		} else if pg.LiveVersion != nil {
			live = draft
		}
		if !draft && !live {
			continue
		}
		slug := pg.Slug
		if pg.LiveSlug != nil && live {
			slug = *pg.LiveSlug
		}
		out.Items = append(out.Items, FormPage{ID: pg.ID.String(), Name: pg.Name, Path: "/" + slug, URL: links.page(slug), Status: pg.Status,
			Live: live && isLive(pg)})
	}
	return out, nil
}

func registerFormPages(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "list-form-pages", Path: "/workspaces/{workspace}/forms/{form}/pages", Summary: "The pages whose signup form block uses a form",
		Tags: []string{"Signup forms"}, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *FormPathInput) (*api.Body[FormPages], error) {
			out, err := s.PagesUsingForm(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Form))
			return api.OK(out), err
		})
}
