import { translate } from '@/i18n'
import { mt } from './templates.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mt, {
  search: 'Vorlagen und Abschnitte durchsuchen',
  found: '{count, plural, =0 {Nichts gefunden} one {# gefunden} other {# gefunden}}',
  moreFor: 'Weitere Aktionen für {name}',
  rename: 'Umbenennen…',
  replace: 'Durch diese E-Mail ersetzen',
  delete: 'Löschen…',
  renameTitle: '„{name}“ umbenennen',
  name: 'Name',
  description: 'Beschreibung (optional)',
  renamed: 'In „{name}“ umbenannt',
  replaceTitle: '„{name}“ durch diese E-Mail ersetzen?',
  replaceBody: 'Neue E-Mails daraus beginnen mit der jetzt geöffneten E-Mail. Bereits daraus erstellte E-Mails ändern sich nicht.',
  replaceBadge: 'Vorlage ersetzen',
  replaceConfirm: 'Vorlage ersetzen',
  replacing: 'Wird ersetzt…',
  replaced: '„{name}“ enthält jetzt diese E-Mail',
  deleteTitle: '„{name}“ löschen?',
  deleteTemplateBody: 'Die Vorlage verschwindet aus der Galerie. Bereits daraus erstellte E-Mails ändern sich nicht.',
  deleteSectionBody: 'Der Abschnitt verschwindet aus den gespeicherten Abschnitten. E-Mails, die ihn bereits nutzen, ändern sich nicht.',
  deleteBadge: 'Löschen',
  deleteConfirm: 'Löschen',
  deleting: 'Wird gelöscht…',
  deleted: '„{name}“ gelöscht',
})
