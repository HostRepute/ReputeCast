import { useState, type FormEvent } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { failed, StateBadge, StatusRow, useSetupActions, type StepProps } from './parts'
import { Ticks } from './text'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

type Form = Schemas['ConnectSMTPForm']

const grid = { gridTemplateColumns: 'repeat(auto-fit, minmax(min(220px, 100%), 1fr))' }
const gridClass = 'grid gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!'

const ENCRYPTION: { value: Form['tls']; label: string; port: number }[] = [
  { value: 'starttls', label: 'STARTTLS · 587', port: 587 },
  { value: 'tls', label: 'Implicit TLS · 465', port: 465 },
  { value: 'none', label: 'None', port: 25 },
]

const MAILBOX_PORTS: Record<'imap' | 'pop3', { port: number; tls: 'tls' | 'starttls'; label: string }[]> = {
  imap: [
    { port: 993, tls: 'tls', label: '993 · TLS' },
    { port: 143, tls: 'starttls', label: '143 · STARTTLS' },
  ],
  pop3: [
    { port: 995, tls: 'tls', label: '995 · TLS' },
    { port: 110, tls: 'starttls', label: '110 · STARTTLS' },
  ],
}

/** 7.2 step 2 for an SMTP server (SetupSMTP): outgoing server and bounce mailbox. */
export function ConnectionSMTP({ setup, go, saveLater, reason }: StepProps) {
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const c = setup.connection
  const f = c.smtpForm
  const [host, setHost] = useState(f?.host ?? '')
  const [port, setPort] = useState(f?.port || 587)
  const [tls, setTls] = useState<Form['tls']>(f?.tls ?? 'starttls')
  const [username, setUsername] = useState(f?.username ?? '')
  const [password, setPassword] = useState('')
  const [maxRate, setMaxRate] = useState(f?.maxRate ?? '')
  const [dailyCap, setDailyCap] = useState(f?.dailyCap ?? '')
  const connected = c.status === 'connected'
  const err = c.status === 'failed' ? c.error : undefined
  const test = (e?: FormEvent) => {
    e?.preventDefault()
    a.connect.mutate(
      { smtp: { host: host.trim(), port, tls, username: username.trim() || undefined, password: password || undefined, maxRate: maxRate.trim() || undefined, dailyCap: dailyCap.trim() || undefined } },
      { onSuccess: () => setPassword(''), onError: failed("Couldn't test the SMTP server", () => test()) },
    )
  }
  const probe = c.smtp
  return (
    <>
      <StepHeading title={setup.info.heading}>{setup.info.lead}</StepHeading>

      <section className="card flex flex-col gap-3.5 px-5 py-4.5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="h2">1. Outgoing server</h3>
          <span className="k">Also fine for a Postal server&apos;s SMTP endpoint</span>
        </div>
        <form id="smtp-server" className={gridClass} style={grid} onSubmit={test}>
          <label className="lbl col-span-2 max-md:col-span-full">
            Host
            <input className="in mono" value={host} onChange={(e) => setHost(e.target.value)} aria-invalid={err?.field === 'host' || undefined} disabled={Boolean(reason)} autoComplete="off" />
          </label>
          <label className="lbl">
            Port
            <select className="in" value={port} onChange={(e) => setPort(Number(e.target.value))} disabled={Boolean(reason)}>
              {[...new Set([587, 465, 25, 2525, port])].map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            Encryption
            <select
              className="in"
              value={tls}
              onChange={(e) => {
                const next = ENCRYPTION.find((x) => x.value === e.target.value)!
                setTls(next.value)
                if (next.value !== 'none') setPort(next.port)
              }}
              disabled={Boolean(reason)}
            >
              {ENCRYPTION.map((x) => (
                <option key={x.value} value={x.value}>
                  {x.label}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            Username
            <input className="in mono" value={username} onChange={(e) => setUsername(e.target.value)} disabled={Boolean(reason)} autoComplete="off" />
          </label>
          <label className="lbl">
            Password
            <input
              className="in"
              type="password"
              placeholder="Stored encrypted, never shown again"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={err?.field === 'password' || undefined}
              disabled={Boolean(reason)}
              autoComplete="new-password"
            />
          </label>
          <label className="lbl">
            Max rate you&apos;re allowed
            <input className="in" value={maxRate} placeholder="5 per second" onChange={(e) => setMaxRate(e.target.value)} disabled={Boolean(reason)} />
            <span className="k font-medium">SMTP servers don&apos;t publish limits; ReputeCast throttles to this and backs off on 4xx replies.</span>
          </label>
          <label className="lbl">
            Daily cap
            <input className="in" value={dailyCap} placeholder="20,000" onChange={(e) => setDailyCap(e.target.value)} disabled={Boolean(reason)} inputMode="numeric" />
          </label>
        </form>
        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit" form="smtp-server" loading={a.connect.isPending} busyLabel="Testing…" disabledReason={reason ?? (host.trim() ? undefined : 'Enter the host first.')}>
            Test connection
          </Button>
          {probe ? (
            probe.ok ? (
              <Badge tone="ok" dot>{`Connected at ${clock(probe.at)} · ${probe.summary}`}</Badge>
            ) : (
              <Badge tone="bad" dot>
                Couldn&apos;t connect
              </Badge>
            )
          ) : null}
        </div>
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
        {probe?.transcript ? <div className="mono k rounded-lg bg-ground px-3 py-2.5 text-[12px] text-ink-nav">{probe.transcript}</div> : null}
      </section>

      <MailboxCard setup={setup} reason={reason} connected={connected} />

      <section className="card flex flex-col" aria-labelledby="smtp-can-h">
        <div className="px-5 py-3.5">
          <h3 className="h2" id="smtp-can-h">
            What SMTP can and can&apos;t do here
          </h3>
        </div>
        <StatusRow badge={<StateBadge state="yes" />}>Bounces and complaints via the bounce mailbox, usually within a few minutes of the event</StatusRow>
        <StatusRow badge={<StateBadge state="yes" />}>Shared suppression list, auto-pause thresholds, per-mailbox-provider throttling</StatusRow>
        <StatusRow badge={<StateBadge state="partial" />}>Domain setup: ReputeCast tells you the DNS records; it can&apos;t add them to the server for you</StatusRow>
        <StatusRow badge={<StateBadge state="partial" />}>Delivered events: inferred from the SMTP 250 reply, not from the mailbox provider</StatusRow>
        <StatusRow badge={<StateBadge state="no" />}>Quota and rate discovery, batch sending, sandbox mode</StatusRow>
      </section>

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
          disabledReason={c.canContinue ? undefined : connected ? 'Test the bounce mailbox first.' : 'Test the SMTP server first.'}
        >
          Continue to domain verification
        </Button>
      </WizardFooter>
    </>
  )
}

function MailboxCard({ setup, reason, connected }: { setup: Schemas['ConnectSetup']; reason?: string; connected: boolean }) {
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const f = setup.connection.smtpForm
  const [address, setAddress] = useState(f?.returnPath ?? '')
  const [protocol, setProtocol] = useState<'imap' | 'pop3'>(f?.protocol ?? 'imap')
  const [host, setHost] = useState(f?.mailboxHost ?? '')
  const [port, setPort] = useState(f?.mailboxPort || 993)
  const [username, setUsername] = useState(f?.mailboxUser ?? '')
  const [password, setPassword] = useState('')
  const [folder, setFolder] = useState(f?.folder || 'INBOX')
  const [pollEvery, setPollEvery] = useState<'1m' | '5m' | '15m'>(f?.pollEvery ?? '1m')
  const [after, setAfter] = useState<'move' | 'delete' | 'leave'>(f?.afterProcessing ?? 'move')
  const ports = MAILBOX_PORTS[protocol]
  const tls = ports.find((p) => p.port === port)?.tls ?? f?.mailboxTls ?? 'tls'
  const mb = setup.connection.mailbox
  const test = (e?: FormEvent) => {
    e?.preventDefault()
    a.mailbox.mutate(
      {
        address: address.trim(),
        protocol,
        host: host.trim(),
        port,
        tls,
        username: username.trim(),
        password: password || undefined,
        folder: folder.trim() || undefined,
        pollEvery,
        afterProcessing: after,
      },
      { onSuccess: () => setPassword(''), onError: failed("Couldn't test the bounce mailbox", () => test()) },
    )
  }
  return (
    <section className="card flex flex-col gap-3.5 px-5 py-4.5" aria-labelledby="mailbox-h">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="h2" id="mailbox-h">
          2. Bounce mailbox
        </h3>
        <Badge tone="info">Required for SMTP</Badge>
      </div>
      <span className="text-[13px] text-ink-2">
        Mailbox providers send delivery failures (DSNs) back to the return-path address. ReputeCast polls that mailbox, parses each DSN and ARF report, and turns it into the same normalized bounce or
        complaint event other providers deliver by webhook.
      </span>
      <form id="smtp-mailbox" className={gridClass} style={grid} onSubmit={test}>
        <label className="lbl">
          Return-path address
          <input className="in mono" type="email" value={address} onChange={(e) => setAddress(e.target.value)} disabled={Boolean(reason)} autoComplete="off" />
        </label>
        <label className="lbl">
          Protocol
          <select
            className="in"
            value={protocol}
            onChange={(e) => {
              const p = e.target.value as 'imap' | 'pop3'
              setProtocol(p)
              setPort(MAILBOX_PORTS[p][0]!.port)
            }}
            disabled={Boolean(reason)}
          >
            <option value="imap">IMAP (recommended)</option>
            <option value="pop3">POP3</option>
          </select>
        </label>
        <label className="lbl">
          Mail server
          <input className="in mono" value={host} onChange={(e) => setHost(e.target.value)} disabled={Boolean(reason)} autoComplete="off" />
        </label>
        <label className="lbl">
          Port
          <select className="in" value={port} onChange={(e) => setPort(Number(e.target.value))} disabled={Boolean(reason)}>
            {ports.map((p) => (
              <option key={p.port} value={p.port}>
                {p.label}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          Username
          <input className="in mono" value={username} onChange={(e) => setUsername(e.target.value)} disabled={Boolean(reason)} autoComplete="off" />
        </label>
        <label className="lbl">
          Password
          <input className="in" type="password" placeholder="Stored encrypted" value={password} onChange={(e) => setPassword(e.target.value)} disabled={Boolean(reason)} autoComplete="new-password" />
        </label>
        <label className="lbl">
          Folder
          <input className="in" value={folder} onChange={(e) => setFolder(e.target.value)} disabled={Boolean(reason)} />
        </label>
        <label className="lbl">
          Poll every
          <select className="in" value={pollEvery} onChange={(e) => setPollEvery(e.target.value as typeof pollEvery)} disabled={Boolean(reason)}>
            <option value="1m">1 minute</option>
            <option value="5m">5 minutes</option>
            <option value="15m">15 minutes</option>
          </select>
        </label>
        <label className="lbl">
          After processing
          <select className="in" value={after} onChange={(e) => setAfter(e.target.value as typeof after)} disabled={Boolean(reason)}>
            <option value="move">Move to &quot;Processed&quot;</option>
            <option value="delete">Delete</option>
            <option value="leave">Leave in place</option>
          </select>
        </label>
      </form>
      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" form="smtp-mailbox" loading={a.mailbox.isPending} disabledReason={reason ?? (connected ? undefined : 'Test the outgoing server first.')}>
          Test mailbox
        </Button>
        {mb ? (
          mb.ok ? (
            <Badge tone="ok" dot>{`Connected · ${mb.summary}`}</Badge>
          ) : (
            <Badge tone="bad" dot>
              Couldn&apos;t read the mailbox
            </Badge>
          )
        ) : null}
      </div>
      {mb && !mb.ok && mb.error ? (
        <Note tone="bad">
          <Ticks text={mb.error} />
        </Note>
      ) : null}
    </section>
  )
}
