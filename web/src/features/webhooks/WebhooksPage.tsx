import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type Ref } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel } from '@/components/ui/Feedback'
import { EmptyRow, Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { ManagedWebhookCaption, ManagedWebhookView } from '@/features/integrations/ManagedWebhook'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { clock, clockSeconds, count, dayMonthDate, logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { DashedEmpty, TableSkeleton } from '../keys/parts'
import { DeliveryDialog } from './DeliveryDialog'
import { attemptsText, endpointText, eventText, hostOf, responseBadge, statusBadge, successText, type Delivery, type Webhook } from './parts'
import { WebhookDialog, WebhookSecretDialog } from './WebhookDialog'

const API_DOCS = 'https://github.com/hostrepute/reputecast/blob/main/docs/api/README.md'

/** A5 §5.10 Webhooks (SettingsAPI): outgoing endpoints, signature check and the delivery log. */
export function WebhooksPage() {
  const { slug } = useWorkspace()
  const can = useCan('webhooks:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/webhooks', { params: { path: { workspace: slug } } }, { enabled: can })
  const [editing, setEditing] = useState<Webhook | 'new'>()
  const [secret, setSecret] = useState<{ secret: string; webhook: Webhook; action: 'created' | 'rotated' }>()
  const [filter, setFilter] = useState('')
  const logRef = useRef<HTMLElement>(null)
  return (
    <Page
      title="Settings"
      tabs={settingsTabs(slug)}
      actions={
        // This board's action row doesn't centre its items, so the buttons
        // stretch to the 38px bell beside them.
        <>
          <ButtonAnchor href={API_DOCS} className="self-stretch">
            API reference
          </ButtonAnchor>
          <ButtonLink to="/w/$workspace/settings/keys" params={{ workspace: slug }} className="self-stretch">
            API keys &amp; MCP
          </ButtonLink>
        </>
      }
    >
      {can ? <MovedCard /> : null}
      {!can ? (
        <ForbiddenState permission="webhooks:manage" title="Your role can't manage webhooks" />
      ) : q.isError ? (
        <ErrorPanel title="Couldn't load the webhooks" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <>
          <div className="flex flex-wrap items-start gap-5">
            {q.data && q.data.items.length === 0 ? (
              <div className="min-w-0 flex-[1.2_1_520px]">
                <DashedEmpty
                  title="No webhooks"
                  action={
                    <Button onClick={() => setEditing('new')} disabledReason={q.data ? undefined : 'Loading…'}>
                      Add endpoint
                    </Button>
                  }
                >
                  Get normalized delivery, bounce and complaint events pushed to your own endpoint.
                </DashedEmpty>
              </div>
            ) : (
              <EndpointsCard items={q.data?.items} onAdd={() => setEditing('new')} onEdit={setEditing} />
            )}
            <VerifyCard header={q.data?.signatureHeader ?? 'X-ReputeCast-Signature'} />
          </div>
          <DeliveryLog
            ref={logRef}
            webhooks={q.data?.items ?? []}
            filter={filter}
            setFilter={setFilter}
            maxAttempts={q.data?.maxAttempts ?? 8}
          />
        </>
      )}
      {editing && q.data ? (
        <WebhookDialog
          webhook={editing === 'new' ? undefined : editing}
          events={q.data.events}
          maxAttempts={q.data.maxAttempts}
          onClose={() => setEditing(undefined)}
          onSecret={(s, w, action) => {
            setEditing(undefined)
            setSecret({ secret: s, webhook: w, action })
          }}
          onShowLog={(id) => {
            setEditing(undefined)
            setFilter(id)
            logRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
          }}
        />
      ) : null}
      {secret ? (
        <WebhookSecretDialog
          secret={secret.secret}
          webhook={secret.webhook}
          action={secret.action}
          header={q.data?.signatureHeader ?? 'X-ReputeCast-Signature'}
          onClose={() => setSecret(undefined)}
        />
      ) : null}
    </Page>
  )
}

/** "API keys and MCP agents moved": the old API screen's keys now live under Keys & MCP. */
function MovedCard() {
  const { slug } = useWorkspace()
  const canKeys = useCan('apikeys:manage')
  const canAgents = useCan('mcp:manage')
  const path = { params: { path: { workspace: slug } } }
  const keys = $api.useQuery('get', '/api/v1/workspaces/{workspace}/api-keys', path, { enabled: canKeys })
  const agents = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', path, { enabled: canAgents })
  const counts = [
    keys.data ? `${keys.data.items.length} API ${keys.data.items.length === 1 ? 'key' : 'keys'}` : '',
    agents.data ? `${agents.data.items.length} MCP ${agents.data.items.length === 1 ? 'agent' : 'agents'}` : '',
  ].filter(Boolean)
  // As the board renders it: its CSS has no rules for the card's section
  // and padding classes, so the block is compact and the button follows the
  // sentence inline (on the same line when it fits, under it otherwise).
  return (
    <section className="card" aria-labelledby="moved-h">
      <div className="flex min-w-0 flex-col gap-0.5">
        <h2 className="h2" id="moved-h">
          API keys and MCP agents moved
        </h2>
        <span className="k">Now under the Keys &amp; MCP tab</span>
      </div>
      <div>
        <span className="k">
          {counts.length ? `${counts.join(' · ')} · ` : ''}permissions use the same keys as roles and never exceed their creator&apos;s. Secrets are shown once.
        </span>
        <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
          Open API keys &amp; MCP
        </ButtonLink>
      </div>
    </section>
  )
}

function EndpointsCard({ items, onAdd, onEdit }: { items?: Webhook[]; onAdd: () => void; onEdit: (w: Webhook) => void }) {
  return (
    <section className="card min-w-0 flex-[1.2_1_520px]" aria-labelledby="hooks-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="hooks-h">
            Outgoing webhooks{items ? ` · ${items.length}` : ''}
          </h2>
          <span className="k">ReputeCast calls your endpoints with normalized events, signed with HMAC-SHA256.</span>
        </div>
        <Button size="sm" variant="primary" disabledReason={items ? undefined : 'Loading…'} onClick={onAdd}>
          Add endpoint
        </Button>
      </div>
      {!items ? (
        <TableSkeleton label="Loading the webhooks" />
      ) : (
        <Table phoneWrap density="dense" caption="Outgoing webhooks" className="[&_td:nth-child(3)]:min-w-[220px] [&_td:nth-child(3)]:whitespace-normal [&_td]:whitespace-nowrap">
          <thead>
            <tr>
              <Th>Endpoint</Th>
              <Th>Events</Th>
              <Th>Success · 24 h</Th>
              <Th>Status</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {items.map((w) => {
              const st = statusBadge(w)
              const failing = w.status === 'failing'
              return (
                <tr key={w.id} className={failing ? 'bg-bad-tint' : undefined}>
                  <Td>
                    <span className="mono">{w.managedBy?.endpoint ?? w.url}</span>
                    {w.managedBy ? (
                      <ManagedWebhookCaption managedBy={w.managedBy} id={`managed-${w.id}`} />
                    ) : (
                      <div className="k">
                        {w.name}
                        {w.description ? ` · ${w.description}` : ''}
                        {w.secretRotatedAt ? ` · secret rotated ${dayMonthDate(w.secretRotatedAt)}` : ''}
                      </div>
                    )}
                  </Td>
                  <Td>
                    <span className="inline-flex flex-wrap gap-x-1.5 gap-y-1 align-middle">
                      {w.events.map((e) => (
                        <Badge key={e} tone="neutral">
                          {e}
                        </Badge>
                      ))}
                    </span>
                  </Td>
                  <Td num className={failing ? 'text-bad-fg' : undefined}>
                    {successText(w)}
                  </Td>
                  <Td>
                    <Badge tone={st.tone}>{st.label}</Badge>
                  </Td>
                  <Td>
                    {w.managedBy ? (
                      <ManagedWebhookView managedBy={w.managedBy} describedBy={`managed-${w.id}`} />
                    ) : (
                      <Button size="sm" onClick={() => onEdit(w)} aria-label={`Edit ${hostOf(w.url)}`}>
                        Edit
                      </Button>
                    )}
                  </Td>
                </tr>
              )
            })}
          </tbody>
        </Table>
      )}
    </section>
  )
}

function VerifyCard({ header }: { header: string }) {
  return (
    <section className="card flex min-w-0 flex-[1_1_380px] flex-col gap-2.5 px-4.5 py-4" aria-labelledby="verify-h">
      <h2 className="h2" id="verify-h">
        Verify a signature
      </h2>
      <span className="k">
        Every request carries <span className="mono">{header}</span> and a timestamp. Reject anything older than 5 minutes.
      </span>
      <ScrollPre className="pre" label="Signature check">
        {`const sig = createHmac("sha256", secret)
  .update(\`\${timestamp}.\${rawBody}\`)
  .digest("hex");
if (!timingSafeEqual(sig, header)) throw 401;`}
      </ScrollPre>
      <span className="k">
        Payloads carry the normalized event plus the provider&apos;s original code:{' '}
        <span className="mono">{`{ "type": "email.bounced", "subtype": "hard", "provider": "ses", "provider_code": "550 5.1.1", … }`}</span>
      </span>
    </section>
  )
}

function DeliveryLog({
  ref,
  webhooks,
  filter,
  setFilter,
  maxAttempts,
}: {
  ref: Ref<HTMLElement>
  webhooks: Webhook[]
  filter: string
  setFilter: (v: string) => void
  maxAttempts: number
}) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const now = useNow()
  const [open, setOpen] = useState<string>()
  const [busy, setBusy] = useState<string>()
  // The five latest attempts, as on the board; "Show more" pages further back.
  const [limit, setLimit] = useState(5)
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/webhooks/deliveries',
    { params: { path: { workspace: slug }, query: { webhookId: filter || undefined, limit } } },
    { placeholderData: keepPreviousData, refetchInterval: 30_000 },
  )
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/webhooks/deliveries'] })
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/webhooks'] })
  }
  const retry = async (d: Delivery) => {
    setBusy(d.id)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/webhooks/deliveries/{deliveryId}/retry', { params: { path: { workspace: slug, deliveryId: d.id } } })
      toast({ title: d.state === 'dead' ? 'Delivery replayed' : 'Retrying now', detail: `${eventText(d)} to ${endpointText(d)}` })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't retry the delivery", detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  const retryAll = async () => {
    setBusy('all')
    try {
      const r = (await http.POST('/api/v1/workspaces/{workspace}/webhooks/retry-failed', { params: { path: { workspace: slug } }, body: { webhookId: filter || undefined } })).data!
      toast({ title: r.queued ? `${count(r.queued)} failed ${r.queued === 1 ? 'delivery' : 'deliveries'} queued again` : 'Nothing to retry', detail: r.queued ? 'they go out in order' : undefined })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't retry the failed deliveries", detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  // "7,176 events are queued for the warehouse endpoint"
  const queuedText = Object.entries((q.data?.queued ?? {}) as Record<string, number>)
    .filter(([, n]) => n > 0)
    .map(([id, n]) => {
      const w = webhooks.find((x) => x.id === id)
      return `${count(n)} ${n === 1 ? 'event is' : 'events are'} queued for the ${w ? hostOf(w.url).split('.')[0] : 'failing'} endpoint`
    })
    .join('; ')
  return (
    <section className="card" aria-labelledby="log-h" ref={ref}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="log-h">
          Delivery log
        </h2>
        <div className="flex gap-2">
          <select className="h-[30px] rounded-[8px] border border-line-strong bg-white px-2 text-[12px]" aria-label="Endpoint" value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="">All endpoints</option>
            {webhooks.map((w) => (
              <option key={w.id} value={w.id}>
                {hostOf(w.url)}
              </option>
            ))}
          </select>
          <Button size="sm" loading={busy === 'all'} onClick={() => void retryAll()}>
            Retry all failed
          </Button>
        </div>
      </div>
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the delivery log" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton label="Loading the delivery log" />
      ) : (
        <Table phoneWrap density="dense" caption="Delivery log" className="[&_td]:whitespace-nowrap">
          <thead>
            <tr>
              <Th>Time</Th>
              <Th>Event</Th>
              <Th>Endpoint</Th>
              <Th>Response</Th>
              <Th>Attempts</Th>
              <Th>Next retry</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.length === 0 ? (
              <EmptyRow colSpan={7}>{filter ? 'No deliveries to this endpoint yet.' : 'No deliveries yet. Add an endpoint and events show up here.'}</EmptyRow>
            ) : (
              q.data.items.map((d) => {
                const resp = responseBadge(d)
                const failed = d.state === 'retrying' || d.state === 'dead'
                return (
                  <tr key={d.id} className={failed ? 'bg-bad-tint' : undefined}>
                    <Td className="mono">{sameDayClock(d.lastAttemptAt ?? d.createdAt, now)}</Td>
                    <Td>
                      <Badge tone="neutral">{eventText(d)}</Badge>
                    </Td>
                    <Td className="mono">{endpointText(d)}</Td>
                    <Td>{resp ? <Badge tone={resp.tone}>{resp.label}</Badge> : <span className="k">{d.state === 'held' ? 'Held' : 'Queued'}</span>}</Td>
                    <Td>{attemptsText(d)}</Td>
                    <Td className={d.state === 'delivered' ? 'k' : undefined}>
                      {d.state === 'dead' ? (
                        <Badge tone="bad">Gave up · in dead-letter queue</Badge>
                      ) : d.state === 'retrying' && d.nextAttemptAt ? (
                        `${clock(d.nextAttemptAt)}${d.backoffMinutes ? ` · backoff ${d.backoffMinutes} min` : ''}`
                      ) : d.state === 'held' ? (
                        'Held · endpoint paused'
                      ) : d.state === 'pending' || d.state === 'delivering' ? (
                        'Now'
                      ) : (
                        '–'
                      )}
                    </Td>
                    <Td>
                      {d.state === 'retrying' ? (
                        <Button size="sm" loading={busy === d.id} onClick={() => void retry(d)}>
                          Retry now
                        </Button>
                      ) : d.state === 'dead' ? (
                        <Button size="sm" loading={busy === d.id} onClick={() => void retry(d)}>
                          Replay
                        </Button>
                      ) : (
                        <Button size="sm" onClick={() => setOpen(d.id)} aria-label={`Payload of ${eventText(d)} at ${clockSeconds(d.createdAt)}`}>
                          Payload
                        </Button>
                      )}
                    </Td>
                  </tr>
                )
              })
            )}
          </tbody>
        </Table>
      )}
      <div className="k flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4.5 py-2.5">
        <span>
          Failed deliveries retry {maxAttempts} times over 6 hours, then wait in the dead-letter queue for 7 days.{queuedText ? ` ${queuedText}.` : ''}
        </span>
        {q.data && q.data.items.length >= limit && limit < 200 ? (
          <Button size="sm" variant="ghost" onClick={() => setLimit((n) => n + 20)}>
            Show more
          </Button>
        ) : null}
      </div>
      {open ? <DeliveryDialog id={open} onClose={() => setOpen(undefined)} /> : null}
    </section>
  )
}

/** "09:45:12" today, otherwise "Mon 6 Oct". */
function sameDayClock(at: string, now: Date) {
  const l = logTime(at, now)
  return l.includes(':') ? clockSeconds(at) : l
}
