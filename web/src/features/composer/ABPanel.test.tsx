import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ABPanel, decisionRules, newTest, PreviewingSwitch, variantLines, withVariant } from './ABPanel'
import { ComposerCtx, type ABTest, type Checks, type ComposerState } from './context'
import { testCampaign, testComposer } from './testing'

// CO-6: A/B in the composer. New variants start from A's subject; the test
// length, winner metric and decision rule are set and said; each variant's
// own check shows under it; the preview says which variant it shows.

const subjectTest = (more: Partial<ABTest> = {}): ABTest => ({
  kind: 'subject',
  testPercent: 20,
  metric: 'clicks',
  durationHours: 2,
  variants: [{ key: 'A' }, { key: 'B', subject: "What shipped this week, and what we're retiring" }],
  ...more,
})

function show(state: ComposerState, ui = <ABPanel />) {
  return render(<ComposerCtx.Provider value={state}>{ui}</ComposerCtx.Provider>)
}

describe('new tests and variants', () => {
  it("start from A's subject, judged by clicks", () => {
    const t = newTest('subject', 'Three launches, one deprecation', () => '')
    expect(t.metric).toBe('clicks')
    expect(t.variants).toEqual([{ key: 'A' }, { key: 'B', subject: 'Three launches, one deprecation' }])
    const three = withVariant(t, 'Three launches, one deprecation', () => '')
    expect(three.variants[2]).toEqual({ key: 'C', subject: 'Three launches, one deprecation' })
    // Three is the most.
    expect(withVariant(three, 'x', () => '')).toBe(three)
    expect(newTest('content', 'x', () => '<mjml/>').variants[1]).toEqual({ key: 'B', mjml: '<mjml/>' })
  })

  it('send what each variant has, or what A has', () => {
    const a = { subject: 'Subject A', preheader: 'Preheader A' }
    expect(variantLines(subjectTest(), 'B', a)).toEqual({ subject: "What shipped this week, and what we're retiring", preheader: 'Preheader A' })
    // An empty subject variant previews empty (its check says so), not as A.
    expect(variantLines(subjectTest({ variants: [{ key: 'A' }, { key: 'B', subject: '' }] }), 'B', a).subject).toBe('')
    // A content variant without a subject of its own sends A's.
    expect(variantLines({ ...subjectTest(), kind: 'content', variants: [{ key: 'A' }, { key: 'B', mjml: '<mjml/>' }] }, 'B', a)).toEqual(a)
    expect(variantLines(subjectTest(), 'A', a)).toEqual(a)
  })

  it("read R05's decision fields when they are there", () => {
    expect(decisionRules(subjectTest())).toEqual({ minSample: 500, confidence: 0.95, noWinner: 'default' })
    expect(decisionRules({ ...subjectTest(), minSample: 1000, confidence: 0.9 } as ABTest).minSample).toBe(1000)
  })
})

describe('A/B panel', () => {
  it('sets the test length and metric, and says the rule and who gets what', () => {
    const setABTest = vi.fn()
    show(testComposer({ abTest: subjectTest(), setABTest, campaign: testCampaign({ audienceCount: 31192 }) }))
    expect(screen.getByText('ReputeCast picks a winner only when it is significant at 95% with at least 500 recipients per variant; otherwise everyone else gets A.')).toBeInTheDocument()
    expect(screen.getByText('3,119 recipients per variant. 2 h after the test starts, the winner, or A, goes to the remaining 24,954.')).toBeInTheDocument()
    expect(screen.getByText("Apple Mail opens every email on download, so opens overstate interest. Clicks don't.")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Test length'), { target: { value: '4' } })
    expect(setABTest).toHaveBeenLastCalledWith(expect.objectContaining({ durationHours: 4 }))
    const metric = screen.getByLabelText(/Winner by/)
    expect(within(metric).getByRole('option', { name: 'Clicks (recommended)' })).toBeInTheDocument()
    fireEvent.change(metric, { target: { value: 'human_opens' } })
    expect(setABTest).toHaveBeenLastCalledWith(expect.objectContaining({ metric: 'human_opens' }))
  })

  it("adds a variant with A's subject", () => {
    const setABTest = vi.fn()
    show(testComposer({ abTest: subjectTest(), setABTest, subject: 'Three launches' }))
    fireEvent.click(screen.getByRole('button', { name: 'Add variant' }))
    expect(setABTest.mock.calls[0]![0].variants[2]).toEqual({ key: 'C', subject: 'Three launches' })
  })

  it("shows each variant's own check under it", () => {
    const checks = {
      issues: [],
      variants: [{ key: 'B', blocks: 1, warnings: 0, issues: [{ id: 'subject.missing', severity: 'block', group: 'Subject', title: 'Write a subject for variant B.', path: [], section: -1, fixes: [] }] }],
    } as unknown as Checks
    show(testComposer({ abTest: subjectTest({ variants: [{ key: 'A' }, { key: 'B', subject: '' }] }), checks }))
    expect(screen.getByText('1 blocking')).toBeInTheDocument()
    expect(screen.getByText('Write a subject for variant B.')).toBeInTheDocument()
  })
})

describe('Previewing', () => {
  it('switches the variant the preview shows', () => {
    const setVariant = vi.fn()
    show(testComposer({ abTest: subjectTest(), setVariant }), <PreviewingSwitch />)
    expect(screen.getByRole('button', { name: 'Subject A' })).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Subject B' }))
    expect(setVariant).toHaveBeenCalledWith('B')
  })

  it('has nothing to switch without a test, or for send times', () => {
    const { container } = show(testComposer(), <PreviewingSwitch />)
    expect(container).toBeEmptyDOMElement()
  })
})
