import { useQueryClient } from '@tanstack/react-query'
import { useId, useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { OptionCard } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { StepTrail } from '@/components/ui/Stepper'
import { toast } from '@/components/ui/Toast'
import { FieldError, failed, fieldErrors } from '@/features/connect/parts'
import { T, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { cn } from '@/lib/cn'
import { CopyButton, inboundPath, MxTable, TestResult, testState, useInboundRoute, useSendTest, type InboundRoute } from './ReceivingCard'
import { m } from './receiving.messages'

type Choice = Schemas['InboundProviderChoice']
const MAILBOX = 'mailbox'

/** The choice's line in step 1: who sets the route up. */
function choiceLine(t: ReturnType<typeof useT>, c: Choice): string {
  if (!c.inbound) return c.reason ?? ''
  if (c.inbound === 'automatic') return t(m.choiceAutomatic, { provider: c.name })
  if (c.adapter === 'ses') return t(m.choiceSES)
  if (c.adapter === 'postal') return t(m.choicePostal)
  return t(m.choiceManual, { provider: c.name })
}

/** A sample reply address on a domain, as step 2 shows it. */
const SAMPLE_TAG = 'r-wfr3k57v665distrwnkpaitjcv7snijhyx7fu4y'

/**
 * Mounted only while open, so each opening starts over from the route as
 * saved. Set up receiving mail (M58, InboundSetup): 1 how ReputeCast receives (a
 * connected provider or a mailbox), 2 the domain or the mailbox, 3 the MX
 * record, a manual provider's steps and the test message.
 */
export function InboundSetupDialog({ open, onOpenChange, route }: { open: boolean; onOpenChange: (o: boolean) => void; route: InboundRoute }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const ids = useId()
  const receivable = route.choices.filter((c) => c.inbound)
  const initialPick = route.kind === 'mailbox' ? MAILBOX : (route.provider?.id ?? receivable[0]?.id ?? MAILBOX)
  const [step, setStep] = useState(1)
  const [pick, setPick] = useState(initialPick)
  const [domain, setDomain] = useState(route.domain || route.suggestion)
  const mb = route.mailbox
  const [mailbox, setMailbox] = useState<MailboxValue>({
    address: mb?.address ?? '',
    protocol: mb?.protocol ?? 'imap',
    host: mb?.host ?? '',
    port: mb?.port ? String(mb.port) : '993',
    security: mb?.security ?? 'tls',
    username: mb?.username ?? '',
    password: '',
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [settings, setSettings] = useState<Record<string, string>>({})
  const choice = route.choices.find((c) => c.id === pick)
  const isMailbox = pick === MAILBOX
  const save = $api.useMutation('put', inboundPath)
  const submit = (extra?: Record<string, string>) => {
    setErrors({})
    const body: Schemas['InboundRouteInput'] = isMailbox
      ? {
          kind: 'mailbox',
          mailbox: {
            address: mailbox.address.trim(),
            protocol: mailbox.protocol === 'pop3' ? 'pop3' : 'imap',
            host: mailbox.host.trim(),
            port: Number(mailbox.port) || 0,
            security: mailbox.security === 'starttls' ? 'starttls' : 'tls',
            username: mailbox.username.trim(),
            password: mailbox.password || undefined,
          },
        }
      : { kind: 'provider', providerId: pick, domain: domain.trim(), settings: extra }
    save.mutate(
      { params: { path: { workspace: slug } }, body },
      {
        onSuccess: (next) => {
          qc.setQueryData($api.queryOptions('get', inboundPath, { params: { path: { workspace: slug } } }).queryKey, next)
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/senders'] })
          if (extra) toast({ title: t(m.settingsSaved) })
          setStep(3)
        },
        onError: (e) => {
          const f = { ...fieldErrors(e, 'body.'), ...fieldErrors(e, 'body.mailbox.'), ...fieldErrors(e, 'body.settings.') }
          if (Object.keys(f).length) setErrors(f)
          else failed(t(m.saveFailed), () => submit(extra))(e)
        },
      },
    )
  }
  const steps = [
    { key: 'how', title: t(m.step1) },
    { key: 'details', title: t(m.step2) },
    { key: 'records', title: t(m.step3) },
  ]
  const eyebrow =
    step === 1 ? t(m.eyebrowStart) : isMailbox ? t(m.eyebrowMailbox) : t(m.eyebrowProvider, { provider: choice?.name ?? '' })
  const title = step === 1 ? t(m.step1) : step === 2 ? t(m.step2) : t(m.step3)
  const routeProvider = route.provider?.name ?? ''
  const lead =
    route.kind === 'mailbox'
      ? t(m.s3Mailbox, { address: route.mailbox?.address ?? '' })
      : route.setup.automatic
        ? t(m.s3Automatic, { provider: routeProvider, domain: route.domain })
        : t(m.s3Manual, { provider: routeProvider, domain: route.domain })
  const description =
    step === 1 ? t(m.s1Description) : step === 2 ? (isMailbox ? t(m.s2Mailbox) : t(m.s2Provider, { provider: choice?.name ?? '' })) : lead
  const footer =
    step === 1 ? (
      <>
        <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
        <Button variant="primary" onClick={() => setStep(2)}>
          {t(m.continue)}
        </Button>
      </>
    ) : step === 2 ? (
      <>
        <Button className="mr-auto" onClick={() => setStep(1)}>
          {t(m.back)}
        </Button>
        <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
        <Button variant="primary" loading={save.isPending} onClick={() => submit()}>
          {t(m.continue)}
        </Button>
      </>
    ) : (
      <>
        <Button className="mr-auto" onClick={() => setStep(2)}>
          {t(m.back)}
        </Button>
        <Button variant="primary" onClick={() => onOpenChange(false)}>
          {t(m.done)}
        </Button>
      </>
    )
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      size="md"
      hideClose
      eyebrow={<span className="k">{eyebrow}</span>}
      title={title}
      description={
        <>
          {description ? <span className="block">{description}</span> : null}
          <StepTrail className="mt-1.5" steps={steps} current={step} label={t(m.stepsLabel)} />
        </>
      }
      footer={footer}
    >
      {step === 1 ? (
        <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
          <legend className="vh">{t(m.step1)}</legend>
          {route.choices.length ? <span className="lbl">{t(m.providersLabel)}</span> : null}
          {route.choices.map((c) => (
            <OptionCard
              key={c.id}
              name={`${ids}-rcv`}
              value={c.id}
              checked={pick === c.id}
              disabled={!c.inbound}
              onChange={setPick}
              className={cn(pick === c.id && 'on', !c.inbound && 'cursor-not-allowed bg-ground')}
              title={<span className={cn(!c.inbound && 'text-ink-2')}>{c.name}</span>}
              description={choiceLine(t, c)}
            />
          ))}
          <span className="lbl mt-1.5">{t(m.orLabel)}</span>
          <OptionCard
            name={`${ids}-rcv`}
            value={MAILBOX}
            checked={isMailbox}
            onChange={setPick}
            className={cn(isMailbox && 'on')}
            title={t(m.choiceMailbox)}
            description={t(m.choiceMailboxLine)}
          />
          <span className="k mt-1">{t(m.noMailServer)}</span>
        </fieldset>
      ) : step === 2 ? (
        isMailbox ? (
          <MailboxFields value={mailbox} onChange={setMailbox} errors={errors} passwordSet={Boolean(mb?.passwordSet)} />
        ) : (
          <>
            <div className="flex min-w-0 flex-col gap-1">
              <label className="lbl" htmlFor={`${ids}-dom`}>
                {t(m.domain)}
              </label>
              <input
                id={`${ids}-dom`}
                className={cn('in mono text-[12.5px]', errors.domain && 'border-bad-strong')}
                type="text"
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
                aria-invalid={Boolean(errors.domain) || undefined}
                aria-describedby={errors.domain ? `${ids}-dom-err` : `${ids}-dom-hint`}
              />
              {errors.domain ? (
                <FieldError id={`${ids}-dom-err`} message={errors.domain} />
              ) : (
                <span className="k" id={`${ids}-dom-hint`}>
                  {t(m.domainHint, { example: route.suggestion || 'reply.example.com' })}
                </span>
              )}
            </div>
            <Note tone="info" role="note">
              <T
                msg={choice?.inbound === 'automatic' ? m.addressesNote : m.addressesNoteManual}
                values={{
                  reply: `${SAMPLE_TAG}@${domain.trim() || route.suggestion}`,
                  provider: choice?.name ?? '',
                  mono: (c) => <span className="mono [overflow-wrap:anywhere]">{c}</span>,
                }}
              />
            </Note>
          </>
        )
      ) : (
        <RecordsAndTest onSaveSettings={(s) => submit({ ...settings, ...s })} saving={save.isPending} settings={settings} setSettings={setSettings} errors={errors} />
      )}
    </Dialog>
  )
}

type MailboxValue = { address: string; protocol: string; host: string; port: string; security: string; username: string; password: string }

function MailboxFields({
  value,
  onChange,
  errors,
  passwordSet,
}: {
  value: MailboxValue
  onChange: (v: MailboxValue) => void
  errors: Record<string, string>
  passwordSet: boolean
}) {
  const t = useT()
  const id = useId()
  const set = (k: keyof MailboxValue) => (e: { target: { value: string } }) => onChange({ ...value, [k]: e.target.value })
  const local = value.address.split('@')[0] || 'replies'
  const field = (k: keyof MailboxValue, label: string, input: (p: { id: string; invalid: boolean; describedBy?: string }) => ReactNode) => {
    const err = errors[k]
    return (
      <div className="flex min-w-0 flex-col gap-1">
        <label className="lbl" htmlFor={`${id}-${k}`}>
          {label}
        </label>
        {input({ id: `${id}-${k}`, invalid: Boolean(err), describedBy: err ? `${id}-${k}-err` : undefined })}
        {err ? <FieldError id={`${id}-${k}-err`} message={err} /> : null}
      </div>
    )
  }
  return (
    <>
      <div className="flex min-w-0 flex-col gap-1">
        <label className="lbl" htmlFor={`${id}-address`}>
          {t(m.mbAddress)}
        </label>
        <input
          id={`${id}-address`}
          className={cn('in mono text-[12.5px]', errors.address && 'border-bad-strong')}
          type="email"
          value={value.address}
          onChange={set('address')}
          aria-invalid={Boolean(errors.address) || undefined}
          aria-describedby={errors.address ? `${id}-address-err` : `${id}-address-hint`}
        />
        {errors.address ? (
          <FieldError id={`${id}-address-err`} message={errors.address} />
        ) : (
          <span className="k" id={`${id}-address-hint`}>
            {t(m.mbAddressHint, { local })}
          </span>
        )}
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(180px,100%),1fr))] gap-3">
        {field('protocol', t(m.mbProtocol), (p) => (
          <select id={p.id} className="in" value={value.protocol} onChange={set('protocol')}>
            <option value="imap">{'IMAP'}</option>
            <option value="pop3">{'POP3'}</option>
          </select>
        ))}
        {field('host', t(m.mbServer), (p) => (
          <input id={p.id} className={cn('in mono text-[12.5px]', p.invalid && 'border-bad-strong')} value={value.host} onChange={set('host')} aria-invalid={p.invalid || undefined} aria-describedby={p.describedBy} />
        ))}
        {field('port', t(m.mbPort), (p) => (
          <input id={p.id} className="in mono text-[12.5px]" inputMode="numeric" value={value.port} onChange={set('port')} aria-invalid={p.invalid || undefined} aria-describedby={p.describedBy} />
        ))}
        {field('security', t(m.mbSecurity), (p) => (
          <select id={p.id} className="in" value={value.security} onChange={set('security')}>
            <option value="tls">{'TLS'}</option>
            <option value="starttls">{'STARTTLS'}</option>
          </select>
        ))}
        {field('username', t(m.mbUser), (p) => (
          <input id={p.id} className={cn('in mono text-[12.5px]', p.invalid && 'border-bad-strong')} value={value.username} onChange={set('username')} aria-invalid={p.invalid || undefined} aria-describedby={p.describedBy} autoComplete="off" />
        ))}
        {field('password', t(m.mbPassword), (p) => (
          <input
            id={p.id}
            className={cn('in mono text-[12.5px]', p.invalid && 'border-bad-strong')}
            type="password"
            value={value.password}
            onChange={set('password')}
            placeholder={passwordSet ? t(m.mbPasswordKept) : undefined}
            aria-invalid={p.invalid || undefined}
            aria-describedby={p.describedBy}
            autoComplete="new-password"
          />
        ))}
      </div>
      <Note tone="info" role="note">
        {t(m.mbNote)}
      </Note>
    </>
  )
}

/** Step 3: the MX record, a manual provider's steps and settings, the test. */
function RecordsAndTest({
  onSaveSettings,
  saving,
  settings,
  setSettings,
  errors,
}: {
  onSaveSettings: (s: Record<string, string>) => void
  saving: boolean
  settings: Record<string, string>
  setSettings: (s: Record<string, string>) => void
  errors: Record<string, string>
}) {
  const t = useT()
  const id = useId()
  const { q, now } = useInboundRoute()
  const test = useSendTest()
  const r = q.data
  if (!r) return null
  const provider = r.provider?.name ?? ''
  const state = testState(r, now)
  const mxFound = r.setup.mx.length === 0 || r.setup.mx.every((x) => x.status === 'found')
  const checked = new Date()
  return (
    <>
      {r.status === 'failing' || r.statusReason === 'register' ? (
        <Note tone="bad">
          <span className="font-bold">{r.statusDetail}</span>
        </Note>
      ) : null}
      {r.setup.mx.length ? (
        <div className="flex min-w-0 flex-col gap-2">
          <span className="font-semibold">{t(m.addRecord)}</span>
          <div className="overflow-hidden rounded-lg border border-line">
            <MxTable records={r.setup.mx} />
          </div>
          <span className="k">{mxFound ? t(m.checkedFound, { time: clock(checked) }) : t(m.checkedLast, { time: clock(checked) })}</span>
        </div>
      ) : null}
      {r.setup.steps.length || r.setup.settings.length ? (
        <div className="flex flex-col gap-2.5 rounded-lg border border-line px-3.5 py-3">
          <span className="font-semibold">{t(m.inProvider, { provider })}</span>
          <ol className="ol">
            {r.setup.steps.map((s) => (
              <li key={s}>
                <span>{s}</span>
              </li>
            ))}
            {r.hookUrl ? (
              <li>
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <span>{t(m.hookAddress)}</span>
                  <div className="inl">
                    <input className="in mono text-[12.5px]" type="text" value={r.hookUrl} readOnly aria-label={t(m.hookAddress)} />
                    <CopyButton value={r.hookUrl} label={t(m.copyHook)} />
                  </div>
                </div>
              </li>
            ) : null}
            {r.setup.settings.map((f) => (
              <li key={f.key}>
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <label htmlFor={`${id}-${f.key}`}>{f.label}</label>
                  <div className="inl">
                    <input
                      id={`${id}-${f.key}`}
                      className="in mono text-[12.5px]"
                      type="text"
                      value={settings[f.key] ?? f.value}
                      onChange={(e) => setSettings({ ...settings, [f.key]: e.target.value })}
                      aria-invalid={Boolean(errors[f.key]) || undefined}
                      aria-describedby={errors[f.key] ? `${id}-${f.key}-err` : f.help ? `${id}-${f.key}-help` : undefined}
                    />
                    <Button size="sm" loading={saving} onClick={() => onSaveSettings({ [f.key]: settings[f.key] ?? f.value })}>
                      {t(m.saveSettings)}
                    </Button>
                  </div>
                  {errors[f.key] ? (
                    <FieldError id={`${id}-${f.key}-err`} message={errors[f.key] ?? ''} />
                  ) : f.help ? (
                    <span className="k" id={`${id}-${f.key}-help`}>
                      {f.help}
                    </span>
                  ) : null}
                </div>
              </li>
            ))}
          </ol>
        </div>
      ) : null}
      <div className="flex flex-col gap-2.5 rounded-lg border border-line px-3.5 py-3">
        <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2">
          <div className="flex min-w-0 flex-[1_1_260px] flex-col gap-0.5">
            <span className="font-semibold">{t(m.sendTest)}</span>
            <span className="k">{t(m.testRowLine, { domain: r.domain })}</span>
          </div>
          <Button size="sm" loading={test.pending} onClick={test.run} disabledReason={mxFound ? undefined : t(m.testNeedsMx, { domain: r.domain })}>
            {t(m.sendTest)}
          </Button>
        </div>
        {mxFound ? null : <span className="k">{t(m.testNeedsMx, { domain: r.domain })}</span>}
        <TestResult route={r} state={state} />
      </div>
    </>
  )
}
