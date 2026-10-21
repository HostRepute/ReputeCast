import { useId, useState, type FormEvent } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mn } from './connection.messages'
import { m } from './messages'
import { copy, failed, StatusRow, useSetupActions, type StepProps } from './parts'
import { Ticks } from './text'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

const REVERSE_PROXY_GUIDE = 'https://github.com/hostrepute/reputecast/blob/main/docs/deploy/reverse-proxy.md'

const mono = (chunks: React.ReactNode[]) => <span className="mono">{chunks}</span>

/** 7.2 step 2 for API providers (SetupError and its happy path): the key and the webhooks. */
export function ConnectionAPI({ setup, go, saveLater, reason }: StepProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const info = setup.info
  const c = setup.connection
  const fields = c.fields.length ? c.fields : info.fields
  const [values, setValues] = useState<Record<string, string>>(() => Object.fromEntries(fields.map((f) => [f.key, f.secret ? '' : f.value || f.options[0]?.value || ''])))
  // Required fields left empty, found when the form is sent: the button
  // stays available and the errors appear next to the fields.
  const [missing, setMissing] = useState<string[]>([])
  const failedKey = c.status === 'failed' ? c.error : undefined
  const connected = c.status === 'connected'
  const connect = (e?: FormEvent) => {
    e?.preventDefault()
    const empty = fields.filter((f) => f.required && !(values[f.key] ?? '').trim() && !(f.secret && f.set)).map((f) => f.key)
    setMissing(empty)
    if (empty.length) {
      document.getElementById(`cn-field-${empty[0]}`)?.focus()
      return
    }
    const body: Record<string, string> = {}
    for (const f of fields) {
      const v = (values[f.key] ?? '').trim()
      // A stored secret stays when its field is left empty.
      if (v || !f.secret) body[f.key] = v
    }
    a.connect.mutate({ values: body }, { onError: failed(t(mn.keyCheckFailed, { provider: info.short }), () => connect()) })
  }
  const change = (key: string, v: string) => {
    setValues((s) => ({ ...s, [key]: v }))
    if (v.trim()) setMissing((m) => m.filter((k) => k !== key))
  }
  // The key, the required fields and the selects come first; optional
  // fields (allowances, signing keys, alternatives) fold away.
  const firstSecret = fields.find((f) => f.secret)?.key
  const main = fields.filter((f) => f.required || f.options.length > 0 || f.key === firstSecret)
  const extra = fields.filter((f) => !main.includes(f))
  const errorFor = (f: Schemas['ConnectField']): Schemas['ConnectError'] | undefined => {
    if (missing.includes(f.key)) return { field: f.key, code: 'missing', message: t(mn.fieldMissing, { field: `${f.label.charAt(0).toLowerCase()}${f.label.slice(1)}` }) }
    return failedKey && (failedKey.field === f.key || (!failedKey.field && f.key === firstSecret)) ? failedKey : undefined
  }
  const extraOpen = extra.some((f) => errorFor(f) || (f.secret ? f.set : Boolean(f.value)))
  return (
    <>
      <StepHeading title={info.heading}>{info.lead}</StepHeading>

      <section className={cn('card flex flex-col gap-3.5 px-5 py-4.5', failedKey && 'border-bad-line')} aria-labelledby="key-h">
        <h3 className="h2" id="key-h">
          {info.section || t(mn.sectionDefault)}
        </h3>
        <form id="api-key" className="contents" onSubmit={connect}>
          <div className="grid items-end gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(260px, 100%), 1fr))' }}>
            {main.map((f) => (
              <KeyField key={f.key} field={f} value={values[f.key] ?? ''} onChange={(v) => change(f.key, v)} error={errorFor(f)} disabled={Boolean(reason)} />
            ))}
          </div>
          {extra.length ? (
            <details className="text-[13px]" open={extraOpen || undefined}>
              <summary className="cursor-pointer font-semibold">{t(mn.moreSettings, { provider: info.short })}</summary>
              <div className="mt-3 grid items-end gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(260px, 100%), 1fr))' }}>
                {extra.map((f) => (
                  <KeyField key={f.key} field={f} value={values[f.key] ?? ''} onChange={(v) => change(f.key, v)} error={errorFor(f)} disabled={Boolean(reason)} />
                ))}
              </div>
            </details>
          ) : null}
          <button type="submit" hidden aria-hidden="true" tabIndex={-1} />
        </form>
        {failedKey && !fields.some((f) => failedKey.field === f.key || (!failedKey.field && f.key === firstSecret)) ? (
          <Note tone="bad">
            <Ticks text={failedKey.message} />
          </Note>
        ) : null}
        <div className="flex flex-wrap items-center gap-2.5">
          <Button
            type="submit"
            form="api-key"
            variant={connected ? 'default' : 'primary'}
            loading={a.connect.isPending}
            disabledReason={reason}
          >
            {failedKey ? t(common.tryAgain) : connected ? t(mn.checkAgain) : t(mn.connect)}
          </Button>
          {info.keysUrl ? <ButtonAnchor href={info.keysUrl}>{t(mn.openKeys, { provider: info.short })}</ButtonAnchor> : null}
          <span className="k">{info.verifyCall ? t(mn.keysChecked, { call: info.verifyCall }) : t(mn.keysCheckedPlain)}</span>
        </div>
      </section>

      {c.webhook && c.webhook.mode !== 'none' && c.webhook.mode !== 'mailbox' ? (
        <WebhooksCard setup={setup} webhook={c.webhook} reason={reason} />
      ) : null}

      <WizardFooter
        back={
          <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            {t(common.back)}
          </ButtonLink>
        }
      >
        <Button onClick={saveLater} disabledReason={reason}>
          {t(mn.saveLater)}
        </Button>
        <Button
          variant="primary"
          className={continueClass}
          onClick={() => go('domain')}
          disabledReason={c.canContinue ? undefined : connected ? t(mn.webhookFirst) : t(mn.connectFirst, { provider: info.short })}
        >
          {t(mn.continueDomain)}
        </Button>
      </WizardFooter>
    </>
  )
}

/** One field of the provider's form; the key spans two columns. */
function KeyField({
  field: f,
  value,
  onChange,
  error,
  disabled,
}: {
  field: Schemas['ConnectField']
  value: string
  onChange: (v: string) => void
  error?: Schemas['ConnectError']
  disabled: boolean
}) {
  const t = useT()
  const errId = useId()
  if (f.options.length) {
    return (
      <label className="lbl">
        {f.label}
        <select id={`cn-field-${f.key}`} className="in" value={value} onChange={(e) => onChange(e.target.value)} disabled={disabled}>
          {f.options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </label>
    )
  }
  return (
    <label className={cn('lbl', f.secret && 'col-span-2 max-md:col-span-full', error && 'text-bad-fg')}>
      {f.label}
      <input
        id={`cn-field-${f.key}`}
        className={cn('in mono', error && 'bg-bad-tint')}
        type={f.secret ? 'password' : f.type === 'number' ? 'number' : 'text'}
        inputMode={f.type === 'number' ? 'numeric' : undefined}
        min={f.type === 'number' ? 0 : undefined}
        step={f.type === 'number' ? 1 : undefined}
        value={value}
        placeholder={f.secret && f.set ? t(mn.storedPlaceholder) : undefined}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errId : undefined}
        disabled={disabled}
        autoComplete={f.secret ? 'new-password' : 'off'}
        spellCheck={false}
      />
      {error ? (
        <span id={errId} role="alert" className="mt-1 flex items-start gap-2 font-medium text-bad-fg">
          <svg width="16" height="16" viewBox="0 0 20 20" fill="none" stroke="#B91C1C" strokeWidth="1.8" aria-hidden="true" className="mt-px flex-none">
            <circle cx="10" cy="10" r="8" />
            <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
          </svg>
          <span>
            <Ticks text={error.message} />
          </span>
        </span>
      ) : f.help ? (
        <span className="k font-medium">{f.help}</span>
      ) : null}
    </label>
  )
}

/** How events come back: the endpoint self-test, registration, or polling. */
function WebhooksCard({ setup, webhook: w, reason }: { setup: Schemas['ConnectSetup']; webhook: Schemas['ConnectWebhook']; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const info = setup.info
  const name = info.short
  const [tunnel, setTunnel] = useState(w.publicUrl)
  const test = w.selfTest
  const unreachable = Boolean(test && !test.ok)
  const reg = w.registration
  const retest = () => {
    a.webhook.mutate(
      { publicUrl: tunnel.trim() || undefined },
      {
        onSuccess: (s) => {
          if (s?.connection.webhook?.selfTest?.ok) toast({ title: t(mn.endpointAnswered), detail: reg?.status === 'done' ? undefined : t(mn.webhooksRegistered, { provider: name }) })
        },
        onError: failed(t(mn.retestFailed), retest),
      },
    )
  }
  const usePolling = () => a.polling.mutate(undefined, { onError: failed(t(mn.pollingFailed), usePolling) })
  let badge = <Badge tone="neutral">{t(mn.badgeAfterKey)}</Badge>
  if (w.mode === 'polling')
    badge = (
      <Badge tone="info" dot>
        {t(mn.badgePolling)}
      </Badge>
    )
  else if (unreachable)
    badge = (
      <Badge tone="warn" dot>
        {t(mn.badgeUnreachable)}
      </Badge>
    )
  else if (reg?.status === 'done')
    badge = (
      <Badge tone="ok" dot>
        {t(mn.badgeRegistered)}
      </Badge>
    )
  else if (reg?.status === 'manual')
    badge = (
      <Badge tone="warn" dot>
        {t(mn.badgeAddManually, { provider: name })}
      </Badge>
    )
  else if (reg?.status === 'failed')
    badge = (
      <Badge tone="bad" dot>
        {t(mn.badgeRegistrationFailed)}
      </Badge>
    )
  const org = orgDomain(setup.domain.suggested)
  return (
    <section className={cn('card flex flex-col', unreachable && 'border-[#FCD34D]')} aria-labelledby="hooks-h">
      <div className={cn('flex flex-col gap-1 px-5 py-4', unreachable && 'rounded-t-[10px] bg-[#FFFBEB]')}>
        <div className="flex flex-wrap items-center gap-2.5">
          <h3 className="h2" id="hooks-h">
            {t(mn.webhooksTitle)}
          </h3>
          {badge}
        </div>
        {unreachable ? (
          <span className="text-[13px] text-[#78350F]">{t(mn.unreachableLead, { provider: name })}</span>
        ) : w.mode === 'polling' ? (
          <span className="text-[13px] text-ink-2">{t(mn.pollingLead, { provider: name })}</span>
        ) : (
          <span className="text-[13px] text-ink-2">{t(mn.callsLead, { provider: name })}</span>
        )}
      </div>
      {w.mode === 'polling' ? (
        <StatusRow
          badge={<Badge tone="ok">{t(m.stateDone)}</Badge>}
          aside={
            <Button size="sm" onClick={retest} loading={a.webhook.isPending} disabledReason={reason}>
              {t(mn.useWebhooks)}
            </Button>
          }
        >
          {t(mn.pollingRow, { provider: name })}
        </StatusRow>
      ) : (
        <>
          <StatusRow
            className={unreachable ? 'border-t-[#FDE68A]' : undefined}
            badge={test ? test.ok ? <Badge tone="ok">{t(m.stateDone)}</Badge> : <Badge tone="bad">{t(m.stateFailed)}</Badge> : <Badge tone="neutral">{t(mn.waiting)}</Badge>}
            aside={
              test ? (
                test.ok ? (
                  <span className="k">{clock(test.at)}</span>
                ) : (
                  <span className="mono k text-bad-fg">{[test.code, test.reason].filter(Boolean).join(' · ')}</span>
                )
              ) : (
                <span className="k">{t(mn.afterKey)}</span>
              )
            }
          >
            <T msg={mn.selfTest} values={{ url: w.url || '…', mono }} />
          </StatusRow>
          <RegisterRow name={name} events={info.webhookEvents} reg={reg} url={w.url} reachable={Boolean(test?.ok)} />
          {reg?.status === 'manual' && w.needsSecret ? <ManualSecret setup={setup} webhook={w} reason={reason} /> : null}
        </>
      )}
      {unreachable ? (
        <div className="flex flex-col gap-2.5 border-t border-line-2 px-5 py-3.5">
          <span className="text-[13px] font-semibold">{t(mn.howToFix)}</span>
          <div className="cn-fix grid gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(var(--cn-fix-min,140px),1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(280px, 100%), 1fr))' }}>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">{t(mn.setPublicUrl)}</span>
              <span className="k">
                <T msg={test?.private ? mn.publicUrlPrivate : mn.publicUrlUnreachable} values={{ env: 'REPUTECAST_PUBLIC_URL', mono }} />
              </span>
              <ScrollPre className="cn-pre" label={t(mn.envLabel)}>{`REPUTECAST_PUBLIC_URL=https://reputecast.${org}`}</ScrollPre>
            </div>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">{t(mn.tunnelTitle)}</span>
              <span className="k">{t(mn.tunnelLine)}</span>
              <label className="lbl">
                {t(mn.publicUrl)}
                <input className="in mono" placeholder={'https://…'} value={tunnel} onChange={(e) => setTunnel(e.target.value)} disabled={Boolean(reason)} inputMode="url" />
              </label>
            </div>
            {info.canPoll ? (
              <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
                <span className="font-semibold">{t(mn.pollTitle)}</span>
                <span className="k">{t(mn.pollLine, { provider: name })}</span>
                <Button size="sm" className="self-start" onClick={usePolling} loading={a.polling.isPending} disabledReason={reason}>
                  {t(mn.usePolling)}
                </Button>
              </div>
            ) : null}
          </div>
          <div className="flex flex-wrap gap-2.5">
            <Button variant="primary" onClick={retest} loading={a.webhook.isPending} busyLabel={t(mn.testing)} disabledReason={reason}>
              {t(mn.retest)}
            </Button>
            <ButtonAnchor href={REVERSE_PROXY_GUIDE}>{t(mn.proxyGuide)}</ButtonAnchor>
          </div>
        </div>
      ) : reg?.status === 'failed' ? (
        <div className="flex flex-wrap gap-2.5 border-t border-line-2 px-5 py-3.5">
          <Button variant="primary" onClick={retest} loading={a.webhook.isPending} disabledReason={reason}>
            {t(mn.registerAgain)}
          </Button>
          {info.canPoll ? (
            <Button onClick={usePolling} loading={a.polling.isPending} disabledReason={reason}>
              {t(mn.usePolling)}
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

/**
 * A webhook ReputeCast can't register: the person adds it in the provider,
 * with a secret ReputeCast issues for it. Like API keys, a secret is shown
 * only when it's issued; a new one replaces the previous one.
 */
function ManualSecret({ setup, webhook: w, reason }: { setup: Schemas['ConnectSetup']; webhook: Schemas['ConnectWebhook']; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [shown, setShown] = useState<Schemas['ConnectWebhookSecret']>()
  const reveal = $api.useMutation('post', '/api/v1/workspaces/{workspace}/connect/setups/{setupId}/webhook-secret')
  const name = setup.info.short
  const header = w.secretHeader
  const show = () =>
    reveal.mutate(
      { params: { path: { workspace: slug, setupId: setup.id } } },
      { onSuccess: (data) => setShown(data), onError: failed(t(mn.secretFailed), show) },
    )
  return (
    <div className="flex flex-col gap-2.5 border-t border-line-2 px-5 py-3.5">
      <span className="text-[13px] font-semibold">{t(mn.manualTitle, { provider: name })}</span>
      <span className="k">{header ? <T msg={mn.manualHeader} values={{ header, mono }} /> : t(mn.manualSign)}</span>
      {shown ? (
        <div className="flex flex-wrap items-center gap-2.5">
          <code className="cn-val">{header ? `${header}: ${shown.secret}` : shown.secret}</code>
          <Button size="sm" onClick={() => void copy(shown.secret, 'secret')}>
            {t(mn.copySecret)}
          </Button>
          <span className="k">{t(mn.secretOnce)}</span>
        </div>
      ) : (
        <div className="flex flex-wrap items-center gap-2.5">
          <Button size="sm" onClick={show} loading={reveal.isPending} busyLabel={t(mn.creating)} disabledReason={reason}>
            {t(mn.createSecret)}
          </Button>
          <span className="k">{t(mn.secretReplaces, { provider: name })}</span>
        </div>
      )}
    </div>
  )
}

function RegisterRow({ name, events, reg, url, reachable }: { name: string; events: string; reg?: Schemas['ConnectRegistration']; url: string; reachable: boolean }) {
  const t = useT()
  if (reg?.status === 'done') {
    return (
      <StatusRow badge={<Badge tone="ok">{t(m.stateDone)}</Badge>} aside={<span className="k">{clock(reg.at)}</span>}>
        {t(mn.registered, { events })}
      </StatusRow>
    )
  }
  if (reg?.status === 'manual') {
    return (
      <StatusRow
        badge={<Badge tone="warn">{t(m.statePending)}</Badge>}
        aside={
          <Button size="sm" onClick={() => void copy(url, 'url')}>
            {t(mn.copyUrl)}
          </Button>
        }
      >
        <T msg={mn.addManually} values={{ url, provider: name, events, mono }} />
      </StatusRow>
    )
  }
  if (reg?.status === 'failed') {
    return (
      <StatusRow badge={<Badge tone="bad">{t(m.stateFailed)}</Badge>} aside={reg.detail ? <span className="mono k text-bad-fg">{reg.detail}</span> : null}>
        {t(mn.register, { events })}
      </StatusRow>
    )
  }
  return (
    <StatusRow badge={<Badge tone="neutral">{t(m.stateSkipped)}</Badge>} aside={<span className="k">{reachable ? t(mn.waitingAside) : t(mn.afterReachable)}</span>}>
      {t(mn.register, { events })}
    </StatusRow>
  )
}

/** "news.acme.io" → "acme.io" (for the PUBLIC_URL example). */
function orgDomain(domain: string): string {
  const labels = domain.split('.').filter(Boolean)
  return labels.length >= 2 ? labels.slice(-2).join('.') : 'example.com'
}
