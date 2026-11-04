import { translate } from '@/i18n'
import { mf } from './followup.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mf, {
  menu: 'Follow-up',
  nonOpeners: 'Erneut an Kontakte ohne Öffnung senden…',
  clickers: 'An Kontakte mit Klick senden…',
  titleNonOpeners: 'Erneut an Personen senden, die nicht geöffnet haben',
  titleClickers: 'An Personen senden, die geklickt haben',
  bodyNonOpeners: '{count, plural, one {<b># Kontakt</b> hat „{name}“ erhalten und nicht geöffnet.} other {<b># Kontakte</b> haben „{name}“ erhalten und nicht geöffnet.}}',
  machineOnly: '{count, plural, one {# Kontakt hat nur über einen Datenschutz-Proxy wie Apple Mail Privacy Protection geöffnet: Er gilt als unbekannt und wird ausgelassen.} other {# Kontakte haben nur über einen Datenschutz-Proxy wie Apple Mail Privacy Protection geöffnet: Diese gelten als unbekannt und werden ausgelassen.}}',
  bodyClickers: '{count, plural, one {<b># Person</b> hat auf einen Link in „{name}“ geklickt.} other {<b># Personen</b> haben auf einen Link in „{name}“ geklickt.}} Klicks von Link-Scannern zählen nicht.',
  nobody: 'Noch ist niemand in dieser Gruppe, das Follow-up würde also vorerst niemanden erreichen. Die Zielgruppe bleibt bis zum Versand aktuell.',
  who: 'Senden an',
  whoNonOpeners: 'Personen, die nicht geöffnet haben',
  whoClickers: 'Personen, die geklickt haben',
  subject: 'Neuer Betreff',
  subjectNonOpeners: 'Falls Sie es verpasst haben: {subject}',
  subjectClickers: 'Mehr zu: {subject}',
  send: 'Senden',
  after: '{days, plural, one {# Tag nach dem Original ({when})} other {# Tage nach dem Original ({when})}}',
  later: 'Unter Prüfen & senden wählen',
  note: 'Erstellt einen Entwurf mit demselben Inhalt. Sperrliste, Abmeldungen und das Marketing-Limit gelten wie gewohnt.',
  create: 'Follow-up erstellen',
  creating: 'Wird erstellt…',
  created: 'Follow-up „{name}“ erstellt',
  loading: 'Reichweite wird gezählt…',
  subjectRequired: 'Geben Sie dem Follow-up einen Betreff.',
})
