import { useQueryClient } from '@tanstack/react-query'
import { Dialog as RDialog } from 'radix-ui'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Field, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ComplimentaryDialog } from './ComplimentaryDialog'
import { EARN, copyText } from './EarnPage'
import { cancelReason, day, money, nextLine, platformName, priceLine, reasonLabel, sourceLine, statusLabel, statusTone, tierLine } from './format'
import { mm } from './members.messages'
import { me } from './messages'

type Member = Schemas['EarnMember']
type Item = Schemas['EarnTimelineItem']

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
  const [dialog, setDialog] = useState<'cancel' | 'resend' | 'change' | 'remove'>()
  const name = m.hidden ? t(mm.hiddenName) : m.name || m.email
  const statusIcon: IconName = m.status === 'past_due' ? 'alert' : m.status === 'canceled' ? 'close' : m.status === 'trialing' ? 'clock' : 'check'
  const tone = { trialing: 'info', active: '', past_due: 'warn', canceled: 'mut', complimentary: 'pur', gift: 'pur' }[m.status] ?? ''
  const rows: [ReactNode, ReactNode][] = [
    [t(mm.tier), m.kind === 'stripe' && m.period ? `${m.tierName} · ${m.period === 'year' ? t(me.yearly) : t(me.monthly)}` : m.tierName || m.tier],
    [t(mm.price), priceLine(t, m)],
    [t(mm.status), <StatusLine key="s" m={m} />],
    [t(mm.memberSince), day(m.since)],
  ]
  if (m.next === 'renews' && m.nextAt) rows.push([t(mm.renewsLabel), m.renewalReminder ? t(mm.renewsWithReminder, { date: day(m.nextAt), reminder: day(m.renewalReminder) }) : day(m.nextAt)])
  else if (m.next === 'ends' && m.nextAt) rows.push([t(mm.endsLabel), day(m.nextAt)])
  else if (m.next === 'trial_ends' && m.nextAt) rows.push([t(mm.trialEndsLabel), day(m.nextAt)])
  else if (m.next === 'retries') rows.push([t(mm.retryLabel), m.nextAt ? day(m.nextAt) : t(me.retriesSoon)])
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
  rows.push([t(mm.ltv), m.refundedCents ? t(mm.ltvRefunded, { ltv: money(m.ltvCents, m.currency, 'auto'), refunded: money(m.refundedCents, m.currency, 'always') }) : money(m.ltvCents, m.currency || 'usd', 'auto')])
  rows.push([
    t(mm.source),
    <span key="src">
      {sourceLine(t, m)}
      {m.clickedAt ? <span className="k"> · {t(mm.clickedAt, { when: `${dayMonthDate(m.clickedAt)}, ${clock(m.clickedAt)}` })}</span> : null}
      {m.importedAt && m.matchedBy ? <span className="k"> · {t(mm.importedAt, { date: day(m.importedAt), by: m.matchedBy })}</span> : null}
    </span>,
  ])
  if (m.kind === 'complimentary' && m.grantNote) rows.push([t(mm.grantNote), m.grantNote])
  if (m.kind === 'stripe') rows.push([t(mm.billingEmail), m.billingDiffers ? m.billingEmail : t(mm.sameAsContact)])
  if (m.failureReason && m.failedAt) rows.push([t(mm.failure), t(mm.failureLine, { reason: m.failureReason, date: day(m.failedAt) })])
  if (m.canceledBy) rows.push([t(mm.canceledBy), m.canceledBy])
  if (m.cancelReason) rows.push([t(mm.source), cancelReason(t, m.cancelReason)])
  return (
    <>
      <div className="ea-sheet-head">
        <Avatar name={name} size={40} />
        <div className="flex min-w-0 flex-1 flex-col">
          <RDialog.Title className="m-0 text-[18px] font-bold">{name}</RDialog.Title>
          <span className="k">
            {m.hidden ? t(mm.hiddenNote) : m.email} · {t(mm.paidMember)}
          </span>
        </div>
        <div className="flex gap-2">
          {m.hidden ? null : (
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
            <b>{statusLabel(t, m.status)}</b> · {tierLine(t, m)} · {priceLine(t, m)} · {nextLine(t, m).replace(/^./, (c) => c.toLowerCase())}
          </span>
        </div>
        {m.test ? <Note tone="info">{t(mm.testText)}</Note> : null}
        {m.warning ? (
          <Note tone="warn" label={t(mm.syncWarnTitle)}>
            {t(mm.syncWarnText, { count: m.warning.subscriptions.length })}
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
              {m.subscriptionId ? (
                <>
                  <dt>{t(mm.subscription)}</dt>
                  <dd className="ea-mono">{m.subscriptionId}</dd>
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
      {m.canManage && !m.test && m.status !== 'canceled' ? (
        <div className="ea-sheet-foot">
          {m.kind === 'stripe' ? (
            <>
              {m.emails.length ? <Button onClick={() => setDialog('resend')}>{t(mm.resend)}</Button> : null}
              <Button variant="danger" onClick={() => setDialog('cancel')}>
                {t(mm.cancelSub)}
              </Button>
            </>
          ) : m.kind === 'complimentary' ? (
            <>
              <Button onClick={() => setDialog('change')}>{t(mm.changeComp)}</Button>
              <Button variant="danger" onClick={() => setDialog('remove')}>
                {t(mm.removeComp)}
              </Button>
            </>
          ) : null}
        </div>
      ) : null}
      {dialog === 'cancel' ? <CancelDialog m={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'resend' ? <ResendDialog m={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'change' ? <ComplimentaryDialog member={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'remove' ? <RemoveDialog m={m} onClose={() => setDialog(undefined)} onRemoved={onClose} /> : null}
    </>
  )
}

function StatusLine({ m }: { m: Member }) {
  const t = useT()
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <Badge tone={statusTone(m.status)}>{statusLabel(t, m.status)}</Badge>
      {m.cancelAt && m.status !== 'canceled' ? <span className="badge sm mut">{t(me.cancelsOn, { date: dayMonthDate(m.cancelAt) })}</span> : null}
      {m.accessPaused ? <span className="badge sm warn">{t(me.accessPaused)}</span> : null}
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
    case 'gift_received':
      return t(mm.emailGiftReceived)
    case 'gift_ending':
      return t(mm.emailGiftEnding)
  }
  return email
}

function TimelineRow({ it, m }: { it: Item; m: Member }) {
  const t = useT()
  const d = it.data as Record<string, unknown>
  const s = (k: string) => (typeof d[k] === 'string' ? (d[k] as string) : '')
  const amount = it.amountCents === undefined ? '' : money(Math.abs(it.amountCents), it.currency || m.currency || 'usd', 'always')
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
      title = t(mm.tlCheckout)
      detail = [s('period') ? `${m.tier} ${s('period') === 'year' ? t(me.yearly) : t(me.monthly)}` : m.tier, s('offer') ? `${s('offer')}` : ''].filter(Boolean).join(' · ')
      break
    case 'paid':
      title = t(mm.tlPaid, { amount })
      detail = [it.stripeEvent, s('offer') ? `${m.tier} · ${s('offer')}` : m.tier].filter(Boolean).join(' · ')
      break
    case 'renewed':
      title = t(mm.tlRenewed, { amount })
      detail = [it.stripeEvent, s('offer')].filter(Boolean).join(' · ')
      break
    case 'refunded':
      icon = 'undo'
      tone = 'mut'
      title = t(mm.tlRefunded, { amount })
      detail = it.stripeEvent
      break
    case 'payment_failed':
      icon = 'alert'
      tone = 'warn'
      title = t(mm.tlFailed, { amount })
      detail = [s('reason'), s('retry') ? t(me.retries, { date: day(s('retry')) }) : ''].filter(Boolean).join(' · ')
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
      detail = s('by') ? t(mm.tlBy, { name: s('by') }) : null
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
      detail = s('matchedBy') ? t(mm.tlImportedDetail, { by: s('matchedBy') }) : null
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
      icon = sent ? 'mail' : 'alert'
      tone = sent ? '' : 'warn'
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

function CancelDialog({ m, onClose }: { m: Member; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [when, setWhen] = useState<'period_end' | 'now'>(m.status === 'past_due' ? 'now' : 'period_end')
  const [email, setEmail] = useState(true)
  const [note, setNote] = useState('')
  const cancel = $api.useMutation('post', `${EARN}/members/{id}/cancel`)
  const name = m.name || m.email
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mm.cancelTitle, { name })}
      description={t(mm.cancelText)}
      footer={
        <>
          <Button onClick={onClose}>{t(mm.cancel)}</Button>
          <Button
            variant="danger"
            loading={cancel.isPending}
            onClick={() =>
              cancel.mutate(
                { params: { path: { workspace: slug, id: m.id } }, body: { when, email, note } },
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
            {t(mm.cancelSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[13px] font-semibold">{t(mm.cancelWhen)}</legend>
          <OptionCard name="when" value="period_end" checked={when === 'period_end'} onChange={() => setWhen('period_end')} title={t(mm.cancelAtEnd)} description={m.nextAt ? t(mm.cancelAtEndHint, { tier: m.tier, date: day(m.nextAt) }) : undefined} />
          <OptionCard name="when" value="now" checked={when === 'now'} onChange={() => setWhen('now')} title={t(mm.cancelNow)} description={t(mm.cancelNowHint)} />
        </fieldset>
        <Checkbox label={t(mm.cancelEmail)} checked={email} onChange={(e) => setEmail(e.target.checked)} />
        <Field label={t(mm.cancelNote)} optional>
          <Textarea rows={2} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>
    </Dialog>
  )
}

function ResendDialog({ m, onClose }: { m: Member; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [email, setEmail] = useState(m.emails[0] ?? 'welcome')
  const resend = $api.useMutation('post', `${EARN}/members/{id}/resend`)
  return (
    <Dialog
      open
      size="sm"
      onOpenChange={(o) => !o && onClose()}
      title={t(mm.resendTitle, { name: m.name || m.email })}
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
      <Field label={t(mm.resendWhich)}>
        <Select value={email} onChange={(e) => setEmail(e.target.value)}>
          {m.emails.map((e) => (
            <option key={e} value={e}>
              {memberEmailName(t, e, m.tier)}
            </option>
          ))}
        </Select>
      </Field>
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
