import { translate } from '@/i18n'
import { m } from './notifuse.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  sourceWhat: 'API-Schlüssel · Kontakte, Listen, Vorlagen',
  templatesOption: 'Vorlagen importieren',
  templatesDetail: '{transactional, plural, =0 {keine Transaktionsvorlagen} one {# Transaktionsvorlage kommt als Entwurf an} other {# Transaktionsvorlagen kommen als Entwürfe an}}, {saved, plural, =0 {keine weiteren} one {# weitere kommt zu den gespeicherten Vorlagen} other {# weitere kommen zu den gespeicherten Vorlagen}}',
  templatesSkipped: '{skipped, plural, one {Die Double-Opt-in- oder Abmelde-E-Mail wird nicht übernommen: ReputeCast sendet eigene.} other {Die # Double-Opt-in- und Abmelde-E-Mails werden nicht übernommen: ReputeCast sendet eigene.}}',
  doneTemplates: '{transactional, plural, one {# Transaktionsvorlage kam} other {# Transaktionsvorlagen kamen}} als Entwurf an, und {saved, plural, one {# Vorlage wurde} other {# Vorlagen wurden}} zu den gespeicherten Vorlagen hinzugefügt.',
  doneTemplatesManual: '{manual, plural, one {# davon braucht} other {# davon brauchen}} vor der Verwendung eine manuelle Korrektur.',
  doneTemplatesLink: 'Transaktionsvorlagen öffnen',
})
