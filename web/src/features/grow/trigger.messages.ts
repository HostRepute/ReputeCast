import { defineMessages } from '@/i18n'

/** The automation trigger "Reached a milestone" (Grow · referral milestones). */
export const m = defineMessages('grow.trigger', {
  kind: 'Reached a milestone',
  anyMilestone: 'any milestone',
  aMilestone: 'a milestone',
  program: 'Program',
  anyProgram: 'Any program',
  milestone: 'Milestone',
  anyMilestoneOption: 'Any milestone',
  programMilestone: '{list} · {milestone}',
  once: 'Once per contact per milestone',
  onceDetail: 'once per contact per milestone',
  explain: 'Starts when a referrer reaches the milestone, whether or not the milestone sends its own email. The milestone and reward tokens work in this flow’s emails.',
  noMilestones: 'No referral program has milestones yet, so nothing starts this flow.',
  setUp: 'Set up referrals',
  needs: 'Choosing this trigger needs automations:edit.',
})
