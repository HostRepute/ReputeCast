package roles_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The Roles and RoleEditor boards of the design expansion: every new group
// in the catalogue's order with its people-only and no-agents marks, and the
// sample's Content Writer with the permissions the phase briefs give it.
func TestRolesExpansionCatalogue(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l struct {
		Groups []string
		Roles  []struct {
			Key         string
			Permissions []string
		}
		Permissions []struct {
			Key, Group                      string
			Sensitive, PeopleOnly, NoAgents bool
		}
	}
	c.Do("GET", ws+"/roles", nil).MustStatus(t, 200).JSON(t, &l)

	want := []string{"Campaigns", "Templates", "Transactional", "Automations", "Contacts", "Lists and segments", "Forms and pages", "Reports",
		"Deliverability", "Sending", "Team", "Developer", "Integrations", "AI", "Audience scoring", "Placement tests", "Grow", "Earn",
		"Website and pages", "Products and orders", "SMS", "Push", "WhatsApp", "Audit", "Workspace"}
	if !slices.Equal(l.Groups, want) {
		t.Fatalf("groups: %v", l.Groups)
	}
	// Permissions come group by group, in the groups' order.
	last := -1
	marks := map[string]string{}
	for _, p := range l.Permissions {
		g := slices.Index(l.Groups, p.Group)
		if g < last {
			t.Fatalf("%s (%s) is out of the groups' order", p.Key, p.Group)
		}
		last = g
		switch {
		case p.PeopleOnly:
			marks[p.Key] = "people"
		case p.NoAgents:
			marks[p.Key] = "no-agents"
		}
	}
	if len(marks) != 5 || marks["integrations:manage"] != "people" || marks["ai:use"] != "people" ||
		marks["subscribers:verify"] != "no-agents" || marks["placement:run"] != "no-agents" || marks["placement:manage"] != "no-agents" {
		t.Fatalf("people-only and no-agents marks: %v", marks)
	}

	var cw []string
	for _, r := range l.Roles {
		if r.Key == "content-writer" {
			cw = r.Permissions
		}
	}
	for _, k := range []string{"transactional:view", "transactional:edit", "integrations:view", "ai:use", "grow:view", "website:view", "website:edit", "products:view", "products:edit",
		"campaigns:edit", "templates:edit", "automations:view", "subscribers:view", "reports:view", "deliverability:view", "providers:view", "team:view"} {
		if !slices.Contains(cw, k) {
			t.Fatalf("Content Writer lacks %s: %v", k, cw)
		}
	}
	for _, k := range []string{"earn:view", "audit:view", "transactional:publish", "campaigns:send", "integrations:manage", "grow:edit", "scoring:edit"} {
		if slices.Contains(cw, k) {
			t.Fatalf("Content Writer holds %s: %v", k, cw)
		}
	}

	// Giving the role campaigns:send keeps Daniel's pending request pending,
	// and says which rules still require approval (RoleEditorStates).
	var cwID string
	var all struct {
		Roles []struct{ ID, Key string }
	}
	c.Do("GET", ws+"/roles", nil).MustStatus(t, 200).JSON(t, &all)
	for _, r := range all.Roles {
		if r.Key == "content-writer" {
			cwID = r.ID
		}
	}
	type impact struct {
		PendingApprovals int
		PendingRequests  []struct {
			Requester, Subject string
			StillMatches       []string
		}
	}
	var withSend, same impact
	c.Do("POST", ws+"/roles/impact", map[string]any{"roleId": cwID, "permissions": append(slices.Clone(cw), "campaigns:send")}).MustStatus(t, 200).JSON(t, &withSend)
	if withSend.PendingApprovals != 1 || len(withSend.PendingRequests) != 1 {
		t.Fatalf("impact with campaigns:send: %+v", withSend)
	}
	pr := withSend.PendingRequests[0]
	if pr.Requester != "Daniel Park" || pr.Subject != "November Product Update" || slices.Contains(pr.StillMatches, "author lacks send") || !slices.Contains(pr.StillMatches, "audience ≥ 10,000") {
		t.Fatalf("pending request: %+v", pr)
	}
	c.Do("POST", ws+"/roles/impact", map[string]any{"roleId": cwID, "permissions": cw}).MustStatus(t, 200).JSON(t, &same)
	if len(same.PendingRequests) != 1 || !slices.Contains(same.PendingRequests[0].StillMatches, "author lacks send") {
		t.Fatalf("impact unchanged: %+v", same)
	}

	// Daniel Park, the Content Writer, gets them through his role.
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	var me struct{ Permissions []string }
	d.Do("GET", ws+"/me", nil).MustStatus(t, 200).JSON(t, &me)
	if !slices.Contains(me.Permissions, "ai:use") || !slices.Contains(me.Permissions, "products:edit") || slices.Contains(me.Permissions, "earn:view") {
		t.Fatalf("daniel: %s", strings.Join(me.Permissions, ","))
	}
}
