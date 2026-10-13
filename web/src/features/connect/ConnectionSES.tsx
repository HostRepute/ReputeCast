import { useState, type FormEvent, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Row } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { clock, clockSeconds, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { failed, SectionCard, StatusRow, useSetupActions, type StepProps } from './parts'
import { Ticks } from './text'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

/** SES regions offered before the account has been read. */
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

/** 7.2 step 2 for Amazon SES (SetupSES): AWS access, region, provisioning, sandbox. */
export function ConnectionSES({ setup, go, saveLater, reason }: StepProps) {
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const c = setup.connection
  const connected = c.status === 'connected'
  const [access, setAccess] = useState<'role' | 'keys'>(c.access || 'role')
  const [roleArn, setRoleArn] = useState(c.roleArn)
  const [keyId, setKeyId] = useState('')
  const [secret, setSecret] = useState('')
  const [region, setRegion] = useState(c.region || 'us-east-1')
  const connecting = a.connect.isPending
  const connect = (e?: FormEvent) => {
    e?.preventDefault()
    a.connect.mutate(
      { ses: access === 'role' ? { access, roleArn: roleArn.trim(), region } : { access, accessKeyId: keyId.trim(), secretAccessKey: secret, region } },
      { onError: failed("Couldn't check the AWS access", () => connect()) },
    )
  }
  const pickRegion = (r: string) => {
    setRegion(r)
    if (connected) a.region.mutate({ region: r }, { onError: failed("Couldn't move to that region") })
  }
  const err = c.error
  const done = c.provisioning.filter((r) => r.status === 'done').length
  return (
    <>
      <StepHeading title={setup.info.heading}>{setup.info.lead}</StepHeading>

      <SectionCard title="1. How to grant access">
        <form className="grid gap-3 max-md:grid-cols-1!" style={fourCols} onSubmit={connect} id="ses-access">
          <AccessOption
            checked={access === 'role'}
            onPick={() => setAccess('role')}
            disabled={Boolean(reason) || connecting}
            title="CloudFormation stack (recommended)"
            description="Opens the AWS console with a stack that creates an IAM role with the exact policy below. No long-lived keys. About 2 minutes."
          >
              <a href={c.template?.launchUrl} className="btn sm mt-2 underline decoration-1 underline-offset-2" target="_blank" rel="noopener noreferrer">
                Launch stack in AWS
              </a>
              {!connected && access === 'role' ? (
                <span className="mt-2 flex flex-col gap-1">
                  <input
                    className="in mono w-full"
                    placeholder={`arn:aws:iam::123456789012:role/${c.template?.roleName || 'ReputeCastSES'}`}
                    aria-label="Role ARN, from the stack's RoleArn output"
                    aria-invalid={err?.field === 'role_arn' || undefined}
                    value={roleArn}
                    onChange={(e) => setRoleArn(e.target.value)}
                    disabled={Boolean(reason)}
                  />
                  <span className="k">Paste the stack&apos;s RoleArn output. External id {c.externalId ? <span className="mono">{c.externalId}</span> : null} is filled in for you.</span>
                </span>
              ) : null}
          </AccessOption>
          <AccessOption
            checked={access === 'keys'}
            onPick={() => setAccess('keys')}
            disabled={Boolean(reason) || connecting}
            title="IAM access keys"
            description="Paste an access key id and secret for a user with the policy below. Keys are encrypted at rest and never shown again."
          >
              <span className="mt-2 grid grid-cols-2 gap-2">
                {/* Disabled until picked, but white as on the board. */}
                <input
                  className="in mono min-w-0 bg-white"
                  placeholder="AKIA…"
                  aria-label="Access key id"
                  aria-invalid={err?.field === 'access_key_id' || undefined}
                  disabled={access !== 'keys' || Boolean(reason)}
                  value={keyId}
                  onChange={(e) => setKeyId(e.target.value)}
                  autoComplete="off"
                />
                <input
                  className="in mono min-w-0 bg-white"
                  type="password"
                  placeholder="Secret access key"
                  aria-label="Secret access key"
                  disabled={access !== 'keys' || Boolean(reason)}
                  value={secret}
                  onChange={(e) => setSecret(e.target.value)}
                  autoComplete="new-password"
                />
              </span>
          </AccessOption>
        </form>
        <details className="text-[13px]">
          <summary className="cursor-pointer font-semibold">Show the IAM policy (least privilege)</summary>
          <pre className="cn-pre mt-2">{c.template?.policy}</pre>
        </details>
        {connected ? (
          <div className="flex flex-wrap items-center gap-3 rounded-lg border border-ok-line bg-ok-tint px-3 py-2.5" role="status">
            <Badge tone="ok" dot>
              Connected
            </Badge>
            <span className="text-[13px]">
              {c.access === 'role' ? (
                <>
                  Role <span className="mono">{c.roleArn}</span> assumed
                </>
              ) : (
                'IAM access keys checked'
              )}
              {c.at ? ` at ${clock(c.at)}` : ''}
              {c.accountId ? ` · account ${c.accountId}` : ''}
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
                {err ? 'Try again' : 'Check access'}
              </Button>
              <span className="k">ReputeCast checks the access with read-only calls first (GetAccount), then stores it encrypted.</span>
            </Row>
          </>
        )}
      </SectionCard>

      <SectionCard title="2. Region">
        <div className="flex flex-wrap items-end gap-3">
          <label className="lbl flex-[1_1_280px]">
            SES region
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
          <span className="k flex-[1_1_280px]">
            {c.regions.length
              ? 'ReputeCast read your account in each region. Pick the one where you already have production access, or the one closest to your recipients.'
              : 'Pick the region where you already have production access, or the one closest to your recipients. ReputeCast reads your account in each region once connected.'}
          </span>
        </div>
      </SectionCard>

      {c.provisioning.length ? (
        <section className="card flex flex-col" aria-labelledby="prov-h">
          <div className="flex flex-wrap items-center justify-between gap-2 px-5 py-3.5">
            <h3 className="h2" id="prov-h">{`3. Provisioning · ${done} of ${c.provisionTotal || c.provisioning.length} done`}</h3>
            <span className="k">SNS topics and configuration sets are created for you, with retry</span>
          </div>
          {c.provisioning.map((r) => (
            <ProvisionRow key={r.key} row={r} setup={setup} onRetry={() => a.provision.mutate(undefined, { onError: failed("Couldn't retry provisioning") })} retrying={a.provision.isPending} reason={reason} />
          ))}
        </section>
      ) : null}

      {c.sandbox ? <SandboxCard setup={setup} sandbox={c.sandbox} reason={reason} /> : null}

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
        <Button variant="primary" className={continueClass} onClick={() => go('domain')} disabledReason={c.canContinue ? undefined : 'Connect your AWS account first.'}>
          Continue to domain verification
        </Button>
      </WizardFooter>
    </>
  )
}

/** One way to grant access (`label.opt`), laid out as on the board: title, description, then its controls. */
function AccessOption({
  checked,
  onPick,
  disabled,
  title,
  description,
  children,
}: {
  checked: boolean
  onPick: () => void
  disabled: boolean
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <label className="opt">
      <input type="radio" name="ses-access" checked={checked} onChange={onPick} disabled={disabled} />
      <span className="min-w-0">
        <span className="font-semibold">{title}</span>
        <br />
        <span className="k">{description}</span>
        <br />
        {children}
      </span>
    </label>
  )
}

function ProvisionRow({ row: r, setup, onRetry, retrying, reason }: { row: Schemas['ConnectProvisionRow']; setup: Schemas['ConnectSetup']; onRetry: () => void; retrying: boolean; reason?: string }) {
  if (r.status === 'done') {
    return (
      <StatusRow badge={<Badge tone="ok">Done</Badge>} aside={r.at ? <span className="k">{clockSeconds(r.at)}</span> : null}>
        <Ticks text={r.text} />
      </StatusRow>
    )
  }
  if (r.status === 'failed') {
    const role = setup.connection.roleArn.split('/').pop()
    const iam = setup.connection.access === 'role' && role ? `https://console.aws.amazon.com/iam/home#/roles/details/${encodeURIComponent(role)}?section=permissions` : 'https://console.aws.amazon.com/iam/home#/users'
    return (
      <StatusRow
        className="bg-bad-tint"
        badge={<Badge tone="bad">Failed</Badge>}
        aside={
          <>
            <Button size="sm" variant="primary" onClick={onRetry} loading={retrying} disabledReason={reason}>
              Retry
            </Button>
            <ButtonAnchor size="sm" href={iam}>
              Fix policy
            </ButtonAnchor>
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
        {r.hint ? ` · ${r.hint}` : null}
      </StatusRow>
    )
  }
  return (
    <div className="cn-row">
      <span className="spin" aria-hidden="true" />
      <span className="min-w-0">
        <Ticks text={r.text} />
      </span>
      <span className="k ml-auto">{r.status === 'running' ? 'running' : 'waiting'}</span>
    </div>
  )
}

function SandboxCard({ setup, sandbox: sb, reason }: { setup: Schemas['ConnectSetup']; sandbox: Schemas['ConnectSandbox']; reason?: string }) {
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const c = setup.connection
  const quota = c.regions.find((r) => r.region === sb.region)
  const [website, setWebsite] = useState(sb.website)
  const [mailType, setMailType] = useState(sb.mailType)
  const [useCase, setUseCase] = useState(sb.useCase)
  const [volume, setVolume] = useState(sb.dailyVolume ? count(sb.dailyVolume) : '')
  const [language, setLanguage] = useState(sb.language)
  const limits = quota ? `${count(quota.max24h)} emails per day, ${quota.maxRate} per second, and only to verified addresses` : 'a small daily quota, and only to verified addresses'
  const request = () =>
    a.production.mutate(
      { region: sb.region, website: website.trim(), mailType, useCase: useCase.trim(), dailyVolume: Number(volume.replace(/[^\d]/g, '')) || 0, language },
      {
        onSuccess: () => toast({ title: `Production access requested for ${sb.region}`, detail: 'AWS usually answers within 24 hours.' }),
        onError: failed("Couldn't file the production-access request", request),
      },
    )
  return (
    <section className="card flex flex-col border-[#FCD34D]" aria-label={`Sandbox detected in ${sb.region}`}>
      <div className="flex flex-wrap items-start gap-3.5 rounded-t-[10px] bg-[#FFFBEB] px-5 py-4">
        <Badge tone="warn" dot className="h-6">
          {`Sandbox detected in ${sb.region}`}
        </Badge>
        <span className="flex-[1_1_320px] text-[13px] text-[#78350F]">
          {sb.selected ? `${sb.region} is in the SES sandbox: ${limits}.` : `If you switch to ${sb.region}, that region is in the SES sandbox: ${limits}.`} ReputeCast can file the
          production-access request for you.
        </span>
      </div>
      <div className="grid gap-3 px-5 py-4 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(280px, 100%), 1fr))' }}>
        <label className="lbl">
          Website
          <input className="in" value={website} onChange={(e) => setWebsite(e.target.value)} disabled={Boolean(reason)} inputMode="url" />
        </label>
        <label className="lbl">
          Mail type
          <select className="in" value={mailType} onChange={(e) => setMailType(e.target.value as typeof mailType)} disabled={Boolean(reason)}>
            <option value="marketing">Marketing (newsletter)</option>
            <option value="transactional">Transactional</option>
          </select>
        </label>
        <label className="lbl col-span-full">
          Use case description (prefilled from your workspace)
          <textarea className="in h-[84px] min-h-0 resize-y" value={useCase} onChange={(e) => setUseCase(e.target.value)} disabled={Boolean(reason)} />
        </label>
        <label className="lbl">
          Expected daily volume
          <input className="in" value={volume} onChange={(e) => setVolume(e.target.value)} disabled={Boolean(reason)} inputMode="numeric" />
        </label>
        <label className="lbl">
          Contact language
          <select className="in" value={language} onChange={(e) => setLanguage(e.target.value as typeof language)} disabled={Boolean(reason)}>
            <option value="en">English</option>
            <option value="ja">Japanese</option>
          </select>
        </label>
      </div>
      <div className="flex flex-wrap gap-2.5 px-5 pb-4">
        {sb.requestedAt ? (
          <Badge tone="info" dot>{`Requested ${clock(sb.requestedAt)}`}</Badge>
        ) : (
          <Button onClick={request} loading={a.production.isPending} busyLabel="Requesting…" disabledReason={reason}>
            {`Request production access for ${sb.region}`}
          </Button>
        )}
        <span className="k self-center">
          AWS usually answers within 24 hours.{sb.selected ? ' Meanwhile you can send to verified addresses.' : ` You can keep going with ${c.region} meanwhile.`}
        </span>
      </div>
    </section>
  )
}
