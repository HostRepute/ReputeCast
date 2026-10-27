import { defineMessages } from '@/i18n'

/**
 * Settings › Team › a member (A5 §5.8 member detail, TeamMember board) and
 * its remove dialog. Permission labels and groups, a permission's source and
 * scope, a key's scope, an audit entry's target and the server's notes come
 * from the API as they are; action names and permission keys are never
 * translated.
 */
export const m = defineMessages('team.member', {
  // The page while the member loads, and when they aren't found.
  member: 'Member',
  crumbSettings: 'Settings',
  crumbTeam: 'Team',
  notFoundBadge: 'Not found',
  notFoundTitle: "This member isn't in the workspace",
  notFoundBody: 'They may have been removed. The audit log still has their history.',
  backToTeam: 'Back to Team',
  loadFailed: "Couldn't load this member",
  loading: 'Loading the member',

  // Why the actions are off.
  manageNeeds: "Your role can't manage the team.",
  manageNeedsAsk: "Your role can't manage the team. Ask {helpers}.",
  cantDeactivateSelf: 'You can’t deactivate yourself.',
  leaveInstead: 'Leave the workspace from Account & security instead.',

  // The header: role, two-factor, activity and the actions.
  customRole: 'custom role',
  twoFAIdp: 'via IdP',
  twoFAOn: '{method, select, passkey {passkey} other {TOTP}}',
  twoFARequired: 'required · off',
  twoFAOff: 'off',
  lastActive: 'Last active {ago}',
  deactivatedOn: 'Deactivated {date}',
  joined: 'Joined {date}',
  changeRole: 'Change role or scope',
  remove: 'Remove',

  // Effective permissions.
  permissionsTitle: 'Effective permissions',
  permissionsCaption: 'What {name} can do and where it comes from: role, SSO group mapping or scope',
  colPermission: 'Permission',
  colEffective: 'Effective',
  colSource: 'Source',
  colScope: 'Scope',
  sensitive: 'Sensitive permission',
  // The word after the ⚑ mark.
  sensitiveFlag: 'sensitive',
  granted: 'Granted',
  // action: the action part of the permission key that includes this one ("manage").
  includedBy: '(included by {action})',

  // Recent activity, from the audit log.
  activityTitle: 'Recent activity',
  activityCaption: 'From the audit log',
  fullAuditLog: 'Full audit log',
  colWhen: 'When',
  colAction: 'Action',
  colTarget: 'Target',
  colResult: 'Result',
  noActivity: 'No activity in this workspace yet.',
  result: '{result, select, allowed {Allowed} denied {Denied} other {Error}}',
  // Why an entry was denied: the permission it needed (campaigns:send).
  needs: 'needs {permission}',

  // API keys they created.
  keysTitle: 'API keys they created',
  keysNoPermission: "None. This role can't create API keys.",
  keysNoneYet: 'None yet.',
  mcpAgent: 'MCP agent',
  expired: 'Expired',

  // Approvals they gave and requested.
  approvalsTitle: 'Approvals',
  given: 'Given',
  givenLast: '{count} · last {campaign} {date}',
  givenLastUnnamed: '{count} · last campaign {date}',
  givenNone: 'None yet',
  givenCantApprove: "None · their role can't approve campaigns",
  requested: 'Requested',
  requestedPending: '{count} pending · <link>{campaign}</link>',
  requestedNonePending: '{count} · none pending',

  // Sign-in and sessions.
  sessionsTitle: 'Sign-in and sessions',
  methods: 'Methods',
  methodsNone: 'None (deactivated)',
  twoFactor: 'Two-factor',
  twoFactorState:
    '{state, select, passkey {Passkey} totp {Authenticator app (TOTP)} idp {Enforced by the IdP} required {Required, not set up yet} other {Off}}',
  sessions: 'Sessions',
  sessionsActive: '{count} active',
  noActiveSessions: 'No active sessions.',
  forceSignOut: 'Force sign-out',
  signedOut: '{name} signed out',
  sessionsEnded: '{count, plural, one {# session} other {# sessions}} ended',
  signOutFailed: "Couldn't sign out",

  // Remove the member.
  removeBadge: 'Remove member',
  removeTitle: 'Remove {name} from {workspace}?',
  removeDescription: 'The membership is deleted; the audit log still names them. Deactivating instead keeps the membership so they can come back.',
  removeConfirm: 'Remove {name}',
  removing: 'Removing…',
  removed: '{name} removed',
  loggedAs: 'logged as {action}',
  rowAccess: 'Access',
  removeAccess: 'Loses access to {workspace} at once; other workspaces are unaffected',
  rowKeys: 'Keys',
  removeKeys: '{count, plural, one {# API key or agent they created is revoked} other {# API keys or agents they created are revoked}}',
  rowKeeps: 'Keeps',
  removeKeeps: 'Campaigns, templates and approvals they made stay in the workspace',
})
