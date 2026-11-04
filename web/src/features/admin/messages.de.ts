import { translate } from '@/i18n'
import { m } from './messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'Instanzverwaltung',
  documentTitle: '{page} · Instanzverwaltung',
  adminsOnly: 'nur für Admins',
  workspaceView: 'Arbeitsbereichsansicht',
  workspaceSettings: 'Einstellungen des Arbeitsbereichs',
  tabsLabel: 'Bereiche der Instanzverwaltung',
  notAdminBadge: '403 · nur für Instanz-Admins',
  notAdminTitle: 'Nur Instanz-Admins können die Instanzverwaltung öffnen',
  backToOverview: 'Zurück zur Übersicht',
  account: 'Konto & Sicherheit',
  notAdminFootnote: 'Wenden Sie sich für Änderungen an Anmeldung, SSO oder Arbeitsbereichen an einen Instanz-Admin.',
  notAdminBody: 'In der Instanzverwaltung werden Konten, Anmeldemethoden, SSO, Sicherheitsrichtlinien und Arbeitsbereiche der gesamten Instanz verwaltet. Ihre eigenen Anmeldemethoden finden Sie unter Konto & Sicherheit.',
})
