package instance

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/auditlog"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// InstanceSignInEvent is one row of Users › Recent sign-in events.
type InstanceSignInEvent struct {
	ID string    `json:"id"`
	At time.Time `json:"at"`
	// Event names the method: "Password + TOTP", "Google Workspace SSO",
	// "Password · 3 failed".
	Event   string `json:"event"`
	Account string `json:"account"`
	// Result: allowed, denied, or recovered (failed attempts, then allowed).
	Result string `json:"result" enum:"allowed,denied,recovered"`
	// Detail explains a rejection ("domain not allowed").
	Detail string `json:"detail,omitempty"`
}

// InstanceSignInList is GET /admin/sign-ins.
type InstanceSignInList struct {
	Items []InstanceSignInEvent `json:"items"`
}

type signInMeta struct {
	Method   string `json:"method"`
	Provider string `json:"provider"`
	Code     string `json:"code"`
	Failures int    `json:"failures"`
	// FailedBefore is set by the demo for "Password · 3 failed".
	FailedBefore int `json:"failed_before"`
}

// SignIns returns the latest sign-in attempts, in the reader's language.
// Failed password attempts followed by a successful sign-in of the same
// account within an hour are folded into that row ("Password · 3 failed ·
// Then allowed").
func (s *Service) SignIns(ctx context.Context, limit int) (InstanceSignInList, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	rows, err := s.Q.InstanceAdminSignInEvents(ctx, int32(limit*6))
	if err != nil {
		return InstanceSignInList{}, err
	}
	l := i18n.LangFrom(ctx)
	out := InstanceSignInList{Items: []InstanceSignInEvent{}}
	type folded struct {
		idx int
		at  time.Time
	}
	// method and failed name each row's method and the failed attempts
	// folded into it, written into its Event at the end.
	var method []string
	var failed []int
	lastAllowed := map[string]folded{}
	for _, r := range rows {
		var m signInMeta
		_ = json.Unmarshal(r.Metadata, &m)
		account := strings.ToLower(r.TargetName)
		allowed := r.Result == "allowed"
		if strings.HasPrefix(r.Action, "signin.") {
			allowed = r.Action == "signin.succeeded"
		}
		if !allowed && strings.HasPrefix(m.Method, "password") && m.Code == "" {
			if f, ok := lastAllowed[account]; ok && f.at.Sub(r.OccurredAt) < time.Hour {
				out.Items[f.idx].Result = "recovered"
				failed[f.idx]++
				continue
			}
		}
		if len(out.Items) >= limit {
			continue
		}
		ev := InstanceSignInEvent{ID: r.ID.String(), At: r.OccurredAt, Event: auditlog.SignInMethod(l, m.Method, m.Provider), Account: account, Result: "denied"}
		n := 0
		if allowed {
			ev.Result = "allowed"
			if m.FailedBefore > 0 {
				ev.Result = "recovered"
				ev.Event, n = auditlog.SignInMethod(l, "password", ""), m.FailedBefore
			}
			lastAllowed[account] = folded{idx: len(out.Items), at: r.OccurredAt}
		} else {
			ev.Detail = auditlog.SignInRejection(l, m.Code, r.Reason)
		}
		out.Items = append(out.Items, ev)
		method, failed = append(method, ev.Event), append(failed, n)
	}
	for i := range out.Items {
		if failed[i] > 0 {
			out.Items[i].Event = auditlog.SignInFailed(l, method[i], failed[i])
		}
	}
	return out, nil
}
