import { defineMessages } from '@/i18n'

/**
 * Real renders (CO-13, ComposerPreview): the note that the built-in
 * previews are approximations, connecting the workspace's own Litmus or
 * Mailgun Inspect account, and screenshots from real email clients.
 * {service} is "Litmus" or "Mailgun Inspect".
 */
export const mr = defineMessages('composer.renders', {
  note: '<b>Approximated in your browser from the compiled HTML.</b> Real renders need a rendering service (Litmus or Mailgun Inspect).',
  noteConnected: '<b>Approximated in your browser from the compiled HTML.</b> Real renders come from your {service} account.',
  noteFailing: "<b>Approximated in your browser from the compiled HTML.</b> {service} isn't answering: {error}",
  connect: 'Connect',
  manage: 'Manage',
  whoConnects: 'People who manage integrations can connect one.',
  renderReal: 'Render in real clients',
  /** The card header of a built-in preview: "Approximated 09:40". */
  approximatedAt: 'Approximated {time}',
  approximating: 'Approximating…',

  connectTitle: 'Connect a rendering service',
  connectLead: 'Screenshots from real email clients, made with your own Litmus or Mailgun Inspect account. The API key is stored encrypted and never shown again.',
  service: 'Service',
  apiKey: 'API key',
  keyHelpLitmus: 'Your Litmus API key for the Instant API.',
  keyHelpMailgun: 'Your Mailgun private API key; Inspect must be on your plan.',
  region: 'Region',
  regionUs: 'US',
  regionEu: 'EU',
  testAndConnect: 'Test and connect',
  testing: 'Testing…',
  connectedLine: '{service} · key ending {hint} · connected by {name}, {date}',
  connectedAnon: '{service} · key ending {hint} · connected {date}',
  clientsLine: '{count, plural, =0 {No email clients listed} one {# email client available} other {# email clients available}}',
  failingLine: 'Last check failed: {error}',
  checkedOk: '{service} answered. {count, plural, one {# email client available} other {# email clients available}}.',
  testAgain: 'Test again',
  replaceKey: 'Replace the key',
  disconnect: 'Disconnect',
  disconnectBadge: 'Disconnect service',
  disconnectTitle: 'Disconnect {service}?',
  disconnectBody: 'The key is deleted from ReputeCast. Renders already made stay with their campaigns.',
  disconnecting: 'Disconnecting…',
  disconnected: '{service} disconnected',
  connectedToast: '{service} connected',

  renderTitle: 'Render in real clients',
  renderLead: 'Sends the saved version to {service}, personalized for the sample contact Sam Rivera. Each client is one preview on your {service} plan.',
  renderVariant: 'Variant {key} of the A/B test.',
  unsavedFirst: 'Your unsaved edits are saved first.',
  clients: 'Email clients',
  noClients: "{service} listed no email clients for this account. Test the connection again from Manage.",
  startRender: '{count, plural, one {Render in # client} other {Render in # clients}}',
  starting: 'Starting…',
  startFailed: "Couldn't start the render",
  saveFailed: "Your edits couldn't be saved, so nothing was sent to {service}.",

  realRenders: 'Real renders · {service}',
  renderMeta: '{variant, select, A {v{version}} other {v{version} · variant {variant}}} · {time} · {name}',
  stale: 'From v{version}; the email has changed since.',
  progress: '{done} of {count} captured…',
  capturing: 'Capturing…',
  notCaptured: "Couldn't capture",
  openFull: 'Open {client} full size',
  renderError: '{error}',
})
