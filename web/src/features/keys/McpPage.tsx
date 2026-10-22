import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Fragment, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { OptionCard } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { EmptyRow, Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { t as tt, useI18n, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { CreateKeyDialog } from './CreateKeyDialog'
import { KeySecretDialog } from './KeySecretDialog'
import { m } from './mcp.messages'
import { DashedEmpty, TableSkeleton } from './parts'
import { McpIntegrationsNote } from '@/features/integrations/ManagedWebhook'
import { McpAssistantNote } from '@/features/ai/parts'

type Overview = Schemas['MCPOverview']

const levelTone: Record<Schemas['MCPCapabilityCell']['level'], Tone> = { allow: 'ok', approve: 'info', partial: 'warn', deny: 'neutral' }

/** A mono chip for a tool name (SettingsMCP `.tool`). */
function Tool({ children }: { children: string }) {
  return <span className="inline-flex h-6 items-center rounded-[6px] border border-line bg-ground px-2 font-mono text-[11.5px]">{children}</span>
}

/** A5 §5.10 MCP server (SettingsMCP): endpoint, the agents' capabilities, recent calls, tools and limits. */
export function McpPage() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const can = me.permissions.includes('mcp:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp', { params: { path: { workspace: slug } } }, { enabled: can })
  const agents = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', { params: { path: { workspace: slug } } }, { enabled: can })
  const [adding, setAdding] = useState(false)
  const [secret, setSecret] = useState<Schemas['APIKeyCreated']>()
  const tabs = settingsTabs(slug).map((tab) => (tab.to === '/w/$workspace/settings/keys' ? { ...tab, current: true } : tab))
  return (
    <Page
      title={t(m.title)}
      tabs={tabs}
      actions={
        q.data ? (
          <>
            {q.data.enabled ? (
              <Badge tone="ok" dot>
                {t(m.serverOn)}
              </Badge>
            ) : (
              <Badge tone="neutral" dot>
                {t(m.serverOff)}
              </Badge>
            )}
            <Button variant="primary" disabledReason={agents.data ? undefined : t(common.loading)} onClick={() => setAdding(true)}>
              {t(m.addAgent)}
            </Button>
          </>
        ) : null
      }
    >
      {!can ? (
        <ForbiddenState permission="mcp:manage" title={t(m.forbidden)} />
      ) : q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !q.data ? (
        <McpSkeleton />
      ) : (
        <>
          <ConnectCard o={q.data} />
          {q.data.agents.length === 0 ? (
            <DashedEmpty
              title={t(m.noAgentsTitle)}
              action={
                <Button onClick={() => setAdding(true)} disabledReason={agents.data ? undefined : t(common.loading)}>
                  {t(m.addAgent)}
                </Button>
              }
            >
              {t(m.noAgentsBody)}
            </DashedEmpty>
          ) : (
            <AgentsMatrix o={q.data} />
          )}
          <div className="flex flex-wrap items-start gap-5">
            <ActionsCard />
            <ToolsCard o={q.data} />
          </div>
        </>
      )}
      {adding && agents.data ? (
        <CreateKeyDialog
          kind="mcp_agent"
          list={agents.data}
          onClose={() => setAdding(false)}
          onCreated={(c) => {
            setAdding(false)
            setSecret(c)
          }}
        />
      ) : null}
      {secret ? <KeySecretDialog created={secret} action="created" onClose={() => setSecret(undefined)} /> : null}
    </Page>
  )
}

function ConnectCard({ o }: { o: Overview }) {
  const t = useT()
  // The transports' labels pass 32 characters outside English, where they
  // are radio cards instead of a select, which can't wrap (Text expansion, rule 1).
  const cards = useI18n().locale !== 'en'
  // Laid out as on the board: the headers object stays on one line.
  const snippet = [
    '{',
    '  "mcpServers": {',
    '    "reputecast": {',
    `      "url": ${JSON.stringify(o.endpoint)},`,
    '      "headers": { "Authorization": "Bearer rc_agent_…" }',
    '    }',
    '  }',
    '}',
  ].join('\n')
  return (
    <section className="card flex flex-wrap items-start gap-5 px-5 py-4.5" aria-labelledby="mcp-h">
      <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-2.5">
        <h2 className="h2" id="mcp-h">
          {t(m.connectTitle)}
        </h2>
        <span className="text-[13px] text-ink-2">{t(m.connectBody)}</span>
        <label className="lbl">
          {t(m.endpoint)}
          <input className="in mono bg-white" value={o.endpoint} readOnly onFocus={(e) => e.currentTarget.select()} />
        </label>
        {cards ? (
          <div className="lbl">
            <span id="mcp-transport-l">{t(m.transport)}</span>
            <div role="radiogroup" aria-labelledby="mcp-transport-l" aria-describedby="mcp-transport-hint" className="flex flex-col gap-1.5">
              <OptionCard name="mcp-transport" value="http" checked onChange={() => {}} title={t(m.transportHttp)} />
              <OptionCard name="mcp-transport" value="stdio" checked={false} disabled onChange={() => {}} title={t(m.transportStdio)} />
            </div>
            <span className="vh" id="mcp-transport-hint">
              {t(m.transportHint)}
            </span>
          </div>
        ) : (
          <label className="lbl">
            {t(m.transport)}
            <select className="in" defaultValue="http" aria-describedby="mcp-transport-hint">
              <option value="http">{t(m.transportHttp)}</option>
              <option value="stdio" disabled>
                {t(m.transportStdio)}
              </option>
            </select>
            <span className="vh" id="mcp-transport-hint">
              {t(m.transportHint)}
            </span>
          </label>
        )}
      </div>
      <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-2">
        <span className="k">{t(m.connectFrom)}</span>
        <ScrollPre className="pre" label={t(m.clientConfig)}>
          {snippet}
        </ScrollPre>
      </div>
    </section>
  )
}

function AgentsMatrix({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const caps = o.capabilities
  return (
    <section className="card" aria-labelledby="mcp-agents-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="mcp-agents-h">
            {t(m.agentsTitle, { count: count(o.agents.length) })}
          </h2>
          <span className="k">{t(m.agentsCaption)}</span>
        </div>
      </div>
      {/* The board's 240 px for the fourth column is a preferred width here, not a
          minimum: the column gives way when a long tool name needs the room. */}
      <Table phoneWrap fit density="dense" caption={t(m.agentsTable)} className="[&_td:first-child]:min-w-[200px] [&_td:nth-child(4)]:w-[240px] [&_td]:whitespace-normal">
        <thead>
          <tr>
            <Th>{t(m.colAgent)}</Th>
            {caps.map((c) => (
              <Th key={c} align="center">
                {c}
              </Th>
            ))}
            <Th>{t(m.colLastCall)}</Th>
            <Th>
              <span className="vh">{t(m.colActions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {o.agents.map((a) => (
            <tr key={a.id}>
              <Td>
                <div className="font-semibold">{a.name}</div>
                <div className="k">{t(m.agentLine, { masked: a.masked, name: a.createdBy, permissions: a.permissions.join(', ') })}</div>
              </Td>
              {caps.map((c) => {
                const cell = a.cells.find((x) => x.capability === c)
                return (
                  <Td key={c} align="center">
                    {cell ? <Badge tone={levelTone[cell.level]}>{cell.label}</Badge> : <span className="k">–</span>}
                  </Td>
                )
              })}
              <Td>
                <LastCall agentId={a.id} />
              </Td>
              <Td>
                <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={{ workspace: slug }} aria-label={t(m.editAgent, { name: a.name })}>
                  {t(common.edit)}
                </ButtonLink>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
      <McpAssistantNote />
    </section>
  )
}

/** "09:36 · get_campaign_report" from the agents list. */
function LastCall({ agentId }: { agentId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', { params: { path: { workspace: slug } } })
  const k = q.data?.items.find((x) => x.id === agentId)
  if (!k) return <Skeleton w={110} />
  if (!k.lastUsedAt) return <span className="k">{t(m.never)}</span>
  return (
    <>
      {logTime(k.lastUsedAt, now)}
      {k.lastCall ? ' · ' : ''}
      {/* A tool name may break after its underscores, so the column fits the card. */}
      {k.lastCall?.split('_').map((part, i, all) => (
        <Fragment key={i}>
          {part}
          {i < all.length - 1 ? (
            <>
              _<wbr />
            </>
          ) : null}
        </Fragment>
      ))}
    </>
  )
}

function ActionsCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents/actions', { params: { path: { workspace: slug }, query: { limit: 5 } } })
  return (
    <section className="card min-w-0 flex-[1_1_100%]" aria-labelledby="mcp-acts-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="mcp-acts-h">
          {t(m.actionsTitle)}
        </h2>
        <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ actorType: 'mcp_agent' }} className="k">
          {t(m.fullAuditLog)}
        </Link>
      </div>
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.actionsLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton rows={3} label={t(m.actionsLoading)} />
      ) : (
        <Table phoneWrap fit density="dense" caption={t(m.actionsTable)} className="[&_td:first-child]:min-w-[200px] [&_td:nth-child(2)]:whitespace-nowrap [&_td:nth-child(4)]:min-w-[240px]">
          <thead>
            <tr>
              <Th>{t(m.colTime)}</Th>
              <Th>{t(m.colAgent)}</Th>
              <Th>{t(m.colTool)}</Th>
              <Th>{t(m.colResult)}</Th>
              <Th>{t(m.colApproval)}</Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.length === 0 ? (
              <EmptyRow colSpan={5}>{t(m.noCalls)}</EmptyRow>
            ) : (
              q.data.items.map((a) => (
                <tr key={a.id} className={a.result === 'allowed' ? undefined : 'bg-bad-tint'}>
                  <Td className="mono">{logTime(a.occurredAt, now)}</Td>
                  <Td>{a.agent}</Td>
                  <Td>
                    <Tool>{a.tool}</Tool>
                  </Td>
                  <Td>{a.summary || a.target}</Td>
                  <Td>
                    {a.result === 'allowed' ? (
                      <Badge tone="neutral">{a.note ? t(m.notNeededNote, { note: a.note }) : t(m.notNeeded)}</Badge>
                    ) : (
                      <span className="inline-flex items-center gap-2">
                        <Badge tone="bad">{a.result === 'denied' ? t(m.denied) : t(m.error)}</Badge>
                        {a.note?.includes('proposal') ? (
                          <ButtonLink size="sm" to="/w/$workspace/approvals" params={{ workspace: slug }}>
                            {t(m.review)}
                          </ButtonLink>
                        ) : null}
                      </span>
                    )}
                  </Td>
                </tr>
              ))
            )}
          </tbody>
        </Table>
      )}
    </section>
  )
}

function ToolsCard({ o }: { o: Overview }) {
  const t = useT()
  const [editing, setEditing] = useState(false)
  const limits = [
    t(m.limitRate, { count: o.rateLimitPerMinute, rate: count(o.rateLimitPerMinute) }),
    t(m.limitApprovals, { hours: o.approvalExpiryHours, n: count(o.approvalExpiryHours) }),
    t(m.limitSecrets),
  ]
  return (
    <section className="card flex min-w-0 flex-[1_1_360px] flex-col gap-2.5 px-4.5 py-4" aria-labelledby="mcp-tools-h">
      <h2 className="h2" id="mcp-tools-h">
        {t(m.toolsTitle, { count: count(o.tools.length) })}
      </h2>
      <ul className="m-0 flex list-none flex-wrap gap-1.5 p-0" aria-label={t(m.toolsList)}>
        {o.tools.map((tool) => (
          <li key={tool.name} title={t(m.toolTitle, { description: tool.description, permission: tool.permission })}>
            <Tool>{tool.name}</Tool>
          </li>
        ))}
      </ul>
      <McpIntegrationsNote />
      <span className="k">{t(m.resourcesPrompts, { resources: o.resources.join(', '), prompts: o.prompts.map((p) => `"${p}"`).join(', ') })}</span>
      <ul className="m-0 flex list-none flex-col gap-2.5 p-0 text-[13px]" aria-label={t(m.limitsList)}>
        {limits.map((line) => (
          <li key={line} className="flex items-start gap-2.5">
            {/* Placed like the board's checked checkbox (16px, 3px 3px 3px 4px margins). */}
            <span className="m-[3px_3px_3px_4px] inline-flex size-4 flex-none items-center justify-center rounded-[3px] bg-accent text-[11px] font-bold text-white" aria-hidden="true">
              ✓
            </span>
            <span>{line}</span>
          </li>
        ))}
      </ul>
      {o.canManage ? (
        <span>
          <Button size="sm" onClick={() => setEditing(true)}>
            {t(m.changeLimits)}
          </Button>
        </span>
      ) : null}
      {editing ? <LimitsDialog o={o} onClose={() => setEditing(false)} /> : null}
    </section>
  )
}

function LimitsDialog({ o, onClose }: { o: Overview; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [enabled, setEnabled] = useState(o.enabled)
  const [rate, setRate] = useState(String(o.rateLimitPerMinute))
  const [hours, setHours] = useState(String(o.approvalExpiryHours))
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/mcp')
  return (
    <Dialog
      open
      onOpenChange={(o2) => !o2 && onClose()}
      eyebrow={<span className="k">{t(m.limitsEyebrow)}</span>}
      title={t(m.limitsTitle)}
      description={<span className="text-[13.5px]">{t(m.limitsDesc)}</span>}
      hideClose
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            onClick={() =>
              save.mutate(
                { params: { path: { workspace: slug } }, body: { enabled, rateLimitPerMinute: Number(rate), approvalExpiryHours: Number(hours) } },
                {
                  onSuccess: (r) => {
                    qc.setQueryData(['get', '/api/v1/workspaces/{workspace}/mcp', { params: { path: { workspace: slug } } }], r)
                    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/mcp'] })
                    toast({ title: tt(m.limitsSaved) })
                    onClose()
                  },
                  onError: (e) => setError(errorMessage(e)),
                },
              )
            }
          >
            {t(m.saveLimits)}
          </Button>
        </>
      }
    >
      <label className="row gap-2.5 text-[13.5px]">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        {t(m.serverOnToggle)}
      </label>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          {t(m.ratePerAgent)}
          <input className="in" type="number" min={1} max={6000} value={rate} onChange={(e) => setRate(e.target.value)} />
        </label>
        <label className="lbl">
          {t(m.approvalHours)}
          <input className="in" type="number" min={1} max={168} value={hours} onChange={(e) => setHours(e.target.value)} />
        </label>
      </div>
      {error ? (
        <Note tone="bad" label={t(m.notSaved)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

function McpSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
      </span>
      <section className="card flex flex-wrap gap-5 px-5 py-4.5">
        <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-3">
          <Skeleton w="70%" h={16} />
          <Skeleton w="95%" h={12} />
          <Skeleton h={36} />
          <Skeleton h={36} />
        </div>
        <Skeleton className="min-w-0 flex-[1_1_380px]" h={170} r={8} />
      </section>
      <section className="card">
        <TableSkeleton rows={2} label={t(m.loadingAgents)} />
      </section>
    </div>
  )
}
