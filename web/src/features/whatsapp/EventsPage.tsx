import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Table, Td, Th } from '@/components/ui/Table'
import { useT, type MessageDescriptor } from '@/i18n'
import { clock, clockSeconds, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useWhatsApp, WhatsAppFrame } from './AccountPage'
import { throwIfForbidden } from './common'
import { m } from './messages'

type Event = Schemas['WhatsAppEvent']

const PAGE = 50

const kindWord: Record<Event['kind'], { tone: Tone; msg: MessageDescriptor }> = {
  status: { tone: 'info', msg: m.kindStatus },
  message: { tone: 'ok', msg: m.kindMessage },
  template: { tone: 'purple', msg: m.kindTemplate },
  quality: { tone: 'warn', msg: m.kindQuality },
  account: { tone: 'neutral', msg: m.kindAccount },
  unknown: { tone: 'neutral', msg: m.kindUnknown },
}

/** Sending providers › WhatsApp › Events: each webhook event, raw next to what it became. */
export function WhatsAppEventsPage() {
  const t = useT()
  const overview = useWhatsApp()
  return (
    <WhatsAppFrame doc={t(m.docTitleEvents)} data={overview.data}>
      <EventLog />
    </WhatsAppFrame>
  )
}

function EventLog() {
  const t = useT()
  const { slug } = useWorkspace()
  const [cursors, setCursors] = useState<(string | undefined)[]>([undefined])
  const first = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp/events', { params: { path: { workspace: slug }, query: { limit: PAGE } } })
  throwIfForbidden(first.error)
  if (first.isError) {
    return (
      <ErrorPanel title={t(m.eventsLoadFailed)} actions={<Button onClick={() => void first.refetch()}>{t(m.retry)}</Button>}>
        {errorMessage(first.error)}
      </ErrorPanel>
    )
  }
  return (
    <section className="card flex flex-col" aria-labelledby="wa-events-title">
      <div className="px-head">
        <div>
          <h2 className="h2" id="wa-events-title">
            {t(m.eventsTitle)}
          </h2>
          <span className="k">{t(m.eventsIntro)}</span>
        </div>
      </div>
      <Table aria-label={t(m.eventsTitle)} className="px-events">
        <thead>
          <tr>
            <Th>{t(m.colReceived)}</Th>
            <Th>{t(m.colKind)}</Th>
            <Th>{t(m.colSummary)}</Th>
            <Th>{t(m.colNumber)}</Th>
            <Th>{t(m.colSignature)}</Th>
          </tr>
        </thead>
        <tbody>
          {cursors.map((c, i) => (
            <Chunk key={c ?? 'first'} before={c} onMore={i === cursors.length - 1 ? (next) => setCursors((cs) => [...cs, next]) : undefined} />
          ))}
        </tbody>
      </Table>
    </section>
  )
}

function Chunk({ before, onMore }: { before?: string; onMore?: (before: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp/events', { params: { path: { workspace: slug }, query: { limit: PAGE, before } } })
  if (!q.data) {
    if (q.isError) {
      return (
        <tr>
          <td colSpan={5}>
            <span className="err" role="alert">
              {errorMessage(q.error)}
            </span>
          </td>
        </tr>
      )
    }
    return (
      <>
        {[0, 1, 2].map((i) => (
          <tr key={i}>
            {[90, 70, 260, 110, 60].map((w, j) => (
              <td key={j}>
                <Skeleton w={w} h={12} />
              </td>
            ))}
          </tr>
        ))}
      </>
    )
  }
  const items = q.data.items
  if (!items.length && !before) {
    return (
      <tr>
        <td colSpan={5} className="py-8 text-center text-ink-3">
          {t(m.noEvents)}
        </td>
      </tr>
    )
  }
  return (
    <>
      {items.map((e) => {
        const k = kindWord[e.kind]
        return (
          <tr key={e.id}>
            <Td nowrap>
              <span className="flex flex-col">
                <span className="num">{clockSeconds(e.receivedAt)}</span>
                <span className="k">{dayMonthDate(e.receivedAt)}</span>
              </span>
            </Td>
            <Td>
              <Badge tone={k.tone}>{t(k.msg)}</Badge>
            </Td>
            <Td wrap>
              <span className="flex min-w-0 flex-col">
                <span>{e.summary}</span>
                <span className="k">
                  {e.normalizedAt ? t(m.normalized, { time: clock(e.normalizedAt) }) : t(m.notNormalized)}
                  {e.error ? ` · ${e.error}` : null}
                </span>
              </span>
            </Td>
            <Td>
              <span className="mono k">{e.number ?? '–'}</span>
            </Td>
            <Td>{e.signatureOk ? <Badge tone="ok">{t(m.sigOk)}</Badge> : <Badge tone="bad">{t(m.sigBad)}</Badge>}</Td>
          </tr>
        )
      })}
      {q.data.next && onMore ? (
        <tr>
          <td colSpan={5} className="text-center">
            <Button size="sm" onClick={() => onMore(q.data.next!)}>
              {t(m.loadMore)}
            </Button>
          </td>
        </tr>
      ) : null}
    </>
  )
}
