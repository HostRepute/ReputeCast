package instance

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
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

// SignIns returns the latest sign-in attempts. Failed password attempts
// followed by a successful sign-in of the same account within an hour are
// folded into that row ("Password · 3 failed · Then allowed").
func (s *Service) SignIns(ctx context.Context, limit int) (InstanceSignInList, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	rows, err := s.Q.InstanceAdminSignInEvents(ctx, int32(limit*6))
	if err != nil {
		return InstanceSignInList{}, err
	}
	out := InstanceSignInList{Items: []InstanceSignInEvent{}}
	type folded struct {
		idx int
		at  time.Time
	}
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
				it := &out.Items[f.idx]
				it.Result = "recovered"
				it.Event = foldFailures(it.Event)
				continue
			}
		}
		if len(out.Items) >= limit {
			continue
		}
		ev := InstanceSignInEvent{ID: r.ID.String(), At: r.OccurredAt, Event: methodLabel(m.Method, m.Provider), Account: account, Result: "denied"}
		if allowed {
			ev.Result = "allowed"
			if m.FailedBefore > 0 {
				ev.Result = "recovered"
				ev.Event = "Password · " + strconv.Itoa(m.FailedBefore) + " failed"
			}
			lastAllowed[account] = folded{idx: len(out.Items), at: r.OccurredAt}
		} else {
			ev.Detail = rejectionDetail(m.Code, r.Reason)
		}
		out.Items = append(out.Items, ev)
	}
	return out, nil
}

// foldFailures turns "Password" into "Password · 1 failed" and counts up.
func foldFailures(event string) string {
	base, rest, ok := strings.Cut(event, " · ")
	if !ok {
		return event + " · 1 failed"
	}
	n, err := strconv.Atoi(strings.TrimSuffix(rest, " failed"))
	if err != nil {
		return event
	}
	return base + " · " + strconv.Itoa(n+1) + " failed"
}

// methodLabel names a sign-in method like the design: "Password + TOTP",
// "Google Workspace SSO", "Passkey".
func methodLabel(method, provider string) string {
	parts := strings.Split(method, "+")
	names := make([]string, 0, len(parts))
	for i, p := range parts {
		switch p {
		case "password":
			names = append(names, "Password")
		case "totp":
			names = append(names, "TOTP")
		case "passkey":
			if i == 0 {
				names = append(names, "Passkey")
			} else {
				names = append(names, "passkey")
			}
		case "recovery":
			names = append(names, "recovery code")
		case "sso":
			if provider == "" {
				provider = "Single sign-on"
			}
			names = append(names, provider+" SSO")
		case "google":
			names = append(names, "Google")
		case "microsoft":
			names = append(names, "Microsoft")
		case "github":
			names = append(names, "GitHub")
		case "":
		default:
			names = append(names, p)
		}
	}
	if len(names) == 0 {
		return "Sign-in"
	}
	return strings.Join(names, " + ")
}

func rejectionDetail(code, reason string) string {
	switch code {
	case "domain_not_allowed":
		return "domain not allowed"
	case "no_account":
		return "no account"
	case "account_deactivated":
		return "deactivated"
	case "not_in_org":
		return "not in the organization"
	case "email_not_verified":
		return "email not verified"
	case "":
	default:
		return strings.ReplaceAll(code, "_", " ")
	}
	switch {
	case strings.Contains(reason, "wrong password"):
		return "wrong password"
	case strings.Contains(reason, "domain not allowed"):
		return "domain not allowed"
	case strings.Contains(reason, "passkey"):
		return "passkey failed"
	}
	if len(reason) > 40 {
		return reason[:40] + "…"
	}
	return reason
}
