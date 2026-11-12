import { useQueryClient } from '@tanstack/react-query'
import { Dialog as RDialog } from 'radix-ui'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Link } from '@tanstack/react-router'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { lowerFirstInSentence, T, useT } from '@/i18n'
import { clock, dayMonthDate, dayMonthYear, orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ComplimentaryDialog } from './ComplimentaryDialog'
import { EARN, copyText } from './EarnPage'
import { cancelReason, day, memberName, money, nextLine, platformName, priceA, priceLine, reasonLabel, sourceLine, statusLabel, statusTone, tierLine } from './format'
import { mm } from './members.messages'
import { me } from './messages'

type Member = Schemas['EarnMember']
type Item = Schemas['EarnTimelineItem']
type Seat = Schemas['EarnTeamSeat']

/** The paid member panel (EarnMemberPanel): a sheet on the right with the summary, Stripe's ids, the timeline and the actions. */
export function MemberPanel({ id, onClose }: { id: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/members/{id}`, { params: { path: { workspace: slug, id } } })
  const m = q.data
  return (
    <RDialog.Root open onOpenChange={(o) => !o && onClose()}>
      <RDialog.Portal>
        <RDialog.Overlay className="ea-sheet-ovl" />
        <RDialog.Content className="ea-sheet" aria-describedby={undefined}>
          {q.error ? (
            <div className="ea-sheet-body">
              <RDialog.Title className="vh">{t(mm.panelError)}</RDialog.Title>
              <ErrorPanel title={t(mm.panelError)} actions={<Button onClick={() => void q.refetch()}>{t(me.retry)}</Button>}>
                {errorMessage(q.error)}
              </ErrorPanel>
            </div>
          ) : !m ? (
            <div className="ea-sheet-body" aria-busy="true">
              <RDialog.Title className="vh">{t(me.loading)}</RDialog.Title>
              <Skeleton w="60%" h={28} />
              <Skeleton w="100%" h={120} />
              <Skeleton w="100%" h={240} />
            </div>
          ) : (
            <PanelBody m={m} onClose={onClose} />
          )}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

function PanelBody({ m, onClose }: { m: Member; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [dialog, setDialog] = useState<'cancel' | 'resend' | 'change' | 'remove' | 'give' | 'pause' | 'resume' | 'seat'>()
  const [seat, setSeat] = useState<Seat>()
  const [resendEmail, setResendEmail] = useState<string>()
  const name = memberName(t, m)
  const statusIcon: IconName = m.status === 'past_due' ? 'alert' : m.status === 'canceled' ? 'close' : m.status === 'trialing' ? 'clock' : 'check'
  const tone = { trialing: 'info', active: '', group: '', past_due: 'warn', canceled: 'mut', complimentary: 'pur', gift: 'pur' }[m.status] ?? ''
  const tierRow = m.kind === 'stripe' && m.period ? `${m.tierName} · ${m.period === 'year' ? t(me.yearly) : t(me.monthly)}` : m.tierName || m.tier
  const rows: [ReactNode, ReactNode][] = [
    [t(mm.tier), m.warning ? <Two key="t" a={tierRow} b={t(mm.higherOf, { count: m.warning.subscriptions.length })} /> : tierRow],
  ]
  if (!m.warning && m.kind !== 'gift')
    rows.push([
      t(mm.price),
      m.kind === 'complimentary'
        ? t(me.complimentaryNoPayment)
        : m.kind === 'group'
          ? t(mm.seatPrice, { team: m.team?.name ?? '' })
          : m.pausedAt
            ? t(mm.pricePaused, { price: priceLine(t, m, { percent: true, refund: false }) })
            : priceLine(t, m, { percent: true, refund: false }),
    ])
  rows.push([t(mm.status), <StatusLine key="s" m={m} />])
  if (m.team && m.team.role === 'buyer')
    rows.push([
      t(mm.team),
      t(mm.teamValue, {
        team: m.team.name,
        used: m.team.used - m.team.invites,
        seats: m.team.seats,
        invites: m.team.invites,
        free: Math.max(m.team.seats - m.team.used, 0),
      }),
    ])
  else if (m.team) rows.push([t(mm.team), t(mm.teamOfSeat, { team: m.team.name, buyer: m.team.buyerName ?? '' })])
  if (m.pausedAt) {
    const pausedBy = m.timeline.find((x) => x.kind === 'paused')?.actor
    const reminder = m.pausedUntil ? new Date(new Date(m.pausedUntil).getTime() - 7 * 864e5).toISOString() : ''
    rows.push([t(mm.pausedRow), t(mm.pausedRowValue, { date: day(m.pausedAt), by: pausedBy || 'none', reminder: reminder ? day(reminder) : 'none' })])
  }
  rows.push([m.kind === 'group' ? t(mm.seatSince) : t(mm.memberSince), dayMonthYear(m.since)])
  if (m.kind === 'group') {
    rows.push([t(mm.endsLabel), t(mm.seatEnds)])
  } else if (m.pausedAt) {
    // The pause's row says when it ends.
  } else if (m.warning) {
    // The higher tier's row stands for the person; its renewal is in the warning.
  } else if (m.next === 'renews' && m.nextAt) rows.push([t(mm.renewsLabel), m.renewalReminder ? t(mm.renewsWithReminder, { date: day(m.nextAt), reminder: day(m.renewalReminder) }) : day(m.nextAt)])
  else if (m.next === 'ends' && m.nextAt) rows.push([t(mm.endsLabel), day(m.nextAt)])
  else if (m.next === 'trial_ends' && m.nextAt) rows.push([t(mm.trialEndsLabel), day(m.nextAt)])
  else if (m.next === 'retries') rows.push([t(mm.retryLabel), m.nextAt ? t(mm.retryLine, { date: day(m.nextAt) }) : t(me.retriesSoon)])
  else if (m.next === 'no_end') rows.push([t(mm.endsLabel), t(me.noEnd)])
  if (m.discount)
    rows.push([
      t(mm.discount),
      m.discount.endsAt && m.discount.percent
        ? t(mm.discountLine, { offer: m.discount.label, percent: m.discount.percent, date: day(m.discount.endsAt) })
        : m.discount.percent
          ? t(mm.discountLineNoDate, { offer: m.discount.label, percent: m.discount.percent })
          : m.discount.label,
    ])
  const ltv = money(m.ltvCents, m.currency || 'usd', 'auto')
  rows.push([
    t(mm.ltv),
    m.refundedCents ? (
      <Two key="l" a={ltv} b={t(mm.ltvPaidRefunded, { paid: money(m.paidCents, m.currency, 'auto'), refunded: money(m.refundedCents, m.currency, 'auto') })} />
    ) : m.warning ? (
      <Two
        key="l"
        a={ltv}
        b={m.warning.subscriptions.map((x) => t(mm.ltvFor, { amount: money(x.ltvCents || x.amountCents, x.currency, 'auto'), tier: tierLine(t, { ...x, periodShown: true }) })).join(', ')}
      />
    ) : (
      ltv
    ),
  ])
  rows.push([t(mm.source), <SourceValue key="src" m={m} />])
  if (m.kind === 'complimentary' && m.grantNote) rows.push([t(mm.grantNote), m.grantNote])
  if (m.kind === 'stripe' && !m.warning && m.status !== 'canceled')
    rows.push([t(mm.billingEmail), m.billingDiffers ? <Two key="b" a={m.billingEmail} b={t(mm.billingDiffersLine)} /> : t(mm.sameAsContact)])
  if (m.cancelReason) rows.push([t(mm.source), cancelReason(t, m.cancelReason)])
  return (
    <>
      <div className="ea-sheet-head">
        <Avatar name={name} size={40} />
        <div className="flex min-w-0 flex-1 flex-col">
          <RDialog.Title className="m-0 text-[18px] font-bold">{name}</RDialog.Title>
          <span className="k">
            {m.hidden ? t(mm.hiddenNote) : m.erased ? t(mm.erasedNote) : m.email} · {t(mm.paidMember)}
          </span>
        </div>
        <div className="flex gap-2">
          {m.hidden || m.erased ? null : (
            <ButtonLink to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: m.contactId }} size="sm">
              {t(mm.openContact)}
            </ButtonLink>
          )}
          <Button size="sm" iconOnly icon="link" aria-label={t(mm.copyMemberLink)} onClick={() => void copyText(m.link, t, toast)} />
          <RDialog.Close className="btn sm ic" aria-label={t(mm.closePanel)}>
            <Icon name="close" size={14} />
          </RDialog.Close>
        </div>
      </div>
      <div className="ea-sheet-body">
        <div className={`ea-status ${tone}`}>
          <span className="ic" aria-hidden="true">
            <Icon name={statusIcon} size={14} />
          </span>
          <span>
            <StatusSentence m={m} />
          </span>
        </div>
        {m.test ? <Note tone="info">{t(mm.testText)}</Note> : null}
        {m.warning ? (
          <Note tone="warn">
            <span className="font-semibold">{t(mm.syncWarnTitle)}</span>
            <div className="flex flex-col gap-2">
              {t(mm.syncWarnText, { count: m.warning.subscriptions.length })}
              {m.warning.subscriptions.map((x) => (
                <div key={x.id} className="flex flex-wrap items-center justify-between gap-2">
                  <span>
                    <b>{tierLine(t, { ...x, periodShown: true })}</b> · {priceLine(t, x)} · {t(mm.sinceDate, { date: dayMonthYear(x.since) })}
                  </span>
                  <ButtonLink size="sm" to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ member: x.id }}>
                    {t(mm.open)}
                  </ButtonLink>
                </div>
              ))}
            </div>
          </Note>
        ) : null}
        <section aria-labelledby="ea-sum">
          <h3 className="ea-eyebrow mb-2" id="ea-sum">
            {t(mm.summaryTitle)}
          </h3>
          <dl className="ea-kv">
            {rows.map(([k, v], i) => (
              <div key={i} className="contents">
                <dt>{k}</dt>
                <dd>{v}</dd>
              </div>
            ))}
          </dl>
        </section>
        {m.seats.length && m.team ? <SeatsSection m={m} onRemove={(s) => (setSeat(s), setDialog('seat'))} /> : null}
        {m.kind === 'group' && m.team ? (
          <Note tone="info">{t(mm.seatHolderNote, { name: (m.name || m.email).split(' ')[0] ?? '', team: m.team.name })}</Note>
        ) : null}
        {m.customerId || m.subscriptionId ? (
          <section aria-labelledby="ea-stripe" className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <h3 className="ea-eyebrow" id="ea-stripe">
                {t(mm.stripeTitle)}
              </h3>
              {m.subscriptionUrl || m.customerUrl ? (
                <ButtonAnchor href={m.subscriptionUrl || m.customerUrl || ''} size="sm" iconRight="external">
                  {t(me.openInStripe)}
                </ButtonAnchor>
              ) : null}
            </div>
            <dl className="ea-kv">
              {m.customerId ? (
                <>
                  <dt>{t(mm.customer)}</dt>
                  <dd className="ea-mono">{m.customerId}</dd>
                </>
              ) : null}
              {m.warning ? (
                <>
                  <dt>{t(mm.subscription)}</dt>
                  <dd>{t(mm.twoActive, { count: m.warning.subscriptions.length })}</dd>
                </>
              ) : m.subscriptionId ? (
                <>
                  <dt>{t(mm.subscription)}</dt>
                  <dd>
                    <span className="ea-mono">{m.subscriptionId}</span>
                    {m.pausedUntil ? <span className="k"> · {t(mm.stripePaused, { date: day(m.pausedUntil) })}</span> : null}
                    {m.quantity > 1 ? <span className="k"> · {t(mm.stripeQuantity, { count: m.quantity })}</span> : null}
                  </dd>
                </>
              ) : null}
            </dl>
            <span className="k">{t(mm.stripeNote)}</span>
          </section>
        ) : null}
        <section aria-labelledby="ea-tl">
          <h3 className="ea-eyebrow" id="ea-tl">
            {t(mm.timeline)}
          </h3>
          {m.timeline.length ? (
            <ul className="ea-tl">
              {m.timeline.map((it) => (
                <TimelineRow key={it.id} it={it} m={m} />
              ))}
            </ul>
          ) : (
            <p className="k">{t(mm.timelineEmpty)}</p>
          )}
        </section>
      </div>
      {m.test ? null : m.canManage ? (
        <div className="ea-sheet-foot">
          {m.kind === 'complimentary' ? (
            <>
              <Button size="sm" onClick={() => setDialog('change')}>
                {t(mm.changeComp)}
              </Button>
              <Button size="sm" variant="danger" onClick={() => setDialog('remove')}>
                {t(mm.removeComp)}
              </Button>
            </>
          ) : m.kind === 'group' && m.team ? (
            <>
              <ButtonLink size="sm" to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ member: m.team.buyerId }}>
                {t(mm.openTeam, { team: m.team.name })}
              </ButtonLink>
              {m.team.seatId && m.status !== 'canceled' ? (
                <Button
                  size="sm"
                  variant="danger"
                  onClick={() => {
                    setSeat({ id: m.team?.seatId ?? '', email: m.email, name: m.name, status: 'active', invitedAt: m.since })
                    setDialog('seat')
                  }}
                >
                  {t(mm.removeSeatButton)}
                </Button>
              ) : null}
            </>
          ) : (
            <>
              <Button size="sm" onClick={() => setDialog('give')}>
                {t(mm.giveComp)}
              </Button>
              {m.canResume ? (
                <Button size="sm" variant="primary" onClick={() => setDialog('resume')}>
                  {t(mm.resumeNow)}
                </Button>
              ) : m.canPause ? (
                <Button size="sm" onClick={() => setDialog('pause')}>
                  {t(mm.pausePayments)}
                </Button>
              ) : null}
              {m.kind === 'stripe' && m.status !== 'canceled' ? (
                <Button size="sm" variant="danger" onClick={() => setDialog('cancel')}>
                  {t(mm.cancelSub)}
                </Button>
              ) : null}
            </>
          )}
          {m.emails.length === 0 ? null : m.status === 'past_due' && m.emails[0] ? (
            <Button
              size="sm"
              onClick={() => {
                setResendEmail(m.emails[0])
                setDialog('resend')
              }}
            >
              {t(mm.resendOne, { email: memberEmailName(t, m.emails[0], m.tier) })}
            </Button>
          ) : (
            <Menu
              align="end"
              label={t(mm.resendMenu)}
              trigger={
                <Button size="sm" iconRight="chevronDown">
                  {t(mm.resendMenu)}
                </Button>
              }
            >
              {m.emails.map((e) => (
                <MenuItem
                  key={e}
                  onSelect={() => {
                    setResendEmail(e)
                    setDialog('resend')
                  }}
                >
                  {memberEmailName(t, e, m.tier)}
                </MenuItem>
              ))}
            </Menu>
          )}
        </div>
      ) : (
        <div className="ea-sheet-foot">
          <span className="k">
            <T msg={m.helpers.length ? mm.manageNeeds : mm.manageNeedsNoHelpers} values={{ helpers: orList(m.helpers) }} />
          </span>
        </div>
      )}
      {dialog === 'cancel' ? <CancelDialog m={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'resend' ? <ResendDialog m={m} email={resendEmail} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'change' ? <ComplimentaryDialog member={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'give' ? <ComplimentaryDialog person={{ id: m.contactId, name: m.name, email: m.email, lists: m.lists }} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'remove' ? <RemoveDialog m={m} onClose={() => setDialog(undefined)} onRemoved={onClose} /> : null}
      {dialog === 'pause' ? <PauseDialog m={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'resume' ? <ResumeDialog m={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'seat' && seat ? <RemoveSeatDialog m={m} seat={seat} onClose={() => setDialog(undefined)} /> : null}
    </>
  )
}

export function StatusLine({ m }: { m: Member }) {
  const t = useT()
  const ended = m.status === 'canceled' ? m.timeline.find((x) => x.kind === 'access_ended') : undefined
  return (
    <span className="inline-flex flex-col items-start gap-0.5">
      <span className="inline-flex flex-wrap items-center gap-1.5">
        {m.pausedAt && m.status !== 'canceled' ? (
          <Badge tone="info">{m.pausedUntil ? t(me.pausedUntil, { date: day(m.pausedUntil) }) : t(me.pausedNoDate)}</Badge>
        ) : (
          <Badge tone={statusTone(m.status)}>{statusLabel(t, m.status)}</Badge>
        )}
        {m.team && m.status !== 'canceled' ? (
          <span className="chip ea-teamchip">
            {m.team.role === 'buyer' ? t(me.teamChip, { used: m.team.used - m.team.invites, seats: m.team.seats }) : t(me.seatChip, { team: m.team.name })}
          </span>
        ) : null}
        {m.cancelAt && m.status !== 'canceled' ? <span className="badge sm mut">{t(me.cancelsOn, { date: dayMonthDate(m.cancelAt) })}</span> : null}
        {m.accessPaused ? <span className="badge sm warn">{t(me.accessPaused)}</span> : null}
      </span>
      {ended || (m.status === 'canceled' && m.nextAt) ? <span className="k">{t(mm.accessEndedOn, { date: dayMonthDate(ended?.at ?? m.nextAt ?? '') })}</span> : null}
    </span>
  )
}

/** A value with a second, quieter line under it. */
function Two({ a, b }: { a: ReactNode; b: ReactNode }) {
  return (
    <span className="flex flex-col">
      <span>{a}</span>
      <span className="k">{b}</span>
    </span>
  )
}

/** The sentence at the top of the panel: the status with the plan, or what happened. */
function StatusSentence({ m }: { m: Member }) {
  const t = useT()
  const first = (m.name || m.email).split(' ')[0] ?? ''
  if (m.status === 'past_due' && m.failedAt) {
    return (
      <>
        {t(mm.stPastDue, {
          date: dayMonthDate(m.failedAt),
          reason: m.failureText ? ` (${m.failureText})` : '',
          next: m.nextAt ? t(mm.stNextTry, { date: dayMonthDate(m.nextAt) }) : '',
          name: first,
          tier: m.tier,
        })}
      </>
    )
  }
  if (m.status === 'canceled' && m.refundedCents > 0) {
    const refunded = m.timeline.find((x) => x.kind === 'refunded')
    return (
      <>
        {t(mm.stRefunded, { amount: money(m.refundedCents, m.currency, 'always'), date: refunded ? dayMonthDate(refunded.at) : '' })}
        {m.canceledBy ? ` ${t(mm.stCanceledBy, { name: m.canceledBy })}` : ''}
      </>
    )
  }
  const next = nextLine(t, m)
  if (m.pausedAt && m.status !== 'canceled') {
    return (
      <>
        <b>{m.pausedUntil ? t(me.pausedUntil, { date: day(m.pausedUntil) }) : t(me.pausedNoDate)}</b> ·{' '}
        {t(mm.stPausedRest, { plan: tierLine(t, { ...m, periodShown: true }), price: priceLine(t, m, { words: true }) })}
      </>
    )
  }
  if (m.kind === 'group' && m.team) {
    return (
      <>
        <b>{statusLabel(t, m.status)}</b> · {t(mm.stSeat, { team: m.team.name, tier: m.tier, buyer: m.team.buyerName ?? '' })}
      </>
    )
  }
  if (m.team?.role === 'buyer' && m.period) {
    return (
      <>
        <b>{statusLabel(t, m.status)}</b> ·{' '}
        {t(mm.stBuyer, {
          plan: tierLine(t, { ...m, team: undefined, periodShown: true }),
          team: m.team.name,
          seats: m.quantity,
          unit: priceA(t, m.amountCents, m.currency, m.period),
          next: lowerFirstInSentence(next),
        })}
      </>
    )
  }
  return (
    <>
      <b>{statusLabel(t, m.status)}</b> · {tierLine(t, m)}
      {m.kind === 'stripe' ? ` · ${priceLine(t, m, { words: true })}` : ''} · {lowerFirstInSentence(next)}
    </>
  )
}

/** Where the paid member came from: the email (a link to its report) and the click, or the import. */
export function SourceValue({ m }: { m: Member }) {
  const t = useT()
  const { slug } = useWorkspace()
  const label = sourceLine(t, m)
  if (m.source.kind === 'import' && m.importedAt) return <>{t(mm.importedOn, { label, date: dayMonthDate(m.importedAt) })}</>
  return (
    <span>
      {m.source.kind === 'campaign' && m.source.campaignId ? (
        <Link to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: m.source.campaignId }}>
          {label}
        </Link>
      ) : (
        label
      )}
      {m.clickedAt ? <span className="k"> · {t(mm.clickedAt, { when: `${dayMonthDate(m.clickedAt)}, ${clock(m.clickedAt)}` })}</span> : null}
    </span>
  )
}

export function memberEmailName(t: ReturnType<typeof useT>, email: string, tier: string): string {
  switch (email) {
    case 'welcome':
      return t(mm.emailWelcome, { tier })
    case 'trial_ending':
      return t(mm.emailTrialEnding)
    case 'payment_failed':
      return t(mm.emailPaymentFailed)
    case 'renewal_reminder':
      return t(mm.emailRenewal)
    case 'canceled':
      return t(mm.emailCanceled)
    case 'gift_purchased':
      return t(mm.emailGiftPurchased)
    case 'gift_payment_failed':
      return t(mm.emailGiftPaymentFailed)
    case 'gift_received':
      return t(mm.emailGiftReceived)
    case 'gift_ending':
      return t(mm.emailGiftEnding)
    case 'group_welcome':
      return t(mm.emailGroupWelcome)
    case 'group_invite':
      return t(mm.emailGroupInvite)
    case 'group_access_ended':
      return t(mm.emailGroupEnded)
    case 'paused':
      return t(mm.emailPaused)
    case 'resuming_soon':
      return t(mm.emailResuming)
  }
  return email
}

export function TimelineRow({ it, m }: { it: Item; m: Member }) {
  const t = useT()
  const d = it.data as Record<string, unknown>
  const s = (k: string) => (typeof d[k] === 'string' ? (d[k] as string) : '')
  const amount = it.amountCents === undefined ? '' : money(Math.abs(it.amountCents), it.currency || m.currency || 'usd', 'always')
  // "Insider yearly": the tier and the period it was paid for.
  const planOf = (period: string) => (period ? `${m.tier} ${period === 'year' ? t(me.yearly) : t(me.monthly)}` : m.tier)
  let icon: IconName = 'check'
  let tone = ''
  let title: ReactNode = it.kind
  let detail: ReactNode = null
  switch (it.kind) {
    case 'clicked_upgrade':
      icon = 'arrowRight'
      tone = 'info'
      title = t(mm.tlClicked, { campaign: s('campaign') })
      detail = t(mm.tlClickedDetail, { campaign: s('campaign').replace(/^.*(#\d+).*$/, '$1'), days: 7 })
      break
    case 'checkout_completed':
      tone = 'ok'
      title = t(mm.tlCheckout)
      detail = [
        planOf(s('period')),
        s('offer') ? t(mm.tlOfferCode, { offer: s('offer') }) : '',
        s('lists') === 'already' && m.lists.length ? t(mm.tlAlreadyOn, { lists: orList(m.lists) }) : s('lists') === 'joined' && m.lists.length ? t(mm.tlJoined, { lists: orList(m.lists) }) : '',
      ]
        .filter(Boolean)
        .join(' · ')
      break
    case 'paid':
    case 'renewed':
      tone = 'ok'
      title = it.kind === 'paid' ? t(mm.tlPaid, { amount }) : t(mm.tlRenewed, { amount })
      // Checkout payments are before tax; renewals and Stripe's history say so in Stripe.
      detail = [
        it.stripeEvent,
        d.history ? t(mm.tlFromHistory) : s('offer') ? t(mm.tlPlanWith, { plan: planOf(s('period')), offer: s('offer') }) : planOf(s('period')),
        m.taxAdded && it.kind === 'paid' && !d.history ? t(mm.tlBeforeTax) : '',
      ]
        .filter(Boolean)
        .join(' · ')
      break
    case 'refunded':
      icon = 'undo'
      tone = 'mut'
      title = t(mm.tlRefunded, { amount: `−${amount}` })
      detail = [it.stripeEvent, t(mm.tlMadeInStripe)].filter(Boolean).join(' · ')
      break
    case 'payment_failed':
      icon = 'alert'
      tone = 'warn'
      title = t(mm.tlFailed, { amount })
      detail = [it.stripeEvent, [s('reason') ? `“${s('reason')}”` : '', s('retry') ? t(mm.tlRetryOn, { date: day(s('retry')) }) : ''].filter(Boolean).join(' ')].filter(Boolean).join(' · ')
      break
    case 'canceled_by_team':
      icon = 'close'
      tone = 'mut'
      title = s('when') === 'now' ? t(mm.tlCanceledNow, { name: it.actor ?? '' }) : t(mm.tlCanceledAtEnd, { name: it.actor ?? '' })
      detail = [s('note') ? `“${s('note')}”` : '', t(mm.tlAudited)].filter(Boolean).join(' · ')
      break
    case 'trial_started':
      icon = 'clock'
      tone = 'info'
      title = t(mm.tlTrialStarted)
      detail = s('trialEnd') ? t(mm.tlTrialStartedDetail, { date: day(s('trialEnd')) }) : null
      break
    case 'subscribed':
      title = t(mm.tlSubscribed)
      break
    case 'cancel_scheduled':
      icon = 'calendar'
      tone = 'mut'
      title = s('date') ? t(mm.tlCancelScheduled, { date: day(s('date')) }) : t(mm.tlCancelScheduled, { date: '' })
      detail = s('reason') ? cancelReason(t, s('reason')) : null
      break
    case 'cancel_undone':
      title = t(mm.tlCancelUndone)
      break
    case 'access_ended':
      icon = 'close'
      tone = 'mut'
      title = t(mm.tlEnded)
      detail =
        s('reason') === 'group_ended'
          ? t(mm.tlGroupEnded)
          : s('reason') === 'seat_removed'
            ? t(mm.tlSeatRemoved)
            : it.stripeEvent || (s('by') ? t(mm.tlBy, { name: s('by') }) : null)
      break
    // Pausing and teams (M65).
    case 'paused':
      icon = 'clock'
      tone = 'info'
      title = t(mm.tlPaused, { date: day(s('until')) })
      detail =
        s('source') === 'team'
          ? [it.actor ? t(mm.tlBy, { name: it.actor }) : '', s('note') ? `“${s('note')}”` : '', t(mm.tlAudited)].filter(Boolean).join(' · ')
          : t(mm.tlPausedFrom, { source: s('source') || 'other' })
      break
    case 'resumed':
      icon = 'refresh'
      tone = 'ok'
      title = it.actor ? t(mm.tlResumedBy, { name: it.actor }) : t(mm.tlResumed)
      detail = [t(mm.tlResumedDetail, { source: s('source') || 'other' }), it.actor ? t(mm.tlAudited) : ''].filter(Boolean).join(' · ')
      break
    case 'team_invited':
      icon = 'mail'
      tone = 'info'
      title = t(mm.tlInvited, { name: s('name') || s('email') })
      detail = s('until') ? t(mm.tlInvitedDetail, { date: day(s('until')) }) : null
      break
    case 'team_joined':
      tone = 'ok'
      title = t(mm.tlJoined, { team: s('team') })
      detail = t(mm.tlJoinedDetail)
      break
    case 'team_accepted':
      tone = 'ok'
      title = t(mm.tlAccepted, { name: s('name') || s('email') })
      detail = typeof d.used === 'number' ? t(mm.tlAcceptedDetail, { used: d.used as number, seats: (d.seats as number) ?? 0 }) : t(mm.tlJoinedDetail)
      break
    case 'team_removed':
      icon = 'close'
      tone = 'mut'
      title = t(mm.tlTeamRemoved, { email: s('email'), team: s('team') })
      break
    case 'seats_changed':
      icon = 'refresh'
      tone = 'info'
      title = t(mm.tlSeats, { from: (d.from as number) ?? 0, to: (d.to as number) ?? 0 })
      break
    case 'plan_changed':
      icon = 'refresh'
      tone = 'info'
      title = t(mm.tlPlanChanged)
      detail = s('fromTier') ? t(mm.tlPlanChangedDetail, { tier: s('fromTier') }) : null
      break
    case 'offer_applied':
      icon = 'tag'
      tone = 'info'
      title = t(mm.tlOffer, { offer: s('offer') })
      break
    case 'granted':
      title = t(mm.tlGranted)
      detail = [reasonLabel(t, s('reason')), it.actor ? t(mm.tlBy, { name: it.actor }) : ''].filter(Boolean).join(' · ')
      break
    case 'grant_changed':
      icon = 'edit'
      title = t(mm.tlGrantChanged)
      detail = it.actor ? t(mm.tlBy, { name: it.actor }) : null
      break
    case 'grant_removed':
      icon = 'close'
      tone = 'mut'
      title = t(mm.tlGrantRemoved)
      detail = it.actor ? t(mm.tlBy, { name: it.actor }) : null
      break
    case 'imported':
      icon = 'upload'
      tone = 'mut'
      title = t(mm.tlImported, { platform: platformName(s('platform')) })
      detail = [s('matchedBy') ? t(mm.tlImportedDetail, { by: s('matchedBy') }) : '', s('platform') ? t(mm.tlImportConsent, { platform: platformName(s('platform')) }) : ''].filter(Boolean).join(' · ')
      break
    case 'gift_redeemed':
      icon = 'star'
      tone = 'info'
      title = t(mm.tlGift)
      detail = s('from') ? t(mm.tlGiftDetail, { from: s('from') }) : null
      break
    case 'email_sent':
    case 'email_not_sent': {
      const sent = it.kind === 'email_sent'
      icon = sent ? 'check' : 'alert'
      tone = sent ? 'ok' : 'warn'
      const nm = memberEmailName(t, s('email'), m.tier)
      title = sent ? t(mm.tlEmailSent, { email: nm }) : t(mm.tlEmailNotSent, { email: nm })
      detail = sent ? (
        <>
          {t(mm.tlEmailDetail, { template: s('template'), version: String(d.version ?? '') })}
          {it.messageStatus === 'delivered' ? (
            <>
              {' · '}
              <b className="text-ok-fg">{t(mm.delivered)}</b> {it.provider ? t(mm.via, { provider: it.provider }) : ''}
            </>
          ) : null}
        </>
      ) : (
        t(mm.tlEmailNotSentDetail, { reason: s('reason') })
      )
      break
    }
    case 'billing_email_changed':
      icon = 'mail'
      tone = 'mut'
      title = t(mm.tlBilling)
      break
    case 'dispute':
      icon = 'alert'
      tone = 'bad'
      title = t(mm.tlDispute)
      break
  }
  return (
    <li>
      <span className={`ic ${tone}`} aria-hidden="true">
        <Icon name={icon} size={14} />
      </span>
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="t">
          {title}
          {it.test ? (
            <Badge tone="purple" className="ml-2">
              {t(mm.test)}
            </Badge>
          ) : null}
        </span>
        {detail ? <span className="d">{detail}</span> : null}
      </div>
      <span className="w">
        {dayMonthDate(it.at)} {clock(it.at)}
      </span>
    </li>
  )
}

export function CancelDialog({ m, onClose }: { m: Member; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const emails = $api.useQuery('get', `${EARN}/settings/member-emails`, { params: { path: { workspace: slug } } })
  const [when, setWhen] = useState<'period_end' | 'now'>(m.status === 'past_due' ? 'now' : 'period_end')
  const [email, setEmail] = useState(true)
  const [note, setNote] = useState('')
  const cancel = $api.useMutation('post', `${EARN}/members/{id}/cancel`)
  const name = m.name || m.email
  const first = name.split(' ')[0] ?? name
  const canceled = emails.data?.emails.find((e) => e.email === 'canceled')
  const live = canceled?.status === 'live'
  const end = m.nextAt && (m.next === 'renews' || m.next === 'trial_ends' || m.next === 'retries') ? dayMonthYear(m.nextAt) : ''
  return (
    <Dialog
      open
      hideClose
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mm.cancelEyebrow)}</span>}
      title={t(mm.cancelTitle, { name, tier: m.tier })}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.cancelKeep)}</Button>
          <Button
            variant="danger-primary"
            loading={cancel.isPending}
            onClick={() =>
              cancel.mutate(
                { params: { path: { workspace: slug, id: m.id } }, body: { when, email: email && live, note } },
                {
                  onSuccess: (r) => {
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
                    toast({ kind: 'success', title: when === 'now' ? t(mm.canceled) : t(mm.canceledAtEnd, { date: day(r.cancelAt ?? r.nextAt) }) })
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {when === 'now' ? t(mm.cancelSubmitNow) : t(mm.cancelSubmit)}
          </Button>
        </>
      }
    >
      <span className="k">{t(mm.cancelPays, { name: first, price: priceLine(t, m) })}</span>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="lbl mb-1">{t(mm.cancelWhen)}</legend>
        <label className={`ea-radio${when === 'period_end' ? ' on' : ''}`}>
          <input type="radio" name="cancel-when" checked={when === 'period_end'} onChange={() => setWhen('period_end')} />
          <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
            {end ? t(mm.cancelAtEndOn, { date: end }) : t(mm.cancelAtEnd)}
            <span className="k">{t(mm.cancelAtEndHint, { name: first, tier: m.tier })}</span>
          </span>
        </label>
        <label className={`ea-radio${when === 'now' ? ' on' : ''}`}>
          <input type="radio" name="cancel-when" checked={when === 'now'} onChange={() => setWhen('now')} />
          <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
            {t(mm.cancelNow)}
            <span className="k">{t(mm.cancelNowHint)}</span>
          </span>
        </label>
      </fieldset>
      <label className="flex items-start gap-2.5">
        <input type="checkbox" className="mt-0.5" checked={email && live} disabled={!live} onChange={(e) => setEmail(e.target.checked)} />
        <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
          {t(mm.cancelEmail, { name: first })}
          <span className="k font-medium">
            {canceled
              ? live
                ? t(mm.cancelEmailTemplate, { name: memberEmailName(t, 'canceled', m.tier), template: canceled.template, version: canceled.liveVersion })
                : t(mm.cancelEmailDraft, { name: memberEmailName(t, 'canceled', m.tier) })
              : memberEmailName(t, 'canceled', m.tier)}
          </span>
        </span>
      </label>
      <label className="lbl">
        {t(mm.cancelNote)}
        <textarea className="in ea-ta" style={{ height: 64 }} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
        <span className="k font-medium">{t(mm.cancelNoteHint)}</span>
      </label>
    </Dialog>
  )
}

export function ResendDialog({ m, email: preset, onClose }: { m: Member; email?: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [email, setEmail] = useState(preset ?? m.emails[0] ?? 'welcome')
  const resend = $api.useMutation('post', `${EARN}/members/{id}/resend`)
  return (
    <Dialog
      open
      size="sm"
      hideClose
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mm.resendEyebrow)}</span>}
      title={t(mm.resendTitle, { name: m.name || m.email, email: memberEmailName(t, email, m.tier) })}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.cancel)}</Button>
          <Button
            variant="primary"
            loading={resend.isPending}
            onClick={() =>
              resend.mutate(
                { params: { path: { workspace: slug, id: m.id } }, body: { email: email as Schemas['EarnResendInput']['email'] } },
                {
                  onSuccess: (r) => {
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
                    const nm = memberEmailName(t, email, m.tier)
                    toast(r.status === 'sent' ? { kind: 'success', title: t(mm.resent, { email: nm }) } : { kind: 'error', title: t(mm.resendNotSent, { email: nm }), detail: t(mm.tlEmailNotSentDetail, { reason: r.reason ?? '' }) })
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(mm.resendSubmit)}
          </Button>
        </>
      }
    >
      {preset ? (
        <span className="k">{t(mm.resendText, { email: m.email })}</span>
      ) : (
        <label className="lbl">
          {t(mm.resendWhich)}
          <Select value={email} onChange={(e) => setEmail(e.target.value)}>
            {m.emails.map((e) => (
              <option key={e} value={e}>
                {memberEmailName(t, e, m.tier)}
              </option>
            ))}
          </Select>
        </label>
      )}
    </Dialog>
  )
}

function RemoveDialog({ m, onClose, onRemoved }: { m: Member; onClose: () => void; onRemoved: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const remove = $api.useMutation('delete', `${EARN}/members/{id}/complimentary`)
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={t(mm.removeComp)}
      title={t(mm.compRemoveTitle)}
      description={t(mm.compRemoveText, { name: m.name || m.email, tier: m.tier })}
      confirmLabel={t(mm.removeComp)}
      onConfirm={async () => {
        await remove.mutateAsync({ params: { path: { workspace: slug, id: m.id } } })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
        toast({ kind: 'deleted', title: t(mm.compRemoved) })
        onClose()
        onRemoved()
      }}
    />
  )
}

/** A team buyer's seats (EarnMemberPanelStates): the buyer first, then each person with Remove…. */
function SeatsSection({ m, onRemove }: { m: Member; onRemove: (s: Seat) => void }) {
  const t = useT()
  const team = m.team
  if (!team) return null
  return (
    <section aria-labelledby="ea-seats" className="flex flex-col gap-2">
      <h3 className="ea-eyebrow" id="ea-seats">
        {t(mm.seatsTitle, { used: team.used, seats: team.seats })}
      </h3>
      <ul className="ea-seats">
        {m.seats.map((s) => (
          <li key={s.id}>
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="font-semibold">{s.name || s.email}</span>
              {s.name ? <span className="k break-anywhere">{s.email}</span> : null}
            </span>
            <span className="flex flex-wrap items-center gap-2">
              <Badge tone={s.status === 'invited' ? 'info' : 'ok'}>{s.status === 'invited' ? t(me.statusInvited) : t(me.statusActive)}</Badge>
              {s.status === 'buyer' ? <span className="k">{t(mm.seatBuyer)}</span> : null}
              {s.status === 'invited' && s.inviteExpiresAt ? <span className="k">{t(mm.seatUntil, { date: day(s.inviteExpiresAt) })}</span> : null}
              {s.status !== 'buyer' && m.canManage ? (
                <Button size="sm" aria-label={t(mm.seatRemoveLabel, { name: s.name || s.email })} onClick={() => onRemove(s)}>
                  {t(mm.seatRemove)}
                </Button>
              ) : null}
            </span>
          </li>
        ))}
      </ul>
      <span className="k">{t(mm.seatsNote, { name: (m.name || m.email).split(' ')[0] ?? '' })}</span>
    </section>
  )
}

/** Pause payments (EarnMemberPanelStates): 1, 3 or 6 months, or until a date at most 6 months away. */
function PauseDialog({ m, onClose }: { m: Member; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [choice, setChoice] = useState<'1' | '3' | '6' | 'date'>('1')
  const [until, setUntil] = useState('')
  const [email, setEmail] = useState(true)
  const [note, setNote] = useState('')
  const pause = $api.useMutation('post', `${EARN}/members/{id}/pause`)
  const first = (m.name || m.email).split(' ')[0] ?? ''
  const plan = tierLine(t, { ...m, periodShown: true })
  // Today, read once when the dialog opens.
  const [now] = useState(() => Date.now())
  const months = (n: number) => {
    const d = new Date(now)
    d.setMonth(d.getMonth() + n)
    return d
  }
  const latest = months(6)
  const iso = (d: Date) => d.toISOString().slice(0, 10)
  const tomorrow = new Date(now + 864e5)
  return (
    <Dialog
      open
      size="sm"
      hideClose
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mm.pauseEyebrow)}</span>}
      title={t(mm.pauseTitle, { name: m.name || m.email })}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.pauseCancel)}</Button>
          <Button
            variant="primary"
            loading={pause.isPending}
            disabled={choice === 'date' && !until}
            onClick={() =>
              pause.mutate(
                {
                  params: { path: { workspace: slug, id: m.id } },
                  body: {
                    ...(choice === 'date' ? { until } : { months: Number(choice) as 1 | 3 | 6 }),
                    ...(note.trim() ? { note: note.trim() } : {}),
                    ...(email ? {} : { skipEmail: true }),
                  },
                },
                {
                  onSuccess: (r) => {
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
                    toast({ kind: 'success', title: t(mm.pausedToast, { date: day(r.pausedUntil ?? '') }) })
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(mm.pauseConfirm)}
          </Button>
        </>
      }
    >
      <span className="k">{t(mm.pauseText, { plan, price: priceLine(t, m), first })}</span>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="lbl mb-1">{t(mm.pauseFor)}</legend>
        {([1, 3, 6] as const).map((n) => (
          <label key={n} className={`ea-radio${choice === String(n) ? ' on' : ''}`}>
            <input type="radio" name="pause-for" checked={choice === String(n)} onChange={() => setChoice(String(n) as '1' | '3' | '6')} />
            <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
              {t(mm.pauseMonths, { count: n })}
              <span className="k font-medium">{t(mm.pauseResumes, { date: day(months(n).toISOString()) })}</span>
            </span>
          </label>
        ))}
        <label className={`ea-radio${choice === 'date' ? ' on' : ''}`}>
          <input type="radio" name="pause-for" checked={choice === 'date'} onChange={() => setChoice('date')} />
          <span className="flex min-w-0 flex-col gap-1 font-semibold">
            {t(mm.pauseUntil)}
            <span className="k font-medium">{t(mm.pauseUntilHint, { date: day(latest.toISOString()) })}</span>
            {choice === 'date' ? (
              <input
                type="date"
                className="in w-[160px]"
                aria-label={t(mm.pauseDate)}
                min={iso(tomorrow)}
                max={iso(latest)}
                value={until}
                onChange={(e) => setUntil(e.target.value)}
              />
            ) : null}
          </span>
        </label>
      </fieldset>
      <label className="flex items-start gap-2.5">
        <input type="checkbox" className="mt-0.5" checked={email} onChange={(e) => setEmail(e.target.checked)} />
        <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
          {t(mm.pauseEmail, { first })}
          <span className="k font-medium">{t(mm.pauseEmailHint)}</span>
        </span>
      </label>
      <label className="lbl">
        {t(mm.pauseNote)}
        <textarea className="in ea-ta" style={{ height: 64 }} maxLength={300} value={note} onChange={(e) => setNote(e.target.value)} />
        <span className="k font-medium">{t(mm.pauseNoteHint)}</span>
      </label>
    </Dialog>
  )
}

/** Resume now: the pause ends today. */
function ResumeDialog({ m, onClose }: { m: Member; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const resume = $api.useMutation('post', `${EARN}/members/{id}/resume`)
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      badge={t(mm.resumeConfirm)}
      tone="ok"
      title={t(mm.resumeTitle, { name: m.name || m.email })}
      description={t(mm.resumeText, { price: priceLine(t, m, { words: true }) })}
      confirmLabel={t(mm.resumeConfirm)}
      onConfirm={async () => {
        await resume.mutateAsync({ params: { path: { workspace: slug, id: m.id } } })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
        toast({ kind: 'success', title: t(mm.resumedToast) })
        onClose()
      }}
    />
  )
}

/** Remove a person from a team (EarnMemberPanelStates): from the buyer's panel, or the seat holder's own. */
function RemoveSeatDialog({ m, seat, onClose }: { m: Member; seat: Seat; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [email, setEmail] = useState(true)
  const remove = $api.useMutation('delete', `${EARN}/members/{id}/seats/{seatId}`)
  const team = m.team
  if (!team) return null
  const buyerId = team.role === 'buyer' ? m.id : (team.buyerId ?? '')
  const name = seat.name || seat.email
  const first = name.split(' ')[0] ?? name
  const buyer = team.role === 'buyer' ? m.name || m.email : (team.buyerName ?? '')
  return (
    <Dialog
      open
      size="sm"
      hideClose
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mm.teamEyebrow, { team: team.name })}</span>}
      title={t(mm.removeSeatTitle, { name, team: team.name })}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.removeSeatKeep)}</Button>
          <Button
            variant="danger-primary"
            loading={remove.isPending}
            onClick={() =>
              remove.mutate(
                { params: { path: { workspace: slug, id: buyerId, seatId: seat.id }, query: email ? {} : { skipEmail: true } } },
                {
                  onSuccess: () => {
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members/{id}`] })
                    toast({ kind: 'deleted', title: t(mm.seatRemovedToast, { name, team: team.name }) })
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(mm.removeSeatConfirm)}
          </Button>
        </>
      }
    >
      <span className="k">
        {seat.status === 'invited'
          ? t(mm.removeInviteText, { name, team: team.name, seats: team.seats })
          : t(mm.removeSeatText, { name: first, tier: m.tier, team: team.name, seats: team.seats, buyer })}
      </span>
      {seat.status === 'invited' ? null : (
        <label className="flex items-start gap-2.5">
          <input type="checkbox" className="mt-0.5" checked={email} onChange={(e) => setEmail(e.target.checked)} />
          <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
            {t(mm.removeSeatEmail, { name: first })}
            <span className="k font-medium">{t(mm.removeSeatEmailHint)}</span>
          </span>
        </label>
      )}
    </Dialog>
  )
}
