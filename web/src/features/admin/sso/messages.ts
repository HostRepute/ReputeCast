import { defineMessages } from '@/i18n'

/**
 * Instance admin › SSO providers (SSOStates): the providers list, turning on
 * domain enforcement and what a provider's page shares with the list. A
 * provider's setup is in setup.messages.ts. Provider names, claim names,
 * scopes, URLs and redirect URIs stay as they are; the providers'
 * two-factor line and the enforcement preview's sign-in methods come from
 * the server in English.
 */
export const m = defineMessages('admin.sso', {
  // The providers list.
  listTitle: 'SSO providers',
  listCaption: 'Configured at instance level · presets for Google Workspace, Microsoft Entra ID, Okta, Keycloak, Authentik and generic OIDC',
  addProvider: 'Add SSO provider',
  listLoadFailed: "Couldn't load the SSO providers",
  listLoadFailedBody: 'Sign-in keeps working with the saved providers.',
  listLoading: 'Loading SSO providers',
  emptyTitle: 'No SSO provider yet',
  emptyBody: "Connect your organization's identity provider over OpenID Connect: people sign in with it, accounts can be created automatically and groups can map to roles.",

  // A provider's card. cardState follows its domains: "acme.io · enabled Oct 3, 2026".
  cardAnyDomain: 'any domain',
  cardState: '{state, select, enabledOn {enabled {date}} enabled {enabled} draft {setup in progress} other {disabled}}',
  cardIssuer: 'Issuer',
  cardDomains: 'Allowed domains',
  cardAnyDomainValue: 'Any domain',
  cardJit: 'JIT',
  cardJitOn: 'On · {workspace} · {role}',
  cardJitOff: 'Off · only existing accounts sign in',
  cardGroupMapping: 'Group mapping',
  cardMappings: '{count, plural, =0 {None yet} one {# mapping} other {# mappings}}',
  cardEnforced: 'Enforced',
  cardEnforcedYes: 'Yes · password off for {domains}{count, plural, =0 {} one {, except break-glass Owner {names}} other {, except break-glass Owners {names}}}',
  cardEnforcedNo: 'No · password sign-in stays on',
  cardTwoFactor: 'Two-factor',
  cardDiscovery: 'Discovery',
  cardDiscoveryFailed: 'Failed · open the setup to fix the issuer',
  testSignIn: 'Test sign-in',
  testStartFailed: "Couldn't start the test sign-in",
  continueSetup: 'Continue setup',
  enable: 'Enable',
  // After enabling a provider, here or on its page.
  enabledToast: '{name} enabled',
  enabledDetail: 'its button is on the sign-in page',
  enableFailed: "Couldn't enable {name}",
  disable: 'Disable',
  disableBadge: 'Disable SSO provider',
  disableTitle: 'Disable {name}?',
  disableBody:
    "Its button leaves the sign-in page and {count, plural, one {# person} other {# people}} who sign in with it can't until it's enabled again{enforced, select, yes {; enforcement for {domains} ends and password sign-in comes back there} other {}}. Settings, mappings and linked identities are kept.",
  disabledToast: '{name} disabled',
  deleteBadge: 'Delete SSO provider',
  deleteTitle: 'Delete {name}?',
  deleteBody: 'Deletes the settings, the client secret and {count, plural, one {# linked identity} other {# linked identities}}. Accounts stay; people sign in another way.',
  deleteConfirm: 'Delete provider',
  deleteTyped: 'Type {name} to confirm',
  deletedToast: '{name} deleted',

  // Google Workspace has no groups claim over OIDC: on the card and in step 4.
  groupMappingUnavailable:
    "<b>Group mapping is unavailable for {name}.</b> {google, select, yes {Google doesn't send a groups claim over OIDC.} other {This identity provider sends no groups claim.}} Map people by email domain (JIT default role) or assign roles individually in Team.",

  // Turning on domain enforcement (EnforcementDialog). The Today and After cells are the server's, in English.
  enforceEyebrow: 'Dialog · turning on domain enforcement',
  enforceTitle: 'Enforce {provider} sign-in for {domains}?',
  enforceBody: 'Password sign-in stops working for {domains} accounts at their next sign-in. Passkeys and social sign-ins for this domain are disabled too.',
  enforceLoading: 'Loading who is affected…',
  enforceConflictReason: '{provider} already enforces this domain',
  enforcedToast: '{provider} is enforced for {domains}',
  enforcedDetail: 'break-glass Owners keep their password',
  enforceConfirm: 'Turn on enforcement',
  enforceTable: 'Who signs in how',
  colMember: 'Member',
  colToday: 'Today',
  colAfter: 'After',
  enforceNobody: 'Nobody has an account at {domains} yet. People who sign in later go straight to {provider}.',
  breakGlassLabel: 'Break-glass',
  // {owners}: the breakGlassOwner items, as a list ("A, B and C").
  breakGlassKeeps: 'Keeps password + two-factor access, even with enforcement on: {owners}. Their password sign-ins are logged as break-glass.',
  breakGlassOwner: '<b>{name}</b> (Owner, {workspace})',
  noBreakGlassLabel: 'No break-glass',
  noBreakGlass: 'No Owner at {domains} keeps a password sign-in. If {provider} is down, nobody at {domains} can sign in; ask an instance admin at another domain to keep access.',
  unlinkedLabel: '{count, plural, one {# member} other {# members}}',
  // {names}: the members' names in bold, as a list; {short}: the provider's first word ("Google").
  unlinkedBody:
    '{names} {count, plural, one {signs in with a password today and has} other {sign in with a password today and have}} no {provider} account linked yet. {count, plural, one {They will be asked} other {They will each be asked}} to continue with {short} at their next sign-in; their role and history stay.',
  enforceConflict: '{provider} already enforces one of these domains. Turn it off there first.',
})
