import { useId, useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { formatList, getI18n, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, count, decimal } from '@/lib/format'
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
import { m } from './messages'
import { whenText } from './parts'
import { EACH, TestLanguageSelect, testLanguages, type SentVersion } from '@/features/languages/review'

/**
 * Audience: the lists, what's excluded and the final count. The sunset
 * policy's line offers "Include them" (onInclude) when the person can edit
 * the campaign (SR-11).
 */
export function AudienceCard({
  audience,
  onInclude,
  including,
  children,
}: {
  audience: Run['estimates']['audience']
  onInclude?: () => void
  including?: boolean
  children?: ReactNode
}) {
  const t = useT()
  return (
    <section className="card rv-card" aria-labelledby="rv-audience">
      <h2 className="h2" id="rv-audience">
        {t(m.audience)}
      </h2>
      {audience.rows.map((r, i) => (
        <div key={i} className={cn('rv-sum', r.kind !== 'base' && 's-minus', r.action && onInclude && 'items-start')}>
          {r.action === 'include_unengaged' && onInclude ? (
            <span className="flex flex-col items-start gap-0.5">
              <span>{r.label}</span>
              <button
                type="button"
                className="rv-lnk"
                aria-label={t(m.includeThemLabel, { count: r.count })}
                aria-busy={including || undefined}
                disabled={including}
                onClick={onInclude}
              >
                {t(m.includeThem)}
              </button>
            </span>
          ) : (
            <span>{r.label}</span>
          )}
          <span className="n">{r.kind === 'minus' ? `−${count(r.count)}` : r.kind === 'plus' ? `+${count(r.count)}` : count(r.count)}</span>
        </div>
      ))}
      <div className="rv-sum s-final">
        <span className="font-semibold">{t(m.finalRecipients)}</span>
        <span className="n text-[20px]">{count(audience.final)}</span>
      </div>
      {children}
    </section>
  )
}

/** Route breakdown: recipients, cost and duration per sending provider. */
export function RouteCard({ route, startAt }: { route: Run['estimates']['route']; startAt?: string | null }) {
  const t = useT()
  const shares = route.shares
  const held = shares.filter((s) => !s.available)
  return (
    <section className={cn('card rv-card', held.length > 0 && 'border-bad-line')} aria-labelledby="rv-route">
      {held.length ? (
        <div className="flex items-center justify-between">
          <h2 className="h2" id="rv-route">
            {t(m.routeBreakdown)}
          </h2>
          <span className="badge bad">{t(m.routesDown, { count: held.length })}</span>
        </div>
      ) : (
        <h2 className="h2" id="rv-route">
          {t(m.routeBreakdown)}
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
                  {s.name} · {decimal(Number(s.percent.toFixed(1)), Number.isInteger(Number(s.percent.toFixed(1))) ? 0 : 1)}%{held.length ? ` · ${s.statusLabel}` : ''}
                </span>
                <span className="n">{count(s.recipients)}</span>
              </div>
              <div className="bar">
                <i className="grow-x" style={{ width: `${Math.min(100, s.percent)}%`, background: color }} />
              </div>
            </div>
          )
        })}
        {shares.map((s) => {
          if (!s.available || !s.warmupCap) return null
          return s.days && s.days > 1 ? (
            <span key={`cap-${s.providerId}`} className="k">
              {t(m.warmupOver, { name: s.short, count: count(s.recipients), cap: count(s.warmupCap), days: s.days })}
            </span>
          ) : s.recipients <= s.warmupCap ? (
            <span key={`cap-${s.providerId}`} className="k">
              {t(m.warmupWithin, { name: s.short, count: count(s.recipients), cap: count(s.warmupCap) })}
            </span>
          ) : null
        })}
        {shares.map((s) =>
          s.available && s.overQuota ? (
            <span key={`quota-${s.providerId}`} className="k">
              {t(m.quotaOver, { name: s.short, count: count(s.overQuota) })}
            </span>
          ) : null,
        )}
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
                <span>{t(m.unavailable, { name: s.short })}</span>
                <span className="n">{t(m.held)}</span>
              </div>
            ),
          )}
          {route.move ? (
            <span className="k">{t(m.moveShare, { from: route.move.from, to: route.move.to, cost: dollars(route.move.addCost), minutes: route.move.addMinutes })}</span>
          ) : null}
        </>
      ) : (
        <>
          <div className="tscroll">
            <table className="rv-tbl">
              <thead>
                <tr>
                  <th scope="col" className="k">
                    {t(m.sendingProvider)}
                  </th>
                  <th scope="col" className="k r">
                    {t(m.cost)}
                  </th>
                  <th scope="col" className="k r">
                    {t(m.duration)}
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
                    <td className="n r">{s.days && s.days > 1 ? t(m.overDays, { duration: minutes(s.minutes), days: s.days }) : minutes(s.minutes)}</td>
                  </tr>
                ))}
                <tr className="t-total">
                  <td>{shares.length > 1 ? t(m.totalParallel) : t(m.total)}</td>
                  <td className="n r">{dollars(route.totalCost)}</td>
                  <td className="n r">{minutes(route.totalMinutes)}</td>
                </tr>
              </tbody>
            </table>
          </div>
          {route.timeline ? <TimelineInset timeline={route.timeline} /> : <FinishLine total={route.totalMinutes} startAt={startAt} />}
        </>
      )}
    </section>
  )
}

/** When the send's parts go out, as the sending engine plays it out: the
 * A/B sample, the winner and the rest, and what the estimate includes. */
function TimelineInset({ timeline }: { timeline: NonNullable<Run['estimates']['route']['timeline']> }) {
  const t = useT()
  const ab = timeline.ab
  return (
    <div className="rv-inset">
      {ab ? (
        <>
          <span className="text-[13px] font-semibold">
            {t(m.abLine, { kind: ab.kind, percent: ab.testPercent, variants: ab.variants, perVariant: count(ab.perVariant) })}
          </span>
          <span className="n text-[13px]">
            {ab.decideAt && !timeline.open
              ? t(m.abTimes, { from: clock(timeline.start), to: clock(ab.sampleEnd), decide: clock(ab.decideAt), end: clock(timeline.end) })
              : t(m.abTimesManual, { from: clock(timeline.start), to: clock(ab.sampleEnd) })}
          </span>
        </>
      ) : timeline.bestTime ? (
        <span className="n text-[13px]">{t(m.bestTimeLine, { end: clock(timeline.end) })}</span>
      ) : (
        <span className="n text-[13px]">{t(m.finishesAbout, { end: clock(timeline.end), start: clock(timeline.start) })}</span>
      )}
      <span className="k">{timeline.basisLabel}</span>
    </div>
  )
}

function FinishLine({ total, startAt }: { total: number; startAt?: string | null }) {
  const t = useT()
  const start = startAt ? new Date(startAt) : null
  if (!start || Number.isNaN(start.getTime())) {
    return <span className="k">{t(m.takesAbout, { duration: minutes(total) })}</span>
  }
  const end = new Date(start.getTime() + total * 60_000)
  return <span className="k">{t(m.finishesAbout, { end: clock(end), start: clock(start) })}</span>
}

/** Throttling per mailbox provider. */
export function ThrottlingCard({ throttling, note }: { throttling: Run['estimates']['throttling']; note: boolean }) {
  const t = useT()
  const slow = throttling.filter((x) => x.rate > 0).map((x) => x.name)
  const rate = (r: number) => decimal(Number(r.toFixed(1)), Number.isInteger(Number(r.toFixed(1))) ? 0 : 1)
  return (
    <section className="card rv-card c-tight" aria-labelledby="rv-throttle">
      <h2 className="h2" id="rv-throttle">
        {t(m.throttling)}
      </h2>
      {throttling.map((x) => (
        <div key={x.key} className="rv-sum">
          <span className="rv-who">
            <span className="dot" style={{ color: mailboxColor[x.key] }} aria-hidden="true" />
            {x.name} · {count(x.recipients)}
          </span>
          <span className="n">{x.rate > 0 ? t(m.perSecond, { rate: rate(x.rate) }) : t(m.fullRate)}</span>
        </div>
      ))}
      {note && slow.length ? <span className="k">{t(m.slowerStreams, { providers: formatList(slow) })}</span> : null}
    </section>
  )
}

const fields: ThresholdField[] = ['bounce', 'complaint', 'mailboxComplaint', 'providerErrors']

/**
 * Auto-pause (SR-13): the thresholds this send pauses at, as one line
 * (the workspace's defaults, or this send's own), and the fields behind
 * "Change for this send". A field saves when it loses focus.
 */
export function ThresholdsCard({
  thresholds,
  own,
  campaign,
  disabledReason,
  scope = true,
  onSave,
}: {
  thresholds: Thresholds
  /** The campaign has thresholds of its own (not the workspace's defaults). */
  own?: boolean
  campaign: string
  disabledReason?: string
  scope?: boolean
  onSave: (t: Thresholds) => Promise<boolean>
}) {
  const t = useT()
  // A choice whose label can pass about 32 characters is a radio card,
  // not a select, outside English (StressReviewVi, rule 1).
  const cards = getI18n().locale !== 'en'
  const [text, setText] = useState<Record<ThresholdField, string>>(() => textOf(thresholds))
  const [errors, setErrors] = useState<Partial<Record<ThresholdField, string>>>({})
  // Open while a field has an error, so it stays in view.
  const [open, setOpen] = useState(false)
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
  const name = useId()
  return (
    <section className="card rv-card" aria-labelledby="rv-thresholds">
      <h2 className="h2" id="rv-thresholds">
        {t(m.autoPause)}
      </h2>
      <span className="text-[13px]">
        {t(m.autoPauseSummary, {
          source: own ? 'campaign' : 'workspace',
          bounce: percentText(thresholds.bounce, 1).replace(' ', ''),
          complaint: percentText(thresholds.complaint, 2).replace(' ', ''),
          mailbox: percentText(thresholds.mailboxComplaint, 2).replace(' ', ''),
          errors: percentText(thresholds.providerErrors, 0).replace(' ', ''),
        })}
      </span>
      <span className="k">{t(m.autoPauseScope, { scope: thresholds.scope })}</span>
      <details className="rv-disc" open={open || Object.values(errors).some(Boolean)} onToggle={(e) => setOpen((e.target as HTMLDetailsElement).open)}>
        <summary>
          <span>{t(m.changeForThisSend)}</span>
          <svg className="rv-chev" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <path d="m4 6 4 4 4-4" />
          </svg>
        </summary>
        <div className="flex flex-col gap-2.5 pt-2.5">
          <Grid cols={2} gap={10}>
            {fields.map((f) => (
              <Field key={f} label={thresholdLimits[f].label} error={errors[f]}>
                <input
                  className="in"
                  inputMode="decimal"
                  value={text[f]}
                  readOnly={Boolean(disabledReason)}
                  title={disabledReason}
                  onChange={(e) => setText((v) => ({ ...v, [f]: e.target.value }))}
                  onBlur={() => void commit(f)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
                  }}
                />
              </Field>
            ))}
          </Grid>
          {scope && cards ? (
            <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
              <legend className="mb-1.5 p-0 text-[12px] font-semibold text-ink-2">{t(m.whenTrips)}</legend>
              {(['route', 'campaign'] as const).map((v) => (
                <label key={v} className="rv-rad">
                  <input
                    type="radio"
                    name={name}
                    checked={thresholds.scope === v}
                    disabled={Boolean(disabledReason)}
                    onChange={() => void onSave({ ...thresholds, scope: v })}
                  />
                  <span>{v === 'route' ? t(m.pauseRoute) : t(m.pauseCampaign)}</span>
                </label>
              ))}
            </fieldset>
          ) : scope ? (
            <Field label={t(m.whenTrips)}>
              <Select
                value={thresholds.scope}
                disabled={Boolean(disabledReason)}
                title={disabledReason}
                onChange={(e) => void onSave({ ...thresholds, scope: e.target.value as Thresholds['scope'] })}
              >
                <option value="route">{t(m.pauseRoute)}</option>
                <option value="campaign">{t(m.pauseCampaign)}</option>
              </Select>
            </Field>
          ) : null}
          <span className="k">{disabledReason ?? t(m.changeForThisSendNote, { name: campaign })}</span>
        </div>
      </details>
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

/**
 * Scheduling: now, a date and time in a zone, or each recipient's best
 * time, as boxes that frame the choice (SendReadiness, SR-13). The main
 * action says what the choice does ("Schedule for Tue 14 Oct, 09:00 ET").
 */
export function SchedulingCard({
  schedule,
  timezone,
  abHours,
  disabledReason,
  hint,
  onSave,
}: {
  schedule: Schedule
  timezone: string
  /** The A/B test's wait: sending now says when its winner is picked. */
  abHours?: number
  disabledReason?: string
  hint?: ReactNode
  onSave: (s: Schedule) => Promise<boolean>
}) {
  const t = useT()
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
    <div className="rv-dt">
      <input className="in" type="date" value={draft.date ?? ''} aria-label={t(m.sendDate)} disabled={off} onChange={(e) => change({ date: e.target.value })} />
      <input className="in" type="time" value={draft.time ?? ''} aria-label={t(m.sendTime)} disabled={off} onChange={(e) => change({ time: e.target.value })} />
      <select className="in" aria-label={t(m.timezone)} value={tz} disabled={off} onChange={(e) => change({ timezone: e.target.value })}>
        {zoneOptions(tz).map((z) => (
          <option key={z} value={z}>
            {zoneLabel(z)}
          </option>
        ))}
      </select>
    </div>
  )
  const at = draft.date && draft.time ? scheduleCaption({ ...draft, timezone: tz }) : ''
  const options: { mode: Schedule['mode']; title: string; caption: string }[] = [
    { mode: 'now', title: t(m.modeNow), caption: abHours ? t(m.modeNowCaptionAb, { hours: abHours }) : t(m.modeNowCaption) },
    { mode: 'at', title: t(m.modeAt), caption: draft.mode === 'at' && at ? at : t(m.modeAtCaption) },
    {
      mode: 'best_time',
      title: t(m.modeBest),
      caption: t(m.modeBestCaption, { time: draft.mode === 'best_time' && at ? at.replace(/ · [^·]*$/, '').replace(' · ', ', ') : '09:00' }),
    },
  ]
  return (
    <section className="card rv-card c-tight" aria-labelledby="rv-schedule">
      <h2 className="h2" id="rv-schedule">
        {t(m.scheduling)}
      </h2>
      {options.map((o) => (
        <div key={o.mode} className={cn('rv-radbox', draft.mode === o.mode && 'on')}>
          <label className="rv-radl">
            <input type="radio" name={name} checked={draft.mode === o.mode} disabled={off} onChange={() => pick(o.mode)} />
            <span className="rv-radt">
              {o.title}
              <br />
              <span className="k">{o.caption}</span>
            </span>
          </label>
          {o.mode !== 'now' && draft.mode === o.mode ? when : null}
        </div>
      ))}
      {disabledReason ? <span className="k">{disabledReason}</span> : hint ? <span className="k">{hint}</span> : null}
    </section>
  )
}

/**
 * Scheduling once it's settled, or locked until the blocks are fixed: no
 * choice to make here. The card is faded as on the boards; its sentence is
 * set in ink so it still reads at 4.5:1 through the fade (in ink-3 it
 * would drop to 2.6:1).
 */
export function ScheduleNote({ text, locked }: { text: string; locked?: boolean }) {
  const t = useT()
  if (locked) {
    // Locked until the blocks are fixed: muted by its colours, not faded,
    // so the badge keeps its contrast.
    return (
      <section className="card rv-card c-tight" aria-labelledby="rv-schedule">
        <h2 className="h2 flex items-center justify-between gap-2 text-ink-3" id="rv-schedule">
          {t(m.scheduling)}
          <Badge tone="neutral">{t(m.locked)}</Badge>
        </h2>
        <span className="k">{text}</span>
      </section>
    )
  }
  return (
    <section className="card rv-card c-tight opacity-60" aria-labelledby="rv-schedule">
      <h2 className="h2" id="rv-schedule">
        {t(m.scheduling)}
      </h2>
      <span className="k text-ink">{text}</span>
    </section>
  )
}

/**
 * Test sends: through the composer's test endpoint (System route). They go
 * out while the checks block too (SR-13): a test checks the fixes before
 * the checks clear (note).
 */
export function TestSendsCard({
  sends,
  me,
  toOthers,
  helpers = [],
  note,
  disabledReason,
  onSend,
  versions = [],
}: {
  sends: Schemas['ReviewTestSend'][]
  me: string
  toOthers: boolean
  /** Who can send tests to others, when the reader can't. */
  helpers?: string[]
  note?: string
  disabledReason?: string
  onSend: (to: string, language?: string) => Promise<boolean>
  /** The email's language versions: the test sends one, or one of each. */
  versions?: SentVersion[]
}) {
  const t = useT()
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
        {t(m.testSends)}
      </h2>
      {note ? <span className="k">{note}</span> : null}
      <div className="flex gap-2 max-md:flex-wrap">
        <label className="vh" htmlFor={id}>
          {t(m.testAddress)}
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
            {t(m.sendTestButton)}
          </Button>
        )}
      </div>
      {versions.length > 1 ? (
        <Button className="self-start" disabledReason={disabledReason} loading={busy} onClick={() => void send()}>
          {t(m.sendTestButton)}
        </Button>
      ) : null}
      {!toOthers ? <span className="k">{t(m.testOwnOnly, { names: helpers.length ? formatList(helpers, 'disjunction') : 'none' })}</span> : null}
      {sends.map((s, i) => (
        <div key={i} className="rv-sum">
          <span className="k">
            {s.to} · {whenText(s.at)}
            {testLanguages(s.languages, versions) ? ` · ${testLanguages(s.languages, versions)}` : s.provider ? ` · ${t(m.via, { provider: s.provider })}` : ''}
          </span>
          {s.delivered ? <Badge tone="ok">{t(m.deliveredTo, { mailbox: s.delivered })}</Badge> : <Badge tone="neutral">{t(m.sent)}</Badge>}
        </div>
      ))}
    </section>
  )
}

/** Recipients the send goes to, for the CTA and the confirmation. */
export function finalCount(review: Review): number {
  return review.run?.estimates.audience.final ?? 0
}
