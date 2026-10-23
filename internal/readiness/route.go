package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// Audience counts the final recipients of a stored audience definition.
func (s *Service) Audience(ctx context.Context, ws *store.Workspace, raw json.RawMessage) (AudienceFacts, error) {
	f, _, err := s.computeAudience(ctx, s.Q, ws.ID, ParseAudience(raw), sunsetDays(ws.SunsetPolicy), s.now())
	return f, err
}

// RouteConfig describes the routing a campaign uses: its override, or the
// Broadcasts default split. Key changes when the split changes; Label is
// "SES 90 / Postal 10" and StreamLabel "Broadcasts 90 / 10".
type RouteConfig struct {
	Key         string
	Label       string
	StreamLabel string
}

// Route returns the routing configuration of a campaign.
func (s *Service) Route(ctx context.Context, ws *store.Workspace, override json.RawMessage) (RouteConfig, error) {
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return RouteConfig{}, err
	}
	splits := []routing.Split{}
	if o := ParseRouteOverride(override); o != nil {
		splits = o.Splits
	} else if def, ok := set.Default(providers.StreamBroadcast); ok {
		splits = def.Splits
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return RouteConfig{}, err
	}
	names := map[uuid.UUID]string{}
	for _, p := range provs {
		names[p.ID] = state.Short(p.Name, p.Adapter)
	}
	total := 0
	for _, sp := range splits {
		total += max(sp.Weight, 0)
	}
	keys, labels, pcts := []string{}, []string{}, []string{}
	for _, sp := range splits {
		if sp.Weight <= 0 {
			continue
		}
		pct := sp.Weight
		if total > 0 && total != 100 {
			pct = sp.Weight * 100 / total
		}
		keys = append(keys, fmt.Sprintf("%s:%d", sp.ProviderID, pct))
		labels = append(labels, fmt.Sprintf("%s %d", firstNonEmpty(names[sp.ProviderID], "provider"), pct))
		pcts = append(pcts, fmt.Sprint(pct))
	}
	slices.Sort(keys)
	return RouteConfig{Key: strings.Join(keys, ","), Label: strings.Join(labels, " / "), StreamLabel: broadcastSplit(strings.Join(pcts, " / ")).en()}, nil
}

// broadcastSplit is the Broadcasts route with its split: "Broadcasts 90 /
// 10", or "Broadcasts" without one.
func broadcastSplit(split string) Text {
	if split == "" {
		return tx("stream.broadcasts", nil)
	}
	return tx("estimate.split", A{"stream": tx("stream.broadcasts", nil), "split": split})
}
