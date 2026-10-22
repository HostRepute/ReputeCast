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
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, createdTime, logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useWorkspace } from '@/lib/session'
import { AccessTable, PersonCell } from '../access/parts'
import { CreateKeyDialog, type KeyKind } from './CreateKeyDialog'
import { KeyActionDialog, type KeyAction } from './KeyActionDialogs'
import { KeySecretDialog } from './KeySecretDialog'
import { m } from './messages'
import { DashedEmpty, ExpiresCell, LastUsedCell, PermBadges, scopeText, TableSkeleton, type Key } from './parts'
import { SendOnlyKeyDialog } from '@/features/transactional/CreateKeyDialog'
import { KeyBadges } from '@/features/transactional/shared'

type Secret = { created: Schemas['APIKeyCreated']; action: 'created' | 'rotated' | 'replaced' }

/** A5 §5.10 Keys & MCP (SettingsKeys): API keys, MCP agents and their recent actions. */
export function KeysPage() {
  const t = useT()
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
      title={t(m.title)}
      tabs={settingsTabs(slug)}
      actions={
        canKeys ? (
          <Button variant="primary" disabledReason={keys.data ? undefined : t(common.loading)} onClick={() => setCreate('api_key')}>
            {t(m.createKey)}
          </Button>
        ) : null
      }
    >
      {!canKeys && !canAgents ? <ForbiddenState permission="apikeys:manage" title={t(m.forbidden)} /> : null}
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
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  if (q.data && q.data.items.length === 0)
    return (
      <DashedEmpty
        title={t(m.noKeysTitle)}
        action={
          <Button variant="primary" onClick={onCreate}>
            {t(m.createKey)}
          </Button>
        }
      >
        <T
          msg={m.noKeysBody}
          values={{
            link: (c) => (
              <Link to="/w/$workspace/transactional/setup" params={{ workspace: slug }}>
                {c}
              </Link>
            ),
          }}
        />
      </DashedEmpty>
    )
  return (
    <section className="card" aria-labelledby="keys-h">
      <CardSection
        title={<span id="keys-h">{q.data ? t(m.keysTitle, { count: count(q.data.items.length) }) : t(m.keysTitleLoading)}</span>}
        caption={t(m.keysCaption)}
        actions={
          <>
            <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={{ workspace: slug }}>
              {t(m.apiReference)}
            </ButtonLink>
            <Button size="sm" variant="primary" disabledReason={q.data ? undefined : t(common.loading)} onClick={onCreate}>
              {t(m.createKey)}
            </Button>
          </>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.keysLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton label={t(m.keysLoading)} />
      ) : (
        <AccessTable caption={t(m.keysTable)}>
          <thead>
            <tr>
              <Th>{t(m.colKey)}</Th>
              <Th>{t(m.colPermissions)}</Th>
              <Th>{t(m.colScope)}</Th>
              <Th>{t(m.colCreatedBy)}</Th>
              <Th>{t(m.colLastUsed)}</Th>
              <Th>{t(m.colExpires)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
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
                        <Badge tone="info">{t(m.conversionsBadge)}</Badge>
                        {k.masked.startsWith('rc_test_') ? <Badge tone="purple">{t(m.testBadge)}</Badge> : <Badge tone="ok">{t(m.liveBadge)}</Badge>}
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
  const t = useT()
  return (
    <div className="row gap-1.5">
      {k.status === 'expired' ? (
        <>
          <Button size="sm" onClick={() => onAction(k, 'replace')} aria-label={t(m.createReplacementFor, { name: k.name })}>
            {t(m.createReplacement)}
          </Button>
          <Button size="sm" onClick={() => onAction(k, 'delete')} aria-label={t(m.deleteNamed, { name: k.name })}>
            {t(common.delete)}
          </Button>
        </>
      ) : (
        <>
          <Button size="sm" onClick={() => onAction(k, 'rotate')} aria-label={t(m.rotateNamed, { name: k.name })}>
            {t(m.rotate)}
          </Button>
          <Button size="sm" variant="danger" onClick={() => onAction(k, 'revoke')} aria-label={t(m.revokeNamed, { name: k.name })}>
            {t(m.revoke)}
          </Button>
        </>
      )}
    </div>
  )
}

/** "The Ops agent can stop a send but never start one: campaigns:pause without campaigns:send" */
function agentsCaption(items: Key[]) {
  const a = items.find((k) => k.permissions.includes('campaigns:pause') && !k.permissions.includes('campaigns:send'))
  if (!a) return tt(m.agentsCaption)
  return tt(m.agentsCaptionPause, { article: /agent$/i.test(a.name) ? 'the' : 'quoted', name: a.name })
}

function AgentsCard({ q, onCreate, onAction }: { q: ListQuery; onCreate: () => void; onAction: (k: Key, a: KeyAction) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  if (q.data && q.data.items.length === 0)
    return (
      <DashedEmpty
        title={t(m.noAgentsTitle)}
        action={
          <Button onClick={onCreate} disabledReason={q.data ? undefined : t(common.loading)}>
            {t(m.addAgent)}
          </Button>
        }
      >
        {t(m.noAgentsBody)}
      </DashedEmpty>
    )
  return (
    <section className="card" aria-labelledby="agents-h">
      <CardSection
        title={<span id="agents-h">{q.data ? t(m.agentsTitle, { count: count(q.data.items.length) }) : t(m.agentsTitleLoading)}</span>}
        caption={q.data ? agentsCaption(q.data.items) : t(m.agentsCaptionLoading)}
        actions={
          <>
            <ButtonLink size="sm" to="/w/$workspace/settings/mcp" params={{ workspace: slug }}>
              {t(m.serverEndpoint)}
            </ButtonLink>
            <Button size="sm" variant="primary" disabledReason={q.data ? undefined : t(common.loading)} onClick={onCreate}>
              {t(m.addAgent)}
            </Button>
          </>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.agentsLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton rows={2} label={t(m.agentsLoading)} />
      ) : (
        <AccessTable caption={t(m.agentsTable)}>
          <thead>
            <tr>
              <Th>{t(m.colAgent)}</Th>
              <Th>{t(m.colPermissions)}</Th>
              <Th>{t(m.colScope)}</Th>
              <Th>{t(m.colCreatedBy)}</Th>
              <Th>{t(m.colLastCall)}</Th>
              <Th>{t(m.colExpires)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
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
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-agents/actions', { params: { path: { workspace: slug }, query: { limit: 5 } } })
  return (
    <section className="card" aria-labelledby="acts-h">
      <CardSection
        title={<span id="acts-h">{t(m.actionsTitle)}</span>}
        caption={t(m.actionsCaption)}
        actions={
          <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ actorType: 'mcp_agent' }}>
            {t(m.fullAuditLog)}
          </ButtonLink>
        }
      />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.actionsLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <TableSkeleton rows={3} label={t(m.actionsLoading)} />
      ) : (
        <AccessTable caption={t(m.actionsTable)}>
          <thead>
            <tr>
              <Th>{t(m.colWhen)}</Th>
              <Th>{t(m.colAgent)}</Th>
              <Th>{t(m.colTool)}</Th>
              <Th>{t(m.colTarget)}</Th>
              <Th>{t(m.colResult)}</Th>
            </tr>
          </thead>
          <tbody>
            {q.data.items.length === 0 ? (
              <EmptyRow colSpan={5}>{t(m.noCalls)}</EmptyRow>
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
                      {a.result === 'allowed' ? (
                        <Badge tone="ok">{t(m.resultAllowed)}</Badge>
                      ) : (
                        <Badge tone="bad">{a.result === 'denied' ? t(m.resultDenied) : t(m.resultError)}</Badge>
                      )}
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
