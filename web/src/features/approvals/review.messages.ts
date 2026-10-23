import { defineMessages } from '@/i18n'

/**
 * Approval review (ApprovalReview, ApprovalStates boards): what a request
 * covers, Send Readiness with its evidence, the audience and route, the
 * discussion and the decision.
 */
export const mr = defineMessages('approvals.review', {
  // ── Page ──────────────────────────────────────────────────────────────
  titleFallback: 'Approval request',
  documentTitle: '{name} · Approvals',
  documentTitleFallback: 'Approvals',
  crumbCampaigns: 'Campaigns',
  crumbApprovals: 'Approvals',
  crumbRequest: 'Request',
  notFoundTitle: "This approval request doesn't exist",
  notFoundBody: 'It may have been withdrawn, or the link is from another workspace.',
  backToApprovals: 'Back to approvals',
  loadFailed: "Couldn't load the approval request",
  nothingChanged: 'Nothing was changed.',
  loading: 'Loading the approval request',

  // ── Toasts ────────────────────────────────────────────────────────────
  decided: '{decision, select, approve {Approved} request_changes {Changes requested} other {Rejected}} · {name}',
  decidedProgress: '{given} of {needed} approvals so far.',
  decidedNotified: '{name} was notified.',
  decideFailed: "Couldn't record the decision",
  commentFailed: "Couldn't post the comment",
  testSent: 'Test sent to {to}',
  testVia: 'via {provider}',
  testFailed: "Couldn't send the test",
  fixApplied: 'Fix applied',
  fixAppliedDetail: 'The campaign changed, so the approval restarts with the diff attached.',
  fixFailed: "Couldn't apply the fix",
  unscheduled: "{name} won't go out",
  unscheduledDetail: 'The send was cancelled; the campaign is a draft again.',
  unscheduleFailed: "Couldn't cancel the send",

  // ── Reject and request-changes dialog ─────────────────────────────────
  rejectBadge: 'Reject',
  changesBadge: 'Request changes',
  rejectTitle: 'Reject “{name}”?',
  changesTitle: 'Ask {name} for changes?',
  rejectDescription: "{name} is told and the campaign goes back to a draft. Reject when it shouldn't go out at all; for fixes, request changes instead.",
  changesDescription: 'The request goes back to {name} with your comment. The campaign stays a draft until they ask again.',
  rejectReason: 'Why reject it?',
  changesReason: 'What should change?',
  rejectConfirm: 'Reject',
  changesConfirm: 'Request changes',

  // ── Heading card ──────────────────────────────────────────────────────
  headLabel: 'Request',
  requestedLine:
    'Requested by {name} · {ago} · {needed, plural, one {# approval} other {# approvals}} needed from {approvers} · {role, select, reviewing {you are reviewing as {me}} reviewingAnon {you are reviewing as an approver} author {you are the author} viewing {you are viewing as {me}} other {you are viewing as a member}}',
  rulesMatched: 'Rules matched',

  // ── Approved, then invalidated ────────────────────────────────────────
  invalidatedAfterTitle: 'Approved, then invalidated',
  invalidatedBeforeTitle: 'Invalidated before a decision',
  invalidatedCaption: '{name} · {reason}{after, select, yes { after approval} other {}}',
  campaignChanged: 'the campaign changed',
  stepApproved: '<b>Approved by {name}, {when}</b> · snapshot v{version}{hasComment, select, yes { · comment: “{comment}”} other {}}',
  stepRequested: '<b>Requested by {name}</b> · snapshot v{version}',
  stepEdited: '{name} {edits}{hasWhen, select, yes {, {when}} other {}} · v{version}',
  stepAudienceMoved: 'The audience moved more than 10% since the request · {count} contacts now',
  stepInvalidated:
    '<b>Approval invalidated{hasWhen, select, yes {, {when}} other {}}</b> · {names} notified · {scheduled, select, yes {the send stays scheduled but won’t start without a new approval} other {the send won’t start without a new approval}}',
  editSubject: 'edited the subject',
  editPreheader: 'edited the preheader',
  editContent: 'edited the content',
  editSender: 'changed the sender',
  editAudience: 'changed the audience',
  editRoute: 'changed the route',
  editSchedule: 'moved the schedule',
  editLinkAfterSubject: '{count, plural, one {replaced the link} other {replaced # links}}',
  editLink: '{count, plural, one {replaced a link} other {replaced # links}}',
  editCampaign: 'changed the campaign',
  whatChangedVersions: 'What changed · v{from} → v{to}',
  unchanged: 'Unchanged',
  nothingElse: 'nothing else',
  approveVersion: 'Approve v{version}',
  requestChanges: 'Request changes',

  // ── Campaign ──────────────────────────────────────────────────────────
  campaignTitle: 'Campaign',
  campaignCaption: 'Snapshot the approval covers',
  subject: 'Subject',
  preheader: 'Preheader',
  from: 'From',
  version: 'Version',
  versionEdited: 'v{version} · edited by {name} {ago}',
  previewTitle: 'Preview of “{name}”, v{version}',
  previewUnavailable: "The preview isn't available: {error}",
  previewLoading: 'The preview is loading.',
  openFullPreview: 'Open full preview',
  sendMeTest: 'Send me a test',
  versionNumber: 'Version {version}',
  fullPreviewTitle: 'Full preview of “{name}”',

  // ── Send Readiness ────────────────────────────────────────────────────
  readinessTitle: 'Send Readiness',
  readinessCaption: 'With evidence; fixes change the campaign and restart the approval',
  checkedLine: '{counts} · checked {ago}, rechecked at send time',
  issuesCaption: 'Send Readiness issues',
  colCheck: 'Check',
  colEvidence: 'Evidence',
  colFix: 'Fix',
  excludeThem: 'Exclude them',
  everythingElse: 'Everything else',
  everyCheck: 'Every check',
  groupsPassed: '{groups} checks passed.',
  everyOtherPassed: 'Every other check passed.',
  allPassed: 'Reputation, authentication, content, links, compliance, audience quality and routing checks passed.',
  allChecks: 'All {count} checks',

  // ── Audience and route ────────────────────────────────────────────────
  audienceRouteTitle: 'Audience and route',
  audienceLabel: 'Audience · {count} after exclusions',
  lists: 'Lists',
  excluded: 'Excluded',
  change: 'Change',
  changeNone: 'No change since the request',
  changeSince: '{change} since request',
  changeLimit: '{change} (limit for invalidation: 10%)',
  routeLabel: 'Route · {split}',
  routeCaption: 'Route',
  colProvider: 'Provider',
  colRecipients: 'Recipients',
  colCost: 'Cost',
  costSelfHosted: '{cost} (self-hosted)',

  // ── Schedule and note ─────────────────────────────────────────────────
  scheduleNoteTitle: 'Schedule and note',
  noteTitle: 'Note',
  scheduled: 'Scheduled',
  sendNow: 'Send now',
  bestTime: 'Best time per recipient within 24 h of {when}',
  notScheduled: 'Not scheduled yet',
  estimated: '{when} · estimated {duration} at current rate limits',
  estimatedMinutes: '{n, plural, one {# minute} other {# minutes}}',
  autoPause: 'Auto-pause',
  changesSinceLast: 'Changes since last approval',
  changedSince: '{changes} changed since the previous request',
  nothingChangedSince: 'Nothing changed since the previous request',
  firstRequest: 'First request · nothing to compare',
  noteForApprovers: '{name} · note for approvers',

  // ── Automation ────────────────────────────────────────────────────────
  automationTitle: 'Automation',
  automationCaption: 'The version the approval publishes',
  contactsInFlight: 'Contacts in flight',
  /** What happens to contacts already in the flow when this version goes live, as the publish dialog offered it. */
  migration:
    "{migration, select, move {Move them to v{version} at their current step} other {Let them finish on the version they're on}}",

  // ── Discussion ────────────────────────────────────────────────────────
  discussionTitle: 'Discussion',
  discussionCaption: 'Visible to the author and approvers',
  noComments: 'No comments yet.',
  entryComment: '<b>{name}</b> · {ago}',
  entryDecision: '<b>{name}</b> · {decision, select, approve {approved} request_changes {requested changes} other {rejected}} · {ago}',
  commentLabel: 'Comment',
  replyTo: 'Reply to {names}…',
  addComment: 'Add a comment…',
  writeFirst: 'Write a comment first.',
  postComment: 'Post comment',

  // ── Decision ──────────────────────────────────────────────────────────
  onlyApprovers: 'Only approvers can decide.',
  yourDecision: 'Your decision',
  decision: 'Decision',
  approve: 'Approve',
  reject: 'Reject',
  approveNote:
    'Approving covers this exact snapshot. If {name} edits the content, audience, route or schedule, or the audience moves more than 10%, the approval is invalidated and you see the diff.',
  invalidatedNote:
    '{decide, select, yes {This request covered v{version}. Deciding on v{next} opens a new request with the diff attached.} other {This request covered v{version}. A new request carries the diff to the approvers.}}',
  bypassedReason: 'Sent without approval: “{reason}”',
  openReview: 'Open Review & send',
  author: 'Author',
  authorRole: '{name} · {role}',
  approvers: 'Approvers',
  needed: 'Needed',
  neededValue: '{left} of {needed}',
  requested: 'Requested',
})
