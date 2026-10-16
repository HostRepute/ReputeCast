import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { getI18n, useT } from '@/i18n'
import { m } from './messages'
import './accessibility.css'

type Result = Schemas['AccessibilityTemplateResult']

/**
 * The template gallery's accessibility results (ComposerEmpty, section 4):
 * every starter, saved template and the last campaign, checked in the
 * workspace's language.
 */
export function useGalleryResults(slug: string, enabled: boolean) {
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/accessibility/templates', { params: { path: { workspace: slug } } }, { enabled, staleTime: 60_000 })
  const byId = new Map<string, Result>()
  for (const r of q.data?.templates ?? []) byId.set(r.id, r)
  return { data: q.data, byId, last: q.data?.lastCampaign }
}

/** "Checked in English, the templates' language · built-in templates and starters are all clear". */
export function GalleryCaption({ data }: { data?: Schemas['AccessibilityGallery'] }) {
  const t = useT()
  if (!data) return null
  const starters = data.templates.filter((r) => r.id.startsWith('starter:'))
  const language = new Intl.DisplayNames([getI18n().locale], { type: 'language' }).of(data.language) ?? data.language
  return <span className="k">{starters.every((r) => !r.issues) ? t(m.galleryChecked, { language }) : t(m.galleryCheckedSome, { language })}</span>
}

/** A tile's result: "All clear", or "2 issues · 1 serious" and what the serious ones are. */
export function GalleryResult({ result }: { result?: Result }) {
  const t = useT()
  if (!result) return null
  if (!result.issues)
    return (
      <Badge tone="ok" className="a11y-gres">
        {t(m.allClear)}
      </Badge>
    )
  return (
    <>
      <Badge tone="warn" className="a11y-gres">
        {result.serious ? t(m.galleryIssuesSerious, { count: result.issues, serious: result.serious }) : t(m.galleryIssues, { count: result.issues })}
      </Badge>
      {result.summary ? <span className="k text-ink-nav">{result.summary}</span> : null}
    </>
  )
}

/** The class of a tile with issues: an amber border, as on the board. */
export function galleryTileClass(result?: Result): string {
  return result?.issues ? 'tpl a11y-tpl-warn' : 'tpl'
}
