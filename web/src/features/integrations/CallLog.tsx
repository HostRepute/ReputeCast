import { Fragment, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { T, formatList, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, dayStamp, entryName, InfoIcon, itemName, tags, type ConnectKind } from './shared'

type Call = Schemas['IntegrationCallView']
type Window = '24h' | '7d'

const PAGE = 25

/** HTTP's own reason phrases, for the response line ("422 Unprocessable Content"). */
const reasons: Record<number, string> = {
  200: 'OK',
  201: 'Created',
  202: 'Accepted',
  204: 'No Content',
  400: 'Bad Request',
  401: 'Unauthorized',
  403: 'Forbidden',
  404: 'Not Found',
  409: 'Conflict',
  410: 'Gone',
  413: 'Content Too Large',
  422: 'Unprocessable Content',
  429: 'Too Many Requests',
  500: 'Internal Server Error',
  502: 'Bad Gateway',
  503: 'Service Unavailable',
  504: 'Gateway Timeout',
}

export function statusText(status: number): string {
  return reasons[status] ?? ''
}

/**
 * Calls and errors (IntegrationsZapier): the 7-day call log of an
 * integration's key, filtered by All or Errors and by 24 hours or 7 days.
 * Each error says what to change in the other tool; a row opens its
 * request and response, secrets masked.
 */
export function CallLog({ connectionId, kind, errorsFirst }: { connectionId: string; kind: ConnectKind; errorsFirst: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[kind])
  const [errors, setErrors] = useState(errorsFirst)
  const [windowKey, setWindow] = useState<Window>(errorsFirst ? '7d' : '24h')
  const [offset, setOffset] = useState(0)
  const q = $api.useQuery('get', `${base}/connections/{id}/calls`, {
    params: { path: { workspace: slug, id: connectionId }, query: { errors, window: windowKey, offset, limit: PAGE } },
  })
  const [open, setOpen] = useState<Set<string>>(new Set())
  const [auto, setAuto] = useState(true)
  const data = q.data
  // The latest error opens with its request, as on the board, until it's closed.
  const first = data?.items[0]
  const autoId = auto && errors && offset === 0 && first && first.status >= 400 ? first.id : undefined
  const isOpenRow = (id: string) => open.has(id) || id === autoId

  const toggle = (id: string) => {
    const was = isOpenRow(id)
    if (id === autoId) setAuto(false)
    setOpen((prev) => {
      const next = new Set(prev)
      if (was) next.delete(id)
      else next.add(id)
      return next
    })
  }
  const pageTotal = data ? (errors ? data.errors : data.total) : 0
  const items = data?.items ?? []

  return (
    <section className="card" aria-labelledby="h-calls">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-calls">
            {t(m.callsTitle)}
          </h2>
          <span className="k">{t(m.callsSub, { name })}</span>
        </div>
      </div>
      <div className="ih-filters">
        <div className="row gap-2">
          <Segmented
            label={t(m.show)}
            value={errors ? 'errors' : 'all'}
            onChange={(v) => {
              setErrors(v === 'errors')
              setOffset(0)
            }}
            items={[
              { value: 'all', label: t(m.all) },
              { value: 'errors', label: t(m.errors) },
            ]}
          />
          <Segmented
            label={t(m.period)}
            value={windowKey}
            onChange={(v) => {
              setWindow(v)
              setOffset(0)
            }}
            items={[
              { value: '24h', label: t(m.h24) },
              { value: '7d', label: t(m.d7) },
            ]}
          />
        </div>
        {data ? (
          <span className="k">
            {errors ? t(m.callsSummaryErrors, { errors: data.errors, total: data.total }) : t(m.callsSummary, { total: data.total, errors: data.errors, window: windowKey === '24h' ? 'h24' : 'd7' })}
          </span>
        ) : null}
      </div>
      {q.isError ? (
        <div className="pad">
          <Note tone="bad">{errorMessage(q.error)}</Note>
        </div>
      ) : !data ? (
        <div className="pad flex flex-col gap-3" aria-busy="true">
          <Skeleton w="100%" h={18} />
          <Skeleton w="100%" h={18} />
          <Skeleton w="80%" h={18} />
        </div>
      ) : items.length === 0 ? (
        <div className="pad k">{errors ? t(m.noErrors, { window: windowKey === '24h' ? 'h24' : 'd7' }) : t(m.noCalls, { window: windowKey === '24h' ? 'h24' : 'd7' })}</div>
      ) : (
        <>
          <div className="ih-no-narrow">
            <div className="tscroll">
              <table className="tbl" aria-label={t(m.callsTitle)}>
                <thead>
                  <tr>
                    <th scope="col">{t(m.colTime)}</th>
                    <th scope="col">{t(m.colWorkflowAction, { kind })}</th>
                    <th scope="col">{t(m.colTarget)}</th>
                    <th scope="col">{t(m.colResult)}</th>
                    <th scope="col" className="r">
                      {t(m.colDuration)}
                    </th>
                    <th scope="col">
                      <span className="vh">{t(m.request)}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((c, i) => {
                    const isOpen = isOpenRow(c.id)
                    const above = i > 0 ? items[i - 1] : undefined
                    return (
                      <Fragment key={c.id}>
                        <tr className={isOpen ? 'ih-sel' : undefined}>
                          <td className="nw mono">{dayStamp(c.at)}</td>
                          <td>
                            <div className="col gap-px">
                              <span className="font-semibold">{c.unnamed ? t(m.unnamed, { kind }) : c.workflow}</span>
                              <span className="k">{c.item ? itemName(t, c.item) : c.operation}</span>
                            </div>
                          </td>
                          <td>
                            <span className="mono text-[12.5px]">{targetText(t, c)}</span>
                          </td>
                          <td className="nw">
                            <ResultBadge call={c} />
                          </td>
                          <td className="r num nw">{t(m.ms, { ms: c.durationMs })}</td>
                          <td className="r">
                            <Button size="sm" aria-expanded={isOpen} onClick={() => toggle(c.id)}>
                              {isOpen ? t(m.hideRequest) : t(m.request)}
                            </Button>
                          </td>
                        </tr>
                        {isOpen || c.status >= 400 ? (
                          <tr className={`ih-xp${isOpen ? ' ih-sel' : ''}`}>
                            <td />
                            <td colSpan={5} className="pt-0 !whitespace-normal">
                              {isOpen ? <CallBodies connectionId={connectionId} call={c} /> : null}
                              {c.status >= 400 ? <Explain call={c} kind={kind} above={above} /> : null}
                            </td>
                          </tr>
                        ) : null}
                      </Fragment>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>
          <div className="ih-only-narrow ih-list">
            {items.map((c, i) => {
              const isOpen = isOpenRow(c.id)
              return (
                <div key={c.id} className="ih-it gap-1.5">
                  <div className="row justify-between gap-2">
                    <ResultBadge call={c} />
                    <span className="k mono">
                      {dayStamp(c.at)} · {t(m.ms, { ms: c.durationMs })}
                    </span>
                  </div>
                  <span className="font-semibold">{c.unnamed ? t(m.unnamed, { kind }) : c.workflow}</span>
                  <span className="k text-ink-nav">
                    {c.item ? itemName(t, c.item) : c.operation} · <span className="mono">{targetText(t, c)}</span>
                  </span>
                  {isOpen ? <CallBodies connectionId={connectionId} call={c} /> : null}
                  {c.status >= 400 ? <Explain call={c} kind={kind} above={i > 0 ? items[i - 1] : undefined} /> : null}
                  <div>
                    <Button size="sm" aria-expanded={isOpen} onClick={() => toggle(c.id)}>
                      {isOpen ? t(m.hideRequest) : t(m.requestAndResponse)}
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>
        </>
      )}
      <div className="ih-foot">
        <span className="k">{t(m.callsFoot)}</span>
        {data && pageTotal > PAGE ? (
          <div className="row gap-2">
            <span className="k">{t(m.callsShowing, { from: offset + 1, to: Math.min(offset + PAGE, pageTotal), total: pageTotal })}</span>
            <Button size="sm" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE))}>
              {t(m.previous)}
            </Button>
            <Button size="sm" disabled={offset + PAGE >= pageTotal} onClick={() => setOffset(offset + PAGE)}>
              {t(m.next)}
            </Button>
          </div>
        ) : null}
      </div>
    </section>
  )
}

/** "422 missing_variables" in red, "409 unsubscribed" in amber, "200" in green. */
export function ResultBadge({ call }: { call: Pick<Call, 'status' | 'code'> }) {
  const tone = call.status >= 500 || (call.status >= 400 && call.status !== 409) ? 'bad' : call.status === 409 ? 'warn' : 'ok'
  return (
    <Badge tone={tone}>
      {call.status}
      {call.code ? ` ${call.code}` : ''}
    </Badge>
  )
}

/** What a call was about: "payment-failed to finance@initech.example", "customer on dmitri@fabrikam.example". */
export function targetText(t: ReturnType<typeof useT>, c: Call): string {
  const tg = c.target
  const email = c.who === 'contact' ? t(m.aContact) : c.who === 'erased' ? t(m.erasedContact) : (tg.email ?? '')
  switch (tg.kind) {
    case 'upsert_contact':
    case 'add_to_list':
      return tg.list ? t(m.callContact, { email, list: tg.list }) : t(m.callContactOnly, { email })
    case 'add_tag':
      return t(m.callTag, { tag: tg.tag ?? '', email })
    case 'send_transactional':
      return t(m.callTemplate, { template: tg.template ?? '', email })
    case 'record_event':
      return t(m.callEvent, { event: tg.event ?? '', email })
    case 'test':
      return t(m.callTest)
    case 'subscribe':
    case 'unsubscribe':
      return t(m.callHook, { event: tg.event ?? '', endpoint: tg.endpoint ?? '' })
    case 'report':
      return t(m.callReport, { version: tg.version ?? '' })
    case 'forms':
      return t(m.callForms)
    case 'draft':
      return t(m.callDraft, { title: tg.title ?? '' })
  }
  return email || c.path
}

/** The error's own words, what to change in the other tool, and whether a later run got past it. */
function Explain({ call, kind, above }: { call: Call; kind: ConnectKind; above?: Call }) {
  const t = useT()
  const name = t(entryName[kind])
  const missing = call.code === 'missing_variables' && call.missing?.length ? call.missing : null
  const line: ReactNode = missing ? (
    <T msg={m.explainMissing} values={{ ...tags, vars: missing.join(', '), count: missing.length, kind }} />
  ) : (
    call.detail || statusText(call.status)
  )
  const fix = fixFor(t, call, kind, name, missing)
  const res = call.resolution
  let resolved: string | null = null
  if (res) {
    const sameFix = above?.resolution?.fixedAt && res.fixedAt && above.resolution.fixedAt === res.fixedAt && above.resolution.fixedBy === res.fixedBy
    if (sameFix && res.fixedAt) resolved = t(m.resolvedAbove, { time: clock(res.fixedAt) })
    else if (res.fixedBy && res.fixedAt)
      resolved = t(m.resolvedBy, { by: res.fixedBy, kind, time: clock(res.fixedAt), runs: res.laterRuns.length, list: formatList(res.laterRuns.map((r) => clock(r))) })
    else if (res.laterRuns.length) resolved = t(m.resolvedRuns, { runs: res.laterRuns.length, list: formatList(res.laterRuns.map((r) => clock(r))) })
  }
  return (
    <div className="ih-explain mb-1">
      <div className="row flex-nowrap items-start gap-2">
        <InfoIcon />
        <span className="text-ink">{line}</span>
      </div>
      {fix ? (
        <div className="ih-ind">
          <T msg={m.toChange} values={{ ...tags, name, fix }} />
        </div>
      ) : null}
      {resolved ? (
        <div className="ih-ind !pl-[23px]">
          <div className="row flex-nowrap items-start gap-2">
            <Badge tone="ok">{t(m.resolved)}</Badge>
            <span className="text-ink-nav">{resolved}</span>
          </div>
        </div>
      ) : null}
    </div>
  )
}

function fixFor(t: ReturnType<typeof useT>, c: Call, kind: ConnectKind, name: string, missing: string[] | null): string | null {
  if (missing) return t(m.fixMissing, { kind, vars: missing.join(', ') })
  switch (c.code) {
    case 'unsubscribed':
    case 'suppressed':
      return t(m.fixNothing, { kind })
    case 'key_revoked':
    case 'key_expired':
      return t(m.fixReconnect, { name })
    case 'owner_deactivated':
      return t(m.fixTakeOver, { name })
    case 'integration_operation_denied':
      return t(m.fixTurnOn, { name })
  }
  if (c.status === 404) return t(m.fixNotFound, { kind })
  if (c.status === 422 || c.status === 400) return t(m.fixFields, { kind })
  if (c.status === 429) return t(m.fixRetry, { name })
  if (c.status >= 500) return t(m.fixServer, { name })
  return null
}

/** The request and the response, as recorded (secrets masked), in the message log's dark boxes. */
function CallBodies({ connectionId, call }: { connectionId: string; call: Call }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', `${base}/connections/{id}/calls/{callId}`, { params: { path: { workspace: slug, id: connectionId, callId: call.id } } }, { staleTime: 60_000 })
  if (q.isError) return <Note tone="bad">{errorMessage(q.error)}</Note>
  if (!q.data)
    return (
      <div className="ih-pair mb-2.5" aria-busy="true">
        <span className="vh" role="status">
          {t(m.loadingCall)}
        </span>
        <Skeleton w="100%" h={120} r={10} />
        <Skeleton w="100%" h={120} r={10} />
      </div>
    )
  const d = q.data
  const masked = t(m.maskedNote)
  return (
    <div className="ih-pair mb-2.5">
      <div className="ih-cbx">
        <div className="ih-cbh">
          <span>{t(m.requestMasked)}</span>
        </div>
        <pre>
          <span className="c4">{call.method}</span> {call.path}
          {Object.entries(d.headers)
            .sort(([a], [b]) => (a === 'X-ReputeCast-Workflow' ? -1 : b === 'X-ReputeCast-Workflow' ? 1 : a.localeCompare(b)))
            .map(([k, v]) => (
            <Fragment key={k}>
              {'\n'}
              <span className="c1">{k}</span>: {v}
            </Fragment>
          ))}
          {d.request !== undefined && d.request !== null ? (
            <>
              {'\n\n'}
              <Json value={d.request} maskedNote={masked} />
            </>
          ) : null}
        </pre>
      </div>
      <div className="ih-cbx">
        <div className="ih-cbh">
          <span>{t(m.response)}</span>
        </div>
        <pre>
          <span className={call.status >= 400 ? 'c5' : 'c2'}>
            {call.status} {statusText(call.status)}
          </span>{' '}
          · {t(m.ms, { ms: call.durationMs })}
          {d.response !== undefined && d.response !== null ? (
            <>
              {'\n'}
              <Json value={d.response} maskedNote={masked} />
            </>
          ) : null}
        </pre>
      </div>
    </div>
  )
}

const MASK = '••••••'

/** Arrays of plain values stay on one line, as the boards print them ("missing": ["amount_due"]). */
const SCALAR = String.raw`(?:"(?:[^"\\]|\\.)*"|-?\d[\d.eE+-]*|true|false|null)`
const SCALAR_ARRAY = new RegExp(String.raw`\[\n\s*(${SCALAR}(?:,\n\s*${SCALAR})*)\n\s*\]`, 'g')

/** Pretty JSON with the boards' colours: keys blue, strings green, masked values amber with a note. */
function Json({ value, maskedNote }: { value: unknown; maskedNote: string }) {
  const text = (JSON.stringify(value, null, 2) ?? '').replace(SCALAR_ARRAY, (_, items: string) => `[${items.split(/,\n\s*/).join(', ')}]`)
  return (
    <>
      {text.split('\n').map((line, i) => (
        <Fragment key={i}>
          {i > 0 ? '\n' : null}
          <JsonLine line={line} maskedNote={maskedNote} />
        </Fragment>
      ))}
    </>
  )
}

function JsonLine({ line, maskedNote }: { line: string; maskedNote: string }) {
  const kv = /^(\s*)("(?:[^"\\]|\\.)*")(: )(.*)$/.exec(line)
  if (kv) {
    const [, indent, key, sep, rest] = kv
    return (
      <>
        {indent}
        <span className="c1">{key}</span>
        {sep}
        <JsonValue text={rest ?? ''} maskedNote={maskedNote} />
      </>
    )
  }
  const item = /^(\s*)(.*)$/.exec(line)
  return (
    <>
      {item?.[1]}
      <JsonValue text={item?.[2] ?? line} maskedNote={maskedNote} />
    </>
  )
}

/** A value's text: every string literal coloured, a masked one amber with the note at the end of the line. */
function JsonValue({ text, maskedNote }: { text: string; maskedNote: string }) {
  const parts: ReactNode[] = []
  let masked = false
  let last = 0
  for (const match of text.matchAll(/"(?:[^"\\]|\\.)*"/g)) {
    const at = match.index ?? 0
    if (at > last) parts.push(text.slice(last, at))
    const str = match[0]
    if (str === `"${MASK}"`) masked = true
    parts.push(
      <span key={at} className={str === `"${MASK}"` ? 'c3' : 'c2'}>
        {str}
      </span>,
    )
    last = at + str.length
  }
  if (last < text.length) parts.push(text.slice(last))
  return (
    <>
      {parts}
      {masked ? <span className="cm"> {maskedNote}</span> : null}
    </>
  )
}
