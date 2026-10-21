import { keepPreviousData } from '@tanstack/react-query'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Note } from '@/components/ui/Note'
import { formatList, T, useT, type MessageDescriptor } from '@/i18n'
import { clock, count, dayMonthDate, money, monthName } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './automation.messages'
import { countryName } from './common'
import { EditorTools } from './EditorTools'
import { measure } from './measure'
import './sms.css'

type Step = Schemas['AutomationSMSStep']
type NodeData = Schemas['AutomationNodeData']

function Prop({ title, id, children }: { title: string; id: string; children: ReactNode }) {
  return (
    <div className="fe-prop" role="group" aria-labelledby={id}>
      <span className="tt" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

/** The value after a pause in typing: the estimate asks the server once per pause. */
function useSettled<T>(value: T, ms: number): T {
  const [settled, setSettled] = useState(value)
  useEffect(() => {
    const id = window.setTimeout(() => setSettled(value), ms)
    return () => window.clearTimeout(id)
  }, [value, ms])
  return settled
}

/**
 * The Send SMS step's settings (AutomationChannels): the text between its
 * locked parts with the counter, the opt-out text, quiet hours, the consent
 * rule, "Skip if" and the estimate.
 */
export function SmsStepProps({ data: d, lists, editable, onChange }: { data: NodeData; lists: { id: string; name: string }[]; editable: boolean; onChange: (data: NodeData) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const s: Step = d.sms ?? {}
  const set = (patch: Partial<Step>) => onChange({ ...d, sms: { ...s, ...patch } })
  const disabled = !editable
  const body = useSettled(s.body ?? '', 400)
  const q = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/sms/step-estimate',
    { params: { path: { workspace: slug } }, body: { body, consentListId: s.consentListId || undefined, noOptOut: s.noOptOut || undefined } },
    { placeholderData: keepPreviousData },
  )
  const e = q.data
  const live = measure(e?.prefix ?? '', s.body ?? '', s.noOptOut ? '' : (e?.optOut ?? ''), e?.shortDomain ?? '')
  const c = live.count
  const listName = lists.find((l) => l.id === s.consentListId)?.name
  const fill = c.fill.length ? c.fill : [0]
  return (
    <>
      <Prop title={t(m.name)} id="sms-name">
        <input aria-labelledby="sms-name" className="in" disabled={disabled} placeholder={t(m.namePlaceholder)} value={d.label ?? ''} onChange={(ev) => onChange({ ...d, label: ev.target.value || undefined })} />
      </Prop>
      <Prop title={t(m.text)} id="sms-text">
        <div className={`sms-ed${disabled ? ' ro' : ''}`}>
          {e?.prefix ? <span className="fx">{e.prefix}</span> : null}
          <textarea aria-labelledby="sms-text" rows={4} disabled={disabled} value={s.body ?? ''} onChange={(ev) => set({ body: ev.target.value })} />
          {!s.noOptOut && e?.optOut ? <span className="fx">{e.optOut}</span> : null}
        </div>
        <div className="sms-segc">
          <div className="sms-segc-h">
            <span>
              <T msg={m.counter} values={{ characters: c.characters, segments: Math.max(1, c.segments), b: (x) => <b className="n">{x}</b> }} />
            </span>
            <Badge tone={c.encoding === 'gsm7' ? 'neutral' : 'warn'}>{c.encoding === 'gsm7' ? 'GSM-7' : 'UCS-2'}</Badge>
          </div>
          <div className={`sms-segc-b${c.encoding === 'ucs2' ? ' w' : ''}`} aria-hidden="true">
            {fill.map((f, i) => (
              <span key={i}>
                <i style={{ width: `${Math.min(100, (f / c.capacity) * 100).toFixed(1)}%` }} />
              </span>
            ))}
          </div>
          <div className="sms-segc-x" aria-hidden="true">
            {fill.map((_, i) => (
              <span key={i}>{c.segments <= 1 ? t(m.segUpTo, { n: i + 1, capacity: c.capacity }) : t(m.segPart, { n: i + 1, capacity: c.capacity })}</span>
            ))}
          </div>
        </div>
        {editable ? <EditorTools body={s.body ?? ''} setBody={(b) => set({ body: b })} /> : null}
        {e?.shorteners.length ? <Note tone="bad">{t(m.shortener, { hosts: formatList(e.shorteners), domain: e.shortDomain })}</Note> : null}
      </Prop>
      <Prop title={t(m.optOutTitle)} id="sms-optout">
        <label className="fe-chk">
          <input type="checkbox" checked={!s.noOptOut} disabled={disabled} onChange={(ev) => set({ noOptOut: ev.target.checked ? undefined : true })} />
          {t(m.optOutAdd, { text: e?.optOut ?? '' })}
        </label>
        <span className="k">{t(m.optOutHint)}</span>
      </Prop>
      <Prop title={t(m.outside)} id="sms-outside">
        <label className="fe-chk">
          <input type="radio" name="sms-outside" checked={(s.outside || 'wait') === 'wait'} disabled={disabled} onChange={() => set({ outside: 'wait' })} />
          {t(m.outsideWait)}
        </label>
        <label className="fe-chk">
          <input type="radio" name="sms-outside" checked={s.outside === 'skip'} disabled={disabled} onChange={() => set({ outside: 'skip' })} />
          {t(m.outsideSkip)}
        </label>
        {e ? <span className="k">{t(m.outsideHint, { window: e.window })}</span> : null}
      </Prop>
      <Prop title={t(m.consent)} id="sms-consent">
        <select aria-labelledby="sms-consent" className="in" disabled={disabled} value={s.consentListId ?? ''} onChange={(ev) => set({ consentListId: ev.target.value || undefined })}>
          {!s.consentListId ? <option value="">{t(m.pickList)}</option> : null}
          {lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
        <span className="k">{listName ? t(m.consentNote, { list: listName }) : t(m.consentPick)}</span>
      </Prop>
      <Prop title={t(m.skipIf)} id="sms-skip">
        <div className="flex flex-wrap items-center gap-2 text-[13px]">
          <label className="fe-chk">
            <input type="checkbox" checked={(s.skipRecentHours ?? 0) > 0} disabled={disabled} onChange={(ev) => set({ skipRecentHours: ev.target.checked ? 24 : undefined })} />
            {t(m.gotText)}
          </label>
          <input
            className="in w-16 text-right"
            type="number"
            min={1}
            max={720}
            aria-label={t(m.hoursLabel)}
            disabled={disabled || !(s.skipRecentHours ?? 0)}
            value={s.skipRecentHours ?? 24}
            onChange={(ev) => set({ skipRecentHours: Math.max(1, Math.min(720, Number(ev.target.value) || 24)) })}
          />
          <span>{t(m.hours)}</span>
        </div>
      </Prop>
      {e ? (
        <Prop title={t(m.estimate)} id="sms-estimate">
          <div className="sms-sum">
            <span>{t(m.estimateLine, { texts: count(e.texts), segments: Math.max(1, c.segments) })}</span>
            <span className="n">{money(e.cost)}</span>
          </div>
          {e.budget.budget ? (
            <span className="k">
              {t(m.estimateBudget, { month: monthName(e.budget.month, 'long'), spent: money(e.budget.spent), budget: money(e.budget.budget, Number.isInteger(e.budget.budget) ? 0 : 2) })}
            </span>
          ) : null}
        </Prop>
      ) : null}
    </>
  )
}

const skipWord: Record<string, MessageDescriptor> = {
  no_consent: m.skipNoConsent,
  no_phone: m.skipNoPhone,
  suppressed: m.skipSuppressed,
  recent: m.skipRecent,
  quiet_hours: m.skipQuietHours,
  no_route: m.skipNoRoute,
  sender_pending: m.skipPending,
  sender_not_registered: m.skipNotRegistered,
  sender_rejected: m.skipRejected,
  sender_not_supported: m.skipNotSupported,
  budget: m.skipBudget,
}
/** Route reasons name the countries: "20 in Ireland, where the sender isn't approved yet". */
const skipWordIn: Record<string, MessageDescriptor> = {
  no_route: m.skipNoRouteIn,
  sender_pending: m.skipPendingIn,
  sender_not_registered: m.skipNotRegisteredIn,
  sender_rejected: m.skipRejectedIn,
  sender_not_supported: m.skipNotSupportedIn,
}

/** The step's texts over 30 days (Step stats · 30 days), or what happens before it first runs. */
export function SmsStepStats({ nodeId, automationId, hasStats }: { nodeId: string; automationId: string; hasStats: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/{id}/sms-stats', { params: { path: { workspace: slug, id: automationId } } }, { enabled: hasStats })
  const st = q.data?.steps.find((x) => x.nodeId === nodeId)
  const ran = Boolean(st && (st.sent || st.skipped))
  const cells: { label: string; value: string }[] = [
    { label: t(m.sSent), value: ran ? count(st!.sent) : '–' },
    { label: t(m.sDelivered), value: ran ? count(st!.delivered) : '–' },
    { label: t(m.sClicks), value: ran ? count(st!.clicks) : '–' },
    { label: t(m.sOptOuts), value: ran ? count(st!.optOuts) : '–' },
    { label: t(m.sSkipped), value: ran ? count(st!.skipped) : '–' },
    { label: t(m.sCost), value: ran ? money(st!.cost) : '–' },
  ]
  const skipped = st
    ? Object.entries(st.skippedBy)
        .filter(([, n]) => n > 0)
        .sort((a, b) => b[1] - a[1])
        .map(([reason, n]) => {
          const countries = Object.keys(st.skippedIn).sort().map(countryName)
          const inMsg = skipWordIn[reason]
          return inMsg && countries.length ? t(inMsg, { count: n, countries: formatList(countries) }) : t(skipWord[reason] ?? m.skipOther, { count: n })
        })
    : []
  return (
    <Prop title={t(m.statsTitle, { days: q.data?.days ?? 30 })} id="sms-stats">
      {st?.firstAt && st.lastAt ? (
        <span className="k">
          {dayMonthDate(st.firstAt) === dayMonthDate(st.lastAt)
            ? t(m.ranDay, { date: dayMonthDate(st.firstAt), from: clock(st.firstAt), to: clock(st.lastAt) })
            : t(m.ranDays, { from: dayMonthDate(st.firstAt), to: dayMonthDate(st.lastAt) })}
        </span>
      ) : null}
      {!ran ? <span className="k">{st?.waiting ? t(m.waiting, { count: st.waiting }) : t(m.noTexts)}</span> : null}
      <div className="fe-stats">
        {cells.map((cl) => (
          <span key={cl.label}>
            {cl.label} <b className="n">{cl.value}</b>
          </span>
        ))}
      </div>
      {skipped.length ? <span className="k">{t(m.skippedLine, { count: st!.skipped, parts: formatList(skipped) })}</span> : null}
    </Prop>
  )
}
