import type { Component } from 'grapesjs'
import { Fragment, useEffect, useMemo, useRef, useState, type ReactNode, type RefObject } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Dialog } from '@/components/ui/Dialog'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useComposer } from './context'
import { blockDefs, type BlockDef } from './editor/blocks'
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
import { t as tt, useT } from '@/i18n'
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

function LayersTab() {
  const c = useComposer()
  const t = useT()
  const [open, setOpen] = useState<Set<string>>(() => new Set())
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
  const toggle = (x: Component) =>
    setOpen((s) => {
      const n = new Set(s)
      const was = isOpen(x)
      n.delete(x.cid)
      n.delete(`-${x.cid}`)
      n.add(was ? `-${x.cid}` : x.cid)
      return n
    })
  if (!root) return <p className="k p-3">{t(mc.layersEmpty)}</p>
  const rows: ReactNode[] = []
  const render = (x: Component, depth: number) => {
    const kids = mjChildren(x)
    const o = isOpen(x)
    const tag = tagOf(x)
    rows.push(
      <div key={x.cid} role="treeitem" aria-level={depth + 1} aria-expanded={kids.length ? o : undefined} aria-selected={c.selected === x}>
        <button
          type="button"
          className={`lay${c.selected === x ? ' on' : ''}`}
          style={{ paddingLeft: 8 + depth * 16 }}
          onClick={(e) => {
            if (kids.length && (e.target as HTMLElement).closest('[data-twisty]')) {
              toggle(x)
              return
            }
            if (tag === 'mjml') {
              toggle(x)
              return
            }
            c.editor?.select(x)
            x.getEl()?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
          }}
          onDoubleClick={() => kids.length && toggle(x)}
          onMouseEnter={() => (c.editor as unknown as { setHovered?: (c: Component) => void })?.setHovered?.(x)}
        >
          <span data-twisty className="inline-flex">
            <ToolIcon name={layerIcon(tag, o, kids.length > 0)} size={14} strokeWidth={1.6} />
          </span>
          <span className="lay-name">{componentLabel(x)}</span>
          <LayerMark c={x} />
          {hiddenTag(x) ? <span className="lay-tag">{hiddenTag(x)}</span> : null}
        </button>
      </div>,
    )
    if (tag === 'mjml' && o) {
      rows.push(
        <div key="head" role="treeitem" aria-level={depth + 2} aria-selected={false}>
          <button type="button" className="lay" style={{ paddingLeft: 8 + (depth + 1) * 16 }} onClick={() => c.setView('mjml')} title={t(mc.headTitle)}>
            <ToolIcon name="chevronRight" size={14} strokeWidth={1.6} />
            {'mj-head'}
          </button>
        </div>,
      )
    }
    if (o) kids.forEach((k) => render(k, depth + 1))
  }
  render(root, 0)
  return (
    <>
      <div style={{ padding: 8, display: 'flex', flexDirection: 'column', gap: 1 }} role="tree" aria-label={t(mc.layerTree)}>
        {rows}
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
