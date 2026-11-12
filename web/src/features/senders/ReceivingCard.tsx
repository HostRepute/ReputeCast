import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { failed } from '@/features/connect/parts'
import { copyText } from '@/features/sms/common'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, clock } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { InboundSetupDialog } from './InboundSetupDialog'
import { m } from './receiving.messages'

export type InboundRoute = Schemas['InboundRouteView']
export const inboundPath = '/api/v1/workspaces/{workspace}/inbound' as const

/** How long a test message may take before the card says it didn't arrive. */
export const TEST_WAIT_MS = 2 * 60 * 1000
/** How long a test's result stays on the card. */
const TEST_SHOWN_MS = 10 * 60 * 1000

/** The last test message, as the card shows it. */
export type TestState = 'none' | 'waiting' | 'received' | 'missing'

export function testState(r: InboundRoute | undefined, now: number): TestState {
  const sent = r?.lastTest.sentAt ? Date.parse(r.lastTest.sentAt) : 0
  if (!sent || now - sent > TEST_SHOWN_MS) return 'none'
  const got = r?.lastTest.receivedAt ? Date.parse(r.lastTest.receivedAt) : 0
  if (got >= sent) return 'received'
  return now - sent < TEST_WAIT_MS ? 'waiting' : 'missing'
}

/** Polls the route every 3 seconds while a test message is on its way. */
export function useInboundRoute() {
  const { slug } = useWorkspace()
  const [now, setNow] = useState(() => Date.now())
  const q = $api.useQuery('get', inboundPath, { params: { path: { workspace: slug } } }, {
    refetchInterval: (query) => (testState(query.state.data, Date.now()) === 'waiting' ? 3000 : false),
  })
  const waiting = testState(q.data, now) === 'waiting'
  useEffect(() => {
    if (!waiting) return
    const id = window.setInterval(() => setNow(Date.now()), 3000)
    return () => window.clearInterval(id)
  }, [waiting])
  return { q, now }
}

const STATUS_TONE: Record<InboundRoute['status'], Tone> = { not_set_up: 'neutral', pending: 'warn', receiving: 'ok', failing: 'bad' }

export function StatusBadge({ status }: { status: InboundRoute['status'] }) {
  const t = useT()
  const label = { not_set_up: m.statusNotSetUp, pending: m.statusPending, receiving: m.statusReceiving, failing: m.statusFailing }[status]
  return (
    <Badge tone={STATUS_TONE[status]} dot={status !== 'not_set_up'}>
      {t(label)}
    </Badge>
  )
}

export function MxBadge({ status }: { status: Schemas['InboundMXRecord']['status'] }) {
  const t = useT()
  switch (status) {
    case 'found':
      return <Badge tone="ok">{t(m.mxFound)}</Badge>
    case 'missing':
      return <Badge tone="bad">{t(m.mxMissing)}</Badge>
    case 'wrong_value':
      return <Badge tone="bad">{t(m.mxWrong)}</Badge>
    default:
      return <Badge tone="neutral">{t(m.mxUnknown)}</Badge>
  }
}

/** Copy, then "Copied" for two seconds: a plain button, as the boards draw it. */
export function CopyButton({ value, label }: { value: string; label: string }) {
  const t = useT()
  const [done, setDone] = useState(false)
  return (
    <Button
      size="sm"
      aria-label={label}
      onClick={() =>
        void copyText(value).then((ok) => {
          if (!ok) return
          setDone(true)
          window.setTimeout(() => setDone(false), 2000)
        })
      }
    >
      <span aria-live="polite">{done ? t(common.copied) : t(common.copy)}</span>
    </Button>
  )
}

/** The MX records a provider route needs, with what DNS says and Copy. */
export function MxTable({ records }: { records: Schemas['InboundMXRecord'][] }) {
  const t = useT()
  return (
    <Table caption={t(m.mxCaption)} fit>
      <thead>
        <tr>
          <Th>{t(m.colType)}</Th>
          <Th>{t(m.colName)}</Th>
          <Th>{t(m.colValue)}</Th>
          <Th>{t(m.colStatus)}</Th>
          <Th>
            <span className="vh">{t(m.copyColumn)}</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {records.map((r) => (
          <tr key={r.value}>
            <td className="mono whitespace-nowrap">{r.type}</td>
            <td className="mono whitespace-nowrap">{r.name}</td>
            <td className="mono whitespace-nowrap">{r.value}</td>
            <td>
              <MxBadge status={r.status} />
            </td>
            <td>
              <CopyButton value={r.value} label={t(m.copyMx)} />
            </td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

/** "Postmark · reply.news.acme.io" or "IMAP · replies@acme.io". */
function routeName(t: ReturnType<typeof useT>, r: InboundRoute) {
  if (r.kind === 'mailbox' && r.mailbox) return t(m.mailboxLine, { protocol: r.mailbox.protocol.toUpperCase(), address: r.mailbox.address })
  return t(m.routeLine, { name: r.provider?.name ?? '', domain: r.domain })
}

/** The first MX host the route needs ("inbound.postmarkapp.com"). */
function mxHost(r: InboundRoute) {
  const v = r.setup.mx[0]?.value ?? ''
  return v.split(/\s+/).pop()?.replace(/\.$/, '') ?? ''
}

/** The test message's state, under the route line. */
export function TestResult({ route, state }: { route: InboundRoute; state: TestState }) {
  const t = useT()
  const sentAt = route.lastTest.sentAt ? clock(route.lastTest.sentAt) : ''
  const via = route.kind === 'mailbox' ? (route.mailbox?.address ?? '') : (route.provider?.name ?? '')
  if (state === 'waiting') {
    return (
      <Note tone="neutral" role="status">
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
          <span className="font-bold">{t(m.testSent, { time: sentAt })}</span>
          <div>{t(m.testSentBody, { domain: route.domain })}</div>
        </div>
      </Note>
    )
  }
  if (state === 'received') {
    return (
      <Note tone="ok" role="status">
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
          <span className="font-bold">{t(m.testReceived, { time: route.lastTest.receivedAt ? clock(route.lastTest.receivedAt) : '' })}</span>
          <div>{t(m.testReceivedBody, { via })}</div>
        </div>
      </Note>
    )
  }
  if (state === 'missing') {
    const why =
      route.kind === 'mailbox'
        ? t(m.testMissingMailbox, { address: route.mailbox?.address ?? '' })
        : route.setup.automatic
          ? t(m.testMissingProvider, { host: mxHost(route), provider: via })
          : t(m.testMissingManual, { host: mxHost(route), provider: via })
    return (
      <Note tone="warn" role="status">
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
          <span className="font-bold">{t(m.testMissing)}</span>
          <div>{why}</div>
        </div>
      </Note>
    )
  }
  return null
}

/** Sends the test message; the card polls until it's back. */
export function useSendTest() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/inbound/test', {
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['get', inboundPath] }),
  })
  const run = () => send.mutate({ params: { path: { workspace: slug } } }, { onError: failed(t(m.testFailed), run) })
  return { run, pending: send.isPending }
}

/**
 * Settings › Senders › Receiving mail (M58, InboundStates): how the
 * workspace receives replies, unsubscribe emails and reports, the MX record
 * and the report addresses, the test message, Edit and Remove.
 */
export function ReceivingCard() {
  const t = useT()
  const { q, now } = useInboundRoute()
  const [setup, setSetup] = useState(false)
  const [removing, setRemoving] = useState(false)
  const test = useSendTest()
  const r = q.data
  const reason = usePermissionReason('domains:manage', r?.helpers)
  // Members who can't see providers don't see how the workspace receives.
  if (q.error instanceof ApiError && q.error.status === 403) return null
  if (q.isError) {
    return (
      <section className="card" aria-labelledby="rcv-h">
        <div className="sec flex-wrap gap-2">
          <h2 className="h2" id="rcv-h">
            {t(m.title)}
          </h2>
        </div>
        <div className="flex flex-col gap-3 px-4.5 py-3.5">
          <Note tone="bad">
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
              <span className="font-bold">{t(m.loadFailed)}</span>
              <div>{t(m.loadFailedBody)}</div>
            </div>
          </Note>
          <div>
            <Button size="sm" onClick={() => void q.refetch()}>
              {t(m.tryAgain)}
            </Button>
          </div>
        </div>
      </section>
    )
  }
  if (!r) return <ReceivingSkeleton />
  const dialog = setup ? <InboundSetupDialog open onOpenChange={setSetup} route={r} /> : null
  if (r.status === 'not_set_up') {
    return (
      <section className="card" aria-labelledby="rcv-h">
        <div className="sec flex-wrap gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h2 className="h2" id="rcv-h">
              {t(m.title)}
            </h2>
            <StatusBadge status={r.status} />
          </div>
          <Button variant="primary" onClick={() => setSetup(true)} disabledReason={reason}>
            {t(m.setUp)}
          </Button>
        </div>
        <div className="flex flex-col gap-1.5 px-4.5 py-3.5">
          <span>{t(m.notSetUpBody)}</span>
          <span className="k">{t(m.notSetUpUntil)}</span>
        </div>
        {dialog}
      </section>
    )
  }
  const state = testState(r, now)
  const mailbox = r.kind === 'mailbox' ? r.mailbox : undefined
  const line = mailbox
    ? r.lastReceivedAt
      ? t(m.mailboxLastMessage, { host: mailbox.host, port: mailbox.port || 993, security: mailbox.security.toUpperCase(), ago: ago(r.lastReceivedAt) })
      : t(m.mailboxNoMessage, { host: mailbox.host, port: mailbox.port || 993, security: mailbox.security.toUpperCase() })
    : r.lastReceivedAt
      ? t(m.lastMessage, { ago: ago(r.lastReceivedAt) })
      : t(m.setUpNoMessage, { time: r.setUpAt ? clock(r.setUpAt) : '' })
  const provider = r.provider?.name ?? ''
  let note: ReactNode = null
  if (r.status === 'failing' || r.statusReason === 'register') {
    const after =
      r.statusReason === 'mx'
        ? t(m.failingMx, { host: mxHost(r) })
        : r.statusReason === 'mailbox_auth' || r.statusReason === 'mailbox_unreachable'
          ? t(m.failingMailbox)
          : r.statusReason === 'register'
            ? t(m.failingRegister, { provider })
            : ''
    note = (
      <Note tone="bad">
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
          <span className="font-bold">{r.statusDetail}</span>
          {after ? <div>{after}</div> : null}
        </div>
      </Note>
    )
  } else if (r.status === 'pending' && state === 'none') {
    const body = mailbox
      ? mailbox.lastPolledAt
        ? t(m.pendingMailbox)
        : t(m.pendingMailboxFirst)
      : r.setup.automatic
        ? t(m.pendingAutomatic, { provider, domain: r.domain })
        : t(m.pendingManual, { provider })
    note = <Note tone="info" role="note">{body}</Note>
  } else if (r.statusReason === 'dropped_reports' && r.statusDetail) {
    note = (
      <Note tone="neutral" role="note">
        {r.statusDetail}
      </Note>
    )
  }
  return (
    <section className="card" aria-labelledby="rcv-h">
      <div className="sec flex-wrap gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="h2" id="rcv-h">
              {t(m.title)}
            </h2>
            <StatusBadge status={r.status} />
          </div>
          <span className="k">{t(m.sub)}</span>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={test.run} loading={test.pending} disabledReason={reason}>
            {t(m.sendTest)}
          </Button>
          <Button size="sm" onClick={() => setSetup(true)} disabledReason={reason}>
            {t(m.edit)}
          </Button>
          <Button size="sm" variant="danger" onClick={() => setRemoving(true)} disabledReason={reason}>
            {t(m.remove)}
          </Button>
        </div>
      </div>
      <div className="flex flex-wrap items-baseline gap-x-3.5 gap-y-1 px-4.5 pt-3">
        <span className="font-semibold">{routeName(t, r)}</span>
        <span className="k">{line}</span>
      </div>
      {reason ? (
        <div className="px-4.5 pt-1.5">
          <span className="k">{reason}</span>
        </div>
      ) : null}
      {note || state !== 'none' ? (
        <div className="flex flex-col gap-2.5 px-4.5 pt-3">
          {note}
          <TestResult route={r} state={state} />
        </div>
      ) : null}
      {mailbox ? (
        <div className="px-4.5 py-3">
          <dl className="kv">
            <dt>{t(m.server)}</dt>
            <dd className="mono">{t(m.serverValue, { host: mailbox.host, port: mailbox.port || 993, security: mailbox.security.toUpperCase() })}</dd>
            <dt>{t(m.userName)}</dt>
            <dd className="mono">{mailbox.username}</dd>
          </dl>
        </div>
      ) : r.setup.mx.length ? (
        <div className="px-1.5 pt-1">
          <MxTable records={r.setup.mx} />
        </div>
      ) : null}
      <ReportAddresses route={r} />
      {dialog}
      <RemoveDialog open={removing} onOpenChange={setRemoving} route={r} />
    </section>
  )
}

/** The fixed addresses on the route's domain, with Copy. */
function ReportAddresses({ route }: { route: InboundRoute }) {
  const t = useT()
  const rows = [
    { label: t(m.dmarcReports), value: route.addresses.dmarc, copy: t(m.copyDmarc) },
    { label: t(m.fblReports), value: route.addresses.fbl, copy: t(m.copyFbl) },
  ]
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-x-6 gap-y-3 border-t border-line-2 px-4.5 py-3.5 max-md:grid-cols-1">
      {rows.map((x) => (
        <div key={x.label} className="flex min-w-0 items-center justify-between gap-x-3 gap-y-2">
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="k">{x.label}</span>
            <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{x.value}</span>
          </div>
          <CopyButton value={x.value} label={x.copy} />
        </div>
      ))}
    </div>
  )
}

function ReceivingSkeleton() {
  const t = useT()
  return (
    <section className="card" aria-busy="true" aria-labelledby="rcv-ld">
      <span className="vh" id="rcv-ld">
        {t(m.loading)}
      </span>
      <div className="sec flex-wrap gap-2">
        <div className="flex min-w-0 flex-col gap-2">
          <Skeleton w={140} h={16} />
          <Skeleton w={320} h={12} className="max-md:w-full" />
        </div>
        <div className="flex gap-2">
          <Skeleton w={132} h={30} />
          <Skeleton w={56} h={30} />
          <Skeleton w={72} h={30} />
        </div>
      </div>
      <div className="flex flex-col gap-2.5 px-4.5 py-3.5">
        <Skeleton w={220} h={14} />
        <Skeleton w="100%" h={34} />
        <Skeleton w="100%" h={34} />
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-x-6 gap-y-3 border-t border-line-2 px-4.5 py-3.5 max-md:grid-cols-1">
        <Skeleton w="100%" h={34} />
        <Skeleton w="100%" h={34} />
      </div>
    </section>
  )
}

/** Stop receiving mail? — the consequences first (InboundStates › Remove). */
function RemoveDialog({ open, onOpenChange, route }: { open: boolean; onOpenChange: (o: boolean) => void; route: InboundRoute }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const del = $api.useMutation('delete', inboundPath)
  const run = () =>
    del.mutate(
      { params: { path: { workspace: slug } } },
      {
        onSuccess: () => {
          onOpenChange(false)
          void qc.invalidateQueries({ queryKey: ['get', inboundPath] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/senders'] })
          toast({ title: t(m.removed), detail: t(m.removedForward) })
        },
        onError: failed(t(m.removeFailed), run),
      },
    )
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.removeEyebrow)}</span>}
      title={t(m.removeTitle)}
      dismissible={false}
      hideClose
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="danger-primary" loading={del.isPending} onClick={run}>
            {t(m.removeConfirm)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1.5 pl-4.5 text-[13.5px] text-ink-2">
        <li>{t(m.removeStays)}</li>
        <li>{t(m.removeForward)}</li>
        <li>{t(m.removeMailto)}</li>
      </ul>
      <span className="k">
        {route.kind === 'mailbox'
          ? t(m.removeMailbox, { address: route.mailbox?.address ?? '' })
          : t(m.removeProvider, { provider: route.provider?.name ?? '', domain: route.domain })}
      </span>
    </Dialog>
  )
}
