import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { m } from './trigger.messages'

type Trigger = Schemas['AutomationTrigger']

/** "Score crosses…", the trigger's name in the palette and on the node. */
export function scoreTriggerName(): string {
  return t(m.kind)
}

/** "Rises to 50 or more", "Falls below 20". */
export function scoreTriggerRef(tr: Trigger): string {
  const threshold = tr.threshold ?? 0
  return tr.direction === 'falls' ? t(m.fallsBelow, { threshold }) : t(m.risesTo, { threshold })
}

/** "At most once per contact in 90 days", under the node's title. */
export function scoreTriggerDetail(tr: Trigger): string {
  return tr.onceInDays ? t(m.oncePer, { days: tr.onceInDays }) : ''
}
