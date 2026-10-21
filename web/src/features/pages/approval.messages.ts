import { defineMessages } from '@/i18n'

/** Pages in the approval inbox and review (design expansion phase 7, §6 "Approval inbox and review"). */
export const m = defineMessages('pages.approval', {
  // The inbox
  badge: 'Page',
  badgeLabel: 'Page: open “{name}” in the page builder',
  line: 'Page · {path} · {visibility}',
  whenApproved: 'When approved',
  scheduled: '{date}, {time}',
  published: 'Published',

  // The review
  cardTitle: 'Page',
  cardCaption: 'What the approval publishes: the content, address, visibility and publish time',
  page: 'Page',
  pageValue: '{name} · {template}',
  address: 'Address',
  visibility: 'Visibility',
  goesLive: 'Goes live',
  goesLiveNow: 'As soon as it’s approved',
  goesLivePublished: 'Published once approved',
  version: 'Version',
  versionValue: 'v{version}',
  previewTitle: 'Preview of “{name}”, v{version}',
  previewFullTitle: 'Full preview of “{name}”',
  previewCaption: 'The page as readers will see it, in the site’s frame',
  previewUnavailable: 'The preview isn’t available: {error}',
  previewLoading: 'The preview is loading.',
  openPreview: 'Open full preview',
  openBuilder: 'Open in the page builder',

  // The decision
  decisionNote:
    'Approving covers this exact version: its content, address, visibility and publish time. {when, select, at {It goes live at its publish time} other {It goes live as soon as it’s approved}}. If {name} changes it first, publishing needs a new approval.',
  rejectDetail: '{name} is told and the page isn’t published. Reject when it shouldn’t go live at all; for fixes, request changes instead.',
  changesDetail: 'The request goes back to {name} with your comment. The page isn’t published until they ask again.',
})
