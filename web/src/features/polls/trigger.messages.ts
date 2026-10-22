import { defineMessages } from '@/i18n'

/**
 * The automation trigger "Answered a poll or survey" (design expansion,
 * X11; brief §6): which poll or survey, which answers, and whether a
 * changed answer starts the flow again.
 */
export const m = defineMessages('pollTrigger', {
  source: 'Poll or survey',
  pick: 'Pick a poll or survey',
  polls: 'Polls',
  surveys: 'Surveys',
  pollIn: '{question} · {campaign}',
  answer: 'Answer',
  any: 'Any answer',
  is: 'Is one of',
  isNot: 'Isn’t one of',
  answers: 'Answers',
  onChange: 'Start again when the answer changes',
  note: 'Test answers and answers from people who aren’t contacts never start it.',
  surveyNote: 'A survey starts the flow when someone submits it.',
  loading: 'Loading the polls and surveys…',
})
