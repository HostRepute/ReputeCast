import { defineMessages } from '@/i18n'

/**
 * The notification center (A5 §5.16): the category filters, the list and my
 * preferences. Each notification's title and body, and the event types'
 * names and audiences in the preferences, come from the server as they are.
 */
export const m = defineMessages('notifications', {
  title: 'Notifications',
  account: 'Account & security',
  filterLabel: 'Filter',
  filter: '{filter, select, unread {Unread} approvals {Approvals} sending {Sending} providers {Providers} team {Team} hostrepute {HostRepute} other {All}}',
  unreadCount: 'Unread · {count}',

  // The list.
  listTitle: 'Notifications',
  listCaption: 'Each one links to the screen where you can act',
  markAll: 'Mark all as read',
  allRead: 'Everything is read',
  loadFailed: "Couldn't load notifications",
  nothingMarked: 'Nothing was marked as read.',
  loading: 'Loading notifications…',
  caughtUp: "You're all caught up",
  nothingUnread: 'Nothing unread',
  noneInCategory:
    '{filter, select, approvals {No Approvals notifications} sending {No Sending notifications} providers {No Providers notifications} team {No Team notifications} hostrepute {No HostRepute notifications} other {No notifications}}',
  emptyBody: 'Approvals, auto-pauses, provider incidents, team changes and HostRepute alerts show up here.',
  // A notification's category, on its badge.
  category:
    '{category, select, approvals {Approvals} sending {Sending} providers {Providers} team {Team} hostrepute {HostRepute} integrations {Integrations} transactional {Transactional} ai {AI} earn {Earn} website {Website} grow {Grow} audience {Audience} other {Other}}',
  new: 'New',
  retention: 'Older notifications are kept for {days, plural, one {# day} other {# days}}.',

  // My preferences.
  prefsTitle: 'My preferences',
  prefsLoadFailed: "Couldn't load your preferences",
  prefsCaptionVia: 'Per event type · emails go out on the System stream via {provider}',
  prefsCaption: 'Per event type · email starts once a sending provider handles the System stream',
  prefsTable: 'Notification preferences',
  colEvent: 'Event',
  colInApp: 'In-app',
  colEmail: 'Email',
  // event: the event type's name, from the server ("Approval requested").
  inAppFor: 'In-app for {event}',
  emailFor: 'Email for {event}',
  quietHours: 'Quiet hours · email only',
  quietStart: 'Quiet hours start',
  quietEnd: 'Quiet hours end',
  // start, end: the two time selects.
  quietRange: '{start} <k>to</k> {end}',
  quietUrgent: 'Auto-pauses and provider outages always send immediately.',
  digest: 'Email digest',
  digestImmediate: 'Immediately',
  digestHourly: 'Hourly digest',
  digestDaily: 'Daily digest at {time}',
  emailOff: 'Email off',
  emailOffBody: 'No sending provider handles the System stream yet, so notifications stay in-app. Your email choices apply once one does.',
  notSaved: 'Not saved',
  save: 'Save preferences',
  saved: 'Preferences saved',

  // Sending summary by email (M57 G-14): a member who can view reports gets
  // the workspace's numbers weekly or monthly. Not the email digest above,
  // which batches notifications.
  summaryTitle: 'Sending summary by email',
  summaryCaption: 'Campaigns, audience and sender health of the last week or month, as far as your role shows them · times in your time zone',
  summaryOff: 'Off',
  summaryOffHint: 'No summary email',
  summaryWeekly: 'Weekly',
  // time: 09:00 on the reader's clock.
  summaryWeeklyHint: 'Mondays at {time}',
  summaryMonthly: 'Monthly',
  summaryMonthlyHint: 'On the 1st at {time}',
  // when: "Mon 17 Nov · 09:00"; email: the reader's sign-in address.
  summaryNext: 'Next one: {when} · sent to {email}',
  summaryLoading: 'Loading your sending summary…',
  summaryLoadFailed: "Couldn't load your sending summary",
  summaryNotSaved: "Sending summary wasn't changed",
})
