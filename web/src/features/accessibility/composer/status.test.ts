import type { Schemas } from '@/api/client'
import { withoutAltText } from './Status'

type Checks = Schemas['ComposerChecks']
type Issue = Schemas['ContentCheckIssue']

const issue = (id: string, severity: Issue['severity']): Issue => ({ id, severity, group: 'Content', title: id, path: [], fixes: [] }) as unknown as Issue

describe('the composer’s content checks', () => {
  it('leave the images’ alt text to the Accessibility tab', () => {
    // ComposerWarnings: Deliverability drops by one.
    const checks = {
      issues: [issue('content.alt_text', 'warn'), issue('links.blocklist', 'block'), issue('subject.spam', 'warn')],
      blocks: 1,
      warnings: 2,
      verdict: 'blocked',
    } as unknown as Checks
    const out = withoutAltText(checks)!
    expect(out.issues.map((i) => i.id)).toEqual(['links.blocklist', 'subject.spam'])
    expect([out.blocks, out.warnings, out.verdict]).toEqual([1, 1, 'blocked'])
    // Nothing to take out: the same object.
    expect(withoutAltText(out)).toBe(out)
    expect(withoutAltText(undefined)).toBeUndefined()
  })
})
