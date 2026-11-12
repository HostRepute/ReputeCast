import { Link } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { perCall, useRefreshCoupons, type CodesResult, type Pool } from './api'
import { m } from './messages'
import { discountText } from './PoolsPage'

const bold = (c: ReactNode[]) => <b>{c}</b>
const MAX_BYTES = 6 << 20

/** Field errors of a 422, by field (body. taken off). */
function fieldErrors(e: unknown): Record<string, string> {
  if (e instanceof ApiError && e.problem.errors?.length) {
    const out: Record<string, string> = {}
    for (const x of e.problem.errors) out[(x.location ?? '').replace(/^body\.?/, '') || 'form'] = x.message ?? ''
    return out
  }
  return { form: errorMessage(e) }
}

/** "1.2 MB" */
function size(bytes: number): string {
  return `${(bytes / (1 << 20)).toFixed(1)} MB`
}

/** Add codes to an uploaded pool: paste or a CSV (DiscountCodePool). */
export function AddCodesDialog({ pool, onClose }: { pool: Pool; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCoupons()
  const [mode, setMode] = useState<'paste' | 'file'>('paste')
  const [text, setText] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<CodesResult | null>(null)
  const [error, setError] = useState<ReactNode>(null)
  const input = useRef<HTMLInputElement>(null)
  const lines = mode === 'paste' ? text.split('\n').filter((l) => l.trim() !== '').length : 0
  const submit = async () => {
    setError(null)
    setResult(null)
    if (mode === 'file' && file && file.size > MAX_BYTES) {
      setError(<T msg={m.addTooBig} values={{ file: file.name, size: size(file.size), b: bold }} />)
      return
    }
    const body = mode === 'paste' ? text : file ? await file.text() : ''
    if (body.trim() === '') {
      setError(t(m.addNone))
      return
    }
    setBusy(true)
    try {
      const res = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/coupons/pools/${pool.id}/codes`, {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'text/plain', Accept: 'application/json', 'Accept-Language': document.documentElement.lang || 'en' },
        body,
      })
      const json = (await res.json()) as CodesResult & { detail?: string; errors?: { message?: string }[] }
      if (!res.ok) {
        setError(json.errors?.[0]?.message || json.detail || String(res.status))
        return
      }
      setResult(json)
      setText('')
      setFile(null)
      void refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const invalid = result && result.invalid > 0 ? t(m.addInvalid, {
    invalid: result.invalid,
    lines: result.invalidLines.map((l) => t(m.addInvalidLine, { line: count(l.line), text: l.text })).join(', '),
  }) : ''
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{pool.name}</span>}
      title={t(m.addTitle)}
      footer={
        <>
          <span className="k mr-auto">{t(m.addLines, { count: lines })}</span>
          <Button onClick={onClose}>{result ? t(common.close) : t(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void submit()}>
            {t(m.addCodes)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {result ? (
          <Note tone="ok">
            <T
              msg={m.addResult}
              values={{
                added: result.added,
                name: pool.name,
                skipped: [t(m.addSkipped, { duplicates: result.duplicates }), invalid].filter(Boolean).join(' '),
                free: count(result.free),
                b: bold,
              }}
            />
          </Note>
        ) : null}
        {error ? (
          <Note tone="bad" role="alert">
            {error}
          </Note>
        ) : null}
        <Segmented<'paste' | 'file'>
          label={t(m.addTitle)}
          value={mode}
          onChange={setMode}
          className="self-start"
          items={[
            { value: 'paste', label: t(m.addPaste) },
            { value: 'file', label: t(m.addFile) },
          ]}
        />
        {mode === 'paste' ? (
          <Field label={t(m.addCodesLabel)} hint={t(m.addHint)}>
            <textarea className="in mono min-h-[140px] py-2.5" value={text} onChange={(e) => setText(e.target.value)} spellCheck={false} />
          </Field>
        ) : (
          <Field label={t(m.addFileLabel)} hint={t(m.addHint)}>
            <input ref={input} type="file" accept=".csv,.txt,text/csv,text/plain" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          </Field>
        )}
      </div>
    </Dialog>
  )
}

/** Create more codes in a store pool (DiscountCodePool). */
export function CreateMoreDialog({ pool, initial, onClose }: { pool: Pool; initial?: number; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCoupons()
  const [n, setN] = useState(String(initial ?? 500))
  const [error, setError] = useState('')
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/coupons/pools/{pool}/generate')
  const store = pool.store?.name ?? ''
  const num = Number(n.replace(/\D/g, ''))
  const submit = async () => {
    setError('')
    if (!Number.isInteger(num) || num < 1 || num > 50000) {
      setError(t(m.moreCountInvalid))
      return
    }
    try {
      await create.mutateAsync({ params: { path: { workspace: slug, pool: pool.id } }, body: { count: num } })
      void refresh()
      onClose()
      toast({ kind: 'progress', title: t(m.moreStarted), detail: t(m.moreStartedDetail, { store }), duration: 6000 })
    } catch (e) {
      const f = fieldErrors(e)
      setError(f.count ?? f.form ?? Object.values(f)[0] ?? '')
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{pool.name}</span>}
      title={t(m.moreTitle, { store })}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={create.isPending} onClick={() => void submit()}>
            {t(m.moreButton, { count: num > 0 ? num : 0 })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <Field label={t(m.moreCount)} error={error || undefined}>
          <span className="flex items-center gap-2">
            <Input inputMode="numeric" className="w-[120px]" value={n} onChange={(e) => setN(e.target.value.replace(/\D/g, ''))} />
            <span className="k">{t(m.moreUnit)}</span>
          </span>
        </Field>
        <span className="text-[13px] text-ink-nav">
          {t(m.moreNote, { store, per: perCall(pool.store?.platform), prefix: pool.prefix || '—', discount: discountText(t, pool) })}
        </span>
      </div>
    </Dialog>
  )
}

/** Remove unused codes: consequences first (DiscountCodesStates). */
export function RemoveUnusedDialog({ pool, onClose }: { pool: Pool; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCoupons()
  const remove = $api.useMutation('post', '/api/v1/workspaces/{workspace}/coupons/pools/{pool}/codes/remove-unused')
  const [error, setError] = useState('')
  const isStore = pool.source !== 'upload'
  const store = pool.store?.name ?? ''
  const live = pool.usedIn.filter((u) => u.kind === 'automation' && u.live)
  const campaigns = pool.usedIn.filter((u) => u.kind === 'campaign')
  const submit = async () => {
    try {
      const out = await remove.mutateAsync({ params: { path: { workspace: slug, pool: pool.id } } })
      void refresh()
      onClose()
      toast({ kind: 'success', title: t(m.removedToast, { count: out.removed }) })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  const rows: [string, ReactNode][] = []
  if (isStore) rows.push([t(m.removeInStore, { store }), t(m.removeInStoreValue)])
  rows.push([t(m.removeSent), t(m.removeSentValue, { count: count(pool.sent) })])
  for (const u of live) rows.push([u.name, isStore ? t(m.removeLiveFlow) : t(m.removeLiveFlowUpload)])
  for (const u of campaigns) rows.push([u.name, t(m.removeCampaign)])
  return (
    <Dialog
      open
      dismissible={false}
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{pool.name}</span>}
      title={t(m.removeTitle, { count: pool.free })}
      footer={
        <>
          <span className="k mr-auto">{t(m.audited)}</span>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="danger-primary" loading={remove.isPending} disabled={pool.free === 0} onClick={() => void submit()}>
            {t(m.removeButton, { count: pool.free })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {pool.free === 0 ? <Note tone="info">{t(m.removeNothing)}</Note> : null}
        <dl className="kv">
          {rows.map(([k, v]) => (
            <Row key={k} k={k} v={v} />
          ))}
        </dl>
        <span className="k">
          <T msg={m.removeExport} values={{ a: (c) => <a href={`/api/v1/workspaces/${slug}/coupons/pools/${pool.id}/export.csv`} download="">{c}</a> }} />
        </span>
        {error ? (
          <Note tone="bad" role="alert">
            {error}
          </Note>
        ) : null}
      </div>
    </Dialog>
  )
}

function Row({ k, v }: { k: string; v: ReactNode }) {
  return (
    <>
      <dt>{k}</dt>
      <dd>{v}</dd>
    </>
  )
}

/** Archive a pool; asks first when a live flow uses it (DiscountCodesStates). */
export function ArchiveDialog({ pool, onClose }: { pool: Pool; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCoupons()
  const archive = $api.useMutation('post', '/api/v1/workspaces/{workspace}/coupons/pools/{pool}/archive')
  const [error, setError] = useState('')
  const live = pool.usedIn.filter((u) => u.kind === 'automation' && u.live)
  const campaigns = pool.usedIn.filter((u) => u.kind === 'campaign')
  const used = pool.usedIn.length > 0
  const store = pool.store?.name ?? ''
  const submit = async () => {
    try {
      await archive.mutateAsync({ params: { path: { workspace: slug, pool: pool.id } }, body: { confirm: live.length > 0 } })
      void refresh()
      onClose()
      toast({ kind: 'success', title: t(m.archivedToast, { name: pool.name }) })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      dismissible={false}
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{pool.name}</span>}
      title={live.length > 0 ? t(m.archiveLiveTitle, { name: pool.name }) : t(m.archiveTitle, { name: pool.name })}
      footer={
        <>
          {used ? <span className="k mr-auto">{t(m.audited)}</span> : null}
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant={live.length > 0 ? 'danger-primary' : 'primary'} loading={archive.isPending} onClick={() => void submit()}>
            {live.length > 0 ? t(m.archiveAnyway) : t(m.archiveButton)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {live.map((u) => (
          <Note tone="warn" key={u.emailId ?? u.id}>
            <T msg={m.archiveLiveNote} values={{ flow: u.name, email: u.emailName ?? '', b: bold }} />
          </Note>
        ))}
        {used ? (
          <dl className="kv">
            {campaigns.map((u) => (
              <Row key={u.id} k={u.name} v={t(m.archiveCampaign)} />
            ))}
            <Row k={t(m.archiveSent)} v={pool.source === 'upload' ? t(m.archiveSentUpload) : t(m.archiveSentStore, { store })} />
            <Row k={t(m.archiveNumbers)} v={t(m.archiveNumbersValue)} />
          </dl>
        ) : (
          <p className="m-0 text-[13.5px] text-ink-nav">{t(m.archiveUnused)}</p>
        )}
        {live[0] ? (
          <span className="k">
            <T
              msg={m.archiveSafer}
              values={{
                flow: live[0].name,
                a: (c) => (
                  <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: live[0]!.id }}>
                    {c}
                  </Link>
                ),
              }}
            />
          </span>
        ) : null}
        {error ? (
          <Note tone="bad" role="alert">
            {error}
          </Note>
        ) : null}
      </div>
    </Dialog>
  )
}

/** Edit a pool's name, prefix, low mark, one code per contact and apply address. */
export function EditPoolDialog({ pool, onClose }: { pool: Pool; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCoupons()
  const [name, setName] = useState(pool.name)
  const [prefix, setPrefix] = useState(pool.prefix)
  const [low, setLow] = useState(String(pool.lowAt))
  const [onePer, setOnePer] = useState(pool.onePerContact)
  const [applyUrl, setApplyUrl] = useState(pool.settings.applyUrl ?? '')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const isStore = pool.source !== 'upload'
  const submit = async () => {
    setErrors({})
    setBusy(true)
    try {
      const { error } = await http.PATCH('/api/v1/workspaces/{workspace}/coupons/pools/{pool}', {
        params: { path: { workspace: slug, pool: pool.id } },
        body: {
          name,
          prefix,
          lowAt: Number(low.replace(/\D/g, '') || 0),
          onePerContact: onePer,
          settings: isStore ? undefined : { applyUrl: applyUrl.trim() || undefined },
        },
      })
      if (error) throw new ApiError(422, error)
      void refresh()
      onClose()
      toast({ kind: 'success', title: t(m.saved) })
    } catch (e) {
      setErrors(fieldErrors(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => (o ? null : onClose())}
      title={t(m.editTitle, { name: pool.name })}
      footer={
        <>
          <span className="k mr-auto">{t(m.audited)}</span>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void submit()}>
            {t(common.save)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {errors.form ? (
          <Note tone="bad" role="alert">
            {errors.form}
          </Note>
        ) : null}
        <Field label={t(m.name)} error={errors.name}>
          <Input value={name} maxLength={80} onChange={(e) => setName(e.target.value)} />
        </Field>
        {isStore ? (
          <>
            <Field label={t(m.prefix)} error={errors.prefix}>
              <Input className="mono" value={prefix} maxLength={12} onChange={(e) => setPrefix(e.target.value.toUpperCase().replace(/[^A-Z0-9-]/g, ''))} />
            </Field>
            <Note tone="info">{t(m.editLocked, { store: pool.store?.name ?? '' })}</Note>
          </>
        ) : (
          <Field label={t(m.applyUrl)} error={errors['settings.applyUrl']}>
            <Input value={applyUrl} onChange={(e) => setApplyUrl(e.target.value)} />
          </Field>
        )}
        <Field label={t(m.setLow)} error={errors.lowAt}>
          <Input inputMode="numeric" className="w-[120px]" value={low} onChange={(e) => setLow(e.target.value.replace(/\D/g, ''))} />
        </Field>
        <Checkbox checked={onePer} onChange={(e) => setOnePer(e.target.checked)} label={<b className="font-semibold">{t(m.onePerContact)}</b>} hint={t(m.onePerContactHint)} />
      </div>
    </Dialog>
  )
}
