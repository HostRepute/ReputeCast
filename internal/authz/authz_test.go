package authz

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestBuiltinMatrix(t *testing.T) {
	// The comparison table in the Roles screen, row by row.
	cases := []struct {
		perm                         Permission
		owner, admin, editor, viewer bool
	}{
		{CampaignsView, true, true, true, true},
		{CampaignsEdit, true, true, true, false},
		{CampaignsSend, true, true, true, false},
		{CampaignsPause, true, true, true, false},
		{CampaignsApprove, true, true, false, false},
		{AutomationsPublish, true, true, true, false},
		{SubscribersExport, true, true, false, false},
		{SubscribersImport, true, true, true, false},
		{ListsManage, true, true, true, false},
		{SegmentsSQL, true, true, false, false},
		{ReportsShare, true, true, true, false},
		{IntegrationsHostRepute, true, true, false, false},
		{ProvidersView, true, true, true, true},
		{ProvidersManage, true, true, false, false},
		{TeamInvite, true, true, false, false},
		{AuditView, true, true, true, true},
		{WorkspaceManage, true, true, false, false},
		{WorkspaceTransfer, true, false, false, false},
		{WorkspaceDelete, true, false, false, false},

		// The expansion's groups (design expansion briefs, Permissions).
		{TransactionalView, true, true, true, true},
		{TransactionalEdit, true, true, true, false},
		{TransactionalPublish, true, true, true, false},
		{TransactionalSend, true, true, false, false},
		{TransactionalContent, true, true, false, false},
		{SubscribersVerify, true, true, false, false},
		{IntegrationsView, true, true, true, true},
		{IntegrationsManage, true, true, false, false},
		{AIUse, true, true, true, false},
		{ScoringEdit, true, true, true, false},
		{PlacementRun, true, true, true, false},
		{PlacementManage, true, true, false, false},
		{GrowView, true, true, true, true},
		{GrowEdit, true, true, true, false},
		{GrowReview, true, true, true, false},
		{GrowConnect, true, true, false, false},
		{EarnView, true, true, true, true},
		{EarnEdit, true, true, false, false},
		{EarnManage, true, true, false, false},
		{PaymentsManage, true, true, false, false},
		{SponsorsManage, true, true, true, false},
		{ConversionsCreate, true, true, false, false},
		{WebsiteView, true, true, true, true},
		{WebsiteEdit, true, true, true, false},
		{WebsitePublish, true, true, true, false},
		{WebsiteManage, true, true, true, false},
		{ProductsView, true, true, true, true},
		{ProductsEdit, true, true, true, false},
		{ProductsPublish, true, true, true, false},
		{OrdersManage, true, true, true, false},
		{SMSSend, true, true, true, false},
		{PushSend, true, true, true, false},
		{WhatsAppSend, true, true, true, false},
	}
	roles := map[string]BuiltinRole{}
	for _, r := range BuiltinRoles {
		roles[r.Key] = r
	}
	for _, c := range cases {
		for key, want := range map[string]bool{RoleOwner: c.owner, RoleAdmin: c.admin, RoleEditor: c.editor, RoleViewer: c.viewer} {
			if got := roles[key].Permissions.Has(c.perm); got != want {
				t.Errorf("%s has %s = %v, want %v", key, c.perm, got, want)
			}
		}
	}
}

func TestCatalogGroups(t *testing.T) {
	groups := map[string]bool{}
	for _, g := range Groups {
		groups[g] = true
	}
	seen := map[Permission]bool{}
	for _, d := range Defs {
		if !groups[d.Group] {
			t.Errorf("%s is in group %q, which isn't in Groups", d.Key, d.Group)
		}
		if seen[d.Key] {
			t.Errorf("%s is defined twice", d.Key)
		}
		seen[d.Key] = true
		for _, imp := range d.Implies {
			if !Valid(imp) {
				t.Errorf("%s implies unknown %s", d.Key, imp)
			}
		}
	}
}

func TestHoldableBy(t *testing.T) {
	cases := []struct {
		perm       Permission
		key, agent bool
	}{
		{CampaignsSend, true, true},
		{IntegrationsManage, false, false},
		{AIUse, false, false},
		{SubscribersVerify, true, false},
		{PlacementRun, true, false},
		{PlacementManage, true, false},
		{TransactionalSend, true, true},
		{ConversionsCreate, true, true},
	}
	for _, c := range cases {
		if got := HoldableBy(c.perm, ActorAPIKey); got != c.key {
			t.Errorf("API key holds %s = %v, want %v", c.perm, got, c.key)
		}
		if got := HoldableBy(c.perm, ActorMCPAgent); got != c.agent {
			t.Errorf("MCP agent holds %s = %v, want %v", c.perm, got, c.agent)
		}
		if !HoldableBy(c.perm, ActorUser) {
			t.Errorf("a member must be able to hold %s", c.perm)
		}
	}
	s := NewSet(IntegrationsManage, SubscribersVerify, CampaignsEdit).HoldableBy(ActorMCPAgent)
	if s.Has(IntegrationsManage) || s.Has(SubscribersVerify) || !s.Has(CampaignsEdit) || !s.Has(CampaignsView) {
		t.Errorf("agent set = %v", s.Strings())
	}
}

func TestImpliesView(t *testing.T) {
	s := NewSet(CampaignsEdit, SegmentsSQL)
	for _, p := range []Permission{CampaignsView, SegmentsEdit, SubscribersView} {
		if !s.Has(p) {
			t.Errorf("expected %s to be implied", p)
		}
	}
}

func TestCheckGrantEscalation(t *testing.T) {
	writer := NewSet(CampaignsView, CampaignsEdit, TemplatesView, TemplatesEdit, ReportsView)
	if _, err := CheckGrant(writer, []Permission{CampaignsEdit, ReportsView}); err != nil {
		t.Fatalf("granting held permissions: %v", err)
	}
	_, err := CheckGrant(writer, []Permission{CampaignsSend})
	var esc *EscalationError
	if !errors.As(err, &esc) || len(esc.Missing) != 1 || esc.Missing[0] != CampaignsSend {
		t.Fatalf("expected escalation error for campaigns:send, got %v", err)
	}
	if !errors.Is(err, ErrDenied) {
		t.Fatal("escalation must unwrap to ErrDenied")
	}
	// Owner-only permissions are never grantable, even by the Owner.
	if _, err := CheckGrant(All(), []Permission{WorkspaceDelete}); err == nil {
		t.Fatal("workspace:delete must not be grantable")
	}
	if _, err := CheckGrant(All(), []Permission{"nope:never"}); err == nil {
		t.Fatal("unknown permission must be rejected")
	}
}

func TestDeniedMessage(t *testing.T) {
	got := DeniedMessage(CampaignsSend, []string{"Marcus Chen", "Linh Nguyen"})
	want := "Sending needs the campaigns:send permission. Ask Marcus Chen or Linh Nguyen."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestScopeWithin(t *testing.T) {
	beta := uuid.New()
	weekly := uuid.New()
	parent := Scope{Lists: []uuid.UUID{beta}}
	if !(Scope{Lists: []uuid.UUID{beta}}).Within(parent) {
		t.Fatal("same scope must be within")
	}
	if (Scope{}).Within(parent) {
		t.Fatal("unscoped key must not be within a scoped creator")
	}
	if (Scope{Lists: []uuid.UUID{weekly}}).Within(parent) {
		t.Fatal("other list must not be within")
	}
	if !(Scope{Lists: []uuid.UUID{weekly}}).Within(Scope{}) {
		t.Fatal("anything is within an unscoped parent")
	}
}
