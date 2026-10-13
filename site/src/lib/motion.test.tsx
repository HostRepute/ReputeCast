import { act, fireEvent, render, screen } from '@testing-library/react'
import { Count } from '../components/Count'
import { Header } from '../sections/Header'
import { getMotion, refresh, setPaused, STORAGE_KEY, useProgress } from './motion'

function mockReducedMotion(reduce: boolean) {
  window.matchMedia = ((query: string) => ({
    matches: reduce && query.includes('reduce'),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia
}

function Counter({ start = true }: { start?: boolean }) {
  const { p, live } = useProgress({ start, delay: 100, duration: 1000 })
  return (
    <b data-testid="n" data-live={live ? 'yes' : 'no'}>
      <Count final={31604} p={p} live={live} />
    </b>
  )
}

const shown = () => screen.getByTestId('n').querySelector('.count-v')!.textContent

beforeEach(() => {
  delete document.documentElement.dataset.motion
  document.documentElement.removeAttribute('data-settled')
  window.localStorage.clear()
  mockReducedMotion(false)
  refresh()
})

describe('the Pause motion control', () => {
  it('pauses on the <html> element, remembers it, and settles entrances for good', () => {
    render(<Header />)
    const button = screen.getByRole('button', { name: /pause motion/i })
    fireEvent.click(button)
    expect(document.documentElement.dataset.motion).toBe('paused')
    expect(document.documentElement.hasAttribute('data-settled')).toBe(true)
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe('paused')
    expect(getMotion().paused).toBe(true)

    fireEvent.click(button)
    expect(document.documentElement.dataset.motion).toBeUndefined()
    // Entrances that already finished never replay.
    expect(document.documentElement.hasAttribute('data-settled')).toBe(true)
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull()
  })

  it('renders both labels so the right one shows before any script runs', () => {
    render(<Header />)
    const button = screen.getByRole('button', { name: /pause motion/i })
    expect(button.querySelector('.when-playing')).toHaveTextContent('Pause motion')
    expect(button.querySelector('.when-paused')).toHaveTextContent('Play motion')
  })
})

describe('counters', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'requestAnimationFrame', 'cancelAnimationFrame', 'performance'] })
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('drop to zero after hydration and count up to the final value', () => {
    render(<Counter />)
    expect(screen.getByTestId('n').dataset.live).toBe('yes')
    expect(shown()).toBe('0')
    act(() => {
      vi.advanceTimersByTime(600)
    })
    const mid = Number(shown()!.replace(/,/g, ''))
    expect(mid).toBeGreaterThan(0)
    expect(mid).toBeLessThan(31604)
    act(() => {
      vi.advanceTimersByTime(2000)
    })
    expect(shown()).toBe('31,604')
  })

  it('show the final value right away with reduced motion', () => {
    mockReducedMotion(true)
    refresh()
    render(<Counter />)
    expect(screen.getByTestId('n').dataset.live).toBe('no')
    expect(shown()).toBe('31,604')
  })

  it('jump to the final value when motion is paused mid-count', () => {
    render(<Counter />)
    act(() => {
      vi.advanceTimersByTime(400)
    })
    act(() => setPaused(true))
    expect(shown()).toBe('31,604')
  })

  it('always expose the final value to screen readers', () => {
    render(<Counter start={false} />)
    expect(screen.getByTestId('n').querySelector('.vh')).toHaveTextContent('31,604')
    expect(screen.getByTestId('n').querySelector('.count-v')).toHaveAttribute('aria-hidden', 'true')
  })
})
