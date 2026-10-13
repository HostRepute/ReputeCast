import type { Component } from 'grapesjs'
import { useEffect, useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { useComposer, type Issue } from './context'
import { mjChildren, sectionOf, tagOf } from './editor/labels'
import { applyFix, bodyOf, reveal } from './fixes'
import { describe, SettingsFields, StyleFields } from './Inspector'

/** Issues the Deliverability tab counts: blocks and warnings about the content (the subject has its own line). */
export function contentIssues(issues: Issue[] | undefined): Issue[] {
  return (issues ?? []).filter((i) => (i.severity === 'block' || i.severity === 'warn') && i.group !== 'Subject')
}

function plural(n: number, one: string, many = `${one}s`) {
  return `${n} ${n === 1 ? one : many}`
}

/** "Deliverability on this section": images with alt text and links, from the last check. */
function SectionHealth({ selected }: { selected: Component | null }) {
  const c = useComposer()
  const body = bodyOf(c)
  const sec = sectionOf(selected)
  const index = sec && body ? mjChildren(body).indexOf(sec) : -1
  const stats = index >= 0 ? c.checks?.sections.find((s) => s.index === index) : undefined
  const issues = (c.checks?.issues ?? []).filter((i) => (index >= 0 ? i.section === index : true) && (i.severity === 'block' || i.severity === 'warn') && i.group !== 'Subject')
  let tone: 'ok' | 'warn' | 'bad' | 'neutral' = 'ok'
  let text: string
  if (!c.checks) {
    tone = 'neutral'
    text = 'Checking…'
  } else if (index < 0) {
    const n = contentIssues(c.checks.issues).length
    tone = c.checks.blocks ? 'bad' : n ? 'warn' : 'ok'
    text = n ? `${plural(c.checks.blocks, 'block')} · ${plural(c.checks.warnings, 'warning')} in the email` : `${plural(c.checks.images, 'image')} · ${plural(c.checks.links, 'link')} · no warnings`
  } else if (!stats) {
    tone = 'neutral'
    text = 'Nothing to check in this section'
  } else {
    const parts: string[] = []
    const missingAlt = stats.images - stats.imagesWithAlt
    const badLinks = stats.links - stats.linksOk
    if (stats.images) parts.push(missingAlt ? `${missingAlt} of ${plural(stats.images, 'image')} missing alt text` : `${plural(stats.images, 'image')} with alt text`)
    if (stats.links) parts.push(badLinks ? `${badLinks} of ${plural(stats.links, 'link')} flagged` : `${plural(stats.links, 'link')} OK`)
    if (!parts.length) parts.push('No images or links')
    if (issues.some((i) => i.severity === 'block')) tone = 'bad'
    else if (issues.length || missingAlt || badLinks) tone = 'warn'
    text = parts.join(' · ')
  }
  return (
    <div className="flex flex-col gap-1.5 border-t border-line-2" style={{ padding: '10px 12px', marginTop: 'auto' }}>
      <span className="k">{index >= 0 ? 'Deliverability on this section' : 'Deliverability'}</span>
      <Badge tone={tone} className="self-start">
        {text}
      </Badge>
    </div>
  )
}

/** "Mobile checks": tap targets and text size, measured on the mobile canvas. */
function MobileChecks({ selected }: { selected: Component | null }) {
  const c = useComposer()
  const [result, setResult] = useState<{ smallButtons: number; smallText: number; buttons: number; texts: number } | null>(null)
  useEffect(() => {
    const h = window.setTimeout(() => {
      const scope = selected?.getEl() ?? c.editor?.Canvas.getBody()
      if (!scope) return setResult(null)
      const win = scope.ownerDocument.defaultView
      let smallButtons = 0,
        buttons = 0,
        smallText = 0,
        texts = 0
      scope.querySelectorAll('[data-gjs-type="mj-button"] a, [data-gjs-type="mj-button"] p').forEach((el) => {
        const box = el as HTMLElement
        if (!box.offsetHeight) return
        buttons++
        if (box.offsetHeight < 44) smallButtons++
      })
      const textEls = [...(selected && tagOf(selected) === 'mj-text' ? [scope] : []), ...Array.from(scope.querySelectorAll('[data-gjs-type="mj-text"]'))]
      textEls.forEach((el) => {
        const inner = (el as HTMLElement).querySelector('div') ?? (el as HTMLElement)
        if (!inner.textContent?.trim() || !(inner as HTMLElement).offsetHeight) return
        texts++
        const size = parseFloat(win?.getComputedStyle(inner).fontSize ?? '0')
        if (size < 14) smallText++
      })
      setResult({ smallButtons, smallText, buttons, texts })
    }, 300)
    return () => window.clearTimeout(h)
  }, [c.editor, c.rev, c.device, selected])
  const ok = result && !result.smallButtons && !result.smallText
  const parts: string[] = []
  if (result) {
    if (result.buttons) parts.push(result.smallButtons ? `${plural(result.smallButtons, 'button')} under 44 px tall` : 'Buttons ≥ 44 px tall')
    if (result.texts) parts.push(result.smallText ? `${plural(result.smallText, 'text block')} under 14 px` : 'text ≥ 14 px')
  }
  return (
    <div className="flex flex-col gap-1.5 border-t border-line-2" style={{ padding: '10px 12px', marginTop: 'auto' }}>
      <span className="k">Mobile checks</span>
      <Badge tone={!result ? 'neutral' : ok ? 'ok' : 'warn'} className="self-start">
        {!result ? 'Checking…' : parts.length ? parts.join(' · ') : 'No buttons or text here'}
      </Badge>
    </div>
  )
}

function InspectorHead({ c, tab }: { c: Component; tab: 'style' | 'settings' }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
      <span className="font-semibold">{tagOf(c) || 'element'}</span>
      <span className="k">{describe(c, tab)}</span>
    </div>
  )
}

function NothingSelected() {
  return (
    <div className="k" style={{ padding: '14px 12px' }}>
      Select a component on the canvas to edit its MJML attributes.
    </div>
  )
}

function IssueRow({ issue }: { issue: Issue }) {
  const c = useComposer()
  const [alt, setAlt] = useState('')
  const tone = issue.severity === 'block' ? 'is-block' : issue.group === 'Subject' ? '' : 'is-warn'
  const select = () => {
    if (issue.path.length) reveal(c, issue.path)
  }
  return (
    <div className={`iss ${tone}`}>
      <div className="t">
        <Badge tone={issue.severity === 'block' ? 'bad' : 'warn'}>{issue.severity === 'block' ? 'Block' : 'Warn'}</Badge>
        {issue.path.length ? (
          <button type="button" className="text-left font-semibold" style={{ border: 0, background: 'none', padding: 0, font: 'inherit', fontWeight: 600, color: 'inherit', cursor: 'pointer' }} onClick={select}>
            {issue.title}
          </button>
        ) : (
          <span>{issue.title}</span>
        )}
      </div>
      {issue.detail ? <span className="k">{issue.detail}</span> : null}
      {issue.fixes.length && c.canEdit ? (
        <div className="flex flex-wrap gap-1.5">
          {issue.fixes.map((f, i) =>
            f.kind === 'set_alt' ? (
              <form
                key={i}
                className="flex flex-1 gap-1.5"
                onSubmit={(e) => {
                  e.preventDefault()
                  void applyFix(c, issue, f, alt)
                  setAlt('')
                }}
              >
                <input aria-label="Describe the image" className="in" placeholder="Describe the image" style={{ flex: 1 }} value={alt} onChange={(e) => setAlt(e.target.value)} />
                <button type="submit" className="btn sm" disabled={!alt.trim()}>
                  {f.label}
                </button>
              </form>
            ) : (
              <button key={i} type="button" className={`btn sm${i === 0 && (f.kind === 'replace_link' || f.kind === 'insert_footer') ? ' pri' : ''}`} onClick={() => void applyFix(c, issue, f)}>
                {f.label}
              </button>
            ),
          )}
        </div>
      ) : null}
    </div>
  )
}

function DeliverabilityTab() {
  const c = useComposer()
  const all = (c.checks?.issues ?? []).filter((i) => i.severity === 'block' || i.severity === 'warn')
  const rank = (i: Issue) => (i.group === 'Subject' ? 2 : i.severity === 'block' ? 0 : 1)
  const sorted = [...all].sort((a, b) => rank(a) - rank(b))
  if (!sorted.length)
    return (
      <div className="k" style={{ padding: '14px 12px' }}>
        No warnings. Send Readiness runs the full set, including link checks, before anything goes out.
      </div>
    )
  return (
    <div className="flex flex-col">
      {sorted.map((i) => (
        <IssueRow key={`${i.id}-${i.path.join('.')}-${i.title}`} issue={i} />
      ))}
    </div>
  )
}

/** Right panel: Style · Settings · Deliverability. */
export function RightPanel() {
  const c = useComposer()
  const issues = contentIssues(c.checks?.issues)
  const tab = c.rightTab === 'deliverability' && !issues.length && !(c.checks?.issues ?? []).some((i) => i.severity === 'warn' || i.severity === 'block') ? 'style' : c.rightTab
  if (c.isEmpty) {
    return (
      <aside className="cpanel right dim" aria-hidden="true">
        <div className="ptab">
          <button type="button" className="on" tabIndex={-1}>
            Style
          </button>
          <button type="button" tabIndex={-1}>
            Settings
          </button>
        </div>
        <NothingSelected />
      </aside>
    )
  }
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  const footer = c.device === 'mobile' ? <MobileChecks selected={sel} /> : <SectionHealth selected={sel} />
  const items: { value: 'style' | 'settings' | 'deliverability'; label: string }[] = [
    { value: 'style', label: 'Style' },
    { value: 'settings', label: 'Settings' },
  ]
  const allWarn = (c.checks?.issues ?? []).filter((i) => i.severity === 'block' || i.severity === 'warn')
  if (allWarn.length) items.push({ value: 'deliverability', label: `Deliverability · ${issues.length || allWarn.length}` })
  return (
    <aside className="cpanel right" aria-label="Style, settings and deliverability">
      <PanelTabs value={tab} onChange={c.setRightTab} label="Inspector" items={items} className="flex flex-1 flex-col">
        <TabPanel value="style" className="flex flex-1 flex-col" tabIndex={-1}>
          {sel ? (
            <>
              <InspectorHead c={sel} tab="style" />
              <fieldset disabled={!c.canEdit} className="contents">
                <StyleFields key={sel.cid} c={sel} />
              </fieldset>
            </>
          ) : (
            <NothingSelected />
          )}
          {footer}
        </TabPanel>
        <TabPanel value="settings" className="flex flex-1 flex-col" tabIndex={-1}>
          {sel ? (
            <>
              <InspectorHead c={sel} tab="settings" />
              <fieldset disabled={!c.canEdit} className="contents">
                <SettingsFields key={sel.cid} c={sel} />
              </fieldset>
            </>
          ) : (
            <NothingSelected />
          )}
          {footer}
        </TabPanel>
        <TabPanel value="deliverability" className="flex flex-1 flex-col" tabIndex={-1}>
          <DeliverabilityTab />
        </TabPanel>
      </PanelTabs>
    </aside>
  )
}
