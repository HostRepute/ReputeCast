import { defineMessages } from '@/i18n'

/**
 * The page behind a review link (CO-10): one version of one campaign's
 * email for someone without an account, who can comment on it. Public,
 * outside the app shell, never indexed.
 */
export const mr = defineMessages('reviewlinks', {
  docTitle: 'Review: {campaign} · {workspace}',
  docTitleLoading: 'Review · ReputeCast',
  skip: 'Skip to the email',
  badge: 'Review copy',
  sharedBy: 'Shared by {workspace}',
  /** Under the campaign's name: "Version 15 · link expires Tue 3 Nov". */
  caption: 'Version {version} · link expires {date}',
  from: 'From',
  subject: 'Subject',
  preheader: 'Preheader',
  sample: 'Personalized for a sample reader, {name}.',
  frame: 'The email',
  comments: 'Comments',
  commentsLead: 'Your comments go to the team at {workspace}. They see your name with each one.',
  yourName: 'Your name',
  namePlaceholder: 'So the team knows who commented',
  comment: 'Comment',
  commentPlaceholder: 'What should change?',
  post: 'Comment',
  posting: 'Posting…',
  reply: 'Reply',
  replyTo: 'Reply to {name}',
  threadBy: 'Comment by {name}',
  cancel: 'Cancel',
  noComments: 'No comments yet.',
  reviewer: 'Reviewer',
  resolved: 'Resolved',
  postFailed: "Couldn't post the comment",
  goneBadge: 'Link not working',
  goneTitle: 'This review link no longer works',
  goneBody: 'It expired or was revoked. Ask whoever shared it for a new one.',
  limitedBadge: 'Too many requests',
  limitedTitle: 'Too many requests from your network',
  limitedBody: 'Wait a few minutes, then try again.',
  errorBadge: 'Something went wrong',
  errorTitle: "Couldn't open the email",
  errorBody: 'Check your connection and try again.',
  madeWith: 'Made with ReputeCast, the self-hosted newsletter platform.',
})
