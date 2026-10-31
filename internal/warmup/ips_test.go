package warmup_test

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// dedicatedSES stands in for Amazon SES's adapter once the account has a
// dedicated IP: its API lists it (providers.IPReporter).
type dedicatedSES struct{}

func (dedicatedSES) Adapter() string { return "ses" }
func (dedicatedSES) Capabilities() providers.Capabilities {
	return providers.Capabilities{ReportsIPs: true}
}
func (dedicatedSES) Send(_ context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return make([]providers.SendResult, len(msgs)), nil
}
func (dedicatedSES) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return []providers.Event{}, nil
}
func (dedicatedSES) SendingIPs(context.Context) (providers.SendingIPs, error) {
	return providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.88"}}, nil
}

// Warm-up offers a provider's sending IPs as the IP check keeps them
// (review GS-10): once Amazon SES's API lists a dedicated IP, it is SES's
// IP among the candidates, and a plan started without naming one warms it.
func TestWarmupUsesReportedIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM providers WHERE adapter = 'ses'
		AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	ses, err := env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	sesID := id.String()
	svc := &manage.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
	if _, err := svc.RefreshIPsWith(ctx, ses, dedicatedSES{}, manage.IPsDaily); err != nil {
		t.Fatal(err)
	}

	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var o warmup.WarmupOverview
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	i := slices.IndexFunc(o.Candidates, func(c warmup.WarmupCandidate) bool { return c.ProviderID == sesID })
	if i < 0 || !slices.Equal(o.Candidates[i].IPs, []string{"198.51.100.88"}) {
		t.Fatalf("candidates %+v", o.Candidates)
	}
	var created warmup.WarmupPlanView
	owner.Do("POST", ws+"/warmup", map[string]any{"providerId": sesID, "days": 14, "engagedFirst": true, "guardBounce": 2, "guardComplaint": 0.1,
		"guardAction": "hold", "spreadMailbox": true}).MustStatus(t, 201).JSON(t, &created)
	if created.IP != "198.51.100.88" {
		t.Fatalf("the plan warms %q", created.IP)
	}
}
