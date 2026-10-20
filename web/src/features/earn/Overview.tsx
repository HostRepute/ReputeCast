import { useQueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useWorkspace } from '@/lib/session'
import { ProductSalesCard } from '../products/ProductSalesCard'
import { EARN, EarnPage } from './EarnPage'
import { day, money, statusLabel, statusTone, tierLine } from './format'
import { me } from './messages'
import { mo } from './overview.messages'

type Checklist = Schemas['EarnChecklist']
type Step = Schemas['EarnChecklistStep']
type T = ReturnType<typeof useT>

/** The launch checklist's query, shared with the overview that replaces it once Earn is live. */
export function useEarnChecklist() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${EARN}/checklist`, { params: { path: { workspace: slug } } })
}

/**
 * Earn › Overview. Until the live key is connected and the six launch steps
 * are done it's the launch checklist (EarnStates: first run, test mode
 * only); then it's `launched`, the revenue overview.
 */
export function EarnOverviewPage({ launched }: { launched: ReactNode }) {
  const t = useT()
  const q = useEarnChecklist()
  return (
    <EarnPage tab="overview">
      {q.error ? (
        <ErrorPanel title={t(mo.errorLoad)} actions={<Button onClick={() => void q.refetch()}>{t(me.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !q.data ? (
        <div className="card pad" aria-busy="true" aria-label={t(me.loading)}>
          <Skeleton w="40%" h={22} />
          <Skeleton w="100%" h={240} className="mt-4" />
        </div>
      ) : q.data.launched ? (
        launched
      ) : (
        <LaunchChecklist c={q.data} />
      )}
      {q.data ? <ProductSalesCard /> : null}
    </EarnPage>
  )
}

function LaunchChecklist({ c }: { c: Checklist }) {
  const t = useT()
  const total = c.steps.length
  const current = c.steps.findIndex((s) => !s.done)
  const pct = total ? Math.round((c.done / total) * 100) : 0
  return (
    <div className="ea-launch">
      <section className="card ea-checklist" aria-label={t(mo.checklistLabel)}>
        <div className="ea-checklist-head">
          <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-1">
            <h2 className="ea-ttl">{t(mo.title)}</h2>
            <span className="ea-lead">{t(mo.lead)}</span>
          </div>
          <Badge tone={c.done > 0 ? 'ok' : 'neutral'}>{t(mo.done, { done: c.done, total })}</Badge>
        </div>
        <div className="bar ea-bar" role="img" aria-label={t(mo.doneLabel, { done: c.done, total })}>
          <i style={{ width: `${pct}%` }} />
        </div>
        <ol className="ea-csteps">
          {c.steps.map((s, i) => (
            <li key={s.key}>
              <div className={cn('ea-cstep', s.done && 'done', i === current && 'cur')} aria-current={i === current ? 'step' : undefined}>
                <span className="n">{s.done ? <Icon name="check" size={13} strokeWidth={2.4} /> : i + 1}</span>
                <div className="tx">
                  <span className="font-semibold">{stepTitle(t, s)}</span>
                  <span className="k text-ink-2">{stepText(t, s, c)}</span>
                </div>
                <div className="act">{s.done ? <span className="k whitespace-nowrap">{stepDone(t, s)}</span> : <StepAction s={s} c={c} primary={i === current} />}</div>
              </div>
            </li>
          ))}
        </ol>
      </section>
      <div className="ea-launch-side">
        {c.testCheckouts.length > 0 ? (
          <TestCheckouts c={c} />
        ) : (
          <>
            <SideCard icon="upload" title={t(mo.importTitle)} text={t(mo.importText)} action={<ImportLink />} />
            <SideCard icon="doc" title={t(mo.sponsorsTitle)} text={t(mo.sponsorsText)} action={<SponsorsLink />} />
          </>
        )}
      </div>
    </div>
  )
}

function stepTitle(t: T, s: Step): string {
  switch (s.key) {
    case 'connect':
      return t(mo.connectTitle)
    case 'tier':
      return t(mo.tierTitle)
    case 'emails':
      return t(mo.emailsTitle, { total: s.total ?? 8 })
    case 'page':
      return t(mo.pageTitle)
    case 'test':
      return t(mo.testTitle)
    case 'live':
      return t(mo.liveTitle)
  }
}

function stepText(t: T, s: Step, c: Checklist): string {
  switch (s.key) {
    case 'connect':
      return t(mo.connectText)
    case 'tier':
      return t(mo.tierText)
    case 'emails':
      return s.blocked === 'no_route' ? `${t(mo.emailsText)} ${t(mo.emailsNoRoute)}.` : t(mo.emailsText)
    case 'page':
      return t(mo.pageText, { host: c.upgradeHost })
    case 'test':
      return t(mo.testText)
    case 'live':
      return t(mo.liveText)
  }
}

function stepDone(t: T, s: Step): string {
  switch (s.key) {
    case 'connect':
      return t(mo.connectDone, { mode: s.mode ?? 'test', account: s.account ?? '', date: day(s.at) })
    case 'tier':
      return t(mo.tierDone, { name: s.name ?? '', prices: s.priceLine ?? '' })
    case 'emails':
      return t(mo.emailsCount, { count: s.count ?? 0, total: s.total ?? 0 })
    case 'page':
      return s.name ? t(mo.pageDone, { name: s.name }) : t(mo.pageDoneAt, { date: day(s.at) })
    case 'test': {
      const total = s.total ?? 0
      const paid = s.count ?? 0
      return t(mo.testDone, { total, count: paid, paid: paid === total ? 'all' : paid === 0 ? 'none' : 'some' })
    }
    case 'live':
      return t(mo.liveDone, { account: s.account ?? '', date: day(s.at) })
  }
}

/** A step's button: what to do next, primary on the current step. */
function StepAction({ s, c, primary }: { s: Step; c: Checklist; primary: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  const variant = primary ? 'primary' : 'default'
  switch (s.key) {
    case 'connect':
      return c.canManage ? (
        <ButtonLink to="/w/$workspace/earn/connect" params={params} search={{ mode: 'test' }} size="sm" variant={variant}>
          {t(mo.connectAction)}
        </ButtonLink>
      ) : null
    case 'tier':
      return c.canEdit ? (
        <ButtonLink to="/w/$workspace/earn/plans/tiers/new" params={params} size="sm" variant={variant}>
          {t(mo.tierAction)}
        </ButtonLink>
      ) : null
    case 'emails':
      return s.blocked === 'no_route' ? (
        <ButtonLink to="/w/$workspace/providers/routing" params={params} size="sm" variant={variant}>
          {t(mo.emailsRoute)}
        </ButtonLink>
      ) : (
        <span className="flex items-center gap-2">
          {s.count ? <span className="k whitespace-nowrap">{t(mo.emailsCount, { count: s.count, total: s.total ?? 0 })}</span> : null}
          <ButtonLink to="/w/$workspace/earn/settings" params={params} search={{ tab: 'emails' }} size="sm" variant={variant}>
            {t(mo.emailsAction)}
          </ButtonLink>
        </span>
      )
    case 'page':
      return (
        <ButtonLink to="/w/$workspace/earn/plans/upgrade-page" params={params} size="sm" variant={variant}>
          {t(mo.pageAction)}
        </ButtonLink>
      )
    case 'test':
      return c.testCheckoutUrl ? (
        <ButtonAnchor href={c.testCheckoutUrl} size="sm" variant={variant}>
          {t(mo.testAction)}
        </ButtonAnchor>
      ) : c.canManage ? (
        // Without a test key, the wizard comes first.
        <ButtonLink to="/w/$workspace/earn/connect" params={params} search={{ mode: 'test' }} size="sm" variant={variant}>
          {t(mo.testAction)}
        </ButtonLink>
      ) : (
        <span className="k whitespace-nowrap">{t(mo.testNeedsKey)}</span>
      )
    case 'live':
      return c.canManage ? (
        <ButtonLink to="/w/$workspace/earn/connect" params={params} search={{ mode: 'live' }} size="sm" variant={variant}>
          {t(mo.liveAction)}
        </ButtonLink>
      ) : null
  }
}

function SideCard({ icon, title, text, action }: { icon: 'upload' | 'doc'; title: string; text: string; action: ReactNode }) {
  return (
    <section className="card ea-side-card">
      <span className="ea-side-ico" aria-hidden="true">
        {icon === 'doc' ? (
          <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
            <path d="M4 1.5h5.5L12.5 4.5V14.5h-8.5z" />
            <path d="M9.5 1.5v3h3M6 8h4.5M6 10.5h4.5" />
          </svg>
        ) : (
          <Icon name={icon} size={18} />
        )}
      </span>
      <h3 className="h2">{title}</h3>
      <span className="k text-ink-2">{text}</span>
      <div>{action}</div>
    </section>
  )
}

function ImportLink() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <ButtonLink to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ import: 'new' }} size="sm">
      {t(mo.importAction)}
    </ButtonLink>
  )
}

function SponsorsLink() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <ButtonLink to="/w/$workspace/earn/sponsors" params={{ workspace: slug }} size="sm">
      {t(mo.sponsorsAction)}
    </ButtonLink>
  )
}

/** Test checkouts: left out of every number, kept until cleared. */
function TestCheckouts({ c }: { c: Checklist }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [confirm, setConfirm] = useState(false)
  const clear = $api.useMutation('post', `${EARN}/test-data/clear`)
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="h2">{t(mo.testsTitle)}</h3>
          <span className="k">{t(mo.testsText)}</span>
        </div>
      </div>
      {c.testCheckouts.map((r) => (
        <div key={r.id} className="ea-lrow">
          <span className="lt">
            <b className="font-semibold">{r.email}</b>
            <span className="inline-flex gap-1.5">
              <Badge tone="purple" size="sm">
                {t(me.test)}
              </Badge>
              <Badge tone={statusTone(r.status)} size="sm">
                {statusLabel(t, r.status)}
              </Badge>
            </span>
          </span>
          <span className="k">
            {r.amountCents > 0
              ? t(mo.testsLine, { tier: tierLine(t, r), amount: money(r.amountCents, r.currency, 'always'), date: day(r.since) })
              : t(mo.testsLineFree, { tier: tierLine(t, r), date: day(r.since) })}
          </span>
        </div>
      ))}
      <div className="ea-foot">
        <span className="k">{t(mo.testsKept)}</span>
        {c.canManage ? (
          <Button size="sm" variant="danger" onClick={() => setConfirm(true)}>
            {t(mo.testsClear)}
          </Button>
        ) : null}
      </div>
      <ConfirmDialog
        open={confirm}
        onOpenChange={setConfirm}
        badge={t(mo.clearBadge)}
        title={t(mo.clearTitle)}
        description={t(mo.clearText)}
        confirmLabel={t(mo.clearConfirm)}
        onConfirm={async () => {
          const r = await clear.mutateAsync({ params: { path: { workspace: slug } } })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/checklist`] })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
          toast({ kind: 'success', title: t(mo.cleared), detail: t(mo.clearedDetail, { members: r.members }) })
        }}
      />
    </section>
  )
}
