import { useNavigate } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { TableScroll, Th } from '@/components/ui/Table'
import { useAudienceSummary } from '@/features/audience/AudienceTabs'
import { saveBlob, fileName as downloadName } from '@/features/audience/download'
import { NotifuseTemplatesOption } from '@/features/transactional-import/NotifuseOption'
import { m as notifuse } from '@/features/transactional-import/notifuse.messages'
import { t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ImportFrame, PreviousImport, stepPath, useImportDetail, useRefreshImport, WizardHeading, type Detail } from './ImportFrame'
import { ReadingSource } from './states'
import { PaidMembersImportHint } from '@/features/earn/entries'
import { m } from './messages'

type Source = Schemas['ImportCreateInput']['source']
type ListChoice = Schemas['ImportListChoice']

interface SourceInfo {
  id: Source
  name: string
  what: string
  /** What the source calls a list. */
  unit: string
  keyUrl?: string
}

// Getters, so the lines read in the app's current language.
export const sources: SourceInfo[] = [
  {
    id: 'brevo',
    name: 'Brevo',
    get what() {
      return t(m.whatBrevo)
    },
    unit: 'list',
    keyUrl: 'https://app.brevo.com/settings/keys/api',
  },
  {
    id: 'mailchimp',
    name: 'Mailchimp',
    get what() {
      return t(m.whatMailchimp)
    },
    unit: 'audience',
    keyUrl: 'https://admin.mailchimp.com/account/api/',
  },
  {
    id: 'mailerlite',
    name: 'MailerLite',
    get what() {
      return t(m.whatMailerlite)
    },
    unit: 'group',
    keyUrl: 'https://dashboard.mailerlite.com/integrations/api',
  },
  {
    id: 'listmonk',
    name: 'listmonk',
    get what() {
      return t(m.whatListmonk)
    },
    unit: 'list',
  },
  {
    id: 'keila',
    name: 'Keila',
    get what() {
      return t(m.whatKeila)
    },
    unit: 'segment',
  },
  {
    id: 'notifuse',
    name: 'Notifuse',
    get what() {
      return t(notifuse.sourceWhat)
    },
    unit: 'list',
  },
  {
    id: 'csv',
    get name() {
      return t(m.fileSource)
    },
    get what() {
      return t(m.whatFile)
    },
    unit: 'file',
  },
]

const isFile = (s?: string) => s === 'csv' || s === 'xlsx'
const sourceInfo = (s?: string) => sources.find((x) => x.id === (isFile(s) ? 'csv' : s))

/** Step 1: where the contacts live now (MigrationSource, MigrationStates). */
export function SourceStep({ importId }: { importId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useImportDetail(importId ?? '')
  const history = $api.useQuery('get', '/api/v1/workspaces/{workspace}/imports', { params: { path: { workspace: slug } } }, { enabled: !importId })
  const d = importId ? q.data : undefined
  const previous = d?.previous ?? history.data?.items.find((i) => i.status === 'done')
  return (
    <ImportFrame
      title={t(m.title)}
      badge={<Badge tone="neutral">{t(m.stepOf, { step: 1, total: 5 })}</Badge>}
      exit={
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          {t(m.exit)}
        </ButtonLink>
      }
      step={1}
      hints={{ 1: t(m.hintSource), 3: t(m.hintConsent), 4: t(m.hintPreview) }}
      footer={previous ? <PreviousImport p={previous} /> : undefined}
    >
      <WizardHeading title={t(m.sourceTitle)} lead={t(m.sourceLead)} />
      {importId && q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : importId && !d ? (
        <ReadingSource title={t(m.loading)} />
      ) : (
        <SourceBody d={d} firstImport={!importId && history.data?.items.length === 0} />
      )}
    </ImportFrame>
  )
}

function SourceBody({ d, firstImport }: { d?: Detail; firstImport: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshImport()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports')
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/imports/{id}')
  const [err, setErr] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const cards = useRef<HTMLDivElement>(null)
  const current = d ? (isFile(d.source) ? 'csv' : d.source) : undefined

  const pick = async (s: Source) => {
    setErr(null)
    try {
      if (!d) {
        const r = await create.mutateAsync({ params: { path: { workspace: slug } }, body: { source: s } })
        await navigate({ to: stepPath.source, params: { workspace: slug, importId: r.id } })
      } else if (s !== current) {
        await update.mutateAsync({ params: { path: { workspace: slug, id: d.id } }, body: { source: s } })
        await refresh()
      }
    } catch (e) {
      setErr(errorMessage(e))
    }
  }

  /** Uploads a file, creating the import first from the empty state. */
  const upload = async (file: File) => {
    setErr(null)
    if (file.size > 100 << 20) {
      setErr(t(m.tooBig))
      return
    }
    setUploading(true)
    try {
      let id = d?.id
      if (!id) {
        id = (await create.mutateAsync({ params: { path: { workspace: slug } }, body: { source: file.name.toLowerCase().endsWith('.xlsx') ? 'xlsx' : 'csv' } })).id
      }
      await http.POST('/api/v1/workspaces/{workspace}/imports/{id}/file', {
        params: { path: { workspace: slug, id }, query: { fileName: file.name } },
        body: (await file.arrayBuffer()) as unknown as string,
        headers: { 'Content-Type': 'application/octet-stream' },
        bodySerializer: (b: unknown) => b as BodyInit,
      })
      await refresh()
      if (!d) await navigate({ to: stepPath.source, params: { workspace: slug, importId: id } })
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setUploading(false)
    }
  }

  return (
    <>
      <div ref={cards} className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(190px,1fr))]" role="group" aria-label={t(m.sources)}>
        {sources.map((s) => (
          <button
            key={s.id}
            type="button"
            className={`src ${current === s.id ? 'on' : ''}`}
            aria-pressed={current === s.id}
            disabled={create.isPending || update.isPending}
            onClick={() => void pick(s.id)}
          >
            <span className="font-bold">{s.name}</span>
            <span className="k">{s.what}</span>
            {current === s.id ? (
              <Badge tone="ok" className="self-start">
                {t(m.selected)}
              </Badge>
            ) : null}
          </button>
        ))}
      </div>
      {/* Paid members import in Earn (X16). */}
      <PaidMembersImportHint />
      {err ? <Note tone="bad">{err}</Note> : null}
      {!d ? (
        <EmptyDrop firstImport={firstImport} busy={uploading} onFile={(f) => void upload(f)} onSources={() => cards.current?.querySelector('button')?.focus()} />
      ) : isFile(d.source) ? (
        <FileCard d={d} busy={uploading} onFile={(f) => void upload(f)} />
      ) : (
        <ConnectCard key={d.source} d={d} />
      )}
      {d && d.connection && d.connection.lists.length > 0 && !(isFile(d.source) && d.error) ? (
        <Choose key={`${d.id}:${d.connection.checkedAt}`} d={d} />
      ) : (
        <div className="flex flex-wrap justify-between gap-2.5">
          <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
            {t(common.cancel)}
          </ButtonLink>
          <Button variant="primary" className="wnext" disabledReason={d ? (isFile(d.source) ? t(m.needFile) : t(m.needConnect, { source: d.sourceName })) : t(m.needSource)}>
            {t(m.continueFields)}
          </Button>
        </div>
      )}
    </>
  )
}

/** Empty: no import yet, the CSV drop zone (MigrationStates). */
function EmptyDrop({ firstImport, busy, onFile, onSources }: { firstImport: boolean; busy: boolean; onFile: (f: File) => void; onSources: () => void }) {
  const t = useT()
  const input = useRef<HTMLInputElement>(null)
  const [over, setOver] = useState(false)
  return (
    <div
      className={`card migdrop ${over ? 'over' : ''}`}
      onDragOver={(e) => {
        e.preventDefault()
        setOver(true)
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        e.preventDefault()
        setOver(false)
        const f = e.dataTransfer.files[0]
        if (f) onFile(f)
      }}
    >
      <svg width="48" height="48" viewBox="0 0 48 48" aria-hidden="true">
        <path d="M24 32V12M15 21l9-9 9 9M10 38h28" fill="none" stroke="#9AA3B2" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
      <span className="text-[17px] font-bold">{t(m.dropTitle)}</span>
      <span className="lead max-w-[520px]">{t(m.dropLead, { max: count(500_000) })}</span>
      <div className="mt-1 flex flex-wrap justify-center gap-2.5">
        <input
          ref={input}
          type="file"
          accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          className="vh"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => {
            const f = e.target.files?.[0]
            if (f) onFile(f)
            e.target.value = ''
          }}
        />
        <Button variant="primary" loading={busy} onClick={() => input.current?.click()}>
          {t(m.chooseFile)}
        </Button>
        <Button onClick={onSources}>{t(m.connectApis)}</Button>
      </div>
      {firstImport ? <span className="k">{t(m.noImportsYet)}</span> : null}
    </div>
  )
}

/** The uploaded file, or what's wrong with it (MigrationStates › broken file). */
function FileCard({ d, busy, onFile }: { d: Detail; busy: boolean; onFile: (f: File) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshImport()
  const skip = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/skip-bad-rows')
  const input = useRef<HTMLInputElement>(null)
  const [err, setErr] = useState<string | null>(null)
  const picker = (
    <input
      ref={input}
      type="file"
      accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
      className="vh"
      tabIndex={-1}
      aria-hidden="true"
      onChange={(e) => {
        const f = e.target.files?.[0]
        if (f) onFile(f)
        e.target.value = ''
      }}
    />
  )
  if (!d.fileName) {
    return <EmptyDrop firstImport={false} busy={busy} onFile={onFile} onSources={() => undefined} />
  }
  const e = d.error
  if (e && (e.code === 'file_columns' || e.code === 'file_unreadable')) {
    const bad = e.badRows ?? []
    const first = bad[0]
    const more = bad.slice(1).map((b) => count(b.line))
    return (
      <div role="alert" className="card flex flex-col gap-2 border-[#FCA5A5] px-4.5 py-4">
        {picker}
        <div className="flex items-center gap-2.5">
          <Badge tone="bad">{t(m.fileBadge)}</Badge>
          <span className="font-semibold">
            {first && e.header
              ? t(m.rowColumns, { file: d.fileName, row: count(first.line), columns: first.columns, header: e.header })
              : t(m.fileProblem, { file: d.fileName, message: e.message })}
          </span>
        </div>
        <span className="k">
          {first
            ? [
                t(m.unquoted),
                ...(more.length > 0 ? [t(m.moreRows, { n: more.length, rows: `${more.slice(0, 5).join(', ')}${more.length > 5 ? ', …' : ''}` })] : []),
                t(m.skipOrFix, { n: bad.length }),
              ].join(' ')
            : e.message}
        </span>
        {err ? <Note tone="bad">{err}</Note> : null}
        <div className="flex flex-wrap gap-2">
          {bad.length > 0 ? (
            <Button
              size="sm"
              variant="primary"
              loading={skip.isPending}
              onClick={async () => {
                setErr(null)
                try {
                  await skip.mutateAsync({ params: { path: { workspace: slug, id: d.id } } })
                  await refresh()
                } catch (x) {
                  setErr(errorMessage(x))
                }
              }}
            >
              {t(m.skipRows, { n: bad.length, count: count(bad.length) })}
            </Button>
          ) : null}
          {bad.length > 0 ? (
            <Button size="sm" onClick={() => void download(slug, d.id, 'unreadable')}>
              {t(m.downloadRows, { n: bad.length, count: count(bad.length) })}
            </Button>
          ) : null}
          <Button size="sm" loading={busy} onClick={() => input.current?.click()}>
            {t(m.uploadFixed)}
          </Button>
        </div>
      </div>
    )
  }
  const list = d.connection?.lists[0]
  return (
    <section className="card flex flex-col gap-3.5 px-5 py-4.5">
      {picker}
      <h3 className="h2">{t(m.yourFile)}</h3>
      <div className="flex flex-wrap items-center gap-3">
        <span className="mono font-semibold">{d.fileName}</span>
        <Badge tone="ok" dot>
          {t(m.fileRead, { rows: list?.subscribed ?? d.stagedRows, rowCount: count(list?.subscribed ?? d.stagedRows), columns: d.fieldCount })}
        </Badge>
        {d.options.skipRows.length > 0 ? <span className="k">{t(m.brokenLeftOut, { n: d.options.skipRows.length, count: count(d.options.skipRows.length) })}</span> : null}
        <Button size="sm" className="ml-auto" loading={busy} onClick={() => input.current?.click()}>
          {t(m.uploadAnother)}
        </Button>
      </div>
      <span className="k">{t(m.fileStays)}</span>
    </section>
  )
}

async function download(slug: string, id: string, kind: 'report' | 'skipped' | 'invalid' | 'unreadable') {
  try {
    const { data, response } = await http.GET('/api/v1/workspaces/{workspace}/imports/{id}/files/{kind}', {
      params: { path: { workspace: slug, id, kind } },
      parseAs: 'blob',
    })
    if (data) saveBlob(data as unknown as Blob, downloadName(response, `import-${kind}.csv`))
  } catch (e) {
    toast({ kind: 'error', title: t(m.downloadFailed), detail: errorMessage(e) })
  }
}
export { download as downloadImportFile }

/** The data center a Mailchimp key belongs to ("…-us21"). */
function dataCenter(key: string) {
  return /-([a-z]{2}\d{1,3})$/.exec(key.trim())?.[1]
}

/** Connect <source>: the key (write-only), its address, and what was found. */
function ConnectCard({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshImport()
  const info = sourceInfo(d.source)
  const connect = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/connect')
  const [key, setKey] = useState('')
  const [user, setUser] = useState('')
  const [base, setBase] = useState(d.connection?.baseUrl ?? (d.source === 'keila' ? 'https://app.keila.io' : ''))
  const [err, setErr] = useState<string | null>(null)
  const selfHosted = d.source === 'listmonk' || d.source === 'keila' || d.source === 'notifuse'
  const authFailed = (connect.error instanceof ApiError && connect.error.code === 'source_auth_failed') || (!connect.isPending && d.error?.code === 'source_auth_failed')
  const submit = async () => {
    setErr(null)
    if (!key.trim()) {
      setErr(t(m.pasteKey, { source: d.sourceName }))
      return
    }
    try {
      await connect.mutateAsync({
        params: { path: { workspace: slug, id: d.id } },
        body: { apiKey: key.trim(), user: user.trim() || undefined, baseUrl: selfHosted ? base.trim() || undefined : undefined },
      })
      setKey('')
      await refresh()
    } catch (e) {
      await refresh()
      if (!(e instanceof ApiError && e.code === 'source_auth_failed')) setErr(errorMessage(e))
    }
  }
  if (connect.isPending) return <ReadingSource title={t(m.checkingKey, { unit: info?.unit ?? 'list', source: d.sourceName })} />
  if (authFailed) {
    const msg = connect.error instanceof ApiError ? connect.error.message : (d.error?.message ?? '')
    // The card title already says who rejected the key.
    const body = msg.replace(/^.*? rejected the API key:\s*/, '')
    return (
      <div role="alert" className="card flex flex-col gap-2 border-[#FCA5A5] px-4.5 py-4">
        <div className="flex items-center gap-2.5">
          <Badge tone="bad">{d.error?.status ?? 401}</Badge>
          <span className="font-semibold">{t(m.keyRejected, { source: d.sourceName })}</span>
        </div>
        <span className="k">
          {body.split(/(x{8}-[a-z]{2}\d{1,3})/).map((part, i) => (i % 2 ? <span key={i} className="mono">{part}</span> : part))}
          {d.source === 'mailchimp' ? '' : ` ${t(m.keyActive)}`}
        </span>
        <form
          className="contents"
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
        >
          <label className="lbl">
            {t(m.apiKey)}
            <input
              className="in mono"
              type="password"
              autoComplete="off"
              placeholder={t(m.pasteAgain)}
              aria-invalid="true"
              style={{ borderColor: '#DC2626', background: '#FFF5F5' }}
              value={key}
              onChange={(e) => setKey(e.target.value)}
            />
          </label>
          {err ? <span className="err">{err}</span> : null}
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="primary" type="submit">
              {t(common.tryAgain)}
            </Button>
            {info?.keyUrl ? (
              <ButtonAnchor size="sm" href={info.keyUrl} newTab>
                {t(m.keysPage, { source: d.sourceName })}
              </ButtonAnchor>
            ) : null}
          </div>
        </form>
      </div>
    )
  }
  const conn = d.connection
  return (
    <section className="card flex flex-col gap-3.5 px-5 py-4.5">
      <h3 className="h2">{t(m.connectSource, { source: d.sourceName })}</h3>
      <form
        className="contents"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <div className="mgrid grid items-end gap-3 [grid-template-columns:repeat(auto-fit,minmax(240px,1fr))]">
          {selfHosted ? (
            <label className="lbl">
              {t(m.serverAddress)}
              <input
                className="in mono"
                type="url"
                placeholder={d.source === 'notifuse' ? 'https://notifuse.example.com' : 'https://lists.example.com'}
                value={base}
                onChange={(e) => setBase(e.target.value)}
              />
            </label>
          ) : null}
          {d.source === 'listmonk' ? (
            <label className="lbl">
              {t(m.apiUser)}
              <input className="in mono" autoComplete="off" value={user} onChange={(e) => setUser(e.target.value)} />
            </label>
          ) : null}
          <label className={`lbl ${selfHosted ? '' : 'col-span-2'}`}>
            {d.source === 'listmonk' ? t(m.apiToken) : t(m.apiKey)}
            <input
              className="in mono"
              type="password"
              autoComplete="off"
              placeholder={t(m.keyPlaceholder, { source: d.sourceName })}
              value={key}
              onChange={(e) => setKey(e.target.value)}
            />
          </label>
          {d.source === 'mailchimp' ? (
            <label className="lbl">
              {t(m.dataCenter)}
              <input className="in mono" value={dataCenter(key) ?? conn?.dataCenter ?? ''} readOnly />
              <span className="k font-medium">{t(m.fromSuffix)}</span>
            </label>
          ) : null}
        </div>
        {err ? <Note tone="bad">{err}</Note> : null}
        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit">{t(m.connect)}</Button>
          {conn ? (
            <Badge tone="ok" dot>
              {t(m.connected, { account: conn.account, n: conn.lists.length, count: count(conn.lists.length), unit: info?.unit ?? 'list' })}
            </Badge>
          ) : null}
          <span className="k">{t(m.nothingWritten, { source: d.sourceName })}</span>
        </div>
      </form>
    </section>
  )
}

/** Choose what to import: lists, where each goes, tags, engagement, suppressions. */
function Choose({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshImport()
  const summary = useAudienceSummary()
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/imports/{id}')
  const stage = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/stage')
  const reason = usePermissionReason('subscribers:import')
  const lists = summary.data?.lists ?? []
  const conn = d.connection!
  const info = sourceInfo(d.source)
  const file = isFile(d.source)
  // Lists picked on connect but not placed yet go to a new list named after
  // them, so the select never shows a list the draft doesn't hold.
  const [opts, setOpts] = useState<Schemas['ImportOptions']>(() => ({
    ...d.options,
    lists: d.options.lists.map((l) => (l.target ? l : { ...l, target: `new:${conn.lists.find((x) => x.id === l.sourceId)?.name ?? t(m.defaultList)}` })),
  }))
  const [err, setErr] = useState<string | null>(null)
  const choice = (id: string) => opts.lists.find((l) => l.sourceId === id)
  const setChoice = (id: string, c: ListChoice | null) =>
    setOpts((o) => ({ ...o, lists: c ? [...o.lists.filter((l) => l.sourceId !== id), c] : o.lists.filter((l) => l.sourceId !== id) }))
  // A new list named after the source's, unless the workspace has exactly one list.
  const defaultTarget = (name: string) => (lists.length === 1 && lists[0] ? lists[0].id : `new:${name}`)
  const picked = conn.lists.filter((l) => choice(l.id))
  const tags = picked.reduce((n, l) => n + l.tags, 0)
  const examples = picked.flatMap((l) => l.tagExamples).slice(0, 3)
  const synced = d.providerSync.filter((p) => p.pulled > 0)
  const missing = picked.length === 0 ? t(m.chooseOne, { unit: info?.unit ?? 'list' }) : opts.lists.some((l) => !l.target) ? t(m.pickTarget) : undefined
  const submit = async () => {
    setErr(null)
    try {
      await update.mutateAsync({ params: { path: { workspace: slug, id: d.id } }, body: { options: opts } })
      await stage.mutateAsync({ params: { path: { workspace: slug, id: d.id } } })
      await refresh()
      await navigate({ to: stepPath.fields, params: { workspace: slug, importId: d.id } })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <>
      <section className="card flex flex-col">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
          <h3 className="h2">{t(m.chooseTitle)}</h3>
          <span className="k">{file ? t(m.readFrom, { file: d.fileName }) : t(m.countsFrom, { source: d.sourceName, time: clock(conn.checkedAt) })}</span>
        </div>
        <TableScroll label={t(m.whatToImport)}>
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">
                  <span className="vh">{t(m.colImport)}</span>
                </th>
                <Th>{file ? t(m.colFile) : t(m.colSourceList, { source: d.sourceName, unit: info?.unit ?? 'list' })}</Th>
                {file ? (
                  <Th align="right">{t(m.colRows)}</Th>
                ) : (
                  <>
                    <Th align="right">{t(m.colSubscribed)}</Th>
                    <Th align="right">{t(m.colUnsubscribed)}</Th>
                    <Th align="right">{t(m.colCleaned)}</Th>
                    <Th align="right">{t(m.colPending)}</Th>
                  </>
                )}
                <Th>{t(m.colInto)}</Th>
              </tr>
            </thead>
            <tbody>
              {conn.lists.map((l) => {
                const c = choice(l.id)
                return (
                  <tr key={l.id}>
                    <td>
                      <input
                        type="checkbox"
                        aria-label={t(m.importList, { name: l.name })}
                        checked={Boolean(c)}
                        onChange={(e) => setChoice(l.id, e.target.checked ? { sourceId: l.id, target: defaultTarget(l.name) } : null)}
                      />
                    </td>
                    <td className="font-semibold">
                      {l.name}
                      <div className="k">
                        {file
                          ? t(m.listColumns, { n: l.fields })
                          : [
                              t(m.listId, { unit: info?.unit ?? 'list', id: l.id }),
                              l.fields ? t(m.listFields, { merge: d.source === 'mailchimp' ? 'yes' : 'no', n: l.fields }) : '',
                              l.tags ? t(m.listTags, { n: l.tags }) : '',
                            ]
                              .filter(Boolean)
                              .join(' · ')}
                      </div>
                    </td>
                    {file ? (
                      <td className="r n">{count(l.subscribed)}</td>
                    ) : (
                      <>
                        <td className="r n">{count(l.subscribed)}</td>
                        <td className="r n">{count(l.unsubscribed)}</td>
                        <td className="r n">{count(l.cleaned)}</td>
                        <td className="r n">{count(l.pending)}</td>
                      </>
                    )}
                    <td>
                      <select
                        aria-label={t(m.intoList)}
                        className="in"
                        disabled={!c}
                        value={c?.target ?? ''}
                        onChange={(e) => setChoice(l.id, { sourceId: l.id, target: e.target.value })}
                      >
                        {!c ? <option value="">{t(m.notImported)}</option> : null}
                        {lists.map((x) => (
                          <option key={x.id} value={x.id}>
                            {x.name}
                          </option>
                        ))}
                        <option value={`new:${l.name}`}>{t(m.newList, { name: l.name })}</option>
                      </select>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
        <div className="flex flex-col gap-2.5 border-t border-line-2 px-5 py-3.5">
          {!file ? (
            <>
              <label className="flex items-start gap-2.5 text-[13px]">
                <input type="checkbox" className="mt-[3px]" checked={opts.importTags} onChange={(e) => setOpts({ ...opts, importTags: e.target.checked })} />
                <span>
                  {t(m.importTags)}{' '}
                  {tags > 0 ? (
                    <span className="k">
                      {examples.length > 0
                        ? t(m.tagCountExamples, { n: tags, count: count(tags), examples: examples.map((x) => `"${x}"`).join(', ') })
                        : t(m.tagCount, { n: tags, count: count(tags) })}
                    </span>
                  ) : null}
                </span>
              </label>
              <label className="flex items-start gap-2.5 text-[13px]">
                <input type="checkbox" className="mt-[3px]" checked={opts.importEngagement} onChange={(e) => setOpts({ ...opts, importEngagement: e.target.checked })} />
                <span>
                  {t(m.importEngagement, { source: d.sourceName })} <span className="k">{t(m.engagementNote, { rating: d.source === 'mailchimp' ? 'yes' : 'no' })}</span>
                </span>
              </label>
            </>
          ) : null}
          {d.source === 'notifuse' && conn.templates ? (
            <NotifuseTemplatesOption counts={conn.templates} checked={opts.importTemplates} onChange={(v) => setOpts({ ...opts, importTemplates: v })} />
          ) : null}
          <label className="flex items-start gap-2.5 text-[13px]">
            <input type="checkbox" className="mt-[3px]" checked={opts.pullSuppressions} onChange={(e) => setOpts({ ...opts, pullSuppressions: e.target.checked })} />
            <span>
              {t(m.pullSuppressions)}{' '}
              {synced.length > 0 ? <span className="k">{t(m.alreadyShared, { providers: synced.map((p) => `${p.short} ${count(p.pulled)}`).join(', ') })}</span> : null}
            </span>
          </label>
        </div>
      </section>
      {err ? <Note tone="bad">{err}</Note> : null}
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          {t(common.cancel)}
        </ButtonLink>
        <Button variant="primary" className="wnext" disabledReason={reason ?? missing} loading={update.isPending || stage.isPending} onClick={() => void submit()}>
          {t(m.continueFields)}
        </Button>
      </div>
    </>
  )
}
