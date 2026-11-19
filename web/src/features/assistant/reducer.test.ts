import { buildItems } from './Conversation'
import { streamBlocks } from './Markdown'
import { initialThreadState, threadReducer, type StreamEvent, type Thread, type ThreadMessage, type ThreadState, type ThreadTurn } from './reducer'

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
