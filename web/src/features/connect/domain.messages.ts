import { defineMessages } from '@/i18n'

/**
 * 7.2 step 3 (SetupDomain) and a domain's records (DomainRecords), shown in
 * the wizard and in Settings › Senders. Record labels, notes, values and
 * status words come from the server; the words here are their fallbacks.
 */
export const md = defineMessages('connect.domain', {
  // ── The step ─────────────────────────────────────────────────────────
  setupFailed: "Couldn't set up {domain}",
  skipFailed: "Couldn't skip the domain step",
  sendingDomain: 'Sending domain',
  fetching: 'Fetching…',
  useDomain: 'Use {domain}',
  enterDomain: 'Enter the domain you send from.',
  showRecords: 'Show the DNS records',
  headingChosen: 'Add these DNS records to {domain}',
  headingChoose: 'Which domain will you send from?',
  /** {provider} is the provider's name: "Amazon SES". */
  leadChosen: 'These are the exact records {provider} needs for this domain. Status updates live; most registrars propagate within minutes, some take up to 48 hours.',
  leadChoose: '{provider} signs mail for the domain in your From address. ReputeCast asks {provider} for the records that domain needs and checks them as they appear.',
  loadFailed: "Couldn't load the records",
  loading: 'Loading the records…',
  skip: 'Skip for now · test send still works from a verified address',
  chooseFirst: 'Choose the sending domain first, or skip for now.',
  continueTest: 'Continue to test send',

  // ── Tools above the records ──────────────────────────────────────────
  /** {host} is the DNS host the domain's nameservers belong to: "Cloudflare". */
  nameservers: 'Nameservers: {host}',
  addViaCloudflare: 'Add all records via Cloudflare API',
  copyZone: 'Copy as zone file',
  allVerified: 'All records verified · last check {time}',
  checking: 'Checking every 30 s · last check {time}',

  // ── The records ──────────────────────────────────────────────────────
  recordsTitle: 'Records · {verified} of {total} verified',
  verifiedCount: '{count} verified',
  pendingCount: '{count} pending',
  missingCount: '{count} missing',
  caption: 'DNS records for {domain}',
  colPurpose: 'Purpose',
  colType: 'Type',
  colName: 'Name',
  colValue: 'Value',
  colStatus: 'Status',
  colCopy: 'Copy',
  /** The value the DNS has now, in the mono face. */
  found: 'Found: <mono>{value}</mono>',
  /** {record} is the record's label: "DKIM 1 of 3", "SPF". */
  copyValueLabel: 'Copy the {record} value',
  /** {tip} is the server's advice. */
  dmarcTip: '<b>DMARC tip:</b> {tip}',
  statusVerified: 'Verified',
  statusPending: 'Pending',
  statusMissing: 'Missing',
  statusUnknown: "Couldn't check",

  // ── BIMI and the tracking domain ─────────────────────────────────────
  addTrackingFailed: "Couldn't add the tracking domain",
  optionalTitle: 'Optional',
  bimiPublished: 'Published: your logo shows in Gmail, Yahoo and Apple Mail where the receiver supports it.',
  bimiInfo: 'Shows your logo in Gmail, Yahoo and Apple Mail. Needs DMARC at quarantine or reject and an SVG Tiny PS logo; a Verified Mark Certificate is optional for Gmail.',
  trackingTitle: 'Custom tracking domain',
  /** {domain} is the tracking domain and {target} the host its CNAME points to. */
  trackingLine:
    '{state, select, verified {<mono>{domain}</mono> CNAME to <mono>{target}</mono>: verified, links and the open pixel use your domain.} pending {<mono>{domain}</mono> CNAME to <mono>{target}</mono>: waiting for DNS.} wrong_value {<mono>{domain}</mono> CNAME to <mono>{target}</mono>: points somewhere else.} missing {<mono>{domain}</mono> CNAME to <mono>{target}</mono>: add the CNAME at your DNS host.} suggested {<mono>{domain}</mono> CNAME to <mono>{target}</mono>, so links and the open pixel use your domain.} other {<mono>{domain}</mono> CNAME to <mono>{target}</mono>.}}',
  addTracking: 'Add tracking domain',

  // ── Cloudflare ───────────────────────────────────────────────────────
  cfAdded: '{count, plural, one {Added # record in Cloudflare} other {Added # records in Cloudflare}}',
  cfNothing: 'Cloudflare already had every record',
  /** {records} lists the records Cloudflare refused. */
  cfFailedSome: "Couldn't add: {records}. Add them by hand.",
  cfChecks: 'ReputeCast checks them again every 30 seconds.',
  cfTitle: 'Add the records via the Cloudflare API',
  cfDescription: "ReputeCast adds the missing records to the {domain} zone and fixes wrong ones. Existing records it doesn't need stay as they are.",
  adding: 'Adding…',
  pasteToken: 'Paste a Cloudflare API token first.',
  addRecords: 'Add records',
  tokenLabel: 'Cloudflare API token',
  /** {permission} is Cloudflare's name for it: "Zone › DNS › Edit". */
  tokenHint: 'Needs {permission} on this zone. Used once and never stored.',
})
