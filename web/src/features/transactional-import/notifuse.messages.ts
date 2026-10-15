import { defineMessages } from '@/i18n'

/** Notifuse in the contact importer (MigrationSource, MigrationDone). */
export const m = defineMessages('imports.notifuse', {
  sourceWhat: 'API key · contacts, lists, templates',
  templatesOption: 'Import templates',
  templatesDetail:
    '{transactional, plural, =0 {no transactional templates} one {# transactional template arrives as a draft} other {# transactional templates arrive as drafts}}, {saved, plural, =0 {no others} one {# other goes to saved templates} other {# others go to saved templates}}',
  templatesSkipped:
    '{skipped, plural, one {Its double opt-in or unsubscribe email stays behind: ReputeCast sends its own.} other {Its # double opt-in and unsubscribe emails stay behind: ReputeCast sends its own.}}',
  doneTemplates:
    '{transactional, plural, one {# transactional template} other {# transactional templates}} arrived as drafts and {saved, plural, one {# template was} other {# templates were}} added to saved templates.',
  doneTemplatesManual: '{manual, plural, one {# of them needs a manual fix} other {# of them need a manual fix}} before it can be used.',
  doneTemplatesLink: 'Open Transactional templates',
})
