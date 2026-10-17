import type { Component, Editor } from 'grapesjs'
import { useEffect, useMemo } from 'react'
import { useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { m } from './messages'
import { componentWithRule, conditionalComponents, ruleOf } from './rules'
import { useConditional } from './state'

interface Box {
  top: number
  left: number
  width: number
  height: number
}

/** The eye-off mark of conditional blocks (badges, layers). */
export function EyeOff({ size = 12 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
      <path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.6 5.3 1.5 8 1.5 8s2.5 4.5 6.5 4.5c1.3 0 2.4-.4 3.3-1M7 3.6c.3 0 .7-.1 1-.1 4 0 6.5 4.5 6.5 4.5s-.5 1-1.5 2.1" />
    </svg>
  )
}

/**
 * The canvas toolbar's "Show all blocks · Show as a contact sees it", with
 * the contact to preview as.
 */
export function ConditionalToggle() {
  const t = useT()
  const cond = useConditional()
  if (!cond?.enabled || !cond.blocks) return null
  return (
    <div className="cv-toggle flex flex-wrap items-center gap-2">
      <div className="seg" role="group" aria-label={t(m.toggleLabel)}>
        <button type="button" aria-pressed={cond.mode === 'all'} onClick={() => cond.setMode('all')}>
          {t(m.showAll)}
        </button>
        <button type="button" aria-pressed={cond.mode === 'contact'} onClick={() => cond.setMode('contact')} disabled={!cond.contacts.length}>
          {t(m.showAsContact)}
        </button>
      </div>
      {cond.mode === 'contact' ? (
        <label className="k flex items-center gap-1.5">
          {t(m.previewAs)}
          <select className="in" value={cond.contactId} onChange={(e) => cond.setContactId(e.target.value)}>
            {cond.contacts.map((ct) => (
              <option key={ct.id} value={ct.id}>
                {ct.trait ? `${ct.name || ct.email} · ${ct.trait}` : ct.name || ct.email}
              </option>
            ))}
          </select>
        </label>
      ) : null}
    </div>
  )
}

/** Hides the blocks a contact doesn't get on the canvas, while the toggle says so. */
function useHiddenOnCanvas(editor: Editor | null, hidden: string[] | null, tick: number) {
  useEffect(() => {
    if (!editor || !hidden?.length) return
    const root = editor.getWrapper()
    const els: HTMLElement[] = []
    for (const id of hidden) {
      const comp = componentWithRule(root, id)
      const el = comp?.getEl() as HTMLElement | undefined
      if (el) {
        el.style.setProperty('display', 'none', 'important')
        els.push(el)
      }
    }
    return () => els.forEach((el) => el.style.removeProperty('display'))
  }, [editor, hidden, tick])
}

/** Badges on each conditional block: the eye-off mark and the condition in words. */
export function ConditionalOverlays({ host, tick, boxOf }: { host: HTMLElement | null; tick: number; boxOf: (editor: Editor, el: Element | null | undefined, host: HTMLElement | null) => Box | null }) {
  const t = useT()
  const c = useComposer()
  const cond = useConditional()
  const ed = c.editor
  const hidden = cond?.mode === 'contact' && cond.preview ? cond.preview.hidden.map((h) => h.id) : null
  const hiddenKey = hidden?.join(',') ?? ''
  useHiddenOnCanvas(ed, hidden, tick)
  const placed = useMemo(() => {
    if (!ed || !cond?.enabled) return []
    const out: { comp: Component; id: string; box: Box; words: string; broken: boolean }[] = []
    for (const { c: comp, rule } of conditionalComponents(ed.getWrapper())) {
      if (hidden?.includes(rule.id)) continue
      const box = boxOf(ed, comp.getEl(), host)
      if (!box) continue
      const info = cond.blockInfo(rule.id)
      out.push({ comp, id: rule.id, box, words: info?.words || t(m.title), broken: !!info?.broken.length })
    }
    return out
    // eslint-disable-next-line react-hooks/exhaustive-deps -- tick follows the canvas
  }, [ed, host, tick, cond?.report, cond?.enabled, hiddenKey])
  if (!placed.length) return null
  return (
    <>
      {placed.map((p) => (
        <button
          key={p.id}
          type="button"
          className={`cv-badge ${p.broken ? 'bad' : 'info'}`}
          style={{ top: Math.max(0, p.box.top - 11), left: p.box.left + 8 }}
          aria-label={p.broken ? t(m.badgeBroken, { words: p.words }) : t(m.badgeAria, { words: p.words })}
          onClick={() => {
            ed?.select(p.comp)
            c.setRightTab('settings')
          }}
        >
          <EyeOff />
          <span>{p.words}</span>
        </button>
      ))}
    </>
  )
}

/** The eye-off mark on a conditional block's row in Layers. */
export function LayerMark({ c: comp }: { c: Component }) {
  const t = useT()
  const cond = useConditional()
  const rule = ruleOf(comp)
  if (!cond?.enabled || !rule || (rule.show !== 'nobody' && !(rule.conditions.length && (rule.show === 'only' || rule.show === 'except')))) return null
  const info = cond.blockInfo(rule.id)
  const words = info?.words || t(m.title)
  return (
    <span className={`cv-lay${info?.broken.length ? ' bad' : ''}`} role="img" aria-label={info?.broken.length ? t(m.badgeBroken, { words }) : t(m.badgeAria, { words })} title={words}>
      <EyeOff />
    </span>
  )
}

/** The line under the layer tree: how many blocks only some contacts see. */
export function LayersNote() {
  const t = useT()
  const cond = useConditional()
  if (!cond?.enabled || !cond.blocks) return null
  return <p className="k cv-lay-note">{t(m.layersSummary, { count: cond.blocks })}</p>
}
