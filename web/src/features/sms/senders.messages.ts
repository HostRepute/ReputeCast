import { defineMessages } from '@/i18n'

/** Settings › Senders: the short-link domain, with its record and status (brief §6). */
export const m = defineMessages('smsShortLink', {
  title: 'Short-link domain',
  sub: 'Every link in a text is shortened here, with a code per recipient. Public shorteners such as bit.ly are blocked: carriers filter them.',
  caption: 'Short-link domain',
  domain: 'Domain',
  usedBy: 'Used by',
  smsLinks: 'Links in texts',
  record: 'Record',
  status: 'Status',
  lastCheck: 'Last check',
  cname: 'CNAME {name} → {target}',
  verifiedOn: 'Verified {date}',
  notVerified: 'Not verified',
  checkFailed: 'Check failed',
  never: 'Not checked yet',
  checkDns: 'Check DNS',
  checked: 'DNS checked',
  manage: 'Routing & rules',
  manageLabel: 'Change the short-link domain in Routing & rules',
  none: 'No short-link domain yet. Texts with links can’t be sent until one is verified.',
  setUp: 'Set one up',
  untilVerified: 'Until it’s verified, texts with links can’t be sent. Texts without links can.',
})
