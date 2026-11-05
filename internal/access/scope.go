package access

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member, API key or agent limited to some lists sees only the contacts
// on one of them, in any status, as Audience's list and search do
// (internal/audience/filter.go). Every other read that names contacts
// applies the same rule through these.

// ScopeLists is the caller's list scope as a query argument: their lists,
// or an empty array when they see every contact. Never nil: Postgres reads
// a nil slice as NULL, and "cardinality(@scope_lists) = 0" would then
// match no one.
func ScopeLists(p *authz.Principal) []uuid.UUID {
	if p == nil || len(p.Scope.Lists) == 0 {
		return []uuid.UUID{}
	}
	return p.Scope.Lists
}

// ContactInScope reports whether the caller may see a contact: they aren't
// limited to some lists, or one of their lists has the contact.
func ContactInScope(ctx context.Context, q *store.Queries, p *authz.Principal, contactID uuid.UUID) (bool, error) {
	if p == nil || len(p.Scope.Lists) == 0 {
		return true, nil
	}
	return q.AccessScopeContact(ctx, store.AccessScopeContactParams{ContactID: contactID, ScopeLists: p.Scope.Lists})
}

// Whoever starts, schedules or resumes a send does it within their own
// scope, whoever built the campaign: its audience on their lists, and its
// From address on one of their sender domains. The composer checks the
// sender a member picks; these check the campaign as it is when it goes
// out, so a sender picked by someone else, or kept by a copy, can't take a
// limited member's send outside their domains.

var mSenderScope = msgs.M("problem.senderScope")

// SenderDomain is the domain a campaign sends from: its sender's sending
// domain, or the domain of the sender's address when that sending domain
// is gone. Empty for a campaign without a sender: there is no other From
// to fall back to when it sends, so Send Readiness blocks it and the
// engine never starts it.
func SenderDomain(domain, email string) string {
	if domain != "" {
		return domain
	}
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return email[i+1:]
	}
	return ""
}

// SendScopeProblem says, in a language, why a member, key or agent can't
// start, schedule or resume a campaign's send, or "" when they can: the
// audience must be their lists only (no segment), and the domain it sends
// from one of their sender domains.
func SendScopeProblem(l i18n.Lang, scope authz.Scope, lists []uuid.UUID, segments int, senderDomain, senderEmail string) string {
	if !AudienceInScope(scope, lists, segments) {
		return AudienceScopeProblem(l)
	}
	if d := SenderDomain(senderDomain, senderEmail); d != "" && !scope.AllowsSenderDomain(d) {
		return mSenderScope.Text(l, i18n.Args{"domain": d})
	}
	return ""
}

// CampaignSendProblem is SendScopeProblem for a campaign by id, for what
// acts on a send without loading its campaign: the live monitor and the
// MCP tools.
func CampaignSendProblem(ctx context.Context, q *store.Queries, l i18n.Lang, p *authz.Principal, wsID, campaignID uuid.UUID) (string, error) {
	if p == nil || p.Scope.IsZero() {
		return "", nil
	}
	c, err := q.AccessScopeCampaign(ctx, store.AccessScopeCampaignParams{WorkspaceID: wsID, ID: campaignID})
	if err != nil {
		return "", err
	}
	var a struct {
		Lists    []string `json:"lists"`
		Segments []string `json:"segments"`
	}
	_ = json.Unmarshal(c.Audience, &a)
	return SendScopeProblem(l, p.Scope, ParseLists(a.Lists), len(a.Segments), c.SenderDomain, c.SenderEmail), nil
}
