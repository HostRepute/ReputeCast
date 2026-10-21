import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useBuilder } from './context'
import { m } from './messages'

/** The media library (as the composer's image picker): upload, paste a URL, or pick an image already there. */
export function MediaDialog({ open, onClose, pick }: { open: boolean; onClose: () => void; pick: (url: string) => void }) {
  const t = useT()
  const b = useBuilder()
  const qc = useQueryClient()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/media', { params: { path: { workspace: b.slug } } }, { enabled: open, retry: false })
  const [uploading, setUploading] = useState(false)
  const [drag, setDrag] = useState(false)
  const [url, setUrl] = useState('')
  const input = useRef<HTMLInputElement>(null)
  const choose = (src: string) => {
    pick(src)
    onClose()
  }
  const upload = async (files: FileList | null) => {
    const file = files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(b.slug)}/media`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
      const body = (await r.json().catch(() => ({}))) as Schemas['MediaAsset'] & { detail?: string; title?: string; errors?: { message?: string }[] }
      if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || String(r.status))
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/media'] })
      toast({ kind: 'success', title: t(m.uploaded, { name: body.fileName }) })
      choose(body.url)
    } catch (e) {
      toast({ kind: 'error', title: t(m.uploadFailed), detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setUploading(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t(m.mediaTitle)} size="lg" description={t(m.mediaLead)}>
      <div className="flex flex-col gap-3">
        <div
          className="flex flex-col items-center justify-center gap-2 rounded-[10px] border border-dashed text-center"
          style={{ padding: 20, borderColor: drag ? 'var(--accent)' : 'var(--line-strong)', background: drag ? 'var(--accent-tint)' : 'var(--ground-3)' }}
          onDragOver={(e) => {
            e.preventDefault()
            setDrag(true)
          }}
          onDragLeave={() => setDrag(false)}
          onDrop={(e) => {
            e.preventDefault()
            setDrag(false)
            void upload(e.dataTransfer.files)
          }}
        >
          <span className="font-semibold">{t(m.dropHere)}</span>
          <span className="k">{t(m.or)}</span>
          <button type="button" className="btn sm" aria-busy={uploading || undefined} disabled={uploading} onClick={() => input.current?.click()}>
            {uploading ? t(m.uploading) : t(m.upload)}
          </button>
          <input ref={input} type="file" accept="image/png,image/jpeg,image/gif" className="vh" tabIndex={-1} aria-hidden="true" onChange={(e) => void upload(e.target.files)} />
        </div>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (url.trim()) choose(url.trim())
          }}
        >
          <input className="in mono flex-1" aria-label={t(m.imageUrl)} placeholder={t(m.imageUrlPlaceholder)} value={url} onChange={(e) => setUrl(e.target.value)} />
          <button type="submit" className="btn" disabled={!/^https:\/\//.test(url.trim())}>
            {t(m.useUrl)}
          </button>
        </form>
        <span className="font-semibold">{t(m.library)}</span>
        {q.isPending ? (
          <p className="k">{t(m.libraryLoading)}</p>
        ) : q.isError ? (
          <p className="k">{t(m.libraryFailed, { error: errorMessage(q.error) })}</p>
        ) : !q.data.items.length ? (
          <p className="k">{t(m.libraryEmpty)}</p>
        ) : (
          <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(140px,1fr))' }}>
            {q.data.items.map((asset) => (
              <button
                key={asset.id}
                type="button"
                className="flex flex-col gap-1.5 rounded-[10px] border border-line bg-surface text-left"
                style={{ padding: 8, cursor: 'pointer', font: 'inherit' }}
                onClick={() => choose(asset.url)}
                aria-label={t(m.useImage, { name: asset.fileName })}
              >
                <span className="overflow-hidden rounded-md border border-line" style={{ height: 90, background: 'repeating-conic-gradient(#F3F4F6 0 25%, #fff 0 50%) 0 0/12px 12px', display: 'block' }}>
                  <img src={asset.url} alt="" style={{ width: '100%', height: '100%', objectFit: 'contain' }} loading="lazy" />
                </span>
                <span className="truncate text-[12px] font-semibold">{asset.fileName}</span>
                <span className="k">
                  {asset.width}×{asset.height}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}
