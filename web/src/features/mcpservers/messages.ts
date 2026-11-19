import { defineMessages } from '@/i18n'

/**
 * Settings › Integrations › MCP servers (assistant brief §4, the
 * MCPServers board): remote MCP servers, their tools and each member's own
 * sign-in. Server, tool and people's names, addresses, header names,
 * protocol words (inputSchema, OAuth, SSRF, MCP) and setting keys are
 * arguments or stay as written, never translated.
 */
export const m = defineMessages('mcpservers', {
  title: 'MCP servers',
  documentTitle: 'MCP servers · Settings',
  crumbSettings: 'Settings',
  crumbIntegrations: 'Integrations',
  loadError: 'Couldn’t load the MCP servers',

  // ── Read-only ───────────────────────────────────────────────────────
  // name, role: the person and their role; helpers: who can change it ("Linh Nguyen or Marcus Chen").
  readOnly: '<b>Read-only · {name}, {role}.</b> You can see the servers, their tools and how each tool is allowed, and connect your own account. Only people who can manage integrations can add servers or review tools. Ask {helpers}.',
  readOnlyAlone: '<b>Read-only · {name}, {role}.</b> You can see the servers, their tools and how each tool is allowed, and connect your own account. Only people who can manage integrations can add servers or review tools.',

  // ── A tool changed after approval ───────────────────────────────────
  // server, tool: "Internal wiki · search changed after approval".
  changedNotice: '{server} · {tool} changed after approval',
  changedNoticeBody: 'The tool is off until a manager reviews it.',
  changedNoticeShown: 'The tool is off until a manager reviews it. The change is shown below.',
  reviewChange: 'Review the change',

  // ── Servers ─────────────────────────────────────────────────────────
  serversTitle: 'Servers',
  addServer: 'Add server',
  emptyTitle: 'No MCP servers yet',
  emptyBody: 'Add a remote MCP server to let the assistant call its tools. Each tool waits for a manager’s review before any thread can use it.',
  emptyReadOnly: 'No MCP servers yet. People who can manage integrations add them.',
  badgeTools: '{count, plural, one {# tool} other {# tools}}',
  badgeChanged: '{count, plural, one {# tool changed} other {# tools changed}}',
  badgeWaiting: '{count, plural, one {# tool to review} other {# tools to review}}',
  badgeOff: 'Off',
  badgeUnreachable: 'Unreachable',
  badgeSignIn: 'Sign-in needed',
  badgeNotSynced: 'Not synced',
  authNone: 'No sign-in',
  authHeaderSet: 'Shared header secret, set',
  authHeaderUnset: 'Shared header secret, not set',
  authOAuth: 'OAuth, each member signs in',
  // url: the server's address; auth: how it signs in ("OAuth, each member signs in").
  serverLine: '{url} · {auth}',
  // roles: role names ("Owner and Admin").
  usedByRoles: 'Only {roles} can use it',
  showTools: 'Show tools',
  hideTools: 'Hide tools',
  showToolsLabel: 'Show tools of {server}',
  hideToolsLabel: 'Hide tools of {server}',
  connect: 'Connect your account',
  connectLabel: 'Connect your account to {server}',
  // name: the signed-in person's name.
  connectedAs: 'Connected as {name}',
  reconnect: 'Reconnect',
  reconnectLabel: 'Reconnect your account to {server}',
  disconnect: 'Disconnect',
  disconnectLabel: 'Disconnect your account from {server}',
  sharedForWorkspace: 'Shared for the workspace',
  noSignInNeeded: 'No sign-in needed',
  notForRole: 'Not available to your role',
  // detail: the server's problem, as the last sync recorded it.
  statusDetail: 'Last sync: {detail}',
  // date, time: "Synced 18 Nov · 10:42".
  syncedAt: 'Synced {date} · {time}',
  edit: 'Edit',
  editLabel: 'Edit {server}',
  serversFoot: 'Remote servers only. Every request goes through the SSRF guard; private addresses need the instance setting.',

  // ── Tools ───────────────────────────────────────────────────────────
  toolsTitle: '{server} tools',
  toolsCaption: 'Descriptions come from the server and are untrusted until a manager reviews them.',
  tableLabel: '{server} tools table',
  sync: 'Sync tools',
  syncing: 'Syncing…',
  syncNeedsSignIn: 'Connect your account to {server} first: a sync lists its tools as you.',
  thTool: 'Tool',
  thKind: 'Kind',
  thState: 'State',
  thActions: 'Actions',
  kindReadOnly: 'Read-only',
  kindWrites: 'Writes',
  kindBlocked: 'Blocked',
  kindPending: 'Not decided',
  stateApproved: 'Approved',
  stateAsks: 'Asks every time',
  stateBlocked: 'Blocked',
  stateChanged: 'Changed · off',
  statePending: 'Waiting for review',
  stateRemoved: 'Gone from the server',
  review: 'Review',
  reviewLabel: 'Review {tool}',
  view: 'View',
  viewLabel: 'View {tool}',
  noTools: 'No tools yet. Sync to list the server’s tools.',
  noToolsSignIn: 'Connect your account, then sync to list the server’s tools.',
  noToolsReadOnly: 'No tools listed yet.',
  noDescription: '(no description)',

  // ── Review ──────────────────────────────────────────────────────────
  reviewTitle: 'Review {tool}',
  reviewFrom: 'From {server}. The description is the server’s own text; read it as untrusted.',
  untrustedLabel: 'Description and schema of {tool}, as the server sent them',
  // The pre's labels stay as the protocol names them: inputSchema, annotations.
  preSchema: 'inputSchema: {schema}',
  preAnnotations: 'annotations (hints only): {annotations}',
  kindLegend: 'Kind',
  optReadOnly: 'Read-only · runs after approval',
  optWrites: 'Writes · asks the person every time',
  optBlocked: 'Blocked · never offered to the model',
  pinNote: 'Approving pins this description and schema. Any change turns the tool off.',
  // hash: the first characters of the version's hash.
  version: 'Version {hash}',
  stale: 'This tool changed on the server while you reviewed it. The new version is shown; review it again.',
  savedReadOnly: '{tool} approved as read-only',
  savedWrites: '{tool} approved; it asks every time',
  savedBlocked: '{tool} blocked',
  decideFailed: 'Couldn’t save the decision for {tool}',

  // ── The change ──────────────────────────────────────────────────────
  changedBadge: 'Changed after approval',
  changedDescription: '{tool} changed its description',
  changedSchema: '{tool} changed its schema',
  changedBoth: '{tool} changed its description and schema',
  changedAnnotations: '{tool} changed its annotations',
  changedUnknown: '{tool} changed after approval',
  before: 'Before',
  after: 'After',
  partDescription: 'Description',
  partSchema: 'Input schema',
  partAnnotations: 'Annotations (hints only)',
  changedBody: 'Read the new text as the model would. The tool stays off until a manager decides.',
  noApproved: 'The approved version wasn’t kept, so there is nothing to compare. Review the tool as it is now.',
  blockTool: 'Block the tool',
  approveNew: 'Approve the new version',
  changedHelpers: '{helpers} can review this change.',
  changedNobody: 'People who can manage integrations review this change.',

  // ── How calls are allowed ───────────────────────────────────────────
  howTitle: 'How calls are allowed',
  howReadOnly: 'Read-only tools run after a manager approves them.',
  howWrites: 'Tools that write ask the person every time, with the exact arguments.',
  howChanged: 'A changed description or schema turns the tool off.',
  howKeys: 'Servers never receive ReputeCast keys, and their requests for model sampling are refused.',

  // ── Add and edit ────────────────────────────────────────────────────
  addTitle: 'Add an MCP server',
  editTitle: 'Edit {server}',
  fieldName: 'Name',
  fieldUrl: 'Server address',
  fieldUrlHint: 'The server’s MCP endpoint: Streamable HTTP, or HTTP with SSE.',
  fieldAuth: 'Sign-in',
  optOAuth: 'OAuth, each member signs in',
  optHeader: 'Shared header secret',
  optNone: 'None',
  fieldHeaderName: 'Header name',
  fieldHeaderValue: 'Header value',
  secretHint: 'Write-only. ReputeCast never shows it again and only sends it to this server.',
  secretSet: 'Set',
  replace: 'Replace',
  keepCurrent: 'Keep the current value',
  clientSummary: 'Use a client registered with the server',
  fieldClientId: 'Client ID',
  fieldClientSecret: 'Client secret',
  clientHint: 'Optional. Leave both empty when the server registers ReputeCast itself.',
  // uri: the callback address to register with the server's OAuth app.
  redirectHint: 'Redirect address to register: {uri}',
  // The server's client is this instance's Client ID Metadata Document (its address is the client ID).
  clientMetadata: 'Client: this instance (client metadata document)',
  // A sign-in that stopped on the server's client: the server's own detail follows.
  noClientTitle: 'Enter a client for {server}',
  clientAuthTitle: '{server}’s sign-in method isn’t supported',
  fieldWho: 'Who can use it',
  whoAll: 'Everyone with “Use the AI assistant”',
  whoManagers: 'Owners and Admins',
  // roles: role names, as a list.
  whoCustom: 'Only {roles}',
  enabled: 'Offer its tools to threads',
  enabledHint: 'Off: threads stop offering its tools. Nothing is removed.',
  privateNote: 'Remote servers only. Addresses on private networks need the instance setting <mono>providers.allow_private_networks</mono>.',
  addAndSync: 'Add and sync tools',
  removeServer: 'Remove server',
  nameRequired: 'Enter a name.',
  urlInvalid: 'Enter the server’s address, starting with https://.',
  headerNameRequired: 'Enter the header’s name.',
  headerValueRequired: 'Enter the header’s value.',
  // count: the tools found by the first sync.
  addedSynced: 'Added {server}: {count, plural, one {# tool} other {# tools}} to review',
  addedSignIn: 'Added {server}. Connect your account to list its tools.',
  addedSyncFailed: 'Added {server}, but its tools couldn’t be listed',
  saved: 'Saved {server}',
  synced: 'Synced {server}: {count, plural, one {# tool} other {# tools}}',
  syncFailed: 'Couldn’t sync {server}',
  removeTitle: 'Remove {server}?',
  removeBadge: 'Remove',
  removeBody: 'Its tools and every member’s sign-in to it are removed. Threads stop offering its tools, and a call waiting for confirmation fails.',
  removed: 'Removed {server}',

  // ── Sign-in (OAuth) ─────────────────────────────────────────────────
  connectedToast: 'Connected to {server}',
  deniedToast: 'You cancelled the sign-in to {server}',
  tokenToast: '{server} didn’t complete the sign-in',
  tokenDetail: 'Try again. If it keeps failing, a manager may need to check the server’s OAuth settings.',
  disconnectedToast: 'Disconnected from {server}',
  signInFailed: 'Couldn’t start the sign-in to {server}',
  disconnectFailed: 'Couldn’t disconnect from {server}',
  aServer: 'the server',
})
