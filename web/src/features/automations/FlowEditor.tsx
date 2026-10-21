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
import { count, dayMonthDate } from '@/lib/format'
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
  triggerKindLabels,
  typeNames,
  type FlowGraph,
  type NodeData,
  type StepType,
} from './graph'
import { DRAG_TYPE, draggedStepType, FlowStatsList, graphOrder, Palette, Prop, StepPanel, StepStatsPanel, type PaletteItem } from './panels'
import { storeOfTrigger, useStoreTriggerOptions } from '@/features/commerce/automation'
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
const EDIT_REASON = 'Editing automations needs the automations:edit permission.'

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
  if (reason) return <EditorError error={new ApiError(403, { status: 403, title: 'Not allowed', detail: reason } as Schemas['Problem'])} />
  if (create.isError) return <EditorError error={create.error} title="Couldn't create the automation" />
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
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const rf = useReactFlow<StepNodeType, StepEdgeType>()
  const reduced = useReducedMotion()
  const coarse = useMediaQuery('(pointer: coarse)')
  const canEdit = useCan('automations:edit')
  const publishReason = usePermissionReason('automations:publish')
  const id = detail.id
  const published = detail.liveVersion !== undefined
  useDocumentTitle(`${detail.name} · Automations · ${me.workspace.name} · ReputeCast`)

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
    if (l.stopped) throw new Error('The draft is not saved.')
    if (l.json === l.savedJson) return
    if (!(await save(l.json))) throw new Error("Couldn't save the draft.")
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
            () => toast({ title: `Draft v${meta.draftVersion} saved`, detail: 'Publish it when it looks right.' }),
            (e: unknown) => toast({ kind: 'error', title: "Couldn't save the draft", detail: errorMessage(e) }),
          )
        }
      : undefined,
  )
  // If the session ends while editing, the notices say the work is safe.
  useUnsavedWork(
    canEdit
      ? { whileEnding: 'the draft saves itself', afterEnded: `"${detail.name}" is saved as draft v${meta.draftVersion} and will reopen where you left off.` }
      : undefined,
  )

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
    [graph, refs, statsMap, stats.data, published, showStats, liveIds, publishedIds, detail.status, meta.draftVersion, issueIds, detail.route.fallback?.id, entranceTimes],
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
    () => flow.nodes.map((n) => ({ ...n, selected: n.id === selectedId, ariaLabel: `${typeNames[n.type]}: ${titles.get(n.id) ?? ''}` })),
    [flow.nodes, selectedId, titles],
  )
  const rfEdges = useMemo(
    () =>
      flow.edges.map((e) => ({
        ...e,
        className: e.id === dropEdge ? 'fe-drop' : undefined,
        ariaLabel: `${titles.get(e.source) ?? e.source} to ${titles.get(e.target) ?? e.target}${e.data?.path ? `, ${pathLabel(e.data.path)} path` : ''}`,
      })),
    [flow.edges, dropEdge, titles],
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
          toast({ title: `Trigger changed to ${triggerKindLabels[kind]}`, detail: 'A flow has one trigger; its other settings start fresh.' })
          return
        }
      }
      const step = { id: nextNodeId(graph), type: item.type, data: newStepData(item.type, item.trigger) }
      const tolerance = 12 / rf.getZoom()
      setFlow((f) => addStep(f, step, { ...where, tolerance, snap: snap ? snapPoint : undefined }))
      select(step.id)
    },
    [editable, graph, rf, snap, select],
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
      toast({ kind: 'error', title: "Couldn't save the draft before publishing", detail: errorMessage(e) })
    } finally {
      setOpening(false)
    }
  }
  const nothingToPublish = published && meta.changes === 0 && !dirty ? `Nothing to publish: the draft is the same as v${detail.liveVersion}.` : undefined
  const publishBlocked = publishReason ?? (conflict ? 'Load the latest draft first.' : nothingToPublish)

  const order = useMemo(() => graphOrder(graph), [graph])
  const zoomKey = isMac ? '⌘' : 'Ctrl'

  return (
    <div className="fe rc-app">
      <a href="#main" className="skip">
        Skip to content
      </a>
      <header className="fe-head">
        <ButtonLink to="/w/$workspace/automations" params={{ workspace: slug }} iconOnly icon="chevronLeft" aria-label="Back to automations" />
        <div className="flex min-w-0 flex-col">
          <span className="k text-[11px]">Automation</span>
          <h1 className="m-0 text-[15px] font-bold leading-[1.45] tracking-normal">{detail.name}</h1>
        </div>
        <StatusBadge detail={detail} />
        {published && meta.changes > 0 ? <Badge tone="warn">Unpublished changes · {meta.changes}</Badge> : null}
        <Badge tone={detail.route.healthy ? 'purple' : 'bad'} className="ml-1.5">
          <Dot color={detail.route.healthy ? 'var(--chart-ses)' : 'currentColor'} />
          Automations route · {detail.route.label}
        </Badge>
        {failed ? (
          <span className="flex items-center gap-2" role="alert">
            <Badge tone="bad">Draft not saved</Badge>
            <Button size="sm" onClick={() => void save(json)}>
              Retry
            </Button>
          </span>
        ) : null}
        <span className="vh" role="status">
          {saving ? 'Saving the draft' : dirty ? '' : 'Draft saved'}
        </span>
        <div className="ml-auto flex flex-wrap gap-2">
          {detail.status === 'live' ? (
            <Button disabledReason={publishReason} onClick={() => setPauseOpen(true)}>
              Pause
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
                      toast({ title: `"${detail.name}" resumed`, detail: 'New contacts enter again and waiting steps continue.' })
                      refreshDetail()
                    },
                    onError: (e) => toast({ kind: 'error', title: "Couldn't resume", detail: errorMessage(e) }),
                  },
                )
              }
            >
              Resume
            </Button>
          ) : null}
          <Button onClick={() => setHistoryOpen(true)}>Version history</Button>
          <Button variant="primary" loading={opening} disabledReason={publishBlocked} onClick={() => void openPublish()}>
            Publish v{meta.draftVersion}
          </Button>
        </div>
      </header>

      {conflict ? (
        <div className="px-4 pt-3">
          <Banner tone="warn" title="This draft changed somewhere else" actions={<Button onClick={() => void onReload()}>Load the latest draft</Button>}>
            Someone saved a newer version of this draft, maybe you in another tab. Your last change here wasn't saved; load the latest draft to keep editing.
          </Banner>
        </div>
      ) : null}

      <div className="fe-body">
        <aside className="fe-panel fe-left" aria-label="Steps">
          <PanelTabs
            value={leftTab}
            onChange={setLeftTab}
            label="Steps and stats"
            className="flex flex-1 flex-col"
            items={[
              { value: 'steps', label: 'Steps' },
              { value: 'stats', label: 'Stats' },
            ]}
          >
            <TabPanel value="steps" className="flex flex-1 flex-col">
              <Palette disabledReason={!canEdit ? EDIT_REASON : !editable ? 'Load the latest draft first.' : undefined} onAdd={(item) => add(item, { after: selectedId })} />
              <div className="k mt-auto border-t border-line-2 px-3 py-2.5">
                {canEdit ? `Drag a step onto the canvas or onto an edge. Snap to grid is ${snap ? 'on' : 'off'}.` : 'You can look around; editing needs the automations:edit permission.'}
              </div>
            </TabPanel>
            <TabPanel value="stats" className="flex flex-1 flex-col">
              {published ? (
                <FlowStatsList graph={graph} order={order} refs={refs} stats={statsMap} selected={selectedId} onSelect={select} />
              ) : (
                <div className="k px-3 py-2.5">Stats start when the automation is published.</div>
              )}
            </TabPanel>
          </PanelTabs>
        </aside>

        <main id="main" className="fe-main">
          <div className="fe-tools">
            <span className="k">
              {published ? `Live stats · last 30 days${stats.data ? ` · ${count(stats.data.entered)} entered` : ''}` : 'No stats until the first publish'}
            </span>
            <label className="ml-2">
              <input type="checkbox" checked={showStats} onChange={(e) => setShowStats(e.target.checked)} />
              Show stats on nodes
            </label>
            <label>
              <input type="checkbox" checked={snap} onChange={(e) => setSnap(e.target.checked)} />
              Snap to grid
            </label>
            <label title={reduced ? 'Your system asks for reduced motion, so the flow stays still.' : undefined}>
              <input
                type="checkbox"
                checked={animate}
                onChange={(e) => {
                  setAnimate(e.target.checked)
                  writeAnimatePref(e.target.checked)
                }}
              />
              Animate flow
            </label>
            <span className="k ml-auto">
              Zoom <ZoomLevel /> · scroll to pan · {zoomKey} + wheel to zoom
            </span>
          </div>

          <div ref={canvasRef} className="fe-canvas" style={{ minHeight }} role="group" aria-label="Flow canvas">
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
                  <span className="font-semibold">Start with a trigger</span>
                  <span className="k">{canEdit ? 'Drag one from the left, or press it to place it here.' : 'This flow has no steps yet.'}</span>
                </div>
              </div>
            ) : null}
          </div>
        </main>

        <aside className="fe-panel fe-right" aria-label="Step settings">
          <PanelTabs
            value={rightTab}
            onChange={showTab}
            label="Step settings and stats"
            className="flex flex-1 flex-col"
            items={[
              { value: 'step', label: 'Step' },
              { value: 'stats', label: 'Stats' },
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
                  editReason={!canEdit ? EDIT_REASON : 'Load the latest draft first.'}
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

      <UnsavedChangesGuard
        dirty={Boolean(failed) && dirty}
        description="Your last changes to this flow couldn't be saved. Leave without them, or try saving again."
        onSave={flush}
      />
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
                toast({ title: `"${detail.name}" paused`, detail: 'Contacts stay where they are until you resume.' })
                refreshDetail()
              },
              onError: (e) => toast({ kind: 'error', title: "Couldn't pause", detail: errorMessage(e) }),
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
  if (detail.status === 'live' && detail.liveVersion !== undefined) {
    return (
      <AutomationStatusBadge status="live">
        Live · v{detail.liveVersion}
        {detail.publishedAt ? ` since ${dayMonthDate(detail.publishedAt)}` : ''}
      </AutomationStatusBadge>
    )
  }
  if (detail.status === 'paused') {
    const why = [detail.pausedBy ? `by ${detail.pausedBy}` : '', detail.pauseReason ?? ''].filter(Boolean).join(': ')
    return (
      <span title={why || undefined}>
        <AutomationStatusBadge status="paused">
          Paused{detail.liveVersion !== undefined ? ` · v${detail.liveVersion}` : ''}
          {detail.pausedAt ? ` since ${dayMonthDate(detail.pausedAt)}` : ''}
        </AutomationStatusBadge>
      </span>
    )
  }
  return <AutomationStatusBadge status="draft">Draft · not published yet</AutomationStatusBadge>
}

function ZoomLevel() {
  const zoom = useStore((s) => s.transform[2])
  return <>{Math.round(zoom * 100)}%</>
}

/** Right panel with no step selected: the automation's name and description. */
function FlowPanel({ detail, editable, issues }: { detail: Detail; editable: boolean; issues: Issue[] }) {
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
        onError: (e) => toast({ kind: 'error', title: "Couldn't save the name", detail: errorMessage(e) }),
      },
    )
  }
  const flowIssues = issues.filter((i) => i.severity === 'error')
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">Automation</span>
        <span className="k">Select a step on the canvas to see its settings.</span>
      </div>
      <Prop title="Name" id="p-flowname">
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
      <Prop title="Description" id="p-flowdesc">
        <textarea aria-labelledby="p-flowdesc" className="in h-auto min-h-[72px] py-2" maxLength={500} disabled={!editable} value={description} onChange={(e) => setDescription(e.target.value)} onBlur={commit} />
      </Prop>
      <Prop title="Before you publish" id="p-flowissues">
        {flowIssues.length ? (
          <ul className="m-0 flex list-none flex-col gap-1 p-0 text-[12.5px]">
            {flowIssues.map((i, n) => (
              <li key={n} className="text-bad-fg">
                {i.message}
              </li>
            ))}
          </ul>
        ) : (
          <span className="k">Every step is connected and set up.</span>
        )}
      </Prop>
    </>
  )
}

/** Right panel › Stats with no step selected: the flow's 30-day numbers. */
function FlowStatsPanel({ stats, inFlow, published }: { stats?: Schemas['AutomationStats']; inFlow: number; published: boolean }) {
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">Flow · last 30 days</span>
        <span className="k">{published ? 'Select a step for its own numbers.' : 'Stats start when the automation is published.'}</span>
      </div>
      <div className="fe-prop">
        <div className="fe-stats">
          <span>
            Entered <b className="n">{stats ? count(stats.entered) : '—'}</b>
          </span>
          <span>
            In the flow now <b className="n">{count(stats?.inFlow ?? inFlow)}</b>
          </span>
        </div>
      </div>
    </>
  )
}

function EditorSkeleton() {
  return (
    <div className="fe rc-app" aria-busy="true">
      <span className="vh" role="status">
        Loading the automation
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
  const { slug } = useWorkspace()
  const status = error instanceof ApiError ? error.status : 0
  let heading: ReactNode = title ?? "Couldn't load the automation"
  let text = `${errorMessage(error)} Your flows keep running; nothing was changed.`
  if (status === 404) {
    heading = "This automation doesn't exist"
    text = 'It may have been deleted, or the link is wrong. Nothing was changed.'
  } else if (status === 403) {
    heading = title ?? "You can't open this automation"
    text = errorMessage(error)
  }
  return (
    <div className="fe rc-app">
      <header className="fe-head">
        <ButtonLink to="/w/$workspace/automations" params={{ workspace: slug }} iconOnly icon="chevronLeft" aria-label="Back to automations" />
        <h1 className="m-0 text-[15px] font-bold">Automation</h1>
      </header>
      <main id="main" className="p-4 sm:p-7">
        <ErrorPanel
          title={heading}
          actions={
            <>
              {onRetry && status !== 404 && status !== 403 ? (
                <Button variant="primary" onClick={onRetry}>
                  Retry
                </Button>
              ) : null}
              <ButtonLink to="/w/$workspace/automations" params={{ workspace: slug }}>
                Back to automations
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
