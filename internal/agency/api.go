package agency

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// The Agency area's operations need a signed-in person on the agency's
// staff holding the agency permission each names (instance-level, so the
// service checks it: workspace roles, API keys and MCP agents never hold
// them). Instance admins manage agency mode itself.

func staffOp(perm authz.Permission) api.Access {
	return api.Access{Kind: api.AccessUser, HumansOnly: true, Permission: perm}
}

// AgencyWorkspacePath names one of the agency's workspaces.
type AgencyWorkspacePath struct {
	WorkspaceID string `path:"workspaceId" format:"uuid"`
}

// AgencyStaffPath names a staff member.
type AgencyStaffPath struct {
	StaffID string `path:"staffId" maxLength:"80" doc:"A staff id, or admin:<user id> for an instance admin"`
}

// AgencyTemplatePath names a workspace template.
type AgencyTemplatePath struct {
	TemplateID string `path:"templateId" format:"uuid"`
}

// AgencyItemPath names a library item.
type AgencyItemPath struct {
	ItemID string `path:"itemId" format:"uuid"`
}

// AgencyCopyPath names a copy of a library item.
type AgencyCopyPath struct {
	CopyID string `path:"copyId" format:"uuid"`
}

// Register adds the agency's operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Agency"}
	view, usage, library, manage := staffOp(authz.AgencyView), staffOp(authz.AgencyUsage), staffOp(authz.AgencyLibrary), staffOp(authz.AgencyManage)

	api.Get(h, api.Op{ID: "get-agency-me", Path: "/agency/me", Summary: "The agency, as the app shell shows it: its clients for staff, its name for everyone", Tags: tags,
		Access: api.Access{Kind: api.AccessUser, HumansOnly: true}},
		func(ctx context.Context, _ *struct{}) (*api.Body[AgencyMe], error) {
			out, err := s.Me(ctx)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-overview", Path: "/agency/overview", Summary: "The agency overview: clients, approvals waiting, incidents and the next sends", Tags: tags, Access: view},
		func(ctx context.Context, _ *struct{}) (*api.Body[AgencyOverviewView], error) {
			out, err := s.Overview(ctx)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-client", Path: "/agency/clients/{workspaceId}", Summary: "One client's row: health, next send, approvals, incidents and setup", Tags: tags, Access: view},
		func(ctx context.Context, in *AgencyWorkspacePath) (*api.Body[AgencyClientPulse], error) {
			out, err := s.Pulse(ctx, in.WorkspaceID)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-agency-client", Path: "/agency/clients", Summary: "Create a client workspace, from a template, with its first person invited", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct{ Body AgencyClientNew }) (*api.Body[AgencyClientCreated], error) {
			out, err := s.CreateClient(ctx, in.Body)
			return api.OK(out), err
		})

	api.Patch(h, api.Op{ID: "update-agency-client", Path: "/agency/clients/{workspaceId}", Summary: "Change a client's contact at the agency", Tags: tags, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			WorkspaceID string `path:"workspaceId" format:"uuid"`
			Body        AgencyClientInput
		}) (*api.Empty, error) {
			return api.NoContent, s.UpdateClient(ctx, in.WorkspaceID, in.Body)
		})

	api.Post(h, api.Op{ID: "delete-agency-client", Path: "/agency/clients/{workspaceId}/delete", Summary: "Delete a client workspace (typed name; restorable for 7 days)", Tags: tags, Access: manage},
		func(ctx context.Context, in *struct {
			WorkspaceID string `path:"workspaceId" format:"uuid"`
			Body        AgencyClientDelete
		}) (*api.Body[AgencyClientDeleted], error) {
			out, err := s.DeleteClient(ctx, in.WorkspaceID, in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-client-details", Path: "/agency/clients/{workspaceId}/details", Summary: "What the agency keeps about a client: contact, rate, people, last months", Tags: tags, Access: view},
		func(ctx context.Context, in *AgencyWorkspacePath) (*api.Body[AgencyClientDetails], error) {
			out, err := s.ClientDetails(ctx, in.WorkspaceID)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-client-setup", Path: "/agency/clients/{workspaceId}/setup", Summary: "The last workspace template applied to a client, and what it added", Tags: tags, Access: view},
		func(ctx context.Context, in *AgencyWorkspacePath) (*api.Body[AgencyTemplatePlan], error) {
			out, err := s.LatestApplication(ctx, in.WorkspaceID)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "set-agency-client-rate", Path: "/agency/clients/{workspaceId}/rate", Summary: "Set a client's rebill rate (applies from the 1st of this month)", Tags: tags, Access: usage},
		func(ctx context.Context, in *struct {
			WorkspaceID string `path:"workspaceId" format:"uuid"`
			Body        AgencyRateInput
		}) (*api.Body[AgencyRateSaved], error) {
			out, err := s.SetRate(ctx, in.WorkspaceID, in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "search-agency", Path: "/agency/search", Summary: "Search your clients: names, or a contact by exact email (recorded in the audit log)", Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			Q string `query:"q" maxLength:"200"`
		}) (*api.Body[AgencySearchResults], error) {
			out, err := s.Search(ctx, in.Q)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-staff", Path: "/agency/staff", Summary: "The agency's staff and the assignment matrix", Tags: tags, Access: view},
		func(ctx context.Context, _ *struct{}) (*api.Body[AgencyStaffView], error) {
			out, err := s.StaffView(ctx)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "list-agency-staff-candidates", Path: "/agency/staff/candidates", Summary: "Accounts of this instance that can join the staff", Tags: tags, Access: manage},
		func(ctx context.Context, _ *struct{}) (*api.Body[[]AgencyStaffCandidate], error) {
			out, err := s.Candidates(ctx)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "add-agency-staff", Path: "/agency/staff", Summary: "Add someone to the staff, from this instance's accounts or by invitation", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct{ Body AgencyStaffInput }) (*api.Body[AgencyStaffAdded], error) {
			out, err := s.AddStaff(ctx, in.Body)
			return api.OK(out), err
		})

	api.Patch(h, api.Op{ID: "update-agency-staff", Path: "/agency/staff/{staffId}", Summary: "Change a staff member's level or title", Tags: tags, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			StaffID string `path:"staffId" maxLength:"80"`
			Body    AgencyStaffUpdate
		}) (*api.Empty, error) {
			return api.NoContent, s.UpdateStaff(ctx, in.StaffID, in.Body)
		})

	api.Post(h, api.Op{ID: "remove-agency-staff", Path: "/agency/staff/{staffId}/remove", Summary: "Take someone off the staff, and optionally out of the agency's workspaces", Tags: tags, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			StaffID string `path:"staffId" maxLength:"80"`
			Body    AgencyStaffRemoveInput
		}) (*api.Empty, error) {
			return api.NoContent, s.RemoveStaff(ctx, in.StaffID, in.Body)
		})

	api.Put(h, api.Op{ID: "assign-agency-staff", Path: "/agency/assignments", Summary: "Assign a staff member to a workspace with a role (only roles whose permissions you hold there)", Tags: tags, Access: view, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct{ Body AgencyAssignInput }) (*api.Empty, error) {
			return api.NoContent, s.Assign(ctx, in.Body)
		})

	api.Post(h, api.Op{ID: "unassign-agency-staff", Path: "/agency/assignments/remove", Summary: "Remove an assignment: the membership goes; the keys they made are kept or revoked", Tags: tags, Access: view, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct{ Body AgencyUnassignInput }) (*api.Empty, error) {
			return api.NoContent, s.Unassign(ctx, in.Body)
		})

	api.Get(h, api.Op{ID: "get-agency-assignment-keys", Path: "/agency/assignments/keys", Summary: "The API keys and agents a staff member created in a workspace", Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			StaffID     string `query:"staffId" maxLength:"80"`
			WorkspaceID string `query:"workspaceId" format:"uuid"`
		}) (*api.Body[AgencyAssignmentKeys], error) {
			out, err := s.AssignmentKeys(ctx, in.StaffID, in.WorkspaceID)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "list-agency-templates", Path: "/agency/templates", Summary: "Workspace templates, what each sets up and the clients using it", Tags: tags, Access: view},
		func(ctx context.Context, _ *struct{}) (*api.Body[[]AgencyTemplate], error) {
			out, err := s.Templates(ctx)
			return api.OK(out), err
		})

	api.Patch(h, api.Op{ID: "update-agency-template", Path: "/agency/templates/{templateId}", Summary: "Rename a workspace template or leave parts out of it", Tags: tags, Access: library, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			TemplateID string `path:"templateId" format:"uuid"`
			Body       AgencyTemplateInput
		}) (*api.Empty, error) {
			return api.NoContent, s.UpdateTemplate(ctx, in.TemplateID, in.Body)
		})

	api.Delete(h, api.Op{ID: "delete-agency-template", Path: "/agency/templates/{templateId}", Summary: "Remove a workspace template (clients keep everything it set up)", Tags: tags, Access: library},
		func(ctx context.Context, in *AgencyTemplatePath) (*api.Empty, error) {
			return api.NoContent, s.DeleteTemplate(ctx, in.TemplateID)
		})

	api.Get(h, api.Op{ID: "plan-agency-template", Path: "/agency/templates/{templateId}/plan", Summary: "What applying a template to a client would add and rename", Tags: tags, Access: library},
		func(ctx context.Context, in *struct {
			TemplateID  string `path:"templateId" format:"uuid"`
			WorkspaceID string `query:"workspaceId" format:"uuid"`
		}) (*api.Body[AgencyTemplatePlan], error) {
			out, err := s.TemplatePlan(ctx, in.TemplateID, in.WorkspaceID)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "apply-agency-template", Path: "/agency/templates/{templateId}/apply", Summary: "Apply a template to a client: adds what's missing, renames clashes, never overwrites", Tags: tags, Access: library},
		func(ctx context.Context, in *struct {
			TemplateID string `path:"templateId" format:"uuid"`
			Body       AgencyCopyInput
		}) (*api.Body[AgencyTemplatePlan], error) {
			out, err := s.ApplyTemplate(ctx, in.TemplateID, in.Body.WorkspaceID)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-library", Path: "/agency/library", Summary: "The shared library, most used first", Tags: tags, Access: view},
		func(ctx context.Context, _ *struct{}) (*api.Body[AgencyLibraryView], error) {
			out, err := s.Library(ctx)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-library-item", Path: "/agency/library/{itemId}", Summary: "A library item: versions, fields it needs, copies and their diff", Tags: tags, Access: view},
		func(ctx context.Context, in *AgencyItemPath) (*api.Body[AgencyLibraryDetail], error) {
			out, err := s.LibraryItem(ctx, in.ItemID)
			return api.OK(out), err
		})

	api.Patch(h, api.Op{ID: "update-agency-library-item", Path: "/agency/library/{itemId}", Summary: "Rename a library item, or publish a client's copy as its next version", Tags: tags, Access: library, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			ItemID string `path:"itemId" format:"uuid"`
			Body   AgencyLibraryEdit
		}) (*api.Empty, error) {
			return api.NoContent, s.EditLibraryItem(ctx, in.ItemID, in.Body)
		})

	api.Delete(h, api.Op{ID: "remove-agency-library-item", Path: "/agency/library/{itemId}", Summary: "Remove a library item (copies in clients stay)", Tags: tags, Access: library},
		func(ctx context.Context, in *AgencyItemPath) (*api.Empty, error) {
			return api.NoContent, s.RemoveLibraryItem(ctx, in.ItemID)
		})

	api.Post(h, api.Op{ID: "copy-agency-library-item", Path: "/agency/library/{itemId}/copies", Summary: "Copy a library item into a client (needs its edit permission there)", Tags: tags, Access: view, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			ItemID string `path:"itemId" format:"uuid"`
			Body   AgencyCopyInput
		}) (*api.Body[AgencyCopyResult], error) {
			out, err := s.CopyToClient(ctx, in.ItemID, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "update-agency-library-copy", Path: "/agency/library/copies/{copyId}/update", Summary: "Bring a client's copy to the library's latest version", Tags: tags, Access: view, Status: http.StatusNoContent},
		func(ctx context.Context, in *AgencyCopyPath) (*api.Empty, error) {
			return api.NoContent, s.UpdateCopy(ctx, in.CopyID)
		})

	api.Get(h, api.Op{ID: "get-agency-usage", Path: "/agency/usage", Summary: "Usage per client for a month, with provider costs and what to rebill", Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			Month string `query:"month" pattern:"^\\d{4}-\\d{2}$" doc:"2026-09; the last full month by default"`
		}) (*api.Body[AgencyUsageView], error) {
			out, err := s.Usage(ctx, in.Month)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "export-agency-usage", Path: "/agency/usage/export", Summary: "Usage as CSV: one row per client and channel (recorded in the audit log)", Tags: tags, Access: usage},
		func(ctx context.Context, in *struct {
			Month string `query:"month" pattern:"^\\d{4}-\\d{2}$"`
		}) (*huma.StreamResponse, error) {
			if _, _, err := s.require(ctx, authz.AgencyUsage); err != nil {
				return nil, err
			}
			from, err := s.parseMonth(ctx, in.Month)
			if err != nil {
				return nil, err
			}
			return &huma.StreamResponse{Body: func(hctx huma.Context) {
				hctx.SetHeader("Content-Type", "text/csv; charset=utf-8")
				hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="agency-usage-%s.csv"`, from.Format("2006-01")))
				hctx.SetHeader("Cache-Control", "no-store")
				hctx.SetStatus(http.StatusOK)
				_ = s.ExportUsage(ctx, from.Format("2006-01"), hctx.BodyWriter())
			}}, nil
		})

	admin := []string{"Instance admin · Instance & updates"}
	api.Get(h, api.Op{ID: "get-agency-mode", Path: "/admin/agency", Summary: "Agency mode: on or off, the agency's name, its workspace, clients and default template", Tags: admin, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[AgencyModeView], error) {
			out, err := s.Mode(ctx)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-agency-mode", Path: "/admin/agency", Summary: "Turn agency mode on or off and choose its workspaces", Tags: admin, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct{ Body AgencyModeInput }) (*api.Body[AgencyModeView], error) {
			out, err := s.SaveMode(ctx, in.Body)
			return api.OK(out), err
		})

	wsTags := []string{"Agency"}
	base := "/workspaces/{workspace}/agency"
	api.Get(h, api.Op{ID: "get-workspace-agency", Path: base, Summary: "The agency managing this workspace: “Managed by”, the contact, staff among the members", Tags: wsTags, Access: api.Workspace("")},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WorkspaceAgency], error) {
			out, err := s.Workspace(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "save-agency-template", Path: base + "/template", Summary: "Save this workspace's setup as a workspace template (agency:library)", Tags: wsTags, Access: api.WorkspaceHuman(""), Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body AgencyTemplateFromWorkspace
		}) (*api.Body[AgencyTemplateRef], error) {
			out, err := s.SaveAsTemplate(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "add-agency-library-item", Path: base + "/library", Summary: "Add an item of this workspace to the agency's library (agency:library and its view permission here)", Tags: wsTags, Access: api.WorkspaceHuman(""), Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body AgencyAddToLibrary
		}) (*api.Body[AgencyLibraryAdded], error) {
			out, err := s.AddToLibrary(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-agency-gallery", Path: base + "/library", Summary: "The agency library tab of this workspace's gallery and automation starters", Tags: wsTags, Access: api.WorkspaceHuman("")},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Kind string `query:"kind" enum:"composer,automation,transactional,sms" default:"composer"`
		}) (*api.Body[AgencyGalleryView], error) {
			out, err := s.Gallery(ctx, api.WorkspaceFrom(ctx), in.Kind)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "use-agency-library-item", Path: base + "/library/{itemId}/use", Summary: "Copy a library item into this workspace", Tags: wsTags, Access: api.WorkspaceHuman(""), Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ItemID string `path:"itemId" format:"uuid"`
		}) (*api.Body[AgencyCopyResult], error) {
			out, err := s.UseItem(ctx, api.WorkspaceFrom(ctx), in.ItemID)
			return api.OK(out), err
		})
}
