import { useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { clock, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { t as tt, useT } from '@/i18n'
import { stepPath, useRefreshImport, type Detail } from './ImportFrame'
import { m } from './messages'

/** Loading · reading the source (MigrationStates). */
export function ReadingSource({ title, detail }: { title: string; detail?: string }) {
  return (
    <div className="card flex flex-col gap-3 px-5 py-4.5" aria-busy="true" role="status">
      <div className="flex flex-wrap items-center gap-3">
        <span className="spin" aria-hidden="true" />
        <span className="font-semibold">{title}</span>
        {detail ? <span className="k">{detail}</span> : null}
      </div>
      <div className="flex gap-3" aria-hidden="true">
        <span className="sk h-9 w-[220px] max-w-full" />
        <span className="sk h-9 w-[120px]" />
        <span className="sk h-9 w-[160px] max-md:hidden" />
      </div>
      <div className="flex flex-col gap-2" aria-hidden="true">
        <span className="sk h-3.5 w-[90%]" />
        <span className="sk h-3.5 w-[70%]" />
        <span className="sk h-3.5 w-[80%]" />
      </div>
    </div>
  )
}

/** What reading the source is doing now: "1 of 1 audiences · 14 fields · 9 tags". */
export function readingDetail(d: Detail) {
  const lists = d.connection?.lists ?? []
  const picked = new Set(d.options.lists.map((l) => l.sourceId))
  const chosen = lists.filter((l) => picked.has(l.id))
  const unit = d.source === 'mailchimp' ? 'audience' : d.source === 'mailerlite' ? 'group' : d.source === 'keila' ? 'segment' : 'list'
  const fields = chosen.reduce((n, l) => Math.max(n, l.fields), 0)
  const tags = chosen.reduce((n, l) => n + l.tags, 0)
  return [
    tt(m.readingLists, { chosen: chosen.length, total: lists.length, unit }),
    fields ? tt(m.readingFields, { n: fields }) : '',
    tags ? tt(m.readingTags, { n: tags }) : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

/** Paused · the source is rate-limiting us (MigrationStates). */
export function PausedCard({ d, now }: { d: Detail; now: number }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshImport()
  const resume = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/resume')
  const [err, setErr] = useState<string | null>(null)
  const go = async (slow: boolean) => {
    setErr(null)
    try {
      await resume.mutateAsync({ params: { path: { workspace: slug, id: d.id }, query: slow ? { slow: true } : {} } })
      await refresh()
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const at = d.error?.retryAt
  const row = d.error?.row ?? (d.stagedRows > 0 ? d.stagedRows : undefined)
  const wait = at ? Math.max(0, Math.round((new Date(at).getTime() - now) / 1000)) : undefined
  return (
    <div role="alert" className="card flex flex-col gap-2 border-[#FCD34D] px-4.5 py-4">
      <div className="flex items-center gap-2.5">
        <Badge tone="warn">{t(m.paused)}</Badge>
        <span className="font-semibold">{row ? t(m.rateLimitedAt, { source: d.sourceName, row: count(row) }) : t(m.rateLimited, { source: d.sourceName })}</span>
      </div>
      <span className="k">
        {t(m.backsOff, {
          when: wait !== undefined ? (wait > 0 ? 'seconds' : 'now') : at ? 'at' : 'later',
          n: wait ?? 0,
          time: at ? clock(at) : '',
        })}
      </span>
      {err ? <Note tone="bad">{err}</Note> : null}
      <div className="flex flex-wrap gap-2">
        <Button size="sm" loading={resume.isPending && !resume.variables?.params.query?.slow} onClick={() => void go(false)}>
          {t(m.resumeNow)}
        </Button>
        {d.options.pace !== 3 ? (
          <Button size="sm" loading={resume.isPending && Boolean(resume.variables?.params.query?.slow)} onClick={() => void go(true)}>
            {t(m.slowDown)}
          </Button>
        ) : null}
      </div>
    </div>
  )
}

/** Reading the source stopped with an error: say why, offer the way back. */
export function SourceFailed({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const e = d.error
  return (
    <div role="alert" className="card flex flex-col gap-2 border-[#FCA5A5] px-4.5 py-4">
      <div className="flex items-center gap-2.5">
        <Badge tone="bad">{e?.status ? String(e.status) : t(m.error)}</Badge>
        <span className="font-semibold">{e?.code === 'source_auth_failed' ? t(m.keyRejected, { source: d.sourceName }) : t(m.readFailed, { source: d.sourceName })}</span>
      </div>
      <span className="k">
        {e?.message ?? t(m.stoppedAnswering)} {t(m.nothingImported)}
      </span>
      <div className="flex flex-wrap gap-2">
        <ButtonLink size="sm" variant="primary" to={stepPath.source} params={{ workspace: slug, importId: d.id }}>
          {t(m.backToSource)}
        </ButtonLink>
      </div>
    </div>
  )
}
