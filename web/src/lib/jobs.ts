import { useQuery } from '@tanstack/react-query'
import { useSyncExternalStore } from 'react'

/**
 * Background jobs (imports and exports) for the jobs tray (Feedback board).
 * Jobs keep running on the server when the tab closes; the tray shows them
 * while they run and a toast says when one finishes or fails.
 *
 * The server list comes from a JobsSource, set with setJobsSource() (the
 * imports feature installs it). Screens can also track a job they started
 * with trackJob() so it shows right away.
 */
export type JobKind = 'import' | 'export' | 'scoring'
export type JobStatus = 'queued' | 'running' | 'done' | 'failed' | 'canceled'

export interface BackgroundJob {
  id: string
  kind: JobKind
  /** "Importing 12,480 contacts from Mailchimp", "Export · contacts with activity" */
  title: string
  status: JobStatus
  /** 0–100 while running. */
  progress?: number
  /** "1 min left" */
  eta?: string
  /** Result or failure in words: "12,480 contacts imported", "The file has 3 bad rows". */
  detail?: string
  /** Where to see the result (an app path). */
  href?: string
  startedAt?: string
  finishedAt?: string
  /** The toast's title when it finishes or fails, where the kind's own doesn't fit ("Scores recalculated"). */
  doneTitle?: string
  failedTitle?: string
  /** The toast's link to the result ("View"; "Open" by default). */
  openLabel?: string
}

export interface JobsSource {
  list(workspace: string, signal?: AbortSignal): Promise<BackgroundJob[]>
  cancel?(workspace: string, id: string): Promise<void>
}

/** Without a server list, only jobs tracked in this browser show. */
export const stubJobsSource: JobsSource = { list: async () => [] }

let source: JobsSource = stubJobsSource
export function setJobsSource(s: JobsSource) {
  source = s
}
export function getJobsSource(): JobsSource {
  return source
}

let local: Record<string, BackgroundJob[]> = {}
const listeners = new Set<() => void>()
const emit = () => listeners.forEach((l) => l())
const EMPTY: BackgroundJob[] = []

/** Shows a job this screen started (merged with the server's list by id). */
export function trackJob(workspace: string, job: BackgroundJob) {
  const list = local[workspace] ?? []
  local = { ...local, [workspace]: [...list.filter((j) => j.id !== job.id), job] }
  emit()
}

export function updateJob(workspace: string, id: string, patch: Partial<BackgroundJob>) {
  const list = local[workspace]
  if (!list?.some((j) => j.id === id)) return
  local = { ...local, [workspace]: list.map((j) => (j.id === id ? { ...j, ...patch } : j)) }
  emit()
}

export function forgetJob(workspace: string, id: string) {
  const list = local[workspace]
  if (!list) return
  local = { ...local, [workspace]: list.filter((j) => j.id !== id) }
  emit()
}

/** Test helper. */
export function resetLocalJobs() {
  local = {}
  emit()
}

function subscribe(l: () => void) {
  listeners.add(l)
  return () => listeners.delete(l)
}

export const isActive = (j: BackgroundJob) => j.status === 'queued' || j.status === 'running'

/** Server jobs merged with locally tracked ones; polls while any is active. */
export function useBackgroundJobs(workspace: string) {
  const mine = useSyncExternalStore(
    subscribe,
    () => local[workspace] ?? EMPTY,
    () => EMPTY,
  )
  const q = useQuery({
    queryKey: ['background-jobs', workspace],
    queryFn: ({ signal }) => source.list(workspace, signal),
    refetchInterval: (query) => ((query.state.data ?? []).some(isActive) || mine.some(isActive) ? 3000 : 30_000),
    staleTime: 2000,
    retry: false,
  })
  const server = q.data ?? EMPTY
  const byId = new Map<string, BackgroundJob>()
  for (const j of mine) byId.set(j.id, j)
  for (const j of server) byId.set(j.id, j) // the server knows best
  const jobs = [...byId.values()]
  return { jobs, active: jobs.filter(isActive), isError: q.isError }
}
