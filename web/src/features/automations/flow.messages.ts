import { defineMessages } from '@/i18n'

/**
 * 7.10 Automation flow: the full-screen editor (AutomationFlow), its canvas
 * and the publish, version history, pause and email dialogs
 * (AutomationPublish, AutomationsStates). Diff rows, pre-publish checks and
 * the migration estimates come from the server.
 */
export const mf = defineMessages('automations.flow', {
  documentTitle: '{name} · Automations · {workspace} · ReputeCast',
  skip: 'Skip to content',
  back: 'Back to automations',
  eyebrow: 'Automation',
  unpublishedChanges: 'Unpublished changes · {count}',
  route: 'Automations route · {route}',
  draftNotSaved: 'Draft not saved',
  savingDraft: 'Saving the draft',
  draftSaved: 'Draft saved',
  pause: 'Pause',
  resume: 'Resume',
  versionHistory: 'Version history',
  publishVersion: 'Publish v{version}',

  editReason: "Your role can't edit automations.",
  loadLatestFirst: 'Load the latest draft first.',
  nothingToPublish: 'Nothing to publish: the draft is the same as v{version}.',
  notAllowed: 'Not allowed',
  createFailed: "Couldn't create the automation",

  // Saving.
  notSavedError: 'The draft is not saved.',
  saveFailedError: "Couldn't save the draft.",
  savedToast: 'Draft v{version} saved',
  publishWhenReady: 'Publish it when it looks right.',
  saveFailed: "Couldn't save the draft",
  saveBeforePublishFailed: "Couldn't save the draft before publishing",
  /** Finishes "…; otherwise {work} and you'll be asked to sign in again." */
  whileEnding: 'the draft saves itself',
  afterEnded: '"{name}" is saved as draft v{version} and will reopen where you left off.',
  unsavedGuard: "Your last changes to this flow couldn't be saved. Leave without them, or try saving again.",

  // Status.
  liveSince: 'Live · v{version} since {date}',
  live: 'Live · v{version}',
  pausedSince: 'Paused · v{version} since {date}',
  pausedVersion: 'Paused · v{version}',
  pausedSinceNoVersion: 'Paused since {date}',
  paused: 'Paused',
  /** The paused badge's tooltip: who paused it and why. */
  pausedBy: 'by {name}',
  pausedByReason: 'by {name}: {reason}',
  draftNotPublished: 'Draft · not published yet',
  resumed: '"{name}" resumed',
  resumedDetail: 'New contacts enter again and waiting steps continue.',
  resumeFailed: "Couldn't resume",
  pausedToast: '"{name}" paused',
  pausedDetail: 'Contacts stay where they are until you resume.',
  pauseFailed: "Couldn't pause",

  conflictTitle: 'This draft changed somewhere else',
  loadLatest: 'Load the latest draft',
  conflictBody: "Someone saved a newer version of this draft, maybe you in another tab. Your last change here wasn't saved; load the latest draft to keep editing.",

  // Left panel.
  stepsPanel: 'Steps',
  stepsTabs: 'Steps and stats',
  tabSteps: 'Steps',
  tabStats: 'Stats',
  dragHint: '{snap, select, on {Drag a step onto the canvas or onto an edge. Snap to grid is on.} other {Drag a step onto the canvas or onto an edge. Snap to grid is off.}}',
  lookAround: "You can look around; your role can't edit automations.",
  statsStart: 'Stats start when the automation is published.',
  triggerChanged: 'Trigger changed to {trigger}',
  triggerChangedDetail: 'A flow has one trigger; its other settings start fresh.',

  // The canvas toolbar.
  liveStats: 'Live stats · last 30 days',
  liveStatsEntered: 'Live stats · last 30 days · {count} entered',
  noStatsYet: 'No stats until the first publish',
  showStats: 'Show stats on nodes',
  snapToGrid: 'Snap to grid',
  reducedMotion: 'Your system asks for reduced motion, so the flow stays still.',
  animate: 'Animate flow',
  /** {zoom} is the zoom level ("100%"); {modifier} is ⌘ or Ctrl. */
  zoomHint: 'Zoom {zoom} · scroll to pan · {modifier} + wheel to zoom',
  canvasLabel: 'Flow canvas',
  emptyTitle: 'Start with a trigger',
  emptyEdit: 'Drag one from the left, or press it to place it here.',
  emptyView: 'This flow has no steps yet.',
  /** A step's accessible name: "Send email: Welcome · step 1". */
  nodeLabel: '{type}: {title}',
  /** An edge's accessible name. */
  edgeLabel: '{from} to {to}',
  edgePathLabel: '{from} to {to}, {path} path',

  // Canvas controls and the minimap.
  controls: 'Canvas controls',
  zoomIn: 'Zoom in',
  zoomOut: 'Zoom out',
  fitView: 'Fit view',
  lock: 'Lock canvas',
  unlock: 'Unlock canvas',
  minimap: 'Minimap',

  // What screen readers say about the canvas's steps and edges (React Flow's descriptions).
  a11yStepMove: 'Press Enter or Space to select a step, then use the arrow keys to move it. Press Delete to remove it, or Escape to cancel.',
  a11yStep: 'Press Enter or Space to select a step. Press Delete to remove it, or Escape to cancel.',
  a11yStepView: 'Press Enter or Space to select a step, or Escape to cancel.',
  a11yEdge: 'Press Enter or Space to select an edge. Press Delete to remove it, or Escape to cancel.',
  a11yEdgeView: 'Press Enter or Space to select an edge, or Escape to cancel.',
  /** Announced after a step moves with an arrow key; x and y are canvas pixels. */
  a11yMoved: 'Moved the step {direction, select, up {up} down {down} left {left} other {right}}. New position: x {x}, y {y}.',

  // The step on the canvas. Its stats words are short: a step is 224 px wide.
  /** A trigger's header before its kind ("Trigger · joined list"), in capitals: one line if it can. */
  triggerHead: 'Trigger',
  newInVersion: 'new in v{version}',
  routeOverride: 'route override',
  automationsRoute: 'Automations route',
  statEntered: '<b>{count}</b> entered · 30 d',
  statWaiting: '<b>{count}</b> waiting or passed',
  statSent: '<b>{count}</b> sent',
  statHumanOpens: '<b>{rate}</b> human opens',
  statOpens: '<b>{rate}</b> opens',
  statClicks: '<b>{rate}</b> clicks',
  /** A fallback provider's sends on an email step: "Postmark 212 · failover". */
  statFailover: '{provider} <n>{count}</n> · failover',
  statReached: '<b>{count}</b> reached · {rate}',
  statOfEntered: '{rate} of {count}',
  statExited: '<b>{count}</b> exited',
  statPassed: '<b>{count}</b> passed',
  statFilteredOut: '<b>{count}</b> filtered out',
  statFailedCalls: '<b>{count}</b> failed calls',
  noStatsUntilPublished: 'no stats until published',

  // Right panel.
  settingsPanel: 'Step settings',
  settingsTabs: 'Step settings and stats',
  tabStep: 'Step',
  flowTitle: 'Automation',
  selectStep: 'Select a step on the canvas to see its settings.',
  name: 'Name',
  description: 'Description',
  beforePublish: 'Before you publish',
  allConnected: 'Every step is connected and set up.',
  nameFailed: "Couldn't save the name",
  flowStatsTitle: 'Flow · last 30 days',
  selectStepStats: 'Select a step for its own numbers.',
  entered: 'Entered <b>{count}</b>',
  inFlowNow: 'In the flow now <b>{count}</b>',

  loading: 'Loading the automation',
  loadFailed: "Couldn't load the automation",
  /** After the server's error. */
  loadFailedBody: 'Your flows keep running; nothing was changed.',
  notFoundTitle: "This automation doesn't exist",
  notFoundBody: 'It may have been deleted, or the link is wrong. Nothing was changed.',
  forbiddenTitle: "You can't open this automation",

  // The publish dialog.
  kindAdded: 'Added',
  kindChanged: 'Changed',
  kindRemoved: 'Removed',
  kindUnchanged: 'Unchanged',
  prepareFailed: "Couldn't prepare the publish",
  /** After the server's error. */
  nothingPublished: 'Nothing was published.',
  publish: 'Publish',
  liveBadge: 'Live · v{version}',
  draftBadge: 'Draft · v{version}',
  publishTitle: 'Publish version {version} of "{name}"?',
  publishLead:
    '{count, plural, one {# change} other {# changes}} since v{from} was published{hasDate, select, yes { on {date}} other {}}{hasBy, select, yes { by {name}} other {}}. Every step is re-validated before it goes live.',
  publishLeadFirst: 'This is the first version. Every step is validated before it goes live.',
  versionDiff: 'Version diff · v{from} → v{to}',
  inFlowRightNow: '{count, plural, one {# contact is} other {# contacts are}} in this flow right now',
  migrationLabel: 'Contacts already in the flow',
  finishOn: 'Let them finish on v{version} (recommended)',
  /** {estimate} is when the last contact leaves, from the server. */
  finishDetail: 'New entrants start on v{to}. Nobody gets a step twice or skips one. v{from} retires when the last contact leaves, {estimate}.',
  moveTo: 'Move them to v{version} at their current step',
  checks: 'Pre-publish checks',
  backToCanvas: 'Back to canvas',
  saveDraftOnly: 'Save draft only',
  draftSavedDetail: 'Nothing changed for contacts in the flow.',
  waitingApproval: 'v{version} is waiting for approval',
  waitingApprovalDetail: 'It goes live once approved.',
  isLive: '"{name}" v{version} is live',
  movedDetail: 'Contacts moved to it at their current step.',
  finishDetailToast: 'Contacts in the flow finish on v{version}.',
  publishFailed: "Couldn't publish",

  // Version history.
  restored: 'v{version} restored as draft v{draft}',
  restoreFailed: "Couldn't restore the version",
  changesSinceLive: 'Changes since the live version',
  historyTitle: 'Version history · {name}',
  backToVersions: 'Back to versions',
  compare: 'Compare',
  noChanges: 'No changes.',
  liveWord: 'Live',
  retiring: 'retiring, {count} still in flow',
  retired: 'retired {date}',
  liveInFlow: 'Live · {count} in flow',
  replaceDraft: 'Replace the unpublished draft?',
  replace: 'Replace',
  keep: 'Keep',
  restoreAsDraft: 'Restore as draft',

  // Pausing.
  pauseTitle: 'Pause "{name}"?',
  pauseBody:
    'No new contacts enter. The {count, plural, one {# contact} other {# contacts}} in the flow stay where they are; delays keep counting, and nothing is sent until you resume.',
  keepRunning: 'Keep it running',
  pauseAutomation: 'Pause automation',
  reason: 'Reason',
  reasonHint: 'Shown on the automations list, e.g. “beta is full”.',

  // The email dialogs.
  preview: 'Preview',
  previewOf: 'Preview of {name}',
  composer: 'Composer',
  composerOf: 'Composer · {name}',
  editingLead: 'Editing v{version}. Saving makes v{next}; this step switches to it, other steps keep their version until you pick it.',
  savedAs: 'Saved as v{version}',
  savedAsDetail: 'The step now sends it once you publish.',
  emailSaveFailed: "Couldn't save the email",
  saveAsNew: 'Save as new version',
  subject: 'Subject',
  preheader: 'Preheader',
  html: 'HTML',
  /** {example} and {unsubscribe} are Liquid tags, shown as code. */
  htmlHint: 'Liquid tags like {example} and {unsubscribe} are filled in when it sends.',
})
