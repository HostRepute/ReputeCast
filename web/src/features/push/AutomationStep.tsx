import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { clock, count, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './automation.messages'
import { views } from './cuts'
import { Preview } from './Previews'
import './push.css'
import './composer.css'

type Step = Schemas['AutomationPushStep']
type NodeData = Schemas['AutomationNodeData']
type Estimate = Schemas['PushStepEstimate']

/** The notification's limits, as the server checks them (internal/push TitleMax, MessageMax). */
const TITLE_MAX = 50
const MESSAGE_MAX = 120
const DEFAULT_EXPIRY = 240
const EXPIRIES = [
  { minutes: 30, msg: m.expiry30 },
  { minutes: 240, msg: m.expiry240 },
  { minutes: 1440, msg: m.expiry1440 },
] as const

/** One setting of a channel step's panel, as the automations panel draws its own. */
export function StepProp({ title, id, children }: { title: string; id: string; children: ReactNode }) {
  return (
    <div className="fe-prop" role="group" aria-labelledby={id}>
      <span className="tt" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

/** Who the flow's push steps reach: one request per flow, shared by the panel and the canvas. */
export function usePushEstimate(automationId: string | undefined, enabled = true): Estimate | undefined {
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automations/{id}/push-estimate',
    { params: { path: { workspace: slug, id: automationId ?? '' } } },
    { enabled: enabled && Boolean(automationId), staleTime: 60_000, retry: false },
  )
  return q.data
}

/** The canvas node's lines for a Send push step: who it reaches, and what happens outside quiet hours. */
export function pushNodeLines(s: Step | undefined, t: ReturnType<typeof useT>, est?: Estimate): string[] {
  if (!s?.title) return []
  const out: string[] = []
  if (est?.pushOn) out.push(t(m.nodeLinked, { n: est.linked }))
  out.push(s.outside === 'skip' ? t(m.nodeSkip) : t(m.nodeWait))
  return out
}

/** "news.acme.io" for a link, as the notification names its site. */
function hostOf(link: string): string {
  try {
    return new URL(/^https?:\/\//.test(link) ? link : `https://${link}`).host
  } catch {
    return ''
  }
}

/**
 * The Send push step's settings (AutomationChannels › Push step selected):
 * the notification's title and message with their counters, the link and
 * its button, the expiry, quiet hours, the subscription rule with who it
 * reaches, and the previews on Chrome for macOS and Android.
 */
export function PushStepProps({ data: d, automationId, editable, onChange }: { data: NodeData; automationId: string; editable: boolean; onChange: (data: NodeData) => void }) {
  const t = useT()
  const s: Step = d.push ?? {}
  const set = (patch: Partial<Step>) => onChange({ ...d, push: { ...s, ...patch } })
  const disabled = !editable
  const est = usePushEstimate(automationId)
  const title = s.title ?? ''
  const message = s.message ?? ''
  const titleLen = [...title].length
  const msgLen = [...message].length
  const expiry = s.expiryMinutes || DEFAULT_EXPIRY
  const n = { title, message, host: hostOf(s.link ?? ''), buttons: s.button ? [s.button] : [] }
  const shown = views(title, message).filter((v) => v.platform.key === 'chrome_mac' || v.platform.key === 'chrome_android')
  return (
    <>
      <StepProp title={t(m.name)} id="push-name">
        <input aria-labelledby="push-name" className="in" disabled={disabled} placeholder={t(m.namePlaceholder)} value={d.label ?? ''} onChange={(e) => onChange({ ...d, label: e.target.value || undefined })} />
      </StepProp>
      <div className="fe-prop">
        <span className="flex items-center justify-between gap-2">
          <label className="tt" htmlFor="push-step-title">
            {t(m.title)}
          </label>
          <span className={titleLen > TITLE_MAX ? 'pc-cnt w' : 'pc-cnt'}>{t(m.counter, { n: titleLen, max: TITLE_MAX })}</span>
        </span>
        <input id="push-step-title" className={titleLen > TITLE_MAX ? 'in w' : 'in'} disabled={disabled} maxLength={200} value={title} onChange={(e) => set({ title: e.target.value })} />
        {titleLen > TITLE_MAX ? <span className="err">{t(m.titleLong)}</span> : null}
        <span className="flex items-center justify-between gap-2">
          <label className="tt" htmlFor="push-step-message">
            {t(m.message)}
          </label>
          <span className={msgLen > MESSAGE_MAX ? 'pc-cnt w' : 'pc-cnt'}>{t(m.counter, { n: msgLen, max: MESSAGE_MAX })}</span>
        </span>
        <textarea id="push-step-message" className="in" rows={3} disabled={disabled} maxLength={500} value={message} onChange={(e) => set({ message: e.target.value })} />
      </div>
      <StepProp title={t(m.linkAndButton)} id="push-link">
        <label className="sr-only" htmlFor="push-step-link">
          {t(m.link)}
        </label>
        <input id="push-step-link" className="in mono" disabled={disabled} placeholder={t(m.linkPlaceholder)} maxLength={2000} value={s.link ?? ''} onChange={(e) => set({ link: e.target.value || undefined })} />
        <label className="sr-only" htmlFor="push-step-button">
          {t(m.button)}
        </label>
        <input id="push-step-button" className="in" disabled={disabled} placeholder={t(m.buttonPlaceholder)} maxLength={40} value={s.button ?? ''} onChange={(e) => set({ button: e.target.value || undefined })} />
        <span className="k">{t(m.buttonHint)}</span>
      </StepProp>
      <StepProp title={t(m.expiry)} id="push-expiry">
        {EXPIRIES.map((x) => (
          <label key={x.minutes} className="fe-chk">
            <input type="radio" name="push-expiry" checked={expiry === x.minutes} disabled={disabled} onChange={() => set({ expiryMinutes: x.minutes })} />
            {t(x.msg)}
          </label>
        ))}
        {!EXPIRIES.some((x) => x.minutes === expiry) ? (
          <label className="fe-chk">
            <input type="radio" name="push-expiry" checked disabled={disabled} readOnly />
            {t(m.expiryOther, { n: expiry })}
          </label>
        ) : null}
        <span className="k">{t(m.expiryHint)}</span>
      </StepProp>
      <StepProp title={t(m.outside)} id="push-outside">
        <label className="fe-chk">
          <input type="radio" name="push-outside" checked={(s.outside || 'wait') === 'wait'} disabled={disabled} onChange={() => set({ outside: 'wait' })} />
          {t(m.outsideWait)}
        </label>
        <label className="fe-chk">
          <input type="radio" name="push-outside" checked={s.outside === 'skip'} disabled={disabled} onChange={() => set({ outside: 'skip' })} />
          {t(m.outsideSkip)}
        </label>
        {est ? <span className="k">{est.quiet ? t(m.outsideHint, { window: est.window }) : t(m.quietOff)}</span> : null}
      </StepProp>
      <StepProp title={t(m.who)} id="push-who">
        <span className="text-[12.5px]">{t(m.whoText)}</span>
        {est && !est.pushOn ? <Note tone="warn">{t(m.pushOff)}</Note> : null}
        {est?.pushOn ? (
          <span className="k">
            {est.linked ? `${t(m.reach, { linked: est.linked, contacts: est.contacts })} ${est.linked < est.contacts ? t(m.reachRest) : ''}`.trim() : t(m.reachNone)}{' '}
            {t(m.anonymous)}
          </span>
        ) : null}
      </StepProp>
      <StepProp title={t(m.previews)} id="push-previews">
        <div className="flex flex-col gap-3">
          {shown.map((v) => (
            <Preview key={v.platform.key} v={v} n={n} />
          ))}
        </div>
      </StepProp>
    </>
  )
}

/** The step's notifications over 30 days (Step stats), with the engine's skips. */
export function PushStepStats({ nodeId, automationId, hasStats, skipped }: { nodeId: string; automationId: string; hasStats: boolean; skipped: number }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/{id}/push-stats', { params: { path: { workspace: slug, id: automationId } } }, { enabled: hasStats })
  const st = q.data?.steps.find((x) => x.nodeId === nodeId)
  const ran = Boolean(st && st.sent)
  const cells: { label: string; value: string }[] = [
    { label: t(m.sSent), value: ran ? count(st!.sent) : '–' },
    { label: t(m.sShown), value: ran ? count(st!.shown) : '–' },
    { label: t(m.sClicks), value: ran ? count(st!.clicks) : '–' },
    { label: t(m.sSkipped), value: ran || skipped ? count(skipped) : '–' },
  ]
  return (
    <StepProp title={t(m.statsTitle)} id="push-stats">
      {st?.firstAt && st.lastAt ? (
        <span className="k">
          {dayMonthDate(st.firstAt) === dayMonthDate(st.lastAt)
            ? t(m.ranDay, { date: dayMonthDate(st.firstAt), from: clock(st.firstAt), to: clock(st.lastAt) })
            : t(m.ranDays, { from: dayMonthDate(st.firstAt), to: dayMonthDate(st.lastAt) })}
        </span>
      ) : null}
      {!ran ? <span className="k">{t(m.noPush)}</span> : null}
      <div className="fe-stats">
        {cells.map((c) => (
          <span key={c.label}>
            {c.label} <b className="n">{c.value}</b>
          </span>
        ))}
      </div>
      {skipped ? <span className="k">{t(m.skippedNote)}</span> : null}
    </StepProp>
  )
}
