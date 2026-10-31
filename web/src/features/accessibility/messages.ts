import { defineMessages } from '@/i18n'

/**
 * Accessibility checks (design expansion X06): the workspace policy on
 * Settings › General, Review & send's Accessibility group, the composer's
 * Accessibility panel and the color picker's contrast chip.
 */
export const m = defineMessages('accessibility', {
  // Settings › General › Accessibility (5.5).
  policyTitle: 'Accessibility',
  policyCaption: 'Checks run on every language version: in the composer, in Review & send, and when an automation or a transactional template is published.',
  whenSerious: 'When serious issues remain',
  warn: 'Warn',
  warnDefault: '· default',
  warnDetail: 'Review & send shows them as warnings. Sending stays possible.',
  block: 'Block',
  blockDetail:
    'Sends wait until they’re fixed, and so does publishing an automation or a transactional template. Live automations and published templates keep sending.',
  seriousTitle: 'What counts as serious',
  seriousDetail: 'An image without alt text, a link or button with no text, and text contrast below AA. Everything else is moderate and only warns.',
  minSize: 'Minimum text size',
  minSizeOption: '{px} px',
  minSizeHint: 'Smaller text is a moderate issue.',
  changedOn: 'Last changed by {name} on {date}',
  changedAt: 'Changed by {name} at {time}',
  neverChanged: 'Nobody has changed it yet: Warn and 12 px are the defaults',
  readOnlyTitle: 'Read-only · {name}, {role}',
  readOnlyDetail: "Your role can't change the accessibility policy. Ask {helpers}.",
  readOnlyNobody: "Your role can't change the accessibility policy.",
  readOnlyReason: "Your role can't change the accessibility policy.",
  unsaved: 'Your changes to the accessibility policy aren’t saved.',
  saved: 'Accessibility policy saved · {time}',
  savedBlock: 'Accessibility policy set to Block · {time}',
  savedWarn: 'Accessibility policy set to Warn · {time}',
  saveFailed: 'Couldn’t save the accessibility policy',
  undoFailed: 'Couldn’t undo the change',
  loadFailed: 'Couldn’t load the accessibility policy',
  loading: 'Loading the accessibility policy',

  // Switching to Block: the impact first.
  impactEyebrow: 'Settings › General · Accessibility',
  impactTitle: 'Block sends with serious accessibility issues?',
  impactBody:
    'Sends, publishing an automation and publishing a transactional template will wait until serious issues are fixed. Live automations and published templates keep sending; their next version needs the fix.',
  impactNow: 'What this affects now',
  impactChecking: 'Checking the drafts and scheduled sends…',
  impactFailed: 'Couldn’t check what this affects: {error}',
  impactNone: 'Nothing waiting to be sent or published has a serious issue now.',
  impactCampaignDraft: 'Draft · {count, plural, one {# serious issue} other {# serious issues}}: {summary}',
  impactCampaignApproval: 'Waiting for approval · {count, plural, one {# serious issue} other {# serious issues}}: {summary}',
  impactCampaignApproved: 'Approved · {count, plural, one {# serious issue} other {# serious issues}}: {summary}',
  impactCampaignScheduled:
    'Scheduled for {when} · {count, plural, one {# serious issue} other {# serious issues}}: {summary}. Its send-time recheck would hold it.',
  impactFlowName: '{name} v{version}, {step}',
  impactFlowLive:
    'Draft · {count, plural, one {# serious issue} other {# serious issues}}: {summary}. The live v{live} keeps sending; publishing v{version} needs the fix.',
  impactFlowNew: 'Draft · {count, plural, one {# serious issue} other {# serious issues}}: {summary}. Publishing v{version} needs the fix.',
  impactHeld:
    '{count, plural, =0 {No scheduled campaign would be held at its send-time recheck.} one {# scheduled campaign would be held at its send-time recheck.} other {# scheduled campaigns would be held at their send-time recheck.}}',
  open: 'Open',
  openLabel: 'Open {name}',
  audited: 'Recorded in the audit log',
  switchToBlock: 'Switch to Block',

  // Review & send's Accessibility group (5.13).
  newBadge: 'New',
  blockedTitle: '{count, plural, one {Blocked by # accessibility issue} other {Blocked by # accessibility issues}}',
  blockedLine: '{counts}. {note}',
  openInComposer: 'Open in composer',
  runAgain: 'Run again',
  quoted: '“{text}”',
  oneClickEyebrow: 'Review & send · Accessibility',
  oneClickTitle: '{count, plural, one {Fix # issue in every language?} other {Fix # issues in every language?}}',
  oneClickTitleOne: '{count, plural, one {Fix # issue?} other {Fix # issues?}}',
  oneClickTitleLoading: 'Fix in one click?',
  oneClickSubject: '{subject} · {languages}',
  oneClickHeadingFrom: '− Heading {level} · {texts}',
  oneClickHeadingTo: '+ Heading {level} · the same look',
  oneClickSizeFrom: '− {px} px',
  oneClickSizeTo: '+ {px} px',
  oneClickStay: '{items} stay. Link text is written in each language, in the composer.',
  oneClickAfter: 'The checks run again right after, and the fix is recorded in the audit log.',
  oneClickConfirm: 'Fix {count} in every language',
  oneClickConfirmOne: 'Fix {count}',
  oneClickLoading: 'Reading what the fix changes…',
  oneClickFailed: 'Couldn’t read what the fix changes: {error}',
  oneClickNothing: 'There’s nothing to fix in one click any more. Run the checks again.',
})
