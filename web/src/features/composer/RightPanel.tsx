import type { Component } from 'grapesjs'
import { useEffect, useState, type ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { LanguageBlockPanel } from '@/features/languages/composerParts'
import { useComposer, type Issue, type RightTab } from './context'
import { CommentsTab, useCommentsTabLabel } from './Comments'
import { mjChildren, sectionOf, tagOf } from './editor/labels'
import { applyFix, bodyOf, reveal } from './fixes'
import { describe, SettingsFields, StyleFields } from './Inspector'
import { SyncedNotice } from './BrandKit'
import { syncedSectionOf } from './editor/synced'
import { AccessibilityPanel } from '@/features/accessibility/composer/Panel'
import { useAccessibility } from '@/features/accessibility/composer/state'
import { accessibilityTab } from '@/features/accessibility/composer/Status'
import { FeedItemsInspector, isFeedItems, useDigestMode } from '@/features/digests/composer/DigestMode'
import { isGrowBlock } from '@/features/grow/composer/blocks'
import { GrowBlockInspector, GrowStyleNote } from '@/features/grow/composer/Panel'
import { dc } from '@/features/digests/composer/messages'
import { T, t as tt, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useWebMode, WebTabPanel } from '@/features/website/composer/campaign'
import { mc } from '@/features/website/composer/messages'
import { mc as canvasMessages } from './canvas.messages'
import { m } from './messages'

/**
 * The issues of the email on the canvas: variant A's, or, while the canvas
 * shows another variant of a content test, that variant's own (CO-6).
 */
export function canvasIssues(c: Pick<ReturnType<typeof useComposer>, 'checks' | 'abTest' | 'variant'>): Issue[] {
  if (c.abTest?.kind === 'content' && c.variant !== 'A') return c.checks?.variants?.find((v) => v.key === c.variant)?.issues ?? []
  return c.checks?.issues ?? []
}

/** Issues the Deliverability tab counts: blocks and warnings about the content (the subject has its own line). */
export function contentIssues(issues: Issue[] | undefined): Issue[] {
  return (issues ?? []).filter((i) => (i.severity === 'block' || i.severity === 'warn') && i.group !== 'Subject')
}

/** "Deliverability on this section": images with alt text and links, from the last check. */
function SectionHealth({ selected }: { selected: Component | null }) {
  const c = useComposer()
  const t = useT()
  const body = bodyOf(c)
  const sec = sectionOf(selected)
  const index = sec && body ? mjChildren(body).indexOf(sec) : -1
  const stats = index >= 0 ? c.checks?.sections.find((s) => s.index === index) : undefined
  const issues = (c.checks?.issues ?? []).filter((i) => (index >= 0 ? i.section === index : true) && (i.severity === 'block' || i.severity === 'warn') && i.group !== 'Subject')
  let tone: 'ok' | 'warn' | 'bad' | 'neutral' = 'ok'
  let text: string
  if (!c.checks) {
    tone = 'neutral'
    text = t(m.checking)
  } else if (index < 0) {
    const n = contentIssues(c.checks.issues).length
    tone = c.checks.blocks ? 'bad' : n ? 'warn' : 'ok'
    text = n ? t(m.emailIssues, { blocks: c.checks.blocks, warnings: c.checks.warnings }) : t(m.emailClean, { images: c.checks.images, links: c.checks.links })
  } else if (!stats) {
    tone = 'neutral'
    text = t(m.nothingToCheck)
  } else {
    const parts: string[] = []
    const missingAlt = stats.images - stats.imagesWithAlt
    const badLinks = stats.links - stats.linksOk
    if (stats.images) parts.push(missingAlt ? t(m.imagesMissingAlt, { missing: missingAlt, count: stats.images }) : t(m.imagesWithAlt, { count: stats.images }))
    if (stats.links) parts.push(badLinks ? t(m.linksFlagged, { flagged: badLinks, count: stats.links }) : t(m.linksOk, { count: stats.links }))
    if (!parts.length) parts.push(t(m.noImagesOrLinks))
    if (issues.some((i) => i.severity === 'block')) tone = 'bad'
    else if (issues.length || missingAlt || badLinks) tone = 'warn'
    text = parts.join(' · ')
  }
  return (
    <div className="flex flex-col gap-1.5 border-t border-line-2" style={{ padding: '10px 12px', marginTop: 'auto' }}>
      <span className="k">{index >= 0 ? t(m.sectionHealth) : t(m.deliverability)}</span>
      <Badge tone={tone} className="self-start">
        {text}
      </Badge>
    </div>
  )
}

/** "Mobile checks": tap targets and text size, measured on the mobile canvas. */
function MobileChecks({ selected }: { selected: Component | null }) {
  const c = useComposer()
  const t = useT()
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
    if (result.buttons) parts.push(result.smallButtons ? t(m.smallButtons, { count: result.smallButtons }) : t(m.buttonsOk))
    if (result.texts) parts.push(result.smallText ? t(m.smallText, { count: result.smallText }) : t(m.textOk))
  }
  return (
    <div className="flex flex-col gap-1.5 border-t border-line-2" style={{ padding: '10px 12px', marginTop: 'auto' }}>
      <span className="k">{t(m.mobileChecks)}</span>
      <Badge tone={!result ? 'neutral' : ok ? 'ok' : 'warn'} className="self-start">
        {!result ? t(m.checking) : parts.length ? parts.join(' · ') : t(m.nothingMobile)}
      </Badge>
    </div>
  )
}

function InspectorHead({ c, tab }: { c: Component; tab: 'style' | 'settings' }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
      <span className="font-semibold">{tagOf(c) || tt(canvasMessages.element)}</span>
      <span className="k">{describe(c, tab)}</span>
    </div>
  )
}

function NothingSelected() {
  const t = useT()
  return (
    <div className="k" style={{ padding: '14px 12px' }}>
      {t(m.selectHint)}
    </div>
  )
}

function IssueRow({ issue, fixable = true }: { issue: Issue; fixable?: boolean }) {
  const c = useComposer()
  const t = useT()
  const [alt, setAlt] = useState('')
  const tone = issue.severity === 'block' ? 'is-block' : issue.group === 'Subject' ? '' : 'is-warn'
  const select = () => {
    if (issue.path.length && fixable) reveal(c, issue.path)
  }
  return (
    <div className={`iss ${tone}`}>
      <div className="t">
        <Badge tone={issue.severity === 'block' ? 'bad' : 'warn'}>{issue.severity === 'block' ? t(m.severityBlock) : t(m.severityWarn)}</Badge>
        {issue.path.length && fixable ? (
          <button type="button" className="text-left font-semibold" style={{ border: 0, background: 'none', padding: 0, font: 'inherit', fontWeight: 600, color: 'inherit', cursor: 'pointer' }} onClick={select}>
            {issue.title}
          </button>
        ) : (
          <span>{issue.title}</span>
        )}
      </div>
      {issue.detail ? <span className="k">{issue.detail}</span> : null}
      {issue.fixes.length && c.canEdit && fixable ? (
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
                <input aria-label={t(m.describeImage)} className="in" placeholder={t(m.describeImage)} style={{ flex: 1 }} value={alt} onChange={(e) => setAlt(e.target.value)} />
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

/** The A/B variants' own findings, each under its variant (CO-6). */
export function variantIssueCount(checks: ReturnType<typeof useComposer>['checks']): number {
  return (checks?.variants ?? []).reduce((n, v) => n + v.issues.length, 0)
}

function VariantIssues() {
  const c = useComposer()
  const t = useT()
  const variants = (c.checks?.variants ?? []).filter((v) => v.issues.length)
  return variants.map((v) => {
    // A content variant's issues point at its own body: fix them while the canvas shows it.
    const shown = c.abTest?.kind === 'content' && c.variant === v.key
    return (
      <section key={v.key} className="flex flex-col border-t border-line-2" aria-label={t(m.variantIssues, { key: v.key })}>
        <div className="prow" style={{ padding: '10px 12px 2px' }}>
          <h3 className="m-0 text-[13px] font-semibold">{t(m.variantIssues, { key: v.key })}</h3>
          {c.variant !== v.key ? (
            <button type="button" className="btn sm" onClick={() => c.setVariant(v.key)}>
              {t(m.showVariant, { key: v.key })}
            </button>
          ) : null}
        </div>
        <span className="k" style={{ padding: '0 12px 6px' }}>
          {t(m.variantIssuesNote, { key: v.key })}
        </span>
        {v.issues.map((i) => (
          <IssueRow key={`${v.key}-${i.id}-${i.path.join('.')}-${i.title}`} issue={i} fixable={shown} />
        ))}
      </section>
    )
  })
}

/**
 * When the checks couldn't run, the last results stay with a note and a
 * Retry (ComposerWarnings): "Checks couldn't run (network) · showing
 * results from 09:31".
 */
export function ChecksRunBar() {
  const c = useComposer()
  const t = useT()
  const at = c.checks?.checkedAt
  const b = (chunks: ReactNode[]) => <b>{chunks}</b>
  if (c.checksFailure) {
    if (c.checksPending && at)
      return (
        <div className="ckfail" role="status">
          <span className="spin" aria-hidden="true" />
          <span className="t">
            <T msg={m.checksRunning} values={{ time: clock(at), b }} />
          </span>
        </div>
      )
    return (
      <div className="ckfail" role="status">
        <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flex: 'none' }}>
          <path d="M8 2 1.5 13.5h13z" />
          <path d="M8 6.5v3.2M8 11.8v.2" />
        </svg>
        <span className="t">
          <T msg={at ? m.checksFailed : m.checksFailedNone} values={{ reason: c.checksFailure.reason, time: at ? clock(at) : '', b }} />
        </span>
        <button type="button" className="btn sm" onClick={() => c.retryChecks?.()} aria-busy={c.checksPending || undefined}>
          {t(m.checksRetry)}
        </button>
      </div>
    )
  }
  if (c.checksRecovered && c.checks)
    return (
      <div className="ckfail ok" role="status">
        <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flex: 'none' }}>
          <path d="m3 8.5 3.2 3L13 4.5" />
        </svg>
        <span className="t">
          <T msg={m.checksRanOk} values={{ blocks: c.checks.blocks, warnings: c.checks.warnings, b }} />
        </span>
      </div>
    )
  return null
}

function DeliverabilityTab() {
  const c = useComposer()
  const t = useT()
  const all = (c.checks?.issues ?? []).filter((i) => i.severity === 'block' || i.severity === 'warn')
  const rank = (i: Issue) => (i.group === 'Subject' ? 2 : i.severity === 'block' ? 0 : 1)
  const sorted = [...all].sort((a, b) => rank(a) - rank(b))
  if (!sorted.length && !variantIssueCount(c.checks))
    return (
      <>
        <ChecksRunBar />
        <div className="k" style={{ padding: '14px 12px' }}>
          {t(m.noWarningsLong)}
        </div>
      </>
    )
  return (
    <div className="flex flex-col">
      <ChecksRunBar />
      {sorted.map((i) => (
        <IssueRow key={`${i.id}-${i.path.join('.')}-${i.title}`} issue={i} />
      ))}
      <VariantIssues />
    </div>
  )
}

/** Right panel: Style · Settings · Deliverability. */
export function RightPanel() {
  const c = useComposer()
  const t = useT()
  const a11y = useAccessibility()
  const digest = useDigestMode()
  const web = useWebMode()
  const commentsLabel = useCommentsTabLabel()
  const issues = contentIssues(c.checks?.issues)
  const variantCount = variantIssueCount(c.checks)
  // Checks that couldn't run keep their tab, with the stale note and Retry.
  const stale = Boolean(c.checksFailure)
  const tab =
    (c.rightTab === 'deliverability' && !stale && !issues.length && !variantCount && !(c.checks?.issues ?? []).some((i) => i.severity === 'warn' || i.severity === 'block')) || (c.rightTab === 'web' && !web)
      ? 'style'
      : c.rightTab
  if (c.isEmpty) {
    return (
      <aside className="cpanel right dim" aria-hidden="true" inert>
        <div className="ptab">
          <button type="button" className="on" disabled>
            {t(m.tabStyle)}
          </button>
          <button type="button" disabled>
            {t(m.tabSettings)}
          </button>
        </div>
        <NothingSelected />
      </aside>
    )
  }
  const sel = c.selected && tagOf(c.selected) ? c.selected : null
  // The Feed items block has its own checks (the posts it draws are not in the template).
  const footer = sel && (isFeedItems(sel) || isGrowBlock(sel)) ? null : c.device === 'mobile' ? <MobileChecks selected={sel} /> : <SectionHealth selected={sel} />
  const items: { value: RightTab; label: string }[] = [
    { value: 'style', label: t(m.tabStyle) },
    { value: 'settings', label: t(m.tabSettings) },
  ]
  // The website's Web tab (features/website), while it's on.
  if (web) items.push({ value: 'web', label: tt(mc.tabWeb) })
  const allWarn = (c.checks?.issues ?? []).filter((i) => i.severity === 'block' || i.severity === 'warn')
  if (allWarn.length || variantCount || stale) items.push({ value: 'deliverability', label: t(m.tabDeliverability, { count: (issues.length || allWarn.length) + variantCount }) })
  const a11yTab = accessibilityTab(a11y)
  if (a11yTab) items.push({ value: 'accessibility', label: a11yTab })
  // Comments (CO-10): campaigns only.
  if (c.kind !== 'transactional') items.push({ value: 'comments', label: commentsLabel })
  return (
    // Digest mode keeps the wide panel: the Feed items settings need the room.
    <aside className={tab === 'accessibility' || items.filter((i) => i.value !== 'comments').length > 3 || digest ? 'cpanel right a11y-wide' : 'cpanel right'} aria-label={t(m.rightPanelLabel)}>
      <PanelTabs value={tab} onChange={c.setRightTab} label={t(m.inspector)} items={items} className="flex flex-1 flex-col">
        <TabPanel value="style" className="flex flex-1 flex-col" tabIndex={-1}>
          {sel && isFeedItems(sel) ? (
            <div className="k" style={{ padding: '14px 12px' }}>
              {tt(dc.styleNote)}
            </div>
          ) : sel && isGrowBlock(sel) ? (
            <GrowStyleNote />
          ) : sel ? (
            <>
              <InspectorHead c={sel} tab="style" />
              {/* A synced block changes in the brand kit only (CO-11). */}
              <SyncedNotice component={sel} />
              <fieldset disabled={!c.canEdit || !!syncedSectionOf(sel)} className="contents">
                <StyleFields key={sel.cid} c={sel} />
              </fieldset>
            </>
          ) : (
            <NothingSelected />
          )}
          {footer}
        </TabPanel>
        <TabPanel value="settings" className="flex flex-1 flex-col" tabIndex={-1}>
          {sel && isFeedItems(sel) ? (
            <fieldset disabled={!c.canEdit} className="contents">
              <FeedItemsInspector key={sel.cid} c={sel} />
            </fieldset>
          ) : sel && isGrowBlock(sel) ? (
            <fieldset disabled={!c.canEdit} className="contents">
              <GrowBlockInspector key={sel.cid} c={sel} />
            </fieldset>
          ) : sel ? (
            <>
              <InspectorHead c={sel} tab="settings" />
              <SyncedNotice component={sel} />
              <fieldset disabled={!c.canEdit || !!syncedSectionOf(sel)} className="contents">
                <SettingsFields key={sel.cid} c={sel} />
              </fieldset>
              <div className="px-3 pb-3">
                <LanguageBlockPanel />
              </div>
            </>
          ) : (
            <NothingSelected />
          )}
          {footer}
        </TabPanel>
        <TabPanel value="deliverability" className="flex flex-1 flex-col" tabIndex={-1}>
          <DeliverabilityTab />
        </TabPanel>
        <TabPanel value="accessibility" className="flex flex-1 flex-col" tabIndex={-1}>
          <AccessibilityPanel />
        </TabPanel>
        <TabPanel value="web" className="flex flex-1 flex-col overflow-y-auto" tabIndex={-1}>
          <WebTabPanel />
        </TabPanel>
        <TabPanel value="comments" className="flex flex-1 flex-col overflow-y-auto" tabIndex={-1}>
          <CommentsTab thread={c.thread} />
        </TabPanel>
      </PanelTabs>
    </aside>
  )
}
