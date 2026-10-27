import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { andList, count, shortDate } from '@/lib/format'
import { useCan, useOptionalWorkspace } from '@/lib/session'
import { money } from '@/features/earn/format'
import { mrp } from '@/features/earn/revenue/report.messages'
import { m } from './messages'
import type { CampaignReport } from './words'

type Insight = Schemas['ReportInsight']

const tone: Record<Insight['tone'], Tone> = { bad: 'bad', warn: 'warn', info: 'info' }

/**
 * "Insights": what stands out in the report's numbers, each with what to
 * do about it (Report board). Read-only links show them without actions.
 */
export function Insights({ r, onShowLink }: { r: CampaignReport; onShowLink?: (position: number) => void }) {
  const t = useT()
  const ws = useOptionalWorkspace()
  const word: Record<Insight['kind'], string> = { blocked: t(m.insightBlocked), mailbox_bounces: t(m.insightWarn), top_link: t(m.insightInfo) }
  const action = (x: Insight): ReactNode => {
    if (!x.action || !x.actionLabel) return null
    switch (x.action) {
      case 'check_delisting':
        return ws ? (
          <ButtonLink size="sm" to="/w/$workspace/deliverability" params={{ workspace: ws.slug }}>
            {x.actionLabel}
          </ButtonLink>
        ) : null
      case 'open_snds':
      case 'open_postmaster':
        return x.link ? (
          <ButtonAnchor size="sm" href={x.link}>
            {x.actionLabel}
            <span className="vh"> {t(m.newTab)}</span>
          </ButtonAnchor>
        ) : null
      case 'show_link':
        return onShowLink && x.position !== undefined ? (
          <Button size="sm" className="rep-noprint" onClick={() => onShowLink(x.position!)}>
            {x.actionLabel}
          </Button>
        ) : null
    }
    return null
  }
  return (
    <section className="card min-w-0 flex-[3_1_560px]" aria-labelledby="rep-ins-h">
      <div className="sec">
        <h2 className="h2" id="rep-ins-h">
          {t(m.insightsTitle)}
        </h2>
        <span className="k">{t(m.insightsCaption)}</span>
      </div>
      {r.insights.length ? (
        <ul className="rep-ins">
          {r.insights.map((x, i) => (
            <li key={`${x.kind}-${i}`}>
              <Badge tone={tone[x.tone]} dot>
                {word[x.kind]}
              </Badge>
              <div className="rt">
                <span>{x.title}</span>
                <span className="k">{x.detail}</span>
              </div>
              {action(x)}
            </li>
          ))}
        </ul>
      ) : (
        <p className="k m-0 px-4.5 py-3.5">{t(m.insightsNone)}</p>
      )}
    </section>
  )
}

/** One of the growth card's tiles. */
function GrowthTile({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="rep-gt">
      <span className="k">{label}</span>
      <span className="n text-[20px]">{value}</span>
      <span className="k">{sub}</span>
    </div>
  )
}

/**
 * "Growth & revenue from this issue" (review AN-12, EarnInReport): people
 * who joined because of it and its views on the web, then the upgrades and
 * conversions Earn credits to it, when the reader may see them.
 */
export function Growth({ r, campaignId }: { r: CampaignReport; campaignId?: string }) {
  const t = useT()
  const ws = useOptionalWorkspace()
  const canEarn = useCan('earn:view')
  const rev = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{id}/revenue',
    { params: { path: { workspace: ws?.slug ?? '', id: campaignId ?? '' } } },
    { enabled: Boolean(ws && campaignId && canEarn) },
  )
  const g = r.growth
  const v = rev.data?.active && !rev.data.locked ? rev.data : undefined
  const c = v?.conversions
  const kind = c?.name
    ? c.sender
      ? t(mrp.convKind, { name: c.name, value: money(c.typicalCents ?? 0, c.currency), sender: c.sender })
      : t(mrp.convKindNoSender, { name: c.name, value: money(c.typicalCents ?? 0, c.currency) })
    : t(mrp.convKinds)
  return (
    <section className="card min-w-0 flex-[2_1_400px]" aria-labelledby="rep-gr-h">
      <div className="sec">
        <h2 className="h2" id="rep-gr-h">
          {v ? t(m.growthTitle) : t(m.growthTitleOnly)}
        </h2>
        {g.since ? <span className="k">{t(m.growthSince, { date: shortDate(g.since) })}</span> : null}
      </div>
      <div className="flex flex-col gap-3 px-4.5 py-3.5">
        <div className="rep-gts">
          <GrowthTile
            label={t(m.newSubscribers)}
            value={count(g.newSubscribers)}
            sub={t(m.newSubscribersSub, { forwards: count(g.forwards), referrals: count(g.referrals), web: count(g.webPost) })}
          />
          <GrowthTile label={t(m.webViews)} value={count(g.webViews)} sub={t(m.webViewsSub)} />
          {v ? (
            <>
              <GrowthTile
                label={t(m.upgradesToPaid)}
                value={count(v.upgrades)}
                sub={v.hasUpgradeLinks ? t(m.upgradesSub, { paid: count(v.paidAtOnce), trials: v.trials }) : t(m.upgradesNoLinks)}
              />
              <GrowthTile
                label={t(m.productConversions)}
                value={count(v.conversions.count)}
                sub={t(m.productConversionsSub, { value: money(v.conversions.valueCents, v.conversions.currency, 'never') })}
              />
            </>
          ) : null}
        </div>
        {v && c ? (
          <>
            <dl className="kv">
              <dt>{t(mrp.kvConv)}</dt>
              <dd>{c.count ? kind : <span className="k">{t(mrp.convNone)}</span>}</dd>
              {c.count ? (
                <>
                  <dt>{t(mrp.kvMatched)}</dt>
                  <dd>
                    {c.domains.length
                      ? t(mrp.matchedTokenOn, { token: count(c.byToken), email: count(c.byEmail), domains: andList(c.domains) })
                      : t(mrp.matched, { token: count(c.byToken), email: count(c.byEmail) })}
                  </dd>
                </>
              ) : null}
              <dt>{t(mrp.kvWindow)}</dt>
              <dd>{t(mrp.window, { days: v.window })}</dd>
            </dl>
            <p className="k m-0">{t(m.creditRule, { days: v.window })}</p>
          </>
        ) : null}
        {ws ? (
          <div className="flex flex-wrap gap-x-4 gap-y-2">
            <Link className="rep-lnk" to="/w/$workspace/grow" params={{ workspace: ws.slug }}>
              {t(m.openGrow)}
            </Link>
            {v ? (
              <Link className="rep-lnk" to="/w/$workspace/earn" params={{ workspace: ws.slug }}>
                {t(m.openEarn)}
              </Link>
            ) : null}
          </div>
        ) : null}
      </div>
    </section>
  )
}
