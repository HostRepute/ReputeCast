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
