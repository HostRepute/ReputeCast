import { translate } from '@/i18n'
import { m } from './messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'Zielgruppe',
  sections: 'Bereiche der Zielgruppe',
  tabContacts: 'Kontakte',
  tabContactsCount: 'Kontakte · {count}',
  tabSegments: 'Segmente',
  tabSegmentsCount: 'Segmente · {count}',
  tabHealth: 'Listenqualität',
  tabSuppressions: 'Sperrliste',
  tabSuppressionsCount: 'Sperrliste · {count}',
  tierActive: 'Aktiv',
  tierCooling: 'Nachlassend',
  tierAtRisk: 'Gefährdet',
  tierSunset: 'Auslaufend',
  windowActive: 'in 30 Tagen geöffnet',
  windowCooling: '31 bis 90 Tage',
  windowAtRisk: '91 bis 180 Tage',
  windowSunset: 'keine Öffnung in 180 Tagen',
  consentDouble: 'Double-Opt-in',
  consentSingle: 'Single-Opt-in',
  consentPending: 'Importiert · ausstehend',
  consentPendingFilter: 'Importiert, ausstehend',
  mailboxOther: 'Andere',
})
