import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, shortDate, usDate, notificationTime } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { mp } from './messages'
import './contact.css'

type Grow = Schemas['ReferralContact']
type Referrer = Schemas['ReferralContactReferrer']
type ReferredBy = Schemas['ReferralContactReferredBy']
export type GrowEvent = Schemas['ReferralTimelineEvent']
type TFn = ReturnType<typeof useT>

/** Grow on a contact's profile, for people who can see Grow. */
export function useContactGrow(contactId: string) {
  const { slug } = useWorkspace()
  const can = useCan('grow:view')
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contact}/referrals', { params: { path: { workspace: slug, contact: contactId } } }, { enabled: can, staleTime: 30_000 })
}

/**
 * The profile's Referrals card (brief §3.7): a referrer's link and its
 * status, their referrals by state, milestones and rewards, and "Open in
 * Grow"; for a referred contact, who referred them and where that stands.
 * Nothing for contacts outside Grow.
 */
export function ProfileReferralsCard({ contactId }: { contactId: string }) {
  const t = useT()
  const q = useContactGrow(contactId)
  const g: Grow | undefined = q.data
  if (!g || (!g.referrer && !g.referredBy)) return null
  if (!g.referrer) {
    return (
      <section className="card gw-pref" aria-label={t(mp.cardTitle)}>
        <ReferredByLine r={g.referredBy!} />
      </section>
    )
  }
  return <ReferrerCard r={g.referrer} by={g.referredBy} />
}

function ReferrerCard({ r, by }: { r: Referrer; by?: ReferredBy }) {
  const t = useT()
  const { slug } = useWorkspace()
  // Every subscriber of the program's list has a link: one nobody used yet is a line.
  const quiet = r.counted + r.inHold + r.awaiting + r.held + r.visits + r.signups === 0 && r.rewards.length === 0
  return (
    <section className="card gw-pref" aria-labelledby="gw-pref-h">
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="gw-pref-h">
            {t(mp.cardTitle)}
          </h2>
          <span className="k">{t(mp.cardCaptionList, { list: r.listName })}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/grow/referrals/referrers" params={{ workspace: slug }} search={{ program: r.programId, referrer: r.linkId }}>
          {t(mp.openInGrow)}
        </ButtonLink>
      </div>
      {by ? <ReferredByLine r={by} /> : null}
      <div className="gw-pref-link">
        <span className="k">{t(mp.link)}</span>
        <span className="mono">{r.link}</span>
        <Badge tone={r.linkStatus === 'active' ? 'ok' : 'neutral'}>{t(r.linkStatus === 'active' ? mp.linkActive : mp.linkPaused)}</Badge>
        <span className="k">{quiet ? t(mp.noReferrals) : t(mp.visits, { visits: r.visits, signups: r.signups })}</span>
      </div>
      {quiet ? null : <ReferrerNumbers r={r} />}
    </section>
  )
}

function ReferrerNumbers({ r }: { r: Referrer }) {
  const t = useT()
  return (
    <>
      <dl className="gw-pref-stats">
        <Stat label={t(mp.counted)} value={count(r.counted)} />
        <Stat label={t(mp.inHold)} value={count(r.inHold)} />
        <Stat label={t(mp.awaiting)} value={count(r.awaiting)} />
        {r.held ? <Stat label={t(mp.held)} value={count(r.held)} /> : null}
      </dl>
      <div className="flex flex-col gap-1.5">
        <span className="lbl">{t(mp.rewards)}</span>
        {r.rewards.length ? (
          <ul className="gw-pref-rewards">
            {r.rewards.map((w) => {
              const b = rewardBadge(t, w.status, Boolean(w.usedAt))
              return (
                <li key={`${w.milestone}-${w.name}`}>
                  <span>{t(mp.rewardLine, { name: w.name, milestone: w.milestone })}</span>
                  <Badge tone={b.tone}>{b.label}</Badge>
                  <span className="k">{w.usedAt ? t(mp.usedOn, { date: dayMonthDate(w.usedAt) }) : w.sentAt ? t(mp.sentOn, { date: dayMonthDate(w.sentAt) }) : ''}</span>
                </li>
              )
            })}
          </ul>
        ) : (
          <span className="k">{t(mp.noRewards)}</span>
        )}
      </div>
    </>
  )
}

function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="k">{label}</dt>
      <dd className="m-0 text-[18px] font-bold tabular-nums">{value}</dd>
    </div>
  )
}

function rewardBadge(t: TFn, status: string, used: boolean): { tone: Tone; label: string } {
  if (used) return { tone: 'ok', label: t(mp.rewardUsed) }
  switch (status) {
    case 'sent':
      return { tone: 'ok', label: t(mp.rewardSent) }
    case 'fulfilled':
      return { tone: 'ok', label: t(mp.rewardFulfilled) }
    case 'to_fulfill':
      return { tone: 'info', label: t(mp.rewardToFulfill) }
    case 'waiting_codes':
      return { tone: 'warn', label: t(mp.rewardWaiting) }
    case 'failed':
      return { tone: 'bad', label: t(mp.rewardFailed) }
    case 'expired':
      return { tone: 'neutral', label: t(mp.rewardExpired) }
  }
  return { tone: 'neutral', label: t(mp.rewardPending) }
}

/** "Referred by Sam Rivera · counts on 19 Oct". */
function ReferredByLine({ r }: { r: ReferredBy }) {
  const t = useT()
  const { slug } = useWorkspace()
  const state =
    r.status === 'hold' && r.countsOn
      ? t(mp.referredByHold, { date: dayMonthDate(r.countsOn) })
      : r.status === 'counted'
        ? r.countedAt
          ? t(mp.referredByCounted, { date: dayMonthDate(r.countedAt) })
          : t(mp.referredByCountedNoDate)
        : r.status === 'review'
          ? t(mp.referredByReview)
          : r.status === 'not_counted'
            ? t(mp.referredByNotCounted)
            : t(mp.referredByAwaiting)
  const name = r.referrer || '–'
  return (
    <p className="gw-pref-by">
      <T
        msg={mp.referredBy}
        values={{
          name,
          link: (c: ReactNode) =>
            r.referrerId ? (
              <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: r.referrerId }}>
                {c}
              </Link>
            ) : (
              <b>{c}</b>
            ),
        }}
      />
      <span className="k"> · {state}</span>
    </p>
  )
}

// --- Timeline ----------------------------------------------------------------------

/** "Today 10:42", else "Tue 7 Oct 09:41" (as the profile's other events). */
const when = (iso: string) => (usDate(iso) === usDate(new Date()) ? notificationTime(iso) : `${shortDate(iso)} ${clock(iso)}`)

export type GrowEntry = { kind: 'grow'; at: string; e: GrowEvent }

/**
 * The timeline with Grow's events in it, by date: within the events loaded
 * so far, all of them once the list is complete (as transactional mail).
 */
export function withGrow<E extends { kind: string }>(entries: E[], events: GrowEvent[], complete: boolean, oldest: number): (E | GrowEntry)[] {
  const out: (E | GrowEntry)[] = [...entries]
  for (const e of events) {
    const t = Date.parse(e.at)
    if (!complete && t < oldest) continue
    let i = out.findIndex((x) => 'at' in x && typeof x.at === 'string' && Date.parse(x.at) < t)
    if (i < 0) i = out.length
    out.splice(i, 0, { kind: 'grow', at: e.at, e })
  }
  return out
}

/**
 * The Grow events to list among the contact's activity: all of them in All,
 * joins in Consent. A reward sent with its milestone is part of the
 * milestone's event ("Reached milestone 1 · Sticker pack sent").
 */
export function growEvents(g: Grow | undefined, filter: string): GrowEvent[] {
  if (!g) return []
  if (filter === 'consent') return g.events.filter((e) => e.kind === 'joined' || e.kind === 'partner_request')
  if (filter !== 'all') return []
  const withMilestone = (e: GrowEvent) =>
    g.events.some((m) => m.kind === 'milestone' && m.milestone === e.milestone && Math.abs(Date.parse(m.at) - Date.parse(e.at)) <= 10 * 60_000)
  const sentWith = (m: GrowEvent) =>
    g.events.some((e) => e.kind === 'reward_sent' && e.milestone === m.milestone && Math.abs(Date.parse(m.at) - Date.parse(e.at)) <= 10 * 60_000)
  return g.events
    .filter((e) => !(e.kind === 'reward_sent' && withMilestone(e)))
    .map((e) => (e.kind === 'milestone' && !sentWith(e) ? { ...e, reward: '' } : e))
}

function eventTitle(t: TFn, e: GrowEvent): string {
  switch (e.kind) {
    case 'joined':
      return e.referrer ? t(mp.evJoined, { name: e.referrer }) : t(mp.evJoinedUnknown)
    case 'counted':
      return e.referrer ? t(mp.evCounted, { name: e.referrer, count: e.counted }) : t(mp.evCountedUnknown)
    case 'milestone':
      return e.reward ? t(mp.evMilestoneReward, { milestone: e.milestone, reward: e.reward }) : t(mp.evMilestone, { milestone: e.milestone })
    case 'reward_sent':
      return t(mp.evRewardSent, { reward: e.reward })
    case 'partner_request':
      return t(mp.evPartner, { partner: e.partner })
  }
  return e.kind
}

/** A Grow event on the profile's timeline (the activity list's row). */
export function GrowActivityRow({ e }: { e: GrowEvent }) {
  const t = useT()
  return (
    <li className="ev">
      <span className="ic" style={{ background: 'var(--accent-tint)' }}>
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="var(--accent)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          {e.kind === 'partner_request' ? <path d="M3 8h10M9 4l4 4-4 4" /> : <path d="M8 14V7M8 9c0-3 2-4.5 5-4.5 0 3-2 4.5-5 4.5zM8 11C8 8.5 6.5 7 3.5 7c0 2.5 1.5 4 4.5 4z" />}
        </svg>
      </span>
      <div className="flex flex-1 flex-col gap-0.5">
        <span>
          <b>{eventTitle(t, e)}</b>
        </span>
        <span className="k">
          {when(e.at)} · {t(mp.evMeta)}
        </span>
      </div>
    </li>
  )
}

