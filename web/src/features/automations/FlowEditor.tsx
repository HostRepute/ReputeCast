import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  applyEdgeChanges,
  applyNodeChanges,
  Background,
  BackgroundVariant,
  ReactFlow,
  ReactFlowProvider,
  useNodesInitialized,
  useReactFlow,
  useStore,
  type Connection,
  type Edge,
  type EdgeChange,
  type NodeChange,
  type OnDelete,
} from '@xyflow/react'
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type DragEvent, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, Dot } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Banner } from '@/components/ui/Note'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { count, dayMonthDate, pct } from '@/lib/format'
import { useDocumentTitle, useMediaQuery, useReducedMotion } from '@/lib/hooks'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { useUnsavedWork } from '@/lib/sessionClock'
import { useShortcutHandler } from '@/lib/shortcuts'
import { CanvasControls, FlowMiniMap } from './canvas/CanvasChrome'
import { Motion } from './canvas/Motion'
import { edgeTypes } from './canvas/StepEdge'
import { nodeTypes } from './canvas/StepNode'
import { EmailEditDialog, EmailPreviewDialog, PauseDialog, PublishDialog, VersionHistoryDialog } from './dialogs'
import {
  addStep,
  canConnect,
  connect,
  duplicateStep,
  edgeAt,
  fromGraph,
  healAfterDelete,
  nodeHeight,
  nodeWidth,
  toGraph,
  type FlowState,
  type StepEdgeType,
  type StepNodeType,
  type XY,
} from './edit'
import {
  entrance,
  isTerminal,
  newStepData,
  nextNodeId,
  nodeTitle,
  pathLabel,
  refsFrom,
  serialize,
  triggerKindLabel,
  typeName,
  type FlowGraph,
  type NodeData,
  type StepType,
} from './graph'
import { DRAG_TYPE, draggedStepType, FlowStatsList, graphOrder, Palette, Prop, StepPanel, StepStatsPanel, type PaletteItem } from './panels'
import { storeOfTrigger, useStoreTriggerOptions } from '@/features/commerce/automation'
import { t as tt, T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { mf } from './flow.messages'
import { AutomationStatusBadge } from './StatusBadge'
import { FlowViewProvider, startCountUp, type FlowView, type StepStats } from './view'
import './automations.css'

type Detail = Schemas['AutomationDetail']
type Issue = Schemas['AutomationGraphIssue']

export type EditorPanel = 'step' | 'stats'
export interface EditorSearch {
  step?: string
  panel?: EditorPanel
}

const GRID = 20
const snapPoint = (p: XY): XY => ({ x: Math.round(p.x / GRID) * GRID, y: Math.round(p.y / GRID) * GRID })
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.userAgent)
const ANIMATE_KEY = 'reputecast.automations.animate'

function readAnimatePref(): boolean {
  try {
    return window.localStorage.getItem(ANIMATE_KEY) !== 'off'
  } catch {
    return true
  }
}

function writeAnimatePref(on: boolean) {
  try {
    window.localStorage.setItem(ANIMATE_KEY, on ? 'on' : 'off')
  } catch {
    // Private mode: the toggle still works for this visit.
  }
}

/**
 * 7.10 Automation flow: the full-screen editor (AutomationFlow board). The
 * draft saves itself as you edit; publishing makes it the live version.
 */
export function FlowEditorPage({ automationId, search, onSearch }: { automationId: string; search: EditorSearch; onSearch: (s: EditorSearch) => void }) {
  const { slug } = useWorkspace()
  const detail = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automations/{id}',
    { params: { path: { workspace: slug, id: automationId } } },
    { staleTime: 60_000, refetchOnWindowFocus: false },
  )
  // A new generation re-reads the draft from the server (after a restore or a conflict).
  const [generation, setGeneration] = useState(0)
  const { refetch } = detail
  const reload = useCallback(async () => {
    await refetch()
    setGeneration((g) => g + 1)
  }, [refetch])
  if (detail.isPending) return <EditorSkeleton />
  if (detail.isError) return <EditorError error={detail.error} onRetry={() => void refetch()} />
  return (
    <ReactFlowProvider key={`${automationId}:${generation}`}>
      <Editor detail={detail.data} search={search} onSearch={onSearch} onReload={reload} />
    </ReactFlowProvider>
  )
}

/** Creates a draft from ?template= (or blank), on ?list= when given, and opens it. */
export function NewAutomationPage({ template, list }: { template?: Schemas['AutomationCreateBody']['template']; list?: string }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const reason = usePermissionReason('automations:edit')
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations')
  const started = useRef(false)
  const { mutate } = create
  useEffect(() => {
    if (started.current || reason) return
    started.current = true
    mutate(
      { params: { path: { workspace: slug } }, body: template ? { template, list } : {} },
      { onSuccess: (r) => void navigate({ to: '/w/$workspace/automations/$automationId', params: { workspace: slug, automationId: r.id }, search: {}, replace: true }) },
    )
  }, [reason, slug, template, list, mutate, navigate])
  if (reason) return <EditorError error={new ApiError(403, { status: 403, title: tt(mf.notAllowed), detail: reason } as Schemas['Problem'])} />
  if (create.isError) return <EditorError error={create.error} title={tt(mf.createFailed)} />
  return <EditorSkeleton />
}

interface Meta {
  revision: number
  changes: number
  draftVersion: number
  issues: Issue[]
}

function firstSelection(g: FlowGraph, step?: string): string | undefined {
  if (step && g.nodes.some((n) => n.id === step)) return step
  const order = graphOrder(g)
  return order.find((id) => g.nodes.find((n) => n.id === id)?.type === 'email') ?? order[0]
}

function Editor({ detail, search, onSearch, onReload }: { detail: Detail; search: EditorSearch; onSearch: (s: EditorSearch) => void; onReload: () => Promise<void> }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const rf = useReactFlow<StepNodeType, StepEdgeType>()
  const reduced = useReducedMotion()
  const coarse = useMediaQuery('(pointer: coarse)')
  const canEdit = useCan('automations:edit')
  const publishReason = usePermissionReason('automations:publish')
  const id = detail.id
  const published = detail.liveVersion !== undefined
  const editReason = t(mf.editReason)
  useDocumentTitle(t(mf.documentTitle, { name: detail.name, workspace: me.workspace.name }))

  const stats = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automations/{id}/stats',
    { params: { path: { workspace: slug, id }, query: { days: 30 } } },
    { enabled: published, refetchInterval: 30_000 },
  )
  const options = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations/options', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  // The store triggers name their store (X20).
  const storeOptions = useStoreTriggerOptions()
  const refs = useMemo(() => ({ ...refsFrom(options.data), stores: new Map((storeOptions.data?.stores ?? []).map((s) => [s.id, s.name])) }), [options.data, storeOptions.data])

  const [flow, setFlow] = useState<FlowState>(() => fromGraph(detail.graph))
  const graph = useMemo(() => toGraph(flow.nodes, flow.edges), [flow])
  const json = useMemo(() => JSON.stringify(serialize(flow.nodes, flow.edges)), [flow])
  const [meta, setMeta] = useState<Meta>({ revision: detail.revision, changes: detail.changes, draftVersion: detail.draftVersion, issues: detail.issues })

  const [savedJson, setSavedJson] = useState(json)
  const [saving, setSaving] = useState(false)
  const [failed, setFailed] = useState<{ json: string; error: unknown } | null>(null)
  const [conflict, setConflict] = useState(false)
  const [restoring, setRestoring] = useState(false)
  const latest = useRef({ json, savedJson, revision: detail.revision, inflight: null as Promise<boolean> | null, stopped: false })
  const { mutateAsync: putDraft } = $api.useMutation('put', '/api/v1/workspaces/{workspace}/automations/{id}/draft')
  const dirty = json !== savedJson
  const stopped = conflict || restoring || !canEdit
  useLayoutEffect(() => {
    latest.current.json = json
    latest.current.savedJson = savedJson
    latest.current.stopped = stopped
  }, [json, savedJson, stopped])

  const save = useCallback(
    (body: string): Promise<boolean> => {
      const run = async () => {
        setSaving(true)
        try {
          const res = await putDraft({ params: { path: { workspace: slug, id } }, body: { graph: JSON.parse(body) as FlowGraph, revision: latest.current.revision } })
          latest.current.revision = res.revision
          latest.current.savedJson = body
          setSavedJson(body)
          setFailed(null)
          setMeta({ revision: res.revision, changes: res.changes, draftVersion: res.draftVersion, issues: res.issues })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations'] })
          return true
        } catch (e) {
          if (e instanceof ApiError && e.status === 409) setConflict(true)
          else setFailed({ json: body, error: e })
          return false
        } finally {
          setSaving(false)
        }
      }
      const p = run()
      latest.current.inflight = p
      void p.finally(() => {
        if (latest.current.inflight === p) latest.current.inflight = null
      })
      return p
    },
    [putDraft, slug, id, qc],
  )

  useEffect(() => {
    if (stopped || saving || json === savedJson || failed?.json === json) return
    const h = window.setTimeout(() => void save(json), 700)
    return () => window.clearTimeout(h)
  }, [json, savedJson, stopped, saving, failed, save])

  /** Saves now (Publish and "Save draft only" need the draft on the server). */
  const flush = useCallback(async () => {
    const l = latest.current
    if (l.inflight) await l.inflight
    if (l.stopped) throw new Error(tt(mf.notSavedError))
    if (l.json === l.savedJson) return
    if (!(await save(l.json))) throw new Error(tt(mf.saveFailedError))
  }, [save])

  // Leaving the editor saves what is left; closing the tab with unsaved edits asks first.
  useEffect(() => {
    const l = latest
    return () => {
      const s = l.current
      if (s.stopped || s.json === s.savedJson) return
      void http.PUT('/api/v1/workspaces/{workspace}/automations/{id}/draft', { params: { path: { workspace: slug, id } }, body: { graph: JSON.parse(s.json) as FlowGraph, revision: s.revision } }).catch(() => undefined)
    }
  }, [slug, id])
  useEffect(() => {
    if (!dirty && !saving) return
    const h = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', h)
    return () => window.removeEventListener('beforeunload', h)
  }, [dirty, saving])
  // ⌘S saves the draft right away (it saves itself anyway).
  useShortcutHandler(
    'save',
    canEdit
      ? () => {
          flush().then(
            () => toast({ title: t(mf.savedToast, { version: meta.draftVersion }), detail: t(mf.publishWhenReady) }),
            (e: unknown) => toast({ kind: 'error', title: t(mf.saveFailed), detail: errorMessage(e) }),
          )
        }
      : undefined,
  )
  // If the session ends while editing, the notices say the work is safe.
  useUnsavedWork(canEdit ? { whileEnding: t(mf.whileEnding), afterEnded: t(mf.afterEnded, { name: detail.name, version: meta.draftVersion }) } : undefined)

  // The selection and the right panel are kept in the URL (?step=n3&panel=stats).
  const [selectedId, setSelectedId] = useState<string | undefined>(() => firstSelection(detail.graph, search.step))
  const [rightTab, setRightTab] = useState<EditorPanel>(search.panel ?? 'step')
  const [seen, setSeen] = useState(search)
  if (seen.step !== search.step || seen.panel !== search.panel) {
    // The URL changed: our own update arriving, or the back button or a link
    // (then follow it).
    setSeen(search)
    if (search.step !== seen.step && search.step && search.step !== selectedId && flow.nodes.some((n) => n.id === search.step)) setSelectedId(search.step)
    if (search.panel !== seen.panel && (search.panel ?? 'step') !== rightTab) setRightTab(search.panel ?? 'step')
  }
  const select = useCallback(
    (step?: string) => {
      setSelectedId(step)
      onSearch({ step, panel: rightTab === 'step' ? undefined : rightTab })
    },
    [onSearch, rightTab],
  )
  const showTab = (tab: EditorPanel) => {
    setRightTab(tab)
    const panel = tab === 'step' ? undefined : tab
    onSearch({ step: selectedId, panel })
  }
  const [leftTab, setLeftTab] = useState<'steps' | 'stats'>('steps')

  const [showStats, setShowStats] = useState(true)
  const [snap, setSnap] = useState(true)
  const [animate, setAnimate] = useState(readAnimatePref)
  const [locked, setLocked] = useState(false)
  const editable = canEdit && !conflict && !restoring
  const canvasEditable = editable && !locked

  const statsMap = useMemo(() => new Map<string, StepStats>((stats.data?.steps ?? []).map((s) => [s.nodeId, s])), [stats.data])
  const liveIds = useMemo(() => new Set(detail.liveNodeIds), [detail.liveNodeIds])
  const publishedIds = useMemo(() => new Set(detail.publishedNodeIds), [detail.publishedNodeIds])
  const issueIds = useMemo(() => new Set(meta.issues.filter((i) => i.severity === 'error' && i.nodeId).map((i) => i.nodeId!)), [meta.issues])
  const [entranceTimes] = useState(() => (reduced ? undefined : entrance(detail.graph)))
  const view = useMemo<FlowView>(
    () => ({
      automationId: id,
      graph,
      refs,
      stats: statsMap,
      entered: stats.data?.entered ?? 0,
      hasStats: published && stats.data !== undefined,
      showStats,
      liveIds,
      publishedIds,
      published,
      live: detail.status === 'live',
      draftVersion: meta.draftVersion,
      issues: issueIds,
      fallbackId: detail.route.fallback?.id,
      entrance: entranceTimes,
    }),
    [id, graph, refs, statsMap, stats.data, published, showStats, liveIds, publishedIds, detail.status, meta.draftVersion, issueIds, detail.route.fallback?.id, entranceTimes],
  )

  // Numbers on the canvas count up once, when the stats arrive (like the board).
  const counted = useRef(false)
  useLayoutEffect(() => {
    if (!stats.data || counted.current) return
    counted.current = true
    startCountUp(reduced)
  }, [stats.data, reduced])

  const [dropEdge, setDropEdge] = useState<string>()
  const titles = useMemo(() => new Map(graph.nodes.map((n) => [n.id, nodeTitle(graph, n, refs)])), [graph, refs])
  const rfNodes = useMemo(
    () => flow.nodes.map((n) => ({ ...n, selected: n.id === selectedId, ariaLabel: t(mf.nodeLabel, { type: typeName(n.type), title: titles.get(n.id) ?? '' }) })),
    [flow.nodes, selectedId, titles, t],
  )
  const rfEdges = useMemo(
    () =>
      flow.edges.map((e) => {
        const from = titles.get(e.source) ?? e.source
        const to = titles.get(e.target) ?? e.target
        return {
          ...e,
          className: e.id === dropEdge ? 'fe-drop' : undefined,
          ariaLabel: e.data?.path ? t(mf.edgePathLabel, { from, to, path: pathLabel(e.data.path) }) : t(mf.edgeLabel, { from, to }),
        }
      }),
    [flow.edges, dropEdge, titles, t],
  )

  const onNodesChange = useCallback(
    (changes: NodeChange<StepNodeType>[]) => {
      const rest: NodeChange<StepNodeType>[] = []
      let picked: string | null | undefined
      for (const c of changes) {
        if (c.type !== 'select') {
          rest.push(c)
          continue
        }
        if (c.selected) picked = c.id
        else if (c.id === selectedId && picked === undefined) picked = null
      }
      if (rest.length) setFlow((f) => ({ ...f, nodes: applyNodeChanges(rest, f.nodes) }))
      if (picked !== undefined && (picked ?? undefined) !== selectedId) select(picked ?? undefined)
    },
    [selectedId, select],
  )
  const onEdgesChange = useCallback((changes: EdgeChange<StepEdgeType>[]) => setFlow((f) => ({ ...f, edges: applyEdgeChanges(changes, f.edges) })), [])
  const onDelete: OnDelete<StepNodeType, StepEdgeType> = useCallback(
    ({ nodes: gone, edges: goneEdges }) => {
      if (gone.length === 0) return
      // The deleted step's parents now lead to its next step.
      setFlow((f) => ({ ...f, edges: [...f.edges, ...healAfterDelete(gone, goneEdges, f.edges)] }))
      if (gone.some((n) => n.id === selectedId)) select(undefined)
    },
    [selectedId, select],
  )
  const isValidConnection = useCallback((c: Connection | Edge) => canConnect(graph, c.source, c.target), [graph])
  const onConnect = useCallback((c: Connection) => setFlow((f) => connect(f, c.source, c.target)), [])

  /** Adds a step: dropped at a point (or onto an edge), or after the selected step. */
  const add = useCallback(
    (item: PaletteItem, where: { at?: XY; after?: string }) => {
      if (!editable) return
      if (item.type === 'trigger') {
        const existing = graph.nodes.find((n) => n.type === 'trigger')
        if (existing) {
          const kind = item.trigger ?? 'joined_list'
          const trigger = { ...newStepData('trigger', kind).trigger!, reentry: existing.data.trigger?.reentry ?? 'once' }
          setFlow((f) => ({ ...f, nodes: f.nodes.map((n) => (n.id === existing.id ? { ...n, data: { ...n.data, trigger } } : n)) }))
          select(existing.id)
          toast({ title: t(mf.triggerChanged, { trigger: triggerKindLabel(kind) }), detail: t(mf.triggerChangedDetail) })
          return
        }
      }
      const step = { id: nextNodeId(graph), type: item.type, data: newStepData(item.type, item.trigger) }
      const tolerance = 12 / rf.getZoom()
      setFlow((f) => addStep(f, step, { ...where, tolerance, snap: snap ? snapPoint : undefined }))
      select(step.id)
    },
    [editable, graph, rf, snap, select, t],
  )

  const onDragOver = useCallback(
    (e: DragEvent) => {
      if (!e.dataTransfer.types.includes(DRAG_TYPE)) return
      e.preventDefault()
      e.dataTransfer.dropEffect = canvasEditable ? 'move' : 'none'
      const type = draggedStepType(e.dataTransfer.types)
      const on = canvasEditable && type && type !== 'trigger' && !isTerminal(type) ? edgeAt(flow, rf.screenToFlowPosition({ x: e.clientX, y: e.clientY }), 12 / rf.getZoom()) : undefined
      if (on?.id !== dropEdge) setDropEdge(on?.id)
    },
    [canvasEditable, flow, rf, dropEdge],
  )
  const onDrop = useCallback(
    (e: DragEvent) => {
      const raw = e.dataTransfer.getData(DRAG_TYPE)
      setDropEdge(undefined)
      if (!raw || !canvasEditable) return
      e.preventDefault()
      let item: PaletteItem
      try {
        item = JSON.parse(raw) as PaletteItem
      } catch {
        return
      }
      add(item, { at: rf.screenToFlowPosition({ x: e.clientX, y: e.clientY }) })
    },
    [add, canvasEditable, rf],
  )

  const selected = selectedId ? graph.nodes.find((n) => n.id === selectedId) : undefined
  const changeData = (nodeId: string, data: NodeData) => setFlow((f) => ({ ...f, nodes: f.nodes.map((n) => (n.id === nodeId ? { ...n, data } : n)) }))
  const changeType = (nodeId: string, type: StepType) =>
    setFlow((f) => ({
      ...f,
      nodes: f.nodes.map((n) => (n.id === nodeId ? { ...n, type, data: type === 'goal' ? { ...newStepData('goal'), label: n.data.label } : { label: n.data.label } } : n)),
    }))
  const duplicate = () => {
    if (!selectedId) return
    const copy = nextNodeId(graph)
    setFlow((f) => duplicateStep(f, selectedId, copy))
    select(copy)
  }
  const remove = () => {
    if (selectedId) void rf.deleteElements({ nodes: [{ id: selectedId }] })
  }

  // Canvas size and first view: the board centres a 900px frame.
  const canvasRef = useRef<HTMLDivElement>(null)
  const userMoved = useRef(false)
  const [frameWidth] = useState(() => {
    const xs = detail.graph.nodes.map((n) => n.position.x)
    if (xs.length === 0) return 900
    return Math.max(900, Math.max(...xs) + 224 + Math.max(0, Math.min(...xs)))
  })
  const minHeight = useMemo(() => Math.max(400, ...flow.nodes.map((n) => n.position.y + nodeHeight(n) + 50)), [flow.nodes])
  useEffect(() => {
    const el = canvasRef.current
    if (!el) return
    const place = () => {
      if (!userMoved.current) void rf.setViewport({ x: Math.max(0, (el.clientWidth - frameWidth) / 2), y: 0, zoom: 1 })
    }
    const ro = new ResizeObserver(place)
    ro.observe(el)
    return () => ro.disconnect()
  }, [rf, frameWidth])

  // A link to a step (from an alert) scrolls the canvas to it when it is out of view.
  const initialized = useNodesInitialized()
  const focused = useRef(false)
  useEffect(() => {
    if (!initialized || focused.current) return
    focused.current = true
    const n = search.step ? rf.getNode(search.step) : undefined
    const el = canvasRef.current
    if (!n || !el) return
    const { x, y, zoom } = rf.getViewport()
    const left = n.position.x * zoom + x
    const top = n.position.y * zoom + y
    if (left >= 0 && top >= 0 && left + nodeWidth(n) * zoom <= el.clientWidth && top + nodeHeight(n) * zoom <= el.clientHeight) return
    userMoved.current = true
    void rf.setCenter(n.position.x + nodeWidth(n) / 2, n.position.y + nodeHeight(n) / 2, { zoom, duration: 0 })
  }, [initialized, rf, search.step])

  const [publishOpen, setPublishOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [pauseOpen, setPauseOpen] = useState(false)
  const [preview, setPreview] = useState<{ id: string; version: number } | null>(null)
  const [editingEmail, setEditingEmail] = useState<{ id: string; version: number; node: string } | null>(null)
  const [opening, setOpening] = useState(false)
  const pause = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/pause')
  const resume = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automations/{id}/resume')
  const refreshDetail = () => {
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/{id}'] })
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations'] })
  }
  const openPublish = async () => {
    setOpening(true)
    try {
      await flush()
      setPublishOpen(true)
    } catch (e) {
      toast({ kind: 'error', title: t(mf.saveBeforePublishFailed), detail: errorMessage(e) })
    } finally {
      setOpening(false)
    }
  }
  const nothingToPublish = published && meta.changes === 0 && !dirty ? t(mf.nothingToPublish, { version: detail.liveVersion ?? '' }) : undefined
  const publishBlocked = publishReason ?? (conflict ? t(mf.loadLatestFirst) : nothingToPublish)

  const order = useMemo(() => graphOrder(graph), [graph])
  const zoomKey = isMac ? '⌘' : 'Ctrl'

  return (
    <div className="fe rc-app">
      <a href="#main" className="skip">
        {t(mf.skip)}
      </a>
      <header className="fe-head">
        <ButtonLink to="/w/$workspace/automations" params={{ workspace: slug }} iconOnly icon="chevronLeft" aria-label={t(mf.back)} />
        <div className="flex min-w-0 flex-col">
          <span className="k text-[11px]">{t(mf.eyebrow)}</span>
          <h1 className="m-0 text-[15px] font-bold leading-[1.45] tracking-normal">{detail.name}</h1>
        </div>
        <StatusBadge detail={detail} />
        {published && meta.changes > 0 ? <Badge tone="warn">{t(mf.unpublishedChanges, { count: count(meta.changes) })}</Badge> : null}
        <Badge tone={detail.route.healthy ? 'purple' : 'bad'} className="ml-1.5">
          <Dot color={detail.route.healthy ? 'var(--chart-ses)' : 'currentColor'} />
          {t(mf.route, { route: detail.route.label })}
        </Badge>
        {failed ? (
          <span className="flex items-center gap-2" role="alert">
            <Badge tone="bad">{t(mf.draftNotSaved)}</Badge>
            <Button size="sm" onClick={() => void save(json)}>
              {t(common.retry)}
            </Button>
          </span>
        ) : null}
        <span className="vh" role="status">
          {saving ? t(mf.savingDraft) : dirty ? '' : t(mf.draftSaved)}
        </span>
        <div className="ml-auto flex flex-wrap gap-2">
          {detail.status === 'live' ? (
            <Button disabledReason={publishReason} onClick={() => setPauseOpen(true)}>
              {t(mf.pause)}
            </Button>
          ) : null}
          {detail.status === 'paused' ? (
            <Button
              disabledReason={publishReason}
              loading={resume.isPending}
              onClick={() =>
                resume.mutate(
                  { params: { path: { workspace: slug, id } } },
                  {
                    onSuccess: () => {
                      toast({ title: t(mf.resumed, { name: detail.name }), detail: t(mf.resumedDetail) })
                      refreshDetail()
                    },
                    onError: (e) => toast({ kind: 'error', title: t(mf.resumeFailed), detail: errorMessage(e) }),
                  },
                )
              }
            >
              {t(mf.resume)}
            </Button>
          ) : null}
          <Button onClick={() => setHistoryOpen(true)}>{t(mf.versionHistory)}</Button>
          <Button variant="primary" loading={opening} disabledReason={publishBlocked} onClick={() => void openPublish()}>
            {t(mf.publishVersion, { version: meta.draftVersion })}
          </Button>
        </div>
      </header>

      {conflict ? (
        <div className="px-4 pt-3">
          <Banner tone="warn" title={t(mf.conflictTitle)} actions={<Button onClick={() => void onReload()}>{t(mf.loadLatest)}</Button>}>
            {t(mf.conflictBody)}
          </Banner>
        </div>
      ) : null}

      <div className="fe-body">
        <aside className="fe-panel fe-left" aria-label={t(mf.stepsPanel)}>
          <PanelTabs
            value={leftTab}
            onChange={setLeftTab}
            label={t(mf.stepsTabs)}
            className="flex flex-1 flex-col"
            items={[
              { value: 'steps', label: t(mf.tabSteps) },
              { value: 'stats', label: t(mf.tabStats) },
            ]}
          >
            <TabPanel value="steps" className="flex flex-1 flex-col">
              <Palette disabledReason={!canEdit ? editReason : !editable ? t(mf.loadLatestFirst) : undefined} onAdd={(item) => add(item, { after: selectedId })} />
              <div className="k mt-auto border-t border-line-2 px-3 py-2.5">{canEdit ? t(mf.dragHint, { snap: snap ? 'on' : 'off' }) : t(mf.lookAround)}</div>
            </TabPanel>
            <TabPanel value="stats" className="flex flex-1 flex-col">
              {published ? (
                <FlowStatsList graph={graph} order={order} refs={refs} stats={statsMap} selected={selectedId} onSelect={select} />
              ) : (
                <div className="k px-3 py-2.5">{t(mf.statsStart)}</div>
              )}
            </TabPanel>
          </PanelTabs>
        </aside>

        <main id="main" className="fe-main">
          <div className="fe-tools">
            <span className="k">
              {published ? (stats.data ? t(mf.liveStatsEntered, { count: count(stats.data.entered) }) : t(mf.liveStats)) : t(mf.noStatsYet)}
            </span>
            <label className="ml-2">
              <input type="checkbox" checked={showStats} onChange={(e) => setShowStats(e.target.checked)} />
              {t(mf.showStats)}
            </label>
            <label>
              <input type="checkbox" checked={snap} onChange={(e) => setSnap(e.target.checked)} />
              {t(mf.snapToGrid)}
            </label>
            <label title={reduced ? t(mf.reducedMotion) : undefined}>
              <input
                type="checkbox"
                checked={animate}
                onChange={(e) => {
                  setAnimate(e.target.checked)
                  writeAnimatePref(e.target.checked)
                }}
              />
              {t(mf.animate)}
            </label>
            <span className="k ml-auto">
              <T msg={mf.zoomHint} values={{ zoom: <ZoomLevel />, modifier: zoomKey }} />
            </span>
          </div>

          <div ref={canvasRef} className="fe-canvas" style={{ minHeight }} role="group" aria-label={t(mf.canvasLabel)}>
            <FlowViewProvider value={view}>
              <ReactFlow<StepNodeType, StepEdgeType>
                nodes={rfNodes}
                edges={rfEdges}
                nodeTypes={nodeTypes}
                edgeTypes={edgeTypes}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onDelete={onDelete}
                onConnect={onConnect}
                isValidConnection={isValidConnection}
                onPaneClick={() => select(undefined)}
                onDragOver={onDragOver}
                onDragLeave={() => setDropEdge(undefined)}
                onDrop={onDrop}
                onMoveStart={(e) => {
                  if (e) userMoved.current = true
                }}
                defaultViewport={{ x: 0, y: 0, zoom: 1 }}
                minZoom={0.3}
                maxZoom={2}
                snapToGrid={snap}
                snapGrid={[GRID, GRID]}
                panOnScroll
                zoomOnScroll={false}
                zoomOnDoubleClick={false}
                panOnDrag={!coarse}
                preventScrolling={!coarse}
                nodesDraggable={canvasEditable}
                nodesConnectable={canvasEditable}
                elementsSelectable
                deleteKeyCode={canvasEditable ? ['Backspace', 'Delete'] : null}
                selectionKeyCode={null}
                multiSelectionKeyCode={null}
                defaultEdgeOptions={{ type: 'step' }}
                proOptions={{ hideAttribution: true }}
                className={cn(!(animate && !reduced) && 'fe-still')}
              >
                <Background variant={BackgroundVariant.Dots} gap={GRID} size={2} offset={1} color="#D4D8DF" />
                <Motion playing={animate} reduced={reduced} />
                <CanvasControls locked={locked} onLock={setLocked} />
                <FlowMiniMap nodes={flow.nodes} selected={selectedId} />
              </ReactFlow>
            </FlowViewProvider>
            {flow.nodes.length === 0 ? (
              <div className="fe-empty">
                <div className="card flex max-w-[320px] flex-col gap-1 px-5 py-4 text-center">
                  <span className="font-semibold">{t(mf.emptyTitle)}</span>
                  <span className="k">{canEdit ? t(mf.emptyEdit) : t(mf.emptyView)}</span>
                </div>
              </div>
            ) : null}
          </div>
        </main>

        <aside className="fe-panel fe-right" aria-label={t(mf.settingsPanel)}>
          <PanelTabs
            value={rightTab}
            onChange={showTab}
            label={t(mf.settingsTabs)}
            className="flex flex-1 flex-col"
            items={[
              { value: 'step', label: t(mf.tabStep) },
              { value: 'stats', label: t(mf.tabStats) },
            ]}
          >
            <TabPanel value="step" className="flex flex-1 flex-col">
              {selected ? (
                <StepPanel
                  key={selected.id}
                  graph={graph}
                  node={selected}
                  refs={refs}
                  options={options.data}
                  stats={statsMap.get(selected.id)}
                  hasStats={view.hasStats && publishedIds.has(selected.id)}
                  editable={editable}
                  editReason={!canEdit ? editReason : t(mf.loadLatestFirst)}
                  automationId={id}
                  automationName={detail.name}
                  routeLabel={detail.route.label}
                  routePrimary={detail.route.primary.map((p) => p.id)}
                  issues={meta.issues.filter((i) => i.nodeId === selected.id).map((i) => i.message)}
                  onChange={(data) => changeData(selected.id, data)}
                  onChangeType={(type) => changeType(selected.id, type)}
                  onDuplicate={duplicate}
                  onDelete={remove}
                  onPreview={(emailId, version) => setPreview({ id: emailId, version })}
                  onEditEmail={(emailId, version) => setEditingEmail({ id: emailId, version, node: selected.id })}
                />
              ) : (
                <FlowPanel key={detail.name} detail={detail} editable={editable} issues={meta.issues} />
              )}
            </TabPanel>
            <TabPanel value="stats" className="flex flex-1 flex-col">
              {selected ? (
                <StepStatsPanel node={selected} stats={statsMap.get(selected.id)} automationId={id} hasStats={published} />
              ) : (
                <FlowStatsPanel stats={stats.data} inFlow={detail.inFlow} published={published} />
              )}
            </TabPanel>
          </PanelTabs>
        </aside>
      </div>

      <UnsavedChangesGuard dirty={Boolean(failed) && dirty} description={t(mf.unsavedGuard)} onSave={flush} />
      <PublishDialog open={publishOpen} onOpenChange={setPublishOpen} automationId={id} onSaveDraft={flush} />
      <VersionHistoryDialog
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        automationId={id}
        hasDraftChanges={meta.changes > 0 || dirty}
        onBeforeRestore={() => setRestoring(true)}
        onRestoreFailed={() => setRestoring(false)}
        onRestored={() => void onReload()}
      />
      <PauseDialog
        open={pauseOpen}
        onOpenChange={setPauseOpen}
        name={detail.name}
        inFlow={detail.inFlow}
        pending={pause.isPending}
        onPause={(reason) =>
          pause.mutate(
            { params: { path: { workspace: slug, id } }, body: reason ? { reason } : {} },
            {
              onSuccess: () => {
                setPauseOpen(false)
                toast({ title: t(mf.pausedToast, { name: detail.name }), detail: t(mf.pausedDetail) })
                refreshDetail()
              },
              onError: (e) => toast({ kind: 'error', title: t(mf.pauseFailed), detail: errorMessage(e) }),
            },
          )
        }
      />
      <EmailPreviewDialog email={preview} onClose={() => setPreview(null)} />
      <EmailEditDialog
        store={storeOfTrigger(graph)}
        email={editingEmail}
        onClose={() => setEditingEmail(null)}
        onSaved={(version) => {
          const target = editingEmail
          setEditingEmail(null)
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/options'] })
          if (!target) return
          setFlow((f) => ({
            ...f,
            nodes: f.nodes.map((n) => (n.id === target.node && n.data.email ? { ...n, data: { ...n.data, email: { ...n.data.email, emailVersion: version } } } : n)),
          }))
        }}
      />
    </div>
  )
}

/** "Live · v3 since 2 Oct", "Paused · v1", "Draft · not published yet". */
function StatusBadge({ detail }: { detail: Detail }) {
  const t = useT()
  if (detail.status === 'live' && detail.liveVersion !== undefined) {
    return (
      <AutomationStatusBadge status="live">
        {detail.publishedAt
          ? t(mf.liveSince, { version: detail.liveVersion, date: dayMonthDate(detail.publishedAt) })
          : t(mf.live, { version: detail.liveVersion })}
      </AutomationStatusBadge>
    )
  }
  if (detail.status === 'paused') {
    const by = detail.pausedBy
    const reason = detail.pauseReason ?? ''
    const why = by ? (reason ? t(mf.pausedByReason, { name: by, reason }) : t(mf.pausedBy, { name: by })) : reason
    const version = detail.liveVersion
    const since = detail.pausedAt ? dayMonthDate(detail.pausedAt) : undefined
    return (
      <span title={why || undefined}>
        <AutomationStatusBadge status="paused">
          {version !== undefined
            ? since
              ? t(mf.pausedSince, { version, date: since })
              : t(mf.pausedVersion, { version })
            : since
              ? t(mf.pausedSinceNoVersion, { date: since })
              : t(mf.paused)}
        </AutomationStatusBadge>
      </span>
    )
  }
  return <AutomationStatusBadge status="draft">{t(mf.draftNotPublished)}</AutomationStatusBadge>
}

function ZoomLevel() {
  const zoom = useStore((s) => s.transform[2])
  return <>{pct(Math.round(zoom * 100), 0)}</>
}

/** Right panel with no step selected: the automation's name and description. */
function FlowPanel({ detail, editable, issues }: { detail: Detail; editable: boolean; issues: Issue[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [name, setName] = useState(detail.name)
  const [description, setDescription] = useState(detail.description)
  const save = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/automations/{id}')
  const commit = () => {
    const n = name.trim()
    if (!n) {
      setName(detail.name)
      return
    }
    if (n === detail.name && description === detail.description) return
    save.mutate(
      { params: { path: { workspace: slug, id: detail.id } }, body: { name: n, description } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/{id}'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations'] })
        },
        onError: (e) => toast({ kind: 'error', title: t(mf.nameFailed), detail: errorMessage(e) }),
      },
    )
  }
  const flowIssues = issues.filter((i) => i.severity === 'error')
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">{t(mf.flowTitle)}</span>
        <span className="k">{t(mf.selectStep)}</span>
      </div>
      <Prop title={t(mf.name)} id="p-flowname">
        <input
          aria-labelledby="p-flowname"
          className="in"
          maxLength={120}
          disabled={!editable}
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') e.currentTarget.blur()
          }}
        />
      </Prop>
      <Prop title={t(mf.description)} id="p-flowdesc">
        <textarea aria-labelledby="p-flowdesc" className="in h-auto min-h-[72px] py-2" maxLength={500} disabled={!editable} value={description} onChange={(e) => setDescription(e.target.value)} onBlur={commit} />
      </Prop>
      <Prop title={t(mf.beforePublish)} id="p-flowissues">
        {flowIssues.length ? (
          <ul className="m-0 flex list-none flex-col gap-1 p-0 text-[12.5px]">
            {flowIssues.map((i, n) => (
              <li key={n} className="text-bad-fg">
                {i.message}
              </li>
            ))}
          </ul>
        ) : (
          <span className="k">{t(mf.allConnected)}</span>
        )}
      </Prop>
    </>
  )
}

const bold = (chunks: ReactNode[]) => <b className="n">{chunks}</b>

/** Right panel › Stats with no step selected: the flow's 30-day numbers. */
function FlowStatsPanel({ stats, inFlow, published }: { stats?: Schemas['AutomationStats']; inFlow: number; published: boolean }) {
  const t = useT()
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">{t(mf.flowStatsTitle)}</span>
        <span className="k">{published ? t(mf.selectStepStats) : t(mf.statsStart)}</span>
      </div>
      <div className="fe-prop">
        <div className="fe-stats">
          <span>
            <T msg={mf.entered} values={{ count: stats ? count(stats.entered) : '—', b: bold }} />
          </span>
          <span>
            <T msg={mf.inFlowNow} values={{ count: count(stats?.inFlow ?? inFlow), b: bold }} />
          </span>
        </div>
      </div>
    </>
  )
}

function EditorSkeleton() {
  const t = useT()
  return (
    <div className="fe rc-app" aria-busy="true">
      <span className="vh" role="status">
        {t(mf.loading)}
      </span>
      <header className="fe-head">
        <Skeleton w={32} h={32} r={7} />
        <div className="flex flex-col gap-1.5">
          <Skeleton w={64} h={10} />
          <Skeleton w={150} h={15} />
        </div>
        <Skeleton w={150} h={22} r={999} />
        <Skeleton w={240} h={22} r={999} />
        <div className="ml-auto flex gap-2">
          <Skeleton w={64} h={32} r={7} />
          <Skeleton w={118} h={32} r={7} />
          <Skeleton w={92} h={32} r={7} />
        </div>
      </header>
      <div className="fe-body">
        <aside className="fe-panel fe-left">
          <div className="flex flex-col gap-1.5 p-3">
            {Array.from({ length: 8 }, (_, i) => (
              <Skeleton key={i} w="100%" h={30} r={8} />
            ))}
          </div>
        </aside>
        <div className="fe-main">
          <div className="fe-tools">
            <Skeleton w={260} h={14} />
          </div>
          <div className="fe-canvas fe-dots" style={{ minHeight: 600 }} />
        </div>
        <aside className="fe-panel fe-right">
          <div className="flex flex-col gap-2 p-3">
            <Skeleton w={180} h={14} />
            <Skeleton w="100%" h={32} r={7} />
            <Skeleton w="100%" h={32} r={7} />
            <Skeleton w="70%" h={14} />
          </div>
        </aside>
      </div>
    </div>
  )
}

function EditorError({ error, onRetry, title }: { error: unknown; onRetry?: () => void; title?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = error instanceof ApiError ? error.status : 0
  let heading: ReactNode = title ?? t(mf.loadFailed)
  let text = `${errorMessage(error)} ${t(mf.loadFailedBody)}`
  if (status === 404) {
    heading = t(mf.notFoundTitle)
    text = t(mf.notFoundBody)
  } else if (status === 403) {
    heading = title ?? t(mf.forbiddenTitle)
    text = errorMessage(error)
  }
  return (
    <div className="fe rc-app">
      <header className="fe-head">
        <ButtonLink to="/w/$workspace/automations" params={{ workspace: slug }} iconOnly icon="chevronLeft" aria-label={t(mf.back)} />
        <h1 className="m-0 text-[15px] font-bold">{t(mf.flowTitle)}</h1>
      </header>
      <main id="main" className="p-4 sm:p-7">
        <ErrorPanel
          title={heading}
          actions={
            <>
              {onRetry && status !== 404 && status !== 403 ? (
                <Button variant="primary" onClick={onRetry}>
                  {t(common.retry)}
                </Button>
              ) : null}
              <ButtonLink to="/w/$workspace/automations" params={{ workspace: slug }}>
                {t(mf.back)}
              </ButtonLink>
            </>
          }
        >
          {text}
        </ErrorPanel>
      </main>
    </div>
  )
}
