import { defineMessages } from '@/i18n'

/** Account & security's header (A5 §5.6, AccountSecurity and LanguageRegion boards). */
export const m = defineMessages('account', {
  title: 'Account & security',
  roleIn: '{role} of {workspace}',
  instanceAdmin: 'instance admin',
  loadFailed: "Couldn't load your account",
  nothingChanged: 'Nothing was changed.',
  retry: 'Retry',
})
