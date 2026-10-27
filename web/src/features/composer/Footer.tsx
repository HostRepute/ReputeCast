import type { Component } from 'grapesjs'
import { DigestFooterSummary, useDigestMode } from '@/features/digests/composer/DigestMode'
import { Fragment } from 'react'
import { $api } from '@/api/client'
import { useVersionName } from '@/features/languages/composerParts'
import { useComposer } from './context'
import { ancestry, mjChildren, sectionName, tagOf } from './editor/labels'
import { kb } from './format'
import { t as tt, useT } from '@/i18n'
import { modKeyLabel } from '@/lib/shortcuts'
import { mh } from './chrome.messages'
import { m } from './messages'
import { contentIssues } from './RightPanel'
import { useAccessibility } from '@/features/accessibility/composer/state'
import { AccessibilityChip, AccessibilityClear } from '@/features/accessibility/composer/Status'
import { ConditionalFooter, useVersionSizes } from '@/features/conditional/Versions'

/** A crumb as the board names it: Email › Body › Feature highlights · 2 columns › Column 1 › Button. */
function crumbLabel(c: Component): string {
  const tag = tagOf(c)
  if (tag === 'mjml') return tt(mh.crumbEmail)
  if (tag === 'mj-body') return tt(mh.crumbBody)
  if (tagOf(c.parent()) === 'mj-body') {
    const cols = mjChildren(c).filter((x) => tagOf(x) === 'mj-column').length
    const name = sectionName(c)
    return cols > 1 ? tt(mh.crumbColumns, { name, count: cols }) : name
  }
  if (tag === 'mj-column') {
    const parent = c.parent()
    const n = parent ? mjChildren(parent).filter((x) => tagOf(x) === 'mj-column').indexOf(c) + 1 : 1
    return tt(mh.crumbColumn, { n })
  }
  return tt(mh.crumbTag, { tag: tag.replace(/-/g, '_') })
}

export function Breadcrumb() {
  const c = useComposer()
  const t = useT()
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  if (!sel) {
    return (
      <nav className="bc" aria-label={t(mh.selectionPath)}>
        <span>{t(mh.crumbEmail)}</span>
        <span aria-hidden="true">›</span>
        <span>{t(mh.crumbBody)}</span>
        <span className="k">{t(m.selectOnCanvas)}</span>
      </nav>
    )
  }
  const chain = ancestry(sel)
  return (
    <nav className="bc" aria-label={t(mh.selectionPath)}>
      {chain.map((x, i) => {
        const current = x === sel
        return (
          <Fragment key={x.cid}>
            {i ? <span aria-hidden="true">›</span> : null}
            {current ? (
              <b aria-current="true">{crumbLabel(x)}</b>
            ) : tagOf(x) === 'mjml' ? (
              <span>{crumbLabel(x)}</span>
            ) : (
              <button type="button" onClick={() => c.editor?.select(x)}>
                {crumbLabel(x)}
              </button>
            )}
          </Fragment>
        )
      })}
    </nav>
  )
}

/** The canvas's keys (Composer board): Enter edits text, Esc selects the parent, undo, the shortcuts. */
function KeyHints() {
  const t = useT()
  const mod = modKeyLabel()
  return (
    <span className="hint" aria-label={t(mh.keysLabel)}>
      <span>
        <span className="kbd">{t(mh.keyEnter)}</span>
        {t(mh.keyEditText)}
      </span>
      <span>
        <span className="kbd">{t(mh.keyEsc)}</span>
        {t(mh.keyParent)}
      </span>
      <span>
        <span className="kbd">{mod}Z</span>
        {t(mh.keyUndo)}
      </span>
      <span>
        <span className="kbd">{mod}/</span>
        {t(mh.keyShortcuts)}
      </span>
    </span>
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
  if (c.view === 'write')
    // Write mode (ComposerWrite board): what it edits, and the history.
    return (
      <footer className="cmp-foot">
        <span className="wfootn">
          <b>{t(mh.writeFootStrong)}</b> <span className="k">{t(mh.writeFootNote)}</span>
        </span>
        <span style={{ marginLeft: 'auto' }}>
          <VersionLink />
        </span>
      </footer>
    )
  return (
    <footer className="cmp-foot">
      {code ? <span className="k">{c.view === 'mjml' ? t(m.footerMjml) : t(m.footerHtml)}</span> : <Breadcrumb />}
      {code ? null : <KeyHints />}
      <SizeSummary />
      {code ? null : <ConditionalFooter />}
      <AccessibilityChip />
      {code ? null : <VersionLink />}
    </footer>
  )
}
