import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard } from '@/components/ui/Field'
import { Banner, Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { dayMonthYear, orList } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { EARN, EarnPage } from './EarnPage'
import { memberEmailName } from './MemberPanel'
import { me } from './messages'
import { PaymentsTab } from './PaymentsTab'
import { ms } from './settings.messages'

export type SettingsTab = 'payments' | 'emails' | 'access' | 'attribution' | 'pages'
const tabs: SettingsTab[] = ['payments', 'emails', 'access', 'attribution', 'pages']

/** Earn › Settings (EarnSettings, EarnSettingsTabs): Payments · Member emails · Access · Attribution · Pages. */
export function SettingsPage({ tab, onTab }: { tab: SettingsTab; onTab: (t: SettingsTab) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const pay = $api.useQuery('get', `${EARN}/payments`, { params: { path: { workspace: slug } } })
  const labels: Record<SettingsTab, string> = { payments: t(ms.tabPayments), emails: t(ms.tabEmails), access: t(ms.tabAccess), attribution: t(ms.tabAttribution), pages: t(ms.tabPages) }
  return (
    <EarnPage tab="settings" title={labels[tab]}>
      <div className="flex flex-col gap-4">
        <div className="ea-pill-tabs" role="group" aria-label={t(ms.tabsLabel)}>
          {tabs.map((x) => (
            <button key={x} type="button" className="ea-schip" aria-pressed={tab === x} onClick={() => onTab(x)}>
              {labels[x]}
            </button>
          ))}
        </div>
        {pay.data && !pay.data.canManage ? (
          <Note tone="neutral">
            <T
              msg={pay.data.helpers.length ? ms.readOnly : ms.readOnlyNoHelpers}
              values={{ payments: <span className="mono">payments:manage</span>, publish: <span className="mono">transactional:publish</span>, helpers: orList(pay.data.helpers) }}
            />
          </Note>
        ) : null}
        {pay.error ? (
          <ErrorPanel title={t(me.errorLoad)} actions={<Button onClick={() => void pay.refetch()}>{t(me.retry)}</Button>}>
            {errorMessage(pay.error)}
          </ErrorPanel>
        ) : !pay.data ? (
          <div className="card pad" aria-busy="true">
            <Skeleton w="100%" h={300} />
          </div>
        ) : tab === 'payments' ? (
          <PaymentsTab pay={pay.data} />
        ) : tab === 'emails' ? (
          <MemberEmailsTab />
        ) : tab === 'access' ? (
          <AccessTab />
        ) : tab === 'attribution' ? (
          <AttributionTab />
        ) : (
          <PagesTab />
        )}
      </div>
    </EarnPage>
  )
}

function emailWhen(t: ReturnType<typeof useT>, email: string): string {
  return (
    {
      welcome: t(ms.whenWelcome),
      trial_ending: t(ms.whenTrialEnding),
      payment_failed: t(ms.whenPaymentFailed),
      renewal_reminder: t(ms.whenRenewal),
      canceled: t(ms.whenCanceled),
      gift_purchased: t(ms.whenGiftPurchased),
      gift_received: t(ms.whenGiftReceived),
      gift_ending: t(ms.whenGiftEnding),
    }[email] ?? ''
  )
}

/** Member emails (EarnSettingsTabs): the eight templates, the sender and the route. */
function MemberEmailsTab() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/settings/member-emails`, { params: { path: { workspace: slug } } })
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(me.retry)}</Button>} />
  if (!q.data) return <Skeleton w="100%" h={300} />
  const d = q.data
  const drafts = d.emails.filter((e) => e.status === 'draft')
  return (
    <div className="flex flex-col gap-4">
      {drafts.map((e) => (
        <Banner
          key={e.email}
          tone="warn"
          title={t(ms.emailsWarnTitle, { template: e.template })}
          actions={
            e.templateId ? (
              <ButtonLink to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: e.templateId }} size="sm">
                {t(ms.openTemplate, { template: e.template })}
              </ButtonLink>
            ) : null
          }
        >
          {e.email === 'gift_ending' && d.firstGiftEnding
            ? t(ms.emailsWarnText, { template: e.template, date: dayMonthYear(d.firstGiftEnding) })
            : t(ms.emailsWarnTextNoDate, { name: memberEmailName(t, e.email, d.tierShort), template: e.template })}
        </Banner>
      ))}
      {!d.routePrimary ? (
        <Banner
          tone="bad"
          title={t(ms.routeNone)}
          actions={
            <ButtonLink to="/w/$workspace/providers/routing" params={{ workspace: slug }} size="sm">
              {t(ms.chooseRoute)}
            </ButtonLink>
          }
        >
          {t(ms.routeNoneText)}
        </Banner>
      ) : null}
      <section className="card">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2">{t(ms.emailsTitle, { count: d.emails.length })}</h2>
            <span className="k">{t(ms.emailsCaption)}</span>
          </div>
          <ButtonLink to="/w/$workspace/transactional" params={{ workspace: slug }} size="sm">
            {t(ms.allTemplates)}
          </ButtonLink>
        </div>
        <TableScroll label={t(ms.tabEmails)}>
          <table className="tbl" aria-label={t(ms.tabEmails)}>
            <thead>
              <tr>
                <th scope="col">{t(ms.colEmail)}</th>
                <th scope="col">{t(ms.colWhen)}</th>
                <th scope="col">{t(ms.colTemplate)}</th>
                <th scope="col">{t(ms.colVersion)}</th>
                <th scope="col">
                  <span className="vh">{t(ms.open)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {d.emails.map((e) => (
                <tr key={e.email}>
                  <td className="font-semibold">{memberEmailName(t, e.email, d.tierShort)}</td>
                  <td>{emailWhen(t, e.email)}</td>
                  <td className="ea-mono">{e.template || '—'}</td>
                  <td>
                    {e.status === 'live' ? (
                      <Badge tone="ok">{t(ms.vLive, { version: e.liveVersion })}</Badge>
                    ) : e.status === 'draft' ? (
                      <Badge tone="warn">{t(ms.vDraft, { version: e.draftVersion || 1 })}</Badge>
                    ) : (
                      <Badge tone="neutral">{t(ms.createDrafts)}</Badge>
                    )}
                  </td>
                  <td className="r">
                    {e.templateId ? (
                      <ButtonLink to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: e.templateId }} size="sm" variant={e.status === 'draft' ? 'primary' : 'default'}>
                        {e.status === 'draft' ? t(ms.reviewPublish) : t(ms.open)}
                      </ButtonLink>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </section>
      <section className="card">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2">{t(ms.senderTitle)}</h2>
            <span className="k">{t(ms.senderCaption)}</span>
          </div>
        </div>
        <div className="pad">
          <dl className="ea-kv">
            <dt>{t(ms.from)}</dt>
            <dd>{d.fromEmail ? `${d.fromName} <${d.fromEmail}>` : '—'}</dd>
            <dt>{t(ms.replyTo)}</dt>
            <dd>{d.replyTo || '—'}</dd>
            <dt>{t(ms.category)}</dt>
            <dd className="flex items-center gap-2">
              {d.categoryName || '—'}
              {d.categoryKind === 'essential' ? <Badge tone="neutral">{t(ms.essential)}</Badge> : null}
            </dd>
            <dt>{t(ms.route)}</dt>
            <dd className="flex flex-wrap items-center gap-2">
              {d.routePrimary ? (
                <>
                  <b>{d.routePrimary}</b>
                  {d.routeFallback ? (
                    <>
                      → <b>{d.routeFallback}</b> <span className="k">{t(ms.fallback)}</span>
                    </>
                  ) : null}
                  <span className="k">· {t(ms.stream)}</span>
                </>
              ) : (
                t(ms.routeNone)
              )}
              <ButtonLink to="/w/$workspace/providers/routing" params={{ workspace: slug }} size="sm">
                {t(ms.editRoute)}
              </ButtonLink>
            </dd>
          </dl>
        </div>
      </section>
      <section className="card pad flex flex-col items-start gap-2">
        <h2 className="h2">{t(ms.stripeEmailsTitle)}</h2>
        <p className="m-0 max-w-[760px] text-ink-2">{t(ms.stripeEmailsText)}</p>
        {d.emailsUrl ? (
          <ButtonAnchor href={d.emailsUrl} size="sm" iconRight="external">
            {t(ms.stripeEmails)}
          </ButtonAnchor>
        ) : null}
      </section>
    </div>
  )
}

/** Access (EarnSettingsTabs): what a failed payment does to paid access. */
function AccessTab() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const q = $api.useQuery('get', `${EARN}/settings/access`, { params: { path: { workspace: slug } } })
  const save = $api.useMutation('put', `${EARN}/settings/access`)
  const [rule, setRule] = useState<'keep' | 'end'>()
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} />
  if (!q.data) return <Skeleton w="100%" h={240} />
  const value = rule ?? q.data.onPaymentFailure
  const ro = !q.data.canManage
  const apply = (next: 'keep' | 'end') => {
    const before = q.data!.onPaymentFailure
    setRule(next)
    save.mutate(
      { params: { path: { workspace: slug } }, body: { onPaymentFailure: next } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/settings/access`] })
          toast({
            kind: 'success',
            title: t(ms.accessSaved),
            detail: t(ms.accessSavedDetail),
            undo: () => apply(before),
          })
        },
        onError: (e) => {
          setRule(undefined)
          toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  }
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.accessTitle)}</h2>
          <span className="k">{t(ms.accessCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-4">
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="mb-1 text-[14px] font-semibold">{t(ms.onFailure)}</legend>
          <OptionCard name="failure" value="keep" checked={value === 'keep'} onChange={() => apply('keep')} title={t(ms.keep)} description={t(ms.keepHint)} disabled={ro} />
          <OptionCard name="failure" value="end" checked={value === 'end'} onChange={() => apply('end')} title={t(ms.end)} description={t(ms.endHint)} disabled={ro} />
        </fieldset>
        <div className="flex flex-col gap-1">
          <h3 className="m-0 text-[14px] font-semibold">{t(ms.always)}</h3>
          <ul className="m-0 flex flex-col gap-1 pl-5 text-[13px]">
            <li>{t(ms.always1)}</li>
            <li>{t(ms.always2)}</li>
            <li>{t(ms.always3)}</li>
          </ul>
        </div>
        <span className="k">{t(ms.accessNote)}</span>
      </div>
    </section>
  )
}

/** Attribution (the frame; the window and click tokens come with revenue reports). */
function AttributionTab() {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.attrTitle)}</h2>
          <span className="k">{t(ms.attrCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-2">
        <h3 className="m-0 text-[14px] font-semibold">{t(ms.attrWindow)}</h3>
        <span className="badge mut w-fit">{t(ms.attrDays, { days: 7 })}</span>
        <span className="k">{t(ms.attrLater)}</span>
      </div>
    </section>
  )
}

/** Pages (EarnSettingsTabs): the Earn pages' addresses, terms of sale and the support email. */
function PagesTab() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const q = $api.useQuery('get', `${EARN}/settings/pages`, { params: { path: { workspace: slug } } })
  const save = $api.useMutation('put', `${EARN}/settings/pages`)
  const [v, setV] = useState<Schemas['EarnPagesInput']>()
  const [errors, setErrors] = useState<Record<string, string>>({})
  const canEdit = useCan('earn:edit')
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} />
  if (!q.data) return <Skeleton w="100%" h={240} />
  const d = q.data
  const cur = v ?? { termsUrl: d.termsUrl, supportEmail: d.supportEmail }
  const dirty = cur.termsUrl !== d.termsUrl || cur.supportEmail !== d.supportEmail
  const ro = !d.canEdit || !canEdit
  return (
    <div className="flex flex-col gap-4">
      <section className="card">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2">{t(ms.pagesTitle)}</h2>
            <span className="k">{t(ms.pagesCaption)}</span>
          </div>
        </div>
        <div className="pad">
          <dl className="ea-kv">
            <dt>{t(ms.pageUpgrade)}</dt>
            <dd className="flex flex-wrap items-center gap-2">
              <span className="mono">{d.upgradeHost}</span>
              <Badge tone={d.upgradeLive ? 'ok' : 'neutral'}>{d.upgradeLive ? t(me.live) : '—'}</Badge>
              <span className="k">{t(ms.pageVisits, { count: d.visits })}</span>
              <ButtonLink to="/w/$workspace/earn/plans/upgrade-page" params={{ workspace: slug }} size="sm">
                {t(ms.editPage)}
              </ButtonLink>
            </dd>
            <dt>{t(ms.pageWelcome)}</dt>
            <dd className="mono">{d.upgradeHost}/welcome</dd>
            <dt>{t(ms.pageCancel)}</dt>
            <dd className="mono">{d.cancelHost}</dd>
            <dt>{t(ms.pageGift)}</dt>
            <dd className="mono">{d.giftHost}</dd>
          </dl>
        </div>
      </section>
      <section className="card">
        <div className="sec">
          <h2 className="h2">{t(ms.terms)}</h2>
          {ro ? null : (
            <Button
              size="sm"
              variant="primary"
              disabled={!dirty}
              loading={save.isPending}
              onClick={() =>
                save.mutate(
                  { params: { path: { workspace: slug } }, body: cur },
                  {
                    onSuccess: () => {
                      setErrors({})
                      setV(undefined)
                      void qc.invalidateQueries({ queryKey: ['get', `${EARN}/settings/pages`] })
                      toast({ kind: 'success', title: t(ms.pagesSaved) })
                    },
                    onError: (e) => {
                      if (e instanceof ApiError && e.problem.errors?.length) {
                        const next: Record<string, string> = {}
                        for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
                        setErrors(next)
                      } else toast({ kind: 'error', title: errorMessage(e) })
                    },
                  },
                )
              }
            >
              {t(ms.save)}
            </Button>
          )}
        </div>
        <div className="pad grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))' }}>
          <Field label={t(ms.terms)} hint={t(ms.termsHint)} error={errors.termsUrl}>
            <Input type="url" value={cur.termsUrl} disabled={ro} maxLength={500} onChange={(e) => setV({ ...cur, termsUrl: e.target.value })} />
          </Field>
          <Field label={t(ms.support)} hint={t(ms.supportHint)} error={errors.supportEmail}>
            <Input type="email" value={cur.supportEmail} disabled={ro} maxLength={320} onChange={(e) => setV({ ...cur, supportEmail: e.target.value })} />
          </Field>
        </div>
      </section>
    </div>
  )
}
