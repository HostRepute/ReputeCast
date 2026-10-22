import { useQueryClient } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Icon } from '@/components/icons/Icon'
import { getI18n, t as tt, useT } from '@/i18n'
import { formatDecimal } from '@/i18n/regions'
import { clock, shortDate, timeOfDay } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

export type Digest = Schemas['DigestDetail']
export type DigestListItem = Schemas['DigestListItem']
export type DigestSchedule = Schemas['DigestSchedule']
export type DigestIssue = Schemas['DigestIssue']
export type DigestNextIssue = Schemas['DigestNextIssue']
export type FeedItem = Schemas['DigestFeedItem']
export type HistoryItem = Schemas['DigestHistoryItem']
export type IssueStatus = DigestIssue['status']
export type IssueInfo = Schemas['DigestIssueInfo']

/**
 * The digest issues among some campaigns, for the screens that show a
 * campaign (approvals, Review & send, the report, the composer).
 */
export function useIssueInfos(campaignIds: readonly string[]) {
  const { slug } = useWorkspace()
  const ids = [...new Set(campaignIds.filter(Boolean))].sort().slice(0, 100)
  return $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/digest-issues',
    { params: { path: { workspace: slug }, query: { campaigns: ids.join(',') } } },
    { enabled: ids.length > 0, staleTime: 15_000 },
  )
}

/** The digest issue a campaign is, if it is one. */
export function useIssueInfo(campaignId: string | undefined): IssueInfo | undefined {
  const q = useIssueInfos(campaignId ? [campaignId] : [])
  return q.data?.items.find((x) => x.campaignId === campaignId)
}

/** The digest's queries, keyed as openapi-react-query keys them. */
export function useDigest(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/{digestId}', { params: { path: { workspace: slug, digestId: id } } })
}

export function useDigestIssues(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/{digestId}/issues', { params: { path: { workspace: slug, digestId: id } } })
}

export function useDigestHistory(id: string, limit = 5) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/{digestId}/history', { params: { path: { workspace: slug, digestId: id }, query: { limit } } })
}

export function useDigestFeed(id: string, limit = 6) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/{digestId}/feed', { params: { path: { workspace: slug, digestId: id }, query: { limit } } })
}

/** Refreshes everything a digest screen shows after an action. */
export function useRefreshDigest() {
  const qc = useQueryClient()
  return () => {
    for (const key of [
      '/api/v1/workspaces/{workspace}/digests',
      '/api/v1/workspaces/{workspace}/digests/{digestId}',
      '/api/v1/workspaces/{workspace}/digests/{digestId}/issues',
      '/api/v1/workspaces/{workspace}/digests/{digestId}/history',
      '/api/v1/workspaces/{workspace}/digests/{digestId}/feed',
      '/api/v1/workspaces/{workspace}/campaigns',
      '/api/v1/workspaces/{workspace}/digest-issues',
      // An issue's Review & send and its approval request.
      '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/review',
      '/api/v1/workspaces/{workspace}/approvals',
    ]) {
      void qc.invalidateQueries({ queryKey: ['get', key] })
    }
  }
}

/** The schedule in words: "Every Friday at 08:00 America/New_York" (on the reader's clock), or its short form for tables. */
export function scheduleText(s: DigestSchedule, form: boolean | 'lower' = false): string {
  const tz = s.timezone
  const time = timeOfDay(s.time)
  const day = tt(m.weekday, { day: String(s.weekday) })
  if (form === 'lower') return tt(m.scheduleLower, { frequency: s.frequency, weekdays: s.weekdaysOnly ? 'yes' : 'no', rule: s.monthRule === 'first' ? 'first' : 'day', day, monthDay: s.monthDay ?? 1, time, timezone: tz, count: s.count ?? 3 })
  const short = form === true
  switch (s.frequency) {
    case 'daily':
      if (s.weekdaysOnly) return short ? tt(m.scheduleWeekdaysShort, { time }) : tt(m.scheduleWeekdays, { time, timezone: tz })
      return short ? tt(m.scheduleDailyShort, { time }) : tt(m.scheduleDaily, { time, timezone: tz })
    case 'monthly':
      if (s.monthRule === 'first') return short ? tt(m.scheduleMonthlyFirstShort, { day, time }) : tt(m.scheduleMonthlyFirst, { day, time, timezone: tz })
      return short ? tt(m.scheduleMonthlyDayShort, { day: s.monthDay ?? 1, time }) : tt(m.scheduleMonthlyDay, { day: s.monthDay ?? 1, time, timezone: tz })
    case 'items':
      return short
        ? tt(m.scheduleItemsShort, { count: s.count ?? 3 })
        : tt(m.scheduleItems, { count: s.count ?? 3, from: timeOfDay(s.windowFrom || '08:00'), until: timeOfDay(s.windowUntil || '18:00'), timezone: tz })
  }
  return short ? tt(m.scheduleWeeklyShort, { day, time }) : tt(m.scheduleWeekly, { day, time, timezone: tz })
}

const statusTone: Record<IssueStatus, Tone> = {
  collecting: 'neutral',
  preparing: 'bad',
  prepared: 'info',
  waiting: 'warn',
  held: 'bad',
  draft: 'info',
  sending: 'info',
  sent: 'ok',
  skipped: 'neutral',
  not_sent: 'neutral',
}

/** The issue status badges of the design system: Collecting (neutral), Prepared (info), Waiting for approval (warn), Held (bad), Sending (info, pulse), Sent (ok), Skipped and Not sent (neutral). */
export function IssueStatusBadge({ status, feedFailing }: { status: IssueStatus; feedFailing?: boolean }) {
  const t = useT()
  if (status === 'preparing' || feedFailing) {
    return (
      <Badge tone="bad" dot>
        {t(m.feedUnreachableBadge)}
      </Badge>
    )
  }
  const dot = status === 'waiting' || status === 'held' || status === 'sent' || status === 'sending'
  return (
    <Badge tone={statusTone[status]} dot={dot && status !== 'sending'} pulse={status === 'sending'}>
      {t(m.issueStatus, { status })}
    </Badge>
  )
}

/** "On · Send automatically", "Paused", "Off". */
export function DigestStateBadge({ state, mode }: { state: Digest['state']; mode: Digest['mode'] }) {
  const t = useT()
  if (state === 'on') {
    return (
      <Badge tone="ok" dot>
        {t(m.stateOn, { mode })}
      </Badge>
    )
  }
  if (state === 'paused') {
    return (
      <Badge tone="warn" dot>
        {t(m.statePaused)}
      </Badge>
    )
  }
  return <Badge tone="neutral">{t(m.stateOff)}</Badge>
}

/** The "Digest" badge on an issue's campaign row: it links to its digest. */
export function DigestIcon({ size = 12 }: { size?: number }) {
  return <Icon name="repeat" size={size} />
}

/** "Fri 17 Oct, 08:00" */
export function dateTime(d: string): string {
  return `${shortDate(d)}, ${clock(d)}`
}

/** A signed number: "+31", "−1.4" (with a real minus sign), "−1,4" in a region format that writes a decimal comma. */
export function signed(v: number, digits = 0): string {
  const region = getI18n().regionFormat
  const s = region === 'standard' ? Math.abs(v).toFixed(digits) : formatDecimal(region, Math.abs(v), digits)
  if (v > 0) return `+${s}`
  if (v < 0) return `−${s}`
  return s
}
