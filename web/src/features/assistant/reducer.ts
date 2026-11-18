import type { Schemas } from '@/api/client'

/**
 * A thread as the panel shows it, kept from its server-sent events
 * (stream-ai-thread): the snapshot first, then each turn, delta, tool,
 * message and usage event in order. Pure, so the stream hook can batch a
 * frame's events into one update and tests can replay a stream.
 */
export type Thread = Schemas['AIThread']
export type ThreadMessage = Schemas['AIThreadMessage']
export type ThreadTurn = Schemas['AIThreadTurn']
export type ToolRow = Schemas['AIThreadToolRow']
export type ThreadUsage = Schemas['AIThreadUsage']
export type ThreadContext = Schemas['AIThreadContext']
export type StreamProblem = Schemas['AIThreadStreamProblem']
export type Confirmation = Schemas['AIThreadConfirmation']

export type StreamEvent =
  | { type: 'snapshot'; id: number; data: Schemas['AIThreadSnapshotEvent'] }
  | { type: 'turn'; id: number; data: Schemas['AIThreadTurnEvent'] }
  | { type: 'delta'; id: number; data: Schemas['AIThreadDeltaEvent'] }
  | { type: 'tool'; id: number; data: Schemas['AIThreadToolEvent'] }
  | { type: 'message'; id: number; data: Schemas['AIThreadMessageEvent'] }
  | { type: 'usage'; id: number; data: Schemas['AIThreadUsageEvent'] }
  | { type: 'confirmation'; id: number; data: Schemas['AIThreadConfirmationEvent'] }
  | { type: 'problem'; id: number; data: StreamProblem }

export interface ThreadState {
  thread?: Thread
  messages: ThreadMessage[]
  turn?: ThreadTurn
  usage?: ThreadUsage
  /** The answer of the latest turn as it arrives, until its message replaces it. */
  live?: { turnId: string; text: string }
  /** Tool rows of a running turn, until its tool message holds them. */
  tools: ToolRow[]
  toolsTurn?: string
  /**
   * What a stopped turn had written: the server keeps no message of it, so
   * the panel shows it until the thread is read again.
   */
  stopped?: { turnId: string; text: string }
  /**
   * The call of a writing MCP tool the latest turn waits on, or decided:
   * pending, the card asks the person; decided, the thread says so until
   * the turn goes on.
   */
  confirmation?: Confirmation
  /** The person's message on its way, shown before the server has it. */
  pending?: string
  /** The last event applied: a resumed stream starts after it. */
  lastId: number
  problem?: StreamProblem
}

export type Action =
  | { type: 'events'; events: StreamEvent[] }
  /** The thread as an API call returned it (create, send, stop, attach…). */
  | { type: 'thread'; thread: Thread }
  | { type: 'pending'; text?: string }
  | { type: 'reset' }

export const initialThreadState: ThreadState = { messages: [], tools: [], lastId: 0 }

/** A turn still in progress: queued, answering or waiting for the person. */
export function turnActive(turn: ThreadTurn | undefined): boolean {
  return Boolean(turn && (turn.state === 'queued' || turn.state === 'answering' || turn.state === 'waiting'))
}

function upsert(messages: ThreadMessage[], m: ThreadMessage): ThreadMessage[] {
  const i = messages.findIndex((x) => x.id === m.id)
  if (i >= 0) {
    const next = messages.slice()
    next[i] = m
    return next
  }
  return [...messages, m]
}

/** The whole thread replaces what was kept (a snapshot, or an API response at least as new). */
function fromThread(s: ThreadState, thread: Thread, id: number): ThreadState {
  const turn = thread.turn
  const live = turn && turnActive(turn) && turn.text ? { turnId: turn.id, text: turn.text } : undefined
  const pending = s.pending && !thread.messages.some((m) => m.role === 'user' && m.text === s.pending) ? s.pending : undefined
  // A stopped turn keeps what it wrote on screen, unless the thread has it
  // saved (Stop saves what arrived as a partial message).
  const wrote = s.stopped ?? (s.live?.text ? s.live : undefined)
  const saved = wrote && thread.messages.some((m) => m.role === 'assistant' && m.turnId === wrote.turnId)
  const stopped = wrote && !saved && turn?.id === wrote.turnId && turn.state === 'stopped' ? wrote : undefined
  // A thread read while its turn waits has the call; otherwise a decided one of the same turn stays.
  const confirmation = thread.confirmation ?? (s.confirmation && s.confirmation.turnId === turn?.id && s.confirmation.state !== 'pending' ? s.confirmation : undefined)
  return { ...s, thread, messages: thread.messages, turn, usage: thread.usage, live, tools: [], toolsTurn: undefined, stopped, pending, confirmation, lastId: Math.max(id, 0), problem: undefined }
}

function applyEvent(s: ThreadState, e: StreamEvent): ThreadState {
  // Each event comes once, in order; after a reconnection a repeat is skipped.
  if (e.type !== 'snapshot' && e.type !== 'problem' && e.id > 0 && e.id <= s.lastId) return s
  const lastId = e.id > 0 ? Math.max(s.lastId, e.id) : s.lastId
  switch (e.type) {
    case 'snapshot':
      return fromThread(s, e.data.thread, e.id)
    case 'problem':
      return { ...s, problem: e.data }
    case 'turn': {
      const turn = e.data.turn
      if (s.turn && s.turn.id !== turn.id && s.turn.sequence > turn.sequence) return { ...s, lastId }
      let { live, stopped } = s
      if (live && live.turnId === turn.id && !turnActive(turn)) {
        // Its message (sent before the turn ends) has replaced the text;
        // a stopped turn keeps what it wrote on screen.
        if (turn.state === 'stopped' && live.text) stopped = { turnId: turn.id, text: live.text }
        live = undefined
      }
      if (live && live.turnId !== turn.id) live = undefined
      const tools = s.toolsTurn && s.toolsTurn !== turn.id ? [] : s.tools
      // Another turn's call is over; a turn that ended without a decision closed its call.
      let confirmation = s.confirmation && s.confirmation.turnId === turn.id ? s.confirmation : undefined
      if (confirmation && confirmation.state === 'pending' && !turnActive(turn)) confirmation = undefined
      return { ...s, turn, live, stopped, tools, confirmation, lastId }
    }
    case 'delta': {
      const { turnId, text } = e.data
      const live = s.live && s.live.turnId === turnId ? { turnId, text: s.live.text + text } : { turnId, text }
      return { ...s, live, lastId }
    }
    case 'tool': {
      const { turnId, tool } = e.data
      const base = s.toolsTurn === turnId ? s.tools : []
      const tools = base.some((t) => t.callId === tool.callId) ? base : [...base, tool]
      return { ...s, tools, toolsTurn: turnId, lastId }
    }
    case 'message': {
      const m = e.data.message
      const messages = upsert(s.messages, m)
      let { live, tools, toolsTurn, pending, stopped } = s
      if (m.role === 'assistant' && live && live.turnId === m.turnId) live = undefined
      // A stopped answer's saved message replaces the copy kept on screen.
      if (m.role === 'assistant' && stopped?.turnId === m.turnId) stopped = undefined
      if (m.role === 'tool' && toolsTurn === m.turnId) {
        const held = new Set((m.tools ?? []).map((t) => t.callId))
        tools = tools.filter((t) => !held.has(t.callId))
        if (!tools.length) toolsTurn = undefined
      }
      if (m.role === 'user' && pending === m.text) pending = undefined
      return { ...s, messages, live, tools, toolsTurn, pending, stopped, lastId }
    }
    case 'usage':
      return { ...s, usage: e.data.usage, lastId }
    case 'confirmation':
      return { ...s, confirmation: e.data.confirmation, lastId }
  }
}

export function threadReducer(s: ThreadState, a: Action): ThreadState {
  switch (a.type) {
    case 'reset':
      return initialThreadState
    case 'pending':
      return { ...s, pending: a.text }
    case 'thread':
      // An API response is as new as its eventSeq: older than what the
      // stream already applied, it would undo events.
      if (s.thread && s.thread.id === a.thread.id && a.thread.eventSeq < s.lastId) return { ...s, thread: { ...a.thread, messages: s.messages, turn: s.turn, usage: s.usage ?? a.thread.usage } }
      return fromThread(s, a.thread, a.thread.eventSeq)
    case 'events':
      return a.events.reduce(applyEvent, s)
  }
}
