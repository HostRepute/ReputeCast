import { defineMessages } from '@/i18n'

/** The automation trigger "Score crosses…" (ScoringUses › Automation flow). */
export const m = defineMessages('scoring.trigger', {
  kind: 'Score crosses…',
  risesTo: 'Rises to {threshold} or more',
  fallsBelow: 'Falls below {threshold}',
  oncePer: 'At most once per contact in {days, plural, one {# day} other {# days}}',
  startsWhen: 'Starts when the score',
  rises: 'Rises to',
  falls: 'Falls below',
  direction: 'Direction',
  threshold: 'Score',
  orMore: 'or more · {band} starts at {from}',
  below: 'and below · {band} starts at {from}',
  orMorePlain: 'or more',
  belowPlain: 'and below',
  bandHot: 'Hot',
  bandWarm: 'Warm',
  oncePerLabel: 'At most once per contact in',
  days: 'days',
  explain:
    'Starts on changes from activity and decay. Contacts who cross only because a scoring rule changed are asked about when the rule is saved; the default is not to start it.',
  needs: 'Choosing this trigger needs automations:edit.',
  noScoring: 'Scoring has no rules yet, so no score crosses anything.',
  setUp: 'Set up scoring',
  isOneOf: 'is one of',
  isOneOfPhrase: '{field} is {values}',
})
