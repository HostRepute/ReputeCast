import { translate } from '@/i18n'
import { m } from './approval.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  pushTitle: 'Die Benachrichtigung',
  pushCaption: '{n, plural, one {# Push-Abonnent} other {# Push-Abonnenten}} auf {sites} · {schedule}',
  link: 'Link',
  buttons: 'Buttons',
  expiry: 'Läuft ab nach',
  hours: '{n, plural, one {# Stunde} other {# Stunden}}',
  minutes: '{n, plural, one {# Minute} other {# Minuten}}',
  tag: 'Tag',
  none: '–',
  waTitle: 'Die WhatsApp-Nachricht',
  waCaption: '{template} · {category} · {language} · {n, plural, one {# Person} other {# Personen}} · {schedule}',
  business: 'Gesendet als',
  variables: 'Variablen',
  varField: '\'{{\'{key}\'}}\' aus {field}',
  varFieldFallback: '\'{{\'{key}\'}}\' aus {field}, sonst „{fallback}“',
  varText: '\'{{\'{key}\'}}\' {text}',
  cost: 'Geschätzte Kosten',
  costLine: '{amount}, von Meta bei Zustellung berechnet',
})
