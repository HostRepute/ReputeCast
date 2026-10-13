import { useNavigate } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useAudienceSummary } from '@/features/audience/AudienceTabs'
import { saveBlob, fileName as downloadName } from '@/features/audience/download'
import { clock, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ImportFrame, PreviousImport, stepPath, useImportDetail, useRefreshImport, WizardHeading, type Detail } from './ImportFrame'
import { ReadingSource } from './states'

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

export const sources: SourceInfo[] = [
  { id: 'brevo', name: 'Brevo', what: 'API key · lists, attributes, blocklist', unit: 'list', keyUrl: 'https://app.brevo.com/settings/keys/api' },
  { id: 'mailchimp', name: 'Mailchimp', what: 'API key · audiences, tags, merge fields, cleaned', unit: 'audience', keyUrl: 'https://admin.mailchimp.com/account/api/' },
  { id: 'mailerlite', name: 'MailerLite', what: 'API key · groups, fields', unit: 'group', keyUrl: 'https://dashboard.mailerlite.com/integrations/api' },
  { id: 'listmonk', name: 'listmonk', what: 'API · lists, attribs', unit: 'list' },
  { id: 'keila', name: 'Keila', what: 'API key · segments, data', unit: 'segment' },
  { id: 'csv', name: 'CSV or XLSX', what: 'Drag a file · map columns yourself', unit: 'file' },
]

const isFile = (s?: string) => s === 'csv' || s === 'xlsx'
const sourceInfo = (s?: string) => sources.find((x) => x.id === (isFile(s) ? 'csv' : s))

/** Step 1: where the contacts live now (MigrationSource, MigrationStates). */
export function SourceStep({ importId }: { importId?: string }) {
  const { slug } = useWorkspace()
  const q = useImportDetail(importId ?? '')
  const history = $api.useQuery('get', '/api/v1/workspaces/{workspace}/imports', { params: { path: { workspace: slug } } }, { enabled: !importId })
  const d = importId ? q.data : undefined
  const previous = d?.previous ?? history.data?.items.find((i) => i.status === 'done')
  return (
    <ImportFrame
      title="Import contacts"
      badge={<Badge tone="neutral">Step 1 of 5</Badge>}
      exit={
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          Exit
        </ButtonLink>
      }
      step={1}
      hints={{ 1: 'Where the contacts live now', 3: 'Suppressions stay suppressed', 4: 'Duplicates, invalid, risky' }}
      footer={previous ? <PreviousImport p={previous} /> : undefined}
    >
      <WizardHeading
        title="Where are your contacts today?"
        lead="API imports bring lists, fields, tags and consent status. Unsubscribed, bounced and complained contacts become suppressions and are never re-subscribed."
      />
      {importId && q.isError ? (
        <ErrorPanel title="Couldn't load this import" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : importId && !d ? (
        <ReadingSource title="Loading the import" />
      ) : (
        <SourceBody d={d} firstImport={!importId && history.data?.items.length === 0} />
      )}
    </ImportFrame>
  )
}

function SourceBody({ d, firstImport }: { d?: Detail; firstImport: boolean }) {
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
      setErr('That file is over 100 MB. Split it, or import from the source’s API.')
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
      <div ref={cards} className="mgrid grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(190px,1fr))]" role="group" aria-label="Sources">
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
                Selected
              </Badge>
            ) : null}
          </button>
        ))}
      </div>
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
            Cancel
          </ButtonLink>
          <Button variant="primary" className="wnext" disabledReason={d ? (isFile(d.source) ? 'Upload a file first.' : `Connect ${d.sourceName} first.`) : 'Pick a source first.'}>
            Continue to field mapping
          </Button>
        </div>
      )}
    </>
  )
}

/** Empty: no import yet, the CSV drop zone (MigrationStates). */
function EmptyDrop({ firstImport, busy, onFile, onSources }: { firstImport: boolean; busy: boolean; onFile: (f: File) => void; onSources: () => void }) {
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
      <span className="text-[17px] font-bold">Drop a CSV or XLSX here, or choose a source above</span>
      <span className="lead max-w-[520px]">Up to 500,000 rows. The first row should be headers. You'll map columns and pick a consent column on the next steps.</span>
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
          Choose a file
        </Button>
        <Button onClick={onSources}>Connect Brevo, Mailchimp, MailerLite, listmonk or Keila</Button>
      </div>
      {firstImport ? <span className="k">No imports yet in this workspace.</span> : null}
    </div>
  )
}

/** The uploaded file, or what's wrong with it (MigrationStates › broken file). */
function FileCard({ d, busy, onFile }: { d: Detail; busy: boolean; onFile: (f: File) => void }) {
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
          <Badge tone="bad">File</Badge>
          <span className="font-semibold">
            {first && e.header ? `${d.fileName}: row ${count(first.line)} has ${first.columns} columns, the header has ${e.header}` : `${d.fileName}: ${e.message}`}
          </span>
        </div>
        <span className="k">
          {first
            ? `Usually an unquoted comma inside a value.${more.length > 0 ? ` ${more.length} more ${more.length === 1 ? 'row has' : 'rows have'} the same problem (${more.slice(0, 5).join(', ')}${more.length > 5 ? ', …' : ''}).` : ''} You can skip ${bad.length === 1 ? 'that row' : 'those rows'} or fix the file and upload again.`
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
              Skip the {count(bad.length)} {bad.length === 1 ? 'row' : 'rows'} and continue
            </Button>
          ) : null}
          {bad.length > 0 ? (
            <Button size="sm" onClick={() => void download(slug, d.id, 'unreadable')}>
              Download the {count(bad.length)} {bad.length === 1 ? 'row' : 'rows'}
            </Button>
          ) : null}
          <Button size="sm" loading={busy} onClick={() => input.current?.click()}>
            Upload a fixed file
          </Button>
        </div>
      </div>
    )
  }
  const list = d.connection?.lists[0]
  return (
    <section className="card flex flex-col gap-3.5 px-5 py-4.5">
      {picker}
      <h3 className="h2">Your file</h3>
      <div className="flex flex-wrap items-center gap-3">
        <span className="mono font-semibold">{d.fileName}</span>
        <Badge tone="ok" dot>
          Read · {count(list?.subscribed ?? d.stagedRows)} rows · {d.fieldCount} columns
        </Badge>
        {d.options.skipRows.length > 0 ? <span className="k">{count(d.options.skipRows.length)} broken rows left out</span> : null}
        <Button size="sm" className="ml-auto" loading={busy} onClick={() => input.current?.click()}>
          Upload another file
        </Button>
      </div>
      <span className="k">The file stays in this workspace's database until the import finishes or you discard it.</span>
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
    toast({ kind: 'error', title: "Couldn't download the rows", detail: errorMessage(e) })
  }
}
export { download as downloadImportFile }

/** The data center a Mailchimp key belongs to ("…-us21"). */
function dataCenter(key: string) {
  return /-([a-z]{2}\d{1,3})$/.exec(key.trim())?.[1]
}

/** Connect <source>: the key (write-only), its address, and what was found. */
function ConnectCard({ d }: { d: Detail }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshImport()
  const info = sourceInfo(d.source)
  const connect = $api.useMutation('post', '/api/v1/workspaces/{workspace}/imports/{id}/connect')
  const [key, setKey] = useState('')
  const [user, setUser] = useState('')
  const [base, setBase] = useState(d.connection?.baseUrl ?? (d.source === 'keila' ? 'https://app.keila.io' : ''))
  const [err, setErr] = useState<string | null>(null)
  const selfHosted = d.source === 'listmonk' || d.source === 'keila'
  const authFailed = (connect.error instanceof ApiError && connect.error.code === 'source_auth_failed') || (!connect.isPending && d.error?.code === 'source_auth_failed')
  const submit = async () => {
    setErr(null)
    if (!key.trim()) {
      setErr(`Paste the ${d.sourceName} API key.`)
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
  if (connect.isPending) return <ReadingSource title={`Checking the key and reading ${info?.unit ?? 'list'}s from ${d.sourceName}`} />
  if (authFailed) {
    const msg = connect.error instanceof ApiError ? connect.error.message : (d.error?.message ?? '')
    // The source's own words: "Mailchimp rejected the API key (API key invalid)." → "API key invalid".
    const said = /\(([^()]+)\)\.?$/.exec(msg)?.[1]
    const dc = dataCenter(key) ?? d.connection?.dataCenter
    return (
      <div role="alert" className="card flex flex-col gap-2 border-[#FCA5A5] px-4.5 py-4">
        <div className="flex items-center gap-2.5">
          <Badge tone="bad">{d.error?.status ?? 401}</Badge>
          <span className="font-semibold">{d.sourceName} rejected the API key</span>
        </div>
        <span className="k">
          {d.source === 'mailchimp' ? (
            <>
              {said ? `"${said}"` : msg}
              {dc ? ` from the ${dc} data center` : ''}. Keys look like <span className="mono">xxxxxxxx-us21</span>; the suffix must match the data center the account lives
              in.
            </>
          ) : (
            `${said ? `"${said}".` : msg} Check that the key is active and allowed to read contacts.`
          )}
        </span>
        <form
          className="contents"
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
        >
          <label className="lbl">
            API key
            <input
              className="in mono"
              type="password"
              autoComplete="off"
              placeholder="Paste again"
              aria-invalid="true"
              style={{ borderColor: '#DC2626', background: '#FFF5F5' }}
              value={key}
              onChange={(e) => setKey(e.target.value)}
            />
          </label>
          {err ? <span className="err">{err}</span> : null}
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="primary" type="submit">
              Try again
            </Button>
            {info?.keyUrl ? (
              <ButtonAnchor size="sm" href={info.keyUrl} newTab>
                {d.sourceName} API keys
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
      <h3 className="h2">Connect {d.sourceName}</h3>
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
              Server address
              <input className="in mono" type="url" placeholder="https://lists.example.com" value={base} onChange={(e) => setBase(e.target.value)} />
            </label>
          ) : null}
          {d.source === 'listmonk' ? (
            <label className="lbl">
              API user
              <input className="in mono" autoComplete="off" value={user} onChange={(e) => setUser(e.target.value)} />
            </label>
          ) : null}
          <label className={`lbl ${selfHosted ? '' : 'col-span-2'}`}>
            {d.source === 'listmonk' ? 'API token' : 'API key'}
            <input
              className="in mono"
              type="password"
              autoComplete="off"
              placeholder={`Paste a ${d.sourceName} API key (read-only is enough)`}
              value={key}
              onChange={(e) => setKey(e.target.value)}
            />
          </label>
          {d.source === 'mailchimp' ? (
            <label className="lbl">
              Data center
              <input className="in mono" value={dataCenter(key) ?? conn?.dataCenter ?? ''} readOnly />
              <span className="k font-medium">detected from the key suffix</span>
            </label>
          ) : null}
        </div>
        {err ? <Note tone="bad">{err}</Note> : null}
        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit">Connect</Button>
          {conn ? (
            <Badge tone="ok" dot>
              Connected · account "{conn.account}" · {count(conn.lists.length)} {conn.lists.length === 1 ? info?.unit : `${info?.unit}s`}
            </Badge>
          ) : null}
          <span className="k">Nothing is written to {d.sourceName}.</span>
        </div>
      </form>
    </section>
  )
}

/** Choose what to import: lists, where each goes, tags, engagement, suppressions. */
function Choose({ d }: { d: Detail }) {
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
  const [opts, setOpts] = useState<Schemas['ImportOptions']>(() => d.options)
  const [err, setErr] = useState<string | null>(null)
  const choice = (id: string) => opts.lists.find((l) => l.sourceId === id)
  const setChoice = (id: string, c: ListChoice | null) =>
    setOpts((o) => ({ ...o, lists: c ? [...o.lists.filter((l) => l.sourceId !== id), c] : o.lists.filter((l) => l.sourceId !== id) }))
  const defaultTarget = (name: string) => lists.find((l) => l.name === 'Product Updates')?.id ?? lists[0]?.id ?? `new:${name}`
  const picked = conn.lists.filter((l) => choice(l.id))
  const tags = picked.reduce((n, l) => n + l.tags, 0)
  const examples = picked.flatMap((l) => l.tagExamples).slice(0, 3)
  const synced = d.providerSync.filter((p) => p.pulled > 0)
  const missing = picked.length === 0 ? `Choose at least one ${info?.unit ?? 'list'} to import.` : opts.lists.some((l) => !l.target) ? 'Pick the list each one goes into.' : undefined
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
          <h3 className="h2">Choose what to import</h3>
          <span className="k">{file ? `read from ${d.fileName}` : `counts from the ${d.sourceName} API, ${clock(conn.checkedAt)}`}</span>
        </div>
        <div className="tscroll">
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">
                  <span className="vh">Import</span>
                </th>
                <th scope="col">{file ? 'File' : `${d.sourceName} ${info?.unit ?? 'list'}`}</th>
                {file ? (
                  <th scope="col" className="r">
                    Rows
                  </th>
                ) : (
                  <>
                    <th scope="col" className="r">
                      Subscribed
                    </th>
                    <th scope="col" className="r">
                      Unsubscribed
                    </th>
                    <th scope="col" className="r">
                      Cleaned (bounced)
                    </th>
                    <th scope="col" className="r">
                      Pending
                    </th>
                  </>
                )}
                <th scope="col">Import into</th>
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
                        aria-label={`Import ${l.name}`}
                        checked={Boolean(c)}
                        onChange={(e) => setChoice(l.id, e.target.checked ? { sourceId: l.id, target: defaultTarget(l.name) } : null)}
                      />
                    </td>
                    <td className="font-semibold">
                      {l.name}
                      <div className="k">
                        {file
                          ? `${l.fields} columns`
                          : [`${info?.unit ?? 'list'} id ${l.id}`, l.fields ? `${l.fields} ${d.source === 'mailchimp' ? 'merge fields' : 'fields'}` : '', l.tags ? `${l.tags} tags` : '']
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
                        aria-label="Import into list"
                        className="in"
                        disabled={!c}
                        value={c?.target ?? ''}
                        onChange={(e) => setChoice(l.id, { sourceId: l.id, target: e.target.value })}
                      >
                        {!c ? <option value="">Not imported</option> : null}
                        {lists.map((x) => (
                          <option key={x.id} value={x.id}>
                            {x.name}
                          </option>
                        ))}
                        <option value={`new:${l.name}`}>New list "{l.name}"</option>
                      </select>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        <div className="flex flex-col gap-2.5 border-t border-line-2 px-5 py-3.5">
          {!file ? (
            <>
              <label className="flex items-start gap-2.5 text-[13px]">
                <input type="checkbox" className="mt-[3px]" checked={opts.importTags} onChange={(e) => setOpts({ ...opts, importTags: e.target.checked })} />
                <span>
                  Import tags as ReputeCast tags{' '}
                  {tags > 0 ? (
                    <span className="k">
                      · {count(tags)} tags{examples.length > 0 ? `, for example ${examples.map((t) => `"${t}"`).join(', ')}` : ''}
                    </span>
                  ) : null}
                </span>
              </label>
              <label className="flex items-start gap-2.5 text-[13px]">
                <input type="checkbox" className="mt-[3px]" checked={opts.importEngagement} onChange={(e) => setOpts({ ...opts, importEngagement: e.target.checked })} />
                <span>
                  Import engagement history where {d.sourceName} exposes it{' '}
                  <span className="k">· last open and click dates{d.source === 'mailchimp' ? ', member rating' : ''} → used for tiers</span>
                </span>
              </label>
            </>
          ) : null}
          <label className="flex items-start gap-2.5 text-[13px]">
            <input type="checkbox" className="mt-[3px]" checked={opts.pullSuppressions} onChange={(e) => setOpts({ ...opts, pullSuppressions: e.target.checked })} />
            <span>
              Also pull existing suppressions from connected sending providers{' '}
              {synced.length > 0 ? <span className="k">· {synced.map((p) => `${p.short} ${count(p.pulled)}`).join(', ')} are already in the shared list</span> : null}
            </span>
          </label>
        </div>
      </section>
      {err ? <Note tone="bad">{err}</Note> : null}
      <div className="flex flex-wrap justify-between gap-2.5">
        <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }}>
          Cancel
        </ButtonLink>
        <Button variant="primary" className="wnext" disabledReason={reason ?? missing} loading={update.isPending || stage.isPending} onClick={() => void submit()}>
          Continue to field mapping
        </Button>
      </div>
    </>
  )
}
