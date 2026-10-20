import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { ForbiddenState, problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { NavTabs, type NavTab } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, formatList, t, useT } from '@/i18n'
import { dayMonthDate, decimal, zonedParts } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import './grow.css'

export const GROW = '/api/v1/workspaces/{workspace}/grow' as const

export type GrowTab = 'overview' | 'referrals' | 'recommendations'

export type Helpers = Schemas['GrowHelperSets']

/**
 * One Grow screen: the title with the section tabs (Overview · Referrals ·
 * Recommendations) after it. Without grow:view the section is a 403.
 */
export function GrowPage({ tab, documentTitle, actions, children }: { tab: GrowTab; documentTitle?: string; actions?: ReactNode; children: ReactNode }) {
  const tr = useT()
  const can = useCan('grow:view')
  if (!can) {
    return (
      <Page title={tr(m.title)}>
        <ForbiddenState permission="grow:view" title={tr(m.forbiddenTitle)} />
      </Page>
    )
  }
  return (
    <Page title={tr(m.title)} documentTitle={documentTitle} filters={<GrowTabs tab={tab} />} actions={actions}>
      {children}
    </Page>
  )
}

/** Overview · Referrals · Recommendations, inline after the title. */
export function GrowTabs({ tab }: { tab: GrowTab }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  const tabs: NavTab[] = [
    { label: tr(m.tabOverview), to: '/w/$workspace/grow', params, exact: true, current: tab === 'overview' },
    { label: tr(m.tabReferrals), to: '/w/$workspace/grow/referrals', params, current: tab === 'referrals' },
    { label: tr(m.tabRecommendations), to: '/w/$workspace/grow/recommendations', params, current: tab === 'recommendations' },
  ]
  return <NavTabs tabs={tabs} label={tr(m.tabsLabel)} className="ml-3 max-md:ml-0" />
}

/** "Read-only · Aisha Bello, Viewer" over what the member can't do here. */
export function ReadOnlyNote({ children }: { children: ReactNode }) {
  const tr = useT()
  const { me } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: me.workspace.roleKey, name: me.workspace.roleName })
  return (
    <div className="note n" role="note">
      <span className="mt-0.5 text-ink-2" aria-hidden="true">
        <Icon name="info" size={16} strokeWidth={1.8} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{tr(m.readOnlyWho, { name: user?.name ?? '', role })}</span>
        <div>{children}</div>
      </div>
    </div>
  )
}

/** `<mono>` for permission keys inside messages. */
export const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** "Sofia Rossi, Marcus Chen or Linh Nguyen". */
export function orNames(names: readonly string[] | undefined): string {
  return formatList(names ?? [], 'disjunction')
}

/** A sentence about a missing permission with who can help: "Changing … needs grow:edit. Ask …". */
export function Needs({ msg, helpers }: { msg: Parameters<typeof T>[0]['msg']; helpers?: readonly string[] }) {
  return <T msg={msg} values={{ helpers: orNames(helpers), mono }} />
}

/** The program's sub-sections: Results · Milestones · Referrers · Review · Settings. */
export function ProgramNav({ current, held, programId }: { current: 'results' | 'milestones' | 'referrers' | 'review' | 'settings'; held?: number; programId?: string }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  // The Review badge on every sub-section: the program's held referrals.
  const { program } = useReferralPrograms(programId)
  const detail = $api.useQuery('get', `${GROW}/referrals/{program}`, { params: { path: { workspace: slug, program: program?.id ?? '' } } }, { enabled: held === undefined && Boolean(program) })
  const heldCount = held ?? detail.data?.held
  const search = programId ? { program: programId } : {}
  const items = [
    { key: 'results', label: tr(m.subResults), to: '/w/$workspace/grow/referrals' as const },
    { key: 'milestones', label: tr(m.subMilestones), to: '/w/$workspace/grow/referrals/milestones' as const },
    { key: 'referrers', label: tr(m.subReferrers), to: '/w/$workspace/grow/referrals/referrers' as const },
    { key: 'review', label: tr(m.subReview), to: '/w/$workspace/grow/referrals/review' as const },
    { key: 'settings', label: tr(m.subSettings), to: '/w/$workspace/grow/referrals/settings' as const },
  ]
  return (
    <nav className="gw-subnav" aria-label={tr(m.programNav)}>
      {items.map((x) => (
        <Link key={x.key} to={x.to} params={params} search={search as never} className={x.key === current ? 'on' : undefined} aria-current={x.key === current ? 'page' : undefined}>
          {x.label}
          {x.key === 'review' && heldCount ? <Badge tone="warn">{heldCount}</Badge> : null}
        </Link>
      ))}
    </nav>
  )
}

/** The workspace's referral programs, and the one a screen is about (?program=, else the first). */
export function useReferralPrograms(programParam?: string) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/referrals`, { params: { path: { workspace: slug } } })
  const items = q.data?.items ?? []
  const program = items.find((p) => p.id === programParam) ?? items[0]
  return { q, list: q.data, program }
}

/** A calendar day ("2026-09-22" or a UTC midnight timestamp) as a moment inside that day anywhere. */
export function calendarDay(s: string): Date {
  return new Date(`${s.slice(0, 10)}T12:00:00Z`)
}

/** "15–21 Sep", "29 Sep–5 Oct". */
export function dayRange(from: string, to: string): string {
  const a = calendarDay(from)
  const b = calendarDay(to)
  if (from.slice(0, 10) === to.slice(0, 10)) return dayMonthDate(b)
  const end = dayMonthDate(b)
  const pa = zonedParts(a)
  const pb = zonedParts(b)
  // Same month, and the region writes the day first: "15–21 Sep".
  if (pa.month === pb.month && pa.year === pb.year && /^\d/.test(end)) return t(m.sameMonthRange, { from: String(pa.day), to: end })
  return t(m.sameMonthRange, { from: dayMonthDate(a), to: end })
}

/** The error panel of a screen or card that failed to load. */
export function LoadError({ error, title, onRetry, children, secondary }: { error: unknown; title: string; onRetry: () => void; children?: ReactNode; secondary?: ReactNode }) {
  const tr = useT()
  const status = error instanceof ApiError ? error.status : undefined
  return (
    <ErrorPanel
      title={title}
      reference={problemReference(error)}
      actions={
        <>
          <Button onClick={onRetry}>{tr(m.retry)}</Button>
          {secondary}
        </>
      }
    >
      {children ?? (status ? tr(m.loadErrorDetail, { status }) : tr(m.loadErrorDetailNoStatus))}
    </ErrorPanel>
  )
}

/** Copies text, with a toast either way. */
export async function copy(text: string, title?: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast({ kind: 'success', title: title ?? t(m.copied) })
  } catch {
    toast({ kind: 'error', title: t(m.copyFailed), detail: t(m.copyFailedDetail) })
  }
}

/** Initials for a newsletter tile ("OW" for Ops Weekly). */
export function tileInitials(name: string): string {
  const words = name.replace(/^the\s+/i, '').split(/\s+/).filter(Boolean)
  return words
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}

/** A newsletter's tile: its logo, else its initials. */
export function NewsletterTile({ name, initials, logo, size = 36, me }: { name: string; initials?: string; logo?: string; size?: number; me?: boolean }) {
  return (
    <span className={me ? 'gw-mtile me' : 'gw-mtile'} style={{ width: size, height: size }} aria-hidden="true">
      {logo ? <img src={logo} alt="" /> : (initials || tileInitials(name))}
    </span>
  )
}

/** Refetches every Grow query (after a change: counts and states move together). */
export function useGrowRefresh() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ predicate: (q) => String((q.queryKey as unknown[])[1] ?? '').startsWith(GROW) })
}

/** "84 KB", "2.4 MB", "18 MB". */
export function fileSize(bytes: number): string {
  if (bytes < 1024 * 1024) return t(m.sizeKB, { n: String(Math.max(1, Math.round(bytes / 1024))) })
  const mb = bytes / (1024 * 1024)
  return t(m.sizeMB, { n: mb < 10 ? decimal(mb, 1) : String(Math.round(mb)) })
}

/** "PDF", "ZIP", "Image" from a content type. */
export function fileKind(contentType: string): string {
  if (contentType === 'application/pdf') return t(m.fileTypePdf)
  if (contentType === 'application/zip') return t(m.fileTypeZip)
  if (contentType.startsWith('image/')) return t(m.fileTypeImage)
  return t(m.fileTypeFile)
}

/** Posts a file as multipart form data; resolves with the JSON answer or throws its problem. */
export async function uploadFile<T>(url: string, file: File, field = 'file'): Promise<T> {
  const fd = new FormData()
  fd.append(field, file)
  const r = await fetch(url, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
  const body = (await r.json().catch(() => ({}))) as T & { detail?: string; title?: string; errors?: { message?: string }[] }
  if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || t(m.uploadFailed, { status: r.status }))
  return body
}
