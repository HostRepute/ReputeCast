import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { T, useT } from '@/i18n'
import { ago, createdTime, lastUsed, usDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { Footer } from './ConnectDialog'
import { m } from './messages'
import { base, entryName, isPlatform, ManagedByInner, stamp, tags, useRefreshHub, type EntryKey } from './shared'

type ManagedKey = Schemas['IntegrationManagedKey']

/** Where a managed key's integration lives: Zapier's, Make's, n8n's page, or the WordPress sites. */
function IntegrationLink({ kind, className, children, label }: { kind: string; className?: string; children: React.ReactNode; label?: string }) {
  const { slug } = useWorkspace()
  return isPlatform(kind) ? (
    <Link to="/w/$workspace/settings/integrations/$kind" params={{ workspace: slug, kind }} className={className} aria-label={label}>
      {children}
    </Link>
  ) : (
    <Link to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} className={className} aria-label={label}>
      {children}
    </Link>
  )
}

/** "Managed by Zapier", linking to the integration (SettingsKeys, SettingsAPI). */
export function ManagedBy({ kind }: { kind: string }) {
  const t = useT()
  const name = t(entryName[kind as EntryKey] ?? m.title)
  return (
    <IntegrationLink kind={kind} className="ih-mby">
      <ManagedByInner kind={kind} label={t(m.managedBy, { name })} />
    </IntegrationLink>
  )
}

/**
 * Keys & MCP › Managed by integrations (SettingsKeys): each integration's
 * key, with Open (its page) and Revoke, which warns with the workflows
 * that stop.
 */
export function ManagedKeysCard() {
  const { slug } = useWorkspace()
  const t = useT()
  const now = useNow()
  const q = $api.useQuery('get', `${base}/keys`, { params: { path: { workspace: slug } } })
  const [revoke, setRevoke] = useState<ManagedKey>()
  const items = q.data?.items ?? []
  if (!q.data || items.length === 0) return null
  const scope = (k: ManagedKey) => (k.kind === 'wordpress' ? t(m.scopeSite) : t(m.scopeOwnWebhooks, { scope: k.scope.all || !k.scope.label ? t(m.allLists) : k.scope.label }))
  const used = (k: ManagedKey) => (k.lastUsedAt ? lastUsed(k.lastUsedAt, now) : t(m.never))
  return (
    <section className="card" aria-labelledby="managed-keys-h">
      <CardSection
        title={<span id="managed-keys-h">{t(m.keysTitle, { count: items.length })}</span>}
        caption={t(m.keysSub)}
        actions={
          <ButtonLink size="sm" to="/w/$workspace/settings/integrations" params={{ workspace: slug }}>
            {t(m.title)}
          </ButtonLink>
        }
      />
      <div className="ih-no-phone">
        <AccessTable caption={t(m.keysTitle, { count: items.length })}>
          <thead>
            <tr>
              <Th>{t(m.key)}</Th>
              <Th>{t(m.colPermsScope)}</Th>
              <Th>{t(m.colCreatedBy)}</Th>
              <Th>{t(m.lastUsed)}</Th>
              <Th>{t(m.expires)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {items.map((k) => (
              <tr key={k.id}>
                <Td>
                  <div className="col gap-1">
                    <span className="font-semibold">{k.name}</span>
                    <span className="k mono">{k.masked}</span>
                    <div className="row">
                      <ManagedBy kind={k.kind} />
                    </div>
                  </div>
                </Td>
                <Td wrap>
                  <div className="col gap-1">
                    <div className="row gap-1.5">
                      {k.permissions.map((p) => (
                        <span key={p} className="row gap-1">
                          <Badge tone="neutral" className="mono font-medium!">
                            {p}
                          </Badge>
                          {k.sensitive.includes(p) ? (
                            <span title={t(m.sensitivePermission)}>
                              <SensitiveFlag>{t(m.sensitive)}</SensitiveFlag>
                            </span>
                          ) : null}
                        </span>
                      ))}
                    </div>
                    <span className="k">{scope(k)}</span>
                  </div>
                </Td>
                <Td>
                  <PersonCell name={k.owner.name} sub={createdTime(k.createdAt, now)} color={k.owner.avatarColor as never} />
                </Td>
                <Td>
                  <div className="col gap-0">
                    <span>{used(k)}</span>
                    {k.lastUsedIp ? <span className="k mono">{k.lastUsedIp}</span> : null}
                  </div>
                </Td>
                <Td>{k.expiresAt ? usDate(k.expiresAt) : <span className="k">{t(m.never)}</span>}</Td>
                <Td>
                  <KeyButtons k={k} onRevoke={() => setRevoke(k)} />
                </Td>
              </tr>
            ))}
          </tbody>
        </AccessTable>
      </div>
      <div className="ih-only-phone ih-list">
        {items.map((k) => (
          <div key={k.id} className="ih-it">
            <div className="row justify-between gap-2">
              <span className="font-semibold">{k.name}</span>
              <ManagedBy kind={k.kind} />
            </div>
            <span className="k">
              <T msg={m.keyPhoneLine} values={{ ...tags, masked: k.masked, count: k.permissions.length, scope: scope(k) }} />
            </span>
            <span className="k">{k.lastUsedAt ? t(m.keyPhoneCreated, { name: k.owner.name, ago: ago(k.lastUsedAt) }) : t(m.keyPhoneCreatedNever, { name: k.owner.name })}</span>
            <div>
              <KeyButtons k={k} onRevoke={() => setRevoke(k)} />
            </div>
          </div>
        ))}
      </div>
      {revoke ? <RevokeManagedKeyDialog k={revoke} onClose={() => setRevoke(undefined)} /> : null}
    </section>
  )
}

function KeyButtons({ k, onRevoke }: { k: ManagedKey; onRevoke: () => void }) {
  const t = useT()
  return (
    <div className="row flex-nowrap gap-2">
      <IntegrationLink kind={k.kind} className="btn sm" label={t(m.openEntry, { name: k.name })}>
        {t(m.open)}
      </IntegrationLink>
      {k.status === 'active' || k.status === 'grace' ? (
        <Button size="sm" variant="danger" onClick={onRevoke} aria-label={t(m.revokeNamed, { key: k.name })}>
          {t(m.revoke)}
        </Button>
      ) : null}
    </div>
  )
}

/** Revoking an integration's key from Keys & MCP: the workflows that stop, and Rotate in Integrations instead (IntegrationsStates). */
export function RevokeManagedKeyDialog({ k, onClose }: { k: ManagedKey; onClose: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const navigate = useNavigate()
  const refresh = useRefreshHub()
  const [busy, setBusy] = useState(false)
  const kind = k.kind as EntryKey
  const name = t(entryName[kind] ?? m.title)
  const wp = k.kind === 'wordpress'
  async function revoke() {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/api-keys/{keyId}/revoke', { params: { path: { workspace: slug, keyId: k.id } } })
      await refresh()
      onClose()
      toast({ kind: 'deleted', title: t(m.revokedToast, { key: k.name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.revokeError), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const rotate = () => {
    onClose()
    if (isPlatform(k.kind)) void navigate({ to: '/w/$workspace/settings/integrations/$kind', params: { workspace: slug, kind: k.kind } })
    else void navigate({ to: '/w/$workspace/settings/integrations/wordpress', params: { workspace: slug } })
  }
  const last = k.lastUsedAt ? ago(k.lastUsedAt) : t(m.never)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(m.revokeEyebrow)}</span>}
      title={t(m.revokeTitle, { key: k.name })}
      className="card border-bad-line"
      hideClose
      dismissible={!busy}
    >
      <span className="text-[13.5px] text-ink-nav">
        {wp
          ? t(m.revokeSiteBody, { calls: k.calls7d, ago: last })
          : k.workflows.length
            ? t(m.revokeBody, { calls: k.calls7d, ago: last, name, kind: k.kind })
            : t(m.revokeBodyNone, { calls: k.calls7d, ago: last, name })}
      </span>
      {!wp && k.workflows.length ? (
        <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13.5px]">
          {k.workflows.map((w, i) => (
            <li key={i}>
              <span className="font-semibold">{w.unnamed ? t(m.unnamed, { kind: k.kind }) : w.name}</span> <span className="k">{t(m.lastRun, { time: stamp(w.lastRunAt) })}</span>
            </li>
          ))}
        </ul>
      ) : null}
      <Note tone="info">
        <div>{t(m.rotateInstead)}</div>
        <div className="mt-2">
          <Button size="sm" onClick={rotate}>
            {t(m.rotateInIntegrations)}
          </Button>
        </div>
      </Note>
      <Footer>
        <Button onClick={onClose}>{t(m.cancel)}</Button>
        <Button variant="danger-primary" onClick={revoke} loading={busy}>
          {t(m.revokeKey)}
        </Button>
      </Footer>
    </Dialog>
  )
}
