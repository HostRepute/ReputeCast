import { defineMessages } from '@/i18n'

/**
 * Review & send (SendReadiness, SendReadinessStates, StressReviewVi): the
 * verdict, the checklist's frame and the words the page writes itself. The
 * checks' findings come from the server with the run.
 */
export const m = defineMessages('review', {
  // Verdicts and check statuses.
  verdictReady: 'Ready',
  verdictWarnings: 'Ready with warnings',
  verdictBlocked: 'Blocked',
  statusPass: 'Pass',
  statusWarn: 'Warn',
  statusBlock: 'Block',
  statusUnknown: 'Unknown',
  countsLine:
    '{passed, plural, one {# check passed} other {# checks passed}}, {warnings, plural, one {# warning} other {# warnings}}, {blocks, plural, one {# block} other {# blocks}}',
  countsShort: '{passed} passed · {warnings, plural, one {# warning} other {# warnings}} · {blocks, plural, one {# block} other {# blocks}}',
  blockedHead:
    '{blocks, plural, one {# blocking issue} other {# blocking issues}}, {warnings, plural, one {# warning} other {# warnings}}, {passed, plural, one {# check passed} other {# checks passed}}. Sending stays disabled until every block is fixed.',
  blockedTailOne: 'It has a one-click fix below.',
  blockedTailEach: 'Each one has a one-click fix below.',
  blockedTailSome: '{fixable, plural, one {# of them has a one-click fix below.} other {# of them have a one-click fix below.}}',
  fixAllBlocks: '{n, plural, =1 {Apply the fix} =2 {Apply both fixes} other {Apply all # fixes}}',
  fixAllWarnings: '{n, plural, =1 {Fix the warning} =2 {Fix both warnings} other {Fix all # warnings}}',
  sendNow: '{recipients, plural, one {Send to # recipient} other {Send to # recipients}}',
  sendScheduled: '{recipients, plural, one {Schedule send to # recipient} other {Schedule send to # recipients}}',
  confirmNote:
    "{warnings, plural, =0 {You'll confirm the audience on the next step.} =1 {You'll confirm the audience and the warning on the next step.} =2 {You'll confirm the audience and the two warnings on the next step.} =3 {You'll confirm the audience and the three warnings on the next step.} other {You'll confirm the audience and the # warnings on the next step.}}",
  ackDefault: 'I understand: {title}.',

  // Times and amounts.
  pickDateTime: 'Pick a date and a time',
  scheduleSentence: '{date} at {time} {timezone}',
  now: 'now',
  days: '{n} d',
  hours: '{n} h',
  minutes: '{n} m',
  waitMinutes: '{n, plural, one {about # minute} other {about # minutes}}',
  waitHours: '{n, plural, one {about # hour} other {about # hours}}',
  durMinutes: '{m} min',
  durHours: '{h} h',
  durHoursMinutes: '{h} h {m} min',
  agoJustNow: 'just now',
  agoMinutes: '{n, plural, one {# minute ago} other {# minutes ago}}',
  agoHours: '{n, plural, one {# hour ago} other {# hours ago}}',
  agoYesterday: 'yesterday',
  agoDays: '{n} days ago',
  thresholdsLine: 'bounce {bounce} · complaint {complaint}',

  // Auto-pause thresholds.
  thresholdBounce: 'Bounce rate',
  thresholdComplaint: 'Complaint rate',
  thresholdMailbox: 'Complaints, per mailbox provider',
  thresholdErrors: 'Provider errors over 3 min',
  errPercent: 'Enter a percentage, for example {example}.',
  errRange: 'Keep it between {min} % and {max} %.',
  errFiveTimes: 'Keep it within five times the overall complaint rate.',
  errLowerFirst: 'The per-mailbox threshold would be more than five times this; lower it first.',

  // Check groups: headings, and the words in running text.
  groupReputation: 'Sender reputation',
  groupAuthentication: 'Authentication',
  groupContent: 'Content',
  groupLinks: 'Links',
  groupCompliance: 'Compliance',
  groupAudience: 'Audience quality',
  groupRouting: 'Routing',
  groupAccessibility: 'Accessibility',
  groupAudienceShort: 'Audience',
  wordReputation: 'reputation',
  wordAuthentication: 'authentication',
  wordContent: 'content',
  wordLinks: 'links',
  wordCompliance: 'compliance',
  wordAudience: 'audience quality',
  wordRouting: 'routing',
  wordAccessibility: 'accessibility',

  // Banners.
  verdictLine:
    '{counts}. Checks finished {when}, {mode, select, now {and will run again when the send starts} other {and will run again 15 minutes before the scheduled send}}.',
  running: 'Running Send Readiness',
  runningCount: 'Running Send Readiness · {done} of {total} checks done',
  checksDone: 'Checks done',
  runningNote: 'Blocklist lookups across 84 sources take the longest, usually 8 to 12 seconds.',
  incompleteHostRepute: "Couldn't finish: HostRepute didn't respond",
  incompleteTitle: "Couldn't finish every check",
  incompleteLine:
    "{passed, plural, one {# check passed} other {# checks passed}}, {warnings, plural, one {# warning} other {# warnings}}{blocks, plural, =0 {} one {, # block} other {, # blocks}}. {hostRepute, select, yes {{unknown, plural, one {The # blocklist check is unknown} other {The # blocklist checks are unknown}}} other {{unknown, plural, one {# check is unknown} other {# checks are unknown}}}}, so the verdict can't be Ready yet.",
  scheduledTitle: '{name} is scheduled',
  startsWithinMinute: 'Starts within a minute',
  scheduledLine:
    '{when}, to {recipients, plural, one {# recipient} other {# recipients}}{hasVia, select, yes { via {via}} other {}}. Checks run again {hasRecheck, select, yes {at {recheck}} other {before the send}}; if anything turns to Block, the send pauses and you get an email{slack, select, yes { and a Slack message} other {}}.',
  tileSendsIn: 'Sends in',
  tileAbTest: '{kind, select, subject {A/B subject test} content {A/B content test} other {A/B send-time test}}',
  tileAbValue: '{percent}% sample · winner at {time}',
  tileRecipients: 'Recipients',
  tileAutoPause: 'Auto-pause',
  tileScheduledBy: 'Scheduled by',
  staleTitle: 'The campaign changed since the checks ran',
  staleLine: 'These results are from {when} (v{version}). Run the checks again before sending: the result can change.',

  // The checklist.
  checklistLabel: 'Send Readiness checklist',
  blockingIssues: 'Blocking issues · {count}',
  fixFirst: 'Fix these first',
  warningsTitle: 'Warnings · {count}',
  notChecked: 'Not checked · {count}',
  runChecksAgain: 'Run the checks again',
  passedTitle: 'Passed · {count}',
  expanded: 'Expanded',
  collapsed: 'Collapsed',
  collapse: 'Collapse',
  showAll: 'Show all {count}',
  queued: 'Queued',
  ofTotal: '{checked} of {total}',
  runningNow: 'running',
  groupSummary: '{group}: {summary}',
  unaffected: '{group} · {short} (from {source}, unaffected)',
  fromApi: 'the {name} API',
})
