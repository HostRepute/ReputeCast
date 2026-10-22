import { defineMessages } from '@/i18n'

/**
 * 7.10 Automations: the list (AutomationsList, AutomationsStates), the
 * status words and the starter templates. A template's name and outline
 * come from the server.
 */
export const m = defineMessages('automations', {
  title: 'Automations',
  /** The Automations route in the header: "Automations route · Amazon SES". */
  route: 'Automations route · {route}',
  newAutomation: 'New automation',
  loadFailed: "Couldn't load automations",
  /** After the server's error. */
  stillRunning: 'Flows keep running; only this page failed to load.',
  createTitle: 'Start with a template or a blank canvas',
  routeLead: 'Flows send on the Automations route ({route}) and fail over automatically, so a provider outage never stalls a welcome email.',

  filterLabel: 'Filter',
  filter: '{filter, select, live {Live} paused {Paused} drafts {Drafts} other {All}}',
  filterCount: '{filter, select, live {Live} paused {Paused} drafts {Drafts} other {All}} · {count}',
  noneWithStatus: '{status, select, live {No live automations} paused {No paused automations} drafts {No drafts automations} other {No automations}}',
  showAll: 'Show all automations',
  noneWithStatusBody: '{count, plural, one {# automation} other {# automations}} in this workspace. Pick another filter to see them.',

  // The summary cards (last 7 days).
  inFlowsNow: 'Contacts in flows now',
  acrossLive: 'across {count, plural, one {# live automation} other {# live automations}}',
  emailsSent: 'Emails sent · {days} d',
  /** A provider's sends with the day it took over: "Postmark 212 (failover on 9 Oct)". */
  sentByFailover: '{provider} {count} (failover on {date})',
  noEmailsYet: 'no emails yet',
  humanOpens: 'Human opens',
  clicksRate: 'clicks {rate}',
  goalsReached: 'Goals reached · {days} d',
  noneYet: 'none yet',

  // A row's line under its name.
  pausedLine: 'v{version} · paused {date}{hasBy, select, yes { by {name}} other {}}{hasReason, select, yes {: {reason}} other {}}',
  draftWord: 'draft',
  publishedOn: 'published {date}',

  tableLabel: 'Automations',
  colAutomation: 'Automation',
  colStatus: 'Status',
  colTrigger: 'Trigger',
  colSteps: 'Steps',
  colEntered: 'Entered · 30 d',
  colInFlow: 'In flow',
  colCompleted: 'Completed',
  colHumanOpens: 'Human opens',
  colGoal: 'Goal',
  colActions: 'Actions',
  edit: 'Edit',
  open: 'Open',

  // Alerts above the list.
  pausedAutomatically: 'Paused automatically',
  stepFailing: 'Step failing',
  loadingList: 'Loading automations and their 30-day stats',

  // Status words.
  statusLive: 'Live',
  statusPaused: 'Paused',
  statusDraft: 'Draft',

  // Starter templates.
  createFailed: "Couldn't create the automation",
  starterTemplates: 'Starter templates',
  starterLead: 'Each opens as a draft on the canvas; every step is editable',
  templateSteps: '{count, plural, one {# step} other {# steps}}',
  blankCanvas: 'Blank canvas',
  blankLead: 'Pick a trigger and build from scratch',
})
