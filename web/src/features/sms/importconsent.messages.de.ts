import { translate } from '@/i18n'
import { m } from './importconsent.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'SMS-Einwilligung',
  withEvidence: 'Wird mit Nachweis übernommen',
  withEvidenceText: 'Zeilen mit „Ja“, einem Datum, einer Quelle und dem Einwilligungstext werden auf der Zielliste für SMS als „Abonniert“ übernommen und als Import mit diesem Nachweis erfasst. Niemand erhält eine SMS.',
  missing: 'Die SMS-Einwilligung braucht ihren Nachweis: Ordnen Sie unter „Felder zuordnen“ {count, plural, one {die Spalte} other {die Spalten}} {missing} zu. Bis dahin werden Nummern übernommen und die SMS-Einwilligung bleibt „Nicht abonniert“.',
  numbersOnly: 'Nummern werden ohne SMS-Einwilligung übernommen: Für SMS bleiben sie „Nicht abonniert“. Um die SMS-Einwilligung mitzunehmen, ordnen Sie unter „Felder zuordnen“ eine Einwilligungsspalte samt Datum, Quelle und Text zu.',
  date: 'Datum',
  source: 'Quelle',
  text: 'Text',
  mapFields: 'Felder zuordnen',
})
