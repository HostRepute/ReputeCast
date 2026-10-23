package routing

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
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

// RoutingIssue codes.
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

// RoutingIssue is a problem with a routing set. Blocking issues stop a save;
// the others are warnings shown next to the route.
type RoutingIssue struct {
	Code       string `json:"code"`
	Stream     string `json:"stream"`
	Field      string `json:"field,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
	Message    string `json:"message"`
	Blocking   bool   `json:"blocking"`
}

// Validate checks a routing set against the connected providers; the
// issues' messages are in the reader's language.
func Validate(l i18n.Lang, set Set, facts map[uuid.UUID]Facts) []RoutingIssue {
	issues := []RoutingIssue{}
	for _, stream := range Streams {
		st := string(stream)
		rules := set.StreamRules(stream)
		def, hasDefault := set.Default(stream)
		if !hasDefault && stream == StreamTransactional && len(rules) == 0 {
			// Transactional is optional: a workspace that sends no product
			// mail has no route for it.
			continue
		}
		if !hasDefault {
			issues = append(issues, RoutingIssue{Code: IssueNoRoute, Stream: st, Field: "splits", Message: text(l, "issue.noRoute", i18n.Args{"stream": streamArg(stream)}), Blocking: true})
			continue
		}
		total := 0
		seen := map[uuid.UUID]bool{}
		for _, sp := range def.Splits {
			total += sp.Weight
			if seen[sp.ProviderID] {
				issues = append(issues, RoutingIssue{Code: IssueDuplicate, Stream: st, Field: "splits", ProviderID: sp.ProviderID.String(), Message: text(l, "issue.duplicate", nil), Blocking: true})
			}
			seen[sp.ProviderID] = true
			if _, ok := facts[sp.ProviderID]; !ok {
				issues = append(issues, RoutingIssue{Code: IssueUnknownProvider, Stream: st, Field: "splits", ProviderID: sp.ProviderID.String(), Message: text(l, "issue.unknownProvider", nil), Blocking: true})
			}
			if sp.Weight < 0 || sp.Weight > 100 {
				issues = append(issues, RoutingIssue{Code: IssueWeights, Stream: st, Field: "splits", Message: text(l, "issue.weightRange", i18n.Args{"min": 0, "max": 1}), Blocking: true})
			}
		}
		if total != 100 {
			issues = append(issues, RoutingIssue{Code: IssueWeights, Stream: st, Field: "splits",
				Message: text(l, "issue.weightSum", i18n.Args{"stream": streamArg(stream), "total": float64(total) / 100, "full": 1}), Blocking: true})
		}
		if def.Fallback != nil {
			if _, ok := facts[*def.Fallback]; !ok {
				issues = append(issues, RoutingIssue{Code: IssueUnknownProvider, Stream: st, Field: "fallback", ProviderID: def.Fallback.String(), Message: text(l, "issue.fallbackUnknown", nil), Blocking: true})
			} else if seen[*def.Fallback] {
				issues = append(issues, RoutingIssue{Code: IssueFallbackIsRoute, Stream: st, Field: "fallback", ProviderID: def.Fallback.String(), Message: text(l, "issue.fallbackInSplit", nil), Blocking: true})
			}
		}
		for i, r := range rules {
			if r.Conditions.Empty() {
				continue
			}
			field := fmt.Sprintf("conditions[%d]", i)
			if len(r.Splits) != 1 {
				issues = append(issues, RoutingIssue{Code: IssueWeights, Stream: st, Field: field, Message: text(l, "issue.conditionOne", nil), Blocking: true})
				continue
			}
			if _, ok := facts[r.Splits[0].ProviderID]; !ok {
				issues = append(issues, RoutingIssue{Code: IssueUnknownProvider, Stream: st, Field: field, ProviderID: r.Splits[0].ProviderID.String(), Message: text(l, "issue.unknownProvider", nil), Blocking: true})
			}
		}
		// Terms: every provider that can carry Broadcasts must allow bulk mail.
		if stream == providers.StreamBroadcast {
			check := func(id uuid.UUID, field string) {
				f, ok := facts[id]
				if !ok || f.BulkAllowed {
					return
				}
				msg := text(l, "issue.bulkTerms", i18n.Args{"provider": f.Name})
				if f.Bulk == providers.BulkBroadcastStreams {
					msg = text(l, "issue.bulkStreams", i18n.Args{"provider": f.Name})
				}
				issues = append(issues, RoutingIssue{Code: IssueBulkTerms, Stream: st, Field: field, ProviderID: id.String(), Message: msg})
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
				args := i18n.Args{"provider": f.Name, "hasDay": "no", "day": f.WarmupDays}
				if f.WarmupDays > 0 {
					args["hasDay"] = "yes"
				}
				issues = append(issues, RoutingIssue{Code: IssueWarmingFallback, Stream: st, Field: "fallback", ProviderID: f.ID.String(), Message: text(l, "issue.warmingFallback", args)})
			}
		}
	}
	return issues
}

// Blocking reports whether any issue stops a save.
func Blocking(issues []RoutingIssue) bool {
	for _, i := range issues {
		if i.Blocking {
			return true
		}
	}
	return false
}

