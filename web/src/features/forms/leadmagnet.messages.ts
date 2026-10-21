import { defineMessages } from '@/i18n'

/**
 * The form builder's lead magnet and pages (design expansion phase 7, §6
 * "Form builder"): "After confirmation: offer a download", "Used on pages",
 * and the site's own pages among the allowed domains.
 */
export const m = defineMessages('forms.leadMagnet', {
  // After confirmation: offer a download
  title: 'After confirmation',
  offer: 'Offer a download',
  offerHint: 'A file for people who confirm, like a checklist or a guide. It’s on the confirmation page, in a link of their own.',
  upload: 'Upload a file',
  chooseFile: 'Choose the file to offer',
  fileMeta: '{pages, plural, =0 {{kind} · {size}} one {{kind} · # page · {size}} other {{kind} · # pages · {size}}}',
  fileKind: 'File',
  replace: 'Replace',
  replaceLabel: 'Replace {name}',
  remove: 'Remove',
  removeLabel: 'Stop offering {name}',
  days: 'Links work for',
  daysOption: '{days, plural, one {# day} other {# days}}',
  daysText: 'Links work for {days, plural, one {# day} other {# days}}.',
  daysSaved: 'Download links now work for {days, plural, one {# day} other {# days}}',
  daysFailed: 'Couldn’t change how long links work',
  numbers: '{links, plural, one {# link} other {# links}} · {downloaded} downloaded',
  needsDoubleOptIn: 'The file is offered on the confirmation page, so it needs double opt-in.',
  none: 'No download is offered.',
  uploading: 'Uploading · {done} of {size}',
  checking: 'Checking the file…',
  progress: 'Upload progress',
  cancel: 'Cancel',
  cancelLabel: 'Cancel the upload of {name}',
  stopped: 'The upload stopped at {pct}%.',
  tryAgain: 'Try again',
  uploaded: '{name} is offered after confirmation',
  uploadFailed: 'Couldn’t upload {name}',
  loadFailed: 'Couldn’t load the download: {error}',
  retry: 'Retry',
  removeBadge: 'Remove download',
  removeTitle: 'Stop offering {name}?',
  removeBody:
    '{links, plural, =0 {People who confirm no longer get a download.} one {People who confirm no longer get a download, and the link already sent stops working.} other {People who confirm no longer get a download, and the # links already sent stop working.}}',
  removeConfirm: 'Remove the file',
  removing: 'Removing…',
  removed: '{name} is no longer offered',

  // Used on pages
  pagesTitle: 'Used on pages',
  pagesNone: 'No page uses this form yet. Add a signup form block to a page in <link>Pages</link>.',
  pagesFailed: 'Couldn’t load the pages: {error}',
  pageOpen: 'Open {name} in the page builder',
  statusPublished: 'Published',
  statusScheduled: 'Scheduled',
  statusDraft: 'Draft',
  statusUnpublished: 'Unpublished',
  draftOnly: '{path} · in its unpublished changes',

  // Allowed domains
  ownOrigins: 'Your own pages on {hosts} are always allowed.',
})
