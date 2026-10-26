import type { Component } from 'grapesjs'
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from 'react'
import { http, type Schemas } from '@/api/client'
import { count } from '@/lib/format'
import { T, useT } from '@/i18n'
import { useComposer } from './context'
import { isKnownField, knownFields, suggestField } from './editor/fields'
import { buildToken, parseToken } from './editor/liquid'
import { mc } from './canvas.messages'

type Preview = Schemas['ComposerPreview']

/** The words around a chip ("Hi" before it, "," after), so its preview reads as the sentence does. */
function around(chip: Component): { before: string; after: string } {
  const sibs = chip.parent()?.components().models as Component[] | undefined
  if (!sibs) return { before: '', after: '' }
  const i = sibs.indexOf(chip)
  const text = (c: Component | undefined) => (c?.get('type') === 'textnode' ? String(c.get('content') ?? '') : '')
  const before = (text(sibs[i - 1]).match(/(\S+)\s*$/)?.[1] ?? '').slice(-16)
  const after = text(sibs[i + 1]).match(/^[,.;:!?]/)?.[0] ?? ''
  return { before, after }
}

/** The chip again after its text was re-read (a text's editing ends by rebuilding its parts). */
function liveChip(open: Opened): Component | undefined {
  if (open.chip.parent()) return open.chip
  return open.text.components().filter((x: Component) => x.get('type') === 'rc-liquid')[open.nth]
}

interface Opened {
  chip: Component
  /** The text the chip is in, and which of its chips it is. */
  text: Component
  nth: number
}

/**
 * The fallback editor of a Liquid chip (Composer board): the field, its
 * fallback, how many of the audience lack it, what a contact and someone
 * without it see, and the Liquid it saves as. Unknown fields are flagged
 * even with a fallback (CO-4): they print it for everyone.
 */
type BoxOf = (editor: NonNullable<ReturnType<typeof useComposer>['editor']>, el: Element | null | undefined, host: HTMLElement | null) => { top: number; left: number; width: number; height: number } | null

export function LiquidPopover({ host, tick, boxOf }: { host: HTMLElement | null; tick: number; boxOf: BoxOf }) {
  const c = useComposer()
  const ed = c.editor
  const [open, setOpen] = useState<Opened | null>(null)
  useEffect(() => {
    if (!ed) return
    const on = (chip: Component) => {
      let text = chip.parent()
      while (text && text.get('type') === 'textnode') text = text.parent()
      if (!text) return
      const nth = text.components().filter((x: Component) => x.get('type') === 'rc-liquid').indexOf(chip)
      setOpen({ chip, text, nth })
    }
    ed.on('rc:liquid:open', on)
    return () => {
      ed.off('rc:liquid:open', on)
    }
  }, [ed])
  const chip = open ? liveChip(open) : undefined
  const box = useMemo(() => (ed && chip ? boxOf(ed, chip.getEl(), host) : null), [ed, chip, host, tick]) // eslint-disable-line react-hooks/exhaustive-deps
  if (!ed || !open || !chip || !box) return null
  return (
    <LiquidEditor
      key={open.chip.cid}
      chip={chip}
      readOnly={!c.canEdit}
      style={{ top: box.top + box.height + 10, left: Math.max(0, box.left - 12) }}
      onClose={() => setOpen(null)}
      onApply={(token) => {
        const live = liveChip(open)
        if (!live) return
        // An edit like any other (saved, undoable).
        live.set('token', token)
        live.view?.render()
      }}
      onRemove={() => {
        liveChip(open)?.remove()
      }}
    />
  )
}

function LiquidEditor({
  chip,
  readOnly,
  style,
  onClose,
  onApply,
  onRemove,
}: {
  chip: Component
  readOnly: boolean
  style: { top: number; left: number }
  onClose: () => void
  onApply: (token: string) => void
  onRemove: () => void
}) {
  const c = useComposer()
  const t = useT()
  const id = useId()
  const parsed = useMemo(() => parseToken(String(chip.get('token') ?? '')), [chip])
  const [key, setKey] = useState(parsed.key)
  const [fallback, setFallback] = useState(parsed.fallback ?? '')
  const [preview, setPreview] = useState<Preview | null>(null)
  const ref = useRef<HTMLDivElement>(null)
  const known = useMemo(() => knownFields(c.context.tokens, c.campaign.digestRole === 'template'), [c.context.tokens, c.campaign.digestRole])
  const fields = c.context.tokens.filter((x) => (x.kind === 'contact' && x.key !== 'email') || x.kind === 'custom')
  const unknown = !isKnownField(key, known)
  const suggestion = unknown ? suggestField(key, known) : undefined
  const token = buildToken({ ...parsed, key, fallback })
  const words = around(chip)

  // How many lack the field, and what a contact sees: the preview as the composer has it.
  useEffect(() => {
    let live = true
    const shown = c.variantContent?.(c.variant) ?? { subject: c.subject, preheader: c.preheader, html: c.compileCurrent()?.html ?? '' }
    void http
      .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/preview', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: { sample: 'contact', subject: shown.subject, preheader: shown.preheader, html: shown.html },
      })
      .then(({ data }) => live && data && setPreview(data))
      .catch(() => undefined)
    return () => {
      live = false
    }
    // Once per opening.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Escape closes; the fallback field takes the focus. The canvas's text
  // editing ends on any mousedown outside it, and this editor isn't outside.
  useEffect(() => {
    const el = ref.current
    el?.querySelector<HTMLInputElement>('input')?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    const stop = (e: Event) => e.stopPropagation()
    window.addEventListener('keydown', onKey)
    el?.addEventListener('mousedown', stop)
    return () => {
      window.removeEventListener('keydown', onKey)
      el?.removeEventListener('mousedown', stop)
    }
  }, [onClose])

  const used = preview?.tokens.find((x) => x.key === key)
  const label = fields.find((x) => x.key === key)?.label ?? key
  const sentence = (value: string) => `${words.before ? `${words.before} ` : ''}${value}${words.after}`
  const b = (chunks: ReactNode[]) => <b>{chunks}</b>
  return (
    <div ref={ref} className="lqpop" role="dialog" aria-labelledby={`${id}-h`} style={style}>
      <div className="ph-h">
        <b id={`${id}-h`}>{t(mc.liquidTitle)}</b>
        <span className="k">{t(mc.liquidSaved)}</span>
      </div>
      <label className="lbl">
        {t(mc.liquidField)}
        <select className="in" value={key} disabled={readOnly} onChange={(e) => setKey(e.target.value)}>
          {fields.some((x) => x.key === key) ? null : <option value={key}>{unknown ? t(mc.liquidUnknownOption, { key }) : key}</option>}
          {fields.map((x) => (
            <option key={x.key} value={x.key}>
              {x.label}
            </option>
          ))}
        </select>
      </label>
      {unknown ? (
        <div className="w" role="note">
          <span>
            <T msg={mc.liquidUnknown} values={{ key, b }} />
            {suggestion ? ` ${t(mc.liquidDidYouMean, { suggestion })}` : ''} {fallback ? t(mc.liquidUnknownFallback, { fallback }) : t(mc.liquidUnknownNothing)}
          </span>
          {suggestion && !readOnly ? (
            <button type="button" className="btn sm" onClick={() => setKey(suggestion)}>
              {t(mc.liquidUse, { suggestion })}
            </button>
          ) : null}
        </div>
      ) : null}
      <label className="lbl">
        {t(mc.liquidFallback)}
        <input className="in" value={fallback} readOnly={readOnly} onChange={(e) => setFallback(e.target.value)} placeholder={t(mc.liquidFallbackPlaceholder)} />
        {used?.audienceMissing !== undefined && used.audienceMissing !== null && preview?.counted ? (
          <span className="k">
            {fallback
              ? t(mc.liquidMissing, { missing: used.audienceMissing, counted: count(preview.counted), label, fallback })
              : t(mc.liquidMissingNothing, { missing: used.audienceMissing, counted: count(preview.counted), label })}
          </span>
        ) : null}
      </label>
      {!unknown ? (
        <div className="lqprev">
          {preview?.contact && used && !used.missing ? (
            <span>
              <T msg={mc.liquidSees} values={{ name: preview.contact.name || preview.contact.email, text: sentence(used.value), b }} />
            </span>
          ) : null}
          <span>
            <T msg={mc.liquidWithout} values={{ key, text: sentence(fallback), b }} />
          </span>
        </div>
      ) : null}
      <code>{token}</code>
      <div className="row">
        {readOnly ? null : (
          <button
            type="button"
            className="btn sm"
            onClick={() => {
              onRemove()
              onClose()
            }}
          >
            {t(mc.liquidRemove)}
          </button>
        )}
        <button
          type="button"
          className="btn sm pri"
          onClick={() => {
            if (!readOnly && token !== String(chip.get('token') ?? '')) onApply(token)
            onClose()
          }}
        >
          {t(mc.liquidDone)}
        </button>
      </div>
    </div>
  )
}
