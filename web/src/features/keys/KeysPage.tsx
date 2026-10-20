import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { EmptyRow, Td, Th } from '@/components/ui/Table'
import { ManagedKeysCard } from '@/features/integrations/ManagedKeys'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { createdTime, logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useWorkspace } from '@/lib/session'
import { AccessTable, PersonCell } from '../access/parts'
import { CreateKeyDialog, type KeyKind } from './CreateKeyDialog'
import { KeyActionDialog, type KeyAction } from './KeyActionDialogs'
import { KeySecretDialog } from './KeySecretDialog'
import { DashedEmpty, ExpiresCell, LastUsedCell, PermBadges, scopeText, TableSkeleton, type Key } from './parts'
import { SendOnlyKeyDialog } from '@/features/transactional/CreateKeyDialog'
import { KeyBadges } from '@/features/transactional/shared'

type Secret = { created: Schemas['APIKeyCreated']; action: 'created' | 'rotated' | 'replaced' }

/** A5 §5.10 Keys & MCP (SettingsKeys): API keys, MCP agents and their recent actions. */
export function KeysPage() {
  const { slug, me } = useWorkspace()
  const canKeys = me.permissions.includes('apikeys:manage')
  const canAgents = me.permissions.includes('mcp:manage')
  const path = { params: { path: { workspace: slug } } }
  const keys = $api.useQuery('get', '/api/v1/workspaces/{workspace}/api-keys', path, { enabled: canKeys })
  const agents = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents', path, { enabled: canAgents })
  const [create, setCreate] = useState<KeyKind>()
  const [sendOnly, setSendOnly] = useState(false)
  const canTransactional = me.permissions.includes('transactional:view')
  const [act, setAct] = useState<{ k: Key; action: KeyAction }>()
  const [secret, setSecret] = useState<Secret>()
  const createList = create === 'mcp_agent' ? agents.data : keys.data
  return (
    <Page
      title="Settings"
      tabs={settingsTabs(slug)}
      actions={
        canKeys ? (
          <Button variant="primary" disabledReason={keys.data ? undefined : 'Loading…'} onClick={() => setCreate('api_key')}>
            Create API key
          </Button>
        ) : null
      }
    >
      {!canKeys && !canAgents ? <ForbiddenState permission="apikeys:manage" title="Your role can't manage API keys or MCP agents" /> : null}
      {!canKeys && !canAgents ? null : canKeys ? (
        <>
          <KeysCard q={keys} onCreate={() => setCreate('api_key')} onAction={(k, action) => setAct({ k, action })} />
          <ManagedKeysCard />
        </>
      ) : (
        <Note tone="neutral">{deniedReason('apikeys:manage')}</Note>
      )}
      {canAgents ? (
        <>
          <AgentsCard q={agents} onCreate={() => setCreate('mcp_agent')} onAction={(k, action) => setAct({ k, action })} />
          {agents.data?.items.length === 0 ? null : <ActionsCard />}
        </>
      ) : null}
      {create && createList ? (
        <CreateKeyDialog
          kind={create}
          list={createList}
          onClose={() => setCreate(undefined)}
          onCreated={(c) => {
            setCreate(undefined)
            setSecret({ created: c, action: 'created' })
          }}
          onSendOnly={
            create === 'api_key' && canTransactional
              ? () => {
                  setCreate(undefined)
                  setSendOnly(true)
                }
              : undefined
          }
        />
      ) : null}
      {sendOnly ? <SendOnlyKeyDialog onClose={() => setSendOnly(false)} /> : null}
      {act ? (
        <KeyActionDialog
          k={act.k}
          action={act.action}
          onClose={() => setAct(undefined)}
          onSecret={(c, action) => {
            setAct(undefined)
            setSecret({ created: c, action })
          }}
        />
      ) : null}
      {secret ? <KeySecretDialog created={secret.created} action={secret.action} onClose={() => setSecret(undefined)} /> : null}
    </Page>
  )
}

type ListQuery = { data?: Schemas['APIKeyList']; isPending: boolean; isError: boolean; error: unknown; refetch: () => unknown }

function KeysCard({ q, onCreate, onAction }: { q: ListQuery; onCreate: () => void; onAction: (k: Key, a: KeyAction) => void }) {
  const { slug } = useWorkspace()
  const now = useNow()
  if (q.data && q.data.items.length === 0)
    return (
      <DashedEmpty
        title="No API keys"
        action={
          <Button variant="primary" onClick={onCreate}>
            Create API key
          </Button>
        }
      >
        Create a scoped key to add contacts, log events or{' '}
        <Link to="/w/$workspace/transactional/setup" params={{ workspace: slug }}>
          send transactional mail from your product
        </Link>
        .
      </DashedEmpty>
    )
  return (
    <section className="card" aria-labelledby="keys-h">
      <CardSection
        title={<span id="keys-h">API keys{q.data ? ` · ${q.data.items.length}` : ''}</span>}
        caption="Keys use the same permission keys as roles, never more than their creator holds. Secrets are shown once at creation."
        actions={
          <>
            <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={{ workspace: slug }}>
              API reference
            </ButtonLink>
            <Button size="sm" variant="primary" disabledReason={q.data ? undefined : 'Loading…'} onClick={onCreate}>
              Create API key
            </Button>
          </>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the API keys" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton label="Loading the API keys" />
      ) : (
        <AccessTable caption="API keys">
          <thead>
            <tr>
              <Th>Key</Th>
              <Th>Permissions</Th>
              <Th>Scope</Th>
              <Th>Created by</Th>
              <Th>Last used</Th>
              <Th>Expires</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.map((k) => (
              <tr key={k.id} className={k.status === 'expired' ? 'text-ink-3' : undefined}>
                <Td>
                  <div className="col gap-0.5">
                    <span className="font-semibold">{k.name}</span>
                    <span className="k mono">{k.masked}</span>
                    {k.transactional ? (
                      <span className="mt-[3px]">
                        <KeyBadges preset={k.transactional.preset} mode={k.transactional.mode} />
                      </span>
                    ) : k.permissions.length === 1 && k.permissions[0] === 'conversions:create' ? (
                      // Earn's conversion API key (EarnSettingsStates).
                      <span className="mt-[3px] inline-flex flex-wrap gap-1.5">
                        <Badge tone="info">Conversions</Badge>
                        {k.masked.startsWith('rc_test_') ? <Badge tone="purple">Test</Badge> : <Badge tone="ok">Live</Badge>}
                      </span>
                    ) : null}
                  </div>
                </Td>
                <Td>
                  <PermBadges keys={k.permissions} />
                </Td>
                <Td>{scopeText(k)}</Td>
                <Td>
                  <PersonCell name={k.createdBy.name} sub={createdTime(k.createdAt, now)} color={k.createdBy.avatarColor} />
                </Td>
                <Td>
                  <LastUsedCell k={k} now={now} />
                </Td>
                <Td>
                  <ExpiresCell k={k} />
                </Td>
                <Td>
                  <KeyButtons k={k} onAction={onAction} />
                </Td>
              </tr>
            ))}
          </tbody>
        </AccessTable>
      )}
    </section>
  )
}

function KeyButtons({ k, onAction }: { k: Key; onAction: (k: Key, a: KeyAction) => void }) {
  return (
    <div className="row gap-1.5">
      {k.status === 'expired' ? (
        <>
          <Button size="sm" onClick={() => onAction(k, 'replace')} aria-label={`Create replacement for ${k.name}`}>
            Create replacement
          </Button>
          <Button size="sm" onClick={() => onAction(k, 'delete')} aria-label={`Delete ${k.name}`}>
            Delete
          </Button>
        </>
      ) : (
        <>
          <Button size="sm" onClick={() => onAction(k, 'rotate')} aria-label={`Rotate ${k.name}`}>
            Rotate
          </Button>
          <Button size="sm" variant="danger" onClick={() => onAction(k, 'revoke')} aria-label={`Revoke ${k.name}`}>
            Revoke
          </Button>
        </>
      )}
    </div>
  )
}

/** "The Ops agent can stop a send but never start one: campaigns:pause without campaigns:send" */
function agentsCaption(items: Key[]) {
  const a = items.find((k) => k.permissions.includes('campaigns:pause') && !k.permissions.includes('campaigns:send'))
  if (!a) return 'Agents call the MCP server with their own token; sending is always proposed to a person'
  const who = /agent$/i.test(a.name) ? `The ${a.name}` : `“${a.name}”`
  return `${who} can stop a send but never start one: campaigns:pause without campaigns:send`
}

function AgentsCard({ q, onCreate, onAction }: { q: ListQuery; onCreate: () => void; onAction: (k: Key, a: KeyAction) => void }) {
  const { slug } = useWorkspace()
  const now = useNow()
  if (q.data && q.data.items.length === 0)
    return (
      <DashedEmpty
        title="No MCP agents"
        action={
          <Button onClick={onCreate} disabledReason={q.data ? undefined : 'Loading…'}>
            Add agent
          </Button>
        }
      >
        Give an AI assistant read access first; grant drafting or sending later, with approvals.
      </DashedEmpty>
    )
  return (
    <section className="card" aria-labelledby="agents-h">
      <CardSection
        title={<span id="agents-h">MCP agents{q.data ? ` · ${q.data.items.length}` : ''}</span>}
        caption={q.data ? agentsCaption(q.data.items) : 'Agents call the MCP server with their own token'}
        actions={
          <>
            <ButtonLink size="sm" to="/w/$workspace/settings/mcp" params={{ workspace: slug }}>
              Server endpoint &amp; transport
            </ButtonLink>
            <Button size="sm" variant="primary" disabledReason={q.data ? undefined : 'Loading…'} onClick={onCreate}>
              Add agent
            </Button>
          </>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the MCP agents" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton rows={2} label="Loading the MCP agents" />
      ) : (
        <AccessTable caption="MCP agents">
          <thead>
            <tr>
              <Th>Agent</Th>
              <Th>Permissions</Th>
              <Th>Scope</Th>
              <Th>Created by</Th>
              <Th>Last call</Th>
              <Th>Expires</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.map((k) => (
              <tr key={k.id} className={k.status === 'expired' ? 'text-ink-3' : undefined}>
                <Td>
                  <div className="col gap-0.5">
                    <span className="font-semibold">{k.name}</span>
                    <span className="k mono">
                      {k.masked}
                      {k.description ? ` · ${k.description}` : ''}
                    </span>
                  </div>
                </Td>
                <Td>
                  <PermBadges keys={k.permissions} />
                </Td>
                <Td>{scopeText(k)}</Td>
                <Td>
                  <PersonCell name={k.createdBy.name} sub={createdTime(k.createdAt, now)} color={k.createdBy.avatarColor} />
                </Td>
                <Td>
                  <LastUsedCell k={k} now={now} />
                </Td>
                <Td>
                  <ExpiresCell k={k} />
                </Td>
                <Td>
                  <KeyButtons k={k} onAction={onAction} />
                </Td>
              </tr>
            ))}
          </tbody>
        </AccessTable>
      )}
    </section>
  )
}

/** The last agent calls (SettingsKeys, "Recent agent actions"). */
function ActionsCard() {
  const { slug } = useWorkspace()
  const now = useNow()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents/actions', { params: { path: { workspace: slug }, query: { limit: 5 } } })
  return (
    <section className="card" aria-labelledby="acts-h">
      <CardSection
        title={<span id="acts-h">Recent agent actions</span>}
        caption="Every call is in the audit log with the agent as actor"
        actions={
          <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ actorType: 'mcp_agent' }}>
            Full audit log
          </ButtonLink>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the agent actions" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton rows={3} label="Loading the agent actions" />
      ) : (
        <AccessTable caption="Recent agent actions">
          <thead>
            <tr>
              <Th>When</Th>
              <Th>Agent</Th>
              <Th>Tool</Th>
              <Th>Target</Th>
              <Th>Result</Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.length === 0 ? (
              <EmptyRow colSpan={5}>No agent calls yet.</EmptyRow>
            ) : (
              q.data.items.map((a) => (
                <tr key={a.id}>
                  <Td>{logTime(a.occurredAt, now)}</Td>
                  <Td>{a.agent}</Td>
                  <Td>
                    <span className="mono">{a.tool}</span>
                  </Td>
                  <Td>{a.target}</Td>
                  <Td>
                    <span className="row gap-1.5">
                      {a.result === 'allowed' ? <Badge tone="ok">Allowed</Badge> : <Badge tone="bad">{a.result === 'denied' ? 'Denied' : 'Error'}</Badge>}
                      {a.note ? <span className="k">{a.note}</span> : null}
                    </span>
                  </Td>
                </tr>
              ))
            )}
          </tbody>
        </AccessTable>
      )}
    </section>
  )
}

