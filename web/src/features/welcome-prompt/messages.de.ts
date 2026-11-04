import { translate } from '@/i18n'
import { m } from './messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  label: 'Vorschlag Willkommensserie',
  title: 'Neue Abonnenten von {list} erhalten noch keine Willkommens-E-Mail',
  body: 'Die Startvorlage Willkommensserie sendet in der ersten Woche drei E-Mails.',
  covered: '{count, plural, one {{lists} hat bereits eine.} other {{lists} haben bereits eine.}}',
  start: 'Willkommensserie starten',
  notNow: 'Nicht jetzt',
  dismissed: 'Wir schlagen für {list} keine Willkommensserie mehr vor',
  dismissedDetail: 'Starten Sie jederzeit eine unter Automationen › Startvorlagen.',
  dismissFailed: 'Der Vorschlag konnte nicht ausgeblendet werden',
})
