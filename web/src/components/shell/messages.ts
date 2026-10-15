import { defineMessages } from '@/i18n'

/** Every string of the app shell (sidebar, rail, tab bar, More sheet, top bar, menus, banners). */
export const m = defineMessages('shell', {
  // Sidebar sections, in sidebar order.
  navOverview: 'Overview',
  navCampaigns: 'Campaigns',
  navAutomations: 'Automations',
  navTransactional: 'Transactional',
  navAudience: 'Audience',
  navGrow: 'Grow',
  navEarn: 'Earn',
  navForms: 'Forms & pages',
  navDeliverability: 'Deliverability',
  navProviders: 'Sending providers',
  navSettings: 'Settings',

  // Sidebar, rail and tab bar.
  skipToContent: 'Skip to content',
  sidebarLabel: 'Workspace navigation',
  mainNavLabel: 'Main',
  homeLabel: 'ReputeCast home',
  brandName: 'ReputeCast',
  importContacts: 'Import contacts',
  hostReputeConnected: 'HostRepute connected',
  hostReputeAttention: 'HostRepute needs attention',
  hostReputeConnect: 'Connect HostRepute',
  hostReputeSynced: '{team} · synced {ago}',
  hostReputeReconnect: 'Reconnect to resume monitoring',
  hostReputeOptional: 'Optional · free',

  // Phone tab bar: four sections and More.
  moreTab: 'More',
  moreTabLabel: 'More: {sections}',

  // The More sheet.
  moreSheetTitle: 'More',
  moreSheetDescription: 'The other sections, your workspace and HostRepute',
  moreSheetClose: 'Close',
  moreSheetNavLabel: 'More sections',
  moreSheetSwitcherRole: '{role} · switch workspace',

  // Workspace switcher.
  switcherLabel: '{workspace}, {role}. Switch workspace',
  switcherMenuLabel: 'Workspaces',
  switcherHeading: 'Your workspaces',
  switcherWorkspaceLine: '{role} · {count, plural, one {# subscriber} other {# subscribers}}',
  switcherCreate: 'Create workspace',
  switcherAllWorkspaces: 'All workspaces (instance admin)',

  // Settings › General › Sidebar sections.
  sectionsTitle: 'Sidebar sections',
  sectionsCaption: "Hide the sections this workspace doesn't use. Nothing stops working, and members can still open them by link.",
  sectionsTransactional: 'Product mail sent through the API or SMTP relay. Hiding it keeps sending working.',
  sectionsGrow: 'Referral program and recommendations.',
  sectionsEarn: 'Paid subscriptions, offers, gifts and sponsors.',
  sectionsSwitch: 'Show {section} in the sidebar',
  sectionsHidden: '{section} hidden from the sidebar',
  sectionsShown: '{section} shown in the sidebar',
  sectionsHiddenDetail: 'Members can still open it by link.',
  sectionsFailed: "Couldn't change the sidebar",

  // Screens that aren't in this build yet.
  notBuiltTitle: '{screen} is being built',
  notBuiltBody: "This screen isn't available in this build yet.",
})
