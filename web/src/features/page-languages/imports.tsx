import { $api, errorMessage } from '@/api/client'
import { formatList, useI18n, useT } from '@/i18n'
import { languageLabel } from '@/features/languages/composerParts'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { WarnIcon } from './contacts'
import { mi } from './messages'

/**
 * The line under a source field mapped to the contact's language (Import ›
 * Map fields): what its values become, and the values that aren't
 * languages, which are left empty.
 */
export function ImportLanguageRow({ importId, field, columns }: { importId: string; field: string; columns: number }) {
  const t = useT()
  const { locale } = useI18n()
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/imports/{id}/fields/{field}/languages',
    { params: { path: { workspace: slug, id: importId, field } } },
    { staleTime: 60_000 },
  )
  const d = q.data
  let body
  if (q.isError) {
    body = <span className="k">{t(mi.failed, { error: errorMessage(q.error) })}</span>
  } else if (!d) {
    body = (
      <span className="k" role="status">
        {t(mi.loading)}
      </span>
    )
  } else {
    const normalized = d.normalized.map((v) => t(mi.example, { value: v.value, code: v.code }))
    const unknown = d.unknown.map((v) => t(mi.unknownExample, { value: v.value, rows: v.rows }))
    const langs = d.languages.map((l) => t(mi.languageCount, { language: languageLabel(l.code, locale), rows: count(l.rows) }))
    body = (
      <div className="flex flex-col gap-1">
        {langs.length ? <span className="k">{t(mi.languages, { list: formatList(langs) })}</span> : null}
        {normalized.length ? <span className="k">{t(mi.normalized, { examples: normalized.join(' · ') })}</span> : null}
        {d.unknownValues > 0 ? (
          <span className="imp-lang-unknown">
            <WarnIcon />
            <span>{t(mi.unknown, { values: d.unknownValues, rows: d.unknownRows, examples: unknown.join(' · ') })}</span>
          </span>
        ) : (
          <span className="k">{t(mi.allKnown)}</span>
        )}
      </div>
    )
  }
  return (
    <tr className="imp-lang">
      <td colSpan={columns}>{body}</td>
    </tr>
  )
}
