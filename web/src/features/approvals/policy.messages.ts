import { defineMessages } from '@/i18n'

/** Settings › Approvals: the send approval policy (ApprovalPolicy board). */
export const mp = defineMessages('approvals.policy', {
  settings: 'Settings',
  documentTitle: 'Approvals · Settings',
  loadFailed: "Couldn't load the approval policy",
  nothingChanged: 'Nothing was changed.',
  loading: 'Loading the approval policy',

  numberRange: 'Enter a number of recipients between {min} and {max}.',
  saved: 'Approval policy saved',
  savedDetail: 'It applies to the next request; open requests keep the rules they matched.',
  saveFailed: "Couldn't save the approval policy",

  readOnly: 'Read-only',
  readOnlyLead: 'You can view the approval policy.',
  readOnlyReason: 'Changing it needs the workspace:manage permission.',

  // ── When is approval required? ────────────────────────────────────────
  whenTitle: 'When is approval required?',
  whenCaption: 'Per workspace',
  whenIntro: 'Approval is required when any enabled rule matches. The author sees which rules matched and can add a note.',
  audienceTitle: 'A Broadcast audience of at least',
  audienceDetail: 'Counted after exclusions, at request time and again at send time.',
  minimumRecipients: 'Minimum recipients',
  recipients: 'recipients',
  warmingTitle: 'The route includes a Warming-up provider, or one connected in the last {days} days',
  matchesToday: 'Matches today: {what}',
  warmingNone: 'Matches nothing today: no provider in the Broadcasts route is warming up or new.',
  warningsTitle: 'Send Readiness is “Ready with warnings”',
  warningsDetail: 'Blocked sends never start; this rule adds a human check for warnings.',
  lacksSendTitle: 'The author lacks the send permission',
  lacksSendDetail:
    'Members with campaigns:edit but not campaigns:send can still prepare a send and request approval. {who, select, roles {{roles} today: {names}.} members {Members today: {names}.} other {Nobody is in that position today.}}',
  listsTitle: 'The audience includes a list marked “Requires approval”',
  listsDetail:
    '{marked, select, yes {Marked today: {lists}. Mark more in <link>Audience → Lists</link>.} other {No list is marked yet. Mark lists in <link>Audience → Lists</link>.}}',
  automationTitle: 'Publishing an automation also requires approval',
  automationDetail: 'Applies to automations:publish. Off: publishing is logged only.',

  // ── Approvers ─────────────────────────────────────────────────────────
  approversTitle: 'Approvers',
  approversNote:
    'Approvers are members with <mono>campaigns:approve</mono>. {has, select, yes {Today: {list}.} other {Nobody holds it today: give it to a role in Settings › Roles.}} Authors can never approve their own campaign.',
  approverWithRole: '<b>{name}</b> ({role})',
  approverName: '<b>{name}</b>',
  requiredLegend: 'Approvals required',
  oneApproval: '1 approval',
  oneApprovalDetail: 'Any approver other than the author',
  twoApprovals: '2 approvals',
  twoApprovalsDetail: 'Two different approvers; with {count} approvers today one of them must not be the author',
  twoNeeds: '{count, plural, one {Needs two members with campaigns:approve; today there is #.} other {Needs two members with campaigns:approve; today there are #.}}',
  snapshot: 'Snapshot',
  snapshotDetail:
    'An approval covers content and subject, the audience definition, the sending route and the schedule. Changing any of them, or the audience count moving by more than 10%, invalidates it; the approver sees exactly what changed.',
  recheck: 'Recheck',
  recheckDetail: 'Send Readiness runs again when the send starts. A Blocked result stops the send and notifies author and approvers.',

  // ── Emergency bypass ──────────────────────────────────────────────────
  bypassTitle: 'Emergency bypass',
  bypassCaption: 'Owners only',
  bypassSwitch: 'Owners can send without approval by entering a reason',
  bypassLogged: 'The bypass is logged as <mono>campaign.approval_bypassed</mono> and every approver is notified immediately.',
  used: 'Used',
  usedCount: '{count, plural, one {# time} other {# times}} in the last 90 days',

  discard: 'Discard',
  savePolicy: 'Save policy',
  unsaved: "Your changes to the approval policy aren't saved. Requests keep following the saved policy.",

  // ── Pending now ───────────────────────────────────────────────────────
  pendingTitle: 'Pending now · {count}',
  approvalInbox: 'Approval inbox',
  pendingCaption: 'Pending approval requests',
  colCampaign: 'Campaign',
  colAuthor: 'Author',
  colAudience: 'Audience',
  colRequested: 'Requested',
  colActions: 'Actions',
  nothingPending: 'Nothing is waiting for approval.',
  review: 'Review',
})
