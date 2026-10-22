import { defineMessages } from '@/i18n'

/**
 * The composer's link editor offers the workspace's surveys (design
 * expansion, X11; SurveyBuilder · publishing): the link carries each
 * contact's token, so their answers are linked to them.
 */
export const m = defineMessages('surveyLink', {
  title: 'Or link to a survey',
  survey: 'Survey: {name}',
  draft: 'not published yet',
  hint: 'Adds the contact’s token, so answers are linked to them.',
})
