import { defineMessages } from '@/i18n'

/** 7.11 List health (ListHealth board): tiers, the trend, the sunset policy and health per list. */
export const m = defineMessages('audience.health', {
  // ── Page ─────────────────────────────────────────────────────────────
  documentTitle: 'List health',
  recalculated: 'Tiers recalculated nightly · last run {time}',
  loadFailed: "Couldn't load list health",
  nothingChanged: 'Nothing was changed.',
  loading: 'Loading list health',

  // ── Tier cards ───────────────────────────────────────────────────────
  tierWindow: '{tier} · {window}',
  pts: '<c>{delta} pts</c>',
  ptsVsLastMonth: '<c>{delta} pts</c> vs last month',
  entering: '{count} entering this week',

  // ── Trend ────────────────────────────────────────────────────────────
  trendTitle: 'Tier trend · last 12 weeks',
  trendDesc: 'Weekly tier mix over {weeks} weeks; {active}, {sunset}',
  trendDescPolicy: 'Weekly tier mix over {weeks} weeks; {active}, {sunset} after the sunset policy started on {date}',
  tierStays: '{tier} stays at {to} percent',
  tierGrows: '{tier} grows from {from} to {to} percent',
  tierShrinks: '{tier} shrinks from {from} to {to} percent',
  noHistory: 'No tier history yet',
  policyEnabledOn: '{date} · sunset policy enabled',
  trendNote:
    'Each column is one week; heights are shares of the list. Direct values for this week: Active {active}, Cooling {cooling}, At risk {atRisk}, Sunset {sunset}.',

  // ── Sunset policy ────────────────────────────────────────────────────
  daysWithoutOpen: '{n} days without an open',
  days: '{n} days',
  thresholdsNumbers: 'Enter the tier thresholds as numbers of days.',
  policySaved: 'Saved the sunset policy',
  policySavedBody: 'Tiers follow the new thresholds after tonight’s run at {time}.',
  winBack: '{rate} win back so far',
  policyTitle: 'Sunset policy',
  onSince: 'On since {date}',
  onSinceToday: 'On since today',
  activeUntil: 'Active until',
  atRiskFrom: 'At risk from',
  sunsetFrom: 'Sunset from',
  countMachine: 'Count machine opens?',
  humanOnly: 'No, human opens and clicks only',
  reengage: 'Send one re-engagement email when a contact enters Sunset',
  reengageAutomation: '· automation "{name}"',
  reengageDefault: 'Re-engagement',
  stopBroadcasts: "Stop sending Broadcasts {n} days after that if there's still no open",
  stopNote: '· Automations and System mail continue',
  deleteAfter: 'Delete the contact after {n} months in Sunset',
  deleteOn: '· erased with their consent history; suppressions stay',
  deleteOff: '· off; GDPR retention is handled in Settings',
  savePolicy: 'Save policy',
  editFlow: 'Edit re-engagement flow',
  thisWeek: '<b>This week:</b> {text}.',
  weekEnter:
    '{day, select, none {{reengage, select, yes {{count} contacts enter Sunset and get the re-engagement email} other {{count} contacts enter Sunset}}} other {{reengage, select, yes {{count} contacts enter Sunset on {day} and get the re-engagement email} other {{count} contacts enter Sunset on {day}}}}}',
  weekStop:
    "{day, select, none {{count} who didn't respond last month stop receiving Broadcasts} other {{count} who didn't respond last month stop receiving Broadcasts on {day}}}",

  // ── Health per list ──────────────────────────────────────────────────
  perList: 'Health per list',
  figures: '30-day figures · {link}',
  colList: 'List',
  colContacts: 'Contacts',
  colMix: 'Tier mix',
  colNet: 'Net growth · 30 d',
  colHumanOpens: 'Human opens',
  colUnsub: 'Unsub rate',
  colComplaint: 'Complaint rate',
  colBounce: 'Bounce rate',
  colHealth: 'Health',
  savedView: '(saved view)',
  actionsFor: 'Actions for {name}',
  deleteList: 'Delete list',
  tierShare: '{tier} {percent}',
  noLists: 'No lists yet. <link>Add contacts</link> to see how each list is doing.',
  startFlow: 'Start re-permission flow',
})
