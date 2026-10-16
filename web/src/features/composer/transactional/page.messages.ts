import { defineMessages } from '@/i18n'

/** Opening and creating a transactional template: loading and error states. */
export const mt = defineMessages('transactional.editorPage', {
  documentTitle: '{name} · Transactional template · ReputeCast',
  documentTitleLoading: 'Transactional template · ReputeCast',
  newTitle: 'New transactional template · ReputeCast',
  opening: 'Opening the template',
  openingDetail: 'Content, variables and test data · usually under a second',
  creating: 'Creating a draft',
  creatingDetail: 'New template · nothing is sent until it is published',
  untitled: 'Untitled template',
  backToTemplates: 'Back to templates',
  goToTemplates: 'Go to templates',
  notFoundBody: 'The template may have been deleted, or the link was copied before it was saved. Deleted templates stay in the audit log.',
  cantOpen: "Your role can't open this template",
  cantCreate: "Your role can't create transactional templates",
  titleNotFound: 'Not found · ReputeCast',
  titleForbidden: 'No access · ReputeCast',
  titleBroken: 'Something broke · ReputeCast',
})
