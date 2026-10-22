import { defineMessages } from '@/i18n'

/**
 * First sign-in setup (A5 §5.5, InstanceSetup board): the install Owner's
 * steps and the personal steps other people must finish. `{command}` is the
 * CLI command (`reputecast install`), never translated.
 */
export const m = defineMessages('auth.setup', {
  documentTitle: 'Set up {name} · ReputeCast',

  signInToContinue: 'Sign in to continue setup',
  loadFailed: "Setup couldn't load",
  signedOut: 'Signed out',
  error: 'Error',
  sessionEnded: 'Your session ended. Sign in again; the steps you finished are saved.',
  nothingChanged: 'Nothing was changed.',
  signIn: 'Sign in',
  loading: 'Loading setup…',

  stepOf: 'Step {n} of {total}',
  twoFactorOn: 'Two-factor is on',
  accountProtected: 'Your account is protected.',

  progressLabel: 'Setup progress',
  setUpName: 'Set up {name}',
  finishAccount: 'Finish setting up your account',
  signedInOwner: 'Signed in as {email}, the Owner created by <mono>{command}</mono>',
  signedInReason: 'Signed in as {email}. {reason}',
  stepsLabel: 'Setup steps',
  stepLabel:
    '{step, select, password {Replace the temporary password} second_factor {Add a passkey or authenticator} recovery_codes {Save your recovery codes} instance {Instance basics} workspace {Create the first workspace} provider {Connect a sending provider} other {{label}}}',
  stepCompleted: '(completed)',
  stepCurrent: '(current step)',
  doneSoFar: 'Done so far',
  doneTempPassword: 'Temporary password replaced',
  donePasskey: 'Passkey “{name}” added',
  doneAuthenticator: 'Authenticator app set up',
  doneRecovery: '{count, plural, one {# recovery code downloaded} other {# recovery codes downloaded}}',
  doneInstance: 'Instance basics saved',
  doneWorkspace: 'Workspace “{name}” created',
  firstOwnerRequired: "Two-factor is required for the first Owner; it can't be skipped.",

  // ── Password ─────────────────────────────────────────────────────────
  passwordTitle: 'Replace the temporary password',
  passwordSub: 'The password printed by <mono>{command}</mono> works only until you choose your own. Choose one only you know.',
  tempPasswordRequired: 'Enter the temporary password from the terminal.',
  passwordTooShort: 'Use at least {min} characters. Passphrases are fine.',
  passwordsDiffer: "The two passwords don't match.",
  notSaved: 'Not saved',
  tempPassword: 'Temporary password',
  newPassword: 'New password',
  passwordHint: 'At least {min} characters. Passphrases are fine.',
  passwordHintBreached: 'At least {min} characters. Checked against known breaches.',
  confirmPassword: 'Confirm new password',
  saveContinue: 'Save and continue',

  // ── Second factor ────────────────────────────────────────────────────
  totpTitle: 'Set up an authenticator app',
  totpSub: 'Scan the QR code, then enter the code the app shows to confirm.',
  secondFactorTitle: 'Add a passkey or an authenticator app',
  passkeyQuickest: 'A passkey is the quickest: Touch ID, Face ID, Windows Hello or a security key.',
  passkeyAdded: 'Passkey added',
  addPasskey: 'Add a passkey (recommended)',
  setUpAuthenticator: 'Set up an authenticator app',
  changeLater: 'You can add more passkeys and change your authenticator later in Account & security.',

  // ── Recovery codes ───────────────────────────────────────────────────
  recoveryTitle: 'Save your recovery codes',
  recoverySub: 'Each code signs you in once if you lose your passkey or authenticator. You see them only now.',
  shownOnce: 'Shown once',
  earlierCodes: "Your earlier codes can't be shown again. Generate a new set to save them; the old ones stop working.",
  generateCodes: 'Generate recovery codes',

  // ── Instance basics ──────────────────────────────────────────────────
  instanceTitle: 'Instance basics',
  instanceSub: 'These settings apply to every workspace on this installation. You can change them later in Instance admin.',
  instanceNameRequired: 'Give the instance a name, like Acme ReputeCast.',
  urlInvalid: 'Use the full address people open, like https://reputecast.acme.io.',
  instanceName: 'Instance name',
  instanceNameHint: 'Shown on the sign-in page and in emails to your team.',
  defaultTimezone: 'Default timezone',
  defaultTimezoneHint: 'New workspaces start with this timezone.',
  publicUrl: 'Public URL',
  checkAgain: 'Check again',
  publicUrlHint:
    'Used for provider webhooks, SSO redirect URIs, public pages and the links in every email. Changing it later breaks existing webhooks until they are re-registered.',
  urlCheckLabel: 'URL check result',
  checkReachable: 'Reachable',
  checking: 'Checking…',
  checked: 'Checked {ago}',
  secondsAgo: '{n} s ago',
  minutesAgo: '{n} min ago',
  checkingHost: 'Checking {host}…',
  updateCheck: 'Check GitHub for new releases once a day',
  updateCheckHint: 'Only the version number is fetched. ReputeCast never sends usage data anywhere. You can turn this off for offline installs.',
  saveCreateWorkspace: 'Save and create the first workspace',

  // ── First workspace ──────────────────────────────────────────────────
  workspaceTitle: 'Create the first workspace',
  workspaceSub: 'A workspace has its own lists, campaigns, automations, sending providers and team. You can create more later from the workspace switcher.',
  workspaceNameRequired: 'Give the workspace a name, like Acme Newsletter.',
  notCreated: 'Not created',
  workspaceName: 'Workspace name',
  workspaceNamePlaceholder: 'Acme Newsletter',
  workspaceNameHint: 'Shown in the workspace switcher and on public pages until you set a brand.',
  timezone: 'Timezone',
  timezoneHint: 'Schedules and reports use this timezone.',
  createWorkspace: 'Create workspace',

  // ── Sending provider ─────────────────────────────────────────────────
  providerTitle: 'Connect a sending provider',
  providerNeedsWorkspace: 'Create a workspace first.',
  continue: 'Continue',
  providerSub:
    'ReputeCast sends through your own provider: Amazon SES, Postmark, Mailgun, SendGrid, Resend, a Postal server or any SMTP server. Connecting one takes about 10 minutes.',
  ready: 'Ready',
  readyBody:
    "{instance} is set up and {workspace} is created; you're its Owner. Invites, password resets and notifications by email start working once a provider handles the System stream.",
  skipForNow: 'Skip for now',
  connectProvider: 'Connect a sending provider',
})
