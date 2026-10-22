import { useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mn } from './connection.messages'
import { m } from './messages'
import { failed, type StepProps } from './parts'
import { mst } from './steps.messages'
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
  const t = useT()
  const { slug } = useWorkspace()
  const init = { params: { path: { workspace: slug, setupId: setup.id } } }
  const q = $api.useQuery('get', testPath, init, { refetchInterval: (query) => (query.state.data?.listening ? 2000 : false) })
  const info = setup.info
  return (
    <>
      <StepHeading title={t(mst.testHeading)}>
        {info.flow === 'smtp' ? t(mst.testLeadSmtp) : info.webhooks === 'SNS' ? t(mst.testLeadSns, { provider: info.short }) : t(mst.testLead, { provider: info.short })}
      </StepHeading>
      {q.data ? (
        <TestPanels setup={setup} status={q.data} reason={reason} />
      ) : q.isError ? (
        <ErrorPanel title={t(mst.testLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <div className="flex flex-wrap items-start gap-5" aria-busy="true">
          <span className="vh" role="status">
            {t(mst.testLoading)}
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
      <WizardFooter back={<Button onClick={() => go('domain')}>{t(common.back)}</Button>}>
        <Button variant="primary" className={continueClass} onClick={() => go('streams')}>
          {t(mst.continueStreams)}
        </Button>
      </WizardFooter>
    </>
  )
}

function TestPanels({ setup, status: s, reason }: { setup: Schemas['ConnectSetup']; status: Status; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const init = { params: { path: { workspace: slug, setupId: setup.id } } }
  const key = $api.queryOptions('get', testPath, init).queryKey
  const [senderId, setSenderId] = useState(s.senderId)
  const [to, setTo] = useState(s.to)
  const [subject, setSubject] = useState(s.subject || t(mst.defaultSubject, { provider: setup.info.name }))
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
        onError: failed(t(mst.testFailed), () => run(body, done)),
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
          {t(mst.testMessage)}
        </h3>
        <form id="test-form" className="contents" onSubmit={submit}>
          <label className="lbl">
            {t(mst.from)}
            <select className="in" value={senderId} onChange={(e) => setSenderId(e.target.value)} disabled={Boolean(reason)}>
              {s.senders.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            {t(mst.to)}
            <input className="in" type="email" value={to} onChange={(e) => setTo(e.target.value)} disabled={Boolean(reason)} autoComplete="email" />
          </label>
          <label className="lbl">
            {t(mst.subject)}
            <input className="in" value={subject} onChange={(e) => setSubject(e.target.value)} disabled={Boolean(reason)} />
          </label>
        </form>
        <div className="flex flex-wrap gap-2.5">
          <Button type="submit" form="test-form" variant="primary" loading={send.isPending} busyLabel={t(mst.sending)} disabledReason={reason ?? (s.senders.length ? undefined : t(mst.addSenderFirst))}>
            {s.messageId ? t(mst.sendAnother) : t(mst.sendTest)}
          </Button>
          <Button onClick={() => setSeedsOpen(true)} disabledReason={reason}>
            {t(mst.sendSeeds)}
          </Button>
        </div>
        {auth ? (
          <div className="flex flex-col gap-1.5 rounded-lg border border-line bg-ground-3 px-3.5 py-3 text-[13px]">
            <span className="font-semibold">{t(mst.authSeenBy, { receiver: auth.receiver })}</span>
            <span className="mono k text-ink-nav">{auth.header}</span>
            <div className="flex flex-wrap gap-1.5">
              <AuthBadge name="DKIM" result={auth.dkim} />
              <AuthBadge name="SPF" result={auth.spf} />
              <AuthBadge name="DMARC" result={auth.dmarc} />
              <Badge tone={auth.clipped ? 'warn' : 'ok'}>{t(mst.clipped, { clipped: auth.clipped ? 'yes' : 'no', size: count(auth.sizeKb) })}</Badge>
            </div>
          </div>
        ) : null}
        {s.seeds.length ? <span className="k">{t(mst.alsoSent, { addresses: s.seeds.join(', ') })}</span> : null}
      </section>

      <section className="card flex flex-[1_1_420px] flex-col px-5 py-4.5" aria-labelledby="timeline-h">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="h2" id="timeline-h">
            {t(mst.timelineTitle)}
          </h3>
          {s.listening ? (
            <Badge tone="ok" dot>
              {t(mst.listening)}
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
          <p className="k mt-2">{t(mst.timelineEmpty)}</p>
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
            toast({ title: t(mst.seedsSent), detail: t(mst.seedsSentDetail) })
          })
        }}
      />
    </div>
  )
}

/** "DKIM pass": the result as the Authentication-Results header writes it. */
function AuthBadge({ name, result }: { name: string; result: 'pass' | 'fail' | 'none' }) {
  const tone: Tone = result === 'pass' ? 'ok' : result === 'fail' ? 'bad' : 'neutral'
  return <Badge tone={tone}>{`${name} ${result}`}</Badge>
}

function EventIcon({ status }: { status: 'done' | 'waiting' | 'failed' }) {
  const t = useT()
  if (status === 'done') {
    return (
      <span className="ic bg-ok-bg">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#166534" strokeWidth="2" aria-hidden="true">
          <path d="m3 8 3.5 3.5L13 5" />
        </svg>
        <span className="vh">{t(m.stateDone)}</span>
      </span>
    )
  }
  if (status === 'failed') {
    return (
      <span className="ic bg-bad-bg">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#991B1B" strokeWidth="2" aria-hidden="true">
          <path d="M4 4l8 8M12 4l-8 8" />
        </svg>
        <span className="vh">{t(m.stateFailed)}</span>
      </span>
    )
  }
  return (
    <span className="ic bg-line-2">
      <span className="spin sm" aria-hidden="true" />
      <span className="vh">{t(mn.waiting)}</span>
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
  const t = useT()
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
      title={t(mst.seedsTitle)}
      description={t(mst.seedsLine)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button
            variant="primary"
            loading={busy}
            busyLabel={t(mst.sending)}
            disabledReason={any ? undefined : t(mst.enterOne)}
            onClick={() => onSend(Object.fromEntries(Object.entries(seeds).map(([k, v]) => [k, v.trim()])))}
          >
            {t(mst.send)}
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
