import type { Component } from 'grapesjs'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { diffWords } from '@/features/languages/compare'
import { versionLanguage } from '@/features/languages/composerParts'
import { estimateMicros } from '../format'
import { m, ms } from '../messages'
import { tags } from '../parts'
import { Refusal, Reputation, ReputationAction, sendingText, sentText, Sparkle, SuggestFlow, SuggestionCard, useOutcome, type Check } from '../suggest'
import { useComposerAssist, type Rewrite, type RewriteAction } from './state'

const ACTIONS: { action: RewriteAction; label: keyof typeof ms }[] = [
  { action: 'shorten', label: 'shorten' },
  { action: 'friendlier', label: 'friendlier' },
  { action: 'formal', label: 'formal' },
  { action: 'direct', label: 'direct' },
  { action: 'instruction', label: 'instruction' },
]

/** An mj-text's content as HTML (without the mj-text tag). */
export function innerHtml(c: Component): string {
  return c
    .toHTML()
    .replace(/^\s*<mj-text\b[^>]*>/i, '')
    .replace(/<\/mj-text>\s*$/i, '')
    .trim()
}

/** The text of some HTML, as the reader sees it. */
function plain(html: string): string {
  const d = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html')
  return (d.body.textContent ?? '').replace(/\s+/g, ' ').trim()
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// The canvas is another document: the marks carry their own styles.
const DEL = 'background:#FEE2E2;color:#991B1B;text-decoration:line-through;text-decoration-thickness:1px;border-radius:3px;padding:0 2px'
const INS = 'background:#DCFCE7;color:#166534;text-decoration:none;border-radius:3px;padding:0 2px'

type Piece = ReturnType<typeof diffWords>[number]

/**
 * Word changes read as phrases: removals and additions separated only by
 * spaces join into one struck-through phrase followed by one new phrase.
 */
export function phrases(pieces: Piece[]): Piece[] {
  const out: Piece[] = []
  let del = ''
  let add = ''
  let gap = ''
  const flush = () => {
    if (del) out.push({ kind: 'del', text: del })
    if (add) out.push({ kind: 'add', text: add })
    del = ''
    add = ''
  }
  for (const p of pieces) {
    if (p.kind === 'same') {
      if ((del || add) && /^\s+$/.test(p.text)) {
        gap += p.text
        continue
      }
      flush()
      if (gap) out.push({ kind: 'same', text: gap })
      gap = ''
      out.push(p)
      continue
    }
    if (gap) {
      if (del) del += gap
      if (add) add += gap
      gap = ''
    }
    if (p.kind === 'del') del += p.text
    else add += p.text
  }
  flush()
  if (gap) out.push({ kind: 'same', text: gap })
  return out
}

/** The suggestion marked in the text: struck through is removed, green is new. */
function diffHtml(before: string, after: string): string {
  return phrases(diffWords(plain(before), plain(after)))
    .map((p) => (p.kind === 'same' ? esc(p.text) : p.kind === 'add' ? `<ins style="${INS}">${esc(p.text)}</ins>` : `<del style="${DEL}">${esc(p.text)}</del>`))
    .join('')
}

/**
 * The rich-text toolbar's AI actions (AIComposer · Shorten): Shorten,
 * Friendlier, More formal, More direct, With an instruction…, on the text
 * block being edited. The text editing ends; the suggestion opens beside
 * the canvas.
 */
export function RewriteActions() {
  const ca = useComposerAssist()
  const c = useComposer()
  const t = useT()
  if (!ca || !ca.assist.can || c.kind === 'transactional' || !c.canEdit) return null
  const off = ca.assist.configured && !ca.assist.featureOn('rewrite')
  const go = async (action: RewriteAction) => {
    const comp = c.editor?.getEditing()
    if (!comp) return
    await c.ge?.flushEditing()
    const html = innerHtml(comp)
    ca.rewrite(
      { action, component: comp, html },
      action === 'instruction' ? null : { campaignId: c.campaign.id, action, html, language: versionLanguage(c.campaign) },
    )
  }
  const active = ca.panel?.kind === 'rewrite' ? ca.panel.action : undefined
  return (
    <>
      <span className="rsep" aria-hidden="true" />
      {off ? (
        <button type="button" aria-disabled="true" title={t(ms.rewriteOff)} className="ai-rte-off">
          <Sparkle color="#9CA3AF" />
          {t(ms.rewriteActions)}
        </button>
      ) : (
        ACTIONS.map((a, i) => (
          <button key={a.action} type="button" aria-pressed={active === a.action} onClick={() => void go(a.action)}>
            {i === 0 ? <Sparkle color="#5EEAD4" /> : null}
            {t(ms[a.label])}
          </button>
        ))
      )}
    </>
  )
}

/** Marks the suggestion in the block on the canvas while it's open; the block is drawn again from its content after. */
function useCanvasDiff(component: Component | undefined, before: string, after: string | undefined) {
  useEffect(() => {
    const el = component?.getEl()
    if (!component || !el || after === undefined) return
    // mj-text draws its content inside the column's table: mark the innermost text.
    const target = (el.querySelector('div') as HTMLElement | null) ?? el
    const original = target.innerHTML
    target.innerHTML = diffHtml(before, after)
    target.style.outline = '2px solid #1D4ED8'
    target.style.outlineOffset = '4px'
    return () => {
      target.innerHTML = original
      target.style.outline = ''
      target.style.outlineOffset = ''
    }
  }, [component, before, after])
}

function words(html: string): number {
  const s = plain(html)
  return s ? s.split(/\s+/).length : 0
}

/** The checks under a rewrite: tokens and links kept, and the length. */
function rewriteChecks(t: ReturnType<typeof useT>, r: Rewrite): Check[] {
  const out: Check[] = []
  out.push(
    r.tokens.length
      ? { key: 'tokens', ok: true, text: <T msg={ms.tokensKept} values={{ ...tags, count: r.tokens.length, names: r.tokens.join(', ') }} /> }
      : { key: 'tokens', ok: true, text: t(ms.noTokens) },
  )
  if (r.tokensRestored) out.push({ key: 'tokres', ok: false, text: t(ms.tokensRestored, { count: r.tokensRestored }) })
  out.push(r.links ? { key: 'links', ok: true, text: t(ms.linksKept, { count: r.links }) } : { key: 'links', ok: true, text: t(ms.noLinks) })
  if (r.linksRestored) out.push({ key: 'linkres', ok: false, text: t(ms.linksRestored, { count: r.linksRestored }) })
  out.push({ key: 'words', ok: true, text: t(ms.wordsChange, { before: r.wordsBefore, after: r.wordsAfter }) })
  return out
}

/** The rewrite's card beside the canvas (AIComposer · Shorten; AIStates · refusal, reputation, errors). */
export function RewritePanel() {
  const ca = useComposerAssist()!
  const c = useComposer()
  const t = useT()
  const outcome = useOutcome()
  const panel = ca.panel?.kind === 'rewrite' ? ca.panel : null
  const s = ca.rewriting.state
  const [instruction, setInstruction] = useState(panel?.instruction ?? '')
  const done: Rewrite | undefined = s.phase === 'done' ? s.data : undefined
  const showDiff = done && !done.refusal && !done.reputation ? done.html : undefined
  useCanvasDiff(panel?.component, panel?.html ?? '', showDiff)
  if (!panel) return null
  const title = panel.action === 'instruction' ? t(ms.instructionTitle) : t(ms[panel.action])
  const chars = plain(panel.html).length
  const prices = ca.assist.status?.prices ?? { input: 0, output: 0 }
  const sentMsg = panel.action === 'instruction' ? ms.sentInstruction : ms.sentParagraph
  const close = (decided?: 'rejected') => {
    if (done?.requestId && decided) outcome(done.requestId, decided)
    ca.closePanel()
  }
  const ask = (text: string) => {
    const input = { campaignId: c.campaign.id, action: panel.action, html: panel.html, instruction: text.trim() || undefined, language: versionLanguage(c.campaign) }
    ca.rewrite({ action: panel.action, component: panel.component, html: panel.html, instruction: text.trim() }, input)
  }
  const apply = (edit: boolean) => {
    if (!done) return
    // Someone typed in the block while the model was writing: their words win.
    if (innerHtml(panel.component) !== panel.html) {
      toast({ kind: 'error', title: t(ms.blockChanged) })
      return
    }
    panel.component.components(done.html)
    outcome(done.requestId, edit ? 'accepted_edits' : 'accepted')
    // The version history names it (Edit saves what the person makes of it later, as any edit).
    if (!edit) void c.save({ label: t(ms.rewriteVersion, { action: panel.action }) })
    ca.closePanel()
    c.editor?.select(panel.component)
    if (edit) window.setTimeout(() => panel.component.trigger('active'), 50)
  }
  // "With an instruction…": the instruction first.
  if (panel.action === 'instruction' && !panel.input) {
    return (
      <aside className="cpanel right ai-panel ai-panel-card" aria-label={title}>
        <SuggestionCard title={title} model={ca.assist.model} onClose={() => close()}>
          <form
            className="flex flex-col gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              if (instruction.trim()) ask(instruction)
            }}
          >
            <label className="lbl">
              {t(ms.instructionLabel)}
              <textarea className="in" rows={3} value={instruction} onChange={(e) => setInstruction(e.target.value)} placeholder={t(ms.instructionPlaceholder)} maxLength={500} autoFocus />
            </label>
            <div className="flex gap-2">
              <Button type="submit" size="sm" variant="primary" disabledReason={instruction.trim() ? undefined : t(ms.instructionPlaceholder)}>
                {t(ms.ask)}
              </Button>
              <Button size="sm" onClick={() => close()}>
                {t(m.cancel)}
              </Button>
            </div>
          </form>
        </SuggestionCard>
      </aside>
    )
  }
  return (
    <aside className="cpanel right ai-panel ai-panel-card" aria-label={title}>
      <SuggestFlow
        title={title}
        assist={ca.assist}
        s={s}
        sending={sendingText(t, sentMsg, ca.assist.status, estimateMicros(chars + (panel.instruction?.length ?? 0), Math.ceil(chars / 3), prices), { count: chars })}
        onClose={() => close()}
        onRetry={ca.rewriting.retry}
        done={(r) => {
          if (r.reputation) {
            return {
              body: <Reputation hostRepute={r.reputation.hostRepute} instruction={panel.instruction} />,
              actions: (
                <>
                  <ReputationAction hostRepute={r.reputation.hostRepute} />
                  <Button size="sm" onClick={() => close()}>
                    {t(m.close)}
                  </Button>
                </>
              ),
            }
          }
          if (r.refusal) {
            return {
              body: <Refusal model={ca.assist.model} text={r.refusal} instruction={panel.instruction} />,
              actions: (
                <>
                  {panel.action === 'instruction' ? (
                    <Button size="sm" onClick={() => ca.rewrite({ action: 'instruction', component: panel.component, html: panel.html, instruction: panel.instruction }, null)}>
                      {t(m.changeInstruction)}
                    </Button>
                  ) : null}
                  <Button size="sm" onClick={() => close()}>
                    {t(m.close)}
                  </Button>
                </>
              ),
              sent: sentText(t, sentMsg, r.sent, { count: r.sent.characters }),
            }
          }
          const before = r.wordsBefore || words(panel.html)
          const after = r.wordsAfter || words(r.html)
          return {
            body: (
              <>
                <span>{before === after && plain(r.html) === plain(panel.html) ? t(ms.unchanged) : after < before ? t(ms.shorterBy, { count: before - after, before, after }) : t(ms.rewritten, { before, after })}</span>
                <span className="k">{t(ms.editOpens)}</span>
              </>
            ),
            checks: rewriteChecks(t, { ...r, wordsBefore: before, wordsAfter: after }),
            actions: (
              <>
                <Button size="sm" variant="primary" onClick={() => apply(false)}>
                  {t(m.accept)}
                </Button>
                <Button size="sm" onClick={() => apply(true)}>
                  {t(m.editSuggestion)}
                </Button>
                <Button size="sm" onClick={() => close('rejected')}>
                  {t(m.reject)}
                </Button>
                <Button size="sm" icon="refresh" onClick={ca.rewriting.retry}>
                  {t(m.tryAgain)}
                </Button>
              </>
            ),
            sent: sentText(t, sentMsg, r.sent, { count: r.sent.characters }),
          }
        }}
      />
    </aside>
  )
}

/** The Assist menu's "Rewrite selection": the instruction for the selected text block. */
export function useRewriteSelected(): (() => void) | undefined {
  const ca = useComposerAssist()
  const c = useComposer()
  const sel = c.selected
  if (!ca || !sel || String(sel.get('type') ?? '') !== 'mj-text') return undefined
  return () => {
    void (async () => {
      await c.ge?.flushEditing()
      ca.rewrite({ action: 'instruction', component: sel, html: innerHtml(sel) }, null)
    })()
  }
}
