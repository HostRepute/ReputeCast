import { addStep, canConnect, connect, duplicateStep, edgeAt, fromGraph, healAfterDelete, insertIntoEdge, newEdgeId, toGraph, type FlowState } from './edit'
import { boardHeights, welcomeV4 } from './fixtures.test-utils'
import { newStepData } from './graph'

/** The board's flow with measured step sizes, as React Flow holds it. */
function board(): FlowState {
  const s = fromGraph(welcomeV4())
  return { ...s, nodes: s.nodes.map((n) => ({ ...n, measured: { width: 224, height: boardHeights[n.id] } })) }
}

const step = (id: string, type: Parameters<typeof newStepData>[0]) => ({ id, type, data: newStepData(type) })
const pos = (s: FlowState, id: string) => s.nodes.find((n) => n.id === id)!.position
const links = (s: FlowState) => s.edges.map((e) => `${e.source}>${e.target}${e.data?.path ? `:${e.data.path}` : ''}`).sort()

describe('edge ids', () => {
  it('are source-target, numbered when taken', () => {
    expect(newEdgeId([], 'n1', 'n2')).toBe('n1-n2')
    expect(newEdgeId([{ id: 'n1-n2' }, { id: 'n1-n2-2' }], 'n1', 'n2')).toBe('n1-n2-3')
  })
})

describe('dropping a step', () => {
  it('finds the edge under the pointer, including bus segments', () => {
    expect(edgeAt(board(), { x: 482, y: 135 })?.id).toBe('n1-n2')
    expect(edgeAt(board(), { x: 350, y: 503 })?.id).toBe('n4-n5')
    expect(edgeAt(board(), { x: 650, y: 503 })?.id).toBe('n4-n11')
    expect(edgeAt(board(), { x: 482, y: 300 })).toBeUndefined()
  })
  it('inserts into an edge: takes the lower step’s place and pushes the rest down', () => {
    const s = board()
    const next = addStep(s, step('n14', 'delay'), { at: { x: 482, y: 135 } })
    expect(pos(next, 'n14')).toEqual({ x: 370, y: 144 })
    // The delay is 53px tall plus the 18px gap.
    expect(pos(next, 'n2').y).toBe(144 + 71)
    expect(pos(next, 'n13').y).toBe(735 + 71)
    expect(links(next)).toContain('n1>n14')
    expect(links(next)).toContain('n14>n2')
    expect(links(next)).not.toContain('n1>n2')
  })
  it('keeps the path when inserting on a branch path, and a new fork continues on Yes', () => {
    const next = insertIntoEdge(board(), board().edges.find((e) => e.id === 'n4-n11')!, step('n14', 'branch'))
    expect(links(next)).toContain('n4>n14:no')
    expect(links(next)).toContain('n14>n11:yes')
    expect(pos(next, 'n14')).toEqual({ x: 590, y: 531 })
    // Only the No path moves.
    expect(pos(next, 'n5').y).toBe(531)
    expect(pos(next, 'n11').y).toBeGreaterThan(531)
  })
  it('places a goal where it is dropped, unconnected', () => {
    const next = addStep(board(), step('n14', 'goal'), { at: { x: 482, y: 135 } })
    expect(links(next)).toEqual(links(board()))
    expect(pos(next, 'n14')).toEqual({ x: 370, y: 135 - 53 / 2 })
  })
  it('snaps a free drop to the grid', () => {
    const next = addStep(board(), step('n14', 'delay'), { at: { x: 900, y: 300 }, snap: (p) => ({ x: Math.round(p.x / 20) * 20, y: Math.round(p.y / 20) * 20 }) })
    expect(pos(next, 'n14')).toEqual({ x: 780, y: 280 })
  })
})

describe('adding after the selected step', () => {
  it('inserts after a plain step', () => {
    const next = addStep(board(), step('n14', 'email'), { after: 'n12' })
    expect(links(next)).toContain('n12>n14')
    expect(links(next)).toContain('n14>n13')
  })
  it('goes in front of a goal or exit', () => {
    const next = addStep(board(), step('n14', 'delay'), { after: 'n13' })
    expect(links(next)).toContain('n12>n14')
    expect(links(next)).toContain('n14>n13')
  })
  it('continues a dangling step and fills a fork’s free path', () => {
    const s: FlowState = { nodes: board().nodes.filter((n) => n.id !== 'n11' && n.id !== 'n12' && n.id !== 'n13'), edges: board().edges.filter((e) => !['n4-n11', 'n11-n12', 'n12-n13'].includes(e.id)) }
    const next = addStep(s, step('n14', 'exit'), { after: 'n4' })
    expect(links(next)).toContain('n4>n14:no')
  })
  it('adds a missing trigger above the first step and connects it', () => {
    const s: FlowState = { nodes: board().nodes.filter((n) => n.id !== 'n1'), edges: board().edges.filter((e) => e.source !== 'n1') }
    const next = addStep(s, { id: 'n14', type: 'trigger', data: newStepData('trigger', 'tag_added') }, {})
    expect(links(next)).toContain('n14>n2')
    expect(pos(next, 'n14').y).toBeLessThan(144)
  })
})

describe('connecting steps', () => {
  const g = toGraph(board().nodes, board().edges)
  it('refuses loops, inputs on the trigger, outputs from goals and full forks', () => {
    expect(canConnect(g, 'n6', 'n3')).toBe(false)
    expect(canConnect(g, 'n5', 'n1')).toBe(false)
    expect(canConnect(g, 'n10', 'n13')).toBe(false)
    expect(canConnect(g, 'n4', 'n13')).toBe(false)
    expect(canConnect(g, 'n12', 'n10')).toBe(true)
  })
  it('replaces a plain step’s output', () => {
    const next = connect(board(), 'n12', 'n10')
    expect(links(next)).toContain('n12>n10')
    expect(links(next)).not.toContain('n12>n13')
  })
})

describe('deleting a step', () => {
  it('joins its parent to its next step on the same path', () => {
    const s = board()
    const gone = s.nodes.filter((n) => n.id === 'n11')
    const goneEdges = s.edges.filter((e) => e.source === 'n11' || e.target === 'n11')
    const remaining = s.edges.filter((e) => !goneEdges.includes(e))
    const heal = healAfterDelete(gone, goneEdges, remaining)
    expect(heal.map((e) => `${e.source}>${e.target}:${e.data?.path}`)).toEqual(['n4>n12:no'])
  })
  it('leaves a deleted fork’s paths to the editor', () => {
    const s = board()
    const goneEdges = s.edges.filter((e) => e.source === 'n7' || e.target === 'n7')
    expect(healAfterDelete(s.nodes.filter((n) => n.id === 'n7'), goneEdges, [])).toEqual([])
  })
})

describe('duplicating a step', () => {
  it('copies it to the right without its signing secret', () => {
    const s: FlowState = { nodes: [{ id: 'n1', type: 'webhook', position: { x: 10, y: 20 }, data: { webhook: { url: 'https://example.com', hasSecret: true } } }], edges: [] }
    const next = duplicateStep(s, 'n1', 'n2')
    expect(next.nodes[1]).toMatchObject({ id: 'n2', position: { x: 250, y: 20 }, data: { webhook: { url: 'https://example.com' } } })
    expect(next.nodes[1]!.data.webhook?.hasSecret).toBeUndefined()
  })
})
