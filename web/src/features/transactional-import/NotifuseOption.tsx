import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
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
        {t(m.templatesOption)}{' '}
        <span className="k">· {t(m.templatesDetail, { transactional: counts.transactional, saved: counts.saved })}</span>
        {counts.skipped > 0 ? <span className="k"> {t(m.templatesSkipped, { skipped: counts.skipped })}</span> : null}
      </span>
    </label>
  )
}
