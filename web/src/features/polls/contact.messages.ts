import { defineMessages } from '@/i18n'

/**
 * Poll answers and survey responses on the contact profile (design
 * expansion, X11; brief §6): timeline items, and where a field came from.
 */
export const m = defineMessages('pollContact', {
  answered: 'Answered {answer} to “{question}” in {campaign}',
  answeredComment: 'Answered {answer} to “{question}” in {campaign} · with a comment',
  changed: 'changed from {previous}',
  quote: '“{text}”',
  responded: 'Responded to the survey “{name}” · {answered} of {questions} answered',
  started: 'Started the survey “{name}” · {answered} of {questions} answered',
  editable: 'can change it until {date}',
  fromPoll: 'from a poll, {date}',
  fromSurvey: 'from a survey, {date}',
})
