import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useMe } from '@/lib/session'
import { copyText } from '../access/parts'
import { keySummary } from './parts'

/**
 * The one time a key's secret is shown (SettingsKeysStates, "Copy your new
 * key now"). Done stays disabled until the person confirms they copied it.
 */
export function KeySecretDialog({ created, action, onClose }: { created: Schemas['APIKeyCreated']; action: 'created' | 'rotated' | 'replaced'; onClose: () => void }) {
  const me = useMe().data
  const [copied, setCopied] = useState(false)
  const k = created.key
  const agent = k.kind === 'mcp_agent'
  const endpoint = agent ? created.endpoint.replace(/\/api\/v1$/, '/mcp') : created.endpoint
  const prefix = created.secret.slice(0, created.secret.indexOf('_', 3) + 1) || 'rc_'
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && copied && onClose()}
      eyebrow={<span className="k">Dialog · after {action === 'created' ? 'creating' : action === 'rotated' ? 'rotating' : 'replacing'} {agent ? 'an agent token' : 'a key'}</span>}
      title={agent ? 'Copy the agent token now' : 'Copy your new key now'}
      description={
        <span className="text-[13.5px]">
          {keySummary(k)}. This is the only time the full {agent ? 'token' : 'key'} is shown.
        </span>
      }
      hideClose
      dismissible={false}
      footer={
        <>
          <label className="row mr-auto gap-2 text-[13px]">
            <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />I copied the {agent ? 'token' : 'key'}
          </label>
          <Button variant="primary" disabledReason={copied ? undefined : `Confirm that you copied the ${agent ? 'token' : 'key'}`} onClick={onClose}>
            Done
          </Button>
        </>
      }
    >
      <div className="row flex-nowrap gap-2">
        <input
          className="in mono flex-1 text-[12.5px]"
          value={created.secret}
          readOnly
          aria-label={agent ? 'New agent token' : 'New API key'}
          onFocus={(e) => e.currentTarget.select()}
        />
        <Button
          variant="primary"
          onClick={async () => {
            const ok = await copyText(created.secret)
            if (ok) setCopied(true)
            toast({ kind: ok ? 'success' : 'error', title: ok ? 'Copied' : 'Copy failed', detail: ok ? 'store it in your secret manager' : 'select the value and copy it' })
          }}
        >
          Copy
        </Button>
      </div>
      <Note tone="warn" label="Shown once">
        Store it in your secret manager. If you lose it, rotate the {agent ? 'token' : 'key'}: the old value stops working and a new one is shown once.
      </Note>
      <KeyValues
        className="text-[12.5px]"
        items={[
          [
            'Use it',
            <>
              <span className="mono">Authorization: Bearer {prefix}…</span> against <span className="mono">{endpoint}</span>
            </>,
          ],
          ['Logged', `${agent ? 'agent' : 'apikey'}.${action}${me ? ` · ${me.user.name}` : ''} · every call by this ${agent ? 'agent' : 'key'} is logged with the ${agent ? 'agent' : 'key'} as actor`],
        ]}
      />
    </Dialog>
  )
}
