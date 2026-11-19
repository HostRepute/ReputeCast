import { useId, type ReactNode } from 'react'
import { Badge, type Tone } from '@/components/ui/Badge'
import { useT, type MessageDescriptor } from '@/i18n'
import { m } from './messages'

/**
 * A state of the assistant (AssistantStates): a badge with the state's
 * word, what happened and what to do, and its action. Each says who can fix
 * it when the viewer can't (the action's disabled reason names them).
 */
export function StateCard({ badge, tone, title, children, actions }: { badge: string; tone: Tone; title: string; children: ReactNode; actions?: ReactNode }) {
  const id = useId()
  return (
    <section className="asst-state" aria-labelledby={id}>
      <Badge tone={tone} dot>
        {badge}
      </Badge>
      <h3 id={id}>{title}</h3>
      <div className="asst-state-body">{children}</div>
      {actions ? <div className="asst-state-actions">{actions}</div> : null}
    </section>
  )
}

/** What a failed turn's reason means (internal/ai/assistant AIThreadTurn.reason). */
export type TurnProblem =
  | 'threadCap'
  | 'monthly'
  | 'personalCap'
  | 'key'
  | 'rate'
  | 'provider'
  | 'refusal'
  | 'lost'
  | 'off'
  | 'modelGone'
  | 'interrupted'
  | 'expired'
  | 'failed'

export function problemOf(reason: string | undefined): TurnProblem {
  switch (reason) {
    case 'thread_cap':
      return 'threadCap'
    case 'cap_reached':
    case 'reservation_cap':
      return 'monthly'
    case 'personal_cap_reached':
      return 'personalCap'
    case 'key_rejected':
      return 'key'
    case 'rate_limited':
      return 'rate'
    case 'provider_error':
    case 'timeout':
    case 'too_large':
      return 'provider'
    case 'refused':
      return 'refusal'
    case 'access_lost':
      return 'lost'
    case 'assistant_off':
      return 'off'
    case 'model_gone':
    case 'not_configured':
    case 'tools_unchecked':
    case 'personal_keys_off':
    case 'account_disconnected':
      return 'modelGone'
    case 'interrupted':
    case 'not_started':
      return 'interrupted'
    case 'expired':
    case 'assistant_thread_expired':
      return 'expired'
    case 'assistant_model_gone':
      return 'modelGone'
    default:
      return 'failed'
  }
}

/** The title, badge and tone of each problem, as the boards word them. */
export const problemCopy: Record<TurnProblem, { title: MessageDescriptor; badge: MessageDescriptor; tone: Tone }> = {
  threadCap: { title: m.threadCapTitle, badge: m.threadCapBadge, tone: 'warn' },
  monthly: { title: m.monthlyTitle, badge: m.monthlyBadge, tone: 'warn' },
  personalCap: { title: m.personalCapTitle, badge: m.monthlyBadge, tone: 'warn' },
  key: { title: m.keyTitle, badge: m.keyBadge, tone: 'bad' },
  rate: { title: m.rateTitle, badge: m.rateBadge, tone: 'warn' },
  provider: { title: m.providerTitle, badge: m.providerBadge, tone: 'bad' },
  refusal: { title: m.refusalTitle, badge: m.refusalBadge, tone: 'bad' },
  lost: { title: m.lostTitle, badge: m.lostBadge, tone: 'bad' },
  off: { title: m.offTitle, badge: m.offBadge, tone: 'neutral' },
  modelGone: { title: m.modelGoneTitle, badge: m.failedBadge, tone: 'bad' },
  interrupted: { title: m.interruptedTitle, badge: m.failedBadge, tone: 'bad' },
  expired: { title: m.expiredTitle, badge: m.expiredBadge, tone: 'neutral' },
  failed: { title: m.failedTitle, badge: m.failedBadge, tone: 'bad' },
}

/** A problem's title, with its provider or model filled in. */
export function useProblemTitle() {
  const t = useT()
  return (p: TurnProblem, values: { provider?: string; model?: string }) => t(problemCopy[p].title, { provider: values.provider ?? '', model: values.model ?? '' })
}
