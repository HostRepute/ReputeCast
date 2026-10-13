import { boardBoxes, welcomeRefs, welcomeV4 } from './fixtures.test-utils'
import { edgeGeometry, edgeKind, entrance, nodeById, nodeDetail, nodeTitle, serialize, timeline, triggerLabel, wouldLoop } from './graph'

describe('step titles (as on the AutomationFlow board)', () => {
  const g = welcomeV4()
  const title = (id: string) => nodeTitle(g, nodeById(g, id)!, welcomeRefs)
  it('names every step like the board', () => {
    expect(title('n1')).toBe('Weekly Digest')
    expect(title('n2')).toBe('Wait 10 minutes')
    expect(title('n3')).toBe('Welcome · step 1 "You\'re in"')
    expect(title('n4')).toBe('Opened step 1 within 2 days?')
    expect(title('n5')).toBe('Wait 2 days')
    expect(title('n6')).toBe('Step 2 "Three things to try first"')
    expect(title('n7')).toBe('50 / 50 on step 3')
    expect(title('n10')).toBe('Clicked a docs link')
    expect(title('n12')).toBe('"Did you get our welcome?" · plain text')
    expect(title('n13')).toBe('Leave the flow · stays on the list')
  })
  it('describes settings under the title', () => {
    expect(nodeDetail(nodeById(g, 'n1')!)).toBe('double opt-in confirmed only')
    expect(nodeDetail(nodeById(g, 'n4')!)).toBe('human opens only')
  })
  it('labels the trigger for the list', () => {
    expect(triggerLabel(nodeById(g, 'n1')!.data.trigger, welcomeRefs)).toBe('Joined list · Weekly Digest')
  })
})

describe('edge geometry', () => {
  it('drops straight when steps line up', () => {
    expect(edgeGeometry(482, 131, 482, 139, 'plain')).toEqual({ path: 'M482,131 L482,139', length: 8 })
  })
  it('routes a branch path along a bus 24px under the step, label at the drop', () => {
    const geo = edgeGeometry(482, 482, 262, 526, 'branch')
    expect(geo.busY).toBe(501)
    expect(geo.label).toEqual({ x: 276, y: 490 })
    expect(geo.path).toBe('M482,482 L482,501 L262,501 L262,526')
  })
  it('puts split buses 22px under the step and merges 18px above the target', () => {
    expect(edgeGeometry(262, 777, 142, 811, 'split')).toMatchObject({ busY: 794, label: { x: 150, y: 783 } })
    expect(edgeGeometry(142, 899, 262, 927, 'merge').busY).toBe(914)
  })
  it('classifies edges', () => {
    const g = welcomeV4()
    const kind = (s: string, t: string) => edgeKind(g, g.edges.find((e) => e.source === s && e.target === t)!)
    expect(kind('n4', 'n5')).toBe('branch')
    expect(kind('n7', 'n9')).toBe('split')
    expect(kind('n8', 'n10')).toBe('merge')
    expect(kind('n1', 'n2')).toBe('plain')
  })
})

describe('pulse timeline (the board keyframes)', () => {
  const g = welcomeV4()
  const tl = timeline(g, boardBoxes(g), new Map([['n4', new Map([['yes', 778], ['no', 498]])]]))
  const pulse = (key: string) => tl.pulses.find((p) => p.key === key)!
  it('rests 0.45 s in the trigger, then runs at 220 px/s', () => {
    // pu1: 2.5% → 3.409% of the board's 18 s loop.
    expect(pulse('n1-n2:0').start).toBeCloseTo(0.45, 3)
    expect(pulse('n1-n2:0').end).toBeCloseTo(0.6136, 3)
    // pu2 leaves the delay at 5.909%.
    expect(pulse('n2-n3:0').start).toBeCloseTo(1.0636, 3)
  })
  it('turns onto the branch bus like pu5', () => {
    expect(pulse('n4-n5:1').start).toBeCloseTo(2.4818, 3)
    expect(pulse('n4-n5:1').horizontal).toBe(true)
  })
  it('pulses the minor path fainter', () => {
    expect(pulse('n4-n5:0').opacity).toBe(1)
    expect(pulse('n4-n11:0').opacity).toBe(0.55)
  })
  it('loops every 9 s (the board repeats at 50% of 18 s)', () => {
    expect(tl.period).toBe(9)
  })
  it('glows every step once and flashes handles', () => {
    expect(tl.hits.map((h) => h.node).sort()).toEqual(g.nodes.map((n) => n.id).sort())
    expect(tl.flashIn.some((f) => f.node === 'n1')).toBe(false)
    expect(tl.flashOut.some((f) => f.node === 'n13')).toBe(false)
  })
})

describe('entrance', () => {
  it('staggers steps by depth like the board (.n1 0.1 s, .n2 0.25 s …)', () => {
    const e = entrance(welcomeV4())
    expect(e.nodes.get('n1')).toBeCloseTo(0.1)
    expect(e.nodes.get('n2')).toBeCloseTo(0.25)
    expect(e.nodes.get('n4')).toBeCloseTo(0.55)
    expect(e.nodes.get('n5')).toBeCloseTo(0.7)
    expect(e.nodes.get('n11')).toBeCloseTo(0.7)
    expect(e.nodes.get('n10')).toBeCloseTo(1.3)
  })
})

describe('graph helpers', () => {
  it('detects loops', () => {
    const g = welcomeV4()
    expect(wouldLoop(g, 'n6', 'n3')).toBe(true)
    expect(wouldLoop(g, 'n3', 'n6')).toBe(false)
  })
  it('serializes without write-only markers', () => {
    const g = serialize(
      [{ id: 'n1', type: 'webhook', position: { x: 1.4, y: 2.6 }, data: { webhook: { url: 'https://example.com', hasSecret: true } } }],
      [],
    )
    expect(g.nodes[0]).toEqual({ id: 'n1', type: 'webhook', position: { x: 1, y: 3 }, data: { webhook: { url: 'https://example.com' } } })
  })
})
