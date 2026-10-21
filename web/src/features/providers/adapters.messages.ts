import { defineMessages } from '@/i18n'

/**
 * Sending providers › Adapters (ProvidersAdapters): built-in and community
 * adapters, the generic HTTP adapter (with http-adapter-model) and the SDK.
 * Code (interface and function names, header names, JSONPath, the
 * `{{ secrets.api_key }}` token, event type names) comes in as values.
 */
export const ma = defineMessages('providers.adapters', {
  adminReason: 'Adapters apply to every workspace on this instance, so only instance admins can install, update or remove them.',
  sdkDocs: 'Adapter SDK docs',
  installFromRegistry: 'Install from registry',
  loadFailed: "Couldn't load the adapters",
  loadFailedBody: 'Connected providers keep sending.',
  loading: 'Loading adapters',

  // ── Built-in adapters ────────────────────────────────────────────────
  builtinTitle: 'Built-in adapters · {count}',
  builtinLine: 'Ship with ReputeCast, maintained by HostRepute, updated with each release',
  /** {version} and {latest} are ReputeCast versions ("1.0.3"). */
  versionUpToDate: 'ReputeCast {version} · up to date',
  versionAvailable: 'ReputeCast {version} · {latest} available',
  colAdapter: 'Adapter',
  colVersion: 'Version',
  colMaintainer: 'Maintainer',
  colEvents: 'Events',
  colDomainSetup: 'Domain setup',
  colSuppressionSync: 'Suppression sync',
  colStatus: 'Status',
  /** {count} is the number of workspaces that connected the adapter. */
  connectedCount: 'Connected · {count}',
  available: 'Available',

  // ── Community adapters ───────────────────────────────────────────────
  communityTitle: 'Community adapters',
  communityLine: 'Installed from the adapter registry. Each runs in its own process with only the permissions its manifest declares.',
  browseRegistry: 'Browse the registry',
  colCapabilities: 'Capabilities declared',
  colUpdate: 'Update',
  communityMaintainer: 'community · {maintainer}',
  /** {version} is the adapter's newer version ("2.1.0"). */
  versionAvailableShort: '{version} available',
  unmaintained: 'Unmaintained',
  unmaintainedSince: 'Unmaintained · last release {months, plural, one {# month ago} other {# months ago}}',
  upToDate: 'Up to date',
  installedNotConnected: 'Installed · not connected',
  noCommunity: 'No community adapters yet. Install one from the registry, or map any HTTP API with the generic adapter.',
  updateTo: 'Update {name} to {version}',
  removeNamed: 'Remove {name}',
  updated: '{name} updated to {version}',
  updateFailed: "The adapter wasn't updated",
  removed: '{name} removed',
  removedNoName: 'Adapter removed',
  removeFailed: "The adapter wasn't removed",
  removeTitle: 'Remove {name}?',
  removeBody: "It isn't connected in any workspace, so nothing sends through it. You can install it again from the registry later.",
  keepIt: 'Keep it',
  remove: 'Remove',

  // ── Generic HTTP adapter ─────────────────────────────────────────────
  httpTitle: 'Generic HTTP adapter',
  httpLine: 'Send through any HTTP API without writing code. Map the request, response and webhook once.',
  sendEndpoint: 'Send endpoint',
  requestHeaders: 'Request headers',
  /** {token} is the template token "{{ secrets.api_key }}". */
  requestHeadersHint: 'One per line, as Name: value. {token} is the key entered when connecting.',
  bodyTemplate: 'Request body template',
  messageIdPath: 'Message id in response',
  webhookMapping: 'Webhook mapping',
  /** {types} lists event type names as providers send them ("delivered, bounce, spamreport…"). */
  jsonpathHint: 'Event types are read as {types}; map other names in a manifest.',
  mappingJsonpath: 'Custom JSONPath map',
  webhookAuth: 'Webhook authentication',
  testSample: 'Test with a sample message',
  testReason: 'Test sends need the generic HTTP adapter in this build; save the mapping now and test it once it ships.',
  saveAsAdapter: 'Save as adapter',
  sampleRequest: 'Sample request',
  sampleNote: 'The request for a sample message. Nothing was sent; secrets show as ••••.',
  saveDialogBody: 'It shows up in Add provider for every workspace on this instance.',
  saveAdapter: 'Save adapter',
  name: 'Name',
  nameHint: "Like the provider's name: “Example ESP”.",
  saved: '{name} saved',
  savedDetail: 'Connect it from Add provider in any workspace.',

  // How the provider's webhooks are authenticated (http-adapter-model).
  authHmac: 'HMAC signature in a header',
  authStandard: 'Standard Webhooks (webhook-signature)',
  authBasic: 'Basic auth',
  authHeader: 'Secret in a header',
  authNone: 'None: the webhook URL only',
  authHmacHeader: 'Signature header',
  authSecretHeader: 'Header with the secret',
  // The paths a custom JSONPath map reads.
  pathType: 'Event type',
  pathRecipient: 'Recipient',
  pathMessageId: 'Message id',
  pathEvents: 'Events in one request',
  headerLineError: 'Line {line}: write each header as Name: value.',

  // ── Build your own ───────────────────────────────────────────────────
  buildTitle: 'Build your own adapter',
  /** The values are code: the interface, its functions and the manifest's file name. */
  buildBody:
    'An adapter is a small TypeScript package implementing the {iface} interface: {send}, {normalize}, optional {setup}, {sync} and {quota}. Declare capabilities and terms in {manifest}; the catalog badges come from it.',
  readSdk: 'Read the SDK guide',
  templateRepo: 'Use the template repo',
  exampleAdapter: 'Example adapter',

  // ── Install dialog ───────────────────────────────────────────────────
  installBody: "Paste the https:// address of the adapter's manifest.json from the registry. ReputeCast shows what it declares before anything runs.",
  install: 'Install',
  manifestUrl: 'Manifest URL',
  installed: '{name} {version} installed',
  installedDetail: 'Connect it from Add provider.',
})
