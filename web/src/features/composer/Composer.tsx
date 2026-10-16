import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import { lazy, Suspense, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { andList, clock } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useUnsavedWork } from '@/lib/sessionClock'
import { runShortcutHandler, useShortcutHandler } from '@/lib/shortcuts'
import { useCan, useWorkspace } from '@/lib/session'
import { CanvasArea } from './CanvasArea'
import { ComposerCtx, type ABTest, type Campaign, type Checks, type ComposerCtxData, type ComposerDialog, type ComposerState, type SaveStatus, type View } from './context'
import { Dialogs } from './Dialogs'
import { trackChanges, type ChangeTracker } from './editor/changes'
import { createComposerEditor, type ComposerEditor, type DeviceId } from './editor/grapes'
import { atPath } from './editor/labels'
import { canvasHead, isEmptyBody, joinMjml, splitMjml, withHeadMeta } from './editor/mjmlDoc'
import { Footer, VersionLink } from './Footer'
import { Header, PermissionBanner } from './Header'
import { LeftPanel } from './LeftPanel'
import { PreviewView } from './PreviewView'
import { RightPanel } from './RightPanel'
import { SubjectBar } from './SubjectBar'
import { ComposerCompare, ComposerCompareFooter, ComposerMissingLanguage, versionLanguage } from '@/features/languages/composerParts'
import { useLanguagesCache } from '@/features/languages/composer'
import { useSourceEdits } from '@/features/languages/sourceEdits'
import './composer.css'

// CodeMirror loads with the code view only.
const CodeView = lazy(() => import('./CodeView').then((m) => ({ default: m.CodeView })))

export interface ComposerSearch {
  view?: View | 'compare'
  /** A language version (internal/languages); none is the source. */
  language?: string
  device?: DeviceId
  variant?: string
  /** Component to select once the canvas is ready (element path from mj-body, "1-0-0"). */
  select?: string
  panel?: 'style' | 'settings' | 'deliverability'
  left?: 'blocks' | 'layers' | 'templates'
}

const BACKUP_KEY = (id: string) => `rc-composer-backup:${id}`

function offlineError(e: unknown) {
  return e instanceof TypeError || (e instanceof ApiError && (e.status === 502 || e.status === 503 || e.status === 504))
}

type PendingSource = { source: 'template' | 'import' | 'code' | 'restore'; label?: string }

/**
 * The campaign composer: GrapesJS + MJML on a canvas with ReputeCast's own UI
 * around it, autosave with conflict detection, inline deliverability checks,
 * the code view and client previews.
 */
export function Composer({
  campaign,
  context,
  search,
  onSearch,
  missingLanguage,
}: {
  campaign: Campaign
  context: ComposerCtxData
  search: ComposerSearch
  onSearch: (s: ComposerSearch) => void
  /** A language without a version was picked: the canvas gives way to "No … version". */
  missingLanguage?: string
}) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const canEditPerm = useCan('campaigns:edit')
  const canEdit = canEditPerm && campaign.access.editable && !missingLanguage
  const view: View = search.view === 'compare' ? 'visual' : (search.view ?? 'visual')
  const comparing = search.view === 'compare'
  const language = versionLanguage(campaign)
  const afterSourceSave = useSourceEdits(slug, campaign.id, useLanguagesCache(campaign.id).set)
  const savedMjml = useRef(campaign.mjml)
  const device: DeviceId = search.device ?? 'desktop'
  const variant = search.variant ?? 'A'

  const initial = useMemo(() => splitMjml(campaign.mjml), [campaign.mjml])
  const [head, setHeadState] = useState(initial.head)
  const headRef = useRef(initial.head)
  const setHead = useCallback((h: string) => {
    headRef.current = h
    setHeadState(h)
  }, [])
  const [name, setName] = useState(campaign.name)
  const [subject, setSubjectState] = useState(campaign.subject)
  const [preheader, setPreheaderState] = useState(campaign.preheader)
  const [abTest, setABTestState] = useState<ABTest | null>(campaign.abTest ?? null)
  const abRef = useRef(abTest)
  const [empty, setEmpty] = useState(() => isEmptyBody(campaign.mjml))
  const bodyRef = useRef(initial.body)

  const [ge, setGe] = useState<ComposerEditor | null>(null)
  const [selected, setSelected] = useState<Component | null>(null)
  const [rev, setRev] = useState(0)
  const [status, setStatus] = useState<SaveStatus>(() =>
    campaign.autosavedAt ? { kind: 'saved', at: campaign.autosavedAt, version: campaign.version } : { kind: 'new' },
  )
  const [checks, setChecks] = useState<Checks>()
  const [checksPending, setChecksPending] = useState(false)
  const [checksNonce, setChecksNonce] = useState(0)
  const [rightTab, setRightTab] = useState<'style' | 'settings' | 'deliverability'>(search.panel ?? 'style')
  const [leftTab, setLeftTab] = useState<'blocks' | 'layers' | 'templates'>(search.left ?? 'blocks')
  // A deep link selects its component once the canvas is ready.
  const deepLink = useRef(search.select)
  const [dialog, setDialog] = useState<ComposerDialog>(null)
  const [savedCard, setSavedCard] = useState<{ version: number; at: string; template?: string } | null>(null)
  const [darkCanvas, setDarkCanvas] = useState(false)
  const [assetTarget, setAssetTarget] = useState<ComposerState['assetTarget']>(null)
  const [previewSlot, setPreviewSlot] = useState<HTMLElement | null>(null)
  const [codeDirty, setCodeDirty] = useState(false)

  const version = useRef(campaign.version)
  const revision = useRef(0)
  const savedRevision = useRef(0)
  const tracker = useRef<ChangeTracker | null>(null)
  const saving = useRef<Promise<boolean> | null>(null)
  const pendingSource = useRef<PendingSource | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const geRef = useRef<ComposerEditor | null>(null)
  const leaveCheck = useRef<(() => boolean) | null>(null)
  /** Content A/B tests: the canvas shows one variant; the others' bodies wait here. */
  const variantBodies = useRef<Record<string, string>>({})
  const shownVariant = useRef('A')

  useDocumentTitle(`${name || 'Untitled campaign'} · Composer · ReputeCast`)

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
    // Each editor gets its own mount: a cancelled one (React re-running the
    // effect) is destroyed later and must not take the next one with it.
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
        // Dev builds expose the live editor to the visual QA scripts.
        if (import.meta.env.DEV) (window as unknown as { __rcComposer?: typeof ed }).__rcComposer = ed
        tracker.current = trackChanges(ed)
        ed.on('update', markDirty)
        // Typing in a text only reports input; it's an edit all the same.
        ed.on('component:input', markDirty)
        ed.on('component:toggled', () => setSelected(ed.getSelected() ?? null))
        ed.on('component:remove', () => setSelected(ed.getSelected() ?? null))
        ed.on('undo redo', () => setRev((r) => r + 1))
        if (!canEdit) {
          // Read-only: no dragging, no text editing.
          ed.getWrapper()?.onAll((c: Component) => c.set({ draggable: false, droppable: false, editable: false, removable: false, copyable: false }))
        }
        setGe(inst)
        const link = deepLink.current
        if (link) {
          deepLink.current = undefined
          const target = atPath(ed.getWrapper()?.findType('mj-body')[0], link.split('-').map(Number))
          if (target) {
            ed.select(target)
            window.setTimeout(() => target.getEl()?.scrollIntoView({ block: 'center' }), 200)
          }
        }
        if (pendingSource.current) markDirty()
      })
      .catch((e: unknown) => {
        toast({ kind: 'error', title: "The editor couldn't start", detail: e instanceof Error ? e.message : String(e) })
      })
    return () => {
      cancelled = true
      tracker.current?.dispose()
      tracker.current = null
      if (geRef.current === created) geRef.current = null
      created?.editor.destroy()
      if (created) mount.remove()
      else mount.style.display = 'none' // removed once its editor has loaded and been destroyed
      setGe(null)
      setSelected(null)
    }
    // The editor is created once per content load; device changes are applied below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editorWanted])

  useEffect(() => {
    if (ge && ge.editor.getDevice() !== device) ge.editor.setDevice(device)
  }, [ge, device])

  // Re-render the canvas when the head changes (mj-attributes, styles).
  useEffect(() => {
    if (!ge) return
    const body = ge.editor.getWrapper()
    for (const t of ['mj-section', 'mj-hero', 'mj-wrapper', 'mj-raw']) body?.findType(t).forEach((c: Component) => (c.view as unknown as { rerender?: () => void })?.rerender?.())
  }, [ge, head])

  const canvasBody = useCallback(() => (geRef.current ? geRef.current.getBody() : bodyRef.current), [])

  /** Body of variant A: the canvas, unless it shows another variant of a content test. */
  const bodyA = useCallback(() => (shownVariant.current === 'A' ? canvasBody() : (variantBodies.current.A ?? canvasBody())), [canvasBody])

  const currentMjml = useCallback(() => {
    if (empty && !geRef.current) return ''
    return joinMjml(withHeadMeta(headRef.current, { title: subject, preview: preheader }), bodyA())
  }, [empty, subject, preheader, bodyA])

  const compileCurrent = useCallback(() => {
    const inst = geRef.current
    const src = currentMjml()
    if (!inst || !src) return null
    return inst.compile(src)
  }, [currentMjml])

  /** The A/B test to save: content variants carry their MJML and compiled HTML. */
  const abForSave = useCallback((): ABTest | undefined => {
    const t = abRef.current
    if (!t) return undefined
    if (t.kind !== 'content' || !geRef.current) return t
    const inst = geRef.current
    const h = withHeadMeta(headRef.current, { title: subject, preview: preheader })
    return {
      ...t,
      variants: t.variants.map((v) => {
        if (v.key === 'A') return { key: 'A' }
        const body = shownVariant.current === v.key ? canvasBody() : (variantBodies.current[v.key] ?? (v.mjml ? splitMjml(v.mjml).body : bodyA()))
        const mjml = joinMjml(h, body)
        return { ...v, mjml, html: inst.compile(mjml).html }
      }),
    }
  }, [subject, preheader, canvasBody, bodyA])

  const loadContent = useCallback(
    (mjml: string, how: { source: 'template' | 'import' | 'code' | 'restore' | 'reload'; label?: string }) => {
      const parts = splitMjml(mjml)
      setHead(parts.head)
      bodyRef.current = parts.body
      variantBodies.current = {}
      if (shownVariant.current !== 'A') {
        shownVariant.current = 'A'
        onSearch({ ...search, variant: undefined })
      }
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
      }
    },
    [markDirty, setHead, onSearch, search],
  )

  const save = useCallback(
    async (opts: { manual?: boolean; quiet?: boolean; force?: boolean; background?: boolean; source?: 'autosave' | 'manual' | 'import' | 'template'; label?: string } = {}): Promise<boolean> => {
      if (!canEdit) return false
      // Text being typed reaches the model when its editing ends: timed autosaves
      // wait for that; anything the person asked for ends the editing first.
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
        const compiled = compileCurrent()
        const pending = pendingSource.current
        const label = opts.label ?? pending?.label ?? tracker.current?.take() ?? ''
        const source = opts.source ?? (pending ? (pending.source === 'restore' || pending.source === 'code' ? 'manual' : pending.source) : opts.manual ? 'manual' : 'autosave')
        setStatus({ kind: 'saving', version: version.current })
        try {
          const { data } = await http.PUT('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/content', {
            params: { path: { workspace: slug, campaignId: campaign.id }, query: language ? { language } : {} },
            body: { baseVersion: version.current, subject, preheader, mjml, html: compiled?.html ?? '', abTest: abForSave(), label, source, force: opts.force || undefined },
          })
          version.current = data!.version
          savedRevision.current = rev0
          pendingSource.current = null
          try {
            window.localStorage.removeItem(BACKUP_KEY(campaign.id))
          } catch {
            /* storage unavailable */
          }
          setStatus(revision.current === rev0 ? { kind: 'saved', at: data!.savedAt, version: data!.version } : { kind: 'dirty', at: data!.savedAt, version: data!.version })
          if (opts.manual && !opts.quiet) setSavedCard({ version: data!.version, at: data!.savedAt })
          // The source's versions: offer to keep them in step (LanguageStates).
          if (!language) void afterSourceSave(savedMjml.current, mjml, data!.version)
          savedMjml.current = mjml
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions'] })
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
              window.localStorage.setItem(BACKUP_KEY(campaign.id), JSON.stringify({ at: new Date().toISOString(), baseVersion: version.current, subject, preheader, mjml }))
            } catch {
              /* storage unavailable */
            }
            setStatus((s) => ({ kind: 'offline', since: s.kind === 'offline' ? s.since : new Date().toISOString(), version: version.current, message: "you're offline" }))
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
    [canEdit, currentMjml, compileCurrent, abForSave, slug, campaign.id, subject, preheader, qc, language, afterSourceSave],
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
    const { data } = await http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}', {
      params: { path: { workspace: slug, campaignId: campaign.id }, query: language ? { language } : {} },
    })
    if (!data) return
    qc.setQueryData(['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: campaign.id }, query: language ? { language } : undefined } }], data)
    setName(data.name)
    setSubjectState(data.subject)
    setPreheaderState(data.preheader)
    setABTestState(data.abTest ?? null)
    abRef.current = data.abTest ?? null
    loadContent(data.mjml, { source: 'reload' })
    version.current = data.version
    tracker.current?.take()
    pendingSource.current = null
    savedRevision.current = revision.current
    setStatus(data.autosavedAt ? { kind: 'saved', at: data.autosavedAt, version: data.version } : { kind: 'saved', at: data.updatedAt, version: data.version })
    window.setTimeout(() => {
      // The canvas reload fires change events: they aren't edits.
      savedRevision.current = revision.current
      setStatus({ kind: 'saved', at: data.autosavedAt ?? data.updatedAt, version: data.version })
    }, 50)
  }, [slug, campaign.id, qc, loadContent, language])

  const setSubject = useCallback(
    (v: string) => {
      setSubjectState(v)
      tracker.current?.note(variant === 'A' ? 'subject' : `A/B subject ${variant}`)
      markDirty()
    },
    [markDirty, variant],
  )
  const setPreheader = useCallback(
    (v: string) => {
      setPreheaderState(v)
      tracker.current?.note('preheader')
      markDirty()
    },
    [markDirty],
  )
  const setABTest = useCallback(
    (t: ABTest | null) => {
      setABTestState(t)
      abRef.current = t
      tracker.current?.note(t ? `A/B ${t.kind === 'send_time' ? 'send time' : t.kind}` : 'A/B test removed')
      markDirty()
    },
    [markDirty],
  )
  const updateMeta = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}')
  const commitName = useCallback(() => {
    const n = name.trim()
    if (!canEdit || n === campaign.name || !n) {
      if (!n) setName(campaign.name)
      return
    }
    updateMeta.mutate(
      { params: { path: { workspace: slug, campaignId: campaign.id } }, body: { name: n } },
      {
        onSuccess: (d) => {
          qc.setQueryData(['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: campaign.id } } }], d)
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
        },
        onError: (e) => toast({ kind: 'error', title: "Couldn't rename the campaign", detail: errorMessage(e) }),
      },
    )
  }, [name, canEdit, campaign.name, campaign.id, slug, updateMeta, qc])

  const seq = useRef(0)
  useEffect(() => {
    if (empty) return
    const h = window.setTimeout(
      () => {
        const compiled = compileCurrent()
        const html = compiled?.html ?? (geRef.current ? '' : campaign.html)
        const mjml = geRef.current ? currentMjml() : campaign.mjml
        const n = ++seq.current
        setChecksPending(true)
        void http
          .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/checks', {
            params: { path: { workspace: slug, campaignId: campaign.id } },
            body: { subject, preheader, mjml, html, language },
          })
          .then(({ data }) => {
            if (n === seq.current && data) setChecks(data)
          })
          .catch(() => undefined)
          .finally(() => {
            if (n === seq.current) setChecksPending(false)
          })
      },
      geRef.current ? 900 : 0,
    )
    return () => window.clearTimeout(h)
    // rev covers canvas edits; ge triggers the first run once the editor is ready.
  }, [rev, ge, subject, preheader, empty, slug, campaign.id, campaign.html, campaign.mjml, compileCurrent, currentMjml, checksNonce, language])

  const onSettingsChanged = useCallback(() => {
    setChecksNonce((n) => n + 1)
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}'] })
  }, [qc])

  // ⌘S and ⌘Enter go through the app's shortcut layer (see the handlers
  // below); the canvas is an iframe, so its keys are forwarded here. ⌘/ opens
  // the composer's own shortcut list.
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

  const registerLeaveCheck = useCallback((fn: () => boolean) => {
    leaveCheck.current = fn
    return () => {
      if (leaveCheck.current === fn) leaveCheck.current = null
    }
  }, [])
  const setView = useCallback(
    (v: View) => {
      if (v === view) return
      if ((view === 'mjml' || view === 'html') && leaveCheck.current && !leaveCheck.current()) {
        toast({ kind: 'error', title: 'Fix the MJML first', detail: 'The canvas keeps the last good version until the code compiles.' })
        return
      }
      if (v !== 'visual' && revision.current !== savedRevision.current) void save()
      onSearch({ ...search, view: v === 'visual' ? undefined : v })
    },
    [view, onSearch, search, save],
  )
  const setDevice = useCallback((d: DeviceId) => onSearch({ ...search, device: d === 'desktop' ? undefined : d }), [onSearch, search])

  /** Switches variant; a content test swaps the canvas body. */
  const setVariant = useCallback(
    (v: string) => {
      const inst = geRef.current
      const t = abRef.current
      if (t?.kind === 'content' && inst && v !== shownVariant.current) {
        variantBodies.current[shownVariant.current] = inst.getBody()
        const target = variantBodies.current[v] ?? (v === 'A' ? undefined : t.variants.find((x) => x.key === v)?.mjml)
        const body = target ? (target.trim().startsWith('<mjml') && target.includes('<mj-head') ? splitMjml(target).body : target) : inst.getBody()
        shownVariant.current = v
        inst.setBody(body)
        inst.editor.UndoManager.clear()
        inst.editor.select(undefined)
      }
      shownVariant.current = v
      onSearch({ ...search, variant: v === 'A' ? undefined : v })
    },
    [onSearch, search],
  )

  // The variant in the address must exist.
  useEffect(() => {
    if (variant !== 'A' && !abTest?.variants.some((x) => x.key === variant)) onSearch({ ...search, variant: undefined })
  }, [variant, abTest, onSearch, search])

  useEffect(() => {
    if (view === 'visual' && ge) ge.editor.refresh()
  }, [view, ge])

  const state: ComposerState = {
    slug,
    campaign,
    context,
    ge,
    editor: ge?.editor ?? null,
    selected,
    view,
    device,
    variant,
    setView,
    setDevice,
    setVariant,
    name,
    subject,
    preheader,
    abTest,
    setName,
    commitName,
    setSubject,
    setPreheader,
    setABTest,
    head,
    setHead: (h: string) => {
      setHead(h)
      markDirty()
    },
    status,
    save,
    currentMjml,
    compileCurrent,
    loadContent,
    reloadFromServer,
    onSettingsChanged,
    registerLeaveCheck,
    setCodeDirty,
    isEmpty: empty,
    checks: empty ? undefined : checks,
    checksPending,
    rightTab,
    setRightTab,
    leftTab,
    setLeftTab,
    openDialog: setDialog,
    rev,
    canEdit,
    darkCanvas,
    setDarkCanvas,
    assetTarget,
    setAssetTarget,
    previewSlot,
    setPreviewSlot,
    missingLanguage,
  }

  const goReview = useCallback(async () => {
    if (revision.current !== savedRevision.current && !(await save())) return
    // Just saved: the unsaved-changes guard has nothing to ask.
    void navigate({ to: '/w/$workspace/campaigns/$campaignId/review', params: { workspace: slug, campaignId: campaign.id }, ignoreBlocker: true })
  }, [save, navigate, slug, campaign.id])

  useShortcutHandler(
    'save',
    canEdit
      ? () => {
          void save({ manual: true })
        }
      : undefined,
  )
  useShortcutHandler('readiness', () => {
    void goReview()
  })

  const unsaved = status.kind === 'dirty' || status.kind === 'offline' || status.kind === 'error' || status.kind === 'conflict'
  const savedVersion = status.kind === 'new' ? 0 : status.version
  useUnsavedWork(
    canEdit
      ? {
          whileEnding: 'the draft autosaves',
          afterEnded: savedVersion ? `The campaign you were editing is saved as draft v${savedVersion} and will reopen where you left off.` : undefined,
        }
      : undefined,
  )
  const lastSave = status.kind === 'saved' || status.kind === 'dirty' ? status.at : undefined
  // Dialogs board: "You changed the subject line and two blocks since the last autosave …
  // Leaving now keeps draft v16; saving creates v17." (The tracker names what changed.)
  // eslint-disable-next-line react-hooks/refs -- re-read on every edit: each change bumps `rev`
  const changed = tracker.current?.peek() ?? []
  const leaveText = `You changed ${changed.length ? `the ${andList(changed)}` : 'this email'} since ${lastSave ? `the last autosave at ${clock(lastSave)}` : 'it was last saved'}. Leaving now keeps ${campaign.status === 'draft' ? 'draft' : 'version'} v${savedVersion}; saving creates v${savedVersion + 1}.`

  return (
    <ComposerCtx.Provider value={state}>
      <div className="cmp rc-app">
        <a href="#main" className="skip">
          Skip to content
        </a>
        <h1 className="vh">{view === 'preview' ? `Preview of ${name}` : `Campaign composer: ${name || 'Untitled campaign'}`}</h1>
        <Header onReview={goReview} />
        {!campaign.access.canSend && canEditPerm ? <PermissionBanner /> : null}
        {view === 'preview' ? <PreviewView onReview={goReview} /> : null}
        {/* The canvas stays mounted off screen in the other views: GrapesJS keeps its state and compiles. */}
        <div className={`flex flex-1 flex-col${view === 'preview' ? ' cmp-offstage' : ''}`} inert={view === 'preview' || undefined} aria-hidden={view === 'preview' || undefined}>
          {view === 'visual' && !comparing && !missingLanguage ? <SubjectBar /> : null}
          <div className="cmp-body">
            {view === 'visual' && !comparing && !missingLanguage ? <LeftPanel /> : null}
            {comparing ? <ComposerCompare /> : null}
            {missingLanguage && !comparing ? <ComposerMissingLanguage language={missingLanguage} /> : null}
            <div
              className={`cmp-main${view !== 'visual' || comparing || missingLanguage ? ' cmp-offstage' : ''}`}
              inert={view !== 'visual' || comparing || Boolean(missingLanguage) || undefined}
              aria-hidden={view !== 'visual' || comparing || Boolean(missingLanguage) || undefined}
            >
              <CanvasArea containerRef={containerRef} savedCard={savedCard} onCloseSaved={() => setSavedCard(null)} onReview={goReview} />
            </div>
            {view === 'mjml' || view === 'html' ? (
              <Suspense fallback={<div className="cmp-main" style={{ alignItems: 'center', justifyContent: 'center' }} aria-busy="true"><span className="spin" aria-hidden="true" /></div>}>
                <CodeView />
              </Suspense>
            ) : null}
            {view === 'visual' && !comparing && !missingLanguage ? <RightPanel /> : null}
          </div>
          {view === 'preview' || (missingLanguage && !comparing) ? null : comparing ? (
            <ComposerCompareFooter>
              <VersionLink />
            </ComposerCompareFooter>
          ) : (
            <Footer />
          )}
        </div>
        <Dialogs dialog={dialog} onClose={() => setDialog(null)} onSavedTemplate={(t) => setSavedCard({ version: version.current, at: new Date().toISOString(), template: t })} />
        <UnsavedChangesGuard
          dirty={canEdit && (unsaved || codeDirty)}
          title="Leave the composer?"
          description={leaveText}
          saveLabel={`Save as v${savedVersion + 1} and leave`}
          onSave={async () => {
            if (!(await save({ manual: true, quiet: true }))) throw new Error("The draft couldn't be saved. Check your connection, or download the MJML as a backup.")
          }}
        />
      </div>
    </ComposerCtx.Provider>
  )
}
