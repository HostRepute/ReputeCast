import { translate } from '@/i18n'
import { mpi } from './entries.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mpi, {
  stripe: 'Stripe',
  paddle: 'Paddle',
  lemonsqueezy: 'Lemon Squeezy',
  byOwnStripe: 'Ihr eigenes Stripe-Konto',
  byMerchant: 'Merchant of Record',
  stripeLine: 'Bezahlabos und Einmalzahlungen, mit einem Kundenportal für Ihre Leserschaft.',
  merchantLine: 'Ein Merchant of Record: Er verkauft in Ihrem Auftrag an Ihre Leserschaft und kümmert sich um Sales Tax und Umsatzsteuer.',
  capSubscriptions: 'Abos',
  capOneTime: 'Einmalzahlungen',
  capPortal: 'Kundenportal',
  capTestMode: 'Testmodus',
  capTax: 'Steuerberechnung',
  factRestricted: 'Ein eingeschränkter Schlüssel, nie ein geheimer',
  factConnected: '{account} · {mode, select, live {Live} other {Test}}',
  factNotInRelease: 'Nicht in dieser Version',
  notAvailable: 'Noch nicht verfügbar',
  connect: 'Verbinden',
  manage: 'Verwalten',
  view: 'Ansehen',
  importHint: 'Sie ziehen mit zahlenden Mitgliedern von Substack, Ghost, beehiiv oder Kit um? <a>Importieren Sie sie unter Einnahmen</a>, abgeglichen mit Ihren Stripe-Abos.',
})
