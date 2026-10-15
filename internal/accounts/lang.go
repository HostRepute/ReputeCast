package accounts

import (
	"context"
	"embed"
	"encoding/json"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// The texts the server writes for invitees and members (what a role can
// do, why two-factor is required) follow the reader's app language
// (i18n.LangFrom): the page's language for an invitation, the member's
// own for Account & security. They are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "accounts")

var (
	mInviterUnknown      = msgs.M("inviter.unknown")
	mInviterUnknownLower = msgs.M("inviter.unknownLower")

	mScopeLists   = msgs.M("scope.lists")
	mScopeDomains = msgs.M("scope.domains")
	mScopeBoth    = msgs.M("scope.both")

	mRoleOwner     = msgs.M("role.owner")
	mRoleAdmin     = msgs.M("role.admin")
	mRoleViewer    = msgs.M("role.viewer")
	mRoleSend      = msgs.M("role.send")
	mRoleSendLists = msgs.M("role.sendLists")
	mRoleEdit      = msgs.M("role.edit")
	mRoleEditLists = msgs.M("role.editLists")
	mRoleApproval  = msgs.M("role.approval")

	mTwoFactorEveryone  = msgs.M("twoFactor.everyone")
	mTwoFactorSensitive = msgs.M("twoFactor.sensitive")

	mClosedExpired  = msgs.M("closed.expired")
	mClosedRevoked  = msgs.M("closed.revoked")
	mClosedAccepted = msgs.M("closed.accepted")
)

// sensitivePerms are the sensitive permissions people know best, in the
// order "why two-factor" names them.
var sensitivePerms = []authz.Permission{
	authz.CampaignsSend, authz.CampaignsApprove, authz.ProvidersManage, authz.SubscribersExport,
	authz.TeamManage, authz.AutomationsPublish, authz.APIKeysManage,
}

// sensitiveReason names the most familiar sensitive permission a role
// holds: "Your role can send campaigns, so two-factor is required."
func sensitiveReason(l i18n.Lang, perms authz.Set) string {
	for _, p := range sensitivePerms {
		if perms.Has(p) {
			return msgs.M("twoFactor."+string(p)).Text(l, nil)
		}
	}
	return mTwoFactorSensitive.Text(l, nil)
}

// describeScope writes a member scope: "Beta Program list only".
func describeScope(l i18n.Lang, lists, domains []string) string {
	switch {
	case len(lists) > 0 && len(domains) > 0:
		return mScopeBoth.Text(l, i18n.Args{"lists": i18n.AndList(l.Locale, lists), "listCount": len(lists),
			"domains": i18n.AndList(l.Locale, domains), "domainCount": len(domains)})
	case len(lists) > 0:
		return mScopeLists.Text(l, i18n.Args{"lists": i18n.AndList(l.Locale, lists), "count": len(lists)})
	case len(domains) > 0:
		return mScopeDomains.Text(l, i18n.Args{"domains": i18n.AndList(l.Locale, domains), "count": len(domains)})
	}
	return ""
}

// roleDescription explains a role in plain words for the invitee.
func roleDescription(l i18n.Lang, roleKey, workspace string, perms authz.Set, lists []string, approvalPolicy []byte) string {
	switch roleKey {
	case authz.RoleOwner:
		return mRoleOwner.Text(l, i18n.Args{"workspace": workspace})
	case authz.RoleAdmin:
		return mRoleAdmin.Text(l, i18n.Args{"workspace": workspace})
	}
	where := i18n.Args{"lists": i18n.AndList(l.Locale, lists), "count": len(lists)}
	switch {
	case perms.Has(authz.CampaignsSend) && perms.Has(authz.CampaignsEdit):
		out := mRoleSend.Text(l, nil)
		if len(lists) > 0 {
			out = mRoleSendLists.Text(l, where)
		}
		if n := approvalMinimum(approvalPolicy); n > 0 {
			out += " " + mRoleApproval.Text(l, i18n.Args{"count": n})
		}
		return out
	case perms.Has(authz.CampaignsEdit):
		if len(lists) > 0 {
			return mRoleEditLists.Text(l, where)
		}
		return mRoleEdit.Text(l, nil)
	}
	return mRoleViewer.Text(l, nil)
}

// approvalMinimum is the audience size from which sends need a second
// person's approval, or 0 when the workspace has no such rule.
func approvalMinimum(policy []byte) int {
	var pol struct {
		Rules struct {
			AudienceMin struct {
				Enabled bool `json:"enabled"`
				N       int  `json:"n"`
			} `json:"audience_min"`
		} `json:"rules"`
	}
	if json.Unmarshal(policy, &pol) != nil || !pol.Rules.AudienceMin.Enabled {
		return 0
	}
	return max(pol.Rules.AudienceMin.N, 0)
}

// langOf is the language the request reads in.
func langOf(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }
