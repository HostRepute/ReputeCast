import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { EmptyRow, Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { CreateKeyDialog } from './CreateKeyDialog'
import { KeySecretDialog } from './KeySecretDialog'
import { DashedEmpty, TableSkeleton } from './parts'

type Overview = Schemas['MCPOverview']

const levelTone: Record<Schemas['MCPCapabilityCell']['level'], Tone> = { allow: 'ok', approve: 'info', partial: 'warn', deny: 'neutral' }

/** A mono chip for a tool name (SettingsMCP `.tool`). */
function Tool({ children }: { children: string }) {
  return <span className="mono inline-flex h-6 items-center rounded-[6px] border border-line bg-ground px-2 text-[11.5px]">{children}</span>
}

/** A5 §5.10 MCP server (SettingsMCP): endpoint, the agents' capabilities, recent calls, tools and limits. */
export function McpPage() {
  const { slug, me } = useWorkspace()
  const can = me.permissions.includes('mcp:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp', { params: { path: { workspace: slug } } }, { enabled: can })
  const agents = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', { params: { path: { workspace: slug } } }, { enabled: can })
  const [adding, setAdding] = useState(false)
  const [secret, setSecret] = useState<Schemas['APIKeyCreated']>()
  const tabs = settingsTabs(slug).map((t) => (t.to === '/w/$workspace/settings/keys' ? { ...t, current: true } : t))
  return (
    <Page
      title="Settings"
      tabs={tabs}
      actions={
        q.data ? (
          <>
            {q.data.enabled ? (
              <Badge tone="ok" dot>
                MCP server on
              </Badge>
            ) : (
              <Badge tone="neutral" dot>
                MCP server off
              </Badge>
            )}
            <Button variant="primary" disabledReason={agents.data ? undefined : 'Loading…'} onClick={() => setAdding(true)}>
              Add agent
            </Button>
          </>
        ) : null
      }
    >
      {!can ? (
        <ForbiddenState permission="mcp:manage" title="Your role can't manage MCP agents" />
      ) : q.isError ? (
        <ErrorPanel title="Couldn't load the MCP server settings" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !q.data ? (
        <McpSkeleton />
      ) : (
        <>
          <ConnectCard o={q.data} />
          {q.data.agents.length === 0 ? (
            <DashedEmpty
              title="No MCP agents"
              action={
                <Button onClick={() => setAdding(true)} disabledReason={agents.data ? undefined : 'Loading…'}>
                  Add agent
                </Button>
              }
            >
              Give an AI assistant read access first; grant drafting or sending later, with approvals.
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
          Let AI agents work in this workspace, with limits
        </h2>
        <span className="text-[13px] text-ink-2">
          ReputeCast exposes a Model Context Protocol server. Each agent gets its own token and permission set; anything that sends mail needs a human approval unless you
          say otherwise. Every call lands in the audit log.
        </span>
        <label className="lbl">
          Endpoint
          <input className="in mono bg-white" value={o.endpoint} readOnly onFocus={(e) => e.currentTarget.select()} />
        </label>
        <label className="lbl">
          Transport
          <select className="in" defaultValue="http" aria-describedby="mcp-transport-hint">
            <option value="http">Streamable HTTP · OAuth 2.1 or per-agent token</option>
            <option value="stdio" disabled>
              stdio · local bridge (run an MCP proxy next to the client)
            </option>
          </select>
          <span className="vh" id="mcp-transport-hint">
            The server speaks Streamable HTTP; stdio clients connect through a local bridge.
          </span>
        </label>
      </div>
      <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-2">
        <span className="k">Connect from Claude Desktop, Claude Code or any MCP client</span>
        <pre className="pre" aria-label="Client configuration">
          {snippet}
        </pre>
      </div>
    </section>
  )
}

function AgentsMatrix({ o }: { o: Overview }) {
  const { slug } = useWorkspace()
  const caps = o.capabilities
  return (
    <section className="card" aria-labelledby="mcp-agents-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="mcp-agents-h">
            Agents · {o.agents.length}
          </h2>
          <span className="k">Permissions per agent. &quot;Approve&quot; means a team member confirms in ReputeCast before the action runs.</span>
        </div>
      </div>
      <Table phoneWrap density="dense" caption="Agents and what they can do" className="[&_td:first-child]:min-w-[200px] [&_td:nth-child(4)]:min-w-[240px] [&_td]:whitespace-normal">
        <thead>
          <tr>
            <Th>Agent</Th>
            {caps.map((c) => (
              <Th key={c} align="center">
                {c}
              </Th>
            ))}
            <Th>Last call</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {o.agents.map((a) => (
            <tr key={a.id}>
              <Td>
                <div className="font-semibold">{a.name}</div>
                <div className="k">
                  token {a.masked} · created by {a.createdBy} · {a.permissions.join(', ')}
                </div>
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
                <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={{ workspace: slug }} aria-label={`Edit ${a.name} in Keys & MCP`}>
                  Edit
                </ButtonLink>
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </section>
  )
}

/** "09:36 · get_campaign_report" from the agents list. */
function LastCall({ agentId }: { agentId: string }) {
  const { slug } = useWorkspace()
  const now = useNow()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', { params: { path: { workspace: slug } } })
  const k = q.data?.items.find((x) => x.id === agentId)
  if (!k) return <Skeleton w={110} />
  if (!k.lastUsedAt) return <span className="k">Never</span>
  return (
    <>
      {logTime(k.lastUsedAt, now)}
      {k.lastCall ? ` · ${k.lastCall}` : ''}
    </>
  )
}

function ActionsCard() {
  const { slug } = useWorkspace()
  const now = useNow()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents/actions', { params: { path: { workspace: slug }, query: { limit: 5 } } })
  return (
    <section className="card min-w-0 flex-[1.4_1_560px]" aria-labelledby="mcp-acts-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="mcp-acts-h">
          Recent agent actions
        </h2>
        <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ actorType: 'mcp_agent' }} className="k">
          Full audit log
        </Link>
      </div>
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the agent actions" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton rows={3} label="Loading the agent actions" />
      ) : (
        <Table phoneWrap density="dense" caption="Recent agent actions" className="[&_td:first-child]:min-w-[200px] [&_td:nth-child(2)]:whitespace-nowrap [&_td:nth-child(4)]:min-w-[240px]">
          <thead>
            <tr>
              <Th>Time</Th>
              <Th>Agent</Th>
              <Th>Tool</Th>
              <Th>Result</Th>
              <Th>Approval</Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.length === 0 ? (
              <EmptyRow colSpan={5}>No agent calls yet.</EmptyRow>
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
                      <Badge tone="neutral">{a.note ? `Not needed · ${a.note}` : 'Not needed'}</Badge>
                    ) : (
                      <span className="inline-flex items-center gap-2">
                        <Badge tone="bad">{a.result === 'denied' ? 'Denied' : 'Error'}</Badge>
                        {a.note?.includes('proposal') ? (
                          <ButtonLink size="sm" to="/w/$workspace/approvals" params={{ workspace: slug }}>
                            Review
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
  const [editing, setEditing] = useState(false)
  return (
    <section className="card flex min-w-0 flex-[1_1_360px] flex-col gap-2.5 px-4.5 py-4" aria-labelledby="mcp-tools-h">
      <h2 className="h2" id="mcp-tools-h">
        Tools exposed · {o.tools.length}
      </h2>
      <ul className="m-0 flex list-none flex-wrap gap-1.5 p-0" aria-label="Tools">
        {o.tools.map((t) => (
          <li key={t.name} title={`${t.description} Needs ${t.permission}.`}>
            <Tool>{t.name}</Tool>
          </li>
        ))}
      </ul>
      <span className="k">
        Resources: {o.resources.join(', ')}. Prompts: {o.prompts.map((p) => `"${p}"`).join(', ')}.
      </span>
      <ul className="m-0 flex list-none flex-col gap-2.5 p-0 text-[13px]" aria-label="Limits">
        {[
          `Rate limit each agent to ${o.rateLimitPerMinute} calls per minute`,
          `Approvals expire after ${o.approvalExpiryHours} ${o.approvalExpiryHours === 1 ? 'hour' : 'hours'}`,
          "Agents never see API keys, SSO settings or other agents' tokens",
        ].map((t) => (
          <li key={t} className="flex items-start gap-2.5">
            <span className="mt-px inline-flex size-4 flex-none items-center justify-center rounded-[3px] bg-accent text-[11px] font-bold text-white" aria-hidden="true">
              ✓
            </span>
            <span>{t}</span>
          </li>
        ))}
      </ul>
      {o.canManage ? (
        <span>
          <Button size="sm" onClick={() => setEditing(true)}>
            Change limits
          </Button>
        </span>
      ) : null}
      {editing ? <LimitsDialog o={o} onClose={() => setEditing(false)} /> : null}
    </section>
  )
}

function LimitsDialog({ o, onClose }: { o: Overview; onClose: () => void }) {
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
      eyebrow={<span className="k">Dialog · MCP server limits</span>}
      title="MCP server limits"
      description={<span className="text-[13.5px]">Apply to every agent in this workspace. Changes are logged.</span>}
      hideClose
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
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
                    toast({ title: 'MCP server limits saved' })
                    onClose()
                  },
                  onError: (e) => setError(errorMessage(e)),
                },
              )
            }
          >
            Save limits
          </Button>
        </>
      }
    >
      <label className="row gap-2.5 text-[13.5px]">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        MCP server on · agents can connect
      </label>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          Calls per minute, per agent
          <input className="in" type="number" min={1} max={6000} value={rate} onChange={(e) => setRate(e.target.value)} />
        </label>
        <label className="lbl">
          Approvals expire after (hours)
          <input className="in" type="number" min={1} max={168} value={hours} onChange={(e) => setHours(e.target.value)} />
        </label>
      </div>
      {error ? (
        <Note tone="bad" label="Not saved">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

function McpSkeleton() {
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <span className="vh" role="status">
        Loading the MCP server settings
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
        <TableSkeleton rows={2} label="Loading the agents" />
      </section>
    </div>
  )
}
