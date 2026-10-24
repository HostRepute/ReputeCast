import { defineMessages } from '@/i18n/define'

/** Sentences for problems the API client words itself. */
export const m = defineMessages('api', {
  unexpected: 'ReputeCast hit an unexpected error; the details are in the server log.',
  unexpectedRef: 'ReputeCast hit an unexpected error; the server log has the details under {ref}.',
})
