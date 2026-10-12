package routing

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Facts is what validation needs to know about a connected provider.
type Facts struct {
	ID          uuid.UUID
	Name        string
	Short       string
	Adapter     string
	Status      string
	Bulk        providers.BulkPolicy
	BulkAllowed bool
	// BroadcastStreamCreatable is true when ReputeCast can create the
	// stream that makes bulk mail allowed (Postmark).
	BroadcastStreamCreatable bool
	WarmingUp                bool
	WarmupDay                int
	WarmupDays               int
}

// Issue codes.
const (
	IssueWeights          = "weights"
	IssueUnknownProvider  = "unknown_provider"
	IssueDuplicate        = "duplicate_provider"
	IssueFallbackIsRoute  = "fallback_in_split"
	IssueNoRoute          = "no_route"
	IssueBulkTerms        = "bulk_terms"
	IssueWarmingFallback  = "warming_fallback"
	IssueUnavailable      = "unavailable"
	IssueConditionNoValue = "condition_values"
)

// Issue is a problem with a routing set. Blocking issues stop a save;
// the others are warnings shown next to the route.
type Issue struct {
	Code       string `json:"code"`
	Stream     string `json:"stream"`
	Field      string `json:"field,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
	Message    string `json:"message"`
	Blocking   bool   `json:"blocking"`
}

// Validate checks a routing set against the connected providers.
func Validate(set Set, facts map[uuid.UUID]Facts) []Issue {
	issues := []Issue{}
	for _, stream := range Streams {
		st := string(stream)
		rules := set.StreamRules(stream)
		def, hasDefault := set.Default(stream)
		if !hasDefault {
			issues = append(issues, Issue{Code: IssueNoRoute, Stream: st, Field: "splits", Message: streamName(stream) + " has no primary provider.", Blocking: true})
			continue
		}
		total := 0
		seen := map[uuid.UUID]bool{}
		for _, sp := range def.Splits {
			total += sp.Weight
			if seen[sp.ProviderID] {
				issues = append(issues, Issue{Code: IssueDuplicate, Stream: st, Field: "splits", ProviderID: sp.ProviderID.String(), Message: "A provider appears twice in the split.", Blocking: true})
			}
			seen[sp.ProviderID] = true
			if _, ok := facts[sp.ProviderID]; !ok {
				issues = append(issues, Issue{Code: IssueUnknownProvider, Stream: st, Field: "splits", ProviderID: sp.ProviderID.String(), Message: "This provider is not connected to the workspace.", Blocking: true})
			}
			if sp.Weight < 0 || sp.Weight > 100 {
				issues = append(issues, Issue{Code: IssueWeights, Stream: st, Field: "splits", Message: "Shares go from 0 to 100%.", Blocking: true})
			}
		}
		if total != 100 {
			issues = append(issues, Issue{Code: IssueWeights, Stream: st, Field: "splits", Message: fmt.Sprintf("The %s split adds up to %d%%; it must add up to 100%%.", streamName(stream), total), Blocking: true})
		}
		if def.Fallback != nil {
			if _, ok := facts[*def.Fallback]; !ok {
				issues = append(issues, Issue{Code: IssueUnknownProvider, Stream: st, Field: "fallback", ProviderID: def.Fallback.String(), Message: "The fallback is not connected to the workspace.", Blocking: true})
			} else if seen[*def.Fallback] {
				issues = append(issues, Issue{Code: IssueFallbackIsRoute, Stream: st, Field: "fallback", ProviderID: def.Fallback.String(), Message: "The fallback is already part of the split; pick another provider or none.", Blocking: true})
			}
		}
		for i, r := range rules {
			if r.Conditions.Empty() {
				continue
			}
			field := fmt.Sprintf("conditions[%d]", i)
			if len(r.Splits) != 1 {
				issues = append(issues, Issue{Code: IssueWeights, Stream: st, Field: field, Message: "A condition sends to exactly one provider.", Blocking: true})
				continue
			}
			if _, ok := facts[r.Splits[0].ProviderID]; !ok {
				issues = append(issues, Issue{Code: IssueUnknownProvider, Stream: st, Field: field, ProviderID: r.Splits[0].ProviderID.String(), Message: "This provider is not connected to the workspace.", Blocking: true})
			}
		}
		// Terms: every provider that can carry Broadcasts must allow bulk mail.
		if stream == providers.StreamBroadcast {
			check := func(id uuid.UUID, field string) {
				f, ok := facts[id]
				if !ok || f.BulkAllowed {
					return
				}
				msg := f.Name + "'s terms don't allow bulk marketing."
				if f.Bulk == providers.BulkBroadcastStreams {
					msg = f.Name + " allows bulk mail only on a Broadcast stream, and this route uses its transactional stream."
				}
				issues = append(issues, Issue{Code: IssueBulkTerms, Stream: st, Field: field, ProviderID: id.String(), Message: msg})
			}
			for _, r := range rules {
				for _, sp := range r.Splits {
					if sp.Weight > 0 {
						check(sp.ProviderID, "splits")
					}
				}
			}
			if def.Fallback != nil {
				check(*def.Fallback, "fallback")
			}
		}
		// A warming-up fallback can't take a broadcast until it's warm.
		if def.Fallback != nil {
			if f, ok := facts[*def.Fallback]; ok && f.WarmingUp && stream == providers.StreamBroadcast && set.Policy(stream) != PauseAndAsk {
				msg := f.Name + " is warming up"
				if f.WarmupDays > 0 {
					msg += fmt.Sprintf(": failover to it is blocked by the warm-up guard until day %d", f.WarmupDays)
				}
				issues = append(issues, Issue{Code: IssueWarmingFallback, Stream: st, Field: "fallback", ProviderID: f.ID.String(), Message: msg + "."})
			}
		}
	}
	return issues
}

// Blocking reports whether any issue stops a save.
func Blocking(issues []Issue) bool {
	for _, i := range issues {
		if i.Blocking {
			return true
		}
	}
	return false
}

func streamName(s providers.Stream) string {
	switch s {
	case providers.StreamBroadcast:
		return "Broadcasts"
	case providers.StreamAutomation:
		return "Automations"
	}
	return "System"
}
