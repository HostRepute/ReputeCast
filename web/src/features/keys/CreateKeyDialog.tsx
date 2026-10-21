import { useQueryClient } from '@tanstack/react-query'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge, SensitiveFlag } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { T, useT, type MessageDescriptor, type Values } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

type List = Schemas['APIKeyList']
export type KeyKind = 'api_key' | 'mcp_agent'

/** "Start from" (SettingsKeysStates): a preset fills the permissions; Send-only opens the transactional key's own dialog. */
const PRESETS: { key: string; label: MessageDescriptor; permissions?: string[] }[] = [
  { key: 'custom', label: m.presetCustom, permissions: [] },
  { key: 'send_only', label: m.presetSendOnly },
  { key: 'signup', label: m.presetSignup, permissions: ['subscribers:create'] },
  { key: 'analytics', label: m.presetAnalytics, permissions: ['reports:view', 'subscribers:view'] },
  // Earn › Settings › Attribution: a key for the conversion API only.
  { key: 'conversions', label: m.presetConversions, permissions: ['conversions:create'] },
]

/** A conversions key holds conversions:create and nothing else. */
const conversionsOnly = (picked: string[]) => picked.length === 1 && picked[0] === 'conversions:create'

const EXPIRY: { value: string; label: MessageDescriptor; values?: Values }[] = [
  { value: '', label: m.expiryNever },
  { value: '30', label: m.expiryDays, values: { days: 30 } },
  { value: '90', label: m.expiryDays, values: { days: 90 } },
  { value: '365', label: m.expiryYears, values: { years: 1 } },
]

/** The "· optional" note after a field's label. */
const opt = (c: ReactNode) => <span className="k font-medium">{c}</span>

/**
 * Create an API key or an MCP agent token (SettingsKeysStates, escalation
 * guard): a name, the permissions (never more than the creator holds; locked
 * rows say why and who can help), a list scope and an expiry.
 */
export function CreateKeyDialog({
  kind,
  list,
  onClose,
  onCreated,
  onSendOnly,
  initialPreset,
}: {
  kind: KeyKind
  list: List
  onClose: () => void
  onCreated: (c: Schemas['APIKeyCreated']) => void
  /** Opens the send-only transactional key's dialog instead (the Send-only preset). */
  onSendOnly?: () => void
  /** The preset to start from (Earn's "Create a conversions key" opens on Conversions key). */
  initialPreset?: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const agent = kind === 'mcp_agent'
  const catalog = useMemo(() => list.permissions.filter((p) => !p.ownerOnly), [list.permissions])
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [picked, setPicked] = useState<string[]>(() => {
    const perms = PRESETS.find((p) => p.key === initialPreset)?.permissions ?? []
    return perms.filter((k) => catalog.some((c) => c.key === k) && !list.locked.some((l) => l.key === k))
  })
  const [listId, setListId] = useState('')
  const [expiry, setExpiry] = useState(initialPreset === 'conversions' ? '365' : '')
  const [preset, setPreset] = useState(initialPreset ?? 'custom')
  const conversionKey = preset === 'conversions' && conversionsOnly(picked)
  const [error, setError] = useState<{ field?: string; message: string }>()
  const createKey = $api.useMutation('post', '/api/v1/workspaces/{workspace}/api-keys')
  const createAgent = $api.useMutation('post', '/api/v1/workspaces/{workspace}/mcp-agents')
  const create = agent ? createAgent : createKey
  const locked = new Map(list.locked.map((l) => [l.key, l.reason]))
  const includedBy = (key: string) => catalog.find((p) => p.key !== key && picked.includes(p.key) && p.implies.includes(key))?.key
  const groups = list.groups.map((g) => ({ g, rows: catalog.filter((p) => p.group === g) })).filter((x) => x.rows.length > 0)

  const submit = () => {
    setError(undefined)
    if (!name.trim()) {
      setError({ field: 'name', message: t(m.nameRequired, { kind: agent ? 'agent' : 'key' }) })
      return
    }
    if (picked.length === 0) {
      setError({ field: 'permissions', message: t(m.permissionsRequired) })
      return
    }
    const expiresAt = expiry ? new Date(Date.now() + Number(expiry) * 86_400_000).toISOString() : undefined
    const body = { name: name.trim(), description: description.trim() || undefined, permissions: picked, lists: listId ? [listId] : undefined, expiresAt }
    const handlers = {
      onSuccess: (r: Schemas['APIKeyCreated']) => {
        void qc.invalidateQueries({ queryKey: ['get', agent ? '/api/v1/workspaces/{workspace}/mcp-agents' : '/api/v1/workspaces/{workspace}/api-keys'] })
        void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/mcp'] })
        onCreated(r)
      },
      onError: (e: unknown) => {
        const loc = e instanceof ApiError ? e.problem.errors?.[0]?.location : undefined
        setError({ field: loc === 'body.name' ? 'name' : loc?.startsWith('body.permissions') ? 'permissions' : undefined, message: errorMessage(e) })
      },
    }
    if (agent) createAgent.mutate({ params: { path: { workspace: slug } }, body }, handlers)
    else createKey.mutate({ params: { path: { workspace: slug } }, body }, handlers)
  }

  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{agent ? t(m.createEyebrowAgent) : t(m.createEyebrowKey)}</span>}
      title={agent ? t(m.createTitleAgent) : t(m.createTitleKey)}
      description={<span className="text-[13.5px]">{agent ? t(m.createDescAgent) : t(m.createDescKey)}</span>}
      size="lg"
      hideClose
      dismissible={false}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={create.isPending} onClick={submit}>
            {agent ? t(m.submitAgent) : t(m.submitKey)}
          </Button>
        </>
      }
    >
      {list.locked.length > 0 ? (
        <span>
          <Badge tone="warn">{t(m.lockedCount, { count: count(list.locked.length) })}</Badge>
        </span>
      ) : null}
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          {t(m.nameLabel)}
          <input
            className="in"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
            placeholder={agent ? t(m.namePlaceholderAgent) : t(m.namePlaceholderKey)}
            aria-invalid={error?.field === 'name' || undefined}
            aria-describedby={error?.field === 'name' ? 'key-name-err' : undefined}
          />
          {error?.field === 'name' ? (
            <span className="err" id="key-name-err" role="alert">
              {error.message}
            </span>
          ) : null}
        </label>
        <label className="lbl">
          <span>
            <T msg={agent ? m.purposeLabel : m.descriptionLabel} values={{ opt }} />
          </span>
          <input
            className="in"
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            maxLength={200}
            placeholder={agent ? 'Claude Desktop' : t(m.descriptionPlaceholder)}
          />
        </label>
      </div>
      {agent ? null : (
        <div className="col gap-1.5">
          <span className="text-[12px] font-semibold text-ink-2" id="key-preset">
            {t(m.startFrom)}
          </span>
          <div className="row gap-1.5" role="group" aria-labelledby="key-preset">
            {PRESETS.filter((p) => p.key !== 'send_only' || onSendOnly).map((p) => {
              const on = preset === p.key
              return (
                <button
                  key={p.key}
                  type="button"
                  className="btn sm"
                  aria-pressed={on}
                  style={on ? { background: '#0B1220', color: '#fff', borderColor: '#0B1220' } : undefined}
                  onClick={() => {
                    if (p.key === 'send_only') {
                      onSendOnly?.()
                      return
                    }
                    setPreset(p.key)
                    // Only what the creator may grant: locked permissions stay off.
                    setPicked((p.permissions ?? []).filter((k) => catalog.some((c) => c.key === k) && !locked.has(k)))
                    if (p.key === 'conversions') {
                      setExpiry('365')
                      setListId('')
                    }
                  }}
                >
                  {t(p.label)}
                </button>
              )
            })}
          </div>
          <span className="k">{t(m.presetHint)}</span>
        </div>
      )}
      {conversionKey ? (
        <div className="rounded-lg border border-line bg-[#f7f8fa] px-3 py-2.5 text-[13.5px]">
          <T
            msg={m.conversionsOnly}
            values={{
              permission: 'conversions:create',
              mono: (c) => <span className="mono rounded bg-[#eceef2] px-1.5 py-0.5 text-[12.5px]">{c}</span>,
            }}
          />
        </div>
      ) : (
        <div className="col gap-1.5" role="group" aria-labelledby="key-perms">
          <span className="text-[12px] font-semibold text-ink-2" id="key-perms">
            {t(m.permissionsLabel)}
          </span>
          <div className="col max-h-[340px] gap-1.5 overflow-y-auto pr-1">
            {groups.map(({ g, rows }) => (
              <div key={g} className="col gap-1.5" role="group" aria-label={g}>
                <span className="pt-1 text-[11px] font-bold uppercase tracking-[.04em] text-ink-3">{g}</span>
                {rows.map((p) => {
                  const by = includedBy(p.key)
                  const reason = locked.get(p.key)
                  const on = picked.includes(p.key) || Boolean(by)
                  const id = `k-${p.key.replace(':', '-')}`
                  const disabled = Boolean(reason) || Boolean(by)
                  return (
                    <div key={p.key} className="row flex-nowrap gap-3 rounded-lg border border-line px-2.5 max-sm:flex-wrap max-sm:gap-y-0">
                      <input
                        type="checkbox"
                        id={id}
                        checked={on}
                        disabled={disabled}
                        aria-describedby={reason || by ? `${id}-d` : undefined}
                        onChange={(e) => setPicked((cur) => (e.target.checked ? [...cur, p.key] : cur.filter((k) => k !== p.key)))}
                      />
                      <label htmlFor={id} className={cn('flex min-w-0 flex-1 flex-col py-2 text-[13.5px]', !disabled && 'cursor-pointer')}>
                        <span>
                          {p.label}
                          {p.sensitive ? (
                            <>
                              {' '}
                              <span title={t(m.sensitiveTitle)}>
                                <SensitiveFlag />
                              </span>
                            </>
                          ) : null}
                        </span>
                        <span className="k mono">{p.key}</span>
                      </label>
                      {reason ? (
                        <span className="k max-w-[240px] whitespace-normal text-bad-fg!" id={`${id}-d`}>
                          {reason}
                        </span>
                      ) : by ? (
                        <span className="k" id={`${id}-d`}>
                          {t(m.includedBy, { key: by })}
                        </span>
                      ) : null}
                    </div>
                  )
                })}
              </div>
            ))}
          </div>
          {error?.field === 'permissions' ? (
            <span className="err" role="alert">
              {error.message}
            </span>
          ) : null}
        </div>
      )}
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        {conversionKey ? null : (
          <label className="lbl">
            <span>
              <T msg={m.listScopeLabel} values={{ opt }} />
            </span>
            <select className="in max-w-[260px]" value={listId} onChange={(e) => setListId(e.target.value)}>
              <option value="">{t(m.allListsOption)}</option>
              {list.lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </label>
        )}
        <label className="lbl">
          {t(m.expiresLabel)}
          <select className="in max-w-[260px]" value={expiry} onChange={(e) => setExpiry(e.target.value)}>
            {EXPIRY.map((x) => (
              <option key={x.value} value={x.value}>
                {t(x.label, x.values)}
              </option>
            ))}
          </select>
        </label>
      </div>
      {error && !error.field ? (
        <Note tone="bad" label={t(m.notCreated)}>
          {error.message}
        </Note>
      ) : null}
    </Dialog>
  )
}
