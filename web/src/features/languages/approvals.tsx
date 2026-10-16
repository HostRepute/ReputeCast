import { useId } from 'react'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { mc, mr } from './messages'

/**
 * The approval review's language picker (spec 6, Review & send, approvals):
 * the preview shows one version of the snapshot at a time. Nothing for an
 * email in one language.
 */
export function PreviewLanguagePicker({
  preview,
  onChange,
}: {
  preview: Pick<Schemas['ApprovalPreview'], 'language' | 'languages'> | undefined
  onChange: (language: string) => void
}) {
  const t = useT()
  const id = useId()
  const languages = preview?.languages ?? []
  if (languages.length < 2) return null
  return (
    <span className="inline-flex items-center gap-2">
      <label className="k" htmlFor={id}>
        {t(mr.previewLanguage)}
      </label>
      <select id={id} className="in" style={{ width: 'auto', height: 30 }} value={preview?.language ?? ''} onChange={(e) => onChange(e.target.value)}>
        {languages.map((l) => (
          <option key={l.language} value={l.language}>
            {l.status === 'translated' ? l.name : t(mr.previewOption, { name: l.name, status: t(l.status === 'source' ? mc.source : mc.draft) })}
          </option>
        ))}
      </select>
    </span>
  )
}
