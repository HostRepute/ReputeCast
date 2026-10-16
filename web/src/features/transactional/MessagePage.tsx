import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, dayMonthYear } from '@/lib/format'
import { useMediaQuery } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { ml } from './log.messages'
import { LogPhoneList, RejectionExplain, useLanguageName, useLog, useRangeLabel, type LogSearch } from './MessagesPage'
import { mt } from './messages'
import { CategoryCell, secs, StatusBadge, txTabs } from './shared'

type Detail = Schemas['TransactionalMessageDetail']
type Entry = Schemas['TransactionalTimelineEntry']

const API = '/api/v1/workspaces/{workspace}/transactional' as const

/** "15 Oct 2026, 10:42:18" */
function stamp(d: string) {
  const t = new Date(d)
  return `${dayMonthYear(t)}, ${clock(t)}:${String(t.getSeconds()).padStart(2, '0')}`
}

/** "10:42:18.120" */
export function stampMs(d: string) {
  const t = new Date(d)
  return `${clock(t)}:${String(t.getSeconds()).padStart(2, '0')}.${String(t.getMilliseconds()).padStart(3, '0')}`
}

function kb(n: number) {
  return n >= 1024 ? `${Math.round(n / 1024)} KB` : `${n} B`
}

/** Transactional › Messages › one message (TransactionalMessage, TransactionalMessageStates). */
export function TransactionalMessagePage({ messageId, search }: { messageId: string; search: LogSearch }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canView = me.permissions.includes('transactional:view')
  const q = $api.useQuery('get', `${API}/messages/{messageId}`, { params: { path: { workspace: slug, messageId } } }, { enabled: canView, retry: (n, e) => !(e instanceof ApiError && e.status === 404) && n < 2 })
  const phone = useMediaQuery('(max-width: 767px)')
  return (
    <Page
      title={t(mt.title)}
      tabs={txTabs(slug, t)}
      tabsLabel={t(mt.tabsLabel)}
      documentTitle={q.data ? t(ml.titleTemplate, { template: q.data.raw ? t(mt.raw) : (q.data.template?.name ?? ''), email: q.data.email }) : t(mt.title)}
      actions={
        <Link to="/w/$workspace/transactional/messages" params={{ workspace: slug }} search={search as never} className="tx-lnk">
          <Icon name="chevronLeft" size={14} />
          {t(ml.backToLog)}
        </Link>
      }
    >
      {!canView ? (
        <ForbiddenState permission="transactional:view" title={t(mt.forbiddenTitle)} />
      ) : (
        <div className="flex flex-wrap items-start gap-4">
          {phone ? null : <SideList current={messageId} search={search} />}
          <section className="card min-w-0" aria-label={t(ml.messageLabel)} style={{ flex: '999 1 560px' }}>
            {q.error instanceof ApiError && q.error.status === 404 ? (
              <div className="flex flex-col items-center gap-3 px-6 py-10 text-center" role="status">
                <h2 className="m-0 text-[17px] font-semibold">{t(ml.notFoundTitle)}</h2>
                <div className="max-w-[560px] text-ink-2">{t(ml.notFoundBody, { days: 90 })}</div>
                <ButtonLink to="/w/$workspace/transactional/messages" params={{ workspace: slug }} search={search as never}>
                  {t(ml.backToLog)}
                </ButtonLink>
              </div>
            ) : q.error ? (
              <div className="pad">
                <ErrorPanel title={t(ml.errorTitle)} actions={<Button onClick={() => void q.refetch()}>{t(mt.retry)}</Button>}>
                  {errorMessage(q.error)}
                </ErrorPanel>
              </div>
            ) : !q.data ? (
              <div className="pad flex flex-col gap-3" aria-busy="true">
                <Skeleton w="40%" />
                <Skeleton w="70%" h={22} />
                <Skeleton w="100%" h={160} />
              </div>
            ) : (
              <MessageDetail d={q.data} />
            )}
          </section>
        </div>
      )}
    </Page>
  )
}

function SideList({ current, search }: { current: string; search: LogSearch }) {
  const t = useT()
  const q = useLog(search, true)
  const items = useMemo(() => q.data?.pages.flatMap((p) => p.items) ?? [], [q.data])
  const first = q.data?.pages[0]
  const rangeLabel = useRangeLabel()
  return (
    <section className="card self-start overflow-hidden" aria-label={t(ml.logLabel)} style={{ flex: '1 1 360px', maxWidth: 400 }}>
      <div className="flex flex-col gap-2.5 px-4 py-3">
        <div className="row" style={{ justifyContent: 'space-between' }}>
          <span className="k">{t(ml.listSummary, { range: rangeLabel(search.range), count: first?.counts.all ?? 0 })}</span>
          <span className="tx-live">
            <span className="dot" />
            {t(ml.live)}
          </span>
        </div>
      </div>
      {!first ? (
        <div className="flex flex-col gap-2 px-4 pb-4">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} w="100%" h={16} />
          ))}
        </div>
      ) : (
        <>
          <LogPhoneList items={items} current={current} compact />
          {q.hasNextPage ? (
            <div className="tx-foot" style={{ justifyContent: 'center' }}>
              <Button size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
                {t(ml.loadMore)}
              </Button>
            </div>
          ) : null}
        </>
      )}
    </section>
  )
}

function resendReason(t: ReturnType<typeof useT>, d: Detail): string | undefined {
  if (d.resend.available) return undefined
  switch (d.resend.reason) {
    case 'sensitive': {
      const names = d.resend.sensitive ?? []
      return t(ml.resendSensitive, { names: names.join(', '), count: names.length })
    }
    case 'attachments':
      return t(ml.resendAttachments)
    case 'raw':
      return t(ml.resendRaw)
    case 'template':
      return t(ml.resendTemplate)
    case 'permission':
      return t(ml.resendPermission, { permission: 'transactional:send' })
    case 'not_sent':
      return t(ml.resendNotSent)
  }
  return undefined
}

function curlOf(d: Detail, origin: string) {
  const body = JSON.stringify(d.request.body ?? {}, null, 2)
    .split('\n')
    .map((l, i) => (i === 0 ? l : `  ${l}`))
    .join('\n')
  return [
    `curl -X POST "${origin}${d.request.path}" \\`,
    `  -H "Authorization: Bearer $REPUTECAST_KEY" \\`,
    ...(d.request.idempotencyKey ? [`  -H "Idempotency-Key: ${d.request.idempotencyKey}-again" \\`] : []),
    `  -H "Content-Type: application/json" \\`,
    `  -d '${body.replace(/'/g, "'\\''")}'`,
  ].join('\n')
}

function MessageDetail({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const resend = $api.useMutation('post', `${API}/messages/{messageId}/resend`)
  const why = resendReason(t, d)
  const title = d.raw ? t(ml.titleRaw, { subject: d.subject, email: d.email }) : t(ml.titleTemplate, { template: d.template?.name ?? '', email: d.email })
  const copy = (text: string, done: string) =>
    void navigator.clipboard?.writeText(text).then(
      () => toast({ kind: 'success', title: done }),
      () => toast({ kind: 'error', title: text }),
    )
  return (
    <>
      <div className="flex flex-wrap items-start gap-3 border-b border-line-2" style={{ padding: '18px 22px 16px' }}>
        <div className="flex min-w-0 flex-[1_1_360px] flex-col gap-1.5">
          <span className="k">
            <T msg={ml.message} values={{ id: <span className="mono">{d.id}</span> }} />
          </span>
          <h2 className="m-0 text-[19px] font-bold" style={{ letterSpacing: '-.01em' }}>
            {title}
          </h2>
          <div className="flex flex-wrap items-center gap-2.5">
            <span className="inline-flex flex-wrap items-center gap-2">
              <StatusBadge status={d.status} />
              {d.deliveryMs !== undefined && (d.status === 'delivered' || d.status === 'complained') ? <span className="k">{t(ml.inTime, { time: secs(d.deliveryMs) })}</span> : null}
            </span>
            <span className="k">{stamp(d.at)}</span>
          </div>
        </div>
        <div className="flex flex-none flex-wrap gap-2">
          <Button
            size="sm"
            disabledReason={why}
            loading={resend.isPending}
            onClick={() =>
              resend.mutate(
                { params: { path: { workspace: slug, messageId: d.id } } },
                {
                  onSuccess: (r) => {
                    toast({ kind: 'success', title: t(ml.resent) })
                    void qc.invalidateQueries()
                    if (r?.id) void navigate({ to: '/w/$workspace/transactional/messages/$messageId', params: { workspace: slug, messageId: r.id } })
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(ml.resend)}
          </Button>
          {!d.raw ? (
            <Button size="sm" icon="code" onClick={() => copy(curlOf(d, window.location.origin), t(ml.curlCopied))}>
              {t(ml.copyCurl)}
            </Button>
          ) : null}
          <Button size="sm" icon="link" aria-label={t(ml.copyLinkLabel)} onClick={() => copy(window.location.href, t(ml.linkCopied))}>
            {t(ml.copyLink)}
          </Button>
        </div>
      </div>
      {why && d.resend.reason === 'sensitive' ? (
        <Note tone="neutral" style={{ margin: '14px 22px 0' }}>
          <T
            msg={ml.resendSensitive}
            values={{
              names: (
                <>
                  {(d.resend.sensitive ?? []).map((n, i) => (
                    <span key={n}>
                      {i ? ', ' : ''}
                      <span className="mono">{n}</span>
                    </span>
                  ))}
                </>
              ),
              count: d.resend.sensitive?.length ?? 0,
            }}
          />
        </Note>
      ) : null}
      {d.rejection || d.status === 'failed' ? (
        <div style={{ margin: '14px 22px 0' }}>
          <RejectionExplain it={d} />
        </div>
      ) : null}
      <div className="grid items-start gap-[22px]" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(360px, 100%), 1fr))', padding: '18px 22px 22px' }}>
        <div className="flex min-w-0 flex-col gap-[18px]">
          <Summary d={d} />
          <Timeline d={d} />
          <Request d={d} />
        </div>
        <ContentPanel d={d} />
      </div>
    </>
  )
}

function Row({ k, children }: { k: string; children: ReactNode }) {
  return (
    <>
      <dt>{k}</dt>
      <dd>{children}</dd>
    </>
  )
}

function languageLine(t: ReturnType<typeof useT>, name: (c?: string) => string, d: Detail): string | null {
  const l = d.languageInfo
  if (!l || (!l.sent && !l.reason)) return null
  const sent = name(l.sent)
  const contact = l.contact ? name(l.contact) : ''
  switch (l.reason) {
    case 'match':
      return t(ml.languageContact, { language: sent })
    case 'requested':
      return l.contact && l.contact !== l.sent ? t(ml.languageRequestedFallback, { requested: l.contact, language: sent }) : t(ml.languageRequested, { language: sent })
    case 'no_language':
      return t(ml.languageNoLanguage, { language: sent })
    case 'not_offered':
      return t(ml.languageNotOffered, { language: sent, contact })
    case 'missing':
      return t(ml.languageMissing, { language: sent, contact })
    case 'draft':
      return t(ml.languageDraft, { language: sent, contact })
    case 'held':
      return t(ml.languageHeld, { language: sent, contact })
  }
  return sent
}

function Summary({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useLanguageName()
  const lang = languageLine(t, name, d)
  const meta = Object.entries(d.metadata ?? {})
  return (
    <div className="flex flex-col gap-2.5">
      <h3 className="h2">{t(ml.summary)}</h3>
      <dl className="kv">
        <Row k={t(ml.recipient)}>
          {d.contact ? (
            <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: d.contact.id }}>
              {d.contact.name}
            </Link>
          ) : d.name ? (
            <span>{d.name}</span>
          ) : null}{' '}
          <span className="mono k">{d.email}</span>
          {d.roleAddress ? (
            <>
              {' '}
              <Badge tone="neutral" className="tx-kind">
                {t(ml.roleAddress)}
              </Badge>
            </>
          ) : null}
        </Row>
        <Row k={t(ml.template)}>
          {d.raw || !d.template ? (
            t(mt.raw)
          ) : (
            <>
              <Link to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: d.template.key }}>
                {d.template.name}
              </Link>{' '}
              <span className="mono k">{d.template.key}</span>
              {d.template.version ? ` · v${d.template.version}` : ''}
            </>
          )}
        </Row>
        {d.category ? (
          <Row k={t(ml.category)}>
            <CategoryCell category={{ id: '', name: d.category, kind: d.categoryKind === 'optional' ? 'optional' : 'essential' }} />
          </Row>
        ) : null}
        <Row k={t(ml.provider)}>{d.provider ? t(ml.providerLine, { provider: d.provider.name }) : <span className="k">{t(ml.notSent)}</span>}</Row>
        {lang ? <Row k={t(ml.language)}>{lang}</Row> : null}
        {d.keyRef ? (
          <Row k={t(ml.key)}>
            {d.keyRef.name}
            {d.ip ? <span className="k"> · {d.ip}</span> : null}
          </Row>
        ) : d.sentBy ? (
          <Row k={t(ml.sentBy)}>{d.sentBy}</Row>
        ) : null}
        {d.idempotencyKey ? (
          <Row k={t(ml.idempotencyKey)}>
            <span className="mono">{d.idempotencyKey}</span>
          </Row>
        ) : null}
        {d.tags.length ? (
          <Row k={t(ml.tags)}>
            <span className="inline-flex flex-wrap gap-1.5">
              {d.tags.map((tag) => (
                <span key={tag} className="chip">
                  {tag}
                </span>
              ))}
            </span>
          </Row>
        ) : null}
        {meta.length ? (
          <Row k={t(ml.metadata)}>
            <span className="mono">{meta.map(([k, v]) => `${k}: ${v}`).join(' · ')}</span>
          </Row>
        ) : null}
        {d.cc.length ? <Row k={t(ml.cc)}>{d.cc.join(', ')}</Row> : null}
        {d.bcc.length ? <Row k={t(ml.bcc)}>{d.bcc.join(', ')}</Row> : null}
        {d.dropped.length ? <Row k={t(ml.dropped)}>{d.dropped.join(', ')}</Row> : null}
        {d.attachments.length ? (
          <Row k={t(ml.attachments)}>
            {d.attachments.map((a) => (
              <span key={a.filename} className="mr-2 inline-flex items-center gap-1">
                <Icon name="link" size={12} />
                {a.filename} <span className="k">· {kb(a.size)}</span>
              </span>
            ))}
          </Row>
        ) : null}
      </dl>
    </div>
  )
}

function entryLook(kind: Entry['kind']): { bg: string; icon: IconName } {
  switch (kind) {
    case 'deferred':
    case 'retry':
    case 'failed_over':
      return { bg: '#FEF3C7', icon: 'clock' }
    case 'bounced':
    case 'complained':
    case 'failed':
    case 'provider_rejected':
      return { bg: '#FEE2E2', icon: 'alertCircle' }
    case 'rejected':
    case 'not_delivered':
      return { bg: '#EEF0F3', icon: 'ban' }
    case 'opened':
    case 'clicked':
    case 'webhook':
      return { bg: '#DBEAFE', icon: kind === 'webhook' ? 'webhook' : 'eye' }
  }
  return { bg: '#DCFCE7', icon: 'check' }
}

function entryText(t: ReturnType<typeof useT>, d: Detail, e: Entry): { title: string; detail: ReactNode } {
  const provider = e.provider ?? d.provider?.name ?? ''
  switch (e.kind) {
    case 'accepted': {
      const ip = d.ip ? ` · ${d.ip}` : ''
      if (d.raw) return { title: t(ml.evAccepted), detail: t(ml.evAcceptedSmtp, { key: d.keyRef?.name ?? '', ip }) }
      if (!d.keyRef && d.sentBy) return { title: t(ml.evAccepted), detail: t(ml.evAcceptedPerson, { name: d.sentBy }) }
      return { title: t(ml.evAccepted), detail: t(ml.evAcceptedApi, { key: d.keyRef?.name ?? '', ip, status: e.status ?? 202 }) }
    }
    case 'rendered': {
      const tpl = d.template ? `${d.template.key} v${d.template.version ?? ''}` : t(mt.raw)
      const sensitive = d.sensitiveSent.length ? t(ml.evRenderedSensitive, { count: d.sensitiveSent.length }) : ''
      return { title: t(ml.evRendered), detail: t(ml.evRenderedDetail, { template: tpl, size: kb(d.contentBytes), variables: d.variables, sensitive }) }
    }
    case 'scheduled':
      return { title: t(ml.evScheduled), detail: d.sendAt ? stamp(d.sendAt) : '' }
    case 'routed': {
      const which = d.route.failedOver ? t(ml.evFallback) : t(ml.evPrimary)
      const matched = d.route.matched ? t(ml.evMatched, { matched: d.route.matched }) : t(ml.evNoCondition)
      return { title: t(ml.evRouted, { provider }), detail: t(ml.evRoutedDetail, { which, matched }) }
    }
    case 'handed':
      return {
        title: t(ml.evHanded, { provider }),
        detail: e.providerMessageId ? <T msg={ml.evHandedDetail} values={{ provider, id: <span className="mono">{e.providerMessageId}</span> }} /> : e.detail,
      }
    case 'retry':
      return { title: t(ml.evRetry, { provider }), detail: e.detail }
    case 'failed_over':
      return { title: t(ml.evFailedOver, { to: e.to ?? provider }), detail: e.detail }
    case 'failed':
      return { title: t(ml.evFailed), detail: e.detail }
    case 'rejected':
      return { title: t(ml.evRejected), detail: e.detail }
    case 'not_delivered':
      return { title: t(ml.evNotDelivered), detail: e.detail }
    case 'delivered': {
      const after = d.deliveryMs !== undefined ? t(ml.evDeliveredAfter, { time: secs(d.deliveryMs) }) : ''
      return {
        title: t(ml.evDelivered),
        detail: (
          <>
            {e.detail?.split(' · ')[0] ?? ''}
            {e.code ? (
              <>
                {' · '}
                <span className="mono">
                  {e.code}
                  {e.detail?.includes(' · ') ? ` ${e.detail.split(' · ').slice(1).join(' · ')}` : ''}
                </span>
              </>
            ) : null}
            {after}
          </>
        ),
      }
    }
    case 'deferred':
      return { title: t(ml.evDeferred), detail: [e.code, e.detail].filter(Boolean).join(' ') }
    case 'bounced':
      return { title: t(ml.evBounced), detail: [e.code, e.detail].filter(Boolean).join(' ') }
    case 'complained':
      return { title: t(ml.evComplained), detail: e.detail }
    case 'opened':
      return { title: t(ml.evOpened), detail: e.detail }
    case 'clicked':
      return { title: t(ml.evClicked), detail: e.url ?? e.detail }
    case 'provider_rejected':
      return { title: t(ml.evProviderRejected, { provider }), detail: [e.code, e.detail].filter(Boolean).join(' ') }
    case 'webhook':
      return { title: t(ml.evWebhook, { event: e.event ?? '' }), detail: t(ml.evWebhookDetail, { url: e.url ?? '', state: e.state ?? '' }) }
  }
  return { title: e.kind, detail: e.detail }
}

function Timeline({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="flex flex-col gap-1">
      <h3 className="h2">{t(ml.timeline)}</h3>
      {d.timeline.map((e, i) => {
        const look = entryLook(e.kind)
        const txt = entryText(t, d, e)
        return (
          <div key={i} className="tx-ev">
            <span className="ic" style={{ background: look.bg }}>
              <Icon name={look.icon} size={14} />
            </span>
            <div className="tx">
              <span className="font-semibold">{txt.title}</span>
              {txt.detail ? <span className="k">{txt.detail}</span> : null}
            </div>
            <span className="tm">{stampMs(e.at)}</span>
          </div>
        )
      })}
      {d.provider && !d.raw ? (
        <Link to="/w/$workspace/providers/events" params={{ workspace: slug }} className="tx-lnk" style={{ marginTop: 6 }}>
          {t(ml.rawEvents, { provider: d.provider.name })}
        </Link>
      ) : null}
    </div>
  )
}

/** The request as it was made, sensitive values masked: JSON with the boards' colours. */
function Request({ d }: { d: Detail }) {
  const t = useT()
  const body = JSON.stringify(d.request.body ?? {}, null, 2)
  const masked = new Set(d.sensitiveSent)
  const lines = body.split('\n').map((line, i) => {
    const m = /^(\s*)("[^"]+")(:\s*)(.*?)(,?)$/.exec(line)
    if (!m) return <span key={i}>{line}</span>
    const [, pad, k, colon, v, comma] = m
    const name = JSON.parse(k!) as string
    const isMask = masked.has(name) && v === '"••••••"'
    const cls = /^"/.test(v ?? '') ? (isMask ? 'c3' : 'c2') : 'c3'
    return (
      <span key={i}>
        {pad}
        <span className="c1">{k}</span>
        {colon}
        {v && !/^[[{]/.test(v) ? <span className={cls}>{v}</span> : v}
        {comma}
        {isMask ? (
          <>
            {'  '}
            <span className="cm">{`// ${t(ml.sensitiveNever)}`}</span>
          </>
        ) : null}
      </span>
    )
  })
  return (
    <div className="flex flex-col gap-2.5">
      <h3 className="h2">{t(ml.request)}</h3>
      <pre className="code tx-code">
        {d.request.smtp ? (
          <span className="c4">{t(ml.smtpRequest)}</span>
        ) : (
          <>
            <span className="c4">{d.request.method}</span> {d.request.path}
            {'\n'}
            {d.request.idempotencyKey ? (
              <>
                <span className="c1">Idempotency-Key</span>: {d.request.idempotencyKey}
                {'\n'}
              </>
            ) : null}
            {'\n'}
            {lines.map((l, i) => (
              <span key={i}>
                {l}
                {i < lines.length - 1 ? '\n' : ''}
              </span>
            ))}
          </>
        )}
      </pre>
    </div>
  )
}

type View = 'html' | 'text' | 'headers'

function ContentPanel({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [view, setView] = useState<View>('html')
  const c = d.content
  const q = $api.useQuery('get', `${API}/messages/{messageId}/content`, { params: { path: { workspace: slug, messageId: d.id } } }, { enabled: c.available && c.canView, staleTime: Infinity })
  let body: ReactNode
  if (c.removedAt) body = <Note tone="neutral">{t(ml.contentRemoved, { date: dayMonthYear(c.removedAt), days: c.retentionDays, until: dayMonthYear(c.metadataUntil) })}</Note>
  else if (!c.available) body = <Note tone="neutral">{t(ml.contentNotKept, { until: dayMonthYear(c.metadataUntil) })}</Note>
  else if (!c.canView)
    body = (
      <Note tone="neutral">
        <T
          msg={c.helpers.length ? ml.contentForbidden : ml.contentForbiddenNoNames}
          values={{ permission: <span className="mono">transactional:content</span>, names: c.helpers.join(', ') }}
        />
      </Note>
    )
  else if (q.error) body = <ErrorPanel title={errorMessage(q.error)} />
  else if (!q.data) body = <Skeleton w="100%" h={320} />
  else if (view === 'html')
    body = (
      <div className="flex justify-center rounded-[10px] border border-line p-4" style={{ background: '#E9ECF0' }}>
        <iframe
          title={t(ml.renderedLabel)}
          sandbox=""
          srcDoc={q.data.html}
          className="block rounded bg-white"
          style={{ width: 560, maxWidth: '100%', height: 620, border: 0, boxShadow: '0 1px 3px rgba(11,18,32,.12)' }}
        />
      </div>
    )
  else if (view === 'text') body = <pre className="code tx-code whitespace-pre-wrap">{q.data.text}</pre>
  else
    body = (
      <dl className="kv">
        {q.data.headers.map((h) => (
          <Row key={h.name} k={h.name}>
            <span className="mono" style={{ overflowWrap: 'anywhere' }}>
              {h.value}
            </span>
          </Row>
        ))}
      </dl>
    )
  return (
    <div className="flex min-w-0 flex-col gap-2.5">
      <div className="row" style={{ justifyContent: 'space-between' }}>
        <h3 className="h2">{t(ml.content)}</h3>
        {c.available && c.canView && !c.removedAt ? (
          <Segmented<View>
            label={t(ml.contentView)}
            value={view}
            onChange={setView}
            items={[
              { value: 'html', label: t(ml.html) },
              { value: 'text', label: t(ml.text) },
              { value: 'headers', label: t(ml.headers) },
            ]}
          />
        ) : null}
      </div>
      {body}
      {c.available && c.canView && !c.removedAt && c.expiresAt ? (
        <span className="k">
          {t(ml.contentKept, { days: c.retentionDays, date: dayMonthYear(c.expiresAt) })}
          {c.masked.length ? (
            <>
              {' '}
              <T
                msg={ml.contentMasked}
                values={{
                  names: (
                    <>
                      {c.masked.map((n, i) => (
                        <span key={n}>
                          {i ? ', ' : ''}
                          <span className="mono">{n}</span>
                        </span>
                      ))}
                    </>
                  ),
                  count: c.masked.length,
                }}
              />
            </>
          ) : null}
        </span>
      ) : null}
    </div>
  )
}
