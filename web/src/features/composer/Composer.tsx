import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import { lazy, Suspense, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { andList, clock } from '@/lib/format'
import { mn } from './banners.messages'
import { useDocumentTitle } from '@/lib/hooks'
import { useUnsavedWork } from '@/lib/sessionClock'
import { runShortcutHandler, useShortcutHandler } from '@/lib/shortcuts'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { ComposerBanners, type Recovered } from './Banners'
import { ApprovalBanner, ApprovalDialog, type Approval } from './Approval'
import { CanvasArea } from './CanvasArea'
import { ComposerCtx, type ABTest, type Campaign, type Checks, type ComposerCtxData, type ComposerDialog, type ComposerState, type RightTab, type SaveStatus, type View } from './context'
import { Dialogs } from './Dialogs'
import { variantLines } from './ABPanel'
import { trackChanges, type ChangeTracker } from './editor/changes'
import { createComposerEditor, type ComposerEditor, type DeviceId, type EmailDevice } from './editor/grapes'
import { installCanvasKeys } from './editor/keys'
import { lockCanvas, unlockCanvas, type Locked } from './editor/lock'
import { isKnownField, knownFields } from './editor/fields'
import { setChipLabels } from './editor/liquid'
import { trackMeta, type Meta, type MetaUndo } from './editor/metaUndo'
import { atPath, sectionWord } from './editor/labels'
import { applySections } from './editor/merge'
import { mx } from './collab.messages'
import { canvasHead, isEmptyBody, joinMjml, splitMjml, withHeadMeta } from './editor/mjmlDoc'
import { Footer, VersionLink } from './Footer'
import { Header, PermissionBanner } from './Header'
import { LeftPanel } from './LeftPanel'
import { PreviewView } from './PreviewView'
import { RightPanel } from './RightPanel'
import { forgetRecovered, journalWriter, useJournalKey, useRecoveredJournal, type JournalWriter } from './journal'
import { serialQueue } from './saveQueue'
import { SubjectBar } from './SubjectBar'
import { AccessibilityProvider } from '@/features/accessibility/composer/state'
import { ConditionalProvider } from '@/features/conditional/state'
import { withoutAltText } from '@/features/accessibility/composer/Status'
import { ComposerCompare, ComposerCompareFooter, ComposerMissingLanguage, versionLanguage } from '@/features/languages/composerParts'
import { useLanguagesCache } from '@/features/languages/composer'
import { useSourceEdits } from '@/features/languages/sourceEdits'
import { DigestModeProvider, digestLeftPanel } from '@/features/digests/composer/DigestMode'
import { AssistRight } from '@/features/ai/composer/panel'
import { ComposerAssistScope, useComposerAssistState } from '@/features/ai/composer/state'
import { AITranslateFooter, AITranslateView } from '@/features/ai/composer/translate'
import { SponsorSync } from '@/features/earn/sponsors/composer/SponsorSettings'
import { WebCanvasView, WebModeProvider } from '@/features/website/composer/campaign'
import { t as tt, useT } from '@/i18n'
import { m } from './messages'
import './composer.css'
import './chrome.css'

// CodeMirror loads with the code view only.
const CodeView = lazy(() => import('./CodeView').then((m) => ({ default: m.CodeView })))

export interface ComposerSearch {
  view?: View | 'compare'
  /** A language version (internal/languages); none is the source. */
  language?: string
  device?: EmailDevice
  variant?: string
  /** Component to select once the canvas is ready (element path from mj-body, "1-0-0"). */
  select?: string
  panel?: RightTab
  left?: 'blocks' | 'layers' | 'templates'
  /** A comment thread to show with panel=comments (a mention's link, CO-10). */
  thread?: string
}

function offlineError(e: unknown) {
  return e instanceof TypeError || (e instanceof ApiError && (e.status === 502 || e.status === 503 || e.status === 504))
}

type PendingSource = { source: 'template' | 'import' | 'code' | 'restore' | 'recovered'; label?: string }

type SaveOptions = Parameters<ComposerState['save']>[0] & {}

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
  const t = useT()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const canEditPerm = useCan('campaigns:edit')
  // An approved or pending campaign opens without editing: editing cancels its
  // approval, so the person says so first (CO-7, ComposerStates · Approval).
  const [approvalGate, setApprovalGate] = useState(() => Boolean(campaign.approval) && canEditPerm && campaign.access.editable && !missingLanguage)
  const [approvalInfo, setApprovalInfo] = useState<Approval | null>(campaign.approval ?? null)
  const [askApproval, setAskApproval] = useState<'open' | 'save' | null>(null)
  /** The person agreed that saving cancels the approval: saves say so. */
  const cancelApproval = useRef(false)
  const canEdit = canEditPerm && campaign.access.editable && !missingLanguage && !approvalGate
  const view: View = search.view === 'compare' ? 'visual' : (search.view ?? 'visual')
  // The assistant (features/ai): its panel in the right column, and translating with AI in place of the canvas.
  const assist = useComposerAssistState(campaign.id)
  const comparing = search.view === 'compare' || Boolean(assist.translating)
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
  // Saves read what is typed from these refs, not from a render's closure:
  // a subject typed while a slow save is on its way belongs to the next save.
  const subjectRef = useRef(campaign.subject)
  const preheaderRef = useRef(campaign.preheader)
  const abRef = useRef(abTest)
  const [empty, setEmpty] = useState(() => isEmptyBody(campaign.mjml))
  const bodyRef = useRef(initial.body)
  /** Undo and redo of the subject, preheader and A/B test, on the editor's undo stack (CO-8). */
  const meta = useRef<MetaUndo | null>(null)
  const currentMeta = useCallback((): Meta => ({ subject: subjectRef.current, preheader: preheaderRef.current, abTest: abRef.current }), [])
  const canEditRef = useRef(canEdit)
  useLayoutEffect(() => {
    canEditRef.current = canEdit
  }, [canEdit])

  const [ge, setGe] = useState<ComposerEditor | null>(null)
  const [selected, setSelected] = useState<Component | null>(null)
  const [rev, setRev] = useState(0)
  const [status, setStatus] = useState<SaveStatus>(() =>
    campaign.autosavedAt ? { kind: 'saved', at: campaign.autosavedAt, version: campaign.version } : { kind: 'new' },
  )
  const statusRef = useRef(status)
  useLayoutEffect(() => {
    statusRef.current = status
  }, [status])
  const [checks, setChecks] = useState<Checks>()
  /** The last check run failed (its results stay, marked stale), or succeeded again after a failure. */
  const [checksRun, setChecksRun] = useState<{ state: 'ok' | 'failed' | 'recovered'; reason?: 'network' | 'server' }>({ state: 'ok' })
  const [checksPending, setChecksPending] = useState(false)
  const [checksNonce, setChecksNonce] = useState(0)
  const [rightTab, setRightTab] = useState<RightTab>(search.panel ?? 'style')
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
  const pendingSource = useRef<PendingSource | null>(null)
  /** Told the revision a save stored (the browser journal forgets what is saved). */
  const afterSaved = useRef<((rev: number) => void) | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const geRef = useRef<ComposerEditor | null>(null)
  /** What the read-only canvas took away from each component, to give back when editing is agreed to. */
  const locked = useRef<Locked | null>(null)
  const leaveCheck = useRef<(() => boolean) | null>(null)
  /** Content A/B tests: the canvas shows one variant; the others' bodies wait here. */
  const variantBodies = useRef<Record<string, string>>({})
  const shownVariant = useRef('A')
  /** Resolves once the canvas is ready (a recovered draft waits for it before saving). */
  const editorReady = useRef<{ promise: Promise<void>; resolve: () => void } | null>(null)
  const whenEditorReady = useCallback(() => {
    if (geRef.current) return Promise.resolve()
    if (!editorReady.current) {
      let resolve!: () => void
      const promise = new Promise<void>((r) => (resolve = r))
      editorReady.current = { promise, resolve }
    }
    return editorReady.current.promise
  }, [])

  // The browser journal (CO-1): every edit reaches this browser's storage within
  // two seconds, keystrokes in a text included; a save that catches up clears it.
  // Edits found there when the composer opens are offered back (Recovered edits).
  const { data: meData } = useMe()
  const jKey = useJournalKey(meData?.user.id ?? 'me', campaign.id, language)
  const journal = useRef<JournalWriter | null>(null)
  const [recoveredEntry, setRecoveredEntry] = useRecoveredJournal(jKey, campaign, canEdit)

  useDocumentTitle(t(m.documentTitle, { name: name || t(m.untitled) }))

  const markDirty = useCallback(() => {
    revision.current++
    journal.current?.touch()
    setRev((r) => r + 1)
    setStatus((s) => (s.kind === 'saving' || s.kind === 'conflict' || s.kind === 'offline' ? s : { kind: 'dirty', at: s.kind === 'saved' ? s.at : undefined, version: version.current }))
  }, [])

  /** What an undo or a redo brings back to the subject, preheader and A/B test. */
  const applyMeta = useCallback(
    (next: Meta) => {
      if (next.subject !== subjectRef.current) {
        subjectRef.current = next.subject
        setSubjectState(next.subject)
        tracker.current?.note(tt(m.noteSubject))
      }
      if (next.preheader !== preheaderRef.current) {
        preheaderRef.current = next.preheader
        setPreheaderState(next.preheader)
        tracker.current?.note(tt(m.notePreheader))
      }
      if (JSON.stringify(next.abTest) !== JSON.stringify(abRef.current)) {
        abRef.current = next.abTest
        setABTestState(next.abTest)
        tracker.current?.note(next.abTest ? tt(m.noteAb, { kind: next.abTest.kind }) : tt(m.noteAbRemoved))
      }
      markDirty()
    },
    [markDirty],
  )
  const applyMetaRef = useRef(applyMeta)
  useLayoutEffect(() => {
    applyMetaRef.current = applyMeta
  }, [applyMeta])

  const editorWanted = !empty
  useEffect(() => {
    if (!editorWanted || !containerRef.current) return
    let cancelled = false
    let created: ComposerEditor | null = null
    let stopKeys: (() => void) | undefined
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
        meta.current = trackMeta(ed, currentMeta(), (next) => applyMetaRef.current(next))
        stopKeys = installCanvasKeys(ed, () => canEditRef.current)
        ed.on('update', markDirty)
        // Typing in a text only reports input; it's an edit all the same, saved and
        // journaled as it is typed (getLiveBody), without ending the editing.
        ed.on('component:input', (comp: Component) => {
          tracker.current?.typing(comp)
          markDirty()
        })
        ed.on('component:toggled', () => setSelected(ed.getSelected() ?? null))
        ed.on('component:remove', () => setSelected(ed.getSelected() ?? null))
        ed.on('undo redo', () => setRev((r) => r + 1))
        // Read-only: no dragging, no text editing (given back if editing is agreed to later).
        if (!canEdit) locked.current = lockCanvas(ed)
        setGe(inst)
        editorReady.current?.resolve()
        editorReady.current = null
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
        toast({ kind: 'error', title: tt(m.editorFailed), detail: e instanceof Error ? e.message : String(e) })
      })
    return () => {
      cancelled = true
      tracker.current?.dispose()
      tracker.current = null
      meta.current?.dispose()
      meta.current = null
      stopKeys?.()
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

  // Liquid chips read the workspace's fields: their names, and which keys aren't fields (CO-4).
  const tokens = context.tokens
  const digestTemplate = campaign.digestRole === 'template'
  useEffect(() => {
    if (!ge) return
    const known = knownFields(tokens, digestTemplate)
    setChipLabels(ge.editor, (p) => ({ label: tokens.find((x) => x.key === p.key)?.label ?? p.key, fallback: p.fallback, unknown: !isKnownField(p.key, known) }))
  }, [ge, tokens, digestTemplate])

  // Editing agreed to: the canvas takes edits again.
  useEffect(() => {
    if (canEdit && ge && locked.current) {
      unlockCanvas(locked.current)
      locked.current = null
    }
  }, [canEdit, ge])

  // Re-render the canvas when the head changes (mj-attributes, styles).
  useEffect(() => {
    if (!ge) return
    const body = ge.editor.getWrapper()
    for (const t of ['mj-section', 'mj-hero', 'mj-wrapper', 'mj-raw']) body?.findType(t).forEach((c: Component) => (c.view as unknown as { rerender?: () => void })?.rerender?.())
  }, [ge, head])

  /** The canvas as MJML, with the words of a text being typed (they reach the model only when its editing ends). */
  const canvasBody = useCallback(() => (geRef.current ? geRef.current.getLiveBody() : bodyRef.current), [])

  /** Body of variant A: the canvas, unless it shows another variant of a content test. */
  const bodyA = useCallback(() => (shownVariant.current === 'A' ? canvasBody() : (variantBodies.current.A ?? canvasBody())), [canvasBody])

  const currentMjml = useCallback(() => {
    if (empty && !geRef.current) return ''
    return joinMjml(withHeadMeta(headRef.current, { title: subjectRef.current, preview: preheaderRef.current }), bodyA())
  }, [empty, bodyA])

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
    const h = withHeadMeta(headRef.current, { title: subjectRef.current, preview: preheaderRef.current })
    return {
      ...t,
      variants: t.variants.map((v) => {
        if (v.key === 'A') return { key: 'A' }
        const body = shownVariant.current === v.key ? canvasBody() : (variantBodies.current[v.key] ?? (v.mjml ? splitMjml(v.mjml).body : bodyA()))
        const mjml = joinMjml(h, body)
        return { ...v, mjml, html: inst.compile(mjml).html }
      }),
    }
  }, [canvasBody, bodyA])

  /** What variant `key` sends, unsaved edits included: its subject and preheader (or A's), and its compiled HTML. */
  const variantContent = useCallback(
    (key: string) => {
      const t = abRef.current
      const lines = variantLines(t, key, { subject: subjectRef.current, preheader: preheaderRef.current })
      if (key !== 'A' && t?.kind === 'content') return { ...lines, html: abForSave()?.variants.find((x) => x.key === key)?.html ?? '' }
      return { ...lines, html: compileCurrent()?.html ?? '' }
    },
    [abForSave, compileCurrent],
  )

  const loadContent = useCallback(
    (mjml: string, how: { source: 'template' | 'import' | 'code' | 'restore' | 'reload' | 'recovered'; label?: string }) => {
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

  /**
   * A save merged with someone else's (CO-10): their sections, head,
   * subject and preheader come onto the canvas, as stored. With no edits
   * since the save, the canvas is the saved version.
   */
  const applyMerged = useCallback(
    (merged: Schemas['ComposerMerged'], rev0: number, saved: { at: string; version: number }) => {
      const clean = revision.current === rev0
      if (geRef.current) applySections(geRef.current.editor, merged.sections)
      if (merged.head) setHead(merged.head)
      if (merged.subject != null) {
        subjectRef.current = merged.subject
        setSubjectState(merged.subject)
      }
      if (merged.preheader != null) {
        preheaderRef.current = merged.preheader
        setPreheaderState(merged.preheader)
      }
      if (clean) {
        // The editor reports these changes a moment later: they're saved.
        window.setTimeout(() => {
          tracker.current?.take()
          savedRevision.current = revision.current
          setStatus({ kind: 'saved', at: saved.at, version: saved.version })
        }, 0)
      }
      const parts = [
        ...merged.sections.map((x) => sectionWord(x.label)),
        ...(merged.subject != null ? [tt(mx.mergedSubject)] : []),
        ...(merged.preheader != null ? [tt(mx.mergedPreheader)] : []),
        ...(merged.head ? [tt(mx.mergedHead)] : []),
      ]
      toast({ kind: 'info', title: tt(mx.mergedTitle, { name: merged.with }), detail: tt(mx.mergedDetail, { name: merged.with, parts: andList(parts), version: merged.version }) })
    },
    [setHead],
  )

  /** Saves run one after another: each reads what is on screen when its turn comes. */
  const [enqueueSave] = useState(serialQueue)
  const save = useCallback(
    (opts: SaveOptions = {}): Promise<boolean> => {
      if (!canEdit) return Promise.resolve(false)
      const run = async (): Promise<boolean> => {
        // A save the person asked for ends the text editing first (the model gets
        // the words, the canvas compiles them); a timed autosave saves the words
        // as they are typed, without interrupting the typing.
        if (!opts.background && geRef.current?.isEditingText()) await geRef.current.flushEditing()
        const pending = pendingSource.current
        if (!opts.force && !opts.stash && !opts.manual && !pending && revision.current === savedRevision.current) return true
        const rev0 = revision.current
        const mjml = currentMjml()
        const compiled = mjml && geRef.current ? geRef.current.compile(mjml) : null
        const label = opts.label ?? pending?.label ?? (opts.stash ? '' : (tracker.current?.take() ?? ''))
        const source = opts.source ?? (pending ? (pending.source === 'restore' || pending.source === 'code' ? 'manual' : pending.source) : opts.manual ? 'manual' : 'autosave')
        setStatus({ kind: 'saving', version: version.current })
        try {
          const { data } = await http.PUT('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/content', {
            params: { path: { workspace: slug, campaignId: campaign.id }, query: language ? { language } : {} },
            body: {
              baseVersion: version.current,
              subject: subjectRef.current,
              preheader: preheaderRef.current,
              mjml,
              html: compiled?.html ?? '',
              abTest: abForSave(),
              label,
              source,
              force: opts.force || undefined,
              stash: opts.stash || undefined,
              cancelApproval: cancelApproval.current || undefined,
            },
          })
          version.current = data!.version
          savedRevision.current = rev0
          pendingSource.current = null
          // Someone saved a newer version and changed other sections: theirs
          // are in the saved version; the canvas takes them too (CO-10).
          if (data!.merged) applyMerged(data!.merged, rev0, { at: data!.savedAt, version: data!.version })
          afterSaved.current?.(rev0)
          setStatus(revision.current === rev0 ? { kind: 'saved', at: data!.savedAt, version: data!.version } : { kind: 'dirty', at: data!.savedAt, version: data!.version })
          if (opts.manual && !opts.quiet) setSavedCard({ version: data!.version, at: data!.savedAt })
          // The source's versions: offer to keep them in step (LanguageStates).
          if (!language && !opts.stash && !data!.unchanged) void afterSourceSave(savedMjml.current, mjml, data!.version)
          if (!opts.stash) savedMjml.current = data!.merged ? currentMjml() : mjml
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions'] })
          return true
        } catch (e) {
          if (e instanceof ApiError && e.status === 409 && e.code === 'approval_would_cancel') {
            // Approved since the composer opened: ask before the save cancels it.
            setStatus({ kind: 'error', version: version.current, message: e.message })
            void http
              .GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: campaign.id } } })
              .then(({ data }) => {
                if (data?.approval) setApprovalInfo(data.approval)
                setAskApproval('save')
              })
              .catch(() => setAskApproval('save'))
            return false
          }
          if (e instanceof ApiError && e.status === 409 && e.code === 'version_conflict') {
            const meta = (e.problem.meta ?? {}) as Record<string, unknown>
            setStatus({
              kind: 'conflict',
              version: version.current,
              theirs: { version: Number(meta.version ?? 0), by: String(meta.savedBy ?? ''), at: String(meta.savedAt ?? ''), theirChanges: (meta.theirChanges as string[]) ?? [], yourChanges: (meta.yourChanges as string[]) ?? [] },
            })
            return false
          }
          if (offlineError(e)) {
            setStatus((s) => ({ kind: 'offline', since: s.kind === 'offline' ? s.since : new Date().toISOString(), version: version.current, message: tt(m.offlineReason) }))
            return false
          }
          setStatus({ kind: 'error', version: version.current, message: errorMessage(e) })
          return false
        }
      }
      return enqueueSave(run)
    },
    [canEdit, currentMjml, abForSave, slug, campaign.id, qc, language, afterSourceSave, enqueueSave, applyMerged],
  )

  const saveRef = useRef(save)
  useLayoutEffect(() => {
    saveRef.current = save
  })
  // Autosave every 30 s while there are changes, the words of a text being typed
  // included, and 2 s after a structural change. A conflict waits for the person.
  useEffect(() => {
    if (!canEdit) return
    const h = window.setInterval(() => {
      if (revision.current !== savedRevision.current && statusRef.current.kind !== 'conflict') void saveRef.current({ background: true })
    }, 30_000)
    return () => window.clearInterval(h)
  }, [canEdit])
  useEffect(() => {
    if (!canEdit || revision.current === savedRevision.current || statusRef.current.kind === 'conflict') return
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
    subjectRef.current = data.subject
    preheaderRef.current = data.preheader
    setSubjectState(data.subject)
    setPreheaderState(data.preheader)
    setABTestState(data.abTest ?? null)
    abRef.current = data.abTest ?? null
    loadContent(data.mjml, { source: 'reload' })
    // What the server has replaces the email: undo starts over from it.
    meta.current?.reset(currentMeta())
    geRef.current?.editor.UndoManager.clear()
    version.current = data.version
    tracker.current?.take()
    pendingSource.current = null
    savedRevision.current = revision.current
    setStatus(data.autosavedAt ? { kind: 'saved', at: data.autosavedAt, version: data.version } : { kind: 'saved', at: data.updatedAt, version: data.version })
    window.setTimeout(() => {
      // The canvas reload fires change events: they aren't edits.
      savedRevision.current = revision.current
      geRef.current?.editor.UndoManager.clear()
      setStatus({ kind: 'saved', at: data.autosavedAt ?? data.updatedAt, version: data.version })
    }, 50)
  }, [slug, campaign.id, qc, loadContent, language, currentMeta])

  const setSubject = useCallback(
    (v: string) => {
      subjectRef.current = v
      setSubjectState(v)
      tracker.current?.note(variant === 'A' ? tt(m.noteSubject) : tt(m.noteSubjectVariant, { variant }))
      meta.current?.change(currentMeta())
      markDirty()
    },
    [markDirty, variant, currentMeta],
  )
  const setPreheader = useCallback(
    (v: string) => {
      preheaderRef.current = v
      setPreheaderState(v)
      tracker.current?.note(tt(m.notePreheader))
      meta.current?.change(currentMeta())
      markDirty()
    },
    [markDirty, currentMeta],
  )
  const setABTest = useCallback(
    (t: ABTest | null) => {
      setABTestState(t)
      abRef.current = t
      tracker.current?.note(t ? tt(m.noteAb, { kind: t.kind }) : tt(m.noteAbRemoved))
      meta.current?.change(currentMeta())
      markDirty()
    },
    [markDirty, currentMeta],
  )
  // The journal's writer: it reads the editor as it is (the words being typed
  // included) and writes at once when the tab is hidden or closed.
  const currentMjmlRef = useRef(currentMjml)
  const abForSaveRef = useRef(abForSave)
  useLayoutEffect(() => {
    currentMjmlRef.current = currentMjml
    abForSaveRef.current = abForSave
  })
  useEffect(() => {
    if (!canEdit) return
    const w = journalWriter({
      key: jKey,
      campaignId: campaign.id,
      language,
      snapshot: () => ({ baseVersion: version.current, subject: subjectRef.current, preheader: preheaderRef.current, mjml: currentMjmlRef.current(), abTest: abForSaveRef.current() ?? null }),
    })
    journal.current = w
    afterSaved.current = (rev0) => {
      // Everything typed is saved: the journal forgets it.
      if (revision.current === rev0) w.clear()
    }
    const hidden = () => {
      if (document.visibilityState === 'hidden') w.flush()
    }
    window.addEventListener('pagehide', w.flush)
    document.addEventListener('visibilitychange', hidden)
    return () => {
      window.removeEventListener('pagehide', w.flush)
      document.removeEventListener('visibilitychange', hidden)
      w.flush()
      w.dispose()
      if (journal.current === w) journal.current = null
      afterSaved.current = null
    }
  }, [canEdit, jKey, campaign.id, language])

  const restoreRecovered = useCallback(async (): Promise<boolean> => {
    const e = recoveredEntry
    if (!e) return false
    const label = tt(mn.recoveredLabel, { from: clock(e.startedAt), to: clock(e.updatedAt) })
    subjectRef.current = e.subject
    preheaderRef.current = e.preheader
    setSubjectState(e.subject)
    setPreheaderState(e.preheader)
    abRef.current = e.abTest
    setABTestState(e.abTest)
    // Taken back in one undo with the canvas it brings (same moment, same step).
    meta.current?.change(currentMeta(), { step: true })
    loadContent(e.mjml, { source: 'recovered', label })
    await whenEditorReady()
    // On top of whatever is current now: the person chose these edits.
    const ok = await save({ force: true, manual: true, quiet: true, source: 'recovered', label })
    if (ok) {
      forgetRecovered(jKey)
      setRecoveredEntry(null)
      toast({ kind: 'success', title: tt(mn.recoveredDone), detail: tt(mn.recoveredDoneDetail, { version: version.current }) })
    } else {
      toast({ kind: 'error', title: tt(mn.recoveredFailed), detail: tt(mn.recoveredFailedDetail) })
    }
    return ok
  }, [recoveredEntry, setRecoveredEntry, loadContent, whenEditorReady, save, jKey, currentMeta])
  const discardRecovered = useCallback(() => {
    forgetRecovered(jKey)
    setRecoveredEntry(null)
  }, [jKey, setRecoveredEntry])
  const recovered = useMemo<Recovered | null>(
    () => (recoveredEntry ? { entry: recoveredEntry, restore: restoreRecovered, discard: discardRecovered } : null),
    [recoveredEntry, restoreRecovered, discardRecovered],
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
        onError: (e) => toast({ kind: 'error', title: tt(m.renameFailed), detail: errorMessage(e) }),
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
            // Each A/B variant is checked too (its subject, or its content).
            body: { subject, preheader, mjml, html, language, abTest: geRef.current ? abForSave() : undefined },
          })
          .then(({ data }) => {
            if (n === seq.current && data) {
              setChecks(data)
              setChecksRun((r) => (r.state === 'failed' ? { state: 'recovered' } : { state: 'ok' }))
            }
          })
          .catch((e: unknown) => {
            // The last results stay, shown as stale with a Retry (ComposerWarnings).
            if (n === seq.current) setChecksRun({ state: 'failed', reason: offlineError(e) ? 'network' : 'server' })
          })
          .finally(() => {
            if (n === seq.current) setChecksPending(false)
          })
      },
      geRef.current ? 900 : 0,
    )
    return () => window.clearTimeout(h)
    // rev covers canvas edits; ge triggers the first run once the editor is ready.
  }, [rev, ge, subject, preheader, abTest, empty, slug, campaign.id, campaign.html, campaign.mjml, compileCurrent, currentMjml, abForSave, checksNonce, language])

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
        toast({ kind: 'error', title: tt(m.fixMjmlFirst), detail: tt(m.fixMjmlFirstDetail) })
        return
      }
      if (v !== 'visual' && revision.current !== savedRevision.current) void save()
      onSearch({ ...search, view: v === 'visual' ? undefined : v })
    },
    [view, onSearch, search, save],
  )
  const setDevice = useCallback((d: DeviceId) => onSearch({ ...search, device: d === 'mobile' ? 'mobile' : undefined }), [onSearch, search])

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
    variantContent,
    hasUnsaved: () => revision.current !== savedRevision.current || pendingSource.current !== null || statusRef.current.kind === 'offline' || statusRef.current.kind === 'error' || statusRef.current.kind === 'conflict',
    currentVersion: () => version.current,
    currentMjml,
    compileCurrent,
    loadContent,
    reloadFromServer,
    onSettingsChanged,
    registerLeaveCheck,
    setCodeDirty,
    isEmpty: empty,
    // The images' alt text is listed in the Accessibility tab.
    checks: empty ? undefined : withoutAltText(checks),
    checksPending,
    checksFailure: checksRun.state === 'failed' ? { reason: checksRun.reason ?? 'server' } : null,
    checksRecovered: checksRun.state === 'recovered',
    retryChecks: () => setChecksNonce((n) => n + 1),
    rightTab,
    setRightTab,
    thread: search.thread,
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

  /** "Edit anyway": saves from now on cancel the approval; a save that asked goes through. */
  const editAnyway = useCallback(async () => {
    cancelApproval.current = true
    setApprovalGate(false)
    const via = askApproval
    setAskApproval(null)
    if (via === 'save') await save({ manual: true, quiet: true })
  }, [askApproval, save])

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
          whileEnding: t(m.whileEnding),
          afterEnded: savedVersion ? t(m.afterEnded, { version: savedVersion }) : undefined,
        }
      : undefined,
  )
  const lastSave = status.kind === 'saved' || status.kind === 'dirty' ? status.at : undefined
  // Dialogs board: "You changed the subject line and two blocks since the last autosave …
  // Leaving now keeps draft v16; saving creates v17." (The tracker names what changed.)
  // eslint-disable-next-line react-hooks/refs -- re-read on every edit: each change bumps `rev`
  const changed = tracker.current?.peek() ?? []
  const leaveValues = { changes: andList(changed), time: lastSave ? clock(lastSave) : '', draft: campaign.status === 'draft' ? 'yes' : 'no', version: savedVersion, next: savedVersion + 1 }
  const leaveText = t(changed.length ? (lastSave ? m.leaveChangesAt : m.leaveChanges) : lastSave ? m.leaveEmailAt : m.leaveEmail, leaveValues)

  return (
    <ComposerCtx.Provider value={state}>
      <WebModeProvider campaign={campaign}>
      <ComposerAssistScope value={assist}>
      <AccessibilityProvider>
      <DigestModeProvider>
      <ConditionalProvider>
      <div className="cmp rc-app">
        <a href="#main" className="skip">
          {t(m.skip)}
        </a>
        <h1 className="vh">{view === 'preview' ? t(m.previewHeading, { name }) : t(m.heading, { name: name || t(m.untitled) })}</h1>
        <Header onReview={goReview} />
        {/* Sponsor blocks fill themselves from the issue's booking (Earn, X17). */}
        <SponsorSync />
        {!campaign.access.canSend && canEditPerm ? <PermissionBanner /> : null}
        {approvalGate && approvalInfo && view !== 'preview' ? <ApprovalBanner approval={approvalInfo} onEdit={() => setAskApproval('open')} /> : null}
        {view === 'preview' ? null : <ComposerBanners recovered={recovered} />}
        {view === 'preview' ? <PreviewView onReview={goReview} /> : null}
        {/* The canvas stays mounted off screen in the other views: GrapesJS keeps its state and compiles. */}
        <div className={`flex flex-1 flex-col${view === 'preview' ? ' cmp-offstage' : ''}`} inert={view === 'preview' || undefined} aria-hidden={view === 'preview' || undefined}>
          {view === 'visual' && !comparing && !missingLanguage ? <SubjectBar /> : null}
          <div className="cmp-body">
            {view === 'visual' && !comparing && !missingLanguage ? <LeftPanel {...digestLeftPanel(campaign, context.postalAddress)} /> : null}
            {comparing ? assist.translating ? <AITranslateView /> : <ComposerCompare /> : null}
            {missingLanguage && !comparing ? <ComposerMissingLanguage language={missingLanguage} /> : null}
            <div
              className={`cmp-main${view !== 'visual' || comparing || missingLanguage ? ' cmp-offstage' : ''}`}
              inert={view !== 'visual' || comparing || Boolean(missingLanguage) || undefined}
              aria-hidden={view !== 'visual' || comparing || Boolean(missingLanguage) || undefined}
            >
              <CanvasArea containerRef={containerRef} savedCard={savedCard} onCloseSaved={() => setSavedCard(null)} onReview={goReview} />
              {/* The website's Web view of the canvas (features/website). */}
              <WebCanvasView />
            </div>
            {view === 'mjml' || view === 'html' ? (
              <Suspense fallback={<div className="cmp-main" style={{ alignItems: 'center', justifyContent: 'center' }} aria-busy="true"><span className="spin" aria-hidden="true" /></div>}>
                <CodeView />
              </Suspense>
            ) : null}
            {view === 'visual' && !comparing && !missingLanguage ? <AssistRight fallback={<RightPanel />} /> : null}
          </div>
          {view === 'preview' || (missingLanguage && !comparing) ? null : assist.translating ? (
            <AITranslateFooter />
          ) : comparing ? (
            <ComposerCompareFooter>
              <VersionLink />
            </ComposerCompareFooter>
          ) : (
            <Footer />
          )}
        </div>
        <Dialogs dialog={dialog} onClose={() => setDialog(null)} onSavedTemplate={(t) => setSavedCard({ version: version.current, at: new Date().toISOString(), template: t })} />
        {approvalInfo ? (
          <ApprovalDialog approval={approvalInfo} scheduledAt={campaign.scheduledAt} open={askApproval !== null} onOpenChange={(o) => !o && setAskApproval(null)} onEdit={editAnyway} />
        ) : null}
        <UnsavedChangesGuard
          dirty={canEdit && (unsaved || codeDirty)}
          title={t(m.leaveTitle)}
          description={leaveText}
          saveLabel={t(m.leaveSave, { version: savedVersion + 1 })}
          onSave={async () => {
            if (!(await save({ manual: true, quiet: true }))) throw new Error(t(m.leaveSaveFailed))
          }}
        />
      </div>
      </ConditionalProvider>
      </DigestModeProvider>
      </AccessibilityProvider>
      </ComposerAssistScope>
      </WebModeProvider>
    </ComposerCtx.Provider>
  )
}
