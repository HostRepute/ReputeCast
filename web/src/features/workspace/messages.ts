import { defineMessages } from '@/i18n'

/**
 * Settings › General (WorkspaceSettings), its transfer and delete dialogs
 * (WorkspaceDialogs) and Create a workspace (A5 §3.4). The workspace's name
 * and slug, timezone names, the approval rules, the sign-in summary and the
 * two-factor notes of the transfer candidates come from the API as they are.
 */
export const m = defineMessages('workspace', {
  // Settings › General.
  title: 'Settings',
  loadFailed: "Couldn't load the workspace settings",
  nothingChanged: 'Nothing was changed.',
  loading: 'Loading the workspace settings',
  ownerOnly: 'Only the Owner can do this.',

  // Workspace card.
  workspaceTitle: 'Workspace',
  workspaceCaption: 'Name, default sender and timezone',
  nameLabel: 'Workspace name',
  nameRequired: 'Give the workspace a name.',
  timezoneLabel: 'Timezone',
  timezoneHint: 'Schedules and reports use this timezone.',
  senderLabel: 'Default sender',
  noSenders: 'No senders yet',
  senderHintEmpty: 'Add a sending domain and sender in <link>senders & domains</link>.',
  senderHint: 'New campaigns start with this sender. <link>senders & domains</link>.',
  // providers: the sending providers that verified the sender, joined as "SES and Postal".
  senderHintVerified: 'New campaigns start with this sender. Verified on {providers}; <link>senders & domains</link>.',
  idLabel: 'Workspace ID',
  idHint: 'Used in API URLs and the audit log.',
  saved: 'Workspace settings saved',
  unsaved: "Your changes to the workspace name, timezone or default sender aren't saved.",

  // Send approvals card.
  approvalsTitle: 'Send approvals',
  approvalsCaption: 'Who must sign off before a risky send',
  approvalNeeded: 'Approval needed when',
  noRule: 'Never: no rule is on',
  approvers: 'Approvers',
  approversValue:
    'Members with campaigns:approve: {hasApprovers, select, yes {{names}} other {nobody yet}} · {count, plural, one {# approval} other {# approvals}} required',
  automations: 'Automations',
  automationsOn: 'Publishing needs approval',
  automationsOff: 'Publishing does not need approval',
  bypass: 'Emergency bypass',
  bypassOn: 'Owners, with a reason · logged, approvers notified',
  editPolicy: 'Edit approval policy',
  inbox: 'Approval inbox · {count} pending',

  // Members and access card.
  membersTitle: 'Members and access',
  membersCaption: 'Summary · managed in Team and Roles',
  members: 'Members',
  membersValue: '{active} active · {invited} invited · {deactivated} deactivated',
  roles: 'Roles',
  // builtin: the built-in roles, joined with commas; custom: the custom roles' names in quotes.
  rolesCustom: '{builtin} + custom {custom}',
  signIn: 'Sign-in',
  teamButton: 'Team',
  rolesButton: 'Roles & permissions',

  // Danger zone.
  dangerTitle: 'Danger zone',
  dangerCaption: 'Owner only · both actions are logged',
  transferTitle: 'Transfer ownership',
  transferBody: 'Make another member the Owner. You become an Admin. Only the Owner can do this.',
  transferButton: 'Transfer ownership',
  deleteTitle: 'Delete this workspace',
  deleteBody:
    "Deletes {contacts, plural, one {{contactsText} contact} other {{contactsText} contacts}}, {campaigns, plural, one {{campaignsText} campaign} other {{campaignsText} campaigns}}, {automations, plural, one {# automation} other {# automations}}, providers and routing. Suppressions are kept at instance level. You'll be asked to type the name and offered an export first.",
  deleteButton: 'Delete workspace',

  // Transfer ownership (WorkspaceDialogs, left).
  transferEyebrow: 'Dialog · Owner only',
  transferDialogTitle: 'Transfer ownership of {name}',
  transferDescription:
    'The new Owner gets every permission, including deleting the workspace and transferring it again. You become an Admin and keep your break-glass sign-in as an instance admin.',
  transferPick: 'Pick a member with two-factor.',
  transferConfirm: 'Transfer ownership',
  newOwner: 'New owner',
  noCandidates: 'No other active members yet',
  candidateNoTwoFactor: '{name} · {role} · {email} (no two-factor)',
  transferHint: 'Only active members with two-factor can become Owner.',
  // reason: the server's sentence, "Linh Nguyen has no two-factor yet."
  askTwoFactor: '{reason} Ask them to add a passkey or an authenticator app first.',
  passwordLabel: 'Confirm with your password',
  passwordRequired: 'Enter your password to confirm.',
  reauthHint: 'Re-authentication is required for this action.',
  ssoNote: 'You sign in through single sign-on: the transfer is accepted within {minutes, plural, one {# minute} other {# minutes}} of signing in.',
  transferred: '{name} is now the Owner',
  transferredDetail: 'you are an Admin',

  // Delete the workspace (WorkspaceDialogs, right).
  deleteEyebrow: 'Dialog · Owner only · typed confirmation',
  deleteDialogTitle: 'Delete {name}?',
  deleteDescription:
    'This permanently deletes the workspace after a {days}-day hold during which an instance admin can restore it. Scheduled sends stop immediately.',
  deleteContacts:
    '{contacts, plural, one {{contactsText} contact} other {{contactsText} contacts}}, {lists, plural, one {# list} other {# lists}}, {segments, plural, one {# segment} other {# segments}} and consent history',
  deleteCampaigns:
    '{campaigns, plural, one {{campaignsText} campaign} other {{campaignsText} campaigns}} with reports, {automations, plural, one {# automation} other {# automations}}, {templates, plural, one {# template} other {# templates}}',
  deleteProviders:
    '{count, plural, one {# sending provider} other {# sending providers}}, routing rules and warm-up plans (provider accounts themselves are untouched)',
  deleteKeys: 'API keys and MCP agents of this workspace stop working',
  deleteKept: "<b>Kept:</b> the shared suppression list, the audit log and the members' accounts",
  recommended: 'Recommended',
  exportBody: 'Export everything first: contacts with consent, campaigns as MJML, reports as CSV, audit log as JSON.',
  exportButton: 'Export everything first',
  exportNeeds: 'Exporting contacts needs the subscribers:export permission.',
  // name: the workspace's name, which must be typed as it is.
  typeName: 'Type <b>{name}</b> to confirm',
  mustMatch: 'Must match exactly.',
  typeToEnable: 'Type the workspace name to enable',
  deleteConfirm: 'Delete workspace',
  deleted: '{name} deleted',
  deletedDetail: 'an instance admin can restore it until {date}',
  deleteFailed: "Couldn't delete the workspace",

  // Create a workspace.
  createTitle: 'Create a workspace',
  createIntro:
    'You become its Owner. Each workspace keeps its own contacts, campaigns, providers and team; the suppression list is shared across the instance.',
  namePlaceholder: 'Acme Newsletter',
  createNameRequired: 'Give the workspace a name, for example your newsletter or brand.',
  createTimezoneHint: 'Schedules and reports use this timezone. You can change it later in Settings.',
  notCreated: 'Not created',
  createButton: 'Create workspace',
  backTo: 'Back to {name}',
})
