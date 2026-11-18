import { act, renderHook } from '@testing-library/react'
import { buildItems, mcpOf } from './Conversation'
import { streamBlocks } from './Markdown'
import { initialThreadState, threadReducer, type Confirmation, type StreamEvent, type Thread, type ThreadMessage, type ThreadState, type ThreadTurn } from './reducer'
import { problemOf } from './states'
import { useThreadState } from './useThreadStream'

const at = '2026-11-17T10:00:00Z'
const turn = (state: ThreadTurn['state'], extra: Partial<ThreadTurn> = {}): ThreadTurn => ({ id: 't1', sequence: 1, state, createdAt: at, ...extra })
const msg = (id: string, role: ThreadMessage['role'], text: string, extra: Partial<ThreadMessage> = {}): ThreadMessage => ({ id, role, text, createdAt: at, turnId: 't1', ...extra })
const usage = { costMicros: 0, requests: 0, tokens: 0, toolCalls: 0, uncertainRequests: 0, uncertainTokens: 0, uncertainCostMicros: 0 }
const thread = (extra: Partial<Thread> = {}): Thread => ({
  id: 'th1',
  title: 'Why opens fell',
  state: 'open',
  screen: 'campaign.report',
  eventSeq: 2,
  createdAt: at,
  lastActivityAt: at,
  expiresAt: at,
  limits: { toolCalls: 40, tokens: 300000, costMicros: 2000000 },
  usage,
  contexts: [],
  mcpServerIds: [],
  messages: [msg('u1', 'user', 'Why did opens fall?')],
  turn: turn('queued'),
  ...extra,
})

const run = (s: ThreadState, ...events: StreamEvent[]) => threadReducer(s, { type: 'events', events })
const snap = (id: number, th = thread()): StreamEvent => ({ type: 'snapshot', id, data: { thread: th } })
const turnEv = (id: number, t: ThreadTurn): StreamEvent => ({ type: 'turn', id, data: { turn: t } })
const delta = (id: number, text: string): StreamEvent => ({ type: 'delta', id, data: { turnId: 't1', text } })
const message = (id: number, mm: ThreadMessage): StreamEvent => ({ type: 'message', id, data: { message: mm } })
const tool = (id: number, callId: string): StreamEvent => ({ type: 'tool', id, data: { turnId: 't1', tool: { callId, name: 'read_report', ok: true, ms: 300, label: 'September Product Update' } } })

describe('threadReducer', () => {
  it('builds an answer from a snapshot, its deltas and its message', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Opens '), delta(5, 'fell.'))
    expect(s.live).toEqual({ turnId: 't1', text: 'Opens fell.' })
    expect(s.lastId).toBe(5)
    s = run(s, message(6, msg('a1', 'assistant', 'Opens fell.')), turnEv(7, turn('done')))
    expect(s.live).toBeUndefined()
    expect(s.messages.map((x) => x.id)).toEqual(['u1', 'a1'])
    expect(s.turn?.state).toBe('done')
  })

  it('skips events it already applied after a reconnection', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Opens '))
    // The resumed stream starts after 4; a repeat of 4 and earlier is ignored.
    s = run(s, delta(4, 'Opens '), delta(3, 'x'), delta(5, 'fell.'))
    expect(s.live?.text).toBe('Opens fell.')
  })

  it('holds tool rows until the tool message has them, once each', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), tool(4, 'c1'), tool(5, 'c1'), tool(6, 'c2'))
    expect(s.tools.map((x) => x.callId)).toEqual(['c1', 'c2'])
    s = run(s, message(7, msg('tm', 'tool', '', { tools: [s.tools[0]!, s.tools[1]!] })))
    expect(s.tools).toEqual([])
    expect(s.toolsTurn).toBeUndefined()
  })

  it('keeps what a stopped turn wrote, which the server doesn’t save', () => {
    const s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Here are subjects:'), turnEv(5, turn('stopped', { reason: 'stopped' })))
    expect(s.live).toBeUndefined()
    expect(s.stopped).toEqual({ turnId: 't1', text: 'Here are subjects:' })
  })

  it('keeps what a stopped turn wrote when Stop’s response comes before the turn event', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Here are'))
    s = threadReducer(s, { type: 'thread', thread: thread({ eventSeq: 5, turn: turn('stopped', { reason: 'stopped' }) }) })
    expect(s.stopped).toEqual({ turnId: 't1', text: 'Here are' })
    s = run(s, turnEv(5, turn('stopped', { reason: 'stopped' })))
    expect(s.stopped?.text).toBe('Here are')
  })

  it('shows a stopped answer once when Stop’s response already has it saved', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Here are'))
    const saved = msg('a1', 'assistant', 'Here are', { partial: true })
    s = threadReducer(s, { type: 'thread', thread: thread({ eventSeq: 6, turn: turn('stopped', { reason: 'stopped' }), messages: [msg('u1', 'user', 'Why did opens fall?'), saved] }) })
    expect(s.stopped).toBeUndefined()
    // The stream's message and turn events (5, 6) come after the response, and are skipped.
    s = run(s, message(5, saved), turnEv(6, turn('stopped', { reason: 'stopped' })))
    const answer = buildItems(s).find((x) => x.kind === 'answer')
    expect(answer && answer.kind === 'answer' ? [answer.texts.map((x) => x.text), answer.stopped] : []).toEqual([['Here are'], undefined])
  })

  it('drops the kept copy when the stopped answer’s saved message comes after its turn event', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Here are'), turnEv(5, turn('stopped', { reason: 'stopped' })))
    expect(s.stopped?.text).toBe('Here are')
    s = run(s, message(6, msg('a1', 'assistant', 'Here are', { partial: true })))
    expect(s.stopped).toBeUndefined()
  })

  it('takes an API response only when it is as new as the stream', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), message(4, msg('a1', 'assistant', 'Done.')))
    // A response read before event 4 must not take the answer away.
    s = threadReducer(s, { type: 'thread', thread: thread({ eventSeq: 3, title: 'Renamed' }) })
    expect(s.messages.map((x) => x.id)).toEqual(['u1', 'a1'])
    expect(s.thread?.title).toBe('Renamed')
    s = threadReducer(s, { type: 'thread', thread: thread({ eventSeq: 9, messages: [], turn: undefined }) })
    expect(s.messages).toEqual([])
    expect(s.lastId).toBe(9)
  })

  it('shows the person’s message until the server has it', () => {
    let s = threadReducer(run(initialThreadState, snap(2)), { type: 'pending', text: 'And clicks?' })
    expect(buildItems(s).at(-2)).toMatchObject({ kind: 'user', text: 'And clicks?', pending: true })
    s = run(s, message(3, msg('u2', 'user', 'And clicks?', { turnId: 't2' })))
    expect(s.pending).toBeUndefined()
  })

  it('ends on a problem event', () => {
    const s = run(initialThreadState, snap(2), { type: 'problem', id: 0, data: { status: 409, code: 'ai_assistant_off', detail: 'off' } })
    expect(s.problem?.code).toBe('ai_assistant_off')
  })

  const conf = (state: Confirmation['state'], extra: Partial<Confirmation> = {}): Confirmation => ({
    id: 'c1',
    turnId: 't1',
    serverId: 's1',
    serverName: 'Linear',
    tool: 'create_issue',
    modelName: 'mcp__linear__create_issue',
    arguments: { title: 'Compare subjects' },
    state,
    createdAt: at,
    ...extra,
  })
  const confEv = (id: number, c: Confirmation): StreamEvent => ({ type: 'confirmation', id, data: { confirmation: c } })

  it('keeps the call a waiting turn asks about, then its decision until the turn goes on', () => {
    let s = run(initialThreadState, snap(2), turnEv(3, turn('answering')), confEv(4, conf('pending')), turnEv(5, turn('waiting')))
    expect(s.confirmation?.state).toBe('pending')
    expect(s.turn?.state).toBe('waiting')
    s = run(s, confEv(6, conf('allowed', { decision: 'allow' })), turnEv(7, turn('queued')))
    expect(s.confirmation?.state).toBe('allowed')
    // The next turn has nothing to confirm.
    s = run(s, turnEv(8, turn('done')), turnEv(9, turn('queued', { id: 't2', sequence: 2 })))
    expect(s.confirmation).toBeUndefined()
  })

  it('takes the call from a snapshot of a waiting thread, and drops it when the turn ended unanswered', () => {
    let s = run(initialThreadState, snap(2, thread({ turn: turn('waiting'), confirmation: conf('pending') })))
    expect(s.confirmation?.id).toBe('c1')
    // Stopped while it waited: the call is closed.
    s = run(s, turnEv(3, turn('stopped')))
    expect(s.confirmation).toBeUndefined()
    // A thread read again after a decision keeps the decided call of its turn.
    s = run(initialThreadState, snap(2, thread({ turn: turn('waiting'), confirmation: conf('pending') })), confEv(3, conf('denied', { decision: 'deny' })))
    s = threadReducer(s, { type: 'thread', thread: thread({ eventSeq: 4, turn: turn('queued') }) })
    expect(s.confirmation?.state).toBe('denied')
  })
})

describe('buildItems', () => {
  it('puts a turn’s tool rows and text in one answer, the live text last', () => {
    const s = run(
      initialThreadState,
      snap(2),
      turnEv(3, turn('answering')),
      tool(4, 'c1'),
      message(5, msg('tm', 'tool', '', { tools: [{ callId: 'c1', name: 'read_report', ok: true, ms: 300 }] })),
      delta(6, 'Opens fell'),
    )
    const items = buildItems(s)
    expect(items.map((i) => i.kind)).toEqual(['user', 'answer'])
    const answer = items[1]!
    expect(answer.kind === 'answer' && answer.tools.map((x) => x.callId)).toEqual(['c1'])
    expect(answer.kind === 'answer' && answer.live).toBe('Opens fell')
    expect(answer.kind === 'answer' && answer.status).toBe('answering')
  })

  it('marks where earlier messages were summarized', () => {
    const s = run(initialThreadState, snap(2, thread({ messages: [msg('s1', 'summary', 'Earlier…'), msg('u1', 'user', 'Next?')], turn: turn('done') })))
    expect(buildItems(s).map((i) => i.kind)).toEqual(['summary', 'user'])
  })
})

describe('streamBlocks', () => {
  it('splits a streaming answer at blank lines, never inside a code fence', () => {
    const text = 'First paragraph.\n\n```liquid\nHi {{ name }}\n\nBye\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nLast, still arriv'
    expect(streamBlocks(text)).toEqual(['First paragraph.\n\n', '```liquid\nHi {{ name }}\n\nBye\n```\n\n', '| a | b |\n|---|---|\n| 1 | 2 |\n\n', 'Last, still arriv'])
  })
})

describe('mcpOf', () => {
  it('labels an MCP row "Server · tool" from its label, else from its key', () => {
    expect(mcpOf({ name: 'mcp__linear__create_issue', label: 'Linear · create_issue' })).toEqual({ key: 'linear', server: 'Linear', tool: 'create_issue', label: 'Linear · create_issue' })
    expect(mcpOf({ name: 'mcp__wiki__search' })?.label).toBe('wiki · search')
    expect(mcpOf({ name: 'read_report', label: 'September' })).toBeUndefined()
    // A name the model can't take is sanitized in its tool name, never in the label.
    expect(mcpOf({ name: 'mcp__wiki__note_b_', label: '<i>Wiki</i> · note<b>' })).toMatchObject({ server: '<i>Wiki</i>', tool: 'note<b>' })
  })
})

describe('useThreadState', () => {
  it('holds deltas that only lengthen the answer, and applies them before anything else', () => {
    const { result } = renderHook(() => useThreadState())
    const send = (...events: StreamEvent[]) => act(() => result.current[1]({ type: 'events', events }))
    send(snap(2), turnEv(3, turn('answering')), delta(4, 'Opens '))
    const shown = result.current[0]
    expect(shown.live?.text).toBe('Opens ')
    const heard: string[] = []
    const off = result.current[2].subscribe(() => heard.push(result.current[2].get()?.text ?? ''))
    send(delta(5, 'fell '))
    send(delta(6, 'by 4%.'))
    // The panel's state didn't change; only the live text did.
    expect(result.current[0]).toBe(shown)
    expect(heard).toEqual(['Opens fell ', 'Opens fell by 4%.'])
    // The next other event comes after them, as if each had been dispatched.
    send(message(7, msg('a1', 'assistant', 'Opens fell by 4%.')), turnEv(8, turn('done')))
    const direct = run(initialThreadState, snap(2), turnEv(3, turn('answering')), delta(4, 'Opens '), delta(5, 'fell '), delta(6, 'by 4%.'), message(7, msg('a1', 'assistant', 'Opens fell by 4%.')), turnEv(8, turn('done')))
    expect(result.current[0]).toEqual(direct)
    expect(result.current[2].get()).toBeUndefined()
    off()
  })

  it('applies held deltas before a stop, so the stopped text is whole', () => {
    const { result } = renderHook(() => useThreadState())
    const send = (...events: StreamEvent[]) => act(() => result.current[1]({ type: 'events', events }))
    send(snap(2), turnEv(3, turn('answering')), delta(4, 'Opens '))
    send(delta(5, 'fell'))
    send(turnEv(6, turn('stopped')))
    expect(result.current[0].stopped).toEqual({ turnId: 't1', text: 'Opens fell' })
  })
})

describe('problemOf', () => {
  it('tells each model problem and an expired call apart, as reasons and as refused messages’ codes', () => {
    const cases: [string, string][] = [
      ['model_gone', 'modelGone'],
      ['assistant_model_gone', 'modelGone'],
      ['account_disconnected', 'disconnected'],
      ['key_rejected', 'key'],
      ['tools_unchecked', 'toolsUnchecked'],
      ['personal_keys_off', 'personalKeysOff'],
      ['not_configured', 'notConfigured'],
      ['confirmation_expired', 'confirmationExpired'],
      ['something_new', 'failed'],
    ]
    for (const [reason, problem] of cases) expect(problemOf(reason), reason).toBe(problem)
  })
})
