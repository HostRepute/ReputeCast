import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type ReactNode, type Ref } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel } from '@/components/ui/Feedback'
import { EmptyRow, Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { ManagedWebhookCaption, ManagedWebhookView } from '@/features/integrations/ManagedWebhook'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, clockSeconds, count, dayMonthDate, logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { DashedEmpty, TableSkeleton } from '../keys/parts'
import { DeliveryDialog } from './DeliveryDialog'
import { m } from './messages'
import { attemptsText, endpointText, eventText, hostOf, responseBadge, statusBadge, successText, type Delivery, type Webhook } from './parts'
import { WebhookDialog, WebhookSecretDialog } from './WebhookDialog'

const API_DOCS = 'https://github.com/hostrepute/reputecast/blob/main/docs/api/README.md'

const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** A5 §5.10 Webhooks (SettingsAPI): outgoing endpoints, signature check and the delivery log. */
export function WebhooksPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('webhooks:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/webhooks', { params: { path: { workspace: slug } } }, { enabled: can })
  const [editing, setEditing] = useState<Webhook | 'new'>()
  const [secret, setSecret] = useState<{ secret: string; webhook: Webhook; action: 'created' | 'rotated' }>()
  const [filter, setFilter] = useState('')
  const logRef = useRef<HTMLElement>(null)
  return (
    <Page
      title={t(m.title)}
      tabs={settingsTabs(slug)}
      actions={
        // This board's action row doesn't centre its items, so the buttons
        // stretch to the 38px bell beside them.
        <>
          <ButtonAnchor href={API_DOCS} className="self-stretch">
            {t(m.apiReference)}
          </ButtonAnchor>
          <ButtonLink to="/w/$workspace/settings/keys" params={{ workspace: slug }} className="self-stretch">
            {t(m.keysLink)}
          </ButtonLink>
        </>
      }
    >
      {can ? <MovedCard /> : null}
      {!can ? (
        <ForbiddenState permission="webhooks:manage" title={t(m.forbidden)} />
      ) : q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <>
          <div className="flex flex-wrap items-start gap-5">
            {q.data && q.data.items.length === 0 ? (
              <div className="min-w-0 flex-[1_1_100%]">
                <DashedEmpty
                  title={t(m.emptyTitle)}
                  action={
                    <Button onClick={() => setEditing('new')} disabledReason={q.data ? undefined : t(common.loading)}>
                      {t(m.addEndpoint)}
                    </Button>
                  }
                >
                  {t(m.emptyBody)}
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
  const t = useT()
  const { slug } = useWorkspace()
  const canKeys = useCan('apikeys:manage')
  const canAgents = useCan('mcp:manage')
  const path = { params: { path: { workspace: slug } } }
  const keys = $api.useQuery('get', '/api/v1/workspaces/{workspace}/api-keys', path, { enabled: canKeys })
  const agents = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', path, { enabled: canAgents })
  const counts = [
    keys.data ? t(m.movedKeys, { count: keys.data.items.length }) : '',
    agents.data ? t(m.movedAgents, { count: agents.data.items.length }) : '',
  ].filter(Boolean)
  // A card header and a padded row, the sentence and the button at its ends (SettingsAPI).
  return (
    <section className="card" aria-labelledby="moved-h">
      <CardSection id="moved-h" title={t(m.movedTitle)} caption={t(m.movedCaption)} />
      <div className="pad row justify-between">
        <span className="k">{counts.length ? t(m.movedBodyCounts, { counts: counts.join(' · ') }) : t(m.movedBody)}</span>
        <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
          {t(m.openKeys)}
        </ButtonLink>
      </div>
    </section>
  )
}

function EndpointsCard({ items, onAdd, onEdit }: { items?: Webhook[]; onAdd: () => void; onEdit: (w: Webhook) => void }) {
  const t = useT()
  return (
    <section className="card min-w-0 flex-[1_1_100%]" aria-labelledby="hooks-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="hooks-h">
            {items ? t(m.endpointsTitle, { count: count(items.length) }) : t(m.endpointsTitleLoading)}
          </h2>
          <span className="k">{t(m.endpointsCaption)}</span>
        </div>
        <Button size="sm" variant="primary" disabledReason={items ? undefined : t(common.loading)} onClick={onAdd}>
          {t(m.addEndpoint)}
        </Button>
      </div>
      {!items ? (
        <TableSkeleton label={t(m.endpointsLoading)} />
      ) : (
        <Table phoneWrap density="dense" caption={t(m.endpointsTable)} className="[&_td:nth-child(3)]:min-w-[220px] [&_td:nth-child(3)]:whitespace-normal [&_td]:whitespace-nowrap">
          <thead>
            <tr>
              <Th>{t(m.colEndpoint)}</Th>
              <Th>{t(m.colEvents)}</Th>
              <Th>{t(m.colSuccess)}</Th>
              <Th>{t(m.colStatus)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
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
                        {w.secretRotatedAt ? ` · ${t(m.secretRotated, { date: dayMonthDate(w.secretRotatedAt) })}` : ''}
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
                      <Button size="sm" onClick={() => onEdit(w)} aria-label={t(m.editNamed, { host: hostOf(w.url) })}>
                        {t(common.edit)}
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
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-[1_1_380px] flex-col gap-2.5 px-4.5 py-4" aria-labelledby="verify-h">
      <h2 className="h2" id="verify-h">
        {t(m.verifyTitle)}
      </h2>
      <span className="k">
        <T msg={m.verifyBody} values={{ header, mono }} />
      </span>
      <ScrollPre className="pre" label={t(m.verifyCode)}>
        {`const sig = createHmac("sha256", secret)
  .update(\`\${timestamp}.\${rawBody}\`)
  .digest("hex");
if (!timingSafeEqual(sig, header)) throw 401;`}
      </ScrollPre>
      <span className="k">
        <T msg={m.payloadNote} values={{ sample: `{ "type": "email.bounced", "subtype": "hard", "provider": "ses", "provider_code": "550 5.1.1", … }`, mono }} />
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
  const t = useT()
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
      toast({ title: d.state === 'dead' ? tt(m.replayedToast) : tt(m.retryingToast), detail: tt(m.retryDetail, { event: eventText(d), endpoint: endpointText(d) }) })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.retryFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  const retryAll = async () => {
    setBusy('all')
    try {
      const r = (await http.POST('/api/v1/workspaces/{workspace}/webhooks/retry-failed', { params: { path: { workspace: slug } }, body: { webhookId: filter || undefined } })).data!
      toast({ title: r.queued ? tt(m.requeued, { count: r.queued }) : tt(m.nothingToRetry), detail: r.queued ? tt(m.requeuedDetail) : undefined })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.retryAllFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  // "7,176 events are queued for the warehouse endpoint"
  const queuedText = Object.entries((q.data?.queued ?? {}) as Record<string, number>)
    .filter(([, n]) => n > 0)
    .map(([id, n]) => {
      const w = webhooks.find((x) => x.id === id)
      return w ? t(m.queuedFor, { count: n, name: hostOf(w.url).split('.')[0] ?? '' }) : t(m.queuedForFailing, { count: n })
    })
    .join('; ')
  return (
    <section className="card" aria-labelledby="log-h" ref={ref}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="log-h">
          {t(m.logTitle)}
        </h2>
        <div className="flex gap-2">
          <select
            className="h-[30px] rounded-[8px] border border-line-strong bg-white px-2 text-[12px]"
            aria-label={t(m.filterEndpoint)}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          >
            <option value="">{t(m.allEndpoints)}</option>
            {webhooks.map((w) => (
              <option key={w.id} value={w.id}>
                {hostOf(w.url)}
              </option>
            ))}
          </select>
          <Button size="sm" loading={busy === 'all'} onClick={() => void retryAll()}>
            {t(m.retryAll)}
          </Button>
        </div>
      </div>
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.logLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton label={t(m.logLoading)} />
      ) : (
        <Table phoneWrap density="dense" caption={t(m.logTable)} className="[&_td]:whitespace-nowrap">
          <thead>
            <tr>
              <Th>{t(m.colTime)}</Th>
              <Th>{t(m.colEvent)}</Th>
              <Th>{t(m.colLogEndpoint)}</Th>
              <Th>{t(m.colResponse)}</Th>
              <Th>{t(m.colAttempts)}</Th>
              <Th>{t(m.colNextRetry)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.length === 0 ? (
              <EmptyRow colSpan={7}>{filter ? t(m.noDeliveriesHere) : t(m.noDeliveries)}</EmptyRow>
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
                    <Td>{resp ? <Badge tone={resp.tone}>{resp.label}</Badge> : <span className="k">{d.state === 'held' ? t(m.held) : t(m.queued)}</span>}</Td>
                    <Td>{attemptsText(d)}</Td>
                    <Td className={d.state === 'delivered' ? 'k' : undefined}>
                      {d.state === 'dead' ? (
                        <Badge tone="bad">{t(m.gaveUp)}</Badge>
                      ) : d.state === 'retrying' && d.nextAttemptAt ? (
                        d.backoffMinutes ? (
                          t(m.nextRetryBackoff, { time: clock(d.nextAttemptAt), minutes: count(d.backoffMinutes) })
                        ) : (
                          clock(d.nextAttemptAt)
                        )
                      ) : d.state === 'held' ? (
                        t(m.heldPaused)
                      ) : d.state === 'pending' || d.state === 'delivering' ? (
                        t(m.now)
                      ) : (
                        '–'
                      )}
                    </Td>
                    <Td>
                      {d.state === 'retrying' ? (
                        <Button size="sm" loading={busy === d.id} onClick={() => void retry(d)}>
                          {t(m.retryNow)}
                        </Button>
                      ) : d.state === 'dead' ? (
                        <Button size="sm" loading={busy === d.id} onClick={() => void retry(d)}>
                          {t(m.replay)}
                        </Button>
                      ) : (
                        <Button size="sm" onClick={() => setOpen(d.id)} aria-label={t(m.payloadOf, { event: eventText(d), time: clockSeconds(d.createdAt) })}>
                          {t(m.payload)}
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
        <span>{queuedText ? t(m.retryPolicyQueued, { max: maxAttempts, queued: queuedText }) : t(m.retryPolicy, { max: maxAttempts })}</span>
        {q.data && q.data.items.length >= limit && limit < 200 ? (
          <Button size="sm" variant="ghost" onClick={() => setLimit((n) => n + 20)}>
            {t(common.showMore)}
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
