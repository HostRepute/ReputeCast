import { render } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { m } from './messages'
import { accessibilityBlock, CodeText } from './review'

type Run = Schemas['ReadinessRun']
type Check = Schemas['ReadinessCheck']

const check = (over: Partial<Check>): Check => ({ key: 'k', group: 'content', status: 'pass', title: 'A check', fixes: [], flatTitle: '', ...over }) as Check

describe('Review & send · Accessibility', () => {
  it('sets code in backticks apart', () => {
    const { container } = render(<CodeText text={'`role="presentation"` on 16 layout tables'} />)
    expect(container.querySelector('code')?.textContent).toBe('role="presentation"')
    expect(container.textContent).toBe('role="presentation" on 16 layout tables')
    // Unpaired backticks stay as they are.
    expect(render(<CodeText text="a ` b" />).container.querySelector('code')).toBeNull()
  })

  it('says when the accessibility policy alone blocks the send', () => {
    const note = 'Your workspace blocks sends with serious accessibility issues (Settings › General). Changed by Linh Nguyen at 14:22.'
    const run = {
      checks: [
        check({ key: 'accessibility.alt_text:1-0-0', group: 'accessibility', status: 'block' }),
        check({ key: 'accessibility.contrast:6-0-0', group: 'accessibility', status: 'block' }),
        check({ key: 'accessibility.moderate', group: 'accessibility', status: 'warn' }),
      ],
      groups: [{ key: 'accessibility', title: 'Accessibility', caption: '', note, passed: 2, total: 5 }],
    } as unknown as Run
    expect(accessibilityBlock(run)).toEqual({ count: 2, note })
    expect(t(m.blockedTitle, { count: 2 })).toBe('Blocked by 2 accessibility issues')
    // Another block among them: the usual Blocked banner.
    run.checks.push(check({ key: 'compliance.footer', group: 'compliance', status: 'block' }))
    expect(accessibilityBlock(run)).toBeUndefined()
  })

  it('words the one-click preview', () => {
    expect(t(m.oneClickTitle, { count: 2 })).toBe('Fix 2 issues in every language?')
    expect(t(m.oneClickConfirm, { count: 2 })).toBe('Fix 2 in every language')
    expect(t(m.oneClickHeadingFrom, { level: 3, texts: '“Scheduled rollouts”, “Faster CSV exports”' })).toBe('− Heading 3 · “Scheduled rollouts”, “Faster CSV exports”')
    expect(t(m.impactHeld, { count: 0 })).toBe('No scheduled campaign would be held at its send-time recheck.')
    expect(t(m.impactFlowLive, { count: 1, summary: 'button contrast', live: 3, version: 4 })).toBe(
      'Draft · 1 serious issue: button contrast. The live v3 keeps sending; publishing v4 needs the fix.',
    )
  })
})
