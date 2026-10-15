import { useQueryClient } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { useEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { clock, shortDate } from '@/lib/format'
import { useComposer, type ComposerDialog } from './context'
import { diffLines } from './diff'
import { sectionLabel, sectionOf } from './editor/labels'
import { formatMjml } from './editor/mjmlDoc'
import { kb, readText } from './format'

type Version = Schemas['ComposerVersion']

function sameDay(a: Date, b: Date) {
  return a.toDateString() === b.toDateString()
}

function when(at: string) {
  const d = new Date(at)
  return sameDay(d, new Date()) ? clock(d) : shortDate(d)
}

function VersionsDialog({ open, onClose, onCompare }: { open: boolean; onClose: () => void; onCompare: (a: number, b: number) => void }) {
  const c = useComposer()
  const qc = useQueryClient()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions', { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } }, { enabled: open })
  const [busy, setBusy] = useState<number | null>(null)
  const [pick, setPick] = useState<number[]>([])
  const restore = async (v: Version) => {
    setBusy(v.version)
    try {
      if (c.status.kind === 'dirty') await c.save()
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions/{version}/restore', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id, version: v.version } },
        body: { baseVersion: c.status.kind === 'new' ? 0 : c.status.version },
      })
      await c.reloadFromServer()
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions'] })
      toast({ kind: 'success', title: `Version ${v.version} restored`, detail: 'It is saved as a new version; nothing was deleted.' })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't restore the version", detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }
  const label = (v: Version) => {
    if (v.source === 'create' && v.label.startsWith('created')) return v.label
    return v.label || (v.source === 'autosave' ? 'autosave' : v.source)
  }
  const toggle = (n: number) => setPick((p) => (p.includes(n) ? p.filter((x) => x !== n) : [...p.slice(-1), n]))
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={`Version history · ${c.name || 'Untitled campaign'}`}
      size="lg"
      footer={
        <>
          <span className="k mr-auto">{pick.length === 2 ? `Compare v${Math.min(...pick)} with v${Math.max(...pick)}` : 'Tick two versions to compare them.'}</span>
          <button type="button" className="btn" disabled={pick.length !== 2} onClick={() => onCompare(Math.min(...pick), Math.max(...pick))}>
            Compare two versions
          </button>
        </>
      }
    >
      {q.isPending ? (
        <p className="k">Loading versions…</p>
      ) : q.isError ? (
        <p className="k">The history couldn't load. {errorMessage(q.error)}</p>
      ) : (
        <div className="-mx-1 flex flex-col">
          {q.data.items.map((v) => (
            <div key={v.version} className={`ver${v.current ? ' cur' : ''}`}>
              <input type="checkbox" aria-label={`Compare version ${v.version}`} checked={pick.includes(v.version)} onChange={() => toggle(v.version)} />
              <span className="mono k" style={{ width: 38 }}>
                v{v.version}
              </span>
              <span className="min-w-0 flex-1">
                {v.current ? <b>Current</b> : null}
                {v.current ? ' · ' : ''}
                {when(v.savedAt)}
                {v.savedBy ? ` · ${v.savedBy}` : ''} · {label(v)}
                <span className="k"> · {kb(v.sizeBytes)}</span>
              </span>
              {v.current ? (
                <Badge tone="ok">Live draft</Badge>
              ) : c.canEdit ? (
                <button type="button" className="btn sm" aria-busy={busy === v.version || undefined} disabled={busy !== null} onClick={() => void restore(v)}>
                  Restore
                </button>
              ) : null}
            </div>
          ))}
          {q.data.total > q.data.items.length ? <p className="k px-3.5 pt-2">Showing the latest {q.data.items.length} of {q.data.total} versions.</p> : null}
        </div>
      )}
    </Dialog>
  )
}

type CompareTexts = { a: string; b: string; la: string; lb: string; sections: string[] }

/** Loads and diffs two versions, or the latest save against the unsaved edits after a conflict. */
function CompareBody({ versions }: { versions: [number, number] | 'conflict' }) {
  const c = useComposer()
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
        return { a: formatMjml(theirs.mjml), b: formatMjml(mine.mjml), la: `v${latest} · ${theirs.savedBy}`, lb: 'your edits (not saved)', sections: mine.theirs?.theirChanges ?? [] }
      }
      const [x, y] = await Promise.all([get(versions[0]), get(versions[1])])
      return { a: `Subject: ${x.subject}\nPreheader: ${x.preheader}\n\n${formatMjml(x.mjml)}`, b: `Subject: ${y.subject}\nPreheader: ${y.preheader}\n\n${formatMjml(y.mjml)}`, la: `v${x.version} · ${when(x.savedAt)}`, lb: `v${y.version} · ${when(y.savedAt)}`, sections: y.sections }
    }
    run()
      .then((t) => !cancelled && setTexts(t))
      .catch((e: unknown) => toast({ kind: 'error', title: "Couldn't load the versions", detail: errorMessage(e) }))
    return () => {
      cancelled = true
    }
  }, [versions, slug, campaignId, mine])
  const lines = useMemo(() => (texts ? diffLines(texts.a, texts.b) : []), [texts])
  return (
    <>
      {texts ? <p className="k m-0">{`${texts.la} → ${texts.lb}`}</p> : null}
      {!texts ? (
        <p className="k">Loading…</p>
      ) : (
        <div className="flex flex-col gap-2">
          {texts.sections.length ? <span className="k">Sections: {texts.sections.join(', ')}</span> : null}
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
                <span className="vh">{l.kind === 'add' ? 'added: ' : l.kind === 'del' ? 'removed: ' : ''}</span>
                {l.text}
              </div>
            ))}
            {lines.every((l) => l.kind === 'same' || l.kind === 'fold') ? <p className="k p-3">No differences.</p> : null}
          </div>
        </div>
      )}
    </>
  )
}

function CompareDialog({ open, onClose, versions }: { open: boolean; onClose: () => void; versions: [number, number] | 'conflict' | null }) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title="Compare versions" size="lg">
      {open && versions ? <CompareBody key={String(versions)} versions={versions} /> : null}
    </Dialog>
  )
}

function AssetsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const c = useComposer()
  const qc = useQueryClient()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/media', { params: { path: { workspace: c.slug } } }, { enabled: open })
  const [uploading, setUploading] = useState(false)
  const [drag, setDrag] = useState(false)
  const [url, setUrl] = useState('')
  const input = useRef<HTMLInputElement>(null)
  const target = c.assetTarget
  const choose = (src: string) => {
    const comp = target?.component as Component | undefined
    if (comp) {
      comp.addAttributes({ [target!.attr]: src })
      if (target!.attr === 'src') comp.set('src', src)
    }
    onClose()
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
      if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || `Upload failed (${r.status})`)
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/media'] })
      toast({ kind: 'success', title: `${body.fileName} uploaded` })
      if (target) choose(body.url)
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't upload the image", detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setUploading(false)
    }
  }
  const remove = async (a: Schemas['MediaAsset']) => {
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/media/{assetId}', { params: { path: { workspace: c.slug, assetId: a.id } } })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/media'] })
      toast({ kind: 'info', title: `${a.fileName} removed from the library`, detail: 'Emails already sent keep showing it.' })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't remove the image", detail: errorMessage(e) })
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={target?.attr === 'background-url' ? 'Choose a background image' : 'Choose an image'} size="lg" description="PNG, JPEG or GIF up to 5 MB. Images are served from your ReputeCast address, so they load in every inbox.">
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
          <span className="font-semibold">Drop an image here</span>
          <span className="k">or</span>
          <button type="button" className="btn sm" aria-busy={uploading || undefined} disabled={!c.canEdit || uploading} onClick={() => input.current?.click()}>
            {uploading ? 'Uploading…' : 'Upload from your computer'}
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
          <input className="in mono flex-1" aria-label="Image URL" placeholder="Or paste an image URL: https://…" value={url} onChange={(e) => setUrl(e.target.value)} />
          <button type="submit" className="btn" disabled={!/^https:\/\//.test(url.trim())}>
            Use URL
          </button>
        </form>
        <span className="font-semibold">Library</span>
        {q.isPending ? (
          <p className="k">Loading images…</p>
        ) : q.isError ? (
          <p className="k">The library couldn't load. {errorMessage(q.error)}</p>
        ) : !q.data.items.length ? (
          <p className="k">No images yet. Uploads appear here for every campaign of the workspace.</p>
        ) : (
          <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(140px,1fr))' }}>
            {q.data.items.map((a) => (
              <div key={a.id} className="flex flex-col gap-1.5 rounded-[10px] border border-line bg-surface" style={{ padding: 8 }}>
                <button type="button" className="overflow-hidden rounded-md border border-line" style={{ height: 90, padding: 0, background: 'repeating-conic-gradient(#F3F4F6 0 25%, #fff 0 50%) 0 0/12px 12px', cursor: 'pointer' }} onClick={() => choose(a.url)} disabled={!target} aria-label={`Use ${a.fileName}`}>
                  <img src={a.url} alt="" style={{ width: '100%', height: '100%', objectFit: 'contain' }} loading="lazy" />
                </button>
                <span className="truncate text-[12px] font-semibold" title={a.fileName}>
                  {a.fileName}
                </span>
                <span className="k">
                  {a.width}×{a.height} · {kb(a.sizeBytes)}
                  {a.hasAlpha ? ' · transparent' : ''}
                </span>
                {c.canEdit ? (
                  <button type="button" className="btn sm self-start" onClick={() => void remove(a)}>
                    Remove
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
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const kind = /<mjml[\s>]/i.test(text) ? 'mjml' : text.trim() ? 'html' : ''
  const submit = () => {
    if (!kind) return
    const src = kind === 'mjml' ? text : htmlToMjml(text)
    if (kind === 'mjml' && !/<mj-body[\s>]/i.test(text)) {
      setError('The MJML has no <mj-body>.')
      return
    }
    c.loadContent(src, { source: 'import', label: kind === 'mjml' ? 'imported MJML' : 'imported HTML' })
    setText('')
    setError('')
    onClose()
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title="Import MJML or HTML"
      description="The canvas content is replaced; the current version stays in the history."
      footer={
        <>
          <button type="button" className="btn" onClick={onClose}>
            Cancel
          </button>
          <button type="button" className="btn pri" disabled={!kind || !c.canEdit} onClick={submit}>
            {kind === 'html' ? 'Import as raw HTML' : 'Import'}
          </button>
        </>
      }
    >
      <div className="flex flex-col gap-2">
        <label className="lbl">
          Paste MJML or HTML
          <textarea className="in mono" rows={12} value={text} onChange={(e) => setText(e.target.value)} placeholder="<mjml>…</mjml>" />
        </label>
        <label className="k inline-flex items-center gap-2">
          Or upload a file
          <input type="file" accept=".mjml,.html,.htm,text/html" onChange={(e) => e.target.files?.[0] && void readText(e.target.files[0]).then(setText)} />
        </label>
        {kind === 'html' ? <span className="k">HTML goes into one Raw HTML section: it is sent as it is, but can't be edited block by block. Rebuild it with MJML blocks to edit it on the canvas.</span> : null}
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
  const qc = useQueryClient()
  const section = sectionOf(c.selected)
  const defaultName = (k: 'template' | 'section') => (k === 'section' && section ? `${sectionLabel(section)[0]!.toUpperCase()}${sectionLabel(section).slice(1)} section` : `${c.name || 'Untitled'} template`)
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
      title="Save as template"
      footer={
        <>
          <button type="button" className="btn" onClick={onClose}>
            Cancel
          </button>
          <button type="button" className="btn pri" disabled={!name.trim() || busy} aria-busy={busy || undefined} onClick={() => void submit()}>
            Save
          </button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="seg" role="group" aria-label="What to save">
          <button type="button" aria-pressed={kind === 'template'} onClick={() => setKind('template')}>
            The whole email
          </button>
          <button type="button" aria-pressed={kind === 'section'} disabled={!section} onClick={() => setKind('section')} title={section ? undefined : 'Select a section on the canvas first'}>
            The selected section
          </button>
        </div>
        <label className="lbl">
          Name
          <input className="in" value={name} onChange={(e) => setName(e.target.value)} maxLength={120} />
        </label>
        <label className="lbl">
          Description
          <input className="in" value={description} onChange={(e) => setDescription(e.target.value)} maxLength={300} placeholder={kind === 'section' ? 'Shown under the name in Saved sections' : 'Shown in the template gallery'} />
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

const SHORTCUTS: [string, string][] = [
  ['⌘S', 'Save a version now'],
  ['⌘Z', 'Undo'],
  ['⇧⌘Z', 'Redo'],
  ['⌘C · ⌘V', 'Copy and paste the selected component'],
  ['Delete', 'Delete the selected component'],
  ['↑ ↓ on Move', 'Move the selected component up or down'],
  ['Double-click', 'Edit text on the canvas'],
  ['⌘B · ⌘I', 'Bold and italic while editing text'],
  ['Esc', 'Stop editing text'],
  ['⌘/', 'Show these shortcuts'],
]

function ShortcutsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title="Keyboard shortcuts" size="sm" description="Ctrl instead of ⌘ on Windows and Linux.">
      <dl className="m-0 grid gap-x-4 gap-y-2" style={{ gridTemplateColumns: 'auto 1fr' }}>
        {SHORTCUTS.map(([k, v]) => (
          <div key={k} className="contents">
            <dt className="mono font-semibold">{k}</dt>
            <dd className="m-0">{v}</dd>
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
