import { useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { useWorkspace } from '@/lib/session'
import { failed, type StepProps } from './parts'
import { Ticks } from './text'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

type Status = Schemas['ConnectTestStatus']

const testPath = '/api/v1/workspaces/{workspace}/connect/setups/{setupId}/test' as const

const SEEDS: { key: string; label: string; placeholder: string }[] = [
  { key: 'gmail', label: 'Gmail', placeholder: 'you@gmail.com' },
  { key: 'outlook', label: 'Outlook', placeholder: 'you@outlook.com' },
  { key: 'yahoo', label: 'Yahoo', placeholder: 'you@yahoo.com' },
  { key: 'icloud', label: 'iCloud', placeholder: 'you@icloud.com' },
]

/** 7.2 step 4 (SetupTestSend): send through the provider and watch the events come back. */
export function TestStep({ setup, go, reason }: StepProps) {
  const { slug } = useWorkspace()
  const init = { params: { path: { workspace: slug, setupId: setup.id } } }
  const q = $api.useQuery('get', testPath, init, { refetchInterval: (query) => (query.state.data?.listening ? 2000 : false) })
  const info = setup.info
  return (
    <>
      <StepHeading title="Send a test and watch it land">
        {info.flow === 'smtp'
          ? 'This proves the whole loop: ReputeCast hands the message to the SMTP server, the server accepts it, and bounces come back through the bounce mailbox.'
          : `This proves the whole loop: ReputeCast hands the message to ${info.short}, ${info.short} delivers it, and the delivery event comes back${info.webhooks === 'SNS' ? ' through SNS' : ''}.`}
      </StepHeading>
      {q.data ? (
        <TestPanels setup={setup} status={q.data} reason={reason} />
      ) : q.isError ? (
        <ErrorPanel title="Couldn't load the test" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <div className="flex flex-wrap items-start gap-5" aria-busy="true">
          <span className="vh" role="status">
            Loading the test…
          </span>
          <section className="card flex flex-[1_1_360px] flex-col gap-3 px-5 py-4.5">
            <Skeleton w="40%" h={16} />
            <Skeleton w="100%" h={36} />
            <Skeleton w="100%" h={36} />
          </section>
          <section className="card flex flex-[1_1_420px] flex-col gap-3 px-5 py-4.5">
            <Skeleton w="50%" h={16} />
            <Skeleton w="100%" h={160} />
          </section>
        </div>
      )}
      <WizardFooter back={<Button onClick={() => go('domain')}>Back</Button>}>
        <Button variant="primary" className={continueClass} onClick={() => go('streams')}>
          Continue to streams
        </Button>
      </WizardFooter>
    </>
  )
}

function TestPanels({ setup, status: s, reason }: { setup: Schemas['ConnectSetup']; status: Status; reason?: string }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const init = { params: { path: { workspace: slug, setupId: setup.id } } }
  const key = $api.queryOptions('get', testPath, init).queryKey
  const [senderId, setSenderId] = useState(s.senderId)
  const [to, setTo] = useState(s.to)
  const [subject, setSubject] = useState(s.subject || `ReputeCast test from ${setup.info.name}`)
  const [seedsOpen, setSeedsOpen] = useState(false)
  const send = $api.useMutation('post', testPath)
  const run = (body: Schemas['ConnectTestInput'], done?: () => void) =>
    send.mutate(
      { ...init, body },
      {
        onSuccess: (data) => {
          qc.setQueryData(key, data)
          done?.()
        },
        onError: failed("Couldn't send the test", () => run(body, done)),
      },
    )
  const submit = (e?: FormEvent) => {
    e?.preventDefault()
    run({ senderId, to: to.trim(), subject: subject.trim() })
  }
  const auth = s.auth
  return (
    <div className="flex flex-wrap items-start gap-5">
      <section className="card flex flex-[1_1_360px] flex-col gap-3 px-5 py-4.5" aria-labelledby="test-h">
        <h3 className="h2" id="test-h">
          Test message
        </h3>
        <form id="test-form" className="contents" onSubmit={submit}>
          <label className="lbl">
            From
            <select className="in" value={senderId} onChange={(e) => setSenderId(e.target.value)} disabled={Boolean(reason)}>
              {s.senders.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            To
            <input className="in" type="email" value={to} onChange={(e) => setTo(e.target.value)} disabled={Boolean(reason)} autoComplete="email" />
          </label>
          <label className="lbl">
            Subject
            <input className="in" value={subject} onChange={(e) => setSubject(e.target.value)} disabled={Boolean(reason)} />
          </label>
        </form>
        <div className="flex flex-wrap gap-2.5">
          <Button type="submit" form="test-form" variant="primary" loading={send.isPending} busyLabel="Sending…" disabledReason={reason ?? (s.senders.length ? undefined : 'Add a sender first.')}>
            {s.messageId ? 'Send another test' : 'Send test'}
          </Button>
          <Button onClick={() => setSeedsOpen(true)} disabledReason={reason}>
            Send to a seed list (Gmail, Outlook, Yahoo, iCloud)
          </Button>
        </div>
        {auth ? (
          <div className="flex flex-col gap-1.5 rounded-lg border border-line bg-ground-3 px-3.5 py-3 text-[13px]">
            <span className="font-semibold">{`Authentication seen by ${auth.receiver}`}</span>
            <span className="mono k text-ink-nav">{auth.header}</span>
            <div className="flex flex-wrap gap-1.5">
              <AuthBadge name="DKIM" result={auth.dkim} />
              <AuthBadge name="SPF" result={auth.spf} />
              <AuthBadge name="DMARC" result={auth.dmarc} />
              <Badge tone={auth.clipped ? 'warn' : 'ok'}>{`${auth.clipped ? 'Clipped' : 'Not clipped'} · ${auth.sizeKb} KB`}</Badge>
            </div>
          </div>
        ) : null}
        {s.seeds.length ? <span className="k">{`Also sent to ${s.seeds.join(', ')}.`}</span> : null}
      </section>

      <section className="card flex flex-[1_1_420px] flex-col px-5 py-4.5" aria-labelledby="timeline-h">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="h2" id="timeline-h">
            Delivery timeline
          </h3>
          {s.listening ? (
            <Badge tone="ok" dot>
              Listening for events
            </Badge>
          ) : null}
        </div>
        {s.timeline.length ? (
          <ol className="m-0 list-none p-0" aria-live="polite">
            {s.timeline.map((ev, i) => (
              <li key={ev.key} className={i === 0 ? 'cn-ev mt-2 border-t-0' : 'cn-ev'}>
                <EventIcon status={ev.status} />
                <div className="flex flex-col gap-0.5">
                  <span className={ev.status === 'waiting' ? 'font-semibold text-ink-3' : 'font-semibold'}>{ev.title}</span>
                  <span className="k">
                    <Ticks text={ev.detail} />
                  </span>
                </div>
              </li>
            ))}
          </ol>
        ) : (
          <p className="k mt-2">Send a test: each step shows here as the events come back.</p>
        )}
        {s.roundTrip ? <div className="mt-2.5 rounded-lg border border-ok-line bg-ok-tint px-3 py-2.5 text-[13px] text-[#14532D]">{s.roundTrip}</div> : null}
      </section>
      <SeedDialog
        open={seedsOpen}
        onOpenChange={setSeedsOpen}
        initial={s.seedList}
        busy={send.isPending}
        onSend={(seeds) => {
          run({ senderId, subject: subject.trim(), seeds }, () => {
            setSeedsOpen(false)
            toast({ title: 'Test sent to the seed list', detail: 'Each mailbox provider shows how it filed the message.' })
          })
        }}
      />
    </div>
  )
}

function AuthBadge({ name, result }: { name: string; result: 'pass' | 'fail' | 'none' }) {
  const tone: Tone = result === 'pass' ? 'ok' : result === 'fail' ? 'bad' : 'neutral'
  return <Badge tone={tone}>{`${name} ${result}`}</Badge>
}

function EventIcon({ status }: { status: 'done' | 'waiting' | 'failed' }) {
  if (status === 'done') {
    return (
      <span className="ic bg-ok-bg">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#166534" strokeWidth="2" aria-hidden="true">
          <path d="m3 8 3.5 3.5L13 5" />
        </svg>
        <span className="vh">Done</span>
      </span>
    )
  }
  if (status === 'failed') {
    return (
      <span className="ic bg-bad-bg">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#991B1B" strokeWidth="2" aria-hidden="true">
          <path d="M4 4l8 8M12 4l-8 8" />
        </svg>
        <span className="vh">Failed</span>
      </span>
    )
  }
  return (
    <span className="ic bg-line-2">
      <span className="spin sm" aria-hidden="true" />
      <span className="vh">Waiting</span>
    </span>
  )
}

function SeedDialog({
  open,
  onOpenChange,
  initial,
  busy,
  onSend,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  initial: Record<string, string>
  busy: boolean
  onSend: (seeds: Record<string, string>) => void
}) {
  const [seeds, setSeeds] = useState<Record<string, string>>(initial)
  const any = Object.values(seeds).some((v) => v.trim())
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (o) setSeeds(initial)
        onOpenChange(o)
      }}
      size="sm"
      title="Send to a seed list"
      description="One address of yours at each mailbox provider. ReputeCast keeps the list for the next test."
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button
            variant="primary"
            loading={busy}
            busyLabel="Sending…"
            disabledReason={any ? undefined : 'Enter at least one address.'}
            onClick={() => onSend(Object.fromEntries(Object.entries(seeds).map(([k, v]) => [k, v.trim()])))}
          >
            Send
          </Button>
        </>
      }
    >
      {SEEDS.map((sd) => (
        <Field key={sd.key} label={sd.label} optional>
          <Input type="email" placeholder={sd.placeholder} value={seeds[sd.key] ?? ''} onChange={(e) => setSeeds((s) => ({ ...s, [sd.key]: e.target.value }))} />
        </Field>
      ))}
    </Dialog>
  )
}

