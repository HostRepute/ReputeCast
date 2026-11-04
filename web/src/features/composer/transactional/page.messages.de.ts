import { translate } from '@/i18n'
import { mt } from './page.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mt, {
  documentTitle: '{name} · Transaktionsvorlage · ReputeCast',
  documentTitleLoading: 'Transaktionsvorlage · ReputeCast',
  newTitle: 'Neue Transaktionsvorlage · ReputeCast',
  opening: 'Vorlage wird geöffnet',
  openingDetail: 'Inhalt, Variablen und Testdaten · meist unter einer Sekunde',
  creating: 'Entwurf wird erstellt',
  creatingDetail: 'Neue Vorlage · nichts wird gesendet, bevor sie veröffentlicht ist',
  untitled: 'Unbenannte Vorlage',
  backToTemplates: 'Zurück zu den Vorlagen',
  goToTemplates: 'Zu den Vorlagen',
  notFoundBody: 'Die Vorlage wurde möglicherweise gelöscht, oder der Link wurde kopiert, bevor sie gespeichert war. Gelöschte Vorlagen bleiben im Audit-Protokoll.',
  cantOpen: 'Ihre Rolle darf diese Vorlage nicht öffnen',
  cantCreate: 'Ihre Rolle darf keine Transaktionsvorlagen erstellen',
  titleNotFound: 'Nicht gefunden · ReputeCast',
  titleForbidden: 'Kein Zugriff · ReputeCast',
  titleBroken: 'Etwas ist schiefgelaufen · ReputeCast',
})
