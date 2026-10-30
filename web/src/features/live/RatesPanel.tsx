import { useId, useState } from 'react'
import { Dot } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { clock } from '@/lib/format'
import { m } from './messages'
import { mailboxColor, mailboxName, type MailboxKey, type Snapshot } from './model'

/** The rates a person can pick per mailbox provider, in messages per second; 0 is the full rate. */
export const RATE_LEVELS = [1, 2, 3, 4, 5, 6, 8, 10, 12, 0]

/** The stepper's − and + (the board's 14px strokes). */
function StepIcon({ plus }: { plus: boolean }) {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
      <path d={plus ? 'M8 3.5v9M3.5 8h9' : 'M3.5 8h9'} />
    </svg>
  )
}

/** A level's label: "5 /s" or "full rate". */
export function rateLabel(t: ReturnType<typeof useT>, v: number): string {
  return v > 0 ? t(m.ratePerSecond, { rate: v }) : t(m.rateFull)
}

/** The nearest level to a rate (a throttle of 2.5 /s shows as 2 /s). */
export function nearestLevel(v: number): number {
  if (v <= 0) return 0
  let best = RATE_LEVELS[0]!
  for (const l of RATE_LEVELS) if (l > 0 && Math.abs(l - v) < Math.abs(best - v)) best = l
  return best
}

/** The rates the send uses now, per mailbox provider still to send. */
export function currentRates(s: Snapshot): Record<string, number> {
  const out: Record<string, number> = {}
  for (const mb of s.mailboxes) if (mb.remaining > 0 || mb.held) out[mb.key] = mb.throttle ?? 0
  return out
}

/**
 * When everything is sent at given rates, in seconds from the snapshot: a
 * throttled mailbox provider takes its remaining mail over its rate; the
 * routes take theirs over their rates.
 */
export function finishSeconds(s: Snapshot, rates: Record<string, number>): number {
  let t = 0
  for (const p of s.providers) if (p.rate > 0 && p.remaining > 0 && p.routeStatus !== 'held' && p.routeStatus !== 'paused') t = Math.max(t, p.remaining / p.rate)
  for (const mb of s.mailboxes) {
    const v = rates[mb.key] ?? 0
    if (v > 0 && mb.remaining > 0 && !mb.held) t = Math.max(t, mb.remaining / v)
  }
  return t
}

/** One mailbox provider's time at a rate, in seconds (0: only the sending providers limit it). */
function mailboxSeconds(s: Snapshot, key: string, v: number): number {
  const mb = s.mailboxes.find((x) => x.key === key)
  return mb && v > 0 && mb.remaining > 0 ? mb.remaining / v : 0
}

/**
 * Adjust rates (LiveMonitorStates, review GS-13): a stepper per mailbox
 * provider for the rest of the send, on every route, with when each and
 * the whole send finish; Apply sends the change. During a canary the
 * rates are the main send's, which starts when the watch ends.
 */
export function RatesPanel({
  s,
  startsAt,
  onApply,
  onCancel,
  busy,
}: {
  s: Snapshot
  /** When the main send goes out (the canary's watch end); now without a canary. */
  startsAt?: string
  onApply: (rates: Record<string, number>, summary: string) => void
  onCancel: () => void
  busy?: boolean
}) {
  const t = useT()
  const id = useId()
  const applied = currentRates(s)
  const [draft, setDraft] = useState<Record<string, number>>(() => Object.fromEntries(Object.entries(applied).map(([k, v]) => [k, nearestLevel(v)])))
  const base = startsAt ? new Date(startsAt).getTime() : new Date(s.at).getTime()
  const at = (secs: number) => clock(new Date(base + secs * 1000).toISOString())
  const keys = Object.keys(applied).filter((k) => !s.mailboxes.find((mb) => mb.key === k)?.held)
  const changed = keys.filter((k) => (draft[k] ?? 0) !== nearestLevel(applied[k] ?? 0))
  const w0 = finishSeconds(s, applied)
  const w1 = finishSeconds(s, draft)
  const diff = Math.round((base + w1 * 1000) / 60000) - Math.round((base + w0 * 1000) / 60000)
  const canary = !!startsAt
  const whole = !changed.length
    ? t(canary ? m.ratesWholeSameCanary : m.ratesWholeSame, { time: at(w0) })
    : t(canary ? m.ratesWholeChangeCanary : m.ratesWholeChange, { from: at(w0), to: at(w1), diff: Math.abs(diff), way: diff > 0 ? 'later' : diff < 0 ? 'sooner' : 'same' })
  const step = (k: string, dir: -1 | 1) => {
    const i = RATE_LEVELS.indexOf(draft[k] ?? 0)
    const next = RATE_LEVELS[Math.min(RATE_LEVELS.length - 1, Math.max(0, i + dir))]!
    setDraft({ ...draft, [k]: next })
  }
  const apply = () => {
    if (!changed.length) return
    const rates = Object.fromEntries(changed.map((k) => [k, draft[k] ?? 0]))
    const changes = changed.map((k) => `${mailboxName(k)} ${rateLabel(t, draft[k] ?? 0)}`).join(', ')
    const summary = canary ? t(m.ratesAppliedCanary, { changes, at: at(0), time: at(w1) }) : t(m.ratesApplied, { changes, time: at(w1) })
    onApply(rates, summary)
  }
  return (
    <section className="lv-rates" id="lv-rates" aria-labelledby={`${id}-h`}>
      <div className="flex flex-col gap-0.5">
        <h3 className="h2 text-[14px]" id={`${id}-h`}>
          {t(m.ratesTitle)}
        </h3>
        <span className="k">{canary ? t(m.ratesCaptionCanary) : t(m.ratesCaption)}</span>
      </div>
      {s.mailboxes
        .filter((mb) => mb.held && mb.remaining > 0)
        .map((mb) => (
          <p key={mb.key} className="lv-rates-held">
            <b>{t(m.ratesHeld, { who: mailboxName(mb.key) })}</b> {t(m.ratesHeldNote)}
          </p>
        ))}
      <ul className="lv-rates-list">
        {keys.map((k) => {
          const v = draft[k] ?? 0
          const was = nearestLevel(applied[k] ?? 0)
          const i = RATE_LEVELS.indexOf(v)
          const t0 = mailboxSeconds(s, k, was)
          const t1 = mailboxSeconds(s, k, v)
          const effect = v === was ? '' : t1 && t0 ? t(m.rateEffectInstead, { to: at(t1), from: at(t0) }) : t1 ? t(m.rateEffect, { to: at(t1) }) : t(m.rateEffectFull)
          const name = mailboxName(k)
          return (
            <li key={k} className="lv-rate">
              <div className="lv-rate-name">
                <Dot color={mailboxColor[k as MailboxKey]} />
                <span>{name}</span>
              </div>
              <span className="k lv-rate-was">{v !== was ? t(m.rateWas, { rate: rateLabel(t, was) }) : t(canary ? m.ratePlanned : m.rateNow, { rate: rateLabel(t, was) })}</span>
              <div className="lv-stepper" role="group" aria-label={t(m.rateGroup, { who: name })}>
                <button type="button" onClick={() => step(k, -1)} disabled={i <= 0 || busy} aria-label={t(m.rateLower, { who: name })}>
                  <StepIcon plus={false} />
                </button>
                <output aria-live="polite">{rateLabel(t, v)}</output>
                <button type="button" onClick={() => step(k, 1)} disabled={v === 0 || busy} aria-label={t(m.rateRaise, { who: name })}>
                  <StepIcon plus />
                </button>
              </div>
              <span className="lv-rate-effect k">{effect}</span>
            </li>
          )
        })}
      </ul>
      <div className="lv-rates-f">
        <span className="text-[13px]">
          <span className="k">{t(m.ratesWhole)} </span>
          <span className="n">{whole}</span>
        </span>
        <div className="ml-auto flex flex-wrap gap-2">
          <Button onClick={onCancel} disabled={busy}>
            {t(m.cancelShort)}
          </Button>
          <Button variant="primary" onClick={apply} disabled={!changed.length} loading={busy}>
            {t(m.apply)}
          </Button>
        </div>
      </div>
    </section>
  )
}

