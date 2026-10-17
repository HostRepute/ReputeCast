import type { Component } from 'grapesjs'
import { useState, type ReactNode } from 'react'
import { http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { formatList, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { sectionOf } from '@/features/composer/editor/labels'
import { versionLanguage } from '@/features/languages/composerParts'
import { aiBase } from '../api'
import { estimateMicros } from '../format'
import { m, ms } from '../messages'
import { sendingText, sentText, Sparkle, SuggestFlow, useOutcome, useSuggest } from '../suggest'
import { useComposerAssist } from './state'

/** The words around an element (its section's text), which alt text and link text send with it. */
export function contextOf(c: Component | undefined): string {
  const scope = c ? (sectionOf(c) ?? c) : undefined
  const text = scope?.getEl()?.textContent ?? ''
  return text.replace(/\s+/g, ' ').trim().slice(0, 1500)
}

function fileOf(src: string): string {
  try {
    return decodeURIComponent(new URL(src, window.location.href).pathname.split('/').pop() || src)
  } catch {
    return src
  }
}

/** Whether Suggest alt text has a place here: the campaign composer, for someone who holds ai:use and can edit. */
export function useAltTextAvailable(): boolean {
  const ca = useComposerAssist()
  const c = useComposer()
  return Boolean(ca?.assist.can && c.canEdit && c.kind !== 'transactional')
}

/** "Decorative image": an empty alt text, so screen readers skip the image (AIComposer · image settings). */
export function DecorativeToggle({ component }: { component: Component }) {
  const t = useT()
  const c = useComposer()
  const on = (component.getAttributes() as Record<string, unknown>).alt === ''
  return (
    <label className="ai-deco" title={t(ms.decorativeHint)}>
      <input
        type="checkbox"
        checked={on}
        disabled={!c.canEdit}
        onChange={(e) => (e.target.checked ? component.addAttributes({ alt: '' }) : component.removeAttributes(['alt']))}
      />
      {t(ms.decorative)}
    </label>
  )
}

/**
 * Suggest alt text (AIComposer · alt text in the image's settings, and in
 * the accessibility panel): the image at 1,024 px and the text around it.
 * Accept writes the alt text; nothing changes until then. `beside` sits
 * next to the button (the image settings' Decorative image).
 */
export function AltTextAssist({
  component,
  src,
  onAccept,
  onEdit,
  beside,
}: {
  component?: Component
  src: string
  onAccept: (text: string) => void | Promise<unknown>
  onEdit?: (text: string) => void
  beside?: ReactNode
}) {
  const ca = useComposerAssist()
  const c = useComposer()
  const { slug } = useWorkspace()
  const t = useT()
  const outcome = useOutcome()
  const s = useSuggest<{ src: string; context: string }, Schemas['AIAltText']>(async (v, signal) =>
    (await http.POST(`${aiBase}/alt-text`, { params: { path: { workspace: slug } }, body: { campaignId: c.campaign.id, src: v.src, context: v.context, language: versionLanguage(c.campaign) }, signal })).data!,
  )
  const [open, setOpen] = useState(false)
  if (!ca || !ca.assist.can || !c.canEdit || c.kind === 'transactional') return null
  const a = ca.assist
  const noImage = a.configured && a.status && !a.status.imageInput
  const off = a.configured && !noImage && !a.featureOn('alt_text')
  const ask = () => {
    setOpen(true)
    void s.start({ src, context: contextOf(component) })
  }
  const close = (decided?: 'rejected') => {
    if (decided && s.state.phase === 'done') outcome(s.state.data.requestId, decided)
    s.cancel()
    setOpen(false)
  }
  const prices = a.status?.prices ?? { input: 0, output: 0 }
  const file = fileOf(src)
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          className="ai-assist"
          onClick={ask}
          disabledReason={!src ? t(ms.altNoSrc) : noImage ? t(m.altNeedsImages) : off ? t(ms.featureOff) : undefined}
        >
          <Sparkle />
          {t(ms.suggestAlt)}
        </Button>
        {beside}
      </div>
      {noImage ? <span className="k">{a.status?.helpers.length ? t(ms.altNeedsImageModel, { helpers: formatList(a.status.helpers, 'disjunction') }) : t(m.altNeedsImages)}</span> : null}
      {open ? (
        <SuggestFlow
          title={t(ms.altTitle)}
          assist={a}
          s={s.state}
          sending={sendingText(t, ms.sentAlt, a.status, estimateMicros(1500, 60, prices) + 1600 * prices.input, { file })}
          onClose={() => close(s.state.phase === 'done' ? 'rejected' : undefined)}
          onRetry={s.retry}
          done={(r) => ({
            body: r.refusal ? <span>{t(ms.quoted, { text: r.refusal })}</span> : <span>{r.text}</span>,
            checks: r.refusal
              ? undefined
              : [
                  { key: 'len', ok: r.short, text: r.short ? t(ms.characters, { count: r.characters }) : t(ms.longAlt) },
                  { key: 'what', ok: r.saysWhat, text: t(ms.saysWhat) },
                  { key: 'of', ok: r.noImageOf, text: r.noImageOf ? t(ms.noImageOf) : t(ms.startsImageOf) },
                ],
            actions: r.refusal ? (
              <Button size="sm" onClick={() => close()}>
                {t(m.close)}
              </Button>
            ) : (
              <>
                <Button
                  size="sm"
                  variant="primary"
                  onClick={() => {
                    outcome(r.requestId, 'accepted')
                    void onAccept(r.text)
                    setOpen(false)
                  }}
                >
                  {t(m.accept)}
                </Button>
                {onEdit ? (
                  <Button
                    size="sm"
                    onClick={() => {
                      outcome(r.requestId, 'accepted_edits')
                      onEdit(r.text)
                      setOpen(false)
                    }}
                  >
                    {t(m.editSuggestion)}
                  </Button>
                ) : null}
                <Button size="sm" onClick={() => close('rejected')}>
                  {t(m.reject)}
                </Button>
                <Button size="sm" icon="refresh" onClick={s.retry}>
                  {t(m.tryAgain)}
                </Button>
              </>
            ),
            sent: sentText(t, ms.sentAlt, r.sent, { file: r.fileName || file }),
          })}
        />
      ) : null}
    </div>
  )
}

/**
 * Suggest a fix for link text that says nothing (AIComposer · accessibility
 * fix): the link text, its address and the sentence around it. The address
 * stays the same.
 */
export function LinkTextAssist({ component, text, href, onAccept }: { component?: Component; text: string; href: string; onAccept: (text: string) => void | Promise<unknown> }) {
  const ca = useComposerAssist()
  const c = useComposer()
  const { slug } = useWorkspace()
  const t = useT()
  const outcome = useOutcome()
  const s = useSuggest<{ context: string }, Schemas['AILinkText']>(async (v, signal) =>
    (await http.POST(`${aiBase}/link-text`, { params: { path: { workspace: slug } }, body: { campaignId: c.campaign.id, text, href, context: v.context, language: versionLanguage(c.campaign) }, signal })).data!,
  )
  const [open, setOpen] = useState(false)
  if (!ca || !ca.assist.can || !c.canEdit || c.kind === 'transactional') return null
  const a = ca.assist
  const off = a.configured && !a.featureOn('accessibility_fix')
  const close = (decided?: 'rejected') => {
    if (decided && s.state.phase === 'done') outcome(s.state.data.requestId, decided)
    s.cancel()
    setOpen(false)
  }
  const prices = a.status?.prices ?? { input: 0, output: 0 }
  const context = contextOf(component)
  return (
    <>
      <Button
        size="sm"
        className="ai-assist"
        onClick={() => {
          setOpen(true)
          void s.start({ context })
        }}
        disabledReason={off ? t(ms.featureOff) : undefined}
      >
        <Sparkle />
        {t(ms.suggestFix)}
      </Button>
      {open ? (
        <div className="basis-full">
          <SuggestFlow
            title={t(ms.fixTitle)}
            assist={a}
            s={s.state}
            sending={sendingText(t, ms.sentLink, a.status, estimateMicros(context.length + text.length + href.length, 40, prices))}
            onClose={() => close(s.state.phase === 'done' ? 'rejected' : undefined)}
            onRetry={s.retry}
            done={(r) => ({
              body: r.refusal ? (
                <span>{t(ms.quoted, { text: r.refusal })}</span>
              ) : (
                <span>
                  <del className="ai-del">{text}</del> <ins className="ai-ins">{r.text}</ins>
                </span>
              ),
              checks: r.refusal
                ? undefined
                : [
                    { key: 'where', ok: r.saysWhere, text: r.saysWhere ? t(ms.saysWhere) : t(ms.saysWhereNot) },
                    { key: 'href', ok: true, text: t(ms.sameLink, { href: (r.href || href).replace(/^https?:\/\//, '') }) },
                  ],
              actions: r.refusal ? (
                <Button size="sm" onClick={() => close()}>
                  {t(m.close)}
                </Button>
              ) : (
                <>
                  <Button
                    size="sm"
                    variant="primary"
                    onClick={() => {
                      outcome(r.requestId, 'accepted')
                      void onAccept(r.text)
                      setOpen(false)
                    }}
                  >
                    {t(m.accept)}
                  </Button>
                  <Button size="sm" onClick={() => close('rejected')}>
                    {t(m.reject)}
                  </Button>
                  <Button size="sm" icon="refresh" onClick={s.retry}>
                    {t(m.tryAgain)}
                  </Button>
                </>
              ),
              sent: sentText(t, ms.sentLink, r.sent),
            })}
          />
        </div>
      ) : null}
    </>
  )
}
