import { useQueryClient } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { clock, shortDate } from '@/lib/format'
import { useComposer, type ComposerDialog } from './context'
import { diffLines } from './diff'
import { sectionLabel, sectionOf } from './editor/labels'
import { formatMjml } from './editor/mjmlDoc'
import { fileSize, kb, readText } from './format'
import { t as tt, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { sectionWord } from './editor/labels'
import { md } from './dialogs.messages'
import { m } from './messages'

type Version = Schemas['ComposerVersion']

function sameDay(a: Date, b: Date) {
  return a.toDateString() === b.toDateString()
}

function when(at: string) {
  const d = new Date(at)
  return sameDay(d, new Date()) ? clock(d) : shortDate(d)
}

type VersionPage = Schemas['ComposerVersionList']

/** One version of the history: what it is, and restore or name it. */
function VersionRow({ v, pick, onPick, busy, onRestore, onNamed }: { v: Version; pick: boolean; onPick: () => void; busy: number | null; onRestore: (v: Version) => void; onNamed: () => void }) {
  const c = useComposer()
  const t = useT()
  const [naming, setNaming] = useState(false)
  const [name, setName] = useState(v.name)
  const [saving, setSaving] = useState(false)
  const label = v.source === 'create' && v.label.startsWith('created') ? v.label : v.label || t(md.source, { source: v.source })
  const submit = async () => {
    setSaving(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions/{version}/name', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id, version: v.version } },
        body: { name: name.trim() },
      })
      setNaming(false)
      onNamed()
    } catch (e) {
      toast({ kind: 'error', title: t(md.nameFailed), detail: errorMessage(e) })
    } finally {
      setSaving(false)
    }
  }
  return (
    <div className={`ver${v.current ? ' cur' : ''}`}>
      <input type="checkbox" aria-label={t(md.compareVersion, { version: v.version })} checked={pick} onChange={onPick} />
      <span className="mono k" style={{ width: 38 }}>
        v{v.version}
      </span>
      {naming ? (
        <form
          className="flex min-w-0 flex-[1_1_220px] items-center gap-1.5"
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
        >
          <input className="in min-w-0 flex-1" aria-label={t(md.versionName, { version: v.version })} value={name} maxLength={120} placeholder={t(md.namePlaceholder)} onChange={(e) => setName(e.target.value)} autoFocus />
          <button type="submit" className="btn sm pri" aria-busy={saving || undefined} disabled={saving}>
            {t(md.saveName)}
          </button>
          <button type="button" className="btn sm" onClick={() => setNaming(false)}>
            {t(common.cancel)}
          </button>
        </form>
      ) : (
        <span className="min-w-0 flex-[1_1_220px]">
          {v.current ? <b>{t(md.current)}</b> : null}
          {v.current ? ' · ' : ''}
          {v.name ? <b className="vname">{v.name}</b> : null}
          {v.name ? ' · ' : ''}
          {when(v.savedAt)}
          {v.savedBy ? ` · ${v.savedBy}` : ''} · {label}
          <span className="k"> · {kb(v.sizeBytes)}</span>
        </span>
      )}
      {naming ? null : (
        <span className="flex flex-wrap items-center gap-1.5">
          {c.canEdit ? (
            <button
              type="button"
              className="btn sm"
              aria-label={v.name ? t(md.renameVersion, { version: v.version }) : t(md.nameVersion, { version: v.version })}
              onClick={() => {
                setName(v.name)
                setNaming(true)
              }}
            >
              {v.name ? t(md.rename) : t(md.nameButton)}
            </button>
          ) : null}
          {v.current ? (
            <Badge tone="ok">{t(md.liveDraft)}</Badge>
          ) : c.canEdit ? (
            <button type="button" className="btn sm" aria-label={t(md.restoreVersion, { version: v.version })} aria-busy={busy === v.version || undefined} disabled={busy !== null} onClick={() => onRestore(v)}>
              {t(md.restore)}
            </button>
          ) : null}
        </span>
      )}
    </div>
  )
}

/**
 * Version history (ComposerStates · Saved): every version, newest first, a
 * page at a time, or only the named ones; restore one (unsaved edits are
 * saved as a version first), name one, or tick two to compare them.
 */
function VersionsDialog({ open, onClose, onCompare }: { open: boolean; onClose: () => void; onCompare: (a: number, b: number) => void }) {
  const c = useComposer()
  const t = useT()
  const qc = useQueryClient()
  const [named, setNamed] = useState(false)
  const query = { workspace: c.slug, campaignId: c.campaign.id }
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions', { params: { path: query, query: named ? { named: true } : {} } }, { enabled: open })
  // Older pages, fetched with "Show older versions" (the first comes from the query).
  const [older, setOlder] = useState<{ key: string; pages: VersionPage[] }>({ key: '', pages: [] })
  const key = `${named}:${q.dataUpdatedAt}`
  const pages = older.key === key ? older.pages : []
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [busy, setBusy] = useState<number | null>(null)
  const [pick, setPick] = useState<number[]>([])
  const last = pages.length ? pages[pages.length - 1] : q.data
  const items = q.data ? [...q.data.items, ...pages.flatMap((p) => p.items)] : []
  const refresh = () => void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions'] })
  const loadOlder = async () => {
    if (!last?.nextBefore) return
    setLoadingOlder(true)
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions', { params: { path: query, query: { before: last.nextBefore, ...(named ? { named: true } : {}) } } })
      if (data) setOlder({ key, pages: [...pages, data] })
    } catch (e) {
      toast({ kind: 'error', title: t(md.loadVersionsFailed), detail: errorMessage(e) })
    } finally {
      setLoadingOlder(false)
    }
  }
  const restore = async (v: Version) => {
    setBusy(v.version)
    try {
      // Unsaved edits become a version of their own first, so a restore never drops them.
      if (c.hasUnsaved?.() && !(await c.save({ manual: true, quiet: true, label: t(md.beforeRestore, { version: v.version }) }))) {
        toast({ kind: 'error', title: t(md.restoreFailed), detail: t(md.restoreNeedsSave, { version: v.version }) })
        return
      }
      const base = c.currentVersion?.() ?? (c.status.kind === 'new' ? 1 : c.status.version)
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions/{version}/restore', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id, version: v.version } },
        body: { baseVersion: base },
      })
      await c.reloadFromServer()
      refresh()
      toast({ kind: 'success', title: t(md.restored, { version: v.version }), detail: t(md.restoredDetail) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(md.restoreFailed), detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }
  const toggle = (n: number) => setPick((p) => (p.includes(n) ? p.filter((x) => x !== n) : [...p.slice(-1), n]))
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={t(md.versionsTitle, { name: c.name || t(m.untitled) })}
      size="lg"
      footer={
        <>
          <span className="k mr-auto">{pick.length === 2 ? t(md.compareWith, { a: Math.min(...pick), b: Math.max(...pick) }) : t(md.tickTwo)}</span>
          <button type="button" className="btn" disabled={pick.length !== 2} onClick={() => onCompare(Math.min(...pick), Math.max(...pick))}>
            {t(md.compareTwo)}
          </button>
        </>
      }
    >
      <div className="seg mb-2 self-start" role="group" aria-label={t(md.filterLabel)}>
        <button type="button" aria-pressed={!named} onClick={() => setNamed(false)}>
          {q.data && !named ? t(md.allVersions, { count: q.data.total }) : t(md.allVersionsBare)}
        </button>
        <button type="button" aria-pressed={named} onClick={() => setNamed(true)}>
          {q.data ? t(md.namedVersions, { count: q.data.named }) : t(md.namedVersionsBare)}
        </button>
      </div>
      {q.isPending ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <span className="vh" role="status">
            {t(md.loadingVersions)}
          </span>
          {[0, 1, 2].map((i) => (
            <span key={i} className="sk" style={{ height: 36 }} />
          ))}
        </div>
      ) : q.isError ? (
        <p className="k" role="alert">
          {t(md.historyFailed, { error: errorMessage(q.error) })}
        </p>
      ) : !items.length ? (
        <p className="k">{named ? t(md.noNamed) : t(md.noVersions)}</p>
      ) : (
        <div className="-mx-1 flex flex-col">
          {items.map((v) => (
            <VersionRow key={v.version} v={v} pick={pick.includes(v.version)} onPick={() => toggle(v.version)} busy={busy} onRestore={(x) => void restore(x)} onNamed={refresh} />
          ))}
          {last?.nextBefore ? (
            <div className="flex items-center gap-3 px-3.5 pt-3">
              <button type="button" className="btn sm" aria-busy={loadingOlder || undefined} disabled={loadingOlder} onClick={() => void loadOlder()}>
                {t(md.showOlder)}
              </button>
              <span className="k">{t(md.showing, { count: items.length, total: named ? q.data.named : q.data.total })}</span>
            </div>
          ) : null}
        </div>
      )}
    </Dialog>
  )
}

type CompareTexts = { a: string; b: string; la: string; lb: string; sections: string[] }

/** Loads and diffs two versions, or the latest save against the unsaved edits after a conflict. */
function CompareBody({ versions }: { versions: [number, number] | 'conflict' }) {
  const c = useComposer()
  const t = useT()
  const { slug } = c
  const campaignId = c.campaign.id
  // What the conflict compares against, taken when the dialog opens.
  const [mine] = useState(() => ({ mjml: c.currentMjml(), theirs: c.status.kind === 'conflict' ? c.status.theirs : null }))
  const [texts, setTexts] = useState<CompareTexts | null>(null)
  useEffect(() => {
    let cancelled = false
    const get = (v: number) => http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions/{version}', { params: { path: { workspace: slug, campaignId, version: v } } }).then((r) => r.data!)
    const run = async (): Promise<CompareTexts> => {
      if (versions === 'conflict') {
        const latest = mine.theirs?.version ?? 0
        const theirs = await get(latest)
        return { a: formatMjml(theirs.mjml), b: formatMjml(mine.mjml), la: `v${latest} · ${theirs.savedBy}`, lb: tt(md.yourEdits), sections: mine.theirs?.theirChanges ?? [] }
      }
      const [x, y] = await Promise.all([get(versions[0]), get(versions[1])])
      const head = (v: typeof x) => `${tt(md.diffSubject, { value: v.subject })}\n${tt(md.diffPreheader, { value: v.preheader })}\n\n`
      return { a: `${head(x)}${formatMjml(x.mjml)}`, b: `${head(y)}${formatMjml(y.mjml)}`, la: `v${x.version} · ${when(x.savedAt)}`, lb: `v${y.version} · ${when(y.savedAt)}`, sections: y.sections }
    }
    run()
      .then((r) => !cancelled && setTexts(r))
      .catch((e: unknown) => toast({ kind: 'error', title: tt(md.loadVersionsFailed), detail: errorMessage(e) }))
    return () => {
      cancelled = true
    }
  }, [versions, slug, campaignId, mine])
  const lines = useMemo(() => (texts ? diffLines(texts.a, texts.b) : []), [texts])
  return (
    <>
      {texts ? <p className="k m-0">{`${texts.la} → ${texts.lb}`}</p> : null}
      {!texts ? (
        <p className="k">{t(common.loading)}</p>
      ) : (
        <div className="flex flex-col gap-2">
          {texts.sections.length ? <span className="k">{t(md.sections, { list: texts.sections.map(sectionWord).join(', ') })}</span> : null}
          <div className="mono overflow-auto rounded-lg border border-line" style={{ maxHeight: '60vh', fontSize: 12, lineHeight: 1.6 }}>
            {lines.map((l, i) => (
              <div
                key={i}
                style={{
                  whiteSpace: 'pre',
                  padding: '0 10px',
                  background: l.kind === 'add' ? '#DCFCE7' : l.kind === 'del' ? '#FEE2E2' : undefined,
                  color: l.kind === 'fold' ? 'var(--ink-3)' : undefined,
                }}
              >
                <span aria-hidden="true" style={{ display: 'inline-block', width: 14 }}>
                  {l.kind === 'add' ? '+' : l.kind === 'del' ? '−' : ' '}
                </span>
                <span className="vh">{l.kind === 'add' ? t(md.added) : l.kind === 'del' ? t(md.removed) : ''}</span>
                {l.text}
              </div>
            ))}
            {lines.every((l) => l.kind === 'same' || l.kind === 'fold') ? <p className="k p-3">{t(md.noDifferences)}</p> : null}
          </div>
        </div>
      )}
    </>
  )
}

function CompareDialog({ open, onClose, versions }: { open: boolean; onClose: () => void; versions: [number, number] | 'conflict' | null }) {
  const t = useT()
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t(md.compareTitle)} size="lg">
      {open && versions ? <CompareBody key={String(versions)} versions={versions} /> : null}
    </Dialog>
  )
}

/** What the server did to an upload, in one line ("Resized from 4032×3024 (3.1 MB) for email"); empty when it's stored as uploaded or barely changed. */
export function optimizedNote(t: ReturnType<typeof useT>, a: Schemas['MediaAsset']): string {
  const o = a.original
  if (!o) return ''
  const size = fileSize(o.sizeBytes)
  if (o.contentType !== a.contentType) return t(md.convertedFrom, { size, type: o.contentType.replace('image/', '').toUpperCase() })
  if (o.width !== a.width || o.height !== a.height) return t(md.resizedFrom, { width: o.width, height: o.height, size })
  return a.sizeBytes < o.sizeBytes * 0.95 ? t(md.compressedFrom, { size }) : ''
}

type Picked = { src: string; asset?: Schemas['MediaAsset'] }

/**
 * The alt text prompt after an image is chosen (CO-12, ComposerWrite ·
 * image block): describe it, or say it's decorative (an empty alt, which
 * screen readers skip). Either clears the "No alt text" warning.
 */
function AltStep({ component, picked, onDone }: { component: Component; picked: Picked; onDone: () => void }) {
  const t = useT()
  const id = useId()
  const before = (component.getAttributes() as Record<string, unknown>).alt
  const [alt, setAlt] = useState(typeof before === 'string' ? before : '')
  const name = picked.asset?.fileName ?? (picked.src.split('?')[0]!.split('/').pop() || picked.src)
  const save = (value: string) => {
    component.addAttributes({ alt: value })
    onDone()
  }
  const a = picked.asset
  const note = a ? optimizedNote(t, a) : ''
  const done = alt.trim().length > 0
  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault()
        if (done) save(alt.trim())
      }}
    >
      <div className="flex items-start gap-3">
        <img
          src={picked.src}
          alt=""
          className="shrink-0 rounded-md border border-line"
          style={{ width: 120, height: 80, objectFit: 'contain', background: 'repeating-conic-gradient(#F3F4F6 0 25%, #fff 0 50%) 0 0/12px 12px' }}
        />
        <div className="flex min-w-0 flex-col items-start gap-1">
          <span className="max-w-full truncate font-semibold" title={name}>
            {name}
          </span>
          {a ? <span className="k">{t(md.imageMeta, { width: a.width, height: a.height, size: kb(a.sizeBytes) })}</span> : null}
          {note ? <span className="k">{note}</span> : null}
          <Badge tone={done ? 'ok' : 'warn'} size="sm">
            {done ? t(md.altAdded) : t(md.altMissing)}
          </Badge>
        </div>
      </div>
      <label className="vh" htmlFor={id}>
        {t(md.altFor, { name })}
      </label>
      <input id={id} className="in" autoFocus placeholder={t(md.altPlaceholder)} value={alt} onChange={(e) => setAlt(e.target.value)} />
      <div className="flex flex-wrap items-center gap-2">
        <button type="button" className="btn sm" onClick={() => save('')}>
          {t(md.altDecorative)}
        </button>
        <span className="flex-1" />
        <button type="button" className="btn" onClick={onDone}>
          {t(md.altSkip)}
        </button>
        <button type="submit" className="btn pri" disabled={!done}>
          {t(md.altSave)}
        </button>
      </div>
    </form>
  )
}

function AssetsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const c = useComposer()
  const t = useT()
  const qc = useQueryClient()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/media', { params: { path: { workspace: c.slug } } }, { enabled: open })
  const [uploading, setUploading] = useState(false)
  const [drag, setDrag] = useState(false)
  const [url, setUrl] = useState('')
  const [picked, setPicked] = useState<Picked | null>(null)
  // A prompt left open when the dialog closed doesn't come back next time.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (!open) setPicked(null)
  }
  const input = useRef<HTMLInputElement>(null)
  const target = c.assetTarget
  const close = () => {
    setPicked(null)
    onClose()
  }
  const choose = (src: string, asset?: Schemas['MediaAsset']) => {
    const comp = target?.component as Component | undefined
    if (comp) {
      comp.addAttributes({ [target!.attr]: src })
      if (target!.attr === 'src') {
        comp.set('src', src)
        // An image needs alt text, or to be marked decorative: ask now.
        if (c.canEdit) {
          setPicked({ src, asset })
          return
        }
      }
    }
    close()
  }
  const upload = async (files: FileList | null) => {
    const file = files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(c.slug)}/media`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
      const body = (await r.json().catch(() => ({}))) as Schemas['MediaAsset'] & { detail?: string; title?: string; errors?: { message?: string }[] }
      if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || t(md.uploadFailedStatus, { status: r.status }))
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/media'] })
      toast({ kind: 'success', title: t(md.uploaded, { name: body.fileName }), detail: optimizedNote(t, body) || undefined })
      if (target) choose(body.url, body)
    } catch (e) {
      toast({ kind: 'error', title: t(md.uploadFailed), detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setUploading(false)
    }
  }
  const remove = async (a: Schemas['MediaAsset']) => {
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/media/{assetId}', { params: { path: { workspace: c.slug, assetId: a.id } } })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/media'] })
      toast({ kind: 'info', title: t(md.removedFromLibrary, { name: a.fileName }), detail: t(md.removedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(md.removeFailed), detail: errorMessage(e) })
    }
  }
  if (picked && target) {
    return (
      <Dialog open={open} onOpenChange={(o) => !o && close()} title={t(md.altTitle)} description={t(md.altLead)}>
        <AltStep component={target.component as Component} picked={picked} onDone={close} />
      </Dialog>
    )
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && close()} title={target?.attr === 'background-url' ? t(md.chooseBackground) : t(md.chooseImage)} size="lg" description={t(md.assetsLead)}>
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
          <span className="font-semibold">{t(md.dropHere)}</span>
          <span className="k">{t(md.or)}</span>
          <button type="button" className="btn sm" aria-busy={uploading || undefined} disabled={!c.canEdit || uploading} onClick={() => input.current?.click()}>
            {uploading ? t(md.uploading) : t(md.upload)}
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
          <input className="in mono flex-1" aria-label={t(md.imageUrl)} placeholder={t(md.pasteUrl)} value={url} onChange={(e) => setUrl(e.target.value)} />
          <button type="submit" className="btn" disabled={!/^https:\/\//.test(url.trim())}>
            {t(md.useUrl)}
          </button>
        </form>
        <span className="font-semibold">{t(md.library)}</span>
        {q.isPending ? (
          <p className="k">{t(md.loadingImages)}</p>
        ) : q.isError ? (
          <p className="k">{t(md.libraryFailed, { error: errorMessage(q.error) })}</p>
        ) : !q.data.items.length ? (
          <p className="k">{t(md.noImages)}</p>
        ) : (
          <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(140px,1fr))' }}>
            {q.data.items.map((a) => (
              <div key={a.id} className="flex flex-col gap-1.5 rounded-[10px] border border-line bg-surface" style={{ padding: 8 }}>
                <button type="button" className="overflow-hidden rounded-md border border-line" style={{ height: 90, padding: 0, background: 'repeating-conic-gradient(#F3F4F6 0 25%, #fff 0 50%) 0 0/12px 12px', cursor: 'pointer' }} onClick={() => choose(a.url, a)} disabled={!target} aria-label={t(md.useImage, { name: a.fileName })}>
                  <img src={a.url} alt="" style={{ width: '100%', height: '100%', objectFit: 'contain' }} loading="lazy" />
                </button>
                <span className="truncate text-[12px] font-semibold" title={a.fileName}>
                  {a.fileName}
                </span>
                <span className="k">{t(a.hasAlpha ? md.imageMetaTransparent : md.imageMeta, { width: a.width, height: a.height, size: kb(a.sizeBytes) })}</span>
                {optimizedNote(t, a) ? <span className="k text-[12px]">{optimizedNote(t, a)}</span> : null}
                {c.canEdit ? (
                  <button type="button" className="btn sm self-start" onClick={() => void remove(a)}>
                    {t(md.remove)}
                  </button>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}

function htmlToMjml(html: string): string {
  const body = /<body[^>]*>([\s\S]*)<\/body>/i.exec(html)?.[1] ?? html
  return `<mjml>\n  <mj-body width="600px">\n    <mj-raw>\n${body.trim()}\n    </mj-raw>\n  </mj-body>\n</mjml>\n`
}

function ImportDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const c = useComposer()
  const t = useT()
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const kind = /<mjml[\s>]/i.test(text) ? 'mjml' : text.trim() ? 'html' : ''
  const submit = () => {
    if (!kind) return
    const src = kind === 'mjml' ? text : htmlToMjml(text)
    if (kind === 'mjml' && !/<mj-body[\s>]/i.test(text)) {
      setError(t(md.noBody, { tag: '<mj-body>' }))
      return
    }
    c.loadContent(src, { source: 'import', label: kind === 'mjml' ? t(md.noteImportedMjml) : t(md.noteImportedHtml) })
    setText('')
    setError('')
    onClose()
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={t(md.importTitle)}
      description={t(md.importLead)}
      footer={
        <>
          <button type="button" className="btn" onClick={onClose}>
            {t(common.cancel)}
          </button>
          <button type="button" className="btn pri" disabled={!kind || !c.canEdit} onClick={submit}>
            {kind === 'html' ? t(md.importRaw) : t(md.import)}
          </button>
        </>
      }
    >
      <div className="flex flex-col gap-2">
        <label className="lbl">
          {t(md.paste)}
          <textarea className="in mono" rows={12} value={text} onChange={(e) => setText(e.target.value)} placeholder={'<mjml>…</mjml>'} />
        </label>
        <label className="k inline-flex items-center gap-2">
          {t(md.uploadFile)}
          <input type="file" accept=".mjml,.html,.htm,text/html" onChange={(e) => e.target.files?.[0] && void readText(e.target.files[0]).then(setText)} />
        </label>
        {kind === 'html' ? <span className="k">{t(md.htmlNote)}</span> : null}
        {error ? (
          <span role="alert" className="k" style={{ color: 'var(--bad-fg)' }}>
            {error}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

function SaveTemplateDialog({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: (name: string) => void }) {
  const c = useComposer()
  const t = useT()
  const qc = useQueryClient()
  const section = sectionOf(c.selected)
  const capital = (s: string) => s.charAt(0).toLocaleUpperCase() + s.slice(1)
  const defaultName = (k: 'template' | 'section') => (k === 'section' && section ? capital(t(md.sectionName, { name: sectionWord(sectionLabel(section)) })) : t(md.templateName, { name: c.name || t(md.untitled) }))
  // The dialog is remounted each time it opens (see Dialogs), so these start fresh.
  const [kind, setKindState] = useState<'template' | 'section'>('template')
  const [name, setName] = useState(() => defaultName('template'))
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const setKind = (k: 'template' | 'section') => {
    if (name === defaultName(kind)) setName(defaultName(k))
    setKindState(k)
  }
  const submit = async () => {
    setBusy(true)
    setError('')
    try {
      const mjml = kind === 'section' && section ? section.toHTML() : c.currentMjml()
      if (kind === 'template') await c.save({ manual: true, quiet: true })
      await http.POST('/api/v1/workspaces/{workspace}/templates', {
        params: { path: { workspace: c.slug } },
        body: {
          kind,
          name: name.trim(),
          description: description.trim() || undefined,
          mjml,
          html: kind === 'template' ? c.compileCurrent()?.html : undefined,
          // A transactional template isn't a campaign.
          sourceCampaignId: c.kind === 'transactional' ? undefined : c.campaign.id,
        },
      })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/templates'] })
      onSaved(name.trim())
      onClose()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={t(md.saveTitle)}
      footer={
        <>
          <button type="button" className="btn" onClick={onClose}>
            {t(common.cancel)}
          </button>
          <button type="button" className="btn pri" disabled={!name.trim() || busy} aria-busy={busy || undefined} onClick={() => void submit()}>
            {t(common.save)}
          </button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="seg" role="group" aria-label={t(md.whatToSave)}>
          <button type="button" aria-pressed={kind === 'template'} onClick={() => setKind('template')}>
            {t(md.wholeEmail)}
          </button>
          <button type="button" aria-pressed={kind === 'section'} disabled={!section} onClick={() => setKind('section')} title={section ? undefined : t(md.selectSectionFirst)}>
            {t(md.selectedSection)}
          </button>
        </div>
        <label className="lbl">
          {t(md.name)}
          <input className="in" value={name} onChange={(e) => setName(e.target.value)} maxLength={120} />
        </label>
        <label className="lbl">
          {t(md.description)}
          <input className="in" value={description} onChange={(e) => setDescription(e.target.value)} maxLength={300} placeholder={kind === 'section' ? t(md.descriptionSection) : t(md.descriptionTemplate)} />
        </label>
        {error ? (
          <span role="alert" className="k" style={{ color: 'var(--bad-fg)' }}>
            {error}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

/** The keys (a key name, or a gesture to translate) and what they do. */
const SHORTCUTS: [string | MessageDescriptor, MessageDescriptor][] = [
  ['⌘S', md.scSave],
  ['⌘Z', md.scUndo],
  ['⇧⌘Z', md.scRedo],
  ['⌘C · ⌘V', md.scCopy],
  ['Delete', md.scDelete],
  ['Enter', md.scEnter],
  ['Esc', md.scEscape],
  [md.keyMove, md.scMove],
  [md.keyLayers, md.scLayers],
  [md.keyDoubleClick, md.scDoubleClick],
  ['⌘B · ⌘I', md.scBold],
  ['⌘/', md.scHelp],
]

function ShortcutsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t(md.shortcutsTitle)} size="sm" description={t(md.shortcutsLead)}>
      <dl className="m-0 grid gap-x-4 gap-y-2" style={{ gridTemplateColumns: 'auto 1fr' }}>
        {SHORTCUTS.map(([k, v]) => (
          <div key={v.id} className="contents">
            <dt className="mono font-semibold">{typeof k === 'string' ? k : t(k)}</dt>
            <dd className="m-0">{t(v)}</dd>
          </div>
        ))}
      </dl>
    </Dialog>
  )
}

export function Dialogs({ dialog, onClose, onSavedTemplate }: { dialog: ComposerDialog; onClose: () => void; onSavedTemplate: (name: string) => void }) {
  const c = useComposer()
  const [compare, setCompare] = useState<[number, number] | null>(null)
  return (
    <>
      <VersionsDialog
        open={dialog === 'versions'}
        onClose={onClose}
        onCompare={(a, b) => {
          setCompare([a, b])
          c.openDialog('compare')
        }}
      />
      <CompareDialog
        open={dialog === 'compare'}
        onClose={() => {
          setCompare(null)
          onClose()
        }}
        versions={dialog === 'compare' ? (compare ?? (c.status.kind === 'conflict' ? 'conflict' : null)) : null}
      />
      <AssetsDialog
        open={dialog === 'assets'}
        onClose={() => {
          c.setAssetTarget(null)
          onClose()
        }}
      />
      <ImportDialog open={dialog === 'import'} onClose={onClose} />
      <SaveTemplateDialog key={dialog === 'saveTemplate' ? 'open' : 'closed'} open={dialog === 'saveTemplate'} onClose={onClose} onSaved={onSavedTemplate} />
      <ShortcutsDialog open={dialog === 'shortcuts'} onClose={onClose} />
    </>
  )
}
