import type { Component } from 'grapesjs'
import { DigestFooterSummary, useDigestMode } from '@/features/digests/composer/DigestMode'
import { Fragment } from 'react'
import { $api } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useVersionName } from '@/features/languages/composerParts'
import { useComposer } from './context'
import { ancestry, mjChildren, sectionLabel, sectionWord, tagOf } from './editor/labels'
import { kb } from './format'
import { useT } from '@/i18n'
import { m } from './messages'
import { contentIssues } from './RightPanel'
import { useAccessibility } from '@/features/accessibility/composer/state'
import { AccessibilityChip, AccessibilityClear, useAccessibilityIssueCount } from '@/features/accessibility/composer/Status'
import { ConditionalFooter, useVersionSizes } from '@/features/conditional/Versions'

/** "mj-section · hero" for the current element; plain tag for the others. */
function crumbLabel(c: Component, current: boolean): string {
  const tag = tagOf(c)
  if (!current) return tag
  if (tagOf(c.parent()) === 'mj-body') {
    const l = sectionLabel(c)
    return /^\d+ columns$/.test(l) ? tag : `${tag} · ${sectionWord(l)}`
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

export function Breadcrumb() {
  const c = useComposer()
  const t = useT()
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  if (!sel) {
    return (
      <div className="bc">
        <span>{'mjml'}</span>
        <span aria-hidden="true">›</span>
        <span>{'mj-body'}</span>
        <span className="k">{t(m.selectOnCanvas)}</span>
      </div>
    )
  }
  const chain = ancestry(sel)
  const kids = childrenSummary(sel)
  return (
    <nav className="bc" aria-label={t(m.selectedComponent)}>
      {chain.map((x, i) => {
        const current = x === sel
        return (
          <Fragment key={x.cid}>
            {i ? <span aria-hidden="true">›</span> : null}
            {current ? (
              <b aria-current="true">{crumbLabel(x, true)}</b>
            ) : tagOf(x) === 'mjml' ? (
              <span>{'mjml'}</span>
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
  const t = useT()
  // A digest's template: the next issue's size and the largest issue's (Feed items).
  const digest = useDigestMode()
  // The Accessibility chip covers alt text when its checks run.
  const a11y = useAccessibility()?.enabled
  const language = useVersionName()
  // With conditional content no one gets the whole email: each version's size.
  const sizes = useVersionSizes()
  if (digest?.render) {
    return (
      <span style={{ marginLeft: 'auto' }}>
        <DigestFooterSummary />
      </span>
    )
  }
  const r = c.checks
  if (!r) return <span className="k" style={{ marginLeft: 'auto' }}>{t(m.checking)}</span>
  const issues = contentIssues(r.issues)
  const images = !r.images
    ? t(m.noImages)
    : a11y
      ? t(m.images, { count: r.images })
      : r.imagesMissingAlt
        ? t(m.imagesMissingAlt, { missing: r.imagesMissingAlt, count: r.images })
        : t(m.imagesAllAlt, { count: r.images })
  const extra = r.linksBlocklisted || r.linksBroken
  const links = t(m.links, { count: r.links, blocklisted: r.linksBlocklisted, broken: r.linksBroken })
  const near = r.sizeBytes >= 85 * 1024
  return (
    <span className="k" style={{ marginLeft: 'auto', ...(issues.length && (near || (!a11y && r.imagesMissingAlt) || extra) ? { color: '#92400E', fontWeight: 600 } : {}) }}>
      {language ? `${language} · ` : ''}
      {sizes ?? t(m.sizeOf, { size: kb(r.sizeBytes) })} · {images} · {links}
      <AccessibilityClear />
    </span>
  )
}

export function ChecksBadge() {
  const c = useComposer()
  const t = useT()
  const a11y = useAccessibilityIssueCount()
  const r = c.checks
  if (!r) return null
  const issues = contentIssues(r.issues)
  const blocks = issues.filter((i) => i.severity === 'block').length
  const warns = issues.length - blocks
  // With accessibility issues the status bar shows their chip instead.
  if (!issues.length && a11y) return null
  if (!issues.length)
    return (
      <Badge tone="ok" dot>
        {t(m.noWarnings)}
      </Badge>
    )
  const text = blocks && warns ? t(m.blockingAndWarnings, { blocks, warnings: warns }) : blocks ? t(m.blocking, { count: blocks }) : t(m.warnings, { count: warns })
  return (
    <Badge tone={blocks ? 'bad' : 'warn'} dot>
      {text}
    </Badge>
  )
}

export function VersionLink() {
  const c = useComposer()
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions', { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } }, { staleTime: 30_000 })
  const n = q.data?.total ?? c.campaign.versionCount
  return (
    <button type="button" className="k" style={{ border: 0, background: 'none', padding: 0, fontFamily: 'inherit', lineHeight: 'inherit', fontSize: 12, color: 'var(--ink-3)', cursor: 'pointer', minHeight: 24, display: 'inline-flex', alignItems: 'center' }} onClick={() => c.openDialog('versions')}>
      {t(m.versionHistory, { count: n })}
    </button>
  )
}

export function Footer() {
  const c = useComposer()
  const t = useT()
  if (c.isEmpty) {
    return (
      <footer className="cmp-foot">
        <span className="k">{t(m.footerEmpty)}</span>
        <span className="k" style={{ marginLeft: 'auto' }}>
          {t(m.sizeOf, { size: kb(0) })}
        </span>
      </footer>
    )
  }
  const code = c.view === 'mjml' || c.view === 'html'
  return (
    <footer className="cmp-foot">
      {code ? <span className="k">{c.view === 'mjml' ? t(m.footerMjml) : t(m.footerHtml)}</span> : <Breadcrumb />}
      <SizeSummary />
      {code ? null : <ConditionalFooter />}
      <ChecksBadge />
      <AccessibilityChip />
      {code ? null : <VersionLink />}
    </footer>
  )
}
