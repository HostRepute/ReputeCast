import { translate } from '@/i18n'
import { m } from './approval.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  ruleTitle: 'Auch das Veröffentlichen einer Transaktionsvorlage erfordert eine Freigabe',
  ruleDetail: 'Gilt für alle, die Transaktionsvorlagen veröffentlichen können. Aus: Das Veröffentlichen wird nur protokolliert, und die API sendet die neue Version sofort.',
  cardTitle: 'Transaktionsvorlage',
  cardCaption: 'Der Entwurf, den der Autor mit der Freigabe veröffentlichen darf',
  version: 'Version',
  versionValue: 'v{version} von {key}',
  versionReplaces: 'v{version} von {key}, ersetzt live v{live}',
  category: 'Kategorie',
  subject: 'Betreff',
  variables: 'Variablen',
  content: 'Inhalt',
  noContentChange: 'Keine inhaltlichen Änderungen',
  breaking: 'Inkompatible Änderung',
  testRenders: 'Test-Renderings',
  renders: 'Rendert',
  fails: 'Schlägt fehl',
  checksLine: '{passed, plural, one {# Prüfung bestanden} other {# Prüfungen bestanden}}{warnings, plural, =0 {} one {, # Warnung} other {, # Warnungen}}',
  published: 'Als v{version} veröffentlicht',
  decisionNote: 'Die Freigabe gilt genau für diesen Entwurf. Bearbeitet {name} ihn vor dem Veröffentlichen, braucht die Veröffentlichung eine neue Freigabe.',
  rejectDetail: '{name} wird informiert, und der Entwurf wird nicht veröffentlicht. Die API sendet weiter die Live-Version. Lehnen Sie ab, wenn der Entwurf gar nicht live gehen soll; für Korrekturen fordern Sie stattdessen Änderungen an.',
  changesDetail: 'Die Anfrage geht mit Ihrem Kommentar an {name} zurück. Der Entwurf bleibt unveröffentlicht, bis erneut angefragt wird.',
})
