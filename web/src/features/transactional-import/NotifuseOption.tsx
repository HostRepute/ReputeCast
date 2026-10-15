import type { Schemas } from '@/api/client'
import { ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './notifuse.messages'

/**
 * "Import templates" in the importer's "Choose what to import" card, for a
 * Notifuse workspace: its transactional templates arrive as drafts, the
 * others as saved templates.
 */
export function NotifuseTemplatesOption({
  counts,
  checked,
  onChange,
}: {
  counts: Schemas['ImportTemplateCounts']
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  const t = useT()
  if (counts.transactional + counts.saved === 0) return null
  return (
    <label className="flex items-start gap-2.5 text-[13px]">
      <input type="checkbox" className="mt-[3px]" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span>
        {t(m.templatesOption)} <span className="k">· {t(m.templatesDetail, { transactional: counts.transactional, saved: counts.saved })}</span>
        {counts.skipped > 0 ? <span className="k"> {t(m.templatesSkipped, { skipped: counts.skipped })}</span> : null}
      </span>
    </label>
  )
}

/** The templates a Notifuse import brought along, on its summary. */
export function NotifuseTemplatesDone({ stats }: { stats: Schemas['ImportTemplateStats'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (stats.transactional + stats.saved === 0) return null
  return (
    <Note tone={stats.manual > 0 ? 'warn' : 'info'}>
      <span className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <span>
          {t(m.doneTemplates, { transactional: stats.transactional, saved: stats.saved })}
          {stats.manual > 0 ? ` ${t(m.doneTemplatesManual, { manual: stats.manual })}` : ''}
        </span>
        {stats.transactional > 0 ? (
          <ButtonLink size="sm" to="/w/$workspace/transactional" params={{ workspace: slug }}>
            {t(m.doneTemplatesLink)}
          </ButtonLink>
        ) : null}
      </span>
    </Note>
  )
}
