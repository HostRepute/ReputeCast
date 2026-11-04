import { translate } from '@/i18n'
import { m } from './role.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  eyebrow: 'Dialog · Rolle oder Bereich ändern',
  title: 'Rolle oder Bereich für {name} ändern',
  description: 'Änderungen gelten sofort, bei der nächsten Anfrage von {name}. Alter und neuer Wert werden im Audit-Protokoll erfasst.',
  nothingChanged: 'Noch nichts geändert.',
  current: '{role} (aktuell)',
  updated: '{name} aktualisiert',
  scopeChanged: 'Bereich geändert',
  nowRole: 'jetzt {role}',
  ssoGroup: 'Die Rolle von {name} stammt aus einer SSO-Gruppenzuordnung. Eine Änderung hier gilt bis zur nächsten Anmeldung, die die Zuordnung erneut anwendet; ändern Sie die Zuordnung, damit sie dauerhaft gilt.',
  allScope: 'Alle Listen und Absender-Domains',
  lastOwnerEyebrow: 'Dialog · Rolle ändern · blockiert',
  lastOwnerTitle: '{name} ist der letzte Inhaber',
  onlyOwnerTransfers: 'Nur Inhaber können die Inhaberschaft übertragen.',
  transfer: 'Inhaberschaft übertragen',
  cantDemote: 'Herabstufen nicht möglich',
  keepsOwner: 'Ein Arbeitsbereich hat immer mindestens einen Inhaber. Daher kann diese Rolle nicht in Admin geändert und das Mitglied weder entfernt noch deaktiviert werden.',
  transferFirst: 'Übertragen Sie dazu zuerst die Inhaberschaft an ein anderes Mitglied mit aktivierter Zwei-Faktor-Authentifizierung. {name} wird dann automatisch Admin.',
  transferFirstSuggest: 'Übertragen Sie dazu zuerst die Inhaberschaft an ein anderes Mitglied mit aktivierter Zwei-Faktor-Authentifizierung, zum Beispiel {suggest}. {name} wird dann automatisch Admin.',
})
