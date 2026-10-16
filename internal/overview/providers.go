package overview

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProviderCards are the sending providers as the providers screen sees them,
// named the Overview's way, with any failing health refresh.
func (s *Service) ProviderCards(ctx context.Context, ws *store.Workspace) (OverviewProviders, error) {
	po, err := s.Providers.Overview(ctx, ws)
	if err != nil {
		return OverviewProviders{}, err
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return OverviewProviders{}, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	out := OverviewProviders{Connected: po.Summary.Connected, MonthStart: po.Summary.MonthStart, Providers: []OverviewProviderCard{}}
	for _, c := range po.Providers {
		// A paused provider sends nothing; the providers screen still lists it.
		if c.Status == state.Paused {
			continue
		}
		card := OverviewProviderCard{Card: c, DisplayName: displayName(c), Detail: detail(c)}
		if id, err := uuid.Parse(c.ID); err == nil {
			if p, ok := byID[id]; ok {
				card.Refresh = refreshFailure(p)
			}
		}
		card.StatusLabel, card.StatusURL = statusPage(c.Adapter)
		if card.Refresh != nil {
			card.Refresh.StatusLabel, card.Refresh.StatusURL = card.StatusLabel, card.StatusURL
		}
		out.Providers = append(out.Providers, card)
	}
	return out, nil
}

// displayName is "Postal (self-hosted)" for servers people run themselves.
func displayName(c manage.ProviderCard) string {
	if d, ok := providers.Lookup(c.Adapter); ok && (d.Kind == "self_hosted") && !strings.Contains(c.Name, "self-hosted") {
		return c.Name + " (self-hosted)"
	}
	return c.Name
}

var streamTitle = map[string]string{"broadcast": "Broadcasts", "automation": "Automations", "system": "System", "transactional": "Transactional"}

// detail is the line under a provider's name: where it runs, its dedicated
// IP and what it carries ("mta.acme.io · 203.0.113.25 dedicated · 10% of
// Broadcasts"). A minority share of a split is written as a share.
func detail(c manage.ProviderCard) string {
	var parts []string
	// A provider's own stream label ("Transactional stream" at Postmark)
	// reads as ReputeCast's stream once it carries Transactional: the
	// streams below say it.
	carries := slices.ContainsFunc(c.Roles, func(r manage.ProviderStreamRole) bool {
		return r.Stream == "transactional" && (r.Role == "primary" || r.Role == "split")
	})
	label := strings.TrimSpace(strings.TrimPrefix(c.Label, "self-hosted · "))
	if carries && strings.HasSuffix(label, " stream") {
		label = ""
	}
	if label != "" && label != "self-hosted" {
		parts = append(parts, label)
	}
	if c.IPMode == "dedicated" && len(c.IPs) > 0 {
		parts = append(parts, strings.Join(c.IPs, ", ")+" dedicated")
	}
	var streams []string
	for _, r := range c.Roles {
		if r.Role != "primary" && r.Role != "split" {
			continue
		}
		name := streamTitle[r.Stream]
		if r.Role == "split" && r.Percent > 0 && r.Percent <= 50 {
			name = fmt.Sprintf("%d%% of %s", r.Percent, name)
		}
		if !slices.Contains(streams, name) {
			streams = append(streams, name)
		}
	}
	if len(streams) > 0 {
		parts = append(parts, strings.Join(streams, ", "))
	}
	return strings.Join(parts, " · ")
}

// refreshFailure is set when the latest health check of a provider that is
// still up failed: its rates and quota are from the last good check.
func refreshFailure(p store.Provider) *OverviewRefreshFailure {
	h := state.HealthOf(p)
	if p.Status == state.Down || h.Down != nil || h.LastError == nil {
		return nil
	}
	if h.LastSuccessAt != nil && !h.LastError.At.After(*h.LastSuccessAt) {
		return nil
	}
	f := &OverviewRefreshFailure{Op: h.LastError.Op, Status: h.LastError.Status, Code: h.LastError.Code, Message: h.LastError.Message, LastSuccessAt: h.LastSuccessAt}
	for _, e := range h.RecentErrors {
		if e.Op == h.LastError.Op && (h.LastSuccessAt == nil || e.At.After(*h.LastSuccessAt)) {
			f.Attempts = append(f.Attempts, e.At)
		}
	}
	if len(f.Attempts) == 0 {
		f.Attempts = append(f.Attempts, h.LastError.At)
	}
	slices.SortFunc(f.Attempts, func(a, b time.Time) int { return a.Compare(b) })
	return f
}

// statusPage is a provider's public status page, when it has one.
func statusPage(adapter string) (string, string) {
	switch adapter {
	case "ses":
		return "AWS status page", "https://health.aws.amazon.com/health/status"
	case "postmark":
		return "Postmark status page", "https://status.postmarkapp.com"
	case "mailgun":
		return "Mailgun status page", "https://status.mailgun.com"
	case "sendgrid":
		return "SendGrid status page", "https://status.sendgrid.com"
	case "resend":
		return "Resend status page", "https://resend-status.com"
	}
	return "", ""
}
