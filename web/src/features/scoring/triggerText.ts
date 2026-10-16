import type { Schemas } from '@/api/client'
import { formatList, t } from '@/i18n'
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

/** The "is one of" field comparison's name in the condition editor. */
export function isOneOfLabel(): string {
  return t(m.isOneOf)
}

/** "plan is team or business": a field compared with comma-separated values. */
export function isOneOfPhrase(field: string, values: string): string {
  const vs = values
    .split(',')
    .map((v) => v.trim())
    .filter(Boolean)
  return t(m.isOneOfPhrase, { field, values: formatList(vs, 'disjunction') })
}
