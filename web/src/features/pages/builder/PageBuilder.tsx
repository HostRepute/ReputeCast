import { useQueryClient } from '@tanstack/react-query'
import type { Block, Component } from 'grapesjs'
import { useCallback, useEffect, useLayoutEffect, useMemo, useReducer, useRef, useState } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useT } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'
import { useUnsavedWork } from '@/lib/sessionClock'
import { useShortcutHandler } from '@/lib/shortcuts'
import { useCan, useWorkspace } from '@/lib/session'
import { nameOf, type BlockContext, type PaletteId } from './blocks'
import { Canvas } from './Canvas'
import { BuilderCtx, type Builder, type BuilderDialog, type ConflictInfo, type PageCanvas, type PageDetail, type PublishWhen, type RightTab } from './context'
import { useForms, useProducts } from './data'
import { Dialogs } from './Dialogs'
import { duplicateBlock, insertBlock, moveBlock, settleDrop } from './editor/actions'
import { blockOf, byCid, changesBlocks, findById, nodeOf, topOf } from './editor/nodes'
import { createPageEditor, type FrameParts, type PageDevice, type PageEditor } from './editor/pageEditor'
import { Header, PermissionNote } from './Header'
import { LeftPanel } from './LeftPanel'
import { conflictBlocks, datePlus, soleLive } from './model'
import { m } from './messages'
import { RightPanel } from './RightPanel'
import { AUTOSAVE_MS, isOffline, isUnsaved, OFFLINE_RETRY_MS, PROJECT, saveReducer, STRUCTURAL_MS, theirsFrom, type SaveStatus } from './save'
import { StatusBar } from './StatusBar'
import { blockCopy, livePlaceholder, localTime } from './wording'
import '@/features/composer/composer.css'
import './builder.css'

export interface BuilderSearch {
  view?: 'preview'
  tab?: RightTab
  device?: PageDevice
  /** Opens the publish dialog (an approval notification links here). */
  publish?: boolean
  /** A block to select once the canvas is ready, by its id. */
  select?: string
}

function frameOf(canvas: PageCanvas, hideHeader: boolean, path: string): FrameParts {
  return { header: hideHeader ? canvas.focusHeader : canvas.header, footer: canvas.footer, lang: canvas.lang, dir: canvas.dir, path }
}

function initialStatus(p: PageDetail): SaveStatus {
  return { kind: 'saved', version: p.draft.version, at: p.draft.savedAt, by: p.draft.savedBy?.name ?? '', auto: p.draft.autosaved }
}

function initials(name: string): string {
  const out = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]!.toUpperCase())
    .join('')
  return out || 'A'
}

/**
 * The page builder (5.10): the composer's interface in page mode. Blocks
 * on the publisher's theme in the canvas, the palette and the layers on
 * the left, the Block, Page and Checks tabs on the right; autosave with
 * conflict detection, versions, and the publish dialog with its diff.
 */
export function PageBuilder({ initial, canvas, search, onSearch }: { initial: PageDetail; canvas: PageCanvas; search: BuilderSearch; onSearch: (s: BuilderSearch) => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const canEditPerm = useCan('website:edit')
  const canPublishPerm = useCan('website:publish')
  const [page, setPageState] = useState(initial)
  const canEdit = canEditPerm && page.publish.canEdit
  const device: PageDevice = search.device ?? 'desktop'
  const tz = me.timezone || 'UTC'

  const [pe, setPe] = useState<PageEditor | null>(null)
  const [selected, setSelected] = useState<Component | null>(null)
  const [rev, setRev] = useState(0)
  const [status, dispatch] = useReducer(saveReducer, initial, initialStatus)
  const statusRef = useRef(status)
  useLayoutEffect(() => {
    statusRef.current = status
  })
  const [savedCard, setSavedCard] = useState<{ version: number; at: string } | null>(null)
  const [conflict, setConflict] = useState<ConflictInfo | null>(null)
  const [dialog, setDialog] = useState<BuilderDialog>(() => (search.view === 'preview' ? { kind: 'preview' } : null))
  const [rightTab, setRightTabState] = useState<RightTab>(search.tab ?? 'block')
  const [focusField, setFocusField] = useState<string | null>(null)
  const [publishWhen, setPublishWhen] = useState<PublishWhen>(() => ({ when: 'now', date: datePlus(new Date(), 1, tz), time: '09:00', tz }))

  const peRef = useRef<PageEditor | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const version = useRef(initial.draft.version)
  const revision = useRef(0)
  const savedRevision = useRef(0)
  /** The draft the editor started from (or last saved): what a conflict is compared with. */
  const baseHtml = useRef(initial.draft.html)
  const saving = useRef<Promise<boolean> | null>(null)
  /** A version is being loaded into the canvas: its changes aren't edits. */
  const loading = useRef(false)
  const structuralTimer = useRef<number | undefined>(undefined)
  const deepLink = useRef(search.select)
  const openPublishOnLoad = useRef(Boolean(search.publish))

  useDocumentTitle(t(m.documentTitle, { name: page.name }))

  const setPage = useCallback((fn: (p: PageDetail) => PageDetail) => setPageState(fn), [])
  const bump = useCallback(() => setRev((r) => r + 1), [])

  // What new blocks start with: the first live form and published product, the workspace's time zone.
  const forms = useForms()
  const products = useProducts()
  const ctxRef = useRef<BlockContext | null>(null)
  const [today] = useState(() => new Date())
  const blockCtx = useMemo<BlockContext>(
    () => ({
      copy: blockCopy(t),
      tz,
      date: datePlus(today, 14, tz),
      formId: forms.items.find((f) => f.status === 'live')?.id ?? '',
      productId: products.items.find((p) => p.status === 'published')?.id ?? '',
      pageName: page.name,
      initials: initials(me.workspace.name),
    }),
    [t, tz, today, forms.items, products.items, page.name, me.workspace.name],
  )
  useLayoutEffect(() => {
    ctxRef.current = blockCtx
  })

  const canEditRef = useRef(canEdit)
  useLayoutEffect(() => {
    canEditRef.current = canEdit
  })
  const markEdit = useCallback(() => {
    // Read-only canvases change too (their blocks are locked as they load): nothing to save there.
    if (loading.current || !canEditRef.current) return
    revision.current++
    dispatch({ type: 'edit' })
    setRev((r) => r + 1)
  }, [])

  const select = useCallback((c: Component | null, opts?: { scroll?: boolean }) => {
    const ed = peRef.current?.editor
    if (!ed) return
    if (!c) {
      ed.select(undefined)
      return
    }
    ed.select(c)
    if (opts?.scroll) {
      // The canvas grows with the page: the window scrolls, not the frame,
      // and only when the block isn't in view already.
      const el = c.getEl()
      const frame = ed.Canvas.getFrameEl()
      if (el && frame) {
        const zoom = ed.Canvas.getZoomDecimal()
        const r = el.getBoundingClientRect()
        const fr = frame.getBoundingClientRect()
        const top = fr.top + r.top * zoom
        const bottom = top + r.height * zoom
        if (top < 60 || bottom > window.innerHeight - 20) {
          const y = top + window.scrollY - window.innerHeight / 2 + (r.height * zoom) / 2
          window.scrollTo({ top: Math.max(0, y), behavior: 'smooth' })
        }
      }
    }
  }, [])

  // ---------------------------------------------------------------- saving

  const save = useCallback(
    async (opts: { manual?: boolean; force?: boolean; quiet?: boolean; background?: boolean } = {}): Promise<boolean> => {
      if (!canEdit) return false
      // A conflict waits for the person: Keep mine or Load theirs.
      if (opts.background && statusRef.current.kind === 'conflict') return false
      const inst = peRef.current
      // Text being typed reaches the model when its editing ends: timed
      // autosaves wait for that; anything the person asked for ends it first.
      if (inst?.isEditingText()) {
        if (opts.background) return false
        await inst.flushEditing()
      }
      if (saving.current) {
        await saving.current
        if (revision.current === savedRevision.current && !opts.force) return true
      }
      const run = async (): Promise<boolean> => {
        const rev0 = revision.current
        const html = inst ? inst.getHtml() : baseHtml.current
        dispatch({ type: 'start' })
        try {
          const { data } = await http.PUT('/api/v1/workspaces/{workspace}/pages/{page}/content', {
            params: { path: { workspace: slug, page: page.id } },
            body: { html, project: PROJECT, baseVersion: version.current, manual: Boolean(opts.manual), force: opts.force || undefined },
          })
          const saved = data!
          version.current = Math.max(version.current, saved.version)
          savedRevision.current = rev0
          baseHtml.current = html
          setConflict(null)
          setPageState((p) => ({
            ...p,
            checks: saved.checks,
            counts: saved.counts,
            changes: saved.changes,
            versions: Math.max(p.versions, saved.version),
            draft: { ...p.draft, html, version: saved.version, savedAt: saved.savedAt, savedBy: saved.savedBy, autosaved: !opts.manual },
          }))
          dispatch({ type: 'saved', version: saved.version, at: saved.savedAt, by: saved.savedBy?.name ?? '', manual: Boolean(opts.manual), dirtyAgain: revision.current !== rev0 })
          if (opts.manual && !opts.quiet) setSavedCard({ version: saved.version, at: saved.savedAt })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/pages'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/pages/{page}/versions'] })
          return true
        } catch (e) {
          if (e instanceof ApiError && e.status === 409 && e.code === 'version_conflict') {
            const theirs = theirsFrom(e.problem.meta)
            dispatch({ type: 'conflict', theirs })
            // What each of you changed since the version this canvas started from.
            void http
              .GET('/api/v1/workspaces/{workspace}/pages/{page}', { params: { path: { workspace: slug, page: page.id } } })
              .then(({ data: d }) => {
                if (!d) return
                const c = conflictBlocks(baseHtml.current, html, d.draft.html)
                setConflict({ html: d.draft.html, both: c.both, theirs: c.theirs })
              })
              .catch(() => undefined)
            return false
          }
          if (isOffline(e)) {
            dispatch({ type: 'offline', at: new Date().toISOString() })
            return false
          }
          dispatch({ type: 'failed', message: errorMessage(e) })
          return false
        }
      }
      const p = run()
      saving.current = p
      try {
        return await p
      } finally {
        saving.current = null
      }
    },
    [canEdit, slug, page.id, qc],
  )

  const saveRef = useRef(save)
  useLayoutEffect(() => {
    saveRef.current = save
  })

  // Autosave: every 30 seconds while there are edits, soon after a block was added, moved or deleted.
  useEffect(() => {
    if (!canEdit) return
    const h = window.setInterval(() => {
      if (revision.current !== savedRevision.current) void saveRef.current({ background: true })
    }, AUTOSAVE_MS)
    return () => window.clearInterval(h)
  }, [canEdit])
  useEffect(() => {
    if (status.kind !== 'offline') return
    const h = window.setInterval(() => void saveRef.current({ background: true }), OFFLINE_RETRY_MS)
    return () => window.clearInterval(h)
  }, [status.kind])

  const reloadFromServer = useCallback(async () => {
    const { data } = await http.GET('/api/v1/workspaces/{workspace}/pages/{page}', { params: { path: { workspace: slug, page: page.id } } })
    if (!data) return
    loading.current = true
    try {
      peRef.current?.setHtml(data.draft.html)
      peRef.current?.setFrame(frameOf(canvas, data.settings.hideHeader, data.path))
    } finally {
      loading.current = false
    }
    version.current = data.draft.version
    baseHtml.current = data.draft.html
    savedRevision.current = revision.current
    setConflict(null)
    setPageState(data)
    setSelected(null)
    dispatch({ type: 'loaded', version: data.draft.version, at: data.draft.savedAt, by: data.draft.savedBy?.name ?? '', auto: data.draft.autosaved })
    setRev((r) => r + 1)
  }, [slug, page.id, canvas])

  const applyDetail = useCallback((d: PageDetail) => {
    version.current = Math.max(version.current, d.draft.version)
    dispatch({ type: 'version', version: d.draft.version })
    // The canvas keeps what's on it: the server's draft may be older than unsaved edits.
    setPageState((prev) => ({ ...d, draft: { ...d.draft, html: prev.draft.html } }))
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/pages'] })
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/pages/{page}/versions'] })
  }, [qc])

  const refreshChecks = useCallback(async () => {
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/pages/{page}', { params: { path: { workspace: slug, page: page.id } } })
      if (data) setPageState((p) => ({ ...p, checks: data.checks, counts: data.counts, publish: data.publish, changes: data.changes }))
    } catch {
      // The next save brings them anyway.
    }
  }, [slug, page.id])

  // ---------------------------------------------------------------- the editor

  const render = useCallback(
    async (blocks: string[]) => {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/render', { params: { path: { workspace: slug, page: page.id } }, body: { blocks } })
      return data?.blocks ?? []
    },
    [slug, page.id],
  )

  useEffect(() => {
    const host = containerRef.current
    if (!host) return
    let cancelled = false
    let created: PageEditor | null = null
    // Each editor gets its own mount: a cancelled one (React re-running the
    // effect) is destroyed later and must not take the next one with it.
    const mount = document.createElement('div')
    host.appendChild(mount)
    void createPageEditor(mount, {
      html: initial.draft.html,
      styles: canvas.styles,
      frame: frameOf(canvas, initial.settings.hideHeader, initial.path),
      device,
      readOnly: !canEdit,
      render,
      placeholder: (tag, attrs, state) => livePlaceholder(tt, tag, attrs, state),
      localTime,
      frameTitle: tt(m.frameTitle),
    })
      .then((inst) => {
        if (cancelled) {
          inst.destroy()
          mount.remove()
          return
        }
        created = inst
        peRef.current = inst
        const ed = inst.editor
        // Dev builds expose the live editor to the visual QA scripts.
        if (import.meta.env.DEV) (window as unknown as { __rcPageBuilder?: typeof ed }).__rcPageBuilder = ed
        ed.on('update', markEdit)
        ed.on('component:input', markEdit)
        const structural = (c?: Component) => {
          // A text edit swaps the text's own nodes when it ends: that waits for the timed autosave.
          if (loading.current || (c && !changesBlocks(c))) return
          window.clearTimeout(structuralTimer.current)
          structuralTimer.current = window.setTimeout(() => void saveRef.current({ background: true }), STRUCTURAL_MS)
        }
        ed.on('component:add', structural)
        ed.on('component:remove', structural)
        // A live block's settings and a block's style change what the checks see: they save soon too.
        ed.on('component:update:attributes component:update:classes', structural)
        ed.on('component:toggled', () => {
          const s = ed.getSelected() ?? null
          setSelected(s)
          if (s) setRightTabState('block')
        })
        ed.on('component:remove', () => setSelected(ed.getSelected() ?? null))
        ed.on('undo redo component:update component:update:attributes', () => setRev((r) => r + 1))
        ed.on('block:drag:stop', (c: Component | undefined, block: Block | undefined) => {
          if (!c) return
          const id = block?.getId() as PaletteId | undefined
          const settled = settleDrop(inst, c, id, { name: (pid) => nameOf(pid, ctxRef.current?.copy ?? blockCopy(tt)) })
          const top = topOf(settled)
          const live = top === settled ? soleLive(nodeOf(top)) : undefined
          ed.select((live && top ? byCid(top, live.key) : undefined) ?? settled)
        })
        setPe(inst)
        const link = deepLink.current
        if (link) {
          deepLink.current = undefined
          const target = findById(ed.getWrapper(), link)
          if (target) window.setTimeout(() => select(target, { scroll: true }), 300)
        }
        if (openPublishOnLoad.current) {
          openPublishOnLoad.current = false
          setDialog({ kind: 'publish' })
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) toast({ kind: 'error', title: tt(m.editorFailed), detail: e instanceof Error ? e.message : String(e) })
      })
    return () => {
      cancelled = true
      window.clearTimeout(structuralTimer.current)
      if (peRef.current === created) peRef.current = null
      created?.destroy()
      if (created) mount.remove()
      else mount.style.display = 'none'
      setPe(null)
      setSelected(null)
    }
    // The editor is created once for the page; the device and the frame are applied below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (pe && pe.editor.getDevice() !== device) pe.editor.setDevice(device)
  }, [pe, device])

  useEffect(() => {
    pe?.setFrame(frameOf(canvas, page.settings.hideHeader, page.path))
  }, [pe, canvas, page.settings.hideHeader, page.path])

  // ⌘S goes through the app's shortcut layer, ⌘/ lists the shortcuts. GrapesJS hands the canvas's
  // key presses on to the page (as events on its frame), so both work from the canvas too; there,
  // only the browser's own ⌘S (saving the frame as a file) is stopped.
  useEffect(() => {
    const onFrameKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === 's') e.preventDefault()
    }
    const onPageKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === '/') {
        e.preventDefault()
        setDialog({ kind: 'shortcuts' })
      }
    }
    window.addEventListener('keydown', onPageKey)
    const frameDoc = pe?.editor.Canvas.getDocument()
    frameDoc?.addEventListener('keydown', onFrameKey)
    return () => {
      window.removeEventListener('keydown', onPageKey)
      frameDoc?.removeEventListener('keydown', onFrameKey)
    }
  }, [pe])

  useShortcutHandler(
    'save',
    canEdit
      ? () => {
          void save({ manual: true })
        }
      : undefined,
  )

  // ---------------------------------------------------------------- actions

  const setDevice = useCallback((d: PageDevice) => onSearch({ ...search, device: d === 'desktop' ? undefined : d, view: undefined, publish: undefined, select: undefined }), [onSearch, search])
  const setRightTab = useCallback(
    (tab: RightTab) => {
      setRightTabState(tab)
      onSearch({ ...search, tab: tab === 'block' ? undefined : tab, view: undefined, publish: undefined, select: undefined })
    },
    [onSearch, search],
  )
  const openDialog = useCallback(
    (d: BuilderDialog) => {
      setDialog(d)
      if (!d && search.view) onSearch({ ...search, view: undefined, publish: undefined })
    },
    [onSearch, search],
  )

  const selectAdded = useCallback(
    (added: Component | undefined) => {
      if (!added) return
      const top = topOf(added)
      const sole = top === added ? soleLive(nodeOf(added)) : undefined
      const target = (sole ? byCid(added, sole.key) : undefined) ?? added
      select(target, { scroll: true })
      setRightTabState('block')
    },
    [select],
  )

  const insert = useCallback(
    (id: PaletteId) => {
      const inst = peRef.current
      if (!inst || !canEdit) return
      selectAdded(insertBlock(inst, id, inst.editor.getSelected() ?? null, ctxRef.current ?? blockCtx))
    },
    [canEdit, selectAdded, blockCtx],
  )

  const moveSelected = useCallback(
    (dir: -1 | 1) => {
      const c = peRef.current?.editor.getSelected()
      if (!c || !canEdit) return
      if (moveBlock(c, dir)) {
        peRef.current?.editor.select(c)
        bump()
      }
    },
    [canEdit, bump],
  )
  const duplicateSelected = useCallback(() => {
    const inst = peRef.current
    const c = inst?.editor.getSelected()
    if (!inst || !c || !canEdit) return
    const copy = duplicateBlock(inst, c)
    if (copy) inst.editor.select(copy)
  }, [canEdit])
  const removeSelected = useCallback(() => {
    const c = peRef.current?.editor.getSelected()
    if (!c || !canEdit || c.get('removable') === false) return
    const parent = blockOf(c.parent()) ?? topOf(c.parent())
    c.remove()
    if (parent && parent !== c) peRef.current?.editor.select(parent)
    setSelected(peRef.current?.editor.getSelected() ?? null)
  }, [canEdit])

  const startPublish = useCallback(() => {
    void (async () => {
      if (revision.current !== savedRevision.current || status.kind === 'conflict') {
        if (!(await saveRef.current({ manual: true, quiet: true }))) return
      }
      setDialog(page.publish.needsApproval && !page.publish.canPublish ? { kind: 'approval' } : { kind: 'publish' })
    })()
  }, [status.kind, page.publish.needsApproval, page.publish.canPublish])

  const value: Builder = {
    slug,
    page,
    setPage,
    applyDetail,
    canvas,
    pe,
    editor: pe?.editor ?? null,
    selected,
    select,
    rev,
    canEdit,
    canPublishPerm,
    device,
    setDevice,
    rightTab,
    setRightTab,
    status,
    save,
    reloadFromServer,
    refreshChecks,
    conflict,
    savedCard,
    closeSavedCard: () => setSavedCard(null),
    dialog,
    openDialog,
    publishWhen,
    setPublishWhen,
    insertBlock: insert,
    moveSelected,
    duplicateSelected,
    removeSelected,
    startPublish,
    focusField,
    setFocusField,
    tz,
    blockContext: blockCtx,
  }

  const unsaved = canEdit && isUnsaved(status)
  useUnsavedWork(canEdit ? { whileEnding: tt(m.saving) } : undefined)
  const savedVersion = status.version

  return (
    <BuilderCtx.Provider value={value}>
      <div className="cmp pb rc-app">
        <a href="#main" className="skip">
          {t(m.skip)}
        </a>
        <h1 className="vh">{t(m.heading, { name: page.name })}</h1>
        <Header />
        <PermissionNote />
        <div className="pb-wrap">
          <LeftPanel />
          <Canvas containerRef={containerRef} />
          <RightPanel />
        </div>
        <StatusBar />
        <Dialogs />
        <UnsavedChangesGuard
          dirty={unsaved}
          title={t(m.leaveTitle)}
          description={<T msg={m.leaveBody} values={{ version: savedVersion, next: savedVersion + 1 }} />}
          saveLabel={t(m.leaveSave, { version: savedVersion + 1 })}
          onSave={async () => {
            if (!(await save({ manual: true, quiet: true }))) throw new Error(tt(m.leaveSaveFailed))
          }}
        />
      </div>
    </BuilderCtx.Provider>
  )
}
