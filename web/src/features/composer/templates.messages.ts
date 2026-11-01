import { defineMessages } from '@/i18n'

/** Saved templates and sections in the composer: search, rename, replace, delete (review 2026-11 CA-8; CampaignsComplete board, strip 5). */
export const mt = defineMessages('composer.templates', {
  search: 'Search templates and sections',
  found: '{count, plural, =0 {Nothing found} one {# found} other {# found}}',
  moreFor: 'More actions for {name}',
  rename: 'Rename…',
  replace: 'Replace with this email',
  delete: 'Delete…',
  renameTitle: 'Rename "{name}"',
  name: 'Name',
  description: 'Description (optional)',
  renamed: 'Renamed to "{name}"',
  replaceTitle: 'Replace "{name}" with this email?',
  replaceBody: "New emails made from it start from the email open now. Emails already made from it don't change.",
  replaceBadge: 'Replace template',
  replaceConfirm: 'Replace template',
  replacing: 'Replacing…',
  replaced: '"{name}" now holds this email',
  deleteTitle: 'Delete "{name}"?',
  deleteTemplateBody: "It leaves the gallery. Emails already made from it don't change.",
  deleteSectionBody: "It leaves the saved sections. Emails that already use it don't change.",
  deleteBadge: 'Delete',
  deleteConfirm: 'Delete',
  deleting: 'Deleting…',
  deleted: 'Deleted "{name}"',
})
