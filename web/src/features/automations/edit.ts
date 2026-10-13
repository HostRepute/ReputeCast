import type { StepEdgeType } from './canvas/StepEdge'
import type { StepNodeType } from './canvas/StepNode'
import {
  edgeGeometry,
  edgeKind,
  isCompact,
  isFork,
  isTerminal,
  nodeById,
  nodePaths,
  outEdges,
  wouldLoop,
  type FlowGraph,
  type NodeData,
  type StepType,
} from './graph'

/*
 * Editing the flow on the canvas: React Flow state in, React Flow state out.
 * Pure functions so the rules (where a dropped step goes, how edges heal
 * when a step is deleted, which connections are allowed) are unit-tested.
 */

export type { StepEdgeType, StepNodeType }

export interface XY {
  x: number
  y: number
}

export interface NewStep {
  id: string
  type: StepType
  data: NodeData
}

export interface FlowState {
  nodes: StepNodeType[]
  edges: StepEdgeType[]
}

/** Steps are 224px wide (AutomationFlow board). */
export const NODE_W = 224
/** Vertical gap between a step and the next one on a straight edge. */
export const GAP = 18

export function nodeWidth(n: StepNodeType): number {
  return n.measured?.width ?? NODE_W
}

/** Measured height, or the board's height for a step without stats. */
export function nodeHeight(n: StepNodeType): number {
  return n.measured?.height ?? estimateHeight(n.type)
}

export function estimateHeight(type: StepType): number {
  return isCompact(type) ? 53 : 60
}

/** The flow as the canvas, side panels and API see it (secrets markers kept). */
export function toGraph(nodes: StepNodeType[], edges: StepEdgeType[]): FlowGraph {
  return {
    nodes: nodes.map((n) => ({ id: n.id, type: n.type, position: n.position, data: n.data })),
    edges: edges.map((e) => ({ id: e.id, source: e.source, target: e.target, data: e.data?.path ? { path: e.data.path } : {} })),
  }
}

export function fromGraph(g: FlowGraph): FlowState {
  return {
    nodes: g.nodes.map((n) => ({ id: n.id, type: n.type, position: { ...n.position }, data: n.data })),
    edges: g.edges.map((e) => edge(e.id, e.source, e.target, e.data.path)),
  }
}

function edge(id: string, source: string, target: string, path?: string): StepEdgeType {
  return { id, source, target, type: 'step', data: path ? { path } : {} }
}

/** "n3-n4", or "n3-n4-2" when that id is taken. */
export function newEdgeId(edges: { id: string }[], source: string, target: string): string {
  const base = `${source}-${target}`
  const taken = new Set(edges.map((e) => e.id))
  if (!taken.has(base)) return base
  let i = 2
  while (taken.has(`${base}-${i}`)) i++
  return `${base}-${i}`
}

/** Nodes reachable from id (id included). */
export function descendants(edges: { source: string; target: string }[], id: string): Set<string> {
  const seen = new Set<string>()
  const stack = [id]
  while (stack.length) {
    const cur = stack.pop()!
    if (seen.has(cur)) continue
    seen.add(cur)
    for (const e of edges) if (e.source === cur) stack.push(e.target)
  }
  return seen
}

/** The corner points an edge is drawn through (see edgeGeometry). */
export function edgePoints(s: StepNodeType, t: StepNodeType, kind: ReturnType<typeof edgeKind>): XY[] {
  const sx = s.position.x + nodeWidth(s) / 2
  const sy = s.position.y + nodeHeight(s) + 5
  const tx = t.position.x + nodeWidth(t) / 2
  const ty = t.position.y - 5
  const geo = edgeGeometry(sx, sy, tx, ty, kind)
  if (geo.busY === undefined) {
    return [
      { x: sx, y: sy },
      { x: tx, y: ty },
    ]
  }
  return [
    { x: sx, y: sy },
    { x: sx, y: geo.busY },
    { x: tx, y: geo.busY },
    { x: tx, y: ty },
  ]
}

function distToSegment(p: XY, a: XY, b: XY): number {
  const dx = b.x - a.x
  const dy = b.y - a.y
  const len2 = dx * dx + dy * dy
  const t = len2 === 0 ? 0 : Math.max(0, Math.min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / len2))
  return Math.hypot(p.x - (a.x + t * dx), p.y - (a.y + t * dy))
}

/** The edge under a point (flow coordinates), if any is within `tolerance`. */
export function edgeAt(state: FlowState, p: XY, tolerance = 12): StepEdgeType | undefined {
  const byId = new Map(state.nodes.map((n) => [n.id, n]))
  const g = toGraph(state.nodes, state.edges)
  let best: { e: StepEdgeType; d: number } | undefined
  for (const e of state.edges) {
    const s = byId.get(e.source)
    const t = byId.get(e.target)
    const ge = g.edges.find((x) => x.id === e.id)
    if (!s || !t || !ge) continue
    const pts = edgePoints(s, t, edgeKind(g, ge))
    for (let i = 1; i < pts.length; i++) {
      const d = distToSegment(p, pts[i - 1]!, pts[i]!)
      if (d <= tolerance && (!best || d < best.d)) best = { e, d }
    }
  }
  return best?.e
}

/** The first path of a fork step (Yes, A) that has no edge yet; null when all do. */
export function freePath(g: FlowGraph, source: string): string | undefined | null {
  const s = nodeById(g, source)
  if (!s || !isFork(s.type)) return undefined
  const used = new Set(outEdges(g, source).map((e) => e.data.path))
  return nodePaths(s).find((p) => !used.has(p)) ?? null
}

/**
 * Whether a drag from source's output to target's input may become an edge:
 * no loops, no input on the trigger, no output from a goal or exit, no
 * duplicate edges, and a fork needs a free path.
 */
export function canConnect(g: FlowGraph, source: string | null | undefined, target: string | null | undefined): boolean {
  if (!source || !target || source === target) return false
  const s = nodeById(g, source)
  const t = nodeById(g, target)
  if (!s || !t) return false
  if (isTerminal(s.type) || t.type === 'trigger') return false
  if (g.edges.some((e) => e.source === source && e.target === target)) return false
  if (wouldLoop(g, source, target)) return false
  if (isFork(s.type) && freePath(g, source) === null) return false
  return true
}

/**
 * Adds the edge for a new connection. A fork takes its next free path; a
 * plain step keeps a single output, so the new edge replaces the old one.
 */
export function connect(state: FlowState, source: string, target: string): FlowState {
  const g = toGraph(state.nodes, state.edges)
  if (!canConnect(g, source, target)) return state
  const s = nodeById(g, source)!
  const path = freePath(g, source) ?? undefined
  const rest = isFork(s.type) ? state.edges : state.edges.filter((e) => e.source !== source)
  return { nodes: state.nodes, edges: [...rest, edge(newEdgeId(rest, source, target), source, target, path)] }
}

/** The path a newly inserted fork sends the rest of the flow down. */
function firstPath(step: NewStep): string | undefined {
  if (step.type === 'branch') return 'yes'
  if (step.type === 'split') return step.data.variants?.[0]?.key ?? 'A'
  return undefined
}

/**
 * Puts a step on an edge: it takes the place of the edge's lower step, which
 * moves down with everything after it.
 */
export function insertIntoEdge(state: FlowState, on: StepEdgeType, step: NewStep): FlowState {
  const byId = new Map(state.nodes.map((n) => [n.id, n]))
  const s = byId.get(on.source)
  const t = byId.get(on.target)
  if (!s || !t) return state
  const h = estimateHeight(step.type)
  const position = { x: t.position.x + nodeWidth(t) / 2 - NODE_W / 2, y: t.position.y }
  const shift = descendants(state.edges, t.id)
  const nodes = state.nodes.map((n) => (shift.has(n.id) ? { ...n, position: { x: n.position.x, y: n.position.y + h + GAP } } : n))
  nodes.push({ id: step.id, type: step.type, position, data: step.data })
  const rest = state.edges.filter((e) => e.id !== on.id)
  const e1 = edge(newEdgeId(rest, s.id, step.id), s.id, step.id, on.data?.path)
  const e2 = edge(newEdgeId([...rest, e1], step.id, t.id), step.id, t.id, firstPath(step))
  return { nodes, edges: [...rest, e1, e2] }
}

function place(state: FlowState, step: NewStep, position: XY): FlowState {
  return { nodes: [...state.nodes, { id: step.id, type: step.type, position, data: step.data }], edges: state.edges }
}

/** Where a step goes when nothing says where: under the flow, in the trigger's column. */
function belowAll(state: FlowState, step: NewStep): XY {
  if (state.nodes.length === 0) return { x: 370, y: 24 }
  const anchor = state.nodes.find((n) => n.type === 'trigger') ?? state.nodes[0]!
  const bottom = Math.max(...state.nodes.map((n) => n.position.y + nodeHeight(n)))
  return { x: anchor.position.x + nodeWidth(anchor) / 2 - NODE_W / 2, y: bottom + (step.type === 'trigger' ? 60 : 40) }
}

/** A trigger added to a flow without one goes above its first step and feeds it. */
function addTrigger(state: FlowState, step: NewStep, at?: XY): FlowState {
  const hasInput = new Set(state.edges.map((e) => e.target))
  const roots = state.nodes.filter((n) => !hasInput.has(n.id))
  if (roots.length === 1 && !at) {
    const root = roots[0]!
    const h = estimateHeight('trigger')
    const next = place(state, step, { x: root.position.x + nodeWidth(root) / 2 - NODE_W / 2, y: root.position.y - h - GAP })
    return { nodes: next.nodes, edges: [...next.edges, edge(newEdgeId(next.edges, step.id, root.id), step.id, root.id)] }
  }
  return place(state, step, at ? { x: at.x - NODE_W / 2, y: at.y - estimateHeight('trigger') / 2 } : belowAll(state, step))
}

/**
 * Adds a step dropped at a point (onto an edge, it is inserted there) or
 * added from the palette after the selected step.
 */
export function addStep(state: FlowState, step: NewStep, where: { at?: XY; after?: string; tolerance?: number; snap?: (p: XY) => XY }): FlowState {
  if (step.type === 'trigger') return addTrigger(state, step, where.at)
  if (where.at) {
    const on = isTerminal(step.type) ? undefined : edgeAt(state, where.at, where.tolerance)
    if (on) return insertIntoEdge(state, on, step)
    const p = { x: where.at.x - NODE_W / 2, y: where.at.y - estimateHeight(step.type) / 2 }
    return place(state, step, where.snap ? where.snap(p) : p)
  }
  const s = where.after ? state.nodes.find((n) => n.id === where.after) : undefined
  if (!s) return place(state, step, belowAll(state, step))
  const g = toGraph(state.nodes, state.edges)
  if (isTerminal(s.type)) {
    // Nothing comes after a goal or an exit: the step goes in front of it.
    const into = state.edges.find((e) => e.target === s.id)
    return into ? insertIntoEdge(state, into, step) : place(state, step, belowAll(state, step))
  }
  const outs = state.edges.filter((e) => e.source === s.id)
  if (outs.length > 0) {
    const order = nodePaths(nodeById(g, s.id)!)
    const first = [...outs].sort((a, b) => order.indexOf(a.data?.path ?? '') - order.indexOf(b.data?.path ?? ''))[0]!
    if (!isTerminal(step.type)) return insertIntoEdge(state, first, step)
  }
  const path = freePath(g, s.id)
  const position = { x: s.position.x + nodeWidth(s) / 2 - NODE_W / 2, y: s.position.y + nodeHeight(s) + (isFork(s.type) ? 54 : GAP) }
  if (path === null || (outs.length > 0 && !isFork(s.type))) return place(state, step, { x: position.x + NODE_W + 16, y: position.y })
  const next = place(state, step, position)
  return { nodes: next.nodes, edges: [...next.edges, edge(newEdgeId(next.edges, s.id, step.id), s.id, step.id, path)] }
}

/**
 * Edges that keep the flow connected after steps are deleted: the step's
 * parents now lead to its single next step (on the same path). Forks are
 * not healed; which path to keep is the editor's call.
 */
export function healAfterDelete(deleted: StepNodeType[], deletedEdges: StepEdgeType[], remaining: StepEdgeType[]): StepEdgeType[] {
  const gone = new Set(deleted.map((n) => n.id))
  const out: StepEdgeType[] = []
  for (const n of deleted) {
    if (isFork(n.type)) continue
    const ins = deletedEdges.filter((e) => e.target === n.id && !gone.has(e.source))
    const outs = deletedEdges.filter((e) => e.source === n.id && !gone.has(e.target))
    if (outs.length !== 1) continue
    const target = outs[0]!.target
    for (const i of ins) {
      const all = [...remaining, ...out]
      if (all.some((e) => e.source === i.source && e.target === target)) continue
      out.push(edge(newEdgeId(all, i.source, target), i.source, target, i.data?.path))
    }
  }
  return out
}

/** A copy of a step to the right of it, unconnected; secrets are not copied. */
export function duplicateStep(state: FlowState, id: string, newId: string): FlowState {
  const s = state.nodes.find((n) => n.id === id)
  if (!s) return state
  let data = structuredClone(s.data)
  if (data.webhook) {
    const w = { ...data.webhook }
    delete w.hasSecret
    data = { ...data, webhook: w }
  }
  return place(state, { id: newId, type: s.type, data }, { x: s.position.x + NODE_W + 16, y: s.position.y })
}
