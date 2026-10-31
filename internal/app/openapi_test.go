package app

import (
	"regexp"
	"sort"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// The OpenAPI command registers every operation on an App without config or
// database; wiring that reads them must not panic.
func TestRegisterOperationsWithoutConfig(t *testing.T) {
	hapi := humachi.New(chi.NewRouter(), NewHumaConfig())
	RegisterOperations(hapi)
	if n := len(hapi.OpenAPI().Paths); n < 100 {
		t.Fatalf("only %d paths registered", n)
	}
}

// registeredOperations registers every operation, as the server does, and
// returns them by operation id.
func registeredOperations(t *testing.T) []*huma.Operation {
	t.Helper()
	hapi := humachi.New(chi.NewRouter(), NewHumaConfig())
	RegisterOperations(hapi)
	var ops []*huma.Operation
	for _, item := range hapi.OpenAPI().Paths {
		for _, op := range []*huma.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
			if op != nil {
				ops = append(ops, op)
			}
		}
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].OperationID < ops[j].OperationID })
	return ops
}

var pathParam = regexp.MustCompile(`\{([^}]+)\}`)

// Every {name} in an operation's path is one of its path parameters. Huma
// skips embedded fields of unexported types, so an input that embeds one
// silently loses its path parameters and the handler reads empty values.
func TestOperationsDeclareTheirPathParameters(t *testing.T) {
	for _, op := range registeredOperations(t) {
		declared := map[string]bool{}
		for _, p := range op.Parameters {
			if p.In == "path" {
				declared[p.Name] = true
			}
		}
		for _, m := range pathParam.FindAllStringSubmatch(op.Path, -1) {
			if !declared[m[1]] {
				t.Errorf("%s %s (%s): {%s} isn't a path parameter of its input", op.Method, op.Path, op.OperationID, m[1])
			}
		}
	}
}

// openToMembers are the workspace operations that name no permission on
// purpose, each with why. The middleware lets every member of the workspace
// (and, unless the operation is people-only, its API keys and agents)
// through an operation that names none.
var openToMembers = map[string]string{
	"get-workspace-me":             "the caller's own role, permissions and scope",
	"leave-workspace":              "every member may leave (the service keeps the last Owner)",
	"get-workspace-settings":       "Settings › General, read-only without workspace:manage",
	"get-workspace-agency":         "the “Managed by” line every member sees",
	"get-accessibility-policy":     "whether serious issues warn or block, shown wherever content is checked; changing it needs workspace:manage",
	"list-workspace-languages":     "the languages every editor picks from, read-only without workspace:manage",
	"list-language-catalog":        "languages to add, with counts the languages list already shows every member",
	"get-language-removal":         "what removing a language affects, in counts the languages list already shows every member",
	"get-fallback-impact":          "contacts a new fallback moves, in counts the languages list already shows every member",
	"list-notifications":           "the caller's own notifications",
	"read-notification":            "the caller's own notifications",
	"read-all-notifications":       "the caller's own notifications",
	"suggest-access":               "asking for a missing permission can't need one",
	"request-access":               "asking for a missing permission can't need one",
	"save-agency-template":         "agency staff only: the service checks the instance-level agency:library",
	"add-agency-library-item":      "agency staff only: the service checks agency:library and the item's view permission here",
	"get-agency-gallery":           "agency staff only: the service checks the instance-level agency:view",
	"use-agency-library-item":      "agency staff only: the service checks agency:view and the item's edit permission here",
	"report-wordpress-site":        "WordPress integration keys only, limited to the operations their integration was given",
	"report-woocommerce-checkouts": "WordPress integration keys only, limited to the operations their integration was given",
}

// Every operation declares its access (an operation without one is
// public), and every workspace operation names the permission it needs,
// except the few in openToMembers.
func TestWorkspaceOperationsNameAPermission(t *testing.T) {
	workspaceOps := map[string]bool{}
	for _, op := range registeredOperations(t) {
		acc, ok := api.AccessOf(op)
		if !ok {
			t.Errorf("%s %s (%s) declares no access, so anyone can call it: register it with api.Op and an Access", op.Method, op.Path, op.OperationID)
			continue
		}
		if acc.Kind != api.AccessWorkspace {
			continue
		}
		workspaceOps[op.OperationID] = true
		_, open := openToMembers[op.OperationID]
		switch {
		case acc.Permission == "" && !open:
			t.Errorf("%s %s (%s) names no permission, so every member, API key and agent of the workspace can call it: "+
				"name one (api.Workspace(authz.…)) or, when any member may call it, add it to openToMembers with why", op.Method, op.Path, op.OperationID)
		case acc.Permission != "" && open:
			t.Errorf("%s needs %s: take it off openToMembers", op.OperationID, acc.Permission)
		case acc.Permission != "" && !authz.Valid(acc.Permission):
			t.Errorf("%s needs %q, which isn't a permission", op.OperationID, acc.Permission)
		}
	}
	for id := range openToMembers {
		if !workspaceOps[id] {
			t.Errorf("openToMembers lists %s, which isn't a workspace operation any more", id)
		}
	}
}

// Deciding on what waits for approval is a person's judgement: every
// operation that needs campaigns:approve is people-only, as decide-approval
// is, so an API key or agent holding it can't approve a send or release one
// a failed canary holds.
func TestApprovalOperationsArePeopleOnly(t *testing.T) {
	n := 0
	for _, op := range registeredOperations(t) {
		acc, _ := api.AccessOf(op)
		if acc.Permission != authz.CampaignsApprove {
			continue
		}
		n++
		if !acc.HumansOnly {
			t.Errorf("%s %s (%s) needs %s but lets API keys and agents call it: use api.WorkspaceHuman", op.Method, op.Path, op.OperationID, acc.Permission)
		}
	}
	if n == 0 {
		t.Fatalf("no operation needs %s: decide-approval should", authz.CampaignsApprove)
	}
}
