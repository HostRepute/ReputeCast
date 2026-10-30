import { defineMessages } from '@/i18n'

/**
 * Collaboration in the composer (CO-10, Composer and ComposerWrite boards):
 * comments on blocks with @mentions, who else has the campaign open,
 * sharing a version for review, and saves merged with someone else's.
 */
export const mx = defineMessages('composer.collab', {
  /** The inspector's tab: "Comments · 2" (open threads). */
  tabComments: '{count, plural, =0 {Comments} other {Comments · #}}',
  commentsRegion: 'Comments',
  /** What a new comment is on: "On the button “Read the docs”". */
  onAnchor: 'On {label}',
  onEmail: 'On the whole email',
  newPlaceholder: 'Comment, or type @ to mention someone',
  newLabel: 'New comment',
  replyPlaceholder: 'Reply, or type @ to mention someone',
  /** The reply box's label: "Reply to Marcus Chen". */
  replyTo: 'Reply to {name}',
  comment: 'Comment',
  posting: 'Posting…',
  reply: 'Reply',
  resolve: 'Resolve',
  reopen: 'Reopen',
  mentionsNote: 'People you @mention get an email with a link to the comment.',
  mentionsList: 'People to mention',
  noMatch: 'Nobody in this workspace matches “{q}”.',
  resolvedCount: '{count, plural, one {# resolved comment} other {# resolved comments}}',
  showResolved: 'Show',
  hideResolved: 'Hide',
  resolvedBy: 'Resolved by {name}',
  reviewer: 'Reviewer',
  noComments: 'No comments yet. Select a block to comment on it, or comment on the whole email.',
  yesterdayAt: 'Yesterday {time}',
  /** The avatar on a commented block: "Comment by Marcus Chen on this button. Open comments". */
  marker: 'Comment by {name} on this {what}. Open comments',
  /** A thread, for screen readers: "Comment thread on the button “Read the docs”". */
  threadOn: 'Comment thread on {label}',
  threadOnEmail: 'Comment thread on the whole email',
  /** A thread's block that is no longer in the email. */
  anchorGone: '{label} · no longer in the email',
  selectBlock: 'Show the block',
  whoResolves: 'People who edit campaigns can resolve comments.',
  commentFailed: "Couldn't post the comment",
  resolveFailed: "Couldn't change the comment",
  loadFailed: "Couldn't load the comments.",
  retry: 'Retry',
  /** What a block is, inside "this {what}" and the anchor labels. */
  what: '{tag, select, mj_button {button} mj_image {image} mj_section {section} mj_column {column} mj_text {text} mj_hero {hero image} mj_divider {divider} mj_social {social links} mj_navbar {navigation} mj_table {table} mj_spacer {spacer} mj_wrapper {wrapper} heading {heading} other {block}}',
  /** "the button “Read the docs”", "the heading “Retiring the v1 API”", "the image “Q4 roadmap”". */
  anchorQuoted: 'the {what} “{text}”',
  anchorPlain: 'the {what}',

  /** Presence (the header): "Also in this campaign: MC Marcus Chen viewing". */
  presenceGroup: 'Also in this campaign',
  presenceName: '{name}, {editing, select, yes {editing} other {viewing}}',
  presenceWord: '{editing, select, yes {editing} other {viewing}}',
  presenceMore: '{count, plural, one {and # more} other {and # more}}',

  /** Share for review (ComposerWrite). */
  share: 'Share for review',
  shareTitle: 'Share for review',
  shareLead: 'Anyone with the link can view and comment without signing in.',
  reviewLink: 'Review link',
  copyLink: 'Copy link',
  copied: 'Link copied',
  copyFailed: "Couldn't copy. Select the link and copy it.",
  shareNote: 'Expires in 7 days. Reviewers see this email only, not your audience, settings or other campaigns.',
  /** "Created by Linh Nguyen just now · shows v15". */
  createdBy: 'Created by {name} {when} · shows v{version}',
  justNow: 'just now',
  revoke: 'Revoke',
  revoked: 'Link revoked. Anyone who opens it sees that it no longer works.',
  newLink: 'Create a new link',
  creating: 'Creating the link…',
  createFailed: "Couldn't create the link",
  /** A link made before the last save. */
  olderVersion: 'Reviewers see v{linkVersion}. You have saved v{version} since.',
  shareLatest: 'Share v{version} instead',
  savedFirst: 'Your edits were saved first, so reviewers see them.',
  closeShare: 'Done',

  /** A save merged with someone else's (CO-10). */
  mergedTitle: "Saved together with {name}'s changes",
  mergedDetail: '{name} changed {parts} (their v{version}) while you edited other parts. Both are kept.',
  mergedSubject: 'the subject',
  mergedPreheader: 'the preheader',
  mergedHead: 'the email styles',
})
