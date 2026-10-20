import { defineMessages } from '@/i18n'

/**
 * The website's changes elsewhere in the app (WebsiteStates · elsewhere):
 * Campaigns' Web column, Review & send's Web line, the report's Web
 * section, the contact profile's Website card, the contacts filter, the
 * segment condition and approvals.
 */
export const me = defineMessages('website.elsewhere', {
  // Review & send
  webLine: 'Web',
  webLineText: 'Also publishes {url}, {visibility}, {when}.',
  webWithSend: 'with the send',
  webAt: 'at {date}',
  webNow: 'as soon as it goes out',
  visPublic: 'public',
  visMembers: 'for members',
  visPaid: 'for paid members',
  edit: 'Edit',

  // Campaigns · the Web column
  colWeb: 'Web',
  webScheduled: 'Scheduled',
  webPublished: 'Published',
  webWithSendLine: 'with the send',
  webFeatured: 'featured on home',
  webUnpublishedLine: 'unpublished {date}',
  webNotOn: 'not on the web',
  webAtLine: '{date}',

  // The report's Web section
  reportTitle: 'Web',
  reportSub: '{url} · {status}',
  reportViews: 'Views · 30 days',
  reportViewsSub: 'all time {total}',
  reportSignups: 'Signups',
  reportSignupsSub: 'members who joined from the post',
  reportUpgrades: 'Upgrades',
  reportUpgradesSub: 'to a paid tier',
  topReferrers: 'Top referrers',
  referrersLabel: 'Views by referrer',
  utm: 'UTM sources: {sources}. Counted on the server, without cookies.',
  utmNone: 'Counted on the server, without cookies.',
  openPost: 'Open the post',
  noViews: 'No views yet.',
  statusPublished: 'published {date}',
  statusScheduled: 'scheduled',
  statusUnpublished: 'unpublished',
  statusDraft: 'not published yet',

  // The contact profile's Website card
  cardTitle: 'Website',
  cardSub: '{host}',
  member: 'Member',
  memberSince: 'since {date} · {kind}',
  memberFree: 'free',
  memberPaid: 'paid · {tiers}',
  notMember: "Not a member: hasn't signed in to {host}",
  lastSignin: 'Last sign-in',
  lastSigninText: '{date} · {method}',
  methodLink: 'with a sign-in link',
  methodCode: 'with a code',
  methodSignup: 'signing up on the site',
  sessions: 'Sessions',
  sessionsText: '{count, plural, =0 {none active} one {# active · ends {ends}} other {# active · the last ends {ends}}}',
  sessionsNone: 'none active',
  meter: 'Meter',
  meterText: '{used} of {limit} free paid {limit, plural, one {post} other {posts}} used in {month}',
  signOutEverywhere: 'Sign out everywhere…',
  signOutHint: 'Needs subscribers:edit · audited',
  signOutTitle: 'Sign {name} out everywhere?',
  signOutText: 'Every session on {host} ends now: {count, plural, one {# browser} other {# browsers}}. They can sign in again with a new link.',
  signOutBadge: 'Sign out everywhere',
  signedOut: '{name} is signed out everywhere',
  eventSignedIn: 'Signed in to {host}',
  eventSignedInDetail: '{time} · {method}{same, select, yes {, opened in the same browser} other {}}',
  eventRefused: "A sign-in link wasn't sent",
  eventSignedOut: 'Signed out everywhere',
  eventSignedOutBy: 'by {actor}',
  eventJoined: 'Joined on {host}',
  eventEmailChanged: 'Changed their email address',
  siteOff: "The website is off: members can't sign in.",

  // The contacts filter
  filterSignedIn: 'Signed in to the website · {count}',
  filterPaid: 'Paid members · {count}',
  filterLabel: 'Members',

  // The segment condition
  condSignedIn: 'Signed in to the website',
  condOperator: 'Operator',
  condWhen: 'When',
  opSince: 'since',
  opNotSince: 'not since',
  opEver: 'ever',
  opNever: 'never',
  days7: '7 days ago',
  days30: '30 days ago',
  days90: '90 days ago',
  onDate: 'a date…',

  // Approvals
  policyPost: 'Publishing a post or page needs approval',
  policyPostText: "Off by default. Approvers are the workspace's approvers (campaigns:approve). The approval covers the content, address, visibility and publish time.",
  policyPostNote: '“The author lacks the send permission” also covers {perm}: an author without it gets Request approval instead of Publish.',
  inboxPost: 'Post',
  inboxPostLine: 'Post · {visibility} · requested by {name}, {date} · web preview',
  reviewWeb: 'The post on the web',
  reviewWebSub: '{url} · as a reader with access sees it',
  reviewTitle: 'Title',
  reviewAddress: 'Address',
  reviewVisibility: 'Visibility',
  reviewWhen: 'Goes live',
  reviewExcerpt: 'Excerpt',
  reviewFrame: '{title} on the web',
  reviewOpen: 'Open the full page',
  reviewLoading: 'The page is loading.',
  whenApproved: 'when approved',
})
