import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, usDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { codeSample, highlight } from './codeSamples'
import { useTemplateVariables } from './OverviewPage'
import { ms } from './setup.messages'
import { CodeBox, type CodeLang } from './shared'

type Created = Schemas['APIKeyCreated']
type Expiry = 'never' | '30' | '90' | '365' | 'date'

const API = '/api/v1/workspaces/{workspace}/transactional' as const

function expiresAt(e: Expiry, date: string): string | undefined {
  if (e === 'never') return undefined
  if (e === 'date') return date ? new Date(`${date}T23:59:59`).toISOString() : undefined
  return new Date(Date.now() + Number(e) * 86_400_000).toISOString()
}

/**
 * Create a send-only key or an SMTP credential (TransactionalSetupStates):
 * name, mode, what it can send and when it expires; then the secret, shown
 * once, with the first call.
 */
export function CreateTransactionalKeyDialog({ preset, setup, onClose }: { preset: 'send_only' | 'smtp'; setup: Schemas['TransactionalSetup']; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const smtp = preset === 'smtp'
  const templates = $api.useQuery('get', `${API}/templates`, { params: { path: { workspace: slug }, query: { period: '7d' } } }, { enabled: !smtp })
  const categories = $api.useQuery('get', `${API}/categories`, { params: { path: { workspace: slug } } }, { enabled: smtp })
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/api-keys')
  const [name, setName] = useState('')
  const [mode, setMode] = useState<'live' | 'test'>('live')
  const [tests, setTests] = useState('')
  const [only, setOnly] = useState(false)
  const [picked, setPicked] = useState<string[]>([])
  const [expiry, setExpiry] = useState<Expiry>(smtp ? 'never' : '365')
  const [date, setDate] = useState('')
  const [error, setError] = useState<string>()
  const [created, setCreated] = useState<Created>()
  const choices = smtp
    ? (categories.data?.items ?? []).map((c) => ({ id: c.id, label: c.name }))
    : (templates.data?.items ?? []).filter((it) => !it.archived).map((it) => ({ id: it.key, label: it.key }))
  const submit = () => {
    if (!name.trim()) return setError(t(ms.nameRequired))
    if (only && picked.length === 0) return setError(smtp ? t(ms.pickCategory) : t(ms.pickTemplate))
    setError(undefined)
    create.mutate(
      {
        params: { path: { workspace: slug } },
        body: {
          name: name.trim(),
          permissions: ['transactional:send'],
          expiresAt: expiresAt(expiry, date),
          transactional: {
            preset,
            mode: smtp ? 'live' : mode,
            templates: !smtp && only ? picked : [],
            categories: smtp && only ? picked : [],
            testAddresses:
              !smtp && mode === 'test'
                ? tests
                    .split(/[,\s]+/)
                    .map((s) => s.trim())
                    .filter(Boolean)
                : [],
          },
        },
      },
      {
        onSuccess: (c) => {
          setCreated(c)
          void qc.invalidateQueries()
        },
        onError: (e) => setError(errorMessage(e)),
      },
    )
  }
  if (created) return <ShownOnce created={created} smtp={smtp} setup={setup} onClose={onClose} />
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(ms.dialogEyebrow)}</span>}
      title={smtp ? t(ms.createSmtpTitle) : t(ms.createTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(ms.cancel)}</Button>
          <Button variant="primary" loading={create.isPending} onClick={submit}>
            {smtp ? t(ms.createSmtp) : t(ms.create)}
          </Button>
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
      <Field label={t(ms.name)}>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={smtp ? t(ms.nameSmtpPlaceholder) : t(ms.namePlaceholder)} maxLength={80} />
      </Field>
      {!smtp ? (
        <div className="lbl">
          <span>{t(ms.mode)}</span>
          <Segmented<'live' | 'test'>
            label={t(ms.mode)}
            value={mode}
            onChange={setMode}
            items={[
              { value: 'live', label: t(ms.modeLive) },
              { value: 'test', label: t(ms.modeTest) },
            ]}
          />
          <span className="k">{t(ms.modeHint)}</span>
        </div>
      ) : null}
      {!smtp && mode === 'test' ? (
        <Field label={t(ms.testAddresses)} hint={t(ms.testAddressesHint)}>
          <Input value={tests} onChange={(e) => setTests(e.target.value)} placeholder={'qa+*@acme.io'} />
        </Field>
      ) : null}
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="mb-1 text-[12px] font-semibold text-ink-2">{smtp ? t(ms.categoryLabel) : t(ms.templatesLabel)}</legend>
        <label className="flex items-center gap-2">
          <input type="radio" name="scope" checked={!only} onChange={() => setOnly(false)} />
          {smtp ? t(ms.categoriesAll) : t(ms.templatesAll)}
        </label>
        <label className="flex items-center gap-2">
          <input type="radio" name="scope" checked={only} onChange={() => setOnly(true)} />
          {smtp ? t(ms.categoriesOnly) : t(ms.templatesOnly)}
        </label>
        {only ? (
          <div className="flex flex-wrap gap-1.5 pl-6">
            {choices.map((c) => {
              const on = picked.includes(c.id)
              return (
                <button
                  key={c.id}
                  type="button"
                  className={`tx-fchip${on ? ' on' : ''}`}
                  aria-pressed={on}
                  onClick={() => setPicked(on ? picked.filter((x) => x !== c.id) : [...picked, c.id])}
                >
                  <span className={smtp ? undefined : 'mono'}>{c.label}</span>
                </button>
              )
            })}
          </div>
        ) : null}
      </fieldset>
      <div className="lbl">
        <label htmlFor="tx-expiry">{t(ms.expires)}</label>
        <select id="tx-expiry" className="in" value={expiry} onChange={(e) => setExpiry(e.target.value as Expiry)}>
          <option value="never">{t(ms.expNever)}</option>
          <option value="30">{t(ms.exp30)}</option>
          <option value="90">{t(ms.exp90)}</option>
          <option value="365">{t(ms.exp365)}</option>
          <option value="date">{t(ms.expDate)}</option>
        </select>
        {expiry === 'date' ? <input className="in" type="date" aria-label={t(ms.expDate)} value={date} onChange={(e) => setDate(e.target.value)} /> : null}
      </div>
      <Note tone="neutral">
        <T msg={ms.onlyPermission} values={{ permission: <span className="mono">{'transactional:send'}</span> }} />
      </Note>
    </Dialog>
  )
}

function ShownOnce({ created, smtp, setup, onClose }: { created: Created; smtp: boolean; setup: Schemas['TransactionalSetup']; onClose: () => void }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const [lang, setLang] = useState<CodeLang>('node')
  const k = created.key
  const tx = k.transactional
  const scope = tx?.templates.length ? tx.templates.join(', ') : tx?.categories.length ? tx.categories.map((c) => c.name).join(', ') : t(ms.scopeAllInline)
  const vars = useTemplateVariables(tx?.templates[0] ?? 'password-reset')
  const code = codeSample(lang, { endpoint: `${setup.baseUrl}/messages`, template: tx?.templates[0] ?? 'password-reset', variables: vars, compact: true })
  const copy = (text: string) =>
    void navigator.clipboard?.writeText(text).then(
      () => {
        setCopied(true)
        toast({ kind: 'success', title: t(ms.copied) })
      },
      () => toast({ kind: 'error', title: text }),
    )
  return (
    <Dialog
      open
      dismissible={false}
      onOpenChange={(o) => !o && copied && onClose()}
      eyebrow={
        <Badge tone="ok" dot>
          {t(ms.createdEyebrow, { time: clock(k.createdAt) })}
        </Badge>
      }
      title={smtp ? t(ms.copyCredentialTitle) : t(ms.copyTitle)}
      description={
        smtp
          ? t(ms.copyCredentialSummary, { name: k.name, scope })
          : t(ms.copySummary, {
              name: k.name,
              permission: 'transactional:send',
              scope,
              expires: k.expiresAt ? t(ms.expiresOn, { date: usDate(k.expiresAt) }) : t(ms.neverExpires),
            })
      }
      footer={
        <>
          <label className="mr-auto flex items-center gap-2">
            <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
            {t(ms.iCopied)}
          </label>
          <Button variant="primary" disabledReason={copied ? undefined : t(ms.iCopied)} onClick={onClose}>
            {t(ms.done)}
          </Button>
        </>
      }
    >
      {smtp ? (
        <dl className="kv">
          <dt>{t(ms.username)}</dt>
          <dd className="flex items-center gap-2">
            <span className="mono" style={{ overflowWrap: 'anywhere' }}>
              {tx?.username ?? k.id}
            </span>
            <Button size="sm" icon="copy" onClick={() => copy(tx?.username ?? k.id)}>
              {t(ms.copy)}
            </Button>
          </dd>
          <dt>{t(ms.password)}</dt>
          <dd className="flex items-center gap-2">
            <span className="mono" style={{ overflowWrap: 'anywhere' }}>
              {created.secret}
            </span>
            <Button size="sm" icon="copy" onClick={() => copy(created.secret)}>
              {t(ms.copy)}
            </Button>
          </dd>
        </dl>
      ) : (
        <div className="flex items-center gap-2 rounded-lg border border-line px-3 py-2.5" style={{ background: '#F9FAFB' }}>
          <Icon name="key" size={16} />
          <span className="mono flex-1" style={{ overflowWrap: 'anywhere' }}>
            {created.secret}
          </span>
          <Button size="sm" icon="copy" onClick={() => copy(created.secret)}>
            {t(ms.copy)}
          </Button>
        </div>
      )}
      <Note tone="warn" label={t(ms.shownOnce)}>
        {smtp ? (
          t(ms.smtpSettings, { host: setup.relay.hostname ?? '', starttls: setup.relay.starttlsPort, tls: setup.relay.tlsPort })
        ) : (
          <T msg={ms.storeIt} values={{ name: <span className="mono">{'REPUTECAST_KEY'}</span> }} />
        )}
      </Note>
      {!smtp ? (
        <CodeBox lang={lang} onLang={setLang} code={code}>
          {highlight(code, lang)}
        </CodeBox>
      ) : null}
    </Dialog>
  )
}

/** The Send-only preset of Settings › Keys & MCP: Transactional's own key dialog, once Setup has loaded. */
export function SendOnlyKeyDialog({ onClose }: { onClose: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${API}/setup`, { params: { path: { workspace: slug } } })
  if (!q.data) return null
  return <CreateTransactionalKeyDialog preset="send_only" setup={q.data} onClose={onClose} />
}
