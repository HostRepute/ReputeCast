import { useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { $api, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { clock, count } from '@/lib/format'
import { useComposer, type ABTest } from './context'
import { inboxTime } from './format'
import { SendTestButton } from './SendTest'
import { usePreviewAsLabel, versionLanguage } from '@/features/languages/composerParts'
import { PreviewVersionLine, ShowWhyToggle, ThisVersionCard, VersionsCard, usePreviewVersion } from '@/features/conditional/Preview'
import { useConditional } from '@/features/conditional/state'

type Preview = Schemas['ComposerPreview']
type Client = 'gmail' | 'outlook' | 'apple' | 'yahoo'

const CLIENTS: { id: Client; label: string; short: string; dark: string }[] = [
  { id: 'gmail', label: 'Gmail · web', short: 'Gmail', dark: 'Gmail inverts light backgrounds; buttons keep their color' },
  { id: 'outlook', label: 'Outlook · Windows', short: 'Outlook', dark: 'Outlook for Windows inverts light backgrounds and dark text' },
  { id: 'apple', label: 'Apple Mail · iOS', short: 'Apple Mail', dark: 'Apple Mail inverts light backgrounds unless the email opts out' },
  { id: 'yahoo', label: 'Yahoo', short: 'Yahoo', dark: 'Yahoo inverts light backgrounds; buttons keep their color' },
]

type RGB = [number, number, number]

function parseRgb(v: string): RGB | null {
  const m = /rgba?\(\s*(\d+)[\s,]+(\d+)[\s,]+(\d+)(?:[\s,/]+([\d.]+))?\s*\)/.exec(v)
  if (!m) return null
  if (m[4] !== undefined && Number(m[4]) === 0) return null
  return [Number(m[1]), Number(m[2]), Number(m[3])]
}

function luminance([r, g, b]: RGB): number {
  const f = (c: number) => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)
}

export function contrast(a: RGB, b: RGB): number {
  const [x, y] = [luminance(a), luminance(b)].sort((p, q) => q - p) as [number, number]
  return (x + 0.05) / (y + 0.05)
}

function toHex([r, g, b]: RGB): string {
  return `#${[r, g, b].map((x) => Math.round(x).toString(16).padStart(2, '0')).join('')}`.toUpperCase()
}

function rgbToHsl([r, g, b]: RGB): [number, number, number] {
  const [R, G, B] = [r / 255, g / 255, b / 255]
  const max = Math.max(R, G, B)
  const min = Math.min(R, G, B)
  const l = (max + min) / 2
  if (max === min) return [0, 0, l]
  const d = max - min
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
  const h = max === R ? (G - B) / d + (G < B ? 6 : 0) : max === G ? (B - R) / d + 2 : (R - G) / d + 4
  return [h / 6, s, l]
}

function hslToRgb([h, s, l]: [number, number, number]): RGB {
  if (!s) return [l * 255, l * 255, l * 255]
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s
  const p = 2 * l - q
  const f = (t: number) => {
    const x = t < 0 ? t + 1 : t > 1 ? t - 1 : t
    if (x < 1 / 6) return p + (q - p) * 6 * x
    if (x < 1 / 2) return q
    if (x < 2 / 3) return p + (q - p) * (2 / 3 - x) * 6
    return p
  }
  return [f(h + 1 / 3) * 255, f(h) * 255, f(h - 1 / 3) * 255]
}

/** Flips lightness and keeps the hue, as clients darken a light colour; `floor` is the lowest lightness kept. */
function invertLightness(rgb: RGB, floor = 0.08): RGB {
  const [h, s, l] = rgbToHsl(rgb)
  return hslToRgb([h, s, Math.min(0.96, Math.max(floor, 1 - l))])
}

/** Partial inversion as Gmail and Yahoo do it: light backgrounds go dark, dark text goes light, saturated colours stay. */
function darkenDocument(doc: Document) {
  const win = doc.defaultView
  if (!win) return
  doc.querySelectorAll<HTMLElement>('body, body *').forEach((el) => {
    if (el.tagName === 'IMG') return
    const cs = win.getComputedStyle(el)
    const bg = parseRgb(cs.backgroundColor)
    if (bg && luminance(bg) > 0.45) {
      const dark = toHex(invertLightness(bg))
      el.style.setProperty('background-color', dark, 'important')
      if (el.hasAttribute('bgcolor')) el.setAttribute('bgcolor', dark)
    }
    const fg = parseRgb(cs.color)
    if (fg && luminance(fg) < 0.18) el.style.setProperty('color', toHex(invertLightness(fg, 0.82)), 'important')
  })
  doc.body.style.setProperty('background-color', '#111827', 'important')
}

function MailFrame({ html, width, dark, onDoc }: { html: string; width: number; dark?: boolean; onDoc?: (doc: Document) => void }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [scale, setScale] = useState(1)
  const [height, setHeight] = useState(400)
  const srcDoc = useMemo(() => {
    const base = '<base target="_blank">'
    return /<head[^>]*>/i.test(html) ? html.replace(/<head([^>]*)>/i, `<head$1>${base}`) : `${base}${html}`
  }, [html])
  useEffect(() => {
    const el = box.current
    if (!el) return
    const measure = () => setScale(Math.min(1, el.clientWidth / width))
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  }, [width])
  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (!doc) return
    if (dark) darkenDocument(doc)
    onDoc?.(doc)
    const fit = () => setHeight(doc.documentElement.scrollHeight)
    fit()
    doc.querySelectorAll('img').forEach((img) => img.complete || img.addEventListener('load', fit, { once: true }))
  }
  return (
    <div ref={box} className="mailcard" style={{ height: height * scale, overflow: 'hidden' }}>
      <iframe
        ref={frame}
        title={dark ? 'Email preview, dark mode' : 'Email preview'}
        sandbox="allow-same-origin allow-popups allow-popups-to-escape-sandbox"
        srcDoc={srcDoc}
        onLoad={onLoad}
        style={{ width, height, transform: `scale(${scale})` }}
      />
    </div>
  )
}

interface DarkCheck {
  tone: 'ok' | 'warn' | 'info'
  text: string
}

function useDarkChecks(doc: Document | null, assets: Schemas['MediaAsset'][] | undefined): DarkCheck[] {
  return useMemo(() => {
    if (!doc) return []
    const win = doc.defaultView
    const out: DarkCheck[] = []
    const img = doc.querySelector('img')
    if (img) {
      const src = img.getAttribute('src') ?? ''
      const asset = assets?.find((a) => src.endsWith(a.url))
      if (asset) out.push(asset.hasAlpha ? { tone: 'ok', text: `${asset.contentType === 'image/png' ? 'Logo is a transparent PNG' : 'Logo has transparency'}, readable on dark` } : { tone: 'warn', text: 'Logo has a solid background, so it shows as a light box in dark mode' })
      else out.push({ tone: 'info', text: 'Logo is hosted elsewhere; check it on a dark background' })
    }
    // Buttons: the link inside a cell with a background colour.
    let worst: { ratio: number; bg: string } | null = null
    doc.querySelectorAll('td[bgcolor] a, td[style*="background"] a').forEach((a) => {
      const fg = parseRgb(win!.getComputedStyle(a).color)
      const bgEl = (a as HTMLElement).closest('td')
      const bg = bgEl ? parseRgb(win!.getComputedStyle(bgEl).backgroundColor) : null
      if (!fg || !bg || luminance(bg) > 0.45) return
      const ratio = contrast(fg, bg)
      if (!worst || ratio < worst.ratio) worst = { ratio, bg: toHex(bg) }
    })
    if (worst) {
      const w = worst as { ratio: number; bg: string }
      out.push(w.ratio >= 4.5 ? { tone: 'ok', text: `Button text contrast ${w.ratio.toFixed(1)}:1 on ${w.bg} in both modes` } : { tone: 'warn', text: `Button text contrast is ${w.ratio.toFixed(1)}:1 on ${w.bg}; aim for 4.5:1` })
    }
    // Small visible text on a light background that clients darken while keeping the text colour (Outlook for Windows).
    let low: { ratio: number; color: string; footer: boolean } | null = null
    const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
      const el = n.parentElement
      if (!el || !n.textContent?.trim()) continue
      const cs = win!.getComputedStyle(el)
      const size = parseFloat(cs.fontSize)
      if (size >= 14 || size < 6 || !el.getClientRects().length || cs.visibility === 'hidden') continue
      const fg = parseRgb(cs.color)
      let bgNode: Element | null = el
      let bg: RGB | null = null
      while (bgNode && !bg) {
        bg = parseRgb(win!.getComputedStyle(bgNode).backgroundColor)
        bgNode = bgNode.parentElement
      }
      if (!fg || !bg || luminance(bg) < 0.45) continue
      const ratio = contrast(fg, invertLightness(bg))
      const block = el.closest('td, div')
      const footer = /unsubscribe/i.test(block?.textContent ?? '')
      if (!low || ratio < low.ratio) low = { ratio, color: toHex(fg), footer }
    }
    if (low) {
      const l = low as { ratio: number; color: string; footer: boolean }
      out.push(
        l.ratio >= 4.5
          ? { tone: 'ok', text: `Small text keeps ${l.ratio.toFixed(1)}:1 contrast on inverted backgrounds` }
          : { tone: 'warn', text: `Outlook for Windows ignores dark-mode meta; ${l.footer ? 'footer grey' : 'small text in'} ${l.color} drops to ${l.ratio.toFixed(1)}:1 on inverted background` },
      )
    }
    return out
  }, [doc, assets])
}

const PERCENTS = [10, 20, 30, 40, 50]

function ABPanel() {
  const c = useComposer()
  const t = c.abTest
  const audience = c.campaign.audienceCount ?? 0
  const start = (kind: ABTest['kind']) => {
    if (!c.canEdit) return
    const base = { testPercent: t?.testPercent ?? 20, metric: t?.metric ?? 'human_opens', durationHours: t?.durationHours ?? 2 } as const
    if (kind === 'subject') c.setABTest({ ...base, kind, variants: [{ key: 'A' }, { key: 'B', subject: '' }] })
    if (kind === 'content') c.setABTest({ ...base, kind, variants: [{ key: 'A' }, { key: 'B', mjml: c.currentMjml() }] })
    if (kind === 'send_time') c.setABTest({ ...base, kind, variants: [{ key: 'A', sendTime: '09:00' }, { key: 'B', sendTime: '14:00' }] })
  }
  const update = (next: Partial<ABTest>) => t && c.setABTest({ ...t, ...next })
  const setVariant = (key: string, v: Partial<ABTest['variants'][number]>) => t && c.setABTest({ ...t, variants: t.variants.map((x) => (x.key === key ? { ...x, ...v } : x)) })
  const addVariant = () => {
    if (!t) return start('subject')
    if (t.variants.length >= 3) return
    const key = t.variants.some((v) => v.key === 'B') ? 'C' : 'B'
    c.setABTest({ ...t, variants: [...t.variants, t.kind === 'content' ? { key, mjml: c.currentMjml() } : t.kind === 'send_time' ? { key, sendTime: '17:00' } : { key, subject: '' }] })
  }
  const share = t ? Math.round(100 / t.variants.length) : 50
  const sample = Math.round((audience * (t?.testPercent ?? 20)) / 100)
  const kinds: { kind: ABTest['kind']; title: string; text: string }[] = [
    { kind: 'subject', title: 'Subject line', text: 'Test two or three subject lines on a sample, then send the winner to everyone else.' },
    { kind: 'content', title: 'Content', text: 'Duplicate the body as variant B and edit it on the canvas. Route and audience stay shared.' },
    { kind: 'send_time', title: 'Send time', text: 'Compare 09:00 against 14:00 recipient-local, or use best time per recipient.' },
  ]
  const winnerAt = c.campaign.scheduledAt && t ? clock(new Date(new Date(c.campaign.scheduledAt).getTime() + t.durationHours * 3600_000)) : null
  return (
    <div className="flex flex-col gap-2.5" style={{ padding: '14px 16px' }}>
      <div className="flex items-center justify-between">
        <h2 className="m-0 text-[14px] font-semibold">A/B variants</h2>
        {c.canEdit ? (
          <button type="button" className="btn sm" onClick={addVariant} disabled={!!t && t.variants.length >= 3}>
            Add variant
          </button>
        ) : null}
      </div>
      {kinds.map((k) =>
        t && t.kind === k.kind ? (
          <div key={k.kind} className="card flex flex-col gap-2" style={{ padding: '10px 12px' }}>
            <div className="flex items-center justify-between">
              <span className="font-semibold">{k.title}</span>
              <Badge tone="info">Active</Badge>
            </div>
            {t.variants.map((v) =>
              t.kind === 'subject' ? (
                <label key={v.key} className="lbl">
                  {v.key} · {share}% of sample
                  <input
                    className="in"
                    value={v.key === 'A' ? c.subject : (v.subject ?? '')}
                    readOnly={!c.canEdit}
                    onChange={(e) => (v.key === 'A' ? c.setSubject(e.target.value) : setVariant(v.key, { subject: e.target.value }))}
                    placeholder="Another subject line"
                  />
                </label>
              ) : t.kind === 'send_time' ? (
                <label key={v.key} className="lbl">
                  {v.key} · {share}% of sample · recipient-local time
                  <input className="in" type="time" value={v.sendTime ?? ''} readOnly={!c.canEdit} onChange={(e) => setVariant(v.key, { sendTime: e.target.value })} />
                </label>
              ) : (
                <div key={v.key} className="flex items-center justify-between gap-2">
                  <span className="lbl" style={{ flexDirection: 'row' }}>
                    {v.key} · {share}% of sample
                  </span>
                  <button
                    type="button"
                    className="btn sm"
                    onClick={() => {
                      c.setVariant(v.key)
                      c.setView('visual')
                    }}
                  >
                    Edit on the canvas
                  </button>
                </div>
              ),
            )}
            {t.kind === 'send_time' ? (
              <label className="chk items-center">
                <input type="checkbox" checked={!!t.bestTime} disabled={!c.canEdit} onChange={(e) => update({ bestTime: e.target.checked })} />
                Use the best time per recipient instead
              </label>
            ) : null}
            <div className="grid gap-2" style={{ gridTemplateColumns: 'repeat(2,minmax(0,1fr))' }}>
              <label className="lbl">
                Sample size
                <select className="in" value={t.testPercent} disabled={!c.canEdit} onChange={(e) => update({ testPercent: Number(e.target.value) })}>
                  {PERCENTS.map((p) => (
                    <option key={p} value={p}>
                      {p}%{audience ? ` · ${count(Math.round((audience * p) / 100))} recipients` : ''}
                    </option>
                  ))}
                </select>
              </label>
              <label className="lbl">
                Winner by
                <select className="in" value={t.metric} disabled={!c.canEdit} onChange={(e) => update({ metric: e.target.value as ABTest['metric'] })}>
                  <option value="human_opens">Human opens after {t.durationHours} h</option>
                  <option value="clicks">Clicks after {t.durationHours} h</option>
                  <option value="manual">Manual pick</option>
                </select>
              </label>
            </div>
            <span className="k">
              {t.metric === 'manual'
                ? `You pick the winner; it then goes to the remaining ${count(Math.max(0, audience - sample))}.`
                : `Winner sends to the remaining ${count(Math.max(0, audience - sample))}${winnerAt ? ` at ${winnerAt}` : ` ${t.durationHours} h after the test starts`}.`}{' '}
              {t.metric === 'human_opens' ? 'Human opens exclude Apple Mail prefetches, so the winner is real.' : null}
            </span>
            {c.canEdit ? (
              <button type="button" className="k self-start" style={{ border: 0, background: 'none', padding: 0, minHeight: 24, cursor: 'pointer', textDecoration: 'underline', textUnderlineOffset: 2 }} onClick={() => c.setABTest(null)}>
                Stop testing
              </button>
            ) : null}
          </div>
        ) : (
          <div key={k.kind} className="card flex flex-col gap-1.5" style={{ padding: '10px 12px', borderStyle: 'dashed' }}>
            <div className="flex items-center justify-between">
              <span className="font-semibold">{k.title}</span>
              {c.canEdit ? (
                <button type="button" className="btn sm" onClick={() => start(k.kind)}>
                  Set up
                </button>
              ) : null}
            </div>
            <span className="k">{k.text}</span>
          </div>
        ),
      )}
    </div>
  )
}

type Sample = { sample: 'contact' | 'no_first_name' | 'random'; contactId?: string }

/**
 * Client previews (ComposerPreview): the personalized email in light and
 * dark mode, dark-mode checks, the plain-text version, the inbox line,
 * personalization values and the A/B test set-up.
 */
export function PreviewView({ onReview }: { onReview: () => void }) {
  const c = useComposer()
  const cond = useConditional()
  const [client, setClient] = useState<Client>('gmail')
  const [device, setDevice] = useState<'desktop' | 'mobile'>(c.device === 'mobile' ? 'mobile' : 'desktop')
  // "Preview this version" in the versions opens the preview as one of its contacts.
  const [sample, setSample] = useState<Sample>(() => (cond?.pendingPreview ? { sample: 'contact', contactId: cond.pendingPreview } : { sample: 'contact' }))
  const [preview, setPreview] = useState<Preview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [lightDoc, setLightDoc] = useState<Document | null>(null)
  const previewAs = usePreviewAsLabel()
  const contacts = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/preview-contacts', { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } }, { staleTime: 60_000 })
  const media = $api.useQuery('get', '/api/v1/workspaces/{workspace}/media', { params: { path: { workspace: c.slug } } }, { staleTime: 60_000 })
  const slot = c.previewSlot
  const takePreview = cond?.takePreview
  useEffect(() => {
    takePreview?.()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once, when the preview opens
  }, [])

  useEffect(() => {
    let cancelled = false
    const compiled = c.compileCurrent()
    void http
      .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/preview', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: {
          ...sample,
          html: compiled?.html,
          subject: c.subject,
          preheader: c.preheader,
          variant: c.variant === 'A' ? undefined : (c.variant as 'B' | 'C'),
          editing: versionLanguage(c.campaign),
        },
      })
      .then(({ data }) => {
        if (!cancelled && data) {
          setPreview(data)
          setError(null)
        }
      })
      .catch((e: unknown) => !cancelled && setError(e instanceof Error ? e.message : String(e)))
    return () => {
      cancelled = true
    }
  }, [sample, c.slug, c.campaign.id, c.subject, c.preheader, c.variant, c.rev]) // eslint-disable-line react-hooks/exhaustive-deps

  const meta = CLIENTS.find((x) => x.id === client)!
  const width = device === 'mobile' || client === 'apple' ? 375 : 600
  const checks = useDarkChecks(lightDoc, media.data?.items)
  const time = inboxTime(c.campaign.scheduledAt ?? new Date())
  const firstName = preview?.tokens.find((t) => t.key === 'first_name')
  const missing = preview ? preview.tokens.filter((t) => !t.value && !t.perRecipient && !t.fallbackUsed).length : 0
  // Without a pick, the server previews as the audience's first contact.
  const sampleValue = sample.sample === 'contact' ? `c:${sample.contactId ?? preview?.contact?.id ?? contacts.data?.items[0]?.id ?? ''}` : sample.sample
  // Conditional content: the version the previewed contact gets, and why.
  const version = usePreviewVersion(sample.sample === 'contact' ? (sample.contactId ?? preview?.contact?.id ?? undefined) : undefined)
  const extra = sample.contactId && !contacts.data?.items.some((ct) => ct.id === sample.contactId) ? cond?.contacts.find((ct) => ct.id === sample.contactId) : undefined
  const senderLine = (darkMode: boolean) => (
    <div className="flex flex-col gap-0.5" style={{ padding: '0 0 10px', borderBottom: `1px solid ${darkMode ? '#1F2937' : 'var(--line-2)'}`, marginBottom: 10 }}>
      <span className="text-[15px] font-semibold">{preview?.subject ?? c.subject}</span>
      <span style={{ fontSize: 12, color: darkMode ? '#9CA3AF' : 'var(--ink-3)', fontWeight: 500 }}>
        <b style={{ color: darkMode ? '#E5E7EB' : 'var(--ink)' }}>{preview?.fromName || 'Sender'}</b> &lt;{preview?.fromEmail || 'not set'}&gt; · to me · {time}
      </span>
    </div>
  )

  return (
    <div className="flex flex-1 flex-wrap items-stretch" id="main">
      {slot
        ? createPortal(
            <>
              <label className="k flex min-w-0 max-w-full items-center gap-1.5">
                Preview as
                <select
                  className="in min-w-0 max-w-full"
                  style={{ maxWidth: 'min(520px, calc(100vw - 110px))' }}
                  value={sampleValue}
                  onChange={(e) => {
                    const v = e.target.value
                    setSample(v.startsWith('c:') ? { sample: 'contact', contactId: v.slice(2) || undefined } : { sample: v as Sample['sample'] })
                  }}
                >
                  {(contacts.data?.items ?? []).map((ct) => (
                    <option key={ct.id ?? ct.email} value={`c:${ct.id ?? ''}`}>
                      {previewAs(ct) ?? `${ct.name || ct.email} · ${ct.email} · ${ct.label}`}
                    </option>
                  ))}
                  {extra ? <option value={`c:${extra.id}`}>{`${extra.name || extra.email} · ${extra.email}`}</option> : null}
                  {/* No contacts in the audience yet: the server previews as a sample person. */}
                  {!contacts.data?.items.length && preview?.contact ? (
                    <option value="c:">
                      {preview.contact.name} · {preview.contact.email} · {preview.contact.label}
                    </option>
                  ) : null}
                  <option value="no_first_name">Contact without first_name · fallback “{firstName?.token.match(/default:\s*"([^"]*)"/)?.[1] ?? 'there'}”</option>
                  <option value="random">Random Active contact</option>
                </select>
              </label>
              <ShowWhyToggle v={version} />
              <SendTestButton toMe />
              <button type="button" className="btn pri" onClick={onReview}>
                {c.campaign.access.canSend ? 'Review & send' : 'Request approval'}
              </button>
            </>,
            slot,
          )
        : null}
      <section className="cmp-main" style={{ flex: '999 1 620px' }} aria-label="Client previews">
        <div className="cmp-tools">
          <div className="seg" role="group" aria-label="Email client">
            {CLIENTS.map((x) => (
              <button key={x.id} type="button" aria-pressed={client === x.id} onClick={() => setClient(x.id)}>
                {x.label}
              </button>
            ))}
          </div>
          <div className="seg" role="group" aria-label="Device">
            {(['desktop', 'mobile'] as const).map((d) => (
              <button key={d} type="button" aria-pressed={device === d} onClick={() => setDevice(d)}>
                {d === 'desktop' ? 'Desktop' : 'Mobile'}
              </button>
            ))}
          </div>
          <span className="k">Approximated in your browser from the compiled HTML; send a test for the real client. Outlook uses Word's engine, so MJML's tables matter there.</span>
        </div>
        <PreviewVersionLine v={version} />
        <div className="grid flex-1 items-start overflow-auto" style={{ padding: 24, gridTemplateColumns: 'repeat(auto-fit,minmax(min(380px,100%),1fr))', gap: 20 }}>
          {error ? (
            <div role="alert" className="card" style={{ padding: 16, borderColor: '#FCA5A5' }}>
              <span className="font-semibold">The preview couldn't render.</span> <span className="k">{error}</span>
            </div>
          ) : null}
          <div className="card overflow-hidden">
            <div className="flex items-center justify-between border-b border-line" style={{ padding: '8px 12px' }}>
              <span className="text-[12px] font-semibold">{meta.short} · light</span>
              <span className="k">{preview ? `rendered ${clock(preview.renderedAt)}` : 'rendering…'}</span>
            </div>
            <div style={{ padding: 12, background: '#fff' }}>
              {senderLine(false)}
              {preview ? <MailFrame key={`l-${width}`} html={version.html(preview.html)} width={width} onDoc={setLightDoc} /> : <div className="sk" style={{ height: 320, width: '100%' }} />}
            </div>
          </div>
          <div className="card overflow-hidden" style={{ borderColor: '#374151' }}>
            <div className="flex items-center justify-between gap-3" style={{ padding: '8px 12px', borderBottom: '1px solid #374151', background: '#111827', color: '#E5E7EB' }}>
              <span className="text-[12px] font-semibold">{meta.short} · dark mode</span>
              <span style={{ fontSize: 12, color: '#9CA3AF' }}>{meta.dark}</span>
            </div>
            <div style={{ padding: 12, background: '#111827', color: '#E5E7EB' }}>
              {senderLine(true)}
              {preview ? <MailFrame key={`d-${width}`} html={version.html(preview.html)} width={width} dark /> : <div className="sk" style={{ height: 320, width: '100%', background: '#1F2937' }} />}
            </div>
          </div>
          <div className="card flex flex-col gap-2" style={{ padding: 12 }}>
            <span className="text-[12px] font-semibold">Dark-mode checks</span>
            {checks.length ? (
              checks.map((x, i) => (
                <div key={i} className="flex items-center gap-2" style={{ fontSize: 12.5 }}>
                  <Badge tone={x.tone}>{x.tone === 'ok' ? 'Pass' : x.tone === 'warn' ? 'Warn' : 'Info'}</Badge>
                  {x.text}
                </div>
              ))
            ) : (
              <span className="k">Checks run once the preview renders.</span>
            )}
          </div>
          <div className="card flex flex-col gap-2" style={{ padding: 12 }}>
            <span className="text-[12px] font-semibold">Plain-text version</span>
            <pre className="mono m-0" style={{ fontSize: 11.5, lineHeight: 1.5, whiteSpace: 'pre-wrap', color: 'var(--ink-nav)', overflowWrap: 'anywhere' }}>
              {preview?.text ?? ''}
            </pre>
          </div>
        </div>
      </section>
      <aside className="cpanel preview-side" aria-label="Inbox, personalization and A/B test">
        <ThisVersionCard v={version} html={preview?.html} />
        <VersionsCard v={version} />
        <div className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '14px 16px' }}>
          <h2 className="m-0 text-[14px] font-semibold">Inbox preview</h2>
          <div className="flex flex-col gap-1 rounded-lg border border-line" style={{ padding: 10, background: 'var(--ground-3)', fontSize: 12 }}>
            <div className="flex justify-between">
              <b>{preview?.fromName || c.campaign.sender?.name || 'Sender'}</b>
              <span className="k">{time}</span>
            </div>
            <span className="font-semibold">{preview?.subject ?? c.subject}</span>
            <span className="k">{preview?.preheader ?? c.preheader}</span>
          </div>
          <span className="k">
            Subject {(preview?.subject ?? c.subject).length} characters · preheader {(preview?.preheader ?? c.preheader).length} · Gmail shows about 70 + 90 on desktop, iPhone about 35 + 80.
          </span>
        </div>
        <div className="flex flex-col gap-2.5 border-b border-line-2" style={{ padding: '14px 16px' }}>
          <div className="flex items-center justify-between">
            <h2 className="m-0 text-[14px] font-semibold">Personalization</h2>
            <Badge tone={missing ? 'warn' : 'ok'}>
              {missing} missing {missing === 1 ? 'field' : 'fields'}
            </Badge>
          </div>
          <div className="flex flex-col gap-1.5" style={{ fontSize: 12.5 }}>
            {(preview?.tokens ?? []).map((t) => (
              <div key={t.token} className="flex justify-between gap-2">
                <span className="mono min-w-0" style={{ fontSize: 12, overflowWrap: 'anywhere' }}>
                  {t.token}
                </span>
                <span className="flex-none">
                  → <b>{t.perRecipient ? 'per-recipient link' : t.value || '(empty)'}</b>
                </span>
              </div>
            ))}
            {preview && !preview.tokens.length ? <span className="k">No personalization in this email.</span> : null}
          </div>
          {preview?.missingFirstName && firstName ? (
            <span className="k">
              {count(preview.missingFirstName)} {preview.missingFirstName === 1 ? 'recipient has' : 'recipients have'} no first_name and will see “{firstName.token.match(/default:\s*"([^"]*)"/)?.[1] ?? ''}”.
            </span>
          ) : null}
          {preview?.problems.length ? (
            <span className="k" style={{ color: 'var(--bad-fg)' }}>
              Line {preview.problems[0]!.line}: {preview.problems[0]!.message}
            </span>
          ) : null}
        </div>
        <ABPanel />
      </aside>
    </div>
  )
}
