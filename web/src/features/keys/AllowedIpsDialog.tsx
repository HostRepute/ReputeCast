import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useMe, useWorkspace } from '@/lib/session'
import { ipCovered, ipEntries, type IpEntry } from './ip'
import { m } from './messages'
import type { Key } from './parts'

/** A refused line of the field: "Line 2: Enter an address or a network…". */
export type IpError = { line: number; message: string }

/** The body.allowedIps[i] errors of a 422, on the lines they came from; undefined when the problem isn't the field's. */
export function ipErrors(e: unknown, sent: IpEntry[]): IpError[] | undefined {
  if (!(e instanceof ApiError)) return undefined
  const out: IpError[] = []
  for (const d of e.problem.errors ?? []) {
    const at = /^body\.allowedIps\[(\d+)\]$/.exec(d.location ?? '')?.[1]
    if (at === undefined) continue
    out.push({ line: sent[Number(at)]?.line ?? Number(at) + 1, message: d.message ?? errorMessage(e) })
  }
  return out.length ? out : undefined
}

const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** "Add this address" after an address the field can take. */
function AddButton({ ip, onAdd }: { ip: string; onAdd: (ip: string) => void }) {
  const t = useT()
  return (
    <button
      type="button"
      className="inline-flex min-h-6 items-center font-semibold text-accent underline underline-offset-2 max-md:min-h-11"
      aria-label={t(m.allowedIpsAddNamed, { ip })}
      onClick={() => onAdd(ip)}
    >
      {t(m.allowedIpsAdd)}
    </button>
  )
}

/**
 * The allowed IPs field (SettingsKeys create dialog, SettingsKeysStates
 * "Allowed IPs" dialog): one address or network a line, the address
 * ReputeCast sees for the person editing, where the key was last used, and a
 * warning while an untrusted proxy may hide the callers' addresses.
 */
export function AllowedIpsField({
  value,
  onChange,
  errors,
  list,
  lastUsedIp,
  optional,
}: {
  value: string
  onChange: (v: string) => void
  errors?: IpError[]
  list: Pick<Schemas['APIKeyList'], 'yourIp' | 'proxyWarning'>
  lastUsedIp?: string
  optional?: boolean
}) {
  const t = useT()
  const id = useId()
  const { slug } = useWorkspace()
  const admin = useMe().data?.user.isInstanceAdmin
  // An address already inside a listed network needs no "Add this address".
  const listed = ipEntries(value).map((e) => e.value)
  const add = (ip: string) => onChange(value.trim() ? `${value.replace(/\s+$/, '')}\n${ip}` : ip)
  const errId = errors?.length ? `${id}-err` : undefined
  return (
    <div className="col gap-2">
      {list.proxyWarning ? (
        <Note tone="warn">
          {t(m.allowedIpsProxy)}
          {admin ? (
            <>
              {' '}
              <Link to="/w/$workspace/admin/instance" params={{ workspace: slug }}>
                {t(m.allowedIpsProxyLink)}
              </Link>
            </>
          ) : null}
        </Note>
      ) : null}
      <div className="lbl">
        <label htmlFor={id}>{optional ? <T msg={m.allowedIpsOptional} values={{ opt: (c) => <span className="k font-medium">{c}</span> }} /> : t(m.allowedIpsLabel)}</label>
        <textarea
          id={id}
          className="in mono"
          rows={3}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={'203.0.113.7\n198.51.100.0/24'}
          spellCheck={false}
          autoComplete="off"
          autoCapitalize="off"
          aria-invalid={errId ? true : undefined}
          aria-describedby={[errId, `${id}-hint`].filter(Boolean).join(' ')}
        />
        {errors?.length ? (
          <ul className="err m-0 list-none p-0" id={errId} role="alert">
            {errors.map((e) => (
              <li key={`${e.line}-${e.message}`}>{t(m.allowedIpsLineError, { line: e.line, message: e.message })}</li>
            ))}
          </ul>
        ) : null}
        <span className="k" id={`${id}-hint`}>
          {t(m.allowedIpsHint)}
        </span>
        {lastUsedIp ? (
          <span className="k">
            <T msg={m.allowedIpsLastUsed} values={{ ip: lastUsedIp, mono }} />
            {ipCovered(listed, lastUsedIp) ? null : (
              <>
                {' · '}
                <AddButton ip={lastUsedIp} onAdd={add} />
              </>
            )}
          </span>
        ) : null}
        <span className="k">
          {list.yourIp ? <T msg={m.allowedIpsYourIp} values={{ ip: list.yourIp, mono }} /> : t(m.allowedIpsYourIpUnknown)}
          {list.yourIp && !ipCovered(listed, list.yourIp) && list.yourIp !== lastUsedIp ? (
            <>
              {' · '}
              <AddButton ip={list.yourIp} onAdd={add} />
            </>
          ) : null}
        </span>
      </div>
    </div>
  )
}

/**
 * "Allowed IPs…" of a key or agent (SettingsKeysStates): the field, saved
 * with its own call. Saving a list that leaves out where the key was last
 * used asks first, as those calls stop at once.
 */
export function AllowedIpsDialog({ k, list, onClose }: { k: Key; list: Pick<Schemas['APIKeyList'], 'yourIp' | 'proxyWarning'>; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const agent = k.kind === 'mcp_agent'
  const kind = agent ? 'agent' : 'key'
  const [text, setText] = useState(() => k.allowedIps.join('\n'))
  const [errors, setErrors] = useState<IpError[]>()
  const [failed, setFailed] = useState<string>()
  const [saving, setSaving] = useState(false)
  const [confirming, setConfirming] = useState(false)

  const save = async () => {
    const sent = ipEntries(text)
    const req = { params: { path: { workspace: slug, keyId: k.id } }, body: { allowedIps: sent.map((e) => e.value) } }
    setErrors(undefined)
    setFailed(undefined)
    setSaving(true)
    try {
      if (agent) await http.PUT('/api/v1/workspaces/{workspace}/mcp-agents/{keyId}/allowed-ips', req)
      else await http.PUT('/api/v1/workspaces/{workspace}/api-keys/{keyId}/allowed-ips', req)
    } catch (e) {
      const lines = ipErrors(e, sent)
      if (lines) setErrors(lines)
      else setFailed(errorMessage(e))
      return
    } finally {
      setSaving(false)
    }
    void qc.invalidateQueries({ queryKey: ['get', agent ? '/api/v1/workspaces/{workspace}/mcp-agents' : '/api/v1/workspaces/{workspace}/api-keys'] })
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/mcp'] })
    toast({ kind: 'success', title: tt(m.allowedIpsSaved), detail: tt(m.loggedAs, { action: `${agent ? 'agent' : 'apikey'}.updated` }) })
    onClose()
  }

  const submit = () => {
    const values = ipEntries(text).map((e) => e.value)
    // Where the key was last used stops working at once: ask first.
    if (k.lastUsedIp && values.length > 0 && !ipCovered(values, k.lastUsedIp)) setConfirming(true)
    else void save()
  }

  return (
    <>
      <Dialog
        open={!confirming}
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">{t(m.allowedIpsEyebrow)}</span>}
        title={t(m.allowedIpsTitle, { name: k.name })}
        description={<span className="text-[13.5px]">{t(m.allowedIpsDesc, { kind })}</span>}
        dismissible={!saving}
        footer={
          <>
            <Button onClick={onClose} disabled={saving}>
              {t(common.cancel)}
            </Button>
            <Button variant="primary" loading={saving} onClick={submit}>
              {t(common.save)}
            </Button>
          </>
        }
      >
        <AllowedIpsField value={text} onChange={setText} errors={errors} list={list} lastUsedIp={k.lastUsedIp} />
        {failed ? (
          <Note tone="bad" label={t(m.allowedIpsNotSaved)}>
            {failed}
          </Note>
        ) : null}
      </Dialog>
      <ConfirmDialog
        open={confirming}
        onOpenChange={(o) => !o && setConfirming(false)}
        tone="warn"
        badge={t(m.lockoutBadge)}
        title={t(m.lockoutTitle, { ip: k.lastUsedIp ?? '' })}
        description={t(m.lockoutText, { kind, ip: k.lastUsedIp ?? '' })}
        confirmLabel={t(m.saveAnyway)}
        cancelLabel={t(common.back)}
        onConfirm={() => {
          setConfirming(false)
          return save()
        }}
      />
    </>
  )
}
