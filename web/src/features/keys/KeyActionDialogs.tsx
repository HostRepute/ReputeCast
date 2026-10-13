import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ConfirmDialog, DialogRow } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { count, usDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
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
  const calls = useKeyCalls(k)
  const agent = k.kind === 'mcp_agent'
  const noun = agent ? 'agent token' : 'API key'
  if (action === 'replace') return <ExpiredKeyDialog k={k} onClose={onClose} onSecret={onSecret} />
  const run = async () => {
    if (action === 'rotate') {
      const c = await calls.rotate()
      calls.refresh()
      onSecret(c, 'rotated')
      return
    }
    if (action === 'revoke') await calls.revoke()
    else await calls.remove()
    calls.refresh()
    toast({ kind: 'deleted', title: `“${k.name}” ${action === 'revoke' ? 'revoked' : 'deleted'}`, detail: `logged as ${agent ? 'agent' : 'apikey'}.${action === 'revoke' ? 'revoked' : 'deleted'}` })
  }
  const copy = {
    rotate: {
      tone: 'warn' as const,
      badge: `Rotate ${noun}`,
      title: `Rotate “${k.name}”?`,
      text: `A new ${noun} is created with the same permissions and shown once. The current value stops working at once, so update whatever uses it right after.`,
      button: `Rotate ${noun}`,
      busy: 'Rotating…',
    },
    revoke: {
      tone: 'bad' as const,
      badge: `Revoke ${noun}`,
      title: `Revoke “${k.name}”?`,
      text: `Calls with this ${noun} fail with 401 from now on. ${agent ? 'The agent loses access to the MCP server.' : 'Integrations that use it stop working.'} This can't be undone; create a new ${noun} to restore access.`,
      button: `Revoke ${noun}`,
      busy: 'Revoking…',
    },
    delete: {
      tone: 'bad' as const,
      badge: `Delete ${noun}`,
      title: `Delete “${k.name}”?`,
      text: `The expired ${noun} is removed from this list. The audit log keeps every call it made.`,
      button: `Delete ${noun}`,
      busy: 'Deleting…',
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
      busyLabel={copy.busy}
      onConfirm={run}
    >
      <DialogRow tone="neutral" label={agent ? 'Agent' : 'Key'} first>
        {keySummary(k)}
      </DialogRow>
    </ConfirmDialog>
  )
}

/** An expired key that something still calls (SettingsKeysStates, "Expired key"). */
function ExpiredKeyDialog({ k, onClose, onSecret }: { k: Key; onClose: () => void; onSecret: (c: Schemas['APIKeyCreated'], action: 'rotated' | 'replaced') => void }) {
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
      toast({ kind: 'deleted', title: `“${k.name}” deleted` })
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
      eyebrow={<Badge tone="bad">Expired</Badge>}
      title="Expired key"
      description={
        <span className="text-[13.5px]">
          {k.name}
          {k.expiresAt ? ` · expired ${usDate(k.expiresAt)}` : ''}
        </span>
      }
      hideClose
      footer={
        <>
          <Button onClick={onClose}>Close</Button>
        </>
      }
    >
      {r && r.calls > 0 ? (
        <Note tone="bad" label="401">
          Requests with this key have returned <span className="mono">401 key_expired</span>
          {r.since ? ` since ${usDate(r.since).replace(/, \d{4}$/, '')}` : ''}: {count(r.calls)} {r.calls === 1 ? 'call' : 'calls'} in the last 7 days
          {r.ip ? (
            <>
              {' '}
              from <span className="mono">{r.ip}</span>
            </>
          ) : null}
          {r.client ? ` (${r.client})` : ''}. Something still uses it.
        </Note>
      ) : (
        <Note tone="neutral">No calls since it expired.</Note>
      )}
      <KeyValues
        className="text-[12.5px]"
        items={[
          ['Permissions', k.permissions.join(', ')],
          ['Created', `${usDate(k.createdAt)} by ${k.createdBy.name}`],
          ['Last successful call', k.lastUsedAt ? usDate(k.lastUsedAt) : 'Never'],
        ]}
      />
      <div className="row gap-2">
        <Button size="sm" variant="primary" loading={busy === 'replace'} onClick={() => void replace()}>
          Create replacement with the same permissions
        </Button>
        <Button size="sm" loading={busy === 'delete'} onClick={() => void remove()}>
          Delete
        </Button>
      </div>
      <span className="k">Expired keys stay listed for 90 days so you can see what still calls them.</span>
      {error ? (
        <Note tone="bad" label="Not done">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}
