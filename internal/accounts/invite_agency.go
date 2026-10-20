package accounts

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
)

// The client variant's one line about each client role.
var (
	mInviteClientViewer   = msgs.M("invite.clientViewer")
	mInviteClientApprover = msgs.M("invite.clientApprover")
)

// InvitationAgency is the client variant of an invitation (agencies,
// channels and agencies brief §6): the workspace is a client of the
// agency that runs this instance. The page explains the role in one line
// and says who manages the workspace.
type InvitationAgency struct {
	// Name is the agency's ("Northwind Studio").
	Name string `json:"name"`
	// ClientRole: the invitation is for Client approver or Client viewer.
	ClientRole bool `json:"clientRole"`
	// RoleLine is what the role sees, in one line: "You'll see campaigns
	// and reports."
	RoleLine string `json:"roleLine,omitempty"`
}

// invitationAgency is set when the invited workspace is an agency client.
func (s *Service) invitationAgency(ctx context.Context, wsID uuid.UUID, roleKey string) *InvitationAgency {
	client, err := s.Q.AgencyIsClientWorkspace(ctx, wsID)
	if err != nil || !client {
		return nil
	}
	st, err := s.Q.AgencySettings(ctx)
	if err != nil {
		return nil
	}
	name := strings.TrimSpace(st.Name)
	if name == "" {
		name = st.InstanceName
	}
	out := &InvitationAgency{Name: name, ClientRole: authz.IsClientRole(roleKey)}
	l := langOf(ctx)
	switch roleKey {
	case authz.RoleClientViewer:
		out.RoleLine = mInviteClientViewer.Text(l, nil)
	case authz.RoleClientApprover:
		out.RoleLine = mInviteClientApprover.Text(l, nil)
	}
	return out
}
