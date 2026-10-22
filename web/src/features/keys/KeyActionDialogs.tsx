import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ConfirmDialog, DialogOption, DialogRow } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, count, monthDay, usDate } from '@/lib/format'
import { m as fm } from '@/lib/format.messages'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { keySummary, type Key } from './parts'

export type KeyAction = 'rotate' | 'revoke' | 'delete' | 'replace'

function useKeyCalls(k: Key) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const agent = k.kind === 'mcp_agent'
  const params = { params: { path: { workspace: slug, keyId: k.id } } }
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', agent ? '/api/v1/workspaces/{workspace}/mcp-agents' : '/api/v1/workspaces/{workspace}/api-keys'] })
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/mcp'] })
  }
  return {
    refresh,
    rotate: async () =>
      (agent
        ? await http.POST('/api/v1/workspaces/{workspace}/mcp-agents/{keyId}/rotate', params)
        : await http.POST('/api/v1/workspaces/{workspace}/api-keys/{keyId}/rotate', params)
      ).data!,
    revoke: async () => {
      if (agent) await http.POST('/api/v1/workspaces/{workspace}/mcp-agents/{keyId}/revoke', params)
      else await http.POST('/api/v1/workspaces/{workspace}/api-keys/{keyId}/revoke', params)
    },
    remove: async () => {
      if (agent) await http.DELETE('/api/v1/workspaces/{workspace}/mcp-agents/{keyId}', params)
      else await http.DELETE('/api/v1/workspaces/{workspace}/api-keys/{keyId}', params)
    },
  }
}

/**
 * Rotate, revoke or delete a key or agent token. Rotating and replacing show
 * the new secret once (onSecret); revoking stops it at once.
 */
export function KeyActionDialog({
  k,
  action,
  onClose,
  onSecret,
}: {
  k: Key
  action: KeyAction
  onClose: () => void
  onSecret: (c: Schemas['APIKeyCreated'], action: 'rotated' | 'replaced') => void
}) {
  const t = useT()
  const calls = useKeyCalls(k)
  // Revoking offers a replacement with the same scopes, checked by default (Dialogs board).
  const [replace, setReplace] = useState(true)
  const agent = k.kind === 'mcp_agent'
  const kind = agent ? 'agent' : 'key'
  if (action === 'replace') return <ExpiredKeyDialog k={k} onClose={onClose} onSecret={onSecret} />
  const run = async () => {
    if (action === 'rotate' || (action === 'revoke' && replace)) {
      // Rotating revokes the key and creates its replacement in one step.
      const c = await calls.rotate()
      calls.refresh()
      onSecret(c, action === 'rotate' ? 'rotated' : 'replaced')
      return
    }
    if (action === 'revoke') await calls.revoke()
    else await calls.remove()
    calls.refresh()
    toast({
      kind: 'deleted',
      title: tt(action === 'revoke' ? m.revokedToast : m.deletedToast, { name: k.name }),
      detail: tt(m.loggedAs, { action: `${agent ? 'agent' : 'apikey'}.${action === 'revoke' ? 'revoked' : 'deleted'}` }),
    })
  }
  const copy = {
    rotate: {
      tone: 'warn' as const,
      badge: t(m.rotateBadge, { kind }),
      title: t(m.rotateTitle, { name: k.name }),
      text: t(m.rotateText, { kind }),
      button: t(m.rotateConfirm, { kind }),
      busy: t(m.rotating),
    },
    revoke: {
      tone: 'warn' as const,
      badge: t(m.revokeBadge, { kind }),
      title: t(m.revokeTitle, { name: k.name }),
      text: t(m.revokeText, { kind, lastUse: lastUse(k) }),
      button: t(m.revokeConfirm, { kind }),
      busy: t(m.revoking),
    },
    delete: {
      tone: 'bad' as const,
      badge: t(m.deleteBadge, { kind }),
      title: t(m.deleteTitle, { name: k.name }),
      text: t(m.deleteText, { kind }),
      button: t(m.deleteConfirm, { kind }),
      busy: t(m.deleting),
    },
  }[action]
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone={copy.tone}
      badge={copy.badge}
      title={copy.title}
      description={copy.text}
      confirmLabel={copy.button}
      confirmVariant={action === 'rotate' ? 'primary' : 'danger'}
      busyLabel={copy.busy}
      onConfirm={run}
    >
      {action === 'revoke' ? (
        <DialogOption type="checkbox" checked={replace} onChange={setReplace} title={t(m.replaceOption, { kind })} />
      ) : (
        <DialogRow tone="neutral" label={agent ? t(m.rowAgent) : t(m.rowKey)} first>
          {keySummary(k)}
        </DialogRow>
      )}
    </ConfirmDialog>
  )
}

/** "This key was last used 4 min ago from 198.51.100.12." (an agent's says "This token"). */
function lastUse(k: Key): string {
  const kind = k.kind === 'mcp_agent' ? 'agent' : 'key'
  if (!k.lastUsedAt) return tt(m.lastUseNever, { kind })
  const ago_ = ago(k.lastUsedAt)
  const when = ago_ === tt(fm.now) ? tt(m.justNow) : ago_
  return k.lastUsedIp ? tt(m.lastUseFrom, { kind, when, ip: k.lastUsedIp }) : tt(m.lastUse, { kind, when })
}

const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** An expired key that something still calls (SettingsKeysStates, "Expired key"). */
function ExpiredKeyDialog({ k, onClose, onSecret }: { k: Key; onClose: () => void; onSecret: (c: Schemas['APIKeyCreated'], action: 'rotated' | 'replaced') => void }) {
  const t = useT()
  const calls = useKeyCalls(k)
  const [busy, setBusy] = useState<'replace' | 'delete'>()
  const [error, setError] = useState<string>()
  const r = k.rejections
  const replace = async () => {
    setBusy('replace')
    try {
      const c = await calls.rotate()
      calls.refresh()
      onSecret(c, 'replaced')
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(undefined)
    }
  }
  const remove = async () => {
    setBusy('delete')
    try {
      await calls.remove()
      calls.refresh()
      toast({ kind: 'deleted', title: tt(m.deletedToast, { name: k.name }) })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<Badge tone="bad">{t(m.expiredEyebrow)}</Badge>}
      title={t(m.expiredTitle)}
      description={<span className="text-[13.5px]">{k.expiresAt ? t(m.expiredDesc, { name: k.name, date: usDate(k.expiresAt) }) : k.name}</span>}
      hideClose
      footer={
        <>
          <Button onClick={onClose}>{t(common.close)}</Button>
        </>
      }
    >
      {r && r.calls > 0 ? (
        <Note tone="bad" label="401">
          <T
            msg={m.rejected}
            values={{
              mono,
              hasSince: r.since ? 'yes' : 'no',
              since: r.since ? monthDay(r.since) : '',
              calls: r.calls,
              callsText: count(r.calls),
              hasIp: r.ip ? 'yes' : 'no',
              ip: r.ip ?? '',
              hasClient: r.client ? 'yes' : 'no',
              client: r.client ?? '',
            }}
          />
        </Note>
      ) : (
        <Note tone="neutral">{t(m.noCallsSinceExpired)}</Note>
      )}
      <KeyValues
        className="text-[12.5px]"
        items={[
          [t(m.kvPermissions), k.permissions.join(', ')],
          [t(m.kvCreated), t(m.createdByOn, { date: usDate(k.createdAt), name: k.createdBy.name })],
          [t(m.kvLastCall), k.lastUsedAt ? usDate(k.lastUsedAt) : t(m.lastCallNever)],
        ]}
      />
      <div className="row gap-2">
        <Button size="sm" variant="primary" loading={busy === 'replace'} onClick={() => void replace()}>
          {t(m.replaceSame)}
        </Button>
        <Button size="sm" loading={busy === 'delete'} onClick={() => void remove()}>
          {t(common.delete)}
        </Button>
      </div>
      <span className="k">{t(m.expiredKept)}</span>
      {error ? (
        <Note tone="bad" label={t(m.notDone)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}
