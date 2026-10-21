import type { Component } from 'grapesjs'
import { Fragment, useEffect, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { contentHtml, PALETTE, PALETTE_GROUPS, sectionHtml, type BlockContext, type PaletteGroup, type PaletteItem } from './blocks'
import { useBuilder } from './context'
import { byCid, isTop, kindOf, nodeOf } from './editor/nodes'
import { CONTENT_TARGETS, SECTION_TARGETS } from './editor/prepare'
import { BlockIcon, type BlockIconName } from './icons'
import { KIND_TAG, layerRows, type LayerRow } from './model'
import { m } from './messages'
import { kindName } from './wording'

type BlocksAPI = {
  startDrag(b: unknown, ev?: Event): void
  endDrag(cancel?: boolean): void
  get(id: string): unknown
  add(id: string, p: Record<string, unknown>): unknown
}

const GROUP_LABEL = { layout: m.grpLayout, content: m.grpContent, conversion: m.grpConversion, site: m.grpSite } as const

/** The tile's label: the block's name, "2 columns" and "3 columns" for the column blocks. */
function tileName(t: ReturnType<typeof useT>, it: PaletteItem): string {
  if (it.id === 'columns2') return t(m.kindColumns2)
  if (it.id === 'columns3') return t(m.kindColumns3)
  return kindName(t, it.kind)
}

/** Registers the palette with GrapesJS, for dragging tiles onto the canvas. */
function useDragBlocks(ctx: BlockContext) {
  const b = useBuilder()
  const ctxRef = useRef(ctx)
  useLayoutEffect(() => {
    ctxRef.current = ctx
  })
  useEffect(() => {
    const pe = b.pe
    if (!pe) return
    const api = pe.editor.Blocks as unknown as BlocksAPI
    for (const it of PALETTE) {
      if (api.get(it.id)) continue
      const tag = KIND_TAG[it.kind]
      api.add(it.id, {
        label: it.id,
        category: it.group,
        select: false,
        activate: false,
        content: () => pe.prepare(it.level === 'content' ? contentHtml(it.id, ctxRef.current) : sectionHtml(it.id, ctxRef.current)),
        // What the drop is checked with while dragging: a section goes between blocks, content also into columns.
        dragDef:
          it.level === 'section'
            ? { type: 'pb-section', tagName: 'section', draggable: SECTION_TARGETS }
            : tag
              ? { type: 'pb-live', tagName: tag }
              : { tagName: it.id === 'image' ? 'img' : 'div', draggable: CONTENT_TARGETS, droppable: false },
      })
    }
  }, [b.pe])
}

function Tile({ it, on }: { it: PaletteItem; on: boolean }) {
  const t = useT()
  const b = useBuilder()
  const disabled = !b.pe || !b.canEdit
  const name = tileName(t, it)
  const live = it.group === 'conversion' || it.group === 'site'
  return (
    <button
      type="button"
      className={cn('blk', live && 'rc', on && 'on')}
      draggable={!disabled}
      aria-disabled={disabled || undefined}
      aria-label={t(m.addBlock, { name })}
      title={t(m.tileHint)}
      onDragStart={(e) => {
        if (disabled || !b.pe) return
        e.dataTransfer.effectAllowed = 'copy'
        e.dataTransfer.setData('text/plain', name)
        const api = b.pe.editor.Blocks as unknown as BlocksAPI
        api.startDrag(api.get(it.id), e.nativeEvent)
      }}
      onDragEnd={() => b.pe && (b.pe.editor.Blocks as unknown as BlocksAPI).endDrag(false)}
      onClick={() => {
        if (!disabled) b.insertBlock(it.id)
      }}
    >
      <BlockIcon name={it.icon} />
      {name}
    </button>
  )
}

function Palette() {
  const t = useT()
  const b = useBuilder()
  const [q, setQ] = useState('')
  useDragBlocks(b.blockContext)
  const selKind = b.selected ? kindOf(b.selected) : null
  const needle = q.trim().toLowerCase()
  const match = (it: PaletteItem) => !needle || tileName(t, it).toLowerCase().includes(needle) || it.keywords.includes(needle)
  const groups = PALETTE_GROUPS.map((g: PaletteGroup) => ({ g, items: PALETTE.filter((it) => it.group === g && match(it)) })).filter((x) => x.items.length)
  return (
    <>
      <div className="pb-ltop">
        <span className="pb-ltitle">{t(m.blocksTitle)}</span>
        <input aria-label={t(m.searchBlocks)} className="in w-full" placeholder={t(m.searchBlocks)} value={q} onChange={(e) => setQ(e.target.value)} type="search" />
      </div>
      {groups.length ? (
        groups.map(({ g, items }) => (
          <Fragment key={g}>
            <div className="grp">{t(GROUP_LABEL[g])}</div>
            <div className="blk-grid">
              {items.map((it) => (
                <Tile key={it.id} it={it} on={Boolean(selKind) && selKind === it.kind && !(it.id === 'columns3' && selKind === 'columns')} />
              ))}
            </div>
          </Fragment>
        ))
      ) : (
        <p className="k px-3 py-2">{t(m.noBlocksMatch, { q })}</p>
      )}
    </>
  )
}

function rowIcon(r: LayerRow): BlockIconName {
  return r.kind === 'columns' && r.tag?.kind === 'columns' && r.tag.n >= 3 ? 'columns3' : r.kind
}

/** The layers: the site's header, each block and its named parts, the site's footer. */
function Layers() {
  const t = useT()
  const b = useBuilder()
  const wrapper = b.editor?.getWrapper()
  const rows = useMemo(
    () => (wrapper ? layerRows(wrapper.components().filter((c: Component) => isTop(c) && String(c.get('type')) !== 'textnode').map(nodeOf)) : []),
    // The rows follow the canvas: rev bumps on every change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [wrapper, b.rev],
  )
  const keys = useMemo(() => new Set(rows.map((r) => r.key)), [rows])
  let on = ''
  for (let cur: Component | undefined = b.selected ?? undefined; cur; cur = cur.parent()) {
    if (keys.has(cur.cid)) {
      on = cur.cid
      break
    }
  }
  const listRef = useRef<HTMLDivElement>(null)
  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp' && e.key !== 'Home' && e.key !== 'End') return
    const items = Array.from(listRef.current?.querySelectorAll<HTMLElement>('[role="treeitem"]') ?? [])
    const i = items.indexOf(document.activeElement as HTMLElement)
    const next = e.key === 'Home' ? 0 : e.key === 'End' ? items.length - 1 : Math.max(0, Math.min(items.length - 1, i + (e.key === 'ArrowDown' ? 1 : -1)))
    items[next]?.focus()
    e.preventDefault()
  }
  const name = (r: LayerRow) => {
    const base = r.name || kindName(t, r.kind)
    if (r.kind === 'image' && r.sub) return r.alt ? t(m.imageAlt, { name: base }) : t(m.imageNoAlt, { name: base })
    return base
  }
  const tag = (r: LayerRow) => (r.tag?.kind === 'columns' ? t(m.tagColumns, { count: r.tag.n }) : r.tag?.kind === 'count' ? String(r.tag.n) : '')
  // One tab stop: the selected row, or the first.
  const focusKey = on || rows[0]?.key || ''
  const theme = (label: string) => (
    <div className="pb-lyr th" role="treeitem" aria-level={1} aria-selected={false} aria-disabled="true" tabIndex={-1} title={t(m.themeRowHint)}>
      <BlockIcon name="theme" size={14} color="#5B6472" />
      <span className="n">{label}</span>
      <span className="k">{t(m.themeTag)}</span>
    </div>
  )
  return (
    <>
      <div className="pb-lsec">
        <h2>{t(m.layersTitle)}</h2>
        <span className="k">{t(m.layersCount, { count: rows.length + 2 })}</span>
      </div>
      <div className="pb-lyrs" role="tree" aria-label={t(m.layersTitle)} ref={listRef} tabIndex={-1} onKeyDown={onKey}>
        {theme(t(m.siteHeader))}
        {rows.map((r) => {
          const selected = r.key === on
          const pick = () => {
            const c = byCid(wrapper, r.key)
            if (c) b.select(c, { scroll: true })
          }
          return (
            <div
              key={r.key}
              className={cn('pb-lyr', r.sub && 'sub', selected && 'on')}
              role="treeitem"
              aria-level={r.sub ? 2 : 1}
              aria-selected={selected}
              aria-current={selected || undefined}
              tabIndex={r.key === focusKey ? 0 : -1}
              onClick={pick}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  pick()
                }
              }}
            >
              <BlockIcon name={rowIcon(r)} size={14} color={selected ? '#1D4ED8' : '#5B6472'} />
              <span className="n">{name(r)}</span>
              {tag(r) ? <span className="k">{tag(r)}</span> : null}
            </div>
          )
        })}
        {theme(t(m.siteFooter))}
      </div>
    </>
  )
}

/** The left panel (PageBuilder): blocks grouped as Layout, Content, Conversion and From your site, then the layers. */
export function LeftPanel() {
  const t = useT()
  return (
    <aside className="pb-left" aria-label={t(m.blocksPanel)}>
      <div className="pb-sticky">
        <Palette />
        <Layers />
      </div>
    </aside>
  )
}
