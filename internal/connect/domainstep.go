package connect

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// ConnectDomainInput picks the sending domain of step 3.
type ConnectDomainInput struct {
	Domain string `json:"domain,omitempty" maxLength:"253"`
	// Skip moves on without a verified domain (the test send still works
	// from an address the provider already verified).
	Skip bool `json:"skip,omitempty"`
}

// ChooseDomain sets up the provider for a sending domain: the domain is
// added when it's new, the provider's records are fetched (through its
// domain API when it has one) and checked live.
func (s *Service) ChooseDomain(ctx context.Context, ws *store.Workspace, rawID string, in ConnectDomainInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	if in.Skip {
		st, err := s.update(ctx, ws, cur.ID, "provider.setup_domain_skip", map[string]any{"provider": p.Name}, func(st *store.ProviderSetup, state *State) error {
			name := ""
			if state.Domain != nil {
				name = state.Domain.Name
			}
			state.Domain = &DomainStep{Name: name, Skipped: true}
			if st.Status == "open" {
				st.Step = "test"
			}
			return nil
		})
		if err != nil {
			return ConnectSetup{}, err
		}
		return s.View(ctx, ws, st)
	}
	if strings.TrimSpace(in.Domain) == "" {
		return ConnectSetup{}, api.Invalid("body.domain", mProblemDomain.Text(i18n.LangFrom(ctx), nil))
	}
	d, err := s.Domains.EnsureDomain(ctx, ws, in.Domain)
	if err != nil {
		return ConnectSetup{}, err
	}
	if _, err := s.Domains.AttachProvider(ctx, ws, d.ID, p.ID); err != nil {
		var pr *api.Problem
		if errors.As(err, &pr) {
			return ConnectSetup{}, pr
		}
		return ConnectSetup{}, err
	}
	verified, total := s.domainCounts(ctx, d.ID, p.ID)
	now := s.now()
	st, err := s.update(ctx, ws, cur.ID, "provider.setup_domain", map[string]any{"provider": p.Name, "domain": d.Domain}, func(_ *store.ProviderSetup, state *State) error {
		state.Domain = &DomainStep{DomainID: d.ID.String(), Name: d.Domain, Verified: verified, Total: total, At: &now}
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, st)
}

// domainCounts counts the records a provider needs for a domain (its own
// and the domain-wide ones) and how many are verified.
func (s *Service) domainCounts(ctx context.Context, domainID, providerID uuid.UUID) (int, int) {
	d, err := s.Q.DomGetAny(ctx, domainID)
	if err != nil {
		return 0, 0
	}
	verified, total := 0, 0
	for _, r := range domains.DecodeRecords(d.DnsRecords) {
		if r.ProviderID != "" && r.ProviderID != providerID.String() {
			continue
		}
		total++
		if r.Status == domains.StatusVerified {
			verified++
		}
	}
	return verified, total
}
