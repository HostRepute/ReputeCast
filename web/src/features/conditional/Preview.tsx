import { useEffect, useState } from 'react'
import { http } from '@/api/client'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { EyeOff } from './Canvas'
import { m } from './messages'
import { cutHidden, emptyLine, stripMarkers } from './mjml'
import { useConditional, type VisibilityPreview } from './state'
import { versionShows } from './Versions'
import './conditional.css'

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
      if (!preview) return stripMarkers(full)
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
    <label className="cv-switch">
      <input type="checkbox" role="switch" checked={v.showWhy} onChange={(e) => v.setShowWhy(e.target.checked)} />
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
    <div className={`cv-pv${p.empty ? ' empty' : ''}`} role="status">
      <EyeOff size={14} />
      {p.version ? <b>{t(m.versionOf, { number: p.version, total: p.versions })}</b> : null}
      <span>{t(m.versionLine, { shows, count: p.recipients })}</span>
      {!p.inAudience ? <span>{t(m.notInAudience, { name: p.name })}</span> : null}
    </div>
  )
}

/** Links in a version's HTML (the email as the contact gets it). */
function linksIn(html: string): number {
  return (html.match(/<a\s[^>]*href="(?:https?:|#poll-|#survey-)/gi) ?? []).length
}

/** The preview's "This version": who, what it shows, how many get it, its size and links. */
export function ThisVersionCard({ v, html }: { v: PreviewVersion; html?: string }) {
  const t = useT()
  const cond = useConditional()
  const p = v.preview
  if (!p) return null
  const blocks = cond?.report?.blocks.length ?? cond?.blocks ?? 0
  const version = cond?.report?.versions.find((x) => x.number === p.version)
  const links = html ? linksIn(v.html(html)) : null
  return (
    <section className="flex flex-col gap-2.5 border-b border-line-2" style={{ padding: '14px 16px' }} aria-labelledby="cv-this-h">
      <div className="flex flex-col gap-0.5">
        <h2 className="m-0 text-[14px] font-semibold" id="cv-this-h">
          {t(m.thisVersion)}
        </h2>
        <span className="k">{[p.name, ...p.traits].join(' · ')}</span>
      </div>
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
        {links !== null ? (
          <>
            <dt>{t(m.versionLinks)}</dt>
            <dd>{t(m.versionLinksCount, { count: links })}</dd>
          </>
        ) : null}
      </dl>
    </section>
  )
}

/** The preview's "Versions · 6": each combination, the previewed one marked. */
export function VersionsCard({ v }: { v: PreviewVersion }) {
  const t = useT()
  const c = useComposer()
  const cond = useConditional()
  const r = cond?.report
  if (!v.preview || !r?.counted || !r.versions.length) return null
  const blocks = r.blocks.length
  const shown = r.versions.length > 8 ? r.versions.slice(0, 8) : r.versions
  return (
    <section className="flex flex-col border-b border-line-2" aria-labelledby="cv-versions-h">
      <div className="flex items-start justify-between gap-3" style={{ padding: '14px 16px 10px' }}>
        <div className="flex flex-col gap-0.5">
          <h2 className="m-0 text-[14px] font-semibold" id="cv-versions-h">
            {t(m.versionsCard, { count: r.versions.length })}
          </h2>
          <span className="k">{t(m.versionsCardSub)}</span>
        </div>
        <button
          type="button"
          className="btn ghost sm"
          style={{ color: 'var(--accent)' }}
          onClick={() => {
            c.setView('visual')
            cond?.setVersionsOpen(true)
          }}
        >
          {t(m.openVersions)}
        </button>
      </div>
      <div className="cv-versions">
        {shown.map((x) => (
          <div key={x.number} className={x.overClip || x.empty ? 'hl' : x.number === v.preview?.version ? 'now' : undefined}>
            <b>{t(m.versionNumber, { number: x.number })}</b>
            <span>{versionShows(t, x, blocks)}</span>
            <b className="r">{count(x.recipients)}</b>
            <span />
            <span className="k">{x.number === v.preview?.version ? t(m.previewedNow) : x.overClip ? t(m.gmailClips) : ''}</span>
            <span className="k r" style={x.overClip ? { color: 'var(--warn-fg)', fontWeight: 600 } : undefined}>
              {Math.round(x.sizeBytes / 1024)} KB
            </span>
          </div>
        ))}
      </div>
    </section>
  )
}
