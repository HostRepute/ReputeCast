import { useId, useState, type FormEvent } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { cn } from '@/lib/cn'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { copy, failed, StatusRow, useSetupActions, type StepProps } from './parts'
import { Ticks } from './text'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

const REVERSE_PROXY_GUIDE = 'https://github.com/hostrepute/reputecast/blob/main/docs/deploy/reverse-proxy.md'

/** 7.2 step 2 for API providers (SetupError and its happy path): the key and the webhooks. */
export function ConnectionAPI({ setup, go, saveLater, reason }: StepProps) {
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
    a.connect.mutate({ values: body }, { onError: failed(`Couldn't check the ${info.short} key`, () => connect()) })
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
    if (missing.includes(f.key)) return { field: f.key, code: 'missing', message: `Enter the ${f.label.charAt(0).toLowerCase()}${f.label.slice(1)}.` }
    return failedKey && (failedKey.field === f.key || (!failedKey.field && f.key === firstSecret)) ? failedKey : undefined
  }
  const extraOpen = extra.some((f) => errorFor(f) || (f.secret ? f.set : Boolean(f.value)))
  return (
    <>
      <StepHeading title={info.heading}>{info.lead}</StepHeading>

      <section className={cn('card flex flex-col gap-3.5 px-5 py-4.5', failedKey && 'border-bad-line')} aria-labelledby="key-h">
        <h3 className="h2" id="key-h">
          {info.section || '1. API key'}
        </h3>
        <form id="api-key" className="contents" onSubmit={connect}>
          <div className="grid items-end gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(260px, 100%), 1fr))' }}>
            {main.map((f) => (
              <KeyField key={f.key} field={f} value={values[f.key] ?? ''} onChange={(v) => change(f.key, v)} error={errorFor(f)} disabled={Boolean(reason)} />
            ))}
          </div>
          {extra.length ? (
            <details className="text-[13px]" open={extraOpen || undefined}>
              <summary className="cursor-pointer font-semibold">{`More ${info.short} settings`}</summary>
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
            {failedKey ? 'Try again' : connected ? 'Check again' : 'Connect'}
          </Button>
          {info.keysUrl ? <ButtonAnchor href={info.keysUrl}>{`Open ${info.short} API keys`}</ButtonAnchor> : null}
          <span className="k">{info.verifyCall ? `Keys are checked with a read-only call (${info.verifyCall}) and stored encrypted.` : 'Keys are checked with a read-only call and stored encrypted.'}</span>
        </div>
      </section>

      {c.webhook && c.webhook.mode !== 'none' && c.webhook.mode !== 'mailbox' ? (
        <WebhooksCard setup={setup} webhook={c.webhook} reason={reason} />
      ) : null}

      <WizardFooter
        back={
          <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            Back
          </ButtonLink>
        }
      >
        <Button onClick={saveLater} disabledReason={reason}>
          Save and finish later
        </Button>
        <Button
          variant="primary"
          className={continueClass}
          onClick={() => go('domain')}
          disabledReason={c.canContinue ? undefined : connected ? 'The webhook endpoint has to work first, or use polling.' : `Connect ${info.short} first.`}
        >
          Continue to domain verification
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
        placeholder={f.secret && f.set ? 'Stored encrypted · paste a new one to replace it' : undefined}
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
          if (s?.connection.webhook?.selfTest?.ok) toast({ title: 'The endpoint answered', detail: reg?.status === 'done' ? undefined : `Webhooks registered at ${name}.` })
        },
        onError: failed("Couldn't test the endpoint again", retest),
      },
    )
  }
  const usePolling = () => a.polling.mutate(undefined, { onError: failed("Couldn't switch to polling", usePolling) })
  let badge = <Badge tone="neutral">After the key is accepted</Badge>
  if (w.mode === 'polling') badge = <Badge tone="info" dot>Polling every minute</Badge>
  else if (unreachable) badge = <Badge tone="warn" dot>Endpoint not reachable from the internet</Badge>
  else if (reg?.status === 'done') badge = <Badge tone="ok" dot>Registered</Badge>
  else if (reg?.status === 'manual') badge = <Badge tone="warn" dot>{`Add the webhook in ${name}`}</Badge>
  else if (reg?.status === 'failed') badge = <Badge tone="bad" dot>Registration failed</Badge>
  const org = orgDomain(setup.domain.suggested)
  return (
    <section className={cn('card flex flex-col', unreachable && 'border-[#FCD34D]')} aria-labelledby="hooks-h">
      <div className={cn('flex flex-col gap-1 px-5 py-4', unreachable && 'rounded-t-[10px] bg-[#FFFBEB]')}>
        <div className="flex flex-wrap items-center gap-2.5">
          <h3 className="h2" id="hooks-h">
            2. Webhooks
          </h3>
          {badge}
        </div>
        {unreachable ? (
          <span className="text-[13px] text-[#78350F]">{`${name} has to call ReputeCast to report deliveries, bounces and complaints. ReputeCast tried to reach its own public URL the way ${name} would, and failed.`}</span>
        ) : w.mode === 'polling' ? (
          <span className="text-[13px] text-ink-2">{`ReputeCast asks ${name}'s events API for new events every minute. Bounce protection during a send lags by up to a minute.`}</span>
        ) : (
          <span className="text-[13px] text-ink-2">{`${name} calls ReputeCast to report deliveries, bounces and complaints.`}</span>
        )}
      </div>
      {w.mode === 'polling' ? (
        <StatusRow badge={<Badge tone="ok">Done</Badge>} aside={<Button size="sm" onClick={retest} loading={a.webhook.isPending} disabledReason={reason}>Use webhooks instead</Button>}>
          {`Polling ${name}'s events API every minute`}
        </StatusRow>
      ) : (
        <>
          <StatusRow
            className={unreachable ? 'border-t-[#FDE68A]' : undefined}
            badge={test ? test.ok ? <Badge tone="ok">Done</Badge> : <Badge tone="bad">Failed</Badge> : <Badge tone="neutral">Waiting</Badge>}
            aside={
              test ? (
                test.ok ? (
                  <span className="k">{clock(test.at)}</span>
                ) : (
                  <span className="mono k text-bad-fg">{[test.code, test.reason].filter(Boolean).join(' · ')}</span>
                )
              ) : (
                <span className="k">after the key is accepted</span>
              )
            }
          >
            Self-test of <span className="mono">{w.url || '…'}</span>
          </StatusRow>
          <RegisterRow name={name} events={info.webhookEvents} reg={reg} url={w.url} reachable={Boolean(test?.ok)} />
          {reg?.status === 'manual' && w.needsSecret ? <ManualSecret setup={setup} webhook={w} reason={reason} /> : null}
        </>
      )}
      {unreachable ? (
        <div className="flex flex-col gap-2.5 border-t border-line-2 px-5 py-3.5">
          <span className="text-[13px] font-semibold">How to fix</span>
          <div className="grid gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(280px, 100%), 1fr))' }}>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">Set a public URL</span>
              <span className="k">
                {test?.private ? 'ReputeCast is running at a private address. ' : "Providers can't reach ReputeCast at its public URL. "}
                Set <span className="mono">REPUTECAST_PUBLIC_URL</span> in your docker compose .env to the HTTPS address your reverse proxy exposes, then restart.
              </span>
              <ScrollPre className="cn-pre" label="Setting for the docker compose .env">{`REPUTECAST_PUBLIC_URL=https://reputecast.${org}`}</ScrollPre>
            </div>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">Or use a tunnel while testing</span>
              <span className="k">For a laptop or staging box, expose port 8080 with Cloudflare Tunnel or ngrok and paste the URL here.</span>
              <label className="lbl">
                Public URL
                <input className="in mono" placeholder="https://…" value={tunnel} onChange={(e) => setTunnel(e.target.value)} disabled={Boolean(reason)} inputMode="url" />
              </label>
            </div>
            {info.canPoll ? (
              <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
                <span className="font-semibold">Or poll instead of webhooks</span>
                <span className="k">{`${name} keeps an events API. ReputeCast can poll it every minute. Slower, and bounce protection during a send lags by up to a minute.`}</span>
                <Button size="sm" className="self-start" onClick={usePolling} loading={a.polling.isPending} disabledReason={reason}>
                  Use polling for now
                </Button>
              </div>
            ) : null}
          </div>
          <div className="flex flex-wrap gap-2.5">
            <Button variant="primary" onClick={retest} loading={a.webhook.isPending} busyLabel="Testing…" disabledReason={reason}>
              Re-test endpoint
            </Button>
            <ButtonAnchor href={REVERSE_PROXY_GUIDE}>Reverse-proxy guide</ButtonAnchor>
          </div>
        </div>
      ) : reg?.status === 'failed' ? (
        <div className="flex flex-wrap gap-2.5 border-t border-line-2 px-5 py-3.5">
          <Button variant="primary" onClick={retest} loading={a.webhook.isPending} disabledReason={reason}>
            Try registering again
          </Button>
          {info.canPoll ? (
            <Button onClick={usePolling} loading={a.polling.isPending} disabledReason={reason}>
              Use polling for now
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

/**
 * A webhook ReputeCast can't register: the person adds it in the provider,
 * with the secret ReputeCast generated when it connected the provider.
 */
function ManualSecret({ setup, webhook: w, reason }: { setup: Schemas['ConnectSetup']; webhook: Schemas['ConnectWebhook']; reason?: string }) {
  const { slug } = useWorkspace()
  const [shown, setShown] = useState<Schemas['ConnectWebhookSecret']>()
  const reveal = $api.useMutation('post', '/api/v1/workspaces/{workspace}/connect/setups/{setupId}/webhook-secret')
  const name = setup.info.short
  const header = w.secretHeader
  const show = () =>
    reveal.mutate(
      { params: { path: { workspace: slug, setupId: setup.id } } },
      { onSuccess: (data) => setShown(data), onError: failed("Couldn't show the webhook secret", show) },
    )
  return (
    <div className="flex flex-col gap-2.5 border-t border-line-2 px-5 py-3.5">
      <span className="text-[13px] font-semibold">{`Set up the webhook in ${name}`}</span>
      <span className="k">
        {header ? (
          <>
            Point it at the URL above and send the secret below in the <span className="mono">{header}</span> header. ReputeCast refuses deliveries without it.
          </>
        ) : (
          <>Point it at the URL above and sign the deliveries with the secret below. ReputeCast refuses deliveries it can&apos;t verify.</>
        )}
      </span>
      {shown ? (
        <div className="flex flex-wrap items-center gap-2.5">
          <code className="cn-val">{header ? `${header}: ${shown.secret}` : shown.secret}</code>
          <Button size="sm" onClick={() => void copy(shown.secret, 'the webhook secret')}>
            Copy secret
          </Button>
          <span className="k">Keep it private. Anyone with it can post events for this provider.</span>
        </div>
      ) : (
        <div className="flex flex-wrap items-center gap-2.5">
          <Button size="sm" onClick={show} loading={reveal.isPending} busyLabel="Showing…" disabledReason={reason}>
            Show the secret
          </Button>
          <span className="k">Showing it is recorded in the audit log.</span>
        </div>
      )}
    </div>
  )
}

function RegisterRow({ name, events, reg, url, reachable }: { name: string; events: string; reg?: Schemas['ConnectRegistration']; url: string; reachable: boolean }) {
  if (reg?.status === 'done') {
    return (
      <StatusRow badge={<Badge tone="ok">Done</Badge>} aside={<span className="k">{clock(reg.at)}</span>}>
        {`Registered webhooks: ${events}`}
      </StatusRow>
    )
  }
  if (reg?.status === 'manual') {
    return (
      <StatusRow
        badge={<Badge tone="warn">Pending</Badge>}
        aside={
          <Button size="sm" onClick={() => void copy(url, 'the webhook URL')}>
            Copy URL
          </Button>
        }
      >
        Add <span className="mono">{url}</span> as a webhook in {name} for {events}
      </StatusRow>
    )
  }
  if (reg?.status === 'failed') {
    return (
      <StatusRow badge={<Badge tone="bad">Failed</Badge>} aside={reg.detail ? <span className="mono k text-bad-fg">{reg.detail}</span> : null}>
        {`Register webhooks: ${events}`}
      </StatusRow>
    )
  }
  return (
    <StatusRow badge={<Badge tone="neutral">Skipped</Badge>} aside={<span className="k">{reachable ? 'waiting' : 'after the endpoint is reachable'}</span>}>
      {`Register webhooks: ${events}`}
    </StatusRow>
  )
}

/** "news.acme.io" → "acme.io" (for the PUBLIC_URL example). */
function orgDomain(domain: string): string {
  const labels = domain.split('.').filter(Boolean)
  return labels.length >= 2 ? labels.slice(-2).join('.') : 'example.com'
}
