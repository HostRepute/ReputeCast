import { defineMessages } from '@/i18n'

/** Settings › Providers: the SMS providers in the summary, and the SMS budget (brief §6). */
export const m = defineMessages('smsSettingsSummary', {
  title: 'SMS providers',
  senders: '{count, plural, one {# sender} other {# senders}} · {kinds}',
  noSenders: 'No senders yet',
  routes: 'Routes',
  notRouted: 'Not routed yet',
  held: 'Held: {countries}',
  sendersLabel: 'Senders',
  connected: 'Connected',
  connectedOn: '{date} · {name}',
  cost: 'Cost · {month}',
  budget: 'SMS budget · {month}',
  spentOf: '<b>{spent}</b> of {budget}',
  alerts: 'Alerts at {alert}% and 100%. At 100%, campaigns can’t start and SMS steps wait until someone with providers:manage raises it.',
  noBudget: 'No monthly budget: {spend} so far in {month}.',
  foot: 'Senders, routes by country, quiet hours, the rules and the budget live on Sending providers › SMS.',
  changeBudget: 'Change the budget',
  open: 'Open SMS providers',
})
