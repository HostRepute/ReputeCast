import { useState, type FormEvent, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Row } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, clockSeconds, count, decimal } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mn, mses } from './connection.messages'
import { m } from './messages'
import { failed, SectionCard, StatusRow, useSetupActions, type StepProps } from './parts'
import { Ticks } from './text'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

/** SES regions offered before the account has been read, with AWS's own names for them. */
const SES_REGIONS: [string, string][] = [
  ['us-east-1', 'N. Virginia'],
  ['us-east-2', 'Ohio'],
  ['us-west-1', 'N. California'],
  ['us-west-2', 'Oregon'],
  ['ca-central-1', 'Canada'],
  ['eu-west-1', 'Ireland'],
  ['eu-west-2', 'London'],
  ['eu-west-3', 'Paris'],
  ['eu-central-1', 'Frankfurt'],
  ['eu-north-1', 'Stockholm'],
  ['eu-south-1', 'Milan'],
  ['ap-south-1', 'Mumbai'],
  ['ap-northeast-1', 'Tokyo'],
  ['ap-northeast-2', 'Seoul'],
  ['ap-southeast-1', 'Singapore'],
  ['ap-southeast-2', 'Sydney'],
  ['sa-east-1', 'São Paulo'],
]

const fourCols = { gridTemplateColumns: 'repeat(auto-fit, minmax(min(300px, 100%), 1fr))' }

const mono = (chunks: ReactNode[]) => <span className="mono">{chunks}</span>

/** A sending rate with up to two decimals and no trailing zeros, in the region format: 1, 14, 0.5. */
function rateText(v: number): string {
  const n = Number(v.toFixed(2))
  return decimal(n, (String(n).split('.')[1] ?? '').length)
}

/** 7.2 step 2 for Amazon SES (SetupSES): AWS access, region, how events arrive, provisioning and the event check, sandbox. */
export function ConnectionSES({ setup, go, saveLater, reason }: StepProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const c = setup.connection
  const connected = c.status === 'connected'
  const host = c.host
  // Off AWS keys come first and are preselected (PR-5); on AWS the stack.
  const recommended = host?.recommended ?? 'keys'
  const [access, setAccess] = useState<'role' | 'keys'>(c.access || recommended)
  const [replacing, setReplacing] = useState(false)
  const [roleArn, setRoleArn] = useState(c.roleArn)
  const [keyId, setKeyId] = useState('')
  const [secret, setSecret] = useState('')
  const [region, setRegion] = useState(c.region || 'us-east-1')
  const [events, setEvents] = useState<'webhook' | 'queue'>(c.events || 'webhook')
  const connecting = a.connect.isPending
  const editing = !connected || replacing || access !== c.access
  const connect = (e?: FormEvent) => {
    e?.preventDefault()
    a.connect.mutate(
      {
        ses:
          access === 'role'
            ? { access, roleArn: roleArn.trim(), region, events }
            : { access, accessKeyId: keyId.trim(), secretAccessKey: secret, region, events },
      },
      { onSuccess: () => setReplacing(false), onError: failed(t(mses.accessFailed), () => connect()) },
    )
  }
  const pickRegion = (r: string) => {
    setRegion(r)
    if (connected) a.region.mutate({ region: r }, { onError: failed(t(mses.regionFailed)) })
  }
  const pickEvents = (v: 'webhook' | 'queue') => {
    setEvents(v)
    if (connected && v !== c.events) a.events.mutate({ events: v }, { onError: failed(t(mses.eventsFailed), () => pickEvents(v)) })
  }
  const err = c.error
  const done = c.provisioning.filter((r) => r.status === 'done').length
  const keysOption = (
    <AccessOption
      key="keys"
      checked={access === 'keys'}
      onPick={() => setAccess('keys')}
      disabled={Boolean(reason) || connecting}
      title={t(mses.keysTitle)}
      recommended={recommended === 'keys' ? t(mses.recommendedHere) : undefined}
      description={t(mses.keysLine)}
    >
      {connected && c.access === 'keys' && !replacing ? (
        <>
          <span className="mt-2 grid grid-cols-2 gap-2">
            <input className="in mono min-w-0" value={c.accessKey || 'AKIA…'} aria-label={t(mses.keyIdLabel)} readOnly />
            <input className="in min-w-0" value={t(mses.savedSecret)} aria-label={t(mses.secretLabel)} readOnly />
          </span>
          <Button size="sm" className="mt-2" onClick={() => setReplacing(true)} disabledReason={reason}>
            {t(mses.replaceKeys)}
          </Button>
        </>
      ) : (
        <span className="mt-2 grid grid-cols-2 gap-2">
          {/* Disabled until picked, but white as on the board. */}
          <input
            className="in mono min-w-0 bg-white"
            placeholder={'AKIA…'}
            aria-label={t(mses.keyIdLabel)}
            aria-invalid={err?.field === 'access_key_id' || undefined}
            disabled={access !== 'keys' || Boolean(reason)}
            value={keyId}
            onChange={(e) => setKeyId(e.target.value)}
            autoComplete="off"
          />
          <input
            className="in mono min-w-0 bg-white"
            type="password"
            placeholder={t(mses.secretPlaceholder)}
            aria-label={t(mses.secretLabel)}
            disabled={access !== 'keys' || Boolean(reason)}
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
            autoComplete="new-password"
          />
        </span>
      )}
    </AccessOption>
  )
  const roleOption = (
    <AccessOption
      key="role"
      checked={access === 'role'}
      onPick={() => setAccess('role')}
      disabled={Boolean(reason) || connecting}
      title={t(mses.stackTitle)}
      recommended={recommended === 'role' ? t(mses.recommendedHere) : undefined}
      description={t(mses.stackLine)}
    >
      <a href={c.template?.launchUrl} className="btn sm mt-2 underline decoration-1 underline-offset-2" target="_blank" rel="noopener noreferrer">
        {t(mses.launchStack)}
      </a>
      {access === 'role' && (!connected || c.access !== 'role') ? (
        <span className="mt-2 flex flex-col gap-1">
          <input
            className="in mono w-full"
            placeholder={`arn:aws:iam::123456789012:role/${c.template?.roleName || 'ReputeCastSES'}`}
            aria-label={t(mses.roleArnLabel)}
            aria-invalid={err?.field === 'role_arn' || undefined}
            value={roleArn}
            onChange={(e) => setRoleArn(e.target.value)}
            disabled={Boolean(reason)}
          />
          <span className="k">{c.externalId ? <T msg={mses.roleArnHint} values={{ id: c.externalId, mono }} /> : t(mses.roleArnHintNoId)}</span>
          {host && !host.onAws ? <span className={host.baseCredentials ? 'k' : 'k text-warn-fg'}>{host.baseCredentials ? t(mses.stackNeedsKeys) : t(mses.stackNoKeys)}</span> : null}
        </span>
      ) : null}
    </AccessOption>
  )
  const verified = c.eventsCheck === 'verified'
  const caption = !connected
    ? undefined
    : verified && c.eventsAt
      ? t(mses.continueVerified, { time: clockSeconds(c.eventsAt) })
      : c.eventsCheck === 'deferred'
        ? t(mses.continueDeferred)
        : c.eventsCheck === 'none'
          ? undefined
          : t(mses.continueWaiting)
  return (
    <>
      <StepHeading title={setup.info.heading} note={t(mses.setupTime)}>
        {setup.info.lead}
      </StepHeading>

      <SectionCard title={t(mses.grantTitle)}>
        {host ? (
          // Where ReputeCast runs decides the access it recommends (PR-5).
          <div className="flex items-start gap-2.5 rounded-[8px] bg-(--info-bg) px-3 py-2.5 text-(--info-fg)">
            <svg className="mt-0.5 flex-none" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
              <circle cx="8" cy="8" r="6.2" />
              <path d="M8 7.2v3.6M8 5.2v.1" />
            </svg>
            <p className="m-0 text-[13px]">
              <span className="font-semibold">{host.onAws ? t(mses.hostOnAws, { kind: hostKind(host.kind) }) : t(mses.hostOffAws)}</span>{' '}
              {host.onAws ? t(mses.hostCheckedOn, { time: clock(host.checkedAt), kind: host.kind }) : t(mses.hostCheckedOff, { time: clock(host.checkedAt) })}
            </p>
          </div>
        ) : null}
        <form id="ses-access" onSubmit={connect}>
          <fieldset className="m-0 min-w-0 border-0 p-0">
            <legend className="sr-only">{t(mses.grantLegend)}</legend>
            <div className="grid gap-3 max-md:grid-cols-1!" style={fourCols}>
              {recommended === 'keys' ? [keysOption, roleOption] : [roleOption, keysOption]}
            </div>
          </fieldset>
        </form>
        <details className="text-[13px]">
          <summary className="cursor-pointer font-semibold">{t(mses.policyToggle)}</summary>
          <ScrollPre className="cn-pre mt-2" label={t(mses.policyLabel)}>
            {c.template?.policy}
          </ScrollPre>
        </details>
        {connected && !editing ? (
          <div className="flex flex-wrap items-center gap-3 rounded-lg border border-ok-line bg-ok-tint px-3 py-2.5" role="status">
            <Badge tone="ok" dot>
              {t(mses.connected)}
            </Badge>
            <span className="text-[13px]">
              <T
                msg={mses.connectedLine}
                values={{
                  access: c.access === 'role' ? 'role' : 'keys',
                  arn: c.roleArn,
                  hasKey: c.accessKey ? 'yes' : 'no',
                  key: c.accessKey,
                  hasUser: c.iamUser ? 'yes' : 'no',
                  user: c.iamUser,
                  hasTime: c.at ? 'yes' : 'no',
                  time: c.at ? clock(c.at) : '',
                  hasAccount: c.accountId ? 'yes' : 'no',
                  account: c.accountId ?? '',
                  mono,
                }}
              />
            </span>
          </div>
        ) : (
          <>
            {err ? (
              <Note tone="bad">
                <Ticks text={err.message} />
                {err.detail ? (
                  <>
                    {' '}
                    <span className="mono">{err.detail}</span>
                  </>
                ) : null}
              </Note>
            ) : null}
            <Row gap={10}>
              <Button type="submit" form="ses-access" variant="primary" loading={connecting} disabledReason={reason}>
                {err ? t(common.tryAgain) : t(mses.checkAccess)}
              </Button>
              {connected ? (
                <Button
                  onClick={() => {
                    setReplacing(false)
                    setAccess(c.access)
                  }}
                >
                  {t(common.cancel)}
                </Button>
              ) : null}
              <span className="k">{t(mses.checkAccessLine)}</span>
            </Row>
          </>
        )}
      </SectionCard>

      <SectionCard title={t(mses.regionTitle)}>
        <div className="flex flex-wrap items-end gap-3">
          <label className="lbl flex-[1_1_280px]">
            {t(mses.regionLabel)}
            <select className="in" value={connected ? c.region : region} onChange={(e) => pickRegion(e.target.value)} disabled={Boolean(reason) || a.region.isPending}>
              {c.regions.length
                ? c.regions.map((r) => (
                    <option key={r.region} value={r.region}>
                      {r.label}
                    </option>
                  ))
                : SES_REGIONS.map(([code, name]) => (
                    <option key={code} value={code}>{`${code} · ${name}`}</option>
                  ))}
            </select>
          </label>
          <span className="k flex-[1_1_280px]">{c.regions.length ? t(mses.regionReadHint) : t(mses.regionPickHint)}</span>
        </div>
      </SectionCard>

      <SectionCard title={t(mses.eventsTitle)}>
        <span className="k -mt-1.5">{t(mses.eventsLine)}</span>
        <fieldset className="m-0 min-w-0 border-0 p-0" disabled={Boolean(reason) || a.events.isPending}>
          <legend className="sr-only">{t(mses.eventsLegend)}</legend>
          <div className="grid gap-3 max-md:grid-cols-1!" style={fourCols}>
            <EventsOption
              value="webhook"
              checked={events === 'webhook'}
              onPick={() => pickEvents('webhook')}
              title={t(mses.eventsWebhook)}
              line={<T msg={mses.eventsWebhookLine} values={{ url: c.eventsUrl, mono }} />}
            />
            <EventsOption value="queue" checked={events === 'queue'} onPick={() => pickEvents('queue')} title={t(mses.eventsQueue)} line={t(mses.eventsQueueLine)} />
          </div>
        </fieldset>
        {connected ? <span className="k">{t(mses.eventsChangeNote)}</span> : null}
      </SectionCard>

      {c.provisioning.length ? (
        <section className="card flex flex-col" aria-labelledby="prov-h">
          <div className="flex flex-wrap items-center justify-between gap-2 px-5 py-3.5">
            <h3 className="h2" id="prov-h">
              {t(mses.provisioningTitle, { done: count(done), total: count(c.provisionTotal || c.provisioning.length) })}
            </h3>
            <span className="k">{t(mses.provisioningLine)}</span>
          </div>
          <ol className="m-0 list-none p-0">
            {c.provisioning.map((r) => (
              <li key={r.key}>
                <ProvisionRow row={r} setup={setup} onRetry={() => a.provision.mutate(undefined, { onError: failed(t(mses.retryFailed)) })} retrying={a.provision.isPending} reason={reason} />
              </li>
            ))}
          </ol>
        </section>
      ) : null}

      {c.sandbox ? <SandboxCard setup={setup} sandbox={c.sandbox} reason={reason} /> : null}

      <WizardFooter
        back={
          <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            {t(common.back)}
          </ButtonLink>
        }
        caption={caption}
        captionId="ses-continue-why"
      >
        <Button onClick={saveLater} disabledReason={reason}>
          {t(mn.saveLater)}
        </Button>
        <Button
          variant="primary"
          className={continueClass}
          onClick={() => go('domain')}
          aria-describedby={caption ? 'ses-continue-why' : undefined}
          disabledReason={c.canContinue ? undefined : connected ? t(mses.eventsFirst) : t(mses.awsFirst)}
        >
          {t(mn.continueDomain)}
        </Button>
      </WizardFooter>
    </>
  )
}

/** AWS's own names for where ReputeCast runs. */
function hostKind(kind: string): string {
  return { ec2: 'EC2', ecs: 'ECS', eks: 'EKS', lambda: 'Lambda' }[kind] ?? kind.toUpperCase()
}

/** One way SES events can reach ReputeCast (PR-4): webhook or queue. */
function EventsOption({ value, checked, onPick, title, line }: { value: string; checked: boolean; onPick: () => void; title: string; line: ReactNode }) {
  const head = <span className="font-semibold">{title}</span>
  const body = <span className="k break-words">{line}</span>
  return (
    <label className="opt">
      <input type="radio" name="ses-events" value={value} checked={checked} onChange={onPick} />
      <span className="min-w-0">
        {head}
        <br />
        {body}
      </span>
    </label>
  )
}

/** One way to grant access (`label.opt`), laid out as on the board: title (and "Recommended here"), description, then its controls. */
function AccessOption({
  checked,
  onPick,
  disabled,
  title,
  recommended,
  description,
  children,
}: {
  checked: boolean
  onPick: () => void
  disabled: boolean
  title: string
  recommended?: string
  description: string
  children: ReactNode
}) {
  return (
    <label className="opt">
      <input type="radio" name="ses-access" checked={checked} onChange={onPick} disabled={disabled} />
      <span className="min-w-0">
        <span className="font-semibold">{title}</span>
        {recommended ? (
          <>
            {' '}
            <Badge tone="ok" dot>
              {recommended}
            </Badge>
          </>
        ) : null}
        <br />
        <span className="k">{description}</span>
        <br />
        {children}
      </span>
    </label>
  )
}

function ProvisionRow({ row: r, setup, onRetry, retrying, reason }: { row: Schemas['ConnectProvisionRow']; setup: Schemas['ConnectSetup']; onRetry: () => void; retrying: boolean; reason?: string }) {
  const t = useT()
  if (r.status === 'done') {
    return (
      <StatusRow badge={<Badge tone="ok" dot>{t(m.stateDone)}</Badge>} aside={r.at ? <span className="k whitespace-nowrap">{clockSeconds(r.at)}</span> : null}>
        <Ticks text={r.text} />
        {r.note ? <span className="k block">{r.note}</span> : null}
      </StatusRow>
    )
  }
  if (r.status === 'deferred') {
    return (
      <StatusRow badge={<Badge tone="neutral" dot>{t(mn.waiting)}</Badge>} aside={<span className="k whitespace-nowrap">{t(mses.waitingDomain)}</span>}>
        <Ticks text={r.text} />
      </StatusRow>
    )
  }
  if (r.status === 'failed') {
    const role = setup.connection.roleArn.split('/').pop()
    const iam = setup.connection.access === 'role' && role ? `https://console.aws.amazon.com/iam/home#/roles/details/${encodeURIComponent(role)}?section=permissions` : 'https://console.aws.amazon.com/iam/home#/users'
    const policyProblem = !r.key.startsWith('event_')
    return (
      <StatusRow
        className="bg-bad-tint"
        badge={<Badge tone="bad" dot>{t(m.stateFailed)}</Badge>}
        aside={
          <>
            <Button size="sm" variant="primary" onClick={onRetry} loading={retrying} disabledReason={reason}>
              {t(common.retry)}
            </Button>
            {policyProblem ? (
              <ButtonAnchor size="sm" href={iam}>
                {t(mses.fixPolicy)}
              </ButtonAnchor>
            ) : null}
          </>
        }
      >
        <Ticks text={r.text} />
        {r.error ? (
          <>
            {' · '}
            <span className="mono text-bad-fg">{r.error}</span>
          </>
        ) : null}
        {r.hint ? (
          <>
            {' · '}
            <Ticks text={r.hint} />
          </>
        ) : null}
      </StatusRow>
    )
  }
  return (
    <div className="cn-row">
      <span className="spin" aria-hidden="true" />
      <span className="cn-row-text min-w-0">
        <Ticks text={r.text} />
      </span>
      <span className="cn-row-aside k ml-auto">{r.status === 'running' ? t(mses.running) : t(mses.waiting)}</span>
    </div>
  )
}

function SandboxCard({ setup, sandbox: sb, reason }: { setup: Schemas['ConnectSetup']; sandbox: Schemas['ConnectSandbox']; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const c = setup.connection
  const quota = c.regions.find((r) => r.region === sb.region)
  const [website, setWebsite] = useState(sb.website)
  const [mailType, setMailType] = useState(sb.mailType)
  const [useCase, setUseCase] = useState(sb.useCase)
  const [volume, setVolume] = useState(sb.dailyVolume ? count(sb.dailyVolume) : '')
  const [language, setLanguage] = useState(sb.language)
  const request = () =>
    a.production.mutate(
      { region: sb.region, website: website.trim(), mailType, useCase: useCase.trim(), dailyVolume: Number(volume.replace(/[^\d]/g, '')) || 0, language },
      {
        onSuccess: () => toast({ title: t(mses.requestedToast, { region: sb.region }), detail: t(mses.requestedDetail) }),
        onError: failed(t(mses.requestFailed), request),
      },
    )
  return (
    <section className="card flex flex-col border-[#FCD34D]" aria-label={t(mses.sandboxBadge, { region: sb.region })}>
      <div className="flex flex-wrap items-start gap-3.5 rounded-t-[10px] bg-[#FFFBEB] px-5 py-4">
        <Badge tone="warn" dot className="h-6">
          {t(mses.sandboxBadge, { region: sb.region })}
        </Badge>
        <span className="flex-[1_1_320px] text-[13px] text-[#78350F]">
          {t(mses.sandboxLine, {
            selected: sb.selected ? 'yes' : 'no',
            region: sb.region,
            quota: quota ? 'yes' : 'no',
            max24h: quota ? count(quota.max24h) : '',
            rate: quota ? rateText(quota.maxRate) : '',
          })}
        </span>
      </div>
      <div className="grid gap-3 px-5 py-4 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(280px, 100%), 1fr))' }}>
        <label className="lbl">
          {t(mses.website)}
          <input className="in" value={website} onChange={(e) => setWebsite(e.target.value)} disabled={Boolean(reason)} inputMode="url" />
        </label>
        <label className="lbl">
          {t(mses.mailType)}
          <select className="in" value={mailType} onChange={(e) => setMailType(e.target.value as typeof mailType)} disabled={Boolean(reason)}>
            <option value="marketing">{t(mses.mailMarketing)}</option>
            <option value="transactional">{t(mses.mailTransactional)}</option>
          </select>
        </label>
        <label className="lbl col-span-full">
          {t(mses.useCase)}
          <textarea className="in h-[84px] min-h-0 resize-y" value={useCase} onChange={(e) => setUseCase(e.target.value)} disabled={Boolean(reason)} />
        </label>
        <label className="lbl">
          {t(mses.dailyVolume)}
          <input className="in" value={volume} onChange={(e) => setVolume(e.target.value)} disabled={Boolean(reason)} inputMode="numeric" />
        </label>
        <label className="lbl">
          {t(mses.contactLanguage)}
          <select className="in" value={language} onChange={(e) => setLanguage(e.target.value as typeof language)} disabled={Boolean(reason)}>
            <option value="en">{t(mses.langEnglish)}</option>
            <option value="ja">{t(mses.langJapanese)}</option>
          </select>
        </label>
      </div>
      <div className="flex flex-wrap gap-2.5 px-5 pb-4">
        {sb.requestedAt ? (
          <Badge tone="info" dot>
            {t(mses.requested, { time: clock(sb.requestedAt) })}
          </Badge>
        ) : (
          <Button onClick={request} loading={a.production.isPending} busyLabel={t(mses.requesting)} disabledReason={reason}>
            {t(mses.requestAccess, { region: sb.region })}
          </Button>
        )}
        <span className="k self-center">{sb.selected ? t(mses.awsAnswersSelected) : t(mses.awsAnswersOther, { region: c.region })}</span>
      </div>
    </section>
  )
}
