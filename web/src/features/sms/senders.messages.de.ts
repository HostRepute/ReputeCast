import { translate } from '@/i18n'
import { m } from './senders.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'Kurzlink-Domain',
  sub: 'Jeder Link in einer SMS wird hier gekürzt, mit einem Code je Empfänger. Öffentliche URL-Kürzer wie bit.ly sind blockiert: Netzbetreiber filtern sie.',
  caption: 'Kurzlink-Domain',
  domain: 'Domain',
  usedBy: 'Genutzt von',
  smsLinks: 'Links in SMS',
  record: 'Eintrag',
  status: 'Status',
  lastCheck: 'Letzte Prüfung',
  cname: 'CNAME {name} → {target}',
  verifiedOn: 'Verifiziert am {date}',
  notVerified: 'Nicht verifiziert',
  checkFailed: 'Prüfung fehlgeschlagen',
  never: 'Noch nicht geprüft',
  checkDns: 'DNS prüfen',
  checked: 'DNS geprüft',
  manage: 'Routing & Regeln',
  manageLabel: 'Kurzlink-Domain unter Routing & Regeln ändern',
  none: 'Noch keine Kurzlink-Domain. SMS mit Links können erst gesendet werden, wenn eine verifiziert ist.',
  setUp: 'Domain einrichten',
  untilVerified: 'Bis zur Verifizierung können SMS mit Links nicht gesendet werden. SMS ohne Links schon.',
})
