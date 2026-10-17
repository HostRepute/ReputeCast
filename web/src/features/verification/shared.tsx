import { useQueryClient } from '@tanstack/react-query'
import { useCallback, type ReactNode } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { T, t as tr, useT } from '@/i18n'
import { clock, orList, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import '@/features/integrations/integrations.css'
import { m } from './messages'
import './verification.css'

export type Overview = Schemas['VerificationOverview']
export type ServiceView = Schemas['VerificationServiceView']
export type ServiceKey = ServiceView['key']
export type Use = Schemas['VerificationUse']
export type Job = Schemas['VerificationJobView']
export type Estimate = Schemas['VerificationEstimate']
export type Target = Schemas['VerificationTarget']
export type StartInput = Schemas['VerificationStartInput']
export type Realtime = Schemas['VerificationRealtime']
export type AddressCheck = Schemas['ContactAddressCheck']
export type FormSetting = Schemas['VerificationFormSetting']
export type Origin = NonNullable<StartInput['origin']>

export const base = '/api/v1/workspaces/{workspace}/verification' as const

/** Results everywhere: valid, invalid, risky and unknown, as the boards colour them. */
export const RESULT = { valid: '#15803D', invalid: '#B91C1C', risky: '#B45309', unknown: '#8B94A3' } as const

export function useRefreshVerification() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => typeof q.queryKey[1] === 'string' && ((q.queryKey[1] as string).startsWith(`${base}/`) || (q.queryKey[1] as string).includes('/integrations/')),
      }),
    [qc],
  )
}

export function useOverview(enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/services`, { params: { path: { workspace: slug } } }, { enabled })
}

export function useRealtime(key: ServiceKey | undefined, enabled: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/services/{service}/realtime`, { params: { path: { workspace: slug, service: key ?? 'zerobounce' } } }, { enabled: Boolean(key) && enabled })
}

const open = (s: Job['status']) => s === 'running' || s === 'queued' || s === 'paused'

export function useJob(id: string, enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    `${base}/jobs/{job}`,
    { params: { path: { workspace: slug, job: id } } },
    { enabled, refetchInterval: (q) => (q.state.data && open(q.state.data.status) ? 2000 : false) },
  )
}

/** "$0.008", "$28.92". */
export function money(v: number, perCheck = false): string {
  if (perCheck) return `$${v.toFixed(3)}`
  return `$${v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

/** "today at 10:12", "Tue 14 Oct at 15:09" — never seconds. */
export function when(d: string): string {
  return shortDate(d) === shortDate(new Date()) ? tr(m.todayAt, { time: clock(d) }) : `${shortDate(d)}, ${clock(d)}`
}

/** "Valid", "Risky · catch-all", in the result's colour. */
export function ResultBadge({ result, reason, children }: { result: string | undefined; reason?: string; children?: ReactNode }) {
  const t = useT()
  const cls = result === 'valid' ? 'ok' : result === 'invalid' ? 'bad' : result === 'risky' ? 'warn' : 'mut'
  const word =
    result === 'valid'
      ? t(m.resultValid)
      : result === 'invalid'
        ? t(m.resultInvalid)
        : result === 'risky'
          ? reason
            ? t(m.resultRiskyReason, { reason })
            : t(m.resultRisky)
          : result === 'unknown'
            ? t(m.resultUnknown)
            : t(m.resultNever)
  return <span className={`badge ${cls}`}>{children ?? word}</span>
}

/** Valid, invalid, risky and unknown as one bar. */
export function ResultBar({ valid, invalid, risky, unknown, total, label }: { valid: number; invalid: number; risky: number; unknown: number; total?: number; label: string }) {
  const sum = valid + invalid + risky + unknown
  const rest = total && total > sum ? total - sum : 0
  const segs: [number, string][] = [
    [valid, RESULT.valid],
    [invalid, RESULT.invalid],
    [risky, RESULT.risky],
    [unknown, RESULT.unknown],
  ]
  return (
    <div className="vf-bar" role="img" aria-label={label}>
      {segs.map(([n, c], i) => (n > 0 ? <i key={i} style={{ flex: `${n} 1 0%`, background: c }} /> : null))}
      {rest ? <i className="w" style={{ flex: `${rest} 1 0%` }} /> : null}
    </div>
  )
}

export function ResultLegend({ valid, invalid, risky, unknown }: { valid: number; invalid: number; risky: number; unknown: number }) {
  const t = useT()
  const items: [string, string, number][] = [
    [RESULT.valid, t(m.resultValid), valid],
    [RESULT.invalid, t(m.resultInvalid), invalid],
    [RESULT.risky, t(m.resultRisky), risky],
    [RESULT.unknown, t(m.resultUnknown), unknown],
  ]
  return (
    <div className="vf-legend">
      {items.map(([c, l, n]) => (
        <span key={l}>
          <i style={{ background: c }} />
          {l}
          <b>{n.toLocaleString('en-US')}</b>
        </span>
      ))}
    </div>
  )
}

/** The notice every use shows: what is sent and what never is. */
export function Notice({ head, name, foot }: { head: ReactNode; name: string; foot?: ReactNode }) {
  const t = useT()
  return (
    <div className="note i vf-notice" role="note">
      <Icon name="info" size={16} className="mt-px flex-none" />
      <div className="flex min-w-0 flex-col gap-1.5">
        <b className="font-bold">{head}</b>
        <dl className="vf-sent">
          <dt>{t(m.sentLabel)}</dt>
          <dd>{t(m.sentValue)}</dd>
          <dt>{t(m.neverSentLabel)}</dt>
          <dd>{t(m.neverSentValue)}</dd>
        </dl>
        <span>{t(m.keeps, { name })}</span>
        {foot}
      </div>
    </div>
  )
}

/** verify:catch-all in the boards' grey tag chip. */
export function Tag({ children }: { children: ReactNode }) {
  return <span className="vf-tag">{children}</span>
}

export const tagValue = { tag: (c: ReactNode) => <Tag>{c}</Tag> }

/** The service's letters (ZB, K, NB, B, E). */
export function ServiceTile({ initials, size }: { initials: string; size?: 'lg' }) {
  return (
    <span className={size ? `ih-tile ${size}` : 'ih-tile'} aria-hidden="true">
      {initials}
    </span>
  )
}

export function StatusBadge({ v }: { v: ServiceView }) {
  const t = useT()
  switch (v.status) {
    case 'connected':
      return (
        <span className="badge ok">
          <span className="dot" aria-hidden="true" />
          {t(m.statusConnected)}
        </span>
      )
    case 'low_credits':
      return (
        <span className="badge warn">
          <span className="dot" aria-hidden="true" />
          {t(m.statusLowCredits)}
        </span>
      )
    case 'error':
      return (
        <span className="badge bad">
          <span className="dot" aria-hidden="true" />
          {t(m.statusError)}
        </span>
      )
  }
  return <span className="badge mut">{t(m.statusNotConnected)}</span>
}

/** The board's load error: what failed, that nothing stopped, Retry. */
export function LoadError({ title, error, body, onRetry }: { title: string; error: unknown; body: (status: number) => string; onRetry: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const status = error instanceof ApiError ? error.status : 0
  const ref = error instanceof ApiError ? (error.problem.instance?.split('/').pop() ?? '') : ''
  return (
    <section className="card flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
      <Icon name="alertCircle" size={28} className="text-bad-strong" />
      <span className="text-[16px] font-bold">{title}</span>
      <span className="k max-w-[440px] text-ink-2!">{status ? body(status) : t(m.loadOfflineBody)}</span>
      <div className="flex gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace/settings/integrations" params={{ workspace: slug }}>
          {t(m.allIntegrations)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(m.request, { id: ref })}</span> : null}
    </section>
  )
}

/** "Valid 2,904 · Invalid 287, suppressed · Risky 318 · Unknown 106". */
export function resultsLine(u: { valid: number; invalid: number; risky: number; unknown: number; suppressed: number }): string {
  return tr(u.suppressed ? m.usesResultsSuppressed : m.usesResults, { valid: u.valid, invalid: u.invalid, risky: u.risky, unknown: u.unknown })
}

/** A missing permission, with code marks: "needs <code>x</code>. Ask …". */
export function Denied({ msg, helpers }: { msg: { ask: Parameters<typeof T>[0]['msg']; nobody: Parameters<typeof T>[0]['msg'] }; helpers: string[] }) {
  return <T msg={helpers.length ? msg.ask : msg.nobody} values={{ helpers: orList(helpers), code: (c) => <span className="mono">{c}</span> }} />
}
