import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { channelIcon } from '@/features/channels/ChannelSwitch'
import { formatList, T, useT, type MessageDescriptor } from '@/i18n'
import { ago, clock, count, dayMonthDate, dayMonthYear, inZone, monthName, money } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { BudgetMeter, CountryChips, price, providerColor, regionSentence, throwIfForbidden, useRefreshSms } from './common'
import { m as sm } from './messages'
import { m } from './review.messages'
import './sms.css'

type Review = Schemas['SmsReview']
type Check = Schemas['SmsCheck']

const checkName: Record<Check['key'], MessageDescriptor> = {
  audience: m.nAudience,
  senders: m.nSenders,
  registration: m.nRegistration,
  routes: m.nRoutes,
  consent: m.nConsent,
  suppressions: m.nSuppressions,
  quiet_hours: m.nQuiet_hours,
  time_zones: m.nTime_zones,
  opt_out: m.nOpt_out,
  sender_named: m.nSender_named,
  links: m.nLinks,
  length: m.nLength,
  budget: m.nBudget,
}

const reasonWord: Record<string, MessageDescriptor> = {
  opted_out: m.rOpted_out,
  landline: m.rLandline,
  not_in_service: m.rNot_in_service,
  invalid: m.rInvalid,
  filtered: m.rFiltered,
  provider_list: m.rProvider_list,
  manual: m.rManual,
}

/** A moment as a UTC clock time: "08:00". */
const utc = (iso: string) => inZone(iso, 'UTC').time

/** Review & send for an SMS campaign (SmsReview, SmsReviewStates). */
export function SmsReviewPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/review', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  const data = q.data
  const name = data?.campaign.name ?? ''
  return (
    <Page
      title={t(m.title)}
      breadcrumb={[
        { label: t(m.campaigns), to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: name, to: '/w/$workspace/campaigns/$campaignId/sms', params: { workspace: slug, campaignId } },
        { label: t(m.title) },
      ]}
      badges={
        <Badge tone="neutral" className="ml-1 gap-[5px]">
          {channelIcon.sms}
          {t(m.sms)}
        </Badge>
      }
      actions={data ? <HeaderActions data={data} onRerun={() => void q.refetch()} rerunning={q.isFetching} /> : null}
    >
      {q.isError ? (
        <ErrorPanel title={t(m.errorTitle)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !data ? (
        <div className="card flex flex-col gap-3 p-5" aria-busy="true" aria-label={t(m.loading)}>
          <Skeleton h={60} />
          <Skeleton h={200} />
        </div>
      ) : (
        <Body data={data} />
      )}
    </Page>
  )
}

function HeaderActions({ data, onRerun, rerunning }: { data: Review; onRerun: () => void; rerunning: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      {data.campaign.status === 'draft' && data.campaign.autosavedAt ? <Badge tone="neutral">{t(m.draftSaved, { time: clock(data.campaign.autosavedAt) })}</Badge> : null}
      <ButtonLink to="/w/$workspace/campaigns/$campaignId/sms" params={{ workspace: slug, campaignId: data.campaign.id }}>
        {t(m.backToEditor)}
      </ButtonLink>
      <Button loading={rerunning} onClick={onRerun}>
        {t(m.rerun)}
      </Button>
    </>
  )
}

function Body({ data }: { data: Review }) {
  const t = useT()
  const [confirming, setConfirming] = useState(false)
  const [showWarnings, setShowWarnings] = useState(false)
  const sendable = data.verdict !== 'blocked' && data.access.canSend && (!data.approval.required || data.approval.valid)
  const scheduled = data.campaign.status === 'scheduled'
  return (
    <>
      <Verdict data={data} onSend={() => setConfirming(true)} sendable={sendable} onWarnings={() => setShowWarnings(true)} />
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-col gap-4" style={{ flex: '1 1 560px' }}>
          <Checks data={data} focusWarnings={showWarnings} />
          <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(300px, 100%), 1fr))' }}>
            <ScheduleCard data={data} />
            <div className="flex min-w-0 flex-col gap-4">
              <Thresholds data={data} />
              <TestSend data={data} />
            </div>
          </div>
        </div>
        <aside className="flex min-w-0 flex-col gap-4" style={{ flex: '1 1 340px', maxWidth: 420 }}>
          <AudienceCard data={data} />
          <RoutesCard data={data} />
          <CostCard data={data} />
          <ApprovalsCard data={data} />
          {scheduled ? null : (
            <SendButton data={data} sendable={sendable} onSend={() => setConfirming(true)} big />
          )}
          {scheduled ? null : (
            <span className="k text-center">{t(m.largeRule, { recipients: count(data.large.recipients), cost: money(data.large.cost, 0) })}</span>
          )}
        </aside>
      </div>
      {confirming ? <ConfirmDialog data={data} onClose={() => setConfirming(false)} /> : null}
    </>
  )
}

function sendLabel(data: Review, t: ReturnType<typeof useT>): string {
  const n = t(m.scheduleCount, { count: data.texts })
  return data.schedule.mode === 'now' ? t(m.sendNowTo, { count: n }) : t(m.scheduleTo, { count: n })
}

function SendButton({ data, sendable, onSend, big }: { data: Review; sendable: boolean; onSend: () => void; big?: boolean }) {
  const t = useT()
  const reason = !data.access.canSend
    ? t(m.needsSend, { perms: formatList(data.access.missing), helpers: data.access.helpers.length ? formatList(data.access.helpers, 'disjunction') : 'none' })
    : undefined
  return (
    <Button
      variant="primary"
      className={big ? 'min-h-11 text-[14px]' : 'min-h-10 px-[18px]'}
      disabled={!sendable}
      disabledReason={reason}
      onClick={onSend}
    >
      {sendLabel(data, t)}
    </Button>
  )
}

/** The verdict: Ready, Ready with warnings, or Blocked. */
function Verdict({ data, onSend, sendable, onWarnings }: { data: Review; onSend: () => void; sendable: boolean; onWarnings: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const cls = { ready: 'g', warnings: 'w', blocked: 'b' }[data.verdict]
  const title = { ready: m.ready, warnings: m.warnings, blocked: m.blocked }[data.verdict]
  const scheduled = data.campaign.status === 'scheduled'
  return (
    <section className={`sms-verdict ${cls}`}>
      <Icon name={data.verdict === 'blocked' ? 'alertCircle' : data.verdict === 'warnings' ? 'warningTriangle' : 'check'} size={28} />
      <div className="flex min-w-0 flex-col gap-0.5" style={{ flex: '1 1 300px' }}>
        <span className="vt">{t(title)}</span>
        <span>
          {data.verdict === 'blocked'
            ? t(m.summaryBlocked, { blocks: data.blocks, warnings: data.warnings, passed: data.passed })
            : t(m.summary, { passed: data.passed, warnings: data.warnings, blocks: data.blocks, time: clock(data.checkedAt) })}
        </span>
      </div>
      <div className="flex flex-wrap gap-2.5">
        {data.warnings && data.verdict !== 'blocked' ? <Button onClick={onWarnings}>{t(m.reviewWarnings, { count: data.warnings })}</Button> : null}
        {scheduled ? (
          <Unschedule data={data} />
        ) : data.campaign.status === 'sent' || data.campaign.status === 'sending' ? (
          <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/sms-report" params={{ workspace: slug, campaignId: data.campaign.id }}>
            {t(m.viewReport)}
          </ButtonLink>
        ) : (
          <SendButton data={data} sendable={sendable} onSend={onSend} />
        )}
      </div>
    </section>
  )
}

function Unschedule({ data }: { data: Review }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const qc = useQueryClient()
  const un = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/unschedule', {
    onSuccess: () => {
      void refresh()
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
      toast({ kind: 'success', title: t(m.unscheduled) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (
    <>
      {data.send?.scheduledAt ? <span className="self-center">{t(m.scheduledFor, { date: dayMonthYear(data.send.scheduledAt), time: clock(data.send.scheduledAt) })}</span> : null}
      <Button loading={un.isPending} onClick={() => un.mutate({ params: { path: { workspace: slug, campaignId: data.campaign.id } } })}>
        {t(m.unschedule)}
      </Button>
    </>
  )
}

/** The checks: blocks, warnings, then the passes (collapsed while anything needs attention). */
function Checks({ data, focusWarnings }: { data: Review; focusWarnings: boolean }) {
  const t = useT()
  const blocks = data.checks.filter((c) => c.status === 'block')
  const warns = data.checks.filter((c) => c.status === 'warn')
  const passes = data.checks.filter((c) => c.status === 'pass')
  const [showPasses, setShowPasses] = useState(blocks.length === 0)
  return (
    <section className="card flex min-w-0 flex-none flex-col overflow-hidden" aria-label={t(m.checksLabel)}>
      {blocks.length ? (
        <>
          <div className="sms-chkg">
            <h2 className="h2">{t(m.blocking, { count: blocks.length })}</h2>
            <span className="k">{t(m.fixFirst, { count: blocks.length })}</span>
          </div>
          {blocks.map((c, i) => (
            <CheckRow key={`b${i}`} c={c} data={data} />
          ))}
        </>
      ) : null}
      {warns.length ? (
        <>
          <div className="sms-chkg" tabIndex={focusWarnings ? -1 : undefined} ref={(el) => (focusWarnings && el ? el.scrollIntoView({ block: 'nearest' }) : undefined)}>
            <h2 className="h2">{t(m.warningsGroup, { count: warns.length })}</h2>
          </div>
          {warns.map((c, i) => (
            <CheckRow key={`w${i}`} c={c} data={data} />
          ))}
        </>
      ) : null}
      <div className="sms-chkg">
        <h2 className="h2">{t(m.passed, { count: passes.length })}</h2>
        {showPasses ? (
          <span className="flex items-center gap-2">
            <span className="k">{t(m.smsChecks)}</span>
            {blocks.length || warns.length ? (
              <button type="button" className="sms-lbtn k" onClick={() => setShowPasses(false)}>
                {t(m.hide)}
              </button>
            ) : null}
          </span>
        ) : (
          <Badge tone="neutral">{t(m.collapsed)}</Badge>
        )}
      </div>
      {showPasses ? (
        passes.map((c, i) => <CheckRow key={`p${i}`} c={c} data={data} />)
      ) : (
        <div className="sms-chkr">
          <Badge tone="ok">{t(m.pass)}</Badge>
          <div className="rt">
            <span className="t">{t(m.passedSummary, { count: passes.length, names: formatList(passes.map((c) => t(checkName[c.key]))) })}</span>
          </div>
          <div className="ac">
            <Button size="sm" onClick={() => setShowPasses(true)}>
              {t(m.showAll, { count: passes.length })}
            </Button>
          </div>
        </div>
      )}
    </section>
  )
}

function CheckRow({ c, data }: { c: Check; data: Review }) {
  const t = useT()
  const { title, text, actions } = useCheckWords(c, data)
  const cls = c.status === 'warn' ? ' w' : c.status === 'block' ? ' b' : ''
  return (
    <div className={`sms-chkr${cls}`}>
      <Badge tone={c.status === 'pass' ? 'ok' : c.status === 'warn' ? 'warn' : 'bad'}>{c.status === 'pass' ? t(m.pass) : c.status === 'warn' ? t(m.warn) : t(m.block)}</Badge>
      <div className="rt">
        <span className="t">{title}</span>
        {text ? <span className="k">{text}</span> : null}
      </div>
      {actions ? <div className="ac">{actions}</div> : null}
    </div>
  )
}

/** Words a check from its facts. */
function useCheckWords(c: Check, data: Review): { title: string; text?: string; actions?: ReactNode } {
  const t = useT()
  const { slug } = useWorkspace()
  const month = monthName(data.budget.month, 'long')
  const helpers = data.access.helpers.length ? formatList(data.access.helpers, 'disjunction') : ''
  switch (c.key) {
    case 'audience':
      return c.status === 'block' ? { title: t(m.cAudienceBlock), text: t(m.cAudienceBlockText) } : { title: t(m.cAudiencePass, { count: count(c.count) }) }
    case 'senders':
      return { title: t(m.cSendersBlock) }
    case 'consent': {
      const noConsent = c.total - c.count
      return {
        title: t(m.cConsentPass, { count: count(data.funnel.canReceive), lists: formatList(c.lists) }),
        text: noConsent > 0 ? t(m.cConsentText, { count: count(noConsent) }) : undefined,
      }
    }
    case 'suppressions': {
      const reasons = Object.entries(c.reasons)
        .filter(([, n]) => n > 0)
        .map(([r, n]) => (reasonWord[r] ? t(reasonWord[r]!, { count: n }) : `${n} ${r}`))
      return c.count
        ? { title: t(m.cSuppressionsPass, { count: c.count }), text: t(m.cSuppressionsText, { reasons: capitalize(formatList(reasons)), providers: formatList(data.routes.map((r) => r.provider)) }) }
        : { title: t(m.cSuppressionsNone) }
    }
    case 'registration': {
      if (c.status === 'warn') {
        const region = regionSentence(c.countries)
        return {
          title: c.choice === 'wait' ? t(m.cRegistrationWait, { count: c.count, region, sender: c.sender ?? '' }) : t(m.cRegistrationWarn, { count: c.count, region, sender: c.sender ?? '' }),
          text: t(m.cRegistrationWarnText, { provider: c.provider ?? '', date: c.since ? dayMonthYear(c.since) : '', sender: c.sender ?? '' }),
          actions: <PendingPolicy data={data} region={region} />,
        }
      }
      return {
        title: c.total ? t(m.cRoutesPass) : t(m.cRoutesPassAll),
        text: c.senders
          .map((s) =>
            t(m.senderLine, {
              sender: s.sender,
              status: s.since && (s.registration === 'verified' || s.registration === 'approved') ? `${s.registration}_on` : s.registration,
              date: s.since ? dayMonthDate(s.since) : '',
              region: regionSentence(s.countries),
            }),
          )
          .join(' · '),
      }
    }
    case 'routes':
      return { title: t(m.cRoutesBlock, { count: c.count, region: regionSentence(c.countries) }), text: t(m.cRoutesBlockText, { region: regionSentence(c.countries) }) }
    case 'quiet_hours': {
      const groups = data.schedule.groups
      const first = groups[0]
      const last = groups[groups.length - 1]
      return {
        title: c.local ? t(m.cQuietPass, { time: c.local, window: c.window ?? data.schedule.window }) : t(m.cQuietPassNow, { window: c.window ?? data.schedule.window }),
        text:
          first && last && groups.length > 1
            ? t(m.cQuietSpan, { from: utc(first.from), fromRegion: regionSentence(first.countries), to: utc(last.to), toRegion: regionSentence(last.countries) })
            : undefined,
      }
    }
    case 'time_zones':
      return {
        title: t(m.cTimeZonesWarn, { count: c.count }),
        text: t(m.cTimeZonesText, { area: c.byArea, country: c.byCountry }),
        actions: (
          <ButtonLink size="sm" to="/w/$workspace/audience" params={{ workspace: slug }}>
            {t(m.reviewContacts, { count: c.count })}
          </ButtonLink>
        ),
      }
    case 'opt_out':
      return c.status === 'block'
        ? {
            title: t(m.cOptOutBlock),
            text: t(m.cOptOutBlockText),
            actions: (
              <ButtonLink size="sm" to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }}>
                {t(m.openRules)}
              </ButtonLink>
            ),
          }
        : { title: t(m.cOptOutPass, { text: c.text ?? '' }) }
    case 'sender_named':
      return c.status === 'pass' ? { title: t(m.cSenderPass, { text: c.text ?? '' }) } : { title: t(m.cSenderWarn), text: t(m.cSenderWarnText) }
    case 'links':
      if (c.status === 'block' && c.shorteners.length) {
        return { title: t(m.cLinksBlock, { hosts: formatList(c.shorteners), count: c.shorteners.length }), text: t(m.cLinksBlockText, { domain: c.domain ?? '' }) }
      }
      if (c.status === 'block') return { title: t(m.cLinksWarn, { domain: c.domain ?? '' }), text: t(m.cLinksWarnText) }
      return c.links
        ? { title: t(m.cLinksPass, { count: c.links, domain: c.domain ?? '' }), text: c.verifiedOn ? t(m.cLinksPassText, { domain: c.domain ?? '', date: dayMonthDate(c.verifiedOn) }) : undefined }
        : { title: t(m.cLinksNone) }
    case 'length': {
      const encoding = c.encoding === 'ucs2' ? 'UCS-2' : 'GSM-7'
      return c.status === 'warn'
        ? { title: t(m.cLengthWarn, { segments: c.segments, encoding }), text: t(m.cLengthWarnText, { segments: c.segments }) }
        : { title: t(m.cLengthPass, { characters: c.characters, segments: c.segments, encoding }) }
    }
    case 'budget':
      if (!c.budget) return { title: t(m.cBudgetNone) }
      if (c.status === 'block') {
        return {
          title: t(m.cBudgetBlock, { cost: money(c.cost), remaining: money(c.remaining), month }),
          text: t(m.cBudgetBlockText, { month, spent: money(c.spent), budget: money(c.budget, Number.isInteger(c.budget) ? 0 : 2), helpers }),
          actions: (
            <>
              {data.access.canManageRules ? (
                <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }} hash="budget">
                  {t(m.raiseBudget)}
                </ButtonLink>
              ) : null}
              <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/sms" params={{ workspace: slug, campaignId: data.campaign.id }}>
                {t(m.fewerPeople)}
              </ButtonLink>
            </>
          ),
        }
      }
      return {
        title: t(m.cBudgetPass, { cost: money(c.cost), remaining: money(c.remaining), month }),
        text: t(m.cBudgetPassText, { after: money(c.after), budget: money(c.budget, Number.isInteger(c.budget) ? 0 : 2), under: (c.after / c.budget) * 100 < c.alertPct ? 'yes' : 'no', alert: c.alertPct }),
      }
  }
  return { title: c.key }
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

/** Skip them, or wait for approval: what a pending registration does to its recipients. */
function PendingPolicy({ data, region }: { data: Review; region: string }) {
  const t = useT()
  const save = useSendSettings(data)
  return (
    <div className="seg" role="group" aria-label={t(m.recipientsIn, { region })}>
      <button type="button" aria-pressed={data.pendingPolicy === 'skip'} disabled={!data.access.canEdit} onClick={() => save({ pendingPolicy: 'skip' })}>
        {t(m.skipThem)}
      </button>
      <button type="button" aria-pressed={data.pendingPolicy === 'wait'} disabled={!data.access.canEdit} onClick={() => save({ pendingPolicy: 'wait' })}>
        {t(m.waitApproval)}
      </button>
    </div>
  )
}

/** Saves the send settings; the review comes back with the checks run again. */
function useSendSettings(data: Review) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/send-settings', {
    onSuccess: (next) => qc.setQueryData(['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/review', { params: { path: { workspace: slug, campaignId: data.campaign.id } } }], next),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (body: Schemas['SmsSendSettings']) => save.mutate({ params: { path: { workspace: slug, campaignId: data.campaign.id } }, body })
}

function Card({ title, lead, aside, children }: { title: string; lead?: ReactNode; aside?: ReactNode; children: ReactNode }) {
  return (
    <section className="card flex min-w-0 flex-col gap-3 px-5 py-[18px]">
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{title}</h2>
          {lead ? <span className="k">{lead}</span> : null}
        </div>
        {aside}
      </div>
      {children}
    </section>
  )
}

/** Schedule: now, at one time, or at a time in each recipient's time zone. */
function ScheduleCard({ data }: { data: Review }) {
  const t = useT()
  const save = useSendSettings(data)
  const s = data.schedule
  const editable = data.access.canEdit && data.campaign.status === 'draft'
  const [date, setDate] = useState(s.localDate ?? new Date().toISOString().slice(0, 10))
  const [time, setTime] = useState(s.localTime ?? '10:00')
  const [at, setAt] = useState(s.sendAt ? s.sendAt.slice(0, 16) : '')
  const times: string[] = []
  for (let mins = s.windowFrom; mins < s.windowTo; mins += 30) times.push(`${String(Math.floor(mins / 60)).padStart(2, '0')}:${String(mins % 60).padStart(2, '0')}`)
  const held = s.now.filter((g) => g.held)
  const heldCount = held.reduce((n, g) => n + g.recipients, 0)
  return (
    <Card title={t(m.schedule)}>
      <OptionCard
        name="when"
        value="now"
        checked={s.mode === 'now'}
        disabled={!editable}
        onChange={() => save({ mode: 'now' })}
        title={t(m.now)}
        description={t(m.nowText, { window: s.window })}
      />
      <OptionCard
        name="when"
        value="at"
        checked={s.mode === 'at'}
        disabled={!editable}
        onChange={() => save({ mode: 'at', sendAt: at ? new Date(at).toISOString() : undefined })}
        title={t(m.at)}
        description={t(m.atText)}
      >
        {s.mode === 'at' ? (
          <span className="mt-2 flex">
            <input
              className="in"
              type="datetime-local"
              aria-label={t(m.sendAt)}
              value={at}
              disabled={!editable}
              onChange={(e) => {
                setAt(e.target.value)
                if (e.target.value) save({ mode: 'at', sendAt: new Date(e.target.value).toISOString() })
              }}
            />
          </span>
        ) : null}
      </OptionCard>
      <OptionCard
        name="when"
        value="local"
        checked={s.mode === 'local'}
        disabled={!editable}
        onChange={() => save({ mode: 'local', localDate: date, localTime: time })}
        title={t(m.local)}
        description={s.localDate && s.localTime ? t(m.localText, { date: dayMonthYear(`${s.localDate}T12:00:00Z`), time: s.localTime }) : undefined}
      >
        {s.mode === 'local' ? (
          <span className="mt-2 flex flex-col gap-1.5">
            <span className="grid grid-cols-2 gap-2">
              <input
                className="in"
                type="date"
                aria-label={t(m.date)}
                value={date}
                disabled={!editable}
                onChange={(e) => {
                  setDate(e.target.value)
                  save({ mode: 'local', localDate: e.target.value, localTime: time })
                }}
              />
              <select
                className="in"
                aria-label={t(m.localTime)}
                value={time}
                disabled={!editable}
                onChange={(e) => {
                  setTime(e.target.value)
                  save({ mode: 'local', localDate: date, localTime: e.target.value })
                }}
              >
                {times.map((x) => (
                  <option key={x}>{x}</option>
                ))}
              </select>
            </span>
            <span className="k">{t(m.onlyInside, { window: s.window })}</span>
            <span className="mt-0.5 flex flex-col gap-1.5">
              {s.groups.map((g) => (
                <span key={g.from + g.countries.join()} className="sms-sum">
                  <span>
                    <T
                      msg={m.groupLine}
                      values={{
                        utc: utc(g.from) === utc(g.to) ? t(m.groupUtc, { time: utc(g.from) }) : t(m.groupUtcRange, { from: utc(g.from), to: utc(g.to) }),
                        region: regionSentence(g.countries),
                        mono: (c) => <span className="mono text-[12px]">{c}</span>,
                      }}
                    />
                  </span>
                  <span className="n">{count(g.recipients)}</span>
                </span>
              ))}
            </span>
          </span>
        ) : null}
      </OptionCard>
      {s.mode === 'now' && heldCount > 0 ? (
        <Note tone="warn" role="note">
          <span className="flex flex-col gap-1">
            <span className="font-semibold">{t(m.heldTitle)}</span>
            <span className="flex flex-col gap-1">
              {s.now.map((g) => (
                <span key={g.countries.join()} className="sms-sum">
                  <span>
                    {regionSentence(g.countries)} · {g.held ? t(m.groupOutside, { time: g.localFrom, window: s.window }) : t(m.groupLocal, { from: g.localFrom, to: g.localTo })}
                  </span>
                  <span className="flex items-center gap-2">
                    <span className="n">{count(g.recipients)}</span>
                    <Badge tone={g.held ? 'warn' : 'ok'}>{g.held ? t(m.groupHeld, { time: g.until }) : t(m.groupSent)}</Badge>
                  </span>
                </span>
              ))}
            </span>
          </span>
        </Note>
      ) : null}
    </Card>
  )
}

function Thresholds({ data }: { data: Review }) {
  const t = useT()
  const save = useSendSettings(data)
  const editable = data.access.canEdit && data.campaign.status === 'draft'
  const th = data.thresholds
  const row = (k: keyof Schemas['SmsThresholds'], label: string) => (
    <div className="sms-nr">
      <span className="flex min-w-0 flex-[1_1_150px] flex-col gap-px">
        <span className="text-[13px] font-medium">{label}</span>
      </span>
      <span className="inline-flex flex-none items-center gap-1.5">
        <input
          className="in"
          type="text"
          inputMode="decimal"
          aria-label={label}
          style={{ width: 60, textAlign: 'right' }}
          defaultValue={String(th[k])}
          disabled={!editable}
          onBlur={(e) => {
            const v = Number(e.target.value)
            if (v > 0 && v <= 100 && v !== th[k]) save({ thresholds: { ...th, [k]: v } })
          }}
        />
        <span className="k">%</span>
      </span>
    </div>
  )
  return (
    <Card title={t(m.thresholds)}>
      <div>
        {row('failed', t(m.failedAbove))}
        {row('filtered', t(m.filteredAbove))}
        {row('optOuts', t(m.optOutsAbove))}
      </div>
      <span className="k">{t(m.thresholdsNote)}</span>
    </Card>
  )
}

function TestSend({ data }: { data: Review }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [to, setTo] = useState('all')
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/test', {
    onSuccess: (d) => {
      toast({ kind: 'success', title: t(m.testsSent, { count: d.items.length }) })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/review'] })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (
    <Card title={t(m.testSend)}>
      <div className="flex flex-nowrap gap-2">
        <Select aria-label={t(m.testNumbers)} className="min-w-0 flex-1" value={to} onChange={(e) => setTo(e.target.value)}>
          {data.tests.length > 1 ? <option value="all">{t(m.allTests, { count: data.tests.length })}</option> : null}
          {data.tests.map((n) => (
            <option key={n.id} value={n.id}>
              {n.name} · {n.display}
            </option>
          ))}
        </Select>
        <Button
          loading={send.isPending}
          disabled={!data.tests.length}
          onClick={() => send.mutate({ params: { path: { workspace: slug, campaignId: data.campaign.id } }, body: to === 'all' ? {} : { numbers: [to] } })}
        >
          {t(m.sendTest)}
        </Button>
      </div>
      {data.tests
        .filter((n) => n.lastTest)
        .map((n) => (
          <div key={n.id} className="sms-sum items-center">
            <span className="flex flex-col">
              <span>
                {n.name} · <span className="mono text-[12px]">{n.display}</span>
              </span>
              <span className="k">{t(m.testVia, { time: clock(n.lastTest!.at), provider: n.lastTest!.provider, sender: n.lastTest!.senderId ? n.lastTest!.sender : 'none' })}</span>
            </span>
            <Badge tone={n.lastTest!.status === 'delivered' ? 'ok' : n.lastTest!.status === 'failed' || n.lastTest!.status === 'undelivered' ? 'bad' : 'info'}>
              {n.lastTest!.status === 'delivered' ? t(m.delivered) : n.lastTest!.status === 'failed' || n.lastTest!.status === 'undelivered' ? t(m.notDelivered) : t(m.sending)}
            </Badge>
          </div>
        ))}
    </Card>
  )
}

function AudienceCard({ data }: { data: Review }) {
  const t = useT()
  const f = data.funnel
  return (
    <Card title={t(m.audience)}>
      <div className="sms-sum">
        <span>{formatList(data.lists)}</span>
        <span className="n">{count(f.base)}</span>
      </div>
      <div className="sms-sum text-ink-3">
        <span>{t(m.noConsent)}</span>
        <span className="n">−{count(f.noConsent)}</span>
      </div>
      <div className="sms-sum text-ink-3">
        <span>{t(m.suppressionList)}</span>
        <span className="n">−{count(f.suppressed)}</span>
      </div>
      <div className="sms-sum border-t border-line-2 pt-2">
        <span>{t(m.canReceive)}</span>
        <span className="n">{count(f.canReceive)}</span>
      </div>
      {f.skipped ? (
        <div className="sms-sum text-ink-3">
          <span>{data.pendingPolicy === 'wait' ? t(m.heldUntilApproved, { region: regionSentence(f.skippedIn) }) : t(m.skippedForNow, { region: regionSentence(f.skippedIn) })}</span>
          <span className="n">−{count(f.skipped)}</span>
        </div>
      ) : null}
      <div className="sms-sum border-t border-line pt-2.5 text-[15px]">
        <span className="font-semibold">{t(m.toSend)}</span>
        <span className="n text-[20px]">{count(f.toSend)}</span>
      </div>
    </Card>
  )
}

function RoutesCard({ data }: { data: Review }) {
  const t = useT()
  const order = data.routes.map((r) => r.provider)
  const duration = (secs: number) => (secs < 60 ? t(m.seconds) : t(m.underMinutes, { n: Math.ceil(secs / 60) }))
  const last = data.schedule.groups[data.schedule.groups.length - 1]
  return (
    <Card title={t(m.routeBreakdown)}>
      <div className="flex flex-col gap-1.5">
        {data.routes.map((r) => (
          <div key={r.provider + r.sender} className="flex flex-col gap-1.5">
            <div className="sms-sum">
              <span className="flex items-center gap-2">
                <span className="dot" style={{ color: providerColor(order, r.provider) }} aria-hidden="true" />
                {r.provider} · <span className={r.sender.startsWith('+') ? 'mono' : undefined}>{r.sender}</span>
              </span>
              <span className="n">{count(r.recipients)}</span>
            </div>
            <div className="bar">
              <i style={{ width: `${r.share.toFixed(1)}%`, background: providerColor(order, r.provider) }} />
            </div>
          </div>
        ))}
      </div>
      <table className="tbl text-[13px]">
        <thead>
          <tr>
            <th scope="col" className="!px-0">
              {t(m.provider)}
            </th>
            <th scope="col" className="r !px-0">
              {t(m.cost)}
            </th>
            <th scope="col" className="r !pl-3 !pr-0">
              {t(m.duration)}
            </th>
          </tr>
        </thead>
        <tbody>
          {data.routes.map((r) => (
            <tr key={r.provider + r.sender}>
              <td className="!px-0">{t(m.rate, { provider: r.provider, rate: r.rate })}</td>
              <td className="r n !px-0">{money(r.cost)}</td>
              <td className="r !pl-3 !pr-0">{duration(r.seconds)}</td>
            </tr>
          ))}
          <tr>
            <td className="!px-0 font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
              {t(m.total)}
            </td>
            <td className="r n !px-0" style={{ borderTop: '1px solid var(--line)' }}>
              {money(data.cost)}
            </td>
            <td className="r !pl-3 !pr-0" style={{ borderTop: '1px solid var(--line)' }}>
              {data.schedule.end ? t(m.byTime, { time: utc(data.schedule.end) }) : last ? t(m.byTime, { time: utc(last.to) }) : ''}
            </td>
          </tr>
        </tbody>
      </table>
      <span className="k">
        {data.routes
          .map((r) => t(m.routeNote, { provider: r.provider, count: r.recipients, duration: duration(r.seconds), spread: data.schedule.mode === 'local' && data.schedule.groups.length > 1 ? 'yes' : 'no' }))
          .join(' ')}
      </span>
    </Card>
  )
}

function CostCard({ data }: { data: Review }) {
  const t = useT()
  const month = monthName(data.budget.month, 'long')
  const b = data.budget
  const providers = formatList([...new Set(data.routes.map((r) => r.provider))].map((provider) => t(sm.pricingOf, { provider })))
  return (
    <Card title={t(m.costTitle)} lead={data.pricedOn ? t(m.costLead, { providers, date: dayMonthYear(data.pricedOn) }) : undefined}>
      {data.costs.map((c) => (
        <div key={c.country} className="sms-sum items-center">
          <span className="inline-flex items-center gap-2">
            <CountryChips codes={[c.country]} />
            {c.skipped >= c.recipients && c.skipped > 0 ? (
              <span className="k" style={{ color: '#92400E' }}>
                {t(m.skipped, { count: count(c.skipped) })}
              </span>
            ) : (
              <span className="k">{t(m.costLine, { count: count(c.recipients - c.skipped), price: price(c.price) })}</span>
            )}
          </span>
          <span className="n">{money(c.cost)}</span>
        </div>
      ))}
      <div className="sms-sum border-t border-line pt-2.5 text-[15px]">
        <span className="font-semibold">{t(m.totalLine, { texts: data.texts, segments: data.segmentsEach })}</span>
        <span className="n text-[20px]">{money(data.cost)}</span>
      </div>
      {b.budget ? (
        <div className="sms-bm">
          <div className="sms-sum">
            <span className="k">{t(m.budgetLabel, { month })}</span>
            <span className="n">
              <T msg={m.budgetValue} values={{ spent: money(b.spent), after: money(b.after), budget: money(b.budget), k: (x) => <span className="k">{x}</span> }} />
            </span>
          </div>
          <BudgetMeter spent={b.spent} budget={b.budget} alertPct={b.alertPct} send={data.cost} month={month} legend />
          {b.after > b.budget ? <span className="k text-bad-strong">{t(m.overBudget, { amount: money(b.after - b.budget), month })}</span> : null}
        </div>
      ) : null}
    </Card>
  )
}

/** Approvals: not needed and why, required (request it), waiting, approved. */
function ApprovalsCard({ data }: { data: Review }) {
  const t = useT()
  const { slug } = useWorkspace()
  const who = useMe().data
  const a = data.approval
  const [note, setNote] = useState('')
  const qc = useQueryClient()
  const refetch = () => qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/review'] })
  const request = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/approval', {
    onSuccess: () => {
      void refetch()
      toast({ kind: 'success', title: t(m.requested) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const remind = $api.useMutation('post', '/api/v1/workspaces/{workspace}/approvals/{approvalId}/remind', {
    onSuccess: () => toast({ kind: 'success', title: t(m.reminded) }),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const cancel = $api.useMutation('post', '/api/v1/workspaces/{workspace}/approvals/{approvalId}/cancel', {
    onSuccess: () => {
      void refetch()
      toast({ kind: 'success', title: t(m.cancelled) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const rules = (
    <Link to="/w/$workspace/settings/approvals" params={{ workspace: slug }} className="sms-lnk">
      {t(m.approvalRules)}
    </Link>
  )
  const req = a.request
  if (req && req.status === 'pending') {
    return (
      <Card title={t(m.waiting)} aside={<Badge tone="warn">{t(m.pending)}</Badge>}>
        <span className="k">{t(m.requestedAgo, { ago: ago(req.requestedAt), given: req.given, needed: req.needed })}</span>
        {a.approvers.map((p) => (
          <div key={p.id} className="flex items-center gap-2.5">
            <Avatar name={p.name} color={p.color} size={28} fontSize={11} />
            <span className="flex flex-col leading-[1.3]">
              <span className="font-semibold">{p.name}</span>
              <span className="k">
                {t(m.approverRole)}
                {p.presence ? ` · ${p.presence}` : ''}
              </span>
            </span>
          </div>
        ))}
        <span className="k">{t(m.lockedNote)}</span>
        <div className="flex flex-wrap gap-2">
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/sms" params={{ workspace: slug, campaignId: data.campaign.id }}>
            {t(m.editRestarts)}
          </ButtonLink>
          <Button size="sm" loading={remind.isPending} onClick={() => remind.mutate({ params: { path: { workspace: slug, approvalId: req.id } } })}>
            {t(m.remind)}
          </Button>
          <Button size="sm" variant="danger" loading={cancel.isPending} onClick={() => cancel.mutate({ params: { path: { workspace: slug, approvalId: req.id } } })}>
            {t(m.cancelRequest)}
          </Button>
        </div>
      </Card>
    )
  }
  if (req && req.status === 'approved' && a.valid) {
    return (
      <Card title={t(m.approvals)} aside={rules}>
        <Note tone="ok">{t(m.approvedSend)}</Note>
        <span className="k">{t(m.approved, { names: formatList(req.decisions.filter((d) => d.decision === "approve").map((d) => d.approver.name)) })}</span>
      </Card>
    )
  }
  if (a.required) {
    return (
      <Card title={t(m.approvalRequired)} aside={<Badge tone="warn">{t(m.rulesMatched, { count: a.rules.length })}</Badge>}>
        {a.rules.map((r) => (
          <span key={r.key} className="text-[13px] text-[#374151]">
            {r.label}
          </span>
        ))}
        <Field label={t(m.note)}>
          <Textarea rows={2} value={note} onChange={(e) => setNote(e.target.value)} disabled={!a.canRequest} />
        </Field>
        <span className="k">{t(m.approvers, { names: formatList(a.approvers.map((p) => p.name)), needed: a.needed })}</span>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="primary"
            disabled={!a.canRequest || data.verdict === 'blocked'}
            disabledReason={a.requestReason || undefined}
            loading={request.isPending}
            onClick={() => request.mutate({ params: { path: { workspace: slug, campaignId: data.campaign.id } }, body: { note: note.trim() || undefined } })}
          >
            {t(m.requestApproval)}
          </Button>
          {rules}
        </div>
      </Card>
    )
  }
  const p = data.policy
  return (
    <Card title={t(m.approvals)}>
      <span className="text-[13px] text-[#374151]">
        {t(m.noApprovalWhy, {
          audience: p.audienceRule ? 'yes' : 'no',
          min: count(p.audienceMin),
          cost: p.costRule ? 'yes' : 'no',
          amount: money(p.costAmount, Number.isInteger(p.costAmount) ? 0 : 2),
          name: who?.user.name ?? '',
        })}
      </span>
      {rules}
    </Card>
  )
}

/** Confirms the send; a large one asks for the recipient count. */
function ConfirmDialog({ data, onClose }: { data: Review; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshSms()
  const qc = useQueryClient()
  const [typed, setTyped] = useState('')
  const [error, setError] = useState<string>()
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/send')
  const n = data.texts
  const now = data.schedule.mode === 'now'
  const month = monthName(data.budget.month, 'long')
  const go = async () => {
    setError(undefined)
    if (data.large.required && typed.replace(/[,\s.]/g, '') !== String(n)) {
      setError(t(m.typeWrong, { count: count(n) }))
      return
    }
    try {
      const r = await send.mutateAsync({
        params: { path: { workspace: slug, campaignId: data.campaign.id } },
        body: { fingerprint: data.fingerprint, confirm: data.large.required ? String(n) : undefined, acknowledged: data.checks.filter((c) => c.status === 'warn').map((c) => c.key) },
      })
      void refresh()
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })
      toast({ kind: 'success', title: r.status === 'sending' ? t(m.sent, { name: data.campaign.name, count: count(r.recipients) }) : t(m.scheduled, { name: data.campaign.name, count: count(r.recipients) }) })
      void navigate({ to: '/w/$workspace/campaigns/$campaignId/sms-report', params: { workspace: slug, campaignId: data.campaign.id } })
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        setError(t(m.changed))
        void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/review'] })
        return
      }
      setError(errorMessage(e))
    }
  }
  const when = data.schedule.mode === 'local' && data.schedule.localDate ? t(m.localText, { date: dayMonthYear(`${data.schedule.localDate}T12:00:00Z`), time: data.schedule.localTime ?? '' }) : data.schedule.sendAt ? `${dayMonthYear(data.schedule.sendAt)} ${clock(data.schedule.sendAt)}` : t(m.now)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<Badge tone={data.verdict === 'ready' ? 'ok' : 'warn'}>{data.verdict === 'ready' ? t(m.ready) : t(m.warnings)}</Badge>}
      title={now ? t(m.confirmTitleNow, { name: data.campaign.name, count: t(m.scheduleCount, { count: n }) }) : t(m.confirmTitle, { name: data.campaign.name, count: t(m.scheduleCount, { count: n }) })}
      description={t(m.confirmChecks, { passed: data.passed })}
      footer={
        <>
          <Button onClick={onClose}>{t(m.back)}</Button>
          <Button variant="primary" loading={send.isPending} onClick={() => void go()}>
            {now ? t(m.confirmSend) : t(m.confirmSchedule)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {error ? <Note tone="bad">{error}</Note> : null}
        <dl className="kv">
          <dt>{t(m.confirmText)}</dt>
          <dd>{data.campaign.text}</dd>
          <dt>{t(m.confirmAudience)}</dt>
          <dd>{t(m.confirmAudienceValue, { lists: formatList(data.lists), count: count(data.funnel.canReceive) })}</dd>
          <dt>{t(m.confirmRoute)}</dt>
          <dd>{data.routes.map((r) => `${r.provider} ${count(r.recipients)}`).join(' · ')}</dd>
          <dt>{t(m.confirmCost)}</dt>
          <dd>{t(m.confirmCostValue, { cost: money(data.cost), month, spent: money(data.budget.spent), after: money(data.budget.after), budget: money(data.budget.budget) })}</dd>
          <dt>{t(m.confirmWhen)}</dt>
          <dd>{when}</dd>
        </dl>
        {data.large.required ? (
          <Field label={t(m.typeCount)} hint={t(m.typeHint)}>
            <Input value={typed} inputMode="numeric" onChange={(e) => setTyped(e.target.value)} />
          </Field>
        ) : null}
      </div>
    </Dialog>
  )
}

