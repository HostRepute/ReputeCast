import { Link, useRouter, useRouterState } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { isActive, useBackgroundJobs, type BackgroundJob, type JobStatus } from '@/lib/jobs'
import { useWorkspace } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { toast } from '../ui/Toast'

function meta(j: BackgroundJob): string {
  if (j.status === 'queued') return 'queued'
  const pct = j.progress !== undefined ? `${Math.round(j.progress)}%` : 'running'
  return j.eta ? `${pct} · ${j.eta}` : pct
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

  // Toast when a job this page watched finishes or fails.
  useEffect(() => {
    for (const j of jobs) {
      const before = seen.current.get(j.id)
      seen.current.set(j.id, j.status)
      if (!before || !isActive({ status: before } as BackgroundJob) || isActive(j)) continue
      if (j.status === 'done') {
        toast({
          kind: 'success',
          title: `${j.kind === 'import' ? 'Import' : 'Export'} finished`,
          detail: j.detail,
          action: j.href ? { label: 'Open', onClick: () => router.history.push(j.href!) } : undefined,
        })
      } else if (j.status === 'failed') {
        toast({ kind: 'error', title: `${j.title} failed`, detail: j.detail ?? 'see the job history for what went wrong' })
      }
    }
  }, [jobs, router])

  if (!active.length) return null
  return (
    <section className="tray" aria-label="Background jobs">
      <div className="th">
        <button type="button" aria-expanded={open} onClick={() => setOpen(!open)}>
          Background jobs · {active.length}
          <Icon name={open ? 'chevronDown' : 'chevronUp'} size={14} />
        </button>
        <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} className="k">
          History
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
          <span className="k">Jobs keep running if you close the tab. You get a toast here and an email when one finishes or fails.</span>
        </>
      ) : null}
    </section>
  )
}
