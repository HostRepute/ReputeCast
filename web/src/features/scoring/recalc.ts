import type { QueryClient } from '@tanstack/react-query'
import { $api, type Schemas } from '@/api/client'
import { formatList, t } from '@/i18n'
import { trackJob, updateJob, type BackgroundJob, type JobStatus } from '@/lib/jobs'
import { m } from './messages'

export type Recalc = Schemas['ScoringRecalcInfo']

export const scoringPath = '/api/v1/workspaces/{workspace}/scoring'

export function scoringQuery(slug: string) {
  return $api.queryOptions('get', scoringPath, { params: { path: { workspace: slug } } })
}

/** "rule 3", "rules 2 and 3, decay and the thresholds", "the first rules". */
export function savedWhat(c: Recalc['changes']): string {
  if (c.setup) return t(m.changedSetup)
  const parts: string[] = []
  if (c.rules.length === 1) parts.push(t(m.changedRule, { n: c.rules[0]! }))
  else if (c.rules.length > 1) parts.push(t(m.changedRules, { list: formatList(c.rules.map(String)) }))
  if (c.decay) parts.push(t(m.changedDecay))
  if (c.bands) parts.push(t(m.changedBands))
  return formatList(parts)
}

/** "1,312 contacts changed band · Sales-ready handoff not started for 1,108". */
export function finishedDetail(rc: Recalc): string {
  const s = rc.summary
  if (!s) return ''
  const parts = [t(m.trayDoneDetail, { count: s.changed })]
  for (const a of s.automations) {
    if (a.crossed > 0) parts.push(t(rc.startAutomations ? m.finishedFlowsStarted : m.finishedFlows, { name: a.name, count: a.crossed }))
  }
  return parts.join(' · ')
}

const statusOf: Record<Recalc['status'], JobStatus> = { running: 'running', finished: 'done', failed: 'failed', superseded: 'canceled' }

/** The recalculation as a background job (the jobs tray and its toasts). */
export function recalcJob(slug: string, rc: Recalc): BackgroundJob {
  const status = statusOf[rc.status]
  const minutes = rc.remainingSeconds !== undefined && rc.remainingSeconds !== null ? Math.max(1, Math.ceil(rc.remainingSeconds / 60)) : undefined
  return {
    id: `scoring-${rc.id}`,
    kind: 'scoring',
    title: t(rc.reason === 'setup' ? m.trayTitleSetup : m.trayTitle),
    status,
    progress: rc.total ? Math.min(100, (rc.done / rc.total) * 100) : 0,
    eta: status === 'running' && minutes ? t(m.trayEta, { minutes }) : undefined,
    detail: status === 'done' ? finishedDetail(rc) : status === 'failed' ? t(m.trayFailedDetail) : undefined,
    href: `/w/${slug}/audience/scoring`,
    startedAt: rc.startedAt,
    finishedAt: rc.finishedAt ?? rc.failedAt,
    doneTitle: t(m.trayDone),
    failedTitle: t(m.trayFailed),
    openLabel: t(m.viewAction),
  }
}

const following = new Set<string>()

/**
 * Follows a running recalculation until it ends, for the jobs tray and the
 * toast that says it finished, even after leaving the page. Each check
 * refreshes the page's data, so an open Scoring page shows the progress.
 */
export function followRecalc(qc: QueryClient, slug: string, rc: Recalc) {
  if (rc.status !== 'running' || following.has(rc.id)) return
  following.add(rc.id)
  const id = `scoring-${rc.id}`
  trackJob(slug, recalcJob(slug, rc))
  const tick = async () => {
    try {
      const page = await qc.fetchQuery({ ...scoringQuery(slug), staleTime: 0 })
      const now = page.recalculation
      if (!now || now.id !== rc.id) {
        updateJob(slug, id, { status: 'canceled' })
        following.delete(rc.id)
        return
      }
      updateJob(slug, id, recalcJob(slug, now))
      if (now.status === 'running') window.setTimeout(() => void tick(), 2000)
      else following.delete(rc.id)
    } catch {
      window.setTimeout(() => void tick(), 6000)
    }
  }
  window.setTimeout(() => void tick(), 2000)
}
