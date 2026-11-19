import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { LoadFailed } from '@/components/ui/LoadFailed'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, dayMonthDate } from '@/lib/format'
import { useReducedMotion } from '@/lib/hooks'
import { roleName } from '@/lib/roles'
import { deniedReason, useCan, useMe, useWorkspace } from '@/lib/session'
import { useAISettings } from '@/features/ai/api'
import { AINote, CardHead, tags } from '@/features/ai/parts'
import '@/features/ai/ai.css'
import '@/features/integrations/integrations.css'
import { disconnect, startSignIn, syncServer, useMCPServers, useRefreshMCP, type MCPServer, type MCPTool } from './api'
import { m } from './messages'
import { rolesList, ServerDialog } from './ServerDialog'
import { ChangedCard, kindLabel, ReviewDialog, toolState } from './ToolReview'
import { firstLine, isChanged } from './tools'
import './mcp.css'

export type Callback = { server?: string; connected?: boolean; error?: 'denied' | 'token' }

type Dialog = { kind: 'add' } | { kind: 'edit'; server: string } | { kind: 'review'; server: string; tool: string } | null

/** The server's badge: off, unreachable, sign-in, changed or waiting tools, else its tool count. */
function ServerBadge({ s }: { s: MCPServer }) {
  const t = useT()
  const live = s.tools.filter((x) => !x.removed)
  const changed = live.filter(isChanged).length
  const pending = live.filter((x) => x.decision === 'pending').length
  if (!s.enabled) return <Badge tone="neutral">{t(m.badgeOff)}</Badge>
  if (s.status === 'unreachable')
    return (
      <Badge tone="bad" dot>
        {t(m.badgeUnreachable)}
      </Badge>
    )
  if (s.status === 'sign_in')
    return (
      <Badge tone="warn" dot>
        {t(m.badgeSignIn)}
      </Badge>
    )
  if (changed)
    return (
      <Badge tone="warn" dot>
        {t(m.badgeChanged, { count: changed })}
      </Badge>
    )
  if (pending)
    return (
      <Badge tone="warn" dot>
        {t(m.badgeWaiting, { count: pending })}
      </Badge>
    )
  if (s.status === 'new' && !s.syncedAt) return <Badge tone="neutral">{t(m.badgeNotSynced)}</Badge>
  return (
    <Badge tone="neutral" dot>
      {t(m.badgeTools, { count: live.length })}
    </Badge>
  )
}

function authLine(t: ReturnType<typeof useT>, s: MCPServer): string {
  if (s.auth === 'oauth') return t(m.authOAuth)
  if (s.auth === 'header') return s.headerSet ? t(m.authHeaderSet) : t(m.authHeaderUnset)
  return t(m.authNone)
}

/**
 * Settings › Integrations › MCP servers (MCPServers): the servers with
 * their status, each one's tools with how they are allowed, a tool that
 * changed after approval with its change, and how calls are allowed.
 * Managers add, edit, sync and review; everyone with Use the AI assistant
 * connects their own account to an OAuth server. The OAuth callback lands
 * here (?server=…&connected=1 or &error=denied|token): it is announced and
 * the address cleaned.
 */
export function MCPServersPage({ callback }: { callback: Callback }) {
  const { slug, me } = useWorkspace()
  const t = useT()
  const navigate = useNavigate()
  const q = useMCPServers()
  const canManage = useCan('integrations:manage')
  const canUse = useCan('ai:use')
  const helpers = useAISettings().data?.helpers ?? []
  const user = useMe().data?.user
  const refresh = useRefreshMCP()
  const reduced = useReducedMotion()
  const servers = q.data?.servers
  // The server whose tools are shown; null: the person hid them.
  const [picked, setPicked] = useState<string | null | undefined>(callback.server)
  const [dialog, setDialog] = useState<Dialog>(null)
  const [syncing, setSyncing] = useState<string>()
  const [signing, setSigning] = useState<string>()

  // The server shown: the one picked, else the first with a change, else the first; none once hidden.
  const selected = picked === null ? undefined : (servers?.find((s) => s.id === picked) ?? servers?.find((s) => s.tools.some(isChanged)) ?? servers?.[0])

  async function sync(s: MCPServer) {
    setSyncing(s.id)
    try {
      const out = await syncServer(slug, s.id)
      await refresh()
      toast({ kind: 'success', title: t(m.synced, { server: s.name, count: out.tools.filter((x) => !x.removed).length }) })
    } catch (e) {
      await refresh()
      toast({ kind: 'error', title: t(m.syncFailed, { server: s.name }), detail: errorMessage(e) })
    } finally {
      setSyncing(undefined)
    }
  }

  async function signIn(s: MCPServer) {
    setSigning(s.id)
    try {
      await startSignIn(slug, s.id)
    } catch (e) {
      setSigning(undefined)
      toast({ kind: 'error', title: t(m.signInFailed, { server: s.name }), detail: errorMessage(e) })
    }
  }

  async function signOut(s: MCPServer) {
    try {
      await disconnect(slug, s.id)
      await refresh()
      toast({ kind: 'success', title: t(m.disconnectedToast, { server: s.name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.disconnectFailed, { server: s.name }), detail: errorMessage(e) })
    }
  }

  // Back from the server's sign-in: say how it went once, then clean the address.
  const handled = useRef(false)
  useEffect(() => {
    if (handled.current || !(callback.connected || callback.error) || !servers) return
    handled.current = true
    const s = servers.find((x) => x.id === callback.server)
    const server = s?.name ?? t(m.aServer)
    if (callback.connected) {
      toast({ kind: 'success', title: t(m.connectedToast, { server }) })
      // A manager's first sign-in to a server that was never synced lists its tools.
      if (s && canManage && !s.syncedAt) window.setTimeout(() => void sync(s))
    } else if (callback.error === 'denied') toast({ kind: 'info', title: t(m.deniedToast, { server }) })
    else toast({ kind: 'error', title: t(m.tokenToast, { server }), detail: t(m.tokenDetail) })
    void navigate({ to: '/w/$workspace/settings/integrations/mcp-servers', params: { workspace: slug }, search: {}, replace: true })
    // Once, when the list is in (sync and t are stable enough for a one-off).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [servers, callback, slug, navigate, canManage])

  const changed = (servers ?? []).flatMap((s) => s.tools.filter(isChanged).map((tool) => ({ s, tool })))
  const reviewing = dialog?.kind === 'review' ? servers?.find((s) => s.id === dialog.server) : undefined
  const reviewTool = dialog?.kind === 'review' ? reviewing?.tools.find((x) => x.id === dialog.tool) : undefined
  const editing = dialog?.kind === 'edit' ? servers?.find((s) => s.id === dialog.server) : undefined
  const myRole = me.workspace.roleKey

  const reviewChange = (s: MCPServer) => {
    setPicked(s.id)
    window.setTimeout(() => {
      const card = document.getElementById(`mcp-changed-${s.id}`)
      card?.scrollIntoView({ block: 'start', behavior: reduced ? 'auto' : 'smooth' })
      card?.focus({ preventScroll: true })
    })
  }

  return (
    <Page
      title={t(m.title)}
      documentTitle={t(m.documentTitle)}
      breadcrumb={[
        { label: t(m.crumbSettings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(m.crumbIntegrations), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: t(m.title) },
      ]}
    >
      {q.isError ? (
        <LoadFailed title={t(m.loadError)} onRetry={() => void q.refetch()} />
      ) : !servers ? (
        <PageSkeleton />
      ) : (
        <div className="mcp-cols">
          <div className="mcp-main">
            {canManage ? null : (
              <AINote tone="neutral">
                <T
                  msg={helpers.length ? m.readOnly : m.readOnlyAlone}
                  values={{ ...tags, name: user?.name ?? '', role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }), helpers: formatList(helpers, 'disjunction') }}
                />
              </AINote>
            )}
            {changed.map(({ s, tool }) => (
              <div key={tool.id} className="flex flex-col items-start gap-3">
                {/* The board's notice: a quiet box with the warning's icon. */}
                <div className="note n mcp-notice self-stretch" role="note">
                  <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-fg" />
                  <div className="min-w-0 flex-1">
                    <b>{t(m.changedNotice, { server: s.name, tool: tool.name })}</b>
                    <div>{selected?.id === s.id ? t(m.changedNoticeShown) : t(m.changedNoticeBody)}</div>
                  </div>
                </div>
                {selected?.id === s.id ? null : <Button onClick={() => reviewChange(s)}>{t(m.reviewChange)}</Button>}
              </div>
            ))}

            <section className="card ai-card" aria-labelledby="mcp-servers-h">
              <CardHead id="mcp-servers-h" title={t(m.serversTitle)}>
                {canManage ? (
                  <Button id="mcp-add" variant="primary" icon="plus" onClick={() => setDialog({ kind: 'add' })}>
                    {t(m.addServer)}
                  </Button>
                ) : null}
              </CardHead>
              {servers.length === 0 ? (
                <div className="mcp-empty">
                  <b>{t(m.emptyTitle)}</b>
                  <span className="k">{canManage ? t(m.emptyBody) : t(m.emptyReadOnly)}</span>
                </div>
              ) : (
                servers.map((s) => {
                  const shown = selected?.id === s.id
                  const allowed = !s.roles.length || s.roles.includes(myRole)
                  return (
                    <div key={s.id} className="mcp-item">
                      <div className="mcp-item-head">
                        <h3>{s.name}</h3>
                        <ServerBadge s={s} />
                      </div>
                      <p className="sub m-0">{t(m.serverLine, { url: s.url, auth: authLine(t, s) })}</p>
                      {s.roles.length ? <small className="k">{t(m.usedByRoles, { roles: rolesList(s.roles) })}</small> : null}
                      {s.statusDetail && (s.status === 'unreachable' || s.status === 'sign_in') ? <small className="k [overflow-wrap:anywhere]">{t(m.statusDetail, { detail: s.statusDetail })}</small> : null}
                      <div className="mcp-item-acts">
                        <Button
                          className="mcp-show"
                          aria-expanded={shown}
                          aria-controls={selected ? 'mcp-tools' : undefined}
                          aria-label={t(shown ? m.hideToolsLabel : m.showToolsLabel, { server: s.name })}
                          onClick={() => setPicked(shown ? null : s.id)}
                        >
                          {shown ? t(m.hideTools) : t(m.showTools)}
                        </Button>
                        {!allowed ? (
                          <small>{t(m.notForRole)}</small>
                        ) : s.auth === 'oauth' ? (
                          !canUse ? (
                            <Button disabledReason={deniedReason('ai:use', helpers)}>{t(m.connect)}</Button>
                          ) : s.connected ? (
                            <>
                              <small>{t(m.connectedAs, { name: user?.name ?? '' })}</small>
                              <Button size="sm" aria-label={t(m.reconnectLabel, { server: s.name })} loading={signing === s.id} onClick={() => void signIn(s)}>
                                {t(m.reconnect)}
                              </Button>
                              <Button size="sm" variant="ghost" aria-label={t(m.disconnectLabel, { server: s.name })} onClick={() => void signOut(s)}>
                                {t(m.disconnect)}
                              </Button>
                            </>
                          ) : (
                            <Button aria-label={t(m.connectLabel, { server: s.name })} loading={signing === s.id} onClick={() => void signIn(s)}>
                              {t(m.connect)}
                            </Button>
                          )
                        ) : (
                          <small>{s.auth === 'header' ? t(m.sharedForWorkspace) : t(m.noSignInNeeded)}</small>
                        )}
                        {canManage ? (
                          <Button size="sm" variant="ghost" className="ml-auto" aria-label={t(m.editLabel, { server: s.name })} onClick={() => setDialog({ kind: 'edit', server: s.id })}>
                            {t(m.edit)}
                          </Button>
                        ) : null}
                      </div>
                    </div>
                  )
                })
              )}
              <div className="ai-foot">
                <small className="k">{t(m.serversFoot)}</small>
              </div>
            </section>

            {selected ? (
              <ToolsCard
                s={selected}
                canManage={canManage}
                syncing={syncing === selected.id}
                onSync={() => void sync(selected)}
                onReview={(tool) => setDialog({ kind: 'review', server: selected.id, tool: tool.id })}
              />
            ) : null}
            {selected
              ? selected.tools.filter(isChanged).map((tool, i) => (
                  <ChangedCard
                    key={tool.id}
                    id={i === 0 ? `mcp-changed-${selected.id}` : `mcp-changed-${tool.id}`}
                    server={selected}
                    tool={tool}
                    canManage={canManage}
                    helpers={helpers}
                    onReview={() => setDialog({ kind: 'review', server: selected.id, tool: tool.id })}
                  />
                ))
              : null}
          </div>
          <div className="mcp-side">
            <section className="card mcp-aside" aria-labelledby="mcp-how-h">
              <h2 className="h2" id="mcp-how-h">
                {t(m.howTitle)}
              </h2>
              <ul className="mcp-checklist">
                {[m.howReadOnly, m.howWrites, m.howChanged, m.howKeys].map((x) => (
                  <li key={x.id}>
                    <Icon name="check" size={16} />
                    <span>{t(x)}</span>
                  </li>
                ))}
              </ul>
            </section>
          </div>
        </div>
      )}
      {dialog?.kind === 'add' && canManage ? (
        <ServerDialog
          onClose={() => setDialog(null)}
          onSaved={(id) => {
            setDialog(null)
            if (id) setPicked(id)
          }}
        />
      ) : null}
      {editing && canManage ? (
        <ServerDialog
          server={editing}
          onClose={() => setDialog(null)}
          onSaved={(id) => {
            setDialog(null)
            if (!id) {
              setPicked(undefined)
              window.setTimeout(() => document.getElementById('mcp-add')?.focus())
            }
          }}
        />
      ) : null}
      {reviewing && reviewTool ? <ReviewDialog key={reviewTool.id} server={reviewing} tool={reviewTool} canManage={canManage} onClose={() => setDialog(null)} /> : null}
    </Page>
  )
}

/** One server's tools (MCPServers › tools): name and description, kind, state, Review; Sync tools. */
function ToolsCard({ s, canManage, syncing, onSync, onReview }: { s: MCPServer; canManage: boolean; syncing: boolean; onSync: () => void; onReview: (tool: MCPTool) => void }) {
  const t = useT()
  const tools = s.tools.filter((x) => !x.removed || x.decision !== 'pending')
  const needsSignIn = s.auth === 'oauth' && !s.connected
  const caption = (
    <>
      {t(m.toolsCaption)}
      {s.syncedAt ? (
        <>
          <br />
          {t(m.syncedAt, { date: dayMonthDate(s.syncedAt), time: clock(s.syncedAt) })}
        </>
      ) : null}
    </>
  )
  return (
    <section className="card ai-card" id="mcp-tools" aria-labelledby="mcp-tools-h">
      <CardHead id="mcp-tools-h" title={t(m.toolsTitle, { server: s.name })} caption={caption}>
        {canManage ? (
          <Button onClick={onSync} loading={syncing} busyLabel={t(m.syncing)} disabledReason={needsSignIn ? t(m.syncNeedsSignIn, { server: s.name }) : undefined}>
            {t(m.sync)}
          </Button>
        ) : null}
      </CardHead>
      {tools.length === 0 ? (
        <div className="mcp-empty">
          <span className="k">{!canManage ? t(m.noToolsReadOnly) : needsSignIn ? t(m.noToolsSignIn) : t(m.noTools)}</span>
        </div>
      ) : (
        <>
          {/* Below 768 px each tool is a stacked row, as the integrations' tables become lists. */}
          <div className="ih-no-phone">
            <Table fit caption={t(m.tableLabel, { server: s.name })}>
              <thead>
                <tr>
                  <Th>{t(m.thTool)}</Th>
                  <Th>{t(m.thKind)}</Th>
                  <Th>{t(m.thState)}</Th>
                  <Th>
                    <span className="vh">{t(m.thActions)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {tools.map((tool) => {
                  const state = toolState(tool)
                  return (
                    <tr key={tool.id}>
                      <Td>
                        <code className="mcp-tool-name">{tool.name}</code>
                        <span className="mcp-tool-desc">{firstLine(tool.description) || t(m.noDescription)}</span>
                      </Td>
                      <Td className="mcp-nw">{t(kindLabel[tool.decision])}</Td>
                      <Td className="mcp-nw">
                        <Badge tone={state.tone} dot>
                          {t(state.label)}
                        </Badge>
                      </Td>
                      <Td align="right">
                        <Button size="sm" aria-label={t(canManage ? m.reviewLabel : m.viewLabel, { tool: tool.name })} onClick={() => onReview(tool)}>
                          {t(canManage ? m.review : m.view)}
                        </Button>
                      </Td>
                    </tr>
                  )
                })}
              </tbody>
            </Table>
          </div>
          <ul className="ih-only-phone ih-list mcp-list" aria-labelledby="mcp-tools-h">
            {tools.map((tool) => {
              const state = toolState(tool)
              return (
                <li key={tool.id} className="ih-it">
                  <code className="mcp-tool-name">{tool.name}</code>
                  <span className="mcp-tool-desc">{firstLine(tool.description) || t(m.noDescription)}</span>
                  <span className="mcp-badges">
                    {/* Blocked and pending say it in the state already. */}
                    {(tool.decision === 'read_only' || tool.decision === 'writes') && <Badge tone="neutral">{t(kindLabel[tool.decision])}</Badge>}
                    <Badge tone={state.tone} dot>
                      {t(state.label)}
                    </Badge>
                  </span>
                  <span>
                    <Button size="sm" aria-label={t(canManage ? m.reviewLabel : m.viewLabel, { tool: tool.name })} onClick={() => onReview(tool)}>
                      {t(canManage ? m.review : m.view)}
                    </Button>
                  </span>
                </li>
              )
            })}
          </ul>
        </>
      )}
    </section>
  )
}

function PageSkeleton() {
  const t = useT()
  return (
    <div className="mcp-cols" aria-busy="true">
      <span className="vh" role="status">
        {t(common.loading)}
      </span>
      <div className="mcp-main">
        <section className="card">
          {[0, 1, 2].map((i) => (
            <div key={i} className="mcp-item">
              <Skeleton w="30%" h={14} />
              <Skeleton w="60%" h={10} />
              <Skeleton w={120} h={30} />
            </div>
          ))}
        </section>
      </div>
      <section className="card mcp-aside">
        <Skeleton w="60%" h={16} />
        <Skeleton w="90%" h={10} />
        <Skeleton w="80%" h={10} />
      </section>
    </div>
  )
}
