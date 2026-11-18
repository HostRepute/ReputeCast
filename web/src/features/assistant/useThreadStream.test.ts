import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useThreadStream } from './useThreadStream'

/** An EventSource the test drives: its URL, and the events it delivers. */
class FakeSource {
  static OPEN = 1
  static all: FakeSource[] = []
  readyState = 1
  onopen: (() => void) | null = null
  onerror: ((e: Event) => void) | null = null
  private listeners = new Map<string, ((e: MessageEvent) => void)[]>()
  readonly url: string
  constructor(url: string) {
    this.url = url
    FakeSource.all.push(this)
  }
  addEventListener(kind: string, fn: (e: MessageEvent) => void) {
    this.listeners.set(kind, [...(this.listeners.get(kind) ?? []), fn])
  }
  close() {
    this.readyState = 2
  }
  emit(kind: string, id: number, data: unknown) {
    for (const fn of this.listeners.get(kind) ?? []) fn(new MessageEvent(kind, { data: JSON.stringify(data), lastEventId: String(id) }))
  }
  fail() {
    this.onerror?.(new Event('error'))
  }
}

const after = (s: FakeSource) => new URL(s.url, 'http://x').searchParams.get('lastEventId')
const latest = () => FakeSource.all[FakeSource.all.length - 1]!

beforeEach(() => {
  FakeSource.all = []
  vi.stubGlobal('EventSource', FakeSource)
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('useThreadStream', () => {
  it('starts a switched-to thread from its own events, not the previous thread’s', () => {
    const dispatch = vi.fn()
    const { rerender } = renderHook((p: { thread: string; applied: { thread?: string; id: number } }) => useThreadStream('acme', p.thread, p.applied, true, dispatch), {
      initialProps: { thread: 'A', applied: { thread: 'A', id: 50 } },
    })
    expect(after(latest())).toBe('50')
    // The first render after the switch still has A's state (its event 50).
    rerender({ thread: 'B', applied: { thread: 'A', id: 50 } })
    expect(latest().url).toContain('/threads/B/')
    expect(after(latest())).toBeNull()
    // B's own state comes; after a drop the stream resumes after B's event 5.
    rerender({ thread: 'B', applied: { thread: 'B', id: 5 } })
    act(() => latest().fail())
    act(() => void vi.advanceTimersByTime(60_000))
    expect(after(latest())).toBe('5')
  })

  it('takes a snapshot’s number as the cursor, even when lower', () => {
    const dispatch = vi.fn()
    const { rerender } = renderHook((p: { applied: { thread?: string; id: number } }) => useThreadStream('acme', 'B', p.applied, true, dispatch), {
      initialProps: { applied: { thread: 'B', id: 50 } },
    })
    act(() => latest().emit('snapshot', 7, { thread: { id: 'B' } }))
    rerender({ applied: { thread: 'B', id: 7 } })
    act(() => latest().fail())
    act(() => void vi.advanceTimersByTime(60_000))
    expect(after(latest())).toBe('7')
  })
})
