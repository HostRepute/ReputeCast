import { defineMessages } from '@/i18n'

/** The workspace Overview's Earn card (EarnCard.tsx, EarnStates). */
export const mcard = defineMessages('earn.overviewCard', {
  title: 'Earn · last 30 days',
  label: 'Earn, last 30 days',
  line: '{provider} · {account} · {mode, select, test {Test} other {Live}}',
  lineNoAccount: '{provider} · {mode, select, test {Test} other {Live}}',
  mrr: 'MRR',
  members: 'Paid members',
  churn: 'Churn',
  trials: 'Trials',
  nextSponsor: 'Next sponsor',
  nextIssue: '#{issue} · {sponsor}',
  creativeDue: 'Creative due {date}',
  creativeReady: 'Creative ready',
  open: 'Open Earn',
  silentTitle: '{provider} webhook silent',
  silentText:
    'No event from {provider} for {hours, plural, one {# hour} other {# hours}}, while {active, plural, one {# subscription is} other {# subscriptions are}} active. Statuses are at most an hour behind.',
  checkWebhook: 'Check the webhook',
  inviteLabel: 'Earn',
  inviteTitle: 'Earn from your newsletter',
  inviteText: 'Paid subscriptions, gifts and sponsorships. Readers pay you through your own Stripe account; ReputeCast takes no fee.',
  setUp: 'Set up Earn',
  notNow: 'Not now',
})
