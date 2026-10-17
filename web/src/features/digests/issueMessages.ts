import { defineMessages } from '@/i18n'

/** §6: a digest's issues on the screens that show a campaign (approvals, Review & send, the report). */
export const im = defineMessages('digestIssues', {
  recurringDigest: 'Recurring digest',
  recurringIssue: 'Recurring digest issue',
  openDigest: 'Open the digest',

  // Approvals: inbox and review.
  approveIssue: 'Approve issue #{number}',
  postsTitle: 'Posts in this issue',
  postsCaption: '{count, plural, one {# post} other {# posts}} from {digest}',
  noPosts: 'This issue has no posts.',
  sinceTitle: 'Since #{number}',
  sinceNothing: 'Nothing changed: the template, audience, route and schedule are as approved on {date}.',
  sinceChanged: 'Changed since it was approved on {date}: {changes}. The rest is as approved.',
  sinceFirst: 'The first issue: no earlier issue was approved, so everything is new to the approvers.',
  change: '{what, select, template {the template} audience {the audience} route {the route} schedule {the schedule} other {the settings}}',

  // Review & send.
  issueOf: 'Issue #{number} of {name} · prepared {time} · {recipients} recipients',
  issueOfCollecting: 'Issue #{number} of {name} · {recipients} recipients',
  rulesEach: 'Rules apply to each issue of a recurring digest.',
  releaseTitle: 'Release',
  releaseBody: 'In place of Schedule. Release once the block is fixed; the issue then waits for its approval and goes out at {time}.',
  releaseBodyNoApproval: 'In place of Schedule. Release once the block is fixed; the issue then goes out at {time}.',
  release: 'Release issue',
  released: 'Issue #{number} is released.',
  skip: 'Skip this issue',
  lateLimit: 'Late limit: skipped if not sent by {date}.',
  autoTitle: 'Sent by the digest',
  autoBody: 'Issue #{number} goes out at {time} on its own, {approval, select, yes {once it is approved} other {with no one to press Send}}. Edit it in the composer, or rebuild it from the template on the digest.',
  draftBody: 'Draft mode: issue #{number} goes out when someone sends it here. It is skipped if it isn’t sent by {date}.',
  scheduleNote: 'The digest sends this issue at {when}.',

  // The overview's "Next scheduled campaign".
  nextScheduled: 'Next scheduled campaign',
  issueName: '{name} #{number}',
  nextIssueLine: '{when} · {posts, plural, =0 {no posts yet} one {# post so far} other {# posts so far}}{approval, select, yes { · needs approval} other {}}',

  // The report.
  comparedTitle: 'Compared with issue #{number}',
  comparedFirst: 'The first issue of {name}: there is no earlier issue to compare with.',
  comparedCaption: 'Issue #{number} of {name}',
  metricDelivered: 'Delivered',
  metricOpens: 'Human opens',
  metricClicks: 'Clicks',
  metricUnsubscribes: 'Unsubscribes',
  vsIssue: '{delta} vs #{number}',
  pts: '{delta} pts',
})
