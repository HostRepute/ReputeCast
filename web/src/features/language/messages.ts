import { defineMessages } from '@/i18n'

/**
 * App languages (connect-and-reach brief §3.7, §5.9): Account & security ›
 * Language & region, the browser's suggestion, the instance defaults, the
 * invitation language and the language menu of the sign-in pages.
 */
export const m = defineMessages('language', {
  // Account & security › Language & region.
  title: 'Language & region',
  caption: 'How ReputeCast looks to you. Emails to your contacts use their own language.',
  newBadge: 'New',
  appLanguage: 'App language',
  instanceDefault: 'Instance default ({language})',
  regionFormat: 'Region format',
  followsLanguage: 'Follows the app language',
  time: 'Time',
  time24: '24-hour',
  time12: '12-hour',
  fromRegion: 'From the region format',
  weekStartsOn: 'Week starts on',
  monday: 'Monday',
  sunday: 'Sunday',
  preview: 'Preview',
  timesNote: 'Times in {workspace} use the workspace timezone, {timezone} (<link>Settings › General</link>). Your own timezone, in Profile, sets your quiet hours.',
  helpTranslate: 'Help translate ReputeCast',
  opensInNewTab: '(opens in a new tab)',
  notSaved: 'Not saved',
  save: 'Save',
  saving: 'Saving…',
  pseudoBadge: 'Development',
  pseudoDescription: 'Every string accented, about 40% longer and wrapped in ⟦ ⟧. Strings that aren’t translated show without them.',
  switched: 'Switched ReputeCast to {language}.',
  formatsSaved: 'Language & region saved.',
  undo: 'Undo',
  saveFailed: 'Couldn’t save your language and region',

  // The browser's suggestion, offered once.
  suggestLabel: 'Language suggestion',
  suggestQuestion: 'Your browser prefers {language}. Use ReputeCast in {language}?',
  suggestOnce: 'Offered once. You can change it any time in Account & security.',
  suggestUse: 'Use {language}',
  suggestKeep: 'Keep {language}',
  dismiss: 'Dismiss',

  // The user menu.
  menuItem: 'Language: {language}',

  // Instance admin › Instance & updates › Instance, and the first sign-in.
  instanceCaption: 'Name, logo, public URL, timezone and language',
  defaultLanguage: 'Default language',
  defaultRegion: 'Default region format',
  defaultsHint: 'Used for the sign-in page, members who haven’t chosen, and invitations.',

  // Team › Invite a member.
  invitationLanguage: 'Invitation language',
  invitationDefault: '{language} · instance default',
  invitationHint: 'The invitation email and the page it opens use it. Once they sign in, members pick their own.',

  // The sign-in, invitation and password pages.
  menuLabel: 'Language',
})
