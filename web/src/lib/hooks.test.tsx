import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import { act, render, renderHook, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { useFirstLoadSettled, useOpenedOnce } from './hooks'
import { getSystemStatus, resetSystemStatus, systemStatusMiddleware, useSystemStatus } from './systemStatus'

type OnResponse = NonNullable<typeof systemStatusMiddleware.onResponse>

async function respond(status = 200) {
  const request = new Request('/api/v1/workspaces/acme-newsletter/campaigns')
  await (systemStatusMiddleware.onResponse as OnResponse)({ request, response: new Response('{}', { status }), schemaPath: '', params: {}, options: {} as never, id: '1' })
}

describe('useSystemStatus', () => {
  beforeEach(() => resetSystemStatus())

  it('re-renders a component that picks a part only when that part changes', async () => {
    let renders = 0
    function Limit() {
      renders++
      const limit = useSystemStatus((s) => s.rateLimit)
      return <p>{limit ? 'limited' : 'ok'}</p>
    }
    render(<Limit />)
    const before = renders
    // Each response moves lastActivity; the rate limit stays as it was.
    await act(async () => {
      await respond()
      await respond()
      await respond()
    })
    expect(getSystemStatus().lastActivity).toBeGreaterThan(0)
    expect(renders).toBe(before)
    await act(() => respond(429))
    expect(screen.getByText('limited')).toBeInTheDocument()
  })
})

describe('useFirstLoadSettled', () => {
  it('re-renders when the first loads are in, not once per query', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const done: (() => void)[] = []
    const later = (key: string) => () =>
      new Promise<string>((resolve) => {
        done.push(() => resolve(key))
      })
    let renders = 0
    function Screen({ children }: { children: ReactNode }) {
      renders++
      const settled = useFirstLoadSettled()
      return (
        <>
          <p>{settled ? 'settled' : 'loading'}</p>
          {children}
        </>
      )
    }
    function Part({ k }: { k: string }) {
      useQuery({ queryKey: [k], queryFn: later(k) })
      return null
    }
    render(
      <QueryClientProvider client={client}>
        <Screen>
          <Part k="a" />
          <Part k="b" />
          <Part k="c" />
        </Screen>
      </QueryClientProvider>,
    )
    // After the first commit it looks for first loads, and finds three.
    await act(() => new Promise((r) => setTimeout(r, 0)))
    expect(screen.getByText('loading')).toBeInTheDocument()
    const loading = renders
    // TanStack Query tells its listeners on a timer: each step waits for it.
    const tick = () => new Promise((r) => setTimeout(r, 10))
    await act(async () => {
      done[0]!()
      await tick()
      done[1]!()
      await tick()
    })
    expect(renders).toBe(loading)
    await act(async () => {
      done[2]!()
      await tick()
    })
    expect(screen.getByText('settled')).toBeInTheDocument()
    const settled = renders
    // Settled, it no longer listens: a later fetch doesn't re-render it.
    await act(async () => {
      await client.prefetchQuery({ queryKey: ['d'], queryFn: () => 'd' })
      await tick()
    })
    expect(renders).toBe(settled)
  })
})

describe('useOpenedOnce', () => {
  it('is false until opened, then stays true', () => {
    const { result, rerender } = renderHook(({ open }) => useOpenedOnce(open), { initialProps: { open: false } })
    expect(result.current).toBe(false)
    rerender({ open: true })
    expect(result.current).toBe(true)
    rerender({ open: false })
    expect(result.current).toBe(true)
  })
})
