import type { Component } from 'grapesjs'
import { Fragment, useEffect, useMemo, useRef, useState, type KeyboardEvent, type ReactNode, type RefObject } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Dialog } from '@/components/ui/Dialog'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useComposer } from './context'
import { blockDefs, type BlockDef } from './editor/blocks'
import { focusCanvas } from './editor/keys'
import { blockAdded, blockContent, componentLabel, mjChildren, sectionOf, tagOf } from './editor/labels'
import { BlockIcon, ToolIcon } from './icons'
import { Thumb } from './Thumb'
import { isDigestTemplate } from '@/features/digests/composer/DigestMode'
import { LayerMark, LayersNote } from '@/features/conditional/Canvas'
import { EarnBlocksHint } from '@/features/earn/composer/EarnBlocks'
import { AddToAgencyLibrary } from '@/features/agency/ComposerLibrary'
import { PollBlockHint } from '@/features/polls/Block'
import { GrowBlocksNote, OpenGrowLink } from '@/features/grow/composer/Panel'
import { mb as growMessages } from '@/features/grow/composer/messages'
import { T, t as tt, useT } from '@/i18n'
import { useStoreBlockDefs } from '@/features/commerce/composer'
import { mc } from './canvas.messages'
import { m } from './messages'

type BlocksAPI = { startDrag(b: unknown, ev?: Event): void; endDrag(cancel?: boolean): void; get(id: string): unknown; add(id: string, p: Record<string, unknown>): unknown }

function blocksApi(editor: NonNullable<ReturnType<typeof useComposer>['editor']>): BlocksAPI {
  return editor.Blocks as unknown as BlocksAPI
}

/** Inserts a block next to the selection (keyboard and click alternative to dragging). */
export function insertBlock(c: ReturnType<typeof useComposer>, def: Pick<BlockDef, 'kind' | 'content' | 'label'>) {
  const ed = c.editor
  if (!ed) return
  const body = ed.getWrapper()?.findType('mj-body')[0]
  if (!body) return
  const sel = c.selected
  if (def.kind === 'section') {
    const sec = sectionOf(sel)
    const at = sec ? body.components().indexOf(sec) + 1 : body.components().length
    blockAdded(ed, body.append(blockContent(def.content), { at }) as Component | Component[])
    return
  }
  let col: Component | undefined = sel ?? undefined
  while (col && tagOf(col) !== 'mj-column' && tagOf(col) !== 'mj-hero') col = col.parent()
  if (!col) {
    const sec = sectionOf(sel)
    col = sec?.findType('mj-column')[0]
  }
  if (!col) {
    toast({ kind: 'info', title: tt(mc.selectColumn), detail: tt(mc.selectColumnDetail, { block: def.label }) })
    return
  }
  const at = sel && sel.parent() === col ? col.components().indexOf(sel) + 1 : col.components().length
  blockAdded(ed, col.append(blockContent(def.content), { at }) as Component | Component[])
}

function BlockTile({ def, flagged, onSaved }: { def: BlockDef | { id: string; label: string; icon: string; group: 'reputecast'; kind: 'section'; content: '' }; flagged?: boolean; onSaved?: () => void }) {
  const c = useComposer()
  const t = useT()
  const disabled = !c.editor || !c.canEdit
  const isSaved = def.id === 'rc-saved'
  return (
    <button
      type="button"
      className={`blk${def.group === 'reputecast' || def.group === 'grow' || def.group === 'earn' ? ' rc' : ''}${flagged ? ' flagged' : ''}`}
      draggable={!disabled && !isSaved}
      aria-disabled={disabled || undefined}
      onDragStart={(e) => {
        if (!c.editor || isSaved) return
        e.dataTransfer.effectAllowed = 'copy'
        e.dataTransfer.setData('text/plain', def.label)
        const api = blocksApi(c.editor)
        api.startDrag(api.get(def.id), e.nativeEvent)
      }}
      onDragEnd={() => c.editor && blocksApi(c.editor).endDrag(false)}
      onClick={() => {
        if (disabled) return
        if (isSaved) {
          onSaved?.()
          return
        }
        insertBlock(c, def as BlockDef)
      }}
      title={isSaved ? t(mc.showSaved) : def.kind === 'section' ? t(mc.dragSection) : t(mc.dragContent)}
    >
      <BlockIcon name={def.icon} />
      {def.label}
      {'badge' in def && def.badge ? <span className="blk-new">{def.badge}</span> : null}
    </button>
  )
}

function BlocksTab({ onShowSections, blocks, note, reputecastFirst }: { onShowSections: () => void; blocks?: BlockDef[]; note?: ReactNode; reputecastFirst?: boolean }) {
  const c = useComposer()
  const t = useT()
  const [q, setQ] = useState('')
  const sections = $api.useQuery('get', '/api/v1/workspaces/{workspace}/templates', { params: { path: { workspace: c.slug } } }, { staleTime: 60_000 })
  // The Product grid joins the ReputeCast blocks once a store is connected (X20).
  const storeDefs = useStoreBlockDefs(!blocks && c.kind !== 'transactional')
  const defs = useMemo(() => [...(blocks ?? blockDefs(c.context.postalAddress)), ...storeDefs], [blocks, c.context.postalAddress, storeDefs])
  useEffect(() => {
    if (!c.editor) return
    const api = blocksApi(c.editor)
    for (const d of defs) {
      // A drop selects in blockAdded (after the marker is back), not here.
      if (!api.get(d.id)) api.add(d.id, { label: d.label, content: d.content, category: d.group, select: false, activate: false })
    }
  }, [c.editor, defs])
  useEffect(() => {
    const ed = c.editor
    if (!ed) return
    // A dropped block's <!-- rc:… --> marker, which the parser drops, goes back in front of it.
    const dropped = (added: Component | Component[] | undefined, block?: { get(key: string): unknown }) => {
      blockAdded(ed, added, block?.get('content'))
    }
    ed.on('block:drag:stop', dropped)
    return () => {
      ed.off('block:drag:stop', dropped)
    }
  }, [c.editor])
  const needle = q.trim().toLowerCase()
  const match = (d: BlockDef) => !needle || d.label.toLowerCase().includes(needle) || (d.keywords ?? '').includes(needle)
  const footerMissing = c.checks?.issues.some((i) => i.id === 'compliance.footer')
  const savedCount = sections.data?.sections.length ?? 0
  const groups: { key: BlockDef['group']; label: string }[] = footerMissing
    ? [
        { key: 'reputecast', label: t(mc.groupReputecast) },
        { key: 'content', label: t(mc.groupContent) },
        { key: 'layout', label: t(mc.groupLayout) },
        { key: 'grow', label: tt(growMessages.group) },
        { key: 'earn', label: t(mc.groupEarn) },
      ]
    : reputecastFirst
      ? [
          { key: 'reputecast', label: t(mc.groupReputecast) },
          { key: 'layout', label: t(mc.groupLayout) },
          { key: 'content', label: t(mc.groupContent) },
          { key: 'grow', label: tt(growMessages.group) },
          { key: 'earn', label: t(mc.groupEarn) },
        ]
      : [
          { key: 'layout', label: t(mc.groupLayout) },
          { key: 'content', label: t(mc.groupContent) },
          { key: 'reputecast', label: t(mc.groupReputecast) },
          { key: 'grow', label: tt(growMessages.group) },
          { key: 'earn', label: t(mc.groupEarn) },
        ]
  return (
    <>
      <div style={{ padding: '8px 12px' }}>
        <input aria-label={t(mc.searchBlocks)} className="in w-full" placeholder={t(mc.searchBlocks)} value={q} onChange={(e) => setQ(e.target.value)} type="search" />
      </div>
      {groups.map((g) => {
        const items = defs.filter((d) => d.group === g.key && match(d))
        const pollHint = g.key === 'reputecast' && items.some((d) => d.id === 'rc-poll')
        const showSaved = g.key === 'reputecast' && (!needle || t(mc.savedSections).toLowerCase().includes(needle))
        if (!items.length && !showSaved) return null
        return (
          <Fragment key={g.key}>
            <div>
              <div className="grp">
                {g.label}
                {g.key === 'grow' ? <OpenGrowLink /> : null}
              </div>
              <div className={g.key === 'grow' ? 'blk-grid gw-cmp-grid' : 'blk-grid'}>
                {items.map((d) => (
                  <BlockTile key={d.id} def={d} flagged={footerMissing && d.id === 'rc-footer'} />
                ))}
                {showSaved ? (
                  <BlockTile def={{ id: 'rc-saved', label: t(mc.savedSectionsCount, { count: savedCount }), icon: 'saved', group: 'reputecast', kind: 'section', content: '' }} onSaved={onShowSections} />
                ) : null}
              </div>
              {pollHint ? <PollBlockHint /> : null}
              {g.key === 'grow' ? <GrowBlocksNote milestone={items.some((d) => d.id === 'rc-reward')} /> : null}
              {g.key === 'earn' && items.length ? <EarnBlocksHint /> : null}
            </div>
            {reputecastFirst && g.key === 'reputecast' ? note : null}
          </Fragment>
        )
      })}
      {reputecastFirst ? null : note}
    </>
  )
}

function layerIcon(tag: string, open: boolean, hasKids: boolean): string {
  if (hasKids) return open ? 'chevronDown' : 'chevronRight'
  switch (tag) {
    case 'mj-image':
      return 'layImage'
    case 'mj-text':
      return 'layText'
    case 'mj-button':
      return 'layButton'
    case 'mj-divider':
    case 'mj-spacer':
      return 'layDivider'
  }
  return 'layDot'
}

function hiddenTag(c: Component): string {
  const cls = String((c.getAttributes() as Record<string, unknown>)['css-class'] ?? '')
  if (cls.includes('rc-hide-mobile')) return tt(mc.hiddenMobile)
  if (cls.includes('rc-hide-desktop')) return tt(mc.hiddenDesktop)
  return ''
}

/** A row of the layer tree: a component, or the head (edited in the code view). */
interface LayerRow {
  key: string
  comp?: Component
  depth: number
  /** How many rows it holds (the root also holds the head). */
  kids: number
  open: boolean
  /** Key of the row it sits under. */
  parent?: string
  /** Position among its siblings, and their number (aria-posinset, aria-setsize). */
  pos: number
  size: number
}

/**
 * The layer tree (CO-8), a tree in the ARIA sense: one row takes the Tab
 * key, ↑ ↓ move between rows, → opens a row (or goes into an open one),
 * ← closes it (or goes up to its parent), Home and End go to the first and
 * last row, and Enter selects the row's element on the email and moves the
 * keyboard there (Enter then edits a text, Esc selects the parent). Space
 * selects without leaving the tree.
 */
export function LayersTab() {
  const c = useComposer()
  const t = useT()
  const [open, setOpen] = useState<Set<string>>(() => new Set())
  const [active, setActive] = useState<string | null>(null)
  const rowEls = useRef(new Map<string, HTMLDivElement>())
  const root = c.editor?.getWrapper()?.components().filter((x: Component) => tagOf(x) === 'mjml')[0]
  // The selection and its ancestors are open.
  const auto = new Set<string>()
  let cur = c.selected ?? undefined
  while (cur) {
    auto.add(cur.cid)
    cur = cur.parent()
  }
  if (root) {
    auto.add(root.cid)
    const body = mjChildren(root).find((x) => tagOf(x) === 'mj-body')
    if (body) auto.add(body.cid)
  }
  const isOpen = (x: Component) => (open.has(x.cid) ? true : open.has(`-${x.cid}`) ? false : auto.has(x.cid))
  const setOpenRow = (x: Component, value: boolean) =>
    setOpen((s) => {
      const n = new Set(s)
      n.delete(x.cid)
      n.delete(`-${x.cid}`)
      n.add(value ? x.cid : `-${x.cid}`)
      return n
    })
  const toggle = (x: Component) => setOpenRow(x, !isOpen(x))
  if (!root) return <p className="k p-3">{t(mc.layersEmpty)}</p>

  const rows: LayerRow[] = []
  const walk = (x: Component, depth: number, parent: string | undefined, pos: number, size: number) => {
    const kids = mjChildren(x)
    // The root holds the head first (it is edited in the code view).
    const head = tagOf(x) === 'mjml' ? 1 : 0
    const o = kids.length + head > 0 && isOpen(x)
    rows.push({ key: x.cid, comp: x, depth, kids: kids.length + head, open: o, parent, pos, size })
    if (!o) return
    if (head) rows.push({ key: 'head', depth: depth + 1, kids: 0, open: false, parent: x.cid, pos: 1, size: kids.length + 1 })
    kids.forEach((k, i) => walk(k, depth + 1, x.cid, i + 1 + head, kids.length + head))
  }
  walk(root, 0, undefined, 1, 1)
  // The row that takes the Tab key: the one last focused, or the selection, or the first.
  const tabRow = rows.find((r) => r.key === active) ?? rows.find((r) => r.comp !== undefined && r.comp === c.selected) ?? rows[0]!

  const focusRow = (key: string | undefined) => {
    if (!key) return
    setActive(key)
    rowEls.current.get(key)?.focus()
  }
  const pick = (row: LayerRow, toCanvas: boolean) => {
    setActive(row.key)
    if (!row.comp) {
      c.setView('mjml')
      return
    }
    const x = row.comp
    if (tagOf(x) === 'mjml') {
      toggle(x)
      return
    }
    c.editor?.select(x)
    x.getEl()?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    if (toCanvas && c.editor) focusCanvas(c.editor)
  }
  const onKey = (e: KeyboardEvent<HTMLDivElement>, row: LayerRow) => {
    const i = rows.indexOf(row)
    switch (e.key) {
      case 'ArrowDown':
        focusRow(rows[i + 1]?.key)
        break
      case 'ArrowUp':
        focusRow(rows[i - 1]?.key)
        break
      case 'Home':
        focusRow(rows[0]?.key)
        break
      case 'End':
        focusRow(rows[rows.length - 1]?.key)
        break
      case 'ArrowRight':
        if (row.kids && row.comp) {
          if (!row.open) setOpenRow(row.comp, true)
          else focusRow(rows[i + 1]?.key)
        }
        break
      case 'ArrowLeft':
        if (row.kids && row.open && row.comp) setOpenRow(row.comp, false)
        else focusRow(row.parent)
        break
      case 'Enter':
        pick(row, true)
        break
      case ' ':
        pick(row, false)
        break
      default:
        return
    }
    e.preventDefault()
  }
  return (
    <>
      <p className="k lay-help">
        <T msg={mc.layersHelp} values={{ k: (chunks) => <kbd>{chunks}</kbd> }} />
      </p>
      <div className="lay-tree" role="tree" aria-label={t(mc.layerTree)}>
        {rows.map((row) => {
          const x = row.comp
          const on = x !== undefined && c.selected === x
          return (
            <div
              key={row.key}
              ref={(el) => {
                if (el) rowEls.current.set(row.key, el)
                else rowEls.current.delete(row.key)
              }}
              role="treeitem"
              tabIndex={row === tabRow ? 0 : -1}
              aria-level={row.depth + 1}
              aria-posinset={row.pos}
              aria-setsize={row.size}
              aria-expanded={row.kids ? row.open : undefined}
              aria-selected={on}
              className={`lay${on ? ' on' : ''}`}
              style={{ paddingLeft: 8 + row.depth * 16 }}
              title={x ? undefined : t(mc.headTitle)}
              onKeyDown={(e) => onKey(e, row)}
              onFocus={() => setActive(row.key)}
              onClick={(e) => {
                if (x && row.kids && (e.target as HTMLElement).closest('[data-twisty]')) {
                  setActive(row.key)
                  toggle(x)
                  return
                }
                pick(row, false)
              }}
              onDoubleClick={() => x && row.kids && toggle(x)}
              onMouseEnter={() => x && (c.editor as unknown as { setHovered?: (c: Component) => void })?.setHovered?.(x)}
            >
              {x ? (
                <>
                  <span data-twisty className="inline-flex">
                    <ToolIcon name={layerIcon(tagOf(x), row.open, row.kids > 0)} size={14} strokeWidth={1.6} />
                  </span>
                  <span className="lay-name">{componentLabel(x)}</span>
                  <LayerMark c={x} />
                  {hiddenTag(x) ? <span className="lay-tag">{hiddenTag(x)}</span> : null}
                </>
              ) : (
                <>
                  <ToolIcon name="chevronRight" size={14} strokeWidth={1.6} />
                  {'mj-head'}
                </>
              )}
            </div>
          )
        })}
      </div>
      <LayersNote />
    </>
  )
}

function TemplatesTab({ sectionsRef }: { sectionsRef: RefObject<HTMLDivElement | null> }) {
  const c = useComposer()
  const tr = useT()
  const g = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/templates',
    // A recurring digest's template also gets the digest starters (Blog digest).
    { params: { path: { workspace: c.slug }, query: isDigestTemplate(c.campaign) ? { mode: 'digest' } : {} } },
    { staleTime: 60_000 },
  )
  const [confirm, setConfirm] = useState<{ id: string; name: string } | null>(null)
  if (g.isPending) return <p className="k p-3">{tr(mc.loadingTemplates)}</p>
  if (g.isError) return <p className="k p-3">{tr(m.templatesFailed, { error: errorMessage(g.error) })}</p>
  const use = async (id: string, name: string) => {
    const { data } = await http.GET('/api/v1/workspaces/{workspace}/templates/{templateId}', { params: { path: { workspace: c.slug, templateId: id } } })
    if (data) c.loadContent(data.mjml, { source: 'template', label: tr(mc.noteApplied, { name }) })
    setConfirm(null)
  }
  const insertSection = async (id: string, name: string) => {
    const { data } = await http.GET('/api/v1/workspaces/{workspace}/templates/{templateId}', { params: { path: { workspace: c.slug, templateId: id } } })
    if (data) insertBlock(c, { kind: 'section', content: data.mjml, label: name })
  }
  const all = [...g.data.starters, ...g.data.templates]
  return (
    <div className="flex flex-col gap-2 p-3">
      <span className="k">{tr(mc.templatesLead)}</span>
      <div className="grid grid-cols-2 gap-2">
        {all.map((t) => (
          <button key={t.id} type="button" className="tpl" style={{ padding: 8, gap: 6 }} disabled={!c.canEdit} onClick={() => setConfirm({ id: t.id, name: t.name })}>
            <Thumb items={t.thumb} blank={t.id === 'starter:blank'} small />
            <span className="text-[12px] font-semibold">{t.name}</span>
          </button>
        ))}
      </div>
      <div ref={sectionsRef} className="mt-2 flex flex-col gap-1.5">
        <span className="font-semibold">{tr(mc.savedSectionsCount, { count: g.data.sections.length })}</span>
        {g.data.sections.map((s) => (
          <button key={s.id} type="button" className="blk rc" style={{ flexDirection: 'row', justifyContent: 'flex-start', padding: '8px 10px' }} disabled={!c.canEdit || !c.editor} onClick={() => void insertSection(s.id, s.name)}>
            <BlockIcon name="saved" />
            <span className="flex flex-col items-start text-left">
              <span className="font-semibold">{s.name}</span>
              <span className="k">{s.description || tr(mc.savedSection)}</span>
            </span>
          </button>
        ))}
        {c.canEdit && c.campaign.access.canEditTemplates ? (
          <button type="button" className="btn sm self-start" onClick={() => c.openDialog('saveTemplate')} disabled={!c.editor}>
            {tr(mc.saveAs)}
          </button>
        ) : null}
        {/* The agency's staff share a saved section or template with every client (agency accounts). */}
        <AddToAgencyLibrary sections={g.data.sections} templates={g.data.templates} />
      </div>
      <Dialog
        open={confirm !== null}
        onOpenChange={(o) => !o && setConfirm(null)}
        title={tr(mc.useTitle, { name: confirm?.name ?? '' })}
        size="sm"
        footer={
          <>
            <button type="button" className="btn" onClick={() => setConfirm(null)}>
              {tr(mc.keepEditing)}
            </button>
            <button type="button" className="btn pri" onClick={() => confirm && void use(confirm.id, confirm.name)}>
              {tr(mc.replaceContent)}
            </button>
          </>
        }
      >
        <p>{tr(mc.replaceBody, { version: c.status.kind === 'new' ? 1 : c.status.version })}</p>
      </Dialog>
    </div>
  )
}

/**
 * Left panel: Blocks · Layers · Templates. The transactional mode passes its
 * own blocks (the transactional footer) and a note under them; digest mode
 * puts ReputeCast's blocks (Feed items first) on top, with its note under them.
 */
export function LeftPanel({ blocks, note, reputecastFirst }: { blocks?: BlockDef[]; note?: ReactNode; reputecastFirst?: boolean } = {}) {
  const c = useComposer()
  const t = useT()
  const sectionsRef = useRef<HTMLDivElement | null>(null)
  if (c.isEmpty) {
    const defs = blocks ?? blockDefs(c.context.postalAddress)
    return (
      // A preview of what comes next, inactive until a start is picked.
      <aside className="cpanel left dim" aria-hidden="true" inert>
        <div className="ptab">
          <button type="button" className="on" disabled>
            {t(mc.tabBlocks)}
          </button>
          <button type="button" disabled>
            {t(mc.tabLayers)}
          </button>
          <button type="button" disabled>
            {t(mc.tabTemplates)}
          </button>
        </div>
        <div className="grp">{t(mc.groupLayout)}</div>
        <div className="blk-grid">
          {defs.slice(0, 3).map((d) => (
            <div key={d.id} className="blk" style={{ cursor: 'default' }}>
              <BlockIcon name={d.icon} />
              {d.label}
            </div>
          ))}
        </div>
        <div className="grp">{t(mc.groupContent)}</div>
        <div className="blk-grid">
          {defs
            .filter((d) => ['mj-text', 'mj-image', 'mj-button'].includes(d.id))
            .map((d) => (
              <div key={d.id} className="blk" style={{ cursor: 'default' }}>
                <BlockIcon name={d.icon} />
                {d.label}
              </div>
            ))}
        </div>
      </aside>
    )
  }
  return (
    <aside className="cpanel left" aria-label={t(mc.leftPanelLabel)}>
      <PanelTabs
        value={c.leftTab}
        onChange={c.setLeftTab}
        label={t(mc.panels)}
        items={[
          { value: 'blocks', label: t(mc.tabBlocks) },
          { value: 'layers', label: t(mc.tabLayers) },
          { value: 'templates', label: t(mc.tabTemplates) },
        ]}
      >
        <TabPanel value="blocks">
          <BlocksTab
            blocks={blocks}
            note={note}
            reputecastFirst={reputecastFirst}
            onShowSections={() => {
              c.setLeftTab('templates')
              window.setTimeout(() => sectionsRef.current?.scrollIntoView({ block: 'start', behavior: 'smooth' }), 50)
            }}
          />
        </TabPanel>
        <TabPanel value="layers">
          <LayersTab />
        </TabPanel>
        <TabPanel value="templates">
          <TemplatesTab sectionsRef={sectionsRef} />
        </TabPanel>
      </PanelTabs>
    </aside>
  )
}
