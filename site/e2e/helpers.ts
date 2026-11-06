import { createRequire } from 'node:module'
import type { Page } from '@playwright/test'

const axePath = createRequire(import.meta.url).resolve('axe-core/axe.min.js')
const options = { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice'] }, resultTypes: ['violations'] }

type Violation = { id: string; nodes: { target: string[] }[] }

/** axe's violations on the whole page, as "rule: first targets". */
export async function axe(page: Page): Promise<string[]> {
  await page.addScriptTag({ path: axePath })
  const violations = (await page.evaluate(`axe.run(document, ${JSON.stringify(options)}).then((r) => r.violations)`)) as Violation[]
  return violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).slice(0, 3).join(', ')}`)
}

/** WCAG 1.4.12's text spacing, as a reader's own style sheet would set it. */
export const textSpacing = '* { line-height: 1.5 !important; letter-spacing: 0.12em !important; word-spacing: 0.16em !important; } p { margin-bottom: 2em !important; }'

/** How far the page scrolls sideways. */
export const sideways = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)

/** The data-p of the comparison's visible body cells. */
export const shownColumns = (page: Page) =>
  page.evaluate(() => [
    ...new Set(
      [...document.querySelectorAll<HTMLElement>('.cmp-table > [role="rowgroup"]:last-child .cmp-cell')]
        .filter((c) => getComputedStyle(c).display !== 'none')
        .map((c) => c.dataset.p),
    ),
  ])

/** The title of the receipt's open card. */
export const openFeature = (page: Page) =>
  page.evaluate(() =>
    [...document.querySelectorAll('.fx-detail')]
      .filter((d) => getComputedStyle(d).display !== 'none')
      .map((d) => d.querySelector('h3')?.textContent),
  )
