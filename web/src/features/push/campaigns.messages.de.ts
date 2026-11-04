import { translate } from '@/i18n'
import { m } from './campaigns.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  newPush: 'Push-Benachrichtigung',
  newPushBody: 'Eine Benachrichtigung an die Browser Ihrer Leserschaft auf Ihren Push-Websites.',
  newPushHint: 'Titel, Nachricht und Link, mit einer Vorschau je Plattform.',
  newWhatsApp: 'WhatsApp-Nachricht',
  newWhatsAppBody: 'Eine genehmigte Vorlage an Personen mit WhatsApp-Einwilligung.',
  newWhatsAppHint: 'Variablen aus Kontaktfeldern; die Kosten werden vor dem Senden angezeigt.',
  untitledPush: 'Unbenannte Push-Kampagne',
  untitledWhatsApp: 'Unbenannte WhatsApp-Kampagne',
  noOpens: 'Benachrichtigungen und WhatsApp-Nachrichten haben keine Öffnungen',
  shownTitle: 'Angezeigt, laut Service Worker',
  local: '{date} · {time} in der Zeitzone jedes Empfängers',
  notSite: 'Keine Website',
  siteSubscribers: 'Push-Abonnenten der Website',
  duplicated: 'Dupliziert als „{name}“',
  duplicateFailed: 'Duplizieren fehlgeschlagen',
})
