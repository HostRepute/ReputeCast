import { parse } from '@formatjs/icu-messageformat-parser'
import { describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { dc } from './composer/messages'
import { wrapCount } from './composer/DigestMode'
import { im } from './issueMessages'
import { m } from './messages'
import { s } from './settingsMessages'

describe('digest messages', () => {
  it('parse as ICU MessageFormat', () => {
    for (const set of [m, s, dc, im]) {
      for (const d of Object.values(set)) {
        expect(() => parse(d.message), d.id).not.toThrow()
      }
    }
  })

  it('format the samples the boards show', () => {
    expect(t(s.hopRedirect, { status: '301' })).toBe('301 Moved Permanently · checked by the SSRF guard')
    expect(t(s.hopRedirect, { status: '418' })).toBe('418 · checked by the SSRF guard')
    expect(t(dc.wrapIt)).toBe('Wrap it in {% if issue.items.size > 1 %}')
    expect(t(im.approveIssue, { number: 3 })).toBe('Approve issue #3')
    expect(t(im.nextIssueLine, { when: 'Fri 17 Oct, 08:00', posts: 2, approval: 'yes' })).toBe('Fri 17 Oct, 08:00 · 2 posts so far · needs approval')
    expect(t(dc.sizesRoom, { limit: '102 KB', fit: 7, max: 6 })).toBe('Gmail clips at 102 KB. Up to 7 posts fit, so the maximum of 6 leaves room for a long excerpt.')
    expect(t(dc.feedItemOf, { number: 1, total: 2 })).toBe('Feed item 1 of 2')
    expect(t(dc.selectionLabel, { layout: 'grid' })).toBe('Feed items · Grid')
  })
})

describe('wrapCount', () => {
  it('wraps "and N more" so one post never reads "and 0 more"', () => {
    expect(wrapCount('{{ issue.items.first.title }} and {{ issue.items.size | minus: 1 }} more from the Acme blog')).toBe(
      '{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %} from the Acme blog',
    )
  })
  it('leaves a subject alone when the count is already wrapped or absent', () => {
    expect(wrapCount('{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %}')).toBeNull()
    expect(wrapCount('New on the blog')).toBeNull()
  })
})
