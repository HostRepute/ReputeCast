import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { m } from './trigger.messages'

type Trigger = Schemas['AutomationTrigger']

/** "Reached a milestone", the trigger's name in the palette and on the node. */
export function milestoneTriggerName(): string {
  return t(m.kind)
}

/** "3 referrals · The Acme Field Guide", or "any milestone". */
export function milestoneTriggerRef(tr: Trigger, milestones?: ReadonlyMap<string, string>): string {
  if (!tr.milestoneId) return t(m.anyMilestone)
  return milestones?.get(tr.milestoneId) ?? t(m.aMilestone)
}

/** "once per contact per milestone", under the node's title. */
export function milestoneTriggerDetail(): string {
  return t(m.onceDetail)
}
