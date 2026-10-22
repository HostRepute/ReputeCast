import { defineMessages } from '@/i18n'

/**
 * Instance admin › Users (InstanceUsers, InstanceUserDialogs): every account
 * on the instance, the sign-in policy snapshot, recent sign-in events and the
 * dialogs for two-factor resets, password resets, instance admins and
 * deactivation. Workspace names, sign-in event names and rejection details
 * come from the server as they are.
 */
export const m = defineMessages('admin.users', {
  documentTitle: 'Users',
  title: 'Users',
  titleCount: 'Users · {count, plural, one {# account} other {# accounts}}',
  caption: 'Every account on this instance. Deactivating here signs the user out of every workspace.',
  loadFailed: "Couldn't load the users",
  nothingChanged: 'Nothing was changed.',
  noMatch: 'No accounts match these filters. Clear the search or pick “Any” in the filters.',
  loading: 'Loading users',

  // Filters.
  searchLabel: 'Search users',
  searchPlaceholder: 'Search name or email',
  filterWorkspace: 'Filter by workspace',
  anyWorkspace: 'Any workspace',
  filterStatus: 'Filter by status',
  anyStatus: 'Any status',
  active: 'Active',
  deactivated: 'Deactivated',
  filter2FA: 'Filter by two-factor status',
  any2FA: 'Any 2FA',
  twoFAOn: 'On',
  twoFAOff: 'Off',
  twoFAIdp: 'Via IdP',

  // The table.
  tableCaption: 'Users',
  colUser: 'User',
  colWorkspaces: 'Workspaces · role',
  colSignIn: 'Sign-in methods',
  col2FA: '2FA',
  colLastSignIn: 'Last sign-in',
  colStatus: 'Status',
  colActions: 'Actions',
  instanceAdmin: 'Instance admin',
  noWorkspace: 'No workspace',
  never: 'never',
  locked: 'Locked',
  until: 'until {time}',

  // A user's menu.
  actionsFor: 'Actions for {name}',
  updateFailed: "Couldn't update {name}",
  canSignInAgain: '{name} can sign in again',
  membershipsBack: 'their workspace memberships are back',
  reactivateAccount: 'Reactivate account',
  signedOut: '{name} was signed out',
  sessionsEnded: '{count, plural, one {# session} other {# sessions}} ended',
  forceSignOut: 'Force sign-out',
  sessions: '{count, plural, =0 {no sessions} one {# session} other {# sessions}}',
  resetTwoFactor: 'Reset two-factor',
  reasonRequired: 'reason required',
  viaIdp: 'via IdP',
  noAuthenticator: 'no authenticator',
  sendPasswordReset: 'Send password reset',
  showCliCommand: 'Show CLI reset command',
  managedBy: 'managed by {provider}',
  noPassword: 'no password',
  lockoutCleared: 'the lockout was cleared',
  unlockSignIn: 'Unlock sign-in',
  removeAdmin: 'Remove instance admin',
  makeAdmin: 'Make instance admin',
  deactivateAccount: 'Deactivate account',
  thatsYou: "that's you",

  // Sign-in policy snapshot.
  policyTitle: 'Sign-in policy snapshot',
  authentication: 'Authentication',
  policyLoadFailed: "Couldn't load the policy.",
  ssoEnforced: 'SSO enforced',
  ssoEnforcedNone: 'No domain · password sign-in stays on for every domain',
  ssoEnforcedList: '{providers} (except break-glass Owners)',
  twoFA: '2FA',
  twoFAPolicyOff: 'Off',
  twoFAPolicy:
    'Required for {who, select, everyone {everyone} other {sensitive permissions}} · {grace, plural, =0 {no grace period} other {#-day grace}} · {inGrace, plural, one {# member} other {# members}} in grace',
  lockout: 'Lockout',
  lockoutValue: '{attempts, plural, one {# attempt} other {# attempts}} → {duration} · {locked, plural, one {# account} other {# accounts}} locked now',
  lockHours: '{count, plural, one {# hour} other {# hours}}',
  lockMinutes: '{count, plural, one {# minute} other {# minutes}}',

  // Recent sign-in events.
  signInsTitle: 'Recent sign-in events',
  instanceAuditLog: 'Instance audit log',
  signInsLoadFailed: "Couldn't load the sign-in events.",
  noSignIns: 'No sign-ins yet. Every attempt, allowed or rejected, appears here and in the instance audit log.',
  colWhen: 'When',
  colEvent: 'Event',
  colAccount: 'Account',
  colResult: 'Result',
  allowed: 'Allowed',
  thenAllowed: 'Then allowed',
  rejected: 'Rejected',
  yesterday: 'yesterday',

  // Dialogs.
  dialogEyebrow: 'Dialog · instance admin',

  // Reset two-factor.
  reasonMissing: 'A reason is required',
  passwordMissing: 'Confirm with your password',
  twoFactorReset: 'Two-factor reset for {name}',
  twoFactorResetDetail: '{name} was signed out and notified',
  resetTitle: 'Reset two-factor for {name}?',
  resetBody:
    'Removes their authenticator and recovery codes. {name} signs in with a password only until they set two-factor up again; the {days, plural, other {#-day}} grace period restarts. The reset is logged with your reason, and {name} is notified.',
  reasonLabel: 'Reason <k>· required, recorded in the audit log</k>',
  reasonPlaceholder: 'e.g. Lost phone, identity confirmed by video call on {date}',
  verifyFirst: 'Verify first',
  verifyBody: 'Confirm the request came from {name} (for example on a call), not from an email asking you to reset it.',
  passwordLabel: 'Confirm with your password',
  ssoConfirms: "You sign in with single sign-on, so your session's two-factor confirms this reset.",

  // Send a password reset.
  passwordResetTitle: 'Send a password reset to {name}',
  passwordResetBody: 'A one-time link that works for {minutes, plural, one {# minute} other {# minutes}}, sent on the System stream.',
  cliCopied: 'CLI command copied',
  cliCopiedDetail: 'run it on the server; it prints the link',
  copyFailed: "Couldn't copy",
  copyFailedDetail: 'select the command and copy it',
  copyCli: 'Copy CLI command',
  checkingSystem: 'Checking the System stream…',
  noSystemReason: 'No System-stream provider can send email yet. Use the CLI command.',
  resetSent: 'Reset link sent to {email}',
  resetSentDetail: 'via {via} · works for {minutes, plural, one {# minute} other {# minutes}}',
  sendResetEmail: 'Send reset email',
  systemStream: 'System stream',
  systemHealthy: '{hasProvider, select, yes {{provider}} other {The System-stream provider}} is healthy · the email goes out immediately.',
  systemUnhealthy:
    '{provider} is {status, select, degraded {degraded} down {down} sandbox {sandbox} paused {paused} other {{statusText}}} · the email may be delayed. The CLI command works either way.',
  noSystemLabel: 'No System provider',
  noSystemBody:
    "{hasProvider, select, yes {{provider} is on the System stream but can't send system mail right now, so the email can't go out.} other {No sending provider is connected to the System stream yet, so the email can't go out.}} Run the command below on the server instead.",
  cliNote: "No System provider, or the user can't receive email? Run this on the server instead; it prints the same link:",

  // Make instance admin.
  roleIn: '{role} in {workspace}',
  twoFAByIdp:
    '{hasProvider, select, yes {Two-factor is already enforced for them by {provider}} other {Two-factor is already enforced for them by their identity provider}}',
  twoFAIsOn: 'Two-factor is on ({method, select, passkey {passkey} other {authenticator app}})',
  twoFANotYet: "{name} doesn't use two-factor yet: ask them to add a passkey or an authenticator first",
  makeAdminTitle: 'Make {name} an instance admin?',
  madeAdmin: '{name} is an instance admin',
  managesInstance: 'Manages users, sign-in methods, SSO, security policies and workspaces for the whole instance',
  keepsRoles: 'Does not change their workspace roles ({roles})',
  noWorkspaces: 'Does not add them to any workspace',
  adminsNote:
    'Instance admins: {count, plural, =0 {none} other {{admins}}}. {name} would be the {count, plural, =0 {first.} =1 {second, which also gives you a backup for break-glass access.} =2 {third.} =3 {fourth.} =4 {fifth.} =5 {sixth.} =6 {seventh.} =7 {eighth.} =8 {ninth.} =9 {tenth.} other {{position}th.}}',

  // Remove instance admin.
  removeBadge: 'Instance admin',
  removeTitle: 'Remove {name} as instance admin?',
  removeLast: '{name} is the only instance admin. Make someone else an instance admin first, so the instance is never left without one.',
  removeBody: '{name} keeps their workspace roles but can no longer open Instance admin. Remaining: {remaining}.',
  lastAdmin: 'The last instance admin can’t be removed.',
  removedAdmin: '{name} is no longer an instance admin',

  // Deactivate an account.
  deactivateTitle: 'Deactivate {name}?',
  deactivateBody:
    'Signs {name} out of {sessions, plural, =0 {} one {# session and } other {# sessions and }}every workspace now. Their history, campaigns and audit entries stay; you can reactivate the account later.',
  deactivating: 'Deactivating…',
  deactivateReason: 'Reason · recorded in the audit log',
  soleOwner: '{name} is the only Owner of {workspaces}. Transfer ownership first.',
  deactivatedToast: '{name} deactivated',
  deactivatedDetail: '{sessions, plural, one {# session} other {# sessions}} ended{keys, plural, =0 {} one { · # API key revoked} other { · # API keys revoked}}',
  workspacesRow: 'Workspaces',
  revokeKeys: 'Revoke the {count, plural, one {# API key} other {# API keys}} and agents {name} created',
  revokeKeysHint: 'Leave unchecked to keep integrations running; they stop at the next permission change anyway.',
  soleOwnerNote: "{name} is the only Owner of {workspaces}. <badge>Blocked</badge> Transfer ownership in the workspace's settings first.",
})
