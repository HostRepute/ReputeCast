import { useEffect, useState } from 'react'
import { http } from '@/api/client'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { m } from './messages'
import { cutHidden, emptyLine, stripMarkers } from './mjml'
import { useConditional, type VisibilityPreview } from './state'
import { versionShows } from './Versions'

export interface PreviewVersion {
  preview: VisibilityPreview | null
  showWhy: boolean
  setShowWhy(v: boolean): void
  /** The contact's version of the preview's HTML (with "Show why" lines). */
  html(full: string): string
}

/**
 * The version a previewed contact gets (ComposerPreview, brief §5.5): which
 * blocks they don't get and why, cut from the preview's HTML.
 */
export function usePreviewVersion(contactId: string | undefined): PreviewVersion {
  const t = useT()
  const c = useComposer()
  const cond = useConditional()
  // The version of the contact it was worked out for.
  const [got, setGot] = useState<{ contactId: string; preview: VisibilityPreview | null } | null>(null)
  const [showWhy, setShowWhy] = useState(false)
  const active = !!cond?.enabled && cond.blocks > 0 && !!contactId
  useEffect(() => {
    if (!active || !contactId) return
    let cancelled = false
    const mjml = c.ge ? c.currentMjml() : c.campaign.mjml
    const html = c.ge ? (c.compileCurrent()?.html ?? '') : c.campaign.html
    void http
      .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/visibility/preview', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: { contactId, mjml, html },
      })
      .then(({ data }) => !cancelled && setGot({ contactId, preview: data ?? null }))
      .catch(() => !cancelled && setGot({ contactId, preview: null }))
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers canvas edits
  }, [active, contactId, c.rev, c.slug, c.campaign.id])
  const preview = active && got?.contactId === contactId ? got.preview : null
  return {
    preview,
    showWhy,
    setShowWhy,
    html(full: string) {
      if (!active || !preview) return stripMarkers(full)
      const hidden = new Set(preview.hidden.map((h) => h.id))
      const why = new Map(preview.hidden.map((h) => [h.id, t(m.hiddenWhy, { name: h.name, why: h.why })]))
      let out = cutHidden(full, hidden, showWhy ? (id) => why.get(id) ?? null : undefined)
      if (preview.empty) {
        const box = emptyLine(t(m.nothingBetween))
        const i = out.lastIndexOf('class="rc-why"')
        const end = i >= 0 ? out.indexOf('</div>', i) : -1
        out = end >= 0 ? out.slice(0, end + 6) + box + out.slice(end + 6) : out.replace(/<body([^>]*)>/i, (mt) => `${mt}${box}`)
      }
      return out
    },
  }
}

/** "Show why", next to "Preview as". */
export function ShowWhyToggle({ v }: { v: PreviewVersion }) {
  const t = useT()
  if (!v.preview) return null
  return (
    <label className="k flex items-center gap-1.5">
      <input type="checkbox" checked={v.showWhy} onChange={(e) => v.setShowWhy(e.target.checked)} aria-label={t(m.showWhy)} />
      {t(m.showWhy)}
    </label>
  )
}

/** "Version 1 of 6 · none of the three conditional blocks · 9,812 recipients get this version". */
export function PreviewVersionLine({ v }: { v: PreviewVersion }) {
  const t = useT()
  const cond = useConditional()
  const p = v.preview
  if (!p) return null
  const blocks = cond?.report?.blocks.length ?? cond?.blocks ?? 0
  const version = cond?.report?.versions.find((x) => x.number === p.version)
  const shows = version ? versionShows(t, version, blocks) : p.empty ? t(m.emptyVersion) : p.shows || t(m.noneOf, { count: blocks })
  return (
    <div className="cv-pv" role="status">
      {p.version ? <span className="badge info">{t(m.versionOf, { number: p.version, total: p.versions })}</span> : null}
      <span>{t(m.versionLine, { shows, count: p.recipients })}</span>
      {!p.inAudience ? <span className="k">{t(m.notInAudience, { name: p.name })}</span> : null}
    </div>
  )
}

/** The preview's "This version" card: who, what it shows, how many get it, its size. */
export function ThisVersionCard({ v }: { v: PreviewVersion }) {
  const t = useT()
  const cond = useConditional()
  const p = v.preview
  if (!p) return null
  const blocks = cond?.report?.blocks.length ?? cond?.blocks ?? 0
  const version = cond?.report?.versions.find((x) => x.number === p.version)
  return (
    <div className="card flex flex-col gap-2" style={{ padding: 12 }}>
      <span className="text-[12px] font-semibold">{t(m.thisVersion)}</span>
      <span>{[p.name, ...p.traits].join(' · ')}</span>
      <dl className="cv-dl">
        <dt>{t(m.versionShows)}</dt>
        <dd>{version ? versionShows(t, version, blocks) : p.shows || t(m.noneOf, { count: blocks })}</dd>
        <dt>{t(m.versionRecipients)}</dt>
        <dd>{count(p.recipients)}</dd>
        {version ? (
          <>
            <dt>{t(m.versionSize)}</dt>
            <dd>{t(m.versionSizeOf, { kb: Math.round(version.sizeBytes / 1024) })}</dd>
          </>
        ) : null}
      </dl>
    </div>
  )
}
