import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { mh } from './homenav.messages'

/**
 * Picks an image for the site (a social image, a post's cover): upload one,
 * paste an address, or choose from the workspace's media library.
 */
export function ImagePickerDialog({ open, onClose, onPick, title, description }: { open: boolean; onClose: () => void; onPick: (url: string) => void; title?: string; description?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const lib = $api.useQuery('get', '/api/v1/workspaces/{workspace}/media', { params: { path: { workspace: slug } } }, { enabled: open })
  const [uploading, setUploading] = useState(false)
  const [drag, setDrag] = useState(false)
  const [url, setUrl] = useState('')
  const input = useRef<HTMLInputElement>(null)
  const choose = (src: string) => {
    onPick(src)
    onClose()
  }
  const upload = async (files: FileList | null) => {
    const file = files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/media`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
      const body = (await r.json().catch(() => ({}))) as Schemas['MediaAsset'] & { detail?: string; title?: string; errors?: { message?: string }[] }
      if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || t(mh.imageUploadStatus, { status: String(r.status) }))
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/media'] })
      toast({ kind: 'success', title: t(mh.imageUploaded, { name: body.fileName }) })
      choose(body.url)
    } catch (e) {
      toast({ kind: 'error', title: t(mh.imageUploadFailed), detail: errorMessage(e) })
    } finally {
      setUploading(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={title ?? t(mh.imageTitle)} size="lg" description={description ?? t(mh.imageText)}>
      <div className="flex flex-col gap-3">
        <div
          className="flex flex-col items-center justify-center gap-2 rounded-[10px] border border-dashed text-center"
          style={{ padding: 20, borderColor: drag ? 'var(--accent)' : 'var(--line-strong)', background: drag ? 'var(--accent-tint)' : 'var(--ground)' }}
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
          <span className="font-semibold">{t(mh.imageDrop)}</span>
          <span className="k">{t(mh.imageOr)}</span>
          <button type="button" className="btn sm" aria-busy={uploading || undefined} disabled={uploading} onClick={() => input.current?.click()}>
            {uploading ? t(mh.imageUploading) : t(mh.imageUpload)}
          </button>
          <input ref={input} type="file" accept="image/png,image/jpeg,image/gif" className="vh" tabIndex={-1} onChange={(e) => void upload(e.target.files)} />
        </div>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (url.trim()) choose(url.trim())
          }}
        >
          <input className="in mono flex-1" aria-label={t(mh.imageUrlLabel)} placeholder={t(mh.imageUrl)} value={url} onChange={(e) => setUrl(e.target.value)} />
          <button type="submit" className="btn" disabled={!/^https:\/\//.test(url.trim())}>
            {t(mh.imageUseUrl)}
          </button>
        </form>
        <span className="font-semibold">{t(mh.imageLibrary)}</span>
        {lib.isPending ? (
          <p className="k m-0">{t(mh.imageLoading)}</p>
        ) : lib.isError ? (
          <p className="k m-0">{errorMessage(lib.error)}</p>
        ) : !lib.data.items.length ? (
          <p className="k m-0">{t(mh.imageEmpty)}</p>
        ) : (
          <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(140px,1fr))' }}>
            {lib.data.items.map((a) => (
              <button
                key={a.id}
                type="button"
                className="flex flex-col gap-1.5 rounded-[10px] border border-line bg-surface text-left"
                style={{ padding: 8, cursor: 'pointer' }}
                onClick={() => choose(a.url)}
                aria-label={t(mh.imageUse, { name: a.fileName })}
              >
                <span className="overflow-hidden rounded-md border border-line" style={{ height: 90, background: 'repeating-conic-gradient(#F3F4F6 0 25%, #fff 0 50%) 0 0/12px 12px' }}>
                  <img src={a.url} alt="" style={{ width: '100%', height: '100%', objectFit: 'contain' }} loading="lazy" />
                </span>
                <span className="truncate text-[12px] font-semibold" title={a.fileName}>
                  {a.fileName}
                </span>
                <span className="k">
                  {a.width}×{a.height}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}
