import { translate } from '@/i18n'
import { mcard } from './card.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mcard, {
  title: 'Einnahmen · letzte 30 Tage',
  label: 'Einnahmen, letzte 30 Tage',
  line: '{provider} · {account} · {mode, select, test {Test} other {Live}}',
  lineNoAccount: '{provider} · {mode, select, test {Test} other {Live}}',
  mrr: 'MRR',
  members: 'Zahlende Mitglieder',
  churn: 'Kündigungsrate',
  trials: 'Testphasen',
  nextSponsor: 'Nächster Sponsor',
  nextIssue: '#{issue} · {sponsor}',
  creativeDue: 'Werbemittel fällig am {date}',
  creativeReady: 'Werbemittel bereit',
  open: 'Einnahmen öffnen',
  silentTitle: 'Webhook von {provider} stumm',
  silentText: 'Seit {hours, plural, one {# Stunde} other {# Stunden}} kein Ereignis von {provider}, während {active, plural, one {# Abo aktiv ist} other {# Abos aktiv sind}}. Die Status hinken höchstens eine Stunde hinterher.',
  checkWebhook: 'Webhook prüfen',
  inviteLabel: 'Einnahmen',
  inviteTitle: 'Einnahmen mit Ihrem Newsletter',
  inviteText: 'Bezahlabos, Geschenke und Sponsoring. Ihre Leserschaft bezahlt Sie über Ihr eigenes Stripe-Konto; ReputeCast nimmt keine Gebühr.',
  setUp: 'Einnahmen einrichten',
  notNow: 'Nicht jetzt',
})
