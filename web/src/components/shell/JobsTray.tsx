import { Link, useRouter, useRouterState } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { t, useT } from '@/i18n'
import { isActive, useBackgroundJobs, type BackgroundJob, type JobStatus } from '@/lib/jobs'
import { useWorkspace } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { toast } from '../ui/Toast'
import { m } from './messages'

function meta(j: BackgroundJob): string {
  if (j.status === 'queued') return t(m.jobQueued)
  const pct = j.progress !== undefined ? t(m.jobPercent, { pct: Math.round(j.progress) }) : t(m.jobRunning)
  return j.eta ? t(m.jobWithEta, { progress: pct, eta: j.eta }) : pct
}

/**
 * Background jobs (Feedback board): imports and exports that keep running
 * when the tab closes. The tray shows while any runs; when one finishes or
 * fails, a toast says so (and an email goes out from the server).
 */
export function JobsTray() {
  const { slug } = useWorkspace()
  const router = useRouter()
  const { jobs, active: running } = useBackgroundJobs(slug)
  const [open, setOpen] = useState(true)
  // A job's own page already shows its progress (MigrationProgress has no tray).
  const path = useRouterState({ select: (s) => s.location.pathname })
  const active = running.filter((j) => j.href !== path)
  const seen = useRef(new Map<string, JobStatus>())
  const tr = useT()

  useEffect(() => {
    for (const j of jobs) {
      const before = seen.current.get(j.id)
      seen.current.set(j.id, j.status)
      if (!before || !isActive({ status: before } as BackgroundJob) || isActive(j)) continue
      if (j.status === 'done') {
        toast({
          kind: 'success',
          title: t(j.kind === 'import' ? m.jobImportFinished : m.jobExportFinished),
          detail: j.detail,
          action: j.href ? { label: t(m.jobOpen), onClick: () => router.history.push(j.href!) } : undefined,
        })
      } else if (j.status === 'failed') {
        toast({ kind: 'error', title: t(m.jobFailed, { title: j.title }), detail: j.detail ?? t(m.jobFailedDetail) })
      }
    }
  }, [jobs, router])

  if (!active.length) return null
  return (
    <section className="tray" aria-label={tr(m.jobsLabel)}>
      <div className="th">
        <button type="button" aria-expanded={open} onClick={() => setOpen(!open)}>
          {tr(m.jobsToggle, { count: active.length })}
          <Icon name={open ? 'chevronDown' : 'chevronUp'} size={14} />
        </button>
        <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} className="k">
          {tr(m.jobsHistory)}
        </Link>
      </div>
      {open ? (
        <>
          {active.map((j) => (
            <div className="job" key={j.id}>
              <div className="jh">
                <span>{j.title}</span>
                <span className="k shrink-0">{meta(j)}</span>
              </div>
              <div className="bar" role="progressbar" aria-label={j.title} aria-valuemin={0} aria-valuemax={100} aria-valuenow={j.status === 'queued' ? 0 : Math.round(j.progress ?? 0)}>
                <i style={{ width: `${j.status === 'queued' ? 0 : Math.max(0, Math.min(100, j.progress ?? 0))}%` }} />
              </div>
            </div>
          ))}
          <span className="k">{tr(m.jobsFootnote)}</span>
        </>
      ) : null}
    </section>
  )
}
