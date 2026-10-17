import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './trigger.messages'

type Trigger = Schemas['AutomationTrigger']
type Bands = Schemas['ScoringBands']

/** A new "Score crosses…" trigger: rising into Hot, at most once in 90 days. */
export function scoreTriggerDefaults(bands?: Bands): Partial<Trigger> {
  return { direction: 'rises', threshold: bands?.hotFrom ?? 50, onceInDays: 90 }
}

/**
 * The "Score crosses…" trigger's settings (ScoringUses › Automation flow):
 * the direction, the score, and at most once per contact in so many days.
 */
export function ScoreTriggerFields({ t: tr, set, disabled, bands }: { t: Trigger; set: (p: Partial<Trigger>) => void; disabled: boolean; bands?: Bands }) {
  const t = useT()
  const { slug } = useWorkspace()
  const falls = tr.direction === 'falls'
  const threshold = tr.threshold ?? bands?.hotFrom ?? 50
  // Which band the threshold opens, when it's one of the thresholds.
  const band = bands ? (threshold === bands.hotFrom ? { name: t(m.bandHot), from: bands.hotFrom } : threshold === bands.warmFrom ? { name: t(m.bandWarm), from: bands.warmFrom } : undefined) : undefined
  const hint = band ? t(falls ? m.below : m.orMore, { band: band.name, from: band.from }) : t(falls ? m.belowPlain : m.orMorePlain)
  return (
    <div className="flex flex-col gap-2">
      <span className="k">{t(m.startsWhen)}</span>
      <div className="flex flex-wrap items-center gap-1.5">
        <select className="in" aria-label={t(m.direction)} disabled={disabled} value={falls ? 'falls' : 'rises'} onChange={(e) => set({ direction: e.target.value as Trigger['direction'] })}>
          <option value="rises">{t(m.rises)}</option>
          <option value="falls">{t(m.falls)}</option>
        </select>
        <input
          className="in num w-[70px]"
          type="number"
          min={1}
          max={100}
          aria-label={t(m.threshold)}
          disabled={disabled}
          value={threshold}
          onChange={(e) => set({ threshold: e.target.value === '' ? undefined : Math.max(1, Math.min(100, Math.round(Number(e.target.value)))) })}
        />
        <span className="k">{hint}</span>
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="k">{t(m.oncePerLabel)}</span>
        <input
          className="in num w-[70px]"
          type="number"
          min={1}
          max={365}
          aria-label={t(m.days)}
          disabled={disabled}
          value={tr.onceInDays ?? ''}
          onChange={(e) => set({ onceInDays: e.target.value === '' ? undefined : Math.max(1, Math.min(365, Math.round(Number(e.target.value)))) })}
        />
        <span className="k">{t(m.days)}</span>
      </div>
      <span className="k">{t(m.explain)}</span>
      {bands ? null : (
        <span className="k">
          {t(m.noScoring)}{' '}
          <Link to="/w/$workspace/audience/scoring" params={{ workspace: slug }}>
            {t(m.setUp)}
          </Link>
        </span>
      )}
      {disabled ? <span className="k">{t(m.needs)}</span> : null}
    </div>
  )
}
