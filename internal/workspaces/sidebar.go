package workspaces

import (
	"context"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sidebar sections (Settings › General): a workspace hides the sections it
// doesn't use from the sidebar, the icon rail, the phone tab bar and the More
// sheet. Nothing else changes: the screens open by link and keep working.

// SidebarSections are the sections a workspace can hide, in sidebar order,
// with the names the audit log uses.
var SidebarSections = []struct{ Key, Name string }{
	{"transactional", "Transactional"},
	{"grow", "Grow"},
	{"earn", "Earn"},
}

// WorkspaceSidebar is what the sidebar hides.
type WorkspaceSidebar struct {
	HiddenSections []string `json:"hiddenSections" enum:"transactional,grow,earn" doc:"Sections hidden from the sidebar, in sidebar order"`
}

// WorkspaceSidebarSectionInput shows or hides one section.
type WorkspaceSidebarSectionInput struct {
	Shown bool `json:"shown" doc:"Show the section in the sidebar; false hides it"`
}

// HiddenSections returns the workspace's hidden sections in sidebar order,
// leaving out names this version doesn't know.
func HiddenSections(ws *store.Workspace) []string {
	out := []string{}
	for _, s := range SidebarSections {
		if slices.Contains(ws.HiddenSections, s.Key) {
			out = append(out, s.Key)
		}
	}
	return out
}

func registerSidebar(h huma.API, s *Service, tags []string, base string) {
	api.Put(h, api.Op{ID: "set-sidebar-section", Path: base + "/settings/sidebar/{section}", Summary: "Show or hide a section in the sidebar (members can still open it by link)", Tags: tags, Access: api.Workspace(authz.WorkspaceManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Section string `path:"section" enum:"transactional,grow,earn" doc:"The section"`
			Body    WorkspaceSidebarSectionInput
		}) (*api.Body[WorkspaceSidebar], error) {
			out, err := s.SetSidebarSection(ctx, api.WorkspaceFrom(ctx), in.Section, in.Body.Shown)
			return api.OK(out), err
		})
}

// SetSidebarSection shows or hides one section. Setting the state it already
// has changes nothing and writes no audit entry.
func (s *Service) SetSidebarSection(ctx context.Context, ws *store.Workspace, section string, shown bool) (WorkspaceSidebar, error) {
	name := ""
	for _, sec := range SidebarSections {
		if sec.Key == section {
			name = sec.Name
		}
	}
	if name == "" {
		return WorkspaceSidebar{}, api.Invalid("path.section", "Pick Transactional, Grow or Earn.")
	}
	var out WorkspaceSidebar
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		current, err := q.SidebarHiddenSectionsForUpdate(ctx, ws.ID)
		if err != nil {
			return api.MapError(err, "Workspace")
		}
		before := HiddenSections(&store.Workspace{HiddenSections: current})
		next := []string{}
		for _, sec := range SidebarSections {
			hide := slices.Contains(before, sec.Key)
			if sec.Key == section {
				hide = !shown
			}
			if hide {
				next = append(next, sec.Key)
			}
		}
		out.HiddenSections = next
		if slices.Equal(before, next) {
			return nil
		}
		if err := q.SidebarSetHiddenSections(ctx, store.SidebarSetHiddenSectionsParams{ID: ws.ID, HiddenSections: next}); err != nil {
			return err
		}
		summary := name + " hidden from the sidebar"
		if shown {
			summary = name + " shown in the sidebar"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.sidebar_updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"hidden_sections": before}, After: map[string]any{"hidden_sections": next},
			Metadata: map[string]any{"summary": summary, "section": section, "permission": string(authz.WorkspaceManage)}})
	})
	return out, err
}
