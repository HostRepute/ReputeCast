import { useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { $api, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { clock, count, decimal } from '@/lib/format'
import { T, t as tt, useT, type MessageDescriptor } from '@/i18n'
import { useComposer } from './context'
import { inboxTime } from './format'
import { SendTestButton } from './SendTest'
import { mb } from './blocks.messages'
import { m } from './messages'
import { mp } from './preview.messages'
import { ApproxNote, ConnectRenderingDialog, RealRendersCard, RenderDialog } from './RealRenders'
import { mr } from './renders.messages'
import { fallbackOf, PersonalizationCard } from './Personalization'
import { ABPanel, PreviewingSwitch } from './ABPanel'
import { usePreviewAsLabel, versionLanguage } from '@/features/languages/composerParts'
import { PreviewVersionLine, ShowWhyToggle, ThisVersionCard, VersionsCard, usePreviewVersion } from '@/features/conditional/Preview'
import { useConditional } from '@/features/conditional/state'

type Preview = Schemas['ComposerPreview']
type Client = 'gmail' | 'outlook' | 'apple' | 'yahoo'

const CLIENTS: { id: Client; label: string; short: string; dark: MessageDescriptor }[] = [
  { id: 'gmail', label: 'Gmail · web', short: 'Gmail', dark: mp.darkGmail },
  { id: 'outlook', label: 'Outlook · Windows', short: 'Outlook', dark: mp.darkOutlook },
  { id: 'apple', label: 'Apple Mail · iOS', short: 'Apple Mail', dark: mp.darkApple },
  { id: 'yahoo', label: 'Yahoo', short: 'Yahoo', dark: mp.darkYahoo },
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
  const t = useT()
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
        title={dark ? t(mp.frameDark) : t(mp.frame)}
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
      if (asset) out.push(asset.hasAlpha ? { tone: 'ok', text: asset.contentType === 'image/png' ? tt(mp.logoPng) : tt(mp.logoAlpha) } : { tone: 'warn', text: tt(mp.logoSolid) })
      else out.push({ tone: 'info', text: tt(mp.logoElsewhere) })
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
      out.push(
        w.ratio >= 4.5
          ? { tone: 'ok', text: tt(mp.buttonContrastOk, { ratio: decimal(w.ratio, 1), color: w.bg }) }
          : { tone: 'warn', text: tt(mp.buttonContrastLow, { ratio: decimal(w.ratio, 1), color: w.bg, target: decimal(4.5, 1) }) },
      )
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
          ? { tone: 'ok', text: tt(mp.smallTextOk, { ratio: decimal(l.ratio, 1) }) }
          : { tone: 'warn', text: tt(l.footer ? mp.smallTextFooter : mp.smallTextLow, { color: l.color, ratio: decimal(l.ratio, 1) }) },
      )
    }
    return out
  }, [doc, assets])
}

type Sample = { sample: 'contact' | 'no_first_name' | 'random'; contactId?: string }

/**
 * Client previews (ComposerPreview): the personalized email in light and
 * dark mode, dark-mode checks, the plain-text version, the inbox line,
 * personalization values and the A/B test set-up.
 */
export function PreviewView({ onReview }: { onReview: () => void }) {
  const c = useComposer()
  const tr = useT()
  const cond = useConditional()
  const [client, setClient] = useState<Client>('gmail')
  const [device, setDevice] = useState<'desktop' | 'mobile'>(c.device === 'mobile' ? 'mobile' : 'desktop')
  // "Preview this version" in the versions opens the preview as one of its contacts.
  const [sample, setSample] = useState<Sample>(() => (cond?.pendingPreview ? { sample: 'contact', contactId: cond.pendingPreview } : { sample: 'contact' }))
  const [preview, setPreview] = useState<Preview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [lightDoc, setLightDoc] = useState<Document | null>(null)
  const [renderDialog, setRenderDialog] = useState<'connect' | 'render' | null>(null)
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
    // The variant previewed, as the composer has it: its own subject, preheader and content.
    const shown = c.variantContent?.(c.variant) ?? { subject: c.subject, preheader: c.preheader, html: c.compileCurrent()?.html }
    void http
      .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/preview', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: {
          ...sample,
          html: shown.html,
          subject: shown.subject,
          preheader: shown.preheader,
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
  }, [sample, c.slug, c.campaign.id, c.subject, c.preheader, c.abTest, c.variant, c.rev]) // eslint-disable-line react-hooks/exhaustive-deps

  const meta = CLIENTS.find((x) => x.id === client)!
  // An A/B test: which variant each preview shows.
  const testing = !!c.abTest && c.abTest.kind !== 'send_time'
  const variantTag = testing ? <span className="vtag">{tr(mp.variantKey, { key: c.variant })}</span> : null
  const firstNameToken = preview?.tokens.find((t) => t.key === 'first_name')
  const firstNameFallback = firstNameToken?.token.includes('default:') ? fallbackOf(firstNameToken.token) : undefined
  const width = device === 'mobile' || client === 'apple' ? 375 : 600
  const checks = useDarkChecks(lightDoc, media.data?.items)
  const time = inboxTime(c.campaign.scheduledAt ?? new Date())
  // Without a pick, the server previews as the audience's first contact.
  const sampleValue = sample.sample === 'contact' ? `c:${sample.contactId ?? preview?.contact?.id ?? contacts.data?.items[0]?.id ?? ''}` : sample.sample
  // Conditional content: the version the previewed contact gets, and why.
  const version = usePreviewVersion(sample.sample === 'contact' ? (sample.contactId ?? preview?.contact?.id ?? undefined) : undefined)
  const extra = sample.contactId && !contacts.data?.items.some((ct) => ct.id === sample.contactId) ? cond?.contacts.find((ct) => ct.id === sample.contactId) : undefined
  const senderLine = (darkMode: boolean) => (
    <div className="flex flex-col gap-0.5" style={{ padding: '0 0 10px', borderBottom: `1px solid ${darkMode ? '#1F2937' : 'var(--line-2)'}`, marginBottom: 10 }}>
      <span className="text-[15px] font-semibold">{preview?.subject ?? c.subject}</span>
      <span style={{ fontSize: 12, color: darkMode ? '#9CA3AF' : 'var(--ink-3)', fontWeight: 500 }}>
        <T
          msg={mp.senderLine}
          values={{ name: preview?.fromName || tr(mp.sender), email: `<${preview?.fromEmail || tr(mp.notSet)}>`, time, b: (chunks) => <b style={{ color: darkMode ? '#E5E7EB' : 'var(--ink)' }}>{chunks}</b> }}
        />
      </span>
    </div>
  )

  return (
    <div className="flex flex-1 flex-wrap items-stretch" id="main">
      {slot
        ? createPortal(
            <>
              <label className="k flex min-w-0 max-w-full items-center gap-1.5">
                {tr(mp.previewAs)}
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
                  <option value="no_first_name">{tr(mp.noFirstName, { fallback: firstNameFallback ?? tr(mb.there) })}</option>
                  <option value="random">{tr(mp.randomContact)}</option>
                </select>
              </label>
              <ShowWhyToggle v={version} />
              <SendTestButton toMe />
              <button type="button" className="btn pri" onClick={onReview}>
                {c.campaign.access.canSend ? tr(m.reviewSend) : tr(m.requestApproval)}
              </button>
            </>,
            slot,
          )
        : null}
      <section className="cmp-main" style={{ flex: '999 1 620px' }} aria-label={tr(mp.clientPreviews)}>
        <div className="cmp-tools">
          <PreviewingSwitch />
          <div className="seg" role="group" aria-label={tr(mp.emailClient)}>
            {CLIENTS.map((x) => (
              <button key={x.id} type="button" aria-pressed={client === x.id} onClick={() => setClient(x.id)}>
                {x.label}
              </button>
            ))}
          </div>
          <div className="seg" role="group" aria-label={tr(mp.device)}>
            {(['desktop', 'mobile'] as const).map((d) => (
              <button key={d} type="button" aria-pressed={device === d} onClick={() => setDevice(d)}>
                {d === 'desktop' ? tr(mp.desktop) : tr(mp.mobile)}
              </button>
            ))}
          </div>
        </div>
        <ApproxNote onConnect={() => setRenderDialog('connect')} onRender={() => setRenderDialog('render')} />
        <PreviewVersionLine v={version} />
        <div className="grid flex-1 items-start overflow-auto" style={{ padding: 24, gridTemplateColumns: 'repeat(auto-fit,minmax(min(380px,100%),1fr))', gap: 20 }}>
          <RealRendersCard />
          {error ? (
            <div role="alert" className="card" style={{ padding: 16, borderColor: '#FCA5A5' }}>
              <span className="font-semibold">{tr(mp.renderFailed)}</span> <span className="k">{error}</span>
            </div>
          ) : null}
          <div className="card overflow-hidden">
            <div className="flex items-center justify-between border-b border-line" style={{ padding: '8px 12px' }}>
              <span className="text-[12px] font-semibold">
                {tr(mp.light, { client: meta.short })}
                {variantTag}
              </span>
              <span className="k">{preview ? tr(mr.approximatedAt, { time: clock(preview.renderedAt) }) : tr(mr.approximating)}</span>
            </div>
            <div style={{ padding: 12, background: '#fff' }}>
              {senderLine(false)}
              {preview ? <MailFrame key={`l-${width}`} html={version.html(preview.html)} width={width} onDoc={setLightDoc} /> : <div className="sk" style={{ height: 320, width: '100%' }} />}
            </div>
          </div>
          <div className="card overflow-hidden" style={{ borderColor: '#374151' }}>
            <div className="flex items-center justify-between gap-3" style={{ padding: '8px 12px', borderBottom: '1px solid #374151', background: '#111827', color: '#E5E7EB' }}>
              <span className="text-[12px] font-semibold">
                {tr(mp.dark, { client: meta.short })}
                {variantTag}
              </span>
              <span style={{ fontSize: 12, color: '#9CA3AF' }}>{tr(meta.dark)}</span>
            </div>
            <div style={{ padding: 12, background: '#111827', color: '#E5E7EB' }}>
              {senderLine(true)}
              {preview ? <MailFrame key={`d-${width}`} html={version.html(preview.html)} width={width} dark /> : <div className="sk" style={{ height: 320, width: '100%', background: '#1F2937' }} />}
            </div>
          </div>
          <div className="card flex flex-col gap-2" style={{ padding: 12 }}>
            <span className="text-[12px] font-semibold">{tr(mp.darkChecks)}</span>
            {checks.length ? (
              checks.map((x, i) => (
                <div key={i} className="flex items-center gap-2" style={{ fontSize: 12.5 }}>
                  <Badge tone={x.tone}>{x.tone === 'ok' ? tr(mp.pass) : x.tone === 'warn' ? tr(mp.warn) : tr(mp.info)}</Badge>
                  {x.text}
                </div>
              ))
            ) : (
              <span className="k">{tr(mp.checksPending)}</span>
            )}
          </div>
          <div className="card flex flex-col gap-2" style={{ padding: 12 }}>
            <span className="text-[12px] font-semibold">{testing ? tr(mp.plainTextVariant, { key: c.variant }) : tr(mp.plainText)}</span>
            <pre className="mono m-0" style={{ fontSize: 11.5, lineHeight: 1.5, whiteSpace: 'pre-wrap', color: 'var(--ink-nav)', overflowWrap: 'anywhere' }}>
              {preview?.text ?? ''}
            </pre>
          </div>
        </div>
      </section>
      <aside className="cpanel preview-side" aria-label={tr(mp.sideLabel)}>
        <ThisVersionCard v={version} html={preview?.html} />
        <VersionsCard v={version} />
        <div className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '14px 16px' }}>
          <h2 className="m-0 text-[14px] font-semibold">{testing ? tr(mp.inboxPreviewVariant, { key: c.variant }) : tr(mp.inboxPreview)}</h2>
          <div className="flex flex-col gap-1 rounded-lg border border-line" style={{ padding: 10, background: 'var(--ground-3)', fontSize: 12 }}>
            <div className="flex justify-between">
              <b>{preview?.fromName || c.campaign.sender?.name || tr(mp.sender)}</b>
              <span className="k">{time}</span>
            </div>
            <span className="font-semibold">{preview?.subject ?? c.subject}</span>
            <span className="k">{preview?.preheader ?? c.preheader}</span>
          </div>
          <span className="k">{tr(mp.lengths, { subject: count((preview?.subject ?? c.subject).length), preheader: count((preview?.preheader ?? c.preheader).length) })}</span>
        </div>
        <PersonalizationCard preview={preview} />
        <ABPanel />
      </aside>
      <ConnectRenderingDialog open={renderDialog === 'connect'} onClose={() => setRenderDialog(null)} />
      <RenderDialog open={renderDialog === 'render'} onClose={() => setRenderDialog(null)} />
    </div>
  )
}
