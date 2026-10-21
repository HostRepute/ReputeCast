import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN } from '../EarnPage'
import { money } from '../format'
import { GiftStrip } from '../gifts/GiftStrip'
import { EndOfferDialog } from './EndOfferDialog'
import { discounted, discountLine, forLine, howLine } from './lines'
import { mof } from './messages'
import './offers.css'

type Offer = Schemas['Offer']
type TT = ReturnType<typeof useT>

/** Plans & offers' part after the tiers: the offers, then the gift options line. */
export function OffersSection() {
  return (
    <>
      <OffersCard />
      <GiftStrip />
    </>
  )
}

/**
 * Plans & offers' Offers (EarnPlans, EarnPlansStates): each offer's
 * discount, who it's for, how readers get it, its use and its status, with
 * a note when one reached its limit or ended.
 */
function OffersCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/offers`, { params: { path: { workspace: slug } } })
  const [ending, setEnding] = useState<Offer>()
  if (q.error) return null
  if (!q.data) {
    return (
      <section className="card of-card" aria-busy="true">
        <div className="pad">
          <Skeleton w="30%" h={18} />
          <Skeleton w="100%" h={120} className="mt-3" />
        </div>
      </section>
    )
  }
  const l = q.data
  const ro = !l.canEdit
  const caption = !l.connected ? t(mof.captionDraft) : l.account ? t(mof.caption, { account: l.account, provider: l.provider }) : t(mof.captionNoAccount)
  const atLimit = l.items.find((o) => o.state === 'limit')
  const ended = atLimit ? undefined : l.items.find((o) => o.state === 'ended' && o.endedReason === 'date')
  const flagged = atLimit ?? ended
  return (
    <section className="card of-card" aria-labelledby="of-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="of-h">
            {t(mof.title, { count: l.items.length })}
          </h2>
          <span className="k">{caption}</span>
        </div>
        {ro ? null : (
          <ButtonLink to="/w/$workspace/earn/plans/offers/new" params={{ workspace: slug }} icon="plus">
            {t(mof.newOffer)}
          </ButtonLink>
        )}
      </div>
      {atLimit && !ro ? (
        <div className="pad pb-0">
          <div className="note w" role="status">
            <Icon name="alertCircle" size={16} />
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
              <span className="font-bold">{t(mof.limitTitle, { name: atLimit.name })}</span>
              <span>{t(mof.limitText, { name: atLimit.name, limit: count(atLimit.limit ?? 0) })}</span>
              <div className="mt-1 flex flex-wrap gap-2">
                <ButtonLink size="sm" to="/w/$workspace/earn/plans/offers/$offerId" params={{ workspace: slug, offerId: atLimit.id }}>
                  {t(mof.raiseLimit)}
                </ButtonLink>
                <Button size="sm" onClick={() => setEnding(atLimit)}>
                  {t(mof.endOffer)}
                </Button>
              </div>
            </div>
          </div>
        </div>
      ) : null}
      {ended ? (
        <div className="pad pb-0">
          <Note tone="neutral">{endedNote(t, ended)}</Note>
        </div>
      ) : null}
      {l.items.length === 0 ? (
        <p className="pad k m-0">{t(mof.none)}</p>
      ) : (
        <>
          {/* relative: the hidden header text stays inside the scroller. */}
          <div className="relative overflow-x-auto max-md:hidden">
            <table className="tbl of-tbl" aria-labelledby="of-h">
              <thead>
                <tr>
                  <th scope="col">{t(mof.colOffer)}</th>
                  <th scope="col">{t(mof.colDiscount)}</th>
                  <th scope="col" className="max-lg:hidden">
                    {t(mof.colFor)}
                  </th>
                  <th scope="col" className="max-lg:hidden">
                    {t(mof.colHow)}
                  </th>
                  <th scope="col">{t(mof.colUse)}</th>
                  <th scope="col">{t(mof.colStatus)}</th>
                  <th scope="col">
                    <span className="vh">{ro ? t(mof.view) : t(mof.edit)}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {l.items.map((o) => (
                  <tr key={o.id} className={o.id === flagged?.id ? 'hl' : undefined}>
                    <td>
                      <div className="flex flex-col items-start gap-1">
                        <Link
                          to="/w/$workspace/earn/plans/offers/$offerId"
                          params={{ workspace: slug, offerId: o.id }}
                          className="font-semibold text-ink no-underline"
                        >
                          {o.name}
                        </Link>
                        <Badge tone="neutral" size="sm">
                          {t(mof.kind, { kind: o.kind })}
                        </Badge>
                      </div>
                    </td>
                    <td className="of-wrap">{discountLine(t, o)}</td>
                    <td className="of-wrap max-lg:hidden">{forLine(t, o)}</td>
                    <td className="of-wrap max-lg:hidden">
                      <How o={o} />
                    </td>
                    <td>
                      <Use o={o} t={t} />
                    </td>
                    <td>
                      <Status o={o} provider={l.provider || 'Stripe'} />
                    </td>
                    <td className="text-right">
                      <ButtonLink size="sm" to="/w/$workspace/earn/plans/offers/$offerId" params={{ workspace: slug, offerId: o.id }}>
                        {ro ? t(mof.view) : t(mof.edit)}
                      </ButtonLink>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {/* Phones: a row per offer, the whole row a link. */}
          <ul className="m-0 hidden list-none p-0 max-md:block" aria-labelledby="of-h">
            {l.items.map((o) => (
              <li key={o.id}>
                <Link
                  to="/w/$workspace/earn/plans/offers/$offerId"
                  params={{ workspace: slug, offerId: o.id }}
                  className={`of-row no-underline${o.id === flagged?.id ? ' hl' : ''}`}
                >
                  <span className="flex items-start justify-between gap-2.5">
                    <b className="font-semibold">{o.name}</b>
                    <StatusBadge o={o} provider={l.provider || 'Stripe'} />
                  </span>
                  <span className="k text-ink-2">
                    {t(mof.kind, { kind: o.kind })} · {discountLine(t, o)}
                  </span>
                  <span className="k">
                    {forLine(t, o)} · {usageLine(t, o)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </>
      )}
      {ending ? <EndOfferDialog offer={ending} provider={l.provider || 'Stripe'} onClose={() => setEnding(undefined)} /> : null}
    </section>
  )
}

function endedNote(t: TT, o: Offer): string {
  const a = o.applies[0]
  const date = o.endedAt ? dayMonthDate(o.endedAt) : o.endsAt ? dayMonthDate(o.endsAt) : ''
  if (!a) return t(mof.endedNoteEarly, { name: o.name, date })
  return t(mof.endedNote, { name: o.name, date, count: o.results.redemptions, price: money(discounted(a.priceCents, o), a.currency), period: a.period })
}

function How({ o }: { o: Offer }) {
  const t = useT()
  if (o.kind === 'signup' && o.byLink) {
    const link = (c: ReactNode) => <span className="mono of-link">{c}</span>
    return <T msg={o.byCode ? mof.howBothRich : mof.howLinkRich} values={{ code: o.code, link: o.linkHost ?? '', l: link }} />
  }
  return <>{howLine(t, o)}</>
}

/** The use in words, for the phone rows: "45 of 100 used". */
function usageLine(t: TT, o: Offer): string {
  const r = o.results
  if (o.kind === 'retention') {
    return r.shown ? t(mof.phoneShown, { shown: r.shown, accepted: count(r.accepted), rate: pct(r.acceptRate, 0) }) : t(mof.phoneShownNone)
  }
  if (o.kind === 'referral') return t(mof.phoneCodes, { issued: r.codesIssued, redeemed: count(r.codesRedeemed) })
  if (o.limit) return t(mof.useLabel, { used: count(r.redemptions), limit: count(o.limit) })
  return t(mof.useCount, { count: r.redemptions })
}

function Use({ o, t }: { o: Offer; t: TT }) {
  const r = o.results
  if (o.kind === 'retention') {
    return (
      <div className="flex flex-col gap-0.5">
        <b className="font-semibold">{t(mof.accepted, { count: count(r.accepted) })}</b>
        <span className="k">{r.shown ? t(mof.shown, { count: r.shown, rate: pct(r.acceptRate, 0) }) : t(mof.shownNone)}</span>
      </div>
    )
  }
  if (o.kind === 'referral') {
    return (
      <div className="flex flex-col gap-0.5">
        <b className="font-semibold">{t(mof.redeemed, { count: count(r.codesRedeemed) })}</b>
        <span className="k">{t(mof.issued, { count: r.codesIssued })}</span>
      </div>
    )
  }
  if (o.limit) {
    const full = r.redemptions >= o.limit
    return (
      <div className="flex min-w-[100px] flex-col gap-1">
        <b className="font-semibold tabular-nums">{t(mof.useOf, { used: count(r.redemptions), limit: count(o.limit) })}</b>
        <div className="bar" role="img" aria-label={t(mof.useLabel, { used: count(r.redemptions), limit: count(o.limit) })}>
          <i style={{ width: `${Math.min(100, (r.redemptions / o.limit) * 100)}%`, background: full ? 'var(--warn-bar)' : 'var(--accent)' }} />
        </div>
      </div>
    )
  }
  return <b className="font-semibold">{t(mof.useCount, { count: r.redemptions })}</b>
}

/** The status word: Live, Limit reached, Ended, Coupon missing, Draft. */
function StatusBadge({ o, provider }: { o: Offer; provider: string }) {
  const t = useT()
  switch (o.state) {
    case 'limit':
      return <Badge tone="warn">{t(mof.stLimit)}</Badge>
    case 'ended':
      return <Badge tone="neutral">{t(mof.stEnded)}</Badge>
    case 'missing':
      return <Badge tone="bad">{t(mof.stMissing, { provider })}</Badge>
    case 'draft':
      return <Badge tone="neutral">{t(mof.stDraft)}</Badge>
    default:
      return <Badge tone="ok">{t(mof.stLive)}</Badge>
  }
}

/** The status word and what it means: until when, or what readers see. */
function Status({ o, provider }: { o: Offer; provider: string }) {
  const t = useT()
  let line: string | undefined
  switch (o.state) {
    case 'limit':
      line = t(mof.stNormal)
      break
    case 'ended':
      line = o.endedAt ? dayMonthDate(o.endedAt) : undefined
      break
    case 'missing':
      break
    case 'draft':
      line = o.sync === 'failed' ? t(mof.stFailed, { provider }) : t(mof.stDraftText, { provider })
      break
    default:
      if (o.kind === 'referral') line = t(mof.stCodes, { days: o.codeDays })
      else if (o.endsAt) line = t(mof.stEnds, { date: dayMonthDate(o.endsAt) })
  }
  return (
    <div className="flex flex-col items-start gap-1">
      <StatusBadge o={o} provider={provider} />
      {line ? <span className="k whitespace-nowrap">{line}</span> : null}
    </div>
  )
}
