import { useQueryClient } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { lazy, Suspense, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useUnsavedWork } from '@/lib/sessionClock'
import { runShortcutHandler, useShortcutHandler } from '@/lib/shortcuts'
import { useCan, useWorkspace } from '@/lib/session'
import { CanvasArea } from '../CanvasArea'
import { ComposerCtx, type Campaign, type ComposerCtxData, type ComposerDialog, type ComposerState, type SaveStatus, type View } from '../context'
import { Dialogs } from '../Dialogs'
import { trackChanges, type ChangeTracker } from '../editor/changes'
import { createComposerEditor, type ComposerEditor, type DeviceId } from '../editor/grapes'
import { canvasHead, isEmptyBody, joinMjml, splitMjml, withHeadMeta } from '../editor/mjmlDoc'
import { LeftPanel } from '../LeftPanel'
import { transactionalBlocks } from './blocks'
import { TxFooter } from './Footer'
import { TxHeader, ReadOnlyBanner } from './Header'
import { HtmlView } from './HtmlView'
import { m } from './messages'
import { PublishDialog } from './PublishDialog'
import { TxRightPanel } from './RightPanel'
import { TxCtx, type DraftMeta, type TemplateMetaPatch, type TxState } from './state'
import { TxStateCards, type SavedCard } from './StateCards'
import { TxSubjectBar } from './SubjectBar'
import { TextView } from './TextView'
import { useCanvasTokens } from './tokens'
import { TxToolbar } from './Toolbar'
import type { Category, Checks, Data, Preview, Settings, Template, TestData, TxTab, TxView, Variable } from './types'
import { dataFor, removeOutputs, renameInData, renameInSource, type TokenView } from './variables'
import { VersionsDialog, type CompareTarget } from './VersionsDialog'
import '../composer.css'
import './transactional.css'

// CodeMirror loads with the code view only.
const CodeView = lazy(() => import('../CodeView').then((x) => ({ default: x.CodeView })))

export interface TxSearch {
  view?: TxView
  device?: DeviceId
  tab?: TxTab
  /** Open the publish dialog (the approval notification links here with it). */
  publish?: boolean
}

const BACKUP_KEY = (id: string) => `rc-transactional-backup:${id}`

function offlineError(e: unknown) {
  return e instanceof TypeError || (e instanceof ApiError && (e.status === 502 || e.status === 503 || e.status === 504))
}

type PendingSource = { source: 'template' | 'import' | 'code' | 'restore'; label?: string }

/**
 * The shared composer parts read a campaign; in transactional mode it
 * describes the template (ComposerState.kind says so). Nothing here is
 * ever sent to a campaign endpoint.
 */
function campaignShim(t: Template, canEditTemplates: boolean): Campaign {
  return {
    access: {
      canEdit: t.access.canEdit,
      canEditTemplates,
      canManageRouting: false,
      canSend: false,
      canSendTestToOthers: t.access.canSendTestToOthers,
      canViewContacts: false,
      editable: t.access.canEdit,
      sendHelpers: [],
    },
    audience: { lists: [], segments: [], excludeLists: [], excludeSegments: [] },
    audienceLabel: '',
    contentHash: t.draftHash,
    createdAt: t.createdAt,
    editorMode: t.editorMode,
    html: t.html,
    id: t.id,
    language: '',
    mjml: t.mjml,
    name: t.name,
    plainText: t.plainText,
    preheader: t.preheader,
    replyTo: t.replyTo,
    route: { shares: [], source: 'none' },
    settings: { minify: false },
    sourceLanguage: '',
    status: 'draft',
    subject: t.subject,
    updatedAt: t.updatedAt,
    updatedBy: t.draftSavedBy,
    version: t.draftRevision,
    versionCount: t.versionsCount,
  }
}

/**
 * The template editor (TransactionalTemplate): the campaign composer in
 * transactional mode. The canvas, blocks, layers, inspector and code view
 * are the composer's; the header, subject bar, toolbar, right panel
 * (Design · Variables · Test data · Checks · API), save states, footer and
 * the publish flow are this folder's. Drafts autosave against X02's draft
 * API with its version_conflict, as campaigns do.
 */
export function TransactionalComposer({
  template,
  settings,
  categories,
  context,
  search,
  onSearch,
}: {
  template: Template
  settings: Settings | undefined
  categories: Category[]
  context: ComposerCtxData | undefined
  search: TxSearch
  onSearch: (s: TxSearch) => void
}) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const workspaceName = me.workspace.name
  const qc = useQueryClient()
  const canEditTemplates = useCan('templates:edit')
  const [tpl, setTpl] = useState(template)
  const canEdit = tpl.access.canEdit
  const canPublish = tpl.access.canPublish

  const htmlOnly = template.editorMode === 'html' && !template.mjml.trim()
  const [htmlMode, setHtmlMode] = useState(htmlOnly)
  const view: TxView = search.view ?? (htmlMode ? 'html' : 'visual')
  const device: DeviceId = search.device ?? 'desktop'
  const [tab, setTabState] = useState<TxTab>(search.tab ?? 'variables')
  const [designTab, setDesignTab] = useState<'style' | 'settings'>('style')

  const initial = useMemo(() => splitMjml(template.mjml), [template.mjml])
  const [head, setHeadState] = useState(initial.head)
  const headRef = useRef(initial.head)
  const setHeadRaw = useCallback((h: string) => {
    headRef.current = h
    setHeadState(h)
  }, [])
  const [name, setName] = useState(template.name)
  const [subject, setSubjectState] = useState(template.subject)
  const [preheader, setPreheaderState] = useState(template.preheader)
  const [variables, setVariablesState] = useState<Variable[]>(template.variables)
  const [testData, setTestDataState] = useState<TestData[]>(template.testData)
  const [previewWith, setPreviewWith] = useState(template.testData[0]?.name ?? '')
  const [meta, setMetaState] = useState<DraftMeta>({ senderId: template.sender?.id ?? '', replyTo: template.replyTo, trackOpens: template.trackOpens, trackClicks: template.trackClicks })
  const [plainText, setPlainTextState] = useState(template.plainText)
  const [html, setHtmlState] = useState(template.html)
  const [empty, setEmpty] = useState(() => !htmlOnly && isEmptyBody(template.mjml))
  const bodyRef = useRef(initial.body)

  const [ge, setGe] = useState<ComposerEditor | null>(null)
  const [selected, setSelected] = useState<Component | null>(null)
  const [rev, setRev] = useState(0)
  const [status, setStatus] = useState<SaveStatus>({ kind: 'saved', at: template.draftSavedAt, version: template.draftRevision })
  const [checks, setChecks] = useState<Checks>()
  const [checksPending, setChecksPending] = useState(false)
  const [checksNonce, setChecksNonce] = useState(0)
  const [preview, setPreview] = useState<Preview>()
  const [leftTab, setLeftTab] = useState<'blocks' | 'layers' | 'templates'>('blocks')
  const [dialog, setDialog] = useState<ComposerDialog>(null)
  const [publishOpen, setPublishOpen] = useState(() => !!search.publish && template.access.canPublish)
  const [versions, setVersions] = useState<CompareTarget | 'list' | null>(null)
  const [savedCard, setSavedCard] = useState<SavedCard | null>(null)
  const [darkCanvas, setDarkCanvas] = useState(false)
  const [assetTarget, setAssetTarget] = useState<ComposerState['assetTarget']>(null)
  const [codeDirty, setCodeDirty] = useState(false)
  const [focusVariable, setFocusVariable] = useState<string | null>(null)

  const baseRev = useRef(template.draftRevision)
  const revision = useRef(0)
  const savedRevision = useRef(0)
  const tracker = useRef<ChangeTracker | null>(null)
  /** What changed outside the canvas (variables, test data, settings), for the revision's label. */
  const notes = useRef<string[]>([])
  const saving = useRef<Promise<boolean> | null>(null)
  const pendingSource = useRef<PendingSource | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const geRef = useRef<ComposerEditor | null>(null)
  const leaveCheck = useRef<(() => boolean) | null>(null)

  useDocumentTitle(t(m.documentTitle, { name: name || t(m.untitled) }))

  const markDirty = useCallback(() => {
    revision.current++
    setRev((r) => r + 1)
    setStatus((s) => (s.kind === 'saving' || s.kind === 'conflict' || s.kind === 'offline' ? s : { kind: 'dirty', at: s.kind === 'saved' ? s.at : undefined, version: baseRev.current }))
  }, [])
  const note = useCallback((label: string) => {
    if (!notes.current.includes(label)) notes.current.push(label)
  }, [])

  // The GrapesJS editor: created once per content load, as in the composer.
  const editorWanted = !empty && !htmlMode
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
      .catch((e: unknown) => toast({ kind: 'error', title: t(m.editorFailed), detail: e instanceof Error ? e.message : String(e) }))
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

  const canvasBody = useCallback(() => (geRef.current ? geRef.current.getBody() : bodyRef.current), [])

  const currentMjml = useCallback(() => {
    if (htmlMode || (empty && !geRef.current)) return ''
    return joinMjml(withHeadMeta(headRef.current, { title: subject, preview: preheader }), canvasBody())
  }, [htmlMode, empty, subject, preheader, canvasBody])

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

  const editorMode = htmlMode ? 'html' : view === 'mjml' ? 'mjml' : 'visual'

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
        const compiled = htmlMode ? null : compileCurrent()
        const pending = pendingSource.current
        const parts = [opts.label ?? pending?.label ?? '', tracker.current?.take() ?? '', ...notes.current].filter(Boolean)
        const label = Array.from(new Set(parts)).slice(0, 3).join(', ')
        // The draft API knows autosave, manual, import and restore; a template applied counts as manual.
        const wanted = opts.source ?? (pending ? pending.source : opts.manual ? 'manual' : 'autosave')
        const source = wanted === 'import' || wanted === 'restore' || wanted === 'autosave' ? wanted : 'manual'
        setStatus({ kind: 'saving', version: baseRev.current })
        try {
          const { data } = await http.PUT('/api/v1/workspaces/{workspace}/transactional/templates/{template}/draft', {
            params: { path: { workspace: slug, template: tpl.id } },
            body: {
              baseRevision: baseRev.current,
              subject,
              preheader,
              editorMode,
              mjml,
              html: htmlMode ? html : compiled?.html || undefined,
              plainText,
              variables,
              testData,
              senderId: meta.senderId,
              replyTo: meta.replyTo,
              trackOpens: meta.trackOpens,
              trackClicks: meta.trackClicks,
              label: label || undefined,
              source,
              force: opts.force || undefined,
            },
          })
          const d = data!
          baseRev.current = d.draftRevision
          savedRevision.current = rev0
          pendingSource.current = null
          notes.current = []
          try {
            window.localStorage.removeItem(BACKUP_KEY(tpl.id))
          } catch {
            /* storage unavailable */
          }
          setTpl((x) => ({ ...x, draftRevision: d.draftRevision, draftHash: d.draftHash, draftChanges: d.draftChanges, draftSavedAt: d.savedAt }))
          setStatus(revision.current === rev0 ? { kind: 'saved', at: d.savedAt, version: d.draftRevision } : { kind: 'dirty', at: d.savedAt, version: d.draftRevision })
          if (opts.manual && !opts.quiet) setSavedCard({ at: d.savedAt })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/transactional/templates'] })
          return true
        } catch (e) {
          if (e instanceof ApiError && e.status === 409 && e.code === 'version_conflict') {
            const x = (e.problem.meta ?? {}) as Record<string, unknown>
            setStatus({
              kind: 'conflict',
              version: baseRev.current,
              theirs: {
                version: Number(x.draftRevision ?? 0),
                by: String(x.savedBy ?? ''),
                at: String(x.savedAt ?? ''),
                theirChanges: (x.theirChanges as string[]) ?? [],
                yourChanges: (x.yourChanges as string[]) ?? [],
              },
            })
            return false
          }
          if (offlineError(e)) {
            try {
              window.localStorage.setItem(BACKUP_KEY(tpl.id), JSON.stringify({ at: new Date().toISOString(), baseRevision: baseRev.current, subject, preheader, mjml, variables, testData }))
            } catch {
              /* storage unavailable */
            }
            setStatus((s) => ({ kind: 'offline', since: s.kind === 'offline' ? s.since : new Date().toISOString(), version: baseRev.current, message: '' }))
            return false
          }
          setStatus({ kind: 'error', version: baseRev.current, message: errorMessage(e) })
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
    [canEdit, currentMjml, compileCurrent, htmlMode, slug, tpl.id, subject, preheader, editorMode, html, plainText, variables, testData, meta, qc],
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
    if (pendingSource.current || tracker.current?.structural() || notes.current.length) {
      const h = window.setTimeout(() => void saveRef.current({ background: true }), pendingSource.current ? 300 : 2000)
      return () => window.clearTimeout(h)
    }
  }, [rev, canEdit])
  useEffect(() => {
    if (status.kind !== 'offline') return
    const h = window.setInterval(() => void saveRef.current({ background: true }), 10_000)
    return () => window.clearInterval(h)
  }, [status.kind])

  const applyTemplate = useCallback(
    (d: Template, how: 'reload' | 'restore') => {
      setTpl(d)
      setName(d.name)
      setSubjectState(d.subject)
      setPreheaderState(d.preheader)
      setVariablesState(d.variables)
      setTestDataState(d.testData)
      setPreviewWith((p) => (d.testData.some((s) => s.name === p) ? p : (d.testData[0]?.name ?? '')))
      setMetaState({ senderId: d.sender?.id ?? '', replyTo: d.replyTo, trackOpens: d.trackOpens, trackClicks: d.trackClicks })
      setPlainTextState(d.plainText)
      setHtmlState(d.html)
      const onlyHtml = d.editorMode === 'html' && !d.mjml.trim()
      setHtmlMode(onlyHtml)
      if (!onlyHtml) loadContent(d.mjml, { source: how === 'restore' ? 'restore' : 'reload' })
      baseRev.current = d.draftRevision
      tracker.current?.take()
      notes.current = []
      if (how === 'reload') {
        pendingSource.current = null
        savedRevision.current = revision.current
        setStatus({ kind: 'saved', at: d.draftSavedAt, version: d.draftRevision })
        // The canvas reload fires change events: they aren't edits.
        window.setTimeout(() => {
          savedRevision.current = revision.current
          setStatus({ kind: 'saved', at: d.draftSavedAt, version: d.draftRevision })
        }, 50)
      }
    },
    [loadContent],
  )

  /** Takes what the server says about the template (live version, ID, access) without touching the content. */
  const mergeTemplate = useCallback((d: Template) => {
    setTpl((x) => ({
      ...x,
      key: d.key,
      keyLocked: d.keyLocked,
      name: d.name,
      description: d.description,
      category: d.category,
      liveVersion: d.liveVersion,
      liveSince: d.liveSince,
      liveBy: d.liveBy,
      versionsCount: d.versionsCount,
      draftChanges: d.draftChanges,
      draftHash: d.draftHash,
      archived: d.archived,
      archivedAt: d.archivedAt,
      access: d.access,
    }))
  }, [])

  /** After a publish: making variables optional also saved the draft (a new revision with the defaults). */
  const afterPublish = useCallback(
    (d: Template) => {
      mergeTemplate(d)
      if (d.draftRevision !== baseRev.current) {
        baseRev.current = d.draftRevision
        setVariablesState(d.variables)
        setTpl((x) => ({ ...x, draftRevision: d.draftRevision, draftSavedAt: d.draftSavedAt }))
        savedRevision.current = revision.current
        setStatus({ kind: 'saved', at: d.draftSavedAt, version: d.draftRevision })
      }
    },
    [mergeTemplate],
  )

  const reloadFromServer = useCallback(async () => {
    const { data } = await http.GET('/api/v1/workspaces/{workspace}/transactional/templates/{template}', { params: { path: { workspace: slug, template: tpl.id } } })
    if (data) applyTemplate(data, 'reload')
  }, [slug, tpl.id, applyTemplate])

  const setSubject = useCallback(
    (v: string) => {
      setSubjectState(v)
      note(t(m.noteSubject))
      markDirty()
    },
    [markDirty, note, t],
  )
  const setPreheader = useCallback(
    (v: string) => {
      setPreheaderState(v)
      note(t(m.notePreheader))
      markDirty()
    },
    [markDirty, note, t],
  )
  const setVariables = useCallback(
    (v: Variable[], label?: string) => {
      setVariablesState(v)
      note(label ?? t(m.noteVariables))
      markDirty()
    },
    [markDirty, note, t],
  )
  const setTestData = useCallback(
    (v: TestData[], label?: string) => {
      setTestDataState(v)
      setPreviewWith((p) => (p === '' || v.some((s) => s.name === p) ? p : (v[0]?.name ?? '')))
      note(label ?? t(m.noteTestData))
      markDirty()
    },
    [markDirty, note, t],
  )
  const setMeta = useCallback(
    (patch: Partial<DraftMeta>) => {
      setMetaState((x) => ({ ...x, ...patch }))
      note(t(m.noteSettings))
      markDirty()
    },
    [markDirty, note, t],
  )
  const setPlainText = useCallback(
    (v: string) => {
      setPlainTextState(v)
      note(t(m.notePlainText))
      markDirty()
    },
    [markDirty, note, t],
  )
  const setHtml = useCallback(
    (v: string) => {
      setHtmlState(v)
      note(t(m.noteHtml))
      markDirty()
    },
    [markDirty, note, t],
  )

  const updateTemplate = useCallback(
    async (patch: TemplateMetaPatch): Promise<boolean> => {
      try {
        const { data } = await http.PATCH('/api/v1/workspaces/{workspace}/transactional/templates/{template}', { params: { path: { workspace: slug, template: tpl.id } }, body: patch })
        if (data) {
          setTpl((x) => ({ ...x, name: data.name, key: data.key, keyLocked: data.keyLocked, description: data.description, category: data.category, updatedAt: data.updatedAt }))
          if (patch.categoryId !== undefined) setChecksNonce((n) => n + 1)
        }
        void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/transactional/templates'] })
        return true
      } catch (e) {
        toast({ kind: 'error', title: t(m.metaFailed), detail: errorMessage(e) })
        return false
      }
    },
    [slug, tpl.id, qc, t],
  )

  const commitName = useCallback(() => {
    const n = name.trim()
    if (!canEdit || !n || n === tpl.name) {
      if (!n) setName(tpl.name)
      return
    }
    void updateTemplate({ name: n })
  }, [name, canEdit, tpl.name, updateTemplate])

  const previewData: Data = useMemo(() => dataFor(testData, previewWith, variables), [testData, previewWith, variables])

  // Checks run on the editor's current content, 0.9 s after the last change.
  const seq = useRef(0)
  useEffect(() => {
    const h = window.setTimeout(
      () => {
        const compiled = htmlMode ? null : compileCurrent()
        const ready = htmlMode || !!geRef.current || empty
        const n = ++seq.current
        setChecksPending(true)
        void http
          .POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/checks', {
            params: { path: { workspace: slug, template: tpl.id } },
            body: {
              subject,
              preheader,
              editorMode,
              ...(ready ? { mjml: currentMjml(), html: htmlMode ? html : (compiled?.html ?? '') } : {}),
              plainText,
              variables,
              testData,
              trackOpens: meta.trackOpens,
              trackClicks: meta.trackClicks,
            },
          })
          .then(({ data }) => {
            if (n === seq.current && data) setChecks(data)
          })
          .catch(() => undefined)
          .finally(() => {
            if (n === seq.current) setChecksPending(false)
          })
      },
      geRef.current ? 900 : 300,
    )
    return () => window.clearTimeout(h)
    // rev covers canvas edits; ge runs the first check once the canvas is ready.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rev, ge, subject, preheader, variables, testData, plainText, html, meta.trackOpens, meta.trackClicks, empty, htmlMode, slug, tpl.id, checksNonce, tpl.category?.id])

  // The inbox line, the text view and an HTML template's preview render with the chosen data.
  useEffect(() => {
    const h = window.setTimeout(() => {
      void http
        .POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/preview', {
          params: { path: { workspace: slug, template: tpl.id } },
          body: { subject, preheader, variables, data: previewData, plainText, ...(htmlMode ? { html } : {}) },
        })
        .then(({ data }) => data && setPreview(data))
        .catch(() => undefined)
    }, 450)
    return () => window.clearTimeout(h)
  }, [slug, tpl.id, subject, preheader, variables, previewData, plainText, html, htmlMode, meta.senderId])

  // Variables on the canvas, filled with the preview data.
  const tokenCtx = useMemo(() => ({ declared: new Map(variables.map((v) => [v.name, v])), data: previewData }), [variables, previewData])
  const tokenTitle = useCallback(
    (v: TokenView, src: string) => {
      if (v.tone === 'e' && v.name && !variables.some((x) => x.name === v.name)) return t(m.tokenUndeclared, { name: v.name, src })
      if (v.tone === 'e') return t(m.tokenMissing, { name: v.name, src })
      if (v.tone === 's') return t(m.tokenSensitive, { name: v.name, src })
      return src
    },
    [variables, t],
  )
  useCanvasTokens(view === 'visual' ? (ge?.editor ?? null) : null, tokenCtx, tokenTitle)

  const renameVariable = useCallback(
    (from: string, to: string) => {
      if (from === to) return
      setVariablesState((vs) => vs.map((v) => (v.name === from ? { ...v, name: to } : v)))
      setTestDataState((sets) => renameInData(sets, from, to))
      setSubjectState((s) => renameInSource(s, from, to))
      setPreheaderState((s) => renameInSource(s, from, to))
      setPlainTextState((s) => renameInSource(s, from, to))
      if (htmlMode) setHtmlState((s) => renameInSource(s, from, to))
      else if (geRef.current) {
        const src = currentMjml()
        const next = renameInSource(src, from, to)
        if (next !== src) loadContent(next, { source: 'code', label: t(m.noteRenamed, { from, to }) })
      }
      note(t(m.noteRenamed, { from, to }))
      markDirty()
    },
    [htmlMode, currentMjml, loadContent, note, markDirty, t],
  )
  const removeUsages = useCallback(
    (nameToRemove: string) => {
      setSubjectState((s) => removeOutputs(s, nameToRemove))
      setPreheaderState((s) => removeOutputs(s, nameToRemove))
      setPlainTextState((s) => removeOutputs(s, nameToRemove))
      if (htmlMode) setHtmlState((s) => removeOutputs(s, nameToRemove))
      else if (geRef.current) {
        const src = currentMjml()
        const next = removeOutputs(src, nameToRemove)
        if (next !== src) loadContent(next, { source: 'code', label: t(m.noteRemovedUses, { name: nameToRemove }) })
      }
      note(t(m.noteRemovedUses, { name: nameToRemove }))
      markDirty()
    },
    [htmlMode, currentMjml, loadContent, note, markDirty, t],
  )

  const convertToMjml = useCallback(() => {
    const body = `<mjml><mj-body><mj-section><mj-column><mj-raw>${html}</mj-raw></mj-column></mj-section></mj-body></mjml>`
    setHtmlMode(false)
    setHtmlState('')
    loadContent(body, { source: 'code', label: t(m.noteConverted) })
    onSearch({ ...search, view: undefined })
  }, [html, loadContent, onSearch, search, t])

  const registerLeaveCheck = useCallback((fn: () => boolean) => {
    leaveCheck.current = fn
    return () => {
      if (leaveCheck.current === fn) leaveCheck.current = null
    }
  }, [])
  const setView = useCallback(
    (v: TxView) => {
      if (v === view) return
      if ((view === 'mjml' || view === 'html') && !htmlMode && leaveCheck.current && !leaveCheck.current()) {
        toast({ kind: 'error', title: t(m.fixMjmlFirst), detail: t(m.fixMjmlFirstDetail) })
        return
      }
      if (v !== 'visual' && revision.current !== savedRevision.current) void save()
      onSearch({ ...search, view: v === (htmlMode ? 'html' : 'visual') ? undefined : v })
    },
    [view, htmlMode, onSearch, search, save, t],
  )
  const setTab = useCallback(
    (x: TxTab) => {
      setTabState(x)
      onSearch({ ...search, tab: x === 'variables' ? undefined : x })
    },
    [onSearch, search],
  )

  useEffect(() => {
    if (view === 'visual' && ge) ge.editor.refresh()
  }, [view, ge])

  const openPublish = useCallback(() => setPublishOpen(true), [])
  // The approval notification opens the editor with ?publish=1 (the dialog opens with it); the link is used once.
  useEffect(() => {
    if (search.publish) onSearch({ ...search, publish: undefined })
  }, [search, onSearch])

  const views = (htmlMode ? (['html', 'text'] as const) : (['visual', 'mjml', 'html', 'text'] as const)).map((v) => ({
    key: v,
    label: v === 'visual' ? t(m.viewVisual) : v === 'mjml' ? t(m.viewMjml) : v === 'html' ? t(m.viewHtml) : t(m.viewText),
    on: view === v,
  }))

  // The composer's context: the workspace's (when the member can read it) with the template's variables as tokens.
  const composerContext: ComposerCtxData = useMemo(() => {
    const tokens: ComposerCtxData['tokens'] = [
      ...variables.map((v) => ({ key: v.name, kind: 'custom' as const, label: v.description || v.name })),
      { key: 'to.email', kind: 'contact' as const, label: t(m.tokenToEmail) },
      { key: 'to.name', kind: 'contact' as const, label: t(m.tokenToName) },
      ...(tpl.category?.kind === 'optional'
        ? [
            { key: 'unsubscribe_url', kind: 'system' as const, label: t(m.tokenUnsubscribe) },
            { key: 'preferences_url', kind: 'system' as const, label: t(m.tokenPreferences) },
          ]
        : []),
    ]
    return {
      broadcastRoute: { shares: [], source: 'none' },
      lists: [],
      members: [],
      postalAddress: context?.postalAddress ?? '',
      publicUrl: context?.publicUrl ?? '',
      segments: [],
      senderDomains: context?.senderDomains ?? [],
      senders: [],
      tokens,
    }
  }, [variables, tpl.category?.kind, context, t])

  const kind = tpl.category?.kind ?? 'essential'
  const product = settings?.productName || workspaceName
  const blocks = useMemo(() => transactionalBlocks(context?.postalAddress ?? '', product, kind), [context?.postalAddress, product, kind])

  const campaign = useMemo(() => campaignShim(tpl, canEditTemplates), [tpl, canEditTemplates])
  const composerView: View = view === 'text' ? 'preview' : view

  const state: ComposerState = {
    kind: 'transactional',
    views,
    selectView: (k: string) => setView(k as TxView),
    slug,
    campaign,
    context: composerContext,
    ge,
    editor: ge?.editor ?? null,
    selected,
    // The text view is neither the canvas nor code: the shared parts treat it like the campaign's preview.
    view: composerView,
    device,
    variant: 'A',
    setView: (v: View) => setView(v === 'preview' ? 'text' : v),
    setDevice: (d: DeviceId) => onSearch({ ...search, device: d === 'desktop' ? undefined : d }),
    setVariant: () => undefined,
    name,
    subject,
    preheader,
    abTest: null,
    setName,
    commitName,
    setSubject,
    setPreheader,
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
    onSettingsChanged: () => setChecksNonce((n) => n + 1),
    registerLeaveCheck,
    setCodeDirty,
    isEmpty: empty && !htmlMode,
    // The campaign checks drive the composer's own overlays; the transactional checks have their own panel.
    checks: undefined,
    checksPending,
    rightTab: designTab,
    setRightTab: (x) => {
      setTab('design')
      setDesignTab(x === 'settings' ? 'settings' : 'style')
    },
    leftTab,
    setLeftTab,
    openDialog: (d: ComposerDialog) => {
      if (d === 'versions') setVersions('list')
      else if (d === 'compare') setVersions('conflict')
      else setDialog(d)
    },
    rev,
    canEdit,
    darkCanvas,
    setDarkCanvas,
    assetTarget,
    setAssetTarget,
    previewSlot: null,
    setPreviewSlot: () => undefined,
  }

  const tx: TxState = {
    template: tpl,
    settings,
    categories,
    address: context?.postalAddress ?? '',
    product,
    kind,
    variables,
    setVariables,
    testData,
    setTestData,
    previewWith,
    setPreviewWith,
    previewData,
    meta,
    setMeta,
    plainText,
    setPlainText,
    htmlMode,
    html,
    setHtml,
    convertToMjml,
    checks,
    checksPending,
    preview,
    tab,
    setTab,
    view,
    setView,
    focusVariable,
    setFocusVariable,
    designTab,
    setDesignTab,
    updateTemplate,
    mergeTemplate,
    afterPublish,
    openPublish,
    openVersions: () => setVersions('list'),
    canEdit,
    canPublish,
    renameVariable,
    removeUsages,
  }

  useShortcutHandler(
    'save',
    canEdit
      ? () => {
          void save({ manual: true })
        }
      : undefined,
  )
  useShortcutHandler('readiness', canPublish ? openPublish : undefined)

  // ⌘S, ⌘Enter and ⌘/ inside the canvas frame (its keys don't reach the page).
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
    const onPageKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === '/') {
        e.preventDefault()
        setDialog('shortcuts')
      }
    }
    window.addEventListener('keydown', onPageKey)
    const frameDoc = ge?.editor.Canvas.getDocument()
    frameDoc?.addEventListener('keydown', onFrameKey)
    return () => {
      window.removeEventListener('keydown', onPageKey)
      frameDoc?.removeEventListener('keydown', onFrameKey)
    }
  }, [ge])

  const unsaved = status.kind === 'dirty' || status.kind === 'offline' || status.kind === 'error' || status.kind === 'conflict'
  useUnsavedWork(canEdit ? { whileEnding: t(m.whileEnding), afterEnded: t(m.afterEnded, { name: name || t(m.untitled) }) } : undefined)
  const lastSave = status.kind === 'saved' || status.kind === 'dirty' ? status.at : undefined
  // eslint-disable-next-line react-hooks/refs -- re-read on every edit: each change bumps `rev`
  const changed = [...(tracker.current?.peek() ?? []), ...notes.current].slice(0, 3)
  const leaveText = lastSave
    ? changed.length
      ? t(m.leaveChangedAt, { what: formatList(changed), time: clock(lastSave) })
      : t(m.leaveAt, { time: clock(lastSave) })
    : changed.length
      ? t(m.leaveChanged, { what: formatList(changed) })
      : t(m.leaveUnsaved)

  const showRight = !((view === 'mjml' || view === 'html') && !htmlMode)
  return (
    <ComposerCtx.Provider value={state}>
      <TxCtx.Provider value={tx}>
        <div className="cmp rc-app txe">
          <a href="#main" className="skip">
            {t(m.skip)}
          </a>
          <h1 className="vh">{t(m.heading, { name: name || t(m.untitled) })}</h1>
          <TxHeader />
          {!canEdit ? <ReadOnlyBanner /> : null}
          <div className="flex flex-1 flex-col">
            {view === 'visual' || view === 'text' || htmlMode ? <TxSubjectBar /> : null}
            <div className="cmp-body">
              {view === 'visual' && !htmlMode ? <LeftPanel blocks={blocks} note={<p className="txe-bnote">{t(m.blocksNote)}</p>} /> : null}
              {!htmlMode ? (
                <div className={`cmp-main${view !== 'visual' ? ' cmp-offstage' : ''}`} inert={view !== 'visual' || undefined} aria-hidden={view !== 'visual' || undefined}>
                  <CanvasArea
                    containerRef={containerRef}
                    savedCard={null}
                    onCloseSaved={() => setSavedCard(null)}
                    onReview={openPublish}
                    slots={{ toolbar: (zoom) => <TxToolbar zoom={zoom} />, cards: <TxStateCards saved={savedCard} onCloseSaved={() => setSavedCard(null)} onCompare={() => setVersions('conflict')} />, below: null }}
                  />
                </div>
              ) : null}
              {(view === 'mjml' || view === 'html') && !htmlMode ? (
                <Suspense
                  fallback={
                    <div className="cmp-main" style={{ alignItems: 'center', justifyContent: 'center' }} aria-busy="true">
                      <span className="spin" aria-hidden="true" />
                    </div>
                  }
                >
                  <CodeView aside={<TxRightPanel />} />
                </Suspense>
              ) : null}
              {htmlMode && view !== 'text' ? <HtmlView /> : null}
              {view === 'text' ? <TextView /> : null}
              {showRight ? <TxRightPanel /> : null}
            </div>
            <TxFooter />
          </div>
          <Dialogs dialog={dialog} onClose={() => setDialog(null)} onSavedTemplate={(x) => setSavedCard({ at: new Date().toISOString(), template: x })} />
          <PublishDialog open={publishOpen} onClose={() => setPublishOpen(false)} />
          <VersionsDialog target={versions} onTarget={setVersions} />
          <UnsavedChangesGuard
            dirty={canEdit && (unsaved || codeDirty)}
            title={t(m.leaveTitle)}
            description={leaveText}
            saveLabel={t(m.leaveSave)}
            onSave={async () => {
              if (!(await save({ manual: true, quiet: true }))) throw new Error(t(m.leaveSaveFailed))
            }}
          />
        </div>
      </TxCtx.Provider>
    </ComposerCtx.Provider>
  )
}
