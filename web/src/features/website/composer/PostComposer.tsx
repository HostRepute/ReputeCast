import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { dismissToast, toast } from '@/components/ui/Toast'
import { T, t as tr, useT } from '@/i18n'
import { clock, dayMonthDate, longDate, orList } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useUnsavedWork } from '@/lib/sessionClock'
import { runShortcutHandler, useShortcutHandler } from '@/lib/shortcuts'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { CanvasArea } from '@/features/composer/CanvasArea'
import { ComposerCtx, type Campaign, type ComposerCtxData, type ComposerDialog, type ComposerState, type SaveStatus } from '@/features/composer/context'
import { Dialogs } from '@/features/composer/Dialogs'
import { trackChanges, type ChangeTracker } from '@/features/composer/editor/changes'
import { createComposerEditor, type ComposerEditor, type DeviceId } from '@/features/composer/editor/grapes'
import { componentLabel, tagOf } from '@/features/composer/editor/labels'
import { canvasHead, isEmptyBody, joinMjml, splitMjml, withHeadMeta } from '@/features/composer/editor/mjmlDoc'
import { describe, SettingsFields, StyleFields } from '@/features/composer/Inspector'
import { LeftPanel } from '@/features/composer/LeftPanel'
import { StateCards } from '@/features/composer/StateCards'
import { AccessibilityProvider } from '@/features/accessibility/composer/state'
import { ConditionalProvider } from '@/features/conditional/state'
import { DigestModeProvider } from '@/features/digests/composer/DigestMode'
import { WS, type Overview, type Post } from '../api'
import { UnpublishDialog } from '../PostDialogs'
import { dateTime, scheduledLine, visibilityIcon } from '../postFormat'
import { findMarker, isMarker, placeMarker, useMarkerOnCanvas, useSiteTheme } from './marker'
import { mc } from './messages'
import { postBlocks, PREVIEW_MARKER } from './postBlocks'
import { PreviewOnWeb } from './PreviewOnWeb'
import { useSavePost, useWebSettings, WebPanel } from './WebPanel'
import '@/features/composer/composer.css'
import './composer.css'


export interface PostComposerSearch {
  view?: 'preview'
  as?: 'visitor' | 'free' | 'paid'
  device?: 'desktop' | 'mobile'
  panel?: 'style' | 'settings' | 'web'
  publish?: boolean
}

const BACKUP_KEY = (id: string) => `rc-post-backup:${id}`

function offlineError(e: unknown) {
  return e instanceof TypeError || (e instanceof ApiError && (e.status === 502 || e.status === 503 || e.status === 504))
}

type PendingSource = { source: 'template' | 'import' | 'code' | 'restore'; label?: string }

/** The shared composer parts read a campaign; in post mode it describes the post. Nothing is sent to a campaign endpoint. */
function campaignShim(p: Post, canEdit: boolean): Campaign {
  return {
    access: { canEdit, canEditTemplates: false, canManageRouting: false, canSend: false, canSendTestToOthers: false, canViewContacts: false, editable: canEdit, sendHelpers: [] },
    audience: { lists: [], segments: [], excludeLists: [], excludeSegments: [] },
    audienceLabel: '',
    contentHash: '',
    createdAt: p.updatedAt,
    editorMode: 'visual',
    html: '',
    id: p.id,
    language: '',
    mjml: p.mjml,
    name: p.title,
    plainText: '',
    preheader: p.excerpt,
    replyTo: '',
    route: { shares: [], source: 'none' },
    settings: { minify: false },
    sourceLanguage: '',
    status: 'draft',
    subject: p.title,
    updatedAt: p.updatedAt,
    updatedBy: '',
    version: p.version,
    versionCount: p.version,
  } as Campaign
}

/** The post's body in the site's 680 px column (an email's 600 px body widens). */
export function webColumn(body: string): string {
  return body.replace(/<mj-body\b([^>]*)>/, (whole, attrs: string) => {
    if (/\bwidth\s*=/.test(attrs)) return whole.replace(/\bwidth\s*=\s*"600px"/, 'width="680px"')
    return `<mj-body${attrs} width="680px">`
  })
}

/** A new post starts with one text block. */
function starter(): string {
  const text = tr(mc.starterText).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return `<mjml><mj-body width="680px"><mj-section padding="0px"><mj-column><mj-text font-size="17px" line-height="1.7" color="#1F2937">${text}</mj-text></mj-column></mj-section></mj-body></mjml>`
}

/**
 * The composer in post mode (ComposerPublishToWeb): the canvas is the post's
 * web version in the site's 680 px column; the Web tab holds its address,
 * visibility, free preview and publishing; Preview on web shows the page as
 * a visitor, a free member or a paid member sees it.
 */
export function PostComposer({ post: initialPost, context, overview, search, onSearch }: { post: Post; context: ComposerCtxData; overview: Overview; search: PostComposerSearch; onSearch: (s: PostComposerSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canEdit = useCan('website:edit')
  const host = overview.address.domain || overview.address.host
  const web = useWebSettings({ post: initialPost, save: useSavePost(initialPost.id) })
  const post = web.post
  // Desktop is the site's 680 px reading column.
  const device: DeviceId = search.device === 'mobile' ? 'mobile' : 'web'
  const view = search.view === 'preview' ? 'preview' : 'visual'
  const [rightTab, setRightTabState] = useState<'style' | 'settings' | 'web'>(search.panel ?? 'web')
  // On a phone the inspector is a sheet: folded to its tabs until a tab, the handle or the visibility chip opens it.
  const [sheetOpen, setSheetOpen] = useState(Boolean(search.panel))

  const [starterMjml] = useState(starter)
  const source = initialPost.mjml.trim() ? initialPost.mjml : starterMjml
  const initial = useMemo(() => {
    const parts = splitMjml(source)
    return { ...parts, body: webColumn(parts.body) }
  }, [source])
  const [head, setHeadState] = useState(initial.head)
  const headRef = useRef(initial.head)
  const setHeadRaw = useCallback((h: string) => {
    headRef.current = h
    setHeadState(h)
  }, [])
  const [empty, setEmpty] = useState(() => isEmptyBody(source))
  const bodyRef = useRef(initial.body)
  const [ge, setGe] = useState<ComposerEditor | null>(null)
  const [selected, setSelected] = useState<Component | null>(null)
  const [rev, setRev] = useState(0)
  const [status, setStatus] = useState<SaveStatus>(() => (initialPost.autosavedAt ? { kind: 'saved', at: initialPost.autosavedAt, version: initialPost.version } : { kind: 'new' }))
  const [leftTab, setLeftTab] = useState<'blocks' | 'layers' | 'templates'>('blocks')
  const [dialog, setDialog] = useState<ComposerDialog>(null)
  const [savedCard, setSavedCard] = useState<{ version: number; at: string; template?: string } | null>(null)
  const [darkCanvas, setDarkCanvas] = useState(false)
  const [assetTarget, setAssetTarget] = useState<ComposerState['assetTarget']>(null)
  const [codeDirty, setCodeDirty] = useState(false)
  const [approvalOpen, setApprovalOpen] = useState(false)
  const [unpublishOpen, setUnpublishOpen] = useState(false)
  const [publishing, setPublishing] = useState(false)

  const version = useRef(initialPost.version)
  const revision = useRef(0)
  const savedRevision = useRef(0)
  const tracker = useRef<ChangeTracker | null>(null)
  const saving = useRef<Promise<boolean> | null>(null)
  const pendingSource = useRef<PendingSource | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const geRef = useRef<ComposerEditor | null>(null)
  const leaveCheck = useRef<(() => boolean) | null>(null)

  useDocumentTitle(t(mc.documentTitle, { title: web.draft.title || t(mc.untitled) }))

  const markDirty = useCallback(() => {
    revision.current++
    setRev((r) => r + 1)
    setStatus((s) => (s.kind === 'saving' || s.kind === 'conflict' || s.kind === 'offline' ? s : { kind: 'dirty', at: s.kind === 'saved' ? s.at : undefined, version: version.current }))
  }, [])

  const editorWanted = !empty
  useEffect(() => {
    if (!editorWanted || !containerRef.current) return
    let cancelled = false
    let created: ComposerEditor | null = null
    const mount = document.createElement('div')
    containerRef.current.appendChild(mount)
    void createComposerEditor(mount, { body: bodyRef.current, head: () => canvasHead(headRef.current), device })
      .then((inst) => {
        if (cancelled) {
          inst.editor.destroy()
          mount.remove()
          return
        }
        created = inst
        geRef.current = inst
        const ed = inst.editor
        if (import.meta.env.DEV) (window as unknown as { __rcComposer?: typeof ed }).__rcComposer = ed
        tracker.current = trackChanges(ed)
        ed.on('update', markDirty)
        ed.on('component:input', markDirty)
        ed.on('component:toggled', () => setSelected(ed.getSelected() ?? null))
        ed.on('component:remove', () => setSelected(ed.getSelected() ?? null))
        ed.on('undo redo', () => setRev((r) => r + 1))
        if (!canEdit) ed.getWrapper()?.onAll((c: Component) => c.set({ draggable: false, droppable: false, editable: false, removable: false, copyable: false }))
        setGe(inst)
        if (pendingSource.current) markDirty()
      })
      .catch((e: unknown) => toast({ kind: 'error', title: tr(mc.editorFailed), detail: e instanceof Error ? e.message : String(e) }))
    return () => {
      cancelled = true
      tracker.current?.dispose()
      tracker.current = null
      if (geRef.current === created) geRef.current = null
      created?.editor.destroy()
      if (created) mount.remove()
      else mount.style.display = 'none'
      setGe(null)
      setSelected(null)
    }
    // The editor is created once per content load; device changes are applied below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editorWanted])

  useEffect(() => {
    if (ge && ge.editor.getDevice() !== device) ge.editor.setDevice(device)
  }, [ge, device])

  useEffect(() => {
    if (!ge) return
    const body = ge.editor.getWrapper()
    for (const tag of ['mj-section', 'mj-hero', 'mj-wrapper', 'mj-raw']) body?.findType(tag).forEach((c: Component) => (c.view as unknown as { rerender?: () => void })?.rerender?.())
  }, [ge, head])

  useMarkerOnCanvas(ge?.editor ?? null, rev)
  useSiteTheme(ge?.editor ?? null)

  const canvasBody = useCallback(() => (geRef.current ? geRef.current.getBody() : bodyRef.current), [])
  const currentMjml = useCallback(() => {
    if (empty && !geRef.current) return ''
    return joinMjml(withHeadMeta(headRef.current, { title: web.draft.title, preview: web.draft.excerpt }), canvasBody())
  }, [empty, canvasBody, web.draft.title, web.draft.excerpt])
  const compileCurrent = useCallback(() => {
    const inst = geRef.current
    const src = currentMjml()
    if (!inst || !src) return null
    return inst.compile(src)
  }, [currentMjml])

  const loadContent = useCallback(
    (mjml: string, how: { source: 'template' | 'import' | 'code' | 'restore' | 'reload'; label?: string }) => {
      const parts = splitMjml(mjml)
      setHeadRaw(parts.head)
      bodyRef.current = parts.body
      if (how.source !== 'reload') pendingSource.current = { source: how.source, label: how.label }
      if (isEmptyBody(mjml) && how.source === 'reload') {
        setEmpty(true)
        return
      }
      setEmpty(false)
      const inst = geRef.current
      if (inst) {
        inst.setBody(parts.body)
        if (how.source !== 'reload') markDirty()
      } else if (how.source !== 'reload') markDirty()
    },
    [markDirty, setHeadRaw],
  )

  const refreshPost = useCallback(async () => {
    const { data } = await http.GET(`${WS}/posts/{post}`, { params: { path: { workspace: slug, post: initialPost.id } } })
    if (data) web.refresh(data)
    void qc.invalidateQueries({ queryKey: ['get', `${WS}/posts`] })
    void qc.invalidateQueries({ queryKey: ['website-post-preview', slug, initialPost.id] })
  }, [slug, initialPost.id, web, qc])

  const save = useCallback(
    async (opts: { manual?: boolean; quiet?: boolean; force?: boolean; background?: boolean; source?: 'autosave' | 'manual' | 'import' | 'template'; label?: string } = {}): Promise<boolean> => {
      if (!canEdit) return false
      if (geRef.current?.isEditingText()) {
        if (opts.background) return false
        await geRef.current.flushEditing()
      }
      if (saving.current) {
        await saving.current
        if (revision.current === savedRevision.current && !opts.force) return true
      }
      const run = async (): Promise<boolean> => {
        const rev0 = revision.current
        const mjml = currentMjml()
        const pending = pendingSource.current
        const label = opts.label ?? pending?.label ?? tracker.current?.take() ?? ''
        const wanted = opts.source ?? (pending ? (pending.source === 'code' ? 'manual' : pending.source) : opts.manual ? 'manual' : 'autosave')
        setStatus({ kind: 'saving', version: version.current })
        try {
          const { data } = await http.PUT(`${WS}/posts/{post}/content`, {
            params: { path: { workspace: slug, post: initialPost.id } },
            body: { baseVersion: version.current, mjml, label, source: wanted as 'autosave' | 'manual' | 'import' | 'template' | 'restore', force: opts.force || undefined },
          })
          version.current = data!.version
          savedRevision.current = rev0
          pendingSource.current = null
          try {
            window.localStorage.removeItem(BACKUP_KEY(initialPost.id))
          } catch {
            /* storage unavailable */
          }
          setStatus(revision.current === rev0 ? { kind: 'saved', at: data!.savedAt, version: data!.version } : { kind: 'dirty', at: data!.savedAt, version: data!.version })
          if (opts.manual && !opts.quiet) setSavedCard({ version: data!.version, at: data!.savedAt })
          void refreshPost()
          return true
        } catch (e) {
          if (e instanceof ApiError && e.status === 409 && e.code === 'version_conflict') {
            const m = (e.problem.meta ?? {}) as Record<string, unknown>
            setStatus({
              kind: 'conflict',
              version: version.current,
              theirs: { version: Number(m.version ?? 0), by: String(m.savedBy ?? ''), at: String(m.savedAt ?? ''), theirChanges: (m.theirChanges as string[]) ?? [], yourChanges: (m.yourChanges as string[]) ?? [] },
            })
            return false
          }
          if (offlineError(e)) {
            try {
              window.localStorage.setItem(BACKUP_KEY(initialPost.id), JSON.stringify({ at: new Date().toISOString(), baseVersion: version.current, mjml }))
            } catch {
              /* storage unavailable */
            }
            setStatus((s) => ({ kind: 'offline', since: s.kind === 'offline' ? s.since : new Date().toISOString(), version: version.current, message: tr(mc.offlineReason) }))
            return false
          }
          setStatus({ kind: 'error', version: version.current, message: errorMessage(e) })
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
    [canEdit, currentMjml, slug, initialPost.id, refreshPost],
  )

  // Autosave every 30 s while there are changes, and 2 s after a structural change.
  const saveRef = useRef(save)
  useLayoutEffect(() => {
    saveRef.current = save
  })
  useEffect(() => {
    if (!canEdit) return
    const h = window.setInterval(() => {
      if (revision.current !== savedRevision.current) void saveRef.current({ background: true })
    }, 30_000)
    return () => window.clearInterval(h)
  }, [canEdit])
  useEffect(() => {
    if (!canEdit || revision.current === savedRevision.current) return
    if (pendingSource.current || tracker.current?.structural()) {
      const h = window.setTimeout(() => void saveRef.current({ background: true }), pendingSource.current ? 300 : 2000)
      return () => window.clearTimeout(h)
    }
  }, [rev, canEdit])
  useEffect(() => {
    if (status.kind !== 'offline') return
    const h = window.setInterval(() => void saveRef.current({ background: true }), 10_000)
    return () => window.clearInterval(h)
  }, [status.kind])

  const reloadFromServer = useCallback(async () => {
    const { data } = await http.GET(`${WS}/posts/{post}`, { params: { path: { workspace: slug, post: initialPost.id } } })
    if (!data) return
    web.refresh(data)
    loadContent(data.mjml || starter(), { source: 'reload' })
    version.current = data.version
    tracker.current?.take()
    pendingSource.current = null
    savedRevision.current = revision.current
    const at = data.autosavedAt ?? data.updatedAt
    setStatus({ kind: 'saved', at, version: data.version })
    window.setTimeout(() => {
      savedRevision.current = revision.current
      setStatus({ kind: 'saved', at, version: data.version })
    }, 50)
  }, [slug, initialPost.id, web, loadContent])

  const registerLeaveCheck = useCallback((fn: () => boolean) => {
    leaveCheck.current = fn
    return () => {
      if (leaveCheck.current === fn) leaveCheck.current = null
    }
  }, [])
  const setView = useCallback(
    (v: 'visual' | 'preview') => {
      if (v === view) return
      if (v === 'preview' && revision.current !== savedRevision.current) void save()
      onSearch({ ...search, view: v === 'preview' ? 'preview' : undefined })
    },
    [view, onSearch, search, save],
  )
  const setRightTab = useCallback(
    (x: 'style' | 'settings' | 'web') => {
      setRightTabState(x)
      setSheetOpen(true)
      onSearch({ ...search, panel: x === 'web' ? undefined : x })
    },
    [onSearch, search],
  )
  useEffect(() => {
    if (view === 'visual' && ge) ge.editor.refresh()
  }, [view, ge])

  // The marker: where it is, and the Web tab's Add · Move · Remove.
  // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers canvas changes
  const marker = useMemo(() => findMarker(ge?.editor ?? null), [ge, rev])
  const markerApi = {
    at: marker ? marker.before : null,
    canMove: Boolean(selected && !isMarker(selected) && tagOf(selected)),
    add: () => {
      if (placeMarker(ge?.editor ?? null, selected)) markDirty()
    },
    move: () => {
      const m = findMarker(ge?.editor ?? null)
      if (!m || !selected || isMarker(selected)) return
      m.marker.remove()
      if (!placeMarker(ge?.editor ?? null, selected)) {
        const parent = selected.parent()
        parent?.append(PREVIEW_MARKER, { at: parent.components().indexOf(selected) + 1 })
      }
      markDirty()
    },
    remove: () => {
      findMarker(ge?.editor ?? null)?.marker.remove()
      markDirty()
    },
  }

  const blocks = useMemo(() => postBlocks(context.postalAddress), [context.postalAddress])
  const campaign = useMemo(() => campaignShim(post, canEdit), [post, canEdit])

  const state: ComposerState = {
    slug,
    campaign,
    context,
    ge,
    editor: ge?.editor ?? null,
    selected,
    view: view === 'preview' ? 'preview' : 'visual',
    device,
    variant: 'A',
    setView: (v) => setView(v === 'preview' ? 'preview' : 'visual'),
    setDevice: (d: DeviceId) => onSearch({ ...search, device: d === 'mobile' ? 'mobile' : undefined }),
    setVariant: () => undefined,
    name: web.draft.title,
    subject: web.draft.title,
    preheader: web.draft.excerpt,
    abTest: null,
    setName: (v: string) => web.set({ title: v }),
    commitName: () => undefined,
    setSubject: (v: string) => web.set({ title: v }),
    setPreheader: (v: string) => web.set({ excerpt: v }),
    setABTest: () => undefined,
    head,
    setHead: (h: string) => {
      setHeadRaw(h)
      markDirty()
    },
    status,
    save,
    currentMjml,
    compileCurrent,
    loadContent,
    reloadFromServer,
    onSettingsChanged: () => undefined,
    registerLeaveCheck,
    setCodeDirty,
    isEmpty: empty,
    checks: undefined,
    checksPending: false,
    rightTab: rightTab === 'web' ? 'style' : rightTab,
    setRightTab: (x) => setRightTab(x === 'settings' ? 'settings' : 'style'),
    leftTab,
    setLeftTab,
    openDialog: (d: ComposerDialog) => setDialog(d === 'versions' || d === 'compare' || d === 'sendTest' ? null : d),
    rev,
    canEdit,
    darkCanvas,
    setDarkCanvas,
    assetTarget,
    setAssetTarget,
    previewSlot: null,
    setPreviewSlot: () => undefined,
  }

  // Publishing: now, at a time, or a request for approval.
  const publish = useCallback(async () => {
    if (!(await web.flush())) return
    if (revision.current !== savedRevision.current && !(await save({ manual: true, quiet: true }))) return
    const pub = web.post.publish
    if (pub.needsApproval && !pub.approvedId) {
      setApprovalOpen(true)
      return
    }
    setPublishing(true)
    try {
      const at = web.draft.publishMode === 'at' && web.draft.publishAt ? web.draft.publishAt : undefined
      const { data } = await http.POST(`${WS}/posts/{post}/publish`, {
        params: { path: { workspace: slug, post: initialPost.id } },
        body: at ? { when: 'at', at, tz: web.draft.publishTz } : { when: 'now' },
      })
      if (data) web.refresh(data)
      void qc.invalidateQueries({ queryKey: ['get', `${WS}/posts`] })
      if (at) {
        toast({ kind: 'success', title: t(mc.scheduledToast, { title: web.draft.title, date: scheduledLine(at, web.draft.publishTz) }) })
      } else {
        const id = toast({
          kind: 'success',
          title: t(mc.published, { title: web.draft.title, host, time: clock(new Date()) }),
          action: { label: t(mc.viewPost), onClick: () => window.open(data?.url ?? post.url, '_blank', 'noopener') },
          undo: () => {
            dismissToast(id)
            void http
              .POST(`${WS}/posts/{post}/undo-publish`, { params: { path: { workspace: slug, post: initialPost.id } } })
              .then(({ data: d }) => {
                if (d) web.refresh(d)
                toast({ kind: 'info', title: t(mc.undone) })
              })
              .catch((e: unknown) => toast({ kind: 'error', title: t(mc.undoFailed, { error: errorMessage(e) }) }))
          },
          duration: 10_000,
        })
      }
    } catch (e) {
      toast({ kind: 'error', title: t(mc.publishFailed), detail: errorMessage(e) })
    } finally {
      setPublishing(false)
    }
  }, [web, save, slug, initialPost.id, qc, t, host, post.url])

  // The approval notification opens the editor with ?publish=1; the link is used once.
  useEffect(() => {
    if (!search.publish) return
    onSearch({ ...search, publish: undefined })
    const h = window.setTimeout(() => void publish(), 0)
    return () => window.clearTimeout(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useShortcutHandler(
    'save',
    canEdit
      ? () => {
          void save({ manual: true })
        }
      : undefined,
  )
  useShortcutHandler('readiness', canEdit ? () => void publish() : undefined)
  useEffect(() => {
    const onFrameKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.altKey) return
      const key = e.key.toLowerCase()
      if (key === 's' && !e.shiftKey && runShortcutHandler('save')) e.preventDefault()
      else if (e.key === 'Enter' && runShortcutHandler('readiness')) e.preventDefault()
      else if (e.key === '/') {
        e.preventDefault()
        setDialog('shortcuts')
      }
    }
    const frameDoc = ge?.editor.Canvas.getDocument()
    frameDoc?.addEventListener('keydown', onFrameKey)
    return () => frameDoc?.removeEventListener('keydown', onFrameKey)
  }, [ge])

  const unsaved = status.kind === 'dirty' || status.kind === 'offline' || status.kind === 'error' || status.kind === 'conflict' || web.status === 'dirty' || web.status === 'error'
  useUnsavedWork(canEdit ? { whileEnding: t(mc.whileEnding), afterEnded: t(mc.afterEnded) } : undefined)

  return (
    <ComposerCtx.Provider value={state}>
      <AccessibilityProvider>
        <DigestModeProvider>
          <ConditionalProvider>
            <div className="cmp rc-app pc-post">
              <a href="#main" className="skip">
                {t(mc.skip)}
              </a>
              <h1 className="vh">{t(mc.heading, { title: web.draft.title || t(mc.untitled) })}</h1>
              <PostHeader
                post={post}
                title={web.draft.title}
                onTitle={(v) => web.set({ title: v })}
                status={status}
                settingsStatus={web.status}
                canEdit={canEdit}
                publishing={publishing}
                onPreview={() => setView(view === 'preview' ? 'visual' : 'preview')}
                onPublish={() => void publish()}
                onUnpublish={() => setUnpublishOpen(true)}
                onVisibility={() => setRightTab('web')}
              />
              {post.publish.needsApproval && canEdit ? <WriterBanner post={post} /> : null}
              {view === 'preview' ? <PreviewOnWeb post={post} as={search.as ?? 'visitor'} onAs={(as) => onSearch({ ...search, as: as === 'visitor' ? undefined : as })} onBack={() => setView('visual')} /> : null}
              <div className={`flex flex-1 flex-col${view === 'preview' ? ' cmp-offstage' : ''}`} inert={view === 'preview' || undefined} aria-hidden={view === 'preview' || undefined}>
                <div className="cmp-body">
                  <LeftPanel blocks={blocks} note={<p className="pc-bnote">{t(mc.blocksNote)}</p>} reputecastFirst={false} />
                  <div className="cmp-main">
                    <CanvasArea
                      containerRef={containerRef}
                      savedCard={null}
                      onCloseSaved={() => setSavedCard(null)}
                      onReview={() => void publish()}
                      slots={{
                        toolbar: () => <PostToolbar view={view} onView={setView} device={device} onDevice={(d) => onSearch({ ...search, device: d === 'mobile' ? 'mobile' : undefined })} />,
                        cards: (
                          <>
                            <StateCards saved={savedCard} onCloseSaved={() => setSavedCard(null)} onReview={() => void publish()} />
                            <PostHead post={post} title={web.draft.title} excerpt={web.draft.excerpt} cover={web.draft.coverUrl} alt={web.draft.coverAlt} phone={device === 'mobile'} />
                          </>
                        ),
                        below: null,
                      }}
                    />
                  </div>
                  <PostRightPanel tab={rightTab} onTab={setRightTab} open={sheetOpen} onOpen={setSheetOpen} web={<WebPanel ws={web} mode="post" host={host} canEdit={canEdit} marker={markerApi} />} />
                </div>
                <PostFooter post={post} />
              </div>
              <Dialogs dialog={dialog} onClose={() => setDialog(null)} onSavedTemplate={(x) => setSavedCard({ version: version.current, at: new Date().toISOString(), template: x })} />
              {approvalOpen ? <RequestApprovalDialog post={post} draft={web.draft} host={host} onClose={() => setApprovalOpen(false)} onSent={(p) => web.refresh(p)} /> : null}
              {unpublishOpen ? (
                <UnpublishDialog
                  post={post}
                  onClose={() => {
                    setUnpublishOpen(false)
                    void refreshPost()
                  }}
                />
              ) : null}
              <UnsavedChangesGuard
                dirty={canEdit && (unsaved || codeDirty)}
                title={t(mc.leaveTitle)}
                description={t(mc.leaveText)}
                saveLabel={t(mc.leaveSave)}
                onSave={async () => {
                  if (!(await web.flush()) || !(await save({ manual: true, quiet: true }))) throw new Error(t(mc.leaveFailed))
                }}
              />
            </div>
          </ConditionalProvider>
        </DigestModeProvider>
      </AccessibilityProvider>
    </ComposerCtx.Provider>
  )
}

/** Post mode's header: back, the title, its state and visibility, Preview and Publish (or Request approval). */
function PostHeader({
  post,
  title,
  onTitle,
  status,
  settingsStatus,
  canEdit,
  publishing,
  onPreview,
  onPublish,
  onUnpublish,
  onVisibility,
}: {
  post: Post
  title: string
  onTitle: (v: string) => void
  status: SaveStatus
  settingsStatus: 'saved' | 'dirty' | 'saving' | 'error'
  canEdit: boolean
  publishing: boolean
  onPreview: () => void
  onPublish: () => void
  onUnpublish: () => void
  onVisibility: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const canPublish = useCan('website:publish')
  const pub = post.publish
  const saving = status.kind === 'saving' || settingsStatus === 'saving'
  let badge: React.ReactNode
  if (post.status === 'published') {
    badge = (
      <Badge tone="ok" dot>
        {post.publishedVersion < post.version ? t(mc.publishedChanges) : t(mc.publishedBadge, { date: post.publishedAt ? `${dayMonthDate(post.publishedAt)}, ${clock(post.publishedAt)}` : '' })}
      </Badge>
    )
  } else if (post.status === 'scheduled') {
    badge = <Badge tone="info">{t(mc.scheduledBadge, { date: post.publishAt ? scheduledLine(post.publishAt, post.publishTz) : '' })}</Badge>
  } else if (post.status === 'unpublished') {
    badge = <Badge tone="neutral">{t(mc.unpublishedBadge, { date: post.unpublishedAt ? dayMonthDate(post.unpublishedAt) : '' })}</Badge>
  } else if (pub.pending) {
    badge = <Badge tone="warn">{t(mc.waitingBadge)}</Badge>
  } else {
    const at = status.kind === 'saved' || status.kind === 'dirty' ? status.at : undefined
    badge = (
      <Badge tone="neutral">
        {saving ? t(mc.saving) : status.kind === 'dirty' || settingsStatus === 'dirty' ? t(mc.draftUnsaved) : at ? t(mc.draftAutosaved, { time: clock(at) }) : t(mc.draftNew)}
      </Badge>
    )
  }
  const tiers = post.tiers.map((x) => x.name)
  const vis = post.visibility === 'paid' ? (tiers.length ? t(mc.visPaidTiers, { tiers: tiers.join(', ') }) : t(mc.visPaidAll)) : post.visibility === 'members' ? t(mc.visMembers) : t(mc.visPublic)
  const primaryLabel = pub.needsApproval && !pub.approvedId ? t(mc.requestApproval) : post.status === 'published' ? t(mc.update) : post.publishMode === 'at' ? t(mc.schedule) : t(mc.publish)
  return (
    <header className="pc-head">
      <button type="button" className="btn ic" aria-label={t(mc.back)} onClick={() => void navigate({ to: '/w/$workspace/forms/website', params: { workspace: slug } })}>
        <Icon name="chevronLeft" size={16} />
      </button>
      <div className="ttl">
        <Link to="/w/$workspace/forms/website" params={{ workspace: slug }} className="crumb">
          {t(mc.crumb)}
        </Link>
        <input aria-label={t(mc.titleLabel)} value={title} size={Math.max(12, Math.min(60, title.length + 1))} disabled={!canEdit} onChange={(e) => onTitle(e.target.value)} />
      </div>
      <div className="mid">
        {badge}
        <button type="button" className="pc-vis" onClick={onVisibility}>
          <Icon name={visibilityIcon(post.visibility)} size={14} />
          {vis}
        </button>
      </div>
      <div className="acts">
        {post.status === 'published' ? (
          <>
            <a className="btn" href={post.url} target="_blank" rel="noreferrer">
              {t(mc.viewPost)}
            </a>
            {canPublish ? <Button onClick={onUnpublish}>{t(mc.unpublish)}</Button> : null}
          </>
        ) : (
          <Button onClick={onPreview}>{t(mc.preview)}</Button>
        )}
        {pub.pending && post.status !== 'published' ? (
          <Button variant="primary" onClick={() => void navigate({ to: '/w/$workspace/approvals/$approvalId', params: { workspace: slug, approvalId: pub.pending!.id } })}>
            {t(mc.openRequest)}
          </Button>
        ) : canEdit && (canPublish || pub.needsApproval) && (post.status !== 'published' || post.publishedVersion < post.version) ? (
          <Button variant="primary" loading={publishing} busyLabel={t(mc.publishing)} disabledReason={pub.blocked ? t(mc.blocked, { reason: pub.blockReason ?? '' }) : undefined} onClick={onPublish}>
            {primaryLabel}
          </Button>
        ) : null}
      </div>
    </header>
  )
}

/** "Daniel Park · Content Writer. You can write and edit posts. Your role can't publish them…" */
function WriterBanner({ post }: { post: Post }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data?.user
  const rule = post.publish.rule === 'post_publish' ? t(mc.rulePostPublish) : t(mc.ruleAuthorLacksSend)
  return (
    <div className="pc-banner" role="status">
      <Avatar name={user?.name ?? ''} size={24} color="warn" />
      <span>
        <T
          msg={mc.writerNote}
          values={{ name: <b>{user?.name ?? ''}</b>, role: <b>{me.workspace.roleName}</b>, rule, approvers: orList(post.publish.approvers) }}
        />
      </span>
    </div>
  )
}

/** The canvas toolbar in post mode: Desktop · 680 / Phone · 375, Edit / Preview on web, undo and redo. */
function PostToolbar({ view, onView, device, onDevice }: { view: 'visual' | 'preview'; onView: (v: 'visual' | 'preview') => void; device: DeviceId; onDevice: (d: DeviceId) => void }) {
  const t = useT()
  return (
    <div className="cmp-tools">
      <div className="seg" role="group" aria-label={t(mc.device)}>
        <button type="button" aria-pressed={device !== 'mobile'} onClick={() => onDevice('web')}>
          {t(mc.desktop)}
        </button>
        <button type="button" aria-pressed={device === 'mobile'} onClick={() => onDevice('mobile')}>
          {t(mc.phone)}
        </button>
      </div>
      <div className="seg" role="group" aria-label={t(mc.viewLabel)}>
        <button type="button" aria-pressed={view === 'visual'} onClick={() => onView('visual')}>
          {t(mc.edit)}
        </button>
        <button type="button" aria-pressed={view === 'preview'} onClick={() => onView('preview')}>
          {t(mc.previewOnWeb)}
        </button>
      </div>
      <UndoRedoButtons />
      <span className="k" style={{ marginLeft: 'auto' }}>
        {t(mc.postNote)}
      </span>
    </div>
  )
}

function UndoRedoButtons() {
  const t = useT()
  const c = useContext(ComposerCtx)
  const um = c?.editor?.UndoManager
  const canUndo = !!um?.hasUndo()
  const canRedo = !!um?.hasRedo()
  return (
    <div className="flex gap-1">
      <button type="button" className="btn ic" aria-label={t(mc.undo)} title={t(mc.undoTitle)} disabled={!canUndo || !c?.canEdit} onClick={() => um?.undo()}>
        <Icon name="undo" size={15} />
      </button>
      <button type="button" className="btn ic" aria-label={t(mc.redo)} title={t(mc.redoTitle)} disabled={!canRedo || !c?.canEdit} onClick={() => um?.redo()}>
        <Icon name="redo" size={15} />
      </button>
    </div>
  )
}


/** The post's head above the canvas, as the site shows it: tags, title, excerpt, byline and cover. */
function PostHead({ post, title, excerpt, cover, alt, phone }: { post: Post; title: string; excerpt: string; cover: string; alt: string; phone: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const access = $api.useQuery('get', `${WS}/access`, { params: { path: { workspace: slug } } }, { staleTime: 60_000, enabled: post.visibility === 'paid' })
  // The paid label, as the site shows it: the post's tier, else the first tier's.
  const paid = post.tiers[0]?.label || post.tiers[0]?.name || access.data?.tiers[0]?.label || t(mc.visPaid)
  // A tag that only repeats the paid label is left to the label, as on the site.
  const tags = post.tags.filter((x) => !(post.visibility === 'paid' && x.name.trim().toLowerCase() === paid.trim().toLowerCase()))
  return (
    <div className={`pc-posthead${phone ? ' phone' : ''}`} aria-label={t(mc.postHead)} role="group">
      {tags.length || post.visibility === 'paid' ? (
        <div className="tags">
          {tags.map((x) => (
            <span key={x.id}>{x.name}</span>
          ))}
          {post.visibility === 'paid' ? (
            <span className="ins">
              <Icon name="star" size={12} />
              {paid}
            </span>
          ) : null}
        </div>
      ) : null}
      <h2>{title || t(mc.untitled)}</h2>
      {excerpt ? <p className="ex">{excerpt}</p> : null}
      <div className="by">
        <Avatar name={post.byline.name} size={36} color="ok" />
        <span className="flex flex-col">
          <span className="nm">{post.byline.name}</span>
          <span className="k">
            {longDate(post.publishedAt ?? post.publishAt ?? new Date().toISOString())} · {t(mc.minRead, { n: Math.max(1, post.readingMinutes) })}
          </span>
        </span>
      </div>
      {cover ? (
        <div className="cv">
          <img src={cover} alt={alt} />
        </div>
      ) : null}
    </div>
  )
}

/** Style · Settings · Web. */
function PostRightPanel({
  tab,
  onTab,
  open,
  onOpen,
  web,
}: {
  tab: 'style' | 'settings' | 'web'
  onTab: (t: 'style' | 'settings' | 'web') => void
  open: boolean
  onOpen: (open: boolean) => void
  web: React.ReactNode
}) {
  const t = useT()
  const c = useContext(ComposerCtx)!
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  const head = sel ? (
    <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
      <span className="font-semibold">{isMarker(sel) ? t(mc.blockFreePreview) : componentLabel(sel)}</span>
      <span className="k">{describe(sel, tab === 'settings' ? 'settings' : 'style')}</span>
    </div>
  ) : null
  const nothing = (
    <div className="k" style={{ padding: '14px 12px' }}>
      {t(mc.nothingSelected)}
    </div>
  )
  return (
    <aside
      className={`cpanel right a11y-wide${open ? '' : ' closed'}`}
      aria-label={t(mc.inspector)}
      // A tap on any tab, the current one included, unfolds the phone sheet.
      onClickCapture={(e) => {
        if (!open && (e.target as HTMLElement).closest('[role="tab"]')) onOpen(true)
      }}
    >
      {/* On a phone the panel is a sheet over the canvas: the handle folds it down to its tabs. */}
      <button type="button" className="pc-grab" aria-expanded={open} aria-label={open ? t(mc.sheetHide) : t(mc.sheetShow)} onClick={() => onOpen(!open)}>
        <span aria-hidden="true" />
      </button>
      <PanelTabs
        value={tab}
        onChange={onTab}
        label={t(mc.inspector)}
        items={[
          { value: 'style', label: t(mc.tabStyle) },
          { value: 'settings', label: t(mc.tabSettings) },
          { value: 'web', label: t(mc.tabWeb) },
        ]}
        className="flex flex-1 flex-col"
      >
        <TabPanel value="style" className="flex flex-1 flex-col" tabIndex={-1}>
          {sel ? (
            <>
              {head}
              <fieldset disabled={!c.canEdit} className="contents">
                <StyleFields key={sel.cid} c={sel} />
              </fieldset>
            </>
          ) : (
            nothing
          )}
        </TabPanel>
        <TabPanel value="settings" className="flex flex-1 flex-col" tabIndex={-1}>
          {sel ? (
            <>
              {head}
              <fieldset disabled={!c.canEdit} className="contents">
                <SettingsFields key={sel.cid} c={sel} />
              </fieldset>
            </>
          ) : (
            nothing
          )}
        </TabPanel>
        <TabPanel value="web" className="flex flex-1 flex-col overflow-y-auto" tabIndex={-1}>
          {web}
        </TabPanel>
      </PanelTabs>
    </aside>
  )
}

/** "2,480 words · 11 min read · 1 image, with alt text · No warnings · Version history · 9 versions". */
function PostFooter({ post }: { post: Post }) {
  const t = useT()
  const c = useContext(ComposerCtx)!
  const path: string[] = []
  let cur = c.selected ?? undefined
  while (cur && tagOf(cur) && tagOf(cur) !== 'mj-body') {
    path.unshift(tagOf(cur).replace(/^mj-/, ''))
    cur = cur.parent()
  }
  const blocks = post.checks.filter((x) => x.status === 'block' || x.status === 'warn').length
  return (
    <footer className="pc-foot">
      <span className="path" aria-label={t(mc.selectedBlock)}>
        {t(mc.postWord)}
        {path.length ? ' › ' : ''}
        {path.length ? <b>{path.join(' · ')}</b> : null}
      </span>
      <span className="figs">
        <span>
          {t(mc.wordsCount, { count: post.words })} · {t(mc.minRead, { n: Math.max(1, post.readingMinutes) })}
        </span>
        {blocks ? (
          <Badge tone="warn" dot>
            {blocks}
          </Badge>
        ) : (
          <Badge tone="ok" dot>
            {t(mc.noWarnings)}
          </Badge>
        )}
        <span>{t(mc.versions, { count: post.version })}</span>
      </span>
    </footer>
  )
}

/** Request approval to publish (ComposerPublishToWeb · a Content Writer requesting approval). */
function RequestApprovalDialog({ post, draft, host, onClose, onSent }: { post: Post; draft: { slug: string; visibility: string; publishMode: string; publishAt?: string; publishTz?: string }; host: string; onClose: () => void; onSent: (p: Post) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  void host
  const pub = post.publish
  const vis = draft.visibility === 'paid' ? t(mc.visPaid) : draft.visibility === 'members' ? t(mc.visMembers) : t(mc.visPublic)
  const when = draft.publishMode === 'at' && draft.publishAt ? scheduledLine(draft.publishAt, draft.publishTz) : t(mc.whenApproved)
  const send = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST(`${WS}/posts/{post}/approval`, { params: { path: { workspace: slug, post: post.id } }, body: { note: note.trim() || undefined } })
      if (data) onSent(data)
      toast({ kind: 'success', title: t(mc.approvalSent), detail: t(mc.approvalSentDetail, { approvers: orList(pub.approvers) }) })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mc.approvalEyebrow, { title: post.title })}</span>}
      title={t(mc.approvalTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void send()}>
            {t(mc.requestApproval)}
          </Button>
        </>
      }
    >
      <dl className="ws-kv">
        <dt>{t(mc.approvalRule)}</dt>
        <dd>{pub.rule === 'post_publish' ? t(mc.approvalRulePost) : t(mc.approvalRuleSend)}</dd>
        <dt>{t(mc.approvalApprovers)}</dt>
        <dd>
          {t(mc.approvalApproversText, { approvers: orList(pub.approvers) })}
        </dd>
        <dt>{t(mc.approvalCovers)}</dt>
        <dd>{t(mc.approvalCoversText, { path: `/posts/${draft.slug}`, visibility: vis, when })}</dd>
      </dl>
      <label className="lbl">
        <span>{t(mc.approvalNote)}</span>
        <textarea className="in" rows={3} value={note} maxLength={1000} onChange={(e) => setNote(e.target.value)} />
        <span className="k">{t(mc.approvalNoteHint)}</span>
      </label>
      {error ? (
        <Note tone="bad">
          {t(mc.approvalFailed)}: {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** "13 Oct 08:00" for the header. */
export function headerDate(d: string): string {
  return dateTime(d)
}
