import { defineMessages } from '@/i18n'

/**
 * Deliverability › DMARC reports: the summary per sending source, the
 * "Investigate source" dialog, the empty card and the report upload.
 * Results (aligned, pass, fail, none, quarantined) and policies
 * (p=quarantine) stay as the reports write them.
 */
export const md = defineMessages('deliverability.dmarc', {
  title: 'DMARC reports · last {days} days',
  captionRua: 'aggregate reports to {rua}',
  captionUploaded: '{count, plural, one {# uploaded aggregate report} other {# uploaded aggregate reports}}',
  /** {rate} is the share of messages that passed DMARC: "99.6%". */
  passBadge: '{rate} pass',
  tableCaption: 'DMARC results by sending source',
  colSource: 'Source',
  colMessages: 'Messages',
  colResult: 'Result',
  notYourProvider: 'not one of your providers',
  unknownEnforced: 'Someone is sending as {domain} from an unknown host; p={policy} kept it out of inboxes.',
  unknownNotEnforced: 'Someone is sending as {domain} from an unknown host; with p={policy}, receivers may still deliver it.',
  investigate: 'Investigate source',

  // ── Investigate source ───────────────────────────────────────────────
  sourceTitle: 'Sending source {ip}',
  sourceKnown: '{provider} sends from this IP.',
  sourceUnknown: "This IP isn't one of your providers. It sent mail as {domain} in the last 7 days.",
  /** After the server's error message. */
  nothingChanged: 'Nothing was changed.',
  messages7d: 'Messages, 7 d',
  /** {messages} is a formatted number of messages. */
  messagesInReports: '{messages} in {reports, plural, one {# report} other {# reports}}',
  seen: 'Seen',
  seenRange: '{from} to {to}',
  reportedBy: 'Reported by',
  reverseDns: 'Reverse DNS',
  none: 'none',
  fromDomains: 'From domains',
  receiversDid: 'What receivers did',
  dispositionNone: 'delivered (no DMARC action)',
  dispositionQuarantine: 'quarantined (spam folder)',
  dispositionReject: 'rejected',
  /**
   * {hasPolicy}: yes when the domain has a DMARC policy, written p={policy};
   * {delivered}: yes when receivers still delivered the mail.
   */
  unknownAdvice:
    "If this is a service you use (a CRM, a help desk, an invoicing tool), add it to the SPF record of {domain} and turn on DKIM signing there. If it isn't, nothing to do: {hasPolicy, select, yes {p={policy}} other {your DMARC policy}} {delivered, select, yes {should be raised to quarantine so receivers stop delivering it} other {keeps it out of inboxes}}.",

  // ── Without reports ──────────────────────────────────────────────────
  emptyTitle: 'DMARC report summary',
  noReports: 'No reports yet',
  emptyRua: 'Aggregate reports need a mailbox to land in. Point <mono>rua=</mono> at <mono>{rua}</mono> and HostRepute parses them for you, or paste the XML reports here.',
  emptyNoRua: 'Aggregate reports need a mailbox to land in. Point <mono>rua=</mono> at HostRepute (parsed for you) or at any mailbox you own and paste the XML reports here.',
  useHostRepute: 'Use HostRepute as rua',

  // ── Upload ───────────────────────────────────────────────────────────
  uploadButton: 'Upload report XML',
  chooseFile: 'Choose a report file first.',
  uploaded: 'Report from {org} added',
  /** {messages} is a formatted number of messages. */
  uploadedDetail: '{messages} messages for {domain}',
  uploadTitle: 'Upload a DMARC aggregate report',
  uploadDescription: "The .xml file a mailbox provider sent to your rua= address, or the .xml.gz or .zip it came in. Only the report's counts are stored.",
  upload: 'Upload',
  reportFile: 'Report file',
})
