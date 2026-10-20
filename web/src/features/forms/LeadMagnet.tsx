import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type ChangeEvent } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { fileSize } from '@/features/products/format'
import { FileIcon } from '@/features/products/parts'
import { UploadStopped, isAbort, uploadFile, type UploadTransport } from '@/features/products/upload'
import type { SignupForm } from './common'
import { m } from './leadmagnet.messages'
import '@/features/products/products.css'
import './leadmagnet.css'

type Download = Schemas['FormDownload']
type OfferedFile = Schemas['ProductFile']

const DOWNLOAD = '/api/v1/workspaces/{workspace}/forms/{form}/download' as const

/** The days a link may work, as the server allows them (1 to 30). */
export const DAY_OPTIONS = [1, 2, 3, 5, 7, 10, 14, 21, 30] as const

/** The choices of "Links work for", with the current value when it isn't one of them. */
export function dayOptions(current: number): number[] {
  return [...new Set([...DAY_OPTIONS, current])].filter((d) => d >= 1 && d <= 30).sort((a, b) => a - b)
}

/** The upload calls of a form's download, for uploadFile() (products/upload.ts). */
export function formUploads(workspace: string, form: string, signal?: AbortSignal): UploadTransport<Download> {
  const base = { workspace, form }
  return {
    async start(input) {
      const { data } = await http.POST(`${DOWNLOAD}/uploads`, { params: { path: base }, body: input, signal })
      return data!
    },
    async chunk(id, offset, body) {
      const { data } = await http.PUT(`${DOWNLOAD}/uploads/{upload}`, {
        params: { path: { ...base, upload: id }, query: { offset } },
        body: body as unknown as string,
        bodySerializer: (b: unknown) => b as Blob,
        headers: { 'Content-Type': 'application/octet-stream' },
        signal,
      })
      return data!
    },
    async finish(id) {
      const { data } = await http.POST(`${DOWNLOAD}/uploads/{upload}/complete`, { params: { path: { ...base, upload: id } }, signal })
      return data!
    },
    async cancel(id) {
      await http.DELETE(`${DOWNLOAD}/uploads/{upload}`, { params: { path: { ...base, upload: id } } })
    },
  }
}

type Upload = { name: string; size: number; received: number; status: 'uploading' | 'checking' | 'failed'; file: Blob & { name: string }; id?: string }

/**
 * "After confirmation: offer a download" (§6 Form builder): the file a
 * form offers on its confirmation page (a lead magnet), how many days its
 * links work, and how many were sent and downloaded. The file is uploaded
 * in chunks like a product's, and a new one replaces the old once it
 * finished. Changing it needs forms:edit; the numbers need subscribers:view.
 */
export function FormDownloadSection({ form, readOnly, doubleOptIn }: { form: SignupForm; readOnly: boolean; doubleOptIn: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const init = { params: { path: { workspace: slug, form: form.id } } }
  const q = $api.useQuery('get', DOWNLOAD, init)
  const key = $api.queryOptions('get', DOWNLOAD, init).queryKey
  const input = useRef<HTMLInputElement>(null)
  const controller = useRef<AbortController | undefined>(undefined)
  const [up, setUp] = useState<Upload>()
  const [refused, setRefused] = useState<string>()
  const [removing, setRemoving] = useState(false)
  const [savingDays, setSavingDays] = useState(false)
  const put = (d: Download) => qc.setQueryData(key, d)

  // A person who can't read the numbers (subscribers:view) doesn't see the section.
  if (q.error instanceof ApiError && q.error.status === 403) return null
  const d = q.data
  const f = d?.file

  const run = (file: Blob & { name: string }) => {
    controller.current?.abort()
    const ctl = new AbortController()
    controller.current = ctl
    const api = formUploads(slug, form.id, ctl.signal)
    setRefused(undefined)
    setUp({ name: file.name, size: file.size, received: 0, status: 'uploading', file })
    void uploadFile(api, file, {
      signal: ctl.signal,
      onStart: (id) => setUp((u) => (u ? { ...u, id } : u)),
      onProgress: (received) => setUp((u) => (u ? { ...u, received } : u)),
      onChecking: () => setUp((u) => (u ? { ...u, status: 'checking', received: file.size } : u)),
    }).then(
      (next) => {
        setUp(undefined)
        put(next)
        toast({ title: t(m.uploaded, { name: next.file?.name ?? file.name }) })
      },
      (e: unknown) => {
        const id = e instanceof UploadStopped ? e.uploadId : undefined
        if (isAbort(e)) {
          setUp(undefined)
          if (id) void api.cancel(id).catch(() => undefined)
          return
        }
        const cause = e instanceof UploadStopped ? e.cause : e
        // The server's refusals (too big, a type that doesn't match its name) say what to do.
        if (cause instanceof ApiError && cause.status < 500 && cause.status !== 409) {
          setUp(undefined)
          setRefused(errorMessage(cause))
          return
        }
        setUp((u) => (u ? { ...u, status: 'failed', received: e instanceof UploadStopped ? e.received : 0 } : u))
        if (id) void api.cancel(id).catch(() => undefined)
      },
    )
  }
  const onPick = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (file) run(file)
  }
  const cancel = () => controller.current?.abort()
  const setDays = async (days: number) => {
    setSavingDays(true)
    try {
      const { data } = await http.PUT(DOWNLOAD, { ...init, body: { days } })
      if (data) put(data)
      toast({ title: t(m.daysSaved, { days }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.daysFailed), detail: errorMessage(e) })
    } finally {
      setSavingDays(false)
    }
  }

  const pct = up && up.size ? Math.min(100, Math.floor((up.received / up.size) * 100)) : 0
  return (
    <div className="fb-sec last fl-after">
      <span className="k">{t(m.title)}</span>
      {!d ? (
        q.isError ? (
          <span className="k">
            {t(m.loadFailed, { error: errorMessage(q.error) })}{' '}
            <Button size="xs" onClick={() => void q.refetch()}>
              {t(m.retry)}
            </Button>
          </span>
        ) : null
      ) : (
        <>
          {up ? (
            <div className="pr-frow">
              <span className="fi">
                <FileIcon kind={up.name} />
              </span>
              <span className="ft">
                <b>{up.name}</b>
                <span className="k">
                  {up.status === 'checking'
                    ? t(m.checking)
                    : up.status === 'failed'
                      ? t(m.stopped, { pct })
                      : t(m.uploading, { done: fileSize(t, up.received), size: fileSize(t, up.size) })}
                </span>
                <div className={up.status === 'failed' ? 'pr-upbar bad' : 'pr-upbar'} role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={t(m.progress)}>
                  <i style={{ width: `${pct}%` }} />
                </div>
              </span>
              {up.status === 'uploading' ? (
                <Button size="sm" onClick={cancel} aria-label={t(m.cancelLabel, { name: up.name })}>
                  {t(m.cancel)}
                </Button>
              ) : up.status === 'failed' ? (
                <Button size="sm" variant="primary" onClick={() => run(up.file)}>
                  {t(m.tryAgain)}
                </Button>
              ) : null}
            </div>
          ) : f ? (
            <FileRow f={f} readOnly={readOnly} onReplace={() => input.current?.click()} onRemove={() => setRemoving(true)} />
          ) : readOnly ? (
            <span className="k">{t(m.none)}</span>
          ) : (
            <div className="flex flex-col gap-1.5">
              <span className="font-semibold">{t(m.offer)}</span>
              <span className="k">{t(m.offerHint)}</span>
              <Button size="sm" icon="upload" className="self-start" onClick={() => input.current?.click()}>
                {t(m.upload)}
              </Button>
            </div>
          )}
          {refused ? <Note tone="bad">{refused}</Note> : null}
          {f ? (
            <>
              {readOnly ? (
                <span className="k">{t(m.daysText, { days: d.days })}</span>
              ) : (
                <label className="flex items-center justify-between gap-2.5 text-[13px]">
                  {t(m.days)}
                  <select className="fb-in h-[30px]! w-[120px]!" value={d.days} disabled={savingDays} onChange={(e) => void setDays(Number(e.target.value))}>
                    {dayOptions(d.days).map((n) => (
                      <option key={n} value={n}>
                        {t(m.daysOption, { days: n })}
                      </option>
                    ))}
                  </select>
                </label>
              )}
              <span className="k">{t(m.numbers, { links: d.links, downloaded: count(d.downloaded) })}</span>
            </>
          ) : null}
          {(f || up) && !doubleOptIn ? (
            <Note tone="warn" role="note">
              {t(m.needsDoubleOptIn)}
            </Note>
          ) : null}
        </>
      )}
      {readOnly ? null : <input ref={input} type="file" className="vh" tabIndex={-1} aria-label={t(m.chooseFile)} onChange={onPick} />}
      {removing && f ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setRemoving(false)}
          badge={t(m.removeBadge)}
          title={t(m.removeTitle, { name: f.name })}
          description={t(m.removeBody, { links: d?.links ?? 0 })}
          confirmLabel={t(m.removeConfirm)}
          busyLabel={t(m.removing)}
          onConfirm={async () => {
            // The server may answer 204 without the download: read it again then.
            const { data } = await http.DELETE(DOWNLOAD, init)
            if (data) put(data)
            else await qc.invalidateQueries({ queryKey: key })
            toast({ kind: 'deleted', title: t(m.removed, { name: f.name }) })
          }}
        />
      ) : null}
    </div>
  )
}

/** The file on offer: its kind, pages and size, Replace and Remove. */
function FileRow({ f, readOnly, onReplace, onRemove }: { f: OfferedFile; readOnly: boolean; onReplace: () => void; onRemove: () => void }) {
  const t = useT()
  return (
    <div className="pr-frow">
      <span className="fi">
        <FileIcon kind={f.kind || f.name} />
      </span>
      <span className="ft">
        <b>{f.name}</b>
        <span className="k">{t(m.fileMeta, { kind: f.kindLabel || t(m.fileKind), pages: f.pages, size: fileSize(t, f.sizeBytes) })}</span>
      </span>
      {readOnly ? null : (
        <span className="acts">
          <Button size="sm" onClick={onReplace} aria-label={t(m.replaceLabel, { name: f.name })}>
            {t(m.replace)}
          </Button>
          <Button size="sm" iconOnly icon="trash" aria-label={t(m.removeLabel, { name: f.name })} onClick={onRemove} />
        </span>
      )}
    </div>
  )
}
