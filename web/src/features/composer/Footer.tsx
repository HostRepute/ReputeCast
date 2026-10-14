import type { Component } from 'grapesjs'
import { Fragment } from 'react'
import { $api } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useComposer } from './context'
import { ancestry, mjChildren, sectionLabel, tagOf } from './editor/labels'
import { kb } from './format'
import { contentIssues } from './RightPanel'

function plural(n: number, one: string, many = `${one}s`) {
  return `${n} ${n === 1 ? one : many}`
}

/** "mj-section · hero" for the current element; plain tag for the others. */
function crumbLabel(c: Component, current: boolean): string {
  const tag = tagOf(c)
  if (!current) return tag
  if (tagOf(c.parent()) === 'mj-body') {
    const l = sectionLabel(c)
    return /^\d+ columns$/.test(l) ? tag : `${tag} · ${l}`
  }
  return tag
}

/** "mj-column × 2" when an element's children are all the same tag. */
function childrenSummary(c: Component): string {
  const kids = mjChildren(c)
  if (kids.length < 2) return ''
  const tag = tagOf(kids[0]!)
  return kids.every((k) => tagOf(k) === tag) ? `${tag} × ${kids.length}` : ''
}

function Breadcrumb() {
  const c = useComposer()
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  if (!sel) {
    return (
      <div className="bc">
        <span>mjml</span>
        <span aria-hidden="true">›</span>
        <span>mj-body</span>
        <span className="k">· select a component on the canvas</span>
      </div>
    )
  }
  const chain = ancestry(sel)
  const kids = childrenSummary(sel)
  return (
    <nav className="bc" aria-label="Selected component">
      {chain.map((x, i) => {
        const current = x === sel
        return (
          <Fragment key={x.cid}>
            {i ? <span aria-hidden="true">›</span> : null}
            {current ? (
              <b aria-current="true">{crumbLabel(x, true)}</b>
            ) : tagOf(x) === 'mjml' ? (
              <span>mjml</span>
            ) : (
              <button type="button" onClick={() => c.editor?.select(x)}>
                {crumbLabel(x, false)}
              </button>
            )}
          </Fragment>
        )
      })}
      {kids ? (
        <>
          <span aria-hidden="true">›</span>
          <span>{kids}</span>
        </>
      ) : null}
    </nav>
  )
}

/** Size, images and links of the whole email, from the last check. */
export function SizeSummary() {
  const c = useComposer()
  const r = c.checks
  if (!r) return <span className="k" style={{ marginLeft: 'auto' }}>Checking…</span>
  const issues = contentIssues(r.issues)
  const images = !r.images ? 'no images' : r.imagesMissingAlt ? `${r.imagesMissingAlt} of ${plural(r.images, 'image')} missing alt text` : `${plural(r.images, 'image')}, all with alt text`
  const extra = [r.linksBlocklisted ? `${r.linksBlocklisted} blocklisted` : '', r.linksBroken ? `${r.linksBroken} broken` : ''].filter(Boolean).join(', ')
  const links = `${plural(r.links, 'link')}${extra ? `, ${extra}` : ''}`
  const near = r.sizeBytes >= 85 * 1024
  return (
    <span className="k" style={{ marginLeft: 'auto', ...(issues.length && (near || r.imagesMissingAlt || extra) ? { color: '#92400E', fontWeight: 600 } : {}) }}>
      {kb(r.sizeBytes)} of 102 KB · {images} · {links}
    </span>
  )
}

export function ChecksBadge() {
  const c = useComposer()
  const r = c.checks
  if (!r) return null
  const issues = contentIssues(r.issues)
  const blocks = issues.filter((i) => i.severity === 'block').length
  const warns = issues.length - blocks
  if (!issues.length)
    return (
      <Badge tone="ok" dot>
        No warnings
      </Badge>
    )
  const text = [blocks ? `${blocks} blocking` : '', warns ? plural(warns, 'warning') : ''].filter(Boolean).join(' · ')
  return (
    <Badge tone={blocks ? 'bad' : 'warn'} dot>
      {text}
    </Badge>
  )
}

function VersionLink() {
  const c = useComposer()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions', { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } }, { staleTime: 30_000 })
  const n = q.data?.total ?? c.campaign.versionCount
  return (
    <button type="button" className="k" style={{ border: 0, background: 'none', padding: 0, fontFamily: 'inherit', lineHeight: 'inherit', fontSize: 12, color: 'var(--ink-3)', cursor: 'pointer', minHeight: 24, display: 'inline-flex', alignItems: 'center' }} onClick={() => c.openDialog('versions')}>
      Version history · {plural(n, 'version')}
    </button>
  )
}

export function Footer() {
  const c = useComposer()
  if (c.isEmpty) {
    return (
      <footer className="cmp-foot">
        <span className="k">Nothing on the canvas yet</span>
        <span className="k" style={{ marginLeft: 'auto' }}>
          0 KB of 102 KB
        </span>
      </footer>
    )
  }
  const code = c.view === 'mjml' || c.view === 'html'
  return (
    <footer className="cmp-foot">
      {code ? <span className="k">{c.view === 'mjml' ? 'Editing MJML directly · the Visual view re-parses on Apply' : 'Compiled HTML · read only · what the providers receive, before tracking'}</span> : <Breadcrumb />}
      <SizeSummary />
      <ChecksBadge />
      {code ? null : <VersionLink />}
    </footer>
  )
}
