import { useNavigate } from '@tanstack/react-router'
import { Fragment, useId, useRef, useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './import.css'

type Import = Schemas['TemplateImport']
type Item = Schemas['TemplateImportItem']
type Source = Import['source']

const MAX_FILES = 50
const MAX_BYTES = 2 << 20

/** Text with `code` in backticks, as the conversion report writes it. */
export function CodeText({ text }: { text: string }) {
  const parts = text.split('`')
  return (
    <>
      {parts.map((p, i) =>
        i % 2 ? (
          <span key={i} className="mono">
            {p}
          </span>
        ) : (
          <Fragment key={i}>{p}</Fragment>
        ),
      )}
    </>
  )
}

function useSourceName() {
  const t = useT()
  return (s: Source) => (s === 'postmark' ? t(m.postmark) : s === 'sendgrid' ? t(m.sendgrid) : s === 'mailgun' ? t(m.mailgun) : s === 'notifuse' ? t(m.notifuse) : t(m.files))
}

/** Steps 1 to 3 (Choose a source, Connect, Pick) and the stepper of the report. */
function steps(source: Source) {
  return source === 'files' ? 3 : 4
}

function ResultBadge({ it }: { it: Item }) {
  const t = useT()
  if (it.failed) return <Badge tone="bad">{t(m.resultFailed)}</Badge>
  if (it.status === 'manual') return <Badge tone="bad">{t(m.resultManual)}</Badge>
  if (it.status === 'notes') return <Badge tone="warn">{t(m.resultNotes)}</Badge>
  return <Badge tone="ok">{t(m.resultConverted)}</Badge>
}

function NoteCell({ it }: { it: Item }) {
  if (it.summaryLine && it.summaryKind === 'manual') return <T msg={m.noteAtLine} values={{ line: it.summaryLine, text: <CodeText text={it.summary} /> }} />
  // A note reads as a label in the cell (as on the board): no closing full stop.
  const text = it.summaryKind === 'manual' ? it.summary : it.summary.replace(/\.$/, '')
  if (it.summaryLine && it.summaryKind === 'note') return <T msg={m.noteLine} values={{ line: it.summaryLine, text: <CodeText text={text} /> }} />
  return <CodeText text={text} />
}

function Stepper({ imp }: { imp: Import }) {
  const t = useT()
  const name = useSourceName()
  const done = (label: string) => (
    <li className="txi-step done">
      <span className="txi-dot" aria-hidden="true">
        <Icon name="check" size={12} strokeWidth={2.4} />
      </span>
      {label}
      <span className="k" aria-hidden="true">
        ›
      </span>
    </li>
  )
  return (
    <ol className="txi-steps" aria-label={t(m.reportEyebrow)}>
      {done(t(m.stepperSource, { source: name(imp.source) }))}
      {imp.source === 'files' ? done(t(m.stepperUpload)) : done(t(m.stepperConnect))}
      {imp.source === 'files' ? null : done(t(m.stepperPick, { count: imp.picked.length, total: imp.available.length }))}
      <li className="txi-step now" aria-current="step">
        <span className="txi-dot" aria-hidden="true">
          {steps(imp.source)}
        </span>
        {t(m.stepperReview)}
      </li>
    </ol>
  )
}

function accountLine(t: ReturnType<typeof useT>, imp: Import, source: string): string {
  const account = imp.account ?? ''
  switch (imp.accountKind) {
    case 'server':
      return t(m.accountServer, { source, account })
    case 'workspace':
      return t(m.accountWorkspace, { source, account })
    case 'domains':
      return t(m.accountDomains, { source, account })
  }
  return t(m.accountAccount, { source, account })
}

/**
 * Import templates (TransactionalImport): choose a source, connect it (or
 * upload files), pick templates, read the conversion report, and import
 * them as drafts. The session lives on the server, so `importId` resumes
 * it after a reload.
 */
export function ImportDialog({ initial, onClose, onSession }: { initial?: Import; onClose: () => void; onSession: (id: string | undefined) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const name = useSourceName()
  const ids = useId()
  const [imp, setImp] = useState<Import | null>(initial ?? null)
  const [step, setStep] = useState<'source' | 'connect' | 'files' | 'pick' | 'review'>(() =>
    initial ? (initial.status === 'pick' ? 'pick' : initial.status === 'review' ? 'review' : 'source') : 'source',
  )
  const [source, setSource] = useState<Source>(initial?.source ?? 'postmark')
  const [apiKey, setApiKey] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [region, setRegion] = useState<'us' | 'eu'>('us')
  const [domain, setDomain] = useState('')
  const [files, setFiles] = useState<{ name: string; size: number; content: string }[]>([])
  const [fileError, setFileError] = useState<string>()
  const [picked, setPicked] = useState<string[]>(initial?.picked ?? [])
  const [error, setError] = useState<{ field?: string; message: string }>()
  const [busy, setBusy] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const path = (id: string) => ({ params: { path: { workspace: slug, importId: id } } })

  const fail = (e: unknown) => {
    if (e instanceof ApiError) {
      const field = e.problem.errors?.[0]?.location?.replace(/^body\./, '')
      setError({ field: e.code === 'source_auth_failed' ? 'apiKey' : field, message: e.problem.detail ?? errorMessage(e) })
    } else setError({ message: errorMessage(e) })
  }

  const connect = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/imports', {
        params: { path: { workspace: slug } },
        body: { source, apiKey, baseUrl: source === 'notifuse' ? baseUrl : undefined, region: source === 'mailgun' ? region : undefined, domain: source === 'mailgun' && domain.trim() ? domain.trim() : undefined },
      })
      if (data) {
        setImp(data)
        setPicked(data.picked)
        setApiKey('')
        setStep('pick')
        onSession(data.id)
      }
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  const upload = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/imports', {
        params: { path: { workspace: slug } },
        body: { source: 'files', files: files.map((f) => ({ name: f.name, content: f.content })) },
      })
      if (data) {
        setImp(data)
        setStep('review')
        onSession(data.id)
      }
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  const convert = async () => {
    if (!imp) return
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/imports/{importId}/convert', { ...path(imp.id), body: { refs: picked } })
      if (data) {
        setImp(data)
        setStep('review')
      }
    } catch (e) {
      fail(e)
    } finally {
      setBusy(false)
    }
  }

  const finish = async () => {
    if (!imp) return
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/imports/{importId}/import', path(imp.id))
      if (!data) return
      const made = data.created.filter((c) => !c.error)
      const failed = data.created.filter((c) => c.error)
      const manual = made.filter((c) => c.status === 'manual').length
      const open = made.find((c) => c.status === 'manual') ?? made[0]
      onClose()
      toast({
        kind: failed.length ? 'info' : 'success',
        title: t(m.doneTitle, { count: made.length }),
        detail: [manual ? t(m.doneManual, { count: manual }) : '', failed.length ? t(m.doneFailed, { count: failed.length, first: failed[0]!.error ?? '' }) : ''].filter(Boolean).join(' · ') || undefined,
        action: open?.id
          ? { label: t(m.doneOpen), onClick: () => void navigate({ to: '/w/$workspace/transactional/templates/$templateId', params: { workspace: slug, templateId: open.id! } }) }
          : undefined,
      })
    } catch (e) {
      fail(e)
      toast({ kind: 'error', title: t(m.failed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const cancel = () => {
    if (imp && imp.status !== 'done' && imp.status !== 'cancelled') {
      void http.POST('/api/v1/workspaces/{workspace}/transactional/imports/{importId}/cancel', path(imp.id)).catch(() => undefined)
    }
    onSession(undefined)
    onClose()
  }

  const addFiles = async (list: FileList | null) => {
    if (!list) return
    setFileError(undefined)
    const next = [...files]
    for (const f of Array.from(list)) {
      if (!/\.(mjml|html?)$/i.test(f.name)) {
        setFileError(t(m.notTemplate, { name: f.name }))
        continue
      }
      if (f.size > MAX_BYTES) {
        setFileError(t(m.tooBig, { name: f.name }))
        continue
      }
      if (next.some((x) => x.name === f.name)) continue
      next.push({ name: f.name, size: f.size, content: await f.text() })
    }
    if (next.length > MAX_FILES) {
      setFileError(t(m.tooMany, { count: next.length }))
      setFiles(next.slice(0, MAX_FILES))
      return
    }
    setFiles(next)
  }

  const total = steps(source)
  const errorNote = error && !error.field ? (
    <div className="note b" role="alert">
      <Icon name="alertCircle" size={16} />
      <div>{error.message}</div>
    </div>
  ) : null

  if (step === 'source') {
    const options: { s: Source; what: string }[] = [
      { s: 'postmark', what: t(m.postmarkWhat) },
      { s: 'sendgrid', what: t(m.sendgridWhat) },
      { s: 'mailgun', what: t(m.mailgunWhat) },
      { s: 'notifuse', what: t(m.notifuseWhat) },
      { s: 'files', what: t(m.filesWhat) },
    ]
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && cancel()}
        className="txi-dlg"
        eyebrow={<span className="k">{t(m.stepSource, { total })}</span>}
        title={t(m.title)}
        hideClose
        footer={
          <>
            <Button onClick={cancel}>{t(m.cancel)}</Button>
            <Button variant="primary" onClick={() => setStep(source === 'files' ? 'files' : 'connect')}>
              {t(m.continue)}
            </Button>
          </>
        }
      >
        <span className="k text-ink-nav" id={`${ids}-where`}>
          {t(m.where)}
        </span>
        <div className="txi-sources" role="radiogroup" aria-labelledby={`${ids}-where`}>
          {options.map((o) => (
            <button key={o.s} type="button" role="radio" aria-checked={source === o.s} className={`txi-source${source === o.s ? ' on' : ''}`} onClick={() => setSource(o.s)}>
              <span className="font-semibold">{name(o.s)}</span>
              <span className="k">{o.what}</span>
            </button>
          ))}
        </div>
      </Dialog>
    )
  }

  if (step === 'connect') {
    const keyLabel = source === 'postmark' ? t(m.postmarkKey) : source === 'sendgrid' ? t(m.sendgridKey) : source === 'mailgun' ? t(m.mailgunKey) : t(m.notifuseKey)
    const help = source === 'postmark' ? t(m.postmarkHelp) : source === 'sendgrid' ? t(m.sendgridHelp) : source === 'mailgun' ? t(m.mailgunHelp) : t(m.notifuseHelp)
    const keyError = error?.field === 'apiKey' ? error.message : undefined
    const urlError = error?.field === 'baseUrl' ? error.message : undefined
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && !busy && cancel()}
        className="txi-dlg"
        eyebrow={<span className="k">{t(m.stepConnect, { total })}</span>}
        title={t(m.connectTitle, { source: name(source) })}
        hideClose
        footer={
          <>
            <Button onClick={() => setStep('source')} disabled={busy}>
              {t(m.back)}
            </Button>
            <Button variant="primary" loading={busy} busyLabel={t(m.connecting)} disabled={!apiKey.trim() || (source === 'notifuse' && !baseUrl.trim())} onClick={() => void connect()}>
              {error ? t(m.tryAgain) : t(m.connect)}
            </Button>
          </>
        }
      >
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            void connect()
          }}
        >
          {source === 'notifuse' ? (
            <label className="lbl">
              {t(m.notifuseAddress)}
              <input
                className={urlError ? 'in err' : 'in'}
                type="url"
                value={baseUrl}
                placeholder="https://notifuse.example.com"
                aria-invalid={urlError ? true : undefined}
                aria-describedby={urlError ? `${ids}-urlerr` : undefined}
                onChange={(e) => setBaseUrl(e.target.value)}
              />
              {urlError ? (
                <span id={`${ids}-urlerr`} className="txi-err">
                  {urlError}
                </span>
              ) : null}
            </label>
          ) : null}
          <label className="lbl">
            {keyLabel}
            <input
              className={keyError ? 'in err' : 'in'}
              type="password"
              autoComplete="off"
              value={apiKey}
              aria-invalid={keyError ? true : undefined}
              aria-describedby={keyError ? `${ids}-keyerr` : `${ids}-help`}
              onChange={(e) => setApiKey(e.target.value)}
            />
            {keyError ? (
              <span id={`${ids}-keyerr`} className="txi-err">
                {keyError}
              </span>
            ) : null}
          </label>
          {source === 'mailgun' ? (
            <div className="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
              <label className="lbl">
                {t(m.mailgunRegion)}
                <select className="in" value={region} onChange={(e) => setRegion(e.target.value as 'us' | 'eu')}>
                  <option value="us">{t(m.mailgunUS)}</option>
                  <option value="eu">{t(m.mailgunEU)}</option>
                </select>
              </label>
              <label className="lbl">
                {t(m.mailgunDomain)}
                <input className="in" value={domain} placeholder={t(m.mailgunDomainPlaceholder)} onChange={(e) => setDomain(e.target.value)} />
              </label>
            </div>
          ) : null}
          <span className="k" id={`${ids}-help`}>
            {help}
          </span>
          {errorNote}
          <button type="submit" hidden />
        </form>
      </Dialog>
    )
  }

  if (step === 'files') {
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && !busy && cancel()}
        className="txi-dlg"
        eyebrow={<span className="k">{t(m.stepUpload, { total })}</span>}
        title={t(m.uploadTitle)}
        hideClose
        footer={
          <>
            <Button onClick={() => setStep('source')} disabled={busy}>
              {t(m.back)}
            </Button>
            <Button variant="primary" loading={busy} busyLabel={t(m.converting)} disabled={!files.length} onClick={() => void upload()}>
              {t(m.convert, { count: files.length })}
            </Button>
          </>
        }
      >
        <div
          className="txi-drop"
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            e.preventDefault()
            void addFiles(e.dataTransfer.files)
          }}
        >
          <Icon name="upload" size={20} />
          <span>{t(m.dropHere)}</span>
          <Button size="sm" onClick={() => fileInput.current?.click()}>
            {t(m.chooseFiles)}
          </Button>
          <input
            ref={fileInput}
            type="file"
            multiple
            accept=".mjml,.html,.htm,text/html"
            className="vh"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              void addFiles(e.target.files)
              e.target.value = ''
            }}
          />
        </div>
        {fileError ? (
          <span className="txi-err" role="alert">
            {fileError}
          </span>
        ) : null}
        {files.length ? (
          <div className="txi-list">
            {files.map((f) => (
              <div key={f.name} className="txi-row">
                <span className="mono flex-1">{f.name}</span>
                <span className="k">{Math.max(1, Math.round(f.size / 1024))} KB</span>
                <button type="button" className="btn sm ic" aria-label={t(m.removeFile, { name: f.name })} onClick={() => setFiles((x) => x.filter((y) => y.name !== f.name))}>
                  <Icon name="close" size={12} />
                </button>
              </div>
            ))}
          </div>
        ) : null}
        {errorNote}
      </Dialog>
    )
  }

  if (step === 'pick' && imp) {
    const all = imp.available
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && !busy && cancel()}
        className="txi-dlg"
        eyebrow={<span className="k">{t(m.stepPick, { total: steps(imp.source) })}</span>}
        title={t(m.pickTitle)}
        hideClose
        footer={
          <>
            <span className="k" style={{ marginRight: 'auto' }}>
              {t(m.selected, { count: picked.length, total: all.length })}
            </span>
            <Button onClick={() => setStep('connect')} disabled={busy}>
              {t(m.back)}
            </Button>
            <Button variant="primary" loading={busy} busyLabel={t(m.converting)} disabled={!picked.length} onClick={() => void convert()}>
              {t(m.convertPicked, { count: picked.length })}
            </Button>
          </>
        }
      >
        <div className="flex justify-end gap-2">
          <button type="button" className="txi-linklike" onClick={() => setPicked(all.map((r) => r.ref))}>
            {t(m.pickAll)}
          </button>
          <button type="button" className="txi-linklike" onClick={() => setPicked([])}>
            {t(m.pickNone)}
          </button>
        </div>
        <div className="txi-list">
          {all.map((r) => (
            <label key={r.ref} className="txi-row">
              <input type="checkbox" checked={picked.includes(r.ref)} onChange={(e) => setPicked((p) => (e.target.checked ? [...p, r.ref] : p.filter((x) => x !== r.ref)))} />
              <span className="flex-1 font-medium">{r.name}</span>
              <span className="mono k">{r.alias}</span>
            </label>
          ))}
        </div>
        {errorNote}
      </Dialog>
    )
  }

  if (step === 'review' && imp) {
    const lead = imp.source === 'postmark' ? m.reportLeadMustachio : imp.source === 'notifuse' ? m.reportLeadNotifuse : imp.source === 'files' ? m.reportLeadFiles : m.reportLeadHandlebars
    const importable = imp.items.filter((i) => !i.failed).length
    const b = (chunks: ReactNode[]) => <b>{chunks}</b>
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && !busy && cancel()}
        className="txi-dlg wide"
        eyebrow={<span className="k">{t(m.reportEyebrow)}</span>}
        title={t(m.reportTitle, { source: name(imp.source) })}
        description={t(lead)}
        hideClose
        footer={
          <>
            {imp.source === 'files' ? null : (
              <Button style={{ marginRight: 'auto' }} onClick={() => setStep('pick')} disabled={busy}>
                {t(m.back)}
              </Button>
            )}
            <Button onClick={cancel} disabled={busy}>
              {t(m.cancel)}
            </Button>
            <Button variant="primary" loading={busy} busyLabel={t(m.importing)} disabled={!importable} onClick={() => void finish()}>
              {t(m.importDrafts, { count: importable })}
            </Button>
          </>
        }
      >
        <Stepper imp={imp} />
        <div className="note i">
          <Icon name="info" size={16} />
          <div>{t(m.draftsNote)}</div>
        </div>
        <Table caption={t(m.reportEyebrow)}>
          <thead>
            <tr>
              <Th>{t(m.colTemplate)}</Th>
              <Th className="txi-alias">{imp.source === 'files' ? t(m.colFile) : t(m.colAlias, { source: name(imp.source) })}</Th>
              <Th>{t(m.colId)}</Th>
              <Th>{t(m.colResult)}</Th>
              <Th>{t(m.colNotes)}</Th>
            </tr>
          </thead>
          <tbody>
            {imp.items.map((it) => (
              <tr key={it.ref}>
                <Td>
                  <span className="font-semibold">{it.name}</span>
                  <span className="txi-alias-inline mono k">{it.alias || it.ref}</span>
                </Td>
                <Td nowrap className="txi-alias">
                  <span className="mono k">{it.alias || it.ref}</span>
                </Td>
                <Td nowrap>
                  <span className="mono">{it.key}</span>
                </Td>
                <Td nowrap>
                  <ResultBadge it={it} />
                </Td>
                <Td wrap className="txi-notes">
                  <NoteCell it={it} />
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
        <div className="txi-counts">
          <span>
            <T msg={m.countConverted} values={{ count: imp.converted, b }} />
          </span>
          <span>
            <T msg={m.countNotes} values={{ count: imp.withNotes, b }} />
          </span>
          <span>
            <T msg={m.countManual} values={{ count: imp.manual, b }} />
          </span>
          <span className="k">{imp.source === 'files' ? t(m.filesLine, { count: imp.items.length }) : accountLine(t, imp, name(imp.source))}</span>
        </div>
        {errorNote}
      </Dialog>
    )
  }
  return null
}
