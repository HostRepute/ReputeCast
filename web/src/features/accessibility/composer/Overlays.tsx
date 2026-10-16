import type { Component, Editor } from 'grapesjs'
import { useMemo } from 'react'
import { useComposer } from '@/features/composer/context'
import { sectionOf, tagOf } from '@/features/composer/editor/labels'
import { componentAt, reveal } from '@/features/composer/fixes'
import { ToolIcon } from '@/features/composer/icons'
import { currentLanguage } from './Panel'
import { useAccessibility, type Issue } from './state'

export interface Box {
  top: number
  left: number
  width: number
  height: number
}

type BoxOf = (editor: Editor, el: Element | null | undefined, host: HTMLElement | null) => Box | null

/** The element an issue is about inside its component: the image, the nth link or heading, the button. */
function elementOf(issue: Issue, target: Component): Element | null | undefined {
  const el = target.getEl()
  if (!el) return el
  const tag = tagOf(target)
  if (tag === 'mj-image') return el.querySelector('img') ?? el
  if (tag === 'mj-button') return el.querySelector('a, p') ?? el
  if (issue.kind === 'link_text' || issue.kind === 'link_no_text') return el.querySelectorAll('a')[issue.element.index] ?? el
  if (issue.kind === 'heading') return el.querySelectorAll('h1, h2, h3, h4, h5, h6')[issue.element.index] ?? el
  return el
}

/**
 * The canvas tags of the accessibility issues (ComposerAccessibility): a
 * tag on each element, red for serious issues and amber for moderate ones,
 * that opens the issue in the Accessibility tab.
 */
export function AccessibilityOverlays({ host, tick, boxOf, language }: { host: HTMLElement | null; tick: number; boxOf: BoxOf; language?: string }) {
  const c = useComposer()
  const a = useAccessibility()
  const ed = c.editor
  const report = a?.report
  const placed = useMemo(() => {
    if (!ed || !report || report.mode === 'html') return []
    const lang = currentLanguage(report, language)
    const out: { issue: Issue; box: Box; tagAt: { top: number; left: number; center?: boolean } }[] = []
    for (const issue of report.issues) {
      if (!issue.tag || !issue.path.length || !issue.languages.includes(lang)) continue
      const target = componentAt(c, issue.path)
      if (!target) continue
      const box = boxOf(ed, elementOf(issue, target), host)
      if (!box) continue
      // Where the tag sits, as on the board: on an image's top-left corner,
      // centred above a button or small text, above the row a heading run
      // starts, above the text block of a link.
      const tag = tagOf(target)
      let tagAt: { top: number; left: number; center?: boolean }
      if (tag === 'mj-image') tagAt = { top: box.top + 12, left: box.left + 12 }
      else if (tag === 'mj-button' || issue.kind === 'text_size') tagAt = { top: box.top - 30, left: box.left + box.width / 2, center: true }
      else {
        const anchor = issue.kind === 'heading' && issue.paths.length > 1 ? sectionOf(target) : target
        const ab = boxOf(ed, anchor?.getEl(), host) ?? box
        tagAt = { top: ab.top - 13, left: ab.left }
      }
      out.push({ issue, box, tagAt })
    }
    return out
    // tick re-reads the canvas positions after every change.
  }, [ed, host, tick, report, language]) // eslint-disable-line react-hooks/exhaustive-deps
  return (
    <>
      {placed.map(({ issue, box, tagAt }) => {
        const tone = issue.severity === 'serious' ? 'b' : 'w'
        const tagStyle = { top: tagAt.top, left: tagAt.left, transform: tagAt.center ? 'translateX(-50%)' : undefined }
        return (
          <div key={issue.id}>
            <div className={`wbox ${tone}`} style={{ top: box.top, left: box.left, width: box.width, height: box.height }} aria-hidden="true" />
            <button
              type="button"
              className={`wtag ${tone}`}
              style={tagStyle}
              title={issue.title}
              onClick={() => {
                reveal(c, issue.path)
                c.setRightTab('accessibility')
              }}
            >
              <ToolIcon name="warn" size={12} strokeWidth={2} />
              {issue.tag}
            </button>
          </div>
        )
      })}
    </>
  )
}
