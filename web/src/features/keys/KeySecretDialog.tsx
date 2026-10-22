import { useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useMe } from '@/lib/session'
import { copyText } from '../access/parts'
import { m } from './messages'
import { keySummary } from './parts'

const mono = (c: ReactNode) => <span className="mono">{c}</span>

/**
 * The one time a key's secret is shown (SettingsKeysStates, "Copy your new
 * key now"). Done stays disabled until the person confirms they copied it.
 */
export function KeySecretDialog({ created, action, onClose }: { created: Schemas['APIKeyCreated']; action: 'created' | 'rotated' | 'replaced'; onClose: () => void }) {
  const t = useT()
  const me = useMe().data
  const [copied, setCopied] = useState(false)
  const k = created.key
  const agent = k.kind === 'mcp_agent'
  const kind = agent ? 'agent' : 'key'
  const endpoint = agent ? created.endpoint.replace(/\/api\/v1$/, '/mcp') : created.endpoint
  const prefix = created.secret.slice(0, created.secret.indexOf('_', 3) + 1) || 'rc_'
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && copied && onClose()}
      eyebrow={<span className="k">{t(m.secretEyebrow, { kind, action })}</span>}
      title={agent ? t(m.secretTitleAgent) : t(m.secretTitleKey)}
      description={<span className="text-[13.5px]">{t(m.secretDesc, { kind, summary: keySummary(k) })}</span>}
      hideClose
      dismissible={false}
      footer={
        <>
          <label className="row mr-auto gap-2 text-[13px]">
            <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
            {t(m.secretCopiedCheck, { kind })}
          </label>
          <Button variant="primary" disabledReason={copied ? undefined : t(m.secretConfirm, { kind })} onClick={onClose}>
            {t(common.done)}
          </Button>
        </>
      }
    >
      <div className="row flex-nowrap gap-2">
        <input
          className="in mono flex-1 text-[12.5px]"
          value={created.secret}
          readOnly
          aria-label={agent ? t(m.secretInputAgent) : t(m.secretInputKey)}
          onFocus={(e) => e.currentTarget.select()}
        />
        <Button
          variant="primary"
          onClick={async () => {
            const ok = await copyText(created.secret)
            if (ok) setCopied(true)
            toast({ kind: ok ? 'success' : 'error', title: ok ? tt(common.copied) : tt(m.copyFailed), detail: ok ? tt(m.copiedDetail) : tt(m.copyFailedDetail) })
          }}
        >
          {t(common.copy)}
        </Button>
      </div>
      <Note tone="warn" label={t(m.shownOnce)}>
        {t(m.secretStore, { kind })}
      </Note>
      <KeyValues
        className="text-[12.5px]"
        items={[
          [t(m.useIt), <T msg={m.useItValue} values={{ header: `Authorization: Bearer ${prefix}…`, endpoint, mono }} />],
          [t(m.logged), `${agent ? 'agent' : 'apikey'}.${action}${me ? ` · ${me.user.name}` : ''} · ${t(m.loggedActor, { kind })}`],
        ]}
      />
    </Dialog>
  )
}
