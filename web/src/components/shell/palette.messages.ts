import { defineMessages } from '@/i18n'

/**
 * The command palette on ⌘K (CommandPalette; review 2026-11 FE-1). Names,
 * addresses and counts are arguments, never translated.
 */
export const mp = defineMessages('shell.palette', {
  label: 'Search',
  placeholder: 'Search campaigns, contacts, automations, lists, forms, pages and settings',
  description: 'Results show only what your role can open.',
  close: 'esc',
  recent: 'Recent',
  goTo: 'Go to',
  // kind: a group's name; count: its results.
  groupCount: '{kind} · {count, number}',
  campaigns: 'Campaigns',
  contacts: 'Contacts',
  automations: 'Automations',
  lists: 'Lists',
  segments: 'Segments',
  forms: 'Forms',
  pages: 'Pages',
  settings: 'Settings',
  settingIn: 'Settings › {tab}',
  kindCampaign: 'Campaign',
  kindContact: 'Contact',
  kindAutomation: 'Automation',
  kindList: 'List',
  kindSegment: 'Segment',
  kindForm: 'Form',
  kindPage: 'Page',
  // A campaign's line: "Campaign · sent Oct 14 · 48,213 recipients".
  campaignSent: 'sent {date}',
  campaignScheduled: 'scheduled for {date}',
  recipients: '{count, plural, one {# recipient} other {# recipients}}',
  subscribed: '{count, plural, one {# subscribed} other {# subscribed}}',
  contactsCount: '{count, plural, one {# contact} other {# contacts}}',
  newCampaign: 'New campaign',
  noResults: 'No results for “{q}”',
  noResultsHint: 'Check the spelling, or search by an email address. Archived campaigns and deleted lists aren’t searched.',
  searching: 'Searching…',
  failed: 'Couldn’t search: {error}',
  toMove: 'to move',
  toOpen: 'to open',
  toClose: 'to close',
  onlyYours: 'Only what your role can open',
  results: '{count, plural, one {# result} other {# results}}',
})
