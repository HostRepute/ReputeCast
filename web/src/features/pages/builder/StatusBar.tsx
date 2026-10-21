import type { Component } from 'grapesjs'
import { Fragment, useMemo } from 'react'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { useBuilder } from './context'
import { isTop, nodeOf } from './editor/nodes'
import { crumbs } from './labels'
import { layerRows } from './model'
import { m } from './messages'

function Breadcrumb() {
  const t = useT()
  const b = useBuilder()
  const sel = b.selected && b.selected.get('type') !== 'wrapper' ? b.selected : null
  if (!sel) {
    return (
      <div className="bc">
        <span>{t(m.crumbPage)}</span>
        <span className="k">· {t(m.crumbNone)}</span>
      </div>
    )
  }
  const chain = crumbs(t, sel)
  return (
    <nav className="bc" aria-label={t(m.crumbLabel)}>
      <button type="button" onClick={() => b.select(null)}>
        {t(m.crumbPage)}
      </button>
      {chain.map((x) => (
        <Fragment key={x.c.cid}>
          <span aria-hidden="true">›</span>
          {x.c === sel ? (
            <b aria-current="true">{x.word}</b>
          ) : (
            <button type="button" onClick={() => b.select(x.c)}>
              {x.word}
            </button>
          )}
        </Fragment>
      ))}
    </nav>
  )
}

/**
 * The status bar (PageBuilder): the selection's breadcrumb, the page's
 * blocks, images and links, the checks, and the version history.
 */
export function StatusBar() {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const wrapper = b.editor?.getWrapper()
  const blocks = useMemo(
    () => (wrapper ? layerRows(wrapper.components().filter((c: Component) => isTop(c)).map(nodeOf)).length + 2 : 0),
    // The rows follow the canvas: rev bumps on every change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [wrapper, b.rev],
  )
  const c = p.counts
  const missing = c.images - c.imagesWithAlt
  const blocking = p.checks.filter((x) => x.status === 'block').length
  const warnings = p.checks.filter((x) => x.status === 'warn').length
  const passed = p.checks.filter((x) => x.status === 'pass').length
  return (
    <footer className="cmp-foot pb-foot">
      <Breadcrumb />
      <span className="k" style={{ marginLeft: 'auto' }}>
        {blocks ? t(m.footBlocks, { count: blocks }) : null}
        {blocks ? ' · ' : null}
        {missing > 0 ? t(m.footImagesMissing, { count: c.images, missing }) : t(m.footImages, { count: c.images })}
        {' · '}
        {t(m.footLinks, { count: c.links })}
      </span>
      {blocking ? (
        <Badge tone="bad" dot>
          {t(m.footBlocking, { count: blocking })}
        </Badge>
      ) : warnings ? (
        <Badge tone="warn" dot>
          {t(m.footWarnings, { count: warnings })}
        </Badge>
      ) : (
        <Badge tone="ok" dot>
          {t(m.footPassed, { count: passed })}
        </Badge>
      )}
      <button type="button" className="k pb-linkbtn" onClick={() => b.openDialog({ kind: 'versions' })}>
        {t(m.versionLink, { count: p.versions })}
      </button>
    </footer>
  )
}
