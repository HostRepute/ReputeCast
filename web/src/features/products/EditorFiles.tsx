import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type ChangeEvent, type DragEvent } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { PRODUCT, invalidateProducts, productUploads, type ProductDetail, type ProductFile } from './api'
import { pe } from './editor.messages'
import { contentWord, day, etaText, fileSize, kindName } from './format'
import { CircleIcon, FileIcon } from './parts'
import { UploadStopped, isAbort, speedMeter, uploadFile } from './upload'

type Status = 'uploading' | 'checking' | 'failed' | 'mismatch' | 'rejected'

/** A file on its way, or one the server turned down. */
interface Row {
  key: string
  file: File
  replaces?: string
  status: Status
  received: number
  uploadId?: string
  /** Bytes per second, once a chunk arrived. */
  speed?: number
  /** Why the server refused it (too big, a type that doesn't match its name). */
  error?: string
  at?: string
  controller: AbortController
}

let seq = 0

/** The editor's Files card (ProductEditor, ProductEditorStates): the drop zone, the files, and uploads in progress. */
export function EditorFiles({ detail, canEdit }: { detail: ProductDetail; canEdit: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const input = useRef<HTMLInputElement>(null)
  const replacing = useRef<string | undefined>(undefined)
  const [rows, setRows] = useState<Row[]>([])
  const [dragging, setDragging] = useState(false)
  const [said, setSaid] = useState('')
  const [removing, setRemoving] = useState<ProductFile>()
  const remove = $api.useMutation('delete', `${PRODUCT}/files/{file}`)

  const patch = (key: string, p: Partial<Row>) => setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...p } : r)))
  const drop = (key: string) => setRows((rs) => rs.filter((r) => r.key !== key))
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', PRODUCT] })
    invalidateProducts(qc)
  }

  const run = (row: Row) => {
    const api = productUploads(slug, detail.id, row.controller.signal)
    const speed = speedMeter()
    void uploadFile(api, row.file, {
      replaces: row.replaces,
      signal: row.controller.signal,
      onStart: (id) => patch(row.key, { uploadId: id }),
      onProgress: (received) => patch(row.key, { received, speed: speed(received) }),
      onChecking: () => patch(row.key, { status: 'checking', received: row.file.size }),
    }).then(
      (file) => {
        drop(row.key)
        setSaid(row.replaces ? t(pe.replaced, { name: file.name }) : t(pe.uploaded, { name: file.name }))
        refresh()
      },
      (e: unknown) => {
        const id = e instanceof UploadStopped ? e.uploadId : undefined
        if (isAbort(e)) {
          drop(row.key)
          if (id) void api.cancel(id).catch(() => undefined)
          return
        }
        const cause = e instanceof UploadStopped ? e.cause : e
        // The server's own refusals (a type that doesn't match its name, too big) say what to do.
        if (cause instanceof ApiError && cause.status < 500 && cause.status !== 409) {
          patch(row.key, { status: cause.code === 'file_mismatch' ? 'mismatch' : 'rejected', error: errorMessage(cause), at: new Date().toISOString() })
          return
        }
        // It stopped: nothing is kept, and "Try again" starts over.
        patch(row.key, { status: 'failed', received: e instanceof UploadStopped ? e.received : 0 })
        if (id) void api.cancel(id).catch(() => undefined)
      },
    )
  }

  const add = (list: FileList | File[], replaces?: string) => {
    const files = Array.from(list)
    let budget = detail.filesBytes - (replaces ? (detail.files.find((f) => f.id === replaces)?.sizeBytes ?? 0) : 0)
    const next: Row[] = []
    for (const file of files) {
      const row: Row = { key: `u${++seq}`, file, replaces, status: 'uploading', received: 0, controller: new AbortController() }
      if (file.size === 0) Object.assign(row, { status: 'rejected', error: t(pe.emptyFile) })
      else if (file.size > detail.maxFile) Object.assign(row, { status: 'rejected', error: t(pe.tooBig, { size: fileSize(t, file.size), max: fileSize(t, detail.maxFile) }) })
      else if (budget + file.size > detail.maxBytes)
        Object.assign(row, { status: 'rejected', error: t(pe.productFull, { size: fileSize(t, budget + file.size), max: fileSize(t, detail.maxBytes) }) })
      else budget += file.size
      next.push(row)
    }
    setRows((rs) => [...rs, ...next])
    for (const row of next) if (row.status === 'uploading') run(row)
  }

  const onPick = (e: ChangeEvent<HTMLInputElement>) => {
    if (e.target.files?.length) add(e.target.files, replacing.current)
    replacing.current = undefined
    e.target.value = ''
  }
  const choose = (replaces?: string) => {
    replacing.current = replaces
    input.current?.click()
  }
  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    if (canEdit && e.dataTransfer.files.length) add(e.dataTransfer.files)
  }
  const retry = (row: Row) => {
    const fresh: Row = { ...row, status: 'uploading', received: 0, speed: undefined, uploadId: undefined, error: undefined, controller: new AbortController() }
    setRows((rs) => rs.map((r) => (r.key === row.key ? fresh : r)))
    run(fresh)
  }

  return (
    <section className="card" aria-labelledby="pr-files-h">
      <CardSection id="pr-files-h" title={t(pe.files)} caption={t(pe.filesUsage, { used: fileSize(t, detail.filesBytes), max: fileSize(t, detail.maxBytes) })} />
      <div className="pad col">
        {canEdit ? (
          <div
            className={cn('pr-dz', dragging && 'on')}
            onDragOver={(e) => {
              e.preventDefault()
              setDragging(true)
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={onDrop}
          >
            <Icon name="upload" size={18} />
            <span className="font-semibold">
              {dragging ? (
                t(pe.dropActive)
              ) : (
                <T
                  msg={pe.drop}
                  values={{
                    choose: (c) => (
                      <button type="button" className="linkbtn" onClick={() => choose()}>
                        {c}
                      </button>
                    ),
                  }}
                />
              )}
            </span>
            <span className="k">{t(pe.dropHint, { file: fileSize(t, detail.maxFile), total: fileSize(t, detail.maxBytes) })}</span>
            <input ref={input} type="file" multiple className="vh" tabIndex={-1} aria-label={t(pe.chooseFiles)} onChange={onPick} />
          </div>
        ) : null}
        {detail.files.length || rows.length ? (
          <div>
            {detail.files.map((f) => (
              <FileRow key={f.id} f={f} canEdit={canEdit} onReplace={() => choose(f.id)} onRemove={() => setRemoving(f)} />
            ))}
            {rows.map((r) => (
              <UploadRow key={r.key} r={r} onCancel={() => r.controller.abort()} onRetry={() => retry(r)} onRemove={() => drop(r.key)} />
            ))}
          </div>
        ) : null}
        <span className="vh" role="status">
          {said}
        </span>
      </div>
      <ConfirmDialog
        open={Boolean(removing)}
        onOpenChange={(open) => !open && setRemoving(undefined)}
        tone="bad"
        badge={t(pe.removeBadge)}
        title={t(pe.removeTitle, { name: removing?.name ?? '' })}
        description={t(pe.removeText)}
        confirmLabel={t(pe.remove)}
        busyLabel={t(pe.removing)}
        onConfirm={async () => {
          if (!removing) return
          await remove.mutateAsync({ params: { path: { workspace: slug, product: detail.id, file: removing.id } } })
          toast({ kind: 'deleted', title: t(pe.removed, { name: removing.name }) })
          refresh()
        }}
      />
    </section>
  )
}

function FileRow({ f, canEdit, onReplace, onRemove }: { f: ProductFile; canEdit: boolean; onReplace: () => void; onRemove: () => void }) {
  const t = useT()
  const kind = kindName(t, f.kind, f.name)
  const size = fileSize(t, f.sizeBytes)
  const date = day(f.uploadedAt)
  const meta =
    f.pages > 0
      ? f.uploadedBy
        ? t(pe.fileMetaPages, { kind, pages: f.pages, size, date, name: f.uploadedBy })
        : t(pe.fileMetaPagesNoName, { kind, pages: f.pages, size, date })
      : f.uploadedBy
        ? t(pe.fileMeta, { kind, size, date, name: f.uploadedBy })
        : t(pe.fileMetaNoName, { kind, size, date })
  const content = contentWord(t, f.kind)
  return (
    <div className="pr-frow">
      <span className="fi">
        <FileIcon kind={f.kind || f.name} />
      </span>
      <span className="ft">
        <b>{f.name}</b>
        <span className="k">{meta}</span>
        {content ? (
          <span className="checked">
            <Icon name="check" size={14} strokeWidth={2.2} />
            {t(pe.checked, { content })}
          </span>
        ) : null}
      </span>
      {canEdit ? (
        <span className="acts">
          <Button size="sm" onClick={onReplace} aria-label={t(pe.replaceLabel, { name: f.name })}>
            {t(pe.replace)}
          </Button>
          <Button size="sm" iconOnly icon="trash" aria-label={t(pe.removeLabel, { name: f.name })} onClick={onRemove} />
        </span>
      ) : null}
    </div>
  )
}

function UploadRow({ r, onCancel, onRetry, onRemove }: { r: Row; onCancel: () => void; onRetry: () => void; onRemove: () => void }) {
  const t = useT()
  const name = r.file.name
  const size = fileSize(t, r.file.size)
  const done = fileSize(t, r.received)
  const pct = r.file.size ? Math.min(100, Math.floor((r.received / r.file.size) * 100)) : 0
  const icon = (
    <span className="fi">
      <FileIcon kind={name} />
    </span>
  )
  if (r.status === 'uploading' || r.status === 'checking') {
    const eta = etaText(t, r.file.size - r.received, r.speed)
    return (
      <div className="pr-frow">
        {icon}
        <span className="ft">
          <b>{name}</b>
          <span className="k">{r.status === 'checking' ? t(pe.checking) : eta ? t(pe.uploading, { done, size, eta }) : t(pe.uploadingNoEta, { done, size })}</span>
          <div className="pr-upbar" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={t(pe.upProgress)}>
            <i style={{ width: `${pct}%` }} />
          </div>
        </span>
        {r.status === 'uploading' ? (
          <Button size="sm" onClick={onCancel} aria-label={t(pe.cancelLabel, { name })}>
            {t(pe.cancel)}
          </Button>
        ) : null}
      </div>
    )
  }
  if (r.status === 'failed') {
    return (
      <div className="pr-frow">
        {icon}
        <span className="ft">
          <b>{name}</b>
          <span className="k">
            <span className="stop">{t(pe.stopped, { name, pct })}</span>
          </span>
          <div className="pr-upbar bad" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={t(pe.upStopped)}>
            <i style={{ width: `${pct}%` }} />
          </div>
          <span className="k">{t(pe.stoppedDetail, { done, size })}</span>
        </span>
        <span className="acts">
          <Button size="sm" variant="primary" onClick={onRetry}>
            {t(pe.tryAgain)}
          </Button>
          <Button size="sm" iconOnly icon="trash" aria-label={t(pe.removeLabel, { name })} onClick={onRemove} />
        </span>
      </div>
    )
  }
  // A refusal: the first sentence says what's wrong, the rest what to do.
  const msg = r.error ?? ''
  const cut = msg.search(/[.!?]\s/)
  const head = cut > 0 ? msg.slice(0, cut + 1) : msg
  const rest = cut > 0 ? msg.slice(cut + 2) : ''
  return (
    <div className="pr-frow">
      {icon}
      <span className="ft">
        <b>{name}</b>
        <span className="k">{r.status === 'mismatch' && r.at ? t(pe.uploadedAt, { time: clock(r.at), size }) : size}</span>
        <div className="note b mt-1.5" role="alert">
          <CircleIcon kind="alert" color="#B91C1C" />
          <div>
            <b>{head}</b> {rest}
          </div>
        </div>
      </span>
      <Button size="sm" variant="danger" onClick={onRemove}>
        {t(pe.remove)}
      </Button>
    </div>
  )
}
