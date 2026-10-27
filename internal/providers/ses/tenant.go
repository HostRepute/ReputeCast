package ses

import (
	"context"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// SES tenants (review PR-11): each workspace sends as its own tenant, with
// its own suppression list (scope TENANT, for bounces and complaints) and
// its own reputation: SES's reputation policy pauses that tenant, never the
// whole account, so a bad list in one workspace can't stop another
// workspace or app on the same AWS account. The configuration sets and the
// sending domains are associated with the tenant, and sends name it.

var tenantChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// TenantName is the tenant of a workspace: its slug in the characters SES
// allows (letters, digits, - and _), at most 64.
func TenantName(slug, suffix string) string {
	n := strings.Trim(tenantChars.ReplaceAllString(slug, "-"), "-")
	if n == "" {
		n = "reputecast"
	}
	if suffix != "" && suffix != n {
		n += "-" + suffix
	}
	if len(n) > 64 {
		n = n[:64]
	}
	return n
}

// tenantSuppression keeps the tenant's own list of bounces and complaints.
func tenantSuppression() *types.TenantSuppressionAttributes {
	return &types.TenantSuppressionAttributes{
		SuppressionScope:  types.SuppressionListScopeTenant,
		SuppressedReasons: []types.SuppressionListReason{types.SuppressionListReasonBounce, types.SuppressionListReasonComplaint},
	}
}

// provisionTenant creates the workspace's tenant with its own suppression
// list, then reads the reputation policy SES applies to it.
func (p *Provider) provisionTenant(ctx context.Context, in providers.ProvisionInput) (providers.ProvisionResult, error) {
	name := TenantName(in.Tenant, in.Suffix)
	_, err := p.client.CreateTenant(ctx, &sesv2.CreateTenantInput{TenantName: aws.String(name), SuppressionAttributes: tenantSuppression()})
	switch {
	case isAWSCode(err, "AlreadyExistsException"):
		// A tenant from an earlier setup: it keeps its own list too.
		a := tenantSuppression()
		if _, err := p.client.PutTenantSuppressionAttributes(ctx, &sesv2.PutTenantSuppressionAttributesInput{TenantName: aws.String(name),
			SuppressionScope: a.SuppressionScope, SuppressedReasons: a.SuppressedReasons}); err != nil {
			return providers.ProvisionResult{}, callError("ses:PutTenantSuppressionAttributes", err)
		}
	case err != nil:
		return providers.ProvisionResult{}, callError("ses:CreateTenant", err)
	}
	policy := ""
	if t, err := p.client.GetTenant(ctx, &sesv2.GetTenantInput{TenantName: aws.String(name)}); err == nil && t.Tenant != nil && t.Tenant.TenantArn != nil {
		e, err := p.client.GetReputationEntity(ctx, &sesv2.GetReputationEntityInput{ReputationEntityReference: t.Tenant.TenantArn,
			ReputationEntityType: types.ReputationEntityTypeResource})
		if err == nil && e.ReputationEntity != nil {
			policy = policyName(aws.ToString(e.ReputationEntity.ReputationManagementPolicy))
		}
	}
	res := providers.ProvisionResult{Settings: map[string]any{"tenant": name, "tenant_reputation_policy": strings.ToLower(policy)}}
	res.Done, res.DoneText = say(mDoneTenant, i18n.Args{"tenant": name})
	res.Note, res.NoteText = say(mNoteTenant, i18n.Args{"hasPolicy": yesNo(policy != ""), "policy": policy})
	return res, nil
}

// policyName is a reputation policy's name from its ARN
// (".../reputation-policy/standard" → "Standard").
func policyName(arn string) string {
	if i := strings.LastIndexAny(arn, "/:"); i >= 0 {
		arn = arn[i+1:]
	}
	if arn == "" {
		return ""
	}
	return strings.ToUpper(arn[:1]) + arn[1:]
}

// associate links a resource (a configuration set or a sending domain) to
// the workspace's tenant, so sends that name the tenant may use it.
func (p *Provider) associate(ctx context.Context, resourceARN string) error {
	if p.s.Tenant == "" || resourceARN == "" {
		return nil
	}
	_, err := p.client.CreateTenantResourceAssociation(ctx, &sesv2.CreateTenantResourceAssociationInput{TenantName: aws.String(p.s.Tenant),
		ResourceArn: aws.String(resourceARN)})
	if err != nil && !isAWSCode(err, "AlreadyExistsException") {
		return callError("ses:CreateTenantResourceAssociation", err)
	}
	return nil
}

// resourceARN is the ARN of one of the account's SES resources in this
// region ("configuration-set/rc-broadcasts", "identity/news.acme.io");
// empty while the account isn't known.
func (p *Provider) resourceARN(kind, name string) string {
	if p.s.AccountID == "" {
		return ""
	}
	return "arn:aws:ses:" + p.s.Region + ":" + p.s.AccountID + ":" + kind + "/" + name
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
