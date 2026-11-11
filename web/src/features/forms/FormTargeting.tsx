import { useId, useState } from 'react'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Grid, Row } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './builder.messages'
import { cloneTargeting, fromLocalInput, listOf, scheduleSummary, targetingSummary, toLocalInput, type Targeting } from './targeting'

type Rule = Targeting['pages']['rules'][number]
type Device = Targeting['devices'][number]

const maxRules = 20
const devices: Device[] = ['desktop', 'tablet', 'mobile']

/**
 * "Where and when" (embedded and popup forms) or "When" (hosted): the
 * targeting in one line and the dialog that edits it. Targeting is the
 * same in every variant of an A/B test.
 */
export function WhereWhenSection({
  hosted,
  targeting,
  geoip,
  locked,
  reason,
  error,
  onChange,
}: {
  hosted: boolean
  targeting: Targeting
  geoip: boolean
  /** A variant of an A/B test is selected: targeting is the form's. */
  locked: boolean
  /** Why the person can't edit the form, if they can't. */
  reason?: string
  error?: string
  onChange: (t: Targeting) => void
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  return (
    <div className="fb-sec last">
      <span className="k">{t(hosted ? m.when : m.whereWhen)}</span>
      {locked ? <span className="k">{t(m.sameInEvery)}</span> : null}
      <div className="flex items-center gap-2.5">
        <span className="min-w-0 flex-auto text-[13px]">{hosted ? scheduleSummary(targeting.schedule) : targetingSummary(targeting)}</span>
        <Button size="sm" aria-label={t(hosted ? m.editWhen : m.editWhereWhen)} disabledReason={reason ?? (locked ? t(m.sameInEvery) : undefined)} onClick={() => setOpen(true)}>
          {t(common.edit)}
        </Button>
      </div>
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
      {open ? <TargetingDialog hosted={hosted} targeting={targeting} geoip={geoip} onClose={() => setOpen(false)} onApply={onChange} /> : null}
    </div>
  )
}

/** The dialog's fields: lists as typed, dates as datetime-local values. */
interface Fields {
  pages: Targeting['pages']['mode']
  rules: Rule[]
  utmSource: string
  utmMedium: string
  utmCampaign: string
  referrer: Targeting['sources']['referrer']
  devices: Device[]
  countries: Targeting['countries']['mode']
  codes: string
  visitors: Targeting['visitors']
  starts: string
  ends: string
}

function fieldsOf(x: Targeting): Fields {
  return {
    pages: x.pages.mode,
    rules: x.pages.rules.map((r) => ({ ...r })),
    utmSource: x.sources.utmSource.join(', '),
    utmMedium: x.sources.utmMedium.join(', '),
    utmCampaign: x.sources.utmCampaign.join(', '),
    referrer: x.sources.referrer,
    devices: [...x.devices],
    countries: x.countries.mode,
    codes: x.countries.codes.join(', '),
    visitors: x.visitors,
    starts: toLocalInput(x.schedule.startsAt),
    ends: toLocalInput(x.schedule.endsAt),
  }
}

/**
 * "Where and when" (FormsStates): pages, sources, devices, countries,
 * visitors and the schedule; a hosted form's "When" has the schedule
 * only. Apply changes the draft; Save & publish saves it.
 */
function TargetingDialog({ hosted, targeting, geoip, onClose, onApply }: { hosted: boolean; targeting: Targeting; geoip: boolean; onClose: () => void; onApply: (t: Targeting) => void }) {
  const t = useT()
  const id = useId()
  const [initial] = useState(() => fieldsOf(targeting))
  const [f, setF] = useState(initial)
  const set = (patch: Partial<Fields>) => setF((x) => ({ ...x, ...patch }))
  const setRule = (index: number, patch: Partial<Rule>) => set({ rules: f.rules.map((r, i) => (i === index ? { ...r, ...patch } : r)) })
  const apply = () => {
    const next = cloneTargeting(targeting)
    // An unchanged date keeps the saved moment as the server wrote it.
    next.schedule = {
      startsAt: f.starts === initial.starts ? targeting.schedule.startsAt : fromLocalInput(f.starts),
      endsAt: f.ends === initial.ends ? targeting.schedule.endsAt : fromLocalInput(f.ends),
    }
    if (!hosted) {
      next.pages = { mode: f.pages, rules: f.pages === 'all' ? [] : f.rules.map((r) => ({ op: r.op, value: r.value.trim() })) }
      next.sources = { utmSource: listOf(f.utmSource), utmMedium: listOf(f.utmMedium), utmCampaign: listOf(f.utmCampaign), referrer: f.referrer }
      next.devices = devices.filter((d) => f.devices.includes(d))
      if (geoip) next.countries = { mode: f.countries, codes: f.countries === 'all' ? [] : listOf(f.codes) }
      next.visitors = f.visitors
    }
    onApply(next)
    onClose()
  }
  const schedule = (
    <>
      <Row wrap gap={8}>
        <label className="lbl flex-[1_1_180px]">
          {t(m.starts)}
          <input className="in" type="datetime-local" value={f.starts} onChange={(e) => set({ starts: e.target.value })} />
        </label>
        <label className="lbl flex-[1_1_180px]">
          {t(m.ends)}
          <input className="in" type="datetime-local" value={f.ends} onChange={(e) => set({ ends: e.target.value })} />
        </label>
      </Row>
      <span className="k">{t(hosted ? m.scheduleHostedHelp : m.scheduleHelp)}</span>
    </>
  )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(hosted ? m.when : m.whereWhen)}
      size={hosted ? 'sm' : 'md'}
      className={hosted ? undefined : 'max-w-[560px]'}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={apply}>
            {t(m.apply)}
          </Button>
        </>
      }
    >
      {hosted ? (
        schedule
      ) : (
        <div className="flex flex-col">
          <div role="group" aria-labelledby={`${id}-pg`} className="flex flex-col gap-2 pb-3.5">
            <h3 id={`${id}-pg`} className="m-0 text-[13.5px] font-semibold">
              {t(m.pagesGroup)}
            </h3>
            <select
              className="in"
              aria-label={t(m.pagesGroup)}
              value={f.pages}
              onChange={(e) => {
                const pages = e.target.value as Fields['pages']
                // Only and Except start with a rule to fill in.
                set({ pages, rules: pages !== 'all' && f.rules.length === 0 ? [{ op: 'contains', value: '' }] : f.rules })
              }}
            >
              <option value="all">{t(m.pagesAll)}</option>
              <option value="only">{t(m.pagesOnly)}</option>
              <option value="except">{t(m.pagesExcept)}</option>
            </select>
            {f.pages !== 'all' ? (
              <>
                {f.rules.map((r, i) => (
                  <Row key={i} gap={8}>
                    <select className="in w-[130px] flex-none" aria-label={t(m.ruleMatch, { n: i + 1 })} value={r.op} onChange={(e) => setRule(i, { op: e.target.value as Rule['op'] })}>
                      <option value="contains">{t(m.opContains)}</option>
                      <option value="equals">{t(m.opEquals)}</option>
                      <option value="starts_with">{t(m.opStartsWith)}</option>
                    </select>
                    <input
                      className="in mono flex-[1_1_140px]"
                      aria-label={t(m.ruleAddress, { n: i + 1 })}
                      maxLength={200}
                      placeholder={'/pricing'}
                      value={r.value}
                      onChange={(e) => setRule(i, { value: e.target.value })}
                    />
                    <Button size="sm" aria-label={t(m.removeRule, { n: i + 1 })} onClick={() => set({ rules: f.rules.filter((_, x) => x !== i) })}>
                      {t(m.remove)}
                    </Button>
                  </Row>
                ))}
                <Button
                  size="sm"
                  className="self-start"
                  disabledReason={f.rules.length >= maxRules ? t(m.mostRules) : undefined}
                  onClick={() => set({ rules: [...f.rules, { op: 'contains', value: '' }] })}
                >
                  {t(m.addRule)}
                </Button>
              </>
            ) : null}
            <span className="k">{t(m.pagesHelp)}</span>
          </div>
          <div role="group" aria-labelledby={`${id}-src`} className="flex flex-col gap-2 border-t border-line-2 py-3.5">
            <h3 id={`${id}-src`} className="m-0 text-[13.5px] font-semibold">
              {t(m.sourcesGroup)}
            </h3>
            <Grid cols={3} gap={8}>
              {(
                [
                  ['utmSource', 'utm_source'],
                  ['utmMedium', 'utm_medium'],
                  ['utmCampaign', 'utm_campaign'],
                ] as const
              ).map(([k, param]) => (
                <label key={k} className="lbl">
                  <span className="mono">{param}</span>
                  <input className="in" value={f[k]} onChange={(e) => setF((x) => ({ ...x, [k]: e.target.value }))} />
                </label>
              ))}
            </Grid>
            <span className="k">{t(m.utmHelp)}</span>
            <label className="lbl">
              {t(m.cameFrom)}
              <select className="in" value={f.referrer} onChange={(e) => set({ referrer: e.target.value as Fields['referrer'] })}>
                <option value="any">{t(m.refAny)}</option>
                <option value="search">{t(m.refSearch)}</option>
                <option value="social">{t(m.refSocial)}</option>
                <option value="other_site">{t(m.refOther)}</option>
                <option value="direct">{t(m.refDirect)}</option>
              </select>
            </label>
          </div>
          <div role="group" aria-labelledby={`${id}-dev`} className="flex flex-col gap-2 border-t border-line-2 py-3.5">
            <h3 id={`${id}-dev`} className="m-0 text-[13.5px] font-semibold">
              {t(m.devicesGroup)}
            </h3>
            <Row wrap gap="8px 18px">
              {devices.map((d) => (
                <label key={d} className="fb-check">
                  <input
                    type="checkbox"
                    checked={f.devices.includes(d)}
                    onChange={(e) => set({ devices: e.target.checked ? [...f.devices, d] : f.devices.filter((x) => x !== d) })}
                  />
                  {t(d === 'desktop' ? m.desktop : d === 'tablet' ? m.tablet : m.mobile)}
                </label>
              ))}
            </Row>
            <span className="k">{t(m.devicesHelp)}</span>
          </div>
          <div role="group" aria-labelledby={`${id}-geo`} className="flex flex-col gap-2 border-t border-line-2 py-3.5">
            <h3 id={`${id}-geo`} className="m-0 text-[13.5px] font-semibold">
              {t(m.countriesGroup)}
            </h3>
            <select className="in" aria-label={t(m.countriesGroup)} value={f.countries} disabled={!geoip} onChange={(e) => set({ countries: e.target.value as Fields['countries'] })}>
              <option value="all">{t(m.countriesAll)}</option>
              <option value="only">{t(m.countriesOnly)}</option>
              <option value="except">{t(m.countriesExcept)}</option>
            </select>
            <input
              className="in"
              aria-label={t(m.countryCodes)}
              placeholder={'US, CA'}
              value={f.codes}
              disabled={!geoip || f.countries === 'all'}
              onChange={(e) => set({ codes: e.target.value })}
            />
            {geoip ? null : <Note tone="info">{t(m.geoipNote)}</Note>}
          </div>
          <div role="group" aria-labelledby={`${id}-vis`} className="flex flex-col gap-2 border-t border-line-2 py-3.5">
            <h3 id={`${id}-vis`} className="m-0 text-[13.5px] font-semibold">
              {t(m.visitorsGroup)}
            </h3>
            <Row wrap gap="8px 18px">
              {(
                [
                  ['all', m.visitorsAll],
                  ['new', m.visitorsNew],
                  ['returning', m.visitorsReturning],
                ] as const
              ).map(([v, label]) => (
                <label key={v} className="fb-check">
                  <input type="radio" name={`${id}-visitors`} checked={f.visitors === v} onChange={() => set({ visitors: v })} />
                  {t(label)}
                </label>
              ))}
            </Row>
            <span className="k">{t(m.visitorsHelp)}</span>
          </div>
          <div role="group" aria-labelledby={`${id}-sch`} className="flex flex-col gap-2 border-t border-line-2 pt-3.5">
            <h3 id={`${id}-sch`} className="m-0 text-[13.5px] font-semibold">
              {t(m.scheduleGroup)}
            </h3>
            {schedule}
          </div>
        </div>
      )}
    </Dialog>
  )
}
