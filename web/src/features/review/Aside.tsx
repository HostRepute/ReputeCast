import { useId, useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { clock, count } from '@/lib/format'
import {
  defaultScheduleAt,
  dollars,
  mailboxColor,
  minutes,
  parsePercent,
  percentText,
  providerColor,
  scheduleCaption,
  thresholdError,
  thresholdLimits,
  zoneLabel,
  zoneOptions,
  type Review,
  type Run,
  type Schedule,
  type ThresholdField,
  type Thresholds,
} from './logic'
import { whenText } from './parts'
import { EACH, TestLanguageSelect, testLanguages, type SentVersion } from '@/features/languages/review'

/** Audience: the lists, what's excluded and the final count. */
export function AudienceCard({ audience, children }: { audience: Run['estimates']['audience']; children?: ReactNode }) {
  return (
    <section className="card rv-card" aria-labelledby="rv-audience">
      <h2 className="h2" id="rv-audience">
        Audience
      </h2>
      {audience.rows.map((r, i) => (
        <div key={i} className={cn('rv-sum', r.kind !== 'base' && 's-minus')}>
          <span>{r.label}</span>
          <span className="n">{r.kind === 'minus' ? `−${count(r.count)}` : r.kind === 'plus' ? `+${count(r.count)}` : count(r.count)}</span>
        </div>
      ))}
      <div className="rv-sum s-final">
        <span className="font-semibold">Final recipients</span>
        <span className="n text-[20px]">{count(audience.final)}</span>
      </div>
      {children}
    </section>
  )
}

/** Route breakdown: recipients, cost and duration per sending provider. */
export function RouteCard({ route, startAt }: { route: Run['estimates']['route']; startAt?: string | null }) {
  const shares = route.shares
  const held = shares.filter((s) => !s.available)
  return (
    <section className={cn('card rv-card', held.length > 0 && 'border-bad-line')} aria-labelledby="rv-route">
      {held.length ? (
        <div className="flex items-center justify-between">
          <h2 className="h2" id="rv-route">
            Route breakdown
          </h2>
          <span className="badge bad">
            {held.length} {held.length === 1 ? 'route' : 'routes'} down
          </span>
        </div>
      ) : (
        <h2 className="h2" id="rv-route">
          Route breakdown
        </h2>
      )}
      <div className="flex flex-col gap-1.5">
        {shares.map((s) => {
          const color = s.available ? (providerColor[s.color] ?? providerColor.other) : '#B91C1C'
          return (
            <div key={s.providerId} className="contents">
              <div className="rv-sum">
                <span className="rv-who">
                  <span className="dot" style={{ color }} aria-hidden="true" />
                  {s.name} · {Number(s.percent.toFixed(1))}%{held.length ? ` · ${s.statusLabel}` : ''}
                </span>
                <span className="n">{count(s.recipients)}</span>
              </div>
              <div className="bar">
                <i className="grow-x" style={{ width: `${Math.min(100, s.percent)}%`, background: color }} />
              </div>
            </div>
          )
        })}
      </div>
      {held.length ? (
        <>
          {shares.map((s) =>
            s.available ? (
              <div key={s.providerId} className="rv-sum">
                <span>
                  {s.short} · {s.priceLabel} · {s.rateLabel}
                </span>
                <span className="n">
                  {dollars(s.cost)} · {minutes(s.minutes)}
                </span>
              </div>
            ) : (
              <div key={s.providerId} className="rv-sum text-bad-fg">
                <span>{s.short} · unavailable</span>
                <span className="n">held</span>
              </div>
            ),
          )}
          {route.move ? (
            <span className="k">
              Moving the {route.move.from} share to {route.move.to} adds {dollars(route.move.addCost)} and {route.move.addMinutes}{' '}
              {route.move.addMinutes === 1 ? 'minute' : 'minutes'}.
            </span>
          ) : null}
        </>
      ) : (
        <>
          <div className="tscroll">
            <table className="rv-tbl">
              <thead>
                <tr>
                  <th scope="col" className="k">
                    Sending provider
                  </th>
                  <th scope="col" className="k r">
                    Cost
                  </th>
                  <th scope="col" className="k r">
                    Duration
                  </th>
                </tr>
              </thead>
              <tbody>
                {shares.map((s) => (
                  <tr key={s.providerId}>
                    <td>
                      {s.short} · {s.priceLabel} · {s.rateLabel}
                    </td>
                    <td className="n r">{dollars(s.cost)}</td>
                    <td className="n r">{minutes(s.minutes)}</td>
                  </tr>
                ))}
                <tr className="t-total">
                  <td>{shares.length > 1 ? 'Total, routes run in parallel' : 'Total'}</td>
                  <td className="n r">{dollars(route.totalCost)}</td>
                  <td className="n r">{minutes(route.totalMinutes)}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <FinishLine total={route.totalMinutes} startAt={startAt} />
        </>
      )}
    </section>
  )
}

function FinishLine({ total, startAt }: { total: number; startAt?: string | null }) {
  const start = startAt ? new Date(startAt) : null
  if (!start || Number.isNaN(start.getTime())) {
    return <span className="k">Takes about {minutes(total)} once it starts.</span>
  }
  const end = new Date(start.getTime() + total * 60_000)
  return (
    <span className="k">
      Finishes about {clock(end)} if started at {clock(start)}.
    </span>
  )
}

/** Throttling per mailbox provider. */
export function ThrottlingCard({ throttling, note }: { throttling: Run['estimates']['throttling']; note: boolean }) {
  const slow = throttling.filter((t) => t.rate > 0).map((t) => t.name)
  return (
    <section className="card rv-card c-tight" aria-labelledby="rv-throttle">
      <h2 className="h2" id="rv-throttle">
        Throttling per mailbox provider
      </h2>
      {throttling.map((t) => (
        <div key={t.key} className="rv-sum">
          <span className="rv-who">
            <span className="dot" style={{ color: mailboxColor[t.key] }} aria-hidden="true" />
            {t.name} · {count(t.recipients)}
          </span>
          <span className="n">{t.rate > 0 ? `${Number(t.rate.toFixed(1))} /s` : 'full rate'}</span>
        </div>
      ))}
      {note && slow.length ? (
        <span className="k">
          {slow.length === 1 ? slow[0] : `${slow.slice(0, -1).join(', ')} and ${slow[slow.length - 1]}`} accept slower streams more reliably. Limits adapt if deferrals rise.
        </span>
      ) : null}
    </section>
  )
}

const fields: ThresholdField[] = ['bounce', 'complaint', 'mailboxComplaint', 'providerErrors']

/** Auto-pause thresholds: saved when a field loses focus or the scope changes. */
export function ThresholdsCard({
  thresholds,
  disabledReason,
  scope = true,
  onSave,
}: {
  thresholds: Thresholds
  disabledReason?: string
  scope?: boolean
  onSave: (t: Thresholds) => Promise<boolean>
}) {
  const [text, setText] = useState<Record<ThresholdField, string>>(() => textOf(thresholds))
  const [errors, setErrors] = useState<Partial<Record<ThresholdField, string>>>({})
  // Saved values reset the fields; refetches of the same values don't.
  const key = JSON.stringify(thresholds)
  const [seen, setSeen] = useState(key)
  if (seen !== key) {
    setSeen(key)
    setText(textOf(thresholds))
    setErrors({})
  }
  const commit = async (field: ThresholdField) => {
    const v = parsePercent(text[field])
    const err = thresholdError(field, v, thresholds)
    if (err) {
      setErrors((e) => ({ ...e, [field]: err }))
      return
    }
    setErrors((e) => ({ ...e, [field]: undefined }))
    if (v === null || v === thresholds[field]) {
      setText(textOf(thresholds))
      return
    }
    const ok = await onSave({ ...thresholds, [field]: v })
    if (!ok) setText(textOf(thresholds))
  }
  return (
    <section className="card rv-card" aria-labelledby="rv-thresholds">
      <h2 className="h2" id="rv-thresholds">
        Auto-pause thresholds
      </h2>
      <Grid cols={2} gap={10}>
        {fields.map((f) => (
          <Field key={f} label={thresholdLimits[f].label} error={errors[f]}>
            <input
              className="in"
              inputMode="decimal"
              value={text[f]}
              readOnly={Boolean(disabledReason)}
              title={disabledReason}
              onChange={(e) => setText((t) => ({ ...t, [f]: e.target.value }))}
              onBlur={() => void commit(f)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
              }}
            />
          </Field>
        ))}
      </Grid>
      {scope ? (
        <Field label="When a threshold trips">
          <Select
            value={thresholds.scope}
            disabled={Boolean(disabledReason)}
            title={disabledReason}
            onChange={(e) => void onSave({ ...thresholds, scope: e.target.value as Thresholds['scope'] })}
          >
            <option value="route">Pause only the affected route or mailbox provider</option>
            <option value="campaign">Pause the whole campaign</option>
          </Select>
        </Field>
      ) : null}
      {disabledReason ? <span className="k">{disabledReason}</span> : null}
    </section>
  )
}

function textOf(t: Thresholds): Record<ThresholdField, string> {
  return {
    bounce: percentText(t.bounce, thresholdLimits.bounce.digits),
    complaint: percentText(t.complaint, thresholdLimits.complaint.digits),
    mailboxComplaint: percentText(t.mailboxComplaint, thresholdLimits.mailboxComplaint.digits),
    providerErrors: percentText(t.providerErrors, thresholdLimits.providerErrors.digits),
  }
}

/** Scheduling: now, a date and time in a zone, or each recipient's best time. */
export function SchedulingCard({
  schedule,
  timezone,
  disabledReason,
  hint,
  onSave,
}: {
  schedule: Schedule
  timezone: string
  disabledReason?: string
  hint?: ReactNode
  onSave: (s: Schedule) => Promise<boolean>
}) {
  const name = useId()
  const [draft, setDraft] = useState<Schedule>(schedule)
  const key = JSON.stringify(schedule)
  const [seen, setSeen] = useState(key)
  if (seen !== key) {
    setSeen(key)
    setDraft(schedule)
  }
  const tz = draft.timezone || timezone
  const pick = (mode: Schedule['mode']) => {
    if (mode === 'now') {
      setDraft({ mode, timezone: tz })
      void onSave({ mode, timezone: tz })
      return
    }
    const at = draft.date && draft.time ? { date: draft.date, time: draft.time } : defaultScheduleAt(tz)
    const next = { mode, ...at, timezone: tz }
    setDraft(next)
    void onSave(next)
  }
  const change = (patch: Partial<Schedule>) => {
    const next = { ...draft, ...patch }
    setDraft(next)
    if (next.date && next.time && next.timezone) void onSave(next)
  }
  const off = Boolean(disabledReason)
  const when = (
    <span className="rv-when">
      <input className="in" type="date" value={draft.date ?? ''} aria-label="Date" disabled={off} onChange={(e) => change({ date: e.target.value })} />
      <input className="in" type="time" value={draft.time ?? ''} aria-label="Time" disabled={off} onChange={(e) => change({ time: e.target.value })} />
      <select className="in" aria-label="Timezone" value={tz} disabled={off} onChange={(e) => change({ timezone: e.target.value })}>
        {zoneOptions(tz).map((z) => (
          <option key={z} value={z}>
            {zoneLabel(z)}
          </option>
        ))}
      </select>
    </span>
  )
  const options: { mode: Schedule['mode']; title: string; caption: string }[] = [
    { mode: 'now', title: 'Send now', caption: 'Starts within a minute of confirming' },
    { mode: 'at', title: 'Schedule', caption: draft.mode === 'at' ? scheduleCaption({ ...draft, timezone: tz }) : 'A date and time in any time zone' },
    {
      mode: 'best_time',
      title: 'Best time per recipient',
      caption: `Each contact gets it at their usual open time, within 24 h of ${draft.mode === 'best_time' && draft.time ? draft.time : '09:00'}`,
    },
  ]
  return (
    <section className="card rv-card c-tight" aria-labelledby="rv-schedule">
      <h2 className="h2" id="rv-schedule">
        Scheduling
      </h2>
      {options.map((o) => (
        <label key={o.mode} className="rv-rad">
          <input type="radio" name={name} checked={draft.mode === o.mode} disabled={off} onChange={() => pick(o.mode)} />
          {/* As on the board, a long caption may take a few pixels from the radio. */}
          <span className={o.mode === 'at' || draft.mode === o.mode ? 'flex-1' : undefined}>
            <span className="font-semibold">{o.title}</span>
            <br />
            <span className="k">{o.caption}</span>
            {o.mode !== 'now' && draft.mode === o.mode ? when : null}
          </span>
        </label>
      ))}
      {disabledReason ? <span className="k">{disabledReason}</span> : hint ? <span className="k">{hint}</span> : null}
    </section>
  )
}

/**
 * Scheduling once it's settled: no choice to make here. The card is faded
 * as on the boards; its sentence is set in ink so it still reads at 4.5:1
 * through the fade (in ink-3 it would drop to 2.6:1).
 */
export function ScheduleNote({ text }: { text: string }) {
  return (
    <section className="card rv-card c-tight opacity-60" aria-labelledby="rv-schedule">
      <h2 className="h2" id="rv-schedule">
        Scheduling
      </h2>
      <span className="k text-ink">{text}</span>
    </section>
  )
}

/** Test sends: through the composer's test endpoint (System route). */
export function TestSendsCard({
  sends,
  me,
  toOthers,
  disabledReason,
  onSend,
  versions = [],
}: {
  sends: Schemas['ReviewTestSend'][]
  me: string
  toOthers: boolean
  disabledReason?: string
  onSend: (to: string, language?: string) => Promise<boolean>
  /** The email's language versions: the test sends one, or one of each. */
  versions?: SentVersion[]
}) {
  const id = useId()
  const [to, setTo] = useState(me)
  const [busy, setBusy] = useState(false)
  const [language, setLanguage] = useState(EACH)
  const send = async () => {
    const addr = (toOthers ? to : me).trim()
    if (!addr) return
    setBusy(true)
    await onSend(addr, versions.length > 1 ? language : undefined)
    setBusy(false)
  }
  return (
    <section className="card rv-card c-tight" aria-labelledby="rv-tests">
      <h2 className="h2" id="rv-tests">
        Test sends
      </h2>
      <div className="flex gap-2 max-md:flex-wrap">
        <label className="vh" htmlFor={id}>
          Test address
        </label>
        <input
          id={id}
          className="in min-w-0 flex-1"
          type="email"
          value={toOthers ? to : me}
          readOnly={!toOthers}
          onChange={(e) => setTo(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') void send()
          }}
        />
        {versions.length > 1 ? <TestLanguageSelect versions={versions} value={language} onChange={setLanguage} /> : null}
        {versions.length > 1 ? null : (
          <Button disabledReason={disabledReason} loading={busy} onClick={() => void send()}>
            Send test
          </Button>
        )}
      </div>
      {versions.length > 1 ? (
        <Button className="self-start" disabledReason={disabledReason} loading={busy} onClick={() => void send()}>
          Send test
        </Button>
      ) : null}
      {!toOthers ? <span className="k">Test emails go to your own address. Sending them to others needs campaigns:send.</span> : null}
      {sends.map((s, i) => (
        <div key={i} className="rv-sum">
          <span className="k">
            {s.to} · {whenText(s.at)}
            {testLanguages(s.languages, versions) ? ` · ${testLanguages(s.languages, versions)}` : s.provider ? ` · via ${s.provider}` : ''}
          </span>
          {s.delivered ? <Badge tone="ok">Delivered to {s.delivered}</Badge> : <Badge tone="neutral">Sent</Badge>}
        </div>
      ))}
    </section>
  )
}

/** Recipients the send goes to, for the CTA and the confirmation. */
export function finalCount(review: Review): number {
  return review.run?.estimates.audience.final ?? 0
}
