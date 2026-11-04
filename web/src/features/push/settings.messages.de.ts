import { translate } from '@/i18n'
import { m } from './settings.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  pushTitle: 'Web-Push',
  pushBuiltIn: 'Integriert · keine Anmeldung bei einem Anbieter nötig',
  keys: 'Schlüssel',
  keysCreated: 'Erstellt am {date} · {name}',
  sites: 'Websites',
  noSites: 'Noch keine Website',
  subscribers: 'Push-Abonnenten',
  subscribersLine: '{total} · {linked} mit Kontakten verknüpft',
  cap: 'Häufigkeitslimit',
  capLine: '{n, plural, one {# Benachrichtigung} other {# Benachrichtigungen}} pro Tag und Push-Abonnent',
  pushFoot: 'Schlüssel, Websites, Abfragen, Ruhezeiten und das Häufigkeitslimit finden Sie unter Versandanbieter › Push.',
  pushOpen: 'Push-Einrichtung öffnen',
  waTitle: 'WhatsApp',
  waProvider: 'WhatsApp Cloud API von Meta',
  business: 'Unternehmen',
  businessLine: '{name} · {number}',
  quality: 'Qualitätsbewertung',
  limit: 'Nachrichtenlimit',
  limitLine: '{n, plural, one {# Person} other {# Personen}} pro Tag',
  templates: 'Vorlagen',
  templatesLine: '{approved} genehmigt · {pending} ausstehend · {rejected} abgelehnt',
  consent: 'WhatsApp-Einwilligung',
  consentLine: '{n, plural, one {# Person} other {# Personen}}',
  waFoot: 'Konto, Vorlagen, Preisliste und Ereignisse finden Sie unter Versandanbieter › WhatsApp.',
  waOpen: 'WhatsApp öffnen',
})
