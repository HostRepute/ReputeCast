package verification

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// ContactAddressCheck is the contact profile's address row: "Never checked
// · Verify now", or "Valid · ZeroBounce · 14 Oct".
type ContactAddressCheck struct {
	Checked     bool       `json:"checked"`
	Result      string     `json:"result,omitempty" enum:"valid,invalid,risky,unknown,"`
	Reason      string     `json:"reason,omitempty"`
	Code        string     `json:"code,omitempty"`
	Service     string     `json:"service,omitempty"`
	ServiceName string     `json:"serviceName,omitempty"`
	CheckedAt   *time.Time `json:"checkedAt,omitempty"`
	Tag         string     `json:"tag,omitempty"`
	// Suppressed: an invalid result put the address on the suppression list.
	Suppressed   bool       `json:"suppressed"`
	SuppressedAt *time.Time `json:"suppressedAt,omitempty"`
	// RecheckFrom: an unknown result can be checked again from then.
	RecheckFrom *time.Time `json:"recheckFrom,omitempty"`
	// What "Verify now" would use: the bulk default, its price and region.
	Email             string   `json:"email"`
	VerifyService     string   `json:"verifyService,omitempty"`
	VerifyServiceName string   `json:"verifyServiceName,omitempty"`
	RegionLabel       string   `json:"regionLabel,omitempty"`
	Price             float64  `json:"price"`
	Balance           *int64   `json:"balance,omitempty"`
	Connected         bool     `json:"connected"`
	CanVerify         bool     `json:"canVerify"`
	Helpers           []string `json:"helpers"`
}

// ContactCheck returns a contact's latest result and what "Verify now" uses.
func (s *Service) ContactCheck(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (ContactAddressCheck, error) {
	p := api.PrincipalFrom(ctx)
	out := ContactAddressCheck{CanVerify: p.Can(authz.SubscribersVerify), Helpers: Helpers(ctx, s.Q, ws.ID, authz.SubscribersVerify)}
	var email string
	if err := s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE workspace_id = $1 AND id = $2 AND erased_at IS NULL`, ws.ID, contactID).Scan(&email); err != nil {
		return out, api.MapError(err, "Contact")
	}
	out.Email = email
	c, err := s.Q.VerifContactCheck(ctx, store.VerifContactCheckParams{ContactID: contactID, WorkspaceID: ws.ID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return out, err
	default:
		out.Checked = true
		out.Result, out.Reason, out.Code, out.Service = c.Result, c.Reason, c.Code, c.Service
		out.ServiceName = ServiceName(c.Service)
		at := c.CheckedAt
		out.CheckedAt = &at
		out.Tag = adapter.Answer{Result: adapter.Result(c.Result), Reason: c.Reason}.Tag()
		if c.Result == string(adapter.Unknown) {
			r := c.CheckedAt.Add(RecheckAfter)
			out.RecheckFrom = &r
		}
		if c.Result == string(adapter.Invalid) {
			var sup time.Time
			err := s.Pool.QueryRow(ctx, `SELECT created_at FROM suppressions WHERE workspace_id = $1 AND email = $2`, ws.ID, email).Scan(&sup)
			if err == nil {
				out.Suppressed, out.SuppressedAt = true, &sup
			}
		}
	}
	chosen, all, err := s.pick(ctx, ws, "")
	if err != nil {
		return out, err
	}
	out.Connected = len(all) > 0
	if chosen != nil {
		out.VerifyService, out.VerifyServiceName = chosen.Service, ServiceName(chosen.Service)
		out.RegionLabel = regionIn(i18n.LangFrom(ctx), chosen.Service, chosen.Region)
		out.Price = Money(chosen.PriceMicros).Dollars()
		out.Balance = chosen.Balance
	}
	return out, nil
}

// VerifyContactInput picks the service for "Verify now".
type VerifyContactInput struct {
	Service string `json:"service,omitempty" enum:"zerobounce,kickbox,neverbounce,bouncer,emailable,"`
}

// VerifyContact checks one contact now: a one-address job, audited like
// any use, run at once.
func (s *Service) VerifyContact(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, in VerifyContactInput) (ContactAddressCheck, error) {
	no := false
	j, err := s.startJob(ctx, ws, VerificationStartInput{Target: VerificationTarget{Kind: "contact", ContactID: contactID.String()},
		Service: in.Service, SkipRecent: &no, Origin: "profile"}, true)
	if err != nil {
		return ContactAddressCheck{}, err
	}
	id, _ := uuid.Parse(j.ID)
	if _, err := s.Run(context.WithoutCancel(ctx), id, time.Minute); err != nil {
		return ContactAddressCheck{}, err
	}
	return s.ContactCheck(ctx, ws, contactID)
}
