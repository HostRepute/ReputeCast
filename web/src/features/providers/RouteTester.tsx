import { keepPreviousData } from '@tanstack/react-query'
import { Fragment, useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Field, Input, Select } from '@/components/ui/Field'
import { Segmented } from '@/components/ui/Tabs'
import { formatList, useT } from '@/i18n'
import { clock, count, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { providerColor, streamName, type Stream } from './common'
import { costText, fallbackText, fieldText, mailboxName, splitPair, stepText, toInput, type Condition, type Draft, type RoutingConfig } from './routing-model'
import { mr } from './routing.messages'

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const h = window.setTimeout(() => setV(value), ms)
    return () => window.clearTimeout(h)
  }, [value, ms])
  return v
}

const looksLikeEmail = (s: string) => /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(s.trim())

/** The campaign simulation of the saved routing, and of the unsaved draft when there is one (PR-14). */
export function useCampaignSimulation(draft: Draft | undefined, enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/routing/test-campaign',
    { params: { path: { workspace: slug } }, body: { draft: draft ? toInput(draft, true) : undefined } },
    { enabled, placeholderData: keepPreviousData, retry: false },
  )
}

/**
 * The route tester (PR-14): "Which provider would send this?" for one
 * recipient, or where a whole campaign goes, on the saved routing; with
 * unsaved edits it also says what they would change.
 */
export function RouteTester({ cfg, draft }: { cfg: RoutingConfig; draft?: Draft }) {
  const t = useT()
  const [tab, setTab] = useState<'recipient' | 'campaign'>('recipient')
  return (
    <section className="card flex flex-col" aria-labelledby="pv-tester">
      <div className="flex flex-col gap-0.5 border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-tester">
          {t(mr.testerTitle)}
        </h2>
        <span className="k">{t(mr.testerSaved)}</span>
      </div>
      {/* Recipient or Campaign: a segmented control, as on the board. */}
      <div className="px-5 pt-4">
        <Segmented
          className="pv-seg"
          value={tab}
          onChange={setTab}
          label={t(mr.testerTabs)}
          items={[
            { value: 'recipient', label: t(mr.tabRecipient) },
            { value: 'campaign', label: t(mr.tabCampaign) },
          ]}
        />
      </div>
      {tab === 'recipient' ? <RecipientTester cfg={cfg} draft={draft} /> : <CampaignTester cfg={cfg} draft={draft} />}
    </section>
  )
}

function RecipientTester({ cfg, draft }: { cfg: RoutingConfig; draft?: Draft }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [stream, setStream] = useState<Stream>('broadcast')
  const [listId, setListId] = useState(cfg.lists[0]?.id ?? '')
  const [template, setTemplate] = useState(cfg.templates.find((x) => x.id === 'password-reset')?.id ?? cfg.templates[0]?.id ?? '')
  const txn = stream === 'transactional'
  const [domain, setDomain] = useState(cfg.senderDomains[0] ?? '')
  const [email, setEmail] = useState('sam.rivera@outlook.com')
  const address = useDebounced(email.trim(), 300)
  const valid = looksLikeEmail(address)
  const body = {
    stream,
    listId: txn ? undefined : listId || undefined,
    template: txn ? template || undefined : undefined,
    senderDomain: domain || undefined,
    email: address,
  }
  const path = { params: { path: { workspace: slug } } }
  // The saved routing, and the unsaved draft beside it when there is one.
  const q = $api.useQuery('post', '/api/v1/workspaces/{workspace}/routing/test', { ...path, body }, { enabled: valid, placeholderData: keepPreviousData, retry: false })
  const dq = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/routing/test',
    { ...path, body: { ...body, draft: draft ? toInput(draft, true) : undefined } },
    { enabled: valid && Boolean(draft), placeholderData: keepPreviousData, retry: false },
  )
  const r = valid ? q.data : undefined
  const dr = valid && draft ? dq.data : undefined
  const hasConditions = Boolean(cfg.streams.find((s) => s.stream === stream)?.conditions.length)
  return (
    <div className="flex flex-col gap-3 px-5 pt-3 pb-[18px]">
      <Field label={t(mr.testerStream)}>
        <Select value={stream} onChange={(e) => setStream(e.target.value as Stream)}>
          {(['broadcast', 'automation', 'system', 'transactional'] as Stream[]).map((s) => (
            <option key={s} value={s}>
              {streamName[s]}
            </option>
          ))}
        </Select>
      </Field>
      {txn ? (
        <Field label={t(mr.testerTemplate)}>
          <Select value={template} onChange={(e) => setTemplate(e.target.value)}>
            {cfg.templates.length ? null : <option value="">{t(mr.noTemplates)}</option>}
            {cfg.templates.map((x) => (
              <option key={x.id} value={x.id}>
                {x.id}
              </option>
            ))}
          </Select>
        </Field>
      ) : (
        <Field label={t(mr.testerList)}>
          <Select value={listId} onChange={(e) => setListId(e.target.value)}>
            {cfg.lists.length ? null : <option value="">{t(mr.noLists)}</option>}
            {cfg.lists.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </Select>
        </Field>
      )}
      <Field label={t(mr.testerDomain)}>
        <Select value={domain} onChange={(e) => setDomain(e.target.value)}>
          {cfg.senderDomains.length ? null : <option value="">{t(mr.noDomains)}</option>}
          {cfg.senderDomains.map((d) => (
            <option key={d} value={d}>
              {d}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t(mr.testerRecipient)}>
        <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder={'name@example.com'} autoComplete="off" spellCheck={false} />
      </Field>
      <div className="flex flex-col gap-2.5 rounded-[10px] border border-line bg-ground-3 p-3.5" aria-live="polite">
        {!valid ? (
          <span className="k">{t(mr.enterAddress)}</span>
        ) : q.isError ? (
          <span className="text-[13px] text-bad-fg" role="alert">
            {errorMessage(q.error)}
          </span>
        ) : !r ? (
          <span className="k">{t(mr.working)}</span>
        ) : (
          <>
            <div className="flex items-center gap-2.5">
              <span className="dot h-3 w-3" style={{ color: providerColor[r.color] }} aria-hidden="true" />
              <span className="text-[18px] font-bold">{r.provider}</span>
            </div>
            <Row k={t(mr.mailboxProvider)}>
              <span className="flex min-w-0 flex-col items-end text-right">
                <span className="n">
                  {mailboxName[r.mailboxProvider] ?? r.mailboxProvider}
                  {r.mailboxDetail ? ` · ${r.mailboxDetail}` : ''}
                </span>
                <span className="mono k break-all">{r.mx ? t(mr.mxLine, { host: r.mx }) : r.mailboxProvider === 'other' ? t(mr.mxOther) : t(mr.mxUnknown)}</span>
              </span>
            </Row>
            <Row k={t(mr.dkimSelector)}>
              <span className="mono min-w-0 text-right break-all">{r.dkimSelector || t(mr.notSetUp)}</span>
            </Row>
            <Row k={t(mr.cost)}>
              <span className="n">{costText(r.cost)}</span>
            </Row>
            <ol className="m-0 flex list-none flex-col gap-2 border-t border-line pt-2.5 pl-0">
              {r.steps.map((st, i) => (
                <li key={i} className="pv-step">
                  <span className="num" aria-hidden="true">
                    {i + 1}
                  </span>
                  <span>{stepText(st, stream, hasConditions)}</span>
                </li>
              ))}
            </ol>
            <div className="text-[12.5px] text-ink-2">
              <span className="font-semibold text-ink">{t(mr.ifItFails)}</span> {fallbackText(r, stream)}
            </div>
            {draft && dr ? (
              <div className="text-[12.5px] text-ink-2">
                <span className="font-semibold text-ink">{t(mr.draftLine, { pair: splitPair(draft[stream]) })}</span>{' '}
                {dr.providerId === r.providerId ? t(mr.draftStays, { provider: r.provider }) : t(mr.draftMoves, { provider: dr.provider })}
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  )
}

/** Where a whole campaign goes: the next broadcast's audience through the saved routing, and the unsaved draft beside it. */
function CampaignTester({ cfg, draft }: { cfg: RoutingConfig; draft?: Draft }) {
  const t = useT()
  const q = useCampaignSimulation(draft)
  const r = q.data
  if (q.isError) {
    return (
      <div className="px-5 pt-3 pb-[18px]">
        <span className="text-[13px] text-bad-fg" role="alert">
          {errorMessage(q.error)}
        </span>
      </div>
    )
  }
  if (!r) {
    return (
      <div className="px-5 pt-3 pb-[18px]">
        <span className="k">{t(mr.campaignLoading)}</span>
      </div>
    )
  }
  if (r.none) {
    return (
      <div className="px-5 pt-3 pb-[18px]">
        <span className="k">{t(mr.campaignNone)}</span>
      </div>
    )
  }
  const lists = cfg.lists.filter((l) => r.lists.includes(l.id)).map((l) => l.name)
  return (
    <div className="flex flex-col gap-3 px-5 pt-3 pb-[18px]" aria-live="polite">
      <div className="flex flex-col gap-0.5">
        <span className="k">{t(mr.campaignTitle)}</span>
        <span className="font-semibold">
          {r.name}
          {r.scheduledAt ? ` · ${shortDate(r.scheduledAt)}, ${clock(r.scheduledAt)}` : ''}
        </span>
        <span className="text-[13px]">{t(mr.campaignRecipients, { total: count(r.total) })}</span>
        <span className="k">
          {t(mr.campaignAudience, { lists: lists.join(', ') || '—', domain: r.domain || '—', hasSunset: r.sunset > 0 ? 'yes' : 'no', sunset: count(r.sunset) })}
        </span>
        {r.estimated ? <span className="k">{t(mr.campaignEstimated, { simulated: count(r.simulated) })}</span> : null}
      </div>
      <div className="flex flex-col gap-1.5 border-t border-line pt-2.5">
        {r.shares.map((x) => (
          <Row
            key={x.providerId}
            k={
              <span className="flex items-center gap-2">
                <span className="dot h-2.5 w-2.5" style={{ color: providerColor[x.color] }} aria-hidden="true" />
                {x.provider}
              </span>
            }
          >
            <span className="n">{count(x.count)}</span>
          </Row>
        ))}
        <Row k={t(mr.conditionsMatched)}>
          <span className="n text-right">
            {r.conditions.length
              ? r.conditions
                  .map((c) => t(mr.conditionMatchedLine, { field: fieldText[c.field as Condition['field']] ?? c.field, values: c.values.join(', '), count: count(c.count) }))
                  .join('; ')
              : t(mr.conditionsNone)}
          </span>
        </Row>
        {r.warmup.map((w) => (
          <WarmupRows key={w.providerId} w={w} />
        ))}
      </div>
      <div className="flex flex-col gap-1.5 border-t border-line pt-2.5">
        {r.shares.map((x) => (
          <Fragment key={`${x.providerId}-fails`}>
            <Row k={t(mr.ifProviderFails, { provider: x.provider })}>
              <span className="text-right text-[13px]">{t(mr.policyText, { policy: x.policy, hasFallback: x.fallback ? 'yes' : 'no', fallback: x.fallback ?? '' })}</span>
            </Row>
            {x.down ? <DownNow provider={x.provider} down={x.down} /> : null}
          </Fragment>
        ))}
      </div>
      {draft && r.draft ? (
        <div className="text-[12.5px] text-ink-2">
          <span className="font-semibold text-ink">{t(mr.draftLine, { pair: splitPair(draft.broadcast) })}</span>{' '}
          {t(mr.campaignDraft, { shares: r.draft.shares.map((x) => `${x.provider} ${count(x.count)}`).join(' · '), moved: count(r.draft.moved) })}
        </div>
      ) : null}
    </div>
  )
}

/**
 * What a warming provider's plan does to its share on the send day
 * (R3.U4c): recipients outside the day's tiers go to a warm provider, what's
 * over the cap (per mailbox provider when the plan splits it) overflows to
 * a warm provider or waits for the next day's cap, all of it while the plan
 * is paused.
 */
function WarmupRows({ w }: { w: Schemas['RouteCampaignWarmup'] }) {
  const t = useT()
  const over = w.mailboxCaps.filter((p) => p.over > 0)
  return (
    <>
      {w.tier > 0 && w.tierTo ? (
        <Row k={t(mr.tierTo, { day: w.day, provider: w.tierTo })}>
          <span className="n">{count(w.tier)}</span>
        </Row>
      ) : null}
      {w.to ? (
        <Row k={t(mr.overflowTo, { provider: w.to })}>
          <span className="n">{count(w.over)}</span>
        </Row>
      ) : w.over > 0 ? (
        <Row k={t(w.paused ? mr.heldByPause : mr.heldByCap, { provider: w.provider })}>
          <span className="n">{count(w.over)}</span>
        </Row>
      ) : null}
      <span className="k">
        {w.paused
          ? w.to
            ? t(mr.capPausedOverflow, { provider: w.provider, to: w.to })
            : t(mr.capPaused, { provider: w.provider })
          : t(mr.overflowCap, { provider: w.provider, date: shortDate(w.date), day: w.day, cap: count(w.cap), spread: w.spread ? 'yes' : 'no' })}
        {over.length && !w.paused
          ? ` ${t(mr.overParts, { parts: formatList(over.map((p) => t(mr.overPart, { mailbox: mailboxName[p.mailbox] ?? p.mailbox, recipients: count(p.recipients), cap: count(p.left) }))) })}`
          : ''}
        {w.over > 0 && !w.to && w.resumes ? ` ${t(mr.heldUntil, { date: shortDate(w.resumes) })}` : ''}
      </span>
    </>
  )
}

/** Where a provider that can't send now sends its share, as the send would decide it (R3.U4c). */
function DownNow({ provider, down }: { provider: string; down: Schemas['RouteCampaignDown'] }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-0.5 text-[12.5px] text-warn-fg">
      {down.moved > 0 ? <span>{t(mr.downMoves, { provider, count: count(down.moved), to: down.to ?? '' })}</span> : null}
      {down.held > 0 ? <span>{t(mr.downWaits, { provider, count: count(down.held), reason: down.reason ?? 'pause_and_ask' })}</span> : null}
      {down.checks.length ? <span className="k">{down.checks.join(' ')}</span> : null}
    </div>
  )
}

function Row({ k, children }: { k: ReactNode; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
      <span className="k shrink-0">{k}</span>
      {children}
    </div>
  )
}
