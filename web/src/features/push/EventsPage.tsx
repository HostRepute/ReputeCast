import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Table, Td, Th } from '@/components/ui/Table'
import { useT, type MessageDescriptor } from '@/i18n'
import { clockSeconds, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ChannelSwitch } from '@/features/channels/ChannelSwitch'
import { browserName, PushTabs, throwIfForbidden } from './common'
import { m } from './messages'

type PushEvent = Schemas['PushEvent']

const PAGE = 50

const kindBadge: Partial<Record<PushEvent['kind'], { tone: Tone; msg: MessageDescriptor }>> = {
  sent: { tone: 'ok', msg: m.kindSent },
  shown: { tone: 'ok', msg: m.kindShown },
  clicked: { tone: 'info', msg: m.kindClicked },
  closed: { tone: 'neutral', msg: m.kindClosed },
  gone: { tone: 'warn', msg: m.kindGone },
  failed: { tone: 'bad', msg: m.kindFailed },
  queued: { tone: 'neutral', msg: m.kindQueued },
}

/**
 * Sending providers › Push › Events: each notification as the push service
 * answered it (raw), next to what ReputeCast made of it.
 */
export function PushEventsPage() {
  const t = useT()
  return (
    <Page
      title={t(m.screenTitle)}
      documentTitle={t(m.docTitleEvents)}
      filters={
        <>
          <ChannelSwitch current="push" />
          <PushTabs />
        </>
      }
    >
      <EventLog />
    </Page>
  )
}

function EventLog() {
  const t = useT()
  const { slug } = useWorkspace()
  // Each "Load more" adds a page that starts before the last event shown.
  const [cursors, setCursors] = useState<(string | undefined)[]>([undefined])
  const first = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/events', { params: { path: { workspace: slug }, query: { limit: PAGE } } })
  throwIfForbidden(first.error)
  if (first.isError) {
    return (
      <ErrorPanel title={t(m.eventsLoadFailed)} actions={<Button onClick={() => first.refetch()}>{t(m.retry)}</Button>}>
        {errorMessage(first.error)}
      </ErrorPanel>
    )
  }
  return (
    <section className="card flex flex-col" aria-labelledby="push-events-title">
      <div className="px-head">
        <div>
          <h2 className="h2" id="push-events-title">
            {t(m.eventsTitle)}
          </h2>
          <span className="k">{t(m.eventsIntro)}</span>
        </div>
        {first.data?.removed ? (
          <div>
            <span className="k">{t(m.eventsRemoved, { count: first.data.removed })}</span>
          </div>
        ) : null}
      </div>
      <Table aria-label={t(m.eventsTitle)} className="px-events">
        <thead>
          <tr>
            <Th>{t(m.colWhen)}</Th>
            <Th>{t(m.colEvent)}</Th>
            <Th>{t(m.colMessage)}</Th>
            <Th>{t(m.colBrowser)}</Th>
            <Th>{t(m.colRaw)}</Th>
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
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/events', { params: { path: { workspace: slug }, query: { limit: PAGE, before } } })
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
        {[0, 1, 2, 3].map((i) => (
          <tr key={i}>
            {[90, 70, 220, 120, 160].map((w, j) => (
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
  const last = items[items.length - 1]
  return (
    <>
      {items.map((e, i) => (
        <EventRow key={`${e.at}-${i}`} e={e} />
      ))}
      {q.data.more && onMore && last ? (
        <tr>
          <td colSpan={5} className="text-center">
            <Button size="sm" onClick={() => onMore(last.at)}>
              {t(m.loadMore)}
            </Button>
          </td>
        </tr>
      ) : null}
    </>
  )
}

function EventRow({ e }: { e: PushEvent }) {
  const t = useT()
  const { slug } = useWorkspace()
  const b = kindBadge[e.kind]
  const source = e.sourceKind === 'automation' ? m.sourceAutomation : e.sourceKind === 'test' ? m.sourceTest : m.sourceCampaign
  return (
    <tr>
      <Td nowrap>
        <span className="flex flex-col">
          <span className="num">{clockSeconds(e.at)}</span>
          <span className="k">{dayMonthDate(e.at)}</span>
        </span>
      </Td>
      <Td>{b ? <Badge tone={b.tone}>{t(b.msg)}</Badge> : <Badge tone="neutral">{e.kind}</Badge>}</Td>
      <Td>
        <span className="flex min-w-0 flex-col">
          <span className="font-medium">{e.message}</span>
          <span className="k">
            {t(source)}
            {e.source ? ' · ' : null}
            {e.sourceKind === 'campaign' && e.sourceId ? (
              <Link to="/w/$workspace/campaigns/$campaignId/push-report" params={{ workspace: slug, campaignId: e.sourceId }} className="px-lnk text-[12px]">
                {e.source}
              </Link>
            ) : (
              e.source
            )}
          </span>
        </span>
      </Td>
      <Td>
        <span className="flex flex-col">
          <span>{t(browserName(e.browser, true))}</span>
          <span className="mono k">{e.site}</span>
        </span>
      </Td>
      <Td>
        <span className="px-raw">{e.raw}</span>
      </Td>
    </tr>
  )
}

