import { defineMessages } from '@/i18n'

/*
 * Transactional mail in other screens: the contact profile's timeline, the
 * workspace Overview's card and Instance admin's SMTP relay.
 */

/** A contact's transactional mail in their timeline, and their Optional categories (ContactProfile). */
export const ma = defineMessages('transactional.contact', {
  row: '<b>{status}</b> <link>{name}</link> <badge>Transactional</badge>',
  rowVia: '<b>{status}</b> <link>{name}</link> <badge>Transactional</badge> via <badge>{provider}</badge>',
  afterCall: '{time} after the call',
  contentVisible: 'your role can view its content',
  contentHidden: "your role can't view its content",
  notifications: 'Notifications from {product}',
  categoryOn: '{name} on',
  categoryOff: '{name} off',
  notificationsLine: 'Optional categories the contact can turn off in the preference center · {essential} emails always arrive',
  notificationsLineNoEssential: 'Optional categories the contact can turn off in the preference center',
})

/** The workspace Overview's Transactional card (Main board). */
export const mw = defineMessages('transactional.overviewCard', {
  title: 'Transactional · last 7 days',
  label: 'Transactional, last 7 days',
  line: "{product}'s product mail · {route}",
  route: '{primary}, then {fallback}',
  noRoute: 'no route yet',
  sent: 'Sent',
  delivered: 'Delivered',
  median: 'Median delivery',
  rejected: 'Rejected',
  noAlerts: 'No active alerts',
  alerts: '{count, plural, one {# active alert} other {# active alerts}}',
  failover: 'Failing over to {provider}',
  open: 'Open Transactional',
})

/** Instance admin › Instance & updates: the SMTP relay section (InstanceSettings). */
export const mr = defineMessages('transactional.relayAdmin', {
  title: 'SMTP relay',
  caption: 'For apps that send transactional mail by SMTP instead of the HTTP API',
  turnedOn: '{caption} · turned on by {name} on {date}',
  turnedOnNoName: '{caption} · turned on on {date}',
  on: 'On',
  off: 'Off',
  unreachable: 'Unreachable',
  hostname: 'Hostname',
  ports: 'Ports',
  starttls: 'STARTTLS',
  tls: 'TLS',
  starttlsPort: 'STARTTLS port',
  tlsPort: 'TLS port',
  certificate: 'Certificate',
  acme: "Let's Encrypt",
  acmeRenews: "Let's Encrypt · renews {date}",
  files: 'Certificate and key files',
  certFile: 'Certificate chain (PEM)',
  keyFile: 'Private key (PEM)',
  keyHint: 'Write-only: the key is sealed and never shown again. Leave both empty to keep the files in use.',
  help: "Workspaces create SMTP credentials in Transactional › Setup; each one is a send-only key. Mail that arrives here goes through the workspace's Transactional route.",
  offBody: 'Turn the relay on to let apps that only speak SMTP send through a workspace’s Transactional route. ReputeCast listens on the ports below and checks DNS, the ports and the certificate.',
  checks: 'Checks',
  checkedAt: 'Checked {date}',
  notChecked: 'Not checked yet: run the checks once DNS points here.',
  pass: 'Pass',
  fail: 'Fail',
  runChecks: 'Run checks',
  turnOn: 'Turn on',
  turnOff: 'Turn off',
  save: 'Save',
  saved: 'SMTP relay saved',
  turnedOnToast: 'SMTP relay on',
  turnedOffToast: 'SMTP relay off',
  turnedOffDetail: 'Apps sending by SMTP get a refused connection; the HTTP API keeps working.',
  renewFailed: "The certificate didn't renew: {error}",
  error: "Couldn't load the SMTP relay",
  retry: 'Retry',
})
