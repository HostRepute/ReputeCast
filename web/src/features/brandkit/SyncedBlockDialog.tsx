import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'
import { mk } from './messages'

/**
 * Edit a synced block (BrandKit · Synced blocks): its MJML, a preview the
 * server compiles, and saving it into the drafts and scheduled emails that
 * hold it.
 */
export function SyncedBlockDialog({ block, onClose }: { block: Schemas['SyncedBlock']; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const id = useId()
  // A named block (R4.A3) can be renamed here too.
  const named = /^b-[0-9a-f]{12}$/.test(block.kind)
  const [name, setName] = useState(block.name)
  const [src, setSrc] = useState(block.mjml)
  const [html, setHtml] = useState('')
  const [problem, setProblem] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  // The preview: the block in an empty email, compiled by the server.
  useEffect(() => {
    let cancelled = false
    const timer = window.setTimeout(() => {
      void http
        .POST('/api/v1/workspaces/{workspace}/mjml/compile', { params: { path: { workspace: slug } }, body: { mjml: `<mjml><mj-body>${src}</mj-body></mjml>` } })
        .then(({ data }) => {
          if (cancelled || !data) return
          setProblem(data.errors[0]?.message ?? '')
          if (!data.errors.length) setHtml(data.html)
        })
        .catch(() => {})
    }, 400)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [src, slug])
  const save = async () => {
    setBusy(true)
    setError('')
    try {
      const { data } = await http.PUT('/api/v1/workspaces/{workspace}/brand-kit/synced/{kind}', {
        params: { path: { workspace: slug, kind: block.kind } },
        body: named ? { mjml: src, name: name.trim() } : { mjml: src },
      })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/brand-kit'] })
      const a = data?.applied
      toast({
        kind: 'success',
        title: named ? t(mk.namedSaved, { name: name.trim() }) : t(mk.blockSaved, { kind: block.kind }),
        detail: a ? t(mk.blockSavedDetail, { changed: a.drafts + a.scheduled, sent: a.sent }) : undefined,
      })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      title={named ? t(mk.namedEditTitle, { name: block.name }) : t(mk.syncedEditTitle, { kind: block.kind })}
      description={t(mk.syncedEditLead)}
      size="lg"
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" onClick={() => void save()} disabled={busy || !src.trim() || (named && !name.trim())} aria-busy={busy || undefined}>
            {busy ? t(mk.savingBlock) : t(mk.saveBlock)}
          </Button>
        </>
      }
    >
      {named ? (
        <label className="lbl mb-3" htmlFor={`${id}-name`}>
          {t(mk.namedName)}
          <input id={`${id}-name`} className="in" value={name} onChange={(e) => setName(e.target.value)} maxLength={80} />
        </label>
      ) : null}
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(300px,100%),1fr))' }}>
        <label className="lbl" htmlFor={`${id}-src`}>
          {t(mk.mjml)}
          <textarea id={`${id}-src`} className="in mono" value={src} onChange={(e) => setSrc(e.target.value)} spellCheck={false} rows={14} style={{ fontSize: 12, lineHeight: 1.5, resize: 'vertical', minHeight: 240 }} aria-invalid={problem ? true : undefined} />
        </label>
        <div className="flex min-w-0 flex-col gap-1">
          <span className="lbl">{t(mk.preview)}</span>
          <iframe title={t(mk.preview)} sandbox="" srcDoc={html} className="w-full rounded-md border border-line bg-white" style={{ minHeight: 240 }} />
        </div>
      </div>
      {problem ? (
        <p className="m-0 mt-2 text-bad-fg" role="alert">
          {t(mk.compileError, { error: problem })}
        </p>
      ) : null}
      {error ? (
        <p className="m-0 mt-2 text-bad-fg" role="alert">
          {t(mk.blockFailed)}: {error}
        </p>
      ) : null}
    </Dialog>
  )
}
