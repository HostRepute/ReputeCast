import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { useT } from '@/i18n'
import { useBuilder } from './context'
import { DEVICE_WIDTH, type PageDevice } from './editor/pageEditor'
import { CopyPreviewLink, previewUrl } from './Header'
import { UiIcon } from './icons'
import { m } from './messages'

type PreviewInput = Schemas['PagePreviewInput']

/**
 * A page as readers see it: the preview's document (the site's frame,
 * inert) in a sandboxed frame that runs no scripts, at the device's width
 * and scaled to fit, as tall as the page.
 */
export function PreviewFrame({ html, width, title }: { html: string; width: number; title: string }) {
  const ref = useRef<HTMLIFrameElement>(null)
  const [host, setHost] = useState<HTMLDivElement | null>(null)
  const [avail, setAvail] = useState(width)
  const [height, setHeight] = useState(900)
  useLayoutEffect(() => {
    if (!host) return
    const measure = () => setAvail(host.clientWidth)
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(host)
    return () => ro.disconnect()
  }, [host])
  useEffect(() => {
    const frame = ref.current
    if (!frame) return
    let ro: ResizeObserver | undefined
    const fit = () => {
      const doc = frame.contentDocument
      if (!doc?.documentElement) return
      setHeight(Math.max(400, doc.documentElement.scrollHeight))
      const win = frame.contentWindow as (Window & typeof globalThis) | null
      if (win?.ResizeObserver && doc.body && !ro) {
        ro = new win.ResizeObserver(() => setHeight(Math.max(400, doc.documentElement.scrollHeight)))
        ro.observe(doc.body)
      }
    }
    frame.addEventListener('load', fit)
    return () => {
      frame.removeEventListener('load', fit)
      ro?.disconnect()
    }
  }, [html])
  const scale = Math.min(1, avail / width)
  return (
    <div ref={setHost} className="pb-prevhost">
      <div className="pb-prevbox" style={{ width: Math.floor(width * scale), height: Math.ceil(height * scale) }}>
        <iframe
          ref={ref}
          title={title}
          srcDoc={html}
          // No scripts run in the preview; allow-same-origin lets the theme's fonts load.
          sandbox="allow-same-origin"
          style={{ width, height, transform: `scale(${scale})` }}
        />
      </div>
    </div>
  )
}

/** Preview (PageBuilder · Preview): the draft, a version, or what's live, at each device's width. */
export function PreviewDialog({ open, onClose, version }: { open: boolean; onClose: () => void; version?: number }) {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const [device, setDevice] = useState<PageDevice>(b.device)
  const [source, setSource] = useState<'draft' | 'live'>('draft')
  const [result, setResult] = useState<{ key: string; html?: string; error?: string } | null>(null)
  const key = `${version ?? 'canvas'}:${source}`
  useEffect(() => {
    if (!open) return
    let live = true
    const run = async () => {
      let body: PreviewInput
      if (version) {
        const { data } = await http.GET('/api/v1/workspaces/{workspace}/pages/{page}/versions/{version}', { params: { path: { workspace: b.slug, page: p.id, version } } })
        body = { html: data!.html, name: data!.name, hideHeader: data!.settings.hideHeader }
      } else if (source === 'live') {
        body = { live: true }
      } else {
        body = { html: b.pe?.getHtml() || p.draft.html, hideHeader: p.settings.hideHeader, name: p.name }
      }
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/preview', { params: { path: { workspace: b.slug, page: p.id } }, body })
      if (live) setResult({ key, html: data?.html ?? '' })
    }
    run().catch((e: unknown) => live && setResult({ key, error: errorMessage(e) }))
    return () => {
      live = false
    }
    // The preview is drawn when it opens, and again for another source.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, key])
  const shown = result?.key === key ? result : null
  const devices: { d: PageDevice; label: string }[] = [
    { d: 'desktop', label: t(m.deviceDesktop) },
    { d: 'tablet', label: t(m.deviceTablet) },
    { d: 'phone', label: t(m.devicePhone) },
  ]
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={version ? t(m.versionPreviewTitle, { version, name: p.name }) : t(m.previewTitle, { name: p.name })} description={t(m.previewLead)} size="lg" className="pb-prevdlg">
      <div className="flex flex-wrap items-center gap-2">
        <div className="seg" role="group" aria-label={t(m.deviceLabel)}>
          {devices.map((x) => (
            <button key={x.d} type="button" aria-pressed={device === x.d} onClick={() => setDevice(x.d)}>
              <UiIcon name={x.d} />
              {x.label}
            </button>
          ))}
        </div>
        {!version && p.live ? (
          <div className="seg" role="group" aria-label={t(m.preview)}>
            <button type="button" aria-pressed={source === 'draft'} onClick={() => setSource('draft')}>
              {t(m.previewDraft)}
            </button>
            <button type="button" aria-pressed={source === 'live'} onClick={() => setSource('live')}>
              {t(m.previewLive)}
            </button>
          </div>
        ) : null}
        <span className="flex-1" />
        <CopyPreviewLink small />
        <ButtonAnchor href={previewUrl(b.slug, p.id)} size="sm" iconRight="external">
          {t(m.openNewTab)}
        </ButtonAnchor>
      </div>
      {shown?.error ? (
        <p className="err">
          {t(m.previewFailed)} {shown.error}
        </p>
      ) : shown?.html !== undefined ? (
        <PreviewFrame html={shown.html} width={DEVICE_WIDTH[device]} title={t(m.previewFrame, { name: p.name })} />
      ) : (
        <div className="flex items-center gap-2 py-6" role="status">
          <span className="spin" aria-hidden="true" />
          <span className="k">{t(m.loadingCanvas)}</span>
        </div>
      )}
    </Dialog>
  )
}
